package runtime

import (
	"fmt"
	"reflect"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

type eventPhase string

const (
	eventPhaseTarget  eventPhase = "target"
	eventPhaseCapture eventPhase = "capture"
	eventPhaseBubble  eventPhase = "bubble"
)

type eventListenerRecord struct {
	id               int64
	nodeID           dom.NodeID
	currentTarget    script.Value
	currentTargetKey string
	event            string
	phase            eventPhase
	source           string
	callable         script.Value
	callableKey      string
	once             bool
}

type eventDispatchContext struct {
	store              *dom.Store
	targetNodeID       dom.NodeID
	eventType          string
	key                string
	dataTransferID     string
	defaultPrevented   bool
	propagationStopped bool
}

type EventListenerRegistration struct {
	NodeID int64
	Event  string
	Phase  string
	Source string
	Once   bool
}

func normalizeEventType(event string) string {
	return strings.ToLower(strings.TrimSpace(event))
}

func browserEventTypeUsesDataTransfer(eventType string) bool {
	switch normalizeEventType(eventType) {
	case "dragstart", "dragover", "drop", "dragenter", "dragleave", "dragend":
		return true
	default:
		return false
	}
}

func browserEventTypeUsesClipboardData(eventType string) bool {
	switch normalizeEventType(eventType) {
	case "paste", "copy", "cut":
		return true
	default:
		return false
	}
}

func parseEventPhase(phase string) (eventPhase, error) {
	normalized := strings.ToLower(strings.TrimSpace(phase))
	switch normalized {
	case "", string(eventPhaseTarget):
		return eventPhaseTarget, nil
	case string(eventPhaseCapture):
		return eventPhaseCapture, nil
	case string(eventPhaseBubble):
		return eventPhaseBubble, nil
	default:
		return "", fmt.Errorf("unsupported event listener phase %q", phase)
	}
}

func (s *Session) registerEventListener(nodeID dom.NodeID, event, source, phase string, once bool) error {
	return s.registerEventListenerValue(nodeID, browserElementReferenceValue(nodeID, s.domStore), event, script.StringValue(source), phase, once)
}

func (s *Session) registerEventListenerValue(nodeID dom.NodeID, currentTarget script.Value, event string, listener script.Value, phase string, once bool) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	normalized := normalizeEventType(event)
	if normalized == "" {
		return fmt.Errorf("event type must not be empty")
	}
	normalizedPhase, err := parseEventPhase(phase)
	if err != nil {
		return err
	}
	if currentTarget.Kind == script.ValueKindUndefined {
		currentTarget = browserElementReferenceValue(nodeID, s.domStore)
	}
	source, callableKey, err := eventListenerSourceAndKey(listener)
	if err != nil {
		return err
	}

	s.nextEventListenerID++
	id := s.nextEventListenerID
	s.eventListeners = append(s.eventListeners, eventListenerRecord{
		id:               id,
		nodeID:           nodeID,
		currentTarget:    currentTarget,
		currentTargetKey: eventListenerReferenceKey(currentTarget),
		event:            normalized,
		phase:            normalizedPhase,
		source:           source,
		callable:         listener,
		callableKey:      callableKey,
		once:             once,
	})
	return nil
}

func (s *Session) EventListeners() []EventListenerRegistration {
	if s == nil {
		return nil
	}
	if _, err := s.ensureDOM(); err != nil {
		return nil
	}
	if len(s.eventListeners) == 0 {
		return nil
	}
	out := make([]EventListenerRegistration, len(s.eventListeners))
	for i := range s.eventListeners {
		out[i] = EventListenerRegistration{
			NodeID: int64(s.eventListeners[i].nodeID),
			Event:  s.eventListeners[i].event,
			Phase:  string(s.eventListeners[i].phase),
			Source: func() string {
				if s.eventListeners[i].source != "" {
					return s.eventListeners[i].source
				}
				return s.eventListeners[i].callableKey
			}(),
			Once: s.eventListeners[i].once,
		}
	}
	return out
}

func (s *Session) dispatchEventListeners(store *dom.Store, nodeID dom.NodeID, event string) (bool, error) {
	return s.dispatchEventListenersWithPropagation(store, nodeID, event, "", true, true)
}

func (s *Session) dispatchTargetEventListeners(store *dom.Store, nodeID dom.NodeID, event string) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return false, fmt.Errorf("dom store is unavailable")
	}

	normalized := normalizeEventType(event)
	if normalized == "" {
		return false, nil
	}

	prev := s.eventDispatch
	ctx := &eventDispatchContext{
		store:        store,
		targetNodeID: nodeID,
		eventType:    normalized,
	}
	if browserEventTypeUsesDataTransfer(normalized) {
		ctx.dataTransferID = s.allocateBrowserDataTransferState(nil)
	}
	s.eventDispatch = ctx
	defer func() {
		s.eventDispatch = prev
	}()

	currentTarget := browserNodeReferenceValue(store, nodeID)
	// Target-only events still need the target's own capture/target/bubble listeners.
	if err := s.dispatchListenersOnTarget(store, nodeID, currentTarget, normalized, eventPhaseCapture); err != nil {
		return ctx.defaultPrevented, err
	}
	if err := s.dispatchListenersOnTarget(store, nodeID, currentTarget, normalized, eventPhaseTarget); err != nil {
		return ctx.defaultPrevented, err
	}
	if err := s.dispatchListenersOnTarget(store, nodeID, currentTarget, normalized, eventPhaseBubble); err != nil {
		return ctx.defaultPrevented, err
	}
	return ctx.defaultPrevented, nil
}

func (s *Session) dispatchEventListenersWithPropagation(store *dom.Store, nodeID dom.NodeID, event, key string, capture, bubble bool) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return false, fmt.Errorf("dom store is unavailable")
	}

	normalized := normalizeEventType(event)
	if normalized == "" {
		return false, nil
	}

	prev := s.eventDispatch
	ctx := &eventDispatchContext{
		store:        store,
		targetNodeID: nodeID,
		eventType:    normalized,
		key:          key,
	}
	if browserEventTypeUsesDataTransfer(normalized) {
		ctx.dataTransferID = s.allocateBrowserDataTransferState(nil)
	}
	s.eventDispatch = ctx
	defer func() {
		s.eventDispatch = prev
	}()

	path := s.eventPath(store, nodeID)
	if capture {
		if err := s.dispatchListenersOnTarget(store, store.DocumentID(), browserHostObjectValue("window"), normalized, eventPhaseCapture); err != nil {
			return ctx.defaultPrevented, err
		}
		if !ctx.propagationStopped {
			for i := len(path) - 1; i >= 0; i-- {
				if err := s.dispatchListenersOnTarget(store, path[i], browserNodeReferenceValue(store, path[i]), normalized, eventPhaseCapture); err != nil {
					return ctx.defaultPrevented, err
				}
				if s.domStore != nil && s.domStore != store {
					return ctx.defaultPrevented, nil
				}
				if ctx.propagationStopped {
					break
				}
			}
		}
	}
	if err := s.dispatchListenersOnTarget(store, nodeID, browserNodeReferenceValue(store, nodeID), normalized, eventPhaseTarget); err != nil {
		return ctx.defaultPrevented, err
	}
	if s.domStore != nil && s.domStore != store {
		return ctx.defaultPrevented, nil
	}
	if bubble && !ctx.propagationStopped {
		for i := 0; i < len(path); i++ {
			if err := s.dispatchListenersOnTarget(store, path[i], browserNodeReferenceValue(store, path[i]), normalized, eventPhaseBubble); err != nil {
				return ctx.defaultPrevented, err
			}
			if s.domStore != nil && s.domStore != store {
				return ctx.defaultPrevented, nil
			}
			if ctx.propagationStopped {
				break
			}
		}
		if !ctx.propagationStopped {
			if err := s.dispatchListenersOnTarget(store, store.DocumentID(), browserHostObjectValue("window"), normalized, eventPhaseBubble); err != nil {
				return ctx.defaultPrevented, err
			}
		}
	}
	return ctx.defaultPrevented, nil
}

func (s *Session) dispatchListenersOnTarget(store *dom.Store, nodeID dom.NodeID, currentTarget script.Value, event string, phase eventPhase) error {
	listeners := s.listenersForEvent(nodeID, eventListenerReferenceKey(currentTarget), event, phase)
	for _, listener := range listeners {
		if s.domStore != nil && s.domStore != store {
			return nil
		}
		if !s.eventListenerExists(listener.id) {
			continue
		}
		if listener.source != "" {
			if _, err := s.runScriptOnStore(store, listener.source); err != nil {
				if listener.once {
					s.removeEventListenerByID(listener.id)
				}
				return err
			}
		} else {
			currentTarget := listener.currentTarget
			if currentTarget.Kind == script.ValueKindUndefined || (currentTarget.Kind == script.ValueKindHostReference && strings.HasPrefix(currentTarget.HostReferencePath, "element:") && !strings.Contains(currentTarget.HostReferencePath, "@")) {
				currentTarget = browserNodeReferenceValue(store, nodeID)
			}
			eventValue := browserEventObjectValue(s, store, listener, currentTarget)
			if _, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, listener.callable, []script.Value{eventValue}, currentTarget, true); err != nil {
				if listener.once {
					s.removeEventListenerByID(listener.id)
				}
				return err
			}
		}
		if s.domStore != nil && s.domStore != store {
			return nil
		}
		if listener.once {
			s.removeEventListenerByID(listener.id)
		}
	}
	return nil
}

func (s *Session) listenersForEvent(nodeID dom.NodeID, currentTargetKey, event string, phase eventPhase) []eventListenerRecord {
	if s == nil || len(s.eventListeners) == 0 {
		return nil
	}

	out := make([]eventListenerRecord, 0, len(s.eventListeners))
	for _, listener := range s.eventListeners {
		if listener.nodeID == nodeID && listener.currentTargetKey == currentTargetKey && listener.event == event && listener.phase == phase {
			out = append(out, listener)
		}
	}
	return out
}

func (s *Session) eventPath(store *dom.Store, nodeID dom.NodeID) []dom.NodeID {
	if s == nil || store == nil || nodeID == 0 {
		return nil
	}

	path := make([]dom.NodeID, 0, 4)
	current := nodeID
	for current != 0 {
		node := store.Node(current)
		if node == nil {
			break
		}
		path = append(path, current)
		current = node.Parent
	}
	return path
}

func (s *Session) eventListenerExists(id int64) bool {
	if s == nil || len(s.eventListeners) == 0 {
		return false
	}
	for _, listener := range s.eventListeners {
		if listener.id == id {
			return true
		}
	}
	return false
}

func (s *Session) removeEventListenerByID(id int64) {
	if s == nil || len(s.eventListeners) == 0 {
		return
	}
	for i, listener := range s.eventListeners {
		if listener.id != id {
			continue
		}
		copy(s.eventListeners[i:], s.eventListeners[i+1:])
		s.eventListeners = s.eventListeners[:len(s.eventListeners)-1]
		return
	}
}

func (s *Session) removeEventListener(nodeID dom.NodeID, event, source, phase string) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("session is unavailable")
	}
	normalized := normalizeEventType(event)
	if normalized == "" {
		return false, fmt.Errorf("event type must not be empty")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return false, fmt.Errorf("event listener source must not be empty")
	}
	normalizedPhase, err := parseEventPhase(phase)
	if err != nil {
		return false, err
	}
	currentTargetKey := eventListenerReferenceKey(browserElementReferenceValue(nodeID, s.domStore))

	for i, listener := range s.eventListeners {
		if listener.nodeID != nodeID || listener.event != normalized || listener.phase != normalizedPhase || listener.currentTargetKey != currentTargetKey || listener.source != source {
			continue
		}
		copy(s.eventListeners[i:], s.eventListeners[i+1:])
		s.eventListeners = s.eventListeners[:len(s.eventListeners)-1]
		return true, nil
	}
	return false, nil
}

func (s *Session) removeEventListenerValue(nodeID dom.NodeID, currentTarget script.Value, event string, listener script.Value, phase string) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("session is unavailable")
	}
	normalized := normalizeEventType(event)
	if normalized == "" {
		return false, fmt.Errorf("event type must not be empty")
	}
	normalizedPhase, err := parseEventPhase(phase)
	if err != nil {
		return false, err
	}
	if currentTarget.Kind == script.ValueKindUndefined {
		currentTarget = browserElementReferenceValue(nodeID, s.domStore)
	}
	source, callableKey, err := eventListenerSourceAndKey(listener)
	if err != nil {
		return false, err
	}
	currentTargetKey := eventListenerReferenceKey(currentTarget)

	for i, stored := range s.eventListeners {
		if stored.nodeID != nodeID || stored.event != normalized || stored.phase != normalizedPhase || stored.currentTargetKey != currentTargetKey {
			continue
		}
		if stored.source != "" {
			if stored.source != source {
				continue
			}
		} else if stored.callableKey != callableKey {
			continue
		}
		copy(s.eventListeners[i:], s.eventListeners[i+1:])
		s.eventListeners = s.eventListeners[:len(s.eventListeners)-1]
		return true, nil
	}
	return false, nil
}

func (s *Session) preventDefault() error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if s.eventDispatch == nil {
		return fmt.Errorf("preventDefault() requires an active event dispatch")
	}
	s.eventDispatch.defaultPrevented = true
	return nil
}

func (s *Session) stopPropagation() error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if s.eventDispatch == nil {
		return fmt.Errorf("stopPropagation() requires an active event dispatch")
	}
	s.eventDispatch.propagationStopped = true
	return nil
}

func (s *Session) eventTargetValue() (string, error) {
	if s == nil {
		return "", fmt.Errorf("session is unavailable")
	}
	if s.eventDispatch == nil {
		return "", fmt.Errorf("eventTargetValue() requires an active event dispatch")
	}
	if s.eventDispatch.targetNodeID == 0 {
		return "", fmt.Errorf("event target node is unavailable")
	}

	store := s.eventDispatch.store
	if store == nil {
		return "", fmt.Errorf("event target store is unavailable")
	}
	if store.Node(s.eventDispatch.targetNodeID) == nil {
		return "", fmt.Errorf("event target node is unavailable")
	}
	return store.ValueForNode(s.eventDispatch.targetNodeID), nil
}

func eventListenerSourceAndKey(listener script.Value) (string, string, error) {
	switch listener.Kind {
	case script.ValueKindString:
		trimmed := strings.TrimSpace(listener.String)
		if trimmed == "" {
			return "", "", fmt.Errorf("event listener source must not be empty")
		}
		return trimmed, "", nil
	case script.ValueKindFunction:
		if listener.Function != nil {
			return "", fmt.Sprintf("function:%x", reflect.ValueOf(listener.Function).Pointer()), nil
		}
		if listener.NativeFunction != nil {
			return "", fmt.Sprintf("native:%x", reflect.ValueOf(listener.NativeFunction).Pointer()), nil
		}
	case script.ValueKindHostReference:
		if listener.HostReferenceKind == script.HostReferenceKindFunction || listener.HostReferenceKind == script.HostReferenceKindConstructor {
			return "", eventListenerReferenceKey(listener), nil
		}
	}
	return "", "", fmt.Errorf("event listener must be callable")
}

func eventListenerReferenceKey(value script.Value) string {
	switch value.Kind {
	case script.ValueKindHostReference:
		path := value.HostReferencePath
		if strings.HasPrefix(path, "element:") {
			path = canonicalElementReferencePath(path)
		}
		return string(value.HostReferenceKind) + ":" + path
	case script.ValueKindFunction:
		if value.Function != nil {
			return fmt.Sprintf("function:%x", reflect.ValueOf(value.Function).Pointer())
		}
		if value.NativeFunction != nil {
			return fmt.Sprintf("native:%x", reflect.ValueOf(value.NativeFunction).Pointer())
		}
	}
	return ""
}

func browserEventObjectValue(session *Session, store *dom.Store, listener eventListenerRecord, currentTarget script.Value) script.Value {
	target := script.NullValue()
	if session != nil && store != nil && session.eventDispatch != nil && session.eventDispatch.targetNodeID != 0 {
		target = browserNodeReferenceValue(store, session.eventDispatch.targetNodeID)
	}

	entries := []script.ObjectEntry{
		{Key: "type", Value: script.StringValue(listener.event)},
		{Key: "target", Value: target},
		{Key: "currentTarget", Value: currentTarget},
		{Key: "defaultPrevented", Value: script.BoolValue(session != nil && session.eventDispatch != nil && session.eventDispatch.defaultPrevented)},
		{Key: "preventDefault", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), fmt.Errorf("event.preventDefault is unavailable")
			}
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("event.preventDefault accepts no arguments")
			}
			if err := session.preventDefault(); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		})},
		{Key: "stopPropagation", Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if session == nil {
				return script.UndefinedValue(), fmt.Errorf("event.stopPropagation is unavailable")
			}
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("event.stopPropagation accepts no arguments")
			}
			if err := session.stopPropagation(); err != nil {
				return script.UndefinedValue(), err
			}
			return script.UndefinedValue(), nil
		})},
	}
	if session != nil && session.eventDispatch != nil && strings.TrimSpace(session.eventDispatch.dataTransferID) != "" {
		entries = append(entries, script.ObjectEntry{Key: "dataTransfer", Value: browserDataTransferReferenceValue(session.eventDispatch.dataTransferID)})
	}
	if session != nil && session.eventDispatch != nil && browserEventTypeUsesClipboardData(session.eventDispatch.eventType) {
		entries = append(entries, script.ObjectEntry{Key: "clipboardData", Value: browserClipboardDataReferenceValue()})
	}
	if session != nil && session.eventDispatch != nil && session.eventDispatch.key != "" {
		entries = append(entries, script.ObjectEntry{Key: "key", Value: script.StringValue(session.eventDispatch.key)})
		entries = append(entries,
			script.ObjectEntry{Key: "ctrlKey", Value: script.BoolValue(true)},
			script.ObjectEntry{Key: "metaKey", Value: script.BoolValue(false)},
			script.ObjectEntry{Key: "shiftKey", Value: script.BoolValue(false)},
			script.ObjectEntry{Key: "altKey", Value: script.BoolValue(false)},
		)
	}
	return script.ObjectValue(entries)
}

func browserNodeReferenceValue(store *dom.Store, nodeID dom.NodeID) script.Value {
	if store != nil && nodeID == store.DocumentID() {
		return script.HostObjectReference("document")
	}
	return browserElementReferenceValue(nodeID, store)
}
