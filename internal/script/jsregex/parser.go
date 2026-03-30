package jsregex

import (
	"errors"
	"fmt"
	"regexp/syntax"
)

// ErrNativeUnsupported is returned when the native syntax tree cannot express
// a pattern and the caller should try the fallback engine instead.
var ErrNativeUnsupported = errors.New("jsregex: unsupported syntax for native engine")

// ParseFlags validates and normalizes the ECMAScript flag set.
func ParseFlags(flags string) (FlagSet, error) {
	var set FlagSet
	seen := make(map[byte]struct{}, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		if _, ok := seen[flag]; ok {
			return FlagSet{}, fmt.Errorf("duplicate regular expression flag %q", flag)
		}
		seen[flag] = struct{}{}
		switch flag {
		case 'd':
			set.Indices = true
		case 'g':
			set.Global = true
		case 'i':
			set.IgnoreCase = true
		case 'm':
			set.Multiline = true
		case 's':
			set.DotAll = true
		case 'u':
			set.Unicode = true
		case 'v':
			set.UnicodeSets = true
		case 'y':
			set.Sticky = true
		default:
			return FlagSet{}, fmt.Errorf("unsupported regular expression flag %q", flag)
		}
	}
	return set, nil
}

// Parse builds the regex AST placeholder for a pattern and flag set.
func Parse(pattern, flags string) (*AST, error) {
	translated, err := expandRegExpUnicodeEscapes(pattern)
	if err != nil {
		return nil, err
	}
	return parseTranslatedPattern(translated, flags, true)
}

func parseTranslatedPattern(pattern, flags string, allowBackreferences bool) (*AST, error) {
	parsed, err := ParseFlags(flags)
	if err != nil {
		return nil, err
	}

	rewritten := pattern
	var backreferences []BackreferenceSpec
	if allowBackreferences {
		rewritten, backreferences, err = replaceBackreferences(rewritten)
		if err != nil {
			return nil, err
		}
	}

	rewritten, lookarounds, err := replaceLookarounds(rewritten, flags, allowBackreferences)
	if err != nil {
		return nil, err
	}

	syntaxFlags := syntax.Perl
	if parsed.IgnoreCase {
		syntaxFlags |= syntax.FoldCase
	}
	if parsed.DotAll {
		syntaxFlags |= syntax.DotNL
	}
	if parsed.Multiline {
		syntaxFlags &^= syntax.OneLine
	}

	tree, err := syntax.Parse(rewritten, syntaxFlags)
	if err != nil {
		if regexpNeedsRegexp2(pattern) {
			return nil, ErrNativeUnsupported
		}
		return nil, err
	}
	tree = tree.Simplify()
	if !nativeRegexpSupported(tree) {
		return nil, ErrNativeUnsupported
	}
	captureNames, captureIndexMap, backreferences, err := buildCaptureLayout(tree.CapNames(), lookarounds, backreferences, allowBackreferences)
	if err != nil {
		return nil, err
	}
	return &AST{
		Source:          pattern,
		Flags:           parsed,
		Root:            tree,
		CaptureNames:    captureNames,
		CaptureCount:    len(captureNames) - 1,
		CaptureIndexMap: captureIndexMap,
		Lookarounds:     lookarounds,
		Backreferences:  backreferences,
	}, nil
}

func buildCaptureLayout(names []string, lookarounds []LookaroundSpec, backreferences []BackreferenceSpec, allowBackreferences bool) ([]string, []int, []BackreferenceSpec, error) {
	visibleNames := make([]string, 1, len(names))
	visibleNames[0] = ""
	captureIndexMap := make([]int, len(names))
	if len(captureIndexMap) > 0 {
		captureIndexMap[0] = 0
	}
	visibleCount := 0
	for outer := 1; outer < len(names); outer++ {
		name := names[outer]
		if isLookaroundPlaceholderName(name) {
			index, ok := lookaroundPlaceholderIndex(name)
			if !ok || index < 0 || index >= len(lookarounds) {
				return nil, nil, nil, ErrNativeUnsupported
			}
			spec := &lookarounds[index]
			if spec.AST == nil {
				return nil, nil, nil, ErrNativeUnsupported
			}
			spec.VisibleCaptureStart = visibleCount + 1
			innerCount := spec.AST.CaptureCount
			for inner := 1; inner <= innerCount; inner++ {
				visibleCount++
				innerName := ""
				if inner < len(spec.AST.CaptureNames) {
					innerName = spec.AST.CaptureNames[inner]
				}
				visibleNames = append(visibleNames, innerName)
			}
			captureIndexMap[outer] = -1
			continue
		}
		if isBackreferencePlaceholderName(name) {
			if !allowBackreferences {
				return nil, nil, nil, ErrNativeUnsupported
			}
			index, ok := backreferencePlaceholderIndex(name)
			if !ok || index < 0 || index >= len(backreferences) {
				return nil, nil, nil, ErrNativeUnsupported
			}
			captureIndexMap[outer] = -1
			continue
		}
		visibleCount++
		visibleNames = append(visibleNames, name)
		captureIndexMap[outer] = visibleCount
	}
	if allowBackreferences {
		for i := range backreferences {
			spec := &backreferences[i]
			if spec.TargetName != "" {
				target := -1
				for visible := 1; visible < len(visibleNames); visible++ {
					if visibleNames[visible] == spec.TargetName {
						target = visible
						break
					}
				}
				if target < 0 {
					return nil, nil, nil, fmt.Errorf("undefined named backreference %q", spec.TargetName)
				}
				spec.TargetCapture = target
				continue
			}
			if spec.TargetNumber <= 0 || spec.TargetNumber >= len(visibleNames) {
				return nil, nil, nil, fmt.Errorf("invalid backreference %d", spec.TargetNumber)
			}
			spec.TargetCapture = spec.TargetNumber
		}
	}
	return visibleNames, captureIndexMap, backreferences, nil
}

func nativeRegexpSupported(re *syntax.Regexp) bool {
	if re == nil {
		return true
	}
	switch re.Op {
	case syntax.OpNoMatch,
		syntax.OpEmptyMatch,
		syntax.OpLiteral,
		syntax.OpCharClass,
		syntax.OpAnyCharNotNL,
		syntax.OpAnyChar,
		syntax.OpBeginLine,
		syntax.OpEndLine,
		syntax.OpBeginText,
		syntax.OpEndText,
		syntax.OpWordBoundary,
		syntax.OpNoWordBoundary:
		return true
	case syntax.OpCapture, syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		return len(re.Sub) == 1 && nativeRegexpSupported(re.Sub[0])
	case syntax.OpConcat, syntax.OpAlternate:
		for _, sub := range re.Sub {
			if !nativeRegexpSupported(sub) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
