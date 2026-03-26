package script

import (
	"fmt"
	"math"
	"math/big"
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

type jsValue struct {
	kind        jsValueKind
	value       Value
	method      string
	receiver    Value
	hasReceiver bool
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
	return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, false, false, nil, classDef, nil)
}

func evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, privateClass *classicJSClassDefinition, moduleExports map[string]Value) (Value, error) {
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
			"unsupported script source; this bounded classic-JS slice only supports expression statements, `let`/`const` declarations, block-bodied `if` / `while` / `do...while` / `for` / `switch` / `try` statements, class declarations with static blocks, public `static` fields, getter/setter accessors, computed fields and methods, instance fields, bounded `extends` inheritance, and bounded `new Class()` instantiation, member calls on `host`, and the `expr(...)` compatibility helper",
		)
	}

	return value, nil
}

func evalClassicJSStatementWithEnvAndAllowAwaitAndYield(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, allowReturn bool, resumeState classicJSResumeState, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, allowYield, allowReturn, resumeState, privateClass, nil)
}

func evalClassicJSExpressionWithEnv(source string, host HostBindings, env *classicJSEnvironment, stepLimit int) (Value, error) {
	return evalClassicJSExpressionWithEnvAndAllowAwait(source, host, env, stepLimit, false, nil)
}

func evalClassicJSExpressionWithEnvAndAllowAwait(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, privateClass ...*classicJSClassDefinition) (Value, error) {
	var classDef *classicJSClassDefinition
	if len(privateClass) > 0 {
		classDef = privateClass[0]
	}
	return evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, false, classDef, nil)
}

func evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, privateClass *classicJSClassDefinition) (Value, error) {
	return evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source, host, env, stepLimit, allowAwait, allowYield, privateClass, nil)
}

func evalClassicJSExpressionWithEnvAndAllowAwaitAndYieldAndExports(source string, host HostBindings, env *classicJSEnvironment, stepLimit int, allowAwait bool, allowYield bool, privateClass *classicJSClassDefinition, moduleExports map[string]Value) (Value, error) {
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
	statementLabel          string
	allowUnknownIdentifiers bool
	allowAwait              bool
	allowYield              bool
	allowReturn             bool
	resumeState             classicJSResumeState
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
	env                *classicJSEnvironment
	privateClass       *classicJSClassDefinition
	privateFieldPrefix string
	superTarget        Value
	hasSuperTarget     bool
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
	index              int
	done               bool
	activeState        classicJSResumeState
	delegateArray      []Value
	delegateArrayIndex int
	delegateIterator   *Value
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
	catchName        string
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
}

type classicJSDeleteStep struct {
	key     string
	private bool
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
		async:              s.async,
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
		catchName:       s.catchName,
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
	return evalClassicJSStatementWithEnvAndAllowAwaitAndYieldAndExports(source, p.host, env, p.stepLimit, p.allowAwait, p.allowYield, p.allowReturn, p.resumeState, p.privateClass, nil)
}

func (p *classicJSStatementParser) evalProgramWithEnv(source string, env *classicJSEnvironment) (Value, error) {
	return evalClassicJSProgramWithAllowAwaitAndYieldAndExports(source, p.host, env, p.stepLimit, p.allowAwait, p.allowYield, p.allowReturn, p.resumeState, p.privateClass, nil)
}

func (p *classicJSStatementParser) evalExpressionWithEnv(source string, env *classicJSEnvironment) (Value, error) {
	return evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(source, p.host, env, p.stepLimit, p.allowAwait, p.allowYield, p.privateClass)
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

func (p *classicJSStatementParser) parseLogicalAssignment() (jsValue, error) {
	p.skipSpaceAndComments()
	start := p.pos
	name, steps, op, rhsPos, ok, err := p.tryParseAssignmentTarget()
	if err != nil {
		return jsValue{}, err
	}
	if !ok {
		p.pos = start
		return p.parseNullishCoalescing()
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
	if len(steps) > 0 {
		if op != "=" {
			return jsValue{}, NewError(ErrorKindUnsupported, "logical assignment only works on declared local bindings in this bounded classic-JS slice")
		}
		if current.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar object bindings in this bounded classic-JS slice")
		}
		if current.value.Kind != ValueKindObject {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on object values in this bounded classic-JS slice")
		}

		value, err := p.parseLogicalAssignment()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "assignment only works on scalar values in this bounded classic-JS slice")
		}
		if _, err := assignJSValuePropertyChain(p, current.value, steps, value.value, p.privateFieldPrefix); err != nil {
			return jsValue{}, err
		}
		return value, nil
	}

	if op == "=" {
		value, err := p.parseLogicalAssignment()
		if err != nil {
			return jsValue{}, err
		}
		if err := p.env.assign(name, value); err != nil {
			return jsValue{}, err
		}
		return value, nil
	}

	if op == "" {
		p.pos = start
		return p.parseNullishCoalescing()
	}

	shouldAssign := false
	switch op {
	case "||=":
		shouldAssign = !jsTruthy(current.value)
	case "&&=":
		shouldAssign = jsTruthy(current.value)
	case "??=":
		shouldAssign = isNullishJSValue(current.value)
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported logical assignment operator %q in this bounded classic-JS slice", op))
	}

	if !shouldAssign {
		skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
		skip.pos = rhsPos
		if _, err := skip.parseLogicalAssignment(); err != nil {
			return jsValue{}, err
		}
		p.pos = skip.pos
		return current, nil
	}

	value, err := p.parseLogicalAssignment()
	if err != nil {
		return jsValue{}, err
	}
	if err := p.env.assign(name, value); err != nil {
		return jsValue{}, err
	}
	return value, nil
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
	if keyword, ok := p.peekKeyword("const"); ok {
		p.pos += len(keyword)
		return p.parseVariableDeclaration("const")
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
	if keyword, ok := p.peekKeyword("yield"); ok && p.allowYield {
		p.pos += len(keyword)
		return p.parseYieldStatement()
	}
	if keyword, ok := p.peekKeyword("import"); ok {
		p.pos += len(keyword)
		return p.parseImportStatement()
	}
	if keyword, ok := p.peekKeyword("export"); ok {
		p.pos += len(keyword)
		return p.parseExportStatement()
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
	if !p.allowYield {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "yield statements are only supported inside bounded generator bodies")
	}

	p.skipSpaceAndComments()
	if p.consumeByte('*') {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "yield* delegation is only supported at the top level of a generator statement in this bounded classic-JS slice")
	}

	source := strings.TrimSpace(p.source[p.pos:])
	if source == "" {
		return UndefinedValue(), classicJSYieldSignal{value: UndefinedValue()}
	}

	value, err := p.evalExpressionWithEnv(source, p.env.clone())
	if err != nil {
		return UndefinedValue(), err
	}
	return UndefinedValue(), classicJSYieldSignal{value: value, resumeState: p.resumeState}
}

func (p *classicJSStatementParser) parseReturnStatement() (Value, error) {
	if !p.allowReturn {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "return statements are only supported inside bounded function bodies")
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

func (p *classicJSStatementParser) parseExportStatement() (Value, error) {
	p.skipSpaceAndComments()
	if p.consumeByte('*') {
		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("from"); ok {
			p.pos += len(keyword)
			module, err := p.parseModuleNamespaceReference()
			if err != nil {
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
			return UndefinedValue(), NewError(ErrorKindUnsupported, "export default const declarations are not supported in this bounded classic-JS slice")
		}
		if _, ok := p.peekKeyword("let"); ok {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "export default let declarations are not supported in this bounded classic-JS slice")
		}
		if _, ok := p.peekKeyword("class"); ok {
			p.pos += len("class")
			_, value, err := p.parseClassDeclaration(true)
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
				name, value, err := p.parseFunctionLiteral(true, true, generator)
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
			p.pos = start
		}
		if _, ok := p.peekKeyword("function"); ok {
			start := p.pos
			p.pos += len("function")
			p.skipSpaceAndComments()
			switch {
			case p.peekByte() == '*' || p.peekByte() == '(':
				p.pos = start
				value, err := p.parseExpression()
				if err != nil {
					return UndefinedValue(), err
				}
				if p.moduleExports != nil {
					p.moduleExports["default"] = value
				}
				return value, nil
			case isIdentStart(p.peekByte()):
				name, value, err := p.parseFunctionLiteral(true, false, false)
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
			default:
				p.pos = start
			}
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
			if isClassicJSReservedDeclarationName(local) {
				return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported export specifier name %q in this bounded classic-JS slice", local))
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
				if isClassicJSReservedDeclarationName(alias) {
					return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported export alias name %q in this bounded classic-JS slice", alias))
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
	if p.consumeByte('(') {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "dynamic import() is not supported in this bounded classic-JS slice")
	}

	if p.peekByte() == '\'' || p.peekByte() == '"' {
		module, err := p.parseModuleSpecifier()
		if err != nil {
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
			return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported import binding name %q in this bounded classic-JS slice", name))
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
		if defaultName != "" {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "default imports cannot be combined with namespace imports in this bounded classic-JS slice")
		}
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
			return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported namespace import binding name %q in this bounded classic-JS slice", name))
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
		ns, err := p.lookupModuleNamespace(module)
		if err != nil {
			return UndefinedValue(), err
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
		if isClassicJSReservedDeclarationName(imported) {
			return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported import specifier name %q in this bounded classic-JS slice", imported))
		}
		local := imported

		p.skipSpaceAndComments()
		if keyword, ok := p.peekKeyword("as"); ok {
			p.pos += len(keyword)
			p.skipSpaceAndComments()
			alias, err := p.parseIdentifier()
			if err != nil {
				return nil, NewError(ErrorKindParse, "expected import alias identifier in this bounded classic-JS slice")
			}
			if isClassicJSReservedDeclarationName(alias) {
				return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported import alias name %q in this bounded classic-JS slice", alias))
			}
			local = alias
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

func (p *classicJSStatementParser) parseFunctionLiteral(allowAnonymous bool, async bool, generator bool) (string, Value, error) {
	p.skipSpaceAndComments()
	if generator {
		if !p.consumeByte('*') {
			return "", UndefinedValue(), NewError(ErrorKindParse, "expected `*` after `function` in this bounded classic-JS slice")
		}
		p.skipSpaceAndComments()
	} else if p.consumeByte('*') {
		return "", UndefinedValue(), NewError(ErrorKindUnsupported, "generator function declarations are not supported in this bounded classic-JS slice")
	}

	name := ""
	if isIdentStart(p.peekByte()) {
		parsedName, err := p.parseIdentifier()
		if err != nil {
			return "", UndefinedValue(), err
		}
		if isClassicJSReservedDeclarationName(parsedName) {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported function name %q in this bounded classic-JS slice", parsedName))
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
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
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
				return UndefinedValue(), NewError(ErrorKindParse, "destructuring declarations require an initializer in this bounded classic-JS slice")
			}
			p.skipSpaceAndComments()
			if p.eof() {
				return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
			}
			value, err := p.parseExpression()
			if err != nil {
				return UndefinedValue(), err
			}
			if err := p.declareBindingPattern(pattern, value, kind == "let"); err != nil {
				return UndefinedValue(), err
			}
		} else {
			name, err := p.parseIdentifier()
			if err != nil {
				return UndefinedValue(), err
			}
			if isClassicJSReservedDeclarationName(name) {
				return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported lexical binding name %q in this bounded classic-JS slice", name))
			}

			p.skipSpaceAndComments()
			value := UndefinedValue()
			if p.consumeByte('=') {
				p.skipSpaceAndComments()
				if p.eof() {
					return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
				}
				parsed, err := p.parseExpression()
				if err != nil {
					return UndefinedValue(), err
				}
				value = parsed
			} else if kind == "const" {
				return UndefinedValue(), NewError(ErrorKindParse, "const declarations require an initializer in this bounded classic-JS slice")
			}

			if err := p.env.declare(name, scalarJSValue(value), kind == "let"); err != nil {
				return UndefinedValue(), err
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
	kind       classicJSBindingPatternKind
	name       string
	elements   []classicJSBindingPattern
	properties []classicJSObjectBindingProperty
}

type classicJSObjectBindingProperty struct {
	key     string
	pattern classicJSBindingPattern
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
			return classicJSBindingPattern{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported lexical binding name %q in this bounded classic-JS slice", name))
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
				return classicJSBindingPattern{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported lexical binding name %q in this bounded classic-JS slice", name))
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
			return classicJSBindingPattern{}, NewError(ErrorKindUnsupported, "default binding values are not supported in this bounded classic-JS slice")
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
				return classicJSBindingPattern{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported lexical binding name %q in this bounded classic-JS slice", name))
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

		key, err := p.parseObjectBindingKey()
		if err != nil {
			return classicJSBindingPattern{}, err
		}

		p.skipSpaceAndComments()
		if p.consumeByte(':') {
			p.skipSpaceAndComments()
			if p.eof() {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "unexpected end of script source")
			}
			pattern, err := p.parseBindingPattern()
			if err != nil {
				return classicJSBindingPattern{}, err
			}
			properties = append(properties, classicJSObjectBindingProperty{key: key.name, pattern: pattern})
		} else {
			if !key.identifier {
				return classicJSBindingPattern{}, NewError(ErrorKindParse, "object binding shorthand requires an identifier name")
			}
			if isClassicJSReservedDeclarationName(key.name) {
				return classicJSBindingPattern{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported lexical binding name %q in this bounded classic-JS slice", key.name))
			}
			properties = append(properties, classicJSObjectBindingProperty{key: key.name, pattern: classicJSBindingPattern{kind: classicJSBindingPatternIdentifier, name: key.name}})
		}

		p.skipSpaceAndComments()
		if p.consumeByte('=') {
			return classicJSBindingPattern{}, NewError(ErrorKindUnsupported, "default binding values are not supported in this bounded classic-JS slice")
		}
		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return classicJSBindingPattern{kind: classicJSBindingPatternObject, properties: properties}, nil
		}
		if !p.consumeByte(',') {
			return classicJSBindingPattern{}, NewError(ErrorKindParse, "object binding patterns must separate properties with commas")
		}
	}
}

func (p *classicJSStatementParser) parseObjectBindingKey() (classicJSObjectKey, error) {
	p.skipSpaceAndComments()
	if p.eof() {
		return classicJSObjectKey{}, NewError(ErrorKindParse, "unexpected end of script source")
	}

	switch ch := p.peekByte(); ch {
	case '\'', '"':
		value, err := p.parseStringLiteral()
		if err != nil {
			return classicJSObjectKey{}, err
		}
		return classicJSObjectKey{name: value.String, identifier: false}, nil
	default:
		if isDigit(ch) {
			value, err := p.parseNumberLiteral()
			if err != nil {
				return classicJSObjectKey{}, err
			}
			return classicJSObjectKey{name: ToJSString(value), identifier: false}, nil
		}
		ident, err := p.parseIdentifier()
		if err != nil {
			return classicJSObjectKey{}, err
		}
		return classicJSObjectKey{name: ident, identifier: true}, nil
	}
}

func (p *classicJSStatementParser) declareBindingPattern(pattern classicJSBindingPattern, value Value, mutable bool) error {
	assignments := make([]classicJSBindingAssignment, 0, 4)
	if err := collectBindingAssignments(pattern, value, mutable, &assignments); err != nil {
		return err
	}
	for _, assignment := range assignments {
		if err := p.env.declare(assignment.name, scalarJSValue(assignment.value), assignment.mutable); err != nil {
			return err
		}
	}
	return nil
}

type classicJSBindingAssignment struct {
	name    string
	value   Value
	mutable bool
}

func collectBindingAssignments(pattern classicJSBindingPattern, value Value, mutable bool, assignments *[]classicJSBindingAssignment) error {
	switch pattern.kind {
	case classicJSBindingPatternIdentifier:
		*assignments = append(*assignments, classicJSBindingAssignment{name: pattern.name, value: value, mutable: mutable})
		return nil
	case classicJSBindingPatternHole:
		return nil
	case classicJSBindingPatternRest:
		return NewError(ErrorKindParse, "rest binding syntax must appear directly inside array or object binding patterns")
	case classicJSBindingPatternArray:
		return collectArrayBindingAssignments(pattern.elements, value, mutable, assignments)
	case classicJSBindingPatternObject:
		return collectObjectBindingAssignments(pattern.properties, value, mutable, assignments)
	default:
		return NewError(ErrorKindParse, "unsupported binding pattern in this bounded classic-JS slice")
	}
}

func collectArrayBindingAssignments(elements []classicJSBindingPattern, value Value, mutable bool, assignments *[]classicJSBindingAssignment) error {
	if value.Kind != ValueKindArray {
		return NewError(ErrorKindUnsupported, "array destructuring only works on array values in this bounded classic-JS slice")
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
			if sourceIndex < len(value.Array) {
				restElements = value.Array[sourceIndex:]
			}
			*assignments = append(*assignments, classicJSBindingAssignment{name: element.name, value: ArrayValue(restElements), mutable: mutable})
			sourceIndex = len(value.Array)
		default:
			elementValue := UndefinedValue()
			if sourceIndex < len(value.Array) {
				elementValue = value.Array[sourceIndex]
			}
			if err := collectBindingAssignments(element, elementValue, mutable, assignments); err != nil {
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

func collectObjectBindingAssignments(properties []classicJSObjectBindingProperty, value Value, mutable bool, assignments *[]classicJSBindingAssignment) error {
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
		if err := collectBindingAssignments(property.pattern, propertyValue, mutable, assignments); err != nil {
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
			return jsValue{}, false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported generator function name %q in this bounded classic-JS slice", parsedName))
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

	params := make([]classicJSFunctionParameter, 0, 4)
	restName := ""
	p.pos++
	p.skipSpaceAndComments()
	if !p.consumeByte(')') {
		for {
			p.skipSpaceAndComments()
			if p.consumeEllipsis() {
				p.skipSpaceAndComments()
				name, err := p.parseIdentifier()
				if err != nil {
					p.pos = start
					return jsValue{}, false, nil
				}
				if isClassicJSReservedDeclarationName(name) {
					return jsValue{}, false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported generator function parameter name %q in this bounded classic-JS slice", name))
				}
				restName = name
				p.skipSpaceAndComments()
				if !p.consumeByte(')') {
					return jsValue{}, false, NewError(ErrorKindParse, "rest parameters must be the final parameter in this bounded classic-JS slice")
				}
				break
			}

			switch p.peekByte() {
			case '[', '{', '=', ':':
				p.pos = start
				return jsValue{}, false, nil
			}

			name, err := p.parseIdentifier()
			if err != nil {
				p.pos = start
				return jsValue{}, false, nil
			}
			params = append(params, classicJSFunctionParameter{name: name})

			p.skipSpaceAndComments()
			if p.consumeByte(')') {
				break
			}
			if p.consumeByte(',') {
				p.skipSpaceAndComments()
				if p.consumeByte(')') {
					break
				}
				continue
			}
			p.pos = start
			return jsValue{}, false, nil
		}
	}

	for _, param := range params {
		if isClassicJSReservedDeclarationName(param.name) {
			return jsValue{}, false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported generator function parameter name %q in this bounded classic-JS slice", param.name))
		}
	}
	if restName != "" && isClassicJSReservedDeclarationName(restName) {
		return jsValue{}, false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported generator function parameter name %q in this bounded classic-JS slice", restName))
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
	if p.peekByte() == '*' {
		p.pos = start
		return jsValue{}, false, nil
	}

	name := ""
	if isIdentStart(p.peekByte()) {
		parsedName, err := p.parseIdentifier()
		if err != nil {
			p.pos = start
			return jsValue{}, false, nil
		}
		if isClassicJSReservedDeclarationName(parsedName) {
			return jsValue{}, false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported function name %q in this bounded classic-JS slice", parsedName))
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
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
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

	consequent, err := p.consumeBlockSource()
	if err != nil {
		return UndefinedValue(), err
	}

	var elseSource string
	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("else"); ok {
		p.pos += len(keyword)
		elseSource, err = p.consumeBlockSource()
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

func (p *classicJSStatementParser) resumeClassicJSState(state classicJSResumeState) (Value, classicJSResumeState, error) {
	switch current := state.(type) {
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
	default:
		return UndefinedValue(), nil, NewError(ErrorKindRuntime, "unsupported loop state kind")
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

		_, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYield(statement, p.host, frame.bodyEnv, p.stepLimit, p.allowAwait, p.allowYield, p.allowReturn, frame, nil)
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

	bodySource, err := p.consumeBlockSource()
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
	bodySource, err := p.consumeBlockSource()
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
	headerSource, err := p.consumeParenthesizedSource("for")
	if err != nil {
		return UndefinedValue(), err
	}

	initSource, conditionSource, updateSource, err := splitClassicJSForHeader(headerSource)
	if err != nil {
		return UndefinedValue(), err
	}

	bodySource, err := p.consumeBlockSource()
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
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class name %q in this bounded classic-JS slice", parsedName))
		}
		name = parsedName
	}

	p.skipSpaceAndComments()
	classEnv := p.env
	if classEnv == nil {
		classEnv = newClassicJSEnvironment()
		p.env = classEnv
	}

	var baseClassValue jsValue
	var baseClassDef *classicJSClassDefinition
	if keyword, ok := p.peekKeyword("extends"); ok {
		p.pos += len(keyword)
		p.skipSpaceAndComments()
		baseName, err := p.parseIdentifier()
		if err != nil {
			return "", UndefinedValue(), NewError(ErrorKindParse, "class inheritance requires a base class identifier in this bounded classic-JS slice")
		}
		if name != "" && baseName == name {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, "class inheritance cannot extend the class being declared in this bounded classic-JS slice")
		}
		baseClassValue, ok = classEnv.lookup(baseName)
		if !ok || baseClassValue.kind != jsValueScalar || baseClassValue.value.Kind != ValueKindObject {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("class inheritance requires a previously declared class %q in this bounded classic-JS slice", baseName))
		}
		baseClassDef, ok = classEnv.classDefinition(baseName)
		if !ok || baseClassDef == nil {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("class inheritance requires a previously declared class %q in this bounded classic-JS slice", baseName))
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
		prototypeValue, ok := lookupObjectProperty(baseClassValue.value.Object, "prototype")
		if !ok || prototypeValue.Kind != ValueKindObject {
			return "", UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("class inheritance requires a prototype object for %q in this bounded classic-JS slice", name))
		}
		classDef.superInstanceTarget = prototypeValue
		prototypeEntries = append(prototypeEntries, append([]ObjectEntry(nil), prototypeValue.Object...)...)
		for _, entry := range baseClassValue.value.Object {
			if entry.Key == "prototype" {
				continue
			}
			staticEntries = append(staticEntries, entry)
		}
	}
	if name != "" {
		classEnv.setClassDefinition(name, classDef)
	}
	currentClassValue := ObjectValue([]ObjectEntry{
		{
			Key:   "prototype",
			Value: ObjectValue(append([]ObjectEntry(nil), prototypeEntries...)),
		},
	})
	publishClassValue := func() {
		entries := make([]ObjectEntry, 0, 1+len(staticEntries))
		entries = append(entries, ObjectEntry{
			Key:   "prototype",
			Value: ObjectValue(append([]ObjectEntry(nil), prototypeEntries...)),
		})
		entries = append(entries, staticEntries...)
		currentClassValue = ObjectValue(entries)
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
			fieldName, err := p.resolveClassicJSClassMemberName(member.fieldName, member.fieldNameSource, classEnv)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && fieldName == "prototype" {
				return "", UndefinedValue(), NewError(ErrorKindUnsupported, "static class members named `prototype` are not supported in this bounded classic-JS slice")
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
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && methodName == "prototype" {
				return "", UndefinedValue(), NewError(ErrorKindUnsupported, "static class members named `prototype` are not supported in this bounded classic-JS slice")
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
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && methodName == "prototype" {
				return "", UndefinedValue(), NewError(ErrorKindUnsupported, "static class members named `prototype` are not supported in this bounded classic-JS slice")
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
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv)
			if err != nil {
				return "", UndefinedValue(), err
			}
			if !member.private && methodName == "prototype" {
				return "", UndefinedValue(), NewError(ErrorKindUnsupported, "static class members named `prototype` are not supported in this bounded classic-JS slice")
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
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv)
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
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv)
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
			methodName, err := p.resolveClassicJSClassMemberName(member.methodName, member.methodNameSource, classEnv)
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
			fieldName, err := p.resolveClassicJSClassMemberName(member.fieldName, member.fieldNameSource, classEnv)
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

	p.env = classEnv

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
		catchName     string
		catchBound    bool
		hasCatch      bool
		hasFinally    bool
	)

	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("catch"); ok {
		hasCatch = true
		p.pos += len(keyword)
		catchName, catchBound, err = p.parseCatchBinding()
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
		catchName:        catchName,
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

		value, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYield(statement, p.host, state.env, p.stepLimit, p.allowAwait, true, p.allowReturn, state.owner, nil)
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
						return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "continue statements cannot target labeled switch statements in this bounded classic-JS slice")
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

			_, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYield(statement, p.host, state.env, p.stepLimit, p.allowAwait, true, p.allowReturn, state, nil)
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
						return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "continue statements cannot target labeled switch statements in this bounded classic-JS slice")
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
					if err := catchEnv.declare(state.catchName, scalarJSValue(catchValue), true); err != nil {
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
			return UndefinedValue(), nil, NewError(ErrorKindUnsupported, "continue statements cannot target labeled try statements in this bounded classic-JS slice")
		}
		return UndefinedValue(), nil, state.pendingErr
	}
	return state.result, nil, nil
}

func (p *classicJSStatementParser) parseCatchBinding() (string, bool, error) {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return "", false, nil
	}

	p.skipSpaceAndComments()
	name, err := p.parseIdentifier()
	if err != nil {
		return "", false, err
	}
	if isClassicJSReservedDeclarationName(name) {
		return "", false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported catch binding name %q in this bounded classic-JS slice", name))
	}

	p.skipSpaceAndComments()
	if !p.consumeByte(')') {
		return "", false, NewError(ErrorKindParse, "unterminated `catch` binding")
	}
	return name, true, nil
}

func (p *classicJSStatementParser) evalSwitchClauseBody(body string, env *classicJSEnvironment) (bool, error) {
	statements, err := splitScriptStatements(body)
	if err != nil {
		return false, NewError(ErrorKindParse, err.Error())
	}
	for _, statement := range statements {
		trimmed := strings.TrimSpace(statement)
		if trimmed == "" {
			continue
		}
		if isClassicJSBreakStatement(trimmed) {
			return true, nil
		}
		if _, err := p.evalStatementWithEnv(statement, env); err != nil {
			if _, ok := classicJSYieldSignalValue(err); ok {
				return false, NewError(ErrorKindUnsupported, "yield inside switch clauses is not supported in this bounded classic-JS slice")
			}
			return false, err
		}
	}
	return false, nil
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
		return "", "", false, NewError(ErrorKindUnsupported, "generator class methods are not supported in this bounded classic-JS slice")
	}

	name, err := scanner.parseIdentifier()
	if err != nil {
		return "", "", false, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class body element at %q in this bounded classic-JS slice", scanner.remainingPreview()))
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

			return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class member at %q in this bounded classic-JS slice", scanner.remainingPreview()))
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

		return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class body element at %q in this bounded classic-JS slice", scanner.remainingPreview()))
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
				return nil, "", NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported %s parameter name %q in this bounded classic-JS slice", label, name))
			}
			restName = name
			parser.skipSpaceAndComments()
			if !parser.eof() {
				return nil, "", NewError(ErrorKindParse, fmt.Sprintf("%s rest parameter must be the final parameter in this bounded classic-JS slice", label))
			}
			break
		}

		switch parser.peekByte() {
		case '[', '{', ':':
			return nil, "", NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported %s parameter syntax in this bounded classic-JS slice", label))
		}

		name, err := parser.parseIdentifier()
		if err != nil {
			return nil, "", err
		}
		if isClassicJSReservedDeclarationName(name) {
			return nil, "", NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported %s parameter name %q in this bounded classic-JS slice", label, name))
		}
		param := classicJSFunctionParameter{name: name}

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

		if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 && ch == ',' {
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
		return scanner.pos, NewError(ErrorKindParse, "unterminated quoted string in function parameter default")
	}
	if blockComment {
		return scanner.pos, NewError(ErrorKindParse, "unterminated block comment in function parameter default")
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
	if discriminant.Kind != candidate.Kind {
		return false, nil
	}

	switch discriminant.Kind {
	case ValueKindUndefined, ValueKindNull:
		return true, nil
	case ValueKindString:
		return discriminant.String == candidate.String, nil
	case ValueKindBool:
		return discriminant.Bool == candidate.Bool, nil
	case ValueKindNumber:
		if math.IsNaN(discriminant.Number) || math.IsNaN(candidate.Number) {
			return false, nil
		}
		return discriminant.Number == candidate.Number, nil
	case ValueKindBigInt:
		return discriminant.BigInt == candidate.BigInt, nil
	default:
		return false, NewError(ErrorKindUnsupported, "switch discriminants only work on scalar values in this bounded classic-JS slice")
	}
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
		if left.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "nullish coalescing only works on scalar values in this slice")
		}
		if !isNullishJSValue(left.value) {
			// Short-circuit the right-hand side without running host side effects.
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = p.pos
			if _, err := skip.parseExpression(); err != nil {
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
		if left.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "logical `||` only works on scalar values in this bounded classic-JS slice")
		}
		if jsTruthy(left.value) {
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = p.pos
			if _, err := skip.parseExpression(); err != nil {
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
	left, err := p.parseEquality()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		if p.pos+1 >= len(p.source) || p.source[p.pos] != '&' || p.source[p.pos+1] != '&' {
			return left, nil
		}

		p.pos += 2
		if left.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "logical `&&` only works on scalar values in this bounded classic-JS slice")
		}
		if !jsTruthy(left.value) {
			skip := p.cloneForSkipping(skipHostBindings{delegate: p.host})
			skip.pos = p.pos
			if _, err := skip.parseExpression(); err != nil {
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
		if left.kind != jsValueScalar || right.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "equality comparisons only work on scalar values in this bounded classic-JS slice")
		}
		equal := classicJSEqualValues(left.value, right.value, op)
		left = scalarJSValue(BoolValue(equal))
	}
}

func (p *classicJSStatementParser) parseRelational() (jsValue, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		op := ""
		switch {
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], "<="):
			op = "<="
		case p.pos+2 <= len(p.source) && strings.HasPrefix(p.source[p.pos:], ">="):
			op = ">="
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

		right, err := p.parseAdditive()
		if err != nil {
			return jsValue{}, err
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

	switch p.peekByte() {
	case '+':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar || value.value.Kind != ValueKindNumber {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `+` is only supported for numeric literals in this slice")
		}
		return scalarJSValue(NumberValue(value.value.Number)), nil
	case '-':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `-` is only supported for numeric literals in this slice")
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
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `-` is only supported for numeric literals in this slice")
		}
	case '!':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "logical negation only works on scalar values in this slice")
		}
		return scalarJSValue(BoolValue(!jsTruthy(value.value))), nil
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
			return jsValue{}, NewError(ErrorKindUnsupported, "`await` is only supported inside bounded async arrow functions in this slice")
		}
		p.pos += len(keyword)
		value, err := p.parseUnary()
		if err != nil {
			return jsValue{}, err
		}
		if value.kind != jsValueScalar {
			return jsValue{}, NewError(ErrorKindUnsupported, "await only works on scalar values in this bounded classic-JS slice")
		}
		return scalarJSValue(unwrapPromiseValue(value.value)), nil
	}

	if keyword, ok := p.peekKeyword("delete"); ok {
		p.pos += len(keyword)
		return p.parseDeleteExpression()
	}

	if keyword, ok := p.peekKeyword("new"); ok {
		p.pos += len(keyword)
		return p.parseNewExpression()
	}

	return p.parsePostfix()
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
					p.pos++
					args, err := p.parseArguments()
					if err != nil {
						return jsValue{}, err
					}
					if len(args) != 0 {
						return jsValue{}, NewError(ErrorKindUnsupported, "constructor arguments are not supported in this bounded classic-JS slice")
					}
					value, err := p.instantiateClassicJSClass(name)
					if err != nil {
						return jsValue{}, err
					}
					return p.parsePostfixTail(value)
				}
			}
		}
		p.pos = start
	}

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

	if callee.kind == jsValueScalar && callee.value.Kind == ValueKindFunction {
		value, err := p.invoke(callee, args)
		if err != nil {
			return jsValue{}, err
		}
		return p.parsePostfixTail(value)
	}
	return jsValue{}, NewError(ErrorKindUnsupported, "new expressions only work on class, constructor, or callable values in this bounded classic-JS slice")
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
	if !found {
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("delete target %q is not a declared local binding in this bounded classic-JS slice", baseName))
	}
	if baseValue.kind != jsValueScalar {
		return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on scalar object bindings in this bounded classic-JS slice")
	}
	if baseValue.value.Kind != ValueKindObject {
		return jsValue{}, NewError(ErrorKindUnsupported, "delete only works on object values in this bounded classic-JS slice")
	}

	updated, err := deleteJSValuePropertyChain(baseValue.value, steps, p.privateFieldPrefix)
	if err != nil {
		return jsValue{}, err
	}
	if p.env != nil {
		if err := p.env.assign(baseName, scalarJSValue(updated)); err != nil {
			return jsValue{}, err
		}
	}
	return scalarJSValue(BoolValue(true)), nil
}

func (p *classicJSStatementParser) parseDeleteAccessSteps() ([]classicJSDeleteStep, error) {
	steps, ok, err := p.scanPropertyAccessSteps(true)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewError(ErrorKindUnsupported, "optional chaining is not supported in delete expressions in this bounded classic-JS slice")
	}
	return steps, nil
}

func (p *classicJSStatementParser) scanAssignmentAccessSteps() ([]classicJSDeleteStep, bool, error) {
	return p.scanPropertyAccessSteps(false)
}

func (p *classicJSStatementParser) scanPropertyAccessSteps(rejectOptional bool) ([]classicJSDeleteStep, bool, error) {
	steps := make([]classicJSDeleteStep, 0, 2)
	for {
		p.skipSpaceAndComments()
		if p.eof() {
			return steps, true, nil
		}
		if p.peekByte() == '?' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '.' {
			if rejectOptional {
				return nil, false, nil
			}
			return nil, false, nil
		}
		switch {
		case p.consumeByte('.'):
			name, err := p.parseMemberAccessName()
			if err != nil {
				return nil, false, err
			}
			steps = append(steps, classicJSDeleteStep{
				key:     name,
				private: strings.HasPrefix(name, "#"),
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
			steps = append(steps, classicJSDeleteStep{key: ToJSString(value)})
		default:
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
				value, err = p.resolveBracketAccess(value, index)
				if err != nil {
					return jsValue{}, err
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
				value, err = p.resolveMemberAccess(value, name)
				if err != nil {
					return jsValue{}, err
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
			value, err = p.resolveMemberAccess(value, name)
			if err != nil {
				return jsValue{}, err
			}

		case p.consumeByte('['):
			source, err := p.consumeBracketAccessExpressionSource()
			if err != nil {
				return jsValue{}, err
			}
			if shortCircuited {
				continue
			}
			index, err := p.evalExpressionWithEnv(source, p.env)
			if err != nil {
				return jsValue{}, err
			}
			value, err = p.resolveBracketAccess(value, index)
			if err != nil {
				return jsValue{}, err
			}

		case p.consumeByte(':'):
			method, err := p.parseIdentifier()
			if err != nil {
				return jsValue{}, err
			}
			if shortCircuited {
				continue
			}
			if value.kind != jsValueHostObject {
				return jsValue{}, NewError(
					ErrorKindUnsupported,
					"unsupported legacy host syntax outside a `host:method(...)` call",
				)
			}
			value = hostMethodJSValue(method)

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

		default:
			return value, nil
		}
	}
}

func (p *classicJSStatementParser) resolveMemberAccess(value jsValue, name string) (jsValue, error) {
	switch value.kind {
	case jsValueHostObject:
		return hostMethodJSValue(name), nil
	case jsValueSuper:
		switch value.value.Kind {
		case ValueKindObject:
			if resolved, ok := lookupObjectProperty(value.value.Object, name); ok {
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.receiver, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
			return scalarJSValue(UndefinedValue()), nil
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
		if value.value.Kind == ValueKindString {
			switch name {
			case "length":
				return scalarJSValue(NumberValue(float64(len(value.value.String)))), nil
			default:
				return jsValue{}, NewError(
					ErrorKindUnsupported,
					"unsupported member access in this bounded classic-JS slice; only object properties, string `length`, array `length`, and `host.method(...)` are available",
				)
			}
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
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported member access in this bounded classic-JS slice; only object properties, array `length`, and `host.method(...)` are available",
			)
		default:
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported member access in this bounded classic-JS slice; only object properties, array `length`, and `host.method(...)` are available",
			)
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
				if resolved.Kind == ValueKindFunction && resolved.Function != nil {
					return jsValue{kind: jsValueScalar, value: resolved, receiver: value.receiver, hasReceiver: true}, nil
				}
				return scalarJSValue(resolved), nil
			}
			return scalarJSValue(UndefinedValue()), nil
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
			return scalarJSValue(UndefinedValue()), nil
		default:
			return jsValue{}, NewError(
				ErrorKindUnsupported,
				"unsupported bracket access in this bounded classic-JS slice; only object properties, array indexes, array `length`, and `host[\"method\"]` are available",
			)
		}
	default:
		return jsValue{}, NewError(
			ErrorKindUnsupported,
			"unsupported bracket access in this bounded classic-JS slice; only object properties, array indexes, array `length`, and `host[\"method\"]` are available",
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

func deleteJSValuePropertyChain(value Value, steps []classicJSDeleteStep, privateFieldPrefix string) (Value, error) {
	if len(steps) == 0 {
		return value, nil
	}
	if value.Kind != ValueKindObject {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "delete only works on object values in this bounded classic-JS slice")
	}

	key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}

	if len(steps) == 1 {
		return ObjectValue(deleteObjectProperty(value.Object, key)), nil
	}

	child, ok := lookupObjectProperty(value.Object, key)
	if !ok {
		return value, nil
	}

	updatedChild, err := deleteJSValuePropertyChain(child, steps[1:], privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}
	return ObjectValue(replaceObjectProperty(value.Object, key, updatedChild)), nil
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

func classicJSObjectSpreadEntries(entries []ObjectEntry) []ObjectEntry {
	if len(entries) == 0 {
		return nil
	}
	cloned := make([]ObjectEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Key, "\x00classic-js-setter:") {
			continue
		}
		cloned = append(cloned, entry)
	}
	return cloned
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
	if value.Kind != ValueKindObject {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on object values in this bounded classic-JS slice")
	}

	key, err := deleteJSPropertyKey(steps[0], privateFieldPrefix)
	if err != nil {
		return UndefinedValue(), err
	}

	if len(steps) == 1 {
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
			return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on data properties in this bounded classic-JS slice")
		}
		value.Object[index].Value = rhs
		return value, nil
	}

	child, ok := lookupObjectProperty(value.Object, key)
	if !ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on existing object properties in this bounded classic-JS slice")
	}
	if child.Kind == ValueKindFunction && child.Function != nil && child.Function.objectAccessor {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "assignment only works on data properties in this bounded classic-JS slice")
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
			if len(args) != 1 {
				return jsValue{}, NewError(ErrorKindUnsupported, "dynamic import() requires exactly one module specifier in this bounded classic-JS slice")
			}
			if args[0].Kind != ValueKindString {
				return jsValue{}, NewError(ErrorKindUnsupported, "dynamic import() requires a string module specifier in this bounded classic-JS slice")
			}
			module, err := p.lookupModuleNamespace(args[0].String)
			if err != nil {
				return jsValue{}, err
			}
			return scalarJSValue(PromiseValue(module)), nil
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
			return value, nil
		}
	}
	if p.allowUnknownIdentifiers {
		return scalarJSValue(UndefinedValue()), nil
	}

	switch ident {
	case "let", "const", "var", "function", "class", "if", "else", "for", "while", "do", "switch", "case", "default", "try", "catch", "finally", "return", "break", "continue", "throw", "async", "await", "import", "export", "delete", "yield", "super":
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
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			value, err := p.parseExpression()
			if err != nil {
				return UndefinedValue(), err
			}
			if value.Kind != ValueKindArray {
				return UndefinedValue(), NewError(ErrorKindUnsupported, "array spread only works on array values in this bounded classic-JS slice")
			}
			elements = append(elements, value.Array...)
			p.skipSpaceAndComments()
			if p.consumeByte(']') {
				return ArrayValue(elements), nil
			}
			if !p.consumeByte(',') {
				return UndefinedValue(), NewError(ErrorKindParse, "array literals must separate elements with commas")
			}
			continue
		}

		value, err := p.parseExpression()
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
	for {
		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return ObjectValue(entries), nil
		}
		if p.consumeEllipsis() {
			p.skipSpaceAndComments()
			value, err := p.parseExpression()
			if err != nil {
				return UndefinedValue(), err
			}
			if value.Kind != ValueKindObject {
				return UndefinedValue(), NewError(ErrorKindUnsupported, "object spread only works on object values in this bounded classic-JS slice")
			}
			entries = append(entries, classicJSObjectSpreadEntries(value.Object)...)
			p.skipSpaceAndComments()
			if p.consumeByte('}') {
				return ObjectValue(entries), nil
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
			value, err = p.parseExpression()
			if err != nil {
				return UndefinedValue(), err
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
				return UndefinedValue(), NewError(ErrorKindUnsupported, "unsupported object literal member syntax in this bounded classic-JS slice; only shorthand properties and shorthand methods are available")
			default:
				return UndefinedValue(), NewError(ErrorKindParse, "object literal properties must use `:` or method syntax")
			}
		}
		entries = appendClassicJSObjectLiteralEntry(entries, key, value)

		p.skipSpaceAndComments()
		if p.consumeByte('}') {
			return ObjectValue(entries), nil
		}
		if !p.consumeByte(',') {
			return UndefinedValue(), NewError(ErrorKindParse, "object literals must separate properties with commas")
		}
	}
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
		bodySource, err := p.consumeBlockSource()
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
		env:                p.env,
		privateClass:       p.privateClass,
		privateFieldPrefix: p.privateFieldPrefix,
	}), nil
}

func (p *classicJSStatementParser) resolveObjectLiteralShorthandValue(name string) (Value, error) {
	if p.env != nil {
		if value, ok := p.env.lookup(name); ok {
			if value.kind != jsValueScalar {
				return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported object shorthand property value %q in this bounded classic-JS slice", name))
			}
			return value.value, nil
		}
	}
	if p.allowUnknownIdentifiers {
		return UndefinedValue(), nil
	}
	return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported identifier %q in this bounded classic-JS slice", name))
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
		case ValueKindFunction:
			if callee.value.NativeFunction != nil {
				value, err := callee.value.NativeFunction(args)
				if err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(value), nil
			}
			if callee.value.Function == nil {
				return jsValue{}, NewError(ErrorKindUnsupported, "unsupported call expression in this bounded classic-JS slice")
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
				return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported call expression for browser surface %q in this bounded classic-JS slice", callee.value.HostReferencePath))
			}
			if resolved.NativeFunction != nil {
				value, err := resolved.NativeFunction(args)
				if err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(value), nil
			}
			if resolved.Function == nil {
				return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported call expression for browser surface %q in this bounded classic-JS slice", callee.value.HostReferencePath))
			}
			return p.invokeArrowFunction(resolved.Function, args, callee)
		default:
			return jsValue{}, NewError(ErrorKindUnsupported, "unsupported call expression in this bounded classic-JS slice")
		}
	case jsValueSuper:
		if callee.value.Kind != ValueKindObject {
			return jsValue{}, NewError(ErrorKindUnsupported, "unsupported `super()` call in this bounded classic-JS slice; only object-backed class targets are available")
		}
		constructorValue, ok := lookupObjectProperty(callee.value.Object, "constructor")
		if !ok || constructorValue.Kind != ValueKindFunction || constructorValue.Function == nil {
			return jsValue{}, NewError(ErrorKindUnsupported, "unsupported `super()` call in this bounded classic-JS slice; the base target does not expose a constructor")
		}
		constructorCall := scalarJSValue(constructorValue)
		constructorCall.receiver = callee.receiver
		constructorCall.hasReceiver = true
		return p.invoke(constructorCall, args)
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, "unsupported call expression in this bounded classic-JS slice")
	}
}

func (p *classicJSStatementParser) bindClassicJSFunctionParameters(callEnv *classicJSEnvironment, params []classicJSFunctionParameter, restName string, args []Value, allowAwait bool, privateClass *classicJSClassDefinition) error {
	for _, param := range params {
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
			parsed, err := evalClassicJSExpressionWithEnvAndAllowAwaitAndYield(param.defaultSource, p.host, callEnv, p.stepLimit, allowAwait, false, privateClass)
			if err != nil {
				return err
			}
			value = parsed
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
		return p.invokeGeneratorNext(fn.generatorState, args)
	}
	if fn.generatorFunction != nil {
		return p.invokeGeneratorFunction(fn.generatorFunction, args, callee)
	}

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
		if err := callEnv.declare("super", superJSValue(fn.superTarget, callee.receiver), false); err != nil {
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

	if fn.bodyIsBlock {
		_, err := evalClassicJSProgramWithAllowAwaitAndYieldAndExports(fn.body, p.host, callEnv, p.stepLimit, fn.async, false, fn.allowReturn, nil, fn.privateClass, nil)
		if err != nil {
			if returnedValue, ok := classicJSReturnSignalValue(err); ok {
				if fn.async {
					return scalarJSValue(PromiseValue(unwrapPromiseValue(returnedValue))), nil
				}
				return scalarJSValue(returnedValue), nil
			}
			return jsValue{}, err
		}
		if fn.async {
			return scalarJSValue(PromiseValue(UndefinedValue())), nil
		}
		return scalarJSValue(UndefinedValue()), nil
	}

	value, err := evalClassicJSExpressionWithEnvAndAllowAwait(fn.body, p.host, callEnv, p.stepLimit, fn.async, fn.privateClass)
	if err != nil {
		return jsValue{}, err
	}
	if fn.async {
		return scalarJSValue(PromiseValue(unwrapPromiseValue(value))), nil
	}
	return scalarJSValue(value), nil
}

func (p *classicJSStatementParser) invokeGeneratorFunction(fn *classicJSGeneratorFunction, args []Value, callee jsValue) (jsValue, error) {
	if fn == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "generator function is unavailable")
	}

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
		if err := callEnv.declare("super", superJSValue(fn.superTarget, callee.receiver), false); err != nil {
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
		statements: statements,
		env:        callEnv,
		async:      fn.async,
	}
	nextFn := &classicJSArrowFunction{
		generatorState: state,
	}
	return scalarJSValue(ObjectValue([]ObjectEntry{
		{Key: "next", Value: FunctionValue(nextFn)},
	})), nil
}

func (p *classicJSStatementParser) resolveClassicJSClassMemberName(name string, nameSource string, classEnv *classicJSEnvironment) (string, error) {
	if strings.TrimSpace(nameSource) == "" {
		return name, nil
	}
	if classEnv == nil {
		return "", NewError(ErrorKindRuntime, "class environment is unavailable")
	}
	classEval := p.cloneForClassEvaluation()
	value, err := classEval.evalExpressionWithEnv(nameSource, classEnv.clone())
	if err != nil {
		return "", err
	}
	return ToJSString(value), nil
}

func (p *classicJSStatementParser) instantiateClassicJSClass(name string) (jsValue, error) {
	if p.env == nil {
		return jsValue{}, NewError(ErrorKindRuntime, "class environment is unavailable")
	}
	classDef, ok := p.env.classDefinition(name)
	if !ok || classDef == nil {
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("new expressions only work on declared class identifiers like %q in this bounded classic-JS slice", name))
	}

	classLookupEnv := classDef.env
	if classLookupEnv == nil {
		classLookupEnv = p.env
	}
	classValue, ok := classLookupEnv.lookup(name)
	if !ok || classValue.kind != jsValueScalar || classValue.value.Kind != ValueKindObject {
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("new expressions require a class object binding for %q in this bounded classic-JS slice", name))
	}

	prototypeValue, ok := lookupObjectProperty(classValue.value.Object, "prototype")
	if !ok || prototypeValue.Kind != ValueKindObject {
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
		constructorCall := scalarJSValue(constructorValue)
		constructorCall.receiver = instanceValue
		constructorCall.hasReceiver = true
		if _, err := p.invoke(constructorCall, nil); err != nil {
			return jsValue{}, err
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
	if len(args) != 0 {
		return jsValue{}, NewError(ErrorKindUnsupported, "generator `next()` does not accept arguments in this bounded classic-JS slice")
	}
	wrapResult := func(value Value, done bool) jsValue {
		if state.async {
			return scalarJSValue(PromiseValue(generatorIteratorResultValue(value, done).value))
		}
		return generatorIteratorResultValue(value, done)
	}
	if state.done {
		return wrapResult(UndefinedValue(), true), nil
	}

	for state.index < len(state.statements) {
		if value, yielded, exhausted, err := p.resumeGeneratorDelegate(state); err != nil {
			return jsValue{}, err
		} else if yielded {
			return wrapResult(value, false), nil
		} else if exhausted {
			continue
		}

		if state.activeState != nil {
			value, yielded, err := p.resumeGeneratorState(state)
			if err != nil {
				return jsValue{}, err
			}
			if yielded {
				return wrapResult(value, false), nil
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
				value, err = evalClassicJSExpressionWithEnvAndAllowAwait(yieldSource, p.host, state.env, p.stepLimit, state.async, p.privateClass)
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
			return wrapResult(value, false), nil
		}

		if _, err := evalClassicJSStatementWithEnvAndAllowAwaitAndYield(statement, p.host, state.env, p.stepLimit, state.async, true, false, nil, nil); err != nil {
			if yieldedValue, resumeState, ok := classicJSYieldSignalDetails(err); ok {
				if resumeState != nil {
					state.activeState = resumeState
					return wrapResult(yieldedValue, false), nil
				}
				state.index++
				return wrapResult(yieldedValue, false), nil
			}
			return jsValue{}, err
		}
		state.index++
	}

	state.done = true
	return wrapResult(UndefinedValue(), true), nil
}

func (p *classicJSStatementParser) beginGeneratorDelegate(state *classicJSGeneratorState, value Value) error {
	if state == nil {
		return NewError(ErrorKindRuntime, "generator state is unavailable")
	}

	switch value.Kind {
	case ValueKindArray:
		state.delegateArray = append([]Value(nil), value.Array...)
		state.delegateArrayIndex = 0
		state.delegateIterator = nil
		return nil
	case ValueKindObject:
		nextValue, err := p.resolveMemberAccess(scalarJSValue(value), "next")
		if err != nil {
			return err
		}
		if nextValue.kind != jsValueScalar || nextValue.value.Kind != ValueKindFunction || nextValue.value.Function == nil {
			return NewError(ErrorKindUnsupported, "yield* expects an array or iterator-like object in this bounded classic-JS slice")
		}
		copied := value
		state.delegateIterator = &copied
		state.delegateArray = nil
		state.delegateArrayIndex = 0
		return nil
	default:
		return NewError(ErrorKindUnsupported, "yield* expects an array or iterator-like object in this bounded classic-JS slice")
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
		result, err := p.invoke(nextValue, nil)
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
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, value)

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
	if p.eof() {
		return UndefinedValue(), NewError(ErrorKindParse, "unexpected end of script source")
	}

	p.pos++
	var b strings.Builder
	for !p.eof() {
		ch := p.peekByte()
		switch {
		case ch == '`':
			p.pos++
			return StringValue(b.String()), nil
		case ch == '$' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '{':
			p.pos += 2
			source, err := p.consumeTemplateInterpolationSource()
			if err != nil {
				return UndefinedValue(), err
			}
			value, err := p.evalExpressionWithEnv(source, p.env)
			if err != nil {
				return UndefinedValue(), err
			}
			b.WriteString(templateInterpolationString(value))
		case ch == '\\':
			p.pos++
			if p.eof() {
				return UndefinedValue(), NewError(ErrorKindParse, "unterminated escape sequence in template literal")
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
		default:
			b.WriteByte(ch)
			p.pos++
		}
	}

	return UndefinedValue(), NewError(ErrorKindParse, "unterminated template literal")
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
	sawDigit := false
	lastWasSeparator := false
	for !p.eof() {
		ch := p.peekByte()
		switch {
		case isDigit(ch):
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
	default:
		return math.NaN(), false
	}
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

func templateInterpolationString(value Value) string {
	return ToJSString(value)
}

func isNullishJSValue(value Value) bool {
	return value.Kind == ValueKindUndefined || value.Kind == ValueKindNull
}

func (p *classicJSStatementParser) peekLogicalAssignmentOperator() string {
	if p == nil || p.pos+3 > len(p.source) {
		return ""
	}
	switch {
	case strings.HasPrefix(p.source[p.pos:], "||="):
		return "||="
	case strings.HasPrefix(p.source[p.pos:], "&&="):
		return "&&="
	case strings.HasPrefix(p.source[p.pos:], "??="):
		return "??="
	default:
		return ""
	}
}

func hasClassicJSDeclarationKeyword(source string) bool {
	parser := &classicJSStatementParser{source: strings.TrimSpace(source)}
	if parser.source == "" {
		return false
	}
	parser.skipSpaceAndComments()
	for _, keyword := range []string{"let", "const"} {
		if _, ok := parser.peekKeyword(keyword); ok {
			return true
		}
	}
	return false
}

func isClassicJSReservedDeclarationName(name string) bool {
	switch name {
	case "host", "expr", "true", "false", "undefined", "null", "let", "const", "var", "function", "class", "if", "else", "for", "while", "do", "switch", "case", "default", "try", "catch", "finally", "return", "break", "continue", "throw", "async", "await", "import", "export", "new", "delete", "yield", "super":
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
