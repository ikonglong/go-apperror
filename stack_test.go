package apperror

import (
	"strings"
	"testing"
)

func TestNewAppErrorCapturesStackAtCaller(t *testing.T) {
	err := NewNotFound("user.lookup") // origin frame: this line

	st := err.StackTrace()
	if len(st) == 0 {
		t.Fatal("StackTrace() is empty; expected the construction site to be captured")
	}

	lines := st.Strings()
	if len(lines) == 0 {
		t.Fatal("Strings() returned no frames")
	}

	// The innermost recorded frame must be this test function — i.e. the
	// skip count dropped the package-internal construction frames and the
	// factory, landing on the actual caller.
	if !strings.Contains(lines[0], "TestNewAppErrorCapturesStackAtCaller") {
		t.Errorf("top frame = %q; want it to name the calling function", lines[0])
	}

	// And none of apperror's own construction machinery should leak in.
	for _, l := range lines {
		if strings.Contains(l, "apperror.callers") ||
			strings.Contains(l, "apperror.newAppError") ||
			strings.Contains(l, "apperror.NewNotFound") {
			t.Errorf("internal construction frame leaked into the trace: %q", l)
		}
	}
}

func TestStackTraceStringsAreSingleLine(t *testing.T) {
	err := NewInternal("user.lookup")
	for _, l := range err.StackTrace().Strings() {
		if strings.ContainsAny(l, "\n\r") {
			t.Errorf("stack entry contains a newline, which would split log records: %q", l)
		}
	}
}

func TestZeroStackTraceResolvesEmpty(t *testing.T) {
	var st StackTrace
	if st.Frames() != nil {
		t.Error("Frames() of an empty trace should be nil")
	}
	if st.Strings() != nil {
		t.Error("Strings() of an empty trace should be nil")
	}
}
