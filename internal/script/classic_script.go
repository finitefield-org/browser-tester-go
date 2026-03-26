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
)

type jsValue struct {
	kind   jsValueKind
	value  Value
	method string
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

type noopHostBindings struct{}

func (noopHostBindings) Call(method string, args []Value) (Value, error) {
	return UndefinedValue(), nil
}

func evalClassicJSStatement(source string, host HostBindings) (Value, error) {
	return evalClassicJSStatementWithEnv(source, host, nil, DefaultRuntimeConfig().StepLimit)
}

func evalClassicJSStatementWithEnv(source string, host HostBindings, env *classicJSEnvironment, stepLimit int) (Value, error) {
	if stepLimit <= 0 {
		stepLimit = DefaultRuntimeConfig().StepLimit
	}
	parser := &classicJSStatementParser{
		source:    strings.TrimSpace(source),
		host:      host,
		env:       env,
		stepLimit: stepLimit,
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
			"unsupported script source; this bounded classic-JS slice only supports expression statements, `let`/`const` declarations, block-bodied `if` / `while` / `do...while` / `for` / `switch` / `try` statements, class declarations with static blocks and public `static` fields, member calls on `host`, and the `expr(...)` compatibility helper",
		)
	}

	return value, nil
}

func evalClassicJSExpressionWithEnv(source string, host HostBindings, env *classicJSEnvironment, stepLimit int) (Value, error) {
	if stepLimit <= 0 {
		stepLimit = DefaultRuntimeConfig().StepLimit
	}
	parser := &classicJSStatementParser{
		source:    strings.TrimSpace(source),
		host:      host,
		env:       env,
		stepLimit: stepLimit,
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
	allowUnknownIdentifiers bool
	stepLimit               int
	pos                     int
}

type classicJSSwitchClause struct {
	kind  string
	label string
	body  string
}

type classicJSClassMember struct {
	kind        string
	fieldName   string
	fieldInit   string
	staticBlock string
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
		allowUnknownIdentifiers: true,
		stepLimit:               p.stepLimit,
		pos:                     p.pos,
	}
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
	if p.eof() || !isIdentStart(p.peekByte()) {
		return p.parseNullishCoalescing()
	}

	name, err := p.parseIdentifier()
	if err != nil {
		return jsValue{}, err
	}
	p.skipSpaceAndComments()
	op := p.peekLogicalAssignmentOperator()
	if op == "" {
		p.pos = start
		return p.parseNullishCoalescing()
	}

	p.pos += len(op)
	if p.env == nil {
		if p.allowUnknownIdentifiers {
			skip := p.cloneForSkipping(noopHostBindings{})
			skip.pos = p.pos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(UndefinedValue()), nil
		}
		return jsValue{}, NewError(ErrorKindUnsupported, "logical assignment only works on declared local bindings in this bounded classic-JS slice")
	}
	current, ok := p.env.lookup(name)
	if !ok {
		if p.allowUnknownIdentifiers {
			skip := p.cloneForSkipping(noopHostBindings{})
			skip.pos = p.pos
			if _, err := skip.parseLogicalAssignment(); err != nil {
				return jsValue{}, err
			}
			p.pos = skip.pos
			return scalarJSValue(UndefinedValue()), nil
		}
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("assignment target %q is not a declared local binding in this bounded classic-JS slice", name))
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
		skip := p.cloneForSkipping(noopHostBindings{})
		skip.pos = p.pos
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

func (p *classicJSStatementParser) parseStatement() (Value, error) {
	p.skipSpaceAndComments()
	if keyword, ok := p.peekKeyword("let"); ok {
		p.pos += len(keyword)
		return p.parseVariableDeclaration("let")
	}
	if keyword, ok := p.peekKeyword("const"); ok {
		p.pos += len(keyword)
		return p.parseVariableDeclaration("const")
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
	return p.parseExpression()
}

func (p *classicJSStatementParser) parseVariableDeclaration(kind string) (Value, error) {
	if p.env == nil {
		p.env = newClassicJSEnvironment()
	}

	for {
		p.skipSpaceAndComments()
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

		p.skipSpaceAndComments()
		if !p.consumeByte(',') {
			break
		}
	}

	return UndefinedValue(), nil
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
		return evalClassicJSProgram(elseSource, p.host, p.env.clone(), p.stepLimit)
	}

	return evalClassicJSProgram(consequent, p.host, p.env.clone(), p.stepLimit)
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

	for iter := 0; iter < p.stepLimit; iter++ {
		condition, err := evalClassicJSExpressionWithEnv(conditionSource, p.host, p.env.clone(), p.stepLimit)
		if err != nil {
			return UndefinedValue(), err
		}
		if !jsTruthy(condition) {
			return UndefinedValue(), nil
		}

		if _, err := evalClassicJSProgram(bodySource, p.host, p.env.clone(), p.stepLimit); err != nil {
			return UndefinedValue(), err
		}
	}

	return UndefinedValue(), NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
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

	for iter := 0; iter < p.stepLimit; iter++ {
		if _, err := evalClassicJSProgram(bodySource, p.host, p.env.clone(), p.stepLimit); err != nil {
			return UndefinedValue(), err
		}

		condition, err := evalClassicJSExpressionWithEnv(conditionSource, p.host, p.env.clone(), p.stepLimit)
		if err != nil {
			return UndefinedValue(), err
		}
		if !jsTruthy(condition) {
			return UndefinedValue(), nil
		}
	}

	return UndefinedValue(), NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
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

	loopEnv := p.env.clone()
	if strings.TrimSpace(initSource) != "" {
		if hasClassicJSDeclarationKeyword(initSource) {
			if _, err := evalClassicJSStatementWithEnv(initSource, p.host, loopEnv, p.stepLimit); err != nil {
				return UndefinedValue(), err
			}
		} else {
			if _, err := evalClassicJSExpressionWithEnv(initSource, p.host, loopEnv, p.stepLimit); err != nil {
				return UndefinedValue(), err
			}
		}
	}

	for iter := 0; iter < p.stepLimit; iter++ {
		if strings.TrimSpace(conditionSource) != "" {
			condition, err := evalClassicJSExpressionWithEnv(conditionSource, p.host, loopEnv.clone(), p.stepLimit)
			if err != nil {
				return UndefinedValue(), err
			}
			if !jsTruthy(condition) {
				return UndefinedValue(), nil
			}
		}

		if _, err := evalClassicJSProgram(bodySource, p.host, loopEnv.clone(), p.stepLimit); err != nil {
			return UndefinedValue(), err
		}

		if strings.TrimSpace(updateSource) != "" {
			if _, err := evalClassicJSExpressionWithEnv(updateSource, p.host, loopEnv.clone(), p.stepLimit); err != nil {
				return UndefinedValue(), err
			}
		}
	}

	return UndefinedValue(), NewError(ErrorKindRuntime, "classic-JS loop step limit exceeded")
}

func (p *classicJSStatementParser) parseClassStatement() (Value, error) {
	p.skipSpaceAndComments()
	name, err := p.parseIdentifier()
	if err != nil {
		return UndefinedValue(), NewError(ErrorKindParse, "class declarations require an identifier in this bounded classic-JS slice")
	}
	if isClassicJSReservedDeclarationName(name) {
		return UndefinedValue(), NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class name %q in this bounded classic-JS slice", name))
	}

	p.skipSpaceAndComments()
	if _, ok := p.peekKeyword("extends"); ok {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "class inheritance is not supported in this bounded classic-JS slice")
	}

	bodySource, err := p.consumeBlockSource()
	if err != nil {
		return UndefinedValue(), err
	}

	classEnv := p.env.clone()
	if err := classEnv.declare(name, scalarJSValue(StringValue("[class "+name+"]")), false); err != nil {
		return UndefinedValue(), err
	}

	members, err := splitClassicJSClassMembers(bodySource)
	if err != nil {
		return UndefinedValue(), err
	}
	for _, member := range members {
		switch member.kind {
		case "static-block":
			if _, err := evalClassicJSProgram(member.staticBlock, p.host, classEnv.clone(), p.stepLimit); err != nil {
				return UndefinedValue(), err
			}
		case "static-field":
			if strings.TrimSpace(member.fieldInit) == "" {
				continue
			}
			if _, err := evalClassicJSExpressionWithEnv(member.fieldInit, p.host, classEnv.clone(), p.stepLimit); err != nil {
				return UndefinedValue(), err
			}
		default:
			return UndefinedValue(), NewError(ErrorKindParse, fmt.Sprintf("unsupported class member kind %q in this bounded classic-JS slice", member.kind))
		}
	}

	p.env = classEnv

	return UndefinedValue(), nil
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
	discriminant, err := evalClassicJSExpressionWithEnv(discriminantSource, p.host, switchEnv, p.stepLimit)
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
			label, err := evalClassicJSExpressionWithEnv(clause.label, p.host, switchEnv, p.stepLimit)
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

	for i := startIndex; i < len(clauses); i++ {
		stopped, err := p.evalSwitchClauseBody(clauses[i].body, switchEnv)
		if err != nil {
			return UndefinedValue(), err
		}
		if stopped {
			return UndefinedValue(), nil
		}
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

	result := UndefinedValue()
	runErr := error(nil)
	if trimmed := strings.TrimSpace(trySource); trimmed != "" {
		result, runErr = evalClassicJSProgram(trySource, p.host, p.env.clone(), p.stepLimit)
	}

	if runErr != nil && hasCatch {
		catchEnv := p.env.clone()
		if catchBound {
			if err := catchEnv.declare(catchName, scalarJSValue(StringValue(runErr.Error())), true); err != nil {
				return UndefinedValue(), err
			}
		}
		result, runErr = evalClassicJSProgram(catchSource, p.host, catchEnv, p.stepLimit)
	}

	if hasFinally {
		if _, finallyErr := evalClassicJSProgram(finallySource, p.host, p.env.clone(), p.stepLimit); finallyErr != nil {
			return UndefinedValue(), finallyErr
		}
	}

	if runErr != nil {
		return UndefinedValue(), runErr
	}
	return result, nil
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
		if _, err := evalClassicJSStatementWithEnv(statement, p.host, env, p.stepLimit); err != nil {
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
			case '#':
				return nil, NewError(ErrorKindUnsupported, "private class fields are not supported in this bounded classic-JS slice")
			case '[':
				return nil, NewError(ErrorKindUnsupported, "computed class fields are not supported in this bounded classic-JS slice")
			case '*':
				return nil, NewError(ErrorKindUnsupported, "class methods are not supported in this bounded classic-JS slice")
			}

			fieldName, err := scanner.parseIdentifier()
			if err != nil {
				return nil, NewError(ErrorKindUnsupported, "class fields require a plain identifier name in this bounded classic-JS slice")
			}

			scanner.skipSpaceAndComments()
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
				members = append(members, classicJSClassMember{kind: "static-field", fieldName: fieldName, fieldInit: fieldInit})
				if scanner.consumeByte(';') {
					continue
				}
				if scanner.eof() {
					continue
				}
				return nil, NewError(ErrorKindParse, "expected `;` after static class field initializer")
			}

			if scanner.consumeByte(';') {
				members = append(members, classicJSClassMember{kind: "static-field", fieldName: fieldName})
				continue
			}
			if scanner.eof() {
				members = append(members, classicJSClassMember{kind: "static-field", fieldName: fieldName})
				continue
			}

			return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class member at %q in this bounded classic-JS slice", scanner.remainingPreview()))
		}

		return nil, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported class body element at %q in this bounded classic-JS slice", scanner.remainingPreview()))
	}

	return members, nil
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
	left, err := p.parseUnary()
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
			skip := p.cloneForSkipping(noopHostBindings{})
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

	return p.parsePostfix()
}

func (p *classicJSStatementParser) parsePostfix() (jsValue, error) {
	value, err := p.parsePrimary()
	if err != nil {
		return jsValue{}, err
	}

	for {
		p.skipSpaceAndComments()
		switch {
		case p.peekByte() == '?' && p.pos+1 < len(p.source) && p.source[p.pos+1] == '.':
			p.pos += 2
			method, err := p.parseIdentifier()
			if err != nil {
				return jsValue{}, err
			}
			if value.kind == jsValueScalar && isNullishJSValue(value.value) {
				if err := p.skipOptionalCallArguments(); err != nil {
					return jsValue{}, err
				}
				return scalarJSValue(UndefinedValue()), nil
			}
			if value.kind != jsValueHostObject {
				return jsValue{}, NewError(
					ErrorKindUnsupported,
					"optional chaining in this bounded classic-JS slice only works on nullish scalars or `host?.method(...)`",
				)
			}
			value = hostMethodJSValue(method)

		case p.consumeByte('.'):
			method, err := p.parseIdentifier()
			if err != nil {
				return jsValue{}, err
			}
			if value.kind != jsValueHostObject {
				return jsValue{}, NewError(
					ErrorKindUnsupported,
					"unsupported member access in this bounded classic-JS slice; only `host.method(...)` is available",
				)
			}
			value = hostMethodJSValue(method)

		case p.consumeByte(':'):
			method, err := p.parseIdentifier()
			if err != nil {
				return jsValue{}, err
			}
			if value.kind != jsValueHostObject {
				return jsValue{}, NewError(
					ErrorKindUnsupported,
					"unsupported legacy host syntax outside a `host:method(...)` call",
				)
			}
			value = hostMethodJSValue(method)

		case p.consumeByte('('):
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

func (p *classicJSStatementParser) skipOptionalCallArguments() error {
	p.skipSpaceAndComments()
	if !p.consumeByte('(') {
		return nil
	}

	skip := p.cloneForSkipping(noopHostBindings{})
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
	case '{':
		return jsValue{}, NewError(ErrorKindUnsupported, "object literals are not supported in this bounded classic-JS slice")
	case '[':
		return jsValue{}, NewError(ErrorKindUnsupported, "array literals are not supported in this bounded classic-JS slice")
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
	case "let", "const", "var", "function", "class", "if", "else", "for", "while", "do", "switch", "case", "default", "try", "catch", "finally", "return", "break", "continue", "throw", "async", "await", "import", "export", "new", "delete", "yield":
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported script syntax %q in this bounded classic-JS slice", ident))
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, fmt.Sprintf("unsupported identifier %q in this bounded classic-JS slice", ident))
	}
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
	default:
		return jsValue{}, NewError(ErrorKindUnsupported, "unsupported call expression in this bounded classic-JS slice")
	}
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
		p.pos++
		if ch == '`' {
			return StringValue(b.String()), nil
		}
		if ch == '$' && !p.eof() && p.peekByte() == '{' {
			return UndefinedValue(), NewError(ErrorKindUnsupported, "template literal interpolation is not supported in this bounded classic-JS slice")
		}
		if ch == '\\' {
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
			continue
		}
		b.WriteByte(ch)
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
	case "host", "expr", "true", "false", "undefined", "null", "let", "const", "var", "function", "class", "if", "else", "for", "while", "do", "switch", "case", "default", "try", "catch", "finally", "return", "break", "continue", "throw", "async", "await", "import", "export", "new", "delete", "yield":
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
