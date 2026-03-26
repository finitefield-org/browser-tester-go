package script

import (
	"strings"
	"testing"
)

type fuzzHost struct{}

func (fuzzHost) Call(method string, args []Value) (Value, error) {
	return UndefinedValue(), nil
}

func FuzzSplitScriptStatements(f *testing.F) {
	seeds := []string{
		"",
		"noop",
		"host:foo()",
		"host.foo()",
		`host:foo("a;b")`,
		`host.foo("a;b")`,
		"host.foo(`a;b`)",
		`host:foo('a;b'); host:bar(1, true, expr(host:baz()))`,
		`host.foo("a;b"); host.bar(1, true, expr(host.baz()))`,
		`host:foo("a;\"b");host:bar(false)`,
		`host.foo("a;\"b"); host.bar(false)`,
		`host.echo(-1_234n)`,
		`host.setTextContent("#out", 1_000n)`,
		`let value = "seed"; host.setTextContent("#out", value)`,
		`let flag = true; if (flag) { host.setTextContent("#out", "then") } else { host.setTextContent("#out", "else") }`,
		`let left = "kept"; left ||= host.echo("boom")`,
		`let middle = null; middle ??= "fresh"`,
		`while (false) { host.foo() }`,
		`do { host.foo() } while (false)`,
		`for (let keepGoing = true; keepGoing; keepGoing &&= false) { host.foo() }`,
		`for (;;){ host.foo() }`,
		`class Example { static { host.foo() } }`,
		`class Example { static value = host.foo(); static { host.bar() } }`,
		`class Example { static foo() {} }`,
		`class Example { static #secret = 1 }`,
		`switch ("b") { case "a": host.foo(); break; case "b": host.bar(); case "c": host.baz(); break; default: host.qux() }`,
		`try { host.foo() } catch (err) { host.bar(err) } finally { host.baz() }`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		statements, err := splitScriptStatements(source)
		if err != nil {
			return
		}
		for i, statement := range statements {
			if statement == "" {
				t.Fatalf("statement %d is empty for source %q", i, source)
			}
			if trimmed := strings.TrimSpace(statement); trimmed != statement {
				t.Fatalf("statement %d is not trimmed: got %q", i, statement)
			}
		}
	})
}

func FuzzParseHostInvocation(f *testing.F) {
	seeds := []string{
		"noop",
		"foo",
		"foo()",
		"foo(1, true, false)",
		`foo('a,b', expr(host:bar()))`,
		`foo('a,b', expr(host.bar()))`,
		` foo ( 1 , 2 ) `,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		method, _, err := parseHostInvocation(source)
		if err != nil {
			return
		}
		if method == "" {
			t.Fatalf("parseHostInvocation(%q) returned an empty method", source)
		}
		if trimmed := strings.TrimSpace(method); trimmed != method {
			t.Fatalf("parseHostInvocation(%q) returned an untrimmed method %q", source, method)
		}
		if strings.ContainsAny(method, " \t\r\n") {
			t.Fatalf("parseHostInvocation(%q) returned a method with whitespace %q", source, method)
		}
	})
}

func FuzzEvalClassicJSStatement(f *testing.F) {
	seeds := []string{
		`host.foo()`,
		`host.setTextContent("#out", "text")`,
		`host.echo(1_000)`,
		"host.echo(`text`)",
		`host.echo(null ?? "fallback")`,
		`host?.echo("text")`,
		`null?.echo(host.echo("boom"))`,
		`host.setTextContent("#out", host.documentCurrentScript())`,
		`host.historyGo(-1)`,
		`expr(host.documentCurrentScript())`,
		`host.foo("a;b")`,
		`while (false) { host.foo() }`,
		`do { host.foo() } while (false)`,
		`for (let keepGoing = true; keepGoing; keepGoing &&= false) { host.foo() }`,
		`for (;;){ host.foo() }`,
		`class Example { static { host.foo() } }`,
		`class Example { static value = host.foo(); static { host.bar() } }`,
		`class Example { static foo() {} }`,
		`class Example { static #secret = 1 }`,
		`switch ("b") { case "a": host.foo(); break; case "b": host.bar(); case "c": host.baz(); break; default: host.qux() }`,
		`try { host.foo() } catch (err) { host.bar(err) } finally { host.baz() }`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		_, _ = evalClassicJSStatement(source, fuzzHost{})
	})
}
