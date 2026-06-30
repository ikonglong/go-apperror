package numcase

import (
	"fmt"
	"math"
	"sort"
	"strings"

	apperror "github.com/ikonglong/go-apperror"
)

// FactoryConfig configures a CaseFactory.
type FactoryConfig struct {
	// NumDigitsForAppCode controls the zero-padded width of the app-code part
	// of an identifier. Set to 0 to omit the app-code part entirely.
	NumDigitsForAppCode int
	// NumDigitsForModuleCode is the analogous width for the module-code part.
	NumDigitsForModuleCode int
	// NumDigitsForCaseCode is the width for the case-code part. Always present.
	NumDigitsForCaseCode int
	// CodeMapper provides the case-code segments per operation status code.
	CodeMapper CodeMapper
	// AppCode is the numeric application identifier. Default 0.
	AppCode int
	// ModuleCode is the numeric module identifier. Default 0.
	ModuleCode int
}

// CaseFactory creates NumCase instances with valid identifiers given a
// FactoryConfig.
type CaseFactory struct {
	cfg           FactoryConfig
	appCodeRange  NumRange
	moduleRange   NumRange
	caseCodeRange NumRange
}

// NewCaseFactory builds a CaseFactory, validating the configuration.
func NewCaseFactory(cfg FactoryConfig) (*CaseFactory, error) {
	if cfg.NumDigitsForAppCode < 0 {
		return nil, fmt.Errorf("num_digits_for_app_code < 0")
	}
	if cfg.NumDigitsForModuleCode < 0 {
		return nil, fmt.Errorf("num_digits_for_module_code < 0")
	}
	if cfg.NumDigitsForCaseCode < 0 {
		return nil, fmt.Errorf("num_digits_for_case_code < 0")
	}
	if cfg.CodeMapper == nil {
		return nil, fmt.Errorf("code_mapper is nil")
	}
	if cfg.AppCode < 0 {
		return nil, fmt.Errorf("app_code < 0")
	}
	if cfg.ModuleCode < 0 {
		return nil, fmt.Errorf("module_code < 0")
	}

	appRange, err := New(0, intPow10(cfg.NumDigitsForAppCode)-1)
	if err != nil {
		return nil, err
	}
	modRange, err := New(0, intPow10(cfg.NumDigitsForModuleCode)-1)
	if err != nil {
		return nil, err
	}
	caseRange, err := New(0, intPow10(cfg.NumDigitsForCaseCode)-1)
	if err != nil {
		return nil, err
	}

	if !appRange.Include(cfg.AppCode) {
		return nil, fmt.Errorf(
			"app_code_range %s does not include given app_code %d",
			appRange, cfg.AppCode,
		)
	}
	if !modRange.Include(cfg.ModuleCode) {
		return nil, fmt.Errorf(
			"module_code_range %s does not include given module_code %d",
			modRange, cfg.ModuleCode,
		)
	}

	segs := cfg.CodeMapper.CaseCodeSegments()
	sort.Slice(segs, func(i, j int) bool { return segs[i].End() < segs[j].End() })
	for _, s := range segs {
		if !caseRange.IncludeRange(s) {
			return nil, fmt.Errorf(
				"case_code_range %s does not include CaseCodeSegment %s defined by code_mapper",
				caseRange, s,
			)
		}
	}

	return &CaseFactory{
		cfg:           cfg,
		appCodeRange:  appRange,
		moduleRange:   modRange,
		caseCodeRange: caseRange,
	}, nil
}

// AppCode returns the configured application code.
func (f *CaseFactory) AppCode() int { return f.cfg.AppCode }

// ModuleCode returns the configured module code.
func (f *CaseFactory) ModuleCode() int { return f.cfg.ModuleCode }

// NumDigitsForAppCode returns the configured digit width for app code.
func (f *CaseFactory) NumDigitsForAppCode() int { return f.cfg.NumDigitsForAppCode }

// NumDigitsForModuleCode returns the configured digit width for module code.
func (f *CaseFactory) NumDigitsForModuleCode() int { return f.cfg.NumDigitsForModuleCode }

// NumDigitsForCaseCode returns the configured digit width for case code.
func (f *CaseFactory) NumDigitsForCaseCode() int { return f.cfg.NumDigitsForCaseCode }

// BuildCaseID assembles the identifier for the given case code (without
// validating it against any segment).
func (f *CaseFactory) BuildCaseID(caseCode int) string {
	parts := make([]string, 0, 3)
	if f.cfg.NumDigitsForAppCode > 0 {
		parts = append(parts, padLeftZeros(f.cfg.AppCode, f.cfg.NumDigitsForAppCode))
	}
	if f.cfg.NumDigitsForModuleCode > 0 {
		parts = append(parts, padLeftZeros(f.cfg.ModuleCode, f.cfg.NumDigitsForModuleCode))
	}
	parts = append(parts, padLeftZeros(caseCode, f.cfg.NumDigitsForCaseCode))
	return strings.Join(parts, "_")
}

// create builds a NumCase, validating that caseCode falls within the segment
// configured for status.
func (f *CaseFactory) create(status apperror.Code, caseCode int) (NumCase, error) {
	seg, ok := f.cfg.CodeMapper.CaseCodeSegmentFor(status)
	if !ok {
		return NumCase{}, fmt.Errorf(
			"code_mapper doesn't define a CaseCodeSegment for given status_code %s",
			status,
		)
	}
	if !seg.Include(caseCode) {
		return NumCase{}, fmt.Errorf(
			"CaseCodeSegment %s for given status_code %s doesn't include given case_code %d",
			seg, status, caseCode,
		)
	}
	return NumCase{
		appCode:    f.cfg.AppCode,
		moduleCode: f.cfg.ModuleCode,
		caseCode:   caseCode,
		identifier: f.BuildCaseID(caseCode),
	}, nil
}

// NewIllegalInput creates a NumCase under the IllegalInput segment.
func (f *CaseFactory) NewIllegalInput(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeIllegalInput, caseCode)
}

// NewTimeout creates a NumCase under the Timeout segment.
func (f *CaseFactory) NewTimeout(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeTimeout, caseCode)
}

// NewNotFound creates a NumCase under the NotFound segment.
func (f *CaseFactory) NewNotFound(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeNotFound, caseCode)
}

// NewAlreadyExists creates a NumCase under the AlreadyExists segment.
func (f *CaseFactory) NewAlreadyExists(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeAlreadyExists, caseCode)
}

// NewPermissionDenied creates a NumCase under the PermissionDenied segment.
func (f *CaseFactory) NewPermissionDenied(caseCode int) (NumCase, error) {
	return f.create(apperror.CodePermissionDenied, caseCode)
}

// NewTooManyRequests creates a NumCase under the TooManyRequests segment.
func (f *CaseFactory) NewTooManyRequests(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeTooManyRequests, caseCode)
}

// NewFailedPrecondition creates a NumCase under the FailedPrecondition segment.
func (f *CaseFactory) NewFailedPrecondition(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeFailedPrecondition, caseCode)
}

// NewConflict creates a NumCase under the Conflict segment.
func (f *CaseFactory) NewConflict(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeConflict, caseCode)
}

// NewOutOfRange creates a NumCase under the OutOfRange segment.
func (f *CaseFactory) NewOutOfRange(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeOutOfRange, caseCode)
}

// NewInternal creates a NumCase under the Internal segment.
func (f *CaseFactory) NewInternal(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeInternal, caseCode)
}

// NewIllegalState creates a NumCase under the IllegalState segment.
func (f *CaseFactory) NewIllegalState(caseCode int) (NumCase, error) {
	return f.create(apperror.CodeIllegalState, caseCode)
}

func intPow10(n int) int {
	return int(math.Pow(10, float64(n)))
}

func padLeftZeros(num, minLen int) string {
	s := fmt.Sprintf("%d", num)
	if len(s) >= minLen {
		return s
	}
	return strings.Repeat("0", minLen-len(s)) + s
}
