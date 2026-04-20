package runtime

import (
	"strings"
	"testing"
)

func TestSessionDispatchPasteExposesClipboardData(t *testing.T) {
	const rawHTML = `<main><textarea id="target"></textarea><div id="out"></div><script>document.getElementById("target").addEventListener("paste", function (event) { document.getElementById("out").textContent = event.clipboardData.getData("text/plain"); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.WriteClipboard("A\tB\n1\t2"); err != nil {
		t.Fatalf("WriteClipboard() error = %v", err)
	}

	if err := session.Dispatch("#target", "paste"); err != nil {
		t.Fatalf("Dispatch(#target, paste) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "A\tB\n1\t2" {
		t.Fatalf("TextContent(#out) = %q, want clipboard payload", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after paste dispatch", got)
	}
}

func TestSessionDispatchPasteWithoutSeedReturnsExplicitError(t *testing.T) {
	const rawHTML = `<main><textarea id="target"></textarea><div id="out"></div><script>document.getElementById("target").addEventListener("paste", function (event) { document.getElementById("out").textContent = event.clipboardData.getData("text/plain"); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	err := session.Dispatch("#target", "paste")
	if err == nil {
		t.Fatal("Dispatch(#target, paste) error = nil, want explicit clipboard failure")
	}
	if got := err.Error(); !strings.Contains(got, "clipboard text has not been seeded") {
		t.Fatalf("Dispatch(#target, paste) error = %q, want clipboard failure text", got)
	}
}
