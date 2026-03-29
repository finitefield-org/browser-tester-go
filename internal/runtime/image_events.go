package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func browserScheduleImageSourceEvent(session *Session, nodeID dom.NodeID, source string) error {
	if session == nil {
		return nil
	}
	if strings.TrimSpace(source) == "" {
		return nil
	}
	callback := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if len(args) > 0 {
			return script.UndefinedValue(), fmt.Errorf("image load callback accepts no arguments")
		}
		return script.UndefinedValue(), browserDispatchImageSourceEvent(session, nodeID, source)
	})
	_, err := session.scheduleTimeoutCallback(callback, nil, 0)
	return err
}

func browserDispatchImageSourceEvent(session *Session, nodeID dom.NodeID, source string) error {
	if session == nil {
		return fmt.Errorf("session is unavailable")
	}
	store := session.domStore
	if store == nil {
		return nil
	}
	node := store.Node(nodeID)
	if node == nil || node.Kind != dom.NodeKindElement || node.TagName != "img" {
		return nil
	}
	currentSource, ok := domAttributeValue(store, nodeID, "src")
	if !ok || currentSource != source {
		return nil
	}

	eventType := browserImageSourceEventType(session, source)
	prevented, err := session.dispatchTargetEventListeners(store, nodeID, eventType)
	if err != nil {
		return err
	}
	if session.domStore != nil && session.domStore != store {
		return nil
	}
	return session.dispatchElementEventHandler(store, nodeID, eventType, prevented)
}

func browserImageSourceEventType(session *Session, source string) string {
	normalized := strings.TrimSpace(source)
	if normalized == "" {
		return ""
	}

	trimmedLower := strings.ToLower(normalized)
	if strings.HasPrefix(trimmedLower, "data:image/") {
		return "load"
	}
	if strings.HasPrefix(normalized, "blob:") {
		id, _, ok := parseBrowserBlobInstancePath(normalized)
		if !ok || session == nil {
			return "error"
		}
		state, ok := session.browserBlobStateByID(id)
		if !ok || state == nil {
			return "error"
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(state.mimeType)), "image/") {
			return "load"
		}
		return "error"
	}
	return "error"
}
