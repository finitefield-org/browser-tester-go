package runtime

import (
	"fmt"
	neturl "net/url"
	"strconv"
	"strings"
)

type browserURLState struct {
	href         string
	parsed       *neturl.URL
	searchParams *browserURLSearchParamsState
}

type browserURLSearchParamsState struct {
	rawQuery string
	owner    *browserURLState
}

type urlSearchParamPair struct {
	key   string
	value string
}

func newBrowserURLState(parsed *neturl.URL, href string) *browserURLState {
	state := &browserURLState{}
	if parsed != nil {
		cloned := *parsed
		state.parsed = &cloned
	} else if trimmed := strings.TrimSpace(href); trimmed != "" {
		if parsedHref, err := neturl.Parse(trimmed); err == nil && parsedHref != nil {
			state.parsed = parsedHref
		}
	}
	if state.parsed == nil {
		state.parsed = &neturl.URL{}
	}
	if strings.TrimSpace(href) == "" {
		href = state.parsed.String()
	}
	state.href = strings.TrimSpace(href)
	state.syncSearchParamsFromParsed()
	return state
}

func cloneBrowserURLStateMap(states map[string]*browserURLState) map[string]*browserURLState {
	if len(states) == 0 {
		return nil
	}
	out := make(map[string]*browserURLState, len(states))
	for id, state := range states {
		if state == nil {
			out[id] = nil
			continue
		}
		out[id] = newBrowserURLState(state.parsed, state.hrefString())
	}
	return out
}

func parseBrowserURLInstancePath(path string) (string, string, bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "url:") {
		return "", "", false
	}
	rest := strings.TrimPrefix(normalized, "url:")
	if rest == "" {
		return "", "", false
	}
	id, suffix, found := strings.Cut(rest, ".")
	if strings.TrimSpace(id) == "" {
		return "", "", false
	}
	if !found {
		suffix = ""
	}
	return strings.TrimSpace(id), strings.TrimSpace(suffix), true
}

func (s *Session) allocateBrowserURLState(parsed *neturl.URL, href string) string {
	if s == nil {
		return ""
	}
	if s.urlStates == nil {
		s.urlStates = map[string]*browserURLState{}
	}
	if s.nextURLStateID <= 0 {
		s.nextURLStateID = 1
	}
	id := strconv.FormatInt(s.nextURLStateID, 10)
	s.nextURLStateID++
	s.urlStates[id] = newBrowserURLState(parsed, href)
	return id
}

func (s *Session) browserURLStateByID(id string) (*browserURLState, bool) {
	if s == nil {
		return nil, false
	}
	if s.urlStates == nil {
		return nil, false
	}
	state, ok := s.urlStates[strings.TrimSpace(id)]
	return state, ok
}

func (s *browserURLState) hrefString() string {
	if s == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(s.href); trimmed != "" {
		return trimmed
	}
	if s.parsed != nil {
		return s.parsed.String()
	}
	return ""
}

func (s *browserURLState) searchString() string {
	if s == nil {
		return ""
	}
	if s.parsed != nil {
		if s.parsed.RawQuery != "" {
			return "?" + s.parsed.RawQuery
		}
		if s.parsed.ForceQuery {
			return "?"
		}
	}
	if s.searchParams != nil && s.searchParams.rawQuery != "" {
		return "?" + s.searchParams.rawQuery
	}
	return ""
}

func (s *browserURLState) syncSearchParamsFromParsed() {
	if s == nil {
		return
	}
	params := s.ensureSearchParams()
	if s.parsed == nil {
		params.rawQuery = ""
		return
	}
	params.rawQuery = normalizeBrowserURLSearchParamsRawQuery(s.parsed.RawQuery)
}

func (s *browserURLState) setRawQuery(rawQuery string) {
	if s == nil {
		return
	}
	normalized := normalizeBrowserURLSearchParamsRawQuery(rawQuery)
	if s.parsed == nil {
		s.parsed = &neturl.URL{}
	}
	s.parsed.RawQuery = normalized
	s.parsed.ForceQuery = false
	s.href = s.parsed.String()
	s.syncSearchParamsFromParsed()
}

func (s *browserURLState) setHash(rawHash string) {
	if s == nil {
		return
	}
	normalized := strings.TrimSpace(rawHash)
	normalized = strings.TrimPrefix(normalized, "#")
	if s.parsed == nil {
		s.parsed = &neturl.URL{}
	}
	s.parsed.Fragment = normalized
	s.href = s.parsed.String()
}

func (s *browserURLState) ensureSearchParams() *browserURLSearchParamsState {
	if s == nil {
		return nil
	}
	if s.searchParams == nil {
		s.searchParams = &browserURLSearchParamsState{}
	}
	s.searchParams.owner = s
	return s.searchParams
}

func (s *browserURLState) setHref(raw string) error {
	if s == nil {
		return fmt.Errorf("URL state is unavailable")
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("URL.href assignment requires a non-empty URL")
	}
	resolved := resolveHyperlinkURL(s.hrefString(), trimmed)
	parsed, err := neturl.Parse(resolved)
	if err != nil {
		return err
	}
	if parsed == nil {
		return fmt.Errorf("URL.href assignment produced an empty URL")
	}
	cloned := *parsed
	s.parsed = &cloned
	s.href = cloned.String()
	s.syncSearchParamsFromParsed()
	return nil
}

func (s *browserURLSearchParamsState) rawQueryString() string {
	if s == nil {
		return ""
	}
	return normalizeBrowserURLSearchParamsRawQuery(s.rawQuery)
}

func (s *browserURLSearchParamsState) snapshotPairs() ([]urlSearchParamPair, error) {
	if s == nil {
		return nil, nil
	}
	return parseURLSearchParamPairs(s.rawQueryString())
}

func (s *browserURLSearchParamsState) setRawQuery(rawQuery string) {
	if s == nil {
		return
	}
	normalized := normalizeBrowserURLSearchParamsRawQuery(rawQuery)
	s.rawQuery = normalized
	if s.owner != nil {
		s.owner.setRawQuery(normalized)
	}
}

func (s *browserURLSearchParamsState) setPairs(pairs []urlSearchParamPair) {
	if s == nil {
		return
	}
	s.setRawQuery(serializeURLSearchParamPairs(pairs))
}
