package runtime

import (
	"fmt"
	"strings"
	"time"

	"browsertester/internal/script"
)

const browserFileReferencePrefix = "file:"

type browserFileState struct {
	bytes        []byte
	name         string
	mimeType     string
	lastModified int64
	readable     bool
}

func browserFileReferenceValue(id string) script.Value {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return script.NullValue()
	}
	return script.HostObjectReference(browserFileReferencePrefix + normalized)
}

func parseBrowserFileInstancePath(path string) (string, string, bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, browserFileReferencePrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(normalized, browserFileReferencePrefix)
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

func (s *Session) allocateBrowserFileState(bytes []byte, name, mimeType string, lastModified int64, readable bool) string {
	if s == nil {
		return ""
	}
	if s.fileStates == nil {
		s.fileStates = map[string]*browserFileState{}
	}
	if s.nextFileStateID <= 0 {
		s.nextFileStateID = 1
	}
	id := fmt.Sprintf("%d", s.nextFileStateID)
	s.nextFileStateID++
	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	s.fileStates[id] = &browserFileState{
		bytes:        copied,
		name:         strings.TrimSpace(name),
		mimeType:     strings.TrimSpace(mimeType),
		lastModified: lastModified,
		readable:     readable,
	}
	return id
}

func (s *Session) browserFileStateByID(id string) (*browserFileState, bool) {
	if s == nil || s.fileStates == nil {
		return nil, false
	}
	state, ok := s.fileStates[strings.TrimSpace(id)]
	return state, ok
}

func (s *Session) deleteBrowserFileState(id string) {
	if s == nil || s.fileStates == nil {
		return
	}
	delete(s.fileStates, strings.TrimSpace(id))
}

func cloneBrowserFileStateMap(states map[string]*browserFileState) map[string]*browserFileState {
	if len(states) == 0 {
		return nil
	}
	out := make(map[string]*browserFileState, len(states))
	for id, state := range states {
		if state == nil {
			out[id] = nil
			continue
		}
		copied := make([]byte, len(state.bytes))
		copy(copied, state.bytes)
		out[id] = &browserFileState{
			bytes:        copied,
			name:         state.name,
			mimeType:     state.mimeType,
			lastModified: state.lastModified,
			readable:     state.readable,
		}
	}
	return out
}

func browserFileLastModifiedFromOptions(value script.Value) int64 {
	if value.Kind != script.ValueKindObject {
		return time.Now().UnixMilli()
	}
	for i := len(value.Object) - 1; i >= 0; i-- {
		if value.Object[i].Key != "lastModified" {
			continue
		}
		lastModified, err := browserInt64Value("File.lastModified", value.Object[i].Value)
		if err != nil {
			break
		}
		return lastModified
	}
	return time.Now().UnixMilli()
}

func browserFileConstructor(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "File is unavailable in this bounded classic-JS slice")
	}
	if len(args) < 2 || len(args) > 3 {
		return script.UndefinedValue(), fmt.Errorf("File expects 2 or 3 arguments")
	}

	bytes := browserBlobBytesFromValue(session, args[0])
	name := script.ToJSString(args[1])
	mimeType := ""
	lastModified := time.Now().UnixMilli()
	if len(args) == 3 {
		mimeType = browserBlobTypeFromOptions(args[2])
		lastModified = browserFileLastModifiedFromOptions(args[2])
	}

	id := session.allocateBrowserFileState(bytes, name, mimeType, lastModified, true)
	if strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), fmt.Errorf("File constructor could not allocate state")
	}
	return browserFileReferenceValue(id), nil
}

func resolveFileReference(session *Session, path string) (script.Value, error) {
	id, suffix, ok := parseBrowserFileInstancePath(path)
	if !ok || strings.TrimSpace(id) == "" {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	state, ok := session.browserFileStateByID(id)
	if !ok || state == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("invalid File reference %q in this bounded classic-JS slice", path))
	}

	copyBytes := func() []byte {
		out := make([]byte, len(state.bytes))
		copy(out, state.bytes)
		return out
	}

	switch suffix {
	case "":
		return browserFileReferenceValue(id), nil
	case "name":
		return script.StringValue(state.name), nil
	case "size":
		return script.NumberValue(float64(len(state.bytes))), nil
	case "type":
		return script.StringValue(state.mimeType), nil
	case "lastModified":
		return script.NumberValue(float64(state.lastModified)), nil
	case "text":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("File.text accepts no arguments")
			}
			if !state.readable {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "file content is unavailable in this bounded classic-JS slice")
			}
			return script.PromiseValue(script.StringValue(string(copyBytes()))), nil
		}), nil
	case "arrayBuffer":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("File.arrayBuffer accepts no arguments")
			}
			if !state.readable {
				return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "file content is unavailable in this bounded classic-JS slice")
			}
			return script.PromiseValue(browserUint8ArrayBufferValue(copyBytes())), nil
		}), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("File.%s accepts no arguments", suffix)
			}
			return script.StringValue("[object File]"), nil
		}), nil
	default:
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
}
