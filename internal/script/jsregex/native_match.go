package jsregex

import (
	"regexp/syntax"
	"unicode"
	"unicode/utf8"
)

type nativeSpan struct {
	start int
	end   int
	ok    bool
}

type nativeMatchState struct {
	pos           int
	captureOffset int
	captures      nativeCaptures
}

type nativeCaptures []nativeSpan

type nativeMatchResult struct {
	startRune int
	endRune   int
	captures  nativeCaptures
}

func (s nativeMatchState) clone() nativeMatchState {
	clone := nativeMatchState{
		pos:           s.pos,
		captureOffset: s.captureOffset,
	}
	if len(s.captures) > 0 {
		clone.captures = append(nativeCaptures(nil), s.captures...)
	}
	return clone
}

func (p *CompiledPattern) nativeCaptureCount() int {
	if p == nil || p.AST == nil {
		return 0
	}
	if p.AST.CaptureCount < 0 {
		return 0
	}
	return p.AST.CaptureCount
}

func (s *RegexpState) nativeMatchResult(input string, startRune int) (*nativeMatchResult, error) {
	if s == nil || s.Pattern == nil || s.Pattern.AST == nil || s.Pattern.AST.Root == nil {
		return nil, nil
	}
	runes := []rune(input)
	if startRune < 0 {
		startRune = 0
	}
	if startRune > len(runes) {
		startRune = len(runes)
	}
	base := nativeMatchState{
		pos:           startRune,
		captureOffset: 0,
		captures:      make(nativeCaptures, s.Pattern.nativeCaptureCount()+1),
	}
	states, err := matchNativeNode(s.Pattern, s.Pattern.AST.Root, runes, base)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, nil
	}
	best := states[0]
	best.captures = append(nativeCaptures(nil), best.captures...)
	if len(best.captures) == 0 {
		best.captures = make(nativeCaptures, s.Pattern.nativeCaptureCount()+1)
	}
	best.captures[0] = nativeSpan{start: startRune, end: best.pos, ok: true}
	return &nativeMatchResult{
		startRune: startRune,
		endRune:   best.pos,
		captures:  best.captures,
	}, nil
}

func (s *RegexpState) nativeFindStringSubmatchIndex(input string) ([]int, error) {
	if s == nil || s.Pattern == nil || s.Pattern.AST == nil || s.Pattern.AST.Root == nil {
		return nil, nil
	}
	runes := []rune(input)
	for pos := 0; pos <= len(runes); pos++ {
		result, err := s.nativeMatchResult(input, pos)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		return nativeResultToIndices(s.Pattern, input, result), nil
	}
	return nil, nil
}

func (s *RegexpState) nativeFindAllStringSubmatchIndex(input string, n int) ([][]int, error) {
	if s == nil || s.Pattern == nil || s.Pattern.AST == nil || s.Pattern.AST.Root == nil {
		return nil, nil
	}
	if n == 0 {
		return [][]int{}, nil
	}
	runes := []rune(input)
	if len(runes) == 0 {
		result, err := s.nativeMatchResult(input, 0)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return [][]int{nativeResultToIndices(s.Pattern, input, result)}, nil
	}
	out := make([][]int, 0, 4)
	start := 0
	for start <= len(runes) {
		result, err := s.nativeMatchResult(input, start)
		if err != nil {
			return nil, err
		}
		if result == nil {
			start++
			continue
		}
		indices := nativeResultToIndices(s.Pattern, input, result)
		out = append(out, indices)
		if n > 0 && len(out) >= n {
			break
		}
		if result.endRune <= start {
			start++
			continue
		}
		start = result.endRune
	}
	return out, nil
}

func nativeResultToIndices(pattern *CompiledPattern, input string, result *nativeMatchResult) []int {
	if result == nil {
		return nil
	}
	out := make([]int, 0, len(result.captures)*2)
	for _, capture := range result.captures {
		if !capture.ok {
			out = append(out, -1, -1)
			continue
		}
		out = append(out, runeIndexToByteOffset(input, capture.start), runeIndexToByteOffset(input, capture.end))
	}
	return out
}

func runeIndexToByteOffset(input string, runeIndex int) int {
	if runeIndex <= 0 {
		return 0
	}
	if runeIndex >= utf8.RuneCountInString(input) {
		return len(input)
	}
	offset := 0
	for i := 0; i < runeIndex && offset < len(input); i++ {
		_, size := utf8.DecodeRuneInString(input[offset:])
		offset += size
	}
	return offset
}

func matchNativeNode(pattern *CompiledPattern, re *syntax.Regexp, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if re == nil {
		return []nativeMatchState{state}, nil
	}
	switch re.Op {
	case syntax.OpNoMatch:
		return nil, nil
	case syntax.OpEmptyMatch:
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpLiteral:
		return matchNativeLiteral(re, input, state), nil
	case syntax.OpCharClass:
		return matchNativeCharClass(re, input, state), nil
	case syntax.OpAnyCharNotNL:
		return matchNativeAny(re, input, state, false), nil
	case syntax.OpAnyChar:
		return matchNativeAny(re, input, state, true), nil
	case syntax.OpBeginLine:
		if nativeEmptyOpAt(input, state.pos)&syntax.EmptyBeginLine == 0 {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpEndLine:
		if nativeEmptyOpAt(input, state.pos)&syntax.EmptyEndLine == 0 {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpBeginText:
		if nativeEmptyOpAt(input, state.pos)&syntax.EmptyBeginText == 0 {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpEndText:
		if nativeEmptyOpAt(input, state.pos)&syntax.EmptyEndText == 0 {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpWordBoundary:
		if nativeEmptyOpAt(input, state.pos)&syntax.EmptyWordBoundary == 0 {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpNoWordBoundary:
		if nativeEmptyOpAt(input, state.pos)&syntax.EmptyNoWordBoundary == 0 {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	case syntax.OpCapture:
		return matchNativeCapture(pattern, re, input, state)
	case syntax.OpConcat:
		return matchNativeConcat(pattern, re.Sub, input, state)
	case syntax.OpAlternate:
		return matchNativeAlternate(pattern, re.Sub, input, state)
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		if len(re.Sub) != 1 {
			return nil, ErrNativeUnsupported
		}
		return matchNativeRepeat(pattern, re, re.Sub[0], input, state)
	default:
		return nil, ErrNativeUnsupported
	}
}

func matchNativeConcat(pattern *CompiledPattern, subs []*syntax.Regexp, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if len(subs) == 0 {
		return []nativeMatchState{state.clone()}, nil
	}
	states := []nativeMatchState{state.clone()}
	for _, sub := range subs {
		nextStates := make([]nativeMatchState, 0, len(states))
		for _, current := range states {
			matched, err := matchNativeNode(pattern, sub, input, current)
			if err != nil {
				return nil, err
			}
			nextStates = append(nextStates, matched...)
		}
		states = nextStates
		if len(states) == 0 {
			return nil, nil
		}
	}
	return states, nil
}

func matchNativeAlternate(pattern *CompiledPattern, subs []*syntax.Regexp, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if len(subs) == 0 {
		return nil, nil
	}
	out := make([]nativeMatchState, 0, len(subs))
	for _, sub := range subs {
		matched, err := matchNativeNode(pattern, sub, input, state.clone())
		if err != nil {
			return nil, err
		}
		out = append(out, matched...)
	}
	return out, nil
}

func matchNativeCapture(pattern *CompiledPattern, re *syntax.Regexp, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if len(re.Sub) != 1 {
		return nil, ErrNativeUnsupported
	}
	if isBackreferencePlaceholderName(re.Name) {
		return matchNativePlaceholderBackreference(pattern, re.Name, input, state)
	}
	if isLookaroundPlaceholderName(re.Name) {
		return matchNativePlaceholderLookaround(pattern, re.Name, input, state)
	}
	index := re.Cap
	visible, ok := pattern.visibleCaptureIndex(index)
	if !ok || visible < 0 || visible >= len(state.captures) {
		return nil, ErrNativeUnsupported
	}
	actual := state.captureOffset + visible
	if actual < 0 || actual >= len(state.captures) {
		return nil, ErrNativeUnsupported
	}
	base := state.clone()
	base.captures[actual] = nativeSpan{start: state.pos, ok: true}
	matched, err := matchNativeNode(pattern, re.Sub[0], input, base)
	if err != nil {
		return nil, err
	}
	for i := range matched {
		matched[i].captures[actual] = nativeSpan{start: state.pos, end: matched[i].pos, ok: true}
	}
	return matched, nil
}

func matchNativePlaceholderBackreference(pattern *CompiledPattern, name string, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if pattern == nil || pattern.AST == nil {
		return nil, ErrNativeUnsupported
	}
	index, ok := backreferencePlaceholderIndex(name)
	if !ok || index < 0 || index >= len(pattern.AST.Backreferences) {
		return nil, ErrNativeUnsupported
	}
	matched, err := matchNativeBackreference(pattern, pattern.AST.Backreferences[index], input, state)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, nil
	}
	return matched, nil
}

func matchNativePlaceholderLookaround(pattern *CompiledPattern, name string, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if pattern == nil || pattern.AST == nil {
		return nil, ErrNativeUnsupported
	}
	index, ok := lookaroundPlaceholderIndex(name)
	if !ok || index < 0 || index >= len(pattern.AST.Lookarounds) {
		return nil, ErrNativeUnsupported
	}
	matched, err := matchNativeLookaroundSpec(pattern.AST.Lookarounds[index], input, state)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, nil
	}
	return matched, nil
}

func matchNativeBackreference(pattern *CompiledPattern, spec BackreferenceSpec, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	target := spec.TargetCapture
	if target < 0 {
		return nil, ErrNativeUnsupported
	}
	if target >= len(state.captures) {
		return nil, ErrNativeUnsupported
	}
	capture := state.captures[target]
	if !capture.ok {
		return []nativeMatchState{state.clone()}, nil
	}
	if capture.end < capture.start {
		return nil, ErrNativeUnsupported
	}
	segment := input[capture.start:capture.end]
	if state.pos+len(segment) > len(input) {
		return nil, nil
	}
	foldCase := pattern != nil && pattern.AST != nil && pattern.AST.Flags.IgnoreCase
	for i, want := range segment {
		if !nativeRuneEqual(input[state.pos+i], want, foldCase) {
			return nil, nil
		}
	}
	next := state.clone()
	next.pos += len(segment)
	return []nativeMatchState{next}, nil
}

func matchNativeLookaroundSpec(spec LookaroundSpec, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if spec.AST == nil || spec.AST.Root == nil {
		if spec.Positive {
			return []nativeMatchState{state.clone()}, nil
		}
		return nil, nil
	}
	if spec.Lookbehind {
		return matchNativeLookbehindSpec(spec, input, state)
	}
	return matchNativeLookaheadSpec(spec, input, state)
}

func matchNativeLookaheadSpec(spec LookaroundSpec, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	pattern := &CompiledPattern{AST: spec.AST}
	local := lookaroundNativeState(state, spec)
	local.pos = state.pos
	matched, err := matchNativeNode(pattern, spec.AST.Root, input, local)
	if err != nil {
		return nil, err
	}
	if spec.Positive {
		if len(matched) == 0 {
			return nil, nil
		}
		merged := state.clone()
		merged.captures = mergeLookaroundCaptures(merged.captures, state.captureOffset+spec.VisibleCaptureStart, spec.AST.CaptureCount, matched[0].captures)
		return []nativeMatchState{merged}, nil
	}
	if len(matched) > 0 {
		return nil, nil
	}
	return []nativeMatchState{state.clone()}, nil
}

func matchNativeLookbehindSpec(spec LookaroundSpec, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	if spec.Width < 0 {
		return nil, ErrNativeUnsupported
	}
	end := state.pos
	start := end - spec.Width
	if start < 0 {
		if spec.Positive {
			return nil, nil
		}
		return []nativeMatchState{state.clone()}, nil
	}

	pattern := &CompiledPattern{AST: spec.AST}
	local := lookaroundNativeState(state, spec)
	local.pos = start
	matched, err := matchNativeNode(pattern, spec.AST.Root, input, local)
	if err != nil {
		return nil, err
	}
	var chosen nativeMatchState
	found := false
	for _, candidate := range matched {
		if candidate.pos == end {
			chosen = candidate
			found = true
			break
		}
	}
	if spec.Positive {
		if !found {
			return nil, nil
		}
		merged := state.clone()
		merged.captures = mergeLookaroundCaptures(merged.captures, state.captureOffset+spec.VisibleCaptureStart, spec.AST.CaptureCount, chosen.captures)
		return []nativeMatchState{merged}, nil
	}
	if found {
		return nil, nil
	}
	return []nativeMatchState{state.clone()}, nil
}

func lookaroundNativeState(outer nativeMatchState, spec LookaroundSpec) nativeMatchState {
	clone := nativeMatchState{
		pos:           outer.pos,
		captureOffset: outer.captureOffset + spec.VisibleCaptureStart - 1,
		captures:      append(nativeCaptures(nil), outer.captures...),
	}
	if clone.captureOffset < 0 {
		clone.captureOffset = 0
	}
	return clone
}

func mergeLookaroundCaptures(dst nativeCaptures, start, count int, src nativeCaptures) nativeCaptures {
	if start <= 0 || count <= 0 {
		return dst
	}
	need := start + count
	if len(dst) < need {
		grown := make(nativeCaptures, need)
		copy(grown, dst)
		dst = grown
	}
	for i := 0; i < count; i++ {
		srcIndex := start + i
		if srcIndex >= len(src) {
			break
		}
		dst[start+i] = src[srcIndex]
	}
	return dst
}

func matchNativeRepeat(pattern *CompiledPattern, re *syntax.Regexp, child *syntax.Regexp, input []rune, state nativeMatchState) ([]nativeMatchState, error) {
	min, max := nativeRepeatBounds(re)
	greedy := re.Flags&syntax.NonGreedy == 0
	var out []nativeMatchState
	var explore func(count int, current nativeMatchState) error
	explore = func(count int, current nativeMatchState) error {
		if count >= min && !greedy {
			out = append(out, current.clone())
		}
		if max >= 0 && count == max {
			if count >= min && greedy {
				out = append(out, current.clone())
			}
			return nil
		}
		nextStates, err := matchNativeNode(pattern, child, input, current)
		if err != nil {
			return err
		}
		progressed := false
		for _, next := range nextStates {
			if next.pos == current.pos {
				continue
			}
			progressed = true
			if err := explore(count+1, next); err != nil {
				return err
			}
		}
		if count >= min && greedy {
			out = append(out, current.clone())
		}
		if !progressed && count < min {
			return nil
		}
		return nil
	}
	if err := explore(0, state.clone()); err != nil {
		return nil, err
	}
	return out, nil
}

func nativeRepeatBounds(re *syntax.Regexp) (int, int) {
	switch re.Op {
	case syntax.OpStar:
		return 0, -1
	case syntax.OpPlus:
		return 1, -1
	case syntax.OpQuest:
		return 0, 1
	case syntax.OpRepeat:
		return re.Min, re.Max
	default:
		return 0, 0
	}
}

func matchNativeLiteral(re *syntax.Regexp, input []rune, state nativeMatchState) []nativeMatchState {
	if len(re.Rune) == 0 {
		return []nativeMatchState{state.clone()}
	}
	if state.pos < 0 || state.pos+len(re.Rune) > len(input) {
		return nil
	}
	foldCase := re.Flags&syntax.FoldCase != 0
	for i, want := range re.Rune {
		if !nativeRuneEqual(input[state.pos+i], want, foldCase) {
			return nil
		}
	}
	next := state.clone()
	next.pos += len(re.Rune)
	return []nativeMatchState{next}
}

func matchNativeCharClass(re *syntax.Regexp, input []rune, state nativeMatchState) []nativeMatchState {
	if state.pos < 0 || state.pos >= len(input) {
		return nil
	}
	foldCase := re.Flags&syntax.FoldCase != 0
	if !nativeRuneInClass(input[state.pos], re.Rune, foldCase) {
		return nil
	}
	next := state.clone()
	next.pos++
	return []nativeMatchState{next}
}

func matchNativeAny(re *syntax.Regexp, input []rune, state nativeMatchState, allowNewline bool) []nativeMatchState {
	if state.pos < 0 || state.pos >= len(input) {
		return nil
	}
	if !allowNewline && isNativeLineTerminator(input[state.pos]) {
		return nil
	}
	next := state.clone()
	next.pos++
	return []nativeMatchState{next}
}

func nativeEmptyOpAt(input []rune, pos int) syntax.EmptyOp {
	var r1, r2 rune = -1, -1
	if pos > 0 {
		r1 = input[pos-1]
	}
	if pos < len(input) {
		r2 = input[pos]
	}
	return syntax.EmptyOpContext(r1, r2)
}

func nativeRuneEqual(got, want rune, foldCase bool) bool {
	if got == want {
		return true
	}
	if !foldCase {
		return false
	}
	for fold := unicode.SimpleFold(got); fold != got; fold = unicode.SimpleFold(fold) {
		if fold == want {
			return true
		}
	}
	return false
}

func nativeRuneInClass(r rune, class []rune, foldCase bool) bool {
	for i := 0; i+1 < len(class); i += 2 {
		if nativeRuneInRange(r, class[i], class[i+1], foldCase) {
			return true
		}
	}
	return false
}

func nativeRuneInRange(r, lo, hi rune, foldCase bool) bool {
	if lo <= r && r <= hi {
		return true
	}
	if !foldCase {
		return false
	}
	for fold := unicode.SimpleFold(r); fold != r; fold = unicode.SimpleFold(fold) {
		if lo <= fold && fold <= hi {
			return true
		}
	}
	return false
}

func isNativeLineTerminator(r rune) bool {
	switch r {
	case '\n', '\r', '\u2028', '\u2029':
		return true
	default:
		return false
	}
}
