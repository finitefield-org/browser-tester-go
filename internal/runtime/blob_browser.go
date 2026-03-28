package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"browsertester/internal/script"
)

type browserBlobState struct {
	bytes    []byte
	mimeType string
}

func browserBlobReferenceValue(id string) script.Value {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return script.NullValue()
	}
	return script.HostObjectReference("blob:" + normalized)
}

func parseBrowserBlobInstancePath(path string) (string, string, bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "blob:") {
		return "", "", false
	}
	rest := strings.TrimPrefix(normalized, "blob:")
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

func (s *Session) allocateBrowserBlobState(bytes []byte, mimeType string) string {
	if s == nil {
		return ""
	}
	if s.blobStates == nil {
		s.blobStates = map[string]*browserBlobState{}
	}
	if s.nextBlobStateID <= 0 {
		s.nextBlobStateID = 1
	}
	id := strconv.FormatInt(s.nextBlobStateID, 10)
	s.nextBlobStateID++
	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	s.blobStates[id] = &browserBlobState{
		bytes:    copied,
		mimeType: strings.TrimSpace(mimeType),
	}
	return id
}

func (s *Session) browserBlobStateByID(id string) (*browserBlobState, bool) {
	if s == nil {
		return nil, false
	}
	if s.blobStates == nil {
		return nil, false
	}
	state, ok := s.blobStates[strings.TrimSpace(id)]
	return state, ok
}

func (s *Session) deleteBrowserBlobState(id string) {
	if s == nil || s.blobStates == nil {
		return
	}
	delete(s.blobStates, strings.TrimSpace(id))
}

func cloneBrowserBlobStateMap(states map[string]*browserBlobState) map[string]*browserBlobState {
	if len(states) == 0 {
		return nil
	}
	out := make(map[string]*browserBlobState, len(states))
	for id, state := range states {
		if state == nil {
			out[id] = nil
			continue
		}
		copied := make([]byte, len(state.bytes))
		copy(copied, state.bytes)
		out[id] = &browserBlobState{
			bytes:    copied,
			mimeType: state.mimeType,
		}
	}
	return out
}

func browserBlobBytesFromValue(session *Session, value script.Value) []byte {
	switch value.Kind {
	case script.ValueKindUndefined, script.ValueKindNull:
		return nil
	case script.ValueKindArray:
		out := make([]byte, 0, len(value.Array))
		for _, part := range value.Array {
			out = append(out, browserBlobPartBytes(session, part)...)
		}
		return out
	default:
		return browserBlobPartBytes(session, value)
	}
}

func browserBlobPartBytes(session *Session, value script.Value) []byte {
	if session != nil && value.Kind == script.ValueKindHostReference {
		id, _, ok := parseBrowserBlobInstancePath(value.HostReferencePath)
		if ok {
			if state, ok := session.browserBlobStateByID(id); ok && state != nil {
				copied := make([]byte, len(state.bytes))
				copy(copied, state.bytes)
				return copied
			}
		}
	}
	return []byte(script.ToJSString(value))
}

func browserBlobTypeFromOptions(value script.Value) string {
	if value.Kind != script.ValueKindObject {
		return ""
	}
	for i := len(value.Object) - 1; i >= 0; i-- {
		if value.Object[i].Key == "type" {
			return strings.TrimSpace(script.ToJSString(value.Object[i].Value))
		}
	}
	return ""
}

func browserBlobConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "Blob is unavailable in this bounded classic-JS slice")
	}
	if len(args) > 2 {
		return script.UndefinedValue(), fmt.Errorf("Blob expects at most 2 arguments")
	}

	var bytes []byte
	if len(args) >= 1 {
		bytes = browserBlobBytesFromValue(session, args[0])
	}
	mimeType := ""
	if len(args) >= 2 {
		mimeType = browserBlobTypeFromOptions(args[1])
	}

	id := session.allocateBrowserBlobState(bytes, mimeType)
	if strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), fmt.Errorf("Blob constructor could not allocate state")
	}
	return browserBlobReferenceValue(id), nil
}

func resolveBlobReference(session *Session, path string) (script.Value, error) {
	id, suffix, ok := parseBrowserBlobInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserBlobStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid Blob reference %q in this bounded classic-JS slice", path))
	}

	if suffix == "" {
		return browserBlobReferenceValue(id), nil
	}

	switch suffix {
	case "size":
		return script.NumberValue(float64(len(state.bytes))), nil
	case "type":
		return script.StringValue(state.mimeType), nil
	case "text":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("Blob.text accepts no arguments")
			}
			return script.PromiseValue(script.StringValue(string(state.bytes))), nil
		}), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("Blob.%s accepts no arguments", suffix)
			}
			return script.StringValue("[object Blob]"), nil
		}), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
}

func browserURLCreateObjectURL(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "URL.createObjectURL is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("URL.createObjectURL expects 1 argument")
	}
	id, ok := browserBlobStateIDFromValue(session, args[0])
	if !ok {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "URL.createObjectURL requires a Blob value in this bounded classic-JS slice")
	}
	state, ok := session.browserBlobStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "URL.createObjectURL requires a live Blob value in this bounded classic-JS slice")
	}
	return script.StringValue("blob:" + session.allocateBrowserBlobState(state.bytes, state.mimeType)), nil
}

func browserURLRevokeObjectURL(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "URL.revokeObjectURL is unavailable in this bounded classic-JS slice")
	}
	if len(args) != 1 {
		return script.UndefinedValue(), fmt.Errorf("URL.revokeObjectURL expects 1 argument")
	}
	id, _, ok := parseBrowserBlobInstancePath(script.ToJSString(args[0]))
	if !ok {
		return script.UndefinedValue(), nil
	}
	session.deleteBrowserBlobState(id)
	return script.UndefinedValue(), nil
}

func downloadBytesForBlobDestination(session *Session, destination string) ([]byte, bool) {
	id, _, ok := parseBrowserBlobInstancePath(destination)
	if !ok || session == nil {
		return nil, false
	}
	state, ok := session.browserBlobStateByID(id)
	if !ok || state == nil {
		return nil, false
	}
	copied := make([]byte, len(state.bytes))
	copy(copied, state.bytes)
	return copied, true
}

func browserBlobStateIDFromValue(session *Session, value script.Value) (string, bool) {
	if value.Kind != script.ValueKindHostReference {
		return "", false
	}
	id, _, ok := parseBrowserBlobInstancePath(value.HostReferencePath)
	if !ok {
		return "", false
	}
	if session == nil {
		return "", false
	}
	if state, ok := session.browserBlobStateByID(id); !ok || state == nil {
		return "", false
	}
	return id, true
}
