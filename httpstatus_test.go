package apperror

import "testing"

func TestHTTPStatusForCode(t *testing.T) {
	cases := []struct {
		code int
		want HTTPStatus
	}{
		{200, StatusOK},
		{400, StatusBadRequest},
		{401, StatusUnauthorized},
		{403, StatusForbidden},
		{404, StatusNotFound},
		{409, StatusConflict},
		{429, StatusTooManyRequests},
		{499, StatusClientClosedRequest},
		{500, StatusInternalServerError},
		{501, StatusNotImplemented},
		{503, StatusServiceUnavailable},
		{504, StatusTimeout},
	}
	for _, c := range cases {
		got, ok := HTTPStatusForCode(c.code)
		if !ok || got != c.want {
			t.Errorf("HTTPStatusForCode(%d) = (%v, %v), want (%v, true)", c.code, got, ok, c.want)
		}
	}

	for _, code := range []int{100, 300, 600, -1} {
		if _, ok := HTTPStatusForCode(code); ok {
			t.Errorf("HTTPStatusForCode(%d) should report not-known", code)
		}
	}
}

func TestHTTPStatusString(t *testing.T) {
	cases := map[HTTPStatus]string{
		StatusOK:              "HTTPStatus.OK(200)",
		StatusBadRequest:      "HTTPStatus.BAD_REQUEST(400)",
		StatusUnauthorized:    "HTTPStatus.UNAUTHORIZED(401)",
		StatusForbidden:       "HTTPStatus.FORBIDDEN(403)",
		StatusNotFound:        "HTTPStatus.NOT_FOUND(404)",
		StatusConflict:        "HTTPStatus.CONFLICT(409)",
		StatusTooManyRequests: "HTTPStatus.TOO_MANY_REQUESTS(429)",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("String(%d) = %q, want %q", int(s), got, want)
		}
	}
}
