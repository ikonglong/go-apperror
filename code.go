package apperror

import "fmt"

// Code is the canonical operation status code for an application error.
//
// Sometimes multiple error codes may apply. Services should return the most
// specific error code that applies. For example, prefer CodeOutOfRange over
// CodeFailedPrecondition if both codes apply. Similarly prefer CodeNotFound
// or CodeAlreadyExists over CodeFailedPrecondition.
type Code int

const (
	// CodeOK is not an error code. It exists only for completeness of the
	// Code <-> HTTP-status mapping; no factory function is provided for it
	// because constructing an AppError with CodeOK would be a contradiction.
	// HTTP Mapping: 200 OK
	CodeOK Code = 0

	// CodeCancelled means the operation was cancelled, typically by the caller.
	// HTTP Mapping: 499 Client Closed Request
	CodeCancelled Code = 1

	// CodeUnknown is for an unknown error.
	// HTTP Mapping: 500 Internal Server Error
	CodeUnknown Code = 2

	// CodeIllegalInput means that the client specified an illegal input.
	// HTTP Mapping: 400 Bad Request
	CodeIllegalInput Code = 3

	// CodeTimeout means the deadline expired before the operation could complete.
	// HTTP Mapping: 504 Gateway Timeout
	CodeTimeout Code = 4

	// CodeNotFound means that some requested entity was not found.
	// HTTP Mapping: 404 Not Found
	CodeNotFound Code = 5

	// CodeAlreadyExists means that the entity that the client attempted to
	// create already exists.
	// HTTP Mapping: 409 Conflict
	CodeAlreadyExists Code = 6

	// CodePermissionDenied means the caller does not have permission to
	// execute the specified operation.
	// HTTP Mapping: 403 Forbidden
	CodePermissionDenied Code = 7

	// CodeTooManyRequests means there are too many requests for some resource.
	// HTTP Mapping: 429 Too Many Requests
	CodeTooManyRequests Code = 8

	// CodeFailedPrecondition means the operation was rejected because the
	// system is not in a state required for the operation's execution.
	// HTTP Mapping: 400 Bad Request
	CodeFailedPrecondition Code = 9

	// CodeConflict means there were conflicts between concurrent operation
	// requests.
	// HTTP Mapping: 409 Conflict
	CodeConflict Code = 10

	// CodeOutOfRange means the operation was attempted past the valid range.
	// HTTP Mapping: 400 Bad Request
	CodeOutOfRange Code = 11

	// CodeUnimplemented means the operation is defined but not implemented.
	// HTTP Mapping: 501 Not Implemented
	CodeUnimplemented Code = 12

	// CodeInternal means some invariants expected by the underlying
	// system have been broken.
	// HTTP Mapping: 500 Internal Server Error
	CodeInternal Code = 13

	// CodeUnavailable means the service is currently unavailable.
	// HTTP Mapping: 503 Service Unavailable
	CodeUnavailable Code = 14

	// CodeIllegalState means illegal data found in datastore, unrecoverable
	// data loss or corruption and so on.
	// HTTP Mapping: 500 Internal Server Error
	CodeIllegalState Code = 15

	// CodeUnauthenticated means the request does not have valid authentication
	// credentials for the operation.
	// HTTP Mapping: 401 Unauthorized
	CodeUnauthenticated Code = 16

	// CodeIllegalArg means the arguments passed to an operation within the
	// program are illegal.
	// HTTP Mapping: 500 Internal Server Error
	CodeIllegalArg Code = 29

	// CodeUnauthorized means a user's authorization expired.
	// HTTP Mapping: 401 Unauthorized
	CodeUnauthorized Code = 30
)

type codeInfo struct {
	name        string
	description string
}

var codeRegistry = map[Code]codeInfo{
	CodeOK:                 {"OK", "ok"},
	CodeCancelled:          {"CANCELLED", "cancelled"},
	CodeUnknown:            {"UNKNOWN", "unknown"},
	CodeIllegalInput:       {"ILLEGAL_INPUT", "illegal input"},
	CodeTimeout:            {"TIMEOUT", "timeout"},
	CodeNotFound:           {"NOT_FOUND", "not found"},
	CodeAlreadyExists:      {"ALREADY_EXISTS", "already exists"},
	CodePermissionDenied:   {"PERMISSION_DENIED", "permission denied"},
	CodeTooManyRequests:    {"TOO_MANY_REQUESTS", "too many requests"},
	CodeFailedPrecondition: {"FAILED_PRECONDITION", "failed precondition"},
	CodeConflict:           {"CONFLICT", "conflict"},
	CodeOutOfRange:         {"OUT_OF_RANGE", "out of range"},
	CodeUnimplemented:      {"UNIMPLEMENTED", "unimplemented"},
	CodeInternal:           {"INTERNAL", "internal"},
	CodeUnavailable:        {"UNAVAILABLE", "unavailable"},
	CodeIllegalState:       {"ILLEGAL_STATE", "illegal state"},
	CodeUnauthenticated:    {"UNAUTHENTICATED", "unauthenticated"},
	CodeIllegalArg:         {"ILLEGAL_ARG", "illegal arg"},
	CodeUnauthorized:       {"UNAUTHORIZED", "unauthorized"},
}

// AllCodes returns every Code defined by this package.
func AllCodes() []Code {
	out := make([]Code, 0, len(codeRegistry))
	for c := range codeRegistry {
		out = append(out, c)
	}
	return out
}

// Name returns the canonical upper-snake-case name (e.g. "INTERNAL").
// Returns "" for unknown codes.
func (c Code) Name() string {
	if info, ok := codeRegistry[c]; ok {
		return info.name
	}
	return ""
}

// Description returns the human-readable description of the code.
func (c Code) Description() string {
	if info, ok := codeRegistry[c]; ok {
		return info.description
	}
	return ""
}

// Value returns the underlying integer code.
func (c Code) Value() int {
	return int(c)
}

func (c Code) String() string {
	return fmt.Sprintf("%s(%d)", c.Name(), int(c))
}
