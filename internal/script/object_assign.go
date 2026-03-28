package script

import (
	"fmt"
	"strconv"
)

// ObjectAssign implements the bounded classic-JS slice of Object.assign.
func ObjectAssign(host HostBindings, target Value, sources ...Value) (Value, error) {
	if target.Kind == ValueKindUndefined || target.Kind == ValueKindNull {
		return UndefinedValue(), NewError(ErrorKindRuntime, "Cannot convert undefined or null to object")
	}

	target = objectAssignBoxTarget(target)
	ctx := CurrentBindingUpdateContext()

	for _, source := range sources {
		if source.Kind == ValueKindUndefined || source.Kind == ValueKindNull {
			continue
		}

		keys := objectAssignEnumerableKeys(source)
		for _, key := range keys {
			value, ok, err := objectAssignSourcePropertyValue(host, source, key)
			if err != nil {
				return UndefinedValue(), err
			}
			if !ok {
				continue
			}

			updated, changed, err := objectAssignWriteProperty(host, target, key, value)
			if err != nil {
				return UndefinedValue(), err
			}
			if changed && ctx != nil {
				switch target.Kind {
				case ValueKindObject:
					ctx.ReplaceObjectBindings(target, updated)
				case ValueKindArray:
					ctx.ReplaceArrayBindings(target, updated)
				}
			}
			target = updated
		}
	}

	return target, nil
}

func objectAssignBoxTarget(target Value) Value {
	switch target.Kind {
	case ValueKindObject, ValueKindArray:
		return target
	case ValueKindString:
		return objectAssignStringObject(target.String)
	default:
		return ObjectValue(nil)
	}
}

func objectAssignStringObject(text string) Value {
	entries := make([]ObjectEntry, 0, len(text))
	index := 0
	for _, r := range text {
		entries = append(entries, ObjectEntry{
			Key:   strconv.Itoa(index),
			Value: StringValue(string(r)),
		})
		index++
	}
	return ObjectValue(entries)
}

func objectAssignEnumerableKeys(source Value) []string {
	switch source.Kind {
	case ValueKindObject:
		keys := make([]string, 0, len(source.Object))
		seen := make(map[string]struct{}, len(source.Object))
		for _, entry := range source.Object {
			if classicJSIsInternalObjectKey(entry.Key) {
				continue
			}
			if _, ok := seen[entry.Key]; ok {
				continue
			}
			seen[entry.Key] = struct{}{}
			keys = append(keys, entry.Key)
		}
		return keys
	case ValueKindArray:
		keys := make([]string, 0, len(source.Array))
		for i := range source.Array {
			keys = append(keys, strconv.Itoa(i))
		}
		return keys
	case ValueKindString:
		keys := make([]string, 0, len(source.String))
		index := 0
		for range source.String {
			keys = append(keys, strconv.Itoa(index))
			index++
		}
		return keys
	default:
		return nil
	}
}

func objectAssignSourcePropertyValue(host HostBindings, source Value, key string) (Value, bool, error) {
	switch source.Kind {
	case ValueKindObject:
		value, ok := lookupObjectProperty(source.Object, key)
		if !ok {
			return UndefinedValue(), false, nil
		}
		if value.Kind == ValueKindFunction && value.Function != nil && value.Function.objectAccessor {
			result, err := InvokeCallableValue(host, value, nil, source, true)
			if err != nil {
				return UndefinedValue(), false, err
			}
			return result, true, nil
		}
		return value, true, nil
	case ValueKindArray:
		index, ok := arrayIndexFromBracketKey(key)
		if !ok || index < 0 || index >= len(source.Array) {
			return UndefinedValue(), false, nil
		}
		return source.Array[index], true, nil
	case ValueKindString:
		index, ok := arrayIndexFromBracketKey(key)
		if !ok {
			return UndefinedValue(), false, nil
		}
		runes := []rune(source.String)
		if index < 0 || index >= len(runes) {
			return UndefinedValue(), false, nil
		}
		return StringValue(string(runes[index])), true, nil
	default:
		return UndefinedValue(), false, nil
	}
}

func objectAssignWriteProperty(host HostBindings, target Value, key string, value Value) (Value, bool, error) {
	switch target.Kind {
	case ValueKindObject:
		return objectAssignWriteObjectProperty(host, target, key, value)
	case ValueKindArray:
		return objectAssignWriteArrayProperty(target, key, value)
	default:
		return ObjectValue(nil), false, fmt.Errorf("Object.assign target must be an object or array")
	}
}

func objectAssignWriteObjectProperty(host HostBindings, target Value, key string, value Value) (Value, bool, error) {
	if _, ok := classicJSObjectSizeValue(target); ok && key == "size" {
		return UndefinedValue(), false, NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
	}
	setterKey := classicJSObjectSetterStorageKey(key)
	setterValue, hasSetter := lookupObjectProperty(target.Object, setterKey)
	index := findObjectPropertyIndex(target.Object, key)

	if index >= 0 {
		current := target.Object[index].Value
		if current.Kind == ValueKindFunction && current.Function != nil && current.Function.objectAccessor {
			if hasSetter && setterValue.Kind == ValueKindFunction && setterValue.Function != nil {
				if _, err := InvokeCallableValue(host, setterValue, []Value{value}, target, true); err != nil {
					return UndefinedValue(), false, err
				}
				return target, false, nil
			}
			return UndefinedValue(), false, NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
		}
		target.Object[index].Value = value
		return target, false, nil
	}

	if hasSetter && setterValue.Kind == ValueKindFunction && setterValue.Function != nil {
		if _, err := InvokeCallableValue(host, setterValue, []Value{value}, target, true); err != nil {
			return UndefinedValue(), false, err
		}
		return target, false, nil
	}

	updated := append([]ObjectEntry(nil), target.Object...)
	updated = append(updated, ObjectEntry{Key: key, Value: value})
	updatedValue := objectValueWithMetadata(target, updated)
	return updatedValue, true, nil
}

func objectAssignWriteArrayProperty(target Value, key string, value Value) (Value, bool, error) {
	index, ok := arrayIndexFromBracketKey(key)
	if !ok {
		return UndefinedValue(), false, fmt.Errorf("Object.assign array targets only support numeric keys in this bounded classic-JS slice")
	}

	if index < len(target.Array) {
		target.Array[index] = value
		return target, false, nil
	}

	updated := append([]Value(nil), target.Array...)
	for len(updated) < index {
		updated = append(updated, UndefinedValue())
	}
	if index == len(updated) {
		updated = append(updated, value)
	} else {
		updated[index] = value
	}
	updatedValue := ArrayValue(updated)
	return updatedValue, true, nil
}
