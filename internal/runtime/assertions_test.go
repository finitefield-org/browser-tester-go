package runtime

import (
	"errors"
	"strings"
	"testing"
)

func TestSessionAssertionsSucceed(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main><div id="out">hello</div><input id="name" value="Ada"><input id="agree" type="checkbox" checked><input id="upload" type="file"><select id="mode"><option value="a">A</option><option value="b" selected>B</option></select></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("#out"); err != nil {
		t.Fatalf("AssertExists(#out) error = %v", err)
	}
	if err := s.AssertText("#out", "hello"); err != nil {
		t.Fatalf("AssertText(#out) error = %v", err)
	}
	if err := s.AssertValue("#name", "Ada"); err != nil {
		t.Fatalf("AssertValue(#name) error = %v", err)
	}
	if err := s.AssertChecked("#agree", true); err != nil {
		t.Fatalf("AssertChecked(#agree) error = %v", err)
	}
	if err := s.AssertValue("#mode", "b"); err != nil {
		t.Fatalf("AssertValue(#mode) error = %v", err)
	}
	if err := s.AssertExists("input:checked"); err != nil {
		t.Fatalf("AssertExists(input:checked) error = %v", err)
	}

	if err := s.SetFiles("#upload", []string{"first.txt", "second.txt"}); err != nil {
		t.Fatalf("SetFiles() error = %v", err)
	}
	if err := s.AssertValue("#upload", "first.txt, second.txt"); err != nil {
		t.Fatalf("AssertValue(#upload) after SetFiles error = %v", err)
	}
	if err := s.AssertExists("#upload:user-valid"); err != nil {
		t.Fatalf("AssertExists(#upload:user-valid) after SetFiles error = %v", err)
	}

	if err := s.SetFiles("#upload", []string{"report.csv"}); err != nil {
		t.Fatalf("SetFiles() #2 error = %v", err)
	}
	if err := s.AssertValue("#upload", "report.csv"); err != nil {
		t.Fatalf("AssertValue(#upload) after SetFiles #2 error = %v", err)
	}
}

func TestSessionAssertionsSupportBlankPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><input id="blank" type="text"><textarea id="story"></textarea><input id="filled" type="text" value="Ada"></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:blank"); err != nil {
		t.Fatalf("AssertExists(input:blank) error = %v", err)
	}
	if err := s.AssertExists("textarea:blank"); err != nil {
		t.Fatalf("AssertExists(textarea:blank) error = %v", err)
	}
	if err := s.AssertExists("#blank:blank"); err != nil {
		t.Fatalf("AssertExists(#blank:blank) error = %v", err)
	}
	if err := s.AssertExists("#filled:blank"); err == nil {
		t.Fatalf("AssertExists(#filled:blank) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportBlankPseudoClassForCheckableAndSelectControls(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><input id="checkbox-off" type="checkbox"><input id="checkbox-on" type="checkbox" checked><input id="radio-off" type="radio" name="choice"><input id="radio-on" type="radio" name="choice" checked><select id="empty-select"><option value="a">A</option></select><select id="filled-select"><option value="b" selected>B</option></select></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:blank"); err != nil {
		t.Fatalf("AssertExists(input:blank) error = %v", err)
	}
	if err := s.AssertExists("select:blank"); err != nil {
		t.Fatalf("AssertExists(select:blank) error = %v", err)
	}
	if err := s.AssertExists("#checkbox-off:blank"); err != nil {
		t.Fatalf("AssertExists(#checkbox-off:blank) error = %v", err)
	}
	if err := s.AssertExists("#radio-off:blank"); err != nil {
		t.Fatalf("AssertExists(#radio-off:blank) error = %v", err)
	}
	if err := s.AssertExists("#checkbox-on:blank"); err == nil {
		t.Fatalf("AssertExists(#checkbox-on:blank) error = nil, want no match")
	}
	if err := s.AssertExists("#radio-on:blank"); err == nil {
		t.Fatalf("AssertExists(#radio-on:blank) error = nil, want no match")
	}
	if err := s.AssertExists("#empty-select:blank"); err != nil {
		t.Fatalf("AssertExists(#empty-select:blank) error = %v", err)
	}
	if err := s.AssertExists("#filled-select:blank"); err == nil {
		t.Fatalf("AssertExists(#filled-select:blank) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportDefinedPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><div id="known"></div><x-widget id="widget" defined></x-widget><x-ghost id="ghost"></x-ghost></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("div:defined"); err != nil {
		t.Fatalf("AssertExists(div:defined) error = %v", err)
	}
	if err := s.AssertExists("x-widget:defined"); err != nil {
		t.Fatalf("AssertExists(x-widget:defined) error = %v", err)
	}
	if err := s.AssertExists("#ghost:defined"); err == nil {
		t.Fatalf("AssertExists(#ghost:defined) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportStatePseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><x-widget id="widget"></x-widget><div id="plain" state="checked"></div></main>`
	s := NewSession(cfg)

	if err := s.SetAttribute("#widget", "state", "checked pressed"); err != nil {
		t.Fatalf("SetAttribute(#widget, state, checked pressed) error = %v", err)
	}

	if err := s.AssertExists("x-widget:state(checked)"); err != nil {
		t.Fatalf("AssertExists(x-widget:state(checked)) error = %v", err)
	}
	if err := s.AssertExists("x-widget:state(checked):state(pressed)"); err != nil {
		t.Fatalf("AssertExists(x-widget:state(checked):state(pressed)) error = %v", err)
	}
	if err := s.AssertExists("div:state(checked)"); err == nil {
		t.Fatalf("AssertExists(div:state(checked)) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportSelectorLists(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="wrap" data-note="a,b"><article id="a1"></article><section id="inner"><p id="leaf">Leaf</p></section></section><aside id="plain"></aside></main>`
	s := NewSession(cfg)

	if err := s.AssertExists(`section[data-note="a,b"], #missing`); err != nil {
		t.Fatalf(`AssertExists(section[data-note="a,b"], #missing) error = %v`, err)
	}
	if err := s.AssertExists(`.missing, #plain`); err != nil {
		t.Fatalf(`AssertExists(.missing, #plain) error = %v`, err)
	}

	err := s.AssertExists("section,")
	if err == nil {
		t.Fatalf(`AssertExists(section,) error = nil, want SelectorError`)
	}
	var sel SelectorError
	if !errors.As(err, &sel) {
		t.Fatalf(`AssertExists(section,) error = %T, want SelectorError`, err)
	}
}

func TestSessionAssertionsSupportAutofillPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><input id="name" autofill value="Ada"><input id="other" value="Bob"></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:autofill"); err != nil {
		t.Fatalf("AssertExists(input:autofill) error = %v", err)
	}
	if err := s.AssertExists("input:-webkit-autofill"); err != nil {
		t.Fatalf("AssertExists(input:-webkit-autofill) error = %v", err)
	}

	if err := s.TypeText("#name", "Zed"); err != nil {
		t.Fatalf("TypeText(#name) error = %v", err)
	}
	if err := s.AssertValue("#name", "Zed"); err != nil {
		t.Fatalf("AssertValue(#name) after TypeText error = %v", err)
	}
	if err := s.AssertExists("#name:autofill"); err == nil {
		t.Fatalf("AssertExists(#name:autofill) after TypeText error = nil, want no match")
	}
}

func TestSessionAssertionsSupportActiveHoverPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><div id="wrap"><button id="btn" active>Go</button><span id="hovered" hover>Hover</span></div><label id="active-label" for="active-field" active>Field</label><input id="active-field" type="text"><label id="hover-label" hover><input id="hover-field" type="text"></label><label id="secret-label" for="secret" active>Secret</label><input id="secret" type="hidden"><p id="plain">Text</p></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("button:active"); err != nil {
		t.Fatalf("AssertExists(button:active) error = %v", err)
	}
	if err := s.AssertExists("div:active"); err != nil {
		t.Fatalf("AssertExists(div:active) error = %v", err)
	}
	if err := s.AssertExists("span:hover"); err != nil {
		t.Fatalf("AssertExists(span:hover) error = %v", err)
	}
	if err := s.AssertExists("div:hover"); err != nil {
		t.Fatalf("AssertExists(div:hover) error = %v", err)
	}
	if err := s.AssertExists("input:active"); err != nil {
		t.Fatalf("AssertExists(input:active) error = %v", err)
	}
	if err := s.AssertExists("input:hover"); err != nil {
		t.Fatalf("AssertExists(input:hover) error = %v", err)
	}
	if err := s.AssertExists("#active-field:active"); err != nil {
		t.Fatalf("AssertExists(#active-field:active) error = %v", err)
	}
	if err := s.AssertExists("#hover-field:hover"); err != nil {
		t.Fatalf("AssertExists(#hover-field:hover) error = %v", err)
	}
	if err := s.AssertExists("#secret:active"); err == nil {
		t.Fatalf("AssertExists(#secret:active) error = nil, want no match")
	}
	if err := s.AssertExists("#plain:active"); err == nil {
		t.Fatalf("AssertExists(#plain:active) error = nil, want no match")
	}
}

func TestSessionAssertionsPreserveDefaultPseudoClassAcrossControlUpdates(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><form id="profile"><input id="flag" type="checkbox" checked><button id="submit-1" type="submit">Save</button><button id="submit-2" type="submit">Extra</button><select id="mode"><option id="opt-a" value="a" selected>A</option><option id="opt-b" value="b">B</option></select></form></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:default"); err != nil {
		t.Fatalf("AssertExists(input:default) before updates error = %v", err)
	}
	if err := s.AssertExists("option:default"); err != nil {
		t.Fatalf("AssertExists(option:default) before updates error = %v", err)
	}

	if err := s.SetChecked("#flag", false); err != nil {
		t.Fatalf("SetChecked(#flag) error = %v", err)
	}
	if err := s.SetSelectValue("#mode", "b"); err != nil {
		t.Fatalf("SetSelectValue(#mode, b) error = %v", err)
	}

	if err := s.AssertExists("input:default"); err != nil {
		t.Fatalf("AssertExists(input:default) after updates error = %v", err)
	}
	if err := s.AssertExists("option:default"); err != nil {
		t.Fatalf("AssertExists(option:default) after updates error = %v", err)
	}
	if err := s.AssertExists("#opt-b:default"); err == nil {
		t.Fatalf("AssertExists(#opt-b:default) after updates error = nil, want no match")
	}
}

func TestSessionAssertionsReturnSelectorErrorForInvalidSelectors(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main><div id="out"></div></main>`
	s := NewSession(cfg)

	err := s.AssertExists("div[item=")
	if err == nil {
		t.Fatalf("AssertExists(div[item=]) error = nil, want SelectorError")
	}
	var sel SelectorError
	if !errors.As(err, &sel) {
		t.Fatalf("AssertExists(div[item=]) error = %T, want SelectorError", err)
	}
}

func TestSessionAssertionsSupportAttributeSelectors(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main><div id="root" data-kind="panel"><a id="nav" href="/next" data-role="nav">Go</a><input id="name" type="text"><p id="flag" hidden></p><span id="meta" data-tags="alpha beta gamma" data-locale="en-US" data-note="prefix-middle-suffix" data-code="abc123"></span></div></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("div[data-kind]"); err != nil {
		t.Fatalf("AssertExists(div[data-kind]) error = %v", err)
	}
	if err := s.AssertExists("a[href]"); err != nil {
		t.Fatalf("AssertExists(a[href]) error = %v", err)
	}
	if err := s.AssertExists("a[href=\"/next\"]"); err != nil {
		t.Fatalf("AssertExists(a[href=\"/next\"]) error = %v", err)
	}
	if err := s.AssertExists("a[data-role=nav]"); err != nil {
		t.Fatalf("AssertExists(a[data-role=nav]) error = %v", err)
	}
	if err := s.AssertExists("input[type=text]"); err != nil {
		t.Fatalf("AssertExists(input[type=text]) error = %v", err)
	}
	if err := s.AssertExists("p[hidden]"); err != nil {
		t.Fatalf("AssertExists(p[hidden]) error = %v", err)
	}
	if err := s.AssertExists("span[data-tags~=beta]"); err != nil {
		t.Fatalf("AssertExists(span[data-tags~=beta]) error = %v", err)
	}
	if err := s.AssertExists("span[data-locale|=en]"); err != nil {
		t.Fatalf("AssertExists(span[data-locale|=en]) error = %v", err)
	}
	if err := s.AssertExists("span[data-note^=prefix]"); err != nil {
		t.Fatalf("AssertExists(span[data-note^=prefix]) error = %v", err)
	}
	if err := s.AssertExists("span[data-note$=suffix]"); err != nil {
		t.Fatalf("AssertExists(span[data-note$=suffix]) error = %v", err)
	}
	if err := s.AssertExists("span[data-note*=middle]"); err != nil {
		t.Fatalf("AssertExists(span[data-note*=middle]) error = %v", err)
	}
	if err := s.AssertExists("span[data-tags~=BETA i]"); err != nil {
		t.Fatalf("AssertExists(span[data-tags~=BETA i]) error = %v", err)
	}
	if err := s.AssertExists("input[type=TEXT i]"); err != nil {
		t.Fatalf("AssertExists(input[type=TEXT i]) error = %v", err)
	}
	if err := s.AssertExists("a[data-role=missing]"); err == nil {
		t.Fatalf("AssertExists(a[data-role=missing]) error = nil, want no match")
	}
	if err := s.AssertExists("span[data-tags~=BETA s]"); err == nil {
		t.Fatalf("AssertExists(span[data-tags~=BETA s]) error = nil, want no match")
	}
	if err := s.AssertExists("span[data-tags~=delta]"); err == nil {
		t.Fatalf("AssertExists(span[data-tags~=delta]) error = nil, want no match")
	}
}

func TestSessionAssertionsIncludeDOMDumpOnFailure(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main><div id="out">hello</div><input id="name" value="Ada"></main>`
	s := NewSession(cfg)

	err := s.AssertExists("#missing")
	if err == nil {
		t.Fatalf("AssertExists(#missing) error = nil, want AssertionError")
	}
	var as AssertionError
	if !errors.As(err, &as) {
		t.Fatalf("AssertExists(#missing) error = %T, want AssertionError", err)
	}
	if !strings.Contains(err.Error(), "DOM:\n") || !strings.Contains(err.Error(), `<main>`) {
		t.Fatalf("AssertExists(#missing) error = %q, want DOM dump", err.Error())
	}

	err = s.AssertText("#out", "nope")
	if err == nil {
		t.Fatalf("AssertText(#out) error = nil, want AssertionError")
	}
	if !errors.As(err, &as) {
		t.Fatalf("AssertText(#out) error = %T, want AssertionError", err)
	}
	if !strings.Contains(err.Error(), "DOM:\n") || !strings.Contains(err.Error(), `<div id="out">hello</div>`) {
		t.Fatalf("AssertText(#out) error = %q, want DOM dump including #out", err.Error())
	}

	err = s.AssertChecked("#name", true)
	if err == nil {
		t.Fatalf("AssertChecked(#name) error = nil, want AssertionError")
	}
	if !errors.As(err, &as) {
		t.Fatalf("AssertChecked(#name) error = %T, want AssertionError", err)
	}
	if !strings.Contains(err.Error(), "checkable control") {
		t.Fatalf("AssertChecked(#name) error = %q, want non-checkable message", err.Error())
	}
}

func TestSessionAssertionsSupportLinkDefaultAndPlaceholderPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main><a id="nav" href="/next">Go</a><map><area id="area" href="/popup" alt="Open"></map><form id="profile"><button id="submit-1" type="submit">Save</button><input id="placeholder" type="text" placeholder="Name"><textarea id="story" placeholder="Story"></textarea></form></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("a:link"); err != nil {
		t.Fatalf("AssertExists(a:link) error = %v", err)
	}
	if err := s.AssertExists("area:link"); err != nil {
		t.Fatalf("AssertExists(area:link) error = %v", err)
	}
	if err := s.AssertExists("a:any-link"); err != nil {
		t.Fatalf("AssertExists(a:any-link) error = %v", err)
	}
	if err := s.AssertExists("area:any-link"); err != nil {
		t.Fatalf("AssertExists(area:any-link) error = %v", err)
	}
	if err := s.AssertExists("button:default"); err != nil {
		t.Fatalf("AssertExists(button:default) error = %v", err)
	}
	if err := s.AssertExists("input:placeholder-shown"); err != nil {
		t.Fatalf("AssertExists(input:placeholder-shown) error = %v", err)
	}
	if err := s.AssertExists("textarea:placeholder-shown"); err != nil {
		t.Fatalf("AssertExists(textarea:placeholder-shown) error = %v", err)
	}
}

func TestSessionAssertionsSupportLocalLinkPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.URL = "https://example.test/page#top"
	cfg.HTML = `<main><a id="self" href="#top">Self</a><a id="next" href="/next">Next</a><map><area id="area-self" href="#top" alt="Self"></map></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("a:local-link"); err != nil {
		t.Fatalf("AssertExists(a:local-link) error = %v", err)
	}
	if err := s.AssertExists("area:local-link"); err != nil {
		t.Fatalf("AssertExists(area:local-link) error = %v", err)
	}
	if err := s.AssertExists("#self:local-link"); err != nil {
		t.Fatalf("AssertExists(#self:local-link) error = %v", err)
	}
	if err := s.AssertExists("#next:local-link"); err == nil {
		t.Fatalf("AssertExists(#next:local-link) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportVisitedPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.URL = "https://example.test/page"
	cfg.HTML = `<main id="root"><a id="nav" href="https://example.test/visited">Go</a><a id="other" href="https://example.test/other">Other</a><map><area id="area" href="https://example.test/visited" alt="Visited"></map></main>`
	s := NewSession(cfg)

	if err := s.Navigate("https://example.test/visited"); err != nil {
		t.Fatalf("Navigate() error = %v", err)
	}

	if err := s.AssertExists("a:visited"); err != nil {
		t.Fatalf("AssertExists(a:visited) error = %v", err)
	}
	if err := s.AssertExists("area:visited"); err != nil {
		t.Fatalf("AssertExists(area:visited) error = %v", err)
	}
	if err := s.AssertExists("#nav:visited"); err != nil {
		t.Fatalf("AssertExists(#nav:visited) error = %v", err)
	}
	if err := s.AssertExists("#other:visited"); err == nil {
		t.Fatalf("AssertExists(#other:visited) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportMoreBoundedPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><h1 id="title">Title</h1><details id="details" open><summary>Sum</summary></details><dialog id="dialog" open></dialog><form id="profile"><input id="required" type="text" required><input id="optional" type="text"><input id="readonly" type="text" readonly><textarea id="editable"></textarea><textarea id="readonly-ta" readonly>Locked</textarea></form></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:required"); err != nil {
		t.Fatalf("AssertExists(input:required) error = %v", err)
	}
	if err := s.AssertExists("input:optional"); err != nil {
		t.Fatalf("AssertExists(input:optional) error = %v", err)
	}
	if err := s.AssertExists("input:read-write"); err != nil {
		t.Fatalf("AssertExists(input:read-write) error = %v", err)
	}
	if err := s.AssertExists("input:read-only"); err != nil {
		t.Fatalf("AssertExists(input:read-only) error = %v", err)
	}
	if err := s.AssertExists("textarea:read-write"); err != nil {
		t.Fatalf("AssertExists(textarea:read-write) error = %v", err)
	}
	if err := s.AssertExists("textarea:read-only"); err != nil {
		t.Fatalf("AssertExists(textarea:read-only) error = %v", err)
	}
	if err := s.AssertExists("h1:heading"); err != nil {
		t.Fatalf("AssertExists(h1:heading) error = %v", err)
	}
	if err := s.AssertExists("details:open"); err != nil {
		t.Fatalf("AssertExists(details:open) error = %v", err)
	}
	if err := s.AssertExists("dialog:open"); err != nil {
		t.Fatalf("AssertExists(dialog:open) error = %v", err)
	}
}

func TestSessionAssertionsSupportDisabledFieldsetAndOptgroupPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><form id="profile"><fieldset id="outer" disabled><legend id="legend"><span><input id="legend-input" type="text"></span></legend><input id="disabled-required" type="text" required><input id="disabled-optional" type="text"><textarea id="disabled-textarea"></textarea><select id="mode"><optgroup id="disabled-group" disabled label="Disabled"><option id="disabled-option" value="a">A</option></optgroup><optgroup id="enabled-group" label="Enabled"><option id="enabled-option" value="b">B</option></optgroup></select><fieldset id="inner"><input id="inner-input" type="text"></fieldset></fieldset><fieldset id="plain-fieldset"><input id="plain-input" type="text"></fieldset><input id="outside-required" type="text" required value="Ada"><input id="outside-optional" type="text"><textarea id="outside-textarea"></textarea></form></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("fieldset:disabled"); err != nil {
		t.Fatalf("AssertExists(fieldset:disabled) error = %v", err)
	}
	if err := s.AssertExists("fieldset:enabled"); err != nil {
		t.Fatalf("AssertExists(fieldset:enabled) error = %v", err)
	}
	if err := s.AssertExists("input:required"); err != nil {
		t.Fatalf("AssertExists(input:required) error = %v", err)
	}
	if err := s.AssertExists("option:disabled"); err != nil {
		t.Fatalf("AssertExists(option:disabled) error = %v", err)
	}
	if err := s.AssertExists("option:enabled"); err != nil {
		t.Fatalf("AssertExists(option:enabled) error = %v", err)
	}
	if err := s.AssertExists("select:disabled"); err != nil {
		t.Fatalf("AssertExists(select:disabled) error = %v", err)
	}
	if err := s.AssertExists("textarea:read-only"); err != nil {
		t.Fatalf("AssertExists(textarea:read-only) error = %v", err)
	}
	if err := s.AssertExists("form:valid"); err != nil {
		t.Fatalf("AssertExists(form:valid) error = %v", err)
	}
	if err := s.AssertExists("#legend-input:disabled"); err == nil {
		t.Fatalf("AssertExists(#legend-input:disabled) error = nil, want no match")
	}
	if err := s.AssertExists("#disabled-required:required"); err == nil {
		t.Fatalf("AssertExists(#disabled-required:required) error = nil, want no match")
	}
	if err := s.AssertExists("form:invalid"); err == nil {
		t.Fatalf("AssertExists(form:invalid) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportContentEditablePseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="editable" contenteditable><p id="inherited">Editable</p><div id="false" contenteditable="false"><span id="blocked">Blocked</span></div><div id="plaintext" contenteditable="plaintext-only"><em id="plain-child">Plain</em></div></section><input id="name" type="text"><textarea id="story"></textarea><input id="readonly" type="text" readonly><div id="plain">Plain</div></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("section:read-write"); err != nil {
		t.Fatalf("AssertExists(section:read-write) error = %v", err)
	}
	if err := s.AssertExists("div:read-write"); err != nil {
		t.Fatalf("AssertExists(div:read-write) error = %v", err)
	}
	if err := s.AssertExists("input:read-write"); err != nil {
		t.Fatalf("AssertExists(input:read-write) error = %v", err)
	}
	if err := s.AssertExists("textarea:read-write"); err != nil {
		t.Fatalf("AssertExists(textarea:read-write) error = %v", err)
	}
	if err := s.AssertExists("#blocked:read-only"); err != nil {
		t.Fatalf("AssertExists(#blocked:read-only) error = %v", err)
	}
	if err := s.AssertExists("#plain-child:read-write"); err != nil {
		t.Fatalf("AssertExists(#plain-child:read-write) error = %v", err)
	}
	if err := s.AssertExists("#plain:read-only"); err != nil {
		t.Fatalf("AssertExists(#plain:read-only) error = %v", err)
	}
	if err := s.AssertExists("#blocked:read-write"); err == nil {
		t.Fatalf("AssertExists(#blocked:read-write) error = nil, want no match")
	}
	if err := s.AssertExists("#plain:read-write"); err == nil {
		t.Fatalf("AssertExists(#plain:read-write) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportHasPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="wrap"><article id="a1"><span class="hit">Hit</span></article><article id="a2"><span class="miss">Miss</span></article></section><aside id="adjacent" class="hit"><span class="hit">Sibling</span></aside><aside id="plain"><span class="hit">Outside</span></aside></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("section:has(.hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(.hit)) error = %v", err)
	}
	if err := s.AssertExists("section:has(article > .hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(article > .hit)) error = %v", err)
	}
	if err := s.AssertExists("section:has(:bogus, .hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(:bogus, .hit)) error = %v", err)
	}
	if err := s.AssertExists("section:has(> article > .hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(> article > .hit)) error = %v", err)
	}
	if err := s.AssertExists("section:has(+ aside.hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(+ aside.hit)) error = %v", err)
	}
	if err := s.AssertExists("section:has(~ aside.hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(~ aside.hit)) error = %v", err)
	}
	if err := s.AssertExists("article:has(.hit, .miss)"); err != nil {
		t.Fatalf("AssertExists(article:has(.hit, .miss)) error = %v", err)
	}
	if err := s.AssertExists("section:has(.missing)"); err == nil {
		t.Fatalf("AssertExists(section:has(.missing)) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportNotPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="wrap"><article id="a1" class="match"><span class="hit">Hit</span></article><article id="a2"><span class="miss">Miss</span></article></section><aside id="plain"><span class="hit">Outside</span></aside></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("section:not(.missing)"); err != nil {
		t.Fatalf("AssertExists(section:not(.missing)) error = %v", err)
	}
	if err := s.AssertExists("section:not(:bogus)"); err != nil {
		t.Fatalf("AssertExists(section:not(:bogus)) error = %v", err)
	}
	if err := s.AssertExists("article:not(.match, .other)"); err != nil {
		t.Fatalf("AssertExists(article:not(.match, .other)) error = %v", err)
	}
	if err := s.AssertExists("section:not(:bogus, #wrap)"); err == nil {
		t.Fatalf("AssertExists(section:not(:bogus, #wrap)) error = nil, want no match")
	}
	if err := s.AssertExists("#a1:not(.match)"); err == nil {
		t.Fatalf("AssertExists(#a1:not(.match)) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportIsAndWherePseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="wrap" class="match"><article id="a1" class="hit">One</article><article id="a2" class="miss">Two</article></section><aside id="plain"><span class="hit">Outside</span></aside></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("section:is(#wrap, .missing)"); err != nil {
		t.Fatalf("AssertExists(section:is(#wrap, .missing)) error = %v", err)
	}
	if err := s.AssertExists("section:where(#wrap)"); err != nil {
		t.Fatalf("AssertExists(section:where(#wrap)) error = %v", err)
	}
	if err := s.AssertExists("article:where(.hit, .miss)"); err != nil {
		t.Fatalf("AssertExists(article:where(.hit, .miss)) error = %v", err)
	}
	if err := s.AssertExists("article:is(.hit)"); err != nil {
		t.Fatalf("AssertExists(article:is(.hit)) error = %v", err)
	}
	if err := s.AssertExists("#plain:is(.hit)"); err == nil {
		t.Fatalf("AssertExists(#plain:is(.hit)) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportScopePseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="panel"><p id="child">one</p></section><p id="sibling">two</p></main>`
	s := NewSession(cfg)

	if err := s.AssertExists(":scope"); err != nil {
		t.Fatalf("AssertExists(:scope) error = %v", err)
	}
	if err := s.AssertExists(":scope > section"); err != nil {
		t.Fatalf("AssertExists(:scope > section) error = %v", err)
	}
	if err := s.AssertExists(":scope > p"); err != nil {
		t.Fatalf("AssertExists(:scope > p) error = %v", err)
	}
	if err := s.AssertExists("section :scope"); err == nil {
		t.Fatalf("AssertExists(section :scope) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportModalPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><dialog id="dialog" modal></dialog><video id="player" fullscreen></video><div id="other" open></div></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("dialog:modal"); err != nil {
		t.Fatalf("AssertExists(dialog:modal) error = %v", err)
	}
	if err := s.AssertExists("video:modal"); err != nil {
		t.Fatalf("AssertExists(video:modal) error = %v", err)
	}
	if err := s.AssertExists("video:fullscreen"); err != nil {
		t.Fatalf("AssertExists(video:fullscreen) error = %v", err)
	}
	if err := s.AssertExists("#other:modal"); err == nil {
		t.Fatalf("AssertExists(#other:modal) error = nil, want no match")
	}
	if err := s.AssertExists("#other:fullscreen"); err == nil {
		t.Fatalf("AssertExists(#other:fullscreen) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportPopoverOpenPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><div id="menu" popover popover-open></div><div id="closed" popover></div><dialog id="dialog" open></dialog></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("div:popover-open"); err != nil {
		t.Fatalf("AssertExists(div:popover-open) error = %v", err)
	}
	if err := s.AssertExists("#menu:popover-open"); err != nil {
		t.Fatalf("AssertExists(#menu:popover-open) error = %v", err)
	}
	if err := s.AssertExists("#closed:popover-open"); err == nil {
		t.Fatalf("AssertExists(#closed:popover-open) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportOpenPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><select id="dropdown" open><option id="dropdown-option" value="a">A</option></select><select id="listbox" size="2" open><option id="listbox-option" value="b">B</option></select><input id="file" type="file" open><input id="text" type="text" open><div id="other" open></div></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("select:open"); err != nil {
		t.Fatalf("AssertExists(select:open) error = %v", err)
	}
	if err := s.AssertExists("#dropdown:open"); err != nil {
		t.Fatalf("AssertExists(#dropdown:open) error = %v", err)
	}
	if err := s.AssertExists("input:open"); err != nil {
		t.Fatalf("AssertExists(input:open) error = %v", err)
	}
	if err := s.AssertExists("#listbox:open"); err == nil {
		t.Fatalf("AssertExists(#listbox:open) error = nil, want no match")
	}
	if err := s.AssertExists("#text:open"); err == nil {
		t.Fatalf("AssertExists(#text:open) error = nil, want no match")
	}
	if err := s.AssertExists("#other:open"); err == nil {
		t.Fatalf("AssertExists(#other:open) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportHeadingLevelPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><h1 id="title">Title</h1><section><h2 id="sub">Sub</h2><div><h4 id="deep">Deep</h4></div></section><article><h6 id="final">Final</h6></article><p id="plain">Body</p></main>`
	s := NewSession(cfg)

	if err := s.AssertExists(":heading(1)"); err != nil {
		t.Fatalf("AssertExists(:heading(1)) error = %v", err)
	}
	if err := s.AssertExists(":heading(2, 4)"); err != nil {
		t.Fatalf("AssertExists(:heading(2, 4)) error = %v", err)
	}
	if err := s.AssertExists("h4:heading(4)"); err != nil {
		t.Fatalf("AssertExists(h4:heading(4)) error = %v", err)
	}
	if err := s.AssertExists("h6:heading(6)"); err != nil {
		t.Fatalf("AssertExists(h6:heading(6)) error = %v", err)
	}
}

func TestSessionAssertionsSupportMediaPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><audio id="song" src="song.mp3"></audio><video id="film"></video><video id="pip" picture-in-picture></video><video id="paused" paused></video><video id="seeking" seeking></video><video id="muted" muted></video><video id="buffering" networkstate="loading" readystate="2"></video><video id="stalled" networkstate="loading" readystate="1" stalled volume-locked></video><div id="other" paused muted></div></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("audio:playing"); err != nil {
		t.Fatalf("AssertExists(audio:playing) error = %v", err)
	}
	if err := s.AssertExists("video:picture-in-picture"); err != nil {
		t.Fatalf("AssertExists(video:picture-in-picture) error = %v", err)
	}
	if err := s.AssertExists("video:paused"); err != nil {
		t.Fatalf("AssertExists(video:paused) error = %v", err)
	}
	if err := s.AssertExists("video:seeking"); err != nil {
		t.Fatalf("AssertExists(video:seeking) error = %v", err)
	}
	if err := s.AssertExists("video:muted"); err != nil {
		t.Fatalf("AssertExists(video:muted) error = %v", err)
	}
	if err := s.AssertExists("video:buffering"); err != nil {
		t.Fatalf("AssertExists(video:buffering) error = %v", err)
	}
	if err := s.AssertExists("video:stalled"); err != nil {
		t.Fatalf("AssertExists(video:stalled) error = %v", err)
	}
	if err := s.AssertExists("video:volume-locked"); err != nil {
		t.Fatalf("AssertExists(video:volume-locked) error = %v", err)
	}
	if err := s.AssertExists("#pip:picture-in-picture"); err != nil {
		t.Fatalf("AssertExists(#pip:picture-in-picture) error = %v", err)
	}
	if err := s.AssertExists("#paused:picture-in-picture"); err == nil {
		t.Fatalf("AssertExists(#paused:picture-in-picture) error = nil, want no match")
	}
	if err := s.AssertExists("#other:paused"); err == nil {
		t.Fatalf("AssertExists(#other:paused) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportIndeterminatePseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><input id="mixed" type="checkbox" indeterminate><input id="radio-a" type="radio" name="size"><input id="radio-b" type="radio" name="size"><progress id="task"></progress><progress id="done" value="42"></progress></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:indeterminate"); err != nil {
		t.Fatalf("AssertExists(input:indeterminate) error = %v", err)
	}
	if err := s.AssertExists("progress:indeterminate"); err != nil {
		t.Fatalf("AssertExists(progress:indeterminate) error = %v", err)
	}
	if err := s.AssertExists("#radio-a:indeterminate"); err != nil {
		t.Fatalf("AssertExists(#radio-a:indeterminate) error = %v", err)
	}
	if err := s.AssertExists("#radio-b:indeterminate"); err != nil {
		t.Fatalf("AssertExists(#radio-b:indeterminate) error = %v", err)
	}

	if err := s.SetChecked("#radio-a", true); err != nil {
		t.Fatalf("SetChecked(#radio-a) error = %v", err)
	}
	if err := s.AssertExists("#radio-a:indeterminate"); err == nil {
		t.Fatalf("AssertExists(#radio-a:indeterminate) after SetChecked error = nil, want no match")
	}
	if err := s.AssertExists("#radio-b:indeterminate"); err == nil {
		t.Fatalf("AssertExists(#radio-b:indeterminate) after SetChecked error = nil, want no match")
	}
}

func TestSessionAssertionsSupportFocusPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="panel"><button id="cta">Go</button><input id="name"></section><aside id="sidebar"><input id="secondary"></aside></main>`
	s := NewSession(cfg)

	if err := s.Focus("#name"); err != nil {
		t.Fatalf("Focus(#name) error = %v", err)
	}
	if err := s.AssertExists("input:focus"); err != nil {
		t.Fatalf("AssertExists(input:focus) error = %v", err)
	}
	if err := s.AssertExists("input:focus-visible"); err != nil {
		t.Fatalf("AssertExists(input:focus-visible) error = %v", err)
	}
	if err := s.AssertExists("section:focus-within"); err != nil {
		t.Fatalf("AssertExists(section:focus-within) error = %v", err)
	}
	if err := s.AssertExists("main:focus-within"); err != nil {
		t.Fatalf("AssertExists(main:focus-within) error = %v", err)
	}
	if err := s.AssertExists("aside:focus-within"); err == nil {
		t.Fatalf("AssertExists(aside:focus-within) error = nil, want no match")
	}

	if err := s.Blur(); err != nil {
		t.Fatalf("Blur() error = %v", err)
	}
	if err := s.AssertExists("input:focus"); err == nil {
		t.Fatalf("AssertExists(input:focus) after Blur error = nil, want no match")
	}
	if err := s.AssertExists("input:focus-visible"); err == nil {
		t.Fatalf("AssertExists(input:focus-visible) after Blur error = nil, want no match")
	}
}

func TestSessionAssertionsSupportTargetPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.URL = "https://example.test/page#legacy"
	cfg.HTML = `<main id="root"><a name="legacy">legacy</a><div id="space target">space</div><p id="tail">tail</p></main>`
	s := NewSession(cfg)

	store, err := s.ensureDOM()
	if err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got := store.TargetNodeID(); got == 0 {
		t.Fatalf("TargetNodeID() after bootstrap = 0, want legacy anchor")
	}
	if err := s.AssertText("a:target", "legacy"); err != nil {
		t.Fatalf("AssertText(a:target) error = %v", err)
	}
	if err := s.AssertExists("main:target-within"); err != nil {
		t.Fatalf("AssertExists(main:target-within) after bootstrap error = %v", err)
	}

	if err := s.Navigate("#space%20target"); err != nil {
		t.Fatalf("Navigate(#space%%20target) error = %v", err)
	}
	if err := s.AssertText("div:target", "space"); err != nil {
		t.Fatalf("AssertText(div:target) error = %v", err)
	}
	if err := s.AssertExists("main:target-within"); err != nil {
		t.Fatalf("AssertExists(main:target-within) after encoded fragment error = %v", err)
	}

	if err := s.Navigate("#missing"); err != nil {
		t.Fatalf("Navigate(#missing) error = %v", err)
	}
	if err := s.AssertExists(":target"); err == nil {
		t.Fatalf("AssertExists(:target) after missing fragment error = nil, want no match")
	}
	if err := s.AssertExists(":target-within"); err == nil {
		t.Fatalf("AssertExists(:target-within) after missing fragment error = nil, want no match")
	}
	if got := store.TargetNodeID(); got != 0 {
		t.Fatalf("TargetNodeID() after missing fragment = %d, want 0", got)
	}
}

func TestSessionAssertionsSupportLangPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root" lang="en-US"><section id="panel"><p id="inherited">Hello</p></section><article id="french" lang="fr"><span id="direct">Salut</span><div id="unknown" lang=""><em id="blank">Nada</em></div></article></main>`
	s := NewSession(cfg)

	if err := s.AssertText("p:lang(en)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:lang(en)) error = %v", err)
	}
	if err := s.AssertText("span:lang(fr)", "Salut"); err != nil {
		t.Fatalf("AssertText(span:lang(fr)) error = %v", err)
	}

	if err := s.SetAttribute("#root", "lang", "fr"); err != nil {
		t.Fatalf("SetAttribute(#root, lang, fr) error = %v", err)
	}
	if err := s.AssertText("p:lang(fr)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:lang(fr)) after SetAttribute error = %v", err)
	}
	if err := s.AssertExists("p:lang(en)"); err == nil {
		t.Fatalf("AssertExists(p:lang(en)) after SetAttribute error = nil, want no match")
	}
}

func TestSessionAssertionsSupportDirPseudoClass(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root" dir="rtl"><section id="panel"><p id="inherited">Hello</p><div id="auto-ltr" dir="auto">abc</div><div id="auto-rtl" dir="auto">مرحبا</div></section><article id="ltr" dir="ltr"><span id="nested">Salut</span></article></main>`
	s := NewSession(cfg)

	if err := s.AssertText("p:dir(rtl)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:dir(rtl)) error = %v", err)
	}
	if err := s.AssertText("div:dir(ltr)", "abc"); err != nil {
		t.Fatalf("AssertText(div:dir(ltr)) error = %v", err)
	}
	if err := s.AssertText("div:dir(rtl)", "مرحبا"); err != nil {
		t.Fatalf("AssertText(div:dir(rtl)) error = %v", err)
	}
	if err := s.AssertText("span:dir(ltr)", "Salut"); err != nil {
		t.Fatalf("AssertText(span:dir(ltr)) error = %v", err)
	}

	if err := s.SetAttribute("#root", "dir", "ltr"); err != nil {
		t.Fatalf("SetAttribute(#root, dir, ltr) error = %v", err)
	}
	if err := s.AssertText("p:dir(ltr)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:dir(ltr)) after SetAttribute error = %v", err)
	}
	if err := s.AssertExists("p:dir(rtl)"); err == nil {
		t.Fatalf("AssertExists(p:dir(rtl)) after SetAttribute error = nil, want no match")
	}
}

func TestSessionAssertionsSupportOfTypePseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><section id="single"><em id="only-child">one</em></section><div id="mixed"><p id="para-a">A</p><span id="only-of-type">S</span><p id="para-b">B</p></div><details id="details" open><summary id="summary-a">A</summary><div id="middle">M</div><summary id="summary-b">B</summary></details></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("em:only-child"); err != nil {
		t.Fatalf("AssertExists(em:only-child) error = %v", err)
	}
	if err := s.AssertExists("em:only-of-type"); err != nil {
		t.Fatalf("AssertExists(em:only-of-type) error = %v", err)
	}
	if err := s.AssertExists("span:only-of-type"); err != nil {
		t.Fatalf("AssertExists(span:only-of-type) error = %v", err)
	}
	if err := s.AssertExists("summary:first-of-type"); err != nil {
		t.Fatalf("AssertExists(summary:first-of-type) error = %v", err)
	}
	if err := s.AssertExists("summary:last-of-type"); err != nil {
		t.Fatalf("AssertExists(summary:last-of-type) error = %v", err)
	}
}

func TestSessionAssertionsSupportNthPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><ul id="list"><li id="one" class="selected">1</li><li id="two">2</li><li id="three" class="selected">3</li><li id="four" class="selected">4</li><li id="five">5</li></ul><div id="mixed"><p id="para-a">A</p><span id="mid">M</span><p id="para-b">B</p><p id="para-c">C</p></div></main>`
	s := NewSession(cfg)

	if err := s.AssertText("li:nth-child(3)", "3"); err != nil {
		t.Fatalf("AssertText(li:nth-child(3)) error = %v", err)
	}
	if err := s.AssertExists("li:nth-child(odd)"); err != nil {
		t.Fatalf("AssertExists(li:nth-child(odd)) error = %v", err)
	}
	if err := s.AssertText("li:nth-child(2 of .selected)", "3"); err != nil {
		t.Fatalf("AssertText(li:nth-child(2 of .selected)) error = %v", err)
	}
	if err := s.AssertText("li:nth-child(2 of .selected, #two)", "2"); err != nil {
		t.Fatalf("AssertText(li:nth-child(2 of .selected, #two)) error = %v", err)
	}
	if err := s.AssertText("p:nth-of-type(3)", "C"); err != nil {
		t.Fatalf("AssertText(p:nth-of-type(3)) error = %v", err)
	}
	if err := s.AssertText("li:nth-last-child(1)", "5"); err != nil {
		t.Fatalf("AssertText(li:nth-last-child(1)) error = %v", err)
	}
	if err := s.AssertText("li:nth-last-child(1 of .selected)", "4"); err != nil {
		t.Fatalf("AssertText(li:nth-last-child(1 of .selected)) error = %v", err)
	}
	if err := s.AssertText("p:nth-last-of-type(2)", "B"); err != nil {
		t.Fatalf("AssertText(p:nth-last-of-type(2)) error = %v", err)
	}
	if err := s.AssertExists("span:nth-of-type(2)"); err == nil {
		t.Fatalf("AssertExists(span:nth-of-type(2)) error = nil, want no match")
	}
	if err := s.AssertExists("li:nth-last-child(6)"); err == nil {
		t.Fatalf("AssertExists(li:nth-last-child(6)) error = nil, want no match")
	}
}

func TestSessionAssertionsSupportConstraintValidationPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main id="root"><form id="valid-form"><input id="name" type="text" required value="Ada"><input id="age" type="number" min="1" max="10" value="5"><select id="mode"><option value="a" selected>A</option><option value="b">B</option></select></form><form id="invalid-form"><input id="missing" type="text" required><input id="low" type="number" min="1" max="10" value="0"><input id="high" type="number" min="1" max="10" value="11"></form></main>`
	s := NewSession(cfg)

	if err := s.AssertExists("input:valid"); err != nil {
		t.Fatalf("AssertExists(input:valid) error = %v", err)
	}
	if err := s.AssertExists("input:invalid"); err != nil {
		t.Fatalf("AssertExists(input:invalid) error = %v", err)
	}
	if err := s.AssertExists("input:in-range"); err != nil {
		t.Fatalf("AssertExists(input:in-range) error = %v", err)
	}
	if err := s.AssertExists("input:out-of-range"); err != nil {
		t.Fatalf("AssertExists(input:out-of-range) error = %v", err)
	}
	if err := s.AssertExists("select:valid"); err != nil {
		t.Fatalf("AssertExists(select:valid) error = %v", err)
	}
	if err := s.AssertExists("form:valid"); err != nil {
		t.Fatalf("AssertExists(form:valid) error = %v", err)
	}
	if err := s.AssertExists("form:invalid"); err != nil {
		t.Fatalf("AssertExists(form:invalid) error = %v", err)
	}
}

func TestSessionAssertionsSupportUserValidityPseudoClasses(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.HTML = `<main><form id="profile"><input id="name" type="text" required><input id="agree" type="checkbox" required checked><select id="mode" required><option value="a">A</option><option value="b" selected>B</option></select></form></main>`
	s := NewSession(cfg)

	if err := s.TypeText("#name", "Ada"); err != nil {
		t.Fatalf("TypeText(#name) error = %v", err)
	}
	if err := s.SetChecked("#agree", false); err != nil {
		t.Fatalf("SetChecked(#agree) error = %v", err)
	}
	if err := s.SetSelectValue("#mode", "a"); err != nil {
		t.Fatalf("SetSelectValue(#mode) error = %v", err)
	}

	if err := s.AssertExists("input:user-valid"); err != nil {
		t.Fatalf("AssertExists(input:user-valid) error = %v", err)
	}
	if err := s.AssertExists("input:user-invalid"); err != nil {
		t.Fatalf("AssertExists(input:user-invalid) error = %v", err)
	}
	if err := s.AssertExists("select:user-valid"); err != nil {
		t.Fatalf("AssertExists(select:user-valid) error = %v", err)
	}
}
