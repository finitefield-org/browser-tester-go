package runtime

import "testing"

func TestSessionBootstrapsPerformanceNow(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>host:setTextContent("#out", expr(String(performance.now())))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.SetNowMs(42)

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "42" {
		t.Fatalf("TextContent(#out) = %q, want 42", got)
	}

	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after performance.now bootstrap", got)
	}
}
