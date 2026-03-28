package runtime

import (
	"testing"

	"browsertester/internal/script"
)

func TestDocumentPropertyBridgeDuringAndAfterBootstrap(t *testing.T) {
	session := NewSession(SessionConfig{
		URL: "https://example.test/app?mode=docs#ready",
		HTML: `<html dir="rtl"><head><title>Document Title</title></head><body><main>
<div id="during-ready"></div>
<div id="after-ready"></div>
<div id="title"></div>
<div id="url"></div>
<div id="base-uri"></div>
<div id="document-uri"></div>
<div id="default-view"></div>
<div id="doctype"></div>
<div id="compat-mode"></div>
<div id="content-type"></div>
<div id="design-mode"></div>
<div id="dir"></div>
<div id="active"></div>
<input id="field" type="text" value="seed">
<script>host:setTextContent("#during-ready", expr(document.readyState)); host:setTextContent("#title", expr(document.title)); host:setTextContent("#url", expr(document.URL)); host:setTextContent("#base-uri", expr(document.baseURI)); host:setTextContent("#document-uri", expr(document.documentURI)); host:setTextContent("#default-view", expr(document.defaultView.location.href)); host:setTextContent("#doctype", expr(document.doctype === null ? "null" : "non-null")); host:setTextContent("#compat-mode", expr(document.compatMode)); host:setTextContent("#content-type", expr(document.contentType)); host:setTextContent("#design-mode", expr(document.designMode)); host:setTextContent("#dir", expr(document.dir))</script>
</main></body></html>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := session.DOMReady(); !got {
		t.Fatalf("DOMReady() = %v, want true after bootstrap", got)
	}
	textContent := func(selector string) string {
		got, err := session.TextContent(selector)
		if err != nil {
			t.Fatalf("TextContent(%s) error = %v", selector, err)
		}
		return got
	}

	if got, want := textContent("#during-ready"), "loading"; got != want {
		t.Fatalf("TextContent(#during-ready) = %q, want %q", got, want)
	}
	if got, want := textContent("#title"), "Document Title"; got != want {
		t.Fatalf("TextContent(#title) = %q, want %q", got, want)
	}
	if got, want := textContent("#url"), "https://example.test/app?mode=docs#ready"; got != want {
		t.Fatalf("TextContent(#url) = %q, want %q", got, want)
	}
	if got, want := textContent("#base-uri"), "https://example.test/app?mode=docs#ready"; got != want {
		t.Fatalf("TextContent(#base-uri) = %q, want %q", got, want)
	}
	if got, want := textContent("#document-uri"), "https://example.test/app?mode=docs#ready"; got != want {
		t.Fatalf("TextContent(#document-uri) = %q, want %q", got, want)
	}
	if got, want := textContent("#default-view"), "https://example.test/app?mode=docs#ready"; got != want {
		t.Fatalf("TextContent(#default-view) = %q, want %q", got, want)
	}
	if got, want := textContent("#doctype"), "null"; got != want {
		t.Fatalf("TextContent(#doctype) = %q, want %q", got, want)
	}
	if got, want := textContent("#compat-mode"), "CSS1Compat"; got != want {
		t.Fatalf("TextContent(#compat-mode) = %q, want %q", got, want)
	}
	if got, want := textContent("#content-type"), "text/html"; got != want {
		t.Fatalf("TextContent(#content-type) = %q, want %q", got, want)
	}
	if got, want := textContent("#design-mode"), "off"; got != want {
		t.Fatalf("TextContent(#design-mode) = %q, want %q", got, want)
	}
	if got, want := textContent("#dir"), "rtl"; got != want {
		t.Fatalf("TextContent(#dir) = %q, want %q", got, want)
	}

	if err := session.Focus("#field"); err != nil {
		t.Fatalf("Focus(#field) error = %v", err)
	}
	if _, err := session.runScriptOnStore(store, `host:setTextContent("#after-ready", expr(document.readyState)); host:setTextContent("#active", expr(document.activeElement.id))`); err != nil {
		t.Fatalf("runScriptOnStore(activeElement) error = %v", err)
	}
	if got, want := textContent("#after-ready"), "complete"; got != want {
		t.Fatalf("TextContent(#after-ready) = %q, want %q", got, want)
	}
	if got, want := textContent("#active"), "field"; got != want {
		t.Fatalf("TextContent(#active) = %q, want %q", got, want)
	}
}

func TestDocumentPropertyBridgeReportsUnsupportedSurfacesExplicitly(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main></main>`})
	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if _, err := resolveBrowserGlobalReference(session, store, "document.unknown"); err == nil {
		t.Fatalf("resolveBrowserGlobalReference(document.unknown) error = nil, want unsupported error")
	} else if got, ok := err.(script.Error); !ok || got.Kind != script.ErrorKindUnsupported {
		t.Fatalf("resolveBrowserGlobalReference(document.unknown) error = %#v, want unsupported script error", err)
	}
}

func TestDocumentAllIsExplicitlyUnsupported(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main></main>`})
	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if _, err := session.runScriptOnStore(store, `document.all`); err == nil {
		t.Fatalf("runScriptOnStore(document.all) error = nil, want unsupported error")
	} else if got, ok := err.(script.Error); !ok || got.Kind != script.ErrorKindUnsupported {
		t.Fatalf("runScriptOnStore(document.all) error = %#v, want unsupported script error", err)
	}
}
