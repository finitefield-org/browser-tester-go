package runtime

import "testing"

func TestSessionBootstrapsRawHtmlWithBrowserGlobals(t *testing.T) {
	const rawHTML = `<main><div id="agri-unit-converter-root">root</div><div id="result"></div><div id="formatted"></div><div id="href"></div><script>const root = document.getElementById("agri-unit-converter-root"); const current = new URL(window.location.href); const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); window.location.search.length; sessionStorage.setItem("mode", navigator.onLine && "search"); window.history.replaceState({}, "", "?mode=raw#ready"); localStorage.setItem("format", formatted); matchMedia("(prefers-reduced-motion: reduce)"); clipboard.writeText(root.textContent); setTimeout("noop", 5); queueMicrotask("noop"); host:setTextContent("#result", expr(root.textContent)); host:setTextContent("#formatted", expr(formatted)); host:setTextContent("#href", expr(current.href))</script></main>`

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
	if got, want := session.DumpDOM(), `<main><div id="agri-unit-converter-root">root</div><div id="result">root</div><div id="formatted">1.23</div><div id="href">https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial</div><script>const root = document.getElementById("agri-unit-converter-root"); const current = new URL(window.location.href); const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); window.location.search.length; sessionStorage.setItem("mode", navigator.onLine && "search"); window.history.replaceState({}, "", "?mode=raw#ready"); localStorage.setItem("format", formatted); matchMedia("(prefers-reduced-motion: reduce)"); clipboard.writeText(root.textContent); setTimeout("noop", 5); queueMicrotask("noop"); host:setTextContent("#result", expr(root.textContent)); host:setTextContent("#formatted", expr(formatted)); host:setTextContent("#href", expr(current.href))</script></main>`; got != want {
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

func TestSessionBootstrapsTemplateLocaleAndOptionalWindowGlobals(t *testing.T) {
	const rawHTML = `<main><div id="locale"></div><div id="stamp"></div><script>const locale = navigator.language || "en-US"; const stamp = new Intl.DateTimeFormat(locale, { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(new Date(1700000000000)); if (window.lucide && typeof window.lucide.createIcons === "function") { window.lucide.createIcons(); } host:setTextContent("#locale", expr(locale)); host:setTextContent("#stamp", expr(stamp))</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
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
	if got := session.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after optional browser globals bootstrap", got)
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
	const rawHTML = `<main><div id="seen"></div><script>const params = new URLSearchParams("b=3&a=1&a=2"); let seen = ""; params.forEach(function (value, key, paramsObject) { seen = seen + key + "=" + value + ":" + paramsObject.toString() + ","; }); host:setTextContent("#seen", expr(seen))</script></main>`

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
