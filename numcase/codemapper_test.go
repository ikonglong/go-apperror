package numcase

import (
	"testing"

	apperror "github.com/ikonglong/go-apperror"
)

func TestCodeMapperHasMappingFor(t *testing.T) {
	cm := NewDefaultCodeMapper()
	for _, c := range []apperror.Code{
		apperror.CodeIllegalInput, apperror.CodeTimeout, apperror.CodeNotFound,
	} {
		if !cm.HasMappingFor(c) {
			t.Errorf("expected mapping for %s", c)
		}
	}
	for _, c := range []apperror.Code{
		apperror.CodeOK, apperror.CodeCancelled, apperror.CodeUnknown,
	} {
		if cm.HasMappingFor(c) {
			t.Errorf("did not expect mapping for %s", c)
		}
	}
}

func TestCodeMapperCaseCodeSegmentFor(t *testing.T) {
	cm := NewDefaultCodeMapper()

	cases := map[apperror.Code]NumRange{
		apperror.CodeIllegalInput: MustNew(1, 50),
		apperror.CodeTimeout:      MustNew(51, 100),
		apperror.CodeNotFound:     MustNew(101, 150),
	}
	for c, want := range cases {
		got, ok := cm.CaseCodeSegmentFor(c)
		if !ok || !got.Equal(want) {
			t.Errorf("CaseCodeSegmentFor(%s) = (%s, %v), want (%s, true)", c, got, ok, want)
		}
	}

	for _, c := range []apperror.Code{apperror.CodeOK, apperror.CodeCancelled} {
		if _, ok := cm.CaseCodeSegmentFor(c); ok {
			t.Errorf("CaseCodeSegmentFor(%s) should be missing", c)
		}
	}
}

func TestCodeMapperCaseCodeSegments(t *testing.T) {
	cm := NewDefaultCodeMapper()
	segs := cm.CaseCodeSegments()
	if len(segs) != 11 {
		t.Errorf("len(CaseCodeSegments()) = %d, want 11", len(segs))
	}
	for i, a := range segs {
		for _, b := range segs[i+1:] {
			if a.Overlap(b) {
				t.Errorf("segments overlap: %s and %s", a, b)
			}
		}
	}
}

func TestCodeMapperMappings(t *testing.T) {
	cm := NewDefaultCodeMapper()
	m := cm.Mappings()
	if len(m) != 11 {
		t.Errorf("len(Mappings()) = %d, want 11", len(m))
	}
	cases := map[apperror.Code]NumRange{
		apperror.CodeIllegalInput:       MustNew(1, 50),
		apperror.CodeTimeout:            MustNew(51, 100),
		apperror.CodeNotFound:           MustNew(101, 150),
		apperror.CodeAlreadyExists:      MustNew(151, 200),
		apperror.CodePermissionDenied:   MustNew(201, 250),
		apperror.CodeTooManyRequests:    MustNew(251, 300),
		apperror.CodeFailedPrecondition: MustNew(301, 350),
		apperror.CodeConflict:           MustNew(351, 400),
		apperror.CodeOutOfRange:         MustNew(401, 450),
		apperror.CodeInternal:           MustNew(451, 500),
		apperror.CodeIllegalState:       MustNew(501, 550),
	}
	for c, want := range cases {
		got, ok := m[c]
		if !ok || !got.Equal(want) {
			t.Errorf("Mappings()[%s] = (%s, %v), want (%s, true)", c, got, ok, want)
		}
	}
}

func TestCodeMapperString(t *testing.T) {
	cm := NewDefaultCodeMapper()
	if cm.String() == "" {
		t.Error("String() should not be empty")
	}
}
