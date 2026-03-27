package runtime

import (
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
