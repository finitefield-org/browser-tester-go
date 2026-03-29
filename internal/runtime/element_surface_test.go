package runtime

import (
	"strings"
	"testing"

	"browsertester/internal/script"
)

func TestSessionInlineScriptsCanReadElementReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="box" class="alpha beta" style="color: green; background: transparent" data-x="1">Hello <strong>world</strong></div><div id="probe"></div><script>const box = document.querySelector("#box"); const firstAttr = box.attributes.item(0); const styleAttr = box.attributes.namedItem("style"); const dataAttr = box.getAttributeNode("data-x"); host:setTextContent("#probe", expr(box.className + "|" + box.innerText + "|" + box.outerText + "|" + box.style.cssText + "|" + box.style.length + "|" + box.style.item(0) + "|" + box.style.getPropertyValue("background") + "|" + box.attributes.length + "|" + firstAttr.name + "=" + firstAttr.value + "|" + styleAttr.value + "|" + dataAttr.name + "=" + dataAttr.value + "|" + String(box.getAttributeNode("missing") === null) + "|" + box.attributes.namedItem("data-x").value))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after element reflection bridge error = %v", err)
	} else if want := "alpha beta|Hello world|Hello world|color: green; background: transparent|2|color|transparent|4|id=box|color: green; background: transparent|data-x=1|true|1"; got != want {
		t.Fatalf("TextContent(#probe) after element reflection bridge = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanMutateElementStyleReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="box" style="color: green; background: transparent; --accent: orange !important"></div><div id="probe"></div><script>const box = document.querySelector("#box"); const removed = box.style.removeProperty("color"); box.style.setProperty("background", "blue"); box.style.setProperty("--accent", "purple", "important"); host:setTextContent("#probe", expr([removed, box.style.getPropertyValue("background"), box.style.getPropertyPriority("--accent"), box.style.cssText].join("|")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after style mutation bridge error = %v", err)
	} else if want := "green|blue|important|background: blue; --accent: purple !important"; got != want {
		t.Fatalf("TextContent(#probe) after style mutation bridge = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRemoveMissingStylePropertyWithoutMutation(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="box" style="color: green; background: transparent"></div><div id="probe"></div><script>const box = document.querySelector("#box"); const before = box.style.cssText; const removed = box.style.removeProperty("border"); host:setTextContent("#probe", expr([removed, before, box.style.cssText].join("|")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after missing style removal bridge error = %v", err)
	} else if want := "|color: green; background: transparent|color: green; background: transparent"; got != want {
		t.Fatalf("TextContent(#probe) after missing style removal bridge = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRejectUnsupportedStylePriorityExplicitly(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main><div id="box" style="color: green"></div><script>document.querySelector("#box").style.setProperty("background", "blue", "urgent")</script></main>`})

	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported style priority error")
	} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported || !strings.Contains(scriptErr.Message, `element.style.setProperty priority must be empty or "important"`) {
		t.Fatalf("ensureDOM() error = %#v, want unsupported style priority script error", err)
	}
}
