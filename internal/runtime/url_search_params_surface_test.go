package runtime

import (
	"strings"
	"testing"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func TestRunScriptSupportsURLSearchParamsKeysIterator(t *testing.T) {
	session := NewSession(SessionConfig{})

	result, err := session.runScriptOnStore(dom.NewStore(), `
		let u = new URL("https://example.com/path?b=2&a=1&a=3");
		[...u.searchParams.keys()].join(",");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "b,a,a"; got != want {
		t.Fatalf("URL.searchParams.keys() = %q, want %q", got, want)
	}
}

func TestRunScriptURLSearchParamsKeysEmpty(t *testing.T) {
	session := NewSession(SessionConfig{})

	result, err := session.runScriptOnStore(dom.NewStore(), `
		let u = new URL("https://example.com/path");
		[...u.searchParams.keys()].join(",");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, ""; got != want {
		t.Fatalf("URL.searchParams.keys() empty = %q, want %q", got, want)
	}
}

func TestRunScriptURLSearchParamsKeysRejectsArguments(t *testing.T) {
	session := NewSession(SessionConfig{})

	_, err := session.runScriptOnStore(dom.NewStore(), `
		let u = new URL("https://example.com/path?a=1");
		u.searchParams.keys(1);
	`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want argument error")
	}
}

func TestRunScriptURLAndSearchParamsStayInSync(t *testing.T) {
	session := NewSession(SessionConfig{})

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const url = new URL("https://example.com/path?alpha=1");
		url.search = "";
		url.searchParams.set("mode", "raw");
		[url.href, url.search, url.searchParams.get("mode"), [...url.searchParams.keys()].join(",")].join("|");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "https://example.com/path?mode=raw|?mode=raw|raw|mode"; got != want {
		t.Fatalf("URL/searchParams sync = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsURLSearchParamsMemberParity(t *testing.T) {
	session := NewSession(SessionConfig{})

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const params = new URLSearchParams("b=3&a=1&a=2");
		const entries = [...params.entries()].map((pair) => pair.join("=")).join(",");
		const values = [...params.values()].join(",");
		const all = params.getAll("a").join(",");
		let seen = "";
		params.forEach((value, key, paramsObject) => { seen = seen + key + "=" + value + ":" + paramsObject.toString() + ","; });
		params.sort();
		[entries, values, all, seen, params.toString()].join("|");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "b=3,a=1,a=2|3,1,2|1,2|b=3:b=3&a=1&a=2,a=1:b=3&a=1&a=2,a=2:b=3&a=1&a=2,|a=1&a=2&b=3"; got != want {
		t.Fatalf("URLSearchParams member parity = %q, want %q", got, want)
	}
}

func TestRunScriptURLSearchParamsForEachRejectsArguments(t *testing.T) {
	session := NewSession(SessionConfig{})

	_, err := session.runScriptOnStore(dom.NewStore(), `
		new URLSearchParams("a=1").forEach(1, 2, 3);
	`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want forEach argument error")
	}
	if !strings.Contains(err.Error(), "URLSearchParams.forEach accepts at most 2 arguments") {
		t.Fatalf("runScriptOnStore() error = %v, want forEach arity message", err)
	}
}

func TestRunScriptURLSearchParamsConstructorRejectsTooManyArguments(t *testing.T) {
	session := NewSession(SessionConfig{})

	_, err := session.runScriptOnStore(dom.NewStore(), `
		new URLSearchParams("a=1", "b=2");
	`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want constructor arity error")
	}
	if !strings.Contains(err.Error(), "URLSearchParams expects at most 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want constructor arity message", err)
	}
}

func TestRunScriptURLSearchParamsSortRejectsArguments(t *testing.T) {
	session := NewSession(SessionConfig{})

	_, err := session.runScriptOnStore(dom.NewStore(), `
		new URLSearchParams("a=1").sort(1);
	`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want sort arity error")
	}
	if !strings.Contains(err.Error(), "URLSearchParams.sort expects no arguments") {
		t.Fatalf("runScriptOnStore() error = %v, want sort arity message", err)
	}
}
