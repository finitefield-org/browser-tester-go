package runtime

import "testing"

func TestSessionInlineScriptsCanReadStickyBoundingClientRectTopFromStylesheet(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main>
  <style>
    body { margin: 0; }
    #sticky {
      position: sticky;
      top: 5.75rem;
      height: 40px;
    }
  </style>
  <div id="sticky">sticky</div>
  <div id="out"></div>
  <script>
    const sticky = document.getElementById("sticky");
    document.getElementById("out").textContent = String(Math.round(sticky.getBoundingClientRect().top));
  </script>
</main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "92" {
		t.Fatalf("TextContent(#out) = %q, want 92", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after bounding client rect bootstrap", got)
	}
}
