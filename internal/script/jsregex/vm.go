package jsregex

// Test reports whether the pattern matches the input.
func (s *RegexpState) Test(input string) (bool, error) {
	return s.MatchString(input)
}

// Exec returns the first match result for the input.
func (s *RegexpState) Exec(input string) (*MatchResult, error) {
	indices, err := s.FindStringSubmatchIndex(input)
	if err != nil || indices == nil {
		return nil, err
	}
	return s.matchResultFromIndices(input, indices), nil
}

// MatchString is a convenience alias for Test.
func (s *RegexpState) MatchString(input string) (bool, error) {
	if s == nil || s.Pattern == nil {
		return false, nil
	}
	match, err := s.nativeFindStringSubmatchIndex(input)
	if err != nil {
		return false, err
	}
	return match != nil, nil
}

// FindStringIndex returns the start and end byte offsets of the first match.
func (s *RegexpState) FindStringIndex(input string) ([]int, error) {
	indices, err := s.FindStringSubmatchIndex(input)
	if err != nil || len(indices) < 2 {
		return indices, err
	}
	return indices[:2], nil
}

// FindStringSubmatch returns the matched text and captures for the first match.
func (s *RegexpState) FindStringSubmatch(input string) ([]string, error) {
	indices, err := s.FindStringSubmatchIndex(input)
	if err != nil || indices == nil {
		return nil, err
	}
	return submatchesFromIndex(input, indices), nil
}

// FindStringSubmatchIndex returns the offsets for the first match and its
// captures.
func (s *RegexpState) FindStringSubmatchIndex(input string) ([]int, error) {
	if s == nil || s.Pattern == nil {
		return nil, nil
	}
	return s.nativeFindStringSubmatchIndex(input)
}

// FindAllStringSubmatch returns all matches up to the provided limit.
func (s *RegexpState) FindAllStringSubmatch(input string, n int) ([][]string, error) {
	indices, err := s.FindAllStringSubmatchIndex(input, n)
	if err != nil || len(indices) == 0 {
		return nil, err
	}
	out := make([][]string, len(indices))
	for i, loc := range indices {
		out[i] = submatchesFromIndex(input, loc)
	}
	return out, nil
}

// FindAllStringSubmatchIndex returns all match offsets up to the provided
// limit.
func (s *RegexpState) FindAllStringSubmatchIndex(input string, n int) ([][]int, error) {
	if s == nil || s.Pattern == nil {
		return nil, nil
	}
	if n == 0 {
		return [][]int{}, nil
	}
	return s.nativeFindAllStringSubmatchIndex(input, n)
}

func (s *RegexpState) matchResultFromIndices(input string, indices []int) *MatchResult {
	if s == nil || s.Pattern == nil || len(indices) == 0 {
		return nil
	}
	captures := submatchesFromIndex(input, indices)
	if len(captures) == 0 {
		return nil
	}
	result := &MatchResult{
		Full:     captures[0],
		Captures: append([]string(nil), captures...),
		Index:    indices[0],
		Input:    input,
	}
	clone := append([]int(nil), indices...)
	result.Indices = [][]int{clone}
	if names := s.Pattern.captureNames(); len(names) > 0 {
		result.NamedCaptures = namedCapturesFromNames(names, captures)
	}
	return result
}

// Split separates the input around regex matches.
func (s *RegexpState) Split(input string, n int) ([]string, error) {
	if s == nil || s.Pattern == nil {
		return []string{input}, nil
	}
	if n == 0 {
		return []string{}, nil
	}
	if n == 1 {
		return []string{input}, nil
	}
	matches, err := s.FindAllStringSubmatchIndex(input, -1)
	if err != nil || len(matches) == 0 {
		return []string{input}, err
	}
	out := make([]string, 0, len(matches)+1)
	last := 0
	for _, loc := range matches {
		if n > 0 && len(out) >= n-1 {
			break
		}
		if len(loc) < 2 {
			continue
		}
		start, end := loc[0], loc[1]
		if start < 0 || end < 0 || start > end || end > len(input) {
			continue
		}
		out = append(out, input[last:start])
		last = end
	}
	out = append(out, input[last:])
	if n > 0 && len(out) > n {
		return out[:n], nil
	}
	return out, nil
}

func submatchesFromIndex(input string, indices []int) []string {
	if len(indices) == 0 {
		return nil
	}
	out := make([]string, 0, len(indices)/2)
	for i := 0; i+1 < len(indices); i += 2 {
		start := indices[i]
		end := indices[i+1]
		if start < 0 || end < 0 || start > end || end > len(input) {
			out = append(out, "")
			continue
		}
		out = append(out, input[start:end])
	}
	return out
}

func namedCapturesFromNames(names []string, captures []string) map[string]string {
	if len(names) == 0 || len(captures) == 0 {
		return nil
	}
	out := make(map[string]string)
	for i, name := range names {
		if i == 0 || name == "" || i >= len(captures) {
			continue
		}
		out[name] = captures[i]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
