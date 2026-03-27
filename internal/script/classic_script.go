package script

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type jsValueKind int

const (
	jsValueScalar jsValueKind = iota
	jsValueHostObject
	jsValueBuiltinExpr
	jsValueHostMethod
	jsValueSuper
)

const ClassicJSModuleMetaURLBindingName = "\x00classic-js-module-url"

const (
	classicJSRegExpInternalPrefix = "\x00classic-js-regexp:"
	classicJSRegExpPatternKey     = classicJSRegExpInternalPrefix + "pattern"
	classicJSRegExpFlagsKey       = classicJSRegExpInternalPrefix + "flags"
)

type jsValue struct {
	kind         jsValueKind
	value        Value
	method       string
	receiver     Value
	hasReceiver  bool
	newTarget    Value
	hasNewTarget bool
	assignTarget *classicJSAssignmentTarget
}

type classicJSAssignmentTarget struct {
	name  string
	steps []classicJSDeleteStep
}

func scalarJSValue(value Value) jsValue {
	return jsValue{
		kind:  jsValueScalar,
		value: value,
	}
}

func hostObjectJSValue() jsValue {
	return jsValue{
		kind: jsValueHostObject,
	}
}

func builtinExprJSValue() jsValue {
	return jsValue{
		kind: jsValueBuiltinExpr,
	}
}

func hostMethodJSValue(method string) jsValue {
	return jsValue{
		kind:   jsValueHostMethod,
		method: method,
	}
}

func superJSValue(target Value, receiver Value) jsValue {
	return jsValue{
		kind:        jsValueSuper,
		value:       target,
		receiver:    receiver,
		hasReceiver: true,
	}
}

func (v jsValue) withoutAssignTarget() jsValue {
	v.assignTarget = nil
	return v
}

func (v jsValue) withNewTarget(target Value) jsValue {
	v.newTarget = target
	v.hasNewTarget = true
	return v
}

func (v jsValue) withAssignTarget(name string, steps []classicJSDeleteStep) jsValue {
	clonedSteps := append([]classicJSDeleteStep(nil), steps...)
	v.assignTarget = &classicJSAssignmentTarget{
		name:  name,
		steps: clonedSteps,
	}
	return v
}

func (v jsValue) extendAssignTarget(step classicJSDeleteStep) jsValue {
	if v.assignTarget == nil {
		return v
	}
	clonedSteps := append([]classicJSDeleteStep(nil), v.assignTarget.steps...)
	clonedSteps = append(clonedSteps, step)
	v.assignTarget = &classicJSAssignmentTarget{
		name:  v.assignTarget.name,
		steps: clonedSteps,
	}
	return v
}

type noopHostBindings struct{}

func (noopHostBindings) Call(method string, args []Value) (Value, error) {
	return UndefinedValue(), nil
}

type skipHostBindings struct {
	delegate HostBindings
}

func (h skipHostBindings) Call(method string, args []Value) (Value, error) {
	return UndefinedValue(), nil
}

func (h skipHostBindings) ResolveHostReference(path string) (Value, error) {
	if h.delegate == nil {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	resolver, ok := h.delegate.(HostReferenceResolver)
	if !ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	value, err := resolver.ResolveHostReference(path)
	if err != nil {
		return UndefinedValue(), err
	}
	return sanitizeSkippedValue(value), nil
}

func (h skipHostBindings) DeleteHostReference(path string) error {
	if h.delegate == nil {
		return NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	deleter, ok := h.delegate.(HostReferenceDeleter)
	if !ok {
		return NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	return deleter.DeleteHostReference(path)
}

func sanitizeSkippedValue(value Value) Value {
	switch value.Kind {
	case ValueKindArray:
		if len(value.Array) == 0 {
			return ArrayValue(nil)
		}
		cloned := make([]Value, len(value.Array))
		for i, element := range value.Array {
			cloned[i] = sanitizeSkippedValue(element)
		}
		return ArrayValue(cloned)
	case ValueKindObject:
		if len(value.Object) == 0 {
			return ObjectValue(nil)
		}
		cloned := make([]ObjectEntry, len(value.Object))
		for i, entry := range value.Object {
			cloned[i] = ObjectEntry{Key: entry.Key, Value: sanitizeSkippedValue(entry.Value)}
		}
		return ObjectValue(cloned)
	case ValueKindFunction:
		return NativeFunctionValue(func(args []Value) (Value, error) {
			return UndefinedValue(), nil
		})
	case ValueKindPromise:
		if value.Promise == nil {
			return PromiseValue(UndefinedValue())
		}
		cloned := sanitizeSkippedValue(*value.Promise)
		return PromiseValue(cloned)
	case ValueKindHostReference:
		if value.HostReferenceKind == HostReferenceKindFunction || value.HostReferenceKind == HostReferenceKindConstructor {
			return NativeFunctionValue(func(args []Value) (Value, error) {
				return UndefinedValue(), nil
			})
		}
		return value
	default:
		return value
	}
}

func evalClassicJSStatement(source string, host HostBindings) (Value, error) {
	return evalClassicJSStatementWithEnv(source, host, nil, DefaultRuntimeConfig().StepLimit)
}

func evalClassicJSStatementWithEnv(source string, host HostBindings, env *classicJSEnvironment, stepLimit int) (Value, error) {
	return evalClassicJSStatementWithEnvAndAllowAwait(source, host, env, stepLimit, false, nil)
}

func evalClassicJSStatementWithEnvAndAllowAwait(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, privateClass ...*classicJSClassDefinition) (Value, error) {
	var classDef *classicJSClassDefinition
	if len(privateClass) > 0 {
		classDef = privateClass[0]
	}
	return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, false, false, nil, UndefinedValue(), false, classDef, nil)
}

func evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, newTarget Value, hasNewTarget bool, privateClass *classicJSClassDefinition, moduleExports map[string]Value) (Value, error) {
	if stepLimit <= 0 {
		stepLimit = DefaultRuntimeConfig().StepLimit
	}
	parser := &classicJSStatementParser{
		source:             strings.TrimSpace(source),
		host:               host,
		env:                env,
		privateClass:       privateClass,
		privateFieldPrefix: "",
		stepLimit:          stepLimit,
		allowAwait:         allowAwait,
		allowYield:         allowYield,
		allowReturn:        allowReturn,
		resumeState:        resumeState,
		newTarget:          newTarget,
		hasNewTarget:       hasNewTarget,
		moduleExports:      moduleExports,
	}
	if privateClass != nil {
		parser.privateFieldPrefix = privateClass.privateFieldPrefix
	}
	if parser.source == "" {
		return UndefinedValue(), nil
	}

	value, err := parser.parseStatement()
	if err != nil {
		return UndefinedValue(), err
	}

	parser.skipSpaceAndComments()
	for parser.consumeByte(';') {
		parser.skipSpaceAndComments()
	}
	if !parser.eof() {
		return UndefinedValue(), NewError(
			ErrorKindUnsupported,
			"unsupported script source; this bounded classic-JS slice only supports expression statements, standalone block statements, `let`/`const`/`var` declarations, block-bodied or single-statement `if` / `while` / `do...while` / `for` statements with explicit terminators, `switch` / `try` statements, class declarations with static blocks, public `static` fields, getter/setter accessors, computed fields and methods, instance fields, bounded `extends` inheritance, and bounded `new Class()` instantiation, member calls on `host`, and the `expr(...)` compatibility helper",
		)
	}

	return value, nil
}

func evalClassicJSStatementWithEnvAndAllowAwaitAndYield(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, UndefinedValue(), false, privateClass, nil)
}

func evalClassicJSExpressionWithEnv(source string, host HostBindings, env *classicJSEnvironment, stepLimit int) (Value, error) {
	return evalClassicJSExpressionWithEnvAndAllowAwait(source, host, env, stepLimit, false, nil)
}

func evalClassicJSExpressionWithEnvAndAllowAwait(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, privateClass ...*classicJSClassDefinition) (Value, error) {
	var classDef *classicJSClassDefinition
	if len(privateClass) > 0 {
		classDef = privateClass[0]
	}
	return evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, false, UndefinedValue(), false, classDef, nil)
}

func evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, allowYield, UndefinedValue(), false, privateClass, nil)
}

func evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, newTarget Value, hasNewTarget bool, privateClass *classicJSClassDefinition, moduleExports map[string]Value) (Value, error) {
	if stepLimit <= 0 {
		stepLimit = DefaultRuntimeConfig().StepLimit
	}
	parser := &classicJSStatementParser{
		source:             strings.TrimSpace(source),
		host:               host,
		env:                env,
		privateClass:       privateClass,
		privateFieldPrefix: "",
		stepLimit:          stepLimit,
		allowAwait:         allowAwait,
		allowYield:         allowYield,
		newTarget:          newTarget,
		hasNewTarget:       hasNewTarget,
		moduleExports:      moduleExports,
	}
	if privateClass != nil {
		parser.privateFieldPrefix = privateClass.privateFieldPrefix
	}
	if parser.source == "" {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
	}

	value, err := parser.parseExpression()
	if err != nil {
		return UndefinedValue(), err
	}
	parser.skipSpaceAndComments()
	if !parser.eof() {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected trailing tokens in bounded classic-JS expression")
	}
	return value, nil
}

type classicJSStatementParser struct {
	source                  string
	host                    HostBindings
	env                     *classicJSEnvironment
	privateClass            *classicJSClassDefinition
	privateFieldPrefix      string
	newTarget               Value
	hasNewTarget            bool
	statementLabel          string
	allowUnknownIdentifiers bool
	allowAwait              bool
	allowYield              bool
	allowReturn             bool
	resumeState             classicJSResumeState
	generatorNextValue      Value
	hasGeneratorNextValue   bool
	moduleExports           map[string]Value
	stepLimit               int
	pos                     int
}

type classicJSSwitchClause struct {
	kind  string
	label string
	body  string
}

type classicJSClassMember struct {
	kind             string
	fieldName        string
	fieldNameSource  string
	fieldInit        string
	private          bool
	async            bool
	generator        bool
	staticBlock      string
	methodName       string
	methodNameSource string
	methodParams     []classicJSFunctionParameter
	methodRestName   string
	methodBody       string
}

type classicJSFunctionParameter struct {
	name          string
	pattern       classicJSBindingPattern
	hasPattern    bool
	defaultSource string
}

type classicJSResumeState interface {
	cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState
}

type classicJSArrowFunction struct {
	params             []classicJSFunctionParameter
	restName           string
	name               string
	body               string
	bodyIsBlock        bool
	async              bool
	allowReturn        bool
	objectAccessor     bool
	objectSetter       bool
	objectMethod       bool
	isArrow            bool
	constructible      bool
	constructMarker    string
	newTarget          Value
	hasNewTarget       bool
	env                *classicJSEnvironment
	privateClass       *classicJSClassDefinition
	privateFieldPrefix string
	superTarget        Value
	hasSuperTarget     bool
	generatorMethod    string
	generatorFunction  *classicJSGeneratorFunction
	generatorState     *classicJSGeneratorState
}

type classicJSGeneratorFunction struct {
	name               string
	params             []classicJSFunctionParameter
	restName           string
	body               string
	async              bool
	env                *classicJSEnvironment
	privateClass       *classicJSClassDefinition
	privateFieldPrefix string
	superTarget        Value
	hasSuperTarget     bool
}

type classicJSGeneratorState struct {
	statements         []string
	env                *classicJSEnvironment
	async              bool
	newTarget          Value
	hasNewTarget       bool
	index              int
	done               bool
	hasYielded         bool
	activeState        classicJSResumeState
	delegateArray      []Value
	delegateArrayIndex int
	delegateIterator   *Value
}

type classicJSYieldDeclarationState struct {
	env                *classicJSEnvironment
	kind               string
	name               string
	pattern            classicJSBindingPattern
	hasPattern         bool
	privateClass       *classicJSClassDefinition
	privateFieldPrefix string
}

func (s *classicJSYieldDeclarationState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSYieldDeclarationState{
		kind:               s.kind,
		name:               s.name,
		pattern:            s.pattern,
		hasPattern:         s.hasPattern,
		privateClass:       s.privateClass,
		privateFieldPrefix: s.privateFieldPrefix,
	}
	if s.env != nil {
		if clonedEnv, ok := mapping[s.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = s.env.cloneDetachedWithMapping(mapping)
		}
	}
	return cloned
}

type classicJSYieldAssignmentState struct {
	env                *classicJSEnvironment
	name               string
	steps              []classicJSDeleteStep
	op                 string
	current            jsValue
	privateFieldPrefix string
}

func (s *classicJSYieldAssignmentState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSYieldAssignmentState{
		name:               s.name,
		steps:              append([]classicJSDeleteStep(nil), s.steps...),
		op:                 s.op,
		current:            cloneJSValueDetached(s.current, mapping),
		privateFieldPrefix: s.privateFieldPrefix,
	}
	if s.env != nil {
		if clonedEnv, ok := mapping[s.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = s.env.cloneDetachedWithMapping(mapping)
		}
	}
	return cloned
}

type classicJSBlockState struct {
	statements []string
	env        *classicJSEnvironment
	owner      classicJSResumeState
	index      int
	child      classicJSResumeState
	lastValue  Value
}

type classicJSTryStage int

const (
	classicJSTryStageTry classicJSTryStage = iota
	classicJSTryStageCatch
	classicJSTryStageFinally
	classicJSTryStageDone
)

type classicJSSwitchState struct {
	label          string
	clauses        []classicJSSwitchClause
	env            *classicJSEnvironment
	clauseIndex    int
	bodyStatements []string
	bodyIndex      int
	bodyState      classicJSResumeState
}

type classicJSTryState struct {
	label            string
	tryBlock         *classicJSBlockState
	catchBlock       *classicJSBlockState
	finallyBlock     *classicJSBlockState
	catchSource      string
	finallySource    string
	catchEnvTemplate *classicJSEnvironment
	catchPattern     classicJSBindingPattern
	catchBound       bool
	hasCatch         bool
	hasFinally       bool
	hasPendingThrow  bool
	stage            classicJSTryStage
	result           Value
	pendingThrow     Value
	pendingErr       error
}

type classicJSLoopKind int

const (
	classicJSLoopKindWhile classicJSLoopKind = iota
	classicJSLoopKindDoWhile
	classicJSLoopKindFor
	classicJSLoopKindForOf
	classicJSLoopKindForIn
)

type classicJSLoopState struct {
	label           string
	parent          *classicJSLoopState
	kind            classicJSLoopKind
	loopEnv         *classicJSEnvironment
	initSource      string
	conditionSource string
	updateSource    string
	bodyStatements  []string
	bodyIndex       int
	bodyEnv         *classicJSEnvironment
	bodyState       classicJSResumeState
	initDone        bool
	started         bool
	iterationCount  int
	forOfKind       string
	forOfPattern    classicJSBindingPattern
	forOfValues     []Value
	forOfIterator   *Value
	forOfIndex      int
	forOfAwait      bool
	forInKind       string
	forInPattern    classicJSBindingPattern
	forInKeys       []Value
	forInIndex      int
}

type classicJSDeleteStep struct {
	key      string
	private  bool
	optional bool
}

type classicJSYieldSignal struct {
	value       Value
	resumeState classicJSResumeState
}

func (s classicJSYieldSignal) Error() string {
	return "generator yield"
}

func classicJSYieldSignalValue(err error) (Value, bool) {
	signal, ok := err.(classicJSYieldSignal)
	if !ok {
		return UndefinedValue(), false
	}
	return signal.value, true
}

func classicJSYieldSignalDetails(err error) (Value, classicJSResumeState, bool) {
	signal, ok := err.(classicJSYieldSignal)
	if !ok {
		return UndefinedValue(), nil, false
	}
	return signal.value, signal.resumeState, true
}

type classicJSYieldDelegationState struct {
	delegateArray      []Value
	delegateArrayIndex int
	delegateIterator   *Value
}

func (s *classicJSYieldDelegationState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSYieldDelegationState{
		delegateArray:      append([]Value(nil), s.delegateArray...),
		delegateArrayIndex: s.delegateArrayIndex,
	}
	if s.delegateIterator != nil {
		clonedValue := cloneValueDetached(*s.delegateIterator, mapping)
		cloned.delegateIterator = &clonedValue
	}
	for i, value := range cloned.delegateArray {
		cloned.delegateArray[i] = cloneValueDetached(value, mapping)
	}
	return cloned
}

type classicJSReturnSignal struct {
	value Value
}

func (s classicJSReturnSignal) Error() string {
	return "function return"
}

func classicJSReturnSignalValue(err error) (Value, bool) {
	signal, ok := err.(classicJSReturnSignal)
	if !ok {
		return UndefinedValue(), false
	}
	return signal.value, true
}

type classicJSThrowSignal struct {
	value Value
}

func (s classicJSThrowSignal) Error() string {
	return ToJSString(s.value)
}

func classicJSThrowSignalValue(err error) (Value, bool) {
	signal, ok := err.(classicJSThrowSignal)
	if !ok {
		return UndefinedValue(), false
	}
	return signal.value, true
}

type classicJSBreakSignal struct{ label string }

func (s classicJSBreakSignal) Error() string {
	if s.label == "" {
		return "break signal"
	}
	return fmt.Sprintf("break signal for label %q", s.label)
}

func classicJSBreakSignalValue(err error) bool {
	_, ok := err.(classicJSBreakSignal)
	return ok
}

func classicJSBreakSignalLabel(err error) (string, bool) {
	signal, ok := err.(classicJSBreakSignal)
	if !ok {
		return "", false
	}
	return signal.label, true
}

func classicJSBreakSignalMatchesLabel(err error, label string) bool {
	signalLabel, ok := classicJSBreakSignalLabel(err)
	if !ok {
		return false
	}
	return signalLabel == "" || signalLabel == label
}

type classicJSContinueSignal struct{ label string }

func (s classicJSContinueSignal) Error() string {
	if s.label == "" {
		return "continue signal"
	}
	return fmt.Sprintf("continue signal for label %q", s.label)
}

func classicJSContinueSignalValue(err error) bool {
	_, ok := err.(classicJSContinueSignal)
	return ok
}

func classicJSContinueSignalLabel(err error) (string, bool) {
	signal, ok := err.(classicJSContinueSignal)
	if !ok {
		return "", false
	}
	return signal.label, true
}

func classicJSContinueSignalMatchesLabel(err error, label string) bool {
	signalLabel, ok := classicJSContinueSignalLabel(err)
	if !ok {
		return false
	}
	return signalLabel == "" || signalLabel == label
}

func (f *classicJSArrowFunction) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSArrowFunction {
	if f == nil {
		return nil
	}
	cloned := &classicJSArrowFunction{
		params:             append([]classicJSFunctionParameter(nil), f.params...),
		restName:           f.restName,
		name:               f.name,
		body:               f.body,
		bodyIsBlock:        f.bodyIsBlock,
		async:              f.async,
		allowReturn:        f.allowReturn,
		objectAccessor:     f.objectAccessor,
		objectSetter:       f.objectSetter,
		objectMethod:       f.objectMethod,
		isArrow:            f.isArrow,
		constructible:      f.constructible,
		constructMarker:    f.constructMarker,
		newTarget:          cloneValueDetached(f.newTarget, mapping),
		hasNewTarget:       f.hasNewTarget,
		privateClass:       f.privateClass,
		privateFieldPrefix: f.privateFieldPrefix,
		superTarget:        cloneValueDetached(f.superTarget, mapping),
		hasSuperTarget:     f.hasSuperTarget,
		generatorMethod:    f.generatorMethod,
	}
	if f.env != nil {
		if clonedEnv, ok := mapping[f.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = f.env.cloneDetachedWithMapping(mapping)
		}
	}
	if f.generatorFunction != nil {
		cloned.generatorFunction = f.generatorFunction.cloneDetached(mapping)
	}
	if f.generatorState != nil {
		cloned.generatorState = f.generatorState.cloneDetached(mapping)
	}
	return cloned
}

func (f *classicJSGeneratorFunction) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSGeneratorFunction {
	if f == nil {
		return nil
	}
	cloned := &classicJSGeneratorFunction{
		name:               f.name,
		params:             append([]classicJSFunctionParameter(nil), f.params...),
		restName:           f.restName,
		body:               f.body,
		async:              f.async,
		privateClass:       f.privateClass,
		privateFieldPrefix: f.privateFieldPrefix,
		superTarget:        cloneValueDetached(f.superTarget, mapping),
		hasSuperTarget:     f.hasSuperTarget,
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

func (s *classicJSGeneratorState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSGeneratorState {
	if s == nil {
		return nil
	}
	cloned := &classicJSGeneratorState{
		statements:         append([]string(nil), s.statements...),
		index:              s.index,
		done:               s.done,
		hasYielded:         s.hasYielded,
		async:              s.async,
		newTarget:          cloneValueDetached(s.newTarget, mapping),
		hasNewTarget:       s.hasNewTarget,
		delegateArray:      append([]Value(nil), s.delegateArray...),
		delegateArrayIndex: s.delegateArrayIndex,
	}
	if s.env != nil {
		if clonedEnv, ok := mapping[s.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = s.env.cloneDetachedWithMapping(mapping)
		}
	}
	if s.activeState != nil {
		cloned.activeState = cloneDetachedClassicJSResumeState(s.activeState, mapping)
	}
	if s.delegateIterator != nil {
		clonedValue := cloneValueDetached(*s.delegateIterator, mapping)
		cloned.delegateIterator = &clonedValue
	}
	return cloned
}

func classicJSResumeStateHasYieldDelegation(state classicJSResumeState) bool {
	switch current := state.(type) {
	case *classicJSYieldDelegationState:
		return true
	case *classicJSBlockState:
		return classicJSResumeStateHasYieldDelegation(current.child)
	case *classicJSLoopState:
		return classicJSResumeStateHasYieldDelegation(current.bodyState)
	case *classicJSSwitchState:
		return classicJSResumeStateHasYieldDelegation(current.bodyState)
	case *classicJSTryState:
		return classicJSResumeStateHasYieldDelegation(current.tryBlock) ||
			classicJSResumeStateHasYieldDelegation(current.catchBlock) ||
			classicJSResumeStateHasYieldDelegation(current.finallyBlock)
	default:
		return false
	}
}

func cloneDetachedClassicJSResumeState(state classicJSResumeState, mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if state == nil {
		return nil
	}
	return state.cloneDetached(mapping)
}

func (s *classicJSBlockState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSBlockState{
		statements: append([]string(nil), s.statements...),
		index:      s.index,
		lastValue:  cloneValueDetached(s.lastValue, mapping),
	}
	if s.env != nil {
		if clonedEnv, ok := mapping[s.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = s.env.cloneDetachedWithMapping(mapping)
		}
	}
	if s.child != nil {
		cloned.child = cloneDetachedClassicJSResumeState(s.child, mapping)
	}
	return cloned
}

func (s *classicJSSwitchState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSSwitchState{
		label:          s.label,
		clauses:        append([]classicJSSwitchClause(nil), s.clauses...),
		clauseIndex:    s.clauseIndex,
		bodyStatements: append([]string(nil), s.bodyStatements...),
		bodyIndex:      s.bodyIndex,
	}
	if s.env != nil {
		if clonedEnv, ok := mapping[s.env]; ok {
			cloned.env = clonedEnv
		} else {
			cloned.env = s.env.cloneDetachedWithMapping(mapping)
		}
	}
	if s.bodyState != nil {
		cloned.bodyState = cloneDetachedClassicJSResumeState(s.bodyState, mapping)
	}
	return cloned
}

func (s *classicJSTryState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSTryState{
		label:           s.label,
		catchSource:     s.catchSource,
		finallySource:   s.finallySource,
		catchPattern:    s.catchPattern,
		catchBound:      s.catchBound,
		hasCatch:        s.hasCatch,
		hasFinally:      s.hasFinally,
		hasPendingThrow: s.hasPendingThrow,
		stage:           s.stage,
		result:          cloneValueDetached(s.result, mapping),
		pendingThrow:    cloneValueDetached(s.pendingThrow, mapping),
	}
	if s.pendingErr != nil {
		cloned.pendingErr = fmt.Errorf("%s", s.pendingErr.Error())
	}
	if s.catchEnvTemplate != nil {
		if clonedEnv, ok := mapping[s.catchEnvTemplate]; ok {
			cloned.catchEnvTemplate = clonedEnv
		} else {
			cloned.catchEnvTemplate = s.catchEnvTemplate.cloneDetachedWithMapping(mapping)
		}
	}
	if s.tryBlock != nil {
		cloned.tryBlock = s.tryBlock.cloneDetached(mapping).(*classicJSBlockState)
		cloned.tryBlock.owner = cloned
	}
	if s.catchBlock != nil {
		cloned.catchBlock = s.catchBlock.cloneDetached(mapping).(*classicJSBlockState)
		cloned.catchBlock.owner = cloned
	}
	if s.finallyBlock != nil {
		cloned.finallyBlock = s.finallyBlock.cloneDetached(mapping).(*classicJSBlockState)
		cloned.finallyBlock.owner = cloned
	}
	return cloned
}

func (s *classicJSLoopState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) classicJSResumeState {
	if s == nil {
		return nil
	}
	cloned := &classicJSLoopState{
		label:           s.label,
		kind:            s.kind,
		initSource:      s.initSource,
		conditionSource: s.conditionSource,
		updateSource:    s.updateSource,
		bodyStatements:  append([]string(nil), s.bodyStatements...),
		bodyIndex:       s.bodyIndex,
		initDone:        s.initDone,
		started:         s.started,
		iterationCount:  s.iterationCount,
		forOfKind:       s.forOfKind,
		forOfPattern:    s.forOfPattern,
		forOfValues:     make([]Value, len(s.forOfValues)),
		forOfIterator:   nil,
		forOfIndex:      s.forOfIndex,
		forOfAwait:      s.forOfAwait,
		forInKind:       s.forInKind,
		forInPattern:    s.forInPattern,
		forInKeys:       make([]Value, len(s.forInKeys)),
		forInIndex:      s.forInIndex,
	}
	for i, value := range s.forOfValues {
		cloned.forOfValues[i] = cloneValueDetached(value, mapping)
	}
	if s.forOfIterator != nil {
		clonedIterator := *s.forOfIterator
		cloned.forOfIterator = &clonedIterator
	}
	for i, value := range s.forInKeys {
		cloned.forInKeys[i] = cloneValueDetached(value, mapping)
	}
	if s.loopEnv != nil {
		if clonedEnv, ok := mapping[s.loopEnv]; ok {
			cloned.loopEnv = clonedEnv
		} else {
			cloned.loopEnv = s.loopEnv.cloneDetachedWithMapping(mapping)
		}
	}
	if s.bodyEnv != nil {
		if clonedEnv, ok := mapping[s.bodyEnv]; ok {
			cloned.bodyEnv = clonedEnv
		} else {
			cloned.bodyEnv = s.bodyEnv.cloneDetachedWithMapping(mapping)
		}
	}
	if s.bodyState != nil {
		cloned.bodyState = cloneDetachedClassicJSResumeState(s.bodyState, mapping)
	}
	return cloned
}

func (p *classicJSStatementParser) cloneForSkipping(host HostBindings) *classicJSStatementParser {
	env := p.env
	if env != nil {
		env = env.cloneDetached()
	}
	return &classicJSStatementParser{
		source:                  p.source,
		host:                    host,
		env:                     env,
		privateClass:            p.privateClass,
		privateFieldPrefix:      p.privateFieldPrefix,
		newTarget:               p.newTarget,
		hasNewTarget:            p.hasNewTarget,
		statementLabel:          p.statementLabel,
		allowUnknownIdentifiers: true,
		allowAwait:              p.allowAwait,
		allowYield:              p.allowYield,
		allowReturn:             p.allowReturn,
		resumeState:             p.resumeState,
		moduleExports:           nil,
		stepLimit:               p.stepLimit,
		pos:                     p.pos,
	}
}

func (p *classicJSStatementParser) cloneForClassEvaluation() *classicJSStatementParser {
	if p == nil {
		return &classicJSStatementParser{}
	}
	cloned := *p
	cloned.allowAwait = false
	cloned.allowYield = false
	cloned.allowReturn = false
	return &cloned
}

func (p *classicJSStatementParser) evalStatementWithEnv(source string, env *classicJSEnvironment) (Value, error) {
	return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source, p.host, env, p.stepLimit, p.allowAwait, p.allowYield, p.allowReturn, p.resumeState, p.newTarget, p.hasNewTarget, p.privateClass, nil)
}

func (p *classicJSStatementParser) evalProgramWithEnv(source string, env *classicJSEnvironment) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, p.host, env, p.stepLimit, p.allowAwait, p.allowYield, p.allowReturn, p.resumeState, p.newTarget, p.hasNewTarget, p.privateClass, nil)
}

func (p *classicJSStatementParser) evalExpressionWithEnv(source string, env *classicJSEnvironment) (Value, error) {
	return evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source, p.host, env, p.stepLimit, p.allowAwait, p.allowYield, p.newTarget, p.hasNewTarget, p.privateClass, nil)
}

func (p *classicJSStatementParser) eof() bool {
	return p == nil || p.pos >= len(p.source)
}

func (p *classicJSStatementParser) peekByte() byte {
	if p.eof() {
		return 0
	}
	return p.source[p.pos]
}

func (p *classicJSStatementParser) consumeByte(want byte) bool {
	if p.eof() || p.source[p.pos] != want {
		return false
	}
	p.pos++
	return true
}

func (p *classicJSStatementParser) consumeEllipsis() bool {
	if p.pos+3 > len(p.source) || p.source[p.pos:p.pos+3] != "..." {
		return false
	}
	p.pos += 3
	return true
}

func (p *classicJSStatementParser) skipSpaceAndComments() {
	for !p.eof() {
		switch p.source[p.pos] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				return
			}
			switch p.source[p.pos+1] {
			case '/':
				p.pos += 2
				for !p.eof() {
					if p.source[p.pos] == '\n' || p.source[p.pos] == '\r' {
						break
					}
					p.pos++
				}
			case '*':
				p.pos += 2
				for !p.eof() {
					if p.source[p.pos] == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
						p.pos += 2
						break
					}
					p.pos++
				}
			default:
				return
			}
		default:
			return
		}
	}
}

func (p *classicJSStatementParser) parseExpression() (Value, error) {
	p.skipSpaceAndComments()
	value, err := p.parseSequenceExpression()
	if err != nil {
		return UndefinedValue(), err
	}
	if value.kind != jsValueScalar {
		return UndefinedValue(), NewError(
			ErrorKindUnsupported,
			"unsupported script source; incomplete or unsupported expression in this bounded classic-JS slice",
		)
	}
	return value.value, nil
}

func (p *classicJSStatementParser) parseScalarExpression() (Value, error) {
	value, err := p.parseLogicalAssignment()
	if err != nil {
		return UndefinedValue(), err
	}
	if value.kind != jsValueScalar {
		return UndefinedValue(), NewError(
			ErrorKindUnsupported,
			"unsupported script source; incomplete or unsupported expression in this bounded classic-JS slice",
		)
	}
	return value.value, nil
}

func (p *classicJSStatementParser) parseSequenceExpression() (jsValue, error) {
	left, err := p.parseLogicalAssignment()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if !p.consumeByte(',') {
			return left, nil
		}
		right, err := p.parseLogicalAssignment()
		if err != nil {
			return jsValue{}, err
		}
		left = right
	}
}

func (p *classicJSStatementParser) parseLogicalAssignment() (jsValue, error) {
	p.skipSpaceAndComments()
	start := p.pos
	name, steps, op, rhsPos, ok, err := p.tryParseAssignmentTarget()
	if err != nil {
		return jsValue{}, err
	}
	if !ok {
		p.pos = start
		return p.parseConditional()
	}
	if p.env == nil {
		if p.allowUnknownIdentifiers {
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = rhsPos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(UndefinedValue()), nil
		}
		if len(steps) == 0 {
			return jsValue{}, NewError(ErrorKindUnsupported, "logical assignment only works on declared local bindings in this bounded classic-JS slice")
		}
		return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on declared local bindings in this bounded classic-JS slice")
	}
	current, ok := p.env.lookup(name)
	if !ok {
		if p.allowUnknownIdentifiers {
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = rhsPos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(UndefinedValue()), nil
		}
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("assignment target %q is not a declared local binding in this bounded classic-JS slice", name))
	}

	p.pos = rhsPos
	wrapAssignmentYield := func(err error) (jsValue, error) {
		if yieldedValue, _, ok := classicJSYieldSignalDetails(err); ok {
			return jsValue{}, p.wrapYieldAssignmentSignal(yieldedValue, name, steps, op, current)
		}
		return jsValue{}, err
	}
	if len(steps) > 0 {
		if current.kind == jsValueSuper {
			if current.value.Kind != ValueKindObject && current.value.Kind != ValueKindNull {
				return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on object values in this bounded classic-JS slice")
			}

			if op == "=" {
				value, err := p.parseLogicalAssignment()
				if err != nil {
					return wrapAssignmentYield(err)
				}
				if value.kind != jsValueScalar {
					return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
				}
				if _, err := assignSuperJSValuePropertyChain(p, current, steps, value.value, p.privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return value, nil
			}

			currentValueSource := current.value
			if current.value.Kind == ValueKindNull && len(steps) == 1 && current.receiver.Kind == ValueKindObject {
				currentValueSource = current.receiver
			}
			currentValue, err := resolveJSValuePropertyChain(p, currentValueSource, steps, p.privateFieldPrefix)
			if err != nil {
				return jsValue{}, err
			}

			switch op {
			case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
				value, err := p.parseLogicalAssignment()
				if err != nil {
					return wrapAssignmentYield(err)
				}
				if value.kind != jsValueScalar {
					return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
				}
				result, err := classicJSApplyCompoundAssignment(currentValue, value.value, op)
				if err != nil {
					return jsValue{}, err
				}
				if _, err := assignSuperJSValuePropertyChain(p, current, steps, result, p.privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(result), nil
			case "||=":
				if !jsTruthy(currentValue) {
					value, err := p.parseLogicalAssignment()
					if err != nil {
						return wrapAssignmentYield(err)
					}
					if value.kind != jsValueScalar {
						return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
					}
					if _, err := assignSuperJSValuePropertyChain(p, current, steps, value.value, p.privateFieldPrefix); err != nil {
						return jsValue{}, err
					}
					return value, nil
				}
				skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
				skip.pos = rhsPos
				if _, err := skip.parseLogicalAssignment(); err != nil {
					return jsValue{}, err
				}
				p.pos = skip.pos
				return scalarJSValue(currentValue), nil
			case "&&=":
				if jsTruthy(currentValue) {
					value, err := p.parseLogicalAssignment()
					if err != nil {
						return wrapAssignmentYield(err)
					}
					if value.kind != jsValueScalar {
						return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
					}
					if _, err := assignSuperJSValuePropertyChain(p, current, steps, value.value, p.privateFieldPrefix); err != nil {
						return jsValue{}, err
					}
					return value, nil
				}
				skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
				skip.pos = rhsPos
				if _, err := skip.parseLogicalAssignment(); err != nil {
					return jsValue{}, err
				}
				p.pos = skip.pos
				return scalarJSValue(currentValue), nil
			case "??=":
				if isNullishJSValue(currentValue) {
					value, err := p.parseLogicalAssignment()
					if err != nil {
						return wrapAssignmentYield(err)
					}
					if value.kind != jsValueScalar {
						return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
					}
					if _, err := assignSuperJSValuePropertyChain(p, current, steps, value.value, p.privateFieldPrefix); err != nil {
						return jsValue{}, err
					}
					return value, nil
				}
				skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
				skip.pos = rhsPos
				if _, err := skip.parseLogicalAssignment(); err != nil {
					return jsValue{}, err
				}
				p.pos = skip.pos
				return scalarJSValue(currentValue), nil
			case "**=":
				value, err := p.parseLogicalAssignment()
				if err != nil {
					return wrapAssignmentYield(err)
				}
				if value.kind != jsValueScalar {
					return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
				}
				result, err := classicJSPowerValues(currentValue, value.value)
				if err != nil {
					return jsValue{}, err
				}
				if _, err := assignSuperJSValuePropertyChain(p, current, steps, result, p.privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(result), nil
			default:
				return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
			}
		}
		if current.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar object or array bindings in this bounded classic-JS slice")
		}
		if current.value.Kind != ValueKindObject && current.value.Kind != ValueKindArray && current.value.Kind != ValueKindHostReference {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
		}

		if op == "=" {
			value, err := p.parseLogicalAssignment()
			if err != nil {
				return wrapAssignmentYield(err)
			}
			if value.kind != jsValueScalar {
				return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
			}
			if _, err := assignJSValuePropertyChain(p, current.value, steps, value.value, p.privateFieldPrefix); err != nil {
				return jsValue{}, err
			}
			return value, nil
		}

		currentValue, err := resolveJSValuePropertyChain(p, current.value, steps, p.privateFieldPrefix)
		if err != nil {
			return jsValue{}, err
		}

		switch op {
		case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
			value, err := p.parseLogicalAssignment()
			if err != nil {
				return wrapAssignmentYield(err)
			}
			if value.kind != jsValueScalar {
				return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
			}
			result, err := classicJSApplyCompoundAssignment(currentValue, value.value, op)
			if err != nil {
				return jsValue{}, err
			}
			if _, err := assignJSValuePropertyChain(p, current.value, steps, result, p.privateFieldPrefix); err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(result), nil
		case "||=":
			if !jsTruthy(currentValue) {
				value, err := p.parseLogicalAssignment()
				if err != nil {
					return wrapAssignmentYield(err)
				}
				if value.kind != jsValueScalar {
					return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
				}
				if _, err := assignJSValuePropertyChain(p, current.value, steps, value.value, p.privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return value, nil
			}
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = rhsPos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(currentValue), nil
		case "&&=":
			if jsTruthy(currentValue) {
				value, err := p.parseLogicalAssignment()
				if err != nil {
					return wrapAssignmentYield(err)
				}
				if value.kind != jsValueScalar {
					return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
				}
				if _, err := assignJSValuePropertyChain(p, current.value, steps, value.value, p.privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return value, nil
			}
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = rhsPos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(currentValue), nil
		case "??=":
			if isNullishJSValue(currentValue) {
				value, err := p.parseLogicalAssignment()
				if err != nil {
					return wrapAssignmentYield(err)
				}
				if value.kind != jsValueScalar {
					return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
				}
				if _, err := assignJSValuePropertyChain(p, current.value, steps, value.value, p.privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return value, nil
			}
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = rhsPos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(currentValue), nil
		case "**=":
			value, err := p.parseLogicalAssignment()
			if err != nil {
				return wrapAssignmentYield(err)
			}
			if value.kind != jsValueScalar {
				return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
			}
			result, err := classicJSPowerValues(currentValue, value.value)
			if err != nil {
				return jsValue{}, err
			}
			if _, err := assignJSValuePropertyChain(p, current.value, steps, result, p.privateFieldPrefix); err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(result), nil
		default:
			return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
		}
	}

	if op == "" {
		p.pos = start
		return p.parseConditional()
	}

	switch op {
	case "=":
		rhs, err := p.parseLogicalAssignment()
		if err != nil {
			return wrapAssignmentYield(err)
		}
		if rhs.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
		}
		if p.env == nil {
			return jsValue{}, NewError(ErrorKindRuntime, "generator state environment is unavailable")
		}
		if err := p.env.assign(name, rhs); err != nil {
			return jsValue{}, err
		}
		return rhs, nil
	case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
		value, err := p.parseLogicalAssignment()
		if err != nil {
			return wrapAssignmentYield(err)
		}
		result, err := classicJSApplyCompoundAssignment(current.value, value.value, op)
		if err != nil {
			return jsValue{}, err
		}
		if err := p.env.assign(name, scalarJSValue(result)); err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(result), nil
	case "||=":
		if !jsTruthy(current.value) {
			value, err := p.parseLogicalAssignment()
			if err != nil {
				return wrapAssignmentYield(err)
			}
			if err := p.env.assign(name, value); err != nil {
				return jsValue{}, err
			}
			return value, nil
		}
		skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
		skip.pos = rhsPos
		if _, err := skip.parseLogicalAssignment(); err != nil {
			return jsValue{}, err
		}
		p.pos = skip.pos
		return current, nil
	case "&&=":
		if jsTruthy(current.value) {
			value, err := p.parseLogicalAssignment()
			if err != nil {
				return wrapAssignmentYield(err)
			}
			if err := p.env.assign(name, value); err != nil {
				return jsValue{}, err
			}
			return value, nil
		}
		skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
		skip.pos = rhsPos
		if _, err := skip.parseLogicalAssignment(); err != nil {
			return jsValue{}, err
		}
		p.pos = skip.pos
		return current, nil
	case "??=":
		if isNullishJSValue(current.value) {
			value, err := p.parseLogicalAssignment()
			if err != nil {
				return wrapAssignmentYield(err)
			}
			if err := p.env.assign(name, value); err != nil {
				return jsValue{}, err
			}
			return value, nil
		}
		skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
		skip.pos = rhsPos
		if _, err := skip.parseLogicalAssignment(); err != nil {
			return jsValue{}, err
		}
		p.pos = skip.pos
		return current, nil
	case "**=":
		value, err := p.parseLogicalAssignment()
		if err != nil {
			return wrapAssignmentYield(err)
		}
		result, err := classicJSPowerValues(current.value, value.value)
		if err != nil {
			return jsValue{}, err
		}
		if err := p.env.assign(name, scalarJSValue(result)); err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(result), nil
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
	}
}

func resolveJSValuePropertyChain(p *classicJSStatementParser, value Value, steps []classicJSDeleteStep, privateFieldPrefix string) (Value, error) {
	if len(steps) == 0 {
		return value, nil
	}

	key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}
	switch value.Kind {
	case ValueKindObject:
		resolved, err := p.resolveMemberAccess(scalarJSValue(value), key)
		if err != nil {
			return UndefinedValue(), err
		}
		if len(steps) == 1 {
			if resolved.kind != jsValueScalar {
				return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object values in this bounded classic-JS slice")
			}
			return resolved.value, nil
		}
		if resolved.kind != jsValueScalar || (resolved.value.Kind != ValueKindObject && resolved.value.Kind != ValueKindArray) {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object or array values in this bounded classic-JS slice")
		}
		return resolveJSValuePropertyChain(p, resolved.value, steps[1:], privateFieldPrefix)
	case ValueKindArray:
		if key == "length" {
			if len(steps) == 1 {
				return NumberValue(float64(len(value.Array))), nil
			}
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object or array values in this bounded classic-JS slice")
		}
		index, ok := arrayIndexFromBracketKey(key)
		if !ok {
			if len(steps) == 1 {
				return UndefinedValue(), nil
			}
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object or array values in this bounded classic-JS slice")
		}
		if index >= len(value.Array) {
			if len(steps) == 1 {
				return UndefinedValue(), nil
			}
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing array elements in this bounded classic-JS slice")
		}
		child := value.Array[index]
		if len(steps) == 1 {
			return child, nil
		}
		if child.Kind != ValueKindObject && child.Kind != ValueKindArray {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object or array values in this bounded classic-JS slice")
		}
		return resolveJSValuePropertyChain(p, child, steps[1:], privateFieldPrefix)
	case ValueKindHostReference:
		resolved, err := p.resolveHostReferencePath(joinHostReferencePath(value.HostReferencePath, key))
		if err != nil {
			return UndefinedValue(), err
		}
		if len(steps) == 1 {
			return resolved, nil
		}
		if resolved.Kind != ValueKindObject && resolved.Kind != ValueKindArray && resolved.Kind != ValueKindHostReference {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
		}
		return resolveJSValuePropertyChain(p, resolved, steps[1:], privateFieldPrefix)
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
	}
}

func (p *classicJSStatementParser) tryParseAssignmentTarget() (string, []classicJSDeleteStep, string, int, bool, error) {
	if p == nil {
		return "", nil, "", 0, false, nil
	}

	lookahead := *p
	lookahead.skipSpaceAndComments()
	if lookahead.eof() || !isIdentStart(lookahead.peekByte()) {
		return "", nil, "", 0, false, nil
	}

	name, err := lookahead.parseIdentifier()
	if err != nil {
		return "", nil, "", 0, false, err
	}
	if name != "super" && name != "this" && isClassicJSReservedDeclarationName(name) {
		return "", nil, "", 0, false, nil
	}

	steps, ok, err := lookahead.scanAssignmentAccessSteps()
	if err != nil {
		return "", nil, "", 0, false, err
	}
	if !ok {
		return "", nil, "", 0, false, nil
	}

	lookahead.skipSpaceAndComments()
	if op := lookahead.peekLogicalAssignmentOperator(); op != "" {
		lookahead.pos += len(op)
		return name, steps, op, lookahead.pos, true, nil
	}

	if lookahead.peekByte() == '=' {
		if lookahead.pos+1 >= len(lookahead.source) || (lookahead.source[lookahead.pos+1] != '=' && lookahead.source[lookahead.pos+1] != '>') {
			lookahead.pos++
			return name, steps, "=", lookahead.pos, true, nil
		}
	}

	return "", nil, "", 0, false, nil
}

func (p *classicJSStatementParser) parseStatement() (Value, error) {
	p.skipSpaceAndComments()
	prevLabel := p.statementLabel
	if label, ok, err := p.consumeStatementLabel(); err != nil {
		return UndefinedValue(), err
	} else if ok {
		p.statementLabel = label
		defer func() {
			p.statementLabel = prevLabel
		}()
	}
	if keyword, ok := p.peekKeyword("let"); ok {
		p.pos += len(keyword)
		return p.parseVariableDeclaration("let")
	}
	if keyword, ok := p.peekKeyword("var"); ok {
		p.pos += len(keyword)
		return p.parseVariableDeclaration("var")
	}
	if keyword, ok := p.peekKeyword("const"); ok {
		p.pos += len(keyword)
		return p.parseVariableDeclaration("const")
	}
	if keyword, ok := p.peekKeyword("using"); ok {
		p.pos += len(keyword)
		return p.parseUsingDeclaration("using")
	}
	if keyword, ok := p.peekKeyword("await"); ok {
		start := p.pos
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		if usingKeyword, ok := p.peekKeyword("using"); ok {
			if !p.allowAwait {
				p.pos = start
			} else {
				p.pos += len(usingKeyword)
				return p.parseUsingDeclaration("await using")
			}
		} else {
			p.pos = start
		}
	}
	if keyword, ok := p.peekKeyword("function"); ok {
		start := p.pos
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		switch {
		case p.peekByte() == '*':
			lookahead := *p
			lookahead.pos++
			lookahead.skipSpaceAndComments()
			if isIdentStart(lookahead.peekByte()) {
				return p.parseFunctionStatement(false, true)
			}
			p.pos = start
			return p.parseExpression()
		case p.peekByte() == '(':
			p.pos = start
			return p.parseExpression()
		case isIdentStart(p.peekByte()):
			return p.parseFunctionStatement(false, false)
		default:
			p.pos = start
			return p.parseExpression()
		}
	}
	if keyword, ok := p.peekKeyword("async"); ok {
		start := p.pos
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		if functionKeyword, ok := p.peekKeyword("function"); ok {
			p.pos += len(functionKeyword)
			p.skipSpaceAndComments()
			if p.peekByte() == '*' {
				lookahead := *p
				lookahead.pos++
				lookahead.skipSpaceAndComments()
				if isIdentStart(lookahead.peekByte()) {
					return p.parseFunctionStatement(true, true)
				}
				p.pos = start
				return p.parseExpression()
			}
			if isIdentStart(p.peekByte()) {
				return p.parseFunctionStatement(true, false)
			}
			p.pos = start
			return p.parseExpression()
		}
		p.pos = start
	}
	if keyword, ok := p.peekKeyword("while"); ok {
		p.pos += len(keyword)
		return p.parseWhileStatement()
	}
	if keyword, ok := p.peekKeyword("do"); ok {
		p.pos += len(keyword)
		return p.parseDoWhileStatement()
	}
	if keyword, ok := p.peekKeyword("for"); ok {
		p.pos += len(keyword)
		return p.parseForStatement()
	}
	if keyword, ok := p.peekKeyword("class"); ok {
		p.pos += len(keyword)
		return p.parseClassStatement()
	}
	if keyword, ok := p.peekKeyword("switch"); ok {
		p.pos += len(keyword)
		return p.parseSwitchStatement()
	}
	if keyword, ok := p.peekKeyword("try"); ok {
		p.pos += len(keyword)
		return p.parseTryStatement()
	}
	if keyword, ok := p.peekKeyword("with"); ok {
		p.pos += len(keyword)
		return p.parseWithStatement()
	}
	if keyword, ok := p.peekKeyword("if"); ok {
		p.pos += len(keyword)
		return p.parseIfStatement()
	}
	if keyword, ok := p.peekKeyword("break"); ok {
		p.pos += len(keyword)
		return p.parseBreakStatement()
	}
	if keyword, ok := p.peekKeyword("continue"); ok {
		p.pos += len(keyword)
		return p.parseContinueStatement()
	}
	if keyword, ok := p.peekKeyword("return"); ok {
		p.pos += len(keyword)
		return p.parseReturnStatement()
	}
	if keyword, ok := p.peekKeyword("throw"); ok {
		p.pos += len(keyword)
		return p.parseThrowStatement()
	}
	if keyword, ok := p.peekKeyword("debugger"); ok {
		p.pos += len(keyword)
		return p.parseDebuggerStatement()
	}
	if keyword, ok := p.peekKeyword("yield"); ok && p.allowYield {
		p.pos += len(keyword)
		return p.parseYieldStatement()
	}
	if keyword, ok := p.peekKeyword("import"); ok {
		start := p.pos
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		if p.peekByte() == '.' || p.peekByte() == '(' {
			p.pos = start
			return p.parseExpression()
		}
		return p.parseImportStatement()
	}
	if keyword, ok := p.peekKeyword("export"); ok {
		p.pos += len(keyword)
		return p.parseExportStatement()
	}
	if p.peekByte() == '{' {
		bodySource, err := p.consumeBlockSource()
		if err != nil {
			return UndefinedValue(), err
		}
		return p.evalProgramWithEnv(bodySource, p.env.clone())
	}
	return p.parseExpression()
}

func (p *classicJSStatementParser) consumeStatementLabel() (string, bool, error) {
	if !isIdentStart(p.peekByte()) {
		return "", false, nil
	}

	start := p.pos
	label, err := p.parseIdentifier()
	if err != nil {
		return "", false, err
	}
	p.skipSpaceAndComments()
	if !p.consumeByte(':') {
		p.pos = start
		return "", false, nil
	}
	p.skipSpaceAndComments()
	return label, true, nil
}

func (p *classicJSStatementParser) parseYieldStatement() (Value, error) {
	value, resumeState, yielded, err := p.parseYieldExpressionValue()
	if err != nil {
		return UndefinedValue(), err
	}
	if !yielded {
		return UndefinedValue(), nil
	}
	return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: resumeState}
}

func isClassicJSYieldExpressionTerminator(ch byte) bool {
	switch ch {
	case ')', ']', '}', ',', ';', ':':
		return true
	default:
		return false
	}
}

func (p *classicJSStatementParser) parseYieldExpressionValue() (Value, classicJSResumeState, bool, error) {
	if !p.allowYield {
		return UndefinedValue(), nil, false, NewError(ErrorKindParse, "yield statements are only supported inside bounded generator bodies")
	}

	p.skipSpaceAndComments()
	if p.consumeByte('*') {
		p.skipSpaceAndComments()
		if p.eof() || isClassicJSYieldExpressionTerminator(p.peekByte()) {
			return UndefinedValue(), nil, false, NewError(ErrorKindParse, "yield* requires an expression in this bounded classic-JS slice")
		}
		value, err := p.parseLogicalAssignment()
		if err != nil {
			return UndefinedValue(), nil, false, err
		}
		if value.kind != jsValueScalar {
			return UndefinedValue(), nil, false, NewError(ErrorKindUnsupported, "yield* expects scalar values in this bounded classic-JS slice")
		}
		yieldedValue, resumeState, yielded, err := p.startYieldDelegation(value.value)
		if err != nil {
			return UndefinedValue(), nil, false, err
		}
		return yieldedValue, resumeState, yielded, nil
	}

	if p.eof() || isClassicJSYieldExpressionTerminator(p.peekByte()) {
		return UndefinedValue(), p.resumeState, true, nil
	}

	value, err := p.parseLogicalAssignment()
	if err != nil {
		return UndefinedValue(), nil, false, err
	}
	if value.kind != jsValueScalar {
		return UndefinedValue(), nil, false, NewError(ErrorKindUnsupported, "yield statements are only supported on scalar values in this bounded classic-JS slice")
	}
	return value.value, p.resumeState, true, nil
}

func (p *classicJSStatementParser) parseReturnStatement() (Value, error) {
	if !p.allowReturn {
		return UndefinedValue(), NewError(ErrorKindParse, "return statements are only supported inside bounded function bodies")
	}

	p.skipSpaceAndComments()
	if p.eof() || p.peekByte() == ';' {
		return UndefinedValue(), classicJSReturnSignal{value: UndefinedValue()}
	}

	value, err := p.parseExpression()
	if err != nil {
		return UndefinedValue(), err
	}
	return UndefinedValue(), classicJSReturnSignal{value: value}
}

func (p *classicJSStatementParser) parseBreakStatement() (Value, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return UndefinedValue(), classicJSBreakSignal{}
	}
	if isIdentStart(p.peekByte()) {
		label, err := p.parseIdentifier()
		if err != nil {
			return UndefinedValue(), err
		}
		return UndefinedValue(), classicJSBreakSignal{label: label}
	}
	return UndefinedValue(), classicJSBreakSignal{}
}

func (p *classicJSStatementParser) parseContinueStatement() (Value, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return UndefinedValue(), classicJSContinueSignal{}
	}
	if isIdentStart(p.peekByte()) {
		label, err := p.parseIdentifier()
		if err != nil {
			return UndefinedValue(), err
		}
		return UndefinedValue(), classicJSContinueSignal{label: label}
	}
	return UndefinedValue(), classicJSContinueSignal{}
}

func (p *classicJSStatementParser) parseThrowStatement() (Value, error) {
	p.skipSpaceAndComments()
	if p.eof() || p.peekByte() == ';' || p.peekByte() == '}' {
		return UndefinedValue(), NewError(ErrorKindParse, "throw statements require an expression in this bounded classic-JS slice")
	}

	value, err := p.parseExpression()
	if err != nil {
		return UndefinedValue(), err
	}
	return UndefinedValue(), classicJSThrowSignal{value: value}
}

func (p *classicJSStatementParser) parseDebuggerStatement() (Value, error) {
	p.skipSpaceAndComments()
	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseExportStatement() (Value, error) {
	p.skipSpaceAndComments()
	if p.consumeByte('*') {
		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("as"); ok {
			p.pos += len(keyword)
			p.skipSpaceAndComments()
			alias, err := p.parseIdentifier()
			if err != nil {
				return UndefinedValue(), NewError(ErrorKindParse, "expected namespace export identifier in this bounded classic-JS slice")
			}
			p.skipSpaceAndComments()
			if keyword, ok := p.peekKeyword("from"); ok {
				p.pos += len(keyword)
				module, err := p.parseModuleNamespaceReference()
				if err != nil {
					return UndefinedValue(), err
				}
				if err := p.consumeImportAttributes(); err != nil {
					return UndefinedValue(), err
				}
				if p.moduleExports == nil {
					return UndefinedValue(), nil
				}
				p.moduleExports[alias] = module
				return UndefinedValue(), nil
			}
			return UndefinedValue(), NewError(ErrorKindParse, "expected `from` after `export * as <name>` in this bounded classic-JS slice")
		}
		if keyword, ok := p.peekKeyword("from"); ok {
			p.pos += len(keyword)
			module, err := p.parseModuleNamespaceReference()
			if err != nil {
				return UndefinedValue(), err
			}
			if err := p.consumeImportAttributes(); err != nil {
				return UndefinedValue(), err
			}
			if p.moduleExports == nil {
				return UndefinedValue(), nil
			}
			for _, entry := range module.Object {
				if entry.Key == "default" {
					continue
				}
				p.moduleExports[entry.Key] = entry.Value
			}
			return UndefinedValue(), nil
		}
		return UndefinedValue(), NewError(ErrorKindParse, "expected `from` after `export *` in this bounded classic-JS slice")
	}

	if keyword, ok := p.peekKeyword("default"); ok {
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		if _, ok := p.peekKeyword("const"); ok {
			return UndefinedValue(), NewError(ErrorKindParse, "`export default const` declarations are not supported in this bounded classic-JS slice")
		}
		if _, ok := p.peekKeyword("let"); ok {
			return UndefinedValue(), NewError(ErrorKindParse, "`export default let` declarations are not supported in this bounded classic-JS slice")
		}
		if _, ok := p.peekKeyword("class"); ok {
			p.pos += len("class")
			_, value, err := p.parseClassDeclarationWithBinding(true, false)
			if err != nil {
				return UndefinedValue(), err
			}
			if p.moduleExports != nil {
				p.moduleExports["default"] = value
			}
			return value, nil
		}
		if _, ok := p.peekKeyword("async"); ok {
			start := p.pos
			p.pos += len("async")
			p.skipSpaceAndComments()
			if _, ok := p.peekKeyword("function"); ok {
				p.pos += len("function")
				p.skipSpaceAndComments()
				generator := p.peekByte() == '*'
				return p.parseDefaultExportFunctionLiteral(true, generator)
			}
			p.pos = start
		}
		if _, ok := p.peekKeyword("function"); ok {
			p.pos += len("function")
			p.skipSpaceAndComments()
			generator := p.peekByte() == '*'
			return p.parseDefaultExportFunctionLiteral(false, generator)
		}
		value, err := p.parseExpression()
		if err != nil {
			return UndefinedValue(), err
		}
		if p.moduleExports != nil {
			p.moduleExports["default"] = value
		}
		return value, nil
	}

	if p.consumeByte('{') {
		type exportSpec struct {
			local string
			name  string
		}
		specs := make([]exportSpec, 0, 4)
		for {
			p.skipSpaceAndComments()
			if p.consumeByte('}') {
				break
			}

			local, err := p.parseIdentifier()
			if err != nil {
				return UndefinedValue(), NewError(ErrorKindParse, "expected export specifier identifier in this bounded classic-JS slice")
			}
			if local != "default" && isClassicJSReservedDeclarationName(local) {
				return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved export specifier name %q is not allowed in this bounded classic-JS slice", local))
			}
			name := local

			p.skipSpaceAndComments()
			if keyword, ok := p.peekKeyword("as"); ok {
				p.pos += len(keyword)
				p.skipSpaceAndComments()
				alias, err := p.parseIdentifier()
				if err != nil {
					return UndefinedValue(), NewError(ErrorKindParse, "expected export alias identifier in this bounded classic-JS slice")
				}
				if alias != "default" && isClassicJSReservedDeclarationName(alias) {
					return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved export alias name %q is not allowed in this bounded classic-JS slice", alias))
				}
				name = alias
			}
			specs = append(specs, exportSpec{local: local, name: name})

			p.skipSpaceAndComments()
			if p.consumeByte('}') {
				break
			}
			if !p.consumeByte(',') {
				return UndefinedValue(), NewError(ErrorKindParse, "export specifiers must be comma-separated")
			}
		}

		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("from"); ok {
			p.pos += len(keyword)
			module, err := p.parseModuleNamespaceReference()
			if err != nil {
				return UndefinedValue(), err
			}
			if err := p.consumeImportAttributes(); err != nil {
				return UndefinedValue(), err
			}
			if p.moduleExports == nil {
				return UndefinedValue(), nil
			}
			for _, spec := range specs {
				value, ok := lookupObjectProperty(module.Object, spec.local)
				if !ok {
					return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("module export %q is not available in this bounded classic-JS slice", spec.local))
				}
				p.moduleExports[spec.name] = value
			}
			return UndefinedValue(), nil
		}
		for _, spec := range specs {
			if spec.local == "default" {
				return UndefinedValue(), NewError(ErrorKindParse, "default export specifiers require a `from` clause in this bounded classic-JS slice")
			}
		}
		if p.moduleExports != nil {
			for _, spec := range specs {
				value, ok := p.env.lookup(spec.local)
				if !ok {
					return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("export binding %q is not available in this bounded classic-JS slice", spec.local))
				}
				p.moduleExports[spec.name] = value.value
			}
		}
		return UndefinedValue(), nil
	}

	if keyword, ok := p.peekKeyword("const"); ok {
		p.pos += len(keyword)
		return p.parseExportedVariableDeclaration("const")
	}
	if keyword, ok := p.peekKeyword("let"); ok {
		p.pos += len(keyword)
		return p.parseExportedVariableDeclaration("let")
	}
	if keyword, ok := p.peekKeyword("var"); ok {
		p.pos += len(keyword)
		return p.parseExportedVariableDeclaration("var")
	}
	if keyword, ok := p.peekKeyword("class"); ok {
		p.pos += len(keyword)
		return p.parseExportedClassStatement()
	}
	if keyword, ok := p.peekKeyword("async"); ok {
		start := p.pos
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		if functionKeyword, ok := p.peekKeyword("function"); ok {
			p.pos += len(functionKeyword)
			p.skipSpaceAndComments()
			generator := p.peekByte() == '*'
			if generator {
				lookahead := *p
				lookahead.pos++
				lookahead.skipSpaceAndComments()
				if !isIdentStart(lookahead.peekByte()) {
					return UndefinedValue(), NewError(ErrorKindParse, "exported async generator function declarations require an identifier in this bounded classic-JS slice")
				}
			}
			if !generator && !isIdentStart(p.peekByte()) {
				p.pos = start
				return UndefinedValue(), NewError(ErrorKindParse, "exported async function declarations require an identifier in this bounded classic-JS slice")
			}
			return p.parseExportedFunctionStatement(true, generator)
		}
		p.pos = start
	}
	if keyword, ok := p.peekKeyword("function"); ok {
		start := p.pos
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		generator := p.peekByte() == '*'
		if generator {
			lookahead := *p
			lookahead.pos++
			lookahead.skipSpaceAndComments()
			if !isIdentStart(lookahead.peekByte()) {
				return UndefinedValue(), NewError(ErrorKindParse, "exported generator function declarations require an identifier in this bounded classic-JS slice")
			}
		}
		if !generator && !isIdentStart(p.peekByte()) {
			p.pos = start
			return UndefinedValue(), NewError(ErrorKindParse, "exported function declarations require an identifier in this bounded classic-JS slice")
		}
		return p.parseExportedFunctionStatement(false, generator)
	}

	return UndefinedValue(), NewError(ErrorKindUnsupported, "export statements are only supported as bounded declaration or specifier slices in this classic-JS parser")
}

func (p *classicJSStatementParser) parseImportStatement() (Value, error) {
	p.skipSpaceAndComments()
	if p.peekByte() == '\'' || p.peekByte() == '"' {
		module, err := p.parseModuleSpecifier()
		if err != nil {
			return UndefinedValue(), err
		}
		if err := p.consumeImportAttributes(); err != nil {
			return UndefinedValue(), err
		}
		if _, err := p.lookupModuleNamespace(module); err != nil {
			return UndefinedValue(), err
		}
		return UndefinedValue(), nil
	}

	defaultName := ""
	if p.peekByte() != '{' && p.peekByte() != '*' {
		name, err := p.parseIdentifier()
		if err != nil {
			return UndefinedValue(), NewError(ErrorKindParse, "expected import declaration in this bounded classic-JS slice")
		}
		if isClassicJSReservedDeclarationName(name) {
			return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved import binding name %q is not allowed in this bounded classic-JS slice", name))
		}
		defaultName = name
		p.skipSpaceAndComments()
		if p.consumeByte(',') {
			p.skipSpaceAndComments()
		} else {
			module, err := p.parseModuleSpecifierAfterFrom()
			if err != nil {
				return UndefinedValue(), err
			}
			if err := p.consumeImportAttributes(); err != nil {
				return UndefinedValue(), err
			}
			if module == "" {
				return UndefinedValue(), NewError(ErrorKindParse, "import declarations require a module specifier in this bounded classic-JS slice")
			}
			ns, err := p.lookupModuleNamespace(module)
			if err != nil {
				return UndefinedValue(), err
			}
			if err := p.bindDefaultImport(defaultName, ns); err != nil {
				return UndefinedValue(), err
			}
			return UndefinedValue(), nil
		}
	}

	var namespaceImport string
	var namedImports []classicJSImportSpecifier
	switch p.peekByte() {
	case '*':
		p.pos++
		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("as"); !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "expected `as` after `*` in this bounded classic-JS slice")
		} else {
			p.pos += len(keyword)
		}
		p.skipSpaceAndComments()
		name, err := p.parseIdentifier()
		if err != nil {
			return UndefinedValue(), NewError(ErrorKindParse, "expected namespace import identifier in this bounded classic-JS slice")
		}
		if isClassicJSReservedDeclarationName(name) {
			return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved namespace import binding name %q is not allowed in this bounded classic-JS slice", name))
		}
		namespaceImport = name
		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("from"); !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "expected `from` after namespace import in this bounded classic-JS slice")
		} else {
			p.pos += len(keyword)
		}
		module, err := p.parseModuleSpecifier()
		if err != nil {
			return UndefinedValue(), err
		}
		if err := p.consumeImportAttributes(); err != nil {
			return UndefinedValue(), err
		}
		ns, err := p.lookupModuleNamespace(module)
		if err != nil {
			return UndefinedValue(), err
		}
		if defaultName != "" {
			if err := p.bindDefaultImport(defaultName, ns); err != nil {
				return UndefinedValue(), err
			}
		}
		if err := p.bindNamespaceImport(namespaceImport, ns); err != nil {
			return UndefinedValue(), err
		}
		return UndefinedValue(), nil
	case '{':
		specs, err := p.parseImportSpecifiers()
		if err != nil {
			return UndefinedValue(), err
		}
		namedImports = specs
		module, err := p.parseModuleSpecifierAfterFrom()
		if err != nil {
			return UndefinedValue(), err
		}
		if err := p.consumeImportAttributes(); err != nil {
			return UndefinedValue(), err
		}
		ns, err := p.lookupModuleNamespace(module)
		if err != nil {
			return UndefinedValue(), err
		}
		if defaultName != "" {
			if err := p.bindDefaultImport(defaultName, ns); err != nil {
				return UndefinedValue(), err
			}
		}
		if err := p.bindNamedImports(namedImports, ns); err != nil {
			return UndefinedValue(), err
		}
		return UndefinedValue(), nil
	default:
		if defaultName == "" {
			return UndefinedValue(), NewError(ErrorKindParse, "expected import declaration in this bounded classic-JS slice")
		}
		module, err := p.parseModuleSpecifierAfterFrom()
		if err != nil {
			return UndefinedValue(), err
		}
		ns, err := p.lookupModuleNamespace(module)
		if err != nil {
			return UndefinedValue(), err
		}
		if err := p.bindDefaultImport(defaultName, ns); err != nil {
			return UndefinedValue(), err
		}
		return UndefinedValue(), nil
	}
}

type classicJSImportSpecifier struct {
	imported string
	local    string
}

func (p *classicJSStatementParser) parseModuleSpecifier() (string, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return "", NewError(ErrorKindParse, "expected module specifier in this bounded classic-JS slice")
	}
	if p.peekByte() != '\'' && p.peekByte() != '"' {
		return "", NewError(ErrorKindParse, "module specifiers must be string literals in this bounded classic-JS slice")
	}
	value, err := p.parseStringLiteral()
	if err != nil {
		return "", err
	}
	if value.Kind != ValueKindString {
		return "", NewError(ErrorKindParse, "module specifiers must be string literals in this bounded classic-JS slice")
	}
	return value.String, nil
}

func (p *classicJSStatementParser) parseModuleSpecifierAfterFrom() (string, error) {
	p.skipSpaceAndComments()
	keyword, ok := p.peekKeyword("from")
	if !ok {
		return "", NewError(ErrorKindParse, "expected `from` in this bounded classic-JS slice")
	}
	p.pos += len(keyword)
	return p.parseModuleSpecifier()
}

func (p *classicJSStatementParser) consumeImportAttributes() error {
	p.skipSpaceAndComments()
	keyword, ok := p.peekKeyword("with")
	if !ok {
		return nil
	}
	p.pos += len(keyword)
	p.skipSpaceAndComments()
	if p.peekByte() != '{' {
		return NewError(ErrorKindParse, "expected import attributes object after `with` in this bounded classic-JS slice")
	}
	if _, err := p.consumeBlockSource(); err != nil {
		return err
	}
	return nil
}

func (p *classicJSStatementParser) parseModuleNamespaceReference() (Value, error) {
	module, err := p.parseModuleSpecifier()
	if err != nil {
		return UndefinedValue(), err
	}
	return p.lookupModuleNamespace(module)
}

func (p *classicJSStatementParser) lookupModuleNamespace(name string) (Value, error) {
	if p.env == nil {
		return UndefinedValue(), NewError(ErrorKindRuntime, "module environment is unavailable")
	}
	value, ok := p.env.lookup(name)
	if !ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("module %q is not available in this bounded classic-JS slice", name))
	}
	if value.kind != jsValueScalar || value.value.Kind != ValueKindObject {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("module %q does not resolve to a namespace object in this bounded classic-JS slice", name))
	}
	return value.value, nil
}

func (p *classicJSStatementParser) parseImportSpecifiers() ([]classicJSImportSpecifier, error) {
	if !p.consumeByte('{') {
		return nil, NewError(ErrorKindParse, "expected `{` in this bounded classic-JS slice")
	}
	specs := make([]classicJSImportSpecifier, 0, 4)
	for {
		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			break
		}

		imported, err := p.parseIdentifier()
		if err != nil {
			return nil, NewError(ErrorKindParse, "expected import specifier identifier in this bounded classic-JS slice")
		}
		if imported != "default" && isClassicJSReservedDeclarationName(imported) {
			return nil, NewError(ErrorKindParse, fmt.Sprintf("reserved import specifier name %q is not allowed in this bounded classic-JS slice", imported))
		}
		local := imported
		hasAlias := false

		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("as"); ok {
			hasAlias = true
			p.pos += len(keyword)
			p.skipSpaceAndComments()
			alias, err := p.parseIdentifier()
			if err != nil {
				return nil, NewError(ErrorKindParse, "expected import alias identifier in this bounded classic-JS slice")
			}
			if isClassicJSReservedDeclarationName(alias) {
				return nil, NewError(ErrorKindParse, fmt.Sprintf("reserved import alias name %q is not allowed in this bounded classic-JS slice", alias))
			}
			local = alias
		}
		if imported == "default" && !hasAlias {
			return nil, NewError(ErrorKindParse, "default import specifiers must use `as` in this bounded classic-JS slice")
		}

		specs = append(specs, classicJSImportSpecifier{imported: imported, local: local})

		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			break
		}
		if !p.consumeByte(',') {
			return nil, NewError(ErrorKindParse, "import specifiers must be comma-separated")
		}
	}
	return specs, nil
}

func (p *classicJSStatementParser) bindDefaultImport(localName string, moduleNamespace Value) error {
	if localName == "" {
		return nil
	}
	value, ok := lookupObjectProperty(moduleNamespace.Object, "default")
	if !ok {
		return NewError(ErrorKindUnsupported, "default import requires a `default` export in this bounded classic-JS slice")
	}
	if p.env == nil {
		p.env = newClassicJSEnvironment()
	}
	return p.env.declare(localName, scalarJSValue(value), false)
}

func (p *classicJSStatementParser) bindNamespaceImport(localName string, moduleNamespace Value) error {
	if localName == "" {
		return nil
	}
	if p.env == nil {
		p.env = newClassicJSEnvironment()
	}
	return p.env.declare(localName, scalarJSValue(moduleNamespace), false)
}

func (p *classicJSStatementParser) bindNamedImports(specs []classicJSImportSpecifier, moduleNamespace Value) error {
	for _, spec := range specs {
		value, ok := lookupObjectProperty(moduleNamespace.Object, spec.imported)
		if !ok {
			return NewError(ErrorKindUnsupported, fmt.Sprintf("module export %q is not available in this bounded classic-JS slice", spec.imported))
		}
		if p.env == nil {
			p.env = newClassicJSEnvironment()
		}
		if err := p.env.declare(spec.local, scalarJSValue(value), false); err != nil {
			return err
		}
	}
	return nil
}

func (p *classicJSStatementParser) currentEnvBindingNames() map[string]struct{} {
	out := map[string]struct{}{}
	if p == nil || p.env == nil || len(p.env.bindings) == 0 {
		return out
	}
	for name := range p.env.bindings {
		out[name] = struct{}{}
	}
	return out
}

func (p *classicJSStatementParser) recordNewModuleExports(before map[string]struct{}) {
	if p == nil || p.moduleExports == nil || p.env == nil {
		return
	}
	for name, binding := range p.env.bindings {
		if _, ok := before[name]; ok {
			continue
		}
		p.moduleExports[name] = binding.value.value
	}
}

func (p *classicJSStatementParser) parseExportedVariableDeclaration(kind string) (Value, error) {
	before := p.currentEnvBindingNames()
	value, err := p.parseVariableDeclaration(kind)
	if err != nil {
		return UndefinedValue(), err
	}
	p.recordNewModuleExports(before)
	return value, nil
}

func (p *classicJSStatementParser) parseExportedClassStatement() (Value, error) {
	before := p.currentEnvBindingNames()
	value, err := p.parseClassStatement()
	if err != nil {
		return UndefinedValue(), err
	}
	p.recordNewModuleExports(before)
	return value, nil
}

func (p *classicJSStatementParser) parseFunctionStatement(async bool, generator bool) (Value, error) {
	name, value, err := p.parseFunctionLiteral(false, async, generator)
	if err != nil {
		return UndefinedValue(), err
	}
	if p.env == nil {
		p.env = newClassicJSEnvironment()
	}
	if err := p.env.declare(name, scalarJSValue(value), false); err != nil {
		return UndefinedValue(), err
	}
	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseExportedFunctionStatement(async bool, generator bool) (Value, error) {
	before := p.currentEnvBindingNames()
	value, err := p.parseFunctionStatement(async, generator)
	if err != nil {
		return UndefinedValue(), err
	}
	p.recordNewModuleExports(before)
	return value, nil
}

func (p *classicJSStatementParser) parseDefaultExportFunctionLiteral(async bool, generator bool) (Value, error) {
	name, value, err := p.parseFunctionLiteral(true, async, generator)
	if err != nil {
		return UndefinedValue(), err
	}
	if name != "" {
		if p.env == nil {
			p.env = newClassicJSEnvironment()
		}
		if err := p.env.declare(name, scalarJSValue(value), false); err != nil {
			return UndefinedValue(), err
		}
	}
	if p.moduleExports != nil {
		p.moduleExports["default"] = value
	}
	return value, nil
}

func (p *classicJSStatementParser) parseFunctionLiteral(allowAnonymous bool, async bool, generator bool) (string, Value, error) {
	p.skipSpaceAndComments()
	if generator {
		if !p.consumeByte('*') {
			return "", UndefinedValue(), NewError(ErrorKindParse, "expected `*` after `function` in this bounded classic-JS slice")
		}
		p.skipSpaceAndComments()
	} else if p.consumeByte('*') {
		return "", UndefinedValue(), NewError(ErrorKindParse, "generator function declarations require a generator-aware parse entrypoint in this bounded classic-JS slice")
	}

	name := ""
	if isIdentStart(p.peekByte()) {
		parsedName, err := p.parseIdentifier()
		if err != nil {
			return "", UndefinedValue(), err
		}
		if isClassicJSReservedDeclarationName(parsedName) {
			return "", UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved function name %q is not allowed in this bounded classic-JS slice", parsedName))
		}
		name = parsedName
		p.skipSpaceAndComments()
	} else if !allowAnonymous {
		return "", UndefinedValue(), NewError(ErrorKindParse, "function declarations require an identifier in this bounded classic-JS slice")
	}

	if p.peekByte() != '(' {
		return "", UndefinedValue(), NewError(ErrorKindParse, "expected `(` after function name")
	}

	paramsSource, err := p.consumeParenthesizedSource("function")
	if err != nil {
		return "", UndefinedValue(), err
	}
	params, restName, err := parseClassicJSFunctionParameters(paramsSource, "function")
	if err != nil {
		return "", UndefinedValue(), err
	}
	body, err := p.consumeBlockSource()
	if err != nil {
		return "", UndefinedValue(), err
	}

	fn := &classicJSArrowFunction{
		name:               name,
		params:             params,
		restName:           restName,
		allowReturn:        true,
		body:               body,
		bodyIsBlock:        true,
		async:              async,
		constructible:      !async && !generator,
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
	}
	if fn.constructible {
		classicJSConstructibleFunctionMarker(fn)
	}
	if generator {
		fn.generatorFunction = &classicJSGeneratorFunction{
			name:               name,
			params:             params,
			restName:           restName,
			body:               body,
			async:              async,
			env:                p.env,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		}
	}
	return name, FunctionValue(fn), nil
}

func (p *classicJSStatementParser) parseVariableDeclaration(kind string) (Value, error) {
	return p.parseVariableDeclarationWithLabel(kind, kind)
}

func (p *classicJSStatementParser) parseUsingDeclaration(label string) (Value, error) {
	return p.parseVariableDeclarationWithLabel("const", label)
}

func (p *classicJSStatementParser) parseVariableDeclarationWithLabel(kind string, label string) (Value, error) {
	if p.env == nil {
		p.env = newClassicJSEnvironment()
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() {
			return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
		}

		if p.peekByte() == '[' || p.peekByte() == '{' {
			pattern, err := p.parseBindingPattern()
			if err != nil {
				return UndefinedValue(), err
			}

			p.skipSpaceAndComments()
			if !p.consumeByte('=') {
				return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("%s declarations require an initializer in this bounded classic-JS slice", label))
			}
			p.skipSpaceAndComments()
			if p.eof() {
				return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
			}
			value, err := p.parseScalarExpression()
			if err != nil {
				if yieldedValue, _, ok := classicJSYieldSignalDetails(err); ok {
					p.skipSpaceAndComments()
					if p.peekByte() == ',' {
						return UndefinedValue(), NewError(ErrorKindUnsupported, "yield send values are only supported in single declaration initializers in this bounded classic-JS slice")
					}
					return UndefinedValue(), classicJSYieldSignal{
						value: yieldedValue,
						resumeState: &classicJSYieldDeclarationState{
							env:                p.env,
							kind:               kind,
							pattern:            pattern,
							hasPattern:         true,
							privateClass:       p.privateClass,
							privateFieldPrefix: p.privateFieldPrefix,
						},
					}
				}
				return UndefinedValue(), err
			}
			if err := p.declareBindingPattern(pattern, value, kind); err != nil {
				return UndefinedValue(), err
			}
		} else {
			name, err := p.parseIdentifier()
			if err != nil {
				return UndefinedValue(), err
			}
			if isClassicJSReservedDeclarationName(name) {
				return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved %s binding name %q is not allowed in this bounded classic-JS slice", label, name))
			}

			p.skipSpaceAndComments()
			value := UndefinedValue()
			if p.consumeByte('=') {
				p.skipSpaceAndComments()
				if p.eof() {
					return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
				}
				parsed, err := p.parseScalarExpression()
				if err != nil {
					if yieldedValue, _, ok := classicJSYieldSignalDetails(err); ok {
						p.skipSpaceAndComments()
						if p.peekByte() == ',' {
							return UndefinedValue(), NewError(ErrorKindUnsupported, "yield send values are only supported in single declaration initializers in this bounded classic-JS slice")
						}
						return UndefinedValue(), classicJSYieldSignal{
							value: yieldedValue,
							resumeState: &classicJSYieldDeclarationState{
								env:                p.env,
								kind:               kind,
								name:               name,
								privateClass:       p.privateClass,
								privateFieldPrefix: p.privateFieldPrefix,
							},
						}
					}
					return UndefinedValue(), err
				}
				value = parsed
			} else if kind == "const" {
				return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("%s declarations require an initializer in this bounded classic-JS slice", label))
			}

			if kind == "var" {
				p.env.bindings[name] = classicJSBinding{value: scalarJSValue(value), mutable: true}
			} else {
				if err := p.env.declare(name, scalarJSValue(value), kind == "let"); err != nil {
					return UndefinedValue(), err
				}
			}
		}

		p.skipSpaceAndComments()
		if !p.consumeByte(',') {
			break
		}
	}

	return UndefinedValue(), nil
}

type classicJSBindingPatternKind int

const (
	classicJSBindingPatternIdentifier classicJSBindingPatternKind = iota
	classicJSBindingPatternArray
	classicJSBindingPatternObject
	classicJSBindingPatternHole
	classicJSBindingPatternRest
)

type classicJSBindingPattern struct {
	kind          classicJSBindingPatternKind
	name          string
	defaultSource string
	elements      []classicJSBindingPattern
	properties    []classicJSObjectBindingProperty
}

type classicJSObjectBindingProperty struct {
	key      string
	computed bool
	pattern  classicJSBindingPattern
}

type classicJSObjectKey struct {
	name       string
	identifier bool
}

func (p *classicJSStatementParser) parseBindingPattern() (classicJSBindingPattern, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
	}
	if p.pos+3 <= len(p.source) && p.source[p.pos:p.pos+3] == "..." {
		return classicJSBindingPattern{}, NewError(ErrorKindParse, "rest binding syntax must appear directly inside array or object binding patterns")
	}

	switch p.peekByte() {
	case '[':
		return p.parseArrayBindingPattern()
	case '{':
		return p.parseObjectBindingPattern()
	default:
		name, err := p.parseIdentifier()
		if err != nil {
			return classicJSBindingPattern{}, err
		}
		if isClassicJSReservedDeclarationName(name) {
			return classicJSBindingPattern{}, NewError(ErrorKindParse, fmt.Sprintf("reserved lexical binding name %q is not allowed in this bounded classic-JS slice", name))
		}
		return classicJSBindingPattern{kind: classicJSBindingPatternIdentifier, name: name}, nil
	}
}

func (p *classicJSStatementParser) parseArrayBindingPattern() (classicJSBindingPattern, error) {
	if p.eof() {
		return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
	}

	p.pos++
	elements := make([]classicJSBindingPattern, 0, 4)
	for {
		p.skipSpaceAndComments()
		if p.consumeByte(']') {
			return classicJSBindingPattern{kind: classicJSBindingPatternArray, elements: elements}, nil
		}
		if p.consumeByte(',') {
			elements = append(elements, classicJSBindingPattern{kind: classicJSBindingPatternHole})
			continue
		}
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			name, err := p.parseIdentifier()
			if err != nil {
				return classicJSBindingPattern{}, err
			}
			if isClassicJSReservedDeclarationName(name) {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, fmt.Sprintf("reserved lexical binding name %q is not allowed in this bounded classic-JS slice", name))
			}
			elements = append(elements, classicJSBindingPattern{kind: classicJSBindingPatternRest, name: name})
			p.skipSpaceAndComments()
			if p.consumeByte(',') {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "array rest elements must be the final element in this bounded classic-JS slice")
			}
			if !p.consumeByte(']') {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "array rest elements must be the final element in this bounded classic-JS slice")
			}
			return classicJSBindingPattern{kind: classicJSBindingPatternArray, elements: elements}, nil
		}

		element, err := p.parseBindingPattern()
		if err != nil {
			return classicJSBindingPattern{}, err
		}
		elements = append(elements, element)

		p.skipSpaceAndComments()
		if p.consumeByte('=') {
			p.skipSpaceAndComments()
			if p.eof() {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
			}
			defaultStart := p.pos
			defaultEnd, err := scanClassicJSBindingPatternDefaultTerminator(p, ']')
			if err != nil {
				return classicJSBindingPattern{}, err
			}
			defaultSource := strings.TrimSpace(p.source[defaultStart:defaultEnd])
			if defaultSource == "" {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "binding pattern default requires an expression")
			}
			element.defaultSource = defaultSource
			elements[len(elements)-1] = element
			p.pos = defaultEnd
		}
		p.skipSpaceAndComments()
		if p.consumeByte(']') {
			return classicJSBindingPattern{kind: classicJSBindingPatternArray, elements: elements}, nil
		}
		if !p.consumeByte(',') {
			return classicJSBindingPattern{}, NewError(ErrorKindParse, "array binding patterns must separate elements with commas")
		}
	}
}

func (p *classicJSStatementParser) parseObjectBindingPattern() (classicJSBindingPattern, error) {
	if p.eof() {
		return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
	}

	p.pos++
	properties := make([]classicJSObjectBindingProperty, 0, 4)
	for {
		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return classicJSBindingPattern{kind: classicJSBindingPatternObject, properties: properties}, nil
		}
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			name, err := p.parseIdentifier()
			if err != nil {
				return classicJSBindingPattern{}, err
			}
			if isClassicJSReservedDeclarationName(name) {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, fmt.Sprintf("reserved lexical binding name %q is not allowed in this bounded classic-JS slice", name))
			}
			properties = append(properties, classicJSObjectBindingProperty{pattern: classicJSBindingPattern{kind: classicJSBindingPatternRest, name: name}})
			p.skipSpaceAndComments()
			if p.consumeByte(',') {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "object rest properties must be the final property in this bounded classic-JS slice")
			}
			if !p.consumeByte('}') {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "object rest properties must be the final property in this bounded classic-JS slice")
			}
			return classicJSBindingPattern{kind: classicJSBindingPatternObject, properties: properties}, nil
		}

		key, computed, err := p.parseObjectBindingKey()
		if err != nil {
			return classicJSBindingPattern{}, err
		}
		pattern := classicJSBindingPattern{}

		p.skipSpaceAndComments()
		if p.consumeByte(':') {
			p.skipSpaceAndComments()
			if p.eof() {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
			}
			bindingPattern, err := p.parseBindingPattern()
			if err != nil {
				return classicJSBindingPattern{}, err
			}
			pattern = bindingPattern
		} else {
			if !key.identifier {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "object binding shorthand requires an identifier name")
			}
			if isClassicJSReservedDeclarationName(key.name) {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, fmt.Sprintf("reserved lexical binding name %q is not allowed in this bounded classic-JS slice", key.name))
			}
			pattern = classicJSBindingPattern{kind: classicJSBindingPatternIdentifier, name: key.name}
		}

		p.skipSpaceAndComments()
		if p.consumeByte('=') {
			p.skipSpaceAndComments()
			if p.eof() {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
			}
			defaultStart := p.pos
			defaultEnd, err := scanClassicJSBindingPatternDefaultTerminator(p, '}')
			if err != nil {
				return classicJSBindingPattern{}, err
			}
			defaultSource := strings.TrimSpace(p.source[defaultStart:defaultEnd])
			if defaultSource == "" {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "binding pattern default requires an expression")
			}
			pattern.defaultSource = defaultSource
			p.pos = defaultEnd
		}
		properties = append(properties, classicJSObjectBindingProperty{key: key.name, computed: computed, pattern: pattern})
		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return classicJSBindingPattern{kind: classicJSBindingPatternObject, properties: properties}, nil
		}
		if !p.consumeByte(',') {
			return classicJSBindingPattern{}, NewError(ErrorKindParse, "object binding patterns must separate properties with commas")
		}
	}
}

func (p *classicJSStatementParser) parseObjectBindingKey() (classicJSObjectKey, bool, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return classicJSObjectKey{}, false, NewError(ErrorKindParse, "unexpected end of script source")
	}

	if p.consumeByte('[') {
		keySource, err := p.consumeBracketAccessExpressionSource()
		if err != nil {
			return classicJSObjectKey{}, false, err
		}
		if strings.TrimSpace(keySource) == "" {
			return classicJSObjectKey{}, false, NewError(ErrorKindParse, "computed object binding key requires an expression")
		}
		value, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(keySource, p.host, p.env, p.stepLimit, p.allowAwait, p.allowYield, p.privateClass)
		if err != nil {
			return classicJSObjectKey{}, false, err
		}
		return classicJSObjectKey{name: ToJSString(value), identifier: false}, true, nil
	}

	switch ch := p.peekByte(); ch {
	case '\'', '"':
		value, err := p.parseStringLiteral()
		if err != nil {
			return classicJSObjectKey{}, false, err
		}
		return classicJSObjectKey{name: value.String, identifier: false}, false, nil
	default:
		if isDigit(ch) {
			value, err := p.parseNumberLiteral()
			if err != nil {
				return classicJSObjectKey{}, false, err
			}
			return classicJSObjectKey{name: ToJSString(value), identifier: false}, false, nil
		}
		ident, err := p.parseIdentifier()
		if err != nil {
			return classicJSObjectKey{}, false, err
		}
		return classicJSObjectKey{name: ident, identifier: true}, false, nil
	}
}

func (p *classicJSStatementParser) declareBindingPattern(pattern classicJSBindingPattern, value Value, kind string) error {
	if err := p.bindBindingPattern(pattern, value, kind); err != nil {
		return err
	}
	return nil
}

func (p *classicJSStatementParser) bindBindingPattern(pattern classicJSBindingPattern, value Value, kind string) error {
	mutable := kind != "const" && kind != "using" && kind != "await using"
	switch pattern.kind {
	case classicJSBindingPatternIdentifier:
		if p.env == nil {
			return NewError(ErrorKindRuntime, "classic-JS environment is unavailable")
		}
		if err := p.env.declare(pattern.name, scalarJSValue(UndefinedValue()), mutable); err != nil {
			return err
		}
		if pattern.defaultSource != "" && value.Kind == ValueKindUndefined {
			parsed, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(pattern.defaultSource, p.host, p.env, p.stepLimit, p.allowAwait, p.allowYield, p.privateClass)
			if err != nil {
				return err
			}
			value = parsed
		}
		p.env.bindings[pattern.name] = classicJSBinding{value: scalarJSValue(value), mutable: mutable}
		return nil
	case classicJSBindingPatternHole:
		return nil
	case classicJSBindingPatternRest:
		return NewError(ErrorKindParse, "rest binding syntax must appear directly inside array or object binding patterns")
	case classicJSBindingPatternArray:
		if pattern.defaultSource != "" && value.Kind == ValueKindUndefined {
			parsed, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(pattern.defaultSource, p.host, p.env, p.stepLimit, p.allowAwait, p.allowYield, p.privateClass)
			if err != nil {
				return err
			}
			value = parsed
		}
		values, err := p.collectClassicJSArrayLikeValues(value, "array destructuring")
		if err != nil {
			return err
		}
		sourceIndex := 0
		for i, element := range pattern.elements {
			switch element.kind {
			case classicJSBindingPatternHole:
				sourceIndex++
			case classicJSBindingPatternRest:
				if i != len(pattern.elements)-1 {
					return NewError(ErrorKindParse, "array rest elements must be the final element in this bounded classic-JS slice")
				}
				restElements := []Value(nil)
				if sourceIndex < len(values) {
					restElements = values[sourceIndex:]
				}
				if err := p.bindBindingPattern(classicJSBindingPattern{kind: classicJSBindingPatternIdentifier, name: element.name}, ArrayValue(restElements), kind); err != nil {
					return err
				}
				sourceIndex = len(values)
			default:
				elementValue := UndefinedValue()
				if sourceIndex < len(values) {
					elementValue = values[sourceIndex]
				}
				if err := p.bindBindingPattern(element, elementValue, kind); err != nil {
					return err
				}
				sourceIndex++
			}
			if element.kind == classicJSBindingPatternRest {
				break
			}
		}
		return nil
	case classicJSBindingPatternObject:
		if pattern.defaultSource != "" && value.Kind == ValueKindUndefined {
			parsed, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(pattern.defaultSource, p.host, p.env, p.stepLimit, p.allowAwait, p.allowYield, p.privateClass)
			if err != nil {
				return err
			}
			value = parsed
		}
		if value.Kind != ValueKindObject {
			return NewError(ErrorKindUnsupported, "object destructuring only works on object values in this bounded classic-JS slice")
		}
		excluded := make(map[string]struct{}, len(pattern.properties))
		for i, property := range pattern.properties {
			if property.pattern.kind == classicJSBindingPatternRest {
				if i != len(pattern.properties)-1 {
					return NewError(ErrorKindParse, "object rest properties must be the final property in this bounded classic-JS slice")
				}
				restEntries := make([]ObjectEntry, 0, len(value.Object))
				for _, entry := range value.Object {
					if _, ok := excluded[entry.Key]; ok {
						continue
					}
					restEntries = append(restEntries, entry)
				}
				if err := p.bindBindingPattern(classicJSBindingPattern{kind: classicJSBindingPatternIdentifier, name: property.pattern.name}, ObjectValue(restEntries), kind); err != nil {
					return err
				}
				break
			}

			propertyValue := UndefinedValue()
			for i := len(value.Object) - 1; i >= 0; i-- {
				entry := value.Object[i]
				if entry.Key == property.key {
					propertyValue = entry.Value
					break
				}
			}
			if err := p.bindBindingPattern(property.pattern, propertyValue, kind); err != nil {
				return err
			}
			excluded[property.key] = struct{}{}
		}
		return nil
	default:
		return NewError(ErrorKindParse, "unsupported binding pattern in this bounded classic-JS slice")
	}
}

type classicJSBindingAssignment struct {
	name    string
	value   Value
	mutable bool
}

func (p *classicJSStatementParser) collectBindingAssignments(pattern classicJSBindingPattern, value Value, mutable bool, assignments *[]classicJSBindingAssignment) error {
	switch pattern.kind {
	case classicJSBindingPatternIdentifier:
		*assignments = append(*assignments, classicJSBindingAssignment{name: pattern.name, value: value, mutable: mutable})
		return nil
	case classicJSBindingPatternHole:
		return nil
	case classicJSBindingPatternRest:
		return NewError(ErrorKindParse, "rest binding syntax must appear directly inside array or object binding patterns")
	case classicJSBindingPatternArray:
		return p.collectArrayBindingAssignments(pattern.elements, value, mutable, assignments)
	case classicJSBindingPatternObject:
		return p.collectObjectBindingAssignments(pattern.properties, value, mutable, assignments)
	default:
		return NewError(ErrorKindParse, "unsupported binding pattern in this bounded classic-JS slice")
	}
}

func (p *classicJSStatementParser) collectArrayBindingAssignments(elements []classicJSBindingPattern, value Value, mutable bool, assignments *[]classicJSBindingAssignment) error {
	values, err := p.collectClassicJSArrayLikeValues(value, "array destructuring")
	if err != nil {
		return err
	}

	sourceIndex := 0
	for i, element := range elements {
		switch element.kind {
		case classicJSBindingPatternHole:
			sourceIndex++
		case classicJSBindingPatternRest:
			if i != len(elements)-1 {
				return NewError(ErrorKindParse, "array rest elements must be the final element in this bounded classic-JS slice")
			}
			restElements := []Value(nil)
			if sourceIndex < len(values) {
				restElements = values[sourceIndex:]
			}
			*assignments = append(*assignments, classicJSBindingAssignment{name: element.name, value: ArrayValue(restElements), mutable: mutable})
			sourceIndex = len(values)
		default:
			elementValue := UndefinedValue()
			if sourceIndex < len(values) {
				elementValue = values[sourceIndex]
			}
			if err := p.collectBindingAssignments(element, elementValue, mutable, assignments); err != nil {
				return err
			}
			sourceIndex++
		}
		if element.kind == classicJSBindingPatternRest {
			break
		}
	}
	return nil
}

func (p *classicJSStatementParser) collectClassicJSArrayLikeValues(value Value, context string) ([]Value, error) {
	switch value.Kind {
	case ValueKindArray:
		return append([]Value(nil), value.Array...), nil
	case ValueKindString:
		values := make([]Value, 0, len(value.String))
		for _, r := range value.String {
			values = append(values, StringValue(string(r)))
		}
		return values, nil
	case ValueKindObject:
		nextValue, err := p.resolveMemberAccess(scalarJSValue(value), "next")
		if err != nil {
			return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("%s only works on string, array, or iterator-like object values in this bounded classic-JS slice", context))
		}
		if nextValue.kind != jsValueScalar || !classicJSIsCallableFunctionValue(nextValue.value) {
			return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("%s only works on string, array, or iterator-like object values in this bounded classic-JS slice", context))
		}
		values := make([]Value, 0, len(value.Object))
		for {
			result, err := p.invoke(nextValue, nil)
			if err != nil {
				return nil, err
			}
			if result.kind != jsValueScalar {
				return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("%s iterator must return an object in this bounded classic-JS slice", context))
			}
			resultValue := unwrapPromiseValue(result.value)
			if resultValue.Kind != ValueKindObject {
				return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("%s iterator must return an object in this bounded classic-JS slice", context))
			}
			doneValue, ok := lookupObjectProperty(resultValue.Object, "done")
			if !ok || doneValue.Kind != ValueKindBool {
				return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("%s iterator result must include a boolean `done` property in this bounded classic-JS slice", context))
			}
			if doneValue.Bool {
				break
			}
			itemValue, ok := lookupObjectProperty(resultValue.Object, "value")
			if !ok {
				itemValue = UndefinedValue()
			}
			values = append(values, itemValue)
		}
		return values, nil
	default:
		return nil, NewError(ErrorKindRuntime, fmt.Sprintf("%s requires a string, array, or iterator-like object value in this bounded classic-JS slice", context))
	}
}

func classicJSIsCallableFunctionValue(value Value) bool {
	if value.Kind != ValueKindFunction {
		return false
	}
	return value.Function != nil || value.NativeFunction != nil
}

func (p *classicJSStatementParser) collectObjectBindingAssignments(properties []classicJSObjectBindingProperty, value Value, mutable bool, assignments *[]classicJSBindingAssignment) error {
	if value.Kind != ValueKindObject {
		return NewError(ErrorKindUnsupported, "object destructuring only works on object values in this bounded classic-JS slice")
	}

	excluded := make(map[string]struct{}, len(properties))
	for i, property := range properties {
		if property.pattern.kind == classicJSBindingPatternRest {
			if i != len(properties)-1 {
				return NewError(ErrorKindParse, "object rest properties must be the final property in this bounded classic-JS slice")
			}
			restEntries := make([]ObjectEntry, 0, len(value.Object))
			for _, entry := range value.Object {
				if _, ok := excluded[entry.Key]; ok {
					continue
				}
				restEntries = append(restEntries, entry)
			}
			*assignments = append(*assignments, classicJSBindingAssignment{name: property.pattern.name, value: ObjectValue(restEntries), mutable: mutable})
			break
		}

		propertyValue := UndefinedValue()
		for i := len(value.Object) - 1; i >= 0; i-- {
			entry := value.Object[i]
			if entry.Key == property.key {
				propertyValue = entry.Value
				break
			}
		}
		if err := p.collectBindingAssignments(property.pattern, propertyValue, mutable, assignments); err != nil {
			return err
		}
		excluded[property.key] = struct{}{}
	}
	return nil
}

func (p *classicJSStatementParser) tryParseArrowFunction() (jsValue, bool, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return jsValue{}, false, nil
	}

	start := p.pos
	if keyword, ok := p.peekKeyword("async"); ok {
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		value, parsed, err := p.tryParseArrowFunctionAtPos(start, true)
		if err != nil || parsed {
			return value, parsed, err
		}
		p.pos = start
		return jsValue{}, false, nil
	}

	value, parsed, err := p.tryParseArrowFunctionAtPos(start, false)
	if err != nil || parsed {
		return value, parsed, err
	}
	p.pos = start
	return jsValue{}, false, nil
}

func (p *classicJSStatementParser) tryParseGeneratorFunction() (jsValue, bool, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return jsValue{}, false, nil
	}

	start := p.pos
	async := false
	if keyword, ok := p.peekKeyword("async"); ok {
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		async = true
	}
	keyword, ok := p.peekKeyword("function")
	if !ok {
		if async {
			p.pos = start
		}
		return jsValue{}, false, nil
	}

	p.pos += len(keyword)
	p.skipSpaceAndComments()
	if !p.consumeByte('*') {
		p.pos = start
		return jsValue{}, false, nil
	}

	name := ""
	p.skipSpaceAndComments()
	if isIdentStart(p.peekByte()) {
		parsedName, err := p.parseIdentifier()
		if err != nil {
			p.pos = start
			return jsValue{}, false, nil
		}
		if isClassicJSReservedDeclarationName(parsedName) {
			return jsValue{}, false, NewError(ErrorKindParse, fmt.Sprintf("reserved generator function name %q is not allowed in this bounded classic-JS slice", parsedName))
		}
		name = parsedName
		p.skipSpaceAndComments()
	}

	if p.peekByte() != '(' {
		p.pos = start
		if name != "" {
			return jsValue{}, false, NewError(ErrorKindParse, "expected `(` after generator function name")
		}
		return jsValue{}, false, nil
	}

	paramsSource, err := p.consumeParenthesizedSource("generator function")
	if err != nil {
		p.pos = start
		return jsValue{}, false, err
	}
	params, restName, err := parseClassicJSFunctionParameters(paramsSource, "generator function")
	if err != nil {
		p.pos = start
		return jsValue{}, false, err
	}

	p.skipSpaceAndComments()
	if p.peekByte() != '{' {
		p.pos = start
		return jsValue{}, false, nil
	}
	body, err := p.consumeBlockSource()
	if err != nil {
		return jsValue{}, false, err
	}

	fn := &classicJSArrowFunction{
		params:             params,
		restName:           restName,
		allowReturn:        true,
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
		generatorFunction: &classicJSGeneratorFunction{
			name:               name,
			params:             params,
			restName:           restName,
			body:               body,
			async:              async,
			env:                p.env,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		},
	}
	return scalarJSValue(FunctionValue(fn)), true, nil
}

func (p *classicJSStatementParser) tryParseFunctionExpression() (jsValue, bool, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return jsValue{}, false, nil
	}

	start := p.pos
	async := false
	if keyword, ok := p.peekKeyword("async"); ok {
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		async = true
	}
	keyword, ok := p.peekKeyword("function")
	if !ok {
		if async {
			p.pos = start
		}
		return jsValue{}, false, nil
	}

	p.pos += len(keyword)
	p.skipSpaceAndComments()
	generator := false
	if p.peekByte() == '*' {
		generator = true
		p.pos++
		p.skipSpaceAndComments()
	}

	name := ""
	if isIdentStart(p.peekByte()) {
		parsedName, err := p.parseIdentifier()
		if err != nil {
			p.pos = start
			return jsValue{}, false, nil
		}
		if isClassicJSReservedDeclarationName(parsedName) {
			return jsValue{}, false, NewError(ErrorKindParse, fmt.Sprintf("reserved function name %q is not allowed in this bounded classic-JS slice", parsedName))
		}
		name = parsedName
		p.skipSpaceAndComments()
	} else if p.peekByte() != '(' {
		p.pos = start
		return jsValue{}, false, nil
	}

	paramsSource, err := p.consumeParenthesizedSource("function")
	if err != nil {
		return jsValue{}, false, err
	}
	params, restName, err := parseClassicJSFunctionParameters(paramsSource, "function")
	if err != nil {
		return jsValue{}, false, err
	}
	body, err := p.consumeBlockSource()
	if err != nil {
		return jsValue{}, false, err
	}

	fn := &classicJSArrowFunction{
		name:               name,
		params:             params,
		restName:           restName,
		body:               body,
		bodyIsBlock:        true,
		async:              async,
		allowReturn:        true,
		constructible:      !async && !generator,
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
	}
	if fn.constructible {
		classicJSConstructibleFunctionMarker(fn)
	}
	if generator {
		fn.generatorFunction = &classicJSGeneratorFunction{
			name:               name,
			params:             params,
			restName:           restName,
			body:               body,
			async:              async,
			env:                p.env,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		}
	}
	return scalarJSValue(FunctionValue(fn)), true, nil
}

func (p *classicJSStatementParser) tryParseArrowFunctionAtPos(start int, async bool) (jsValue, bool, error) {
	switch p.peekByte() {
	case '(':
		paramsSource, err := p.consumeParenthesizedSource("arrow function")
		if err != nil {
			p.pos = start
			return jsValue{}, false, err
		}
		p.skipSpaceAndComments()
		if !p.consumeByte('=') || !p.consumeByte('>') {
			p.pos = start
			return jsValue{}, false, nil
		}

		params, restName, err := parseClassicJSFunctionParameters(paramsSource, "arrow function")
		if err != nil {
			p.pos = start
			return jsValue{}, false, err
		}

		body, bodyIsBlock, err := p.consumeArrowFunctionBody()
		if err != nil {
			return jsValue{}, false, err
		}

		fn := &classicJSArrowFunction{
			params:             params,
			restName:           restName,
			body:               body,
			bodyIsBlock:        bodyIsBlock,
			async:              async,
			allowReturn:        true,
			isArrow:            true,
			newTarget:          p.newTarget,
			hasNewTarget:       p.hasNewTarget,
			env:                p.env,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		}
		return scalarJSValue(FunctionValue(fn)), true, nil

	default:
		if !isIdentStart(p.peekByte()) {
			p.pos = start
			return jsValue{}, false, nil
		}
		name, err := p.parseIdentifier()
		if err != nil {
			p.pos = start
			return jsValue{}, false, nil
		}
		p.skipSpaceAndComments()
		if !p.consumeByte('=') || !p.consumeByte('>') {
			p.pos = start
			return jsValue{}, false, nil
		}

		body, bodyIsBlock, err := p.consumeArrowFunctionBody()
		if err != nil {
			return jsValue{}, false, err
		}

		fn := &classicJSArrowFunction{
			params:             []classicJSFunctionParameter{{name: name}},
			body:               body,
			bodyIsBlock:        bodyIsBlock,
			async:              async,
			allowReturn:        true,
			isArrow:            true,
			newTarget:          p.newTarget,
			hasNewTarget:       p.hasNewTarget,
			env:                p.env,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		}
		return scalarJSValue(FunctionValue(fn)), true, nil
	}
}

func (p *classicJSStatementParser) consumeArrowFunctionBody() (string, bool, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return "", false, NewError(ErrorKindParse, "arrow function body requires an expression")
	}

	if p.peekByte() == '{' {
		block, err := p.consumeBlockSource()
		if err != nil {
			return "", false, err
		}
		return block, true, nil
	}

	body, err := p.consumeArrowFunctionExpressionSource()
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(body) == "" {
		return "", false, NewError(ErrorKindParse, "arrow function body requires an expression")
	}
	return body, false, nil
}

func (p *classicJSStatementParser) parseIfStatement() (Value, error) {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return UndefinedValue(), NewError(ErrorKindParse, "expected `(` after `if`")
	}

	condition, err := p.parseExpression()
	if err != nil {
		return UndefinedValue(), err
	}
	p.skipSpaceAndComments()
	if !p.consumeByte(')') {
		return UndefinedValue(), NewError(ErrorKindParse, "unterminated `if` condition")
	}

	consequent, err := p.consumeIfBodySource()
	if err != nil {
		return UndefinedValue(), err
	}

	var elseSource string
	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("else"); ok {
		p.pos += len(keyword)
		elseSource, err = p.consumeIfBodySource()
		if err != nil {
			return UndefinedValue(), err
		}
	}

	if !jsTruthy(condition) {
		if elseSource == "" {
			return UndefinedValue(), nil
		}
		return p.evalProgramWithEnv(elseSource, p.env.clone())
	}

	return p.evalProgramWithEnv(consequent, p.env.clone())
}

func (p *classicJSStatementParser) consumeIfBodySource() (string, error) {
	p.skipSpaceAndComments()
	if p.peekByte() == '{' {
		return p.consumeBlockSource()
	}
	return p.consumeStatementSource()
}

func (p *classicJSStatementParser) parseWithStatement() (Value, error) {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return UndefinedValue(), NewError(ErrorKindParse, "expected `(` after `with`")
	}

	scopeValue, err := p.parseScalarExpression()
	if err != nil {
		return UndefinedValue(), err
	}
	p.skipSpaceAndComments()
	if !p.consumeByte(')') {
		return UndefinedValue(), NewError(ErrorKindParse, "unterminated `with` expression")
	}

	bodySource, err := p.consumeIfBodySource()
	if err != nil {
		return UndefinedValue(), err
	}

	if scopeValue.Kind != ValueKindObject && scopeValue.Kind != ValueKindArray {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "with statements require object or array values in this bounded classic-JS slice")
	}

	withEnv := p.env.withScope(scopeValue)
	return p.evalProgramWithEnv(bodySource, withEnv)
}

func (p *classicJSStatementParser) consumeLoopBodySource() (string, error) {
	p.skipSpaceAndComments()
	if p.peekByte() == '{' {
		return p.consumeBlockSource()
	}
	return p.consumeStatementSource()
}

func (p *classicJSStatementParser) consumeStatementSource() (string, error) {
	p.skipSpaceAndComments()
	start := p.pos
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int
	for !p.eof() {
		ch := p.peekByte()
		if lineComment {
			p.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
				p.pos += 2
				blockComment = false
				continue
			}
			p.pos++
			continue
		}
		if quote != 0 {
			p.pos++
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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && ch == ';' {
			source := strings.TrimSpace(p.source[start:p.pos])
			p.pos++
			return source, nil
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				p.pos++
				continue
			}
			switch p.source[p.pos+1] {
			case '/':
				lineComment = true
				p.pos += 2
			case '*':
				blockComment = true
				p.pos += 2
			default:
				p.pos++
			}
		case '(':
			parenDepth++
			p.pos++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			p.pos++
		case '{':
			braceDepth++
			p.pos++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			p.pos++
		case '[':
			bracketDepth++
			p.pos++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			p.pos++
		default:
			p.pos++
		}
	}

	if quote != 0 {
		return "", NewError(ErrorKindParse, "unterminated quoted string in statement body")
	}
	if blockComment {
		return "", NewError(ErrorKindParse, "unterminated block comment in statement body")
	}
	return strings.TrimSpace(p.source[start:p.pos]), nil
}

func (p *classicJSStatementParser) newClassicJSLoopState(kind classicJSLoopKind, conditionSource, initSource, updateSource, bodySource string) (*classicJSLoopState, error) {
	statements, err := splitScriptStatements(bodySource)
	if err != nil {
		return nil, NewError(ErrorKindParse, err.Error())
	}
	return &classicJSLoopState{
		kind:            kind,
		loopEnv:         p.env.clone(),
		initSource:      strings.TrimSpace(initSource),
		conditionSource: strings.TrimSpace(conditionSource),
		updateSource:    strings.TrimSpace(updateSource),
		bodyStatements:  statements,
	}, nil
}

func (p *classicJSStatementParser) newClassicJSForOfLoopState(bindingKind string, bindingPattern classicJSBindingPattern, iterable Value, awaitEach bool, bodySource string) (*classicJSLoopState, error) {
	statements, err := splitScriptStatements(bodySource)
	if err != nil {
		return nil, NewError(ErrorKindParse, err.Error())
	}

	frame := &classicJSLoopState{
		kind:           classicJSLoopKindForOf,
		loopEnv:        p.env.clone(),
		bodyStatements: statements,
		forOfKind:      bindingKind,
		forOfPattern:   bindingPattern,
		forOfAwait:     awaitEach,
	}

	switch iterable.Kind {
	case ValueKindArray:
		clonedValues := make([]Value, len(iterable.Array))
		for i, value := range iterable.Array {
			clonedValues[i] = cloneValueDetached(value, nil)
		}
		frame.forOfValues = clonedValues
	case ValueKindString:
		values, err := p.collectClassicJSArrayLikeValues(iterable, "for...of loop")
		if err != nil {
			return nil, err
		}
		frame.forOfValues = values
	case ValueKindObject:
		nextValue, err := p.resolveMemberAccess(scalarJSValue(iterable), "next")
		if err != nil {
			return nil, err
		}
		if nextValue.kind != jsValueScalar || !classicJSIsCallableFunctionValue(nextValue.value) {
			return nil, NewError(ErrorKindUnsupported, "for...of loops only work on array values or iterator-like object values in this bounded classic-JS slice")
		}
		clonedIterator := iterable
		frame.forOfIterator = &clonedIterator
	default:
		return nil, NewError(ErrorKindRuntime, "for...of loops require a string, array, or iterator-like object value in this bounded classic-JS slice")
	}

	return frame, nil
}

func (p *classicJSStatementParser) newClassicJSForInLoopState(bindingKind string, bindingPattern classicJSBindingPattern, keys []Value, bodySource string) (*classicJSLoopState, error) {
	statements, err := splitScriptStatements(bodySource)
	if err != nil {
		return nil, NewError(ErrorKindParse, err.Error())
	}
	clonedKeys := make([]Value, len(keys))
	for i, value := range keys {
		clonedKeys[i] = cloneValueDetached(value, nil)
	}
	return &classicJSLoopState{
		kind:           classicJSLoopKindForIn,
		loopEnv:        p.env.clone(),
		bodyStatements: statements,
		forInKind:      bindingKind,
		forInPattern:   bindingPattern,
		forInKeys:      clonedKeys,
	}, nil
}

func (p *classicJSStatementParser) resumeClassicJSState(state classicJSResumeState) (Value, classicJSResumeState, error) {
	switch current := state.(type) {
	case *classicJSYieldDeclarationState:
		if current == nil {
			return UndefinedValue(), nil, NewError(ErrorKindRuntime, "yield declaration state is unavailable")
		}
		rhs := UndefinedValue()
		if p.hasGeneratorNextValue {
			rhs = p.generatorNextValue
		}
		if current.hasPattern {
			if current.env == nil {
				current.env = newClassicJSEnvironment()
			}
			if err := p.declareBindingPattern(current.pattern, rhs, current.kind); err != nil {
				return UndefinedValue(), nil, err
			}
		} else {
			if current.env == nil {
				current.env = newClassicJSEnvironment()
			}
			if current.kind == "var" {
				current.env.bindings[current.name] = classicJSBinding{value: scalarJSValue(rhs), mutable: true}
			} else {
				if err := current.env.declare(current.name, scalarJSValue(rhs), current.kind == "let"); err != nil {
					return UndefinedValue(), nil, err
				}
			}
		}
		return UndefinedValue(), nil, nil
	case *classicJSYieldAssignmentState:
		if current == nil {
			return UndefinedValue(), nil, NewError(ErrorKindRuntime, "yield assignment state is unavailable")
		}
		rhs := scalarJSValue(UndefinedValue())
		if p.hasGeneratorNextValue {
			rhs = scalarJSValue(p.generatorNextValue)
		}
		if current.env != nil {
			p.env = current.env
		}
		result, err := p.applyAssignmentTarget(current.name, current.steps, current.op, current.current, rhs, current.privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), nil, err
		}
		return result.value, nil, nil
	case *classicJSYieldDelegationState:
		value, nextState, err := p.resumeYieldDelegationState(current)
		if err != nil {
			return UndefinedValue(), nil, err
		}
		if nextState != nil {
			return value, nextState, nil
		}
		return value, nil, nil
	case *classicJSLoopState:
		value, nextState, err := p.resumeLoopFrame(current)
		if err != nil {
			return UndefinedValue(), nil, err
		}
		if nextState != nil {
			return value, nextState, nil
		}
		return value, nil, nil
	case *classicJSSwitchState:
		value, nextState, err := p.resumeSwitchState(current)
		if err != nil {
			return UndefinedValue(), nil, err
		}
		if nextState != nil {
			return value, nextState, nil
		}
		return value, nil, nil
	case *classicJSTryState:
		value, nextState, err := p.resumeTryState(current)
		if err != nil {
			return UndefinedValue(), nil, err
		}
		if nextState != nil {
			return value, nextState, nil
		}
		return value, nil, nil
	default:
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "unsupported classic-JS continuation state")
	}
}

func (p *classicJSStatementParser) wrapYieldDeclarationSignal(yieldedValue Value, kind string, name string, pattern classicJSBindingPattern, hasPattern bool) error {
	return classicJSYieldSignal{
		value: yieldedValue,
		resumeState: &classicJSYieldDeclarationState{
			env:                p.env,
			kind:               kind,
			name:               name,
			pattern:            pattern,
			hasPattern:         hasPattern,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		},
	}
}

func (p *classicJSStatementParser) wrapYieldAssignmentSignal(yieldedValue Value, name string, steps []classicJSDeleteStep, op string, current jsValue) error {
	return classicJSYieldSignal{
		value: yieldedValue,
		resumeState: &classicJSYieldAssignmentState{
			env:                p.env,
			name:               name,
			steps:              append([]classicJSDeleteStep(nil), steps...),
			op:                 op,
			current:            current.withoutAssignTarget(),
			privateFieldPrefix: p.privateFieldPrefix,
		},
	}
}

func (p *classicJSStatementParser) applyAssignmentTarget(name string, steps []classicJSDeleteStep, op string, current jsValue, rhs jsValue, privateFieldPrefix string) (jsValue, error) {
	if len(steps) > 0 {
		if current.kind == jsValueSuper {
			if current.value.Kind != ValueKindObject && current.value.Kind != ValueKindNull {
				return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on object values in this bounded classic-JS slice")
			}

			if op == "=" {
				if _, err := assignSuperJSValuePropertyChain(p, current, steps, rhs.value, privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return rhs, nil
			}

			currentValueSource := current.value
			if current.value.Kind == ValueKindNull && len(steps) == 1 && current.receiver.Kind == ValueKindObject {
				currentValueSource = current.receiver
			}
			currentValue, err := resolveJSValuePropertyChain(p, currentValueSource, steps, privateFieldPrefix)
			if err != nil {
				return jsValue{}, err
			}

			switch op {
			case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
				result, err := classicJSApplyCompoundAssignment(currentValue, rhs.value, op)
				if err != nil {
					return jsValue{}, err
				}
				if _, err := assignSuperJSValuePropertyChain(p, current, steps, result, privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(result), nil
			case "||=":
				if !jsTruthy(currentValue) {
					if _, err := assignSuperJSValuePropertyChain(p, current, steps, rhs.value, privateFieldPrefix); err != nil {
						return jsValue{}, err
					}
					return rhs, nil
				}
				return scalarJSValue(currentValue), nil
			case "&&=":
				if jsTruthy(currentValue) {
					if _, err := assignSuperJSValuePropertyChain(p, current, steps, rhs.value, privateFieldPrefix); err != nil {
						return jsValue{}, err
					}
					return rhs, nil
				}
				return scalarJSValue(currentValue), nil
			case "??=":
				if isNullishJSValue(currentValue) {
					if _, err := assignSuperJSValuePropertyChain(p, current, steps, rhs.value, privateFieldPrefix); err != nil {
						return jsValue{}, err
					}
					return rhs, nil
				}
				return scalarJSValue(currentValue), nil
			case "**=":
				result, err := classicJSPowerValues(currentValue, rhs.value)
				if err != nil {
					return jsValue{}, err
				}
				if _, err := assignSuperJSValuePropertyChain(p, current, steps, result, privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(result), nil
			default:
				return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
			}
		}

		if current.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar object or array bindings in this bounded classic-JS slice")
		}
		if current.value.Kind != ValueKindObject && current.value.Kind != ValueKindArray && current.value.Kind != ValueKindHostReference {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
		}

		if op == "=" {
			if _, err := assignJSValuePropertyChain(p, current.value, steps, rhs.value, privateFieldPrefix); err != nil {
				return jsValue{}, err
			}
			return rhs, nil
		}

		currentValue, err := resolveJSValuePropertyChain(p, current.value, steps, privateFieldPrefix)
		if err != nil {
			return jsValue{}, err
		}

		switch op {
		case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
			result, err := classicJSApplyCompoundAssignment(currentValue, rhs.value, op)
			if err != nil {
				return jsValue{}, err
			}
			if _, err := assignJSValuePropertyChain(p, current.value, steps, result, privateFieldPrefix); err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(result), nil
		case "||=":
			if !jsTruthy(currentValue) {
				if _, err := assignJSValuePropertyChain(p, current.value, steps, rhs.value, privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return rhs, nil
			}
			return scalarJSValue(currentValue), nil
		case "&&=":
			if jsTruthy(currentValue) {
				if _, err := assignJSValuePropertyChain(p, current.value, steps, rhs.value, privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return rhs, nil
			}
			return scalarJSValue(currentValue), nil
		case "??=":
			if isNullishJSValue(currentValue) {
				if _, err := assignJSValuePropertyChain(p, current.value, steps, rhs.value, privateFieldPrefix); err != nil {
					return jsValue{}, err
				}
				return rhs, nil
			}
			return scalarJSValue(currentValue), nil
		case "**=":
			result, err := classicJSPowerValues(currentValue, rhs.value)
			if err != nil {
				return jsValue{}, err
			}
			if _, err := assignJSValuePropertyChain(p, current.value, steps, result, privateFieldPrefix); err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(result), nil
		default:
			return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
		}
	}

	switch op {
	case "=":
		if p.env == nil {
			return jsValue{}, NewError(ErrorKindRuntime, "generator state environment is unavailable")
		}
		if err := p.env.assign(name, rhs); err != nil {
			return jsValue{}, err
		}
		return rhs, nil
	case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
		result, err := classicJSApplyCompoundAssignment(current.value, rhs.value, op)
		if err != nil {
			return jsValue{}, err
		}
		if err := p.env.assign(name, scalarJSValue(result)); err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(result), nil
	case "||=":
		if !jsTruthy(current.value) {
			if err := p.env.assign(name, rhs); err != nil {
				return jsValue{}, err
			}
			return rhs, nil
		}
		return current, nil
	case "&&=":
		if jsTruthy(current.value) {
			if err := p.env.assign(name, rhs); err != nil {
				return jsValue{}, err
			}
			return rhs, nil
		}
		return current, nil
	case "??=":
		if isNullishJSValue(current.value) {
			if err := p.env.assign(name, rhs); err != nil {
				return jsValue{}, err
			}
			return rhs, nil
		}
		return current, nil
	case "**=":
		result, err := classicJSPowerValues(current.value, rhs.value)
		if err != nil {
			return jsValue{}, err
		}
		if err := p.env.assign(name, scalarJSValue(result)); err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(result), nil
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
	}
}

func (p *classicJSStatementParser) resumeLoopFrame(frame *classicJSLoopState) (Value, *classicJSLoopState, error) {
	if frame == nil {
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "loop state is unavailable")
	}

	loopParser := *p
	loopParser.resumeState = frame

	switch frame.kind {
	case classicJSLoopKindWhile:
		return loopParser.resumeWhileLoopFrame(frame)
	case classicJSLoopKindDoWhile:
		return loopParser.resumeDoWhileLoopFrame(frame)
	case classicJSLoopKindFor:
		return loopParser.resumeForLoopFrame(frame)
	case classicJSLoopKindForOf:
		return loopParser.resumeForOfLoopFrame(frame)
	case classicJSLoopKindForIn:
		return loopParser.resumeForInLoopFrame(frame)
	default:
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "unsupported loop state kind")
	}
}

func (p *classicJSStatementParser) resumeForOfLoopFrame(frame *classicJSLoopState) (Value, *classicJSLoopState, error) {
	for {
		if frame.bodyEnv == nil && frame.bodyIndex == 0 {
			if frame.iterationCount >= p.stepLimit {
				return UndefinedValue(), nil, NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
			}
			iterationEnv := frame.loopEnv.clone()
			bindingParser := *p
			bindingParser.env = iterationEnv
			iterationValue := UndefinedValue()

			if frame.forOfIterator != nil {
				nextValue, err := p.resolveMemberAccess(scalarJSValue(*frame.forOfIterator), "next")
				if err != nil {
					return UndefinedValue(), nil, err
				}
				if nextValue.kind != jsValueScalar || !classicJSIsCallableFunctionValue(nextValue.value) {
					return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "for...of loops only work on array values or iterator-like object values in this bounded classic-JS slice")
				}
				result, err := p.invoke(nextValue, nil)
				if err != nil {
					return UndefinedValue(), nil, err
				}
				if result.kind != jsValueScalar {
					return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "for...of iterator must return an object in this bounded classic-JS slice")
				}
				resultValue := unwrapPromiseValue(result.value)
				if resultValue.Kind != ValueKindObject {
					return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "for...of iterator must return an object in this bounded classic-JS slice")
				}
				doneValue, ok := lookupObjectProperty(resultValue.Object, "done")
				if !ok || doneValue.Kind != ValueKindBool {
					return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "for...of iterator result must include a boolean `done` property in this bounded classic-JS slice")
				}
				if doneValue.Bool {
					frame.forOfIterator = nil
					return UndefinedValue(), nil, nil
				}
				if itemValue, ok := lookupObjectProperty(resultValue.Object, "value"); ok {
					iterationValue = itemValue
				}
				if frame.forOfAwait {
					iterationValue = unwrapPromiseValue(iterationValue)
				}
				if err := bindingParser.declareBindingPattern(frame.forOfPattern, iterationValue, frame.forOfKind); err != nil {
					return UndefinedValue(), nil, err
				}
				frame.bodyEnv = iterationEnv
			} else {
				if frame.forOfIndex >= len(frame.forOfValues) {
					return UndefinedValue(), nil, nil
				}
				iterationValue = frame.forOfValues[frame.forOfIndex]
				if frame.forOfAwait {
					iterationValue = unwrapPromiseValue(iterationValue)
				}
				if err := bindingParser.declareBindingPattern(frame.forOfPattern, iterationValue, frame.forOfKind); err != nil {
					return UndefinedValue(), nil, err
				}
				frame.bodyEnv = iterationEnv
			}
		}

		value, nextFrame, completed, err := p.resumeLoopBody(frame)
		if err != nil {
			if classicJSBreakSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				return UndefinedValue(), nil, nil
			}
			if classicJSContinueSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				if frame.forOfIterator == nil {
					frame.forOfIndex++
				}
				frame.iterationCount++
				continue
			}
			return UndefinedValue(), nil, err
		}
		if nextFrame != nil {
			return value, nextFrame, nil
		}
		if completed {
			if frame.forOfIterator == nil {
				frame.forOfIndex++
			}
			frame.iterationCount++
			continue
		}
	}
}

func (p *classicJSStatementParser) resumeForInLoopFrame(frame *classicJSLoopState) (Value, *classicJSLoopState, error) {
	for {
		if frame.bodyEnv == nil && frame.bodyIndex == 0 {
			if frame.iterationCount >= p.stepLimit {
				return UndefinedValue(), nil, NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
			}
			if frame.forInIndex >= len(frame.forInKeys) {
				return UndefinedValue(), nil, nil
			}
			iterationEnv := frame.loopEnv.clone()
			bindingParser := *p
			bindingParser.env = iterationEnv
			if err := bindingParser.declareBindingPattern(frame.forInPattern, frame.forInKeys[frame.forInIndex], frame.forInKind); err != nil {
				return UndefinedValue(), nil, err
			}
			frame.bodyEnv = iterationEnv
		}

		value, nextFrame, completed, err := p.resumeLoopBody(frame)
		if err != nil {
			if classicJSBreakSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				return UndefinedValue(), nil, nil
			}
			if classicJSContinueSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				frame.forInIndex++
				frame.iterationCount++
				continue
			}
			return UndefinedValue(), nil, err
		}
		if nextFrame != nil {
			return value, nextFrame, nil
		}
		if completed {
			frame.forInIndex++
			frame.iterationCount++
			continue
		}
	}
}

func (p *classicJSStatementParser) resumeLoopBody(frame *classicJSLoopState) (Value, *classicJSLoopState, bool, error) {
	if frame == nil {
		return UndefinedValue(), nil, false, NewError(ErrorKindRuntime, "loop state is unavailable")
	}
	if frame.bodyEnv == nil {
		frame.bodyEnv = frame.loopEnv.clone()
	}

	if frame.bodyState != nil {
		value, nextState, err := p.resumeClassicJSState(frame.bodyState)
		if err != nil {
			return UndefinedValue(), nil, false, err
		}
		if nextState != nil {
			frame.bodyState = nextState
			return value, frame, false, nil
		}
		frame.bodyState = nil
		frame.bodyIndex++
	}

	for frame.bodyIndex < len(frame.bodyStatements) {
		statement := strings.TrimSpace(frame.bodyStatements[frame.bodyIndex])
		if statement == "" {
			frame.bodyIndex++
			continue
		}

		_, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, p.host, frame.bodyEnv, p.stepLimit, p.allowAwait, p.allowYield, p.allowReturn, frame, p.newTarget, p.hasNewTarget, nil, nil)
		if err != nil {
			if yieldedValue, nextState, ok := classicJSYieldSignalDetails(err); ok {
				if nextState != nil && nextState != classicJSResumeState(frame) {
					frame.bodyState = nextState
					return yieldedValue, frame, false, nil
				}
				frame.bodyIndex++
				return yieldedValue, frame, false, nil
			}
			return UndefinedValue(), nil, false, err
		}
		frame.bodyIndex++
	}

	frame.bodyIndex = 0
	frame.bodyEnv = nil
	frame.bodyState = nil
	return UndefinedValue(), nil, true, nil
}

func resetClassicJSLoopBody(frame *classicJSLoopState) {
	if frame == nil {
		return
	}
	frame.bodyIndex = 0
	frame.bodyEnv = nil
	frame.bodyState = nil
}

func (p *classicJSStatementParser) resumeWhileLoopFrame(frame *classicJSLoopState) (Value, *classicJSLoopState, error) {
	for {
		if frame.bodyEnv == nil && frame.bodyIndex == 0 {
			if frame.iterationCount >= p.stepLimit {
				return UndefinedValue(), nil, NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
			}
			condition, err := p.evalExpressionWithEnv(frame.conditionSource, frame.loopEnv.clone())
			if err != nil {
				return UndefinedValue(), nil, err
			}
			if !jsTruthy(condition) {
				return UndefinedValue(), nil, nil
			}
		}

		value, nextFrame, completed, err := p.resumeLoopBody(frame)
		if err != nil {
			if classicJSBreakSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				return UndefinedValue(), nil, nil
			}
			if classicJSContinueSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				frame.iterationCount++
				continue
			}
			return UndefinedValue(), nil, err
		}
		if nextFrame != nil {
			return value, nextFrame, nil
		}
		if completed {
			frame.iterationCount++
			continue
		}
	}
}

func (p *classicJSStatementParser) resumeDoWhileLoopFrame(frame *classicJSLoopState) (Value, *classicJSLoopState, error) {
	if !frame.started {
		frame.started = true
	}

	for {
		value, nextFrame, completed, err := p.resumeLoopBody(frame)
		if err != nil {
			if classicJSBreakSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				return UndefinedValue(), nil, nil
			}
			if classicJSContinueSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				frame.iterationCount++
				condition, err := p.evalExpressionWithEnv(frame.conditionSource, frame.loopEnv.clone())
				if err != nil {
					return UndefinedValue(), nil, err
				}
				if !jsTruthy(condition) {
					return UndefinedValue(), nil, nil
				}
				if frame.iterationCount >= p.stepLimit {
					return UndefinedValue(), nil, NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
				}
				continue
			}
			return UndefinedValue(), nil, err
		}
		if nextFrame != nil {
			return value, nextFrame, nil
		}
		if completed {
			frame.iterationCount++
			condition, err := p.evalExpressionWithEnv(frame.conditionSource, frame.loopEnv.clone())
			if err != nil {
				return UndefinedValue(), nil, err
			}
			if !jsTruthy(condition) {
				return UndefinedValue(), nil, nil
			}
			if frame.iterationCount >= p.stepLimit {
				return UndefinedValue(), nil, NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
			}
			continue
		}
	}
}

func (p *classicJSStatementParser) resumeForLoopFrame(frame *classicJSLoopState) (Value, *classicJSLoopState, error) {
	if !frame.initDone {
		if frame.initSource != "" {
			if hasClassicJSDeclarationKeyword(frame.initSource) {
				if _, err := p.evalStatementWithEnv(frame.initSource, frame.loopEnv); err != nil {
					return UndefinedValue(), nil, err
				}
			} else {
				if _, err := p.evalExpressionWithEnv(frame.initSource, frame.loopEnv); err != nil {
					return UndefinedValue(), nil, err
				}
			}
		}
		frame.initDone = true
	}

	for {
		if frame.bodyEnv == nil && frame.bodyIndex == 0 {
			if frame.iterationCount >= p.stepLimit {
				return UndefinedValue(), nil, NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
			}
			if frame.conditionSource != "" {
				condition, err := p.evalExpressionWithEnv(frame.conditionSource, frame.loopEnv.clone())
				if err != nil {
					return UndefinedValue(), nil, err
				}
				if !jsTruthy(condition) {
					return UndefinedValue(), nil, nil
				}
			}
		}

		value, nextFrame, completed, err := p.resumeLoopBody(frame)
		if err != nil {
			if classicJSBreakSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				return UndefinedValue(), nil, nil
			}
			if classicJSContinueSignalMatchesLabel(err, frame.label) {
				resetClassicJSLoopBody(frame)
				if frame.updateSource != "" {
					if _, err := p.evalExpressionWithEnv(frame.updateSource, frame.loopEnv.clone()); err != nil {
						return UndefinedValue(), nil, err
					}
				}
				frame.iterationCount++
				continue
			}
			return UndefinedValue(), nil, err
		}
		if nextFrame != nil {
			return value, nextFrame, nil
		}
		if completed {
			if frame.updateSource != "" {
				if _, err := p.evalExpressionWithEnv(frame.updateSource, frame.loopEnv.clone()); err != nil {
					return UndefinedValue(), nil, err
				}
			}
			frame.iterationCount++
			continue
		}
	}
}

func (p *classicJSStatementParser) parseWhileStatement() (Value, error) {
	conditionSource, err := p.consumeParenthesizedSource("while")
	if err != nil {
		return UndefinedValue(), err
	}

	bodySource, err := p.consumeLoopBodySource()
	if err != nil {
		return UndefinedValue(), err
	}

	frame, err := p.newClassicJSLoopState(classicJSLoopKindWhile, conditionSource, "", "", bodySource)
	if err != nil {
		return UndefinedValue(), err
	}
	frame.label = p.statementLabel
	value, nextFrame, err := p.resumeLoopFrame(frame)
	if err != nil {
		return UndefinedValue(), err
	}
	if nextFrame != nil {
		return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextFrame}
	}

	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseDoWhileStatement() (Value, error) {
	bodySource, err := p.consumeLoopBodySource()
	if err != nil {
		return UndefinedValue(), err
	}

	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("while"); !ok {
		return UndefinedValue(), NewError(ErrorKindParse, "expected `while` after `do` block")
	} else {
		p.pos += len(keyword)
	}

	conditionSource, err := p.consumeParenthesizedSource("while")
	if err != nil {
		return UndefinedValue(), err
	}

	frame, err := p.newClassicJSLoopState(classicJSLoopKindDoWhile, conditionSource, "", "", bodySource)
	if err != nil {
		return UndefinedValue(), err
	}
	frame.label = p.statementLabel
	value, nextFrame, err := p.resumeLoopFrame(frame)
	if err != nil {
		return UndefinedValue(), err
	}
	if nextFrame != nil {
		return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextFrame}
	}

	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseForStatement() (Value, error) {
	awaitEach := false
	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("await"); ok {
		if !p.allowAwait {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "`for await...of` is only supported inside bounded async bodies in this slice")
		}
		p.pos += len(keyword)
		awaitEach = true
	}

	headerSource, err := p.consumeParenthesizedSource("for")
	if err != nil {
		return UndefinedValue(), err
	}

	if bindingSource, iterableSource, ok, err := splitClassicJSForOfHeader(headerSource); err != nil {
		return UndefinedValue(), err
	} else if ok {
		bindingKind, bindingPattern, err := parseClassicJSForOfBinding(bindingSource, awaitEach)
		if err != nil {
			return UndefinedValue(), err
		}
		iterableValue, err := p.evalExpressionWithEnv(iterableSource, p.env.clone())
		if err != nil {
			return UndefinedValue(), err
		}
		iterableValue = unwrapPromiseValue(iterableValue)

		bodySource, err := p.consumeLoopBodySource()
		if err != nil {
			return UndefinedValue(), err
		}

		frame, err := p.newClassicJSForOfLoopState(bindingKind, bindingPattern, iterableValue, awaitEach, bodySource)
		if err != nil {
			return UndefinedValue(), err
		}
		frame.label = p.statementLabel
		value, nextFrame, err := p.resumeLoopFrame(frame)
		if err != nil {
			return UndefinedValue(), err
		}
		if nextFrame != nil {
			return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextFrame}
		}
		return UndefinedValue(), nil
	}

	if awaitEach {
		return UndefinedValue(), NewError(ErrorKindParse, "expected `of` after `for await` in this bounded classic-JS slice")
	}

	if bindingSource, iterableSource, ok, err := splitClassicJSForInHeader(headerSource); err != nil {
		return UndefinedValue(), err
	} else if ok {
		bindingKind, bindingPattern, err := parseClassicJSForInBinding(bindingSource)
		if err != nil {
			return UndefinedValue(), err
		}
		iterableValue, err := p.evalExpressionWithEnv(iterableSource, p.env.clone())
		if err != nil {
			return UndefinedValue(), err
		}
		keys, err := classicJSForInKeys(iterableValue)
		if err != nil {
			return UndefinedValue(), err
		}

		bodySource, err := p.consumeLoopBodySource()
		if err != nil {
			return UndefinedValue(), err
		}

		frame, err := p.newClassicJSForInLoopState(bindingKind, bindingPattern, keys, bodySource)
		if err != nil {
			return UndefinedValue(), err
		}
		frame.label = p.statementLabel
		value, nextFrame, err := p.resumeLoopFrame(frame)
		if err != nil {
			return UndefinedValue(), err
		}
		if nextFrame != nil {
			return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextFrame}
		}
		return UndefinedValue(), nil
	}

	initSource, conditionSource, updateSource, err := splitClassicJSForHeader(headerSource)
	if err != nil {
		return UndefinedValue(), err
	}

	bodySource, err := p.consumeLoopBodySource()
	if err != nil {
		return UndefinedValue(), err
	}

	frame, err := p.newClassicJSLoopState(classicJSLoopKindFor, conditionSource, initSource, updateSource, bodySource)
	if err != nil {
		return UndefinedValue(), err
	}
	frame.label = p.statementLabel
	value, nextFrame, err := p.resumeLoopFrame(frame)
	if err != nil {
		return UndefinedValue(), err
	}
	if nextFrame != nil {
		return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextFrame}
	}

	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseClassStatement() (Value, error) {
	_, _, err := p.parseClassDeclaration(false)
	if err != nil {
		return UndefinedValue(), err
	}
	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseClassDeclaration(allowAnonymous bool) (string, Value, error) {
	return p.parseClassDeclarationWithBinding(allowAnonymous, true)
}

func (p *classicJSStatementParser) parseClassDeclarationWithBinding(allowAnonymous bool, bindInEnv bool) (string, Value, error) {
	p.skipSpaceAndComments()
	name := ""
	anonymous := false
	if allowAnonymous {
		if p.peekByte() == '{' {
			anonymous = true
		} else if _, ok := p.peekKeyword("extends"); ok {
			anonymous = true
		}
	}
	if !anonymous {
		parsedName, err := p.parseIdentifier()
		if err != nil {
			return "", UndefinedValue(), NewError(ErrorKindParse, "class declarations require an identifier in this bounded classic-JS slice")
		}
		if isClassicJSReservedDeclarationName(parsedName) {
			return "", UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("reserved class name %q is not allowed in this bounded classic-JS slice", parsedName))
		}
		name = parsedName
	}

	p.skipSpaceAndComments()
	classEnv := p.env
	if !bindInEnv {
		classEnv = classEnv.clone()
	} else if classEnv == nil {
		classEnv = newClassicJSEnvironment()
		p.env = classEnv
	}

	var baseClassValue jsValue
	var baseClassDef *classicJSClassDefinition
	hasExtendsClause := false
	if keyword, ok := p.peekKeyword("extends"); ok {
		hasExtendsClause = true
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		baseValue, err := p.parseScalarExpression()
		if err != nil {
			return "", UndefinedValue(), err
		}
		baseClassValue = scalarJSValue(baseValue)
		if baseClassValue.kind != jsValueScalar {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, "class inheritance requires a class-valued expression or null in this bounded classic-JS slice")
		}
		if baseClassValue.value.Kind != ValueKindObject && baseClassValue.value.Kind != ValueKindNull && baseClassValue.value.Kind != ValueKindFunction {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, "class inheritance requires a class-valued expression, constructible function value, or null in this bounded classic-JS slice")
		}
		if baseClassValue.value.Kind == ValueKindObject {
			baseClassDef, ok = resolveClassicJSClassDefinition(baseClassValue.value, classEnv)
			if !ok || baseClassDef == nil {
				return "", UndefinedValue(), NewError(ErrorKindUnsupported, "class inheritance requires a class expression or class binding in this bounded classic-JS slice")
			}
		}
	}

	bodySource, err := p.consumeBlockSource()
	if err != nil {
		return "", UndefinedValue(), err
	}

	if name != "" {
		if err := classEnv.declare(name, scalarJSValue(ObjectValue([]ObjectEntry{
			{
				Key:   "prototype",
				Value: ObjectValue(nil),
			},
		})), false); err != nil {
			return "", UndefinedValue(), err
		}
	}

	members, err := splitClassicJSClassMembers(bodySource)
	if err != nil {
		return "", UndefinedValue(), err
	}

	staticEntries := make([]ObjectEntry, 0, len(members))
	prototypeEntries := make([]ObjectEntry, 0, len(members))
	classDef := &classicJSClassDefinition{env: classEnv}
	classDef.privateFieldPrefix = fmt.Sprintf("\x00private:%s:%p:", name, classDef)
	classDef.instanceMarker = fmt.Sprintf("%p", classDef)
	classEnv.setClassDefinition(classDef.instanceMarker, classDef)
	if !bindInEnv && p.env != nil {
		p.env.setClassDefinition(classDef.instanceMarker, classDef)
	}
	classEval := p.cloneForClassEvaluation()
	classEval.privateClass = classDef
	classEval.privateFieldPrefix = classDef.privateFieldPrefix
	prevPrivateClass := p.privateClass
	prevPrivateFieldPrefix := p.privateFieldPrefix
	p.privateClass = classDef
	p.privateFieldPrefix = classDef.privateFieldPrefix
	defer func() {
		p.privateClass = prevPrivateClass
		p.privateFieldPrefix = prevPrivateFieldPrefix
	}()
	if baseClassDef != nil {
		classDef.hasSuper = true
		classDef.superStaticTarget = baseClassValue.value
		classDef.instanceFields = append(classDef.instanceFields, baseClassDef.instanceFields...)
		prototypeValue, ok := classicJSClassPrototypeValue(baseClassValue.value)
		if !ok || prototypeValue.Kind != ValueKindObject {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("class inheritance requires a prototype object for %q in this bounded classic-JS slice", name))
		}
		classDef.superInstanceTarget = prototypeValue
		prototypeEntries = append(prototypeEntries, append([]ObjectEntry(nil), prototypeValue.Object...)...)
		for _, entry := range baseClassValue.value.Object {
			if entry.Key == "prototype" || entry.Key == classicJSClassPrototypeKey(baseClassDef.instanceMarker) {
				continue
			}
			staticEntries = append(staticEntries, entry)
		}
	} else if baseClassValue.value.Kind == ValueKindFunction && baseClassValue.value.Function != nil && baseClassValue.value.Function.constructible {
		classDef.hasSuper = true
		classDef.superStaticTarget = baseClassValue.value
		if prototypeValue, ok := classicJSConstructibleFunctionPrototypeValue(baseClassValue.value); ok {
			classDef.superInstanceTarget = prototypeValue
			prototypeEntries = append(prototypeEntries, append([]ObjectEntry(nil), prototypeValue.Object...)...)
		}
	}
	if !hasExtendsClause {
		classDef.hasSuper = true
		classDef.superStaticTarget = classicJSBaseClassDefaultSuperTarget()
		classDef.superInstanceTarget = classicJSBaseClassDefaultSuperTarget()
	}
	prototypeEntries = append(prototypeEntries, ObjectEntry{
		Key:   classicJSInstanceMarkerKey(classDef.instanceMarker),
		Value: BoolValue(true),
	})
	if name != "" {
		classEnv.setClassDefinition(name, classDef)
	}
	prototypeValue := ObjectValue(append([]ObjectEntry(nil), prototypeEntries...))
	currentClassValue := ObjectValue([]ObjectEntry{
		{
			Key:   classicJSClassPrototypeKey(classDef.instanceMarker),
			Value: prototypeValue,
		},
		{
			Key:   "prototype",
			Value: prototypeValue,
		},
	})
	currentClassValue.ClassKey = classDef.instanceMarker
	currentClassValue.ClassDefinition = classDef
	publishClassValue := func() {
		prototypeValue := ObjectValue(append([]ObjectEntry(nil), prototypeEntries...))
		entries := make([]ObjectEntry, 0, 2+len(staticEntries))
		entries = append(entries, ObjectEntry{
			Key:   classicJSClassPrototypeKey(classDef.instanceMarker),
			Value: prototypeValue,
		})
		entries = append(entries, ObjectEntry{
			Key:   "prototype",
			Value: prototypeValue,
		})
		entries = append(entries, staticEntries...)
		currentClassValue = ObjectValue(entries)
		currentClassValue.ClassKey = classDef.instanceMarker
		currentClassValue.ClassDefinition = classDef
		if name != "" {
			classEnv.bindings[name] = classicJSBinding{
				value:   scalarJSValue(currentClassValue),
				mutable: false,
			}
		}
	}
	publishClassValue()

	for _, member := range members {
		switch member.kind {
		case "static-block":
			prevPrivateClass := p.privateClass
			prevPrivateFieldPrefix := p.privateFieldPrefix
			p.privateClass = classDef
			p.privateFieldPrefix = classDef.privateFieldPrefix
			staticBlockEnv := classEnv.clone()
			_ = staticBlockEnv.declare("this", scalarJSValue(currentClassValue), false)
			if classDef.hasSuper {
				_ = staticBlockEnv.declare("super", superJSValue(classDef.superStaticTarget, currentClassValue), false)
			}
			if _, err := classEval.evalProgramWithEnv(member.staticBlock, staticBlockEnv); err != nil {
				p.privateClass = prevPrivateClass
				p.privateFieldPrefix = prevPrivateFieldPrefix
				return "", UndefinedValue(), err
			}
			p.privateClass = prevPrivateClass
			p.privateFieldPrefix = prevPrivateFieldPrefix
		case "static-field":
			fieldName, err := p.resolveClassicJSClassMemberName(member.fieldName, member.fieldNameSource, classEnv, currentClassValue, classDef.superStaticTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && fieldName == "prototype" {
				classDef.hasStaticPrototype = true
			}
			value := UndefinedValue()
			prevPrivateClass := p.privateClass
			prevPrivateFieldPrefix := p.privateFieldPrefix
			p.privateClass = classDef
			p.privateFieldPrefix = classDef.privateFieldPrefix
			staticFieldEnv := classEnv.clone()
			_ = staticFieldEnv.declare("this", scalarJSValue(currentClassValue), false)
			if classDef.hasSuper {
				_ = staticFieldEnv.declare("super", superJSValue(classDef.superStaticTarget, currentClassValue), false)
			}
			if strings.TrimSpace(member.fieldInit) == "" {
				if member.private {
					staticEntries = append(staticEntries, ObjectEntry{Key: classDef.privateFieldKey(fieldName), Value: value})
				} else {
					staticEntries = append(staticEntries, ObjectEntry{Key: fieldName, Value: value})
				}
				p.privateClass = prevPrivateClass
				p.privateFieldPrefix = prevPrivateFieldPrefix
				publishClassValue()
				continue
			}
			parsed, err := classEval.evalExpressionWithEnv(member.fieldInit, staticFieldEnv)
			p.privateClass = prevPrivateClass
			p.privateFieldPrefix = prevPrivateFieldPrefix
			if err != nil {
				return "", UndefinedValue(), err
			}
			value = parsed
			if member.private {
				staticEntries = append(staticEntries, ObjectEntry{Key: classDef.privateFieldKey(fieldName), Value: value})
			} else {
				staticEntries = append(staticEntries, ObjectEntry{Key: fieldName, Value: value})
			}
			publishClassValue()
		case "static-method":
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv, currentClassValue, classDef.superStaticTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && methodName == "prototype" {
				classDef.hasStaticPrototype = true
			}
			callable := &classicJSArrowFunction{
				name:         methodName,
				params:       member.methodParams,
				restName:     member.methodRestName,
				body:         member.methodBody,
				bodyIsBlock:  true,
				async:        member.async,
				allowReturn:  true,
				env:          classEnv,
				privateClass: classDef,
			}
			if member.generator {
				callable.generatorFunction = &classicJSGeneratorFunction{
					name:               methodName,
					params:             member.methodParams,
					restName:           member.methodRestName,
					body:               member.methodBody,
					async:              member.async,
					env:                classEnv,
					privateClass:       classDef,
					privateFieldPrefix: classDef.privateFieldPrefix,
					superTarget:        classDef.superStaticTarget,
					hasSuperTarget:     classDef.hasSuper,
				}
			}
			if classDef.hasSuper {
				callable.superTarget = classDef.superStaticTarget
				callable.hasSuperTarget = true
			}
			key := methodName
			if member.private {
				key = classDef.privateFieldKey(methodName)
			}
			staticEntries = append(staticEntries, ObjectEntry{Key: key, Value: FunctionValue(callable)})
			publishClassValue()
		case "static-getter":
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv, currentClassValue, classDef.superStaticTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && methodName == "prototype" {
				classDef.hasStaticPrototype = true
			}
			callable := &classicJSArrowFunction{
				name:               methodName,
				body:               member.methodBody,
				bodyIsBlock:        true,
				allowReturn:        true,
				objectAccessor:     true,
				env:                classEnv,
				privateClass:       classDef,
				privateFieldPrefix: classDef.privateFieldPrefix,
			}
			if classDef.hasSuper {
				callable.superTarget = classDef.superStaticTarget
				callable.hasSuperTarget = true
			}
			key := methodName
			if member.private {
				key = classDef.privateFieldKey(methodName)
			}
			staticEntries = append(staticEntries, ObjectEntry{Key: key, Value: FunctionValue(callable)})
			publishClassValue()
		case "static-setter":
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv, currentClassValue, classDef.superStaticTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && methodName == "prototype" {
				classDef.hasStaticPrototype = true
			}
			callable := &classicJSArrowFunction{
				name:               methodName,
				params:             member.methodParams,
				body:               member.methodBody,
				bodyIsBlock:        true,
				allowReturn:        true,
				objectSetter:       true,
				env:                classEnv,
				privateClass:       classDef,
				privateFieldPrefix: classDef.privateFieldPrefix,
			}
			if classDef.hasSuper {
				callable.superTarget = classDef.superStaticTarget
				callable.hasSuperTarget = true
			}
			key := methodName
			if member.private {
				key = classDef.privateFieldKey(methodName)
			}
			staticEntries = append(staticEntries, ObjectEntry{Key: classicJSObjectSetterStorageKey(key), Value: FunctionValue(callable)})
			publishClassValue()
		case "instance-method":
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv, currentClassValue, classDef.superInstanceTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			callable := &classicJSArrowFunction{
				name:         methodName,
				params:       member.methodParams,
				restName:     member.methodRestName,
				body:         member.methodBody,
				bodyIsBlock:  true,
				async:        member.async,
				allowReturn:  true,
				env:          classEnv,
				privateClass: classDef,
			}
			if member.generator {
				callable.generatorFunction = &classicJSGeneratorFunction{
					name:               methodName,
					params:             member.methodParams,
					restName:           member.methodRestName,
					body:               member.methodBody,
					async:              member.async,
					env:                classEnv,
					privateClass:       classDef,
					privateFieldPrefix: classDef.privateFieldPrefix,
					superTarget:        classDef.superInstanceTarget,
					hasSuperTarget:     classDef.hasSuper,
				}
			}
			if classDef.hasSuper {
				callable.superTarget = classDef.superInstanceTarget
				callable.hasSuperTarget = true
			}
			key := methodName
			if member.private {
				key = classDef.privateFieldKey(methodName)
			}
			prototypeEntries = append(prototypeEntries, ObjectEntry{Key: key, Value: FunctionValue(callable)})
			publishClassValue()
		case "instance-setter":
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv, currentClassValue, classDef.superInstanceTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			callable := &classicJSArrowFunction{
				name:               methodName,
				params:             member.methodParams,
				body:               member.methodBody,
				bodyIsBlock:        true,
				allowReturn:        true,
				objectSetter:       true,
				env:                classEnv,
				privateClass:       classDef,
				privateFieldPrefix: classDef.privateFieldPrefix,
			}
			if classDef.hasSuper {
				callable.superTarget = classDef.superInstanceTarget
				callable.hasSuperTarget = true
			}
			key := methodName
			if member.private {
				key = classDef.privateFieldKey(methodName)
			}
			prototypeEntries = append(prototypeEntries, ObjectEntry{Key: classicJSObjectSetterStorageKey(key), Value: FunctionValue(callable)})
			publishClassValue()
		case "instance-getter":
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv, currentClassValue, classDef.superInstanceTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			callable := &classicJSArrowFunction{
				name:               methodName,
				body:               member.methodBody,
				bodyIsBlock:        true,
				allowReturn:        true,
				objectAccessor:     true,
				env:                classEnv,
				privateClass:       classDef,
				privateFieldPrefix: classDef.privateFieldPrefix,
			}
			if classDef.hasSuper {
				callable.superTarget = classDef.superInstanceTarget
				callable.hasSuperTarget = true
			}
			key := methodName
			if member.private {
				key = classDef.privateFieldKey(methodName)
			}
			prototypeEntries = append(prototypeEntries, ObjectEntry{Key: key, Value: FunctionValue(callable)})
			publishClassValue()
		case "instance-field":
			fieldName, err := p.resolveClassicJSClassMemberName(member.fieldName, member.fieldNameSource, classEnv, currentClassValue, classDef.superInstanceTarget, classDef.hasSuper)
			if err != nil {
				return "", UndefinedValue(), err
			}
			classDef.instanceFields = append(classDef.instanceFields, classicJSClassFieldDefinition{
				env:              classEnv,
				name:             fieldName,
				init:             member.fieldInit,
				private:          member.private,
				privateKeyPrefix: classDef.privateFieldPrefix,
			})
		default:
			return "", UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("unsupported class member kind %q in this bounded classic-JS slice", member.kind))
		}
	}

	return name, currentClassValue, nil
}

func (p *classicJSStatementParser) parseSwitchStatement() (Value, error) {
	discriminantSource, err := p.consumeParenthesizedSource("switch")
	if err != nil {
		return UndefinedValue(), err
	}

	bodySource, err := p.consumeBlockSource()
	if err != nil {
		return UndefinedValue(), err
	}

	clauses, err := splitClassicJSSwitchClauses(bodySource)
	if err != nil {
		return UndefinedValue(), err
	}
	if len(clauses) == 0 {
		return UndefinedValue(), nil
	}

	switchEnv := p.env.clone()
	discriminant, err := p.evalExpressionWithEnv(discriminantSource, switchEnv)
	if err != nil {
		return UndefinedValue(), err
	}

	startIndex := -1
	defaultIndex := -1
	for i, clause := range clauses {
		switch clause.kind {
		case "default":
			defaultIndex = i
			continue
		case "case":
			if startIndex >= 0 {
				continue
			}
			label, err := p.evalExpressionWithEnv(clause.label, switchEnv)
			if err != nil {
				return UndefinedValue(), err
			}
			matched, err := classicJSSwitchMatches(discriminant, label)
			if err != nil {
				return UndefinedValue(), err
			}
			if matched {
				startIndex = i
			}
		default:
			return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("unsupported switch clause kind %q", clause.kind))
		}
	}
	if startIndex < 0 {
		startIndex = defaultIndex
	}
	if startIndex < 0 {
		return UndefinedValue(), nil
	}

	state := &classicJSSwitchState{
		label:       p.statementLabel,
		clauses:     append([]classicJSSwitchClause(nil), clauses...),
		env:         switchEnv,
		clauseIndex: startIndex,
	}
	value, nextState, err := p.resumeSwitchState(state)
	if err != nil {
		return UndefinedValue(), err
	}
	if nextState != nil {
		return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextState}
	}

	return UndefinedValue(), nil
}

func (p *classicJSStatementParser) parseTryStatement() (Value, error) {
	trySource, err := p.consumeBlockSource()
	if err != nil {
		return UndefinedValue(), err
	}

	var (
		catchSource   string
		finallySource string
		catchPattern  classicJSBindingPattern
		catchBound    bool
		hasCatch      bool
		hasFinally    bool
	)

	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("catch"); ok {
		hasCatch = true
		p.pos += len(keyword)
		catchPattern, catchBound, err = p.parseCatchBinding()
		if err != nil {
			return UndefinedValue(), err
		}
		catchSource, err = p.consumeBlockSource()
		if err != nil {
			return UndefinedValue(), err
		}
	}

	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("finally"); ok {
		hasFinally = true
		p.pos += len(keyword)
		finallySource, err = p.consumeBlockSource()
		if err != nil {
			return UndefinedValue(), err
		}
	}

	if !hasCatch && !hasFinally {
		return UndefinedValue(), NewError(ErrorKindParse, "expected `catch` or `finally` after `try` block")
	}

	state := &classicJSTryState{
		label:            p.statementLabel,
		catchSource:      catchSource,
		finallySource:    finallySource,
		catchEnvTemplate: p.env.clone(),
		catchPattern:     catchPattern,
		catchBound:       catchBound,
		hasCatch:         hasCatch,
		hasFinally:       hasFinally,
		stage:            classicJSTryStageTry,
	}
	state.tryBlock, err = newClassicJSBlockState(trySource, p.env.clone(), state)
	if err != nil {
		return UndefinedValue(), err
	}
	if hasFinally {
		state.finallyBlock, err = newClassicJSBlockState(finallySource, p.env.clone(), state)
		if err != nil {
			return UndefinedValue(), err
		}
	}

	value, nextState, err := p.resumeTryState(state)
	if err != nil {
		return UndefinedValue(), err
	}
	if nextState != nil {
		return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: nextState}
	}

	return value, nil
}

func newClassicJSBlockState(source string, env *classicJSEnvironment, owner classicJSResumeState) (*classicJSBlockState, error) {
	statements, err := splitScriptStatements(source)
	if err != nil {
		return nil, NewError(ErrorKindParse, err.Error())
	}
	return &classicJSBlockState{
		statements: statements,
		env:        env,
		owner:      owner,
	}, nil
}

func (p *classicJSStatementParser) resumeBlockState(state *classicJSBlockState) (Value, *classicJSBlockState, error) {
	if state == nil {
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "block state is unavailable")
	}
	for {
		if state.child != nil {
			value, nextState, err := p.resumeClassicJSState(state.child)
			if err != nil {
				return UndefinedValue(), nil, err
			}
			if nextState != nil {
				state.child = nextState
				return value, state, nil
			}
			state.child = nil
			state.lastValue = value
			state.index++
			continue
		}

		if state.index >= len(state.statements) {
			return state.lastValue, nil, nil
		}

		statement := strings.TrimSpace(state.statements[state.index])
		if statement == "" {
			state.index++
			continue
		}

		value, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, p.host, state.env, p.stepLimit, p.allowAwait, true, p.allowReturn, state.owner, p.newTarget, p.hasNewTarget, nil, nil)
		if err != nil {
			if yieldedValue, nextState, ok := classicJSYieldSignalDetails(err); ok {
				if nextState != nil && nextState != state.owner {
					state.child = nextState
				} else {
					state.index++
				}
				return yieldedValue, state, nil
			}
			return UndefinedValue(), nil, err
		}
		state.lastValue = value
		state.index++
	}
}

func (p *classicJSStatementParser) resumeSwitchState(state *classicJSSwitchState) (Value, *classicJSSwitchState, error) {
	if state == nil {
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "switch state is unavailable")
	}

	for state.clauseIndex < len(state.clauses) {
		if state.bodyStatements == nil {
			statements, err := splitScriptStatements(state.clauses[state.clauseIndex].body)
			if err != nil {
				return UndefinedValue(), nil, NewError(ErrorKindParse, err.Error())
			}
			state.bodyStatements = statements
			state.bodyIndex = 0
			state.bodyState = nil
		}

		if state.bodyState != nil {
			value, nextState, err := p.resumeClassicJSState(state.bodyState)
			if err != nil {
				if classicJSBreakSignalMatchesLabel(err, state.label) {
					return UndefinedValue(), nil, nil
				}
				if label, ok := classicJSContinueSignalLabel(err); ok {
					if label != "" && label == state.label {
						return UndefinedValue(), nil, NewError(ErrorKindParse, "continue statements cannot target labeled switch statements in this bounded classic-JS slice")
					}
				}
				return UndefinedValue(), nil, err
			}
			if nextState != nil {
				state.bodyState = nextState
				return value, state, nil
			}
			state.bodyState = nil
			state.bodyIndex++
			continue
		}

		for state.bodyIndex < len(state.bodyStatements) {
			statement := strings.TrimSpace(state.bodyStatements[state.bodyIndex])
			if statement == "" {
				state.bodyIndex++
				continue
			}
			if isClassicJSBreakStatement(statement) {
				return UndefinedValue(), nil, nil
			}

			_, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, p.host, state.env, p.stepLimit, p.allowAwait, true, p.allowReturn, state, p.newTarget, p.hasNewTarget, nil, nil)
			if err != nil {
				if yieldedValue, nextState, ok := classicJSYieldSignalDetails(err); ok {
					if nextState != nil && nextState != state {
						state.bodyState = nextState
					} else {
						state.bodyIndex++
					}
					return yieldedValue, state, nil
				}
				if classicJSBreakSignalMatchesLabel(err, state.label) {
					return UndefinedValue(), nil, nil
				}
				if label, ok := classicJSContinueSignalLabel(err); ok {
					if label != "" && label == state.label {
						return UndefinedValue(), nil, NewError(ErrorKindParse, "continue statements cannot target labeled switch statements in this bounded classic-JS slice")
					}
				}
				return UndefinedValue(), nil, err
			}
			state.bodyIndex++
		}

		state.clauseIndex++
		state.bodyStatements = nil
		state.bodyIndex = 0
		state.bodyState = nil
	}

	return UndefinedValue(), nil, nil
}

func (p *classicJSStatementParser) resumeTryState(state *classicJSTryState) (Value, *classicJSTryState, error) {
	if state == nil {
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "try state is unavailable")
	}

	for {
		switch state.stage {
		case classicJSTryStageTry:
			value, nextBlock, err := p.resumeBlockState(state.tryBlock)
			if nextBlock != nil {
				state.tryBlock = nextBlock
				return value, state, nil
			}
			if err != nil {
				if throwValue, ok := classicJSThrowSignalValue(err); ok {
					state.pendingThrow = throwValue
					state.hasPendingThrow = true
					state.pendingErr = NewError(ErrorKindRuntime, ToJSString(throwValue))
				} else {
					state.hasPendingThrow = false
					state.pendingThrow = UndefinedValue()
					state.pendingErr = err
				}
				if _, ok := classicJSReturnSignalValue(err); ok || classicJSBreakSignalValue(err) || classicJSContinueSignalValue(err) {
					state.stage = classicJSTryStageFinally
					if !state.hasFinally {
						state.stage = classicJSTryStageDone
						return p.finalizeTryCompletion(state)
					}
					continue
				}
				if state.hasCatch {
					state.stage = classicJSTryStageCatch
					continue
				}
				if state.hasFinally {
					state.stage = classicJSTryStageFinally
					continue
				}
				state.stage = classicJSTryStageDone
				return p.finalizeTryCompletion(state)
			}
			state.result = value
			state.tryBlock = nil
			if state.hasFinally {
				state.stage = classicJSTryStageFinally
				continue
			}
			state.stage = classicJSTryStageDone
			return state.result, nil, nil

		case classicJSTryStageCatch:
			if !state.hasCatch {
				state.stage = classicJSTryStageFinally
				continue
			}
			if state.catchBlock == nil {
				catchEnv := state.catchEnvTemplate.clone()
				if catchEnv == nil {
					catchEnv = newClassicJSEnvironment()
				}
				if state.catchBound {
					catchValue := StringValue("")
					if state.hasPendingThrow {
						catchValue = state.pendingThrow
					} else if state.pendingErr != nil {
						catchValue = StringValue(state.pendingErr.Error())
					}
					catchParser := *p
					catchParser.env = catchEnv
					if err := catchParser.bindBindingPattern(state.catchPattern, catchValue, "let"); err != nil {
						return UndefinedValue(), nil, err
					}
				}
				catchBlock, err := newClassicJSBlockState(state.catchSource, catchEnv, state)
				if err != nil {
					return UndefinedValue(), nil, err
				}
				state.catchBlock = catchBlock
			}

			value, nextBlock, err := p.resumeBlockState(state.catchBlock)
			if nextBlock != nil {
				state.catchBlock = nextBlock
				return value, state, nil
			}
			if err != nil {
				if throwValue, ok := classicJSThrowSignalValue(err); ok {
					state.pendingThrow = throwValue
					state.hasPendingThrow = true
					state.pendingErr = NewError(ErrorKindRuntime, ToJSString(throwValue))
				} else {
					state.hasPendingThrow = false
					state.pendingThrow = UndefinedValue()
					state.pendingErr = err
				}
				if _, ok := classicJSReturnSignalValue(err); ok || classicJSBreakSignalValue(err) || classicJSContinueSignalValue(err) {
					if state.hasFinally {
						state.stage = classicJSTryStageFinally
						continue
					}
					state.stage = classicJSTryStageDone
					return p.finalizeTryCompletion(state)
				}
				if state.hasFinally {
					state.stage = classicJSTryStageFinally
					continue
				}
				state.stage = classicJSTryStageDone
				return p.finalizeTryCompletion(state)
			}
			state.result = value
			state.catchBlock = nil
			state.pendingErr = nil
			state.pendingThrow = UndefinedValue()
			state.hasPendingThrow = false
			if state.hasFinally {
				state.stage = classicJSTryStageFinally
				continue
			}
			state.stage = classicJSTryStageDone
			return state.result, nil, nil

		case classicJSTryStageFinally:
			if !state.hasFinally {
				state.stage = classicJSTryStageDone
				continue
			}

			value, nextBlock, err := p.resumeBlockState(state.finallyBlock)
			if nextBlock != nil {
				state.finallyBlock = nextBlock
				return value, state, nil
			}
			if err != nil {
				if throwValue, ok := classicJSThrowSignalValue(err); ok {
					state.pendingThrow = throwValue
					state.hasPendingThrow = true
					state.pendingErr = NewError(ErrorKindRuntime, ToJSString(throwValue))
				} else {
					state.hasPendingThrow = false
					state.pendingThrow = UndefinedValue()
					state.pendingErr = err
				}
				state.stage = classicJSTryStageDone
				return p.finalizeTryCompletion(state)
			}
			state.finallyBlock = nil
			state.stage = classicJSTryStageDone
			return p.finalizeTryCompletion(state)

		case classicJSTryStageDone:
			return p.finalizeTryCompletion(state)

		default:
			return UndefinedValue(), nil, NewError(ErrorKindRuntime, "unsupported try state stage")
		}
	}
}

func (p *classicJSStatementParser) finalizeTryCompletion(state *classicJSTryState) (Value, *classicJSTryState, error) {
	if state == nil {
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "try state is unavailable")
	}
	if state.pendingErr != nil {
		if label, ok := classicJSBreakSignalLabel(state.pendingErr); ok && label != "" && label == state.label {
			return UndefinedValue(), nil, nil
		}
		if label, ok := classicJSContinueSignalLabel(state.pendingErr); ok && label != "" && label == state.label {
			return UndefinedValue(), nil, NewError(ErrorKindParse, "continue statements cannot target labeled try statements in this bounded classic-JS slice")
		}
		return UndefinedValue(), nil, state.pendingErr
	}
	return state.result, nil, nil
}

func (p *classicJSStatementParser) parseCatchBinding() (classicJSBindingPattern, bool, error) {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return classicJSBindingPattern{}, false, nil
	}

	p.skipSpaceAndComments()
	pattern, err := p.parseBindingPattern()
	if err != nil {
		return classicJSBindingPattern{}, false, err
	}

	p.skipSpaceAndComments()
	if !p.consumeByte(')') {
		return classicJSBindingPattern{}, false, NewError(ErrorKindParse, "unterminated `catch` binding")
	}
	return pattern, true, nil
}

func (p *classicJSStatementParser) consumeBlockSource() (string, error) {
	p.skipSpaceAndComments()
	if !p.consumeByte('{') {
		return "", NewError(ErrorKindParse, "expected `{` to start a block")
	}

	start := p.pos
	depth := 1
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	for !p.eof() {
		ch := p.peekByte()
		if lineComment {
			p.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
				p.pos += 2
				blockComment = false
				continue
			}
			p.pos++
			continue
		}
		if quote != 0 {
			p.pos++
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

		switch ch {
		case '\'', '"', '`':
			quote = ch
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				p.pos++
				continue
			}
			switch p.source[p.pos+1] {
			case '/':
				lineComment = true
				p.pos += 2
			case '*':
				blockComment = true
				p.pos += 2
			default:
				p.pos++
			}
		case '{':
			depth++
			p.pos++
		case '}':
			depth--
			if depth == 0 {
				block := strings.TrimSpace(p.source[start:p.pos])
				p.pos++
				return block, nil
			}
			p.pos++
		default:
			p.pos++
		}
	}

	if quote != 0 {
		return "", NewError(ErrorKindParse, "unterminated quoted string in block statement")
	}
	if blockComment {
		return "", NewError(ErrorKindParse, "unterminated block comment in block statement")
	}
	return "", NewError(ErrorKindParse, "unterminated block statement")
}

func (p *classicJSStatementParser) consumeParenthesizedSource(label string) (string, error) {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return "", NewError(ErrorKindParse, fmt.Sprintf("expected `(` after `%s`", label))
	}

	start := p.pos
	depth := 1
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	for !p.eof() {
		ch := p.peekByte()
		if lineComment {
			p.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
				p.pos += 2
				blockComment = false
				continue
			}
			p.pos++
			continue
		}
		if quote != 0 {
			p.pos++
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

		switch ch {
		case '\'', '"', '`':
			quote = ch
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				p.pos++
				continue
			}
			switch p.source[p.pos+1] {
			case '/':
				lineComment = true
				p.pos += 2
			case '*':
				blockComment = true
				p.pos += 2
			default:
				p.pos++
			}
		case '(':
			depth++
			p.pos++
		case ')':
			depth--
			if depth == 0 {
				inner := strings.TrimSpace(p.source[start:p.pos])
				p.pos++
				return inner, nil
			}
			p.pos++
		default:
			p.pos++
		}
	}

	if quote != 0 {
		return "", NewError(ErrorKindParse, "unterminated quoted string in parenthesized expression")
	}
	if blockComment {
		return "", NewError(ErrorKindParse, "unterminated block comment in parenthesized expression")
	}
	return "", NewError(ErrorKindParse, fmt.Sprintf("unterminated parenthesized expression after `%s`", label))
}

func (p *classicJSStatementParser) consumeTemplateInterpolationSource() (string, error) {
	start := p.pos
	depth := 1
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool

	for !p.eof() {
		ch := p.peekByte()
		if lineComment {
			p.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
				p.pos += 2
				blockComment = false
				continue
			}
			p.pos++
			continue
		}
		if quote != 0 {
			p.pos++
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

		switch ch {
		case '\'', '"', '`':
			quote = ch
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				p.pos++
				continue
			}
			switch p.source[p.pos+1] {
			case '/':
				lineComment = true
				p.pos += 2
			case '*':
				blockComment = true
				p.pos += 2
			default:
				p.pos++
			}
		case '{':
			depth++
			p.pos++
		case '}':
			depth--
			if depth == 0 {
				source := strings.TrimSpace(p.source[start:p.pos])
				p.pos++
				return source, nil
			}
			p.pos++
		default:
			p.pos++
		}
	}

	if quote != 0 {
		return "", NewError(ErrorKindParse, "unterminated quoted string in template interpolation")
	}
	if blockComment {
		return "", NewError(ErrorKindParse, "unterminated block comment in template interpolation")
	}
	return "", NewError(ErrorKindParse, "unterminated template interpolation")
}

func (p *classicJSStatementParser) consumeArrowFunctionExpressionSource() (string, error) {
	start := p.pos
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int

	for !p.eof() {
		ch := p.peekByte()
		if lineComment {
			p.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
				p.pos += 2
				blockComment = false
				continue
			}
			p.pos++
			continue
		}
		if quote != 0 {
			p.pos++
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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
			switch ch {
			case ',', ')', ']', '}', ';':
				source := strings.TrimSpace(p.source[start:p.pos])
				return source, nil
			}
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				p.pos++
				continue
			}
			switch p.source[p.pos+1] {
			case '/':
				lineComment = true
				p.pos += 2
			case '*':
				blockComment = true
				p.pos += 2
			default:
				p.pos++
			}
		case '(':
			parenDepth++
			p.pos++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			p.pos++
		case '{':
			braceDepth++
			p.pos++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			p.pos++
		case '[':
			bracketDepth++
			p.pos++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			p.pos++
		default:
			p.pos++
		}
	}

	if quote != 0 {
		return "", NewError(ErrorKindParse, "unterminated quoted string in arrow function body")
	}
	if blockComment {
		return "", NewError(ErrorKindParse, "unterminated block comment in arrow function body")
	}
	return strings.TrimSpace(p.source[start:p.pos]), nil
}

func splitClassicJSForHeader(source string) (string, string, string, error) {
	text := strings.TrimSpace(source)
	if text == "" {
		return "", "", "", NewError(ErrorKindParse, "expected two `;` separators in `for` header")
	}

	segments := make([]string, 0, 3)
	start := 0
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int
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

		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '/':
			if i+1 < len(text) {
				switch text[i+1] {
				case '/':
					lineComment = true
					i++
				case '*':
					blockComment = true
					i++
				}
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ';':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				segments = append(segments, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}

	if quote != 0 {
		return "", "", "", NewError(ErrorKindParse, "unterminated quoted string in for header")
	}
	if blockComment {
		return "", "", "", NewError(ErrorKindParse, "unterminated block comment in for header")
	}

	segments = append(segments, strings.TrimSpace(text[start:]))
	if len(segments) != 3 {
		return "", "", "", NewError(ErrorKindParse, "expected two `;` separators in `for` header")
	}
	return segments[0], segments[1], segments[2], nil
}

func splitClassicJSForKeywordHeader(source, keyword string) (string, string, bool, error) {
	text := strings.TrimSpace(source)
	if text == "" {
		return "", "", false, nil
	}
	if keyword == "" {
		return "", "", false, NewError(ErrorKindParse, "for header keyword is empty")
	}

	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int
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

		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '/':
			if i+1 < len(text) {
				switch text[i+1] {
				case '/':
					lineComment = true
					i++
				case '*':
					blockComment = true
					i++
				}
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		default:
		}

		if parenDepth != 0 || braceDepth != 0 || bracketDepth != 0 {
			continue
		}
		if i+len(keyword) > len(text) || text[i:i+len(keyword)] != keyword {
			continue
		}
		if i > 0 && isIdentPart(text[i-1]) {
			continue
		}
		if i+len(keyword) < len(text) && isIdentPart(text[i+len(keyword)]) {
			continue
		}
		left := strings.TrimSpace(text[:i])
		right := strings.TrimSpace(text[i+len(keyword):])
		if left == "" || right == "" {
			return "", "", false, NewError(ErrorKindParse, fmt.Sprintf("invalid `for...%s` header", keyword))
		}
		return left, right, true, nil
	}

	if quote != 0 {
		return "", "", false, NewError(ErrorKindParse, "unterminated quoted string in for header")
	}
	if blockComment {
		return "", "", false, NewError(ErrorKindParse, "unterminated block comment in for header")
	}
	return "", "", false, nil
}

func splitClassicJSForOfHeader(source string) (string, string, bool, error) {
	return splitClassicJSForKeywordHeader(source, "of")
}

func splitClassicJSForInHeader(source string) (string, string, bool, error) {
	return splitClassicJSForKeywordHeader(source, "in")
}

func parseClassicJSForBinding(source, keyword string, allowAwaitUsing bool) (string, classicJSBindingPattern, error) {
	parser := &classicJSStatementParser{
		source: strings.TrimSpace(source),
	}
	if parser.source == "" {
		return "", classicJSBindingPattern{}, NewError(ErrorKindParse, fmt.Sprintf("expected binding declaration in `for...%s` header", keyword))
	}
	if !allowAwaitUsing {
		start := parser.pos
		if awaitKeyword, ok := parser.peekKeyword("await"); ok {
			parser.pos += len(awaitKeyword)
			parser.skipSpaceAndComments()
			if _, ok := parser.peekKeyword("using"); ok {
				return "", classicJSBindingPattern{}, NewError(ErrorKindParse, "`await using` bindings are only supported in `for await...of` headers in this bounded classic-JS slice")
			}
			parser.pos = start
		}
	}

	kind := ""
	switch {
	case allowAwaitUsing && func() bool {
		_, ok := parser.peekKeyword("await")
		return ok
	}():
		start := parser.pos
		parser.pos += len("await")
		parser.skipSpaceAndComments()
		if usingKeyword, ok := parser.peekKeyword("using"); ok {
			parser.pos += len(usingKeyword)
			kind = "await using"
		} else {
			parser.pos = start
		}
	case func() bool {
		_, ok := parser.peekKeyword("using")
		return ok
	}():
		parser.pos += len("using")
		kind = "using"
	case func() bool {
		_, ok := parser.peekKeyword("let")
		return ok
	}():
		parser.pos += len("let")
		kind = "let"
	case func() bool {
		_, ok := parser.peekKeyword("const")
		return ok
	}():
		parser.pos += len("const")
		kind = "const"
	case func() bool {
		_, ok := parser.peekKeyword("var")
		return ok
	}():
		parser.pos += len("var")
		kind = "var"
	default:
		if allowAwaitUsing {
			return "", classicJSBindingPattern{}, NewError(ErrorKindUnsupported, fmt.Sprintf("for...%s bindings must use `await using`, `using`, `let`, `const`, or `var` in this bounded classic-JS slice", keyword))
		}
		return "", classicJSBindingPattern{}, NewError(ErrorKindUnsupported, fmt.Sprintf("for...%s bindings must use `using`, `let`, `const`, or `var` in this bounded classic-JS slice", keyword))
	}

	pattern, err := parser.parseBindingPattern()
	if err != nil {
		return "", classicJSBindingPattern{}, err
	}
	parser.skipSpaceAndComments()
	if !parser.eof() {
		return "", classicJSBindingPattern{}, NewError(ErrorKindParse, fmt.Sprintf("for...%s binding declarations must not include an initializer", keyword))
	}
	return kind, pattern, nil
}

func parseClassicJSForOfBinding(source string, allowAwaitUsing bool) (string, classicJSBindingPattern, error) {
	return parseClassicJSForBinding(source, "of", allowAwaitUsing)
}

func parseClassicJSForInBinding(source string) (string, classicJSBindingPattern, error) {
	return parseClassicJSForBinding(source, "in", false)
}

func classicJSForInKeys(value Value) ([]Value, error) {
	switch value.Kind {
	case ValueKindObject:
		keys := make([]Value, 0, len(value.Object))
		for _, entry := range value.Object {
			if classicJSIsInternalObjectKey(entry.Key) {
				continue
			}
			keys = append(keys, StringValue(entry.Key))
		}
		return keys, nil
	case ValueKindArray:
		keys := make([]Value, 0, len(value.Array))
		for i := range value.Array {
			keys = append(keys, StringValue(strconv.Itoa(i)))
		}
		return keys, nil
	case ValueKindString:
		keys := make([]Value, 0, len(value.String))
		index := 0
		for range value.String {
			keys = append(keys, StringValue(strconv.Itoa(index)))
			index++
		}
		return keys, nil
	default:
		return nil, NewError(ErrorKindRuntime, "for...in loops require a string, object, or array value on the right in this bounded classic-JS slice")
	}
}

func parseClassicJSClassMemberName(scanner *classicJSStatementParser) (string, string, bool, error) {
	scanner.skipSpaceAndComments()
	if scanner.eof() {
		return "", "", false, NewError(ErrorKindParse, "unexpected end of class body")
	}

	switch scanner.peekByte() {
	case '[':
		scanner.pos++
		nameSource, err := scanner.consumeBracketAccessExpressionSource()
		if err != nil {
			return "", "", false, err
		}
		if strings.TrimSpace(nameSource) == "" {
			return "", "", false, NewError(ErrorKindParse, "computed class field name requires an expression")
		}
		return "", nameSource, false, nil
	case '#':
		scanner.pos++
		name, err := scanner.parseIdentifier()
		if err != nil {
			return "", "", false, NewError(ErrorKindParse, "private class field name requires an identifier")
		}
		return name, "", true, nil
	case '*':
		return "", "", false, NewError(ErrorKindParse, "generator class methods require a class member name in this bounded classic-JS slice")
	}

	name, err := scanner.parseIdentifier()
	if err != nil {
		return "", "", false, NewError(ErrorKindParse, fmt.Sprintf("unexpected class body element at %q in this bounded classic-JS slice", scanner.remainingPreview()))
	}
	return name, "", false, nil
}

func tryParseClassicJSClassGetterMember(scanner *classicJSStatementParser, static bool) (classicJSClassMember, bool, error) {
	if scanner == nil {
		return classicJSClassMember{}, false, nil
	}

	lookahead := *scanner
	keyword, ok := lookahead.peekKeyword("get")
	if !ok {
		return classicJSClassMember{}, false, nil
	}
	lookahead.pos += len(keyword)
	lookahead.skipSpaceAndComments()
	if lookahead.eof() {
		return classicJSClassMember{}, false, nil
	}
	switch ch := lookahead.peekByte(); ch {
	case '#':
	case '[', '\'', '"':
	default:
		if !isIdentStart(ch) && !isDigit(ch) {
			return classicJSClassMember{}, false, nil
		}
	}

	fieldName, fieldNameSource, privateName, err := parseClassicJSClassMemberName(&lookahead)
	if err != nil {
		return classicJSClassMember{}, false, err
	}

	lookahead.skipSpaceAndComments()
	paramsSource, err := lookahead.consumeParenthesizedSource("class getter")
	if err != nil {
		return classicJSClassMember{}, false, err
	}
	if strings.TrimSpace(paramsSource) != "" {
		return classicJSClassMember{}, false, NewError(ErrorKindParse, "class getter accessors do not accept parameters in this bounded classic-JS slice")
	}

	bodySource, err := lookahead.consumeBlockSource()
	if err != nil {
		return classicJSClassMember{}, false, err
	}

	scanner.pos = lookahead.pos
	member := classicJSClassMember{
		methodName:       fieldName,
		methodNameSource: fieldNameSource,
		methodBody:       bodySource,
		private:          privateName,
	}
	if static {
		member.kind = "static-getter"
	} else {
		member.kind = "instance-getter"
	}
	return member, true, nil
}

func tryParseClassicJSClassSetterMember(scanner *classicJSStatementParser, static bool) (classicJSClassMember, bool, error) {
	if scanner == nil {
		return classicJSClassMember{}, false, nil
	}

	lookahead := *scanner
	keyword, ok := lookahead.peekKeyword("set")
	if !ok {
		return classicJSClassMember{}, false, nil
	}
	lookahead.pos += len(keyword)
	lookahead.skipSpaceAndComments()
	if lookahead.eof() {
		return classicJSClassMember{}, false, nil
	}
	switch ch := lookahead.peekByte(); ch {
	case '#':
	case '[', '\'', '"':
	default:
		if !isIdentStart(ch) && !isDigit(ch) {
			return classicJSClassMember{}, false, nil
		}
	}

	fieldName, fieldNameSource, privateName, err := parseClassicJSClassMemberName(&lookahead)
	if err != nil {
		return classicJSClassMember{}, false, err
	}

	lookahead.skipSpaceAndComments()
	paramsSource, err := lookahead.consumeParenthesizedSource("class setter")
	if err != nil {
		return classicJSClassMember{}, false, err
	}
	params, err := parseClassicJSSetterParameters(paramsSource, "class setter")
	if err != nil {
		return classicJSClassMember{}, false, err
	}

	bodySource, err := lookahead.consumeBlockSource()
	if err != nil {
		return classicJSClassMember{}, false, err
	}

	scanner.pos = lookahead.pos
	member := classicJSClassMember{
		methodName:       fieldName,
		methodNameSource: fieldNameSource,
		methodParams:     params,
		methodBody:       bodySource,
		private:          privateName,
	}
	if static {
		member.kind = "static-setter"
	} else {
		member.kind = "instance-setter"
	}
	return member, true, nil
}

func tryConsumeClassicJSAsyncClassMethodPrefix(scanner *classicJSStatementParser) (bool, error) {
	scanner.skipSpaceAndComments()
	keyword, ok := scanner.peekKeyword("async")
	if !ok {
		return false, nil
	}

	lookahead := *scanner
	lookahead.pos += len(keyword)
	lookahead.skipSpaceAndComments()
	if lookahead.eof() {
		return false, nil
	}
	if lookahead.peekByte() == '*' {
		lookahead.pos++
		lookahead.skipSpaceAndComments()
	}

	if _, _, _, err := parseClassicJSClassMemberName(&lookahead); err != nil {
		return false, nil
	}
	lookahead.skipSpaceAndComments()
	if lookahead.peekByte() != '(' {
		return false, nil
	}

	scanner.pos += len(keyword)
	scanner.skipSpaceAndComments()
	return true, nil
}

func splitClassicJSClassMembers(source string) ([]classicJSClassMember, error) {
	scanner := &classicJSStatementParser{
		source: strings.TrimSpace(source),
	}
	if scanner.source == "" {
		return nil, nil
	}

	members := make([]classicJSClassMember, 0, 4)
	for {
		scanner.skipSpaceAndComments()
		for scanner.consumeByte(';') {
			scanner.skipSpaceAndComments()
		}
		if scanner.eof() {
			break
		}

		if keyword, ok := scanner.peekKeyword("static"); ok {
			scanner.pos += len(keyword)
			scanner.skipSpaceAndComments()
			if scanner.eof() {
				return nil, NewError(ErrorKindParse, "unexpected end of class body after `static`")
			}

			switch scanner.peekByte() {
			case '{':
				blockSource, err := scanner.consumeBlockSource()
				if err != nil {
					return nil, err
				}
				members = append(members, classicJSClassMember{kind: "static-block", staticBlock: blockSource})
				continue
			}

			if member, ok, err := tryParseClassicJSClassGetterMember(scanner, true); err != nil {
				return nil, err
			} else if ok {
				members = append(members, member)
				continue
			}

			if member, ok, err := tryParseClassicJSClassSetterMember(scanner, true); err != nil {
				return nil, err
			} else if ok {
				members = append(members, member)
				continue
			}

			memberAsync, err := tryConsumeClassicJSAsyncClassMethodPrefix(scanner)
			if err != nil {
				return nil, err
			}
			memberGenerator := false
			if scanner.peekByte() == '*' {
				scanner.pos++
				scanner.skipSpaceAndComments()
				memberGenerator = true
			}

			fieldName, fieldNameSource, privateName, err := parseClassicJSClassMemberName(scanner)
			if err != nil {
				return nil, err
			}

			scanner.skipSpaceAndComments()
			if scanner.peekByte() == '(' {
				paramsSource, err := scanner.consumeParenthesizedSource("class method")
				if err != nil {
					return nil, err
				}
				params, restName, err := parseClassicJSFunctionParameters(paramsSource, "class method")
				if err != nil {
					return nil, err
				}
				bodySource, err := scanner.consumeBlockSource()
				if err != nil {
					return nil, err
				}
				members = append(members, classicJSClassMember{
					kind:             "static-method",
					methodName:       fieldName,
					methodNameSource: fieldNameSource,
					methodParams:     params,
					methodRestName:   restName,
					methodBody:       bodySource,
					private:          privateName,
					async:            memberAsync,
					generator:        memberGenerator,
				})
				continue
			}
			if scanner.consumeByte('=') {
				scanner.skipSpaceAndComments()
				if scanner.eof() {
					return nil, NewError(ErrorKindParse, "class field initializer requires an expression")
				}
				initStart := scanner.pos
				initEnd, err := scanClassicJSClassMemberTerminator(scanner)
				if err != nil {
					return nil, err
				}
				fieldInit := strings.TrimSpace(scanner.source[initStart:initEnd])
				if fieldInit == "" {
					return nil, NewError(ErrorKindParse, "class field initializer requires an expression")
				}
				members = append(members, classicJSClassMember{kind: "static-field", fieldName: fieldName, fieldNameSource: fieldNameSource, fieldInit: fieldInit, private: privateName})
				if scanner.consumeByte(';') {
					continue
				}
				if scanner.eof() {
					continue
				}
				return nil, NewError(ErrorKindParse, "expected `;` after static class field initializer")
			}

			if scanner.consumeByte(';') {
				members = append(members, classicJSClassMember{kind: "static-field", fieldName: fieldName, fieldNameSource: fieldNameSource, private: privateName})
				continue
			}
			if scanner.eof() {
				members = append(members, classicJSClassMember{kind: "static-field", fieldName: fieldName, fieldNameSource: fieldNameSource, private: privateName})
				continue
			}

			return nil, NewError(ErrorKindParse, fmt.Sprintf("unexpected class member syntax at %q in this bounded classic-JS slice", scanner.remainingPreview()))
		}

		memberAsync, err := tryConsumeClassicJSAsyncClassMethodPrefix(scanner)
		if err != nil {
			return nil, err
		}
		memberGenerator := false
		if scanner.peekByte() == '*' {
			scanner.pos++
			scanner.skipSpaceAndComments()
			memberGenerator = true
		}

		if member, ok, err := tryParseClassicJSClassGetterMember(scanner, false); err != nil {
			return nil, err
		} else if ok {
			members = append(members, member)
			continue
		}

		if member, ok, err := tryParseClassicJSClassSetterMember(scanner, false); err != nil {
			return nil, err
		} else if ok {
			members = append(members, member)
			continue
		}

		name, nameSource, privateName, err := parseClassicJSClassMemberName(scanner)
		if err != nil {
			return nil, err
		}

		scanner.skipSpaceAndComments()
		if scanner.peekByte() == '(' {
			paramsSource, err := scanner.consumeParenthesizedSource("class method")
			if err != nil {
				return nil, err
			}
			params, restName, err := parseClassicJSFunctionParameters(paramsSource, "class method")
			if err != nil {
				return nil, err
			}
			bodySource, err := scanner.consumeBlockSource()
			if err != nil {
				return nil, err
			}
			members = append(members, classicJSClassMember{
				kind:             "instance-method",
				methodName:       name,
				methodNameSource: nameSource,
				methodParams:     params,
				methodRestName:   restName,
				methodBody:       bodySource,
				private:          privateName,
				async:            memberAsync,
				generator:        memberGenerator,
			})
			continue
		}

		if scanner.consumeByte('=') {
			scanner.skipSpaceAndComments()
			if scanner.eof() {
				return nil, NewError(ErrorKindParse, "class field initializer requires an expression")
			}
			initStart := scanner.pos
			initEnd, err := scanClassicJSClassMemberTerminator(scanner)
			if err != nil {
				return nil, err
			}
			fieldInit := strings.TrimSpace(scanner.source[initStart:initEnd])
			if fieldInit == "" {
				return nil, NewError(ErrorKindParse, "class field initializer requires an expression")
			}
			members = append(members, classicJSClassMember{kind: "instance-field", fieldName: name, fieldNameSource: nameSource, fieldInit: fieldInit, private: privateName})
			if scanner.consumeByte(';') {
				continue
			}
			if scanner.eof() {
				continue
			}
			return nil, NewError(ErrorKindParse, "expected `;` after class field initializer")
		}

		if scanner.consumeByte(';') {
			members = append(members, classicJSClassMember{kind: "instance-field", fieldName: name, fieldNameSource: nameSource, private: privateName})
			continue
		}
		if scanner.eof() {
			members = append(members, classicJSClassMember{kind: "instance-field", fieldName: name, fieldNameSource: nameSource, private: privateName})
			continue
		}

		return nil, NewError(ErrorKindParse, fmt.Sprintf("unexpected class body element at %q in this bounded classic-JS slice", scanner.remainingPreview()))
	}

	return members, nil
}

func parseClassicJSFunctionParameters(source string, label string) ([]classicJSFunctionParameter, string, error) {
	parser := &classicJSStatementParser{
		source: strings.TrimSpace(source),
	}
	if parser.source == "" {
		return nil, "", nil
	}

	params := make([]classicJSFunctionParameter, 0, 4)
	restName := ""
	for {
		parser.skipSpaceAndComments()
		if parser.eof() {
			break
		}

		if parser.consumeEllipsis() {
			parser.skipSpaceAndComments()
			name, err := parser.parseIdentifier()
			if err != nil {
				return nil, "", err
			}
			if isClassicJSReservedDeclarationName(name) {
				return nil, "", NewError(ErrorKindParse, fmt.Sprintf("reserved %s parameter name %q is not allowed in this bounded classic-JS slice", label, name))
			}
			restName = name
			parser.skipSpaceAndComments()
			if !parser.eof() {
				return nil, "", NewError(ErrorKindParse, fmt.Sprintf("%s rest parameter must be the final parameter in this bounded classic-JS slice", label))
			}
			break
		}

		param := classicJSFunctionParameter{}
		switch parser.peekByte() {
		case '[', '{':
			pattern, err := parser.parseBindingPattern()
			if err != nil {
				return nil, "", err
			}
			param.pattern = pattern
			param.hasPattern = true
		case ':':
			return nil, "", NewError(ErrorKindParse, fmt.Sprintf("unsupported %s parameter syntax in this bounded classic-JS slice", label))
		default:
			name, err := parser.parseIdentifier()
			if err != nil {
				return nil, "", err
			}
			if isClassicJSReservedDeclarationName(name) {
				return nil, "", NewError(ErrorKindParse, fmt.Sprintf("reserved %s parameter name %q is not allowed in this bounded classic-JS slice", label, name))
			}
			param.name = name
		}

		parser.skipSpaceAndComments()
		if parser.consumeByte('=') {
			parser.skipSpaceAndComments()
			if parser.eof() {
				return nil, "", NewError(ErrorKindParse, fmt.Sprintf("%s parameter default requires an expression", label))
			}
			defaultStart := parser.pos
			defaultEnd, err := scanClassicJSParameterDefaultTerminator(parser)
			if err != nil {
				return nil, "", err
			}
			defaultSource := strings.TrimSpace(parser.source[defaultStart:defaultEnd])
			if defaultSource == "" {
				return nil, "", NewError(ErrorKindParse, fmt.Sprintf("%s parameter default requires an expression", label))
			}
			param.defaultSource = defaultSource
			parser.pos = defaultEnd
		}

		params = append(params, param)

		parser.skipSpaceAndComments()
		if parser.eof() {
			break
		}
		if !parser.consumeByte(',') {
			return nil, "", NewError(ErrorKindParse, fmt.Sprintf("%s parameter list must separate parameters with commas", label))
		}
		parser.skipSpaceAndComments()
		if parser.eof() {
			break
		}
	}

	return params, restName, nil
}

func parseClassicJSSetterParameters(source string, label string) ([]classicJSFunctionParameter, error) {
	params, restName, err := parseClassicJSFunctionParameters(source, label)
	if err != nil {
		return nil, err
	}
	if len(params) != 1 || restName != "" || params[0].defaultSource != "" {
		return nil, NewError(ErrorKindParse, fmt.Sprintf("%s accessors must accept exactly one parameter in this bounded classic-JS slice", label))
	}
	return params, nil
}

func scanClassicJSParameterDefaultTerminator(scanner *classicJSStatementParser) (int, error) {
	return scanClassicJSDelimitedExpressionTerminator(scanner, ',')
}

func scanClassicJSBindingPatternDefaultTerminator(scanner *classicJSStatementParser, closing byte) (int, error) {
	return scanClassicJSDelimitedExpressionTerminator(scanner, closing)
}

func scanClassicJSDelimitedExpressionTerminator(scanner *classicJSStatementParser, closing byte) (int, error) {
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int

	for !scanner.eof() {
		ch := scanner.peekByte()
		if lineComment {
			scanner.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && scanner.pos+1 < len(scanner.source) && scanner.source[scanner.pos+1] == '/' {
				scanner.pos += 2
				blockComment = false
				continue
			}
			scanner.pos++
			continue
		}
		if quote != 0 {
			scanner.pos++
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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && (ch == ',' || ch == closing) {
			return scanner.pos, nil
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
			scanner.pos++
		case '/':
			if scanner.pos+1 >= len(scanner.source) {
				scanner.pos++
				continue
			}
			switch scanner.source[scanner.pos+1] {
			case '/':
				lineComment = true
				scanner.pos += 2
			case '*':
				blockComment = true
				scanner.pos += 2
			default:
				scanner.pos++
			}
		case '(':
			parenDepth++
			scanner.pos++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			scanner.pos++
		case '{':
			braceDepth++
			scanner.pos++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			scanner.pos++
		case '[':
			bracketDepth++
			scanner.pos++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			scanner.pos++
		default:
			scanner.pos++
		}
	}

	if quote != 0 {
		return scanner.pos, NewError(ErrorKindParse, "unterminated quoted string in delimited expression")
	}
	if blockComment {
		return scanner.pos, NewError(ErrorKindParse, "unterminated block comment in delimited expression")
	}
	return scanner.pos, nil
}

func scanClassicJSClassMemberTerminator(scanner *classicJSStatementParser) (int, error) {
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int

	for !scanner.eof() {
		ch := scanner.peekByte()
		if lineComment {
			scanner.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && scanner.pos+1 < len(scanner.source) && scanner.source[scanner.pos+1] == '/' {
				scanner.pos += 2
				blockComment = false
				continue
			}
			scanner.pos++
			continue
		}
		if quote != 0 {
			scanner.pos++
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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && ch == ';' {
			return scanner.pos, nil
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
			scanner.pos++
		case '/':
			if scanner.pos+1 >= len(scanner.source) {
				scanner.pos++
				continue
			}
			switch scanner.source[scanner.pos+1] {
			case '/':
				lineComment = true
				scanner.pos += 2
			case '*':
				blockComment = true
				scanner.pos += 2
			default:
				scanner.pos++
			}
		case '(':
			parenDepth++
			scanner.pos++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			scanner.pos++
		case '{':
			braceDepth++
			scanner.pos++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			scanner.pos++
		case '[':
			bracketDepth++
			scanner.pos++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			scanner.pos++
		default:
			scanner.pos++
		}
	}

	if quote != 0 {
		return scanner.pos, NewError(ErrorKindParse, "unterminated quoted string in class field initializer")
	}
	if blockComment {
		return scanner.pos, NewError(ErrorKindParse, "unterminated block comment in class field initializer")
	}
	return scanner.pos, nil
}

func splitClassicJSSwitchClauses(source string) ([]classicJSSwitchClause, error) {
	scanner := &classicJSStatementParser{
		source: strings.TrimSpace(source),
	}
	if scanner.source == "" {
		return nil, nil
	}

	clauses := make([]classicJSSwitchClause, 0, 4)
	sawDefault := false
	for {
		scanner.skipSpaceAndComments()
		for scanner.consumeByte(';') {
			scanner.skipSpaceAndComments()
		}
		if scanner.eof() {
			break
		}

		switch {
		case func() bool {
			_, ok := scanner.peekKeyword("case")
			return ok
		}():
			scanner.pos += len("case")
			scanner.skipSpaceAndComments()
			labelStart := scanner.pos
			labelEnd, err := scanClassicJSClauseTerminator(scanner, true)
			if err != nil {
				return nil, err
			}
			if labelEnd <= labelStart {
				return nil, NewError(ErrorKindParse, "invalid switch case label")
			}
			label := strings.TrimSpace(scanner.source[labelStart:labelEnd])
			scanner.pos = labelEnd
			if !scanner.consumeByte(':') {
				return nil, NewError(ErrorKindParse, "expected `:` after `case` label")
			}
			bodyStart := scanner.pos
			bodyEnd, err := scanClassicJSClauseTerminator(scanner, false)
			if err != nil {
				return nil, err
			}
			body := strings.TrimSpace(scanner.source[bodyStart:bodyEnd])
			clauses = append(clauses, classicJSSwitchClause{kind: "case", label: label, body: body})
			scanner.pos = bodyEnd

		case func() bool {
			_, ok := scanner.peekKeyword("default")
			return ok
		}():
			if sawDefault {
				return nil, NewError(ErrorKindParse, "duplicate `default` clause in switch statement")
			}
			sawDefault = true
			scanner.pos += len("default")
			scanner.skipSpaceAndComments()
			if !scanner.consumeByte(':') {
				return nil, NewError(ErrorKindParse, "expected `:` after `default`")
			}
			bodyStart := scanner.pos
			bodyEnd, err := scanClassicJSClauseTerminator(scanner, false)
			if err != nil {
				return nil, err
			}
			body := strings.TrimSpace(scanner.source[bodyStart:bodyEnd])
			clauses = append(clauses, classicJSSwitchClause{kind: "default", body: body})
			scanner.pos = bodyEnd

		default:
			return nil, NewError(ErrorKindParse, fmt.Sprintf("expected `case` or `default` in switch body at %q", scanner.remainingPreview()))
		}
	}

	return clauses, nil
}

func scanClassicJSClauseTerminator(scanner *classicJSStatementParser, stopAtColon bool) (int, error) {
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int

	for !scanner.eof() {
		ch := scanner.peekByte()
		if lineComment {
			scanner.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && scanner.pos+1 < len(scanner.source) && scanner.source[scanner.pos+1] == '/' {
				scanner.pos += 2
				blockComment = false
				continue
			}
			scanner.pos++
			continue
		}
		if quote != 0 {
			scanner.pos++
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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
			if stopAtColon {
				if ch == ':' {
					return scanner.pos, nil
				}
			} else {
				if _, ok := scanner.peekKeyword("case"); ok {
					return scanner.pos, nil
				}
				if _, ok := scanner.peekKeyword("default"); ok {
					return scanner.pos, nil
				}
			}
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
			scanner.pos++
		case '/':
			if scanner.pos+1 >= len(scanner.source) {
				scanner.pos++
				continue
			}
			switch scanner.source[scanner.pos+1] {
			case '/':
				lineComment = true
				scanner.pos += 2
			case '*':
				blockComment = true
				scanner.pos += 2
			default:
				scanner.pos++
			}
		case '(':
			parenDepth++
			scanner.pos++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			scanner.pos++
		case '{':
			braceDepth++
			scanner.pos++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			scanner.pos++
		case '[':
			bracketDepth++
			scanner.pos++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			scanner.pos++
		default:
			scanner.pos++
		}
	}

	return scanner.pos, nil
}

func isClassicJSBreakStatement(source string) bool {
	trimmed := strings.TrimSpace(source)
	for strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	return trimmed == "break"
}

func classicJSSwitchMatches(discriminant Value, candidate Value) (bool, error) {
	return classicJSSameValue(discriminant, candidate), nil
}

func classicJSTypeofJSValue(value jsValue) string {
	switch value.kind {
	case jsValueHostMethod, jsValueBuiltinExpr:
		return "function"
	case jsValueHostObject:
		return "object"
	case jsValueSuper:
		return classicJSTypeofValue(value.value)
	case jsValueScalar:
		return classicJSTypeofValue(value.value)
	default:
		return "undefined"
	}
}

func classicJSTypeofValue(value Value) string {
	switch value.Kind {
	case ValueKindUndefined:
		return "undefined"
	case ValueKindNull:
		return "object"
	case ValueKindString:
		return "string"
	case ValueKindBool:
		return "boolean"
	case ValueKindNumber:
		return "number"
	case ValueKindBigInt:
		return "bigint"
	case ValueKindArray, ValueKindObject, ValueKindPromise, ValueKindInvocation:
		return "object"
	case ValueKindFunction:
		return "function"
	case ValueKindHostReference:
		switch value.HostReferenceKind {
		case HostReferenceKindFunction, HostReferenceKindConstructor:
			return "function"
		default:
			return "object"
		}
	default:
		return "object"
	}
}

func (p *classicJSStatementParser) parseConditional() (jsValue, error) {
	left, err := p.parseNullishCoalescing()
	if err != nil {
		return jsValue{}, err
	}

	p.skipSpaceAndComments()
	if p.eof() || p.peekByte() != '?' || p.pos+1 >= len(p.source) || p.source[p.pos+1] == '.' || p.source[p.pos+1] == '?' {
		return left, nil
	}

	p.pos++
	branchStart := p.pos
	skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
	skip.pos = branchStart
	if _, err := skip.parseLogicalAssignment(); err != nil {
		return jsValue{}, err
	}
	skip.skipSpaceAndComments()
	if !skip.consumeByte(':') {
		return jsValue{}, NewError(ErrorKindParse, "conditional expressions require a `:` alternate branch in this bounded classic-JS slice")
	}
	alternatePos := skip.pos

	if jsTruthy(left.value) {
		consequent, err := p.parseLogicalAssignment()
		if err != nil {
			return jsValue{}, err
		}
		p.skipSpaceAndComments()
		if !p.consumeByte(':') {
			return jsValue{}, NewError(ErrorKindParse, "conditional expressions require a `:` alternate branch in this bounded classic-JS slice")
		}
		skipAlternate := p.cloneForSkipping(skipHostBindings{delegate: p.host})
		skipAlternate.pos = p.pos
		if _, err := skipAlternate.parseLogicalAssignment(); err != nil {
			return jsValue{}, err
		}
		p.pos = skipAlternate.pos
		return consequent, nil
	}

	p.pos = alternatePos
	return p.parseLogicalAssignment()
}

func (p *classicJSStatementParser) parseNullishCoalescing() (jsValue, error) {
	left, err := p.parseLogicalOr()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() || p.peekByte() != '?' || p.pos+1 >= len(p.source) || p.source[p.pos+1] != '?' {
			return left, nil
		}

		p.pos += 2
		if left.kind != jsValueScalar || !isNullishJSValue(left.value) {
			// Short-circuit the right-hand side without running host side effects.
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = p.pos
			if _, err := skip.parseScalarExpression(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			continue
		}

		right, err := p.parseNullishCoalescing()
		if err != nil {
			return jsValue{}, err
		}
		left = right
	}
}

func (p *classicJSStatementParser) parseLogicalOr() (jsValue, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.pos+1 >= len(p.source) || p.source[p.pos] != '|' || p.source[p.pos+1] != '|' {
			return left, nil
		}

		p.pos += 2
		if jsTruthyJSValue(left) {
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = p.pos
			if _, err := skip.parseScalarExpression(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			continue
		}

		right, err := p.parseLogicalOr()
		if err != nil {
			return jsValue{}, err
		}
		left = right
	}
}

func (p *classicJSStatementParser) parseLogicalAnd() (jsValue, error) {
	left, err := p.parseBitwiseOr()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.pos+1 >= len(p.source) || p.source[p.pos] != '&' || p.source[p.pos+1] != '&' {
			return left, nil
		}

		p.pos += 2
		if !jsTruthyJSValue(left) {
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = p.pos
			if _, err := skip.parseScalarExpression(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			continue
		}

		right, err := p.parseLogicalAnd()
		if err != nil {
			return jsValue{}, err
		}
		left = right
	}
}

func (p *classicJSStatementParser) parseBitwiseOr() (jsValue, error) {
	left, err := p.parseBitwiseXor()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() || p.peekByte() != '|' || p.pos+1 < len(p.source) && p.source[p.pos+1] == '|' {
			return left, nil
		}

		p.pos++
		right, err := p.parseBitwiseXor()
		if err != nil {
			return jsValue{}, err
		}
		result, err := classicJSBitwiseBinaryValues(left.value, right.value, "|")
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(result)
	}
}

func (p *classicJSStatementParser) parseBitwiseXor() (jsValue, error) {
	left, err := p.parseBitwiseAnd()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() || p.peekByte() != '^' {
			return left, nil
		}

		p.pos++
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return jsValue{}, err
		}
		result, err := classicJSBitwiseBinaryValues(left.value, right.value, "^")
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(result)
	}
}

func (p *classicJSStatementParser) parseBitwiseAnd() (jsValue, error) {
	left, err := p.parseEquality()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() || p.peekByte() != '&' || p.pos+1 < len(p.source) && p.source[p.pos+1] == '&' {
			return left, nil
		}

		p.pos++
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return jsValue{}, err
		}
		result, err := classicJSBitwiseBinaryValues(left.value, right.value, "&")
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(result)
	}
}

func (p *classicJSStatementParser) parseEquality() (jsValue, error) {
	left, err := p.parseRelational()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		op := ""
		switch {
		case p.pos+3 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "==="):
			op = "==="
		case p.pos+3 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "!=="):
			op = "!=="
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "=="):
			op = "=="
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "!="):
			op = "!="
		default:
			return left, nil
		}
		p.pos += len(op)

		right, err := p.parseRelational()
		if err != nil {
			return jsValue{}, err
		}
		equal := classicJSEqualValues(left.value, right.value, op)
		left = scalarJSValue(BoolValue(equal))
	}
}

func (p *classicJSStatementParser) parseRelational() (jsValue, error) {
	left, err := p.parseShift()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		op := ""
		keyword, ok := p.peekKeyword("instanceof")
		if !ok {
			keyword, ok = p.peekKeyword("in")
		}
		switch {
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "<="):
			op = "<="
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], ">="):
			op = ">="
		case ok:
			op = keyword
		case p.consumeByte('<'):
			op = "<"
		case p.consumeByte('>'):
			op = ">"
		default:
			return left, nil
		}
		if op == "<" || op == ">" {
			// consumeByte already advanced.
		} else {
			p.pos += len(op)
		}

		right, err := p.parseShift()
		if err != nil {
			return jsValue{}, err
		}
		if op == "instanceof" {
			matched, err := classicJSInstanceOf(left.value, right.value)
			if err != nil {
				return jsValue{}, err
			}
			left = scalarJSValue(BoolValue(matched))
			continue
		}
		if op == "in" {
			if left.kind != jsValueScalar {
				return jsValue{}, NewError(ErrorKindUnsupported, "relational `in` only works on scalar values in this bounded classic-JS slice")
			}
			if left.value.Kind == ValueKindPrivateName {
				matched, err := classicJSPrivateContainsProperty(right.value, left.value.PrivateName, p.privateFieldPrefix)
				if err != nil {
					return jsValue{}, err
				}
				left = scalarJSValue(BoolValue(matched))
				continue
			}
			matched, err := classicJSContainsProperty(p, right.value, ToJSString(left.value))
			if err != nil {
				return jsValue{}, err
			}
			left = scalarJSValue(BoolValue(matched))
			continue
		}
		if left.kind != jsValueScalar || right.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "relational comparisons only work on scalar values in this bounded classic-JS slice")
		}
		matched, err := classicJSRelationalCompare(left.value, right.value, op)
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(BoolValue(matched))
	}
}

func (p *classicJSStatementParser) parseShift() (jsValue, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		op := ""
		switch {
		case p.pos+3 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], ">>>"):
			op = ">>>"
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "<<"):
			op = "<<"
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], ">>"):
			op = ">>"
		default:
			return left, nil
		}
		p.pos += len(op)

		right, err := p.parseAdditive()
		if err != nil {
			return jsValue{}, err
		}
		result, err := classicJSBitwiseBinaryValues(left.value, right.value, op)
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(result)
	}
}

func (p *classicJSStatementParser) parseAdditive() (jsValue, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() {
			return left, nil
		}
		op := byte(0)
		switch p.peekByte() {
		case '+', '-':
			op = p.peekByte()
			p.pos++
		default:
			return left, nil
		}

		right, err := p.parseMultiplicative()
		if err != nil {
			return jsValue{}, err
		}
		if left.kind != jsValueScalar || right.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "additive operators only work on scalar values in this bounded classic-JS slice")
		}
		result, err := classicJSAddValues(left.value, right.value, op)
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(result)
	}
}

func (p *classicJSStatementParser) parseMultiplicative() (jsValue, error) {
	left, err := p.parseUnary()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.eof() {
			return left, nil
		}
		op := byte(0)
		switch p.peekByte() {
		case '*', '/', '%':
			op = p.peekByte()
			p.pos++
		default:
			return left, nil
		}

		right, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if left.kind != jsValueScalar || right.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "multiplicative operators only work on scalar values in this bounded classic-JS slice")
		}
		result, err := classicJSMultiplyValues(left.value, right.value, op)
		if err != nil {
			return jsValue{}, err
		}
		left = scalarJSValue(result)
	}
}

func (p *classicJSStatementParser) parseUnary() (jsValue, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return jsValue{}, NewError(ErrorKindParse, "unexpected end of script source")
	}

	switch {
	case p.pos+1 < len(p.source) && p.source[p.pos] == '+' && p.source[p.pos+1] == '+':
		p.pos += 2
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		return p.applyClassicJSIncrementDecrement(value, 1, true)
	case p.pos+1 < len(p.source) && p.source[p.pos] == '-' && p.source[p.pos+1] == '-':
		p.pos += 2
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		return p.applyClassicJSIncrementDecrement(value, -1, true)
	}

	switch p.peekByte() {
	case '+':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `+` is only supported for scalar values in this slice")
		}
		if value.value.Kind == ValueKindBigInt {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `+` is not supported for BigInt values in this slice")
		}
		number, ok := classicJSUnaryNumberValue(value.value)
		if !ok {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `+` is only supported for scalar values in this slice")
		}
		return scalarJSValue(NumberValue(number)), nil
	case '-':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `-` is only supported for scalar values in this slice")
		}
		switch value.value.Kind {
		case ValueKindNumber:
			return scalarJSValue(NumberValue(-value.value.Number)), nil
		case ValueKindBigInt:
			negated, err := negateBigIntLiteral(value.value.BigInt)
			if err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(BigIntValue(negated)), nil
		default:
			number, ok := classicJSUnaryNumberValue(value.value)
			if !ok {
				return jsValue{}, NewError(ErrorKindUnsupported, "unary `-` is only supported for scalar values in this slice")
			}
			return scalarJSValue(NumberValue(-number)), nil
		}
	case '~':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		result, err := classicJSBitwiseNotValue(value.value)
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(result), nil
	case '!':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(BoolValue(!jsTruthyJSValue(value))), nil
	}

	if keyword, ok := p.peekKeyword("void"); ok {
		p.pos += len(keyword)
		if _, err := p.parseUnary(); err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(UndefinedValue()), nil
	}

	if keyword, ok := p.peekKeyword("await"); ok {
		if !p.allowAwait {
			return jsValue{}, NewError(ErrorKindParse, "`await` is only supported inside bounded async arrow functions in this slice")
		}
		p.pos += len(keyword)
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return value, nil
		}
		return scalarJSValue(unwrapPromiseValue(value.value)), nil
	}

	if keyword, ok := p.peekKeyword("typeof"); ok {
		p.pos += len(keyword)
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(StringValue(classicJSTypeofJSValue(value))), nil
	}

	if keyword, ok := p.peekKeyword("delete"); ok {
		p.pos += len(keyword)
		return p.parseDeleteExpression()
	}

	if keyword, ok := p.peekKeyword("yield"); ok && p.allowYield {
		p.pos += len(keyword)
		value, resumeState, yielded, err := p.parseYieldExpressionValue()
		if err != nil {
			return jsValue{}, err
		}
		if !yielded {
			return scalarJSValue(value), nil
		}
		return jsValue{}, classicJSYieldSignal{value: value, resumeState: resumeState}
	}

	if keyword, ok := p.peekKeyword("new"); ok {
		if p.pos+len(keyword)+len(".target") <= len(p.source) && strings.HasPrefix(p.source[p.pos+len(keyword):], ".target") {
			p.pos += len(keyword) + len(".target")
			if !p.hasNewTarget {
				return jsValue{}, NewError(ErrorKindParse, "new.target is only supported inside bounded function or constructor bodies in this slice")
			}
			return scalarJSValue(p.newTarget), nil
		}
		p.pos += len(keyword)
		return p.parseNewExpression()
	}

	return p.parseExponentiation()
}

func (p *classicJSStatementParser) parseExponentiation() (jsValue, error) {
	left, err := p.parsePostfix()
	if err != nil {
		return jsValue{}, err
	}

	p.skipSpaceAndComments()
	if p.pos+1 >= len(p.source) || p.source[p.pos] != '*' || p.source[p.pos+1] != '*' {
		return left, nil
	}
	p.pos += 2

	right, err := p.parseUnary()
	if err != nil {
		return jsValue{}, err
	}
	if left.kind != jsValueScalar || right.kind != jsValueScalar {
		return jsValue{}, NewError(ErrorKindUnsupported, "exponentiation only works on scalar values in this bounded classic-JS slice")
	}
	result, err := classicJSPowerValues(left.value, right.value)
	if err != nil {
		return jsValue{}, err
	}
	return scalarJSValue(result), nil
}

func (p *classicJSStatementParser) parseNewExpression() (jsValue, error) {
	p.skipSpaceAndComments()
	start := p.pos
	if isIdentStart(p.peekByte()) {
		name, err := p.parseIdentifier()
		if err != nil {
			return jsValue{}, err
		}
		p.skipSpaceAndComments()
		if p.peekByte() == '(' {
			if p.env != nil {
				if _, ok := p.env.classDefinition(name); ok {
					classValue, ok := p.env.lookup(name)
					if !ok || classValue.kind != jsValueScalar || classValue.value.Kind != ValueKindObject {
						p.pos = start
						goto parseNewCallee
					}
					p.pos++
					args, err := p.parseArguments()
					if err != nil {
						return jsValue{}, err
					}
					value, err := p.instantiateClassicJSClass(name, classValue.value, args)
					if err != nil {
						return jsValue{}, err
					}
					return p.parsePostfixTail(value)
				}
			}
		}
		p.pos = start
	}

parseNewCallee:
	callee, err := p.parseNewCallee()
	if err != nil {
		return jsValue{}, err
	}

	p.skipSpaceAndComments()
	args := []Value(nil)
	if p.consumeByte('(') {
		args, err = p.parseArguments()
		if err != nil {
			return jsValue{}, err
		}
	}

	if callee.kind == jsValueScalar && callee.value.Kind == ValueKindHostReference && p.host != nil {
		if resolver, ok := p.host.(HostReferenceResolver); ok {
			resolved, err := resolver.ResolveHostReference(callee.value.HostReferencePath)
			if err != nil {
				return jsValue{}, err
			}
			callee = scalarJSValue(resolved)
		}
	}

	if callee.kind == jsValueScalar && callee.value.Kind == ValueKindObject && callee.value.ClassKey != "" {
		value, err := p.instantiateClassicJSClass("", callee.value, args)
		if err != nil {
			return jsValue{}, err
		}
		return p.parsePostfixTail(value)
	}

	if callee.kind == jsValueScalar && callee.value.Kind == ValueKindFunction {
		if callee.value.NativeFunction != nil {
			value, err := callee.value.NativeFunction(args)
			if err != nil {
				return jsValue{}, err
			}
			return p.parsePostfixTail(scalarJSValue(value))
		}
		if callee.value.Function == nil || !callee.value.Function.constructible {
			return jsValue{}, NewError(ErrorKindUnsupported, "new expressions only work on class expressions, class identifiers, or constructible function values in this bounded classic-JS slice")
		}
		marker, ok := classicJSConstructibleFunctionMarker(callee.value.Function)
		if !ok || marker == "" {
			return jsValue{}, NewError(ErrorKindUnsupported, "new expressions only work on class expressions, class identifiers, or constructible function values in this bounded classic-JS slice")
		}
		receiver := ObjectValue([]ObjectEntry{
			{
				Key:   classicJSInstanceMarkerKey(marker),
				Value: BoolValue(true),
			},
		})
		constructorCall := callee.withNewTarget(callee.value)
		constructorCall.receiver = receiver
		constructorCall.hasReceiver = true
		value, err := p.invokeArrowFunction(callee.value.Function, args, constructorCall)
		if err != nil {
			return jsValue{}, err
		}
		if value.kind == jsValueScalar && value.value.Kind == ValueKindObject {
			return p.parsePostfixTail(value)
		}
		return p.parsePostfixTail(scalarJSValue(receiver))
	}
	return jsValue{}, NewError(ErrorKindUnsupported, "new expressions only work on class expressions, class identifiers, or constructible function values in this bounded classic-JS slice")
}

func (p *classicJSStatementParser) parseNewCallee() (jsValue, error) {
	value, err := p.parsePrimary()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		switch {
		case p.consumeByte('.'):
			name, err := p.parseMemberAccessName()
			if err != nil {
				return jsValue{}, err
			}
			value, err = p.resolveMemberAccess(value, name)
			if err != nil {
				return jsValue{}, err
			}
		case p.consumeByte('['):
			source, err := p.consumeBracketAccessExpressionSource()
			if err != nil {
				return jsValue{}, err
			}
			index, err := p.evalExpressionWithEnv(source, p.env)
			if err != nil {
				return jsValue{}, err
			}
			value, err = p.resolveBracketAccess(value, index)
			if err != nil {
				return jsValue{}, err
			}
		default:
			return value, nil
		}
	}
}

func (p *classicJSStatementParser) parseDeleteExpression() (jsValue, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return jsValue{}, NewError(ErrorKindParse, "delete expressions require an identifier target in this bounded classic-JS slice")
	}

	baseName, err := p.parseIdentifier()
	if err != nil {
		return jsValue{}, NewError(ErrorKindParse, "delete expressions require an identifier target in this bounded classic-JS slice")
	}

	steps, err := p.parseDeleteAccessSteps()
	if err != nil {
		return jsValue{}, err
	}

	baseValue, found := jsValue{}, false
	if p.env != nil {
		baseValue, found = p.env.lookup(baseName)
	}
	if len(steps) == 0 {
		if found {
			return scalarJSValue(BoolValue(false)), nil
		}
		return scalarJSValue(BoolValue(true)), nil
	}
	if found && baseValue.kind == jsValueSuper {
		if baseValue.value.Kind != ValueKindObject && baseValue.value.Kind != ValueKindNull {
			return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
		}
		if baseValue.receiver.Kind != ValueKindObject {
			return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
		}
		updated, deleted, err := deleteJSValuePropertyChain(p, baseValue.receiver, steps, p.privateFieldPrefix)
		if err != nil {
			return jsValue{}, err
		}
		if updated.Kind == ValueKindObject {
			updatedReceiver := scalarJSValue(updated)
			p.replaceObjectBindings(baseValue.receiver, updatedReceiver)
			baseValue.receiver = updatedReceiver.value
		}
		return scalarJSValue(BoolValue(deleted)), nil
	}
	if !found {
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("delete target %q is not a declared local binding in this bounded classic-JS slice", baseName))
	}
	if baseValue.kind != jsValueScalar {
		return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on scalar object bindings in this bounded classic-JS slice")
	}
	if isNullishJSValue(baseValue.value) {
		if steps[0].optional {
			return scalarJSValue(BoolValue(true)), nil
		}
		return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
	}
	if baseValue.value.Kind != ValueKindObject &&
		baseValue.value.Kind != ValueKindArray &&
		baseValue.value.Kind != ValueKindString &&
		baseValue.value.Kind != ValueKindHostReference &&
		baseValue.value.Kind != ValueKindFunction &&
		baseValue.value.Kind != ValueKindNumber &&
		baseValue.value.Kind != ValueKindBool &&
		baseValue.value.Kind != ValueKindBigInt {
		return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on object, array, string, host surface, or primitive number/boolean/bigint values in this bounded classic-JS slice")
	}

	updated, deleted, err := deleteJSValuePropertyChain(p, baseValue.value, steps, p.privateFieldPrefix)
	if err != nil {
		return jsValue{}, err
	}
	if baseValue.value.Kind == ValueKindHostReference {
		return scalarJSValue(BoolValue(deleted)), nil
	}
	if p.env != nil {
		if err := p.env.assign(baseName, scalarJSValue(updated)); err != nil {
			return jsValue{}, err
		}
	}
	return scalarJSValue(BoolValue(deleted)), nil
}

func (p *classicJSStatementParser) parseDeleteAccessSteps() ([]classicJSDeleteStep, error) {
	steps, ok, err := p.scanPropertyAccessSteps(false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewError(ErrorKindUnsupported, "optional chaining is not supported in delete expressions in this bounded classic-JS slice")
	}
	return steps, nil
}

func (p *classicJSStatementParser) scanAssignmentAccessSteps() ([]classicJSDeleteStep, bool, error) {
	return p.scanPropertyAccessSteps(true)
}

func (p *classicJSStatementParser) scanPropertyAccessSteps(rejectOptional bool) ([]classicJSDeleteStep, bool, error) {
	steps := make([]classicJSDeleteStep, 0, 2)
	for {
		p.skipSpaceAndComments()
		if p.eof() {
			return steps, true, nil
		}
		optional := false
		if p.peekByte() == '?' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '.' {
			if rejectOptional {
				return nil, false, nil
			}
			optional = true
			p.pos += 2
			p.skipSpaceAndComments()
		}
		switch {
		case p.consumeByte('.'):
			name, err := p.parseMemberAccessName()
			if err != nil {
				return nil, false, err
			}
			steps = append(steps, classicJSDeleteStep{
				key:      name,
				private:  strings.HasPrefix(name, "#"),
				optional: optional,
			})
		case p.consumeByte('['):
			source, err := p.consumeBracketAccessExpressionSource()
			if err != nil {
				return nil, false, err
			}
			value, err := p.evalExpressionWithEnv(source, p.env)
			if err != nil {
				return nil, false, err
			}
			steps = append(steps, classicJSDeleteStep{key: ToJSString(value), optional: optional})
		default:
			if optional {
				name, err := p.parseMemberAccessName()
				if err != nil {
					return nil, false, err
				}
				steps = append(steps, classicJSDeleteStep{
					key:      name,
					private:  strings.HasPrefix(name, "#"),
					optional: true,
				})
				continue
			}
			return steps, true, nil
		}
	}
}

func (p *classicJSStatementParser) parsePostfix() (jsValue, error) {
	value, err := p.parsePrimary()
	if err != nil {
		return jsValue{}, err
	}

	return p.parsePostfixTail(value)
}

func (p *classicJSStatementParser) parsePostfixTail(value jsValue) (jsValue, error) {
	shortCircuited := false
	for {
		p.skipSpaceAndComments()
		switch {
		case p.peekByte() == '?' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '.':
			p.pos += 2
			p.skipSpaceAndComments()
			switch {
			case p.peekByte() == '(':
				if shortCircuited || (value.kind == jsValueScalar && isNullishJSValue(value.value)) {
					if err := p.skipOptionalCallArguments(); err != nil {
						return jsValue{}, err
					}
					shortCircuited = true
					value = scalarJSValue(UndefinedValue())
					continue
				}
				p.pos++
				args, err := p.parseArguments()
				if err != nil {
					return jsValue{}, err
				}
				value, err = p.invoke(value, args)
				if err != nil {
					return jsValue{}, err
				}
				value = value.withoutAssignTarget()
			case p.peekByte() == '[':
				p.pos++
				source, err := p.consumeBracketAccessExpressionSource()
				if err != nil {
					return jsValue{}, err
				}
				if shortCircuited || (value.kind == jsValueScalar && isNullishJSValue(value.value)) {
					shortCircuited = true
					value = scalarJSValue(UndefinedValue())
					continue
				}
				index, err := p.evalExpressionWithEnv(source, p.env)
				if err != nil {
					return jsValue{}, err
				}
				target := value.assignTarget
				value, err = p.resolveBracketAccess(value, index)
				if err != nil {
					return jsValue{}, err
				}
				if target != nil {
					steps := append([]classicJSDeleteStep(nil), target.steps...)
					steps = append(steps, classicJSDeleteStep{key: ToJSString(index)})
					value = value.withAssignTarget(target.name, steps)
				}
			default:
				name, err := p.parseMemberAccessName()
				if err != nil {
					return jsValue{}, err
				}
				if shortCircuited {
					continue
				}
				if value.kind == jsValueScalar && isNullishJSValue(value.value) {
					shortCircuited = true
					value = scalarJSValue(UndefinedValue())
					continue
				}
				target := value.assignTarget
				value, err = p.resolveMemberAccess(value, name)
				if err != nil {
					return jsValue{}, err
				}
				if target != nil {
					steps := append([]classicJSDeleteStep(nil), target.steps...)
					steps = append(steps, classicJSDeleteStep{
						key:     name,
						private: strings.HasPrefix(name, "#"),
					})
					value = value.withAssignTarget(target.name, steps)
				}
			}

		case p.consumeByte('.'):
			name, err := p.parseMemberAccessName()
			if err != nil {
				return jsValue{}, err
			}
			if shortCircuited {
				continue
			}
			target := value.assignTarget
			value, err = p.resolveMemberAccess(value, name)
			if err != nil {
				return jsValue{}, err
			}
			if target != nil {
				steps := append([]classicJSDeleteStep(nil), target.steps...)
				steps = append(steps, classicJSDeleteStep{
					key:     name,
					private: strings.HasPrefix(name, "#"),
				})
				value = value.withAssignTarget(target.name, steps)
			} else {
				value = value.withoutAssignTarget()
			}

		case p.consumeByte('['):
			source, err := p.consumeBracketAccessExpressionSource()
			if err != nil {
				return jsValue{}, err
			}
			if shortCircuited {
				continue
			}
			target := value.assignTarget
			index, err := p.evalExpressionWithEnv(source, p.env)
			if err != nil {
				return jsValue{}, err
			}
			value, err = p.resolveBracketAccess(value, index)
			if err != nil {
				return jsValue{}, err
			}
			if target != nil {
				steps := append([]classicJSDeleteStep(nil), target.steps...)
				steps = append(steps, classicJSDeleteStep{key: ToJSString(index)})
				value = value.withAssignTarget(target.name, steps)
			} else {
				value = value.withoutAssignTarget()
			}

		case p.peekByte() == ':':
			if value.kind != jsValueHostObject {
				return value, nil
			}
			p.pos++
			method, err := p.parseIdentifier()
			if err != nil {
				return jsValue{}, err
			}
			if shortCircuited {
				continue
			}
			value = hostMethodJSValue(method)

		case p.peekByte() == '`':
			if shortCircuited {
				if _, _, err := p.consumeTemplateLiteralParts(); err != nil {
					return jsValue{}, err
				}
				value = scalarJSValue(UndefinedValue())
				continue
			}
			taggedValue, err := p.parseTaggedTemplateLiteral(value)
			if err != nil {
				return jsValue{}, err
			}
			value = taggedValue.withoutAssignTarget()

		case p.pos+1 < len(p.source) && p.source[p.pos] == '+' && p.source[p.pos+1] == '+':
			p.pos += 2
			return p.applyClassicJSIncrementDecrement(value, 1, false)
		case p.pos+1 < len(p.source) && p.source[p.pos] == '-' && p.source[p.pos+1] == '-':
			p.pos += 2
			return p.applyClassicJSIncrementDecrement(value, -1, false)
		case p.consumeByte('('):
			if shortCircuited {
				skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
				skip.pos = p.pos
				if _, err := skip.parseArguments(); err != nil {
					return jsValue{}, err
				}
				p.pos = skip.pos
				continue
			}
			args, err := p.parseArguments()
			if err != nil {
				return jsValue{}, err
			}
			value, err = p.invoke(value, args)
			if err != nil {
				return jsValue{}, err
			}
			value = value.withoutAssignTarget()

		default:
			return value, nil
		}
	}
}

func (p *classicJSStatementParser) applyClassicJSIncrementDecrement(value jsValue, delta int, prefix bool) (jsValue, error) {
	if value.assignTarget == nil {
		return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on declared local bindings or object properties in this bounded classic-JS slice")
	}

	updated, err := classicJSIncrementDecrementValue(value.value, delta)
	if err != nil {
		return jsValue{}, err
	}

	target := value.assignTarget
	if target.name == "super" {
		if p.env == nil {
			return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on declared local bindings or object properties in this bounded classic-JS slice")
		}
		current, ok := p.env.lookup("super")
		if !ok || current.kind != jsValueSuper {
			return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on declared local bindings or object properties in this bounded classic-JS slice")
		}
		if len(target.steps) == 0 {
			return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on declared local bindings or object properties in this bounded classic-JS slice")
		}
		currentValue, err := resolveJSValuePropertyChain(p, current.value, target.steps, p.privateFieldPrefix)
		if err != nil {
			return jsValue{}, err
		}
		updatedSuper, err := assignSuperJSValuePropertyChain(p, current, target.steps, updated, p.privateFieldPrefix)
		if err != nil {
			return jsValue{}, err
		}
		_ = updatedSuper
		if prefix {
			return scalarJSValue(updated), nil
		}
		return scalarJSValue(currentValue), nil
	}
	if len(target.steps) == 0 {
		if p.env == nil {
			return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("assignment target %q is not a declared local binding in this bounded classic-JS slice", target.name))
		}
		if err := p.env.assign(target.name, scalarJSValue(updated)); err != nil {
			return jsValue{}, err
		}
	} else {
		if p.env == nil {
			return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on declared local bindings or object properties in this bounded classic-JS slice")
		}
		current, ok := p.env.lookup(target.name)
		if !ok {
			return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("assignment target %q is not a declared local binding in this bounded classic-JS slice", target.name))
		}
		if current.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on scalar object or array bindings in this bounded classic-JS slice")
		}
		if current.value.Kind != ValueKindObject && current.value.Kind != ValueKindArray {
			return jsValue{}, NewError(ErrorKindUnsupported, "increment and decrement only work on object or array values in this bounded classic-JS slice")
		}
		if _, err := assignJSValuePropertyChain(p, current.value, target.steps, updated, p.privateFieldPrefix); err != nil {
			return jsValue{}, err
		}
	}

	if prefix {
		return scalarJSValue(updated), nil
	}
	return scalarJSValue(value.value), nil
}

func (p *classicJSStatementParser) resolveMemberAccess(value jsValue, name string) (jsValue, error) {
	switch value.kind {
	case jsValueHostObject:
		return hostMethodJSValue(name), nil
	case jsValueSuper:
		switch value.value.Kind {
		case ValueKindObject:
			if resolved, ok := lookupObjectProperty(value.value.Object, name); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
					return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.receiver, hasReceiver: true})
				}
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.receiver, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindNull:
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindFunction:
			if name == "prototype" {
				if resolved, ok := classicJSConstructibleFunctionPrototypeValue(value.value); ok {
					return scalarJSValue(resolved), nil
				}
			}
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported `super` member access in this bounded classic-JS slice; only object-backed class targets are available",
			)
		default:
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported `super` member access in this bounded classic-JS slice; only object-backed class targets are available",
			)
		}
	case jsValueScalar:
		if value.value.Kind == ValueKindHostReference {
			resolved, err := p.resolveHostReferencePath(joinHostReferencePath(value.value.HostReferencePath, name))
			if err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(resolved), nil
		}
		if value.value.Kind == ValueKindFunction && name == "prototype" {
			if resolved, ok := classicJSConstructibleFunctionPrototypeValue(value.value); ok {
				return scalarJSValue(resolved), nil
			}
		}
		if value.value.Kind == ValueKindString {
			switch name {
			case "length":
				return scalarJSValue(NumberValue(float64(len(value.value.String)))), nil
			default:
				if method, ok, err := p.resolveStringPrototypeMethod(value.value, name); ok || err != nil {
					return scalarJSValue(method), err
				}
				return scalarJSValue(UndefinedValue()), nil
			}
		}
		if value.value.Kind == ValueKindBool {
			if method, ok, err := p.resolveBoolPrototypeMethod(value.value, name); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		}
		if name == "prototype" && (value.value.ClassDefinition != nil || value.value.ClassKey != "") {
			if resolved, ok := classicJSClassPrototypeAccessValue(value.value); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
					return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.value, hasReceiver: true})
				}
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.value, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
		}
		if method, ok, err := p.resolvePromisePrototypeMethod(value.value, name); ok || err != nil {
			return scalarJSValue(method), err
		}
		if strings.HasPrefix(name, "#") {
			if p.privateFieldPrefix == "" {
				return jsValue{}, NewError(ErrorKindUnsupported, "private class fields are not accessible outside this class body in this bounded classic-JS slice")
			}
			privateKey := p.privateFieldPrefix + strings.TrimPrefix(name, "#")
			switch value.value.Kind {
			case ValueKindObject:
				if resolved, ok := lookupObjectProperty(value.value.Object, privateKey); ok {
					if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
						return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.value, hasReceiver: true})
					}
					if resolved.Kind == ValueKindFunction && resolved.Function != nil {
						return jsValue{kind: jsValueScalar, value: resolved, receiver: value.value, hasReceiver: true}, nil
					}
					return scalarJSValue(resolved), nil
				}
				return scalarJSValue(UndefinedValue()), nil
			default:
				return jsValue{}, NewError(ErrorKindUnsupported, "private class fields only work on object values in this bounded classic-JS slice")
			}
		}
		switch value.value.Kind {
		case ValueKindObject:
			if method, ok, err := p.resolveDatePrototypeMethod(value.value, name); ok || err != nil {
				return scalarJSValue(method), err
			}
			if resolved, ok := lookupObjectProperty(value.value.Object, name); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
					return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.value, hasReceiver: true})
				}
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.value, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindArray:
			if name == "length" {
				return scalarJSValue(NumberValue(float64(len(value.value.Array)))), nil
			}
			if method, ok, err := p.resolveArrayPrototypeMethod(value.value, name); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindNumber:
			if method, ok, err := p.resolveNumberPrototypeMethod(value.value, name); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindNull, ValueKindUndefined:
			return jsValue{}, NewError(ErrorKindRuntime, fmt.Sprintf("cannot access property %q on nullish value in this bounded classic-JS slice", name))
		default:
			return scalarJSValue(UndefinedValue()), nil
		}
	default:
		return jsValue{}, NewError(
			ErrorKindUnsupported,
			"unsupported member access in this bounded classic-JS slice; only object properties, array `length`, and `host.method(...)` are available",
		)
	}
}

func (p *classicJSStatementParser) resolveBracketAccess(value jsValue, key Value) (jsValue, error) {
	keyString := ToJSString(key)
	switch value.kind {
	case jsValueHostObject:
		return hostMethodJSValue(keyString), nil
	case jsValueSuper:
		switch value.value.Kind {
		case ValueKindObject:
			if resolved, ok := lookupObjectProperty(value.value.Object, keyString); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
					return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.receiver, hasReceiver: true})
				}
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.receiver, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindNull:
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindFunction:
			if keyString == "prototype" {
				if resolved, ok := classicJSConstructibleFunctionPrototypeValue(value.value); ok {
					return scalarJSValue(resolved), nil
				}
			}
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported `super` bracket access in this bounded classic-JS slice; only object-backed class targets are available",
			)
		default:
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported `super` bracket access in this bounded classic-JS slice; only object-backed class targets are available",
			)
		}
	case jsValueScalar:
		if value.value.Kind == ValueKindHostReference {
			resolved, err := p.resolveHostReferencePath(joinHostReferencePath(value.value.HostReferencePath, keyString))
			if err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(resolved), nil
		}
		if value.value.Kind == ValueKindFunction && keyString == "prototype" {
			if resolved, ok := classicJSConstructibleFunctionPrototypeValue(value.value); ok {
				return scalarJSValue(resolved), nil
			}
		}
		if keyString == "prototype" && (value.value.ClassDefinition != nil || value.value.ClassKey != "") {
			if resolved, ok := classicJSClassPrototypeAccessValue(value.value); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
					return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.value, hasReceiver: true})
				}
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.value, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
		}
		if method, ok, err := p.resolvePromisePrototypeMethod(value.value, keyString); ok || err != nil {
			return scalarJSValue(method), err
		}
		switch value.value.Kind {
		case ValueKindObject:
			if resolved, ok := lookupObjectProperty(value.value.Object, keyString); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil && resolved.Function.objectAccessor {
					return p.invokeArrowFunction(resolved.Function, nil, jsValue{kind: jsValueScalar, value: value.value, receiver: value.value, hasReceiver: true})
				}
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.value, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindArray:
			if keyString == "length" {
				return scalarJSValue(NumberValue(float64(len(value.value.Array)))), nil
			}
			if index, ok := arrayIndexFromBracketKey(keyString); ok {
				if index < len(value.value.Array) {
					return scalarJSValue(value.value.Array[index]), nil
				}
				return scalarJSValue(UndefinedValue()), nil
			}
			if method, ok, err := p.resolveArrayPrototypeMethod(value.value, keyString); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindString:
			if keyString == "length" {
				return scalarJSValue(NumberValue(float64(len([]rune(value.value.String))))), nil
			}
			if index, ok := arrayIndexFromBracketKey(keyString); ok {
				runes := []rune(value.value.String)
				if index < len(runes) {
					return scalarJSValue(StringValue(string(runes[index]))), nil
				}
				return scalarJSValue(UndefinedValue()), nil
			}
			if method, ok, err := p.resolveStringPrototypeMethod(value.value, keyString); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindNumber:
			if method, ok, err := p.resolveNumberPrototypeMethod(value.value, keyString); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindBool:
			if method, ok, err := p.resolveBoolPrototypeMethod(value.value, keyString); ok || err != nil {
				return scalarJSValue(method), err
			}
			return scalarJSValue(UndefinedValue()), nil
		case ValueKindNull, ValueKindUndefined:
			return jsValue{}, NewError(ErrorKindRuntime, fmt.Sprintf("cannot access property %q on nullish value in this bounded classic-JS slice", keyString))
		default:
			return scalarJSValue(UndefinedValue()), nil
		}
	default:
		return jsValue{}, NewError(
			ErrorKindUnsupported,
			"unsupported bracket access in this bounded classic-JS slice; only object properties, string indexes, array indexes, array `length`, and `host[\"method\"]` are available",
		)
	}
}

func (p *classicJSStatementParser) resolveHostReferencePath(path string) (Value, error) {
	if p == nil || p.host == nil {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	resolver, ok := p.host.(HostReferenceResolver)
	if !ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", path))
	}
	value, err := resolver.ResolveHostReference(path)
	if err != nil {
		return UndefinedValue(), err
	}
	return value, nil
}

func joinHostReferencePath(base, name string) string {
	if base == "" {
		return name
	}
	if name == "" {
		return base
	}
	return base + "." + name
}

func lookupObjectProperty(entries []ObjectEntry, name string) (Value, bool) {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Key == name {
			return entries[i].Value, true
		}
	}
	return UndefinedValue(), false
}

func classicJSIsInternalObjectKey(name string) bool {
	return strings.HasPrefix(name, "\x00classic-js-setter:") || strings.HasPrefix(name, "\x00classic-js-instanceof:") || strings.HasPrefix(name, "\x00classic-js-static-prototype:") || strings.HasPrefix(name, classicJSRegExpInternalPrefix) || strings.HasPrefix(name, browserDateInternalPrefix)
}

func classicJSInstanceMarkerKey(marker string) string {
	return "\x00classic-js-instanceof:" + marker
}

func classicJSClassPrototypeKey(marker string) string {
	return "\x00classic-js-prototype:" + marker
}

func classicJSClassStaticPrototypeKey(marker string) string {
	return "\x00classic-js-static-prototype:" + marker
}

func classicJSClassPrototypeValue(value Value) (Value, bool) {
	if value.Kind != ValueKindObject {
		if prototypeValue, ok := classicJSConstructibleFunctionPrototypeValue(value); ok {
			return prototypeValue, true
		}
		return UndefinedValue(), false
	}
	marker := ""
	switch {
	case value.ClassDefinition != nil && value.ClassDefinition.instanceMarker != "":
		marker = value.ClassDefinition.instanceMarker
	case value.ClassKey != "":
		marker = value.ClassKey
	}
	if marker != "" {
		if prototypeValue, ok := lookupObjectProperty(value.Object, classicJSClassPrototypeKey(marker)); ok {
			if prototypeValue.Kind == ValueKindObject {
				return prototypeValue, true
			}
		}
	}
	prototypeValue, ok := lookupObjectProperty(value.Object, "prototype")
	if !ok || prototypeValue.Kind != ValueKindObject {
		return UndefinedValue(), false
	}
	return prototypeValue, true
}

func classicJSRegExpLiteralString(value Value) (string, bool) {
	if value.Kind != ValueKindObject {
		return "", false
	}
	patternValue, ok := lookupObjectProperty(value.Object, classicJSRegExpPatternKey)
	if !ok || patternValue.Kind != ValueKindString {
		return "", false
	}
	flagsValue, ok := lookupObjectProperty(value.Object, classicJSRegExpFlagsKey)
	if !ok || flagsValue.Kind != ValueKindString {
		return "", false
	}
	return "/" + patternValue.String + "/" + flagsValue.String, true
}

func classicJSConstructibleFunctionPrototypeValue(value Value) (Value, bool) {
	if value.Kind != ValueKindFunction || value.Function == nil || !value.Function.constructible {
		return UndefinedValue(), false
	}
	marker, ok := classicJSConstructibleFunctionMarker(value.Function)
	if !ok || marker == "" {
		return UndefinedValue(), false
	}
	return ObjectValue([]ObjectEntry{
		{
			Key:   "constructor",
			Value: value,
		},
		{
			Key:   classicJSInstanceMarkerKey(marker),
			Value: BoolValue(true),
		},
	}), true
}

func classicJSClassPrototypeMemberValue(value Value) (Value, bool) {
	if value.Kind != ValueKindObject {
		return UndefinedValue(), false
	}
	marker := ""
	switch {
	case value.ClassDefinition != nil && value.ClassDefinition.instanceMarker != "":
		marker = value.ClassDefinition.instanceMarker
	case value.ClassKey != "":
		marker = value.ClassKey
	}
	if marker != "" {
		if prototypeValue, ok := lookupObjectProperty(value.Object, classicJSClassStaticPrototypeKey(marker)); ok {
			return prototypeValue, true
		}
	}
	return lookupObjectProperty(value.Object, "prototype")
}

func classicJSClassPrototypeMemberStorageKey(value Value) (string, bool) {
	if value.Kind != ValueKindObject {
		return "", false
	}
	marker := ""
	switch {
	case value.ClassDefinition != nil && value.ClassDefinition.instanceMarker != "":
		marker = value.ClassDefinition.instanceMarker
	case value.ClassKey != "":
		marker = value.ClassKey
	}
	if marker != "" {
		if _, ok := lookupObjectProperty(value.Object, classicJSClassStaticPrototypeKey(marker)); ok {
			return classicJSClassStaticPrototypeKey(marker), true
		}
	}
	if _, ok := lookupObjectProperty(value.Object, "prototype"); ok {
		return "prototype", true
	}
	return "", false
}

func classicJSClassPrototypeSetterOnly(value Value) bool {
	if value.Kind != ValueKindObject {
		return false
	}
	if value.ClassDefinition == nil || !value.ClassDefinition.hasStaticPrototype {
		return false
	}
	actualPrototype, ok := classicJSClassPrototypeValue(value)
	if !ok || actualPrototype.Kind != ValueKindObject {
		return false
	}
	if _, hasSetter := lookupObjectProperty(value.Object, classicJSObjectSetterStorageKey("prototype")); !hasSetter {
		return false
	}
	currentPrototype, ok := lookupObjectProperty(value.Object, "prototype")
	if !ok || currentPrototype.Kind != ValueKindObject {
		return false
	}
	return reflect.ValueOf(currentPrototype.Object).Pointer() == reflect.ValueOf(actualPrototype.Object).Pointer()
}

func classicJSClassPrototypeAccessValue(value Value) (Value, bool) {
	if value.Kind != ValueKindObject || (value.ClassDefinition == nil && value.ClassKey == "") {
		return UndefinedValue(), false
	}
	if classicJSClassPrototypeSetterOnly(value) {
		return UndefinedValue(), true
	}
	if currentPrototype, ok := lookupObjectProperty(value.Object, "prototype"); ok {
		return currentPrototype, true
	}
	prototypeValue, ok := classicJSClassPrototypeValue(value)
	if !ok {
		return UndefinedValue(), false
	}
	return prototypeValue, true
}

func classicJSObjectHasInstanceMarker(value Value) bool {
	if value.Kind != ValueKindObject {
		return false
	}
	for _, entry := range value.Object {
		if strings.HasPrefix(entry.Key, "\x00classic-js-instanceof:") {
			return true
		}
	}
	return false
}

func classicJSObjectHasInstanceMarkerFor(value Value, marker string) bool {
	if value.Kind != ValueKindObject || marker == "" {
		return false
	}
	_, ok := lookupObjectProperty(value.Object, classicJSInstanceMarkerKey(marker))
	return ok
}

func classicJSConstructibleFunctionMarker(fn *classicJSArrowFunction) (string, bool) {
	if fn == nil || !fn.constructible {
		return "", false
	}
	if fn.constructMarker == "" {
		fn.constructMarker = fmt.Sprintf("%p", fn)
	}
	return fn.constructMarker, true
}

func (p *classicJSStatementParser) replaceObjectBindings(oldValue Value, newValue jsValue) {
	if oldValue.Kind != ValueKindObject || newValue.kind != jsValueScalar || newValue.value.Kind != ValueKindObject {
		return
	}
	if p.env != nil {
		p.env.replaceObjectBindings(oldValue, newValue)
	}
	if p.moduleExports == nil {
		return
	}
	oldPtr := reflect.ValueOf(oldValue.Object).Pointer()
	if oldPtr == 0 {
		return
	}
	replacement := newValue.value
	for name, value := range p.moduleExports {
		if value.Kind != ValueKindObject {
			continue
		}
		if reflect.ValueOf(value.Object).Pointer() != oldPtr {
			continue
		}
		p.moduleExports[name] = replacement
	}
}

func (p *classicJSStatementParser) replaceArrayBindings(oldValue Value, newValue jsValue) {
	if oldValue.Kind != ValueKindArray || newValue.kind != jsValueScalar || newValue.value.Kind != ValueKindArray {
		return
	}
	oldPtr := reflect.ValueOf(oldValue.Array).Pointer()
	if oldPtr == 0 {
		return
	}
	if p.env != nil {
		p.env.replaceArrayBindings(oldValue, newValue)
	}
	if p.moduleExports == nil {
		return
	}
	for name, value := range p.moduleExports {
		updated, changed := replaceArrayReferencesInValue(value, oldPtr, newValue.value)
		if !changed {
			continue
		}
		p.moduleExports[name] = updated
	}
}

func classicJSInstanceOf(left Value, right Value) (bool, error) {
	if right.Kind != ValueKindObject {
		if right.Kind == ValueKindFunction {
			if right.Function == nil || !right.Function.constructible {
				return false, NewError(ErrorKindRuntime, "relational `instanceof` requires a class object or constructible function value on the right in this bounded classic-JS slice")
			}
			marker, ok := classicJSConstructibleFunctionMarker(right.Function)
			if !ok || marker == "" {
				return false, NewError(ErrorKindRuntime, "relational `instanceof` requires a class object or constructible function value on the right in this bounded classic-JS slice")
			}
			if left.Kind != ValueKindObject {
				return false, nil
			}
			return classicJSObjectHasInstanceMarkerFor(left, marker), nil
		}
		return false, NewError(ErrorKindRuntime, "relational `instanceof` requires a class object or constructible function value on the right in this bounded classic-JS slice")
	}
	prototypeValue, ok := classicJSClassPrototypeValue(right)
	if !ok || prototypeValue.Kind != ValueKindObject {
		return false, NewError(ErrorKindRuntime, "relational `instanceof` requires a class object or constructible function value on the right in this bounded classic-JS slice")
	}

	markers := make([]string, 0, 1)
	for _, entry := range prototypeValue.Object {
		if strings.HasPrefix(entry.Key, "\x00classic-js-instanceof:") {
			markers = append(markers, strings.TrimPrefix(entry.Key, "\x00classic-js-instanceof:"))
		}
	}
	if len(markers) == 0 {
		return false, NewError(ErrorKindRuntime, "relational `instanceof` requires a class object or constructible function value on the right in this bounded classic-JS slice")
	}

	if left.Kind != ValueKindObject {
		return false, nil
	}
	for _, marker := range markers {
		if _, ok := lookupObjectProperty(left.Object, classicJSInstanceMarkerKey(marker)); ok {
			return true, nil
		}
	}
	return false, nil
}

func classicJSContainsProperty(p *classicJSStatementParser, value Value, key string) (bool, error) {
	switch value.Kind {
	case ValueKindObject:
		if _, ok := lookupObjectProperty(value.Object, key); ok {
			return true, nil
		}
		if _, ok := lookupObjectProperty(value.Object, classicJSObjectSetterStorageKey(key)); ok {
			return true, nil
		}
		return false, nil
	case ValueKindArray:
		if key == "length" {
			return true, nil
		}
		if index, ok := arrayIndexFromBracketKey(key); ok {
			return index >= 0 && index < len(value.Array), nil
		}
		return false, nil
	case ValueKindHostReference:
		resolved, err := p.resolveHostReferencePath(joinHostReferencePath(value.HostReferencePath, key))
		if err != nil {
			if scriptErr, ok := err.(Error); ok && scriptErr.Kind == ErrorKindUnsupported {
				return false, nil
			}
			return false, err
		}
		_ = resolved
		return true, nil
	default:
		return false, NewError(ErrorKindRuntime, "relational `in` requires an object or array value on the right in this bounded classic-JS slice")
	}
}

func classicJSPrivateContainsProperty(value Value, privateName string, privateFieldPrefix string) (bool, error) {
	if privateFieldPrefix == "" {
		return false, NewError(ErrorKindUnsupported, "private class fields are not accessible outside this class body in this bounded classic-JS slice")
	}
	if value.Kind != ValueKindObject {
		return false, NewError(ErrorKindUnsupported, "relational `in` only works on object values in this bounded classic-JS slice")
	}
	_, ok := lookupObjectProperty(value.Object, privateFieldPrefix+privateName)
	return ok, nil
}

func deleteJSValuePropertyChain(p *classicJSStatementParser, value Value, steps []classicJSDeleteStep, privateFieldPrefix string) (Value, bool, error) {
	if len(steps) == 0 {
		return value, true, nil
	}
	if isNullishJSValue(value) {
		if steps[0].optional {
			return value, true, nil
		}
		return UndefinedValue(), false, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
	}

	switch value.Kind {
	case ValueKindObject:
		key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}

		if len(steps) == 1 {
			if key == "prototype" && (value.ClassDefinition != nil || value.ClassKey != "") && classicJSClassPrototypeSetterOnly(value) {
				return ObjectValue(deleteObjectProperty(value.Object, classicJSObjectSetterStorageKey(key))), true, nil
			}
			return ObjectValue(deleteObjectProperty(value.Object, key)), true, nil
		}
		if key == "prototype" && (value.ClassDefinition != nil || value.ClassKey != "") && classicJSClassPrototypeSetterOnly(value) {
			return UndefinedValue(), false, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
		}

		child, ok := lookupObjectProperty(value.Object, key)
		if !ok {
			return value, true, nil
		}

		updatedChild, deleted, err := deleteJSValuePropertyChain(p, child, steps[1:], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}
		return ObjectValue(replaceObjectProperty(value.Object, key, updatedChild)), deleted, nil
	case ValueKindArray:
		key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}
		if key == "length" && len(steps) == 1 {
			return value, false, nil
		}
		index, ok := arrayIndexFromBracketKey(key)
		if !ok {
			return value, true, nil
		}
		if index >= len(value.Array) {
			return value, true, nil
		}
		if len(steps) == 1 {
			cloned := append([]Value(nil), value.Array...)
			cloned[index] = UndefinedValue()
			return ArrayValue(cloned), true, nil
		}
		updatedChild, deleted, err := deleteJSValuePropertyChain(p, value.Array[index], steps[1:], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}
		cloned := append([]Value(nil), value.Array...)
		cloned[index] = updatedChild
		return ArrayValue(cloned), deleted, nil
	case ValueKindString:
		key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}
		if len(steps) == 1 {
			if key == "length" {
				return value, false, nil
			}
			if _, ok := arrayIndexFromBracketKey(key); ok {
				return value, false, nil
			}
			return value, true, nil
		}
		if key == "length" {
			return UndefinedValue(), false, NewError(ErrorKindUnsupported, "delete only works on string indexes in this bounded classic-JS slice")
		}
		index, ok := arrayIndexFromBracketKey(key)
		if !ok {
			return UndefinedValue(), false, NewError(ErrorKindUnsupported, "delete only works on string indexes in this bounded classic-JS slice")
		}
		runes := []rune(value.String)
		if index >= len(runes) {
			return value, true, nil
		}
		_, deleted, err := deleteJSValuePropertyChain(p, StringValue(string(runes[index])), steps[1:], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}
		return value, deleted, nil
	case ValueKindHostReference:
		key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), false, err
		}
		nextPath := joinHostReferencePath(value.HostReferencePath, key)
		if len(steps) == 1 {
			if p.host == nil {
				return UndefinedValue(), false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", nextPath))
			}
			deleter, ok := p.host.(HostReferenceDeleter)
			if !ok {
				return UndefinedValue(), false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", nextPath))
			}
			if err := deleter.DeleteHostReference(nextPath); err != nil {
				return UndefinedValue(), false, err
			}
			return value, true, nil
		}
		resolved, err := p.resolveHostReferencePath(nextPath)
		if err != nil {
			return UndefinedValue(), false, err
		}
		if resolved.Kind != ValueKindObject && resolved.Kind != ValueKindArray && resolved.Kind != ValueKindHostReference {
			return UndefinedValue(), false, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
		}
		return deleteJSValuePropertyChain(p, resolved, steps[1:], privateFieldPrefix)
	case ValueKindFunction, ValueKindNumber, ValueKindBool, ValueKindBigInt:
		return value, true, nil
	default:
		return UndefinedValue(), false, NewError(ErrorKindUnsupported, "delete only works on object or array values in this bounded classic-JS slice")
	}
}

func deleteJSPropertyKey(step classicJSDeleteStep, privateFieldPrefix string) (string, error) {
	if !step.private {
		return step.key, nil
	}
	if privateFieldPrefix == "" {
		return "", NewError(ErrorKindUnsupported, "private class fields are not accessible outside this class body in this bounded classic-JS slice")
	}
	return privateFieldPrefix + strings.TrimPrefix(step.key, "#"), nil
}

func deleteObjectProperty(entries []ObjectEntry, name string) []ObjectEntry {
	if len(entries) == 0 {
		return nil
	}

	cloned := append([]ObjectEntry(nil), entries...)
	for i := len(cloned) - 1; i >= 0; i-- {
		if cloned[i].Key == name {
			cloned = append(cloned[:i], cloned[i+1:]...)
			break
		}
	}
	setterKey := classicJSObjectSetterStorageKey(name)
	for i := len(cloned) - 1; i >= 0; i-- {
		if cloned[i].Key == setterKey {
			cloned = append(cloned[:i], cloned[i+1:]...)
		}
	}
	return cloned
}

func classicJSObjectSetterStorageKey(name string) string {
	return "\x00classic-js-setter:" + name
}

func appendClassicJSObjectLiteralEntry(entries []ObjectEntry, name string, value Value) []ObjectEntry {
	if value.Kind == ValueKindFunction && value.Function != nil {
		if value.Function.objectSetter {
			return appendClassicJSObjectLiteralSetter(entries, name, value)
		}
		if value.Function.objectAccessor {
			return appendClassicJSObjectLiteralGetter(entries, name, value)
		}
	}
	return appendClassicJSObjectLiteralData(entries, name, value)
}

func appendClassicJSObjectLiteralData(entries []ObjectEntry, name string, value Value) []ObjectEntry {
	filtered := make([]ObjectEntry, 0, len(entries)+1)
	setterKey := classicJSObjectSetterStorageKey(name)
	for _, entry := range entries {
		if entry.Key == name || entry.Key == setterKey {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, ObjectEntry{Key: name, Value: value})
}

func appendClassicJSObjectLiteralGetter(entries []ObjectEntry, name string, value Value) []ObjectEntry {
	filtered := make([]ObjectEntry, 0, len(entries)+1)
	for _, entry := range entries {
		if entry.Key == name {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, ObjectEntry{Key: name, Value: value})
}

func appendClassicJSObjectLiteralSetter(entries []ObjectEntry, name string, value Value) []ObjectEntry {
	filtered := make([]ObjectEntry, 0, len(entries)+1)
	setterKey := classicJSObjectSetterStorageKey(name)
	for _, entry := range entries {
		if entry.Key == setterKey {
			continue
		}
		if entry.Key == name {
			if entry.Value.Kind == ValueKindFunction && entry.Value.Function != nil && entry.Value.Function.objectAccessor {
				filtered = append(filtered, entry)
			}
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, ObjectEntry{Key: setterKey, Value: value})
}

func (p *classicJSStatementParser) classicJSObjectSpreadEntriesFromValue(value Value) ([]ObjectEntry, error) {
	switch value.Kind {
	case ValueKindUndefined, ValueKindNull:
		return nil, nil
	case ValueKindObject:
		if len(value.Object) == 0 {
			return nil, nil
		}
		cloned := make([]ObjectEntry, 0, len(value.Object))
		for _, entry := range value.Object {
			if classicJSIsInternalObjectKey(entry.Key) {
				continue
			}
			cloned = append(cloned, entry)
		}
		return cloned, nil
	case ValueKindArray:
		if len(value.Array) == 0 {
			return nil, nil
		}
		cloned := make([]ObjectEntry, 0, len(value.Array))
		for i, element := range value.Array {
			cloned = append(cloned, ObjectEntry{Key: strconv.Itoa(i), Value: element})
		}
		return cloned, nil
	case ValueKindString:
		values, err := p.collectClassicJSArrayLikeValues(value, "object spread")
		if err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, nil
		}
		cloned := make([]ObjectEntry, 0, len(values))
		for i, element := range values {
			cloned = append(cloned, ObjectEntry{Key: strconv.Itoa(i), Value: element})
		}
		return cloned, nil
	default:
		return nil, nil
	}
}

func replaceObjectProperty(entries []ObjectEntry, name string, value Value) []ObjectEntry {
	cloned := append([]ObjectEntry(nil), entries...)
	for i := len(cloned) - 1; i >= 0; i-- {
		if cloned[i].Key == name {
			cloned[i].Value = value
			return cloned
		}
	}
	return append(cloned, ObjectEntry{Key: name, Value: value})
}

func assignJSValuePropertyChain(p *classicJSStatementParser, value Value, steps []classicJSDeleteStep, rhs Value, privateFieldPrefix string) (Value, error) {
	if len(steps) == 0 {
		return rhs, nil
	}

	key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}
	switch value.Kind {
	case ValueKindObject:
		if len(steps) == 1 {
			if key == "prototype" && (value.ClassDefinition != nil || value.ClassKey != "") && classicJSClassPrototypeSetterOnly(value) {
				setterKey := classicJSObjectSetterStorageKey(key)
				setterValue, hasSetter := lookupObjectProperty(value.Object, setterKey)
				if hasSetter && setterValue.Kind == ValueKindFunction && setterValue.Function != nil {
					callable := scalarJSValue(setterValue)
					callable.receiver = value
					callable.hasReceiver = true
					if _, err := p.invoke(callable, []Value{rhs}); err != nil {
						return UndefinedValue(), err
					}
					return value, nil
				}
			}
			setterKey := classicJSObjectSetterStorageKey(key)
			setterValue, hasSetter := lookupObjectProperty(value.Object, setterKey)
			index := findObjectPropertyIndex(value.Object, key)
			if index < 0 {
				if hasSetter && setterValue.Kind == ValueKindFunction && setterValue.Function != nil {
					callable := scalarJSValue(setterValue)
					callable.receiver = value
					callable.hasReceiver = true
					if _, err := p.invoke(callable, []Value{rhs}); err != nil {
						return UndefinedValue(), err
					}
					return value, nil
				}
				if classicJSObjectHasInstanceMarker(value) {
					updated := ObjectValue(replaceObjectProperty(value.Object, key, rhs))
					p.replaceObjectBindings(value, scalarJSValue(updated))
					return updated, nil
				}
				return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing object properties in this bounded classic-JS slice")
			}
			current := value.Object[index].Value
			if current.Kind == ValueKindFunction && current.Function != nil && current.Function.objectAccessor {
				if hasSetter && setterValue.Kind == ValueKindFunction && setterValue.Function != nil {
					callable := scalarJSValue(setterValue)
					callable.receiver = value
					callable.hasReceiver = true
					if _, err := p.invoke(callable, []Value{rhs}); err != nil {
						return UndefinedValue(), err
					}
					return value, nil
				}
				return UndefinedValue(), NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
			}
			value.Object[index].Value = rhs
			return value, nil
		}
		if key == "prototype" && (value.ClassDefinition != nil || value.ClassKey != "") && classicJSClassPrototypeSetterOnly(value) {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing object properties in this bounded classic-JS slice")
		}

		child, ok := lookupObjectProperty(value.Object, key)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing object properties in this bounded classic-JS slice")
		}
		if child.Kind == ValueKindFunction && child.Function != nil && child.Function.objectAccessor {
			return UndefinedValue(), NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
		}
		updatedChild, err := assignJSValuePropertyChain(p, child, steps[1:], rhs, privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), err
		}
		index := findObjectPropertyIndex(value.Object, key)
		if index < 0 {
			return UndefinedValue(), NewError(ErrorKindRuntime, "assignment target disappeared during property chain update")
		}
		value.Object[index].Value = updatedChild
		return value, nil
	case ValueKindArray:
		if len(steps) == 1 {
			if key == "length" {
				newLength, err := classicJSArrayLengthFromValue(rhs)
				if err != nil {
					return UndefinedValue(), err
				}
				if newLength < len(value.Array) {
					updated := value.Array[:newLength]
					updatedValue := ArrayValue(updated)
					p.replaceArrayBindings(value, scalarJSValue(updatedValue))
					return updatedValue, nil
				}
				if newLength == len(value.Array) {
					return value, nil
				}
				updated := append([]Value(nil), value.Array...)
				for len(updated) < newLength {
					updated = append(updated, UndefinedValue())
				}
				updatedValue := ArrayValue(updated)
				p.replaceArrayBindings(value, scalarJSValue(updatedValue))
				return updatedValue, nil
			}
			index, ok := arrayIndexFromBracketKey(key)
			if !ok {
				return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing array elements or array length in this bounded classic-JS slice")
			}
			if index < len(value.Array) {
				value.Array[index] = rhs
				return value, nil
			}
			updated := append([]Value(nil), value.Array...)
			for len(updated) < index {
				updated = append(updated, UndefinedValue())
			}
			if index == len(updated) {
				updated = append(updated, rhs)
			} else {
				updated[index] = rhs
			}
			updatedValue := ArrayValue(updated)
			p.replaceArrayBindings(value, scalarJSValue(updatedValue))
			return updatedValue, nil
		}
		if key == "length" {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing array elements in this bounded classic-JS slice")
		}
		index, ok := arrayIndexFromBracketKey(key)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing array elements or array length in this bounded classic-JS slice")
		}
		if index >= len(value.Array) {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing array elements in this bounded classic-JS slice")
		}
		child := value.Array[index]
		updatedChild, err := assignJSValuePropertyChain(p, child, steps[1:], rhs, privateFieldPrefix)
		if err != nil {
			return UndefinedValue(), err
		}
		value.Array[index] = updatedChild
		return value, nil
	case ValueKindHostReference:
		if len(steps) == 1 {
			if mutator, ok := p.host.(HostReferenceMutator); ok {
				if err := mutator.SetHostReference(joinHostReferencePath(value.HostReferencePath, key), rhs); err != nil {
					return UndefinedValue(), err
				}
				return rhs, nil
			}
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
		}
		resolved, err := p.resolveHostReferencePath(joinHostReferencePath(value.HostReferencePath, key))
		if err != nil {
			return UndefinedValue(), err
		}
		if resolved.Kind != ValueKindObject && resolved.Kind != ValueKindArray && resolved.Kind != ValueKindHostReference {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
		}
		return assignJSValuePropertyChain(p, resolved, steps[1:], rhs, privateFieldPrefix)
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object, array, or host surface values in this bounded classic-JS slice")
	}
}

func assignSuperJSValuePropertyChain(p *classicJSStatementParser, superValue jsValue, steps []classicJSDeleteStep, rhs Value, privateFieldPrefix string) (Value, error) {
	if len(steps) == 0 {
		return rhs, nil
	}
	if superValue.kind != jsValueSuper {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "unsupported `super` assignment in this bounded classic-JS slice")
	}
	if superValue.value.Kind != ValueKindObject && superValue.value.Kind != ValueKindNull {
		return UndefinedValue(), NewError(
			ErrorKindUnsupported,
			"unsupported `super` assignment in this bounded classic-JS slice; only object-backed class targets are available",
		)
	}

	key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}

	if len(steps) == 1 {
		if superValue.value.Kind == ValueKindNull {
			if superValue.receiver.Kind != ValueKindObject {
				return UndefinedValue(), NewError(
					ErrorKindUnsupported,
					"unsupported `super` assignment in this bounded classic-JS slice; only object-backed class targets are available",
				)
			}
			index := findObjectPropertyIndex(superValue.receiver.Object, key)
			if index < 0 {
				updatedReceiver := replaceObjectProperty(superValue.receiver.Object, key, rhs)
				updatedReceiverJSValue := scalarJSValue(ObjectValue(updatedReceiver))
				p.replaceObjectBindings(superValue.receiver, updatedReceiverJSValue)
				superValue.receiver = updatedReceiverJSValue.value
				return rhs, nil
			}
			current := superValue.receiver.Object[index].Value
			if current.Kind == ValueKindFunction && current.Function != nil && current.Function.objectAccessor {
				receiverSetterKey := classicJSObjectSetterStorageKey(key)
				receiverSetterValue, receiverHasSetter := lookupObjectProperty(superValue.receiver.Object, receiverSetterKey)
				if receiverHasSetter && receiverSetterValue.Kind == ValueKindFunction && receiverSetterValue.Function != nil {
					callable := scalarJSValue(receiverSetterValue)
					callable.receiver = superValue.receiver
					callable.hasReceiver = true
					if _, err := p.invoke(callable, []Value{rhs}); err != nil {
						return UndefinedValue(), err
					}
					return rhs, nil
				}
				return UndefinedValue(), NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
			}
			superValue.receiver.Object[index].Value = rhs
			return rhs, nil
		}
		setterKey := classicJSObjectSetterStorageKey(key)
		setterValue, hasSetter := lookupObjectProperty(superValue.value.Object, setterKey)
		if hasSetter && setterValue.Kind == ValueKindFunction && setterValue.Function != nil {
			callable := scalarJSValue(setterValue)
			callable.receiver = superValue.receiver
			callable.hasReceiver = true
			if _, err := p.invoke(callable, []Value{rhs}); err != nil {
				return UndefinedValue(), err
			}
			return rhs, nil
		}
		baseValue, hasBaseValue := lookupObjectProperty(superValue.value.Object, key)
		if hasBaseValue && baseValue.Kind == ValueKindFunction && baseValue.Function != nil && baseValue.Function.objectAccessor {
			return UndefinedValue(), NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
		}
		if superValue.receiver.Kind != ValueKindObject {
			return UndefinedValue(), NewError(
				ErrorKindUnsupported,
				"unsupported `super` assignment in this bounded classic-JS slice; only object-backed class targets are available",
			)
		}
		index := findObjectPropertyIndex(superValue.receiver.Object, key)
		if index < 0 {
			updatedReceiver := replaceObjectProperty(superValue.receiver.Object, key, rhs)
			updatedReceiverJSValue := scalarJSValue(ObjectValue(updatedReceiver))
			p.replaceObjectBindings(superValue.receiver, updatedReceiverJSValue)
			superValue.receiver = updatedReceiverJSValue.value
			return rhs, nil
		}
		current := superValue.receiver.Object[index].Value
		if current.Kind == ValueKindFunction && current.Function != nil && current.Function.objectAccessor {
			receiverSetterKey := classicJSObjectSetterStorageKey(key)
			receiverSetterValue, receiverHasSetter := lookupObjectProperty(superValue.receiver.Object, receiverSetterKey)
			if receiverHasSetter && receiverSetterValue.Kind == ValueKindFunction && receiverSetterValue.Function != nil {
				callable := scalarJSValue(receiverSetterValue)
				callable.receiver = superValue.receiver
				callable.hasReceiver = true
				if _, err := p.invoke(callable, []Value{rhs}); err != nil {
					return UndefinedValue(), err
				}
				return rhs, nil
			}
			return UndefinedValue(), NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
		}
		superValue.receiver.Object[index].Value = rhs
		return rhs, nil
	}

	child, ok := lookupObjectProperty(superValue.value.Object, key)
	if !ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing object properties in this bounded classic-JS slice")
	}
	if child.Kind == ValueKindFunction && child.Function != nil && child.Function.objectAccessor {
		return UndefinedValue(), NewError(ErrorKindRuntime, "assignment cannot write to getter-only property in this bounded classic-JS slice")
	}
	updatedChild, err := assignJSValuePropertyChain(p, child, steps[1:], rhs, privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}
	index := findObjectPropertyIndex(superValue.value.Object, key)
	if index < 0 {
		return UndefinedValue(), NewError(ErrorKindRuntime, "assignment target disappeared during property chain update")
	}
	superValue.value.Object[index].Value = updatedChild
	return superValue.value, nil
}

func findObjectPropertyIndex(entries []ObjectEntry, name string) int {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Key == name {
			return i
		}
	}
	return -1
}

func arrayIndexFromBracketKey(key string) (int, bool) {
	if key == "" {
		return 0, false
	}
	if len(key) > 1 && key[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(key); i++ {
		if key[i] < '0' || key[i] > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(key)
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
}

func (p *classicJSStatementParser) consumeBracketAccessExpressionSource() (string, error) {
	start := p.pos
	var quote byte
	var escape bool
	var lineComment bool
	var blockComment bool
	var parenDepth int
	var braceDepth int
	var bracketDepth int

	for !p.eof() {
		ch := p.peekByte()
		if lineComment {
			p.pos++
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '/' {
				p.pos += 2
				blockComment = false
				continue
			}
			p.pos++
			continue
		}
		if quote != 0 {
			p.pos++
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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && ch == ']' {
			source := strings.TrimSpace(p.source[start:p.pos])
			p.pos++
			return source, nil
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
			p.pos++
		case '/':
			if p.pos+1 >= len(p.source) {
				p.pos++
				continue
			}
			switch p.source[p.pos+1] {
			case '/':
				lineComment = true
				p.pos += 2
			case '*':
				blockComment = true
				p.pos += 2
			default:
				p.pos++
			}
		case '(':
			parenDepth++
			p.pos++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			p.pos++
		case '{':
			braceDepth++
			p.pos++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			p.pos++
		case '[':
			bracketDepth++
			p.pos++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			p.pos++
		default:
			p.pos++
		}
	}

	if quote != 0 {
		return "", NewError(ErrorKindParse, "unterminated quoted string in bracket access expression")
	}
	if blockComment {
		return "", NewError(ErrorKindParse, "unterminated block comment in bracket access expression")
	}
	return "", NewError(ErrorKindParse, "unterminated bracket access expression")
}

func (p *classicJSStatementParser) skipOptionalCallArguments() error {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return nil
	}

	skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
	skip.pos = p.pos
	if _, err := skip.parseArguments(); err != nil {
		return err
	}
	p.pos = skip.pos
	return nil
}

func (p *classicJSStatementParser) parsePrimary() (jsValue, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return jsValue{}, NewError(ErrorKindParse, "unexpected end of script source")
	}

	if p.peekByte() == '#' {
		if p.privateFieldPrefix == "" {
			return jsValue{}, NewError(ErrorKindParse, "private identifiers are only supported inside bounded class bodies")
		}
		p.pos++
		ident, err := p.parseIdentifier()
		if err != nil {
			return jsValue{}, NewError(ErrorKindParse, "private identifiers require an identifier name")
		}
		return scalarJSValue(PrivateNameValue(ident)), nil
	}

	if value, ok, err := p.tryParseArrowFunction(); err != nil {
		return jsValue{}, err
	} else if ok {
		return value, nil
	}
	if value, ok, err := p.tryParseGeneratorFunction(); err != nil {
		return jsValue{}, err
	} else if ok {
		return value, nil
	}
	if value, ok, err := p.tryParseFunctionExpression(); err != nil {
		return jsValue{}, err
	} else if ok {
		return value, nil
	}

	switch ch := p.peekByte(); ch {
	case '\'', '"':
		value, err := p.parseStringLiteral()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	case '`':
		value, err := p.parseTemplateLiteral()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	case '/':
		value, err := p.parseRegularExpressionLiteral()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	case '[':
		value, err := p.parseArrayLiteral()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	case '{':
		value, err := p.parseObjectLiteral()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	case '(':
		p.pos++
		value, err := p.parseExpression()
		if err != nil {
			return jsValue{}, err
		}
		p.skipSpaceAndComments()
		if !p.consumeByte(')') {
			return jsValue{}, NewError(ErrorKindParse, "unterminated parenthesized expression")
		}
		return scalarJSValue(value), nil
	}

	if isDigit(p.peekByte()) {
		value, err := p.parseNumberLiteral()
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	}

	if keyword, ok := p.peekKeyword("class"); ok {
		p.pos += len(keyword)
		_, value, err := p.parseClassDeclarationWithBinding(true, false)
		if err != nil {
			return jsValue{}, err
		}
		return scalarJSValue(value), nil
	}

	ident, err := p.parseIdentifier()
	if err != nil {
		return jsValue{}, err
	}

	if ident == "import" {
		p.skipSpaceAndComments()
		if !p.eof() && p.peekByte() == '(' {
			p.pos++
			args, err := p.parseArguments()
			if err != nil {
				return jsValue{}, err
			}
			if len(args) == 0 || len(args) > 2 {
				return jsValue{}, NewError(ErrorKindUnsupported, "dynamic import() requires one module specifier and an optional attributes object in this bounded classic-JS slice")
			}
			if len(args) == 2 && args[1].Kind != ValueKindObject {
				return jsValue{}, NewError(ErrorKindRuntime, "dynamic import() optional attributes must be an object in this bounded classic-JS slice")
			}
			specifier := ToJSString(args[0])
			module, err := p.lookupModuleNamespace(specifier)
			if err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(PromiseValue(module)), nil
		}
		if p.peekByte() == '.' {
			p.pos++
			meta, err := p.parseIdentifier()
			if err != nil {
				return jsValue{}, err
			}
			if meta != "meta" {
				return jsValue{}, NewError(ErrorKindUnsupported, "unsupported `import` member access in this bounded classic-JS slice")
			}
			if p.env == nil {
				return jsValue{}, NewError(ErrorKindParse, "`import.meta` is only supported inside bounded module scripts in this bounded classic-JS slice")
			}
			moduleURL, ok := p.env.lookup(ClassicJSModuleMetaURLBindingName)
			if !ok || moduleURL.kind != jsValueScalar || moduleURL.value.Kind != ValueKindString {
				return jsValue{}, NewError(ErrorKindParse, "`import.meta` is only supported inside bounded module scripts in this bounded classic-JS slice")
			}
			return scalarJSValue(ObjectValue([]ObjectEntry{{Key: "url", Value: moduleURL.value}})), nil
		}
	}

	switch ident {
	case "host":
		return hostObjectJSValue(), nil
	case "expr":
		return builtinExprJSValue(), nil
	case "true":
		return scalarJSValue(BoolValue(true)), nil
	case "false":
		return scalarJSValue(BoolValue(false)), nil
	case "undefined":
		return scalarJSValue(UndefinedValue()), nil
	case "null":
		return scalarJSValue(NullValue()), nil
	}

	if p.env != nil {
		if value, ok := p.env.lookup(ident); ok {
			return value.withAssignTarget(ident, nil), nil
		}
	}
	if ident == "this" {
		return scalarJSValue(UndefinedValue()), nil
	}
	if ident == "super" {
		return jsValue{}, NewError(ErrorKindParse, "`super` is only supported inside bounded class and object literal methods in this slice")
	}
	if p.allowUnknownIdentifiers {
		return scalarJSValue(UndefinedValue()), nil
	}

	switch ident {
	case "let", "const", "var", "function", "class", "if", "else", "for", "while", "do", "switch", "case", "default", "try", "catch", "finally", "return", "break", "continue", "throw", "debugger", "async", "await", "import", "export", "delete", "yield", "void", "in", "instanceof":
		if ident == "yield" {
			return jsValue{}, NewError(ErrorKindParse, "`yield` is only supported inside bounded generator bodies in this slice")
		}
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported script syntax %q in this bounded classic-JS slice", ident))
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported identifier %q in this bounded classic-JS slice", ident))
	}
}

func (p *classicJSStatementParser) parseArrayLiteral() (Value, error) {
	if p.eof() {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
	}

	p.pos++
	elements := make([]Value, 0, 4)
	for {
		p.skipSpaceAndComments()
		if p.consumeByte(']') {
			return ArrayValue(elements), nil
		}
		if p.consumeByte(',') {
			elements = append(elements, UndefinedValue())
			continue
		}
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			value, err := p.parseScalarExpression()
			if err != nil {
				return UndefinedValue(), err
			}
			spreadValues, err := p.collectClassicJSArrayLikeValues(value, "array spread")
			if err != nil {
				return UndefinedValue(), err
			}
			elements = append(elements, spreadValues...)
			p.skipSpaceAndComments()
			if p.consumeByte(']') {
				return ArrayValue(elements), nil
			}
			if !p.consumeByte(',') {
				return UndefinedValue(), NewError(ErrorKindParse, "array literals must separate elements with commas")
			}
			continue
		}

		value, err := p.parseScalarExpression()
		if err != nil {
			return UndefinedValue(), err
		}
		elements = append(elements, value)

		p.skipSpaceAndComments()
		if p.consumeByte(']') {
			return ArrayValue(elements), nil
		}
		if !p.consumeByte(',') {
			return UndefinedValue(), NewError(ErrorKindParse, "array literals must separate elements with commas")
		}
	}
}

func (p *classicJSStatementParser) parseObjectLiteral() (Value, error) {
	if p.eof() {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
	}

	p.pos++
	entries := make([]ObjectEntry, 0, 4)
	superTarget := classicJSObjectLiteralDefaultSuperTarget()
	for {
		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return p.finalizeClassicJSObjectLiteral(entries, superTarget), nil
		}
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			value, err := p.parseScalarExpression()
			if err != nil {
				return UndefinedValue(), err
			}
			spreadEntries, err := p.classicJSObjectSpreadEntriesFromValue(value)
			if err != nil {
				return UndefinedValue(), err
			}
			entries = append(entries, spreadEntries...)
			p.skipSpaceAndComments()
			if p.consumeByte('}') {
				return p.finalizeClassicJSObjectLiteral(entries, superTarget), nil
			}
			if !p.consumeByte(',') {
				return UndefinedValue(), NewError(ErrorKindParse, "object literals must separate properties with commas")
			}
			continue
		}

		key, methodValue, ok, err := tryParseClassicJSObjectLiteralMethod(p)
		if err != nil {
			return UndefinedValue(), err
		}
		if ok {
			entries = appendClassicJSObjectLiteralEntry(entries, key, methodValue)
			p.skipSpaceAndComments()
			if p.consumeByte('}') {
				return p.finalizeClassicJSObjectLiteral(entries, superTarget), nil
			}
			if !p.consumeByte(',') {
				return UndefinedValue(), NewError(ErrorKindParse, "object literals must separate properties with commas")
			}
			continue
		}

		key, shorthand, err := p.parseObjectLiteralKey()
		if err != nil {
			return UndefinedValue(), err
		}

		p.skipSpaceAndComments()
		var value Value
		if p.consumeByte(':') {
			value, err = p.parseScalarExpression()
			if err != nil {
				return UndefinedValue(), err
			}
			if key == "__proto__" && (value.Kind == ValueKindObject || value.Kind == ValueKindNull) {
				superTarget = value
				p.skipSpaceAndComments()
				if p.consumeByte('}') {
					return p.finalizeClassicJSObjectLiteral(entries, superTarget), nil
				}
				if !p.consumeByte(',') {
					return UndefinedValue(), NewError(ErrorKindParse, "object literals must separate properties with commas")
				}
				continue
			}
		} else {
			switch {
			case p.peekByte() == '(':
				paramsSource, err := p.consumeParenthesizedSource("object method")
				if err != nil {
					return UndefinedValue(), err
				}
				params, restName, err := parseClassicJSFunctionParameters(paramsSource, "object method")
				if err != nil {
					return UndefinedValue(), err
				}
				bodySource, err := p.consumeBlockSource()
				if err != nil {
					return UndefinedValue(), err
				}
				value = FunctionValue(&classicJSArrowFunction{
					name:               key,
					params:             params,
					restName:           restName,
					body:               bodySource,
					bodyIsBlock:        true,
					allowReturn:        true,
					objectMethod:       true,
					env:                p.env,
					privateClass:       p.privateClass,
					privateFieldPrefix: p.privateFieldPrefix,
				})
			case shorthand && (key == "get" || key == "set") && isObjectLiteralAccessorKeyStart(p.peekByte()):
				var accessorKey string
				accessorKey, value, err = p.parseObjectLiteralAccessor(key)
				if err != nil {
					return UndefinedValue(), err
				}
				key = accessorKey
			case shorthand && (p.peekByte() == '}' || p.peekByte() == ','):
				value, err = p.resolveObjectLiteralShorthandValue(key)
				if err != nil {
					return UndefinedValue(), err
				}
			case shorthand && (isIdentStart(p.peekByte()) || p.peekByte() == '['):
				return UndefinedValue(), NewError(ErrorKindParse, "object literals must separate properties with commas")
			default:
				return UndefinedValue(), NewError(ErrorKindParse, "object literal properties must use `:` or method syntax")
			}
		}
		entries = appendClassicJSObjectLiteralEntry(entries, key, value)

		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return p.finalizeClassicJSObjectLiteral(entries, superTarget), nil
		}
		if !p.consumeByte(',') {
			return UndefinedValue(), NewError(ErrorKindParse, "object literals must separate properties with commas")
		}
	}
}

func tryParseClassicJSObjectLiteralMethod(scanner *classicJSStatementParser) (string, Value, bool, error) {
	if scanner == nil {
		return "", UndefinedValue(), false, nil
	}

	lookahead := *scanner
	async := false
	if keyword, ok := lookahead.peekKeyword("async"); ok {
		lookahead.pos += len(keyword)
		lookahead.skipSpaceAndComments()
		async = true
	}

	generator := false
	if lookahead.peekByte() == '*' {
		lookahead.pos++
		lookahead.skipSpaceAndComments()
		generator = true
	}

	if !async && !generator {
		return "", UndefinedValue(), false, nil
	}
	if lookahead.eof() {
		return "", UndefinedValue(), false, nil
	}
	if lookahead.peekByte() == '#' {
		return "", UndefinedValue(), false, NewError(ErrorKindParse, "object literal methods do not support private names in this bounded classic-JS slice")
	}

	key, _, err := lookahead.parseObjectLiteralKey()
	if err != nil {
		return "", UndefinedValue(), false, nil
	}
	lookahead.skipSpaceAndComments()
	if lookahead.peekByte() != '(' {
		return "", UndefinedValue(), false, nil
	}

	paramsSource, err := lookahead.consumeParenthesizedSource("object method")
	if err != nil {
		return "", UndefinedValue(), false, err
	}
	params, restName, err := parseClassicJSFunctionParameters(paramsSource, "object method")
	if err != nil {
		return "", UndefinedValue(), false, err
	}
	bodySource, err := lookahead.consumeBlockSource()
	if err != nil {
		return "", UndefinedValue(), false, err
	}

	scanner.pos = lookahead.pos
	callable := &classicJSArrowFunction{
		name:               key,
		params:             params,
		restName:           restName,
		body:               bodySource,
		bodyIsBlock:        true,
		async:              async,
		allowReturn:        true,
		objectMethod:       true,
		env:                scanner.env,
		privateClass:       scanner.privateClass,
		privateFieldPrefix: scanner.privateFieldPrefix,
	}
	if generator {
		callable.generatorFunction = &classicJSGeneratorFunction{
			name:               key,
			params:             params,
			restName:           restName,
			body:               bodySource,
			async:              async,
			env:                scanner.env,
			privateClass:       scanner.privateClass,
			privateFieldPrefix: scanner.privateFieldPrefix,
		}
	}
	return key, FunctionValue(callable), true, nil
}

func (p *classicJSStatementParser) finalizeClassicJSObjectLiteral(entries []ObjectEntry, superTarget Value) Value {
	for i := range entries {
		if entries[i].Value.Kind != ValueKindFunction || entries[i].Value.Function == nil {
			continue
		}
		if !entries[i].Value.Function.objectMethod {
			continue
		}
		entries[i].Value.Function.setObjectLiteralSuperTarget(superTarget)
	}
	return ObjectValue(entries)
}

func (p *classicJSStatementParser) parseObjectLiteralKey() (string, bool, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return "", false, NewError(ErrorKindParse, "unexpected end of script source")
	}

	switch ch := p.peekByte(); ch {
	case '\'', '"':
		value, err := p.parseStringLiteral()
		if err != nil {
			return "", false, err
		}
		return value.String, false, nil
	default:
		if ch == '[' {
			p.pos++
			source, err := p.consumeBracketAccessExpressionSource()
			if err != nil {
				return "", false, err
			}
			value, err := p.evalExpressionWithEnv(source, p.env)
			if err != nil {
				return "", false, err
			}
			return ToJSString(value), false, nil
		}
		if isDigit(ch) {
			value, err := p.parseNumberLiteral()
			if err != nil {
				return "", false, err
			}
			return ToJSString(value), false, nil
		}
		ident, err := p.parseIdentifier()
		if err != nil {
			return "", false, err
		}
		return ident, true, nil
	}
}

func isObjectLiteralAccessorKeyStart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch) || ch == '\'' || ch == '"' || ch == '['
}

func (p *classicJSStatementParser) parseObjectLiteralAccessor(keyword string) (string, Value, error) {
	propertyName, _, err := p.parseObjectLiteralKey()
	if err != nil {
		return "", UndefinedValue(), err
	}

	p.skipSpaceAndComments()
	paramsSource, err := p.consumeParenthesizedSource("object accessor")
	if err != nil {
		return "", UndefinedValue(), err
	}

	switch keyword {
	case "get":
		if strings.TrimSpace(paramsSource) != "" {
			return "", UndefinedValue(), NewError(ErrorKindParse, "object getter accessors do not accept parameters in this bounded classic-JS slice")
		}
	case "set":
		params, err := parseClassicJSSetterParameters(paramsSource, "object setter")
		if err != nil {
			return "", UndefinedValue(), err
		}
		bodySource, err := p.consumeLoopBodySource()
		if err != nil {
			return "", UndefinedValue(), err
		}
		return propertyName, FunctionValue(&classicJSArrowFunction{
			name:               propertyName,
			params:             params,
			body:               bodySource,
			bodyIsBlock:        true,
			allowReturn:        true,
			objectSetter:       true,
			objectMethod:       true,
			env:                p.env,
			privateClass:       p.privateClass,
			privateFieldPrefix: p.privateFieldPrefix,
		}), nil
	default:
		return "", UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("unsupported object accessor keyword %q in this bounded classic-JS slice", keyword))
	}

	bodySource, err := p.consumeBlockSource()
	if err != nil {
		return "", UndefinedValue(), err
	}
	return propertyName, FunctionValue(&classicJSArrowFunction{
		name:               propertyName,
		body:               bodySource,
		bodyIsBlock:        true,
		allowReturn:        true,
		objectAccessor:     true,
		objectMethod:       true,
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
	}), nil
}

func classicJSObjectLiteralDefaultSuperTarget() Value {
	return ObjectValue([]ObjectEntry{
		{
			Key: "toString",
			Value: NativeFunctionValue(func(args []Value) (Value, error) {
				return StringValue("[object Object]"), nil
			}),
		},
	})
}

func classicJSBaseClassDefaultSuperTarget() Value {
	return classicJSObjectLiteralDefaultSuperTarget()
}

func (p *classicJSStatementParser) resolveObjectLiteralShorthandValue(name string) (Value, error) {
	if p.env != nil {
		if value, ok := p.env.lookup(name); ok {
			return value.value, nil
		}
	}
	if p.allowUnknownIdentifiers {
		return UndefinedValue(), nil
	}
	return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported identifier %q in this bounded classic-JS slice", name))
}

func (f *classicJSArrowFunction) setObjectLiteralSuperTarget(superTarget Value) {
	if f == nil || (superTarget.Kind != ValueKindObject && superTarget.Kind != ValueKindNull) {
		return
	}
	f.superTarget = superTarget
	f.hasSuperTarget = true
	if f.generatorFunction != nil {
		f.generatorFunction.superTarget = superTarget
		f.generatorFunction.hasSuperTarget = true
	}
}

func (p *classicJSStatementParser) parseMemberAccessName() (string, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return "", NewError(ErrorKindParse, "unexpected end of member access")
	}
	if p.peekByte() == '#' {
		p.pos++
		ident, err := p.parseIdentifier()
		if err != nil {
			return "", NewError(ErrorKindParse, "private class field access requires an identifier")
		}
		return "#" + ident, nil
	}
	return p.parseIdentifier()
}

func (p *classicJSStatementParser) invoke(callee jsValue, args []Value) (jsValue, error) {
	switch callee.kind {
	case jsValueHostMethod:
		if p.host == nil {
			return jsValue{}, NewError(ErrorKindHost, "host bindings are unavailable")
		}
		value, err := p.host.Call(callee.method, args)
		if err != nil {
			return jsValue{}, NewError(ErrorKindHost, err.Error())
		}
		return scalarJSValue(value), nil
	case jsValueBuiltinExpr:
		if len(args) != 1 {
			return jsValue{}, NewError(ErrorKindUnsupported, "expr(...) expects exactly one argument in this bounded classic-JS slice")
		}
		return scalarJSValue(args[0]), nil
	case jsValueScalar:
		switch callee.value.Kind {
		case ValueKindUndefined:
			return jsValue{}, NewError(ErrorKindRuntime, "cannot call undefined value in this bounded classic-JS slice")
		case ValueKindNull:
			return jsValue{}, NewError(ErrorKindRuntime, "cannot call null value in this bounded classic-JS slice")
		}
		switch callee.value.Kind {
		case ValueKindFunction:
			if callee.value.NativeFunction != nil {
				value, err := callee.value.NativeFunction(args)
				if err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(value), nil
			}
			if callee.value.Function == nil {
				return jsValue{}, NewError(ErrorKindRuntime, "cannot call non-callable value in this bounded classic-JS slice")
			}
			return p.invokeArrowFunction(callee.value.Function, args, callee)
		case ValueKindHostReference:
			if p.host == nil {
				return jsValue{}, NewError(ErrorKindHost, "host bindings are unavailable")
			}
			resolver, ok := p.host.(HostReferenceResolver)
			if !ok {
				return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", callee.value.HostReferencePath))
			}
			resolved, err := resolver.ResolveHostReference(callee.value.HostReferencePath)
			if err != nil {
				return jsValue{}, err
			}
			if resolved.Kind != ValueKindFunction {
				return jsValue{}, NewError(ErrorKindRuntime, fmt.Sprintf("cannot call non-callable browser surface %q in this bounded classic-JS slice", callee.value.HostReferencePath))
			}
			if resolved.NativeFunction != nil {
				value, err := resolved.NativeFunction(args)
				if err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(value), nil
			}
			if resolved.Function == nil {
				return jsValue{}, NewError(ErrorKindRuntime, fmt.Sprintf("cannot call non-callable browser surface %q in this bounded classic-JS slice", callee.value.HostReferencePath))
			}
			return p.invokeArrowFunction(resolved.Function, args, callee)
		default:
			return jsValue{}, NewError(ErrorKindRuntime, "cannot call non-callable value in this bounded classic-JS slice")
		}
	case jsValueSuper:
		if callee.value.Kind != ValueKindObject {
			return jsValue{}, NewError(ErrorKindRuntime, "super() is only supported in derived class constructors in this bounded classic-JS slice")
		}
		constructorValue, ok := lookupObjectProperty(callee.value.Object, "constructor")
		if !ok || constructorValue.Kind != ValueKindFunction || constructorValue.Function == nil {
			return jsValue{}, NewError(ErrorKindRuntime, "super() requires a constructor on the base target in this bounded classic-JS slice")
		}
		newTarget := UndefinedValue()
		if callee.hasNewTarget {
			newTarget = callee.newTarget
		} else if p.hasNewTarget {
			newTarget = p.newTarget
		}
		constructorCall := scalarJSValue(constructorValue).withNewTarget(newTarget)
		constructorCall.receiver = callee.receiver
		constructorCall.hasReceiver = true
		return p.invoke(constructorCall, args)
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, "unsupported call expression in this bounded classic-JS slice")
	}
}

func (p *classicJSStatementParser) bindClassicJSFunctionParameters(callEnv *classicJSEnvironment, params []classicJSFunctionParameter, restName string, args []Value, allowAwait bool, privateClass *classicJSClassDefinition) error {
	prevEnv := p.env
	p.env = callEnv
	defer func() {
		p.env = prevEnv
	}()

	for _, param := range params {
		if param.hasPattern {
			continue
		}
		if err := callEnv.declare(param.name, scalarJSValue(UndefinedValue()), true); err != nil {
			return err
		}
	}

	for i, param := range params {
		value := UndefinedValue()
		if i < len(args) {
			value = args[i]
		}
		if param.defaultSource != "" && (i >= len(args) || value.Kind == ValueKindUndefined) {
			parsed, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(param.defaultSource, p.host, callEnv, p.stepLimit, allowAwait, false, p.newTarget, p.hasNewTarget, privateClass, nil)
			if err != nil {
				return err
			}
			value = parsed
		}
		if param.hasPattern {
			if err := p.bindBindingPattern(param.pattern, value, "let"); err != nil {
				return err
			}
			continue
		}
		if err := callEnv.assign(param.name, scalarJSValue(value)); err != nil {
			return err
		}
	}

	if restName != "" {
		rest := []Value(nil)
		if len(args) > len(params) {
			rest = append(rest, args[len(params):]...)
		}
		if err := callEnv.declare(restName, scalarJSValue(ArrayValue(rest)), true); err != nil {
			return err
		}
	}

	return nil
}

func (p *classicJSStatementParser) invokeArrowFunction(fn *classicJSArrowFunction, args []Value, callee jsValue) (jsValue, error) {
	if fn == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "arrow function is unavailable")
	}

	if fn.generatorState != nil {
		switch fn.generatorMethod {
		case "", "next":
			return p.invokeGeneratorNext(fn.generatorState, args)
		case "return":
			return p.invokeGeneratorReturn(fn.generatorState, args)
		case "throw":
			return p.invokeGeneratorThrow(fn.generatorState, args)
		default:
			return jsValue{}, NewError(ErrorKindRuntime, "unsupported generator iterator method")
		}
	}
	if fn.generatorFunction != nil {
		return p.invokeGeneratorFunction(fn.generatorFunction, args, callee)
	}

	invocationHasNewTarget := true
	invocationNewTarget := UndefinedValue()
	if fn.isArrow {
		invocationHasNewTarget = fn.hasNewTarget
		invocationNewTarget = fn.newTarget
	} else if callee.hasNewTarget {
		invocationNewTarget = callee.newTarget
	}
	prevHasNewTarget := p.hasNewTarget
	prevNewTarget := p.newTarget
	p.hasNewTarget = invocationHasNewTarget
	p.newTarget = invocationNewTarget
	defer func() {
		p.hasNewTarget = prevHasNewTarget
		p.newTarget = prevNewTarget
	}()

	callEnv := fn.env.clone()
	if fn.name != "" {
		if err := callEnv.declare(fn.name, scalarJSValue(FunctionValue(fn)), false); err != nil {
			return jsValue{}, err
		}
	}
	if callee.hasReceiver {
		if err := callEnv.declare("this", scalarJSValue(callee.receiver), false); err != nil {
			return jsValue{}, err
		}
	}
	if fn.hasSuperTarget && callee.hasReceiver {
		if err := callEnv.declare("super", superJSValue(fn.superTarget, callee.receiver).withNewTarget(p.newTarget), false); err != nil {
			return jsValue{}, err
		}
	}
	if err := p.bindClassicJSFunctionParameters(callEnv, fn.params, fn.restName, args, fn.async, fn.privateClass); err != nil {
		return jsValue{}, err
	}

	prevPrivateClass := p.privateClass
	prevPrivateFieldPrefix := p.privateFieldPrefix
	p.privateClass = fn.privateClass
	p.privateFieldPrefix = fn.privateFieldPrefix
	defer func() {
		p.privateClass = prevPrivateClass
		p.privateFieldPrefix = prevPrivateFieldPrefix
	}()

	constructorResult := func(result jsValue) jsValue {
		if !callee.hasNewTarget || !callee.hasReceiver {
			return result
		}
		if result.kind == jsValueScalar && result.value.Kind == ValueKindObject {
			return result
		}
		if thisValue, ok := callEnv.lookup("this"); ok && thisValue.kind == jsValueScalar && thisValue.value.Kind == ValueKindObject {
			return thisValue
		}
		return result
	}

	if fn.bodyIsBlock {
		_, err := evalClassicJSProgramWithAllowAwaitAndYieldAndExports(fn.body, p.host, callEnv, p.stepLimit, fn.async, false, fn.allowReturn, nil, p.newTarget, p.hasNewTarget, fn.privateClass, nil)
		if err != nil {
			if returnedValue, ok := classicJSReturnSignalValue(err); ok {
				if fn.async {
					return scalarJSValue(PromiseValue(unwrapPromiseValue(returnedValue))), nil
				}
				return constructorResult(scalarJSValue(returnedValue)), nil
			}
			return jsValue{}, err
		}
		if fn.async {
			return scalarJSValue(PromiseValue(UndefinedValue())), nil
		}
		return constructorResult(scalarJSValue(UndefinedValue())), nil
	}

	value, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(fn.body, p.host, callEnv, p.stepLimit, fn.async, false, p.newTarget, p.hasNewTarget, fn.privateClass, nil)
	if err != nil {
		return jsValue{}, err
	}
	if fn.async {
		return scalarJSValue(PromiseValue(unwrapPromiseValue(value))), nil
	}
	return constructorResult(scalarJSValue(value)), nil
}

func (p *classicJSStatementParser) invokeGeneratorFunction(fn *classicJSGeneratorFunction, args []Value, callee jsValue) (jsValue, error) {
	if fn == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "generator function is unavailable")
	}

	invocationHasNewTarget := true
	invocationNewTarget := UndefinedValue()
	if callee.hasNewTarget {
		invocationNewTarget = callee.newTarget
	}
	prevHasNewTarget := p.hasNewTarget
	prevNewTarget := p.newTarget
	p.hasNewTarget = invocationHasNewTarget
	p.newTarget = invocationNewTarget
	defer func() {
		p.hasNewTarget = prevHasNewTarget
		p.newTarget = prevNewTarget
	}()

	callEnv := fn.env.clone()
	if fn.name != "" {
		callable := &classicJSArrowFunction{generatorFunction: fn, env: fn.env, privateClass: fn.privateClass, privateFieldPrefix: fn.privateFieldPrefix}
		if err := callEnv.declare(fn.name, scalarJSValue(FunctionValue(callable)), false); err != nil {
			return jsValue{}, err
		}
	}
	if callee.hasReceiver {
		if err := callEnv.declare("this", scalarJSValue(callee.receiver), false); err != nil {
			return jsValue{}, err
		}
	}
	if fn.hasSuperTarget && callee.hasReceiver {
		if err := callEnv.declare("super", superJSValue(fn.superTarget, callee.receiver).withNewTarget(p.newTarget), false); err != nil {
			return jsValue{}, err
		}
	}
	if err := p.bindClassicJSFunctionParameters(callEnv, fn.params, fn.restName, args, fn.async, fn.privateClass); err != nil {
		return jsValue{}, err
	}

	prevPrivateClass := p.privateClass
	prevPrivateFieldPrefix := p.privateFieldPrefix
	p.privateClass = fn.privateClass
	p.privateFieldPrefix = fn.privateFieldPrefix
	defer func() {
		p.privateClass = prevPrivateClass
		p.privateFieldPrefix = prevPrivateFieldPrefix
	}()

	statements, err := splitScriptStatements(fn.body)
	if err != nil {
		return jsValue{}, NewError(ErrorKindParse, err.Error())
	}
	state := &classicJSGeneratorState{
		statements:   statements,
		env:          callEnv,
		async:        fn.async,
		newTarget:    p.newTarget,
		hasNewTarget: p.hasNewTarget,
	}
	nextFn := &classicJSArrowFunction{
		generatorMethod: "next",
		generatorState:  state,
	}
	returnFn := &classicJSArrowFunction{
		generatorMethod: "return",
		generatorState:  state,
	}
	throwFn := &classicJSArrowFunction{
		generatorMethod: "throw",
		generatorState:  state,
	}
	return scalarJSValue(ObjectValue([]ObjectEntry{
		{Key: "next", Value: FunctionValue(nextFn)},
		{Key: "return", Value: FunctionValue(returnFn)},
		{Key: "throw", Value: FunctionValue(throwFn)},
	})), nil
}

func (p *classicJSStatementParser) resolveClassicJSClassMemberName(name string, nameSource string, classEnv *classicJSEnvironment, thisValue Value, superTarget Value, hasSuper bool) (string, error) {
	if strings.TrimSpace(nameSource) == "" {
		return name, nil
	}
	if classEnv == nil {
		return "", NewError(ErrorKindRuntime, "class environment is unavailable")
	}
	classEval := p.cloneForClassEvaluation()
	memberEnv := classEnv.clone()
	_ = memberEnv.declare("this", scalarJSValue(thisValue), false)
	if hasSuper {
		_ = memberEnv.declare("super", superJSValue(superTarget, thisValue), false)
	}
	value, err := classEval.evalExpressionWithEnv(nameSource, memberEnv)
	if err != nil {
		return "", err
	}
	return ToJSString(value), nil
}

func (p *classicJSStatementParser) instantiateClassicJSClass(name string, classValue Value, args []Value) (jsValue, error) {
	if classValue.Kind != ValueKindObject {
		if name == "" {
			return jsValue{}, NewError(ErrorKindUnsupported, "new expressions require a class object binding in this bounded classic-JS slice")
		}
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("new expressions require a class object binding for %q in this bounded classic-JS slice", name))
	}

	classDef, ok := resolveClassicJSClassDefinition(classValue, p.env)
	if !ok || classDef == nil {
		if name == "" {
			return jsValue{}, NewError(ErrorKindUnsupported, "new expressions only work on class expressions or declared class identifiers in this bounded classic-JS slice")
		}
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("new expressions only work on declared class identifiers like %q in this bounded classic-JS slice", name))
	}

	prototypeValue, ok := classicJSClassPrototypeValue(classValue)
	if !ok || prototypeValue.Kind != ValueKindObject {
		if name == "" {
			return jsValue{}, NewError(ErrorKindUnsupported, "class expression does not expose a prototype object in this bounded classic-JS slice")
		}
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("class %q does not expose a prototype object in this bounded classic-JS slice", name))
	}
	constructorValue, hasConstructor := lookupObjectProperty(prototypeValue.Object, "constructor")

	instanceEntries := append([]ObjectEntry(nil), prototypeValue.Object...)
	instanceEnv := classDef.env
	if instanceEnv == nil {
		instanceEnv = p.env
	}
	if instanceEnv == nil {
		instanceEnv = newClassicJSEnvironment()
	}

	for _, field := range classDef.instanceFields {
		value := UndefinedValue()
		if strings.TrimSpace(field.init) != "" {
			currentInstanceValue := ObjectValue(append([]ObjectEntry(nil), instanceEntries...))
			fieldEnv := field.env
			if fieldEnv == nil {
				fieldEnv = classDef.env
			}
			if fieldEnv == nil {
				fieldEnv = instanceEnv
			}
			fieldEvalEnv := fieldEnv.clone()
			if err := fieldEvalEnv.declare("this", scalarJSValue(currentInstanceValue), false); err != nil {
				return jsValue{}, err
			}
			if classDef.hasSuper {
				if err := fieldEvalEnv.declare("super", superJSValue(classDef.superInstanceTarget, currentInstanceValue), false); err != nil {
					return jsValue{}, err
				}
			}
			classEval := p.cloneForClassEvaluation()
			classEval.privateClass = classDef
			classEval.privateFieldPrefix = classDef.privateFieldPrefix
			if field.privateKeyPrefix != "" && field.privateKeyPrefix != classDef.privateFieldPrefix {
				classEval.privateClass = &classicJSClassDefinition{privateFieldPrefix: field.privateKeyPrefix}
				classEval.privateFieldPrefix = field.privateKeyPrefix
			}
			parsed, err := classEval.evalExpressionWithEnv(field.init, fieldEvalEnv)
			if err != nil {
				return jsValue{}, err
			}
			value = parsed
		}
		key := field.name
		if field.private {
			if field.privateKeyPrefix != "" {
				key = field.privateKeyPrefix + field.name
			} else {
				key = classDef.privateFieldKey(field.name)
			}
		}
		instanceEntries = append(instanceEntries, ObjectEntry{Key: key, Value: value})
	}

	instanceValue := ObjectValue(instanceEntries)

	if hasConstructor && constructorValue.Kind == ValueKindFunction && constructorValue.Function != nil {
		constructorCall := scalarJSValue(constructorValue).withNewTarget(constructorValue)
		constructorCall.receiver = instanceValue
		constructorCall.hasReceiver = true
		value, err := p.invoke(constructorCall, args)
		if err != nil {
			return jsValue{}, err
		}
		if value.kind == jsValueScalar && value.value.Kind == ValueKindObject {
			return value, nil
		}
	}
	if !hasConstructor && classDef.hasSuper {
		if superConstructor, ok := lookupObjectProperty(classDef.superInstanceTarget.Object, "constructor"); ok && superConstructor.Kind == ValueKindFunction && superConstructor.Function != nil {
			constructorCall := scalarJSValue(superConstructor).withNewTarget(superConstructor)
			constructorCall.receiver = instanceValue
			constructorCall.hasReceiver = true
			value, err := p.invoke(constructorCall, args)
			if err != nil {
				return jsValue{}, err
			}
			if value.kind == jsValueScalar && value.value.Kind == ValueKindObject {
				return value, nil
			}
		}
	}

	return scalarJSValue(instanceValue), nil
}

func (p *classicJSStatementParser) resumeGeneratorState(state *classicJSGeneratorState) (Value, bool, error) {
	if state == nil {
		return UndefinedValue(), false, NewError(ErrorKindRuntime, "generator state is unavailable")
	}
	if state.activeState == nil {
		return UndefinedValue(), false, nil
	}

	resumeParser := *p
	resumeParser.allowAwait = state.async
	resumeParser.allowYield = true
	resumeParser.allowReturn = false

	value, nextState, err := resumeParser.resumeClassicJSState(state.activeState)
	if err != nil {
		return UndefinedValue(), false, err
	}
	if nextState != nil {
		state.activeState = nextState
		return value, true, nil
	}

	state.activeState = nil
	return value, false, nil
}

func (p *classicJSStatementParser) invokeGeneratorNext(state *classicJSGeneratorState, args []Value) (jsValue, error) {
	if state == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "generator state is unavailable")
	}
	prevHasNewTarget := p.hasNewTarget
	prevNewTarget := p.newTarget
	p.hasNewTarget = state.hasNewTarget
	p.newTarget = state.newTarget
	defer func() {
		p.hasNewTarget = prevHasNewTarget
		p.newTarget = prevNewTarget
		p.generatorNextValue = UndefinedValue()
		p.hasGeneratorNextValue = false
	}()
	wrapResult := func(value Value, done bool) jsValue {
		if state.async {
			return scalarJSValue(PromiseValue(generatorIteratorResultValue(value, done).value))
		}
		return generatorIteratorResultValue(value, done)
	}
	recordYield := func(value Value) jsValue {
		state.hasYielded = true
		return wrapResult(value, false)
	}
	if state.hasYielded && len(args) > 0 {
		p.generatorNextValue = args[0]
		p.hasGeneratorNextValue = true
	}
	if state.done {
		return wrapResult(UndefinedValue(), true), nil
	}

	for state.index < len(state.statements) {
		if value, yielded, exhausted, err := p.resumeGeneratorDelegate(state); err != nil {
			return jsValue{}, err
		} else if yielded {
			return recordYield(value), nil
		} else if exhausted {
			continue
		}

		if state.activeState != nil {
			value, yielded, err := p.resumeGeneratorState(state)
			if err != nil {
				return jsValue{}, err
			}
			if yielded {
				return recordYield(value), nil
			}
			state.index++
			continue
		}

		statement := strings.TrimSpace(state.statements[state.index])
		if statement == "" {
			state.index++
			continue
		}

		if yieldSource, ok, delegated, err := splitGeneratorYieldStatement(statement); err != nil {
			return jsValue{}, err
		} else if ok {
			value := UndefinedValue()
			if yieldSource != "" {
				value, err = evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(yieldSource, p.host, state.env, p.stepLimit, state.async, false, p.newTarget, p.hasNewTarget, p.privateClass, nil)
				if err != nil {
					return jsValue{}, err
				}
			}
			state.index++
			if delegated {
				if err := p.beginGeneratorDelegate(state, value); err != nil {
					return jsValue{}, err
				}
				continue
			}
			return recordYield(value), nil
		}

		if _, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(statement, p.host, state.env, p.stepLimit, state.async, true, false, nil, p.newTarget, p.hasNewTarget, nil, nil); err != nil {
			if yieldedValue, resumeState, ok := classicJSYieldSignalDetails(err); ok {
				if resumeState != nil {
					state.activeState = resumeState
					return recordYield(yieldedValue), nil
				}
				state.index++
				return recordYield(yieldedValue), nil
			}
			return jsValue{}, err
		}
		state.index++
	}

	state.done = true
	return wrapResult(UndefinedValue(), true), nil
}

func (p *classicJSStatementParser) invokeGeneratorReturn(state *classicJSGeneratorState, args []Value) (jsValue, error) {
	if state == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "generator state is unavailable")
	}
	prevHasNewTarget := p.hasNewTarget
	prevNewTarget := p.newTarget
	p.hasNewTarget = state.hasNewTarget
	p.newTarget = state.newTarget
	defer func() {
		p.hasNewTarget = prevHasNewTarget
		p.newTarget = prevNewTarget
	}()

	result := UndefinedValue()
	if len(args) > 0 {
		result = args[0]
	}
	p.closeGeneratorState(state)
	if state.async {
		return scalarJSValue(PromiseValue(generatorIteratorResultValue(result, true).value)), nil
	}
	return generatorIteratorResultValue(result, true), nil
}

func (p *classicJSStatementParser) invokeGeneratorThrow(state *classicJSGeneratorState, args []Value) (jsValue, error) {
	if state == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "generator state is unavailable")
	}
	p.closeGeneratorState(state)
	result := UndefinedValue()
	if len(args) > 0 {
		result = args[0]
	}
	return jsValue{}, NewError(ErrorKindRuntime, ToJSString(result))
}

func (p *classicJSStatementParser) closeGeneratorState(state *classicJSGeneratorState) {
	if state == nil {
		return
	}
	state.done = true
	state.index = len(state.statements)
	state.activeState = nil
	state.delegateArray = nil
	state.delegateArrayIndex = 0
	state.delegateIterator = nil
}

func (p *classicJSStatementParser) beginGeneratorDelegate(state *classicJSGeneratorState, value Value) error {
	if state == nil {
		return NewError(ErrorKindRuntime, "generator state is unavailable")
	}

	switch value.Kind {
	case ValueKindArray:
		fallthrough
	case ValueKindString:
		state.hasYielded = true
		values, err := p.collectClassicJSArrayLikeValues(value, "yield*")
		if err != nil {
			return err
		}
		state.delegateArray = values
		state.delegateArrayIndex = 0
		state.delegateIterator = nil
		return nil
	case ValueKindObject:
		nextValue, err := p.resolveMemberAccess(scalarJSValue(value), "next")
		if err != nil {
			return err
		}
		if nextValue.kind != jsValueScalar || !classicJSIsCallableFunctionValue(nextValue.value) {
			return NewError(ErrorKindRuntime, "yield* expects an array or iterator-like object in this bounded classic-JS slice")
		}
		state.hasYielded = true
		copied := value
		state.delegateIterator = &copied
		state.delegateArray = nil
		state.delegateArrayIndex = 0
		return nil
	default:
		return NewError(ErrorKindRuntime, "yield* expects a string, array, or iterator-like object in this bounded classic-JS slice")
	}
}

func (p *classicJSStatementParser) resumeGeneratorDelegate(state *classicJSGeneratorState) (Value, bool, bool, error) {
	if state == nil {
		return UndefinedValue(), false, false, NewError(ErrorKindRuntime, "generator state is unavailable")
	}
	if state.delegateIterator != nil {
		nextValue, err := p.resolveMemberAccess(scalarJSValue(*state.delegateIterator), "next")
		if err != nil {
			return UndefinedValue(), false, false, err
		}
		var args []Value
		if p.hasGeneratorNextValue {
			args = []Value{p.generatorNextValue}
		}
		result, err := p.invoke(nextValue, args)
		if err != nil {
			return UndefinedValue(), false, false, err
		}
		if result.kind != jsValueScalar {
			return UndefinedValue(), false, false, NewError(ErrorKindUnsupported, "yield* iterator must return an object in this bounded classic-JS slice")
		}
		resultValue := unwrapPromiseValue(result.value)
		if resultValue.Kind != ValueKindObject {
			return UndefinedValue(), false, false, NewError(ErrorKindUnsupported, "yield* iterator must return an object in this bounded classic-JS slice")
		}
		doneValue, ok := lookupObjectProperty(resultValue.Object, "done")
		if !ok || doneValue.Kind != ValueKindBool {
			return UndefinedValue(), false, false, NewError(ErrorKindUnsupported, "yield* iterator result must include a boolean `done` property in this bounded classic-JS slice")
		}
		if doneValue.Bool {
			state.delegateIterator = nil
			return UndefinedValue(), false, true, nil
		}
		value, ok := lookupObjectProperty(resultValue.Object, "value")
		if !ok {
			value = UndefinedValue()
		}
		return value, true, false, nil
	}

	if state.delegateArray != nil {
		if state.delegateArrayIndex >= len(state.delegateArray) {
			state.delegateArray = nil
			state.delegateArrayIndex = 0
			return UndefinedValue(), false, true, nil
		}
		value := state.delegateArray[state.delegateArrayIndex]
		state.delegateArrayIndex++
		return value, true, false, nil
	}

	return UndefinedValue(), false, false, nil
}

func (p *classicJSStatementParser) startYieldDelegation(value Value) (Value, classicJSResumeState, bool, error) {
	switch value.Kind {
	case ValueKindArray, ValueKindString:
		values, err := p.collectClassicJSArrayLikeValues(value, "yield*")
		if err != nil {
			return UndefinedValue(), nil, false, err
		}
		if len(values) == 0 {
			return UndefinedValue(), nil, false, nil
		}
		return values[0], &classicJSYieldDelegationState{
			delegateArray:      values,
			delegateArrayIndex: 1,
		}, true, nil
	case ValueKindObject:
		copied := value
		return p.startYieldDelegationIterator(&copied)
	default:
		return UndefinedValue(), nil, false, NewError(ErrorKindRuntime, "yield* expects a string, array, or iterator-like object in this bounded classic-JS slice")
	}
}

func (p *classicJSStatementParser) startYieldDelegationIterator(delegate *Value) (Value, classicJSResumeState, bool, error) {
	if delegate == nil {
		return UndefinedValue(), nil, false, NewError(ErrorKindRuntime, "yield delegation value is unavailable")
	}
	nextValue, err := p.resolveMemberAccess(scalarJSValue(*delegate), "next")
	if err != nil {
		return UndefinedValue(), nil, false, err
	}
	result, err := p.invoke(nextValue, nil)
	if err != nil {
		return UndefinedValue(), nil, false, err
	}
	if result.kind != jsValueScalar {
		return UndefinedValue(), nil, false, NewError(ErrorKindUnsupported, "yield* iterator must return an object in this bounded classic-JS slice")
	}
	resultValue := unwrapPromiseValue(result.value)
	if resultValue.Kind != ValueKindObject {
		return UndefinedValue(), nil, false, NewError(ErrorKindUnsupported, "yield* iterator must return an object in this bounded classic-JS slice")
	}
	doneValue, ok := lookupObjectProperty(resultValue.Object, "done")
	if !ok || doneValue.Kind != ValueKindBool {
		return UndefinedValue(), nil, false, NewError(ErrorKindUnsupported, "yield* iterator result must include a boolean `done` property in this bounded classic-JS slice")
	}
	if doneValue.Bool {
		value, ok := lookupObjectProperty(resultValue.Object, "value")
		if !ok {
			value = UndefinedValue()
		}
		return value, nil, false, nil
	}
	value, ok := lookupObjectProperty(resultValue.Object, "value")
	if !ok {
		value = UndefinedValue()
	}
	return value, &classicJSYieldDelegationState{
		delegateIterator: delegate,
	}, true, nil
}

func (p *classicJSStatementParser) resumeYieldDelegationState(state *classicJSYieldDelegationState) (Value, classicJSResumeState, error) {
	if state == nil {
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "yield delegation state is unavailable")
	}

	if state.delegateIterator != nil {
		nextValue, err := p.resolveMemberAccess(scalarJSValue(*state.delegateIterator), "next")
		if err != nil {
			return UndefinedValue(), nil, err
		}
		var args []Value
		if p.hasGeneratorNextValue {
			args = []Value{p.generatorNextValue}
		}
		result, err := p.invoke(nextValue, args)
		if err != nil {
			return UndefinedValue(), nil, err
		}
		if result.kind != jsValueScalar {
			return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "yield* iterator must return an object in this bounded classic-JS slice")
		}
		resultValue := unwrapPromiseValue(result.value)
		if resultValue.Kind != ValueKindObject {
			return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "yield* iterator must return an object in this bounded classic-JS slice")
		}
		doneValue, ok := lookupObjectProperty(resultValue.Object, "done")
		if !ok || doneValue.Kind != ValueKindBool {
			return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "yield* iterator result must include a boolean `done` property in this bounded classic-JS slice")
		}
		if doneValue.Bool {
			state.delegateIterator = nil
			value, ok := lookupObjectProperty(resultValue.Object, "value")
			if !ok {
				value = UndefinedValue()
			}
			return value, nil, nil
		}
		value, ok := lookupObjectProperty(resultValue.Object, "value")
		if !ok {
			value = UndefinedValue()
		}
		return value, state, nil
	}

	if state.delegateArray != nil {
		if state.delegateArrayIndex >= len(state.delegateArray) {
			state.delegateArray = nil
			state.delegateArrayIndex = 0
			return UndefinedValue(), nil, nil
		}
		value := state.delegateArray[state.delegateArrayIndex]
		state.delegateArrayIndex++
		return value, state, nil
	}

	return UndefinedValue(), nil, nil
}

func generatorIteratorResultValue(value Value, done bool) jsValue {
	return scalarJSValue(ObjectValue([]ObjectEntry{
		{Key: "value", Value: value},
		{Key: "done", Value: BoolValue(done)},
	}))
}

func splitGeneratorYieldStatement(statement string) (string, bool, bool, error) {
	trimmed := strings.TrimSpace(statement)
	if trimmed == "" {
		return "", false, false, nil
	}
	if len(trimmed) < len("yield") || trimmed[:len("yield")] != "yield" {
		return "", false, false, nil
	}
	if len(trimmed) > len("yield") && isIdentPart(trimmed[len("yield")]) {
		return "", false, false, nil
	}

	rest := strings.TrimSpace(trimmed[len("yield"):])
	if rest == "" {
		return "", true, false, nil
	}
	if strings.HasPrefix(rest, "*") {
		delegated := strings.TrimSpace(rest[1:])
		if delegated == "" {
			return "", false, false, NewError(ErrorKindParse, "yield* requires an expression in this bounded classic-JS slice")
		}
		return delegated, true, true, nil
	}
	return rest, true, false, nil
}

func (p *classicJSStatementParser) parseArguments() ([]Value, error) {
	p.skipSpaceAndComments()
	if p.consumeByte(')') {
		return nil, nil
	}

	args := make([]Value, 0, 4)
	for {
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			value, err := p.parseScalarExpression()
			if err != nil {
				return nil, err
			}
			spreadArgs, err := p.expandClassicJSSpreadValues(value)
			if err != nil {
				return nil, err
			}
			args = append(args, spreadArgs...)
		} else {
			value, err := p.parseScalarExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}

		p.skipSpaceAndComments()
		if p.consumeByte(')') {
			return args, nil
		}
		if !p.consumeByte(',') {
			return nil, NewError(ErrorKindParse, "call arguments must be comma-separated")
		}
		p.skipSpaceAndComments()
		if p.consumeByte(')') {
			return args, nil
		}
	}
}

func (p *classicJSStatementParser) expandClassicJSSpreadValues(value Value) ([]Value, error) {
	return p.collectClassicJSArrayLikeValues(value, "call argument spread")
}

func (p *classicJSStatementParser) parseIdentifier() (string, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return "", NewError(ErrorKindParse, "unexpected end of script source")
	}

	start := p.pos
	if !isIdentStart(p.peekByte()) {
		return "", NewError(ErrorKindParse, fmt.Sprintf("expected identifier at %q", p.remainingPreview()))
	}
	p.pos++
	for !p.eof() && isIdentPart(p.peekByte()) {
		p.pos++
	}
	return p.source[start:p.pos], nil
}

func (p *classicJSStatementParser) parseStringLiteral() (Value, error) {
	if p.eof() {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
	}

	quote := p.peekByte()
	p.pos++
	var b strings.Builder
	for !p.eof() {
		ch := p.peekByte()
		p.pos++
		if ch == quote {
			return StringValue(b.String()), nil
		}
		if ch == '\\' {
			if p.eof() {
				return UndefinedValue(), NewError(ErrorKindParse, "unterminated escape sequence in string literal")
			}
			escaped := p.peekByte()
			p.pos++
			switch escaped {
			case '\\', '\'', '"':
				b.WriteByte(escaped)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'v':
				b.WriteByte('\v')
			case '0':
				b.WriteByte('\x00')
			case '\n':
				continue
			case 'x':
				runeValue, err := p.parseHexEscape(2)
				if err != nil {
					return UndefinedValue(), err
				}
				b.WriteRune(runeValue)
			case 'u':
				runeValue, err := p.parseHexEscape(4)
				if err != nil {
					return UndefinedValue(), err
				}
				b.WriteRune(runeValue)
			default:
				b.WriteByte(escaped)
			}
			continue
		}
		b.WriteByte(ch)
	}

	return UndefinedValue(), NewError(ErrorKindParse, "unterminated string literal")
}

func (p *classicJSStatementParser) parseTemplateLiteral() (Value, error) {
	segments, substitutions, err := p.consumeTemplateLiteralParts()
	if err != nil {
		return UndefinedValue(), err
	}

	var b strings.Builder
	for i, segment := range segments {
		b.WriteString(segment)
		if i < len(substitutions) {
			value, err := p.evalExpressionWithEnv(substitutions[i], p.env)
			if err != nil {
				return UndefinedValue(), err
			}
			b.WriteString(templateInterpolationString(value))
		}
	}
	return StringValue(b.String()), nil
}

func (p *classicJSStatementParser) parseRegularExpressionLiteral() (Value, error) {
	if p.eof() || p.peekByte() != '/' {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
	}

	start := p.pos
	p.pos++
	var escaped bool
	var inClass bool
	for !p.eof() {
		ch := p.peekByte()
		if ch == '\n' || ch == '\r' {
			return UndefinedValue(), NewError(ErrorKindParse, "unterminated regular expression literal")
		}
		if escaped {
			escaped = false
			p.pos++
			continue
		}
		switch ch {
		case '\\':
			escaped = true
			p.pos++
		case '[':
			inClass = true
			p.pos++
		case ']':
			if inClass {
				inClass = false
			}
			p.pos++
		case '/':
			if inClass {
				p.pos++
				continue
			}
			pattern := p.source[start+1 : p.pos]
			p.pos++
			flags, err := p.consumeRegularExpressionFlags()
			if err != nil {
				return UndefinedValue(), err
			}
			return classicJSRegExpLiteralValue(pattern, flags)
		default:
			p.pos++
		}
	}

	return UndefinedValue(), NewError(ErrorKindParse, "unterminated regular expression literal")
}

func (p *classicJSStatementParser) consumeRegularExpressionFlags() (string, error) {
	start := p.pos
	seen := make(map[byte]bool, 8)
	for !p.eof() {
		ch := p.peekByte()
		if ch < 'a' || ch > 'z' {
			if isIdentPart(ch) {
				return "", NewError(ErrorKindParse, fmt.Sprintf("unsupported regular expression flag %q in this bounded classic-JS slice", ch))
			}
			break
		}
		switch ch {
		case 'd', 'g', 'i', 'm', 's', 'u', 'v', 'y':
		default:
			return "", NewError(ErrorKindParse, fmt.Sprintf("unsupported regular expression flag %q in this bounded classic-JS slice", ch))
		}
		if seen[ch] {
			return "", NewError(ErrorKindParse, fmt.Sprintf("duplicate regular expression flag %q in this bounded classic-JS slice", ch))
		}
		seen[ch] = true
		p.pos++
	}
	return p.source[start:p.pos], nil
}

func classicJSRegExpLiteralValue(pattern, flags string) (Value, error) {
	compiled, err := classicJSCompileRegExpLiteral(pattern, flags)
	if err != nil {
		return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("invalid regular expression literal: %v", err))
	}

	literal := "/" + pattern + "/" + flags
	test := func(args []Value) (Value, error) {
		input := UndefinedValue()
		if len(args) > 0 {
			input = args[0]
		}
		return BoolValue(compiled.MatchString(ToJSString(input))), nil
	}
	exec := func(args []Value) (Value, error) {
		input := UndefinedValue()
		if len(args) > 0 {
			input = args[0]
		}
		text := ToJSString(input)
		matches := compiled.FindStringSubmatch(text)
		if matches == nil {
			return NullValue(), nil
		}
		loc := compiled.FindStringSubmatchIndex(text)
		entries := make([]ObjectEntry, 0, len(matches)+3)
		for i, match := range matches {
			entries = append(entries, ObjectEntry{Key: strconv.Itoa(i), Value: StringValue(match)})
		}
		entries = append(entries, ObjectEntry{Key: "length", Value: NumberValue(float64(len(matches)))})
		if len(loc) >= 2 {
			entries = append(entries, ObjectEntry{Key: "index", Value: NumberValue(float64(loc[0]))})
		}
		entries = append(entries, ObjectEntry{Key: "input", Value: StringValue(text)})
		return ObjectValue(entries), nil
	}

	return ObjectValue([]ObjectEntry{
		{Key: classicJSRegExpPatternKey, Value: StringValue(pattern)},
		{Key: classicJSRegExpFlagsKey, Value: StringValue(flags)},
		{Key: "source", Value: StringValue(pattern)},
		{Key: "flags", Value: StringValue(flags)},
		{Key: "test", Value: NativeFunctionValue(test)},
		{Key: "exec", Value: NativeFunctionValue(exec)},
		{Key: "toString", Value: NativeFunctionValue(func(args []Value) (Value, error) {
			return StringValue(literal), nil
		})},
	}), nil
}

func classicJSCompileRegExpLiteral(pattern, flags string) (*regexp.Regexp, error) {
	var prefix strings.Builder
	seen := make(map[byte]bool, len(flags))
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		if seen[flag] {
			return nil, fmt.Errorf("duplicate regular expression flag %q", flag)
		}
		seen[flag] = true
		switch flag {
		case 'i':
			prefix.WriteString("(?i)")
		case 'm':
			prefix.WriteString("(?m)")
		case 's':
			prefix.WriteString("(?s)")
		case 'd', 'g', 'u', 'v', 'y':
		default:
			return nil, fmt.Errorf("unsupported regular expression flag %q", flag)
		}
	}

	return regexp.Compile(prefix.String() + pattern)
}

func (p *classicJSStatementParser) parseTaggedTemplateLiteral(tag jsValue) (jsValue, error) {
	segments, substitutions, err := p.consumeTemplateLiteralParts()
	if err != nil {
		return jsValue{}, err
	}
	if tag.kind == jsValueScalar {
		switch tag.value.Kind {
		case ValueKindFunction, ValueKindHostReference:
		default:
			return jsValue{}, NewError(ErrorKindRuntime, "cannot call non-callable tagged template tag in this bounded classic-JS slice")
		}
	}

	cooked := make([]Value, len(segments))
	for i, segment := range segments {
		cooked[i] = StringValue(segment)
	}
	args := make([]Value, 0, len(substitutions)+1)
	args = append(args, ArrayValue(cooked))
	for _, source := range substitutions {
		value, err := p.evalExpressionWithEnv(source, p.env)
		if err != nil {
			return jsValue{}, err
		}
		args = append(args, value)
	}

	value, err := p.invoke(tag, args)
	if err != nil {
		return jsValue{}, err
	}
	return value, nil
}

func (p *classicJSStatementParser) consumeTemplateLiteralParts() ([]string, []string, error) {
	if p.eof() {
		return nil, nil, NewError(ErrorKindParse, "unexpected end of script source")
	}

	p.pos++
	segments := make([]string, 0, 2)
	substitutions := make([]string, 0, 1)
	var b strings.Builder
	for !p.eof() {
		ch := p.peekByte()
		switch {
		case ch == '`':
			p.pos++
			segments = append(segments, b.String())
			return segments, substitutions, nil
		case ch == '$' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '{':
			p.pos += 2
			source, err := p.consumeTemplateInterpolationSource()
			if err != nil {
				return nil, nil, err
			}
			segments = append(segments, b.String())
			b.Reset()
			substitutions = append(substitutions, source)
		case ch == '\\':
			p.pos++
			if p.eof() {
				return nil, nil, NewError(ErrorKindParse, "unterminated escape sequence in template literal")
			}
			escaped := p.peekByte()
			p.pos++
			switch escaped {
			case '\\', '`', '\'', '"', '$':
				b.WriteByte(escaped)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'v':
				b.WriteByte('\v')
			case '0':
				b.WriteByte('\x00')
			case '\n':
				continue
			case 'x':
				runeValue, err := p.parseHexEscape(2)
				if err != nil {
					return nil, nil, err
				}
				b.WriteRune(runeValue)
			case 'u':
				runeValue, err := p.parseHexEscape(4)
				if err != nil {
					return nil, nil, err
				}
				b.WriteRune(runeValue)
			default:
				b.WriteByte(escaped)
			}
		default:
			b.WriteByte(ch)
			p.pos++
		}
	}

	return nil, nil, NewError(ErrorKindParse, "unterminated template literal")
}

func (p *classicJSStatementParser) parseHexEscape(width int) (rune, error) {
	if p.pos+width > len(p.source) {
		return 0, NewError(ErrorKindParse, "unterminated hex escape in string literal")
	}
	value, err := strconv.ParseUint(p.source[p.pos:p.pos+width], 16, 32)
	if err != nil {
		return 0, NewError(ErrorKindParse, "invalid hex escape in string literal")
	}
	p.pos += width
	return rune(value), nil
}

func (p *classicJSStatementParser) consumeDigitsWithSeparators() (bool, error) {
	return p.consumeDigitsWithSeparatorsWhile(isDigit)
}

func (p *classicJSStatementParser) consumeDigitsWithSeparatorsWhile(validDigit func(byte) bool) (bool, error) {
	sawDigit := false
	lastWasSeparator := false
	for !p.eof() {
		ch := p.peekByte()
		switch {
		case validDigit(ch):
			sawDigit = true
			lastWasSeparator = false
			p.pos++
		case ch == '_':
			if !sawDigit || lastWasSeparator {
				return false, NewError(ErrorKindParse, "invalid numeric literal")
			}
			lastWasSeparator = true
			p.pos++
		default:
			if lastWasSeparator {
				return false, NewError(ErrorKindParse, "invalid numeric literal")
			}
			return sawDigit, nil
		}
	}
	if lastWasSeparator {
		return false, NewError(ErrorKindParse, "invalid numeric literal")
	}
	return sawDigit, nil
}

func (p *classicJSStatementParser) parseNumberLiteral() (Value, error) {
	start := p.pos
	hasFraction := false
	hasExponent := false
	if p.peekByte() == '0' && p.pos+1 < len(p.source) {
		switch p.source[p.pos+1] {
		case 'x', 'X', 'b', 'B', 'o', 'O':
			basePrefix := p.source[p.pos+1]
			p.pos += 2
			var (
				allowDigit func(byte) bool
				base       int
			)
			switch basePrefix {
			case 'x', 'X':
				allowDigit = isHexDigit
				base = 16
			case 'b', 'B':
				allowDigit = isBinaryDigit
				base = 2
			case 'o', 'O':
				allowDigit = isOctalDigit
				base = 8
			}
			if _, err := p.consumeDigitsWithSeparatorsWhile(allowDigit); err != nil {
				return UndefinedValue(), err
			}
			if p.peekByte() == 'n' {
				raw := p.source[start:p.pos]
				normalized := strings.ReplaceAll(raw[2:], "_", "")
				bigInt := new(big.Int)
				if _, ok := bigInt.SetString(normalized, base); !ok {
					return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("invalid numeric literal %q", raw))
				}
				p.pos++
				return BigIntValue(bigInt.String()), nil
			}
			if p.peekByte() == '.' || p.peekByte() == 'e' || p.peekByte() == 'E' {
				return UndefinedValue(), NewError(ErrorKindParse, "invalid numeric literal")
			}
			raw := p.source[start:p.pos]
			normalized := strings.ReplaceAll(raw[2:], "_", "")
			bigInt := new(big.Int)
			if _, ok := bigInt.SetString(normalized, base); !ok {
				return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("invalid numeric literal %q", raw))
			}
			number, _ := new(big.Float).SetInt(bigInt).Float64()
			return NumberValue(number), nil
		}
	}
	if p.consumeByte('.') {
		if p.eof() || !isDigit(p.peekByte()) {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid numeric literal")
		}
		if _, err := p.consumeDigitsWithSeparators(); err != nil {
			return UndefinedValue(), err
		}
		hasFraction = true
	} else {
		if _, err := p.consumeDigitsWithSeparators(); err != nil {
			return UndefinedValue(), err
		}
		if p.consumeByte('.') {
			hasFraction = true
			if !p.eof() && (isDigit(p.peekByte()) || p.peekByte() == '_') {
				if _, err := p.consumeDigitsWithSeparators(); err != nil {
					return UndefinedValue(), err
				}
			}
		}
	}

	if !p.eof() && (p.peekByte() == 'e' || p.peekByte() == 'E') {
		p.pos++
		if !p.eof() && (p.peekByte() == '+' || p.peekByte() == '-') {
			p.pos++
		}
		if p.eof() || !isDigit(p.peekByte()) {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid numeric literal")
		}
		if _, err := p.consumeDigitsWithSeparators(); err != nil {
			return UndefinedValue(), err
		}
		hasExponent = true
	}

	if !p.eof() && p.peekByte() == 'n' {
		if hasFraction || hasExponent {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid numeric literal")
		}
		raw := p.source[start:p.pos]
		normalized := strings.ReplaceAll(raw, "_", "")
		bigInt := new(big.Int)
		if _, ok := bigInt.SetString(normalized, 10); !ok {
			return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("invalid numeric literal %q", raw))
		}
		p.pos++
		return BigIntValue(bigInt.String()), nil
	}

	raw := p.source[start:p.pos]
	normalized := strings.ReplaceAll(raw, "_", "")
	number, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("invalid numeric literal %q", raw))
	}
	return NumberValue(number), nil
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isBinaryDigit(ch byte) bool {
	return ch == '0' || ch == '1'
}

func isOctalDigit(ch byte) bool {
	return ch >= '0' && ch <= '7'
}

func (p *classicJSStatementParser) peekKeyword(keyword string) (string, bool) {
	if p == nil {
		return "", false
	}
	if p.pos+len(keyword) > len(p.source) {
		return "", false
	}
	if p.source[p.pos:p.pos+len(keyword)] != keyword {
		return "", false
	}
	if p.pos+len(keyword) < len(p.source) && isIdentPart(p.source[p.pos+len(keyword)]) {
		return "", false
	}
	if p.pos > 0 && isIdentPart(p.source[p.pos-1]) {
		return "", false
	}
	return keyword, true
}

func (p *classicJSStatementParser) remainingPreview() string {
	if p == nil || p.eof() {
		return ""
	}
	end := p.pos + 24
	if end > len(p.source) {
		end = len(p.source)
	}
	return p.source[p.pos:end]
}

func jsTruthy(value Value) bool {
	switch value.Kind {
	case ValueKindUndefined:
		return false
	case ValueKindNull:
		return false
	case ValueKindBool:
		return value.Bool
	case ValueKindNumber:
		return value.Number != 0 && !math.IsNaN(value.Number)
	case ValueKindBigInt:
		return value.BigInt != "0"
	case ValueKindString:
		return value.String != ""
	default:
		return true
	}
}

func jsTruthyJSValue(value jsValue) bool {
	switch value.kind {
	case jsValueScalar:
		return jsTruthy(value.value)
	case jsValueHostObject, jsValueHostMethod, jsValueBuiltinExpr:
		return true
	case jsValueSuper:
		return true
	default:
		return true
	}
}

func classicJSSameValue(left Value, right Value) bool {
	if left.Kind != right.Kind {
		if left.Kind == ValueKindHostReference && right.Kind == ValueKindHostReference {
			return left.HostReferenceKind == right.HostReferenceKind && left.HostReferencePath == right.HostReferencePath
		}
		return false
	}

	switch left.Kind {
	case ValueKindUndefined, ValueKindNull:
		return true
	case ValueKindString:
		return left.String == right.String
	case ValueKindBool:
		return left.Bool == right.Bool
	case ValueKindNumber:
		if math.IsNaN(left.Number) || math.IsNaN(right.Number) {
			return false
		}
		return left.Number == right.Number
	case ValueKindBigInt:
		return left.BigInt == right.BigInt
	case ValueKindArray:
		return reflect.ValueOf(left.Array).Pointer() == reflect.ValueOf(right.Array).Pointer()
	case ValueKindObject:
		return reflect.ValueOf(left.Object).Pointer() == reflect.ValueOf(right.Object).Pointer()
	case ValueKindHostReference:
		return left.HostReferenceKind == right.HostReferenceKind && left.HostReferencePath == right.HostReferencePath
	case ValueKindFunction:
		if left.Function != nil && right.Function != nil {
			return left.Function == right.Function
		}
		if left.NativeFunction == nil && right.NativeFunction == nil {
			return true
		}
		return false
	case ValueKindPromise:
		return left.Promise == right.Promise
	case ValueKindInvocation:
		return left.Invocation == right.Invocation
	case ValueKindPrivateName:
		return left.PrivateName == right.PrivateName
	default:
		return false
	}
}

func classicJSEqualValues(left Value, right Value, op string) bool {
	equal := classicJSSameValue(left, right)
	if op == "===" || op == "!==" {
		if op == "===" {
			return equal
		}
		return !equal
	}

	if equal {
		return op == "=="
	}

	if (left.Kind == ValueKindNull && right.Kind == ValueKindUndefined) || (left.Kind == ValueKindUndefined && right.Kind == ValueKindNull) {
		return op == "=="
	}

	if left.Kind == ValueKindBool {
		return classicJSEqualValues(NumberValue(boolToNumber(left.Bool)), right, op)
	}
	if right.Kind == ValueKindBool {
		return classicJSEqualValues(left, NumberValue(boolToNumber(right.Bool)), op)
	}

	leftNum, leftOK := classicJSNumberValue(left)
	rightNum, rightOK := classicJSNumberValue(right)
	if leftOK && rightOK {
		if math.IsNaN(leftNum) || math.IsNaN(rightNum) {
			return op == "!="
		}
		if op == "==" {
			return leftNum == rightNum
		}
		return leftNum != rightNum
	}

	if left.Kind == ValueKindString || right.Kind == ValueKindString {
		if op == "==" {
			return ToJSString(left) == ToJSString(right)
		}
		return ToJSString(left) != ToJSString(right)
	}

	if op == "==" {
		return false
	}
	return true
}

func classicJSNumberValue(value Value) (float64, bool) {
	switch value.Kind {
	case ValueKindUndefined:
		return math.NaN(), true
	case ValueKindNull:
		return 0, true
	case ValueKindBool:
		if value.Bool {
			return 1, true
		}
		return 0, true
	case ValueKindNumber:
		return value.Number, true
	case ValueKindBigInt:
		bigInt := new(big.Int)
		if _, ok := bigInt.SetString(value.BigInt, 10); !ok {
			return math.NaN(), false
		}
		result, _ := bigInt.Float64()
		return result, true
	case ValueKindString:
		trimmed := strings.TrimSpace(value.String)
		if trimmed == "" {
			return 0, true
		}
		number, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return math.NaN(), false
		}
		return number, true
	case ValueKindObject:
		if ms, ok := BrowserDateTimestamp(value); ok {
			return float64(ms), true
		}
		return math.NaN(), false
	default:
		return math.NaN(), false
	}
}

func classicJSUnaryNumberValue(value Value) (float64, bool) {
	switch value.Kind {
	case ValueKindUndefined:
		return math.NaN(), true
	case ValueKindNull:
		return 0, true
	case ValueKindBool:
		return boolToNumber(value.Bool), true
	case ValueKindNumber:
		return value.Number, true
	case ValueKindBigInt:
		return 0, false
	case ValueKindString:
		trimmed := strings.TrimSpace(value.String)
		if trimmed == "" {
			return 0, true
		}
		if number, ok := classicJSUnaryNumberFromPrefixedString(trimmed); ok {
			return number, true
		}
		number, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return math.NaN(), true
		}
		return number, true
	case ValueKindObject:
		if ms, ok := BrowserDateTimestamp(value); ok {
			return float64(ms), true
		}
		return math.NaN(), false
	default:
		return math.NaN(), false
	}
}

func classicJSUnaryNumberFromPrefixedString(value string) (float64, bool) {
	if len(value) <= 2 || value[0] != '0' {
		return 0, false
	}

	base := 0
	digits := ""
	switch value[1] {
	case 'x', 'X':
		base = 16
		digits = value[2:]
	case 'b', 'B':
		base = 2
		digits = value[2:]
	case 'o', 'O':
		base = 8
		digits = value[2:]
	default:
		return 0, false
	}

	if digits == "" {
		return math.NaN(), true
	}
	bigInt := new(big.Int)
	if _, ok := bigInt.SetString(digits, base); !ok {
		return math.NaN(), true
	}
	number, _ := new(big.Float).SetInt(bigInt).Float64()
	return number, true
}

func boolToNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func classicJSRelationalCompare(left Value, right Value, op string) (bool, error) {
	if left.Kind == ValueKindString && right.Kind == ValueKindString {
		switch op {
		case "<":
			return left.String < right.String, nil
		case "<=":
			return left.String <= right.String, nil
		case ">":
			return left.String > right.String, nil
		case ">=":
			return left.String >= right.String, nil
		default:
			return false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported relational operator %q in this bounded classic-JS slice", op))
		}
	}

	leftNum, leftOK := classicJSNumberValue(left)
	rightNum, rightOK := classicJSNumberValue(right)
	if !leftOK || !rightOK || math.IsNaN(leftNum) || math.IsNaN(rightNum) {
		return false, nil
	}

	switch op {
	case "<":
		return leftNum < rightNum, nil
	case "<=":
		return leftNum <= rightNum, nil
	case ">":
		return leftNum > rightNum, nil
	case ">=":
		return leftNum >= rightNum, nil
	default:
		return false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported relational operator %q in this bounded classic-JS slice", op))
	}
}

func classicJSToUint32(value float64) uint32 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return 0
	}
	truncated := math.Trunc(value)
	truncated = math.Mod(truncated, 4294967296)
	if truncated < 0 {
		truncated += 4294967296
	}
	return uint32(truncated)
}

func classicJSToInt32(value float64) int32 {
	return int32(classicJSToUint32(value))
}

func classicJSBigIntValue(value Value) (*big.Int, bool) {
	if value.Kind != ValueKindBigInt {
		return nil, false
	}
	bigInt := new(big.Int)
	if _, ok := bigInt.SetString(value.BigInt, 10); !ok {
		return nil, false
	}
	return bigInt, true
}

func classicJSBitwiseNotValue(value Value) (Value, error) {
	switch value.Kind {
	case ValueKindBigInt:
		bigInt, ok := classicJSBigIntValue(value)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
		}
		bigInt.Not(bigInt)
		return BigIntValue(bigInt.String()), nil
	default:
		number, ok := classicJSNumberValue(value)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "bitwise operators only work on scalar values in this bounded classic-JS slice")
		}
		return NumberValue(float64(^classicJSToInt32(number))), nil
	}
}

func classicJSBitwiseBinaryValues(left Value, right Value, op string) (Value, error) {
	if left.Kind == ValueKindBigInt || right.Kind == ValueKindBigInt {
		if left.Kind != ValueKindBigInt || right.Kind != ValueKindBigInt {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "bitwise operators only work on values of the same numeric kind in this bounded classic-JS slice")
		}
		leftInt, ok := classicJSBigIntValue(left)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
		}
		rightInt, ok := classicJSBigIntValue(right)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
		}
		switch op {
		case "&":
			return BigIntValue(new(big.Int).And(leftInt, rightInt).String()), nil
		case "|":
			return BigIntValue(new(big.Int).Or(leftInt, rightInt).String()), nil
		case "^":
			return BigIntValue(new(big.Int).Xor(leftInt, rightInt).String()), nil
		case "<<":
			if rightInt.Sign() < 0 || !rightInt.IsUint64() {
				return UndefinedValue(), NewError(ErrorKindUnsupported, "BigInt shifts only support non-negative shift counts in this bounded classic-JS slice")
			}
			return BigIntValue(new(big.Int).Lsh(leftInt, uint(rightInt.Uint64())).String()), nil
		case ">>":
			if rightInt.Sign() < 0 || !rightInt.IsUint64() {
				return UndefinedValue(), NewError(ErrorKindUnsupported, "BigInt shifts only support non-negative shift counts in this bounded classic-JS slice")
			}
			return BigIntValue(new(big.Int).Rsh(leftInt, uint(rightInt.Uint64())).String()), nil
		case ">>>":
			return UndefinedValue(), NewError(ErrorKindUnsupported, "unsigned right shift is not supported for BigInt values in this bounded classic-JS slice")
		default:
			return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported bitwise operator %q in this bounded classic-JS slice", op))
		}
	}

	leftNum, leftOK := classicJSNumberValue(left)
	rightNum, rightOK := classicJSNumberValue(right)
	if !leftOK || !rightOK {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "bitwise operators only work on scalar values in this bounded classic-JS slice")
	}
	switch op {
	case "&":
		return NumberValue(float64(classicJSToInt32(leftNum) & classicJSToInt32(rightNum))), nil
	case "|":
		return NumberValue(float64(classicJSToInt32(leftNum) | classicJSToInt32(rightNum))), nil
	case "^":
		return NumberValue(float64(classicJSToInt32(leftNum) ^ classicJSToInt32(rightNum))), nil
	case "<<":
		return NumberValue(float64(classicJSToInt32(leftNum) << (classicJSToUint32(rightNum) & 31))), nil
	case ">>":
		return NumberValue(float64(classicJSToInt32(leftNum) >> (classicJSToUint32(rightNum) & 31))), nil
	case ">>>":
		return NumberValue(float64(classicJSToUint32(leftNum) >> (classicJSToUint32(rightNum) & 31))), nil
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported bitwise operator %q in this bounded classic-JS slice", op))
	}
}

func classicJSAddValues(left Value, right Value, op byte) (Value, error) {
	switch op {
	case '+':
		if left.Kind == ValueKindString || right.Kind == ValueKindString {
			return StringValue(ToJSString(left) + ToJSString(right)), nil
		}
		if left.Kind == ValueKindBigInt && right.Kind == ValueKindBigInt {
			leftInt := new(big.Int)
			rightInt := new(big.Int)
			if _, ok := leftInt.SetString(left.BigInt, 10); !ok {
				return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
			}
			if _, ok := rightInt.SetString(right.BigInt, 10); !ok {
				return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
			}
			leftInt.Add(leftInt, rightInt)
			return BigIntValue(leftInt.String()), nil
		}
		leftNum, leftOK := classicJSNumberValue(left)
		rightNum, rightOK := classicJSNumberValue(right)
		if !leftOK || !rightOK {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "addition only works on scalar values in this bounded classic-JS slice")
		}
		return NumberValue(leftNum + rightNum), nil
	case '-':
		leftNum, leftOK := classicJSNumberValue(left)
		rightNum, rightOK := classicJSNumberValue(right)
		if !leftOK || !rightOK {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "subtraction only works on scalar values in this bounded classic-JS slice")
		}
		return NumberValue(leftNum - rightNum), nil
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported additive operator %q in this bounded classic-JS slice", string(op)))
	}
}

func classicJSMultiplyValues(left Value, right Value, op byte) (Value, error) {
	leftNum, leftOK := classicJSNumberValue(left)
	rightNum, rightOK := classicJSNumberValue(right)
	if !leftOK || !rightOK {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "multiplicative operators only work on scalar values in this bounded classic-JS slice")
	}

	switch op {
	case '*':
		return NumberValue(leftNum * rightNum), nil
	case '/':
		return NumberValue(leftNum / rightNum), nil
	case '%':
		return NumberValue(math.Mod(leftNum, rightNum)), nil
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported multiplicative operator %q in this bounded classic-JS slice", string(op)))
	}
}

func classicJSPowerValues(left Value, right Value) (Value, error) {
	if left.Kind == ValueKindBigInt || right.Kind == ValueKindBigInt {
		if left.Kind != ValueKindBigInt || right.Kind != ValueKindBigInt {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "exponentiation only works on values of the same numeric kind in this bounded classic-JS slice")
		}
		leftInt, ok := classicJSBigIntValue(left)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
		}
		rightInt, ok := classicJSBigIntValue(right)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
		}
		if rightInt.Sign() < 0 || !rightInt.IsUint64() {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "BigInt exponentiation only supports non-negative Uint64 exponents in this bounded classic-JS slice")
		}
		return BigIntValue(new(big.Int).Exp(leftInt, new(big.Int).SetUint64(rightInt.Uint64()), nil).String()), nil
	}

	leftNum, leftOK := classicJSNumberValue(left)
	rightNum, rightOK := classicJSNumberValue(right)
	if !leftOK || !rightOK {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "exponentiation only works on scalar values in this bounded classic-JS slice")
	}
	return NumberValue(math.Pow(leftNum, rightNum)), nil
}

func classicJSIncrementDecrementValue(value Value, delta int) (Value, error) {
	if value.Kind == ValueKindBigInt {
		bigInt, ok := classicJSBigIntValue(value)
		if !ok {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid BigInt literal")
		}
		bigInt.Add(bigInt, big.NewInt(int64(delta)))
		return BigIntValue(bigInt.String()), nil
	}

	number, ok := classicJSNumberValue(value)
	if !ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "increment and decrement only work on scalar values in this bounded classic-JS slice")
	}
	return NumberValue(number + float64(delta)), nil
}

func classicJSArrayLengthFromValue(value Value) (int, error) {
	if value.Kind == ValueKindBigInt {
		return 0, NewError(ErrorKindUnsupported, "array length assignment only works on scalar non-BigInt values in this bounded classic-JS slice")
	}
	number, ok := classicJSNumberValue(value)
	if !ok {
		return 0, NewError(ErrorKindUnsupported, "array length assignment only works on scalar values in this bounded classic-JS slice")
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || math.Trunc(number) != number {
		return 0, NewError(ErrorKindUnsupported, "array length assignment only works on non-negative integer values in this bounded classic-JS slice")
	}
	length := int(number)
	if float64(length) != number {
		return 0, NewError(ErrorKindUnsupported, "array length assignment only works on non-negative integer values in this bounded classic-JS slice")
	}
	return length, nil
}

func templateInterpolationString(value Value) string {
	return ToJSString(value)
}

func isNullishJSValue(value Value) bool {
	return value.Kind == ValueKindUndefined || value.Kind == ValueKindNull
}

func (p *classicJSStatementParser) peekLogicalAssignmentOperator() string {
	if p == nil || p.pos+2 > len(p.source) {
		return ""
	}
	switch {
	case strings.HasPrefix(p.source[p.pos:], ">>>="):
		return ">>>="
	case strings.HasPrefix(p.source[p.pos:], "**="):
		return "**="
	case strings.HasPrefix(p.source[p.pos:], "||="):
		return "||="
	case strings.HasPrefix(p.source[p.pos:], "&&="):
		return "&&="
	case strings.HasPrefix(p.source[p.pos:], "??="):
		return "??="
	case strings.HasPrefix(p.source[p.pos:], "<<="):
		return "<<="
	case strings.HasPrefix(p.source[p.pos:], ">>="):
		return ">>="
	case strings.HasPrefix(p.source[p.pos:], "+="):
		return "+="
	case strings.HasPrefix(p.source[p.pos:], "-="):
		return "-="
	case strings.HasPrefix(p.source[p.pos:], "*="):
		return "*="
	case strings.HasPrefix(p.source[p.pos:], "/="):
		return "/="
	case strings.HasPrefix(p.source[p.pos:], "%="):
		return "%="
	case strings.HasPrefix(p.source[p.pos:], "&="):
		return "&="
	case strings.HasPrefix(p.source[p.pos:], "|="):
		return "|="
	case strings.HasPrefix(p.source[p.pos:], "^="):
		return "^="
	default:
		return ""
	}
}

func classicJSApplyCompoundAssignment(current Value, rhs Value, op string) (Value, error) {
	switch op {
	case "+=":
		return classicJSAddValues(current, rhs, '+')
	case "-=":
		return classicJSAddValues(current, rhs, '-')
	case "*=", "/=", "%=":
		return classicJSMultiplyValues(current, rhs, op[0])
	case "&=", "|=", "^=", "<<=", ">>=", ">>>=":
		bitwiseOp := strings.TrimSuffix(op, "=")
		return classicJSBitwiseBinaryValues(current, rhs, bitwiseOp)
	case "**=":
		return classicJSPowerValues(current, rhs)
	default:
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported compound assignment operator %q in this bounded classic-JS slice", op))
	}
}

func hasClassicJSDeclarationKeyword(source string) bool {
	parser := &classicJSStatementParser{source: strings.TrimSpace(source)}
	if parser.source == "" {
		return false
	}
	parser.skipSpaceAndComments()
	for _, keyword := range []string{"let", "const", "using"} {
		if _, ok := parser.peekKeyword(keyword); ok {
			return true
		}
	}
	if keyword, ok := parser.peekKeyword("await"); ok {
		parser.pos += len(keyword)
		parser.skipSpaceAndComments()
		if _, ok := parser.peekKeyword("using"); ok {
			return true
		}
	}
	return false
}

func isClassicJSReservedDeclarationName(name string) bool {
	switch name {
	case "host", "expr", "this", "true", "false", "undefined", "null", "let", "const", "var", "using", "function", "class", "if", "else", "for", "while", "do", "switch", "case", "default", "try", "catch", "finally", "return", "break", "continue", "throw", "debugger", "async", "await", "import", "export", "new", "delete", "yield", "super", "typeof", "void", "in", "instanceof":
		return true
	default:
		return false
	}
}

func negateBigIntLiteral(value string) (string, error) {
	bigInt := new(big.Int)
	if _, ok := bigInt.SetString(value, 10); !ok {
		return "", NewError(ErrorKindParse, "invalid BigInt literal")
	}
	bigInt.Neg(bigInt)
	return bigInt.String(), nil
}

func isIdentStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
