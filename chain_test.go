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
		Canonical: NewUnavailable("user-service.GetUser", WithMessage("user-service degraded")),
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
		Canonical:  NewUnavailable("user-service.GetUser", WithMessage("user-service degraded")),
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

// TestFlatMessage_RemoteErrorNilCanonical_NoPanic guards against a
// regression where a RemoteError missing its Canonical (violating the
// documented convention) would panic during FlatMessage. The own-message
// path on the main branch is the same as the well-formed case
// (r.Error()), so this test exists purely as a nil-safety regression.
func TestFlatMessage_RemoteErrorNilCanonical_NoPanic(t *testing.T) {
	r := &RemoteError{
		Service:   "user-service",
		Operation: "GetUser",
		Response:  &Response{StatusCode: 503},
	}
	if got := FlatMessage(r); got != r.Error() {
		t.Errorf("FlatMessage = %q, want r.Error() %q", got, r.Error())
	}
}

// --- Canonical tests are intentionally commented out, paired with the
// commented-out Canonical function in chain.go. Restore both blocks if the
// codebase adopts the convention of letting *RemoteError reach the
// boundary handler. ---
//
// func TestCanonical_Nil(t *testing.T) {
// 	if got := Canonical(nil); got != nil {
// 		t.Errorf("Canonical(nil) = %v, want nil", got)
// 	}
// }
//
// func TestCanonical_AppErrorAtTop(t *testing.T) {
// 	e := NewNotFound("test.evt", WithMessage("missing"))
// 	if got := Canonical(e); got != e {
// 		t.Errorf("Canonical = %v, want %v", got, e)
// 	}
// }
//
// func TestCanonical_RemoteErrorAtTop_ReturnsCanonicalField(t *testing.T) {
// 	c := NewUnavailable("svc.op", WithMessage("downstream"))
// 	r := &RemoteError{
// 		Canonical: c,
// 		Service:   "svc",
// 		Operation: "op",
// 		Response:  &Response{StatusCode: 503},
// 	}
// 	if got := Canonical(r); got != c {
// 		t.Errorf("Canonical = %v, want %v", got, c)
// 	}
// }
//
// // TestCanonical_AppErrorWrapsRemoteError_OuterWins documents the wire-mapping
// // semantics: the outermost AppError is the canonical view. Wrapping a
// // RemoteError(canonical=Unavailable) with NewInternal means the caller
// // reclassified this failure, and Canonical respects that.
// func TestCanonical_AppErrorWrapsRemoteError_OuterWins(t *testing.T) {
// 	r := &RemoteError{
// 		Canonical: NewUnavailable("svc.op", WithMessage("downstream")),
// 		Service:   "svc",
// 		Operation: "op",
// 		Response:  &Response{StatusCode: 503},
// 	}
// 	outer := NewInternal("test.evt", WithMessage("rephrased"), WithCause(r))
// 	got := Canonical(outer)
// 	if got != outer {
// 		t.Errorf("Canonical = %v, want outer AppError", got)
// 	}
// 	if got.Code() != CodeInternal {
// 		t.Errorf("Canonical.Code() = %v, want CodeInternal", got.Code())
// 	}
// }
//
// func TestCanonical_FmtWrapAroundAppError_FindsInner(t *testing.T) {
// 	inner := NewNotFound("test.evt", WithMessage("missing"))
// 	wrap := fmt.Errorf("ctx: %w", inner)
// 	if got := Canonical(wrap); got != inner {
// 		t.Errorf("Canonical = %v, want inner AppError", got)
// 	}
// }
//
// func TestCanonical_NoAppErrorOrRemoteInChain_ReturnsNil(t *testing.T) {
// 	leaf := errors.New("bare")
// 	wrap := fmt.Errorf("ctx: %w", leaf)
// 	if got := Canonical(wrap); got != nil {
// 		t.Errorf("Canonical = %v, want nil", got)
// 	}
// }
//
// // TestCanonical_BareLeafNoUnwrap_ReturnsNil makes the single-layer
// // degenerate case explicit: an err that is neither *AppError nor
// // *RemoteError AND doesn't implement Unwrap. The switch falls through;
// // errors.Unwrap returns nil; the loop exits and Canonical returns nil.
// func TestCanonical_BareLeafNoUnwrap_ReturnsNil(t *testing.T) {
// 	if got := Canonical(errors.New("bare")); got != nil {
// 		t.Errorf("Canonical = %v, want nil", got)
// 	}
// }
//
// func TestCanonical_RemoteErrorWithNilCanonical_ReturnsNil(t *testing.T) {
// 	r := &RemoteError{
// 		Service:   "svc",
// 		Operation: "op",
// 		Response:  &Response{StatusCode: 503},
// 	}
// 	if got := Canonical(r); got != nil {
// 		t.Errorf("Canonical = %v, want nil", got)
// 	}
// }
