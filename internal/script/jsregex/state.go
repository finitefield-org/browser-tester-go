package jsregex

import (
	"errors"
	"regexp/syntax"
)

// ErrNotImplemented is reserved for explicit unsupported entry points.
var ErrNotImplemented = errors.New("jsregex: native engine not implemented yet")

func notImplemented(op string) error {
	return errors.Join(ErrNotImplemented, errors.New(op))
}

// FlagSet captures the ECMAScript regular-expression flags in structured form.
type FlagSet struct {
	Global      bool
	IgnoreCase  bool
	Multiline   bool
	DotAll      bool
	Unicode     bool
	Sticky      bool
	Indices     bool
	UnicodeSets bool
}

// AST is the parsed regular-expression tree placeholder.
//
// CaptureNames and CaptureCount use the JS-visible capture numbering after
// lookaround expansion has been flattened back into the visible layout.
type AST struct {
	Source          string
	Flags           FlagSet
	Root            *syntax.Regexp
	CaptureNames    []string
	CaptureCount    int
	CaptureIndexMap []int
	Lookarounds     []LookaroundSpec
	Backreferences  []BackreferenceSpec
}

// LookaroundSpec stores a zero-width assertion that was extracted during
// parsing and must be evaluated by the native VM. VisibleCaptureStart points
// at the first visible capture slot contributed by the assertion body.
type LookaroundSpec struct {
	Positive            bool
	Lookbehind          bool
	Width               int
	VisibleCaptureStart int
	AST                 *AST
}

// BackreferenceSpec stores a backreference extracted during parsing and
// resolved against the visible capture layout before execution.
type BackreferenceSpec struct {
	TargetNumber  int
	TargetName    string
	TargetCapture int
}

// CompiledPattern is the immutable compiled form that holds the lowered
// representation and capture metadata.
type CompiledPattern struct {
	AST    *AST
	Source string
	Flags  string
	Mode   FlagSet
}

func (p *CompiledPattern) captureNames() []string {
	if p == nil || p.AST == nil {
		return nil
	}
	return p.AST.CaptureNames
}

func (p *CompiledPattern) visibleCaptureIndex(outer int) (int, bool) {
	if p == nil || p.AST == nil {
		return outer, true
	}
	if outer < 0 || outer >= len(p.AST.CaptureIndexMap) {
		return 0, false
	}
	visible := p.AST.CaptureIndexMap[outer]
	if visible < 0 {
		return 0, false
	}
	return visible, true
}

func (p *CompiledPattern) backreferenceSpec(index int) (BackreferenceSpec, bool) {
	if p == nil || p.AST == nil {
		return BackreferenceSpec{}, false
	}
	if index < 0 || index >= len(p.AST.Backreferences) {
		return BackreferenceSpec{}, false
	}
	return p.AST.Backreferences[index], true
}

// NewState creates the mutable runtime state for a compiled pattern.
func (p *CompiledPattern) NewState() *RegexpState {
	if p == nil {
		return nil
	}
	return &RegexpState{Pattern: p}
}

// RegexpState is the mutable per-instance regex state. The final engine will
// keep lastIndex and other runtime bookkeeping here instead of in plain object
// metadata.
type RegexpState struct {
	Pattern   *CompiledPattern
	LastIndex int
}

// Clone returns a shallow copy of the regex state.
func (s *RegexpState) Clone() *RegexpState {
	if s == nil {
		return nil
	}
	clone := *s
	return &clone
}

// Source returns the original pattern source if one is attached.
func (s *RegexpState) Source() string {
	if s == nil || s.Pattern == nil {
		return ""
	}
	return s.Pattern.Source
}

// Flags returns the original flag string if one is attached.
func (s *RegexpState) Flags() string {
	if s == nil || s.Pattern == nil {
		return ""
	}
	return s.Pattern.Flags
}

// MatchResult holds the data that browser-facing APIs need to surface from a
// successful match.
type MatchResult struct {
	Full          string
	Captures      []string
	NamedCaptures map[string]string
	Index         int
	Input         string
	Indices       [][]int
}
