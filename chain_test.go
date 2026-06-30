package apperror

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestFlatMessage_Nil(t *testing.T) {
	if got := FlatMessage(nil); got != "" {
		t.Errorf("FlatMessage(nil) = %q, want \"\"", got)
	}
}

func TestFlatMessage_SingleAppError(t *testing.T) {
	err := NewNotFound("user.lookup", WithMessage("user not found"))
	want := "user not found"
	if got := FlatMessage(err); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}

func TestFlatMessage_AllAppErrorChain(t *testing.T) {
	inner := NewIllegalState("datastore.read", WithMessage("datastore corrupt"))
	mid := NewInternal("repo.load", WithMessage("repo load failed"), WithCause(inner))
	top := NewUnavailable("user.lookup", WithMessage("user-service degraded"), WithCause(mid))

	want := "user-service degraded -> repo load failed -> datastore corrupt"
	if got := FlatMessage(top); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}

func TestFlatMessage_AppErrorWrapsFmtWrapAndBare_SuffixSubtracted(t *testing.T) {
	leaf := errors.New("connection reset by peer")
	wrap := fmt.Errorf("query users table: %w", leaf)
	top := NewIllegalState("user.lookup", WithMessage("failed to load user"), WithCause(wrap))

	want := "failed to load user -> query users table -> connection reset by peer"
	if got := FlatMessage(top); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}

// TestFlatMessage_UnknownWrapNonColonSeparator_ConservativeFallback documents
// what happens when a wrapper joins its message with something other than the
// ": " convention fmt.Errorf uses. ownMessage cannot safely strip the inner
// text, so the outer layer keeps its full Error() string and the inner
// message appears twice. That's the intentional conservative behavior — see
// ownMessage doc.
func TestFlatMessage_UnknownWrapNonColonSeparator_ConservativeFallback(t *testing.T) {
	leaf := errors.New("inner")
	wrap := fmt.Errorf("ctx; %w", leaf)
	top := NewIllegalState("test.evt", WithMessage("outer"), WithCause(wrap))

	want := "outer -> ctx; inner -> inner"
	if got := FlatMessage(top); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}

func TestFlatMessage_RemoteErrorAtTop(t *testing.T) {
	r := &RemoteError{
		Service:   "user-service",
		Operation: "GetUser",
		Response:  &Response{StatusCode: 503},
	}
	// FlatMessage emits the full RemoteError forensic line so DevOps can
	// see service/operation/status/bodyCode/retryAfter inline.
	if got := FlatMessage(r); got != r.Error() {
		t.Errorf("FlatMessage = %q, want r.Error() %q", got, r.Error())
	}
}

func TestFlatMessage_AppErrorWrapsRemoteError(t *testing.T) {
	r := &RemoteError{
		Service:    "user-service",
		Operation:  "GetUser",
		Response:   &Response{StatusCode: 503},
		RetryAfter: 30 * time.Second,
	}
	top := NewInternal("user.lookup", WithMessage("user lookup failed"), WithCause(r))

	want := "user lookup failed -> " + r.Error()
	if got := FlatMessage(top); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}
