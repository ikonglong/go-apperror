package apperror

import (
	"fmt"
	"time"
)

// RemoteError represents a failure observed when this application called
// another service over the network and **the server returned a response**.
// Any status code (2xx/4xx/5xx) or any in-band error envelope in the body
// qualifies, as long as a response was received. It covers both
// intra-company services and third-party APIs.
//
// # What is NOT a RemoteError
//
// Transport-layer failures — DNS resolution failure, connection refused or
// reset, TLS handshake failure, request timing out before any bytes were
// received — are NOT RemoteErrors. The remote service never "responded",
// so there is nothing for RemoteError to model. Represent those as plain
// AppError (typically NewUnavailable) wrapping the transport error via
// WithCause. The two error kinds are distinguished by *type*, not by
// fields inside one shared type.
//
// # Relationship to AppError
//
// RemoteError is NOT a subtype of AppError, and it carries no normalized
// taxonomy of its own. Instead, at the call boundary the client picks one
// of this package's Code constants (e.g. CodeNotFound, CodeTooManyRequests)
// and constructs a canonical *AppError that wraps this RemoteError as its
// cause:
//
//	remoteErr := &RemoteError{Service: ..., Operation: ..., Response: ...}
//	return NewTooManyRequests("user-service.GetUser", WithCause(remoteErr))
//
// That canonical *AppError is the value that propagates up; it is the
// normalized view cross-cutting logic branches on. The RemoteError stays
// reachable as the cause (errors.As(err, &remoteErr)) so a centralized
// boundary logger can record the remote-side root cause, but, by
// convention, layers above the adapter work with the *AppError, not the
// RemoteError.
//
// # Three views of one error
//
// Together the wrapping *AppError and this RemoteError expose three layers
// of error information, all describing the same failure from different
// angles:
//
//   - Canonical (the wrapping AppError's Code(), Message(), etc.) — our
//     normalized taxonomy. Use for retry / circuit breaker / log
//     aggregation keys. Reach it with errors.As(err, &appErr).
//
//   - Protocol (StatusCode, surfaced from Response.StatusCode) — the
//     transport-protocol status (HTTP status / gRPC status). Use for
//     retry decisions that depend on transport class.
//
//   - Remote application (BodyCode, BodyMessage) — the foreign
//     service's own application-level signals parsed from the response
//     body. Preserved for forensics and ops/runbook references; not the
//     right thing to branch on for cross-cutting logic.
//
// Conventions
//
//   - Response must be non-nil — that's the precondition that makes this a
//     RemoteError in the first place.
//   - Wrap the RemoteError as the cause of a canonical *AppError via
//     WithCause, and propagate that AppError. A RemoteError should not be
//     the top-level error past the adapter boundary.
//   - Operation is a logical operation name (e.g. "GetUser"), not an HTTP
//     method + path.
type RemoteError struct {
	// Service is the logical name of the remote service called
	// (e.g. "user-service", "stripe").
	Service string

	// Operation is the logical name of the operation invoked
	// (e.g. "GetUser", "CreateCharge"). Not an HTTP method + path.
	Operation string

	// Request is the captured outbound request. Often nil to avoid logging
	// sensitive bodies.
	Request *Request

	// Response is the captured response. Must be non-nil — its presence is
	// the precondition for using RemoteError.
	Response *Response

	// BodyCode is the remote service's application-level error code, parsed
	// from Response.Body (e.g. "card_declined"). Empty when the remote
	// didn't supply one. Useful as a low-cardinality aggregation key for
	// observability.
	BodyCode string

	// BodyMessage is the remote service's original error message, parsed
	// from Response.Body. Preserved for forensics.
	BodyMessage string

	// RetryAfter is the duration the remote service asks us to wait before
	// retrying, normalized from whatever form the remote used (HTTP
	// Retry-After header, a body field, etc.). Zero means the remote did
	// not provide a hint.
	RetryAfter time.Duration
}

// StatusCode returns the protocol-level status code of the captured
// Response. Response is required to be non-nil; if it isn't, this will
// panic (which is the correct behavior — it surfaces a convention violation
// loudly).
func (r *RemoteError) StatusCode() int {
	return r.Response.StatusCode
}

// Event returns "Service.Operation", suitable as a single low-cardinality
// key in structured logs or as a span/event name. For dashboards that need
// to slice by service or operation independently, log Service and Operation
// as separate fields too.
func (r *RemoteError) Event() string {
	return r.Service + "." + r.Operation
}

// RemoteError is a leaf error: it has no Unwrap because it has no cause.
// The canonical view lives on the *AppError that wraps it; reach that with
// errors.As(err, &appErr).

// Error implements the error interface.
func (r *RemoteError) Error() string { return r.String() }

// String formats the error for human consumption. Output is RemoteError-
// shaped (not AppError-shaped) so the remote-side fields are visible. The
// canonical Code/Message are not repeated here — they live on the wrapping
// AppError, which is emitted as its own layer (e.g. by FlatMessage).
func (r *RemoteError) String() string {
	return fmt.Sprintf(
		"RemoteError(service=%s, operation=%s, status=%d, bodyCode=%s, retryAfter=%s)",
		r.Service, r.Operation, r.StatusCode(), r.BodyCode, r.RetryAfter,
	)
}
