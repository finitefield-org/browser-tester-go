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

func TestSessionInlineScriptsCanReadAndWriteDocumentElementLang(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<!doctype html><html lang="en-US"><body><div id="probe"></div><script>const root = document.documentElement; const before = root.lang; root.lang = "fr-CA"; host:setTextContent("#probe", expr(before + "|" + root.lang))</script></body></html>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after documentElement.lang reflection bridge error = %v", err)
	} else if got != "en-US|fr-CA" {
		t.Fatalf("TextContent(#probe) after documentElement.lang reflection bridge = %q, want %q", got, "en-US|fr-CA")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after documentElement.lang reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanReadAndWriteElementDirReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="root" dir="rtl"></div><div id="probe"></div><script>const root = document.getElementById("root"); const before = root.dir; root.dir = "ltr"; host:setTextContent("#probe", expr(before + "|" + root.dir + "|" + root.getAttribute("dir")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after element.dir reflection bridge error = %v", err)
	} else if got != "rtl|ltr|ltr" {
		t.Fatalf("TextContent(#probe) after element.dir reflection bridge = %q, want %q", got, "rtl|ltr|ltr")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after element.dir reflection bridge", got)
	}
}

func TestSessionInlineScriptsTreatMissingDocumentElementLangAsEmptyString(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<!doctype html><html><body><div id="probe"></div><script>const root = document.documentElement; host:setTextContent("#probe", expr("[" + root.lang + "]"))</script></body></html>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after missing documentElement.lang reflection bridge error = %v", err)
	} else if got != "[]" {
		t.Fatalf("TextContent(#probe) after missing documentElement.lang reflection bridge = %q, want %q", got, "[]")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after missing documentElement.lang reflection bridge", got)
	}
}

func TestSessionInlineScriptsTreatMissingElementDirAsEmptyString(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="box"></div><div id="probe"></div><script>const box = document.querySelector("#box"); host:setTextContent("#probe", expr("[" + box.dir + "]"))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after missing element.dir reflection bridge error = %v", err)
	} else if got != "[]" {
		t.Fatalf("TextContent(#probe) after missing element.dir reflection bridge = %q, want %q", got, "[]")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after missing element.dir reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanReadAndWriteFormControlTypeReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><input id="field" type="checkbox"><button id="btn"></button><textarea id="ta"></textarea><div id="probe"></div><script>const field = document.querySelector("#field"); const btn = document.querySelector("#btn"); const ta = document.querySelector("#ta"); const before = [field.type, btn.type, ta.type].join("|"); field.type = "radio"; btn.type = "button"; host:setTextContent("#probe", expr(before + "|" + field.type + "|" + btn.type + "|" + ta.type + "|" + field.getAttribute("type") + "|" + btn.getAttribute("type")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after form-control type reflection bridge error = %v", err)
	} else if got != "checkbox|submit|textarea|radio|button|textarea|radio|button" {
		t.Fatalf("TextContent(#probe) after form-control type reflection bridge = %q, want %q", got, "checkbox|submit|textarea|radio|button|textarea|radio|button")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after form-control type reflection bridge", got)
	}
}

func TestSessionInlineScriptsNormalizeInvalidFormControlTypeReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><input id="field" type="bogus"><button id="btn" type="menu"></button><div id="probe"></div><script>const field = document.querySelector("#field"); const btn = document.querySelector("#btn"); host:setTextContent("#probe", expr([field.type, btn.type, field.getAttribute("type"), btn.getAttribute("type")].join("|")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after invalid form-control type normalization error = %v", err)
	} else if got != "text|submit|bogus|menu" {
		t.Fatalf("TextContent(#probe) after invalid form-control type normalization = %q, want %q", got, "text|submit|bogus|menu")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after invalid form-control type normalization", got)
	}
}

func TestSessionInlineScriptsCanReadSelectTypeReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><select id="single"><option>one</option></select><select id="multi" multiple><option>one</option></select><div id="probe"></div><script>const single = document.querySelector("#single"); const multi = document.querySelector("#multi"); host:setTextContent("#probe", expr([single.type, multi.type].join("|")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after select.type reflection bridge error = %v", err)
	} else if got != "select-one|select-multiple" {
		t.Fatalf("TextContent(#probe) after select.type reflection bridge = %q, want %q", got, "select-one|select-multiple")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after select.type reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanReadCanonicalLinkHrefReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		URL:  "https://finitefield.org/en/tools/agri/orchard-pollination-planner/",
		HTML: `<!doctype html><html><head><link rel="canonical" href="/tools/agri/orchard-pollination-planner/"></head><body><div id="probe"></div><script>const canonicalLink = document.querySelector("link[rel=\"canonical\"]"); const canonicalHref = canonicalLink && canonicalLink.href ? canonicalLink.href : ""; host:setTextContent("#probe", expr(canonicalHref))</script></body></html>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after canonical link.href reflection bridge error = %v", err)
	} else if got != "https://finitefield.org/tools/agri/orchard-pollination-planner/" {
		t.Fatalf("TextContent(#probe) after canonical link.href reflection bridge = %q, want %q", got, "https://finitefield.org/tools/agri/orchard-pollination-planner/")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after canonical link.href reflection bridge", got)
	}
}

func TestSessionBootstrapsFormControlTypeReflectionInHelper(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><input id="field" type="checkbox" checked><div id="out"></div><script>function setValue(elm, value) { if (elm.type === "checkbox") { elm.checked = Boolean(value); return; } elm.value = value === null || value === undefined ? "" : String(value); } setValue(document.getElementById("field"), false); document.getElementById("out").textContent = "done";</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after form-control type bootstrap error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) after form-control type bootstrap = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after form-control type bootstrap", got)
	}
}

func TestSessionInlineScriptsCanReadAndWriteFormControlTabIndexReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><button id="seg" type="button">Segment</button><div id="probe"></div><script>const button = document.getElementById("seg"); const before = button.tabIndex; button.tabIndex = -1; host:setTextContent("#probe", expr([before, button.tabIndex, button.getAttribute("tabindex")].join("|")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after button.tabIndex reflection bridge error = %v", err)
	} else if got != "0|-1|-1" {
		t.Fatalf("TextContent(#probe) after button.tabIndex reflection bridge = %q, want %q", got, "0|-1|-1")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after button.tabIndex reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanReadAndWriteFormControlReadOnlyReflectionSurfaces(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><textarea id="ta"></textarea><input id="field" type="text"><div id="probe"></div><script>const ta = document.getElementById("ta"); const field = document.getElementById("field"); const before = [ta.readOnly, field.readOnly].join("|"); ta.readOnly = true; field.readOnly = true; field.readOnly = false; host:setTextContent("#probe", expr(before + "|" + [ta.readOnly, ta.hasAttribute("readonly"), field.readOnly, field.hasAttribute("readonly")].join("|")))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after readOnly reflection bridge error = %v", err)
	} else if got != "false|false|true|true|false|false" {
		t.Fatalf("TextContent(#probe) after readOnly reflection bridge = %q, want %q", got, "false|false|true|true|false|false")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after readOnly reflection bridge", got)
	}
}

func TestSessionInlineScriptsTreatGenericElementTypeReflectionAsUndefined(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="box"></div><div id="probe"></div><script>const box = document.querySelector("#box"); host:setTextContent("#probe", expr(String(box.type)))</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after generic element.type reflection bridge error = %v", err)
	} else if got != "undefined" {
		t.Fatalf("TextContent(#probe) after generic element.type reflection bridge = %q, want %q", got, "undefined")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after generic element.type reflection bridge", got)
	}
}

func TestSessionInlineScriptsCanReadFormControlTypeReflectionDuringSubmit(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><form id="form"><textarea id="note">Hello</textarea><select id="phase"><option value="carbonization" selected>炭化中</option></select><input id="recorded-at" type="datetime-local" value="2026-04-02T12:34"><input id="limit" type="number" value="42"><input id="agree" type="checkbox" checked><button id="submit" type="submit">Save</button></form><div id="probe"></div><script>function getInputValue(target) { if (target.type === "checkbox") return target.checked; return target.value; } function collect() { const note = document.getElementById("note"); const phase = document.getElementById("phase"); const recordedAt = document.getElementById("recorded-at"); const limit = document.getElementById("limit"); const agree = document.getElementById("agree"); document.getElementById("probe").textContent = [getInputValue(note), getInputValue(phase), getInputValue(recordedAt), getInputValue(limit), getInputValue(agree)].join("|"); } document.getElementById("form").addEventListener("submit", (event) => { event.preventDefault(); collect(); });</script></main>`,
	})

	if err := session.Click("#submit"); err != nil {
		t.Fatalf("Click(#submit) error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) after submit reflection bridge error = %v", err)
	} else if got != "Hello|carbonization|2026-04-02T12:34|42|true" {
		t.Fatalf("TextContent(#probe) after submit reflection bridge = %q, want %q", got, "Hello|carbonization|2026-04-02T12:34|42|true")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after submit reflection bridge", got)
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
