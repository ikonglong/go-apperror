package apperror

import "fmt"

// HTTPStatus is the set of HTTP status codes used by this library.
type HTTPStatus int

const (
	StatusOK                  HTTPStatus = 200
	StatusBadRequest          HTTPStatus = 400
	StatusUnauthorized        HTTPStatus = 401
	StatusForbidden           HTTPStatus = 403
	StatusNotFound            HTTPStatus = 404
	StatusConflict            HTTPStatus = 409
	StatusTooManyRequests     HTTPStatus = 429
	StatusClientClosedRequest HTTPStatus = 499
	StatusInternalServerError HTTPStatus = 500
	StatusNotImplemented      HTTPStatus = 501
	StatusServiceUnavailable  HTTPStatus = 503
	StatusTimeout             HTTPStatus = 504
)

var httpStatusName = map[HTTPStatus]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "BAD_REQUEST",
	StatusUnauthorized:        "UNAUTHORIZED",
	StatusForbidden:           "FORBIDDEN",
	StatusNotFound:            "NOT_FOUND",
	StatusConflict:            "CONFLICT",
	StatusTooManyRequests:     "TOO_MANY_REQUESTS",
	StatusClientClosedRequest: "CLIENT_CLOSED_REQUEST",
	StatusInternalServerError: "INTERNAL_SERVER_ERROR",
	StatusNotImplemented:      "NOT_IMPLEMENTED",
	StatusServiceUnavailable:  "SERVICE_UNAVAILABLE",
	StatusTimeout:             "TIMEOUT",
}

// Name returns the canonical short name for the status (e.g. "BAD_REQUEST").
// Returns "" for unknown statuses.
func (s HTTPStatus) Name() string {
	return httpStatusName[s]
}

// Value returns the underlying integer status code.
func (s HTTPStatus) Value() int {
	return int(s)
}

// IsKnown reports whether s is one of the statuses defined in this package.
func (s HTTPStatus) IsKnown() bool {
	_, ok := httpStatusName[s]
	return ok
}

func (s HTTPStatus) String() string {
	return fmt.Sprintf("HTTPStatus.%s(%d)", s.Name(), int(s))
}

// HTTPStatusForCode returns the HTTPStatus for the given numeric status code.
// The second return value is false if the code is not a known status.
func HTTPStatusForCode(code int) (HTTPStatus, bool) {
	s := HTTPStatus(code)
	if _, ok := httpStatusName[s]; ok {
		return s, true
	}
	return 0, false
}
