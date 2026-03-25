package script

import (
	"fmt"
	"math"
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

func evalClassicJSStatement(source string, host HostBindings) (Value, error) {
	parser := &classicJSStatementParser{
		source: strings.TrimSpace(source),
		host:   host,
	}
	if parser.source == "" {
		return UndefinedValue(), nil
	}

	value, err := parser.parseExpression()
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
			"unsupported script source; this bounded classic-JS slice only supports expression statements, member calls on `host`, and the `expr(...)` compatibility helper",
		)
	}

	return value, nil
}

type classicJSStatementParser struct {
	source string
	host   HostBindings
	pos    int
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
	value, err := p.parseUnary()
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
		if value.kind != jsValueScalar || value.value.Kind != ValueKindNumber {
			return jsValue{}, NewError(ErrorKindUnsupported, "unary `-` is only supported for numeric literals in this slice")
		}
		return scalarJSValue(NumberValue(-value.value.Number)), nil
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
	case '`':
		return jsValue{}, NewError(ErrorKindUnsupported, "template literals are not supported in this bounded classic-JS slice")
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

func (p *classicJSStatementParser) parseNumberLiteral() (Value, error) {
	start := p.pos
	if p.consumeByte('.') {
		if p.eof() || !isDigit(p.peekByte()) {
			return UndefinedValue(), NewError(ErrorKindParse, "invalid numeric literal")
		}
		for !p.eof() && isDigit(p.peekByte()) {
			p.pos++
		}
	} else {
		for !p.eof() && isDigit(p.peekByte()) {
			p.pos++
		}
		if p.consumeByte('.') {
			for !p.eof() && isDigit(p.peekByte()) {
				p.pos++
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
		for !p.eof() && isDigit(p.peekByte()) {
			p.pos++
		}
	}

	if !p.eof() && p.peekByte() == 'n' {
		return UndefinedValue(), NewError(ErrorKindUnsupported, "BigInt literals are not supported in this bounded classic-JS slice")
	}

	raw := p.source[start:p.pos]
	number, err := strconv.ParseFloat(raw, 64)
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
	case ValueKindString:
		return value.String != ""
	default:
		return true
	}
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
