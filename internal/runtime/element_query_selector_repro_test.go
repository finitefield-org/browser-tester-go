package runtime

import "testing"

func TestSessionInlineScriptsCanUseElementQuerySelectorAll(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="root"><button data-preset-id="a"></button><button data-preset-id="b"></button></div><div id="probe"></div><script>const root = document.getElementById("root"); const buttons = root.querySelectorAll("[data-preset-id]"); host:setTextContent("#probe", expr(String(buttons.length) + ":" + String(buttons.item(0) != null) + ":" + String(buttons.item(1) != null)))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "2:true:true" {
		t.Fatalf("TextContent(#probe) = %q, want 2:true:true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after element querySelectorAll bootstrap", got)
	}
}

func TestSessionInlineScriptsCanArrayFromElementQuerySelectorAll(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="root"><button data-line-mode="keep_lines"></button><button data-line-mode="single_line"></button></div><div id="probe"></div><script>const root = document.getElementById("root"); const buttons = Array.from(root.querySelectorAll("[data-line-mode]")); host:setTextContent("#probe", expr(String(buttons.length) + ":" + String(buttons[0].dataset.lineMode) + ":" + String(buttons[1].dataset.lineMode)))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "2:keep_lines:single_line" {
		t.Fatalf("TextContent(#probe) = %q, want 2:keep_lines:single_line", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.from(element.querySelectorAll) bootstrap", got)
	}
}
