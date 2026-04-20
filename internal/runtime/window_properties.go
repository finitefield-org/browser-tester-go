package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/script"
)

const browserWindowPropertyReferencePrefix = "windowprop:"

func windowPropertyReferencePath(name string) string {
	return browserWindowPropertyReferencePrefix + name
}

func splitWindowPropertyReferencePath(path string) (base string, rest string, ok bool) {
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, browserWindowPropertyReferencePrefix) {
		return "", "", false
	}
	remainder := strings.TrimPrefix(normalized, browserWindowPropertyReferencePrefix)
	if remainder == "" {
		return "", "", false
	}
	base = remainder
	rest = ""
	if index := strings.IndexByte(remainder, '.'); index >= 0 {
		base = remainder[:index]
		rest = strings.TrimPrefix(remainder[index+1:], ".")
	}
	if base == "" {
		return "", "", false
	}
	return base, rest, true
}

func (s *Session) windowPropertyValue(name string) (script.Value, bool) {
	if s == nil || len(s.windowProperties) == 0 {
		return script.Value{}, false
	}
	if name == "isSecureContext" || name == "geolocation" {
		return script.Value{}, false
	}
	value, ok := s.windowProperties[name]
	return value, ok
}

func (s *Session) setWindowProperty(name string, value script.Value) {
	if s == nil {
		return
	}
	if name == "isSecureContext" || name == "geolocation" {
		return
	}
	if s.windowProperties == nil {
		s.windowProperties = make(map[string]script.Value)
	}
	s.windowProperties[name] = value
}

func (s *Session) deleteWindowProperty(name string) bool {
	if s == nil || len(s.windowProperties) == 0 {
		return false
	}
	if name == "isSecureContext" || name == "geolocation" {
		return false
	}
	if _, ok := s.windowProperties[name]; !ok {
		return false
	}
	delete(s.windowProperties, name)
	return true
}

func (s *Session) resolveWindowPropertyReference(path string) (script.Value, error) {
	base, rest, ok := splitWindowPropertyReferencePath(path)
	if !ok {
		return script.UndefinedValue(), fmt.Errorf("invalid window property reference %q", path)
	}
	current, exists := s.windowPropertyValue(base)
	if !exists {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	if rest == "" {
		return current, nil
	}
	return resolveWindowPropertyValuePath(current, rest)
}

func (s *Session) setWindowPropertyReference(path string, value script.Value) error {
	base, rest, ok := splitWindowPropertyReferencePath(path)
	if !ok {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	if rest == "" {
		s.setWindowProperty(base, value)
		return nil
	}
	current, exists := s.windowPropertyValue(base)
	if !exists {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	updated, ok, err := setWindowPropertyValuePath(current, rest, value)
	if err != nil {
		return err
	}
	if !ok {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("assignment to %q is unsupported in this bounded classic-JS slice", path))
	}
	s.setWindowProperty(base, updated)
	return nil
}

func (s *Session) deleteWindowPropertyReference(path string) error {
	base, rest, ok := splitWindowPropertyReferencePath(path)
	if !ok {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	if rest == "" {
		if s.deleteWindowProperty(base) {
			return nil
		}
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	current, exists := s.windowPropertyValue(base)
	if !exists {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	updated, ok, err := deleteWindowPropertyValuePath(current, rest)
	if err != nil {
		return err
	}
	if !ok {
		return script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("deletion of %q is unsupported in this bounded classic-JS slice", path))
	}
	s.setWindowProperty(base, updated)
	return nil
}

func resolveWindowPropertyValuePath(value script.Value, path string) (script.Value, error) {
	segments := splitWindowPropertySegments(path)
	if len(segments) == 0 {
		return value, nil
	}
	current := value
	for i, segment := range segments {
		switch current.Kind {
		case script.ValueKindObject:
			next, ok := objectProperty(current, segment)
			if !ok {
				return script.UndefinedValue(), nil
			}
			current = next
		default:
			return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", windowPropertyReferencePath(strings.Join(segments[:i+1], "."))))
		}
	}
	return current, nil
}

func setWindowPropertyValuePath(value script.Value, path string, rhs script.Value) (script.Value, bool, error) {
	segments := splitWindowPropertySegments(path)
	if len(segments) == 0 {
		return rhs, true, nil
	}
	if value.Kind != script.ValueKindObject {
		return script.Value{}, false, nil
	}
	updated, ok, err := setWindowObjectPropertyPath(value, segments, rhs)
	if err != nil || !ok {
		return updated, ok, err
	}
	return updated, true, nil
}

func deleteWindowPropertyValuePath(value script.Value, path string) (script.Value, bool, error) {
	segments := splitWindowPropertySegments(path)
	if len(segments) == 0 {
		return script.UndefinedValue(), false, nil
	}
	if value.Kind != script.ValueKindObject {
		return script.Value{}, false, nil
	}
	updated, ok, err := deleteWindowObjectPropertyPath(value, segments)
	if err != nil || !ok {
		return updated, ok, err
	}
	return updated, true, nil
}

func splitWindowPropertySegments(path string) []string {
	segments := strings.Split(strings.TrimSpace(strings.TrimPrefix(path, ".")), ".")
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func setWindowObjectPropertyPath(value script.Value, segments []string, rhs script.Value) (script.Value, bool, error) {
	if len(segments) == 0 {
		return rhs, true, nil
	}
	if value.Kind != script.ValueKindObject {
		return script.Value{}, false, nil
	}
	key := segments[0]
	if len(segments) == 1 {
		return objectValueWithProperty(value, key, rhs), true, nil
	}
	child, ok := objectProperty(value, key)
	if !ok {
		return script.Value{}, false, nil
	}
	updatedChild, ok, err := setWindowObjectPropertyPath(child, segments[1:], rhs)
	if err != nil || !ok {
		return updatedChild, ok, err
	}
	return objectValueWithProperty(value, key, updatedChild), true, nil
}

func deleteWindowObjectPropertyPath(value script.Value, segments []string) (script.Value, bool, error) {
	if len(segments) == 0 {
		return value, false, nil
	}
	if value.Kind != script.ValueKindObject {
		return script.Value{}, false, nil
	}
	key := segments[0]
	if len(segments) == 1 {
		return objectValueWithoutProperty(value, key), true, nil
	}
	child, ok := objectProperty(value, key)
	if !ok {
		return script.Value{}, false, nil
	}
	updatedChild, ok, err := deleteWindowObjectPropertyPath(child, segments[1:])
	if err != nil || !ok {
		return updatedChild, ok, err
	}
	return objectValueWithProperty(value, key, updatedChild), true, nil
}

func objectValueWithProperty(value script.Value, key string, next script.Value) script.Value {
	updated := value
	updated.Object = replaceWindowObjectProperty(updated.Object, key, next)
	return updated
}

func objectValueWithoutProperty(value script.Value, key string) script.Value {
	updated := value
	updated.Object = deleteWindowObjectProperty(updated.Object, key)
	return updated
}

func replaceWindowObjectProperty(entries []script.ObjectEntry, name string, value script.Value) []script.ObjectEntry {
	cloned := append([]script.ObjectEntry(nil), entries...)
	for i := len(cloned) - 1; i >= 0; i-- {
		if cloned[i].Key == name {
			cloned[i].Value = value
			return cloned
		}
	}
	return append(cloned, script.ObjectEntry{Key: name, Value: value})
}

func deleteWindowObjectProperty(entries []script.ObjectEntry, name string) []script.ObjectEntry {
	if len(entries) == 0 {
		return nil
	}
	cloned := make([]script.ObjectEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Key == name {
			continue
		}
		cloned = append(cloned, entry)
	}
	return cloned
}
