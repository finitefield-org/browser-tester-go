package jsregex

import "strings"

// ReplacementTokenKind identifies a parsed replacement template token.
type ReplacementTokenKind string

const (
	ReplacementTokenLiteral ReplacementTokenKind = "literal"
	ReplacementTokenWhole   ReplacementTokenKind = "whole"
	ReplacementTokenPrefix  ReplacementTokenKind = "prefix"
	ReplacementTokenSuffix  ReplacementTokenKind = "suffix"
	ReplacementTokenCapture ReplacementTokenKind = "capture"
	ReplacementTokenNamed   ReplacementTokenKind = "named"
)

// ReplacementToken is a placeholder for the tokenized replacement template.
type ReplacementToken struct {
	Kind       ReplacementTokenKind
	Text       string
	GroupIndex int
	GroupName  string
}

// ReplacementTemplate is the parsed replacement form that will eventually be
// reused by replace() and replaceAll().
type ReplacementTemplate struct {
	Raw    string
	Tokens []ReplacementToken
}

// ParseReplacementTemplate tokenizes a JS replacement string.
func ParseReplacementTemplate(template string) (*ReplacementTemplate, error) {
	parsed := &ReplacementTemplate{Raw: template}
	if template == "" {
		return parsed, nil
	}

	var literal strings.Builder
	flushLiteral := func() {
		if literal.Len() == 0 {
			return
		}
		parsed.Tokens = append(parsed.Tokens, ReplacementToken{
			Kind: ReplacementTokenLiteral,
			Text: literal.String(),
		})
		literal.Reset()
	}

	for i := 0; i < len(template); {
		if template[i] != '$' || i+1 >= len(template) {
			literal.WriteByte(template[i])
			i++
			continue
		}

		next := template[i+1]
		switch next {
		case '$':
			flushLiteral()
			parsed.Tokens = append(parsed.Tokens, ReplacementToken{
				Kind: ReplacementTokenLiteral,
				Text: "$",
			})
			i += 2
		case '&':
			flushLiteral()
			parsed.Tokens = append(parsed.Tokens, ReplacementToken{Kind: ReplacementTokenWhole})
			i += 2
		case '`':
			flushLiteral()
			parsed.Tokens = append(parsed.Tokens, ReplacementToken{Kind: ReplacementTokenPrefix})
			i += 2
		case '\'':
			flushLiteral()
			parsed.Tokens = append(parsed.Tokens, ReplacementToken{Kind: ReplacementTokenSuffix})
			i += 2
		case '<':
			end := strings.IndexByte(template[i+2:], '>')
			if end < 0 {
				literal.WriteByte(template[i])
				i++
				continue
			}
			name := template[i+2 : i+2+end]
			if name == "" {
				literal.WriteByte(template[i])
				i++
				continue
			}
			flushLiteral()
			parsed.Tokens = append(parsed.Tokens, ReplacementToken{
				Kind:      ReplacementTokenNamed,
				GroupName: name,
			})
			i += end + 3
		default:
			if next >= '1' && next <= '9' {
				digits := string(next)
				if i+2 < len(template) && template[i+2] >= '0' && template[i+2] <= '9' {
					digits += string(template[i+2])
					i += 3
				} else {
					i += 2
				}
				flushLiteral()
				parsed.Tokens = append(parsed.Tokens, ReplacementToken{
					Kind: ReplacementTokenCapture,
					Text: digits,
				})
				continue
			}
			literal.WriteByte('$')
			i++
		}
	}

	flushLiteral()
	return parsed, nil
}

// ReplaceString applies a replacement template to the first match.
func (s *RegexpState) ReplaceString(input, template string) (string, error) {
	if s == nil || s.Pattern == nil {
		return input, nil
	}
	parsed, err := ParseReplacementTemplate(template)
	if err != nil {
		return "", err
	}
	indices, err := s.FindStringSubmatchIndex(input)
	if err != nil || indices == nil {
		return input, err
	}
	result := s.matchResultFromIndices(input, indices)
	if result == nil {
		return input, nil
	}
	replacement := expandReplacementTemplate(result, parsed)
	end := result.Index + len(result.Full)
	return input[:result.Index] + replacement + input[end:], nil
}

// ReplaceAllString applies a replacement template to every match.
func (s *RegexpState) ReplaceAllString(input, template string) (string, error) {
	if s == nil || s.Pattern == nil {
		return input, nil
	}
	parsed, err := ParseReplacementTemplate(template)
	if err != nil {
		return "", err
	}
	matches, err := s.FindAllStringSubmatchIndex(input, -1)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return input, nil
	}

	var b strings.Builder
	last := 0
	for _, indices := range matches {
		if len(indices) < 2 {
			continue
		}
		start, end := indices[0], indices[1]
		if start < 0 || end < 0 || start < last {
			continue
		}
		result := s.matchResultFromIndices(input, indices)
		if result == nil {
			continue
		}
		b.WriteString(input[last:start])
		b.WriteString(expandReplacementTemplate(result, parsed))
		last = end
	}
	b.WriteString(input[last:])
	return b.String(), nil
}

func expandReplacementTemplate(result *MatchResult, template *ReplacementTemplate) string {
	if result == nil || template == nil {
		return ""
	}
	var b strings.Builder
	for _, token := range template.Tokens {
		switch token.Kind {
		case ReplacementTokenLiteral:
			b.WriteString(token.Text)
		case ReplacementTokenWhole:
			b.WriteString(result.Full)
		case ReplacementTokenPrefix:
			if result.Index > len(result.Input) {
				continue
			}
			b.WriteString(result.Input[:result.Index])
		case ReplacementTokenSuffix:
			end := result.Index + len(result.Full)
			if end < 0 {
				end = 0
			}
			if end > len(result.Input) {
				end = len(result.Input)
			}
			b.WriteString(result.Input[end:])
		case ReplacementTokenCapture:
			b.WriteString(expandCaptureReference(token.Text, result.Captures))
		case ReplacementTokenNamed:
			if result.NamedCaptures != nil {
				if captured, ok := result.NamedCaptures[token.GroupName]; ok {
					b.WriteString(captured)
					continue
				}
			}
			b.WriteString("$<")
			b.WriteString(token.GroupName)
			b.WriteByte('>')
		}
	}
	return b.String()
}

func expandCaptureReference(raw string, captures []string) string {
	if len(raw) == 0 {
		return "$"
	}
	if len(captures) == 0 {
		return "$" + raw
	}

	if len(raw) >= 2 {
		if idx, ok := parseCaptureIndex(raw[:2], len(captures)); ok {
			return captures[idx]
		}
	}
	if idx, ok := parseCaptureIndex(raw[:1], len(captures)); ok {
		if len(raw) == 1 {
			return captures[idx]
		}
		return captures[idx] + raw[1:]
	}
	return "$" + raw
}

func parseCaptureIndex(raw string, captureCount int) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	if raw[0] < '1' || raw[0] > '9' {
		return 0, false
	}
	index := int(raw[0] - '0')
	if len(raw) == 2 {
		if raw[1] < '0' || raw[1] > '9' {
			return 0, false
		}
		index = index*10 + int(raw[1]-'0')
	}
	if index <= 0 || index >= captureCount {
		return 0, false
	}
	return index, true
}
