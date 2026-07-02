package apperror

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// --- RemoteErrorResp tests ---

func newRemoteErrorRespFixture() *RemoteErrorResp {
	return &RemoteErrorResp{
		Request:     &Request{Method: "GET", URL: "/users/42"},
		Response:    &Response{StatusCode: 503, Body: []byte(`{"code":"DEGRADED"}`)},
		BodyCode:    "DEGRADED",
		BodyMessage: "service degraded",
		RetryAfter:  30 * time.Second,
	}
}

func TestRemoteErrorRespStatusCode(t *testing.T) {
	r := newRemoteErrorRespFixture()
	if r.StatusCode() != 503 {
		t.Errorf("StatusCode() = %d, want 503", r.StatusCode())
	}
}

func TestRemoteErrorRespStringFormat(t *testing.T) {
	r := newRemoteErrorRespFixture()
	got := r.String()

	if !strings.HasPrefix(got, "RemoteErrorResp(") {
		t.Errorf("String() should start with RemoteErrorResp(, got %q", got)
	}

	for _, want := range []string{
		"status=503",
		"bodyCode=DEGRADED",
		"retryAfter=30s",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("String() missing %q\ngot: %s", want, got)
		}
	}
}

// RemoteErrorResp is a DTO, not an error — it does not implement error.
func TestRemoteErrorRespIsNotAnError(t *testing.T) {
	r := newRemoteErrorRespFixture()
	// At compile time, *RemoteErrorResp does not satisfy error.
	// This test just guards the intent.
	_ = r.String() // has String, but no Error()
}

// --- RemoteError tests ---

func newRemoteErrorFixture(opts ...RemoteOption) *RemoteError {
	return NewRemoteUnavailable("user-service.GetUser", opts...)
}

func TestRemoteErrorConstruction(t *testing.T) {
	e := newRemoteErrorFixture()
	if e.Code() != CodeUnavailable {
		t.Errorf("Code() = %s, want %s", e.Code(), CodeUnavailable)
	}
	if e.Event() != "user-service.GetUser" {
		t.Errorf("Event() = %q, want %q", e.Event(), "user-service.GetUser")
	}
	if e.Message() != CodeUnavailable.Description() {
		t.Errorf("Message() = %q, want %q", e.Message(), CodeUnavailable.Description())
	}
	if e.Cause() != nil {
		t.Errorf("Cause() = %v, want nil", e.Cause())
	}
	if e.ErrResp() != nil {
		t.Errorf("ErrResp() = %v, want nil", e.ErrResp())
	}
}

func TestRemoteErrorWithErrResp(t *testing.T) {
	resp := newRemoteErrorRespFixture()
	e := NewRemoteUnavailable("user-service.GetUser", WithErrResp(resp))
	if e.ErrResp() != resp {
		t.Error("ErrResp() should return the attached RemoteErrorResp")
	}
	if e.Cause() != nil {
		t.Error("Cause() should be nil when errResp is set")
	}
	if errors.Unwrap(e) != nil {
		t.Error("Unwrap() should return nil when errResp is set (leaf)")
	}
}

func TestRemoteErrorWithCause(t *testing.T) {
	connErr := errors.New("connection refused")
	e := NewRemoteUnavailable("user-service.GetUser",
		func(re *RemoteError) { re.cause = connErr })
	if !errors.Is(e.Cause(), connErr) {
		t.Error("Cause() should return the transport error")
	}
	if e.ErrResp() != nil {
		t.Error("ErrResp() should be nil when cause is set")
	}
	if !errors.Is(errors.Unwrap(e), connErr) {
		t.Error("Unwrap() should return the cause")
	}
	if !errors.Is(e, connErr) {
		t.Error("errors.Is should find the wrapped cause")
	}
}

func TestRemoteErrorStringFormat(t *testing.T) {
	e := newRemoteErrorFixture()
	got := e.String()

	if !strings.HasPrefix(got, "RemoteError(") {
		t.Errorf("String() should start with RemoteError(, got %q", got)
	}

	for _, want := range []string{
		"code=UNAVAILABLE(14)",
		"event=user-service.GetUser",
		"errResp=None",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("String() missing %q\ngot: %s", want, got)
		}
	}
}

func TestRemoteErrorStringFormatWithErrResp(t *testing.T) {
	resp := newRemoteErrorRespFixture()
	e := NewRemoteUnavailable("user-service.GetUser", WithErrResp(resp))
	got := e.String()

	for _, want := range []string{
		"code=UNAVAILABLE(14)",
		"event=user-service.GetUser",
		"RemoteErrorResp(status=503, bodyCode=DEGRADED, retryAfter=30s)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("String() missing %q\ngot: %s", want, got)
		}
	}
}

func TestRemoteErrorAddNote(t *testing.T) {
	e := newRemoteErrorFixture()
	e.AddNote("during checkout")
	if !strings.HasPrefix(e.Message(), "during checkout -> ") {
		t.Errorf("AddNote should prepend context; got %q", e.Message())
	}
}

func TestRemoteErrorStackTrace(t *testing.T) {
	e := newRemoteErrorFixture()
	st := e.StackTrace()
	if len(st) == 0 {
		t.Error("StackTrace should be non-empty")
	}
	lines := st.Strings()
	if len(lines) == 0 {
		t.Error("StackTrace.Strings() should be non-empty")
	}
}

func TestRemoteErrorEventRequired(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty event")
		}
	}()
	NewRemoteUnavailable("")
}

// A bare RemoteError carries no AppError; errors.As to *AppError should
// not succeed via a lone RemoteError.
func TestRemoteErrorNotAnAppError(t *testing.T) {
	e := newRemoteErrorFixture()
	var ae *AppError
	if errors.As(error(e), &ae) {
		t.Error("errors.As(remoteErr, &*AppError) should NOT succeed")
	}
}

func TestRemoteErrorErrorsAsRecoversRemoteError(t *testing.T) {
	e := newRemoteErrorFixture()
	wrapped := fmt.Errorf("calling user-service: %w", e)

	var re *RemoteError
	if !errors.As(wrapped, &re) {
		t.Fatal("errors.As should find *RemoteError in chain")
	}
	if re != e {
		t.Error("recovered RemoteError is not the original instance")
	}
}

func TestRemoteErrorWrappedByAppError(t *testing.T) {
	resp := newRemoteErrorRespFixture()
	remoteErr := NewRemoteUnavailable("user-service.GetUser", WithErrResp(resp))
	appErr := NewInternal("user.lookup",
		WithMessage("user lookup failed"), WithCause(remoteErr))

	// AppError wraps RemoteError as cause
	var re *RemoteError
	if !errors.As(error(appErr), &re) || re != remoteErr {
		t.Error("errors.As should recover RemoteError cause from AppError")
	}

	// FlatMessage includes both layers
	msg := FlatMessage(appErr)
	if !strings.Contains(msg, "user lookup failed") {
		t.Errorf("FlatMessage missing AppError message: %s", msg)
	}
	if !strings.Contains(msg, "unavailable") {
		t.Errorf("FlatMessage missing RemoteError message: %s", msg)
	}
}
