package runtime

import (
	"strconv"
	"strings"
)

const defaultBrowserInnerWidth = 1280

func matchMediaQueryAgainstDefaultViewport(query string) (bool, bool) {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return false, false
	}

	inverted := false
	if strings.HasPrefix(normalized, "not ") {
		inverted = !inverted
		normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "not "))
	}
	for _, prefix := range []string{"only ", "screen and ", "all and "} {
		if strings.HasPrefix(normalized, prefix) {
			normalized = strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
			break
		}
	}

	if strings.HasPrefix(normalized, "(") && strings.HasSuffix(normalized, ")") {
		normalized = strings.TrimSpace(normalized[1 : len(normalized)-1])
	}

	parts := strings.SplitN(normalized, ":", 2)
	if len(parts) != 2 {
		return false, false
	}

	feature := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if !strings.HasSuffix(value, "px") {
		return false, false
	}
	width, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "px")), 64)
	if err != nil {
		return false, false
	}

	var matches bool
	switch feature {
	case "max-width":
		matches = float64(defaultBrowserInnerWidth) <= width
	case "min-width":
		matches = float64(defaultBrowserInnerWidth) >= width
	case "width":
		matches = float64(defaultBrowserInnerWidth) == width
	default:
		return false, false
	}

	if inverted {
		matches = !matches
	}
	return matches, true
}
