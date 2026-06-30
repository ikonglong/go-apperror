package apperror

import (
	"fmt"
	"strings"
)

// AppError represents an application error using this package's standardized
// Code taxonomy.
//
// # When to use AppError vs RemoteError
//
// AppError covers errors originating in this application — validation
// failures, business-rule violations, internal failures, and importantly:
// failures when calling a remote service where NO response was received
// (DNS failure, connection refused/reset, TLS handshake failure, timeout
// before any bytes arrived). For server-responded remote failures (any
// status code, app-level error envelope in the body), use RemoteError
// instead.
//
// # Construction
//
// Construct an AppError via one of the per-Code factory functions
// (NewNotFound, NewInternal, etc.). The package intentionally does
// not expose a generic "New(code, event, ...)" constructor: every
// AppError must be created with a Code from this package's standardized
// taxonomy, and a caller picks the Code by picking the corresponding
// factory. There is also no factory for CodeOK, because that is not a
// failure.
//
// Each factory takes event as a required positional argument and accepts
// optional Options (WithMessage, WithCase, WithDetails, WithCause). The
// event field is required because this library is designed to pair with
// structured logging where event drives log routing/aggregation; making
// it positional forces the caller to think about it at construction time
// instead of silently omitting it. message is optional — when not set, it
// falls back to Code.Description() so unstructured loggers still get a
// readable string.
type AppError struct {
	code    Code
	event   string
	caseVal Case
	message string
	details any
	cause   error
	stack   StackTrace
}

// Option configures an AppError during construction.
type Option func(*AppError)

// WithCase attaches a Case to the error. Case is optional and names the
// specific business condition that produced the error (e.g.
// "purchase_limit_exceeded"). It is orthogonal to Code — Code categorises
// the failure in this package's taxonomy, Case names the precise
// domain-level condition.
func WithCase(c Case) Option {
	return func(e *AppError) { e.caseVal = c }
}

// WithDetails attaches ad-hoc structured details to the error. Use it for
// one-off, low-frequency, low-structure context (e.g. a few extra fields
// to include in an API error response or a log entry). For high-frequency,
// stable error patterns where callers need to do typed programmatic checks,
// prefer defining a dedicated wrapper type that embeds *AppError (e.g.
// type ValidationError struct { *AppError; Fields []FieldError }) instead
// of stuffing typed data behind an any.
func WithDetails(d any) Option {
	return func(e *AppError) { e.details = d }
}

// WithCause wraps the underlying cause; the resulting error supports
// errors.Is / errors.As / errors.Unwrap on the cause.
func WithCause(err error) Option {
	return func(e *AppError) { e.cause = err }
}

// WithMessage attaches a human-readable message to the error. Optional —
// when not provided (or when the factory is called without it), the
// AppError's message falls back to Code.Description() so unstructured
// loggers and human readers still see a sensible string.
func WithMessage(msg string) Option {
	return func(e *AppError) { e.message = msg }
}

// newAppError is the internal shared constructor used by the per-Code
// factory functions. It is unexported on purpose: callers must pick a
// Code by picking the corresponding factory, not by passing an arbitrary
// Code value.
//
// event is required. Empty event (after TrimSpace) panics — see the
// AppError type doc for why. If WithMessage is not used (or sets the
// message to ""), the message falls back to code.Description() so an
// AppError always renders a non-empty message.
func newAppError(code Code, event string, opts ...Option) *AppError {
	if strings.TrimSpace(event) == "" {
		panic(fmt.Sprintf("apperror: event is required (constructing %s)", code.Name()))
	}
	e := &AppError{
		code:  code,
		event: event,
	}
	for _, opt := range opts {
		opt(e)
	}
	if strings.TrimSpace(e.message) == "" {
		e.message = code.Description()
	}
	// Capture the stack at the construction site. callers(4) drops the
	// four internal frames between here and the caller — runtime.Callers,
	// callers, newAppError, and the per-Code factory (NewNotFound/...) —
	// so the first recorded frame is the code that constructed the error.
	e.stack = callers(4)
	return e
}

// Code returns the canonical operation status code.
func (e *AppError) Code() Code { return e.code }

// Case returns the attached Case, or nil if none was set.
func (e *AppError) Case() Case { return e.caseVal }

// Message returns the error message (after any AddNote prepends). Never
// empty: when WithMessage was omitted or set to "", the constructor falls
// back to Code.Description().
func (e *AppError) Message() string { return e.message }

// Details returns the attached details, or nil if none.
func (e *AppError) Details() any { return e.details }

// Event returns the event name for structured logging. Always non-empty
// (factories panic on empty event at construction time). The event names
// the operation during which the error occurred (e.g. "user.signup",
// "order.create"), not the failure mode (which Code and Case describe).
// Recommended convention: "<domain>.<operation>".
func (e *AppError) Event() string { return e.event }

// Cause returns the underlying cause, or nil. Equivalent to Unwrap.
func (e *AppError) Cause() error { return e.cause }

// Unwrap returns the underlying cause for use with errors.Is / errors.As.
func (e *AppError) Unwrap() error { return e.cause }

// StackTrace returns the call stack captured when this AppError was
// constructed, innermost (origin) frame first. Use StackTrace().Strings()
// to render it as single-line "file:line function" entries suitable for a
// structured log field.
//
// The capture point is where the error was first classified into this
// package's taxonomy — i.e. where a factory (NewNotFound, NewInternal,
// ...) was called. AddNote does not create a new error, so it leaves this
// origin stack intact.
func (e *AppError) StackTrace() StackTrace { return e.stack }

// AddNote enriches the error's message with additional context, prepended
// and joined by " -> ". Use this when you want to add a layer of context to
// an existing AppError *without* changing its identity (Code, Case, Details
// remain the same and the error chain does not grow).
//
// When to use AddNote vs fmt.Errorf:
//
//   - Same error, more context (e.g. you caught an AppError from a lower
//     layer and want to note which higher-level operation it occurred in):
//     use AddNote. The returned error stays *AppError and consumers can
//     reach Code/Case/Details directly without errors.As.
//
//   - Different error or new layer in the chain (e.g. you want to wrap a
//     non-AppError, or attach a cause, or change semantics): use
//     fmt.Errorf("ctx: %w", err) — that creates a new wrapper and grows the
//     errors.Is/As chain, which is the right model when the wrapping layer
//     is genuinely a new error event.
//
// Prepending matches the convention used by fmt.Errorf %w wrapping: the
// outermost context appears first, the innermost root cause last.
//
// Separator choice: " -> " is used (not ": " as fmt.Errorf uses) because
// after multiple calls the message becomes a single flat string and the
// only way a reader can tell layer boundaries apart is by recognising the
// separator token. ": " also frequently appears inside the messages being
// joined (e.g. "id: 42 not found", "url: /x"), so "creating user: id: 42
// not found" has three colons of identical appearance and the boundary is
// ambiguous. " -> " is chosen because it almost never occurs inside an
// error message, so every " -> " in the final string is unambiguously a
// layer separator.
func (e *AppError) AddNote(note string) {
	if e.message != "" {
		e.message = note + " -> " + e.message
	} else {
		e.message = note
	}
}

// Error implements the error interface, returning the same string as String.
func (e *AppError) Error() string { return e.String() }

// String returns a debug representation of the error.
func (e *AppError) String() string {
	caseStr := "None"
	if e.caseVal != nil {
		caseStr = e.caseVal.Identifier()
	}
	detailsStr := "None"
	if e.details != nil {
		detailsStr = fmt.Sprintf("%v", e.details)
	}
	return fmt.Sprintf(
		"AppError(code=%s, event=%s, case=%s, message='%s', details=%s)",
		e.code.String(), e.event, caseStr, e.message, detailsStr,
	)
}

// --- Factory functions for common codes ---
//
// Order below mirrors the order of Code constants declared in code.go.

// NewCancelled creates an AppError for a cancelled operation. (Code 1)
func NewCancelled(event string, opts ...Option) *AppError {
	return newAppError(CodeCancelled, event, opts...)
}

// NewUnknown creates an AppError for an unknown error. (Code 2)
func NewUnknown(event string, opts ...Option) *AppError {
	return newAppError(CodeUnknown, event, opts...)
}

// NewIllegalInput creates an AppError for illegal client input. (Code 3)
func NewIllegalInput(event string, opts ...Option) *AppError {
	return newAppError(CodeIllegalInput, event, opts...)
}

// NewTimeout creates an AppError for a timed-out operation. (Code 4)
func NewTimeout(event string, opts ...Option) *AppError {
	return newAppError(CodeTimeout, event, opts...)
}

// NewNotFound creates an AppError for a missing entity. (Code 5)
func NewNotFound(event string, opts ...Option) *AppError {
	return newAppError(CodeNotFound, event, opts...)
}

// NewAlreadyExists creates an AppError for an already-existing entity. (Code 6)
func NewAlreadyExists(event string, opts ...Option) *AppError {
	return newAppError(CodeAlreadyExists, event, opts...)
}

// NewPermissionDenied creates an AppError for a permission failure. (Code 7)
func NewPermissionDenied(event string, opts ...Option) *AppError {
	return newAppError(CodePermissionDenied, event, opts...)
}

// NewTooManyRequests creates an AppError for quota / rate-limit exhaustion. (Code 8)
func NewTooManyRequests(event string, opts ...Option) *AppError {
	return newAppError(CodeTooManyRequests, event, opts...)
}

// NewFailedPrecondition creates an AppError for a precondition failure. (Code 9)
func NewFailedPrecondition(event string, opts ...Option) *AppError {
	return newAppError(CodeFailedPrecondition, event, opts...)
}

// NewConflict creates an AppError for a concurrent-operation conflict. (Code 10)
func NewConflict(event string, opts ...Option) *AppError {
	return newAppError(CodeConflict, event, opts...)
}

// NewOutOfRange creates an AppError for an out-of-range condition. (Code 11)
func NewOutOfRange(event string, opts ...Option) *AppError {
	return newAppError(CodeOutOfRange, event, opts...)
}

// NewUnimplemented creates an AppError for an unimplemented operation. (Code 12)
func NewUnimplemented(event string, opts ...Option) *AppError {
	return newAppError(CodeUnimplemented, event, opts...)
}

// NewInternal creates an AppError for a server-side internal error. (Code 13)
func NewInternal(event string, opts ...Option) *AppError {
	return newAppError(CodeInternal, event, opts...)
}

// NewUnavailable creates an AppError for a transient unavailable condition. (Code 14)
func NewUnavailable(event string, opts ...Option) *AppError {
	return newAppError(CodeUnavailable, event, opts...)
}

// NewIllegalState creates an AppError for an illegal-state condition. (Code 15)
func NewIllegalState(event string, opts ...Option) *AppError {
	return newAppError(CodeIllegalState, event, opts...)
}

// NewUnauthenticated creates an AppError for missing/invalid credentials. (Code 16)
func NewUnauthenticated(event string, opts ...Option) *AppError {
	return newAppError(CodeUnauthenticated, event, opts...)
}

// NewIllegalArg creates an AppError for illegal program-internal arguments. (Code 29)
func NewIllegalArg(event string, opts ...Option) *AppError {
	return newAppError(CodeIllegalArg, event, opts...)
}

// NewUnauthorized creates an AppError for expired authorization. (Code 30)
func NewUnauthorized(event string, opts ...Option) *AppError {
	return newAppError(CodeUnauthorized, event, opts...)
}
