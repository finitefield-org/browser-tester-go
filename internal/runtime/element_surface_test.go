package runtime

import (
	"strconv"
	"strings"
	"testing"

	"browsertester/internal/script"
)

func TestSessionInlineScriptsCanReadElementReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="box" class="alpha beta" style="color: green; background: transparent" data-x="1">Hello <strong>world</strong></div><div id="probe"></div><script>const box = document.querySelector("#box"); const firstAttr = box.attributes.item(0); const styleAttr = box.attributes.namedItem("style"); host:setTextContent("#probe", expr(box.className + "|" + box.innerText + "|" + box.outerText + "|" + box.style.cssText + "|" + box.style.length + "|" + box.style.item(0) + "|" + box.style.getPropertyValue("background") + "|" + box.attributes.length + "|" + firstAttr.name + "=" + firstAttr.value + "|" + styleAttr.value + "|" + box.attributes.namedItem("data-x").value))</script></main>`,
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after element reflection bridge error = %v", err)
	} else if want := "alpha beta|Hello world|Hello world|color: green; background: transparent|2|color|transparent|4|id=box|color: green; background: transparent|1"; got != want {
		t.Fatalf("TextContent(#probe) after element reflection bridge = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsRejectDeferredElementReflectionMutationSurfacesExplicitly(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main><div id="box" style="color: green"></div></main>`})
	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	boxID, ok, err := store.QuerySelector("#box")
	if err != nil {
		t.Fatalf("QuerySelector(#box) error = %v", err)
	}
	if !ok {
		t.Fatalf("QuerySelector(#box) = no match, want box node")
	}

	path := "element:" + strconv.FormatInt(int64(boxID), 10) + ".style.setProperty"
	if _, err := resolveBrowserGlobalReference(session, store, path); err == nil {
		t.Fatalf("resolveBrowserGlobalReference(%s) error = nil, want unsupported error", path)
	} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported || !strings.Contains(scriptErr.Message, path) {
		t.Fatalf("resolveBrowserGlobalReference(%s) error = %#v, want unsupported script error", path, err)
	}
}
