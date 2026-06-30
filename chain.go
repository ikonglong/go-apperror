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

// --- Canonical is intentionally commented out (kept here for possible revival) ---
//
// Rationale: the codebase's convention has driven adapters / the application
// layer translate *RemoteError into *AppError before it propagates further
// up. Under that convention, the boundary handler only encounters *AppError
// or non-apperror wraps, and stdlib errors.As(err, &appErr) is sufficient.
// Canonical's only value-add over errors.As is handling the
// RemoteError-leaf-with-Canonical-sibling case, which doesn't arise.
//
// Restore this (and its tests in chain_test.go) if a downstream app adopts
// the alternative convention of propagating *RemoteError all the way to the
// boundary.
//
// // Canonical walks the chain and returns the AppError that should drive
// // wire-level decisions (HTTP status, public response body).
// //
// // Lookup order at each layer:
// //   - *AppError:    return it
// //   - *RemoteError: return e.Canonical (may be nil — RemoteError is a leaf)
// //
// // Returns nil if no canonical view exists. Callers at the interface
// // boundary typically fall back to apperror.NewInternal(...) in that
// // case.
// func Canonical(err error) *AppError {
// 	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
// 		// We are deliberately inspecting the concrete type of THIS layer,
// 		// not the chain as a whole — errors.As would skip past the layer
// 		// we care about. Disable errorlint here.
// 		switch e := cur.(type) { //nolint:errorlint
// 		case *AppError:
// 			return e
// 		case *RemoteError:
// 			return e.Canonical
// 		}
// 		// Any other type (wrap-shaped like fmt.Errorf("%w"), or a bare
// 		// leaf) carries no canonical view itself — keep walking via the
// 		// for-loop's errors.Unwrap step.
// 	}
// 	return nil
// }

// ownMessage returns the part of err.Error() that originated at THIS
// layer (excluding any nested cause's message). Per type:
//   - *AppError:    e.Message()
//   - *RemoteError: e.Error() — keeps the full forensic line
//     (service/operation/status/bodyCode/retryAfter/canonical) inline so
//     a one-line FlatMessage still carries the remote-side signals
//     DevOps needs when triaging.
//   - other:        see inline notes below.
func ownMessage(err error) string {
	// Inspecting the concrete type of THIS layer is intentional — the
	// caller walks the chain with errors.Unwrap. Disable errorlint here.
	switch e := err.(type) { //nolint:errorlint
	case *AppError:
		return e.Message()
	case *RemoteError:
		return e.Error()
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
