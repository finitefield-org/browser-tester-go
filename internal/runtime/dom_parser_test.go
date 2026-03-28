package runtime

import (
	"testing"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func TestDOMParserParsesSVGDocuments(t *testing.T) {
	session := NewSession(SessionConfig{})
	result, err := session.runScriptOnStore(dom.NewStore(), `
		const parser = new DOMParser();
		if (!(parser instanceof DOMParser)) {
		  throw new Error("DOMParser instanceof failed");
		}
		const doc = parser.parseFromString(
		  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4" /></svg>',
		  "image/svg+xml"
		);
		[
		  String(doc && doc.documentElement ? doc.documentElement.nodeName : "missing"),
		  String(doc && doc.documentElement ? doc.documentElement.namespaceURI : "missing"),
		  String(doc ? doc.contentType : "missing")
		].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "svg|http://www.w3.org/2000/svg|image/svg+xml"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestDOMParserRejectsUnsupportedMimeType(t *testing.T) {
	session := NewSession(SessionConfig{})
	_, err := session.runScriptOnStore(dom.NewStore(), `new DOMParser().parseFromString("<svg></svg>", "text/html")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want unsupported mime failure")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported {
		t.Fatalf("runScriptOnStore() error = %#v, want unsupported script error", err)
	}
}

func TestDOMParserReturnsParserErrorDocumentForMalformedSVGDocuments(t *testing.T) {
	session := NewSession(SessionConfig{})
	result, err := session.runScriptOnStore(dom.NewStore(), `
		const doc = new DOMParser().parseFromString("<svg><circle></svg>", "image/svg+xml");
		[
		  String(doc && doc.documentElement ? doc.documentElement.nodeName : "missing"),
		  String(doc && doc.documentElement ? doc.documentElement.namespaceURI : "missing"),
		  String(doc ? doc.getElementsByTagName("parsererror").length : "missing")
		].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v, want parsererror document", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "parsererror|http://www.mozilla.org/newlayout/xml/parsererror.xml|1"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}
