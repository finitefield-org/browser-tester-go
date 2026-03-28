package runtime

import (
	"testing"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func TestXMLSerializerSerializesElementNodes(t *testing.T) {
	session := NewSession(SessionConfig{})
	result, err := session.runScriptOnStore(dom.NewStore(), `
		const serializer = new XMLSerializer();
		if (!(serializer instanceof XMLSerializer)) {
		  throw new Error("XMLSerializer instanceof failed");
		}
		const node = document.createElement("div");
		node.setAttribute("data-test", "ok");
		serializer.serializeToString(node)
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, `<div data-test="ok"></div>`; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestXMLSerializerRejectsNonNodeArguments(t *testing.T) {
	session := NewSession(SessionConfig{})
	_, err := session.runScriptOnStore(dom.NewStore(), `new XMLSerializer().serializeToString({})`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want node reference failure")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported {
		t.Fatalf("runScriptOnStore() error = %#v, want unsupported script error", err)
	}
}
