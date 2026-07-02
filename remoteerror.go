package apperror

import (
	"fmt"
	"strings"
	"time"
)

// RemoteErrorResp is a DTO that captures the parsed response from a failed
// remote-service call. It is NOT an error — it is data. Different remote
// services have different error-response formats; RemoteErrorResp is the
// package-wide normalised representation.
//
// Construct it at the driven-adapter boundary after parsing the response
// body, then wrap it into a RemoteError via WithErrResp.
type RemoteErrorResp struct {
	// Request is the captured outbound request. Often nil to avoid logging
	// sensitive bodies.
	Request *Request

	// Response is the captured response. Should be non-nil when a response
	// was received from the remote service — its presence is what
	// distinguishes a remote-response failure from a transport failure. May
	// be nil when the client library returned no response (e.g. a raw
	// transport error without an HTTP response).
	Response *Response

	// BodyCode is the remote service's application-level error code, parsed
	// from Response.Body (e.g. "card_declined"). Empty when the remote
	// didn't supply one.
	BodyCode string

	// BodyMessage is the remote service's original error message, parsed
	// from Response.Body. Preserved for forensics.
	BodyMessage string

	// RetryAfter is the duration the remote service asks us to wait before
	// retrying, normalised from whatever form the remote used (HTTP
	// Retry-After header, a body field, etc.). Zero means no hint.
	RetryAfter time.Duration
}

// StatusCode returns the protocol-level status code of the captured
// Response. Callers should guard with a nil check on Response before
// calling this — a nil Response causes a panic.
func (r *RemoteErrorResp) StatusCode() int {
	return r.Response.StatusCode
}

// String formats the DTO for debugging. Output is RemoteErrorResp-shaped
// (not error-shaped) because this is a data record, not an error.
func (r *RemoteErrorResp) String() string {
	respStr := "None"
	if r.Response != nil {
		respStr = r.Response.String()
	}
	return fmt.Sprintf(
		"RemoteErrorResp(status=%d, bodyCode=%s, bodyMessage=%s, retryAfter=%s, response=%s)",
		r.StatusCode(), r.BodyCode, r.BodyMessage, r.RetryAfter, respStr,
	)
}

// RemoteError represents a failure when calling a remote service. It is a
// first-class error type, structurally parallel to AppError, and carries
// the same Code taxonomy.
//
// Two construction paths:
//
//  1. Remote returned a response (any status code):
//     Parse the response into a RemoteErrorResp, then construct a
//     RemoteError via factory + WithErrResp. Optionally also pass a
//     transport-level cause when both the response body and the
//     underlying error are relevant.
//
//  2. No response received (transport failure):
//     Construct a RemoteError via factory, passing the raw transport error
//     as cause via an inline RemoteOption closure:
//
//     NewRemoteUnavailable("svc.op", func(re *RemoteError) { re.cause = connErr })
//
// # Relationship to AppError
//
// RemoteError is NOT a subtype of AppError. It is the error that a driven
// adapter returns; the caller at the next layer may choose to wrap it as
// the cause of an AppError (e.g. NewInternal("user.lookup",
// WithCause(remoteErr))) if the failure needs reclassification for its own
// caller, or propagate it directly.
//
// # Three views of one error
//
// Together RemoteError and its errResp expose three layers:
//
//   - Canonical (RemoteError.Code(), .Message(), etc.) — our normalised
//     taxonomy. Use for retry / circuit breaker / log aggregation keys.
//
//   - Protocol (errResp.StatusCode()) — HTTP/RPC status. Use for retry
//     decisions that depend on transport class.
//
//   - Remote application (errResp.BodyCode, .BodyMessage) — the foreign
//     service's own signals. Preserved for forensics; not for branching.
//
// Conventions
//
//   - event is required (factories panic on empty). Recommended format:
//     "{namespace}[.{sub-namespace}].{operation}". In the Adapter and
//     Infrastructure layers this is typically "{service}.{operation}"
//     (e.g. "UserService.GetUser").
//   - errResp and cause may both be set, neither is an error. At least one
//     must be set: setting neither panics at construction time (a RemoteError
//     without evidence is meaningless).
type RemoteError struct {
	code    Code
	event   string
	caseVal Case
	message string
	details any
	cause   error
	errResp *RemoteErrorResp
	stack   StackTrace
}

// RemoteOption configures a RemoteError during construction.
type RemoteOption func(*RemoteError)

// WithErrResp attaches the parsed remote error response to the RemoteError.
func WithErrResp(resp *RemoteErrorResp) RemoteOption {
	return func(e *RemoteError) { e.errResp = resp }
}

// newRemoteError is the internal shared constructor used by the per-Code
// factory functions.
func newRemoteError(code Code, event string, opts ...RemoteOption) *RemoteError {
	if strings.TrimSpace(event) == "" {
		panic(fmt.Sprintf("apperror: event is required (constructing %s)", code.Name()))
	}
	e := &RemoteError{
		code:  code,
		event: event,
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.errResp == nil && e.cause == nil {
		panic(fmt.Sprintf("apperror: at least one of errResp or cause must be set (constructing %s)", code.Name()))
	}
	if strings.TrimSpace(e.message) == "" {
		e.message = code.Description()
	}
	e.stack = callers(4)
	return e
}

// Code returns the operation status code.
func (e *RemoteError) Code() Code { return e.code }

// Event returns the event name for structured logging.
func (e *RemoteError) Event() string { return e.event }

// Case returns the attached Case, or nil.
func (e *RemoteError) Case() Case { return e.caseVal }

// Message returns the error message. Never empty: falls back to
// Code.Description() when not explicitly set.
func (e *RemoteError) Message() string { return e.message }

// Details returns the attached details, or nil.
func (e *RemoteError) Details() any { return e.details }

// Cause returns the underlying cause, or nil.
func (e *RemoteError) Cause() error { return e.cause }

// ErrResp returns the parsed remote error response DTO, or nil if this
// RemoteError wraps a transport error instead.
func (e *RemoteError) ErrResp() *RemoteErrorResp { return e.errResp }

// Unwrap returns the underlying cause. Returns nil when no cause was set
// (e.g. when only errResp is set, the RemoteError has no cause to unwrap).
func (e *RemoteError) Unwrap() error { return e.cause }

// StackTrace returns the call stack captured at construction time.
func (e *RemoteError) StackTrace() StackTrace { return e.stack }

// AddNote prepends context to the message, joined by " -> ". See
// AppError.AddNote for the full rationale.
func (e *RemoteError) AddNote(note string) {
	if e.message != "" {
		e.message = note + " -> " + e.message
	} else {
		e.message = note
	}
}

// Error implements the error interface.
func (e *RemoteError) Error() string { return e.String() }

// String returns a debug representation. When errResp is non-nil its
// fields are included inline.
func (e *RemoteError) String() string {
	caseStr := "None"
	if e.caseVal != nil {
		caseStr = e.caseVal.Identifier()
	}
	detailsStr := "None"
	if e.details != nil {
		detailsStr = fmt.Sprintf("%v", e.details)
	}
	errRespStr := "None"
	if e.errResp != nil {
		errRespStr = e.errResp.String()
	}
	return fmt.Sprintf(
		"RemoteError(code=%s, event=%s, case=%s, message='%s', details=%s, errResp=%s)",
		e.code.String(), e.event, caseStr, e.message, detailsStr, errRespStr,
	)
}

// --- Factory functions ---
//
// Order mirrors the order of Code constants in code.go.

// NewRemoteCancelled creates a RemoteError for a cancelled operation. (Code 1)
func NewRemoteCancelled(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeCancelled, event, opts...)
}

// NewRemoteUnknown creates a RemoteError for an unknown error. (Code 2)
func NewRemoteUnknown(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeUnknown, event, opts...)
}

// NewRemoteIllegalInput creates a RemoteError for illegal input. (Code 3)
func NewRemoteIllegalInput(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeIllegalInput, event, opts...)
}

// NewRemoteTimeout creates a RemoteError for a timed-out operation. (Code 4)
func NewRemoteTimeout(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeTimeout, event, opts...)
}

// NewRemoteNotFound creates a RemoteError for a missing entity. (Code 5)
func NewRemoteNotFound(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeNotFound, event, opts...)
}

// NewRemoteAlreadyExists creates a RemoteError for an already-existing entity. (Code 6)
func NewRemoteAlreadyExists(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeAlreadyExists, event, opts...)
}

// NewRemotePermissionDenied creates a RemoteError for a permission failure. (Code 7)
func NewRemotePermissionDenied(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodePermissionDenied, event, opts...)
}

// NewRemoteTooManyRequests creates a RemoteError for quota/rate-limit exhaustion. (Code 8)
func NewRemoteTooManyRequests(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeTooManyRequests, event, opts...)
}

// NewRemoteFailedPrecondition creates a RemoteError for a precondition failure. (Code 9)
func NewRemoteFailedPrecondition(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeFailedPrecondition, event, opts...)
}

// NewRemoteConflict creates a RemoteError for a concurrent-operation conflict. (Code 10)
func NewRemoteConflict(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeConflict, event, opts...)
}

// NewRemoteOutOfRange creates a RemoteError for an out-of-range condition. (Code 11)
func NewRemoteOutOfRange(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeOutOfRange, event, opts...)
}

// NewRemoteUnimplemented creates a RemoteError for an unimplemented operation. (Code 12)
func NewRemoteUnimplemented(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeUnimplemented, event, opts...)
}

// NewRemoteInternal creates a RemoteError for a server-side internal error. (Code 13)
func NewRemoteInternal(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeInternal, event, opts...)
}

// NewRemoteUnavailable creates a RemoteError for a transient unavailable condition. (Code 14)
func NewRemoteUnavailable(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeUnavailable, event, opts...)
}

// NewRemoteIllegalState creates a RemoteError for an illegal-state condition. (Code 15)
func NewRemoteIllegalState(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeIllegalState, event, opts...)
}

// NewRemoteUnauthenticated creates a RemoteError for missing/invalid credentials. (Code 16)
func NewRemoteUnauthenticated(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeUnauthenticated, event, opts...)
}

// NewRemoteIllegalArg creates a RemoteError for illegal program-internal arguments. (Code 29)
func NewRemoteIllegalArg(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeIllegalArg, event, opts...)
}

// NewRemoteUnauthorized creates a RemoteError for expired authorization. (Code 30)
func NewRemoteUnauthorized(event string, opts ...RemoteOption) *RemoteError {
	return newRemoteError(CodeUnauthorized, event, opts...)
}
