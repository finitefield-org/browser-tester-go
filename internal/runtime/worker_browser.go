package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/script"
)

const (
	browserWorkerReferencePrefix     = "worker:"
	browserWorkerSelfReferencePrefix = "worker-self:"
)

type browserWorkerState struct {
	id              string
	source          string
	mainProperties   map[string]script.Value
	globalProperties map[string]script.Value
	terminated      bool
}

func browserWorkerInstanceReferenceValue(id string) script.Value {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return script.NullValue()
	}
	return script.HostObjectReference(browserWorkerReferencePrefix + normalized)
}

func browserWorkerSelfReferenceValue(id string) script.Value {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return script.NullValue()
	}
	return script.HostObjectReference(browserWorkerSelfReferencePrefix + normalized)
}

func parseBrowserWorkerReferencePath(path string, prefix string) (id string, suffix string, ok bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, prefix) {
		return "", "", false
	}
	remainder := strings.TrimPrefix(normalized, prefix)
	if remainder == "" {
		return "", "", false
	}
	id = remainder
	suffix = ""
	if index := strings.IndexByte(remainder, '.'); index >= 0 {
		id = remainder[:index]
		suffix = strings.TrimSpace(remainder[index+1:])
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", false
	}
	return id, suffix, true
}

func (s *Session) allocateBrowserWorkerState(source string) string {
	if s == nil {
		return ""
	}
	if s.workerStates == nil {
		s.workerStates = map[string]*browserWorkerState{}
	}
	if s.nextWorkerStateID <= 0 {
		s.nextWorkerStateID = 1
	}
	id := fmt.Sprintf("%d", s.nextWorkerStateID)
	s.nextWorkerStateID++
	s.workerStates[id] = &browserWorkerState{
		id:     id,
		source: source,
	}
	return id
}

func (s *Session) browserWorkerStateByID(id string) (*browserWorkerState, bool) {
	if s == nil || s.workerStates == nil {
		return nil, false
	}
	state, ok := s.workerStates[strings.TrimSpace(id)]
	return state, ok
}

func (s *Session) deleteBrowserWorkerState(id string) {
	if s == nil || s.workerStates == nil {
		return
	}
	delete(s.workerStates, strings.TrimSpace(id))
}

func browserWorkerConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Worker is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 || len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("Worker expects 1 or 2 arguments")
	}

	sourceURL := script.ToJSString(args[0])
	source, err := browserWorkerSourceFromURL(session, sourceURL)
	if err != nil {
		return script.UndefinedValue(), err
	}

	id := session.allocateBrowserWorkerState(source)
	if strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), fmt.Errorf("Worker constructor could not allocate state")
	}
	state, ok := session.browserWorkerStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), fmt.Errorf("Worker constructor could not initialize state")
	}

	workerHost := &browserWorkerHost{
		session: session,
		state:   state,
	}
	runtime := script.NewRuntimeWithBindings(workerHost, workerGlobalBindings(session, id))
	if _, err := runtime.Dispatch(script.DispatchRequest{Source: source}); err != nil {
		session.deleteBrowserWorkerState(id)
		return script.UndefinedValue(), err
	}

	return browserWorkerInstanceReferenceValue(id), nil
}

func browserWorkerSourceFromURL(session *Session, rawURL string) (string, error) {
	normalized := strings.TrimSpace(rawURL)
	id, _, ok := parseBrowserBlobInstancePath(normalized)
	if !ok {
		return "", script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", rawURL))
	}
	state, ok := session.browserBlobStateByID(id)
	if !ok || state == nil {
		return "", script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid Blob reference %q in this bounded classic-JS slice", rawURL))
	}
	return string(state.bytes), nil
}

func workerGlobalBindings(session *Session, id string) map[string]script.Value {
	bindings := browserGlobalBindings(session, nil)
	selfRef := browserWorkerSelfReferenceValue(id)
	bindings["self"] = selfRef
	bindings["globalThis"] = selfRef
	bindings["window"] = selfRef
	bindings["top"] = selfRef
	bindings["parent"] = selfRef
	bindings["frames"] = selfRef
	bindings["postMessage"] = script.HostFunctionReference(browserWorkerSelfReferencePrefix + id + ".postMessage")
	bindings["close"] = script.HostFunctionReference(browserWorkerSelfReferencePrefix + id + ".close")
	bindings["onmessage"] = script.HostObjectReference(browserWorkerSelfReferencePrefix + id + ".onmessage")
	return bindings
}

type browserWorkerHost struct {
	session *Session
	state   *browserWorkerState
}

func (h *browserWorkerHost) Call(method string, args []script.Value) (script.Value, error) {
	return script.UndefinedValue(), fmt.Errorf("worker script host method %q is not configured", method)
}

func (h *browserWorkerHost) ResolveHostReference(path string) (script.Value, error) {
	if h == nil || h.session == nil || h.state == nil {
		return script.UndefinedValue(), fmt.Errorf("worker script host is unavailable")
	}

	plainName := strings.TrimSpace(path)
	if plainName == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}

	if id, suffix, ok := parseBrowserWorkerReferencePath(plainName, browserWorkerSelfReferencePrefix); ok {
		if id != h.state.id {
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid worker reference %q in this bounded classic-JS slice", path))
		}
		return resolveBrowserWorkerSelfReference(h.session, h.state, suffix)
	}

	if value, ok := h.state.globalProperties[plainName]; ok {
		return value, nil
	}

	if knownBrowserGlobalBinding(h.session, plainName) {
		return resolveBrowserGlobalReference(h.session, nil, plainName)
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
}

func (h *browserWorkerHost) SetHostReference(path string, value script.Value) error {
	if h == nil || h.session == nil || h.state == nil {
		return fmt.Errorf("worker script host is unavailable")
	}

	plainName := strings.TrimSpace(path)
	if plainName == "" {
		return script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}

	if id, suffix, ok := parseBrowserWorkerReferencePath(plainName, browserWorkerSelfReferencePrefix); ok {
		if id != h.state.id {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid worker reference %q in this bounded classic-JS slice", path))
		}
		if suffix == "" {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
		}
		return setBrowserWorkerGlobalValue(h.state, suffix, value)
	}

	return setBrowserWorkerGlobalValue(h.state, plainName, value)
}

func (h *browserWorkerHost) DeleteHostReference(path string) error {
	if h == nil || h.session == nil || h.state == nil {
		return fmt.Errorf("worker script host is unavailable")
	}

	plainName := strings.TrimSpace(path)
	if plainName == "" {
		return script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}

	if id, suffix, ok := parseBrowserWorkerReferencePath(plainName, browserWorkerSelfReferencePrefix); ok {
		if id != h.state.id {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid worker reference %q in this bounded classic-JS slice", path))
		}
		if suffix == "" {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
		}
		if !deleteBrowserWorkerGlobalValue(h.state, suffix) {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
		}
		return nil
	}

	if !deleteBrowserWorkerGlobalValue(h.state, plainName) {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	return nil
}

func knownBrowserGlobalBinding(session *Session, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	_, ok := browserGlobalBindings(session, nil)[name]
	return ok
}

func setBrowserWorkerGlobalValue(state *browserWorkerState, name string, value script.Value) error {
	if state == nil {
		return script.NewError(script.ErrorKindUnsupported, "worker is unavailable in this bounded classic-JS slice")
	}
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return script.NewError(script.ErrorKindUnsupported, "unsupported browser surface \"\" in this bounded classic-JS slice")
	}
	if state.globalProperties == nil {
		state.globalProperties = map[string]script.Value{}
	}
	state.globalProperties[normalized] = value
	return nil
}

func deleteBrowserWorkerGlobalValue(state *browserWorkerState, name string) bool {
	if state == nil || len(state.globalProperties) == 0 {
		return false
	}
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return false
	}
	if _, ok := state.globalProperties[normalized]; !ok {
		return false
	}
	delete(state.globalProperties, normalized)
	return true
}

func resolveBrowserWorkerSelfReference(session *Session, state *browserWorkerState, path string) (script.Value, error) {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return browserWorkerSelfReferenceValue(state.id), nil
	}
	if value, ok := state.globalProperties[normalized]; ok {
		return value, nil
	}

	switch normalized {
	case "self", "window", "globalThis", "top", "parent", "frames":
		return browserWorkerSelfReferenceValue(state.id), nil
	case "onmessage":
		return script.NullValue(), nil
	case "postMessage":
		return script.NativeNamedFunctionValue("postMessage", func(args []script.Value) (script.Value, error) {
			return browserWorkerSendToMain(session, state, args)
		}), nil
	case "close", "terminate":
		return script.NativeNamedFunctionValue(normalized, func(args []script.Value) (script.Value, error) {
			return browserWorkerTerminate(session, state)
		}), nil
	}

	if knownBrowserGlobalBinding(session, normalized) {
		return resolveBrowserGlobalReference(session, nil, normalized)
	}

	return script.UndefinedValue(), nil
}

func browserWorkerMainReferenceValue(state *browserWorkerState) script.Value {
	if state == nil {
		return script.NullValue()
	}
	return browserWorkerInstanceReferenceValue(state.id)
}

func resolveBrowserWorkerReference(session *Session, path string) (script.Value, error) {
	id, suffix, ok := parseBrowserWorkerReferencePath(path, browserWorkerReferencePrefix)
	if !ok {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserWorkerStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid worker reference %q in this bounded classic-JS slice", path))
	}
	if suffix == "" {
		return browserWorkerInstanceReferenceValue(id), nil
	}
	if value, ok := state.mainProperties[suffix]; ok {
		return value, nil
	}

	switch suffix {
	case "self", "window", "globalThis", "top", "parent", "frames":
		return browserWorkerInstanceReferenceValue(id), nil
	case "onmessage":
		return script.NullValue(), nil
	case "postMessage":
		return script.NativeNamedFunctionValue("postMessage", func(args []script.Value) (script.Value, error) {
			return browserWorkerSendToWorker(session, state, args)
		}), nil
	case "terminate":
		return script.NativeNamedFunctionValue("terminate", func(args []script.Value) (script.Value, error) {
			return browserWorkerTerminate(session, state)
		}), nil
	}

	return script.UndefinedValue(), nil
}

func setBrowserWorkerReferenceValue(session *Session, path string, value script.Value) error {
	id, suffix, ok := parseBrowserWorkerReferencePath(path, browserWorkerReferencePrefix)
	if !ok {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
	state, ok := session.browserWorkerStateByID(id)
	if !ok || state == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid worker reference %q in this bounded classic-JS slice", path))
	}
	if suffix == "" {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
	switch suffix {
	case "self", "window", "globalThis", "top", "parent", "frames":
		return script.NewError(script.ErrorKindUnsupported, "worker aliases are read-only in this bounded classic-JS slice")
	case "onmessage":
		if state.mainProperties == nil {
			state.mainProperties = map[string]script.Value{}
		}
		state.mainProperties[suffix] = value
		return nil
	default:
		if state.mainProperties == nil {
			state.mainProperties = map[string]script.Value{}
		}
		state.mainProperties[suffix] = value
		return nil
	}
}

func deleteBrowserWorkerReferenceValue(session *Session, path string) error {
	id, suffix, ok := parseBrowserWorkerReferencePath(path, browserWorkerReferencePrefix)
	if !ok {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	state, ok := session.browserWorkerStateByID(id)
	if !ok || state == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid worker reference %q in this bounded classic-JS slice", path))
	}
	if suffix == "" {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	switch suffix {
	case "self", "window", "globalThis", "top", "parent", "frames":
		return script.NewError(script.ErrorKindUnsupported, "worker aliases are read-only in this bounded classic-JS slice")
	default:
		if state.mainProperties == nil {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
		}
		if _, ok := state.mainProperties[suffix]; !ok {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
		}
		delete(state.mainProperties, suffix)
		return nil
	}
}

func browserWorkerSendToWorker(session *Session, state *browserWorkerState, args []script.Value) (script.Value, error) {
	if session == nil || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Worker is unavailable in this bounded classic-JS slice")
	}
	if state.terminated {
		return script.UndefinedValue(), nil
	}
	message := script.UndefinedValue()
	if len(args) > 0 {
		message = args[0]
	}
	handler := state.globalProperties["onmessage"]
	if handler.Kind != script.ValueKindFunction || (handler.NativeFunction == nil && handler.Function == nil) {
		return script.UndefinedValue(), nil
	}
	event := script.ObjectValue([]script.ObjectEntry{
		{Key: "data", Value: message},
	})
	if _, err := script.InvokeCallableValue(&browserWorkerHost{session: session, state: state}, handler, []script.Value{event}, browserWorkerSelfReferenceValue(state.id), true); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserWorkerSendToMain(session *Session, state *browserWorkerState, args []script.Value) (script.Value, error) {
	if session == nil || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Worker is unavailable in this bounded classic-JS slice")
	}
	if state.terminated {
		return script.UndefinedValue(), nil
	}
	message := script.UndefinedValue()
	if len(args) > 0 {
		message = args[0]
	}
	handler := state.mainProperties["onmessage"]
	if handler.Kind != script.ValueKindFunction || (handler.NativeFunction == nil && handler.Function == nil) {
		return script.UndefinedValue(), nil
	}
	event := script.ObjectValue([]script.ObjectEntry{
		{Key: "data", Value: message},
	})
	if _, err := script.InvokeCallableValue(&inlineScriptHost{session: session}, handler, []script.Value{event}, browserWorkerInstanceReferenceValue(state.id), true); err != nil {
		return script.UndefinedValue(), err
	}
	return script.UndefinedValue(), nil
}

func browserWorkerTerminate(session *Session, state *browserWorkerState) (script.Value, error) {
	if session == nil || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Worker is unavailable in this bounded classic-JS slice")
	}
	state.terminated = true
	return script.UndefinedValue(), nil
}
