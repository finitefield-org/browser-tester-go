package script

import (
	"fmt"
	"reflect"
	"strings"
)

const bindingReplacementWalkLimit = 1024

type RuntimeConfig struct {
	StepLimit int
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		StepLimit: 10_000,
	}
}

type HostBindings interface {
	Call(method string, args []Value) (Value, error)
}

type BindingUpdateContext interface {
	ReplaceObjectBindings(oldValue Value, newValue Value) int
	ReplaceArrayBindings(oldValue Value, newValue Value) int
	SkipEvaluation() bool
}

var currentBindingUpdateContext BindingUpdateContext

func CurrentBindingUpdateContext() BindingUpdateContext {
	return currentBindingUpdateContext
}

func setCurrentBindingUpdateContext(ctx BindingUpdateContext) func() {
	prev := currentBindingUpdateContext
	currentBindingUpdateContext = ctx
	return func() {
		currentBindingUpdateContext = prev
	}
}

type HostReferenceResolver interface {
	ResolveHostReference(path string) (Value, error)
}

type HostReferenceMutator interface {
	SetHostReference(path string, value Value) error
}

type HostReferenceDeleter interface {
	DeleteHostReference(path string) error
}

type DispatchRequest struct {
	Source        string
	Bindings      map[string]Value
	ModuleExports map[string]Value
}

type DispatchResult struct {
	Value Value
}

type Runtime struct {
	config         RuntimeConfig
	host           HostBindings
	globalBindings map[string]Value
}

type classicJSEnvironment struct {
	parent     *classicJSEnvironment
	bindings   map[string]classicJSBinding
	classDefs  map[string]*classicJSClassDefinition
	withScopes []Value
}

type classicJSBinding struct {
	value              jsValue
	mutable            bool
	skipBindingUpdates bool
}

type classicJSClassFieldDefinition struct {
	env              *classicJSEnvironment
	name             string
	init             string
	private          bool
	privateKeyPrefix string
}

type classicJSClassDefinition struct {
	env                 *classicJSEnvironment
	privateFieldPrefix  string
	instanceMarker      string
	instanceFields      []classicJSClassFieldDefinition
	hasStaticPrototype  bool
	superStaticTarget   Value
	superInstanceTarget Value
	hasSuper            bool
}

func newClassicJSEnvironment() *classicJSEnvironment {
	return &classicJSEnvironment{
		bindings:   make(map[string]classicJSBinding),
		classDefs:  make(map[string]*classicJSClassDefinition),
		withScopes: nil,
	}
}

func (e *classicJSEnvironment) clone() *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	return &classicJSEnvironment{
		parent:     e,
		bindings:   make(map[string]classicJSBinding),
		classDefs:  make(map[string]*classicJSClassDefinition),
		withScopes: append([]Value(nil), e.withScopes...),
	}
}

func (e *classicJSEnvironment) cloneDetached() *classicJSEnvironment {
	return e.cloneDetachedWithMapping(make(map[*classicJSEnvironment]*classicJSEnvironment))
}

func (e *classicJSEnvironment) cloneSkipped() *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	sanitizer := newSkippedValueSanitizer()
	cloned := &classicJSEnvironment{
		parent:     e.parent.cloneSkipped(),
		bindings:   make(map[string]classicJSBinding, len(e.bindings)),
		classDefs:  make(map[string]*classicJSClassDefinition, len(e.classDefs)),
		withScopes: make([]Value, len(e.withScopes)),
	}
	for name, binding := range e.bindings {
		clonedBinding := binding
		clonedBinding.value = sanitizer.sanitizeJSValue(binding.value)
		cloned.bindings[name] = clonedBinding
	}
	for name, classDef := range e.classDefs {
		cloned.classDefs[name] = classDef
	}
	for i, scope := range e.withScopes {
		cloned.withScopes[i] = sanitizer.sanitize(scope)
	}
	return cloned
}

func (e *classicJSEnvironment) cloneDetachedWithMapping(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSEnvironment {
	if e == nil {
		return newClassicJSEnvironment()
	}
	if cloned, ok := mapping[e]; ok {
		return cloned
	}
	clonedParent := e.parent.cloneDetachedWithMapping(mapping)
	cloned := &classicJSEnvironment{
		parent:     clonedParent,
		bindings:   make(map[string]classicJSBinding, len(e.bindings)),
		classDefs:  make(map[string]*classicJSClassDefinition, len(e.classDefs)),
		withScopes: make([]Value, len(e.withScopes)),
	}
	mapping[e] = cloned
	for name, binding := range e.bindings {
		cloned.bindings[name] = classicJSBinding{
			value:   cloneJSValueDetached(binding.value, mapping),
			mutable: binding.mutable,
		}
	}
	for name, classDef := range e.classDefs {
		cloned.classDefs[name] = classDef.cloneDetached(mapping)
	}
	for i, scope := range e.withScopes {
		cloned.withScopes[i] = cloneValueDetached(scope, mapping)
	}
	return cloned
}

func (e *classicJSEnvironment) withScope(value Value) *classicJSEnvironment {
	if e == nil {
		e = newClassicJSEnvironment()
	}
	cloned := e.clone()
	cloned.withScopes = append(cloned.withScopes, value)
	return cloned
}

func (d *classicJSClassDefinition) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSClassDefinition {
	if d == nil {
		return nil
	}
	cloned := &classicJSClassDefinition{
		privateFieldPrefix:  d.privateFieldPrefix,
		instanceMarker:      d.instanceMarker,
		instanceFields:      make([]classicJSClassFieldDefinition, 0, len(d.instanceFields)),
		hasStaticPrototype:  d.hasStaticPrototype,
		superStaticTarget:   cloneValueDetached(d.superStaticTarget, mapping),
		superInstanceTarget: cloneValueDetached(d.superInstanceTarget, mapping),
		hasSuper:            d.hasSuper,
	}
	for _, field := range d.instanceFields {
		cloned.instanceFields = append(cloned.instanceFields, field.cloneDetached(mapping))
	}
	if d.env != nil {
		if clonedEnv, ok := mapping[d.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = d.env.cloneDetachedWithMapping(mapping)
		}
	}
	return cloned
}

func (f classicJSClassFieldDefinition) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSClassFieldDefinition {
	cloned := classicJSClassFieldDefinition{
		name:             f.name,
		init:             f.init,
		private:          f.private,
		privateKeyPrefix: f.privateKeyPrefix,
	}
	if f.env != nil {
		if clonedEnv, ok := mapping[f.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = f.env.cloneDetachedWithMapping(mapping)
		}
	}
	return cloned
}

func (d *classicJSClassDefinition) privateFieldKey(name string) string {
	if d == nil || d.privateFieldPrefix == "" {
		return name
	}
	return d.privateFieldPrefix + name
}

func (e *classicJSEnvironment) setClassDefinition(name string, classDef *classicJSClassDefinition) {
	if e == nil {
		return
	}
	if e.classDefs == nil {
		e.classDefs = make(map[string]*classicJSClassDefinition)
	}
	e.classDefs[name] = classDef
}

func (e *classicJSEnvironment) classDefinition(name string) (*classicJSClassDefinition, bool) {
	for current := e; current != nil; current = current.parent {
		if len(current.classDefs) == 0 {
			continue
		}
		classDef, ok := current.classDefs[name]
		if ok {
			return classDef, true
		}
	}
	return nil, false
}

func resolveClassicJSClassDefinition(value Value, env *classicJSEnvironment) (*classicJSClassDefinition, bool) {
	if value.Kind != ValueKindObject {
		return nil, false
	}
	if value.ClassDefinition != nil {
		return value.ClassDefinition, true
	}
	if value.ClassKey != "" && env != nil {
		classDef, ok := env.classDefinition(value.ClassKey)
		if ok && classDef != nil {
			return classDef, true
		}
	}
	return nil, false
}

func cloneJSValueDetached(value jsValue, mapping map[*classicJSEnvironment]*classicJSEnvironment) jsValue {
	return cloneJSValueDetachedSeen(
		value,
		mapping,
		make(map[*classicJSArrowFunction]*classicJSArrowFunction),
		make(map[uintptr]Value),
		make(map[uintptr]Value),
	)
}

func cloneJSValueDetachedSeen(value jsValue, mapping map[*classicJSEnvironment]*classicJSEnvironment, clonedFunctions map[*classicJSArrowFunction]*classicJSArrowFunction, clonedArrays map[uintptr]Value, clonedObjects map[uintptr]Value) jsValue {
	switch value.kind {
	case jsValueScalar:
		cloned := scalarJSValue(cloneValueDetachedSeen(value.value, mapping, clonedFunctions, clonedArrays, clonedObjects))
		if value.hasReceiver {
			cloned.receiver = cloneValueDetachedSeen(value.receiver, mapping, clonedFunctions, clonedArrays, clonedObjects)
			cloned.hasReceiver = true
		}
		if value.hasNewTarget {
			cloned.newTarget = cloneValueDetachedSeen(value.newTarget, mapping, clonedFunctions, clonedArrays, clonedObjects)
			cloned.hasNewTarget = true
		}
		return cloned
	case jsValueHostObject:
		return hostObjectJSValue()
	case jsValueBuiltinExpr:
		return builtinExprJSValue()
	case jsValueHostMethod:
		return hostMethodJSValue(value.method)
	case jsValueSuper:
		cloned := superJSValue(cloneValueDetachedSeen(value.value, mapping, clonedFunctions, clonedArrays, clonedObjects), cloneValueDetachedSeen(value.receiver, mapping, clonedFunctions, clonedArrays, clonedObjects))
		if value.hasNewTarget {
			cloned.newTarget = cloneValueDetachedSeen(value.newTarget, mapping, clonedFunctions, clonedArrays, clonedObjects)
			cloned.hasNewTarget = true
		}
		return cloned
	default:
		return value
	}
}

func cloneValueDetached(value Value, mapping map[*classicJSEnvironment]*classicJSEnvironment) Value {
	return cloneValueDetachedSeen(
		value,
		mapping,
		make(map[*classicJSArrowFunction]*classicJSArrowFunction),
		make(map[uintptr]Value),
		make(map[uintptr]Value),
	)
}

func cloneValueDetachedSeen(value Value, mapping map[*classicJSEnvironment]*classicJSEnvironment, clonedFunctions map[*classicJSArrowFunction]*classicJSArrowFunction, clonedArrays map[uintptr]Value, clonedObjects map[uintptr]Value) Value {
	switch value.Kind {
	case ValueKindArray:
		if len(value.Array) == 0 {
			return arrayValueOwned(nil)
		}
		ptr := reflect.ValueOf(value.Array).Pointer()
		if ptr != 0 {
			if cloned, ok := clonedArrays[ptr]; ok {
				return cloned
			}
		}
		cloned := make([]Value, len(value.Array))
		clonedValue := Value{Kind: ValueKindArray, Array: cloned}
		if ptr != 0 {
			clonedArrays[ptr] = clonedValue
		}
		for i, element := range value.Array {
			cloned[i] = cloneValueDetachedSeen(element, mapping, clonedFunctions, clonedArrays, clonedObjects)
		}
		return clonedValue
	case ValueKindObject:
		if len(value.Object) == 0 {
			cloned := Value{Kind: ValueKindObject}
			cloned.ClassKey = value.ClassKey
			cloned.ClassDefinition = value.ClassDefinition
			cloned.MapState = cloneMapStateDetached(value.MapState, mapping)
			cloned.SetState = cloneSetStateDetached(value.SetState, mapping)
			return cloned
		}
		ptr := reflect.ValueOf(value.Object).Pointer()
		if ptr != 0 {
			if cloned, ok := clonedObjects[ptr]; ok {
				return cloned
			}
		}
		cloned := make([]ObjectEntry, len(value.Object))
		clonedValue := Value{Kind: ValueKindObject, Object: cloned}
		clonedValue.ClassKey = value.ClassKey
		clonedValue.ClassDefinition = value.ClassDefinition
		if ptr != 0 {
			clonedObjects[ptr] = clonedValue
		}
		for i, entry := range value.Object {
			cloned[i] = ObjectEntry{Key: entry.Key, Value: cloneValueDetachedSeen(entry.Value, mapping, clonedFunctions, clonedArrays, clonedObjects)}
		}
		clonedValue.MapState = cloneMapStateDetached(value.MapState, mapping)
		clonedValue.SetState = cloneSetStateDetached(value.SetState, mapping)
		if ptr != 0 {
			clonedObjects[ptr] = clonedValue
		}
		return clonedValue
	case ValueKindFunction:
		if value.NativeFunction != nil || value.NativeConstructibleFunction != nil {
			cloned := NativeFunctionValue(value.NativeFunction)
			cloned.NativeConstructibleFunction = value.NativeConstructibleFunction
			cloned.Function = value.Function
			return cloned
		}
		if value.Function == nil {
			return value
		}
		if clonedFn, ok := clonedFunctions[value.Function]; ok {
			cloned := FunctionValue(clonedFn)
			cloned.NativeFunction = value.NativeFunction
			cloned.NativeConstructibleFunction = value.NativeConstructibleFunction
			return cloned
		}
		cloned := FunctionValue(value.Function.cloneDetachedSeen(mapping, clonedFunctions, clonedArrays, clonedObjects))
		cloned.NativeFunction = value.NativeFunction
		cloned.NativeConstructibleFunction = value.NativeConstructibleFunction
		return cloned
	case ValueKindPromise:
		cloned := value
		if value.Promise != nil {
			clonedPromise := cloneValueDetachedSeen(*value.Promise, mapping, clonedFunctions, clonedArrays, clonedObjects)
			cloned.Promise = &clonedPromise
		}
		return cloned
	default:
		return value
	}
}

func (e *classicJSEnvironment) declare(name string, value jsValue, mutable bool) error {
	if e == nil {
		return NewError(ErrorKindRuntime, "classic-JS environment is unavailable")
	}
	if e.bindings == nil {
		e.bindings = make(map[string]classicJSBinding)
	}
	if _, exists := e.bindings[name]; exists {
		return NewError(ErrorKindParse, fmt.Sprintf("duplicate lexical declaration for %q in this bounded classic-JS slice", name))
	}
	e.bindings[name] = classicJSBinding{value: value.withoutAssignTarget(), mutable: mutable}
	return nil
}

func (e *classicJSEnvironment) initializeBinding(name string, value jsValue, mutable bool) error {
	if e == nil {
		return NewError(ErrorKindRuntime, "classic-JS environment is unavailable")
	}
	if e.bindings == nil {
		e.bindings = make(map[string]classicJSBinding)
	}
	if binding, ok := e.bindings[name]; ok {
		binding.value = value.withoutAssignTarget()
		binding.mutable = mutable
		e.bindings[name] = binding
		return nil
	}
	e.bindings[name] = classicJSBinding{value: value.withoutAssignTarget(), mutable: mutable}
	return nil
}

func cloneBindingsMap(bindings map[string]Value) map[string]Value {
	if len(bindings) == 0 {
		return map[string]Value{}
	}
	cloned := make(map[string]Value, len(bindings))
	for name, value := range bindings {
		cloned[name] = cloneValueDetached(value, nil)
	}
	return cloned
}

func (e *classicJSEnvironment) lookup(name string) (jsValue, bool) {
	for current := e; current != nil; current = current.parent {
		if len(current.bindings) == 0 {
			if value, ok := current.lookupWithScopes(name); ok {
				return value, true
			}
			continue
		}
		binding, ok := current.bindings[name]
		if ok {
			return binding.value, true
		}
		if value, ok := current.lookupWithScopes(name); ok {
			return value, true
		}
	}
	return jsValue{}, false
}

func (e *classicJSEnvironment) lookupWithScopes(name string) (jsValue, bool) {
	if e == nil || len(e.withScopes) == 0 {
		return jsValue{}, false
	}
	if name == "this" || name == "super" {
		return jsValue{}, false
	}
	for i := len(e.withScopes) - 1; i >= 0; i-- {
		if value, ok := classicJSEnvironmentScopeLookup(e.withScopes[i], name); ok {
			return value, true
		}
	}
	return jsValue{}, false
}

func (e *classicJSEnvironment) assign(name string, value jsValue) error {
	if ctx := CurrentBindingUpdateContext(); ctx != nil && ctx.SkipEvaluation() {
		return nil
	}
	for current := e; current != nil; current = current.parent {
		if len(current.bindings) == 0 {
			handled, err := current.assignWithScopes(name, value)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
			continue
		}
		binding, ok := current.bindings[name]
		if !ok {
			handled, err := current.assignWithScopes(name, value)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
			continue
		}
		if !binding.mutable {
			return NewError(ErrorKindRuntime, fmt.Sprintf("cannot assign to immutable binding %q in this bounded classic-JS slice", name))
		}
		binding.value = value.withoutAssignTarget()
		current.bindings[name] = binding
		return nil
	}
	return NewError(ErrorKindUnsupported, fmt.Sprintf("assignment target %q is not a declared local binding in this bounded classic-JS slice", name))
}

func (e *classicJSEnvironment) assignWithScopes(name string, value jsValue) (bool, error) {
	if ctx := CurrentBindingUpdateContext(); ctx != nil && ctx.SkipEvaluation() {
		return false, nil
	}
	if e == nil || len(e.withScopes) == 0 {
		return false, nil
	}
	if name == "this" || name == "super" {
		return false, nil
	}
	if value.kind != jsValueScalar {
		return false, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
	}
	for i := len(e.withScopes) - 1; i >= 0; i-- {
		currentScope := e.withScopes[i]
		updated, handled, err := classicJSEnvironmentScopeAssign(currentScope, name, value.value)
		if err != nil {
			return true, err
		}
		if !handled {
			continue
		}
		switch {
		case currentScope.Kind == ValueKindObject && updated.Kind == ValueKindObject:
			e.replaceObjectBindings(currentScope, scalarJSValue(updated))
		case currentScope.Kind == ValueKindArray && updated.Kind == ValueKindArray:
			e.replaceArrayBindings(currentScope, scalarJSValue(updated))
		}
		e.withScopes[i] = updated
		return true, nil
	}
	return false, nil
}

func classicJSEnvironmentScopeLookup(scope Value, name string) (jsValue, bool) {
	switch scope.Kind {
	case ValueKindObject:
		if sizeValue, ok := classicJSObjectSizeValue(scope); ok && name == "size" {
			return scalarJSValue(sizeValue), true
		}
		if resolved, ok := lookupObjectProperty(scope.Object, name); ok {
			if resolved.Kind == ValueKindFunction && resolved.Function != nil {
				return jsValue{kind: jsValueScalar, value: resolved, receiver: scope, hasReceiver: true}, true
			}
			return scalarJSValue(resolved), true
		}
		if virtualValue, ok, err := classicJSObjectVirtualProperty(scope, name); ok || err != nil {
			return scalarJSValue(virtualValue), ok
		}
		return jsValue{}, false
	case ValueKindArray:
		if name == "length" {
			return scalarJSValue(NumberValue(float64(len(scope.Array)))), true
		}
		if index, ok := arrayIndexFromBracketKey(name); ok && index >= 0 && index < len(scope.Array) {
			return scalarJSValue(scope.Array[index]), true
		}
		return jsValue{}, false
	default:
		return jsValue{}, false
	}
}

func classicJSEnvironmentScopeAssign(scope Value, name string, rhs Value) (Value, bool, error) {
	switch scope.Kind {
	case ValueKindObject:
		if _, ok := classicJSObjectSizeValue(scope); ok && name == "size" {
			return UndefinedValue(), true, NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
		}
		return objectValueWithMetadata(scope, replaceObjectProperty(scope.Object, name, rhs)), true, nil
	case ValueKindArray:
		if name == "length" {
			newLength, err := classicJSArrayLengthFromValue(rhs)
			if err != nil {
				return UndefinedValue(), true, err
			}
			if newLength < len(scope.Array) {
				return ArrayValue(scope.Array[:newLength]), true, nil
			}
			if newLength == len(scope.Array) {
				return scope, true, nil
			}
			updated := append([]Value(nil), scope.Array...)
			for len(updated) < newLength {
				updated = append(updated, UndefinedValue())
			}
			return arrayValueOwned(updated), true, nil
		}
		index, ok := arrayIndexFromBracketKey(name)
		if !ok {
			return scope, false, nil
		}
		if index < len(scope.Array) {
			updated := append([]Value(nil), scope.Array...)
			updated[index] = rhs
			return arrayValueOwned(updated), true, nil
		}
		updated := append([]Value(nil), scope.Array...)
		for len(updated) < index {
			updated = append(updated, UndefinedValue())
		}
		updated = append(updated, rhs)
		return arrayValueOwned(updated), true, nil
	default:
		return scope, false, NewError(ErrorKindUnsupported, "with statements require object or array values in this bounded classic-JS slice")
	}
}

func (e *classicJSEnvironment) setBindingValue(name string, value jsValue) bool {
	for current := e; current != nil; current = current.parent {
		if len(current.bindings) == 0 {
			continue
		}
		binding, ok := current.bindings[name]
		if !ok {
			continue
		}
		binding.value = value.withoutAssignTarget()
		current.bindings[name] = binding
		return true
	}
	return false
}

func (e *classicJSEnvironment) replaceObjectBindings(oldValue Value, newValue jsValue) int {
	if ctx := CurrentBindingUpdateContext(); ctx != nil && ctx.SkipEvaluation() {
		return 0
	}
	return e.replaceObjectBindingsSeen(oldValue, newValue, make(map[*classicJSEnvironment]struct{}))
}

func (e *classicJSEnvironment) replaceObjectBindingsSeen(oldValue Value, newValue jsValue, visited map[*classicJSEnvironment]struct{}) int {
	if oldValue.Kind != ValueKindObject || newValue.kind != jsValueScalar || newValue.value.Kind != ValueKindObject {
		return 0
	}
	oldPtr := reflect.ValueOf(oldValue.Object).Pointer()
	if oldPtr == 0 {
		return 0
	}
	replacement := newValue.withoutAssignTarget()
	replaced := 0
	if e == nil {
		return 0
	}
	for current := e; current != nil; current = current.parent {
		if visited != nil {
			if _, seen := visited[current]; seen {
				continue
			}
			visited[current] = struct{}{}
		}
		if len(current.bindings) == 0 {
			goto updateScopes
		}
		for name, binding := range current.bindings {
			if binding.skipBindingUpdates {
				continue
			}
			updated, changed := replaceObjectReferencesInJSValue(binding.value, oldPtr, replacement.value)
			if !changed {
				continue
			}
			binding.value = updated
			current.bindings[name] = binding
			replaced++
		}
	updateScopes:
		for i, scope := range current.withScopes {
			updated, changed := replaceObjectReferencesInValue(scope, oldPtr, replacement.value)
			if !changed {
				continue
			}
			current.withScopes[i] = updated
			replaced++
		}
	}
	return replaced
}

func replaceObjectReferencesInJSValue(value jsValue, oldPtr uintptr, replacement Value) (jsValue, bool) {
	changed := false

	switch value.kind {
	case jsValueScalar:
		updated, ok := replaceObjectReferencesInValue(value.value, oldPtr, replacement)
		if ok {
			value.value = updated
			changed = true
		}
	case jsValueSuper:
		updatedValue, ok := replaceObjectReferencesInValue(value.value, oldPtr, replacement)
		if ok {
			value.value = updatedValue
			changed = true
		}
		updatedReceiver, ok := replaceObjectReferencesInValue(value.receiver, oldPtr, replacement)
		if ok {
			value.receiver = updatedReceiver
			changed = true
		}
		if value.hasNewTarget {
			updatedNewTarget, ok := replaceObjectReferencesInValue(value.newTarget, oldPtr, replacement)
			if ok {
				value.newTarget = updatedNewTarget
				changed = true
			}
		}
	}

	if value.kind != jsValueSuper && value.hasReceiver {
		updatedReceiver, ok := replaceObjectReferencesInValue(value.receiver, oldPtr, replacement)
		if ok {
			value.receiver = updatedReceiver
			changed = true
		}
	}
	if value.kind != jsValueSuper && value.hasNewTarget {
		updatedNewTarget, ok := replaceObjectReferencesInValue(value.newTarget, oldPtr, replacement)
		if ok {
			value.newTarget = updatedNewTarget
			changed = true
		}
	}

	return value, changed
}

func replaceObjectReferencesInValue(value Value, oldPtr uintptr, replacement Value) (Value, bool) {
	budget := bindingReplacementWalkLimit
	return replaceObjectReferencesInValueSeen(value, oldPtr, replacement, &budget, make(map[uintptr]struct{}), make(map[uintptr]struct{}), make(map[uintptr]struct{}))
}

func replaceObjectReferencesInValueSeen(value Value, oldPtr uintptr, replacement Value, budget *int, seenObjects map[uintptr]struct{}, seenArrays map[uintptr]struct{}, seenFunctions map[uintptr]struct{}) (Value, bool) {
	switch value.Kind {
	case ValueKindObject:
		if budget != nil {
			if *budget <= 0 {
				return value, false
			}
			*budget--
		}
		ptr := reflect.ValueOf(value.Object).Pointer()
		if ptr == oldPtr {
			return replacement, true
		}
		if ptr == 0 {
			return value, false
		}
		if _, seen := seenObjects[ptr]; seen {
			return value, false
		}
		seenObjects[ptr] = struct{}{}
		if len(value.Object) == 0 {
			return value, false
		}
		var updated []ObjectEntry
		changed := false
		for i, entry := range value.Object {
			next, ok := replaceObjectReferencesInValueSeen(entry.Value, oldPtr, replacement, budget, seenObjects, seenArrays, seenFunctions)
			if ok && !changed {
				updated = make([]ObjectEntry, len(value.Object))
				copy(updated, value.Object[:i])
				changed = true
			}
			if changed {
				updated[i] = ObjectEntry{Key: entry.Key, Value: next}
			}
		}
		if !changed {
			return value, false
		}
		return objectValueOwned(value, updated), true
	case ValueKindArray:
		if budget != nil {
			if *budget <= 0 {
				return value, false
			}
			*budget--
		}
		ptr := reflect.ValueOf(value.Array).Pointer()
		if ptr == oldPtr {
			return replacement, true
		}
		if ptr == 0 {
			return value, false
		}
		if _, seen := seenArrays[ptr]; seen {
			return value, false
		}
		seenArrays[ptr] = struct{}{}
		if len(value.Array) == 0 {
			return value, false
		}
		var updated []Value
		changed := false
		for i, element := range value.Array {
			next, ok := replaceObjectReferencesInValueSeen(element, oldPtr, replacement, budget, seenObjects, seenArrays, seenFunctions)
			if ok && !changed {
				updated = make([]Value, len(value.Array))
				copy(updated, value.Array[:i])
				changed = true
			}
			if changed {
				updated[i] = next
			}
		}
		if !changed {
			return value, false
		}
		return arrayValueOwned(updated), true
	case ValueKindPromise:
		if budget != nil {
			if *budget <= 0 {
				return value, false
			}
			*budget--
		}
		if value.PromiseState != nil && value.PromiseState.resolved {
			next, ok := replaceObjectReferencesInValueSeen(value.PromiseState.value, oldPtr, replacement, budget, seenObjects, seenArrays, seenFunctions)
			if !ok {
				return value, false
			}
			cloned := value
			cloned.PromiseState = &classicJSPromiseState{
				resolved: true,
				rejected: value.PromiseState.rejected,
				value:    next,
			}
			return cloned, true
		}
		if value.Promise == nil {
			return value, false
		}
		next, ok := replaceObjectReferencesInValueSeen(*value.Promise, oldPtr, replacement, budget, seenObjects, seenArrays, seenFunctions)
		if !ok {
			return value, false
		}
		cloned := value
		cloned.Promise = &next
		return cloned, true
	case ValueKindFunction:
		return value, false
	default:
		return value, false
	}
}

func (e *classicJSEnvironment) replaceArrayBindings(oldValue Value, newValue jsValue) int {
	if ctx := CurrentBindingUpdateContext(); ctx != nil && ctx.SkipEvaluation() {
		return 0
	}
	return e.replaceArrayBindingsSeen(oldValue, newValue, make(map[*classicJSEnvironment]struct{}))
}

func (e *classicJSEnvironment) replaceArrayBindingsSeen(oldValue Value, newValue jsValue, visited map[*classicJSEnvironment]struct{}) int {
	if oldValue.Kind != ValueKindArray || newValue.kind != jsValueScalar || newValue.value.Kind != ValueKindArray {
		return 0
	}
	oldPtr := reflect.ValueOf(oldValue.Array).Pointer()
	if oldPtr == 0 {
		return 0
	}
	replacement := newValue.withoutAssignTarget()
	replaced := 0
	if e == nil {
		return 0
	}
	for current := e; current != nil; current = current.parent {
		if visited != nil {
			if _, seen := visited[current]; seen {
				continue
			}
			visited[current] = struct{}{}
		}
		for name, binding := range current.bindings {
			if binding.skipBindingUpdates {
				continue
			}
			updated, changed := replaceArrayReferencesInJSValue(binding.value, oldPtr, replacement.value)
			if !changed {
				continue
			}
			binding.value = updated
			current.bindings[name] = binding
			replaced++
		}
		for i, scope := range current.withScopes {
			updated, changed := replaceArrayReferencesInValue(scope, oldPtr, replacement.value)
			if !changed {
				continue
			}
			current.withScopes[i] = updated
			replaced++
		}
	}
	return replaced
}

func replaceArrayReferencesInJSValue(value jsValue, oldPtr uintptr, replacement Value) (jsValue, bool) {
	changed := false

	switch value.kind {
	case jsValueScalar:
		updated, ok := replaceArrayReferencesInValue(value.value, oldPtr, replacement)
		if ok {
			value.value = updated
			changed = true
		}
	case jsValueSuper:
		updatedValue, ok := replaceArrayReferencesInValue(value.value, oldPtr, replacement)
		if ok {
			value.value = updatedValue
			changed = true
		}
		updatedReceiver, ok := replaceArrayReferencesInValue(value.receiver, oldPtr, replacement)
		if ok {
			value.receiver = updatedReceiver
			changed = true
		}
		if value.hasNewTarget {
			updatedNewTarget, ok := replaceArrayReferencesInValue(value.newTarget, oldPtr, replacement)
			if ok {
				value.newTarget = updatedNewTarget
				changed = true
			}
		}
	}

	if value.kind != jsValueSuper && value.hasReceiver {
		updatedReceiver, ok := replaceArrayReferencesInValue(value.receiver, oldPtr, replacement)
		if ok {
			value.receiver = updatedReceiver
			changed = true
		}
	}
	if value.kind != jsValueSuper && value.hasNewTarget {
		updatedNewTarget, ok := replaceArrayReferencesInValue(value.newTarget, oldPtr, replacement)
		if ok {
			value.newTarget = updatedNewTarget
			changed = true
		}
	}

	return value, changed
}

func replaceArrayReferencesInValue(value Value, oldPtr uintptr, replacement Value) (Value, bool) {
	budget := bindingReplacementWalkLimit
	return replaceArrayReferencesInValueSeen(value, oldPtr, replacement, &budget, make(map[uintptr]struct{}), make(map[uintptr]struct{}), make(map[uintptr]struct{}))
}

func replaceArrayReferencesInValueSeen(value Value, oldPtr uintptr, replacement Value, budget *int, seenArrays map[uintptr]struct{}, seenObjects map[uintptr]struct{}, seenFunctions map[uintptr]struct{}) (Value, bool) {
	switch value.Kind {
	case ValueKindArray:
		if budget != nil {
			if *budget <= 0 {
				return value, false
			}
			*budget--
		}
		ptr := reflect.ValueOf(value.Array).Pointer()
		if ptr == oldPtr {
			return replacement, true
		}
		if ptr == 0 {
			return value, false
		}
		if _, seen := seenArrays[ptr]; seen {
			return value, false
		}
		seenArrays[ptr] = struct{}{}
		if len(value.Array) == 0 {
			return value, false
		}
		var updated []Value
		changed := false
		for i, element := range value.Array {
			next, ok := replaceArrayReferencesInValueSeen(element, oldPtr, replacement, budget, seenArrays, seenObjects, seenFunctions)
			if ok && !changed {
				updated = make([]Value, len(value.Array))
				copy(updated, value.Array[:i])
				changed = true
			}
			if changed {
				updated[i] = next
			}
		}
		if !changed {
			return value, false
		}
		return arrayValueOwned(updated), true
	case ValueKindObject:
		if budget != nil {
			if *budget <= 0 {
				return value, false
			}
			*budget--
		}
		ptr := reflect.ValueOf(value.Object).Pointer()
		if ptr == oldPtr {
			return replacement, true
		}
		if ptr == 0 {
			return value, false
		}
		if _, seen := seenObjects[ptr]; seen {
			return value, false
		}
		seenObjects[ptr] = struct{}{}
		if len(value.Object) == 0 {
			return value, false
		}
		var updated []ObjectEntry
		changed := false
		for i, entry := range value.Object {
			next, ok := replaceArrayReferencesInValueSeen(entry.Value, oldPtr, replacement, budget, seenArrays, seenObjects, seenFunctions)
			if ok && !changed {
				updated = make([]ObjectEntry, len(value.Object))
				copy(updated, value.Object[:i])
				changed = true
			}
			if changed {
				updated[i] = ObjectEntry{Key: entry.Key, Value: next}
			}
		}
		if !changed {
			return value, false
		}
		return objectValueOwned(value, updated), true
	case ValueKindPromise:
		if budget != nil {
			if *budget <= 0 {
				return value, false
			}
			*budget--
		}
		if value.PromiseState != nil && value.PromiseState.resolved {
			next, ok := replaceArrayReferencesInValueSeen(value.PromiseState.value, oldPtr, replacement, budget, seenArrays, seenObjects, seenFunctions)
			if !ok {
				return value, false
			}
			cloned := value
			cloned.PromiseState = &classicJSPromiseState{
				resolved: true,
				rejected: value.PromiseState.rejected,
				value:    next,
			}
			return cloned, true
		}
		if value.Promise == nil {
			return value, false
		}
		next, ok := replaceArrayReferencesInValueSeen(*value.Promise, oldPtr, replacement, budget, seenArrays, seenObjects, seenFunctions)
		if !ok {
			return value, false
		}
		cloned := value
		cloned.Promise = &next
		return cloned, true
	case ValueKindFunction:
		return value, false
	default:
		return value, false
	}
}

func NewRuntime(host HostBindings) *Runtime {
	return NewRuntimeWithConfigAndBindings(DefaultRuntimeConfig(), host, nil)
}

func NewRuntimeWithConfig(config RuntimeConfig, host HostBindings) *Runtime {
	return NewRuntimeWithConfigAndBindings(config, host, nil)
}

func NewRuntimeWithBindings(host HostBindings, globalBindings map[string]Value) *Runtime {
	return NewRuntimeWithConfigAndBindings(DefaultRuntimeConfig(), host, globalBindings)
}

func NewRuntimeWithConfigAndBindings(config RuntimeConfig, host HostBindings, globalBindings map[string]Value) *Runtime {
	cfg := config
	if cfg.StepLimit <= 0 {
		cfg.StepLimit = DefaultRuntimeConfig().StepLimit
	}
	return &Runtime{
		config:         cfg,
		host:           host,
		globalBindings: cloneBindingsMap(globalBindings),
	}
}

func skipScriptTriviaForward(source string, index int) int {
	for index < len(source) {
		switch source[index] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			index++
		case '/':
			if index+1 >= len(source) {
				return index
			}
			switch source[index+1] {
			case '/':
				index += 2
				for index < len(source) && source[index] != '\n' && source[index] != '\r' {
					index++
				}
			case '*':
				index += 2
				for index+1 < len(source) && !(source[index] == '*' && source[index+1] == '/') {
					index++
				}
				if index+1 < len(source) {
					index += 2
				}
			default:
				return index
			}
		default:
			return index
		}
	}
	return index
}

func readScriptKeywordAt(source string, index int) (string, int) {
	index = skipScriptTriviaForward(source, index)
	start := index
	for index < len(source) && isIdentPart(source[index]) {
		index++
	}
	if start == index {
		return "", index
	}
	return source[start:index], index
}

func isHoistableTopLevelFunctionDeclarationStatement(source string) bool {
	index := skipScriptTriviaForward(source, 0)
	keyword, next := readScriptKeywordAt(source, index)
	switch keyword {
	case "function":
		return true
	case "async":
		keyword, _ = readScriptKeywordAt(source, next)
		return keyword == "function"
	case "export":
		keyword, next = readScriptKeywordAt(source, next)
		switch keyword {
		case "function":
			return true
		case "async":
			keyword, _ = readScriptKeywordAt(source, next)
			return keyword == "function"
		default:
			return false
		}
	default:
		return false
	}
}

func hoistTopLevelFunctionStatements(statements []string, evaluate func(string) (Value, error)) ([]bool, error) {
	hoisted := make([]bool, len(statements))
	for i, statement := range statements {
		if !isHoistableTopLevelFunctionDeclarationStatement(statement) {
			continue
		}
		if _, err := evaluate(statement); err != nil {
			return nil, err
		}
		hoisted[i] = true
	}
	return hoisted, nil
}

func (r *Runtime) Config() RuntimeConfig {
	if r == nil {
		return DefaultRuntimeConfig()
	}
	return r.config
}

func (r *Runtime) Dispatch(request DispatchRequest) (DispatchResult, error) {
	if r == nil {
		return DispatchResult{}, NewError(ErrorKindRuntime, "script runtime is unavailable")
	}

	source := strings.TrimSpace(request.Source)
	if source == "" || source == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	statements, err := splitScriptStatements(source)
	if err != nil {
		if scriptErr, ok := err.(Error); ok {
			return DispatchResult{}, scriptErr
		}
		return DispatchResult{}, NewError(ErrorKindParse, err.Error())
	}

	if len(statements) == 0 {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	baseEnv := newClassicJSEnvironment()
	if len(r.globalBindings) > 0 {
		if err := seedClassicJSEnvironment(baseEnv, r.globalBindings, true); err != nil {
			return DispatchResult{}, err
		}
	}
	env := baseEnv.clone()
	if len(request.Bindings) > 0 {
		if err := seedClassicJSEnvironment(env, request.Bindings, false); err != nil {
			return DispatchResult{}, err
		}
	}

	hoisted, err := hoistTopLevelFunctionStatements(statements, func(statement string) (Value, error) {
		result, err := r.dispatchStatement(statement, env, request.ModuleExports)
		if err != nil {
			return UndefinedValue(), err
		}
		return result.Value, nil
	})
	if err != nil {
		return DispatchResult{}, err
	}

	var last Value = UndefinedValue()
	for i, statement := range statements {
		if hoisted[i] {
			last = UndefinedValue()
			continue
		}
		result, err := r.dispatchStatement(statement, env, request.ModuleExports)
		if err != nil {
			return DispatchResult{}, err
		}
		last = result.Value
	}
	return DispatchResult{Value: last}, nil
}

func (r *Runtime) dispatchStatement(source string, env *classicJSEnvironment, moduleExports map[string]Value) (DispatchResult, error) {
	if source == "" || source == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}

	if strings.HasPrefix(strings.TrimSpace(source), "host:") {
		return r.dispatchLegacyStatement(source, env)
	}

	value, err := evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, r.host, env, r.config.StepLimit, true, false, false, nil, UndefinedValue(), false, nil, moduleExports)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Value: value}, nil
}

func (r *Runtime) dispatchLegacyStatement(source string, env *classicJSEnvironment) (DispatchResult, error) {
	method, args, err := parseHostInvocation(strings.TrimPrefix(strings.TrimSpace(source), "host:"))
	if err != nil {
		return DispatchResult{}, NewError(ErrorKindParse, err.Error())
	}
	if r.host == nil {
		return DispatchResult{}, NewError(ErrorKindHost, "host bindings are unavailable")
	}
	resolvedArgs, err := r.resolveArgs(args, env)
	if err != nil {
		return DispatchResult{}, err
	}
	value, err := r.host.Call(method, resolvedArgs)
	if err != nil {
		return DispatchResult{}, NewError(ErrorKindHost, err.Error())
	}
	return DispatchResult{Value: value}, nil
}

func (r *Runtime) resolveArgs(args []Value, env *classicJSEnvironment) ([]Value, error) {
	if len(args) == 0 {
		return nil, nil
	}
	resolved := make([]Value, len(args))
	for i, arg := range args {
		if arg.Kind != ValueKindInvocation {
			resolved[i] = arg
			continue
		}
		result, err := r.dispatchSourceWithEnv(arg.Invocation, env)
		if err != nil {
			return nil, err
		}
		resolved[i] = result.Value
	}
	return resolved, nil
}

func (r *Runtime) dispatchSourceWithEnv(source string, env *classicJSEnvironment) (DispatchResult, error) {
	if r == nil {
		return DispatchResult{}, NewError(ErrorKindRuntime, "script runtime is unavailable")
	}
	trimmed := strings.TrimSpace(source)
	if trimmed == "" || trimmed == "noop" {
		return DispatchResult{Value: UndefinedValue()}, nil
	}
	if strings.HasPrefix(trimmed, "host:") {
		return r.dispatchLegacyStatement(trimmed, env)
	}
	value, err := evalClassicJSProgram(trimmed, r.host, env, r.config.StepLimit, nil)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Value: value}, nil
}

func seedClassicJSEnvironment(env *classicJSEnvironment, bindings map[string]Value, skipBindingUpdates bool) error {
	if env == nil {
		return NewError(ErrorKindRuntime, "classic-JS environment is unavailable")
	}
	for name, value := range bindings {
		mutable := name == "Intl"
		if err := env.declare(name, scalarJSValue(value), mutable); err != nil {
			return err
		}
		if skipBindingUpdates {
			binding := env.bindings[name]
			binding.skipBindingUpdates = true
			env.bindings[name] = binding
		}
	}
	return nil
}

func evalClassicJSProgram(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSProgramWithAllowAwait(source, host, env, stepLimit, true, privateClass)
}

func evalClassicJSProgramWithAllowAwait(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, false, false, nil, UndefinedValue(), false, privateClass, nil)
}

func evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, newTarget Value, hasNewTarget bool, privateClass *classicJSClassDefinition, moduleExports map[string]Value, skipCache ...*classicJSSkipCache) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExportsInternal(source, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, newTarget, hasNewTarget, privateClass, moduleExports, true, skipCache...)
}

func evalClassicJSProgramWithAllowAwaitAndYieldAndExportsAllowLoopSignals(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, newTarget Value, hasNewTarget bool, privateClass *classicJSClassDefinition, moduleExports map[string]Value, skipCache ...*classicJSSkipCache) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExportsInternal(source, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, newTarget, hasNewTarget, privateClass, moduleExports, false, skipCache...)
}

func evalClassicJSProgramWithAllowAwaitAndYieldAndExportsInternal(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, newTarget Value, hasNewTarget bool, privateClass *classicJSClassDefinition, moduleExports map[string]Value, convertLoopSignals bool, skipCache ...*classicJSSkipCache) (Value, error) {
	if env == nil {
		env = newClassicJSEnvironment()
	}
	if stepLimit <= 0 {
		stepLimit = DefaultRuntimeConfig().StepLimit
	}

	statements, err := splitScriptStatements(source)
	if err != nil {
		return UndefinedValue(), NewError(ErrorKindParse, err.Error())
	}
	if len(statements) == 0 {
		return UndefinedValue(), nil
	}
	var cachedSkip *classicJSSkipCache
	if len(skipCache) > 0 {
		cachedSkip = skipCache[0]
	}

	hoisted, err := hoistTopLevelFunctionStatements(statements, func(statement string) (Value, error) {
		return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, newTarget, hasNewTarget, privateClass, moduleExports, cachedSkip)
	})
	if err != nil {
		return UndefinedValue(), err
	}

	var last Value = UndefinedValue()
	for i, statement := range statements {
		if hoisted[i] {
			last = UndefinedValue()
			continue
		}
		value, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, newTarget, hasNewTarget, privateClass, moduleExports, cachedSkip)
		if err != nil {
			if awaitedPromise, nextState, ok := classicJSAwaitSignalDetails(err); ok {
				continuation := &classicJSBlockState{
					statements: statements,
					env:        env,
					index:      i,
					lastValue:  last,
				}
				if nextState != nil {
					continuation.child = nextState
				} else {
					continuation.index = i + 1
				}
				return UndefinedValue(), classicJSAwaitSignal{
					promise:     awaitedPromise,
					resumeState: continuation,
				}
			}
			if yieldedValue, state, ok := classicJSYieldSignalDetails(err); ok {
				if state == nil && i+1 < len(statements) {
					return UndefinedValue(), NewError(ErrorKindUnsupported, "yield inside a nested block must be the final statement in this bounded classic-JS slice")
				}
				return yieldedValue, err
			}
			if throwValue, ok := classicJSThrowSignalValue(err); ok {
				return UndefinedValue(), NewError(ErrorKindRuntime, ToJSString(throwValue))
			}
			if convertLoopSignals && classicJSBreakSignalValue(err) {
				return UndefinedValue(), NewError(ErrorKindParse, "break statement is not within a loop or switch in this bounded classic-JS slice")
			}
			if convertLoopSignals && classicJSContinueSignalValue(err) {
				return UndefinedValue(), NewError(ErrorKindParse, "continue statement is not within a loop in this bounded classic-JS slice")
			}
			return UndefinedValue(), err
		}
		last = value
	}
	return last, nil
}

func evalClassicJSProgramWithAllowAwaitAndYield(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, resumeState classicJSResumeState, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, allowYield, false, resumeState, UndefinedValue(), false, privateClass, nil)
}

func scanTemplateLiteralSource(text string, index int) (int, error) {
	if index >= len(text) || text[index] != '`' {
		return index, NewError(ErrorKindParse, "unexpected end of script source")
	}

	index++
	for index < len(text) {
		switch text[index] {
		case '\\':
			index++
			if index >= len(text) {
				return index, NewError(ErrorKindParse, "unterminated escape sequence in template literal")
			}
			index++
		case '`':
			return index + 1, nil
		case '$':
			if index+1 < len(text) && text[index+1] == '{' {
				next, err := scanTemplateInterpolationSource(text, index+2)
				if err != nil {
					return index, err
				}
				index = next
				continue
			}
			index++
		default:
			index++
		}
	}

	return index, NewError(ErrorKindParse, "unterminated template literal")
}

func scanTemplateInterpolationSource(text string, index int) (int, error) {
	parser := &classicJSStatementParser{source: text, pos: index}
	if _, err := parser.consumeTemplateInterpolationSource(); err != nil {
		return index, err
	}
	return parser.pos, nil
}

func splitScriptStatements(source string) ([]string, error) {
	text := strings.TrimSpace(source)
	if text == "" {
		return nil, nil
	}

	statements := make([]string, 0, 4)
	start := 0
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int
	doStates := make([]int, 0, 4)
	canStartRegex := true
	lastWasDot := false
	scanRegularExpressionLiteral := func(index int) (int, error) {
		if index >= len(text) || text[index] != '/' {
			return index, NewError(ErrorKindParse, "unexpected end of script source")
		}
		escaped := false
		inClass := false
		i := index + 1
		for i < len(text) {
			ch := text[i]
			if ch == '\n' || ch == '\r' {
				return index, NewError(ErrorKindParse, "unterminated regular expression literal")
			}
			if escaped {
				escaped = false
				i++
				continue
			}
			switch ch {
			case '\\':
				escaped = true
				i++
			case '[':
				inClass = true
				i++
			case ']':
				if inClass {
					inClass = false
				}
				i++
			case '/':
				if inClass {
					i++
					continue
				}
				i++
				for i < len(text) {
					flag := text[i]
					if flag < 'a' || flag > 'z' {
						if isIdentPart(flag) {
							return index, NewError(ErrorKindParse, fmt.Sprintf("unsupported regular expression flag %q in this bounded classic-JS slice", flag))
						}
						break
					}
					switch flag {
					case 'd', 'g', 'i', 'm', 's', 'u', 'v', 'y':
					default:
						return index, NewError(ErrorKindParse, fmt.Sprintf("unsupported regular expression flag %q in this bounded classic-JS slice", flag))
					}
					i++
				}
				return i, nil
			default:
				i++
			}
		}
		return index, NewError(ErrorKindParse, "unterminated regular expression literal")
	}
	regexStartKeywords := map[string]struct{}{
		"await":      {},
		"case":       {},
		"delete":     {},
		"in":         {},
		"instanceof": {},
		"return":     {},
		"throw":      {},
		"typeof":     {},
		"void":       {},
		"yield":      {},
	}
	skipSpaceAndCommentsForward := func(index int) int {
		for index < len(text) {
			switch text[index] {
			case ' ', '\t', '\n', '\r':
				index++
			case '/':
				if index+1 >= len(text) {
					return index
				}
				switch text[index+1] {
				case '/':
					index += 2
					for index < len(text) && text[index] != '\n' && text[index] != '\r' {
						index++
					}
				case '*':
					index += 2
					for index+1 < len(text) && !(text[index] == '*' && text[index+1] == '/') {
						index++
					}
					if index+1 < len(text) {
						index += 2
					}
				default:
					return index
				}
			default:
				return index
			}
		}
		return index
	}
	nextKeywordAhead := func(index int, keyword string) bool {
		index = skipSpaceAndCommentsForward(index)
		if index+len(keyword) > len(text) {
			return false
		}
		if text[index:index+len(keyword)] != keyword {
			return false
		}
		if index > 0 && isIdentPart(text[index-1]) {
			return false
		}
		if index+len(keyword) < len(text) && isIdentPart(text[index+len(keyword)]) {
			return false
		}
		return true
	}
	readKeywordAt := func(index int) (string, int) {
		index = skipSpaceAndCommentsForward(index)
		start := index
		for index < len(text) && isIdentPart(text[index]) {
			index++
		}
		if start == index {
			return "", index
		}
		return text[start:index], index
	}
	statementCanEndAtBrace := func(index int) bool {
		keyword, next := readKeywordAt(index)
		switch keyword {
		case "function", "class", "if", "for", "while", "switch", "try", "with", "do":
			return true
		case "async":
			nextKeyword, _ := readKeywordAt(next)
			return nextKeyword == "function"
		case "export":
			nextKeyword, nextIndex := readKeywordAt(next)
			switch nextKeyword {
			case "function", "class":
				return true
			case "default":
				afterDefault, _ := readKeywordAt(nextIndex)
				return afterDefault == "function" || afterDefault == "class"
			default:
				return false
			}
		default:
			return false
		}
	}
	canSplitAtBrace := statementCanEndAtBrace(start)
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if lineComment {
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && i+1 < len(text) && text[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		topLevel := parenDepth == 0 && braceDepth == 0 && bracketDepth == 0
		if topLevel {
			if ch == 'd' && i+1 < len(text) && text[i:i+2] == "do" && (i == 0 || !isIdentPart(text[i-1])) && (i+2 >= len(text) || !isIdentPart(text[i+2])) {
				doStates = append(doStates, 1)
			}
			if ch == 'w' && i+4 < len(text) && text[i:i+5] == "while" && (i == 0 || !isIdentPart(text[i-1])) && (i+5 >= len(text) || !isIdentPart(text[i+5])) {
				if n := len(doStates); n > 0 && doStates[n-1] == 1 {
					doStates[n-1] = 2
				}
			}
		}
		switch ch {
		case '\'', '"':
			quote = ch
			canStartRegex = false
			lastWasDot = false
		case '`':
			next, err := scanTemplateLiteralSource(text, i)
			if err != nil {
				return nil, err
			}
			i = next - 1
			canStartRegex = false
			lastWasDot = false
			continue
		case '/':
			if i+1 < len(text) {
				switch text[i+1] {
				case '/':
					lineComment = true
					i++
				case '*':
					blockComment = true
					i++
				default:
					if canStartRegex {
						next, err := scanRegularExpressionLiteral(i)
						if err != nil {
							return nil, err
						}
						i = next - 1
						canStartRegex = false
						lastWasDot = false
						continue
					}
					canStartRegex = true
				}
			} else {
				canStartRegex = true
			}
			lastWasDot = false
		case '(':
			parenDepth++
			canStartRegex = true
			lastWasDot = false
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			canStartRegex = false
			lastWasDot = false
		case '{':
			braceDepth++
			canStartRegex = true
			lastWasDot = false
		case '}':
			wasOpenBlock := braceDepth > 0
			if braceDepth > 0 {
				braceDepth--
			}
			canStartRegex = true
			lastWasDot = false
			if !wasOpenBlock {
				continue
			}
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				if n := len(doStates); n > 0 && doStates[n-1] == 1 {
					continue
				}
				if nextKeywordAhead(i+1, "else") || nextKeywordAhead(i+1, "catch") || nextKeywordAhead(i+1, "finally") {
					continue
				}
				if !canSplitAtBrace {
					continue
				}
				statement := strings.TrimSpace(text[start : i+1])
				if statement != "" {
					statements = append(statements, statement)
				}
				start = i + 1
				canSplitAtBrace = statementCanEndAtBrace(start)
			}
		case '[':
			bracketDepth++
			canStartRegex = true
			lastWasDot = false
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			canStartRegex = false
			lastWasDot = false
		case ';':
			if topLevel {
				if n := len(doStates); n > 0 {
					if doStates[n-1] == 1 {
						doStates[n-1] = 2
						continue
					}
					if doStates[n-1] == 2 {
						doStates = doStates[:n-1]
						if len(doStates) > 0 {
							continue
						}
					}
				}
				if nextKeywordAhead(i+1, "else") {
					continue
				}
				statement := strings.TrimSpace(text[start:i])
				if statement != "" {
					statements = append(statements, statement)
				}
				start = i + 1
				canSplitAtBrace = statementCanEndAtBrace(start)
			}
			canStartRegex = true
			lastWasDot = false
		case ',', ':', '?', '=', '!', '~', '+', '-', '*', '%', '&', '|', '^', '<', '>':
			canStartRegex = true
			lastWasDot = false
		case ' ', '\t', '\n', '\r':
			continue
		case '.':
			canStartRegex = false
			lastWasDot = true
		default:
			if isIdentStart(ch) {
				startIdent := i
				i++
				for i < len(text) && isIdentPart(text[i]) {
					i++
				}
				word := text[startIdent:i]
				if !lastWasDot {
					if _, ok := regexStartKeywords[word]; ok {
						canStartRegex = true
					} else {
						canStartRegex = false
					}
				} else {
					canStartRegex = false
				}
				lastWasDot = false
				i--
				continue
			}
			if isDigit(ch) {
				canStartRegex = false
				lastWasDot = false
				continue
			}
			canStartRegex = false
			lastWasDot = false
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted string in script source")
	}
	if blockComment {
		return nil, fmt.Errorf("unterminated block comment in script source")
	}

	if tail := strings.TrimSpace(text[start:]); tail != "" {
		statements = append(statements, tail)
	}
	return statements, nil
}

func SplitScriptStatementsForRuntime(source string) ([]string, error) {
	return splitScriptStatements(source)
}
