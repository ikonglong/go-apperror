package apperror

import (
	"errors"
	"strings"
)

// FlatMessage walks the error chain and returns each layer's own-message
// joined by " -> ". The human-readable summary, intended as the single
// log field that captures the full error context in one line.
//
// See ownMessage for per-type rules and the de-duplication behavior on
// fmt.Errorf("%w") wrappers.
//
// Returns "" if err is nil.
func FlatMessage(err error) string {
	if err == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		if m := ownMessage(cur); m != "" {
			parts = append(parts, m)
		}
	}
	return strings.Join(parts, " -> ")
}

// To find an *AppError in an error chain, use errors.As(err, &appErr).
// When an AppError wraps a RemoteError as its cause, this recovers the
// AppError's Code/Case/Details directly — no separate helper needed.

// ownMessage returns the part of err.Error() that originated at THIS
// layer (excluding any nested cause's message). Per type:
//   - *AppError:    e.Message()
//   - *RemoteError: e.Message()
//   - other:        see inline notes below.
func ownMessage(err error) string {
	// Inspecting the concrete type of THIS layer is intentional — the
	// caller walks the chain with errors.Unwrap. Disable errorlint here.
	switch e := err.(type) { //nolint:errorlint
	case *AppError:
		return e.Message()
	case *RemoteError:
		return e.Message()
	}

	// Wrap-shaped errors (typically fmt.Errorf("...: %w", inner)) embed
	// the inner Error() text directly into the outer string with ": " as
	// the separator. Subtract that trailing suffix so when the inner
	// layer is later emitted on its own its message doesn't appear twice.
	//
	// Best-effort: if the wrapper uses a different separator (e.g. "; "),
	// the suffix won't match and we fall through to err.Error() as-is.
	// The inner message will then appear twice in FlatMessage — the
	// correct conservative behavior for unknown wrappers.
	if inner := errors.Unwrap(err); inner != nil {
		outer := err.Error()
		suffix := ": " + inner.Error()
		if strings.HasSuffix(outer, suffix) {
			return outer[:len(outer)-len(suffix)]
		}
	}
	return err.Error()
}
