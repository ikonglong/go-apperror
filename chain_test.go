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
	// WithErrResp sets errResp without setting cause → RemoteError is a leaf
	// (Unwrap returns nil). FlatMessage sees only the own-message.
	e := NewRemoteUnavailable("UserService.GetUser",
		WithErrResp(&RemoteErrorResp{Response: &Response{StatusCode: 503}}))
	// RemoteError.ownMessage returns e.Message() (like AppError), not the
	// full debug String(); FlatMessage sees only the human-readable message.
	want := e.Message()
	if got := FlatMessage(e); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}

func TestFlatMessage_RemoteErrorWithErrRespInChain(t *testing.T) {
	resp := &RemoteErrorResp{
		Response:   &Response{StatusCode: 503},
		RetryAfter: 30 * time.Second,
	}
	remoteErr := NewRemoteUnavailable("UserService.GetUser", WithErrResp(resp))
	top := NewInternal("user.lookup", WithMessage("user lookup failed"), WithCause(remoteErr))

	// FlatMessage layers: AppError.Message() -> RemoteError.Message()
	want := "user lookup failed -> " + remoteErr.Message()
	if got := FlatMessage(top); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}

func TestFlatMessage_RemoteErrorWithCauseInChain(t *testing.T) {
	connErr := errors.New("connection refused")
	remoteErr := NewRemoteUnavailable("UserService.GetUser",
		RemoteWithCause(connErr))
	top := NewInternal("user.lookup", WithMessage("user lookup failed"), WithCause(remoteErr))

	// Chain: AppError -> RemoteError -> connErr
	want := "user lookup failed -> unavailable -> connection refused"
	if got := FlatMessage(top); got != want {
		t.Errorf("FlatMessage = %q, want %q", got, want)
	}
}
