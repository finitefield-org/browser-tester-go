package runtime

import (
	"strconv"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

type elementGeometryStyleSnapshot struct {
	position    string
	hasPosition bool
	top         string
	hasTop      bool
	height      string
	hasHeight   bool
	width       string
	hasWidth    bool
}

func resolveElementBoundingClientRectValue(session *Session, store *dom.Store, nodeID dom.NodeID) (script.Value, error) {
	surface := "element:" + strconv.FormatInt(int64(nodeID), 10) + ".getBoundingClientRect"
	if session == nil || store == nil {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}
	node := nodeFromStore(store, nodeID)
	if node == nil || node.Kind != dom.NodeKindElement {
		return script.UndefinedValue(), unsupportedElementSurfaceError(surface)
	}

	snapshot := elementGeometryStyleSnapshotForNode(store, nodeID)
	top := 0.0
	if isPinnedGeometryPosition(snapshot.position) && snapshot.hasTop {
		if value, ok := parseCSSLengthToPixels(snapshot.top); ok {
			top = value
		}
	}
	height := 0.0
	if snapshot.hasHeight {
		if value, ok := parseCSSLengthToPixels(snapshot.height); ok {
			height = value
		}
	}
	width := 0.0
	if snapshot.hasWidth {
		if value, ok := parseCSSLengthToPixels(snapshot.width); ok {
			width = value
		}
	}

	return script.ObjectValue([]script.ObjectEntry{
		{Key: "x", Value: script.NumberValue(0)},
		{Key: "y", Value: script.NumberValue(top)},
		{Key: "top", Value: script.NumberValue(top)},
		{Key: "left", Value: script.NumberValue(0)},
		{Key: "right", Value: script.NumberValue(width)},
		{Key: "bottom", Value: script.NumberValue(top + height)},
		{Key: "width", Value: script.NumberValue(width)},
		{Key: "height", Value: script.NumberValue(height)},
	}), nil
}

func elementGeometryStyleSnapshotForNode(store *dom.Store, nodeID dom.NodeID) elementGeometryStyleSnapshot {
	snapshot := elementGeometryStyleSnapshot{}
	if store == nil {
		return snapshot
	}

	if id, ok := domAttributeValue(store, nodeID, "id"); ok && id != "" {
		for _, candidate := range store.Nodes() {
			if candidate == nil || candidate.Kind != dom.NodeKindElement || candidate.TagName != "style" {
				continue
			}
			applyGeometryStyleText(store.TextContentForNode(candidate.ID), id, &snapshot)
		}
	}

	if declarations := elementStyleDeclarations(store, nodeID); len(declarations) > 0 {
		applyGeometryStyleDeclarations(declarations, &snapshot)
	}

	return snapshot
}

func applyGeometryStyleText(styleText, id string, snapshot *elementGeometryStyleSnapshot) {
	remaining := strings.TrimSpace(styleText)
	for remaining != "" {
		open := strings.IndexByte(remaining, '{')
		if open < 0 {
			return
		}
		selectors := strings.TrimSpace(remaining[:open])
		remaining = remaining[open+1:]
		close := strings.IndexByte(remaining, '}')
		if close < 0 {
			return
		}
		body := remaining[:close]
		remaining = strings.TrimSpace(remaining[close+1:])
		if !geometrySelectorsMatchID(selectors, id) {
			continue
		}
		applyGeometryStyleDeclarations(styleDeclarationsFromText(body), snapshot)
	}
}

func styleDeclarationsFromText(text string) []styleDeclaration {
	parts := splitStyleDeclarations(text)
	if len(parts) == 0 {
		return nil
	}
	declarations := make([]styleDeclaration, 0, len(parts))
	for _, part := range parts {
		declaration, ok := parseStyleDeclaration(part)
		if !ok {
			continue
		}
		declarations = append(declarations, declaration)
	}
	return declarations
}

func applyGeometryStyleDeclarations(declarations []styleDeclaration, snapshot *elementGeometryStyleSnapshot) {
	if snapshot == nil {
		return
	}
	for _, declaration := range declarations {
		snapshot.applyDeclaration(declaration.name, declaration.value)
	}
}

func (snapshot *elementGeometryStyleSnapshot) applyDeclaration(name, value string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "position":
		snapshot.position = strings.ToLower(strings.TrimSpace(value))
		snapshot.hasPosition = true
	case "top":
		snapshot.top = strings.TrimSpace(value)
		snapshot.hasTop = true
	case "height":
		snapshot.height = strings.TrimSpace(value)
		snapshot.hasHeight = true
	case "width":
		snapshot.width = strings.TrimSpace(value)
		snapshot.hasWidth = true
	}
}

func geometrySelectorsMatchID(selectors, id string) bool {
	needle := "#" + strings.TrimSpace(id)
	if needle == "#" {
		return false
	}
	for _, selector := range strings.Split(selectors, ",") {
		if strings.TrimSpace(selector) == needle {
			return true
		}
	}
	return false
}

func parseCSSLengthToPixels(value string) (float64, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || normalized == "auto" {
		return 0, false
	}
	switch {
	case strings.HasSuffix(normalized, "px"):
		value = strings.TrimSpace(strings.TrimSuffix(normalized, "px"))
		if value == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case strings.HasSuffix(normalized, "rem"):
		value = strings.TrimSpace(strings.TrimSuffix(normalized, "rem"))
		if value == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return parsed * 16, true
	case strings.HasSuffix(normalized, "em"):
		value = strings.TrimSpace(strings.TrimSuffix(normalized, "em"))
		if value == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return parsed * 16, true
	default:
		parsed, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
}

func isPinnedGeometryPosition(position string) bool {
	switch strings.ToLower(strings.TrimSpace(position)) {
	case "sticky", "fixed":
		return true
	default:
		return false
	}
}
