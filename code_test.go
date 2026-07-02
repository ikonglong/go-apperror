package apperror

import "testing"

func TestCodeNoDuplicateValue(t *testing.T) {
	seen := map[int]Code{}
	for _, c := range AllCodes() {
		if dup, ok := seen[c.Value()]; ok {
			t.Fatalf("Code value duplication found: %s and %s share value %d",
				dup.Name(), c.Name(), c.Value())
		}
		seen[c.Value()] = c
	}
}

func TestCodeStringRepresentation(t *testing.T) {
	cases := map[Code]string{
		CodeOK:               "OK(0)",
		CodeIllegalInput:     "ILLEGAL_INPUT(3)",
		CodePermissionDenied: "PERMISSION_DENIED(7)",
		CodeTooManyRequests:  "TOO_MANY_REQUESTS(8)",
		CodeUnauthenticated:  "UNAUTHENTICATED(16)",
		CodeNotFound:         "NOT_FOUND(5)",
	}
	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("Code(%d).String() = %q, want %q", int(c), got, want)
		}
	}
}

func TestCodeForHTTPStatus(t *testing.T) {
	cases := map[int]Code{
		200: CodeOK,
		400: CodeIllegalInput,
		401: CodeUnauthenticated,
		403: CodePermissionDenied,
		404: CodeNotFound,
		409: CodeAlreadyExists,
		429: CodeTooManyRequests,
		499: CodeCancelled,
		500: CodeInternal,
		501: CodeUnimplemented,
		503: CodeUnavailable,
		504: CodeTimeout,
		600: CodeUnknown,
	}
	for status, want := range cases {
		if got := CodeFor(status); got != want {
			t.Errorf("CodeFor(%d) = %s, want %s", status, got, want)
		}
	}
}

func TestHTTPStatusForCodeMapping(t *testing.T) {
	cases := map[Code]HTTPStatus{
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
	for code, want := range cases {
		got, ok := HTTPStatusFor(code)
		if !ok || got != want {
			t.Errorf("HTTPStatusFor(%s) = (%s, %v), want (%s, true)", code, got, ok, want)
		}
	}
}
