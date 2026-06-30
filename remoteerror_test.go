package apperror

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func newRemoteErrorFixture() *RemoteError {
	return &RemoteError{
		Service:     "user-service",
		Operation:   "GetUser",
		Request:     &Request{Method: "GET", URL: "/users/42"},
		Response:    &Response{StatusCode: 503, Body: []byte(`{"code":"DEGRADED"}`)},
		BodyCode:    "DEGRADED",
		BodyMessage: "service degraded",
		RetryAfter:  30 * time.Second,
	}
}

func TestRemoteErrorStatusCode(t *testing.T) {
	r := newRemoteErrorFixture()
	if r.StatusCode() != 503 {
		t.Errorf("StatusCode() = %d, want 503", r.StatusCode())
	}
}

// RemoteError is a leaf error: it has no cause and Unwrap returns nil.
func TestRemoteErrorIsLeaf(t *testing.T) {
	r := newRemoteErrorFixture()
	if errors.Unwrap(r) != nil {
		t.Errorf("errors.Unwrap(*RemoteError) = %v, want nil", errors.Unwrap(r))
	}
}

// A bare RemoteError carries no AppError: the canonical view lives on the
// *AppError that wraps it as a cause, not on the RemoteError itself. So
// errors.As to *AppError must NOT succeed via a lone RemoteError.
func TestRemoteErrorCarriesNoAppError(t *testing.T) {
	r := newRemoteErrorFixture()
	var ae *AppError
	if errors.As(error(r), &ae) {
		t.Error("errors.As(remoteErr, &*AppError) should NOT succeed; " +
			"a RemoteError carries no canonical AppError of its own")
	}
}

func TestRemoteErrorErrorsAsRecoversRemoteError(t *testing.T) {
	r := newRemoteErrorFixture()
	wrapped := fmt.Errorf("calling user-service: %w", r) // pretend a caller wraps

	var re *RemoteError
	if !errors.As(wrapped, &re) {
		t.Fatal("errors.As should find *RemoteError in chain")
	}
	if re != r {
		t.Error("recovered RemoteError is not the original instance")
	}
}

func TestRemoteErrorStringFormat(t *testing.T) {
	r := newRemoteErrorFixture()
	got := r.Error()

	if !strings.HasPrefix(got, "RemoteError(") {
		t.Errorf("Error() should start with RemoteError(, got %q", got)
	}

	for _, want := range []string{
		"service=user-service",
		"operation=GetUser",
		"status=503",
		"bodyCode=DEGRADED",
		"retryAfter=30s",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() missing %q\ngot: %s", want, got)
		}
	}
}

func TestRemoteErrorEvent(t *testing.T) {
	r := newRemoteErrorFixture()
	if got, want := r.Event(), "user-service.GetUser"; got != want {
		t.Errorf("Event() = %q, want %q", got, want)
	}
}

// The canonical AppError wraps the RemoteError as its cause, and AddNote on
// that AppError enriches its message without disturbing the RemoteError or
// breaking errors.As recovery of the cause.
func TestRemoteErrorWrappedByCanonical(t *testing.T) {
	r := newRemoteErrorFixture()
	canonical := NewUnavailable("user-service.GetUser",
		WithMessage("user-service call failed"), WithCause(r))
	canonical.AddNote("during checkout")

	if !strings.HasPrefix(canonical.Message(), "during checkout -> ") {
		t.Errorf("AddNote should prepend context; got %q", canonical.Message())
	}
	var re *RemoteError
	if !errors.As(error(canonical), &re) || re != r {
		t.Error("errors.As should recover the wrapped RemoteError cause")
	}
}
