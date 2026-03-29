package runtime

import (
	"fmt"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func isCallableEventHandlerValue(value script.Value) bool {
	switch value.Kind {
	case script.ValueKindFunction:
		return value.Function != nil || value.NativeFunction != nil
	case script.ValueKindHostReference:
		return value.HostReferenceKind == script.HostReferenceKindFunction || value.HostReferenceKind == script.HostReferenceKindConstructor
	default:
		return false
	}
}

func (s *Session) elementEventHandler(nodeID dom.NodeID, event string) (script.Value, bool) {
	if s == nil || len(s.elementEventHandlers) == 0 {
		return script.UndefinedValue(), false
	}
	normalized := normalizeEventType(event)
	if normalized == "" {
		return script.UndefinedValue(), false
	}
	handlers, ok := s.elementEventHandlers[nodeID]
	if !ok || len(handlers) == 0 {
		return script.UndefinedValue(), false
	}
	value, ok := handlers[normalized]
	if !ok {
		return script.UndefinedValue(), false
	}
	return value, true
}

func (s *Session) setElementEventHandler(nodeID dom.NodeID, event string, value script.Value) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	normalized := normalizeEventType(event)
	if normalized == "" {
		return fmt.Errorf("element event type must not be empty")
	}
	if value.Kind == script.ValueKindUndefined || value.Kind == script.ValueKindNull {
		s.deleteElementEventHandler(nodeID, normalized)
		return nil
	}
	if !isCallableEventHandlerValue(value) {
		return fmt.Errorf("element.on%s expects a callable or null in this bounded classic-JS slice", normalized)
	}
	if s.elementEventHandlers == nil {
		s.elementEventHandlers = make(map[dom.NodeID]map[string]script.Value)
	}
	handlers := s.elementEventHandlers[nodeID]
	if handlers == nil {
		handlers = make(map[string]script.Value)
		s.elementEventHandlers[nodeID] = handlers
	}
	handlers[normalized] = value
	return nil
}

func (s *Session) deleteElementEventHandler(nodeID dom.NodeID, event string) bool {
	if s == nil || len(s.elementEventHandlers) == 0 {
		return false
	}
	normalized := normalizeEventType(event)
	if normalized == "" {
		return false
	}
	handlers, ok := s.elementEventHandlers[nodeID]
	if !ok || len(handlers) == 0 {
		return false
	}
	if _, ok := handlers[normalized]; !ok {
		return false
	}
	delete(handlers, normalized)
	if len(handlers) == 0 {
		delete(s.elementEventHandlers, nodeID)
	}
	return true
}

func cloneElementEventHandlerMap(handlers map[dom.NodeID]map[string]script.Value) map[dom.NodeID]map[string]script.Value {
	if len(handlers) == 0 {
		return nil
	}
	out := make(map[dom.NodeID]map[string]script.Value, len(handlers))
	for nodeID, events := range handlers {
		if len(events) == 0 {
			out[nodeID] = nil
			continue
		}
		cloned := make(map[string]script.Value, len(events))
		for event, value := range events {
			cloned[event] = value
		}
		out[nodeID] = cloned
	}
	return out
}

func (s *Session) dispatchElementEventHandler(store *dom.Store, nodeID dom.NodeID, event string, defaultPrevented bool) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return fmt.Errorf("dom store is unavailable")
	}
	handler, ok := s.elementEventHandler(nodeID, event)
	if !ok {
		return nil
	}

	prev := s.eventDispatch
	ctx := &eventDispatchContext{
		store:            store,
		targetNodeID:     nodeID,
		eventType:        normalizeEventType(event),
		defaultPrevented: defaultPrevented,
	}
	s.eventDispatch = ctx
	defer func() {
		s.eventDispatch = prev
	}()

	currentTarget := browserNodeReferenceValue(store, nodeID)
	eventValue := browserEventObjectValue(s, store, eventListenerRecord{event: normalizeEventType(event)}, currentTarget)
	_, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, handler, []script.Value{eventValue}, currentTarget, true)
	return err
}
