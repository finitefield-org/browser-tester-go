package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
)

func (s *Session) resolveExternalScriptSource(src string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("session is unavailable")
	}

	normalized := strings.TrimSpace(src)
	if normalized == "" {
		return "", fmt.Errorf("external script src requires a non-empty URL")
	}

	resolved := strings.TrimSpace(resolveHyperlinkURL(s.URL(), normalized))
	if resolved == "" {
		return "", fmt.Errorf("external script src %q resolved to an empty URL", normalized)
	}

	registry := s.Registry()
	if registry == nil {
		return "", fmt.Errorf("external JS mock registry is unavailable")
	}
	return registry.ExternalJS().Resolve(resolved)
}

func (s *Session) resolveScriptSource(store *dom.Store, nodeID dom.NodeID) (source string, external bool, err error) {
	if s == nil {
		return "", false, fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return "", false, fmt.Errorf("script source is unavailable")
	}

	srcValue, ok, err := store.GetAttribute(nodeID, "src")
	if err != nil {
		return "", false, err
	}
	if ok {
		source, err := s.resolveExternalScriptSource(srcValue)
		if err != nil {
			return "", true, err
		}
		return source, true, nil
	}
	return store.TextContentForNode(nodeID), false, nil
}
