package apperror

var codeToHTTPStatus = map[Code]HTTPStatus{
	CodeOK:                 StatusOK,
	CodeIllegalInput:       StatusBadRequest,
	CodeFailedPrecondition: StatusBadRequest,
	CodeOutOfRange:         StatusBadRequest,
	CodeUnauthenticated:    StatusUnauthorized,
	CodePermissionDenied:   StatusForbidden,
	CodeNotFound:           StatusNotFound,
	CodeConflict:           StatusConflict,
	CodeAlreadyExists:      StatusConflict,
	CodeTooManyRequests:    StatusTooManyRequests,
	CodeCancelled:          StatusClientClosedRequest,
	CodeIllegalState:       StatusInternalServerError,
	CodeUnknown:            StatusInternalServerError,
	CodeInternal:           StatusInternalServerError,
	CodeUnimplemented:      StatusNotImplemented,
	CodeUnavailable:        StatusServiceUnavailable,
	CodeTimeout:            StatusTimeout,
	CodeUnauthorized:       StatusUnauthorized,
	CodeIllegalArg:         StatusInternalServerError,
}

// HTTPStatusFor returns the HTTPStatus that the given Code maps to.
// The second return value is false when no mapping is defined.
func HTTPStatusFor(code Code) (HTTPStatus, bool) {
	s, ok := codeToHTTPStatus[code]
	return s, ok
}

var httpStatusToCode = map[HTTPStatus]Code{
	StatusOK:                  CodeOK,
	StatusBadRequest:          CodeIllegalInput,
	StatusUnauthorized:        CodeUnauthenticated,
	StatusForbidden:           CodePermissionDenied,
	StatusNotFound:            CodeNotFound,
	StatusConflict:            CodeAlreadyExists,
	StatusTooManyRequests:     CodeTooManyRequests,
	StatusClientClosedRequest: CodeCancelled,
	StatusInternalServerError: CodeInternal,
	StatusNotImplemented:      CodeUnimplemented,
	StatusServiceUnavailable:  CodeUnavailable,
	StatusTimeout:             CodeTimeout,
}

// CodeFor returns the Code that the given HTTP status code maps to.
// Unknown statuses map to CodeUnknown.
func CodeFor(httpStatus int) Code {
	if c, ok := httpStatusToCode[HTTPStatus(httpStatus)]; ok {
		return c
	}
	return CodeUnknown
}
