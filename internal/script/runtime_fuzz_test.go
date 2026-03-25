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
		`host:foo('a;b'); host:bar(1, true, expr(host:baz()))`,
		`host.foo("a;b"); host.bar(1, true, expr(host.baz()))`,
		`host:foo("a;\"b");host:bar(false)`,
		`host.foo("a;\"b"); host.bar(false)`,
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
		`host.setTextContent("#out", host.documentCurrentScript())`,
		`host.historyGo(-1)`,
		`expr(host.documentCurrentScript())`,
		`host.foo("a;b")`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		_, _ = evalClassicJSStatement(source, fuzzHost{})
	})
}
