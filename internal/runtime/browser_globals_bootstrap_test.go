package runtime

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"browsertester/internal/script"
)

func TestSessionBootstrapsRawHtmlWithBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><div id="agri-unit-converter-root">root</div><div id="result"></div><div id="formatted"></div><div id="href"></div><script>const root = document.getElementById("agri-unit-converter-root"); const current = new URL(window.location.href); if (!(current instanceof URL)) { throw new Error("URL instanceof failed"); } const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); window.location.search.length; sessionStorage.setItem("mode", navigator.onLine && "search"); window.history.replaceState({}, "", "?mode=raw#ready"); localStorage.setItem("format", formatted); matchMedia("(prefers-reduced-motion: reduce)"); clipboard.writeText(root.textContent); setTimeout("noop", 5); queueMicrotask("noop"); host:setTextContent("#result", expr(root.textContent)); host:setTextContent("#formatted", expr(formatted)); host:setTextContent("#href", expr(current.href))</script></main>`

	session := NewSession(SessionConfig{
		URL:        "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial",
		HTML:       rawHTML,
		MatchMedia: map[string]bool{"(prefers-reduced-motion: reduce)": true},
	})

	store, err := session.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if store == nil {
		t.Fatalf("ensureDOM() store = nil, want DOM store")
	}

	if got, err := session.TextContent("#result"); err != nil {
		t.Fatalf("TextContent(#result) error = %v", err)
	} else if got != "root" {
		t.Fatalf("TextContent(#result) = %q, want root", got)
	}
	if got := session.DOMReady(); !got {
		t.Fatalf("DOMReady() = %v, want true after raw HTML bootstrap", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after raw HTML bootstrap", got)
	}
	if got, want := session.URL(), "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=raw#ready"; got != want {
		t.Fatalf("URL() after browser-global bootstrap = %q, want %q", got, want)
	}
	if got, err := session.TextContent("#formatted"); err != nil {
		t.Fatalf("TextContent(#formatted) error = %v", err)
	} else if got != "1.23" {
		t.Fatalf("TextContent(#formatted) = %q, want 1.23", got)
	}
	if got, err := session.TextContent("#href"); err != nil {
		t.Fatalf("TextContent(#href) error = %v", err)
	} else if got != "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial" {
		t.Fatalf("TextContent(#href) = %q, want %q", got, "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial")
	}
	if got, want := session.DumpDOM(), `<main><div id="agri-unit-converter-root">root</div><div id="result">root</div><div id="formatted">1.23</div><div id="href">https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial</div><script>const root = document.getElementById("agri-unit-converter-root"); const current = new URL(window.location.href); if (!(current instanceof URL)) { throw new Error("URL instanceof failed"); } const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); window.location.search.length; sessionStorage.setItem("mode", navigator.onLine && "search"); window.history.replaceState({}, "", "?mode=raw#ready"); localStorage.setItem("format", formatted); matchMedia("(prefers-reduced-motion: reduce)"); clipboard.writeText(root.textContent); setTimeout("noop", 5); queueMicrotask("noop"); host:setTextContent("#result", expr(root.textContent)); host:setTextContent("#formatted", expr(formatted)); host:setTextContent("#href", expr(current.href))</script></main>`; got != want {
		t.Fatalf("DumpDOM() after browser-global bootstrap = %q, want %q", got, want)
	}

	if got := session.HistoryLength(); got != 1 {
		t.Fatalf("HistoryLength() after browser-global bootstrap = %d, want 1", got)
	}
	if got, ok := session.HistoryState(); !ok || got != "[object Object]" {
		t.Fatalf("HistoryState() after browser-global bootstrap = (%q, %v), want ([object Object], true)", got, ok)
	}
	if got := session.LocalStorage()["format"]; got != "1.23" {
		t.Fatalf("LocalStorage()[format] = %q, want 1.23", got)
	}
	if got := session.SessionStorage()["mode"]; got != "search" {
		t.Fatalf("SessionStorage()[mode] = %q, want search", got)
	}
	if got := session.MatchMediaCalls(); len(got) != 1 || got[0].Query != "(prefers-reduced-motion: reduce)" {
		t.Fatalf("MatchMediaCalls() = %#v, want one prefers-reduced-motion query", got)
	}
	if got := session.ClipboardWrites(); len(got) != 1 || got[0] != "root" {
		t.Fatalf("ClipboardWrites() = %#v, want one root write", got)
	}
	if got := session.PendingTimers(); len(got) != 1 || got[0].Source != "noop" {
		t.Fatalf("PendingTimers() = %#v, want one noop timer", got)
	}
	if got := session.PendingMicrotasks(); len(got) != 0 {
		t.Fatalf("PendingMicrotasks() = %#v, want empty after bootstrap drain", got)
	}
	if got := session.StorageEvents(); len(got) != 2 {
		t.Fatalf("StorageEvents() = %#v, want two storage writes", got)
	}
}

func TestSessionBootstrapsFetchThroughBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><div id="meta"></div><div id="body"></div><script>window.fetch("https://example.test/api/message").then(function (response) { document.getElementById("meta").textContent = [response.url, response.status, response.ok].join("|"); return response.text().then(function (text) { document.getElementById("body").textContent = text; }); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.Registry().Fetch().RespondText("https://example.test/api/message", 201, "ok body")

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#meta"); err != nil {
		t.Fatalf("TextContent(#meta) error = %v", err)
	} else if got != "https://example.test/api/message|201|true" {
		t.Fatalf("TextContent(#meta) = %q, want fetch response metadata", got)
	}
	if got, err := session.TextContent("#body"); err != nil {
		t.Fatalf("TextContent(#body) error = %v", err)
	} else if got != "ok body" {
		t.Fatalf("TextContent(#body) = %q, want response body", got)
	}
	if got := session.FetchCalls(); len(got) != 1 || got[0].URL != "https://example.test/api/message" {
		t.Fatalf("FetchCalls() = %#v, want one fetch call", got)
	}
}

func TestSessionBootstrapsFetchFailureThroughBrowserGlobalsCatchChain(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>fetch("https://example.test/api/broken").catch(function (reason) { document.getElementById("out").textContent = reason; });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.Registry().Fetch().Fail("https://example.test/api/broken", "boom")

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `fetch("https://example.test/api/broken") failed: boom` {
		t.Fatalf("TextContent(#out) = %q, want fetch failure reason", got)
	}
	if got := session.FetchCalls(); len(got) != 1 || got[0].URL != "https://example.test/api/broken" {
		t.Fatalf("FetchCalls() = %#v, want one broken fetch call", got)
	}
}

func TestSessionRejectsWindowNameSymbolThroughBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><script>window.name = Symbol("token")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
	} else if !strings.Contains(err.Error(), "Cannot convert a Symbol value to a string") {
		t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
	}
	if got := session.WindowName(); got != "" {
		t.Fatalf("WindowName() after rejected Symbol input = %q, want empty", got)
	}
}

func TestSessionBootstrapsClassListItemSurface(t *testing.T) {
	const rawHTML = `<main><div id="root" class="alpha beta"></div><div id="out"></div><script>const list = document.getElementById("root").classList; document.getElementById("out").textContent = [list.item(0), list.item(1), list.item(2) === null].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha|beta|true" {
		t.Fatalf("TextContent(#out) = %q, want alpha|beta|true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after classList.item bootstrap", got)
	}
}

func TestSessionRejectsClassListItemNonNumericIndex(t *testing.T) {
	const rawHTML = `<main><div id="root" class="alpha beta"></div><script>document.getElementById("root").classList.item("nope")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "element.classList.item argument must be numeric") {
		t.Fatalf("ensureDOM() error = %v, want numeric conversion error", err)
	}
}

func TestSessionBootstrapsNumberParseIntSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const a = Number.parseInt("42", 10); const b = parseInt("0x10"); document.getElementById("out").textContent = [a, b].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "42|16" {
		t.Fatalf("TextContent(#out) = %q, want 42|16", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Number.parseInt bootstrap", got)
	}
}

func TestSessionBootstrapsNumberParseFloatSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const a = Number.parseFloat("3.2"); const b = parseFloat("4.5"); document.getElementById("out").textContent = [a, b].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "3.2|4.5" {
		t.Fatalf("TextContent(#out) = %q, want 3.2|4.5", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Number.parseFloat bootstrap", got)
	}
}

func TestSessionBootstrapsNumberIsIntegerSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [Number.isInteger(42), Number.isInteger(1.5), Number.isInteger(Number.NaN), Number.isInteger("42")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|false|false" {
		t.Fatalf("TextContent(#out) = %q, want true|false|false|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Number.isInteger bootstrap", got)
	}
}

func TestSessionBootstrapsNumberIsNaNSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [Number.isNaN(Number.NaN), Number.isNaN(1.5), Number.isNaN("42"), Number.isNaN()].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|false|false" {
		t.Fatalf("TextContent(#out) = %q, want true|false|false|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Number.isNaN bootstrap", got)
	}
}

func TestSessionBootstrapsURIComponentHelpers(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const state = { crop: "A&B 春", mode: "C++" }; const shareQuery = Object.keys(state).map(key => encodeURIComponent(key, "ignored") + "=" + encodeURIComponent(String(state[key]), "ignored")).join("&"); window.history.replaceState({}, "", "?" + shareQuery); const restored = {}; window.location.search.slice(1).split("&").forEach(part => { if (!part) return; const index = part.indexOf("="); const key = decodeURIComponent(part.slice(0, index), "ignored"); const value = decodeURIComponent(part.slice(index + 1), "ignored"); restored[key] = value; }); document.getElementById("out").textContent = [shareQuery, restored.crop, restored.mode, window.location.search, decodeURIComponent("%e6%98%a5", "ignored"), encodeURIComponent(), decodeURIComponent(), encodeURIComponent(), decodeURIComponent()].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "crop=A%26B%20%E6%98%A5&mode=C%2B%2B|A&B 春|C++|?crop=A%26B%20%E6%98%A5&mode=C%2B%2B|春|undefined|undefined|undefined|undefined" {
		t.Fatalf("TextContent(#out) = %q, want share URL round-trip", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after encode/decodeURIComponent bootstrap", got)
	}
}

func TestSessionRejectsURIComponentHelpersMalformedSequence(t *testing.T) {
	const rawHTML = `<main><script>decodeURIComponent("%C3%28")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want URI malformed failure")
	} else if !strings.Contains(err.Error(), "URI malformed") {
		t.Fatalf("ensureDOM() error = %v, want URI malformed message", err)
	}
}

func TestSessionBootstrapsURIHelpers(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [encodeURI("https://example.com/A B?x=春&y=1#frag", "ignored"), decodeURI("https://example.com/%2f%3f%23%26%20x", "ignored"), decodeURI("https://example.com/A%20B?x=%E6%98%A5&y=1#frag", "ignored")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "https://example.com/A%20B?x=%E6%98%A5&y=1#frag|https://example.com/%2f%3f%23%26 x|https://example.com/A B?x=春&y=1#frag" {
		t.Fatalf("TextContent(#out) = %q, want URI helper round-trip", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after encode/decodeURI bootstrap", got)
	}
}

func TestSessionRejectsURIHelpersSymbolInput(t *testing.T) {
	const rawHTML = `<main><script>decodeURIComponent(Symbol("token"))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
	} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
		t.Fatalf("ensureDOM() error = %#v, want runtime script error", err)
	} else if !strings.Contains(scriptErr.Message, "Cannot convert a Symbol value to a string") {
		t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
	}
}

func TestSessionRejectsHistoryHelpersSymbolInput(t *testing.T) {
	tests := []struct {
		name    string
		rawHTML string
	}{
		{
			name:    "pushState",
			rawHTML: `<main><script>window.history.pushState(Symbol("token"), "", "?push")</script></main>`,
		},
		{
			name:    "replaceStateState",
			rawHTML: `<main><script>window.history.replaceState(Symbol("token"), "", "?replace")</script></main>`,
		},
		{
			name:    "replaceStateTitle",
			rawHTML: `<main><script>window.history.replaceState({}, Symbol("token"), "?replace")</script></main>`,
		},
		{
			name:    "replaceStateURL",
			rawHTML: `<main><script>window.history.replaceState({}, "", Symbol("token"))</script></main>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(SessionConfig{HTML: tc.rawHTML})
			if _, err := session.ensureDOM(); err == nil {
				t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
			} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
				t.Fatalf("ensureDOM() error = %#v, want runtime script error", err)
			} else if !strings.Contains(scriptErr.Message, "Cannot convert a Symbol value to a string") {
				t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
			}
		})
	}
}

func TestSessionRejectsURIHelpersMalformedSequence(t *testing.T) {
	const rawHTML = `<main><script>decodeURI("%C3%28")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want URI malformed failure")
	} else if !strings.Contains(err.Error(), "URI malformed") {
		t.Fatalf("ensureDOM() error = %v, want URI malformed message", err)
	}
}

func TestSessionBootstrapsWindowInnerWidthSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = String(window.innerWidth < 880)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "false" {
		t.Fatalf("TextContent(#out) = %q, want false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after window.innerWidth bootstrap", got)
	}
}

func TestSessionBootstrapsLogicalAndShortCircuitBeforeTrim(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const value = null; const first = typeof value === "string" && value.trim() ? value : "fallback"; const second = false && value.trim() ? "boom" : "fallback"; document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "fallback|fallback" {
		t.Fatalf("TextContent(#out) = %q, want fallback|fallback", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after logical short-circuit bootstrap", got)
	}
}

func TestSessionBootstrapsStringTrimStartEnd(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["  Go  ".trimStart(), "  Go  ".trimEnd()].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "Go  |  Go" {
		t.Fatalf("TextContent(#out) = %q, want Go  |  Go", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string trimStart/trimEnd bootstrap", got)
	}
}

func TestSessionBootstrapsStringCaseConversion(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["Go".toLowerCase(), "go".toUpperCase()].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "go|GO" {
		t.Fatalf("TextContent(#out) = %q, want go|GO", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string case conversion bootstrap", got)
	}
}

func TestSessionBootstrapsStringConcat(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["go".concat(), "go".concat("!", 1, null, undefined), "あ".concat("い", "う")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "go|go!1nullundefined|あいう" {
		t.Fatalf("TextContent(#out) = %q, want go|go!1nullundefined|あいう", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string concat bootstrap", got)
	}
}

func TestSessionBootstrapsStringLocaleCompare(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["b".localeCompare("a"), "a".localeCompare("a"), "a".localeCompare("b"), "ä".localeCompare("z", "sv"), "item 2".localeCompare("item 10", undefined, { numeric: true })].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|0|-1|1|-1" {
		t.Fatalf("TextContent(#out) = %q, want 1|0|-1|1|-1", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string localeCompare bootstrap", got)
	}
}

func TestSessionRejectsStringLocaleCompareWithInvalidOptions(t *testing.T) {
	const rawHTML = `<main><script>"a".localeCompare("b", "en", "true")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.localeCompare options")
	} else if !strings.Contains(err.Error(), "String.localeCompare options argument must be an object") {
		t.Fatalf("ensureDOM() error = %v, want String.localeCompare options validation error", err)
	}
}

func TestSessionBootstrapsStringPadEnd(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["7".padEnd(3, "0"), "7".padEnd(5, "abc")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "700|7abca" {
		t.Fatalf("TextContent(#out) = %q, want 700|7abca", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string padEnd bootstrap", got)
	}
}

func TestSessionBootstrapsStringPadUnicodeFill(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["go".padStart(3, "あ"), "go".padEnd(3, "あ")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "あgo|goあ" {
		t.Fatalf("TextContent(#out) = %q, want あgo|goあ", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string pad unicode bootstrap", got)
	}
}

func TestSessionBootstrapsStringRepeat(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["go".repeat(3), "go".repeat(2.9), "go".repeat("2"), "あ".repeat(3)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "gogogo|gogo|gogo|あああ" {
		t.Fatalf("TextContent(#out) = %q, want gogogo|gogo|gogo|あああ", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string repeat bootstrap", got)
	}
}

func TestSessionBootstrapsStringReplaceAll(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["gooo".replaceAll("oo", "b"), "gooo".replaceAll(/o/g, "a"), "gooo".replaceAll("o", (match, offset, input) => match + ":" + offset), "A1B2".replaceAll(/([A-Z])([0-9])/g, (match, letter, digit, offset, input) => letter.toLowerCase() + digit + ":" + offset)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "gbo|gaaa|go:1o:2o:3|a1:0b2:2" {
		t.Fatalf("TextContent(#out) = %q, want gbo|gaaa|go:1o:2o:3|a1:0b2:2", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string replaceAll bootstrap", got)
	}
}

func TestSessionBootstrapsStringMatchAll(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["A1B2".matchAll(/([A-Z])([0-9])/g).map(match => match.join(":")).join("|"), "gooo".matchAll("oo").map(match => match[0]).join(",")].join("~")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "A1:A:1|B2:B:2~oo" {
		t.Fatalf("TextContent(#out) = %q, want A1:A:1|B2:B:2~oo", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string matchAll bootstrap", got)
	}
}

func TestSessionBootstrapsUnicodeStringSearchMethods(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["あいう".indexOf("い"), "あいう".indexOf("い", 2), "あいう".indexOf("", 5), "あいう".lastIndexOf("い"), "あいう".lastIndexOf("い", 2), "あいう".startsWith("あ"), "あいう".startsWith("い", 1), "あいう".endsWith("う"), "あいう".endsWith("い", 2), "あいう".includes("い", 1), "あいう".includes("い", 2)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|-1|3|1|1|true|true|true|true|true|false" {
		t.Fatalf("TextContent(#out) = %q, want 1|-1|3|1|1|true|true|true|true|true|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after unicode string search bootstrap", got)
	}
}

func TestSessionRejectsStringIncludesWithWrongArity(t *testing.T) {
	const rawHTML = `<main><script>"go".includes()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.includes() with wrong arity")
	}
}

func TestSessionBootstrapsStringSearch(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["abc".search(), "abc".search("b"), "abc".search(/b/), "あいう".search("い"), "あいう".search(/い/)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "-1|1|1|1|1" {
		t.Fatalf("TextContent(#out) = %q, want -1|1|1|1|1", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string search bootstrap", got)
	}
}

func TestSessionBootstrapsStringCharAt(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["あいう".charAt(-1), "あいう".charAt(1), "あいう".charAt(3)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "|い|" {
		t.Fatalf("TextContent(#out) = %q, want |い|", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string charAt bootstrap", got)
	}
}

func TestSessionBootstrapsStringSubstring(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["アイウ".substring(0, 2), "アイウ".substring(2, 0), "アイウ".substring(-1, 2), "アイウ".substring(1)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "アイ|アイ|アイ|イウ" {
		t.Fatalf("TextContent(#out) = %q, want アイ|アイ|アイ|イウ", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string substring bootstrap", got)
	}
}

func TestSessionBootstrapsArrayAndStringAt(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [[1, 2, 3].at(0), [1, 2, 3].at(-1), [1, 2, 3].at(3) === undefined, "アイウ".at(1)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|3|true|イ" {
		t.Fatalf("TextContent(#out) = %q, want 1|3|true|イ", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array/String.at bootstrap", got)
	}
}

func TestSessionBootstrapsStringCodePointAt(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = ["あいう".codePointAt(-1) === undefined, "あいう".codePointAt(1), "あいう".codePointAt(3) === undefined].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|12356|true" {
		t.Fatalf("TextContent(#out) = %q, want true|12356|true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after string codePointAt bootstrap", got)
	}
}

func TestSessionBootstrapsArrayFindLastFamily(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [[1, 2, 3, 2].findLast(v => v === 2), [1, 2, 3, 2].findLastIndex(v => v === 2), [1, 2, 3].findLastIndex(v => v === 4)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "2|3|-1" {
		t.Fatalf("TextContent(#out) = %q, want 2|3|-1", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.findLast/findLastIndex bootstrap", got)
	}
}

func TestSessionBootstrapsArrayCopyWithin(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const values = [1, 2, 3, 4, 5]; values.copyWithin(0, 3); const mixed = ["a", "b", "c", "d", "e"]; mixed.copyWithin(-2, 1, 3); document.getElementById("out").textContent = [values.join(","), mixed.join(",")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "4,5,3,4,5|a,b,c,b,c" {
		t.Fatalf("TextContent(#out) = %q, want 4,5,3,4,5|a,b,c,b,c", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.copyWithin bootstrap", got)
	}
}

func TestSessionBootstrapsArrayIncludes(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [1, 2, NaN].includes(NaN) + "|" + [1, 2, 3].includes(2, -2) + "|" + [1, 2, 3].includes(1, 1)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|true|false" {
		t.Fatalf("TextContent(#out) = %q, want true|true|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.includes bootstrap", got)
	}
}

func TestSessionBootstrapsArrayReduceAndReduceRight(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [[1, 2, 3, 4].reduce((acc, value) => acc + value), [1, 2, 3, 4].reduce((acc, value) => acc + value, 10), ["a", "b", "c"].reduceRight((acc, value) => acc + value)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "10|20|cba" {
		t.Fatalf("TextContent(#out) = %q, want 10|20|cba", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.reduce/reduceRight bootstrap", got)
	}
}

func TestSessionRejectsArrayReduceOnEmptyArray(t *testing.T) {
	const rawHTML = `<main><script>[].reduce((acc, value) => acc + value)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from Array.reduce() on empty array")
	}
}

func TestSessionRejectsArrayReduceRightOnEmptyArray(t *testing.T) {
	const rawHTML = `<main><script>[].reduceRight((acc, value) => acc + value)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from Array.reduceRight() on empty array")
	}
}

func TestSessionRejectsStringCharAtWithNonScalarIndex(t *testing.T) {
	const rawHTML = `<main><script>"go".charAt({})</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.charAt({})")
	}
}

func TestSessionRejectsStringCodePointAtWithNonScalarIndex(t *testing.T) {
	const rawHTML = `<main><script>"go".codePointAt({})</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.codePointAt({})")
	}
}

func TestSessionRejectsStringRepeatNegativeCount(t *testing.T) {
	const rawHTML = `<main><script>"go".repeat(-1)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.repeat(-1)")
	}
}

func TestSessionRejectsArrayAtWithNonScalarIndex(t *testing.T) {
	const rawHTML = `<main><script>[].at({})</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from Array.at({})")
	}
}

func TestSessionRejectsArrayFindLastWithoutCallback(t *testing.T) {
	const rawHTML = `<main><script>[].findLast()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from Array.findLast()")
	}
}

func TestSessionRejectsArrayFindLastIndexWithoutCallback(t *testing.T) {
	const rawHTML = `<main><script>[].findLastIndex()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from Array.findLastIndex()")
	}
}

func TestSessionRejectsStringAtWithNonScalarIndex(t *testing.T) {
	const rawHTML = `<main><script>"go".at({})</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.at({})")
	}
}

func TestSessionRejectsStringMatchAllWithoutGlobalRegex(t *testing.T) {
	const rawHTML = `<main><script>"gooo".matchAll(/o/)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error from String.matchAll(/o/)")
	}
}

func TestSessionBootstrapsMixedWhitespaceStringIndexingPreservesCharacters(t *testing.T) {
	const rawHTML = `<main><textarea id="source">A B	C D E　F
G</textarea><textarea id="result"></textarea><script>const line = document.getElementById("source").value; let mapped = ""; for (let index = 0; index < line.length; index += 1) { mapped += line[index]; } document.getElementById("result").value = mapped;</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#result"); err != nil {
		t.Fatalf("TextContent(#result) error = %v", err)
	} else if got != "A B\tC\u00A0D\u202FE\u3000F\nG" {
		t.Fatalf("TextContent(#result) = %q, want %q", got, "A B\tC\u00A0D\u202FE\u3000F\nG")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after mixed-whitespace string indexing bootstrap", got)
	}
}

func TestSessionRejectsHrefAssignmentOnNonHyperlinkElement(t *testing.T) {
	const rawHTML = `<main><div id="target"></div><script>document.getElementById("target").href = "/next"</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported href assignment on non-hyperlink element")
	} else if !strings.Contains(err.Error(), `.href`) {
		t.Fatalf("ensureDOM() error = %v, want href assignment error", err)
	}
}

func TestSessionRejectsSelectedIndexAssignmentOnNonSelectElement(t *testing.T) {
	const rawHTML = `<main><div id="target"></div><script>document.getElementById("target").selectedIndex = 0</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported selectedIndex assignment on non-select element")
	} else if !strings.Contains(err.Error(), `.selectedIndex`) {
		t.Fatalf("ensureDOM() error = %v, want selectedIndex assignment error", err)
	}
}

func TestSessionBootstrapsTextareaPlaceholderReflection(t *testing.T) {
	const rawHTML = `<main><textarea id="output"></textarea><script>const output = document.getElementById("output"); output.placeholder = "Sorted output appears here";</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, ok, err := session.GetAttribute("#output", "placeholder"); err != nil {
		t.Fatalf("GetAttribute(#output, placeholder) error = %v", err)
	} else if !ok {
		t.Fatalf("GetAttribute(#output, placeholder) = missing, want Sorted output appears here")
	} else if got != "Sorted output appears here" {
		t.Fatalf("GetAttribute(#output, placeholder) = %q, want Sorted output appears here", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after textarea placeholder bootstrap", got)
	}
}

func TestSessionBootstrapsTextInputPlaceholderReflection(t *testing.T) {
	const rawHTML = `<main><input id="output" type="text"><script>const output = document.getElementById("output"); output.placeholder = "Sorted output appears here";</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, ok, err := session.GetAttribute("#output", "placeholder"); err != nil {
		t.Fatalf("GetAttribute(#output, placeholder) error = %v", err)
	} else if !ok {
		t.Fatalf("GetAttribute(#output, placeholder) = missing, want Sorted output appears here")
	} else if got != "Sorted output appears here" {
		t.Fatalf("GetAttribute(#output, placeholder) = %q, want Sorted output appears here", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after text input placeholder bootstrap", got)
	}
}

func TestSessionRejectsPlaceholderAssignmentOnNonTextControl(t *testing.T) {
	const rawHTML = `<main><input id="target" type="checkbox"><script>document.getElementById("target").placeholder = "Sorted output appears here"</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported placeholder assignment on non-text input")
	} else if !strings.Contains(err.Error(), `.placeholder`) {
		t.Fatalf("ensureDOM() error = %v, want placeholder assignment error", err)
	}
}

func TestSessionBootstrapsAnchorHrefDownloadAndClick(t *testing.T) {
	const rawHTML = `<main><a id="download">Download</a><div id="out"></div><script>const link = document.getElementById("download"); link.href = "data:text/plain;charset=utf-8,Hello%20World"; link.download = "hello.txt"; document.getElementById("out").textContent = [link.href, link.download].join("|"); link.click()</script></main>`

	session := NewSession(SessionConfig{
		URL:  "https://example.test/base/",
		HTML: rawHTML,
	})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "data:text/plain;charset=utf-8,Hello%20World|hello.txt" {
		t.Fatalf("TextContent(#out) = %q, want data:text/plain;charset=utf-8,Hello%%20World|hello.txt", got)
	}
	downloads := session.Registry().Downloads().Artifacts()
	if len(downloads) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured download", downloads)
	}
	if downloads[0].FileName != "hello.txt" {
		t.Fatalf("Downloads()[0].FileName = %q, want hello.txt", downloads[0].FileName)
	}
	if got, want := string(downloads[0].Bytes), "Hello World"; got != want {
		t.Fatalf("Downloads()[0].Bytes = %q, want %q", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after anchor href/download bootstrap", got)
	}
}

func TestSessionBootstrapsBlobObjectUrlDownload(t *testing.T) {
	const rawHTML = `<main><button id="download">Download</button><div id="out"></div><script>function csvLine(values) { return values.map((value) => { const text = String(value === undefined || value === null ? "" : value); if (/[",\n]/.test(text)) return "\"" + text.replace(/"/g, "\"\"") + "\""; return text; }).join(","); } function buildCsv() { const lines = [ ["field_name", "field_group", "crop_name", "start_ym", "end_ym", "caution_tag", "status", "memo"], ["Field 1", "North Block", "Cabbage", "2026-02", "2026-05", "Brassicaceae", "fixed", "Spring crop plan"], ["Field 2", "North Block", "Tomato", "2026-03", "2026-08", "Solanaceae", "plan", "Summer-autumn crop"] ]; return lines.map(csvLine).join("\n"); } document.getElementById("download").addEventListener("click", () => { const blob = new Blob([buildCsv()], { type: "text/csv" }); if (!(blob instanceof Blob)) { throw new Error("Blob instanceof failed"); } const url = URL.createObjectURL(blob); const link = document.createElement("a"); link.href = url; link.download = "sample.csv"; document.body.appendChild(link); link.click(); document.body.removeChild(link); URL.revokeObjectURL(url); document.getElementById("out").textContent = url; });</script></main>`

	session := NewSession(SessionConfig{
		URL:  "https://example.test/base/",
		HTML: rawHTML,
	})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#download"); err != nil {
		t.Fatalf("Click(#download) error = %v", err)
	}

	downloads := session.Registry().Downloads().Artifacts()
	if len(downloads) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured download", downloads)
	}
	if downloads[0].FileName != "sample.csv" {
		t.Fatalf("Downloads()[0].FileName = %q, want sample.csv", downloads[0].FileName)
	}
	if got, want := string(downloads[0].Bytes), "field_name,field_group,crop_name,start_ym,end_ym,caution_tag,status,memo\nField 1,North Block,Cabbage,2026-02,2026-05,Brassicaceae,fixed,Spring crop plan\nField 2,North Block,Tomato,2026-03,2026-08,Solanaceae,plan,Summer-autumn crop"; got != want {
		t.Fatalf("Downloads()[0].Bytes = %q, want %q", got, want)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if !strings.HasPrefix(got, "blob:") {
		t.Fatalf("TextContent(#out) = %q, want blob URL", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after blob download bootstrap", got)
	}
}

func TestSessionBootstrapsXMLSerializerSerializesElementNodes(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const serializer = new XMLSerializer(); if (!(serializer instanceof XMLSerializer)) { throw new Error("XMLSerializer instanceof failed"); } const node = document.createElement("div"); node.setAttribute("data-test", "ok"); document.getElementById("out").textContent = serializer.serializeToString(node);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `<div data-test="ok"></div>` {
		t.Fatalf("TextContent(#out) = %q, want <div data-test=\"ok\"></div>", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after XMLSerializer bootstrap", got)
	}
}

func TestSessionBootstrapsDOMParserParserErrorDocument(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const parsed = new DOMParser().parseFromString("<svg><g></svg>", "image/svg+xml"); document.getElementById("out").textContent = [String(parsed.documentElement.nodeName || ""), String(parsed.documentElement.namespaceURI || ""), String(parsed.getElementsByTagName("parsererror").length)].join("|");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "parsererror|http://www.mozilla.org/newlayout/xml/parsererror.xml|1" {
		t.Fatalf("TextContent(#out) = %q, want parsererror|http://www.mozilla.org/newlayout/xml/parsererror.xml|1", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after DOMParser parsererror bootstrap", got)
	}
}

func TestSessionBootstrapsRevokedBlobObjectUrlSkipsDownloadCapture(t *testing.T) {
	const rawHTML = `<main><button id="download">Download</button><div id="out"></div><script>document.getElementById("download").addEventListener("click", () => { const blob = new Blob(["hello"], { type: "text/plain" }); const url = URL.createObjectURL(blob); const link = document.createElement("a"); link.href = url; link.download = "hello.txt"; document.body.appendChild(link); URL.revokeObjectURL(url); link.click(); document.body.removeChild(link); document.getElementById("out").textContent = url; });</script></main>`

	session := NewSession(SessionConfig{
		URL:  "https://example.test/base/",
		HTML: rawHTML,
	})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#download"); err != nil {
		t.Fatalf("Click(#download) error = %v", err)
	}

	if downloads := session.Registry().Downloads().Artifacts(); len(downloads) != 0 {
		t.Fatalf("Downloads().Artifacts() = %#v, want no captured download for revoked blob URL", downloads)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if !strings.HasPrefix(got, "blob:") {
		t.Fatalf("TextContent(#out) = %q, want revoked blob URL text", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after revoked blob download bootstrap", got)
	}
}

func TestSessionBootstrapsCanvasImagePngExport(t *testing.T) {
	const rawHTML = `<main><div id="state"></div><div id="out"></div><script>const img = new Image(); const canvas = document.createElement("canvas"); canvas.width = 20; canvas.height = 20; const ctx = canvas.getContext("2d"); document.getElementById("state").textContent = [img instanceof HTMLImageElement, canvas instanceof HTMLCanvasElement, !!ctx, canvas.toDataURL().startsWith("data:image/png;base64,"), typeof ctx.drawImage].join("|"); ctx.drawImage(img, 0, 0); canvas.toBlob((blob) => { const url = URL.createObjectURL(blob); const link = document.createElement("a"); link.href = url; link.download = "canvas.png"; document.body.appendChild(link); link.click(); document.body.removeChild(link); URL.revokeObjectURL(url); document.getElementById("out").textContent = blob ? "ok" : "missing"; }, "image/png");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#state"); err != nil {
		t.Fatalf("TextContent(#state) error = %v", err)
	} else if got != "true|true|true|true|function" {
		t.Fatalf("TextContent(#state) = %q, want true|true|true|true|function", got)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	downloads := session.Registry().Downloads().Artifacts()
	if len(downloads) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured PNG download", downloads)
	}
	if downloads[0].FileName != "canvas.png" {
		t.Fatalf("Downloads()[0].FileName = %q, want canvas.png", downloads[0].FileName)
	}
	if len(downloads[0].Bytes) < 8 {
		t.Fatalf("Downloads()[0].Bytes length = %d, want PNG signature bytes", len(downloads[0].Bytes))
	}
	if got, want := string(downloads[0].Bytes[:8]), "\x89PNG\r\n\x1a\n"; got != want {
		t.Fatalf("Downloads()[0].Bytes signature = %q, want PNG signature", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after canvas PNG export bootstrap", got)
	}
}

func TestSessionBootstrapsAsyncTimerCallbackCanvasPngExport(t *testing.T) {
	const rawHTML = `<main><button id="go" type="button">Go</button><div id="out"></div><script>async function svgMarkupToPngBlob(svgMarkup, widthPx, heightPx) { const blob = new Blob([svgMarkup], { type: "image/svg+xml;charset=utf-8" }); const url = URL.createObjectURL(blob); try { const image = await new Promise((resolve, reject) => { const img = new Image(); img.onload = () => resolve(img); img.onerror = reject; img.src = url; }); const canvas = document.createElement("canvas"); canvas.width = widthPx; canvas.height = heightPx; const ctx = canvas.getContext("2d"); ctx.drawImage(image, 0, 0, canvas.width, canvas.height); return await new Promise((resolve, reject) => { canvas.toBlob((pngBlob) => { if (pngBlob) resolve(pngBlob); else reject(new Error("png blob failed")); }, "image/png"); }); } finally { URL.revokeObjectURL(url); } } function downloadBlob(filename, blob) { const url = URL.createObjectURL(blob); const anchor = document.createElement("a"); anchor.href = url; anchor.download = filename; document.body.appendChild(anchor); anchor.click(); anchor.remove(); URL.revokeObjectURL(url); } document.getElementById("go").addEventListener("click", () => { const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#000"/></svg>'; window.setTimeout(async () => { const png = await svgMarkupToPngBlob(svg, 8, 8); downloadBlob("canvas.png", png); document.getElementById("out").textContent = "ok"; }, 230); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := session.AdvanceTime(230); err != nil {
		t.Fatalf("AdvanceTime(230) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	downloads := session.Registry().Downloads().Artifacts()
	if len(downloads) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured PNG download", downloads)
	}
	if downloads[0].FileName != "canvas.png" {
		t.Fatalf("Downloads()[0].FileName = %q, want canvas.png", downloads[0].FileName)
	}
	if len(downloads[0].Bytes) < 8 {
		t.Fatalf("Downloads()[0].Bytes length = %d, want PNG signature bytes", len(downloads[0].Bytes))
	}
	if got, want := string(downloads[0].Bytes[:8]), "\x89PNG\r\n\x1a\n"; got != want {
		t.Fatalf("Downloads()[0].Bytes signature = %q, want PNG signature", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after async timer canvas PNG export bootstrap", got)
	}
}

func TestSessionBootstrapsAsyncTimerCallbackCanvasPngExportWithWindowURLAlias(t *testing.T) {
	const rawHTML = `<main><button id="go" type="button">Go</button><div id="out"></div><script>` +
		`async function svgMarkupToPngBlob(svgMarkup, widthPx, heightPx, surface) { const blob = new Blob([svgMarkup], { type: "image/svg+xml;charset=utf-8" }); const urlSurface = surface || URL; const url = urlSurface.createObjectURL(blob); try { const image = await new Promise((resolve, reject) => { const img = new Image(); img.onload = () => resolve(img); img.onerror = reject; img.src = url; }); const canvas = document.createElement("canvas"); canvas.width = widthPx; canvas.height = heightPx; const ctx = canvas.getContext("2d"); ctx.drawImage(image, 0, 0, canvas.width, canvas.height); return await new Promise((resolve, reject) => { canvas.toBlob((pngBlob) => { if (pngBlob) resolve(pngBlob); else reject(new Error("png blob failed")); }, "image/png"); }); } finally { urlSurface.revokeObjectURL(url); } }` +
		`function downloadBlob(filename, blob, surface) { const urlSurface = surface || URL; const url = urlSurface.createObjectURL(blob); const anchor = document.createElement("a"); anchor.href = url; anchor.download = filename; document.body.appendChild(anchor); anchor.click(); anchor.remove(); urlSurface.revokeObjectURL(url); }` +
		`document.getElementById("go").addEventListener("click", () => { const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#000"/></svg>'; if (window.URL !== URL) { throw new Error("window.URL alias mismatch"); } window.setTimeout(async () => { const direct = await svgMarkupToPngBlob(svg, 8, 8, URL); downloadBlob("canvas-direct.png", direct, URL); document.getElementById("out").textContent = "direct"; }, 230); window.setTimeout(async () => { const alias = await svgMarkupToPngBlob(svg, 8, 8, window.URL); downloadBlob("canvas-window.png", alias, window.URL); document.getElementById("out").textContent += "|window"; }, 460); });` +
		`</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := session.AdvanceTime(460); err != nil {
		t.Fatalf("AdvanceTime(460) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "direct|window" {
		t.Fatalf("TextContent(#out) = %q, want direct|window", got)
	}
	downloads := session.Registry().Downloads().Artifacts()
	if len(downloads) != 2 {
		t.Fatalf("Downloads().Artifacts() = %#v, want two captured PNG downloads", downloads)
	}
	if downloads[0].FileName != "canvas-direct.png" {
		t.Fatalf("Downloads()[0].FileName = %q, want canvas-direct.png", downloads[0].FileName)
	}
	if downloads[1].FileName != "canvas-window.png" {
		t.Fatalf("Downloads()[1].FileName = %q, want canvas-window.png", downloads[1].FileName)
	}
	for i, download := range downloads {
		if len(download.Bytes) < 8 {
			t.Fatalf("Downloads()[%d].Bytes length = %d, want PNG signature bytes", i, len(download.Bytes))
		}
		if got, want := string(download.Bytes[:8]), "\x89PNG\r\n\x1a\n"; got != want {
			t.Fatalf("Downloads()[%d].Bytes signature = %q, want PNG signature", i, got)
		}
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after async timer canvas PNG export with window.URL alias", got)
	}
}

func TestSessionBootstrapsAsyncTimerCallbackAwaitingPromiseResolve(t *testing.T) {
	const rawHTML = `<main><button id="go" type="button">Go</button><div id="out"></div><script>document.getElementById("go").addEventListener("click", () => { window.setTimeout(async () => { const value = await Promise.resolve("ok"); document.getElementById("out").textContent = value; }, 230); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := session.AdvanceTime(230); err != nil {
		t.Fatalf("AdvanceTime(230) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after async timer await bootstrap", got)
	}
}

func TestSessionBootstrapsAsyncTimerCallbackAwaitingNestedTimeout(t *testing.T) {
	const rawHTML = `<main><button id="go" type="button">Go</button><div id="out"></div><script>document.getElementById("go").addEventListener("click", () => { window.setTimeout(async () => { document.getElementById("out").textContent = "start"; const promise = new Promise((resolve) => { window.setTimeout(() => resolve("ok"), 0); }); promise.then(() => { document.getElementById("out").textContent += "|then"; }); try { const value = await promise; document.getElementById("out").textContent = value; } catch (error) { document.getElementById("out").textContent = "catch:" + (error && error.message ? error.message : String(error)); } }, 230); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if err := session.AdvanceTime(230); err != nil {
		t.Fatalf("AdvanceTime(230) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nested timer await bootstrap", got)
	}
}

func TestSessionBootstrapsSvgBlobImageLoadCallbacks(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#000"/></svg>'; const blob = new Blob([svg], { type: "image/svg+xml;charset=utf-8" }); const url = URL.createObjectURL(blob); const img = new Image(); const seen = []; img.onload = () => seen.push("onload"); img.addEventListener("load", () => seen.push("listener")); img.addEventListener("error", () => seen.push("error")); img.src = url; setTimeout(() => { document.getElementById("out").textContent = seen.sort().join("|") || "pending"; URL.revokeObjectURL(url); }, 1);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "listener|onload" {
		t.Fatalf("TextContent(#out) = %q, want listener|onload", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after SVG blob image load bootstrap", got)
	}
}

func TestSessionBootstrapsRevokedSvgBlobImageErrorCallbacks(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#000"/></svg>'; const blob = new Blob([svg], { type: "image/svg+xml;charset=utf-8" }); const url = URL.createObjectURL(blob); const img = new Image(); const seen = []; img.onerror = () => seen.push("onerror"); img.addEventListener("error", () => seen.push("listener")); URL.revokeObjectURL(url); img.src = url; setTimeout(() => { document.getElementById("out").textContent = seen.sort().join("|") || "pending"; }, 1);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "listener|onerror" {
		t.Fatalf("TextContent(#out) = %q, want listener|onerror", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after revoked SVG blob image bootstrap", got)
	}
}

func TestSessionBootstrapsSvgBlobImageHandlerDeletionClearsOnload(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#000"/></svg>'; const blob = new Blob([svg], { type: "image/svg+xml;charset=utf-8" }); const url = URL.createObjectURL(blob); const img = new Image(); const seen = []; img.onload = () => seen.push("onload"); delete img.onload; img.addEventListener("load", () => seen.push("listener")); img.src = url; setTimeout(() => { document.getElementById("out").textContent = seen.sort().join("|") || "pending"; URL.revokeObjectURL(url); }, 1);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "listener" {
		t.Fatalf("TextContent(#out) = %q, want listener", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after deleted SVG blob image onload bootstrap", got)
	}
}

func TestSessionBootstrapsSvgBlobImageHandlerDeletionClearsOnerror(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#000"/></svg>'; const blob = new Blob([svg], { type: "image/svg+xml;charset=utf-8" }); const url = URL.createObjectURL(blob); const img = new Image(); const seen = []; img.onerror = () => seen.push("onerror"); delete img.onerror; img.addEventListener("error", () => seen.push("listener")); URL.revokeObjectURL(url); img.src = url; setTimeout(() => { document.getElementById("out").textContent = seen.sort().join("|") || "pending"; }, 1);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "listener" {
		t.Fatalf("TextContent(#out) = %q, want listener", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after deleted SVG blob image onerror bootstrap", got)
	}
}

func TestSessionRejectsCanvasToBlobWithUnsupportedType(t *testing.T) {
	const rawHTML = `<main><script>const canvas = document.createElement("canvas"); canvas.toBlob(() => {}, "image/jpeg");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported canvas export type")
	} else if !strings.Contains(err.Error(), "image/png") {
		t.Fatalf("ensureDOM() error = %v, want image/png restriction", err)
	}
}

func TestSessionRejectsURLCreateObjectURLForNonBlobValue(t *testing.T) {
	const rawHTML = `<main><script>URL.createObjectURL({})</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported Blob value")
	} else if !strings.Contains(err.Error(), "Blob") {
		t.Fatalf("ensureDOM() error = %v, want blob validation failure", err)
	}
}

func TestSessionBootstrapsCompostInputConverterBuiltins(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const value = NaN; const windowValue = window.NaN; const rounded = [Math.round(1.5), Math.round(-1.5)].join("|"); document.getElementById("out").textContent = [value !== value, windowValue !== windowValue, rounded, String("  Go  ").trim()].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|true|2|-1|Go" {
		t.Fatalf("TextContent(#out) = %q, want true|true|2|-1|Go", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after compost builtins bootstrap", got)
	}
}

func TestSessionBootstrapsInfinityGlobalSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>let bestScore = -Infinity; document.getElementById("out").textContent = [String(bestScore), String(Infinity), String(Number.POSITIVE_INFINITY), String(Number.NEGATIVE_INFINITY)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "-Infinity|Infinity|Infinity|-Infinity" {
		t.Fatalf("TextContent(#out) = %q, want -Infinity|Infinity|Infinity|-Infinity", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Infinity bootstrap", got)
	}
}

func TestSessionBootstrapsDateInstanceOfSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); document.getElementById("out").textContent = [date instanceof Date, Number.isNaN(date.getTime()), date.getFullYear(), date.getUTCFullYear(), date.getMonth(), date.getUTCMonth(), date.getDate(), date.getUTCDate(), date.getDay(), date.getUTCDay(), date.getHours(), date.getUTCHours(), date.getMinutes(), date.getUTCMinutes(), date.getSeconds(), date.getUTCSeconds(), date.getMilliseconds(), date.getUTCMilliseconds(), date.getTimezoneOffset(), date.toDateString(), date.toTimeString(), date.toUTCString(), date.toLocaleString("en-US"), date.toLocaleTimeString("en-US"), String(Date.parse(date.toUTCString())), String(new Date(date.toUTCString()).getTime()), String(date.setTime(1700000004567))].join("|")</script></main>`
	dt := time.UnixMilli(1700000000123).UTC()
	want := strings.Join([]string{
		"true",
		"false",
		strconv.Itoa(dt.Year()),
		strconv.Itoa(dt.Year()),
		strconv.Itoa(int(dt.Month()) - 1),
		strconv.Itoa(int(dt.Month()) - 1),
		strconv.Itoa(dt.Day()),
		strconv.Itoa(dt.Day()),
		strconv.Itoa(int(dt.Weekday())),
		strconv.Itoa(int(dt.Weekday())),
		strconv.Itoa(dt.Hour()),
		strconv.Itoa(dt.Hour()),
		strconv.Itoa(dt.Minute()),
		strconv.Itoa(dt.Minute()),
		strconv.Itoa(dt.Second()),
		strconv.Itoa(dt.Second()),
		strconv.Itoa(dt.Nanosecond() / int(time.Millisecond)),
		strconv.Itoa(dt.Nanosecond() / int(time.Millisecond)),
		"0",
		dt.Format("Mon Jan _2 2006"),
		dt.Format("15:04:05 GMT"),
		dt.Format("Mon, 02 Jan 2006 15:04:05 GMT"),
		dt.Format("1/2/2006, 3:04:05 PM"),
		dt.Format("3:04:05 PM"),
		strconv.FormatInt(dt.Truncate(time.Second).UnixMilli(), 10),
		strconv.FormatInt(dt.Truncate(time.Second).UnixMilli(), 10),
		"1700000004567",
	}, "|")

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != want {
		t.Fatalf("TextContent(#out) = %q, want %s", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date instanceof bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetMillisecondsMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setMilliseconds(567)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCMilliseconds(999)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1700000000567|1700000000567|2023-11-14T22:13:20.567Z|1700000000999|1700000000999|2023-11-14T22:13:20.999Z" {
		t.Fatalf("TextContent(#out) = %q, want 1700000000567|1700000000567|2023-11-14T22:13:20.567Z|1700000000999|1700000000999|2023-11-14T22:13:20.999Z", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setMilliseconds bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetDateMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setDate(5)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCDate(31)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	dt := time.UnixMilli(1700000000123).UTC()
	wantFirst := time.Date(dt.Year(), dt.Month(), 5, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), time.UTC).UnixMilli()
	wantSecond := time.Date(dt.Year(), dt.Month(), 31, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), time.UTC).UnixMilli()
	want := strings.Join([]string{
		strconv.FormatInt(wantFirst, 10),
		strconv.FormatInt(wantFirst, 10),
		time.UnixMilli(wantFirst).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(wantSecond, 10),
		strconv.FormatInt(wantSecond, 10),
		time.UnixMilli(wantSecond).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != want {
		t.Fatalf("TextContent(#out) = %q, want %s", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setDate bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetMonthMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setMonth(0)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCMonth(11, 31)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	dt := time.UnixMilli(1700000000123).UTC()
	wantFirst := time.Date(dt.Year(), time.January, dt.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), time.UTC).UnixMilli()
	wantSecond := time.Date(dt.Year(), time.December, 31, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), time.UTC).UnixMilli()
	want := strings.Join([]string{
		strconv.FormatInt(wantFirst, 10),
		strconv.FormatInt(wantFirst, 10),
		time.UnixMilli(wantFirst).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(wantSecond, 10),
		strconv.FormatInt(wantSecond, 10),
		time.UnixMilli(wantSecond).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != want {
		t.Fatalf("TextContent(#out) = %q, want %s", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setMonth bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetFullYearMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setFullYear(2024)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCFullYear(2025, 0, 15)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	dt := time.UnixMilli(1700000000123).UTC()
	wantFirst := time.Date(2024, dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), time.UTC).UnixMilli()
	wantSecond := time.Date(2025, time.January, 15, dt.Hour(), dt.Minute(), dt.Second(), dt.Nanosecond(), time.UTC).UnixMilli()
	want := strings.Join([]string{
		strconv.FormatInt(wantFirst, 10),
		strconv.FormatInt(wantFirst, 10),
		time.UnixMilli(wantFirst).UTC().Format("2006-01-02T15:04:05.000Z"),
		strconv.FormatInt(wantSecond, 10),
		strconv.FormatInt(wantSecond, 10),
		time.UnixMilli(wantSecond).UTC().Format("2006-01-02T15:04:05.000Z"),
	}, "|")

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != want {
		t.Fatalf("TextContent(#out) = %q, want %s", got, want)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setFullYear bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetSecondsMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setSeconds(5, 7)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCSeconds(59, 8)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1699999985007|1699999985007|2023-11-14T22:13:05.007Z|1700000039008|1700000039008|2023-11-14T22:13:59.008Z" {
		t.Fatalf("TextContent(#out) = %q, want 1699999985007|1699999985007|2023-11-14T22:13:05.007Z|1700000039008|1700000039008|2023-11-14T22:13:59.008Z", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setSeconds bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetMinutesMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setMinutes(4, 5, 6)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCMinutes(59, 58, 57)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1699999445006|1699999445006|2023-11-14T22:04:05.006Z|1700002798057|1700002798057|2023-11-14T22:59:58.057Z" {
		t.Fatalf("TextContent(#out) = %q, want 1699999445006|1699999445006|2023-11-14T22:04:05.006Z|1700002798057|1700002798057|2023-11-14T22:59:58.057Z", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setMinutes bootstrap", got)
	}
}

func TestSessionBootstrapsDateSetHoursMutatesReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const date = new Date(1700000000123); const first = [String(date.setHours(4, 5, 6, 7)), String(date.getTime()), date.toISOString()].join("|"); const second = [String(date.setUTCHours(23, 58, 57, 56)), String(date.getTime()), date.toISOString()].join("|"); document.getElementById("out").textContent = [first, second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1699934706007|1699934706007|2023-11-14T04:05:06.007Z|1700006337056|1700006337056|2023-11-14T23:58:57.056Z" {
		t.Fatalf("TextContent(#out) = %q, want 1699934706007|1699934706007|2023-11-14T04:05:06.007Z|1700006337056|1700006337056|2023-11-14T23:58:57.056Z", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Date.setHours bootstrap", got)
	}
}

func TestSessionBootstrapsMathCeilSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [Math.ceil(1.1), Math.ceil(-1.1), 1 / Math.ceil(-0.1)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "2|-1|-Infinity" {
		t.Fatalf("TextContent(#out) = %q, want 2|-1|-Infinity", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Math.ceil bootstrap", got)
	}
}

func TestSessionBootstrapsMathPowSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [Math.pow(2, 3), Math.pow(9, 0.5), Math.pow(2, -3)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "8|3|0.125" {
		t.Fatalf("TextContent(#out) = %q, want 8|3|0.125", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Math.pow bootstrap", got)
	}
}

func TestSessionRejectsMathPowWrongArity(t *testing.T) {
	const rawHTML = `<main><script>Math.pow(2)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Math.pow arity failure")
	} else if !strings.Contains(err.Error(), "Math.pow expects 2 arguments") {
		t.Fatalf("ensureDOM() error = %v, want Math.pow arity message", err)
	}
}

func TestSessionBootstrapsMathTruncSurface(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [Math.trunc(1.9), Math.trunc(-1.9), 1 / Math.trunc(-0.1)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|-1|-Infinity" {
		t.Fatalf("TextContent(#out) = %q, want 1|-1|-Infinity", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Math.trunc bootstrap", got)
	}
}

func TestSessionRejectsMathTruncWrongArity(t *testing.T) {
	const rawHTML = `<main><script>Math.trunc()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Math.trunc arity failure")
	} else if !strings.Contains(err.Error(), "Math.trunc expects 1 argument") {
		t.Fatalf("ensureDOM() error = %v, want Math.trunc arity message", err)
	}
}

func TestSessionBootstrapsMathRemainingMethods(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [Math.acos(1), Math.atan2(1, 1), Math.clz32(1), Math.imul(-1, 2), Math.fround(16777217), Math.hypot(3, 4), Math.log1p(1), Math.sign(-3), String(1 / Math.sign(-0)), String(1 / Math.min(0, -0)), String(1 / Math.max(-0, 0))].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "0|0.7853981633974483|31|-2|16777216|5|0.6931471805599453|-1|-Infinity|-Infinity|Infinity" {
		t.Fatalf("TextContent(#out) = %q, want 0|0.7853981633974483|31|-2|16777216|5|0.6931471805599453|-1|-Infinity|-Infinity|Infinity", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Math remaining methods bootstrap", got)
	}
}

func TestSessionBootstrapsEmptyObjectPropertyCreation(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = {}; out.alpha = 1; out["beta"] = 2; document.getElementById("out").textContent = Object.keys(out).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha,beta" {
		t.Fatalf("TextContent(#out) = %q, want alpha,beta", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after empty object property bootstrap", got)
	}
}

func TestSessionBootstrapsObjectKeysForEachPropertyCopy(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const source = { alpha: 1, beta: 2 }; const out = {}; Object.keys(source).forEach((key) => { out[key] = source[key]; }); document.getElementById("out").textContent = Object.keys(out).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha,beta" {
		t.Fatalf("TextContent(#out) = %q, want alpha,beta", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.keys forEach bootstrap", got)
	}
}

func TestSessionBootstrapsObjectPrototypeHasOwnPropertyCall(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const sym = Symbol("token"); const object = { alpha: 1, [sym]: 2 }; const array = [1, 2]; const fn = function Base() {}; document.getElementById("out").textContent = [Object.prototype.hasOwnProperty.call(object, "alpha"), Object.prototype.hasOwnProperty.call(object, "beta"), Object.prototype.hasOwnProperty.call(array, "0"), Object.prototype.hasOwnProperty.call(array, "length"), Object.prototype.hasOwnProperty.call(array, "2"), Object.prototype.hasOwnProperty.call(object, sym), Object.prototype.hasOwnProperty.call(object, Symbol("token")), Object.prototype.hasOwnProperty.call(fn, "prototype")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|true|true|false|true|false|true" {
		t.Fatalf("TextContent(#out) = %q, want true|false|true|true|false|true|false|true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.prototype.hasOwnProperty bootstrap", got)
	}
}

func TestSessionBootstrapsObjectHasOwn(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const sym = Symbol("token"); const object = { alpha: 1, [sym]: 2 }; const array = [1, 2]; const text = "go"; const fn = function Base() {}; document.getElementById("out").textContent = [Object.hasOwn(object, "alpha"), Object.hasOwn(object, "beta"), Object.hasOwn(array, "0"), Object.hasOwn(array, "length"), Object.hasOwn(array, "2"), Object.hasOwn(text, "0"), Object.hasOwn(text, "length"), Object.hasOwn(object, sym), Object.hasOwn(object, Symbol("token")), Object.hasOwn(fn, "prototype")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|true|true|false|true|true|true|false|true" {
		t.Fatalf("TextContent(#out) = %q, want true|false|true|true|false|true|true|true|false|true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.hasOwn bootstrap", got)
	}
}

func TestSessionBootstrapsObjectGetOwnPropertyNames(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const sym = Symbol("token"); const object = { alpha: 1, [sym]: 2 }; const array = [1, 2]; const text = "go"; const fn = function Base() {}; document.getElementById("out").textContent = [Object.getOwnPropertyNames(object).join(","), Object.getOwnPropertyNames(array).join(","), Object.getOwnPropertyNames(text).join(","), Object.getOwnPropertyNames(fn).join(",")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha|0,1,length|0,1,length|prototype" {
		t.Fatalf("TextContent(#out) = %q, want alpha|0,1,length|0,1,length|prototype", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.getOwnPropertyNames bootstrap", got)
	}
}

func TestSessionRejectsObjectHasOwnWrongArity(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>Object.hasOwn({ alpha: 1 })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Object.hasOwn expects 2 arguments") {
		t.Fatalf("ensureDOM() error = %v, want arity error", err)
	}
}

func TestSessionRejectsObjectGetOwnPropertyNamesWrongArity(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>Object.getOwnPropertyNames({ alpha: 1 }, { beta: 2 })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Object.getOwnPropertyNames expects 1 argument") {
		t.Fatalf("ensureDOM() error = %v, want arity error", err)
	}
}

func TestSessionRejectsObjectGetOwnPropertyNamesOnNullishReceiver(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>Object.getOwnPropertyNames(null)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want runtime error")
	} else if !strings.Contains(err.Error(), "Cannot convert undefined or null to object") {
		t.Fatalf("ensureDOM() error = %v, want nullish conversion error", err)
	}
}

func TestSessionBootstrapsComputedPropertyKeys(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const key = "alpha"; const source = { [key]: 1, beta: 2 }; document.getElementById("out").textContent = Object.keys(source).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha,beta" {
		t.Fatalf("TextContent(#out) = %q, want alpha,beta", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after computed property bootstrap", got)
	}
}

func TestSessionBootstrapsReturnedComputedCountsObjectKeys(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>function makeGroup(symbol, count) { return { counts: { [symbol]: count }, order: [symbol], normalized: symbol + count }; } const group = makeGroup("alpha", 1); document.getElementById("out").textContent = Object.keys(group.counts).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha" {
		t.Fatalf("TextContent(#out) = %q, want alpha", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after returned computed counts bootstrap", got)
	}
}

func TestSessionBootstrapsMergeCountsFromReturnedComputedCountsObject(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>function makeGroup(symbol, count) { return { counts: { [symbol]: count }, order: [symbol], normalized: symbol + count }; } const group = makeGroup("alpha", 1); const target = {}; Object.keys(group.counts).forEach((key) => { target[key] = (target[key] || 0) + group.counts[key]; }); document.getElementById("out").textContent = Object.keys(target).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha" {
		t.Fatalf("TextContent(#out) = %q, want alpha", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after merge counts returned object bootstrap", got)
	}
}

func TestSessionBootstrapsObjectKeysForEachMultiplicationCopy(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const source = { alpha: 1, beta: 2 }; function multiplyCounts(map, factor) { const out = {}; Object.keys(map).forEach((key) => { out[key] = map[key] * factor; }); return out; } const out = multiplyCounts(source, 3); document.getElementById("out").textContent = JSON.stringify(out)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"alpha":3,"beta":6}` {
		t.Fatalf("TextContent(#out) = %q, want {\"alpha\":3,\"beta\":6}", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.keys multiplication bootstrap", got)
	}
}

func TestSessionBootstrapsObjectKeysForEachMultiplicationCopyInClickHandler(t *testing.T) {
	const rawHTML = `<main><button id="go" type="button">go</button><div id="out"></div><script>(() => { const source = { alpha: 1, beta: 2 }; function multiplyCounts(map, factor) { const out = {}; Object.keys(map).forEach((key) => { out[key] = map[key] * factor; }); return out; } document.getElementById("go").addEventListener("click", () => { const out = multiplyCounts(source, 3); document.getElementById("out").textContent = JSON.stringify(out); }); })();</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"alpha":3,"beta":6}` {
		t.Fatalf("TextContent(#out) = %q, want {\"alpha\":3,\"beta\":6}", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after click bootstrap", got)
	}
}

func TestSessionBootstrapsObjectKeysForEachMergeCounts(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const source = { alpha: 1, beta: 2 }; const target = {}; Object.keys(source).forEach((key) => { target[key] = (target[key] || 0) + source[key]; }); document.getElementById("out").textContent = Object.keys(target).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha,beta" {
		t.Fatalf("TextContent(#out) = %q, want alpha,beta", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after mergeCounts bootstrap", got)
	}
}

func TestSessionBootstrapsDeleteComputedKeysOnConstObjectBindings(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const keys = { lot: "lot", harvestDate: "harvestDate", fieldId: "fieldId", item: "item", grade: "grade" }; const payload = { [keys.lot]: "2026-0205-F12-A", [keys.harvestDate]: "", [keys.fieldId]: "", [keys.item]: "", [keys.grade]: "" }; Object.keys(payload).forEach((key) => { if (payload[key] === "" || payload[key] == null) delete payload[key]; }); document.getElementById("out").textContent = JSON.stringify(payload)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"lot":"2026-0205-F12-A"}` {
		t.Fatalf("TextContent(#out) = %q, want omitted empty fields", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after const delete bootstrap", got)
	}
}

func TestSessionBootstrapsFormulaParserWithoutClick(t *testing.T) {
	const rawHTML = `
		<main>
		  <input id="formula" value="Al2(SO4)3">
		  <div id="out"></div>
		  <script>
		    (() => {
		      const knownElements = { Al: true, S: true, O: true };
		      const input = document.getElementById("formula");
		      const out = document.getElementById("out");

		      function parserError(message) {
		        return { message };
		      }

		      function multiplyCounts(map, factor) {
		        const result = {};
		        Object.keys(map).forEach((key) => {
		          result[key] = map[key] * factor;
		        });
		        return result;
		      }

		      function createParser(source) {
		        let index = 0;

		        function current() {
		          return source[index] || "";
		        }

		        function consume() {
		          const char = source[index] || "";
		          index += 1;
		          return char;
		        }

		        function isDigit(char) {
		          return /[0-9]/.test(char);
		        }

		        function isUpper(char) {
		          return /[A-Z]/.test(char);
		        }

		        function isLower(char) {
		          return /[a-z]/.test(char);
		        }

		        function parseNumber() {
		          const start = index;
		          let sawDigit = false;
		          while (isDigit(current())) {
		            sawDigit = true;
		            consume();
		          }
		          const raw = source.slice(start, index);
		          if (!sawDigit) {
		            throw parserError("invalid number");
		          }
		          return { raw, value: Number(raw) };
		        }

		        function parseOptionalMultiplier() {
		          if (isDigit(current())) return parseNumber();
		          return { raw: "", value: 1 };
		        }

		        function parseElementSymbol() {
		          const first = current();
		          if (!isUpper(first)) {
		            throw parserError("invalid symbol");
		          }
		          let symbol = consume();
		          if (isLower(current())) symbol += consume();
		          if (!knownElements[symbol]) {
		            throw parserError("unknown element");
		          }
		          return symbol;
		        }

		        function parseBracketGroup() {
		          const open = consume();
		          const close = open === "(" ? ")" : "]";
		          const inner = parseSequence(close);
		          if (current() !== close) {
		            throw parserError("Bracket mismatch detected.");
		          }
		          consume();
		          const multiplier = parseOptionalMultiplier();
		          return {
		            counts: multiplyCounts(inner.counts, multiplier.value),
		            order: inner.order.slice(),
		            normalized: open + inner.normalized + close + multiplier.raw,
		          };
		        }

		        function parseElementGroup() {
		          const symbol = parseElementSymbol();
		          const count = parseOptionalMultiplier();
		          return {
		            counts: { [symbol]: count.value },
		            order: [symbol],
		            normalized: symbol + count.raw,
		          };
		        }

		        function parseGroup() {
		          const char = current();
		          if (char === "(" || char === "[") {
		            return parseBracketGroup();
		          }
		          return parseElementGroup();
		        }

		        function parseSequence(stopChar) {
		          const counts = {};
		          const order = [];
		          let normalized = "";
		          while (index < source.length && current() !== stopChar) {
		            if (current() === ")" || current() === "]") {
		              throw parserError("unexpected close");
		            }
		            const group = parseGroup();
		            Object.keys(group.counts).forEach((key) => {
		              counts[key] = (counts[key] || 0) + group.counts[key];
		            });
		            group.order.forEach((item) => {
		              if (!order.includes(item)) order.push(item);
		            });
		            normalized += group.normalized;
		          }
		          return { counts, order, normalized };
		        }

		        function parseFragment() {
		          const body = parseSequence("");
		          if (index !== source.length) {
		            throw parserError("invalid tail");
		          }
		          return {
		            counts: body.counts,
		            order: body.order.slice(),
		            normalized: body.normalized,
		          };
		        }

		        return { parseFragment };
		      }

		      const parsed = createParser(input.value).parseFragment();
		      out.textContent = parsed.normalized + "|" + JSON.stringify(parsed.counts);
		    })();
		  </script>
		</main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `Al2(SO4)3|{"Al":2,"S":3,"O":12}` {
		t.Fatalf("TextContent(#out) = %q, want parsed counts output", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after formula parser bootstrap", got)
	}
}

func TestSessionBootstrapsShadowedLocalOutBinding(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>(() => { const out = document.getElementById("out"); function makeResult() { const out = {}; out.alpha = 1; out.beta = 2; return out; } out.textContent = JSON.stringify(makeResult()); })();</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"alpha":1,"beta":2}` {
		t.Fatalf("TextContent(#out) = %q, want shadowed local object", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after shadowed out bootstrap", got)
	}
}

func TestSessionBootstrapsBuiltinMapSlice(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const pickMap = new Map(); pickMap.set("sku-1", 12); pickMap.set("sku-2", 5); const deleted = pickMap.delete("sku-1", "extra"); const missing = pickMap.delete("missing", "extra"); document.getElementById("out").textContent = [String(deleted), String(missing), String(pickMap.size), String(pickMap.get("sku-2")), String(typeof pickMap.get)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|1|5|function" {
		t.Fatalf("TextContent(#out) = %q, want true|false|1|5|function", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Map bootstrap", got)
	}
}

func TestSessionBootstrapsRequestAnimationFrameFunctionCallback(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = document.getElementById("out"); window.requestAnimationFrame(function () { out.textContent = "done"; }, 0);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) before AdvanceTime error = %v", err)
	} else if got != "" {
		t.Fatalf("TextContent(#out) before AdvanceTime = %q, want empty", got)
	}
	if err := session.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after AdvanceTime error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) after AdvanceTime = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after requestAnimationFrame bootstrap", got)
	}
}

func TestSessionBootstrapsAsyncRequestAnimationFramePromiseContinuation(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>async function run() { const out = document.getElementById("out"); await new Promise((resolve) => { window.requestAnimationFrame(() => resolve()); }); out.textContent = "done"; } run()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) before AdvanceTime error = %v", err)
	} else if got != "" {
		t.Fatalf("TextContent(#out) before AdvanceTime = %q, want empty", got)
	}
	if err := session.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after AdvanceTime error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) after AdvanceTime = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after async requestAnimationFrame bootstrap", got)
	}
}

func TestSessionBootstrapsRequestAnimationFramePromiseResolution(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = document.getElementById("out"); new Promise((resolve) => { window.requestAnimationFrame(() => resolve("done")); }).then((value) => { out.textContent = value; });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) before AdvanceTime error = %v", err)
	} else if got != "" {
		t.Fatalf("TextContent(#out) before AdvanceTime = %q, want empty", got)
	}
	if err := session.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after AdvanceTime error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) after AdvanceTime = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after rAF promise resolution bootstrap", got)
	}
}

func TestSessionBootstrapsPromiseRejectionCatchChain(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>new Promise((resolve, reject) => { reject("boom"); }).catch(function (reason) { document.getElementById("out").textContent = reason; });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "boom" {
		t.Fatalf("TextContent(#out) = %q, want boom", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after promise rejection bootstrap", got)
	}
}

func TestSessionBootstrapsSetTimeoutFunctionCallback(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = document.getElementById("out"); window.setTimeout(function (value) { out.textContent = value; }, 1, "done");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) before AdvanceTime error = %v", err)
	} else if got != "" {
		t.Fatalf("TextContent(#out) before AdvanceTime = %q, want empty", got)
	}
	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after AdvanceTime error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) after AdvanceTime = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after setTimeout callback bootstrap", got)
	}
}

func TestSessionBootstrapsSetTimeoutFunctionCallbackCanUseLocalStorage(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>window.setTimeout(function () { localStorage.setItem("status", "done"); document.getElementById("out").textContent = localStorage.getItem("status"); }, 1);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after setTimeout localStorage bootstrap", got)
	}
}

func TestSessionBootstrapsSetTimeoutCallbackCanUseNestedStorageHelper(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>function saveJson(key, value) { localStorage.setItem(key, JSON.stringify(value)); } function schedulePersist() { window.setTimeout(() => { saveJson("status", { value: "done" }); document.getElementById("out").textContent = localStorage.getItem("status"); }, 1); } schedulePersist();</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"value":"done"}` {
		t.Fatalf("TextContent(#out) = %q, want %q", got, `{"value":"done"}`)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nested storage helper bootstrap", got)
	}
}

func TestSessionBootstrapsSetTimeoutCallbackCanUseClassList(t *testing.T) {
	const rawHTML = `<main><div id="toast" class="error"></div><div id="out"></div><script>window.setTimeout(() => { const toast = document.getElementById("toast"); toast.classList.add("hidden"); toast.classList.remove("error"); document.getElementById("out").textContent = [toast.classList.contains("hidden"), toast.classList.contains("error")].join("|"); }, 1);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false" {
		t.Fatalf("TextContent(#out) = %q, want true|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after classList timer bootstrap", got)
	}
}

func TestSessionBootstrapsClickHandlerCanUseLocalStorage(t *testing.T) {
	const rawHTML = `<main><button id="btn" type="button">go</button><div id="out"></div><script>document.getElementById("btn").addEventListener("click", () => { localStorage.setItem("status", "done"); document.getElementById("out").textContent = localStorage.getItem("status"); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after click localStorage bootstrap", got)
	}
}

func TestSessionBootstrapsClickHandlerCanUseNestedStorageHelper(t *testing.T) {
	const rawHTML = `<main><button id="btn" type="button">go</button><div id="out"></div><script>function saveJson(key, value) { localStorage.setItem(key, JSON.stringify(value)); } function renderAll() { saveJson("status", { value: "done" }); document.getElementById("out").textContent = localStorage.getItem("status"); } document.getElementById("btn").addEventListener("click", () => { renderAll(); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"value":"done"}` {
		t.Fatalf("TextContent(#out) = %q, want %q", got, `{"value":"done"}`)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nested click storage bootstrap", got)
	}
}

func TestSessionBootstrapsClickHandlerCanUseMapForEachHostBindings(t *testing.T) {
	const rawHTML = `<main><button id="btn" type="button">go</button><div id="out"></div><script>document.getElementById("btn").addEventListener("click", () => { const map = new Map([["alpha", 1], ["beta", 2]]); let seen = ""; map.forEach(function (value, key, mapObject) { seen = seen + (seen === "" ? "" : "|") + key + ":" + value + ":" + mapObject.get(key); document.getElementById("out").textContent = seen; }); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha:1:1|beta:2:2" {
		t.Fatalf("TextContent(#out) = %q, want alpha:1:1|beta:2:2", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after click Map.forEach host bootstrap", got)
	}
}

func TestSessionBootstrapsClickHandlerWithTimersCanUseStorageAndClassList(t *testing.T) {
	const rawHTML = `<main><button id="btn" type="button">go</button><div id="toast" class="error"></div><div id="out"></div><script>let saveTimer = null; let toastTimer = null; function saveJson(key, value) { localStorage.setItem(key, JSON.stringify(value)); } function renderAll() { saveJson("ui", { mode: "annual" }); document.getElementById("out").textContent = localStorage.getItem("ui"); } function schedulePersist() { clearTimeout(saveTimer); saveTimer = window.setTimeout(() => { saveJson("workspace", { foo: "bar" }); }, 1); } function showToast(message) { clearTimeout(toastTimer); const toast = document.getElementById("toast"); toast.textContent = message; toast.classList.remove("hidden", "error"); toastTimer = window.setTimeout(() => { toast.classList.add("hidden"); toast.classList.remove("error"); }, 1); } document.getElementById("btn").addEventListener("click", () => { renderAll(); schedulePersist(); showToast("done"); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	if err := session.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"mode":"annual"}` {
		t.Fatalf("TextContent(#out) = %q, want %q", got, `{"mode":"annual"}`)
	}
	if got, err := session.TextContent("#toast"); err != nil {
		t.Fatalf("TextContent(#toast) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#toast) = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after click timer bootstrap", got)
	}
}

func TestSessionBootstrapsFunctionOwnPropertyAssignmentOnLocalFunctions(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>function showToast(message) { showToast._timer = message; document.getElementById("out").textContent = showToast._timer; } showToast("done");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after function property bootstrap", got)
	}
}

func TestSessionBootstrapsPendingPromiseAwaitContinuation(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = document.getElementById("out"); let resolveRun; const promise = new Promise((resolve) => { resolveRun = resolve; }); async function run() { await promise; out.textContent = "done"; } run(); resolveRun("ready");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after bootstrap error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) after bootstrap = %q, want done", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after pending promise await bootstrap", got)
	}
}

func TestSessionBootstrapsPendingPromiseResolveCallbackSideEffects(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = document.getElementById("out"); let resolveRun; const promise = new Promise((resolve) => { resolveRun = (value) => { out.textContent = "resolved:" + value; resolve(value); }; }); async function run() { await promise; out.textContent = out.textContent + "|done"; } run(); resolveRun("ready");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after bootstrap error = %v", err)
	} else if got != "resolved:ready" && got != "resolved:ready|done" {
		t.Fatalf("TextContent(#out) after bootstrap = %q, want resolve callback side effect", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after pending promise resolve side effects bootstrap", got)
	}
}

func TestSessionBootstrapsPendingPromiseResolveOrdering(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const out = document.getElementById("out"); let resolveRun; const promise = new Promise((resolve) => { resolveRun = (value) => { out.textContent += "resolve-start|"; resolve(value); out.textContent += "resolve-end|"; }; }); async function run() { out.textContent += "await-start|"; await promise; out.textContent += "await-end|"; } run(); resolveRun("ready");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after bootstrap error = %v", err)
	} else if got != "await-start|resolve-start|resolve-end|" && got != "await-start|resolve-start|await-end|resolve-end|" {
		t.Fatalf("TextContent(#out) after bootstrap = %q, want promise resolve ordering trace", got)
	} else {
		t.Logf("promise resolve ordering trace = %q", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after pending promise resolve ordering bootstrap", got)
	}
}

func TestSessionBootstrapsWindowOpenPopupStub(t *testing.T) {
	const rawHTML = `<main><button id="go">go</button><div id="out"></div><script>document.getElementById("go").addEventListener("click", () => { const win = window.open("", "_blank", "noopener,noreferrer", "ignored"); win.document.open(); win.document.write("<p>print view</p>"); win.document.close(); win.focus(); win.print(); document.getElementById("out").textContent = [String(win.closed), String(win.opener === null), String(win.document.readyState)].join("|"); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) before click error = %v", err)
	} else if got != "" {
		t.Fatalf("TextContent(#out) before click = %q, want empty", got)
	}
	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after click error = %v", err)
	} else if got != "false|true|complete" {
		t.Fatalf("TextContent(#out) after click = %q, want false|true|complete", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after window.open bootstrap", got)
	}
	if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "" {
		t.Fatalf("OpenCalls() = %#v, want one popup open call", got)
	}
	if got := session.PrintCalls(); len(got) != 1 {
		t.Fatalf("PrintCalls() = %#v, want one print call", got)
	}
}

func TestSessionBootstrapsBareOpenPopupStub(t *testing.T) {
	const rawHTML = `<main><button id="go">go</button><div id="out"></div><script>document.getElementById("go").addEventListener("click", () => { const win = open("", "_blank", "noopener,noreferrer", "ignored"); win.document.open(); win.document.write("<p>print view</p>"); win.document.close(); win.focus(); win.print(); document.getElementById("out").textContent = [String(win.closed), String(win.opener === null), String(win.document.readyState)].join("|"); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#go"); err != nil {
		t.Fatalf("Click(#go) error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after click error = %v", err)
	} else if got != "false|true|complete" {
		t.Fatalf("TextContent(#out) after click = %q, want false|true|complete", got)
	}
	if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "" {
		t.Fatalf("OpenCalls() = %#v, want one popup open call", got)
	}
	if got := session.PrintCalls(); len(got) != 1 {
		t.Fatalf("PrintCalls() = %#v, want one print call", got)
	}
}

func TestSessionBootstrapsOpenPopupStubWithoutArguments(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "window.open",
			html: `<main><button id="go">go</button><div id="out"></div><script>document.getElementById("go").addEventListener("click", () => { const win = window.open(); win.document.open(); win.document.write("<p>print view</p>"); win.document.close(); win.focus(); win.print(); document.getElementById("out").textContent = [String(win.closed), String(win.opener === null), String(win.document.readyState)].join("|"); });</script></main>`,
		},
		{
			name: "bare open",
			html: `<main><button id="go">go</button><div id="out"></div><script>document.getElementById("go").addEventListener("click", () => { const win = open(); win.document.open(); win.document.write("<p>print view</p>"); win.document.close(); win.focus(); win.print(); document.getElementById("out").textContent = [String(win.closed), String(win.opener === null), String(win.document.readyState)].join("|"); });</script></main>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(SessionConfig{HTML: tc.html})
			if _, err := session.ensureDOM(); err != nil {
				t.Fatalf("ensureDOM() error = %v", err)
			}

			if err := session.Click("#go"); err != nil {
				t.Fatalf("Click(#go) error = %v", err)
			}
			if got, err := session.TextContent("#out"); err != nil {
				t.Fatalf("TextContent(#out) after click error = %v", err)
			} else if got != "false|true|complete" {
				t.Fatalf("TextContent(#out) after click = %q, want false|true|complete", got)
			}
			if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "" {
				t.Fatalf("OpenCalls() = %#v, want one popup open call", got)
			}
			if got := session.PrintCalls(); len(got) != 1 {
				t.Fatalf("PrintCalls() = %#v, want one print call", got)
			}
		})
	}
}

func TestSessionReportsOpenFailureThroughBrowserGlobalsWithoutArguments(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "window.open",
			html: `<main><script>window.open()</script></main>`,
		},
		{
			name: "bare open",
			html: `<main><script>open()</script></main>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(SessionConfig{
				URL:         "https://example.test/start",
				HTML:        tc.html,
				OpenFailure: "open blocked",
			})
			if _, err := session.ensureDOM(); err == nil {
				t.Fatalf("ensureDOM() error = nil, want open failure")
			} else if !strings.Contains(err.Error(), "open blocked") {
				t.Fatalf("ensureDOM() error = %v, want open blocked message", err)
			}
			if got, want := session.URL(), "https://example.test/start"; got != want {
				t.Fatalf("URL() after rejected open = %q, want %q", got, want)
			}
			if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "" {
				t.Fatalf("OpenCalls() after rejected open = %#v, want one blank popup call", got)
			}
			if got := session.DOMError(); got == "" || !strings.Contains(got, "open blocked") {
				t.Fatalf("DOMError() = %q, want open blocked message", got)
			}
		})
	}
}

func TestSessionRejectsWindowOpenSymbolThroughBrowserGlobals(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "url",
			html: `<main><script>window.open(Symbol("token"))</script></main>`,
		},
		{
			name: "target",
			html: `<main><script>window.open("", Symbol("token"))</script></main>`,
		},
		{
			name: "features",
			html: `<main><script>window.open("", "", Symbol("token"))</script></main>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(SessionConfig{
				URL:  "https://example.test/start",
				HTML: tc.html,
			})
			if _, err := session.ensureDOM(); err == nil {
				t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
			} else if !strings.Contains(err.Error(), "Cannot convert a Symbol value to a string") {
				t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
			}
			if got, want := session.URL(), "https://example.test/start"; got != want {
				t.Fatalf("URL() after rejected window.open Symbol input = %q, want %q", got, want)
			}
			if got := session.OpenCalls(); len(got) != 0 {
				t.Fatalf("OpenCalls() after rejected window.open Symbol input = %#v, want empty", got)
			}
		})
	}
}

func TestSessionReportsWindowOpenFailureThroughBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><script>window.open("https://example.test/popup", "_blank", "noopener,noreferrer")</script></main>`

	session := NewSession(SessionConfig{
		URL:         "https://example.test/start",
		HTML:        rawHTML,
		OpenFailure: "open blocked",
	})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want open failure")
	} else if !strings.Contains(err.Error(), "open blocked") {
		t.Fatalf("ensureDOM() error = %v, want open blocked message", err)
	}
	if got, want := session.URL(), "https://example.test/start"; got != want {
		t.Fatalf("URL() after rejected window.open = %q, want %q", got, want)
	}
	if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "https://example.test/popup" {
		t.Fatalf("OpenCalls() after rejected window.open = %#v, want one popup call", got)
	}
	if got := session.DOMError(); got == "" || !strings.Contains(got, "open blocked") {
		t.Fatalf("DOMError() = %q, want open blocked message", got)
	}
}

func TestSessionRejectsBareOpenSymbolThroughBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><script>open(Symbol("token"))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
	} else if !strings.Contains(err.Error(), "Cannot convert a Symbol value to a string") {
		t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
	}
	if got := session.OpenCalls(); len(got) != 0 {
		t.Fatalf("OpenCalls() after rejected bare open Symbol input = %#v, want empty", got)
	}
}

func TestSessionReportsBareOpenFailureThroughBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><script>open("https://example.test/popup", "_blank", "noopener,noreferrer")</script></main>`

	session := NewSession(SessionConfig{
		URL:         "https://example.test/start",
		HTML:        rawHTML,
		OpenFailure: "open blocked",
	})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want open failure")
	} else if !strings.Contains(err.Error(), "open blocked") {
		t.Fatalf("ensureDOM() error = %v, want open blocked message", err)
	}
	if got, want := session.URL(), "https://example.test/start"; got != want {
		t.Fatalf("URL() after rejected bare open = %q, want %q", got, want)
	}
	if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "https://example.test/popup" {
		t.Fatalf("OpenCalls() after rejected bare open = %#v, want one popup call", got)
	}
	if got := session.DOMError(); got == "" || !strings.Contains(got, "open blocked") {
		t.Fatalf("DOMError() = %q, want open blocked message", got)
	}
}

func TestSessionRejectsRequestAnimationFrameNonCallableCallback(t *testing.T) {
	const rawHTML = `<main><script>window.requestAnimationFrame("noop")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want requestAnimationFrame callback type error")
	} else if !strings.Contains(err.Error(), "requestAnimationFrame callback must be callable") {
		t.Fatalf("ensureDOM() error = %v, want requestAnimationFrame callback type error", err)
	}
}

func TestSessionBootstrapsDisabledAndSelectedElementProperties(t *testing.T) {
	const rawHTML = `<main><button id="run" type="button">Run</button><select id="s"><option value="g">g</option><option value="kg" selected>kg</option></select><p id="out"></p><script>const run = document.getElementById("run"); const select = document.getElementById("s"); const option = document.createElement("option"); option.value = "ml"; option.textContent = "ml"; option.selected = true; select.appendChild(option); const before = select.selectedIndex; select.selectedIndex = 0; const after = select.selectedIndex; const selected = select.options[after]; run.disabled = true; document.getElementById("out").textContent = [String(run.disabled), String(option.selected), String(before), String(after), select.value, selected ? selected.textContent : ""].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|1|0|g|g" {
		t.Fatalf("TextContent(#out) = %q, want true|false|1|0|g|g", got)
	}
	if got, ok, err := session.GetAttribute("#run", "disabled"); err != nil {
		t.Fatalf("GetAttribute(#run, disabled) error = %v", err)
	} else if !ok || got != "" {
		t.Fatalf("GetAttribute(#run, disabled) = (%q, %v), want (\"\", true)", got, ok)
	}
	if got, ok, err := session.GetAttribute("#s option", "selected"); err != nil {
		t.Fatalf("GetAttribute(#s option, selected) error = %v", err)
	} else if !ok || got != "" {
		t.Fatalf("GetAttribute(#s option, selected) = (%q, %v), want (\"\", true)", got, ok)
	}
}

func TestSessionBootstrapsDocumentBodyFallbackWithoutBodyElement(t *testing.T) {
	const rawHTML = `<div id="out"></div><script>document.body.setAttribute("data-body", "yes")</script>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, ok, err := session.GetAttribute("#out", "data-body"); err != nil {
		t.Fatalf("GetAttribute(#out, data-body) error = %v", err)
	} else if !ok || got != "yes" {
		t.Fatalf("GetAttribute(#out, data-body) = (%q, %v), want (yes, true)", got, ok)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after document.body fallback bootstrap", got)
	}
}

func TestSessionBootstrapsElementTextContentAssignment(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [CSS.escape(), CSS.escape("0", "ignored"), CSS.escape("alpha-beta", "ignored")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `undefined|\30 |alpha-beta` {
		t.Fatalf("TextContent(#out) = %q, want %q", got, `undefined|\30 |alpha-beta`)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after textContent assignment bootstrap", got)
	}
}

func TestSessionRejectsCSSEscapeSymbolInput(t *testing.T) {
	const rawHTML = `<main><script>CSS.escape(Symbol("token"))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
	} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
		t.Fatalf("ensureDOM() error = %#v, want runtime script error", err)
	} else if !strings.Contains(scriptErr.Message, "Cannot convert a Symbol value to a string") {
		t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
	}
}

func TestSessionRejectsLocationAssignReplaceSymbolInput(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "assign",
			html: `<main><script>window.location.assign(Symbol("token"))</script></main>`,
		},
		{
			name: "replace",
			html: `<main><script>window.location.replace(Symbol("token"))</script></main>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(SessionConfig{
				URL:  "https://example.test/start",
				HTML: tc.html,
			})

			if _, err := session.ensureDOM(); err == nil {
				t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
			} else if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
				t.Fatalf("ensureDOM() error = %#v, want runtime script error", err)
			} else if !strings.Contains(scriptErr.Message, "Cannot convert a Symbol value to a string") {
				t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
			}
			if got, want := session.URL(), "https://example.test/start"; got != want {
				t.Fatalf("URL() after rejected location %s symbol input = %q, want %q", tc.name, got, want)
			}
			if got := session.Registry().Location().Navigations(); len(got) != 0 {
				t.Fatalf("Location().Navigations() after rejected location %s symbol input = %#v, want empty", tc.name, got)
			}
		})
	}
}

func TestSessionBootstrapsClosestSelectorVariable(t *testing.T) {
	const rawHTML = `<main><section class="card"><button id="child">open</button></section><p id="out"></p><script>const child = document.getElementById("child"); const selector = ".card"; const matched = child.closest(selector); document.getElementById("out").textContent = matched ? matched.tagName : "none";</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "SECTION" {
		t.Fatalf("TextContent(#out) = %q, want SECTION", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after closest selector bootstrap", got)
	}
}

func TestSessionBootstrapsIntlNumberFormatMaximumSignificantDigits(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = new Intl.NumberFormat("en-US", { maximumSignificantDigits: 4 }).format(26.72665916760405)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "26.73" {
		t.Fatalf("TextContent(#out) = %q, want 26.73", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.NumberFormat significant digits bootstrap", got)
	}
}

func TestSessionBootstrapsIntlNumberFormatMinimumFractionDigits(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = new Intl.NumberFormat("en-US", { minimumFractionDigits: 1, maximumFractionDigits: 1 }).format(12)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "12.0" {
		t.Fatalf("TextContent(#out) = %q, want 12.0", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.NumberFormat minimum fraction bootstrap", got)
	}
}

func TestSessionBootstrapsNumberToLocaleString(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = (600).toLocaleString("en-US", { minimumFractionDigits: 1, maximumFractionDigits: 1 })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "600.0" {
		t.Fatalf("TextContent(#out) = %q, want 600.0", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Number.toLocaleString bootstrap", got)
	}
}

func TestSessionBootstrapsIntlNumberFormatResolvedOptions(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const resolved = new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY", minimumFractionDigits: 0, maximumFractionDigits: 0 }).resolvedOptions(); document.getElementById("out").textContent = [resolved.locale, resolved.style, resolved.currency, String(resolved.minimumFractionDigits), String(resolved.maximumFractionDigits), String(resolved.useGrouping)].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ja-JP|currency|JPY|0|0|true" {
		t.Fatalf("TextContent(#out) = %q, want ja-JP|currency|JPY|0|0|true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.NumberFormat resolvedOptions bootstrap", got)
	}
}

func TestSessionBootstrapsIntlNumberFormatFormatToParts(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const parts = new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY", maximumFractionDigits: 0 }).formatToParts(-1200); document.getElementById("out").textContent = parts.map((part) => part.type + ":" + part.value).join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "minusSign:-|currency:￥|integer:1|group:,|integer:200" {
		t.Fatalf("TextContent(#out) = %q, want minusSign:-|currency:￥|integer:1|group:,|integer:200", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.NumberFormat formatToParts bootstrap", got)
	}
}

func TestSessionBootstrapsIntlNumberFormatSupportedLocalesOf(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = Intl.NumberFormat.supportedLocalesOf(["en-US", "", "sv", "en-US"]).join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "en-US|sv" {
		t.Fatalf("TextContent(#out) = %q, want en-US|sv", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.NumberFormat.supportedLocalesOf bootstrap", got)
	}
}

func TestSessionBootstrapsIntlDateTimeFormatResolvedOptions(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const resolved = new Intl.DateTimeFormat("en-US-u-nu-latn", { timeZone: "America/Chicago", hour12: false }).resolvedOptions(); document.getElementById("out").textContent = [resolved.locale, resolved.timeZone, String(resolved.hour12), resolved.calendar, resolved.numberingSystem, resolved.year, resolved.month, resolved.day, resolved.hour, resolved.minute, resolved.second].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "en-US-u-nu-latn|America/Chicago|false|gregory|latn|numeric|numeric|numeric|numeric|numeric|numeric" {
		t.Fatalf("TextContent(#out) = %q, want en-US-u-nu-latn|America/Chicago|false|gregory|latn|numeric|numeric|numeric|numeric|numeric|numeric", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.DateTimeFormat resolvedOptions bootstrap", got)
	}
}

func TestSessionBootstrapsIntlDateTimeFormatSupportedLocalesOf(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = Intl.DateTimeFormat.supportedLocalesOf(["en-US", "", "sv", "en-US"]).join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "en-US|sv" {
		t.Fatalf("TextContent(#out) = %q, want en-US|sv", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.DateTimeFormat.supportedLocalesOf bootstrap", got)
	}
}

func TestSessionBootstrapsIntlDateTimeFormatFormatRangeAndFormatRangeToParts(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>(() => { const formatter = new Intl.DateTimeFormat("en-US-u-nu-latn", { timeZone: "America/Chicago", hour12: false }); const start = new Date(Date.UTC(2026, 0, 21, 8, 45, 0, 0)); const end = new Date(Date.UTC(2026, 0, 21, 9, 15, 0, 0)); const range = formatter.formatRange(start, end); const parts = formatter.formatRangeToParts(start, end); document.getElementById("out").textContent = [range, parts.map((part) => part.value).join(""), parts[0].source, parts.find((part) => part.type === "literal" && part.value === " – ").source, parts[parts.length - 1].source].join("|"); })();</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "01/21/2026, 02:45 – 01/21/2026, 03:15|01/21/2026, 02:45 – 01/21/2026, 03:15|startRange|shared|endRange" {
		t.Fatalf("TextContent(#out) = %q, want 01/21/2026, 02:45 – 01/21/2026, 03:15|01/21/2026, 02:45 – 01/21/2026, 03:15|startRange|shared|endRange", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.DateTimeFormat formatRange bootstrap", got)
	}
}

func TestSessionBootstrapsIntlCurrencyFormattingWithMapLookup(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>(() => { const state = { currency: "JPY", decimalOverride: "auto", cost: "", adoptedPrice: 1200, }; const currencyMap = new Map([["JPY", { code: "JPY", locale: "ja-JP", decimals: 0 }]]); function getDecimals() { if (state.decimalOverride !== "auto") { return Number(state.decimalOverride); } const meta = currencyMap.get(state.currency); return meta && meta.decimals != null ? meta.decimals : 2; } function formatMoney(value) { const meta = currencyMap.get(state.currency) || { code: state.currency, locale: "en-US", decimals: 2, }; const digits = getDecimals(); return new Intl.NumberFormat(meta.locale, { style: "currency", currency: meta.code, minimumFractionDigits: digits, maximumFractionDigits: digits, }).format(value); } document.getElementById("out").textContent = formatMoney(1200); })();</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "￥1,200" {
		t.Fatalf("TextContent(#out) = %q, want ￥1,200", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl currency bootstrap", got)
	}
}

func TestSessionRejectsIntlNumberFormatMaximumSignificantDigitsTypeMismatch(t *testing.T) {
	const rawHTML = `<main><script>new Intl.NumberFormat("en-US", { maximumSignificantDigits: "4" }).format(26.72665916760405)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Intl.NumberFormat options type error")
	} else if !strings.Contains(err.Error(), "maximumSignificantDigits must be numeric") {
		t.Fatalf("ensureDOM() error = %v, want maximumSignificantDigits type error", err)
	}
}

func TestSessionBootstrapsIntlCollatorNumericAndSwedishSorting(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const values = ["item 10", "item 2", "item 1"]; const collator = new Intl.Collator("en", { usage: "sort", numeric: true, sensitivity: "variant" }); const asc = values.slice().sort(collator.compare).join(","); const desc = values.slice().sort((left, right) => collator.compare(right, left)).join(","); const numeric = String(collator.resolvedOptions().numeric); const sv = new Intl.Collator("sv", { usage: "sort", sensitivity: "variant" }); const swedish = ["Öga", "Zebra", "Äpple", "Ål"].slice().sort(sv.compare).join(","); document.getElementById("out").textContent = [asc, desc, numeric, swedish].join("|");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "item 1,item 2,item 10|item 10,item 2,item 1|true|Zebra,Ål,Äpple,Öga" {
		t.Fatalf("TextContent(#out) = %q, want item 1,item 2,item 10|item 10,item 2,item 1|true|Zebra,Ål,Äpple,Öga", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.Collator bootstrap", got)
	}
}

func TestSessionBootstrapsIntlCollatorSupportedLocalesOf(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = Intl.Collator.supportedLocalesOf(["sv", "", "en-US", "sv"]).join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "sv|en-US" {
		t.Fatalf("TextContent(#out) = %q, want sv|en-US", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.Collator.supportedLocalesOf bootstrap", got)
	}
}

func TestSessionBootstrapsReassignedIntlNumberFormat(t *testing.T) {
	const rawHTML = `<main><pre id="out"></pre><script>Intl = { NumberFormat: function () { throw new Error("forced Intl failure"); } }; window.Intl = Intl; if (window.Intl !== Intl) { throw new Error("window.Intl override mismatch"); } Intl.NumberFormat = function () { throw new Error("forced Intl failure"); }; function formatIndex(value, lang, minimumIntegerDigits) { const safeValue = Math.max(0, Number(value) || 0); try { return new Intl.NumberFormat(lang, { useGrouping: false, minimumIntegerDigits, maximumFractionDigits: 0 }).format(safeValue); } catch (error) { const digits = String(Math.trunc(safeValue)); return digits.padStart(minimumIntegerDigits, "0"); } } const lines = ["A", "B"].map((label, index) => { return "[" + formatIndex(index + 1, "ar-EG", 1) + "] " + label; }); document.getElementById("out").textContent = lines.join("\n");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "[1] A\n[2] B" {
		t.Fatalf("TextContent(#out) = %q, want [1] A\\n[2] B", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after reassigned Intl bootstrap", got)
	}
}

func TestSessionWriteHTMLResetsIntlOverride(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main><div id="out"></div><script>window.Intl = { NumberFormat: function () { return { format: function () { return "override"; } }; } }; document.getElementById("out").textContent = new Intl.NumberFormat("en-US", {}).format(1)</script></main>`})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "override" {
		t.Fatalf("TextContent(#out) = %q, want override", got)
	}

	if err := session.WriteHTML(`<main><div id="out"></div><script>document.getElementById("out").textContent = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23)</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after WriteHTML error = %v", err)
	} else if got != "1.23" {
		t.Fatalf("TextContent(#out) after WriteHTML = %q, want 1.23", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl override reset", got)
	}
}

func TestSessionBootstrapsWindowCryptoStubAndAwaitedDigest(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><div id="err"></div><div id="meta"></div><script>window.crypto = { subtle: { digest: function (_alg, _data) { return Promise.resolve(new Uint8Array([65, 66, 67]).buffer); } } }; (async function () { const digest = await crypto.subtle.digest("SHA-256", new Uint8Array([1, 2, 3])); document.getElementById("meta").textContent = typeof digest + ":" + String(digest && digest.byteLength); document.getElementById("out").textContent = Array.from(new Uint8Array(digest)).join(","); })().catch(function (error) { document.getElementById("err").textContent = error && error.message ? error.message : String(error); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#err"); err != nil {
		t.Fatalf("TextContent(#err) error = %v", err)
	} else if got != "" {
		t.Fatalf("TextContent(#err) = %q, want empty", got)
	}
	if got, err := session.TextContent("#meta"); err != nil {
		t.Fatalf("TextContent(#meta) error = %v", err)
	} else if got != "object:3" {
		t.Fatalf("TextContent(#meta) = %q, want object:3", got)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "65,66,67" {
		t.Fatalf("TextContent(#out) = %q, want 65,66,67", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after crypto bootstrap", got)
	}
}

func TestSessionWriteHTMLResetsWindowCryptoOverride(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main><div id="out"></div><script>window.crypto = { marker: "set" }; document.getElementById("out").textContent = typeof crypto + ":" + crypto.marker</script></main>`})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "object:set" {
		t.Fatalf("TextContent(#out) = %q, want object:set", got)
	}

	if err := session.WriteHTML(`<main><div id="out"></div><script>window.crypto = { marker: "reset" }; document.getElementById("out").textContent = typeof crypto + ":" + crypto.marker</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after WriteHTML error = %v", err)
	} else if got != "object:reset" {
		t.Fatalf("TextContent(#out) after WriteHTML = %q, want object:reset", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after crypto override reset", got)
	}
}

func TestSessionRejectsWindowDocumentAssignmentOnBootstrap(t *testing.T) {
	const rawHTML = `<main><script>window.document = {}</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported window.document assignment")
	} else if !strings.Contains(err.Error(), `assignment to "document"`) {
		t.Fatalf("ensureDOM() error = %v, want document assignment error", err)
	}
}

func TestSessionBootstrapsElementTextContentAssignmentWithRegularExpressionCommaLiteral(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = "1,234".replace(/,/g, "")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1234" {
		t.Fatalf("TextContent(#out) = %q, want 1234", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after regex comma literal bootstrap", got)
	}
}

func TestSessionBootstrapsArrayIndexOfAndLastIndexOf(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const values = ["alpha", "beta", "gamma", "beta"]; document.getElementById("out").textContent = [String(values.indexOf("beta")), String(values.indexOf("beta", 2)), String(values.indexOf("beta", -2)), String(values.lastIndexOf("beta")), String(values.lastIndexOf("beta", 2)), String(values.lastIndexOf("beta", -3))].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "1|3|3|3|1|1" {
		t.Fatalf("TextContent(#out) = %q, want 1|3|3|3|1|1", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.indexOf/lastIndexOf bootstrap", got)
	}
}

func TestSessionBootstrapsArrayEvery(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [1, 2, 3].every((value) => value > 0) ? "true" : "false"</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true" {
		t.Fatalf("TextContent(#out) = %q, want true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.every bootstrap", got)
	}
}

func TestSessionBootstrapsRegularExpressionLiteralContainingQuoteCharacters(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const text = 'a"b'; const other = "c'd"; document.getElementById("out").textContent = text.replace(/\"/g, "&quot;") + "|" + other.replace(/'/g, "&#39;")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "a&quot;b|c&#39;d" {
		t.Fatalf("TextContent(#out) = %q, want a&quot;b|c&#39;d", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after quoted-regex bootstrap", got)
	}
}

func TestSessionBootstrapsStringReplaceCallbackAndFromCharCode(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = "１２３".replace(/[\uFF10-\uFF19]/g, (s) => String.fromCharCode(s.charCodeAt(0) - 65248))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "123" {
		t.Fatalf("TextContent(#out) = %q, want 123", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after String.replace callback bootstrap", got)
	}
}

func TestSessionBootstrapsStringReplaceCallbackUsesRuneOffsets(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = "あ1い2".replace(/([0-9])/g, (match, digit, offset, input) => digit + ":" + offset)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "あ1:1い2:3" {
		t.Fatalf("TextContent(#out) = %q, want あ1:1い2:3", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after String.replace callback rune-offset bootstrap", got)
	}
}

func TestSessionBootstrapsStringMatchNullGuardOnEmptyString(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const runs = "".match(/ {2,}/g); document.getElementById("out").textContent = String(runs ? runs.length : 0)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "0" {
		t.Fatalf("TextContent(#out) = %q, want 0", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after String.match null-guard bootstrap", got)
	}
}

func TestSessionRejectsMalformedQuotedRegularExpressionLiteral(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const text = 'a"b'; document.getElementById("out").textContent = text.replace(/\"g, "&quot;")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want malformed regular expression failure")
	}
	if got := session.DOMError(); !strings.Contains(got, "unterminated regular expression literal") {
		t.Fatalf("DOMError() = %q, want malformed regular expression failure text", got)
	}
}

func TestSessionBootstrapsTemplateLiteralBodyWithQuotedRegexCharacters(t *testing.T) {
	const rawHTML = "<main><div id=\"result\"></div><script>" +
		"const rows = [{ sku: \"A\", qty: 1 }, { sku: \"B\", qty: 2 }];" +
		"const rendered = rows.map((row, index) => `<tr data-idx=\"${index}\">" +
		"<td>${String(row.sku || \"\").replace(/\\\"/g, \"&quot;\")}</td>" +
		"<td>${String(row.qty || \"\").replace(/\\\"/g, \"&quot;\")}</td>" +
		"</tr>`).join(\"\");" +
		"document.getElementById(\"result\").textContent = rendered.includes('data-idx=\"1\"') ? \"ok\" : \"ng\";" +
		"</script></main>"

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#result"); err != nil {
		t.Fatalf("TextContent(#result) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#result) = %q, want ok", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after quoted-regex template literal bootstrap", got)
	}
}

func TestSessionBootstrapsNestedTemplateLiteralInterpolation(t *testing.T) {
	const rawHTML = "<main><div id=\"out\"></div><script>const title = `${true ? ` / ${1}` : \"\"}`; document.getElementById(\"out\").textContent = title;</script></main>"

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != " / 1" {
		t.Fatalf("TextContent(#out) = %q, want ` / 1`", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nested template literal bootstrap", got)
	}
}

func TestSessionBootstrapsNestedTemplateLiteralInArrowAndForHeaders(t *testing.T) {
	const rawHTML = "<main><div id=\"out\"></div><script>" +
		"let render = () => " + "`" + "arrow${" + "`" + "body;value" + "`" + "}" + "`" + ";" +
		"for (let value = " + "`" + "go${" + "`" + "now;later" + "`" + "}" + "`" + "; value; ) { document.getElementById(\"out\").textContent = render() + \"|\" + value; break }" +
		"</script></main>"

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "arrowbody;value|gonow;later" {
		t.Fatalf("TextContent(#out) = %q, want arrowbody;value|gonow;later", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nested template literal arrow/for header bootstrap", got)
	}
}

func TestSessionBootstrapsNestedTemplateLiteralScannerBoundaries(t *testing.T) {
	const rawHTML = "<main><div id=\"out\"></div><script>" +
		"class Box { label = " + "`" + "outer${" + "`" + "inner" + "`" + "}" + "`" + "; read() { return this.label } }" +
		"; const pick = (value = " + "`" + "default${" + "`" + "value" + "`" + "}" + "`" + ") => value;" +
		"const obj = { value: \"done\" }; let status = \"\";" +
		"switch (" + "`" + "state${" + "`" + "1" + "`" + "}" + "`" + ") {" +
		"case " + "`" + "state${" + "`" + "1" + "`" + "}" + "`" + ":" +
		"status = " + "`" + "hit${" + "`" + "!" + "`" + "}" + "`" + "; break;" +
		"default: status = \"miss\"; }" +
		"document.getElementById(\"out\").textContent = \"\".concat(" + "`" + "call${" + "`" + "arg" + "`" + "}" + "`" + ", \"|\", new Box().read(), \"|\", pick(), \"|\", obj[" + "`" + "val${" + "`" + "ue" + "`" + "}" + "`" + "], \"|\", status);" +
		"</script></main>"

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "callarg|outerinner|defaultvalue|done|hit!" {
		t.Fatalf("TextContent(#out) = %q, want callarg|outerinner|defaultvalue|done|hit!", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nested template literal scanner bootstrap", got)
	}
}

func TestSessionBootstrapsClickHandlerRegexLiteralBody(t *testing.T) {
	const rawHTML = `<main><textarea id="bulk">Field 1
Field 2</textarea><button id="apply" type="button">Apply</button><div id="out"></div><script>document.getElementById("apply").addEventListener("click", () => { const rows = String(document.getElementById("bulk").value || "").split(/\r?\n/).filter(Boolean); document.getElementById("out").textContent = String(rows.length); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#apply"); err != nil {
		t.Fatalf("Click(#apply) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "2" {
		t.Fatalf("TextContent(#out) = %q, want 2", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after click-handler regex bootstrap", got)
	}
}

func TestSessionBootstrapsClickHandlerObjectSubtractionComparator(t *testing.T) {
	const rawHTML = `<main><button id="run" type="button">Run</button><div id="out"></div><script>document.getElementById("run").addEventListener("click", () => { const rows = [{ start: { index: 2 }, end: { index: 4 } }, { start: { index: 1 }, end: { index: 3 } }]; rows.sort((left, right) => left.start - right.start || left.end - right.end); document.getElementById("out").textContent = "ok"; });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after click-handler object subtraction bootstrap", got)
	}
}

func TestSessionBootstrapsLogicalAndShortCircuitBeforeAddition(t *testing.T) {
	const rawHTML = `<main><button id="run" type="button">Run</button><div id="out"></div><script>document.getElementById("run").addEventListener("click", () => { document.getElementById("out").textContent = String(false && ({ kind: "box" } + 1)); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "false" {
		t.Fatalf("TextContent(#out) = %q, want false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after logical and short-circuit before addition bootstrap", got)
	}
}

func TestSessionBootstrapsLogicalOrShortCircuitBeforeAddition(t *testing.T) {
	const rawHTML = `<main><button id="run" type="button">Run</button><div id="out"></div><script>document.getElementById("run").addEventListener("click", () => { document.getElementById("out").textContent = String(true || ({ kind: "box" } + 1)); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true" {
		t.Fatalf("TextContent(#out) = %q, want true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after logical or short-circuit before addition bootstrap", got)
	}
}

func TestSessionBootstrapsElementFocusAndBlurMethods(t *testing.T) {
	const rawHTML = `<main><input id="a"><input id="b"><button id="btn">run</button><div id="out"></div><div id="state"></div><script>host:addEventListener("#a", "focus", 'host:insertAdjacentHTML("#out", "beforeend", "aF|")'); host:addEventListener("#a", "blur", 'host:insertAdjacentHTML("#out", "beforeend", "aB|")'); host:addEventListener("#b", "focus", 'host:insertAdjacentHTML("#out", "beforeend", "bF|")'); host:addEventListener("#b", "blur", 'host:insertAdjacentHTML("#out", "beforeend", "bB|")'); const a = document.getElementById("a"); const b = document.getElementById("b"); document.getElementById("btn").addEventListener("click", () => { a.focus(); b.focus(); b.blur(); document.getElementById("state").textContent = document.activeElement === null ? "none" : "active"; })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "aF|aB|bF|bB|" {
		t.Fatalf("TextContent(#out) = %q, want aF|aB|bF|bB|", got)
	}
	if got, err := session.TextContent("#state"); err != nil {
		t.Fatalf("TextContent(#state) error = %v", err)
	} else if got != "active" {
		t.Fatalf("TextContent(#state) = %q, want active", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after element focus/blur bootstrap", got)
	}
}

func TestSessionBootstrapsElementFocusAndBlurMethodsWithDomListeners(t *testing.T) {
	const rawHTML = `<main><input id="a"><input id="b"><button id="btn">run</button><div id="out"></div><div id="state"></div><script>const a = document.getElementById("a"); const b = document.getElementById("b"); let order = ""; a.addEventListener("focus", () => { order += "aF"; }); a.addEventListener("blur", () => { order += "aB"; }); b.addEventListener("focus", () => { order += "bF"; }); b.addEventListener("blur", () => { order += "bB"; }); document.getElementById("btn").addEventListener("click", () => { a.focus(); b.focus(); b.blur(); document.getElementById("state").textContent = order + ":" + (document.activeElement === null ? "none" : "active"); })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, err := session.TextContent("#state"); err != nil {
		t.Fatalf("TextContent(#state) error = %v", err)
	} else if got != "aFaBbFbB:active" {
		t.Fatalf("TextContent(#state) = %q, want aFaBbFbB:active", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after DOM focus/blur bootstrap", got)
	}
}

func TestSessionRejectsElementFocusAndBlurArguments(t *testing.T) {
	t.Run("focus", func(t *testing.T) {
		const rawHTML = `<main><input id="field"><script>document.getElementById("field").focus("now")</script></main>`

		session := NewSession(SessionConfig{HTML: rawHTML})
		if _, err := session.ensureDOM(); err == nil {
			t.Fatalf("ensureDOM() error = nil, want element.focus argument validation failure")
		}
		if got := session.DOMError(); !strings.Contains(got, "element.focus accepts no arguments") {
			t.Fatalf("DOMError() = %q, want element.focus argument validation failure text", got)
		}
	})

	t.Run("blur", func(t *testing.T) {
		const rawHTML = `<main><input id="field"><script>const field = document.getElementById("field"); field.focus(); field.blur("now")</script></main>`

		session := NewSession(SessionConfig{HTML: rawHTML})
		if _, err := session.ensureDOM(); err == nil {
			t.Fatalf("ensureDOM() error = nil, want element.blur argument validation failure")
		}
		if got := session.DOMError(); !strings.Contains(got, "element.blur accepts no arguments") {
			t.Fatalf("DOMError() = %q, want element.blur argument validation failure text", got)
		}
	})
}

func TestSessionBootstrapsOfflineNavigatorOnLineSeed(t *testing.T) {
	const rawHTML = `<main><div id="status"></div><script>host:setTextContent("#status", expr(navigator.onLine ? "online" : "offline"))</script></main>`

	session := NewSession(SessionConfig{
		HTML:               rawHTML,
		NavigatorOnLine:    false,
		HasNavigatorOnLine: true,
	})

	if got, ok := session.NavigatorOnLine(); !ok || got {
		t.Fatalf("NavigatorOnLine() = (%v, %v), want (false, true)", got, ok)
	}

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#status"); err != nil {
		t.Fatalf("TextContent(#status) error = %v", err)
	} else if got != "offline" {
		t.Fatalf("TextContent(#status) = %q, want offline", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after offline navigator bootstrap", got)
	}
}

func TestSessionRejectsObjectDefinePropertyOnNavigatorOnLine(t *testing.T) {
	const rawHTML = `<main><div id="status"></div><script>Object.defineProperty(window.navigator, "onLine", { configurable: true, get: function () { return false; } })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want unsupported Object.defineProperty failure")
	}

	if got := session.DOMError(); !strings.Contains(got, `unsupported browser surface "Object.defineProperty"`) {
		t.Fatalf("DOMError() = %q, want unsupported Object.defineProperty failure text", got)
	}
}

func TestSessionBootstrapsElementDatasetReadsWritesAndDeletes(t *testing.T) {
	const rawHTML = `<main><button id="mode" data-model-mode="grid" data-source-mode="weighted" data-round-mode="nearest"></button><div id="probe"></div><script>const button = document.querySelector("#mode"); const before = button.dataset.modelMode; button.dataset.roundMode = "floor"; const deleted = delete button.dataset.roundMode; host:setTextContent("#probe", expr(before + "|" + button.dataset.sourceMode + "|" + deleted + "|" + button.dataset.roundMode + "|" + button.hasAttribute("data-round-mode")))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "grid|weighted|true|undefined|false" {
		t.Fatalf("TextContent(#probe) = %q, want grid|weighted|true|undefined|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after dataset bootstrap", got)
	}
}

func TestSessionBootstrapsElementToggleAttribute(t *testing.T) {
	const rawHTML = `<main><button id="mode"></button><div id="probe"></div><script>const button = document.querySelector("#mode"); const first = button.toggleAttribute("data-active"); const second = button.toggleAttribute("data-active"); const third = button.toggleAttribute("data-active", "yes"); const fourth = button.toggleAttribute("data-active", 0); host:setTextContent("#probe", expr([first, second, third, fourth, button.hasAttribute("data-active")].join("|")))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "true|false|true|false|false" {
		t.Fatalf("TextContent(#probe) = %q, want true|false|true|false|false", got)
	}
	if got, ok, err := session.GetAttribute("#mode", "data-active"); err != nil {
		t.Fatalf("GetAttribute(#mode, data-active) error = %v", err)
	} else if ok || got != "" {
		t.Fatalf("GetAttribute(#mode, data-active) = (%q, %v), want (\"\", false)", got, ok)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after toggleAttribute bootstrap", got)
	}
}

func TestSessionRejectsElementToggleAttributeWithWrongArity(t *testing.T) {
	const rawHTML = `<main><button id="mode"></button><script>document.querySelector("#mode").toggleAttribute()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want toggleAttribute arity failure")
	}
	if got := session.DOMError(); !strings.Contains(got, "toggleAttribute") || !strings.Contains(got, "requires argument 1") {
		t.Fatalf("DOMError() = %q, want toggleAttribute arity failure text", got)
	}
}

func TestSessionBootstrapsElementHasAttributes(t *testing.T) {
	const rawHTML = `<main><button id="filled" data-active="yes"></button><button></button><div id="probe"></div><script>const filled = document.querySelector("#filled"); const empty = document.querySelectorAll("button")[1]; host:setTextContent("#probe", expr([filled.hasAttributes(), empty.hasAttributes()].join("|")))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "true|false" {
		t.Fatalf("TextContent(#probe) = %q, want true|false", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after hasAttributes bootstrap", got)
	}
}

func TestSessionBootstrapsElementGetAttributeNames(t *testing.T) {
	const rawHTML = `<main><div id="root" data-b="2" data-a="1"></div><div id="probe"></div><script>const root = document.querySelector("#root"); host:setTextContent("#probe", expr(root.getAttributeNames().join("|")))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "id|data-b|data-a" {
		t.Fatalf("TextContent(#probe) = %q, want id|data-b|data-a", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after getAttributeNames bootstrap", got)
	}
}

func TestSessionBootstrapsElementGetAttributeNode(t *testing.T) {
	const rawHTML = `<main><div id="root" data-b="2" data-a="1"></div><div id="probe"></div><script>const root = document.querySelector("#root"); const attr = root.getAttributeNode("data-a"); host:setTextContent("#probe", expr(attr.name + "=" + attr.value + "|" + String(root.getAttributeNode("missing") === null)))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if got != "data-a=1|true" {
		t.Fatalf("TextContent(#probe) = %q, want data-a=1|true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after getAttributeNode bootstrap", got)
	}
}

func TestSessionRejectsElementHasAttributesWithWrongArity(t *testing.T) {
	const rawHTML = `<main><button id="mode"></button><script>document.querySelector("#mode").hasAttributes(1)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want hasAttributes arity failure")
	}
	if got := session.DOMError(); !strings.Contains(got, "hasAttributes") || !strings.Contains(got, "no arguments") {
		t.Fatalf("DOMError() = %q, want hasAttributes arity failure text", got)
	}
}

func TestSessionRejectsElementGetAttributeNamesWithWrongArity(t *testing.T) {
	const rawHTML = `<main><button id="mode"></button><script>document.querySelector("#mode").getAttributeNames(1)</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want getAttributeNames arity failure")
	}
	if got := session.DOMError(); !strings.Contains(got, "getAttributeNames") || !strings.Contains(got, "no arguments") {
		t.Fatalf("DOMError() = %q, want getAttributeNames arity failure text", got)
	}
}

func TestSessionRejectsElementGetAttributeNodeWithWrongArity(t *testing.T) {
	const rawHTML = `<main><button id="mode"></button><script>document.querySelector("#mode").getAttributeNode()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want getAttributeNode arity failure")
	}
	if got := session.DOMError(); !strings.Contains(got, "getAttributeNode") || !strings.Contains(got, "requires argument 1") {
		t.Fatalf("DOMError() = %q, want getAttributeNode arity failure text", got)
	}
}

func TestSessionBootstrapsTemplateLocaleAndExternalWindowGlobals(t *testing.T) {
	const scriptURL = "https://cdn.jsdelivr.net/npm/lucide@latest/dist/umd/lucide.js"
	rawHTML := `<main><div id="locale"></div><div id="stamp"></div><div id="icons"></div><script src="` + scriptURL + `"></script><script>const locale = navigator.language || "en-US"; const stamp = new Intl.DateTimeFormat(locale, { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(new Date(1700000000000)); if (window.lucide && typeof window.lucide.createIcons === "function") { window.lucide.createIcons(); } host:setTextContent("#locale", expr(locale)); host:setTextContent("#stamp", expr(stamp))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.Registry().ExternalJS().RespondSource(scriptURL, `window.lucide = { createIcons: function () { document.getElementById("icons").textContent = "loaded"; } };`)
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#locale"); err != nil {
		t.Fatalf("TextContent(#locale) error = %v", err)
	} else if got != "en-US" {
		t.Fatalf("TextContent(#locale) = %q, want en-US", got)
	}
	if got, err := session.TextContent("#stamp"); err != nil {
		t.Fatalf("TextContent(#stamp) error = %v", err)
	} else if got != "11/14/2023, 10:13 PM" {
		t.Fatalf("TextContent(#stamp) = %q, want 11/14/2023, 10:13 PM", got)
	}
	if got, err := session.TextContent("#icons"); err != nil {
		t.Fatalf("TextContent(#icons) error = %v", err)
	} else if got != "loaded" {
		t.Fatalf("TextContent(#icons) = %q, want loaded", got)
	}
	if got := session.Registry().ExternalJS().Calls(); len(got) != 1 || got[0].URL != scriptURL {
		t.Fatalf("ExternalJS().Calls() = %#v, want one loaded script URL", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after optional browser globals bootstrap", got)
	}
}

func TestSessionBootstrapsWindowLucideFeatureDetectionWithoutSeededExternalScript(t *testing.T) {
	rawHTML := `<main><div id="icons"><i data-lucide="arrow-left-right"></i></div><div id="status"></div><script>document.getElementById("status").textContent = window.lucide ? "loaded" : "skipped"; if (window.lucide && typeof window.lucide.createIcons === "function") { window.lucide.createIcons(); }</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#status"); err != nil {
		t.Fatalf("TextContent(#status) error = %v", err)
	} else if got != "skipped" {
		t.Fatalf("TextContent(#status) = %q, want skipped", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty when guarded custom window globals are absent", got)
	}
}

func TestSessionBootstrapsExternalJSVarGlobalVisibleToBareIdentifier(t *testing.T) {
	const scriptURL = "https://cdn.jsdelivr.net/npm/lucide@latest/dist/umd/lucide.js"
	rawHTML := `<main><div id="icons"></div><div id="status"></div><script src="` + scriptURL + `"></script><script>lucide.createIcons(); host:setTextContent("#status", "ok")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.Registry().ExternalJS().RespondSource(scriptURL, `var lucide = globalThis.lucide = window.lucide = { createIcons: function () { document.getElementById("icons").textContent = "loaded"; } };`)
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#icons"); err != nil {
		t.Fatalf("TextContent(#icons) error = %v", err)
	} else if got != "loaded" {
		t.Fatalf("TextContent(#icons) = %q, want loaded", got)
	}
	if got, err := session.TextContent("#status"); err != nil {
		t.Fatalf("TextContent(#status) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#status) = %q, want ok", got)
	}
	if got := session.Registry().ExternalJS().Calls(); len(got) != 1 || got[0].URL != scriptURL {
		t.Fatalf("ExternalJS().Calls() = %#v, want one loaded script URL", got)
	}
}

func TestSessionBootstrapsUnpkgLucideExternalJSWindowGlobal(t *testing.T) {
	const scriptURL = "https://unpkg.com/lucide@latest"
	rawHTML := `<main><div id="icons"></div><div id="status"></div><script src="` + scriptURL + `"></script><script>if (window.lucide && typeof window.lucide.createIcons === "function") { window.lucide.createIcons(); } document.getElementById("status").textContent = window.lucide && typeof window.lucide.createIcons === "function" ? "ok" : "skipped"</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.Registry().ExternalJS().RespondSource(scriptURL, `window.lucide = { createIcons: function () { document.getElementById("icons").textContent = "loaded"; } };`)
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#icons"); err != nil {
		t.Fatalf("TextContent(#icons) error = %v", err)
	} else if got != "loaded" {
		t.Fatalf("TextContent(#icons) = %q, want loaded", got)
	}
	if got, err := session.TextContent("#status"); err != nil {
		t.Fatalf("TextContent(#status) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#status) = %q, want ok", got)
	}
	if got := session.Registry().ExternalJS().Calls(); len(got) != 1 || got[0].URL != scriptURL {
		t.Fatalf("ExternalJS().Calls() = %#v, want one loaded script URL", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after unpkg Lucide bootstrap", got)
	}
}

func TestSessionRejectsUnmockedExternalScriptSrc(t *testing.T) {
	const scriptURL = "https://cdn.jsdelivr.net/npm/lucide@latest/dist/umd/lucide.js"
	rawHTML := `<main><script src="` + scriptURL + `"></script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want external JS mock failure")
	} else if !strings.Contains(err.Error(), "no external JS mock configured") {
		t.Fatalf("ensureDOM() error = %v, want external JS mock failure text", err)
	}
	if got := session.DOMError(); !strings.Contains(got, "no external JS mock configured") {
		t.Fatalf("DOMError() = %q, want external JS mock failure text", got)
	}
}

func TestSessionBootstrapsSeededNavigatorLanguage(t *testing.T) {
	const rawHTML = `<main><div id="locale"></div><script>host:setTextContent("#locale", expr(navigator.language))</script></main>`

	session := NewSession(SessionConfig{})
	session.Registry().Navigator().SeedLanguage("fr-FR")

	if err := session.WriteHTML(rawHTML); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, err := session.TextContent("#locale"); err != nil {
		t.Fatalf("TextContent(#locale) error = %v", err)
	} else if got != "fr-FR" {
		t.Fatalf("TextContent(#locale) = %q, want fr-FR", got)
	}
}

func TestSessionBootstrapsHistoryAndLocationReplaceMemberReferences(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [typeof window.history.replaceState, typeof window.location.replace].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "function|function" {
		t.Fatalf("TextContent(#out) = %q, want function|function", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after history/location bootstrap", got)
	}
}

func TestSessionBootstrapsIntlDateTimeFormatTimeZoneAndParts(t *testing.T) {
	const rawHTML = `<main><pre id="out"></pre><script>function zonedText(instantMs, zone) { const formatter = new Intl.DateTimeFormat("en-US-u-nu-latn", { timeZone: zone, year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false, }); const parts = formatter.formatToParts(new Date(instantMs)); const get = (type) => parts.find((part) => part.type === type)?.value || "?"; return get("year") + "-" + get("month") + "-" + get("day") + " " + get("hour") + ":" + get("minute") + ":" + get("second"); } const arrivalInstant = Date.UTC(2026, 0, 21, 8, 45, 0, 0); document.getElementById("out").textContent = zonedText(arrivalInstant, "America/Chicago") + "|" + zonedText(arrivalInstant, "America/New_York");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "2026-01-21 02:45:00|2026-01-21 03:45:00" {
		t.Fatalf("TextContent(#out) = %q, want 2026-01-21 02:45:00|2026-01-21 03:45:00", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Intl.DateTimeFormat timeZone bootstrap", got)
	}
}

func TestSessionBootstrapsTemplateMatchMediaMatches(t *testing.T) {
	const rawHTML = `<main><div id="mode"></div><script>const mobile = window.matchMedia("(max-width: 1079px)").matches; host:setTextContent("#mode", expr(mobile ? "mobile" : "desktop"))</script></main>`

	session := NewSession(SessionConfig{
		HTML: rawHTML,
		MatchMedia: map[string]bool{
			"(max-width: 1079px)": true,
		},
	})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#mode"); err != nil {
		t.Fatalf("TextContent(#mode) error = %v", err)
	} else if got != "mobile" {
		t.Fatalf("TextContent(#mode) = %q, want mobile", got)
	}
	if got := session.MatchMediaCalls(); len(got) != 1 || got[0].Query != "(max-width: 1079px)" {
		t.Fatalf("MatchMediaCalls() = %#v, want one max-width query", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after matchMedia.matches bootstrap", got)
	}
}

func TestSessionBootstrapsStringStartsWith(t *testing.T) {
	const rawHTML = `<main><div id="prefix"></div><script>const prefix = navigator.language.startsWith("en") ? "yes" : "no"; host:setTextContent("#prefix", expr(prefix))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#prefix"); err != nil {
		t.Fatalf("TextContent(#prefix) error = %v", err)
	} else if got != "yes" {
		t.Fatalf("TextContent(#prefix) = %q, want yes", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after startsWith bootstrap", got)
	}
}

func TestSessionBootstrapsStringEndsWith(t *testing.T) {
	const rawHTML = `<main><div id="suffix"></div><script>const suffix = navigator.language.endsWith("US") ? "yes" : "no"; host:setTextContent("#suffix", expr(suffix))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#suffix"); err != nil {
		t.Fatalf("TextContent(#suffix) error = %v", err)
	} else if got != "yes" {
		t.Fatalf("TextContent(#suffix) = %q, want yes", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after endsWith bootstrap", got)
	}
}

func TestSessionBootstrapsURLSearchParamsForEach(t *testing.T) {
	const rawHTML = `<main><div id="seen"></div><script>const params = new URLSearchParams("b=3&a=1&a=2"); let seen = ""; params.forEach(function (value, key, paramsObject) { seen = seen + key + "=" + value + ":" + paramsObject.toString() + ","; document.getElementById("seen").textContent = seen; });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#seen"); err != nil {
		t.Fatalf("TextContent(#seen) error = %v", err)
	} else if got != "b=3:b=3&a=1&a=2,a=1:b=3&a=1&a=2,a=2:b=3&a=1&a=2," {
		t.Fatalf("TextContent(#seen) = %q, want URLSearchParams.forEach output", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after URLSearchParams.forEach bootstrap", got)
	}
}

func TestSessionBootstrapsMapForEach(t *testing.T) {
	const rawHTML = `<main><div id="seen"></div><script>const map = new Map([["alpha", 1], ["beta", 2]]); const context = { prefix: "ctx" }; let seen = ""; map.forEach(function (value, key, mapObject) { seen = seen + (seen === "" ? "" : "|") + this.prefix + ":" + key + ":" + value + ":" + mapObject.get(key); document.getElementById("seen").textContent = seen; }, context);</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#seen"); err != nil {
		t.Fatalf("TextContent(#seen) error = %v", err)
	} else if got != "ctx:alpha:1:1|ctx:beta:2:2" {
		t.Fatalf("TextContent(#seen) = %q, want ctx:alpha:1:1|ctx:beta:2:2", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Map.forEach bootstrap", got)
	}
}

func TestSessionBootstrapsArrayFromSet(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = Array.from(new Set(["alpha", "alpha", "beta"])).join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha,beta" {
		t.Fatalf("TextContent(#out) = %q, want alpha,beta", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.from(Set) bootstrap", got)
	}
}

func TestSessionBootstrapsSetConstructorIterableInputs(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>const empty = new Set(); const copy = new Set(new Set(["alpha", "alpha", "beta"])); const params = new URLSearchParams("u=metric&h=3.2&s=4.0"); const fromKeys = new Set(params.keys()); document.getElementById("out").textContent = [String(empty.size), Array.from(copy).join(","), Array.from(fromKeys).join(",")].join("|")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "0|alpha,beta|u,h,s" {
		t.Fatalf("TextContent(#out) = %q, want 0|alpha,beta|u,h,s", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Set constructor iterable bootstrap", got)
	}
}

func TestSessionBootstrapsArraySpreadFromSet(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = [...new Set(["alpha", "alpha", "beta"])].join(",")</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "alpha,beta" {
		t.Fatalf("TextContent(#out) = %q, want alpha,beta", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after array spread from Set bootstrap", got)
	}
}

func TestSessionBootstrapsNullishTimerCleanupNoop(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>clearTimeout(null); clearInterval(undefined); document.getElementById("out").textContent = "ok"</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after nullish timer cleanup bootstrap", got)
	}
}

func TestSessionBootstrapsFileInputTextSnapshot(t *testing.T) {
	const rawHTML = `<main><input id="upload" type="file"><div id="out"></div><script>document.getElementById("upload").addEventListener("change", () => { const file = document.getElementById("upload").files[0]; file.text().then((text) => { document.getElementById("out").textContent = text; }); });</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	session.Registry().FileInput().SeedFileText("#upload", "sample.json", `{"message":"ok"}`)

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.SetFiles("#upload", []string{"sample.json"}); err != nil {
		t.Fatalf("SetFiles(#upload) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"message":"ok"}` {
		t.Fatalf("TextContent(#out) = %q, want seeded file text", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after file-input text bootstrap", got)
	}
}

func TestSessionBootstrapsTemplateURLAndSearchParamsBridge(t *testing.T) {
	const rawHTML = `<main><div id="href"></div><div id="search"></div><div id="keys"></div><div id="mode"></div><script>const url = new URL(window.location.href); url.search = ""; url.searchParams.set("mode", "raw"); const params = new URLSearchParams(window.location.search || ""); host:setTextContent("#href", expr(url.href)); host:setTextContent("#search", expr(url.search)); host:setTextContent("#keys", expr([...params.keys()].join(","))); host:setTextContent("#mode", expr(params.get("mode")))</script></main>`

	session := NewSession(SessionConfig{
		URL:  "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial",
		HTML: rawHTML,
	})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#href"); err != nil {
		t.Fatalf("TextContent(#href) error = %v", err)
	} else if got != "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=raw" {
		t.Fatalf("TextContent(#href) = %q, want https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=raw", got)
	}
	if got, err := session.TextContent("#search"); err != nil {
		t.Fatalf("TextContent(#search) error = %v", err)
	} else if got != "?mode=raw" {
		t.Fatalf("TextContent(#search) = %q, want ?mode=raw", got)
	}
	if got, err := session.TextContent("#keys"); err != nil {
		t.Fatalf("TextContent(#keys) error = %v", err)
	} else if got != "mode" {
		t.Fatalf("TextContent(#keys) = %q, want mode", got)
	}
	if got, err := session.TextContent("#mode"); err != nil {
		t.Fatalf("TextContent(#mode) error = %v", err)
	} else if got != "initial" {
		t.Fatalf("TextContent(#mode) = %q, want initial", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after URL/searchParams bootstrap", got)
	}
}

func TestSessionBootstrapsArrayFromURLSearchParamsKeysIterator(t *testing.T) {
	const rawHTML = `<main><div id="keys"></div><script>const params = new URLSearchParams("u=metric&h=3.2&s=4.0"); host:setTextContent("#keys", expr(Array.from(params.keys()).join(",")))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#keys"); err != nil {
		t.Fatalf("TextContent(#keys) error = %v", err)
	} else if got != "u,h,s" {
		t.Fatalf("TextContent(#keys) = %q, want u,h,s", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Array.from(URLSearchParams.keys()) bootstrap", got)
	}
}

func TestSessionBootstrapsTemplateURLSearchParamsMemberParity(t *testing.T) {
	const rawHTML = `<main><div id="entries"></div><div id="values"></div><div id="all"></div><div id="sorted"></div><script>const params = new URLSearchParams("b=3&a=1&a=2"); const entries = [...params.entries()].map((pair) => pair.join("=")).join(","); const values = [...params.values()].join(","); const all = params.getAll("a").join(","); params.sort(); host:setTextContent("#entries", expr(entries)); host:setTextContent("#values", expr(values)); host:setTextContent("#all", expr(all)); host:setTextContent("#sorted", expr(params.toString()))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#entries"); err != nil {
		t.Fatalf("TextContent(#entries) error = %v", err)
	} else if got != "b=3,a=1,a=2" {
		t.Fatalf("TextContent(#entries) = %q, want b=3,a=1,a=2", got)
	}
	if got, err := session.TextContent("#values"); err != nil {
		t.Fatalf("TextContent(#values) error = %v", err)
	} else if got != "3,1,2" {
		t.Fatalf("TextContent(#values) = %q, want 3,1,2", got)
	}
	if got, err := session.TextContent("#all"); err != nil {
		t.Fatalf("TextContent(#all) error = %v", err)
	} else if got != "1,2" {
		t.Fatalf("TextContent(#all) = %q, want 1,2", got)
	}
	if got, err := session.TextContent("#sorted"); err != nil {
		t.Fatalf("TextContent(#sorted) error = %v", err)
	} else if got != "a=1&a=2&b=3" {
		t.Fatalf("TextContent(#sorted) = %q, want a=1&a=2&b=3", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after URLSearchParams member parity bootstrap", got)
	}
}

func TestSessionBootstrapsTemplateObjectEntriesAndValues(t *testing.T) {
	const rawHTML = `<main><div id="entries"></div><div id="values"></div><script>const assigned = Object.assign({ first: "a" }, { second: "b" }); host:setTextContent("#entries", expr(Object.entries(assigned).map((entry) => entry.join("=")).join(","))); host:setTextContent("#values", expr(Object.values(assigned).join(",")))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#entries"); err != nil {
		t.Fatalf("TextContent(#entries) error = %v", err)
	} else if got != "first=a,second=b" {
		t.Fatalf("TextContent(#entries) = %q, want first=a,second=b", got)
	}
	if got, err := session.TextContent("#values"); err != nil {
		t.Fatalf("TextContent(#values) error = %v", err)
	} else if got != "a,b" {
		t.Fatalf("TextContent(#values) = %q, want a,b", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.entries/Object.values bootstrap", got)
	}
}

func TestSessionBootstrapsObjectFromEntries(t *testing.T) {
	const rawHTML = `<main><pre id="out"></pre><script>const kanaPairs = [["full", "アイウ"], ["half", "ｱｲｳ"]]; const normalized = Object.fromEntries(kanaPairs.map(([key, value]) => [key, value.slice(0, 2)])); const aliases = Object.fromEntries(new Map([["zenkaku", normalized.full], ["hankaku", normalized.half]])); document.getElementById("out").textContent = aliases.zenkaku + "|" + aliases.hankaku + "|" + Object.keys(aliases).join(",");</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "アイ|ｱｲ|zenkaku,hankaku" {
		t.Fatalf("TextContent(#out) = %q, want アイ|ｱｲ|zenkaku,hankaku", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Object.fromEntries bootstrap", got)
	}
}

func TestSessionBootstrapsSymbolAssignToFixedAndInstanceof(t *testing.T) {
	const rawHTML = `<main><button id="btn">run</button><select id="sel"><option value="one">one</option></select><div id="out"></div><script>const button = document.getElementById("btn"); const select = document.getElementById("sel"); const symA = Symbol("token"); const symB = Symbol("token"); const assigned = Object.assign({ plain: "a" }, "go", null, undefined, { extra: "b" }, { [symA]: "symbol" }); const symbols = Object.getOwnPropertySymbols(assigned); button.addEventListener("click", (event) => { document.getElementById("out").textContent = [button instanceof HTMLButtonElement, button instanceof HTMLElement, select instanceof HTMLSelectElement, event.target instanceof HTMLButtonElement, event.currentTarget instanceof HTMLElement, symA === symB, assigned.plain, assigned[0], assigned[1], assigned.extra, symbols.length, symbols[0].toString(), assigned[symbols[0]], (1.2).toFixed(2)].join("|"); })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if err := session.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|true|true|true|true|false|a|g|o|b|1|Symbol(token)|symbol|1.20" {
		t.Fatalf("TextContent(#out) = %q, want %q", got, "true|true|true|true|true|false|a|g|o|b|1|Symbol(token)|symbol|1.20")
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after symbol/instanceof bootstrap", got)
	}
}

func TestSessionBootstrapsClipboardPromiseChain(t *testing.T) {
	const rawHTML = `<main><div id="status"></div><script>clipboard.writeText("copied").then(function () { localStorage.setItem("status", "copied") }).catch(function () { localStorage.setItem("status", "failed") })</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got := session.ClipboardWrites(); len(got) != 1 || got[0] != "copied" {
		t.Fatalf("ClipboardWrites() = %#v, want one copied write", got)
	}
	if got := session.LocalStorage()["status"]; got != "copied" {
		t.Fatalf("LocalStorage()[status] = %q, want copied", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after clipboard promise chain bootstrap", got)
	}
}

func TestSessionBootstrapsOptionalCatchBindingInAsyncHelper(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>async function copyText(text) { try { await navigator.clipboard.writeText(text); return true; } catch { return false; } } async function run() { document.getElementById("out").textContent = String(await copyText("copied")) }; await run()</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true" {
		t.Fatalf("TextContent(#out) = %q, want true", got)
	}
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after optional catch binding bootstrap", got)
	}
}
