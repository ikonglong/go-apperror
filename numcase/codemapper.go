package numcase

import (
	"fmt"
	"sort"
	"strings"

	apperror "github.com/ikonglong/go-apperror"
)

// CodeMapper maps operation status codes to the case-code segment that
// numeric Case identifiers for that status code must fall within.
type CodeMapper interface {
	// HasMappingFor reports whether status has a configured segment.
	HasMappingFor(status apperror.Code) bool

	// CaseCodeSegmentFor returns the segment for status. The second return
	// is false when no mapping is defined.
	CaseCodeSegmentFor(status apperror.Code) (NumRange, bool)

	// CaseCodeSegments returns every configured segment.
	CaseCodeSegments() []NumRange

	// Mappings returns a copy of every status->segment mapping.
	Mappings() map[apperror.Code]NumRange

	IllegalInput() NumRange
	Timeout() NumRange
	NotFound() NumRange
	AlreadyExists() NumRange
	PermissionDenied() NumRange
	TooManyRequests() NumRange
	FailedPrecondition() NumRange
	Conflict() NumRange
	OutOfRange() NumRange
	Internal() NumRange
	IllegalState() NumRange
}

// MapperConfig defines the case-code segments for each status code that has
// a mapping. Pass it to NewBaseCodeMapper to obtain a ready-to-use mapper.
type MapperConfig struct {
	IllegalInput       NumRange
	Timeout            NumRange
	NotFound           NumRange
	AlreadyExists      NumRange
	PermissionDenied   NumRange
	TooManyRequests    NumRange
	FailedPrecondition NumRange
	Conflict           NumRange
	OutOfRange         NumRange
	Internal           NumRange
	IllegalState       NumRange
}

// BaseCodeMapper is a CodeMapper backed by a MapperConfig.
type BaseCodeMapper struct {
	cfg      MapperConfig
	mappings map[apperror.Code]NumRange
}

// NewBaseCodeMapper builds a BaseCodeMapper from cfg.
func NewBaseCodeMapper(cfg MapperConfig) *BaseCodeMapper {
	m := &BaseCodeMapper{cfg: cfg}
	m.mappings = map[apperror.Code]NumRange{
		apperror.CodeIllegalInput:       cfg.IllegalInput,
		apperror.CodeTimeout:            cfg.Timeout,
		apperror.CodeNotFound:           cfg.NotFound,
		apperror.CodeAlreadyExists:      cfg.AlreadyExists,
		apperror.CodePermissionDenied:   cfg.PermissionDenied,
		apperror.CodeTooManyRequests:    cfg.TooManyRequests,
		apperror.CodeFailedPrecondition: cfg.FailedPrecondition,
		apperror.CodeConflict:           cfg.Conflict,
		apperror.CodeOutOfRange:         cfg.OutOfRange,
		apperror.CodeInternal:           cfg.Internal,
		apperror.CodeIllegalState:       cfg.IllegalState,
	}
	return m
}

// HasMappingFor implements CodeMapper.
func (m *BaseCodeMapper) HasMappingFor(status apperror.Code) bool {
	_, ok := m.mappings[status]
	return ok
}

// CaseCodeSegmentFor implements CodeMapper.
func (m *BaseCodeMapper) CaseCodeSegmentFor(status apperror.Code) (NumRange, bool) {
	r, ok := m.mappings[status]
	return r, ok
}

// CaseCodeSegments implements CodeMapper.
func (m *BaseCodeMapper) CaseCodeSegments() []NumRange {
	out := make([]NumRange, 0, len(m.mappings))
	for _, r := range m.mappings {
		out = append(out, r)
	}
	return out
}

// Mappings implements CodeMapper.
func (m *BaseCodeMapper) Mappings() map[apperror.Code]NumRange {
	out := make(map[apperror.Code]NumRange, len(m.mappings))
	for k, v := range m.mappings {
		out[k] = v
	}
	return out
}

func (m *BaseCodeMapper) IllegalInput() NumRange       { return m.cfg.IllegalInput }
func (m *BaseCodeMapper) Timeout() NumRange            { return m.cfg.Timeout }
func (m *BaseCodeMapper) NotFound() NumRange           { return m.cfg.NotFound }
func (m *BaseCodeMapper) AlreadyExists() NumRange      { return m.cfg.AlreadyExists }
func (m *BaseCodeMapper) PermissionDenied() NumRange   { return m.cfg.PermissionDenied }
func (m *BaseCodeMapper) TooManyRequests() NumRange    { return m.cfg.TooManyRequests }
func (m *BaseCodeMapper) FailedPrecondition() NumRange { return m.cfg.FailedPrecondition }
func (m *BaseCodeMapper) Conflict() NumRange           { return m.cfg.Conflict }
func (m *BaseCodeMapper) OutOfRange() NumRange         { return m.cfg.OutOfRange }
func (m *BaseCodeMapper) Internal() NumRange           { return m.cfg.Internal }
func (m *BaseCodeMapper) IllegalState() NumRange       { return m.cfg.IllegalState }

// String renders a table summarising the mappings, useful for diagnostics.
func (m *BaseCodeMapper) String() string {
	type row struct {
		seg    NumRange
		opCode apperror.Code
	}
	rows := make([]row, 0, len(m.mappings))
	for c, r := range m.mappings {
		rows = append(rows, row{seg: r, opCode: c})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].seg.End() < rows[j].seg.End() })

	bar := "+" + strings.Repeat("-", 88) + "+"
	var b strings.Builder
	b.WriteString(bar)
	b.WriteString("\n")
	fmt.Fprintf(&b, "| %-15s | %-20s:%-14s | %-20s:%-14s |\n",
		"Case Code Segment", "Operation Status", "Code", "HTTP Status", "Code")
	b.WriteString(bar)
	b.WriteString("\n")
	for _, r := range rows {
		http, ok := apperror.HTTPStatusFor(r.opCode)
		if !ok {
			panic(fmt.Sprintf("no HTTP status found for %s", r.opCode))
		}
		fmt.Fprintf(&b, "| %-15s | %-20s:%-14d | %-20s:%-14d |\n",
			r.seg.String(), r.opCode.Name(), r.opCode.Value(),
			http.Name(), http.Value())
	}
	b.WriteString(bar)
	return b.String()
}

// NewDefaultCodeMapper returns the default CodeMapper, which uses
// 50-wide segments starting at 1.
func NewDefaultCodeMapper() *BaseCodeMapper {
	return NewBaseCodeMapper(MapperConfig{
		IllegalInput:       MustNew(1, 50),
		Timeout:            MustNew(51, 100),
		NotFound:           MustNew(101, 150),
		AlreadyExists:      MustNew(151, 200),
		PermissionDenied:   MustNew(201, 250),
		TooManyRequests:    MustNew(251, 300),
		FailedPrecondition: MustNew(301, 350),
		Conflict:           MustNew(351, 400),
		OutOfRange:         MustNew(401, 450),
		Internal:           MustNew(451, 500),
		IllegalState:       MustNew(501, 550),
	})
}
