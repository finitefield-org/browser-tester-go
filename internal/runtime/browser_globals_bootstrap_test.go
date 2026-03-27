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
