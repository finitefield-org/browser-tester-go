package script

import "fmt"

type classicJSMapEntry struct {
	key   Value
	value Value
}

type classicJSMapState struct {
	entries []classicJSMapEntry
}

type MapEntry struct {
	Key   Value
	Value Value
}

type classicJSSetState struct {
	entries []Value
}

var builtinMapValue = buildBuiltinMapValue()
var builtinSetValue = buildBuiltinSetValue()

func BuiltinMapValue() Value {
	return builtinMapValue
}

func BuiltinSetValue() Value {
	return builtinSetValue
}

func buildBuiltinMapValue() Value {
	fn := &classicJSArrowFunction{
		name:          "Map",
		constructible: true,
		env:           newClassicJSEnvironment(),
	}
	marker, _ := classicJSConstructibleFunctionMarker(fn)

	callFn := func(args []Value) (Value, error) {
		return UndefinedValue(), NewError(ErrorKindRuntime, "Map constructor must be called with `new` in this bounded classic-JS slice")
	}
	constructFn := func(args []Value) (Value, error) {
		state := &classicJSMapState{}
		if err := state.seed(args); err != nil {
			return UndefinedValue(), err
		}
		return classicJSMapInstanceValue(state, marker), nil
	}

	value := NativeConstructibleFunctionValue(callFn, constructFn)
	value.Function = fn
	return value
}

func buildBuiltinSetValue() Value {
	fn := &classicJSArrowFunction{
		name:          "Set",
		constructible: true,
		env:           newClassicJSEnvironment(),
	}
	marker, _ := classicJSConstructibleFunctionMarker(fn)

	callFn := func(args []Value) (Value, error) {
		return UndefinedValue(), NewError(ErrorKindRuntime, "Set constructor must be called with `new` in this bounded classic-JS slice")
	}
	constructFn := func(args []Value) (Value, error) {
		state := &classicJSSetState{}
		if err := state.seed(args); err != nil {
			return UndefinedValue(), err
		}
		return classicJSSetInstanceValue(state, marker), nil
	}

	value := NativeConstructibleFunctionValue(callFn, constructFn)
	value.Function = fn
	return value
}

func classicJSMapInstanceValue(state *classicJSMapState, marker string) Value {
	entries := make([]ObjectEntry, 0, 1)
	if marker != "" {
		entries = append(entries, ObjectEntry{
			Key:   classicJSInstanceMarkerKey(marker),
			Value: BoolValue(true),
		})
	}
	value := objectValueOwned(Value{}, entries)
	value.MapState = state
	return value
}

func classicJSSetInstanceValue(state *classicJSSetState, marker string) Value {
	entries := make([]ObjectEntry, 0, 1)
	if marker != "" {
		entries = append(entries, ObjectEntry{
			Key:   classicJSInstanceMarkerKey(marker),
			Value: BoolValue(true),
		})
	}
	value := objectValueOwned(Value{}, entries)
	value.SetState = state
	return value
}

func (s *classicJSMapState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSMapState {
	if s == nil {
		return nil
	}
	cloned := &classicJSMapState{
		entries: make([]classicJSMapEntry, len(s.entries)),
	}
	for i, entry := range s.entries {
		cloned.entries[i] = classicJSMapEntry{
			key:   cloneValueDetached(entry.key, mapping),
			value: cloneValueDetached(entry.value, mapping),
		}
	}
	return cloned
}

func cloneMapStateDetached(state *classicJSMapState, mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSMapState {
	if state == nil {
		return nil
	}
	return state.cloneDetached(mapping)
}

func sanitizeSkippedMapState(state *classicJSMapState) *classicJSMapState {
	if state == nil {
		return nil
	}
	cloned := &classicJSMapState{
		entries: make([]classicJSMapEntry, len(state.entries)),
	}
	for i, entry := range state.entries {
		cloned.entries[i] = classicJSMapEntry{
			key:   sanitizeSkippedValue(entry.key),
			value: sanitizeSkippedValue(entry.value),
		}
	}
	return cloned
}

func cloneSetStateDetached(state *classicJSSetState, mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSSetState {
	if state == nil {
		return nil
	}
	cloned := &classicJSSetState{
		entries: make([]Value, len(state.entries)),
	}
	for i, entry := range state.entries {
		cloned.entries[i] = cloneValueDetached(entry, mapping)
	}
	return cloned
}

func sanitizeSkippedSetState(state *classicJSSetState) *classicJSSetState {
	if state == nil {
		return nil
	}
	cloned := &classicJSSetState{
		entries: make([]Value, len(state.entries)),
	}
	for i, entry := range state.entries {
		cloned.entries[i] = sanitizeSkippedValue(entry)
	}
	return cloned
}

func (s *classicJSMapState) size() int {
	if s == nil {
		return 0
	}
	return len(s.entries)
}

func (s *classicJSMapState) seed(args []Value) error {
	if s == nil || len(args) == 0 {
		return nil
	}
	source := args[0]
	if source.Kind == ValueKindUndefined {
		return nil
	}
	if source.Kind == ValueKindNull {
		return NewError(ErrorKindRuntime, "Map constructor cannot iterate over null in this bounded classic-JS slice")
	}
	if source.Kind != ValueKindArray {
		return NewError(ErrorKindUnsupported, "Map constructor requires an array of key/value pairs in this bounded classic-JS slice")
	}
	for i, pair := range source.Array {
		if pair.Kind != ValueKindArray || len(pair.Array) < 2 {
			return NewError(ErrorKindUnsupported, fmt.Sprintf("Map constructor pair %d must be a two-item array in this bounded classic-JS slice", i))
		}
		s.set(pair.Array[0], pair.Array[1])
	}
	return nil
}

func (s *classicJSMapState) findIndex(key Value) int {
	if s == nil {
		return -1
	}
	for i := len(s.entries) - 1; i >= 0; i-- {
		if sameValueZero(s.entries[i].key, key) {
			return i
		}
	}
	return -1
}

func (s *classicJSMapState) get(key Value) (Value, bool) {
	if index := s.findIndex(key); index >= 0 {
		return s.entries[index].value, true
	}
	return UndefinedValue(), false
}

func (s *classicJSMapState) has(key Value) bool {
	return s.findIndex(key) >= 0
}

func (s *classicJSMapState) set(key Value, value Value) {
	if s == nil {
		return
	}
	if index := s.findIndex(key); index >= 0 {
		s.entries[index].value = value
		return
	}
	s.entries = append(s.entries, classicJSMapEntry{key: key, value: value})
}

func (s *classicJSMapState) delete(key Value) bool {
	if s == nil {
		return false
	}
	index := s.findIndex(key)
	if index < 0 {
		return false
	}
	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	return true
}

func (s *classicJSMapState) entryList() []classicJSMapEntry {
	if s == nil || len(s.entries) == 0 {
		return nil
	}
	cloned := make([]classicJSMapEntry, len(s.entries))
	copy(cloned, s.entries)
	return cloned
}

func (s *classicJSSetState) size() int {
	if s == nil {
		return 0
	}
	return len(s.entries)
}

func (s *classicJSSetState) seed(args []Value) error {
	if s == nil || len(args) == 0 {
		return nil
	}
	source := args[0]
	if source.Kind == ValueKindUndefined {
		return nil
	}
	if source.Kind == ValueKindNull {
		return NewError(ErrorKindRuntime, "Set constructor cannot iterate over null in this bounded classic-JS slice")
	}
	values, err := classicJSSetConstructorValues(source)
	if err != nil {
		return err
	}
	for _, entry := range values {
		s.add(entry)
	}
	return nil
}

func classicJSSetConstructorValues(source Value) ([]Value, error) {
	switch source.Kind {
	case ValueKindArray:
		return append([]Value(nil), source.Array...), nil
	case ValueKindString:
		values := make([]Value, 0, len(source.String))
		for _, r := range source.String {
			values = append(values, StringValue(string(r)))
		}
		return values, nil
	case ValueKindObject:
		if source.SetState != nil {
			values := make([]Value, len(source.SetState.entries))
			copy(values, source.SetState.entries)
			return values, nil
		}
		if source.MapState != nil {
			values := make([]Value, 0, len(source.MapState.entries))
			for _, entry := range source.MapState.entries {
				values = append(values, arrayValueOwned([]Value{entry.key, entry.value}))
			}
			return values, nil
		}
		nextValue, ok := lookupObjectProperty(source.Object, "next")
		if !ok || nextValue.Kind != ValueKindFunction || nextValue.NativeFunction == nil {
			return nil, NewError(ErrorKindUnsupported, "Set constructor requires a string, array, set, map, or iterator-like object with a native next() function in this bounded classic-JS slice")
		}
		values := make([]Value, 0, len(source.Object))
		for {
			result, err := nextValue.NativeFunction(nil)
			if err != nil {
				return nil, err
			}
			if result.Kind != ValueKindObject {
				return nil, NewError(ErrorKindUnsupported, "Set constructor iterator must return an object in this bounded classic-JS slice")
			}
			doneValue, ok := lookupObjectProperty(result.Object, "done")
			if !ok || doneValue.Kind != ValueKindBool {
				return nil, NewError(ErrorKindUnsupported, "Set constructor iterator result must include a boolean `done` property in this bounded classic-JS slice")
			}
			if doneValue.Bool {
				break
			}
			itemValue, ok := lookupObjectProperty(result.Object, "value")
			if !ok {
				itemValue = UndefinedValue()
			}
			values = append(values, itemValue)
		}
		return values, nil
	default:
		return nil, NewError(ErrorKindRuntime, "Set constructor requires a string, array, set, map, or iterator-like object with a native next() function in this bounded classic-JS slice")
	}
}

func (s *classicJSSetState) findIndex(value Value) int {
	if s == nil {
		return -1
	}
	for i := len(s.entries) - 1; i >= 0; i-- {
		if sameValueZero(s.entries[i], value) {
			return i
		}
	}
	return -1
}

func (s *classicJSSetState) has(value Value) bool {
	return s.findIndex(value) >= 0
}

func (s *classicJSSetState) add(value Value) {
	if s == nil {
		return
	}
	if s.findIndex(value) >= 0 {
		return
	}
	s.entries = append(s.entries, value)
}

func (s *classicJSSetState) delete(value Value) bool {
	if s == nil {
		return false
	}
	index := s.findIndex(value)
	if index < 0 {
		return false
	}
	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	return true
}

func MapEntries(value Value) ([]MapEntry, bool) {
	if value.Kind != ValueKindObject || value.MapState == nil {
		return nil, false
	}
	entries := value.MapState.entryList()
	out := make([]MapEntry, len(entries))
	for i, entry := range entries {
		out[i] = MapEntry{
			Key:   entry.key,
			Value: entry.value,
		}
	}
	return out, true
}

func SetEntries(value Value) ([]Value, bool) {
	if value.Kind != ValueKindObject || value.SetState == nil {
		return nil, false
	}
	entries := value.SetState.entries
	out := make([]Value, len(entries))
	copy(out, entries)
	return out, true
}

func classicJSObjectSizeValue(value Value) (Value, bool) {
	if value.Kind != ValueKindObject {
		return UndefinedValue(), false
	}
	if value.ObjectSize != nil {
		size, ok := value.ObjectSize()
		if !ok {
			return UndefinedValue(), false
		}
		return size, true
	}
	if value.MapState != nil {
		return NumberValue(float64(value.MapState.size())), true
	}
	if value.SetState != nil {
		return NumberValue(float64(value.SetState.size())), true
	}
	return UndefinedValue(), false
}

func classicJSObjectVirtualProperty(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindObject {
		return UndefinedValue(), false, nil
	}
	if sizeValue, ok := classicJSObjectSizeValue(value); ok && name == "size" {
		return sizeValue, true, nil
	}
	if value.MapState != nil {
		return classicJSMapVirtualProperty(value, name)
	}
	if value.SetState != nil {
		return classicJSSetVirtualProperty(value, name)
	}
	return UndefinedValue(), false, nil
}

func classicJSObjectHasVirtualProperty(value Value, name string) bool {
	if value.Kind != ValueKindObject {
		return false
	}
	if _, ok := classicJSObjectSizeValue(value); ok && name == "size" {
		return true
	}
	if value.MapState != nil {
		return classicJSMapHasVirtualProperty(name)
	}
	if value.SetState != nil {
		return classicJSSetHasVirtualProperty(name)
	}
	return false
}

func classicJSMapVirtualProperty(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindObject || value.MapState == nil {
		return UndefinedValue(), false, nil
	}

	switch name {
	case "constructor":
		return BuiltinMapValue(), true, nil
	case "size":
		return NumberValue(float64(value.MapState.size())), true, nil
	case "get":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			key := UndefinedValue()
			if len(args) > 0 {
				key = args[0]
			}
			if result, ok := value.MapState.get(key); ok {
				return result, nil
			}
			return UndefinedValue(), nil
		}), true, nil
	case "set":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			key := UndefinedValue()
			if len(args) > 0 {
				key = args[0]
			}
			next := UndefinedValue()
			if len(args) > 1 {
				next = args[1]
			}
			value.MapState.set(key, next)
			return value, nil
		}), true, nil
	case "has":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			key := UndefinedValue()
			if len(args) > 0 {
				key = args[0]
			}
			return BoolValue(value.MapState.has(key)), nil
		}), true, nil
	case "delete":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			key := UndefinedValue()
			if len(args) > 0 {
				key = args[0]
			}
			return BoolValue(value.MapState.delete(key)), nil
		}), true, nil
	case "clear":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 0 {
				return UndefinedValue(), fmt.Errorf("Map.clear expects no arguments")
			}
			value.MapState.entries = nil
			return UndefinedValue(), nil
		}), true, nil
	case "forEach":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) == 0 {
				return UndefinedValue(), fmt.Errorf("Map.forEach expects a callback")
			}
			callback := args[0]
			thisArg, hasReceiver := callbackReceiver(args)
			entries := value.MapState.entryList()
			for _, entry := range entries {
				if _, err := InvokeCallableValue(CurrentInvokeHost(), callback, []Value{
					entry.value,
					entry.key,
					value,
				}, thisArg, hasReceiver); err != nil {
					return UndefinedValue(), err
				}
			}
			return UndefinedValue(), nil
		}), true, nil
	case "keys":
		return classicJSMapIteratorMethodValue("keys", value.MapState.entryList(), classicJSMapIterationKeys), true, nil
	case "values":
		return classicJSMapIteratorMethodValue("values", value.MapState.entryList(), classicJSMapIterationValues), true, nil
	case "entries":
		return classicJSMapIteratorMethodValue("entries", value.MapState.entryList(), classicJSMapIterationEntries), true, nil
	default:
		return UndefinedValue(), false, nil
	}
}

func classicJSMapHasVirtualProperty(name string) bool {
	switch name {
	case "constructor", "size", "get", "set", "has", "delete", "clear", "forEach", "keys", "values", "entries":
		return true
	default:
		return false
	}
}

type classicJSMapIterationMode uint8

const (
	classicJSMapIterationValues classicJSMapIterationMode = iota
	classicJSMapIterationKeys
	classicJSMapIterationEntries
)

func classicJSMapIteratorMethodValue(method string, entries []classicJSMapEntry, mode classicJSMapIterationMode) Value {
	snapshot := append([]classicJSMapEntry(nil), entries...)
	return NativeFunctionValue(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return UndefinedValue(), fmt.Errorf("Map.%s expects no arguments", method)
		}
		return classicJSMapIteratorValue(snapshot, mode), nil
	})
}

func classicJSMapIteratorValue(entries []classicJSMapEntry, mode classicJSMapIterationMode) Value {
	index := 0
	return objectValueOwned(Value{}, []ObjectEntry{
		{
			Key: "next",
			Value: NativeFunctionValue(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return UndefinedValue(), fmt.Errorf("Map iterator next expects no arguments")
				}
				if index >= len(entries) {
					return classicJSIteratorResult(UndefinedValue(), true), nil
				}
				current := entries[index]
				index++
				switch mode {
				case classicJSMapIterationKeys:
					return classicJSIteratorResult(current.key, false), nil
				case classicJSMapIterationEntries:
					return classicJSIteratorResult(arrayValueOwned([]Value{current.key, current.value}), false), nil
				default:
					return classicJSIteratorResult(current.value, false), nil
				}
			}),
		},
	})
}

func classicJSSetVirtualProperty(value Value, name string) (Value, bool, error) {
	if value.Kind != ValueKindObject || value.SetState == nil {
		return UndefinedValue(), false, nil
	}

	switch name {
	case "constructor":
		return BuiltinSetValue(), true, nil
	case "size":
		return NumberValue(float64(value.SetState.size())), true, nil
	case "add":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			next := UndefinedValue()
			if len(args) > 0 {
				next = args[0]
			}
			value.SetState.add(next)
			return value, nil
		}), true, nil
	case "has":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			next := UndefinedValue()
			if len(args) > 0 {
				next = args[0]
			}
			return BoolValue(value.SetState.has(next)), nil
		}), true, nil
	case "delete":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			next := UndefinedValue()
			if len(args) > 0 {
				next = args[0]
			}
			return BoolValue(value.SetState.delete(next)), nil
		}), true, nil
	case "clear":
		return NativeFunctionValue(func(args []Value) (Value, error) {
			if len(args) != 0 {
				return UndefinedValue(), fmt.Errorf("Set.clear expects no arguments")
			}
			value.SetState.entries = nil
			return UndefinedValue(), nil
		}), true, nil
	case "keys":
		return classicJSSetIteratorMethodValue("keys", value.SetState.entryList(), classicJSSetIterationValues), true, nil
	case "values":
		return classicJSSetIteratorMethodValue("values", value.SetState.entryList(), classicJSSetIterationValues), true, nil
	case "entries":
		return classicJSSetIteratorMethodValue("entries", value.SetState.entryList(), classicJSSetIterationEntries), true, nil
	default:
		return UndefinedValue(), false, nil
	}
}

func classicJSSetHasVirtualProperty(name string) bool {
	switch name {
	case "constructor", "size", "add", "has", "delete", "clear", "keys", "values", "entries":
		return true
	default:
		return false
	}
}

type classicJSSetIterationMode uint8

const (
	classicJSSetIterationValues classicJSSetIterationMode = iota
	classicJSSetIterationEntries
)

func (s *classicJSSetState) entryList() []Value {
	if s == nil || len(s.entries) == 0 {
		return nil
	}
	cloned := make([]Value, len(s.entries))
	copy(cloned, s.entries)
	return cloned
}

func classicJSSetIteratorMethodValue(method string, entries []Value, mode classicJSSetIterationMode) Value {
	snapshot := append([]Value(nil), entries...)
	return NativeFunctionValue(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return UndefinedValue(), fmt.Errorf("Set.%s expects no arguments", method)
		}
		return classicJSSetIteratorValue(snapshot, mode), nil
	})
}

func classicJSSetIteratorValue(entries []Value, mode classicJSSetIterationMode) Value {
	index := 0
	return objectValueOwned(Value{}, []ObjectEntry{
		{
			Key: "next",
			Value: NativeFunctionValue(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return UndefinedValue(), fmt.Errorf("Set iterator next expects no arguments")
				}
				if index >= len(entries) {
					return classicJSIteratorResult(UndefinedValue(), true), nil
				}
				current := entries[index]
				index++
				if mode == classicJSSetIterationEntries {
					return classicJSIteratorResult(arrayValueOwned([]Value{current, current}), false), nil
				}
				return classicJSIteratorResult(current, false), nil
			}),
		},
	})
}

func classicJSIteratorResult(value Value, done bool) Value {
	return objectValueOwned(Value{}, []ObjectEntry{
		{Key: "value", Value: value},
		{Key: "done", Value: BoolValue(done)},
	})
}
