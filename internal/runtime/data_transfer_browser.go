package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/script"
)

const browserDataTransferReferencePrefix = "datatransfer:"
const browserDataTransferItemsReferencePrefix = "datatransfer-items:"

type browserDataTransferState struct {
	files         []script.Value
	effectAllowed string
	dropEffect    string
}

func browserDataTransferReferenceValue(id string) script.Value {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return script.NullValue()
	}
	return script.HostObjectReference(browserDataTransferReferencePrefix + normalized)
}

func browserDataTransferItemsReferenceValue(id string) script.Value {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return script.NullValue()
	}
	return script.HostObjectReference(browserDataTransferItemsReferencePrefix + normalized)
}

func parseBrowserDataTransferInstancePath(path string) (string, string, bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, browserDataTransferReferencePrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(normalized, browserDataTransferReferencePrefix)
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

func parseBrowserDataTransferItemsInstancePath(path string) (string, string, bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, browserDataTransferItemsReferencePrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(normalized, browserDataTransferItemsReferencePrefix)
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

func (s *Session) allocateBrowserDataTransferState(files []script.Value) string {
	if s == nil {
		return ""
	}
	if s.dataTransferStates == nil {
		s.dataTransferStates = map[string]*browserDataTransferState{}
	}
	if s.nextDataTransferStateID <= 0 {
		s.nextDataTransferStateID = 1
	}
	id := fmt.Sprintf("%d", s.nextDataTransferStateID)
	s.nextDataTransferStateID++
	copied := append([]script.Value(nil), files...)
	s.dataTransferStates[id] = &browserDataTransferState{
		files: copied,
	}
	return id
}

func (s *Session) browserDataTransferStateByID(id string) (*browserDataTransferState, bool) {
	if s == nil || s.dataTransferStates == nil {
		return nil, false
	}
	state, ok := s.dataTransferStates[strings.TrimSpace(id)]
	return state, ok
}

func (s *Session) deleteBrowserDataTransferState(id string) {
	if s == nil || s.dataTransferStates == nil {
		return
	}
	delete(s.dataTransferStates, strings.TrimSpace(id))
}

func cloneBrowserDataTransferStateMap(states map[string]*browserDataTransferState) map[string]*browserDataTransferState {
	if len(states) == 0 {
		return nil
	}
	out := make(map[string]*browserDataTransferState, len(states))
	for id, state := range states {
		if state == nil {
			out[id] = nil
			continue
		}
		out[id] = &browserDataTransferState{
			files:         append([]script.Value(nil), state.files...),
			effectAllowed: state.effectAllowed,
			dropEffect:    state.dropEffect,
		}
	}
	return out
}

func browserDataTransferConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "DataTransfer is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 0 {
		return script.UndefinedValue(), fmt.Errorf("DataTransfer expects no arguments")
	}
	id := session.allocateBrowserDataTransferState(nil)
	if strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), fmt.Errorf("DataTransfer constructor could not allocate state")
	}
	return browserDataTransferReferenceValue(id), nil
}

func resolveDataTransferReference(session *Session, path string) (script.Value, error) {
	id, suffix, ok := parseBrowserDataTransferInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserDataTransferStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid DataTransfer reference %q in this bounded classic-JS slice", path))
	}

	switch suffix {
	case "":
		return browserDataTransferReferenceValue(id), nil
	case "files":
		return script.ArrayValue(append([]script.Value(nil), state.files...)), nil
	case "items":
		return browserDataTransferItemsReferenceValue(id), nil
	case "effectAllowed":
		return script.StringValue(state.effectAllowed), nil
	case "dropEffect":
		return script.StringValue(state.dropEffect), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
}

func resolveDataTransferItemsReference(session *Session, path string) (script.Value, error) {
	id, suffix, ok := parseBrowserDataTransferItemsInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserDataTransferStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid DataTransfer reference %q in this bounded classic-JS slice", path))
	}

	switch suffix {
	case "":
		return browserDataTransferItemsReferenceValue(id), nil
	case "length":
		return script.NumberValue(float64(len(state.files))), nil
	case "add":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("DataTransferItemList.add expects 1 argument")
			}
			state.files = append(state.files, args[0])
			return script.UndefinedValue(), nil
		}), nil
	case "clear":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("DataTransferItemList.clear accepts no arguments")
			}
			state.files = nil
			return script.UndefinedValue(), nil
		}), nil
	case "remove":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("DataTransferItemList.remove expects 1 argument")
			}
			index, err := browserInt64Value("DataTransferItemList.remove", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			if index < 0 || int(index) >= len(state.files) {
				return script.UndefinedValue(), nil
			}
			removed := make([]script.Value, 0, len(state.files)-1)
			removed = append(removed, state.files[:index]...)
			removed = append(removed, state.files[index+1:]...)
			state.files = removed
			return script.UndefinedValue(), nil
		}), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
}

func setBrowserDataTransferReferenceValue(session *Session, path string, value script.Value) error {
	if session == nil {
		return script.NewError(script.ErrorKindUnsupported, "DataTransfer is unavailable in this bounded classic-JS slice")
	}
	id, suffix, ok := parseBrowserDataTransferInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserDataTransferStateByID(id)
	if !ok || state == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid DataTransfer reference %q in this bounded classic-JS slice", path))
	}

	switch suffix {
	case "":
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	case "effectAllowed":
		state.effectAllowed = script.ToJSString(value)
		return nil
	case "dropEffect":
		state.dropEffect = script.ToJSString(value)
		return nil
	case "files":
		if value.Kind != script.ValueKindArray {
			return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q requires a FileList-like array value in this bounded classic-JS slice", path))
		}
		state.files = append([]script.Value(nil), value.Array...)
		return nil
	default:
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
}

func deleteBrowserDataTransferReferenceValue(session *Session, path string) error {
	if session == nil {
		return script.NewError(script.ErrorKindUnsupported, "DataTransfer is unavailable in this bounded classic-JS slice")
	}
	id, suffix, ok := parseBrowserDataTransferInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserDataTransferStateByID(id)
	if !ok || state == nil {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid DataTransfer reference %q in this bounded classic-JS slice", path))
	}
	switch suffix {
	case "effectAllowed":
		state.effectAllowed = ""
		return nil
	case "dropEffect":
		state.dropEffect = ""
		return nil
	case "files":
		state.files = nil
		return nil
	default:
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
}
