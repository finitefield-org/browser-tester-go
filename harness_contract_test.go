package browsertester

import (
	"strings"
	"testing"
)

func TestDebugViewReportsRandomSeedWhenConfigured(t *testing.T) {
	harness, err := NewHarnessBuilder().RandomSeed(42).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	got, ok := harness.Debug().RandomSeed()
	if !ok {
		t.Fatalf("Debug().RandomSeed() ok = false, want true")
	}
	if got != 42 {
		t.Fatalf("Debug().RandomSeed() = %d, want 42", got)
	}

	defaultHarness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() default error = %v", err)
	}
	if got, ok := defaultHarness.Debug().RandomSeed(); ok || got != 0 {
		t.Fatalf("default Debug().RandomSeed() = (%d, %v), want (0, false)", got, ok)
	}
}

func TestDebugViewReportsBuilderFailures(t *testing.T) {
	harness, err := NewHarnessBuilder().
		OpenFailure("open blocked").
		CloseFailure("close blocked").
		PrintFailure("print blocked").
		ScrollFailure("scroll blocked").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got, want := harness.Debug().OpenFailure(), "open blocked"; got != want {
		t.Fatalf("Debug().OpenFailure() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().CloseFailure(), "close blocked"; got != want {
		t.Fatalf("Debug().CloseFailure() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().PrintFailure(), "print blocked"; got != want {
		t.Fatalf("Debug().PrintFailure() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().ScrollFailure(), "scroll blocked"; got != want {
		t.Fatalf("Debug().ScrollFailure() = %q, want %q", got, want)
	}
	if got := harness.Debug().OptionLabels(); len(got) != 0 {
		t.Fatalf("Debug().OptionLabels() = %#v, want empty before DOM bootstrap", got)
	}
	if got := harness.Debug().OptionValues(); len(got) != 0 {
		t.Fatalf("Debug().OptionValues() = %#v, want empty before DOM bootstrap", got)
	}
	if got := harness.Debug().SelectCount(); got != 0 {
		t.Fatalf("Debug().SelectCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().TemplateCount(); got != 0 {
		t.Fatalf("Debug().TemplateCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().TableCount(); got != 0 {
		t.Fatalf("Debug().TableCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().ButtonCount(); got != 0 {
		t.Fatalf("Debug().ButtonCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().TextAreaCount(); got != 0 {
		t.Fatalf("Debug().TextAreaCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().InputCount(); got != 0 {
		t.Fatalf("Debug().InputCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().OptionCount(); got != 0 {
		t.Fatalf("Debug().OptionCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().SelectedOptionCount(); got != 0 {
		t.Fatalf("Debug().SelectedOptionCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().OptgroupCount(); got != 0 {
		t.Fatalf("Debug().OptgroupCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().LinkCount(); got != 0 {
		t.Fatalf("Debug().LinkCount() = %d, want 0 before DOM bootstrap", got)
	}
	if got := harness.Debug().AnchorCount(); got != 0 {
		t.Fatalf("Debug().AnchorCount() = %d, want 0 before DOM bootstrap", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().OpenFailure(); got != "" {
		t.Fatalf("nil Debug().OpenFailure() = %q, want empty", got)
	}
	if got := nilHarness.Debug().CloseFailure(); got != "" {
		t.Fatalf("nil Debug().CloseFailure() = %q, want empty", got)
	}
	if got := nilHarness.Debug().PrintFailure(); got != "" {
		t.Fatalf("nil Debug().PrintFailure() = %q, want empty", got)
	}
	if got := nilHarness.Debug().ScrollFailure(); got != "" {
		t.Fatalf("nil Debug().ScrollFailure() = %q, want empty", got)
	}
	if got := nilHarness.Debug().OptionLabels(); got != nil {
		t.Fatalf("nil Debug().OptionLabels() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().SelectedOptionLabels(); got != nil {
		t.Fatalf("nil Debug().SelectedOptionLabels() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().OptionValues(); got != nil {
		t.Fatalf("nil Debug().OptionValues() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().SelectedOptionValues(); got != nil {
		t.Fatalf("nil Debug().SelectedOptionValues() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().SelectCount(); got != 0 {
		t.Fatalf("nil Debug().SelectCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().TemplateCount(); got != 0 {
		t.Fatalf("nil Debug().TemplateCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().TableCount(); got != 0 {
		t.Fatalf("nil Debug().TableCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().ButtonCount(); got != 0 {
		t.Fatalf("nil Debug().ButtonCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().TextAreaCount(); got != 0 {
		t.Fatalf("nil Debug().TextAreaCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().InputCount(); got != 0 {
		t.Fatalf("nil Debug().InputCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().FieldsetCount(); got != 0 {
		t.Fatalf("nil Debug().FieldsetCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().LegendCount(); got != 0 {
		t.Fatalf("nil Debug().LegendCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().OutputCount(); got != 0 {
		t.Fatalf("nil Debug().OutputCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().LabelCount(); got != 0 {
		t.Fatalf("nil Debug().LabelCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().ProgressCount(); got != 0 {
		t.Fatalf("nil Debug().ProgressCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().MeterCount(); got != 0 {
		t.Fatalf("nil Debug().MeterCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().AudioCount(); got != 0 {
		t.Fatalf("nil Debug().AudioCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().VideoCount(); got != 0 {
		t.Fatalf("nil Debug().VideoCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().IframeCount(); got != 0 {
		t.Fatalf("nil Debug().IframeCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().EmbedCount(); got != 0 {
		t.Fatalf("nil Debug().EmbedCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().TrackCount(); got != 0 {
		t.Fatalf("nil Debug().TrackCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().PictureCount(); got != 0 {
		t.Fatalf("nil Debug().PictureCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SourceCount(); got != 0 {
		t.Fatalf("nil Debug().SourceCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().DialogCount(); got != 0 {
		t.Fatalf("nil Debug().DialogCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().DetailsCount(); got != 0 {
		t.Fatalf("nil Debug().DetailsCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SummaryCount(); got != 0 {
		t.Fatalf("nil Debug().SummaryCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SectionCount(); got != 0 {
		t.Fatalf("nil Debug().SectionCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().MainCount(); got != 0 {
		t.Fatalf("nil Debug().MainCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().ArticleCount(); got != 0 {
		t.Fatalf("nil Debug().ArticleCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().NavCount(); got != 0 {
		t.Fatalf("nil Debug().NavCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().AsideCount(); got != 0 {
		t.Fatalf("nil Debug().AsideCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().FigureCount(); got != 0 {
		t.Fatalf("nil Debug().FigureCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().FigcaptionCount(); got != 0 {
		t.Fatalf("nil Debug().FigcaptionCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().HeaderCount(); got != 0 {
		t.Fatalf("nil Debug().HeaderCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().FooterCount(); got != 0 {
		t.Fatalf("nil Debug().FooterCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().AddressCount(); got != 0 {
		t.Fatalf("nil Debug().AddressCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().BlockquoteCount(); got != 0 {
		t.Fatalf("nil Debug().BlockquoteCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().ParagraphCount(); got != 0 {
		t.Fatalf("nil Debug().ParagraphCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().PreCount(); got != 0 {
		t.Fatalf("nil Debug().PreCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().MarkCount(); got != 0 {
		t.Fatalf("nil Debug().MarkCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().QCount(); got != 0 {
		t.Fatalf("nil Debug().QCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().CiteCount(); got != 0 {
		t.Fatalf("nil Debug().CiteCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().AbbrCount(); got != 0 {
		t.Fatalf("nil Debug().AbbrCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().StrongCount(); got != 0 {
		t.Fatalf("nil Debug().StrongCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SpanCount(); got != 0 {
		t.Fatalf("nil Debug().SpanCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().DataCount(); got != 0 {
		t.Fatalf("nil Debug().DataCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().DfnCount(); got != 0 {
		t.Fatalf("nil Debug().DfnCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().KbdCount(); got != 0 {
		t.Fatalf("nil Debug().KbdCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SampCount(); got != 0 {
		t.Fatalf("nil Debug().SampCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().RubyCount(); got != 0 {
		t.Fatalf("nil Debug().RubyCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().RtCount(); got != 0 {
		t.Fatalf("nil Debug().RtCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().VarCount(); got != 0 {
		t.Fatalf("nil Debug().VarCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().CodeCount(); got != 0 {
		t.Fatalf("nil Debug().CodeCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SmallCount(); got != 0 {
		t.Fatalf("nil Debug().SmallCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().TimeCount(); got != 0 {
		t.Fatalf("nil Debug().TimeCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().OptionCount(); got != 0 {
		t.Fatalf("nil Debug().OptionCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().SelectedOptionCount(); got != 0 {
		t.Fatalf("nil Debug().SelectedOptionCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().OptgroupCount(); got != 0 {
		t.Fatalf("nil Debug().OptgroupCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().LinkCount(); got != 0 {
		t.Fatalf("nil Debug().LinkCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().AnchorCount(); got != 0 {
		t.Fatalf("nil Debug().AnchorCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().OptgroupLabels(); got != nil {
		t.Fatalf("nil Debug().OptgroupLabels() = %#v, want nil", got)
	}
}

func TestDebugViewReportsOptionLabels(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode"><option id="first" label="Display">Fallback</option><option id="second">Text</option><option id="empty" label="">Used text</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	labels := harness.Debug().OptionLabels()
	if len(labels) != 3 {
		t.Fatalf("Debug().OptionLabels() len = %d, want 3", len(labels))
	}
	if labels[0].Label != "Display" || labels[1].Label != "Text" || labels[2].Label != "Used text" {
		t.Fatalf("Debug().OptionLabels() = %#v, want labels in document order", labels)
	}
	if labels[0].NodeID == 0 || labels[1].NodeID == 0 || labels[2].NodeID == 0 {
		t.Fatalf("Debug().OptionLabels() = %#v, want node IDs", labels)
	}

	labels[0].Label = "mutated"
	labels[1].NodeID = 999
	fresh := harness.Debug().OptionLabels()
	if len(fresh) != 3 || fresh[0].Label != "Display" || fresh[1].Label != "Text" || fresh[2].Label != "Used text" {
		t.Fatalf("Debug().OptionLabels() reread = %#v, want original labels", fresh)
	}
}

func TestDebugViewReportsSelectedOptionLabels(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode" multiple><option id="first" label="Display" selected>Fallback</option><option id="second">Text</option><option id="third" selected>Third</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	labels := harness.Debug().SelectedOptionLabels()
	if len(labels) != 2 {
		t.Fatalf("Debug().SelectedOptionLabels() len = %d, want 2", len(labels))
	}
	if labels[0].Label != "Display" || labels[1].Label != "Third" {
		t.Fatalf("Debug().SelectedOptionLabels() = %#v, want selected labels in document order", labels)
	}
	if labels[0].NodeID == 0 || labels[1].NodeID == 0 {
		t.Fatalf("Debug().SelectedOptionLabels() = %#v, want node IDs", labels)
	}

	labels[0].Label = "mutated"
	labels[1].NodeID = 999
	fresh := harness.Debug().SelectedOptionLabels()
	if len(fresh) != 2 || fresh[0].Label != "Display" || fresh[1].Label != "Third" {
		t.Fatalf("Debug().SelectedOptionLabels() reread = %#v, want original labels", fresh)
	}
}

func TestDebugViewReportsOptionValues(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode"><option id="first" value="Display">Fallback</option><option id="second">Text</option><option id="empty" value="">Used text</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	values := harness.Debug().OptionValues()
	if len(values) != 3 {
		t.Fatalf("Debug().OptionValues() len = %d, want 3", len(values))
	}
	if values[0].Value != "Display" || values[1].Value != "Text" || values[2].Value != "" {
		t.Fatalf("Debug().OptionValues() = %#v, want values in document order", values)
	}
	if values[0].NodeID == 0 || values[1].NodeID == 0 || values[2].NodeID == 0 {
		t.Fatalf("Debug().OptionValues() = %#v, want node IDs", values)
	}

	values[0].Value = "mutated"
	values[1].NodeID = 999
	fresh := harness.Debug().OptionValues()
	if len(fresh) != 3 || fresh[0].Value != "Display" || fresh[1].Value != "Text" || fresh[2].Value != "" {
		t.Fatalf("Debug().OptionValues() reread = %#v, want original values", fresh)
	}
}

func TestDebugViewReportsSelectedOptionValues(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode" multiple><option id="first" value="Display" selected>Fallback</option><option id="second">Text</option><option id="third" selected>Third</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	values := harness.Debug().SelectedOptionValues()
	if len(values) != 2 {
		t.Fatalf("Debug().SelectedOptionValues() len = %d, want 2", len(values))
	}
	if values[0].Value != "Display" || values[1].Value != "Third" {
		t.Fatalf("Debug().SelectedOptionValues() = %#v, want selected values in document order", values)
	}
	if values[0].NodeID == 0 || values[1].NodeID == 0 {
		t.Fatalf("Debug().SelectedOptionValues() = %#v, want node IDs", values)
	}

	values[0].Value = "mutated"
	values[1].NodeID = 999
	fresh := harness.Debug().SelectedOptionValues()
	if len(fresh) != 2 || fresh[0].Value != "Display" || fresh[1].Value != "Third" {
		t.Fatalf("Debug().SelectedOptionValues() reread = %#v, want original values", fresh)
	}
}

func TestDebugViewReportsOptionCount(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode"><option id="first" value="Display">Fallback</option><option id="second">Text</option><div><option id="third">Ignored</option></div></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().OptionCount(), 3; got != want {
		t.Fatalf("Debug().OptionCount() = %d, want %d", got, want)
	}
	if got, want := harness.Debug().SelectedOptionCount(), 0; got != want {
		t.Fatalf("Debug().SelectedOptionCount() = %d, want %d", got, want)
	}
}

func TestDebugViewReportsSelectedOptionCount(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode" multiple><option id="first" value="Display" selected>Fallback</option><option id="second">Text</option><option id="third" selected>Third</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().OptionCount(), 3; got != want {
		t.Fatalf("Debug().OptionCount() = %d, want %d", got, want)
	}
	if got, want := harness.Debug().SelectedOptionCount(), 2; got != want {
		t.Fatalf("Debug().SelectedOptionCount() = %d, want %d", got, want)
	}
}

func TestDebugViewReportsOptgroupCount(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode"><optgroup id="first" label="Group"><option>A</option></optgroup><optgroup id="second"><legend>Legend</legend><option>B</option></optgroup><div><optgroup id="ignored" label="Nope"></optgroup></div></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().OptgroupCount(), 3; got != want {
		t.Fatalf("Debug().OptgroupCount() = %d, want %d", got, want)
	}
}

func TestDebugViewReportsLinkAndAnchorCount(t *testing.T) {
	harness, err := FromHTML(`<main><a href="/one">One</a><a name="anchor">Anchor</a><area href="/area" alt="Area"><div><a href="/two">Two</a><a name="inner">Inner</a></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().LinkCount(), 3; got != want {
		t.Fatalf("Debug().LinkCount() = %d, want %d", got, want)
	}
	if got, want := harness.Debug().AnchorCount(), 2; got != want {
		t.Fatalf("Debug().AnchorCount() = %d, want %d", got, want)
	}
	if !harness.Debug().DOMReady() {
		t.Fatalf("Debug().DOMReady() = false, want true after link count bootstrap")
	}
	if got := harness.Debug().DOMError(); got != "" {
		t.Fatalf("Debug().DOMError() = %q, want empty after link count bootstrap", got)
	}
}

func TestDebugViewReportsOptgroupLabels(t *testing.T) {
	harness, err := FromHTML(`<main><select><optgroup id="plain" label="Group"><option>A</option></optgroup><optgroup id="legend"><legend> Legend  Name </legend><option>B</option></optgroup></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	labels := harness.Debug().OptgroupLabels()
	if len(labels) != 2 {
		t.Fatalf("Debug().OptgroupLabels() len = %d, want 2", len(labels))
	}
	if labels[0].Label != "Group" || labels[1].Label != "Legend Name" {
		t.Fatalf("Debug().OptgroupLabels() = %#v, want labels in document order", labels)
	}
	if labels[0].NodeID == 0 || labels[1].NodeID == 0 {
		t.Fatalf("Debug().OptgroupLabels() = %#v, want node IDs", labels)
	}

	labels[0].Label = "mutated"
	labels[1].NodeID = 999
	fresh := harness.Debug().OptgroupLabels()
	if len(fresh) != 2 || fresh[0].Label != "Group" || fresh[1].Label != "Legend Name" {
		t.Fatalf("Debug().OptgroupLabels() reread = %#v, want original labels", fresh)
	}
}

func TestMatchMediaContract(t *testing.T) {
	harness, err := NewHarnessBuilder().
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	got, err := harness.MatchMedia("(prefers-reduced-motion: reduce)")
	if err != nil {
		t.Fatalf("MatchMedia() error = %v", err)
	}
	if !got {
		t.Fatalf("MatchMedia() = false, want true")
	}

	if _, err := harness.MatchMedia("(prefers-color-scheme: dark)"); err == nil {
		t.Fatalf("MatchMedia(unseeded) error = nil, want mock error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindMock {
		t.Fatalf("MatchMedia(unseeded) error = %#v, want mock error", err)
	}

	rules := harness.Mocks().MatchMedia().Rules()
	if len(rules) != 1 || rules[0].Query != "(prefers-reduced-motion: reduce)" || !rules[0].Matches {
		t.Fatalf("MatchMedia().Rules() = %#v, want one seeded rule", rules)
	}
	rules[0].Matches = false
	if got := harness.Mocks().MatchMedia().Rules(); len(got) != 1 || got[0].Matches != true {
		t.Fatalf("MatchMedia().Rules() reread = %#v, want original rule", got)
	}
}

func TestMatchMediaListenerCallsReturnCopies(t *testing.T) {
	harness, err := NewHarnessBuilder().
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	harness.Mocks().MatchMedia().RecordListenerCall("(prefers-reduced-motion: reduce)", "addListener")
	harness.Mocks().MatchMedia().RecordListenerCall("(prefers-reduced-motion: reduce)", "removeListener")

	listeners := harness.Mocks().MatchMedia().ListenerCalls()
	if len(listeners) != 2 || listeners[0].Query != "(prefers-reduced-motion: reduce)" || listeners[0].Method != "addListener" || listeners[1].Method != "removeListener" {
		t.Fatalf("MatchMedia().ListenerCalls() = %#v, want both listener calls", listeners)
	}

	listeners[0].Query = "mutated"
	listeners[1].Method = "mutated"

	fresh := harness.Mocks().MatchMedia().ListenerCalls()
	if len(fresh) != 2 || fresh[0].Query != "(prefers-reduced-motion: reduce)" || fresh[0].Method != "addListener" || fresh[1].Method != "removeListener" {
		t.Fatalf("MatchMedia().ListenerCalls() reread = %#v, want original listener calls", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().MatchMedia(); got != nil {
		t.Fatalf("nil Harness.Mocks().MatchMedia() = %#v, want nil", got)
	}
}

func TestDebugViewReportsScrollPosition(t *testing.T) {
	harness, err := FromHTML(`<main><div>scroll</div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if gotX, gotY := harness.Debug().ScrollPosition(); gotX != 0 || gotY != 0 {
		t.Fatalf("Debug().ScrollPosition() = (%d, %d), want (0, 0)", gotX, gotY)
	}

	if err := harness.ScrollTo(13, 21); err != nil {
		t.Fatalf("ScrollTo() error = %v", err)
	}
	if gotX, gotY := harness.Debug().ScrollPosition(); gotX != 13 || gotY != 21 {
		t.Fatalf("Debug().ScrollPosition() after ScrollTo = (%d, %d), want (13, 21)", gotX, gotY)
	}

	if err := harness.ScrollBy(2, -1); err != nil {
		t.Fatalf("ScrollBy() error = %v", err)
	}
	if gotX, gotY := harness.Debug().ScrollPosition(); gotX != 15 || gotY != 20 {
		t.Fatalf("Debug().ScrollPosition() after ScrollBy = (%d, %d), want (15, 20)", gotX, gotY)
	}

	var nilHarness *Harness
	if gotX, gotY := nilHarness.Debug().ScrollPosition(); gotX != 0 || gotY != 0 {
		t.Fatalf("nil Debug().ScrollPosition() = (%d, %d), want (0, 0)", gotX, gotY)
	}
}

func TestDebugViewReportsTargetNodeID(t *testing.T) {
	harness, err := FromHTMLWithURL("https://example.test/page#target", `<main><a id="target">Target</a><p id="other">Other</p></main>`)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got := harness.Debug().TargetNodeID(); got == 0 {
		t.Fatalf("Debug().TargetNodeID() = %d, want targeted node id", got)
	}

	if err := harness.Navigate("#missing"); err != nil {
		t.Fatalf("Navigate(#missing) error = %v", err)
	}
	if got := harness.Debug().TargetNodeID(); got != 0 {
		t.Fatalf("Debug().TargetNodeID() after missing fragment = %d, want 0", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().TargetNodeID(); got != 0 {
		t.Fatalf("nil Debug().TargetNodeID() = %d, want 0", got)
	}
}

func TestDebugViewReportsFocusedNodeID(t *testing.T) {
	harness, err := FromHTML(`<main><input id="field" type="text" value="hello"><p id="other">other</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Focus("#field"); err != nil {
		t.Fatalf("Focus(#field) error = %v", err)
	}
	if got := harness.Debug().FocusedNodeID(); got == 0 {
		t.Fatalf("Debug().FocusedNodeID() = %d, want focused node id", got)
	}

	if err := harness.Blur(); err != nil {
		t.Fatalf("Blur() error = %v", err)
	}
	if got := harness.Debug().FocusedNodeID(); got != 0 {
		t.Fatalf("Debug().FocusedNodeID() after blur = %d, want 0", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FocusedNodeID(); got != 0 {
		t.Fatalf("nil Debug().FocusedNodeID() = %d, want 0", got)
	}
}

func TestDebugViewReportsNodeCount(t *testing.T) {
	harness, err := FromHTML(`<main><section><p id="one">one</p><p id="two">two</p></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("main"); err != nil {
		t.Fatalf("AssertExists(main) error = %v", err)
	}

	if got := harness.Debug().NodeCount(); got == 0 {
		t.Fatalf("Debug().NodeCount() = %d, want non-zero", got)
	}
	got := harness.Debug().NodeCount()
	if again := harness.Debug().NodeCount(); got != again {
		t.Fatalf("Debug().NodeCount() reread = %d, want stable result %d", again, got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().NodeCount(); got != 0 {
		t.Fatalf("nil Debug().NodeCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsScriptCount(t *testing.T) {
	harness, err := FromHTML(`<main><script></script><div id="host"></div><script>host:setTextContent(#host, changed)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().ScriptCount(); got != 2 {
		t.Fatalf("Debug().ScriptCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", "<script></script>"); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().ScriptCount(); got != 3 {
		t.Fatalf("Debug().ScriptCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().ScriptCount(); got != 0 {
		t.Fatalf("nil Debug().ScriptCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsImageCount(t *testing.T) {
	harness, err := FromHTML(`<main><img id="first" src="/a"><div id="host"></div><img name="second" src="/b"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().ImageCount(); got != 2 {
		t.Fatalf("Debug().ImageCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", "<img>"); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().ImageCount(); got != 3 {
		t.Fatalf("Debug().ImageCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().ImageCount(); got != 0 {
		t.Fatalf("nil Debug().ImageCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsFormCount(t *testing.T) {
	harness, err := FromHTML(`<main><form id="first"></form><div id="host"></div><form name="second"></form></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().FormCount(); got != 2 {
		t.Fatalf("Debug().FormCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", "<form></form>"); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().FormCount(); got != 3 {
		t.Fatalf("Debug().FormCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FormCount(); got != 0 {
		t.Fatalf("nil Debug().FormCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSelectCount(t *testing.T) {
	harness, err := FromHTML(`<main><select id="first"></select><div id="host"></div><div><select name="second"></select></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SelectCount(); got != 2 {
		t.Fatalf("Debug().SelectCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<select id="third"></select>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SelectCount(); got != 3 {
		t.Fatalf("Debug().SelectCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SelectCount(); got != 0 {
		t.Fatalf("nil Debug().SelectCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsTemplateCount(t *testing.T) {
	harness, err := FromHTML(`<main><template id="first"></template><div id="host"></div><div><template name="second"></template></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().TemplateCount(); got != 2 {
		t.Fatalf("Debug().TemplateCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<template id="third"></template>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().TemplateCount(); got != 3 {
		t.Fatalf("Debug().TemplateCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().TemplateCount(); got != 0 {
		t.Fatalf("nil Debug().TemplateCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsTableCount(t *testing.T) {
	harness, err := FromHTML(`<main><table id="first"></table><div id="host"></div><div><table name="second"></table></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().TableCount(); got != 2 {
		t.Fatalf("Debug().TableCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<table id="third"></table>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().TableCount(); got != 3 {
		t.Fatalf("Debug().TableCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().TableCount(); got != 0 {
		t.Fatalf("nil Debug().TableCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsButtonCount(t *testing.T) {
	harness, err := FromHTML(`<main><button id="first"></button><div id="host"></div><div><button name="second"></button></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().ButtonCount(); got != 2 {
		t.Fatalf("Debug().ButtonCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<button id="third"></button>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().ButtonCount(); got != 3 {
		t.Fatalf("Debug().ButtonCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().ButtonCount(); got != 0 {
		t.Fatalf("nil Debug().ButtonCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsTextAreaCount(t *testing.T) {
	harness, err := FromHTML(`<main><textarea id="first"></textarea><div id="host"></div><div><textarea name="second"></textarea></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().TextAreaCount(); got != 2 {
		t.Fatalf("Debug().TextAreaCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<textarea id="third"></textarea>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().TextAreaCount(); got != 3 {
		t.Fatalf("Debug().TextAreaCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().TextAreaCount(); got != 0 {
		t.Fatalf("nil Debug().TextAreaCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsInputCount(t *testing.T) {
	harness, err := FromHTML(`<main><input id="first"><div id="host"></div><div><input name="second"></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().InputCount(); got != 2 {
		t.Fatalf("Debug().InputCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<input id="third">`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().InputCount(); got != 3 {
		t.Fatalf("Debug().InputCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().InputCount(); got != 0 {
		t.Fatalf("nil Debug().InputCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsFieldsetCount(t *testing.T) {
	harness, err := FromHTML(`<main><fieldset id="first"></fieldset><div id="host"></div><div><fieldset name="second"></fieldset></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().FieldsetCount(); got != 2 {
		t.Fatalf("Debug().FieldsetCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<fieldset id="third"></fieldset>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().FieldsetCount(); got != 3 {
		t.Fatalf("Debug().FieldsetCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FieldsetCount(); got != 0 {
		t.Fatalf("nil Debug().FieldsetCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsLegendCount(t *testing.T) {
	harness, err := FromHTML(`<main><fieldset><legend id="first"></legend></fieldset><div id="host"></div><div><legend name="second"></legend></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().LegendCount(); got != 2 {
		t.Fatalf("Debug().LegendCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<legend id="third"></legend>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().LegendCount(); got != 3 {
		t.Fatalf("Debug().LegendCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().LegendCount(); got != 0 {
		t.Fatalf("nil Debug().LegendCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsOutputCount(t *testing.T) {
	harness, err := FromHTML(`<main><output id="first"></output><div id="host"></div><div><output name="second"></output></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().OutputCount(); got != 2 {
		t.Fatalf("Debug().OutputCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<output id="third"></output>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().OutputCount(); got != 3 {
		t.Fatalf("Debug().OutputCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().OutputCount(); got != 0 {
		t.Fatalf("nil Debug().OutputCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsLabelCount(t *testing.T) {
	harness, err := FromHTML(`<main><label id="first">A</label><div id="host"></div><div><label name="second">B</label></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().LabelCount(); got != 2 {
		t.Fatalf("Debug().LabelCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<label id="third">C</label>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().LabelCount(); got != 3 {
		t.Fatalf("Debug().LabelCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().LabelCount(); got != 0 {
		t.Fatalf("nil Debug().LabelCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsProgressCount(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><input id="mixed" type="checkbox" indeterminate><input id="radio-a" type="radio" name="size"><input id="radio-b" type="radio" name="size"><progress id="task"></progress><progress id="done" value="42"></progress></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().ProgressCount(); got != 2 {
		t.Fatalf("Debug().ProgressCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#root", `<progress id="third"></progress>`); err != nil {
		t.Fatalf("SetInnerHTML(#root) error = %v", err)
	}
	if got := harness.Debug().ProgressCount(); got != 1 {
		t.Fatalf("Debug().ProgressCount() after SetInnerHTML = %d, want 1", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().ProgressCount(); got != 0 {
		t.Fatalf("nil Debug().ProgressCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsMeterCount(t *testing.T) {
	harness, err := FromHTML(`<main><meter id="first"></meter><div id="host"></div><div><meter name="second" value="3"></meter></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().MeterCount(); got != 2 {
		t.Fatalf("Debug().MeterCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<meter id="third"></meter>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().MeterCount(); got != 3 {
		t.Fatalf("Debug().MeterCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().MeterCount(); got != 0 {
		t.Fatalf("nil Debug().MeterCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsAudioAndVideoCount(t *testing.T) {
	harness, err := FromHTML(`<main><audio id="first"></audio><video id="second"></video><div id="host"></div><section><audio id="third"></audio><video id="fourth"></video></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().AudioCount(); got != 2 {
		t.Fatalf("Debug().AudioCount() = %d, want 2", got)
	}
	if got := harness.Debug().VideoCount(); got != 2 {
		t.Fatalf("Debug().VideoCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<audio id="fifth"></audio><video id="sixth"></video>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().AudioCount(); got != 3 {
		t.Fatalf("Debug().AudioCount() after SetInnerHTML = %d, want 3", got)
	}
	if got := harness.Debug().VideoCount(); got != 3 {
		t.Fatalf("Debug().VideoCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().AudioCount(); got != 0 {
		t.Fatalf("nil Debug().AudioCount() = %d, want 0", got)
	}
	if got := nilHarness.Debug().VideoCount(); got != 0 {
		t.Fatalf("nil Debug().VideoCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsIframeCount(t *testing.T) {
	harness, err := FromHTML(`<main><iframe id="first"></iframe><div id="host"></div><section><iframe id="second"></iframe></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().IframeCount(); got != 2 {
		t.Fatalf("Debug().IframeCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<iframe id="third"></iframe>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().IframeCount(); got != 3 {
		t.Fatalf("Debug().IframeCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().IframeCount(); got != 0 {
		t.Fatalf("nil Debug().IframeCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsEmbedCount(t *testing.T) {
	harness, err := FromHTML(`<main><embed id="first"><div id="host"></div><section><embed id="second"></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().EmbedCount(); got != 2 {
		t.Fatalf("Debug().EmbedCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<embed id="third">`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().EmbedCount(); got != 3 {
		t.Fatalf("Debug().EmbedCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().EmbedCount(); got != 0 {
		t.Fatalf("nil Debug().EmbedCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsTrackCount(t *testing.T) {
	harness, err := FromHTML(`<main><track id="first"><div id="host"></div><section><track id="second"></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().TrackCount(); got != 2 {
		t.Fatalf("Debug().TrackCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<track id="third">`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().TrackCount(); got != 3 {
		t.Fatalf("Debug().TrackCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().TrackCount(); got != 0 {
		t.Fatalf("nil Debug().TrackCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsPictureCount(t *testing.T) {
	harness, err := FromHTML(`<main><picture id="first"></picture><div id="host"></div><section><picture id="second"></picture></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().PictureCount(); got != 2 {
		t.Fatalf("Debug().PictureCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<picture id="third"></picture>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().PictureCount(); got != 3 {
		t.Fatalf("Debug().PictureCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().PictureCount(); got != 0 {
		t.Fatalf("nil Debug().PictureCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSourceCount(t *testing.T) {
	harness, err := FromHTML(`<main><picture><source id="first"><source id="second"></picture><div id="host"></div><audio><source id="third"></audio></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SourceCount(); got != 3 {
		t.Fatalf("Debug().SourceCount() = %d, want 3", got)
	}

	if err := harness.SetInnerHTML("#host", `<video><source id="fourth"></video>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SourceCount(); got != 4 {
		t.Fatalf("Debug().SourceCount() after SetInnerHTML = %d, want 4", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SourceCount(); got != 0 {
		t.Fatalf("nil Debug().SourceCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsDialogCount(t *testing.T) {
	harness, err := FromHTML(`<main><dialog id="first"></dialog><div id="host"></div><section><dialog id="second"></dialog></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().DialogCount(); got != 2 {
		t.Fatalf("Debug().DialogCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<dialog id="third"></dialog>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().DialogCount(); got != 3 {
		t.Fatalf("Debug().DialogCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DialogCount(); got != 0 {
		t.Fatalf("nil Debug().DialogCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsDetailsCount(t *testing.T) {
	harness, err := FromHTML(`<main><details id="first"></details><div id="host"></div><section><details id="second"></details></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().DetailsCount(); got != 2 {
		t.Fatalf("Debug().DetailsCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<details id="third"></details>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().DetailsCount(); got != 3 {
		t.Fatalf("Debug().DetailsCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DetailsCount(); got != 0 {
		t.Fatalf("nil Debug().DetailsCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSummaryCount(t *testing.T) {
	harness, err := FromHTML(`<main><details id="first"><summary id="one">One</summary></details><div id="host"></div><section><details id="second"><summary id="two">Two</summary></details></section></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SummaryCount(); got != 2 {
		t.Fatalf("Debug().SummaryCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<details id="third"><summary id="three">Three</summary></details>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SummaryCount(); got != 3 {
		t.Fatalf("Debug().SummaryCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SummaryCount(); got != 0 {
		t.Fatalf("nil Debug().SummaryCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSectionCount(t *testing.T) {
	harness, err := FromHTML(`<main><section id="first"></section><div id="host"></div><article><section id="second"></section></article></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SectionCount(); got != 2 {
		t.Fatalf("Debug().SectionCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<section id="third"></section>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SectionCount(); got != 3 {
		t.Fatalf("Debug().SectionCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SectionCount(); got != 0 {
		t.Fatalf("nil Debug().SectionCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsMainCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><main id="first"></main><div id="host"></div><section><main id="second"></main></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().MainCount(); got != 2 {
		t.Fatalf("Debug().MainCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<main id="third"></main>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().MainCount(); got != 3 {
		t.Fatalf("Debug().MainCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().MainCount(); got != 0 {
		t.Fatalf("nil Debug().MainCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsArticleCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><article id="first"></article><div id="host"></div><section><article id="second"></article></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().ArticleCount(); got != 2 {
		t.Fatalf("Debug().ArticleCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<article id="third"></article>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().ArticleCount(); got != 3 {
		t.Fatalf("Debug().ArticleCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().ArticleCount(); got != 0 {
		t.Fatalf("nil Debug().ArticleCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsNavCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><nav id="first"></nav><div id="host"></div><section><nav id="second"></nav></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().NavCount(); got != 2 {
		t.Fatalf("Debug().NavCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<nav id="third"></nav>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().NavCount(); got != 3 {
		t.Fatalf("Debug().NavCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().NavCount(); got != 0 {
		t.Fatalf("nil Debug().NavCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsAsideCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><aside id="first"></aside><div id="host"></div><section><aside id="second"></aside></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().AsideCount(); got != 2 {
		t.Fatalf("Debug().AsideCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<aside id="third"></aside>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().AsideCount(); got != 3 {
		t.Fatalf("Debug().AsideCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().AsideCount(); got != 0 {
		t.Fatalf("nil Debug().AsideCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsFigureCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><figure id="first"></figure><div id="host"></div><section><figure id="second"></figure></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().FigureCount(); got != 2 {
		t.Fatalf("Debug().FigureCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<figure id="third"></figure>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().FigureCount(); got != 3 {
		t.Fatalf("Debug().FigureCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FigureCount(); got != 0 {
		t.Fatalf("nil Debug().FigureCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsFigcaptionCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><figure id="first"><figcaption id="caption-one">One</figcaption></figure><div id="host"></div><section><figure id="second"><figcaption id="caption-two">Two</figcaption></figure></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().FigcaptionCount(); got != 2 {
		t.Fatalf("Debug().FigcaptionCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<figure id="third"><figcaption id="caption-three">Three</figcaption></figure>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().FigcaptionCount(); got != 3 {
		t.Fatalf("Debug().FigcaptionCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FigcaptionCount(); got != 0 {
		t.Fatalf("nil Debug().FigcaptionCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsHeaderCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><header id="first"></header><div id="host"></div><section><header id="second"></header></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().HeaderCount(); got != 2 {
		t.Fatalf("Debug().HeaderCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<header id="third"></header>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().HeaderCount(); got != 3 {
		t.Fatalf("Debug().HeaderCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().HeaderCount(); got != 0 {
		t.Fatalf("nil Debug().HeaderCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsFooterCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><footer id="first"></footer><div id="host"></div><section><footer id="second"></footer></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().FooterCount(); got != 2 {
		t.Fatalf("Debug().FooterCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<footer id="third"></footer>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().FooterCount(); got != 3 {
		t.Fatalf("Debug().FooterCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FooterCount(); got != 0 {
		t.Fatalf("nil Debug().FooterCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsAddressCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><address id="first"></address><div id="host"></div><section><address id="second"></address></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().AddressCount(); got != 2 {
		t.Fatalf("Debug().AddressCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<address id="third"></address>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().AddressCount(); got != 3 {
		t.Fatalf("Debug().AddressCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().AddressCount(); got != 0 {
		t.Fatalf("nil Debug().AddressCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsBlockquoteCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><blockquote id="first"></blockquote><div id="host"></div><section><blockquote id="second"></blockquote></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().BlockquoteCount(); got != 2 {
		t.Fatalf("Debug().BlockquoteCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<blockquote id="third"></blockquote>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().BlockquoteCount(); got != 3 {
		t.Fatalf("Debug().BlockquoteCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().BlockquoteCount(); got != 0 {
		t.Fatalf("nil Debug().BlockquoteCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsParagraphCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><p id="first"></p><div id="host"></div><section><p id="second"></p></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().ParagraphCount(); got != 2 {
		t.Fatalf("Debug().ParagraphCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<p id="third"></p>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().ParagraphCount(); got != 3 {
		t.Fatalf("Debug().ParagraphCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().ParagraphCount(); got != 0 {
		t.Fatalf("nil Debug().ParagraphCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsPreCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><pre id="first"></pre><div id="host"></div><section><pre id="second"></pre></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().PreCount(); got != 2 {
		t.Fatalf("Debug().PreCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<pre id="third"></pre>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().PreCount(); got != 3 {
		t.Fatalf("Debug().PreCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().PreCount(); got != 0 {
		t.Fatalf("nil Debug().PreCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsMarkCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><mark id="first"></mark><div id="host"></div><section><mark id="second"></mark></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().MarkCount(); got != 2 {
		t.Fatalf("Debug().MarkCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<mark id="third"></mark>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().MarkCount(); got != 3 {
		t.Fatalf("Debug().MarkCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().MarkCount(); got != 0 {
		t.Fatalf("nil Debug().MarkCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsQCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><q id="first"></q><div id="host"></div><section><q id="second"></q></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().QCount(); got != 2 {
		t.Fatalf("Debug().QCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<q id="third"></q>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().QCount(); got != 3 {
		t.Fatalf("Debug().QCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().QCount(); got != 0 {
		t.Fatalf("nil Debug().QCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsCiteCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><cite id="first"></cite><div id="host"></div><section><cite id="second"></cite></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().CiteCount(); got != 2 {
		t.Fatalf("Debug().CiteCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<cite id="third"></cite>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().CiteCount(); got != 3 {
		t.Fatalf("Debug().CiteCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().CiteCount(); got != 0 {
		t.Fatalf("nil Debug().CiteCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsAbbrCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><abbr id="first"></abbr><div id="host"></div><section><abbr id="second"></abbr></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().AbbrCount(); got != 2 {
		t.Fatalf("Debug().AbbrCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<abbr id="third"></abbr>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().AbbrCount(); got != 3 {
		t.Fatalf("Debug().AbbrCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().AbbrCount(); got != 0 {
		t.Fatalf("nil Debug().AbbrCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsStrongCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><strong id="first"></strong><div id="host"></div><section><strong id="second"></strong></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().StrongCount(); got != 2 {
		t.Fatalf("Debug().StrongCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<strong id="third"></strong>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().StrongCount(); got != 3 {
		t.Fatalf("Debug().StrongCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().StrongCount(); got != 0 {
		t.Fatalf("nil Debug().StrongCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSpanCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><span id="first"></span><div id="host"></div><section><span id="second"></span></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SpanCount(); got != 2 {
		t.Fatalf("Debug().SpanCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<span id="third"></span>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SpanCount(); got != 3 {
		t.Fatalf("Debug().SpanCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SpanCount(); got != 0 {
		t.Fatalf("nil Debug().SpanCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsDataCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><data id="first"></data><div id="host"></div><section><data id="second"></data></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().DataCount(); got != 2 {
		t.Fatalf("Debug().DataCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<data id="third"></data>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().DataCount(); got != 3 {
		t.Fatalf("Debug().DataCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DataCount(); got != 0 {
		t.Fatalf("nil Debug().DataCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsDfnCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><dfn id="first"></dfn><div id="host"></div><section><dfn id="second"></dfn></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().DfnCount(); got != 2 {
		t.Fatalf("Debug().DfnCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<dfn id="third"></dfn>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().DfnCount(); got != 3 {
		t.Fatalf("Debug().DfnCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DfnCount(); got != 0 {
		t.Fatalf("nil Debug().DfnCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsKbdCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><kbd id="first"></kbd><div id="host"></div><section><kbd id="second"></kbd></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().KbdCount(); got != 2 {
		t.Fatalf("Debug().KbdCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<kbd id="third"></kbd>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().KbdCount(); got != 3 {
		t.Fatalf("Debug().KbdCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().KbdCount(); got != 0 {
		t.Fatalf("nil Debug().KbdCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSampCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><samp id="first"></samp><div id="host"></div><section><samp id="second"></samp></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SampCount(); got != 2 {
		t.Fatalf("Debug().SampCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<samp id="third"></samp>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SampCount(); got != 3 {
		t.Fatalf("Debug().SampCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SampCount(); got != 0 {
		t.Fatalf("nil Debug().SampCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsRubyCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><ruby id="first"><rt>one</rt></ruby><div id="host"></div><section><ruby id="second"></ruby></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().RubyCount(); got != 2 {
		t.Fatalf("Debug().RubyCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<ruby id="third"></ruby>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().RubyCount(); got != 3 {
		t.Fatalf("Debug().RubyCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().RubyCount(); got != 0 {
		t.Fatalf("nil Debug().RubyCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsRtCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><ruby id="first"><rt id="first-rt">one</rt></ruby><div id="host"></div><section><ruby id="second"><rt id="second-rt">two</rt></ruby></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().RtCount(); got != 2 {
		t.Fatalf("Debug().RtCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<ruby id="third"><rt id="third-rt">three</rt></ruby>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().RtCount(); got != 3 {
		t.Fatalf("Debug().RtCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().RtCount(); got != 0 {
		t.Fatalf("nil Debug().RtCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsVarCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><var id="first"></var><div id="host"></div><section><var id="second"></var></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().VarCount(); got != 2 {
		t.Fatalf("Debug().VarCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<var id="third"></var>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().VarCount(); got != 3 {
		t.Fatalf("Debug().VarCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().VarCount(); got != 0 {
		t.Fatalf("nil Debug().VarCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsCodeCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><code id="first"></code><div id="host"></div><section><code id="second"></code></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().CodeCount(); got != 2 {
		t.Fatalf("Debug().CodeCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<code id="third"></code>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().CodeCount(); got != 3 {
		t.Fatalf("Debug().CodeCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().CodeCount(); got != 0 {
		t.Fatalf("nil Debug().CodeCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsSmallCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><small id="first"></small><div id="host"></div><section><small id="second"></small></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().SmallCount(); got != 2 {
		t.Fatalf("Debug().SmallCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<small id="third"></small>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().SmallCount(); got != 3 {
		t.Fatalf("Debug().SmallCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().SmallCount(); got != 0 {
		t.Fatalf("nil Debug().SmallCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsTimeCount(t *testing.T) {
	harness, err := FromHTML(`<div id="root"><time id="first"></time><div id="host"></div><section><time id="second"></time></section></div>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().TimeCount(); got != 2 {
		t.Fatalf("Debug().TimeCount() = %d, want 2", got)
	}

	if err := harness.SetInnerHTML("#host", `<time id="third"></time>`); err != nil {
		t.Fatalf("SetInnerHTML(#host) error = %v", err)
	}
	if got := harness.Debug().TimeCount(); got != 3 {
		t.Fatalf("Debug().TimeCount() after SetInnerHTML = %d, want 3", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().TimeCount(); got != 0 {
		t.Fatalf("nil Debug().TimeCount() = %d, want 0", got)
	}
}

func TestDebugViewReportsHistoryState(t *testing.T) {
	harness, err := FromHTMLWithURL("https://example.test/page", `<main><script>host:historyPushState("step-1", "", "#step-1")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got := harness.Debug().HistoryLength(); got != 2 {
		t.Fatalf("Debug().HistoryLength() = %d, want 2", got)
	}
	if got, ok := harness.Debug().HistoryState(); !ok || got != "step-1" {
		t.Fatalf("Debug().HistoryState() = (%q, %v), want (\"step-1\", true)", got, ok)
	}

	if err := harness.Navigate("#missing"); err != nil {
		t.Fatalf("Navigate(#missing) error = %v", err)
	}
	if got := harness.Debug().HistoryLength(); got != 3 {
		t.Fatalf("Debug().HistoryLength() after Navigate = %d, want 3", got)
	}
	if got, ok := harness.Debug().HistoryState(); ok || got != "null" {
		t.Fatalf("Debug().HistoryState() after Navigate = (%q, %v), want (\"null\", false)", got, ok)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().HistoryLength(); got != 0 {
		t.Fatalf("nil Debug().HistoryLength() = %d, want 0", got)
	}
	if got, ok := nilHarness.Debug().HistoryState(); ok || got != "null" {
		t.Fatalf("nil Debug().HistoryState() = (%q, %v), want (\"null\", false)", got, ok)
	}
}

func TestDebugViewReportsHistoryEntries(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/app",
		`<main><script>host:historyPushState("step-1", "", "#step-1"); host:historyReplaceState("step-2", "", "#step-2")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	entries := harness.Debug().HistoryEntries()
	if len(entries) != 2 {
		t.Fatalf("Debug().HistoryEntries() = %#v, want 2 entries", entries)
	}
	if entries[0].URL != "https://example.test/app" || entries[0].HasState {
		t.Fatalf("Debug().HistoryEntries()[0] = %#v, want initial entry without state", entries[0])
	}
	if entries[1].URL != "https://example.test/app#step-2" || !entries[1].HasState || entries[1].State != "step-2" {
		t.Fatalf("Debug().HistoryEntries()[1] = %#v, want current entry with step-2 state", entries[1])
	}

	entries[0].URL = "mutated"
	entries[1].State = "mutated"
	if fresh := harness.Debug().HistoryEntries(); len(fresh) != 2 || fresh[0].URL != "https://example.test/app" || fresh[1].State != "step-2" {
		t.Fatalf("Debug().HistoryEntries() reread = %#v, want original history entries", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().HistoryEntries(); got != nil {
		t.Fatalf("nil Debug().HistoryEntries() = %#v, want nil", got)
	}
}

func TestDebugViewReportsHistoryIndex(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/app",
		`<main><script>host:historyPushState("step-1", "", "#step-1")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got := harness.Debug().HistoryIndex(); got != 1 {
		t.Fatalf("Debug().HistoryIndex() = %d, want 1", got)
	}
	if err := harness.Navigate("#other"); err != nil {
		t.Fatalf("Navigate(#other) error = %v", err)
	}
	if got := harness.Debug().HistoryIndex(); got != 2 {
		t.Fatalf("Debug().HistoryIndex() after Navigate = %d, want 2", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().HistoryIndex(); got != 0 {
		t.Fatalf("nil Debug().HistoryIndex() = %d, want 0", got)
	}
}

func TestDebugViewReportsVisitedURLs(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/app",
		`<main><script>host:historyPushState("step-1", "", "#step-1")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	visited := harness.Debug().VisitedURLs()
	if len(visited) != 2 || visited[0] != "https://example.test/app" || visited[1] != "https://example.test/app#step-1" {
		t.Fatalf("Debug().VisitedURLs() = %#v, want history-derived visited URLs", visited)
	}

	visited[0] = "mutated"
	visited[1] = "mutated"
	if fresh := harness.Debug().VisitedURLs(); len(fresh) != 2 || fresh[0] != "https://example.test/app" || fresh[1] != "https://example.test/app#step-1" {
		t.Fatalf("Debug().VisitedURLs() reread = %#v, want original visited URLs", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().VisitedURLs(); got != nil {
		t.Fatalf("nil Debug().VisitedURLs() = %#v, want nil", got)
	}
}

func TestDebugViewReportsPendingTimers(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script>host:setTimeout('host:insertAdjacentHTML("#out", "beforeend", "<span>timeout</span>")', 5); host:setInterval('host:insertAdjacentHTML("#out", "beforeend", "<span>interval</span>")', 9); host:requestAnimationFrame('host:insertAdjacentHTML("#out", "beforeend", "<span>frame</span>")')</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	timers := harness.Debug().PendingTimers()
	if len(timers) != 2 {
		t.Fatalf("Debug().PendingTimers() = %#v, want 2 entries", timers)
	}
	if timers[0].DueAtMs != 5 || timers[0].IntervalMs != 5 || timers[0].Repeat {
		t.Fatalf("Debug().PendingTimers()[0] = %#v, want one-shot timer due at 5", timers[0])
	}
	if timers[1].DueAtMs != 9 || timers[1].IntervalMs != 9 || !timers[1].Repeat {
		t.Fatalf("Debug().PendingTimers()[1] = %#v, want repeating timer due at 9", timers[1])
	}
	timers[0].Source = "mutated"
	if fresh := harness.Debug().PendingTimers(); len(fresh) != 2 || fresh[0].Source != `host:insertAdjacentHTML("#out", "beforeend", "<span>timeout</span>")` {
		t.Fatalf("Debug().PendingTimers() reread = %#v, want original timer snapshot", fresh)
	}

	frames := harness.Debug().PendingAnimationFrames()
	if len(frames) != 1 || frames[0].Source != `host:insertAdjacentHTML("#out", "beforeend", "<span>frame</span>")` {
		t.Fatalf("Debug().PendingAnimationFrames() = %#v, want one pending frame", frames)
	}
	frames[0].Source = "mutated"
	if fresh := harness.Debug().PendingAnimationFrames(); len(fresh) != 1 || fresh[0].Source != `host:insertAdjacentHTML("#out", "beforeend", "<span>frame</span>")` {
		t.Fatalf("Debug().PendingAnimationFrames() reread = %#v, want original frame snapshot", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().PendingTimers(); got != nil {
		t.Fatalf("nil Debug().PendingTimers() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().PendingAnimationFrames(); got != nil {
		t.Fatalf("nil Debug().PendingAnimationFrames() = %#v, want nil", got)
	}
}

func TestDebugViewReportsHistoryScrollRestoration(t *testing.T) {
	harness, err := FromHTML(`<main><script>host:historySetScrollRestoration("manual")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().HistoryScrollRestoration(); got != "manual" {
		t.Fatalf("Debug().HistoryScrollRestoration() = %q, want %q", got, "manual")
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().HistoryScrollRestoration(); got != "auto" {
		t.Fatalf("nil Debug().HistoryScrollRestoration() = %q, want %q", got, "auto")
	}
}

func TestDebugViewReportsDocumentCookie(t *testing.T) {
	harness, err := FromHTML(`<main><script>host:setDocumentCookie("theme=dark"); host:setDocumentCookie("lang=en; Path=/")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DocumentCookie(), "lang=en; theme=dark"; got != want {
		t.Fatalf("Debug().DocumentCookie() = %q, want %q", got, want)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DocumentCookie(); got != "" {
		t.Fatalf("nil Debug().DocumentCookie() = %q, want empty", got)
	}
}

func TestHarnessBuilderCopiesConfigurationContract(t *testing.T) {
	localStorage := map[string]string{"token": "abc"}
	sessionStorage := map[string]string{"tab": "main"}
	matchMedia := map[string]bool{"(prefers-reduced-motion: reduce)": true}

	harness, err := NewHarnessBuilder().
		URL("https://example.test/").
		HTML("<main>ok</main>").
		LocalStorage(localStorage).
		SessionStorage(sessionStorage).
		MatchMedia(matchMedia).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	localStorage["token"] = "mutated"
	sessionStorage["tab"] = "mutated"
	matchMedia["(prefers-reduced-motion: reduce)"] = false

	if got, want := harness.URL(), "https://example.test/"; got != want {
		t.Fatalf("URL() = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), "<main>ok</main>"; got != want {
		t.Fatalf("HTML() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().InitialHTML(), "<main>ok</main>"; got != want {
		t.Fatalf("Debug().InitialHTML() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().URL(), "https://example.test/"; got != want {
		t.Fatalf("Debug().URL() = %q, want %q", got, want)
	}
	if got := harness.Debug().MatchMediaRules(); len(got) != 1 || !got["(prefers-reduced-motion: reduce)"] {
		t.Fatalf("Debug().MatchMediaRules() = %#v, want seeded rule", got)
	}
	if got, want := harness.Mocks().Storage().Local()["token"], "abc"; got != want {
		t.Fatalf("Storage().Local()[\"token\"] = %q, want %q", got, want)
	}
	if got, want := harness.Mocks().Storage().Session()["tab"], "main"; got != want {
		t.Fatalf("Storage().Session()[\"tab\"] = %q, want %q", got, want)
	}
	if got := harness.Mocks().MatchMedia().Rules(); len(got) != 1 || got[0].Query != "(prefers-reduced-motion: reduce)" || !got[0].Matches {
		t.Fatalf("MatchMedia().Rules() = %#v, want seeded rule", got)
	}
}

func TestDebugViewReportsClipboard(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := harness.WriteClipboard("copied text"); err != nil {
		t.Fatalf("WriteClipboard() error = %v", err)
	}
	if got, want := harness.Debug().Clipboard(), "copied text"; got != want {
		t.Fatalf("Debug().Clipboard() = %q, want %q", got, want)
	}

	writes := harness.Debug().ClipboardWrites()
	if len(writes) != 1 || writes[0] != "copied text" {
		t.Fatalf("Debug().ClipboardWrites() = %#v, want one write", writes)
	}
	writes[0] = "mutated"
	if fresh := harness.Debug().ClipboardWrites(); len(fresh) != 1 || fresh[0] != "copied text" {
		t.Fatalf("Debug().ClipboardWrites() reread = %#v, want original write", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().Clipboard(); got != "" {
		t.Fatalf("nil Debug().Clipboard() = %q, want empty", got)
	}
	if got := nilHarness.Debug().ClipboardWrites(); got != nil {
		t.Fatalf("nil Debug().ClipboardWrites() = %#v, want nil", got)
	}
}

func TestDebugViewReportsMatchMediaRules(t *testing.T) {
	harness, err := NewHarnessBuilder().
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	rules := harness.Debug().MatchMediaRules()
	if got, want := rules["(prefers-reduced-motion: reduce)"], true; got != want {
		t.Fatalf("Debug().MatchMediaRules()[prefers-reduced-motion] = %v, want %v", got, want)
	}
	rules["(prefers-reduced-motion: reduce)"] = false
	if got, want := harness.Debug().MatchMediaRules()["(prefers-reduced-motion: reduce)"], true; got != want {
		t.Fatalf("Debug().MatchMediaRules() reread = %v, want %v", got, want)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().MatchMediaRules(); got != nil {
		t.Fatalf("nil Debug().MatchMediaRules() = %#v, want nil", got)
	}
}

func TestMatchMediaRulesReturnCopies(t *testing.T) {
	harness, err := NewHarnessBuilder().
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	rules := harness.Mocks().MatchMedia().Rules()
	if len(rules) != 1 {
		t.Fatalf("MatchMedia().Rules() = %#v, want one rule", rules)
	}
	rules[0].Query = "mutated"
	if got := harness.Mocks().MatchMedia().Rules(); len(got) != 1 || got[0].Query != "(prefers-reduced-motion: reduce)" {
		t.Fatalf("MatchMedia().Rules() reread = %#v, want original rule", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().MatchMedia().Rules(); got != nil {
		t.Fatalf("nil Harness.Mocks().MatchMedia().Rules() = %#v, want nil", got)
	}
}

func TestDebugViewReportsWebStorage(t *testing.T) {
	harness, err := NewHarnessBuilder().
		LocalStorage(map[string]string{"theme": "dark"}).
		SessionStorage(map[string]string{"tab": "main"}).
		HTML(`<main><script>host:localStorageSetItem("accent", "blue"); host:sessionStorageRemoveItem("tab")</script></main>`).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := harness.Debug().DumpDOM(); got == "" {
		t.Fatalf("Debug().DumpDOM() = empty, want bootstrapped DOM")
	}

	local := harness.Debug().LocalStorage()
	if got, want := local["theme"], "dark"; got != want {
		t.Fatalf("Debug().LocalStorage()[theme] = %q, want %q", got, want)
	}
	if got, want := local["accent"], "blue"; got != want {
		t.Fatalf("Debug().LocalStorage()[accent] = %q, want %q", got, want)
	}
	local["theme"] = "mutated"
	if got, want := harness.Debug().LocalStorage()["theme"], "dark"; got != want {
		t.Fatalf("Debug().LocalStorage()[theme] after mutation = %q, want %q", got, want)
	}

	session := harness.Debug().SessionStorage()
	if len(session) != 0 {
		t.Fatalf("Debug().SessionStorage() = %#v, want empty after mutation", session)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().LocalStorage(); got != nil {
		t.Fatalf("nil Debug().LocalStorage() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().SessionStorage(); got != nil {
		t.Fatalf("nil Debug().SessionStorage() = %#v, want nil", got)
	}
}

func TestDebugViewReportsNavigationLog(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/start",
		`<main><script>host:locationAssign("/next"); host:locationReplace("/replace")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().NavigationLog(), []string{
		"https://example.test/next",
		"https://example.test/replace",
	}; len(got) != len(want) {
		t.Fatalf("Debug().NavigationLog() = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Debug().NavigationLog()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	}

	logs := harness.Debug().NavigationLog()
	logs[0] = "mutated"
	fresh := harness.Debug().NavigationLog()
	if fresh[0] != "https://example.test/next" {
		t.Fatalf("Debug().NavigationLog()[0] after mutation = %q, want %q", fresh[0], "https://example.test/next")
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().NavigationLog(); got != nil {
		t.Fatalf("nil Debug().NavigationLog() = %#v, want nil", got)
	}
}

func TestDebugViewReportsLocationParts(t *testing.T) {
	harness, err := FromHTMLWithURL("https://example.test:8443/path/name?mode=full#step-1", "<main></main>")
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().LocationOrigin(), "https://example.test:8443"; got != want {
		t.Fatalf("Debug().LocationOrigin() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationProtocol(), "https:"; got != want {
		t.Fatalf("Debug().LocationProtocol() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationHost(), "example.test:8443"; got != want {
		t.Fatalf("Debug().LocationHost() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationHostname(), "example.test"; got != want {
		t.Fatalf("Debug().LocationHostname() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationPort(), "8443"; got != want {
		t.Fatalf("Debug().LocationPort() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationPathname(), "/path/name"; got != want {
		t.Fatalf("Debug().LocationPathname() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationSearch(), "?mode=full"; got != want {
		t.Fatalf("Debug().LocationSearch() = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LocationHash(), "#step-1"; got != want {
		t.Fatalf("Debug().LocationHash() = %q, want %q", got, want)
	}

	var nilHarness *Harness
	for _, tc := range []struct {
		name string
		got  string
	}{
		{name: "LocationOrigin", got: nilHarness.Debug().LocationOrigin()},
		{name: "LocationProtocol", got: nilHarness.Debug().LocationProtocol()},
		{name: "LocationHost", got: nilHarness.Debug().LocationHost()},
		{name: "LocationHostname", got: nilHarness.Debug().LocationHostname()},
		{name: "LocationPort", got: nilHarness.Debug().LocationPort()},
		{name: "LocationPathname", got: nilHarness.Debug().LocationPathname()},
		{name: "LocationSearch", got: nilHarness.Debug().LocationSearch()},
		{name: "LocationHash", got: nilHarness.Debug().LocationHash()},
	} {
		if tc.got != "" {
			t.Fatalf("nil Debug().%s() = %q, want empty", tc.name, tc.got)
		}
	}
}

func TestLocationMockNavigationsReturnCopies(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/start",
		`<main><script>host:locationAssign("/next"); host:locationReplace("/replace")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().NavigationLog(), []string{"https://example.test/next", "https://example.test/replace"}; len(got) != len(want) {
		t.Fatalf("Debug().NavigationLog() = %#v, want %#v", got, want)
	} else if got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Debug().NavigationLog() = %#v, want %#v", got, want)
	}

	navigations := harness.Mocks().Location().Navigations()
	if len(navigations) != 2 {
		t.Fatalf("Location().Navigations() = %#v, want 2 entries", navigations)
	}
	navigations[0] = "mutated"
	navigations = append(navigations, "extra")

	fresh := harness.Mocks().Location().Navigations()
	if len(fresh) != 2 {
		t.Fatalf("Location().Navigations() reread len = %d, want 2", len(fresh))
	}
	if fresh[0] != "https://example.test/next" || fresh[1] != "https://example.test/replace" {
		t.Fatalf("Location().Navigations() reread = %#v, want original entries", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().Location(); got != nil {
		t.Fatalf("nil Harness.Mocks().Location() = %#v, want nil", got)
	}
}

func TestFetchMockSnapshotsReturnCopies(t *testing.T) {
	harness, err := FromHTML(`<main></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	harness.Mocks().Fetch().RespondText("https://example.test/api/message", 200, "ok")
	harness.Mocks().Fetch().Fail("https://example.test/api/broken", "boom")
	if _, err := harness.Fetch("https://example.test/api/message"); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if _, err := harness.Fetch("https://example.test/api/broken"); err == nil {
		t.Fatalf("Fetch() broken error = nil, want failure")
	}

	calls := harness.Mocks().Fetch().Calls()
	if len(calls) != 2 {
		t.Fatalf("Fetch().Calls() = %#v, want 2 entries", calls)
	}
	calls[0].URL = "mutated"
	calls = append(calls, FetchCall{URL: "extra"})

	responses := harness.Mocks().Fetch().Responses()
	if len(responses) != 1 {
		t.Fatalf("Fetch().Responses() = %#v, want 1 entry", responses)
	}
	responses[0].URL = "mutated"
	responses[0].Body = "mutated"
	responses = append(responses, FetchResponseRule{URL: "extra"})

	errors := harness.Mocks().Fetch().Errors()
	if len(errors) != 1 {
		t.Fatalf("Fetch().Errors() = %#v, want 1 entry", errors)
	}
	errors[0].URL = "mutated"
	errors[0].Message = "mutated"
	errors = append(errors, FetchErrorRule{URL: "extra"})

	freshCalls := harness.Mocks().Fetch().Calls()
	if len(freshCalls) != 2 || freshCalls[0].URL != "https://example.test/api/message" || freshCalls[1].URL != "https://example.test/api/broken" {
		t.Fatalf("Fetch().Calls() reread = %#v, want original entries", freshCalls)
	}
	freshResponses := harness.Mocks().Fetch().Responses()
	if len(freshResponses) != 1 || freshResponses[0].URL != "https://example.test/api/message" || freshResponses[0].Body != "ok" {
		t.Fatalf("Fetch().Responses() reread = %#v, want original response rule", freshResponses)
	}
	freshErrors := harness.Mocks().Fetch().Errors()
	if len(freshErrors) != 1 || freshErrors[0].URL != "https://example.test/api/broken" || freshErrors[0].Message != "boom" {
		t.Fatalf("Fetch().Errors() reread = %#v, want original error rule", freshErrors)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().Fetch(); got != nil {
		t.Fatalf("nil Harness.Mocks().Fetch() = %#v, want nil", got)
	}
}

func TestDebugViewReportsFetchCalls(t *testing.T) {
	harness, err := FromHTML(`<main></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	harness.Mocks().Fetch().RespondText("https://example.test/api/message", 200, "ok")
	harness.Mocks().Fetch().Fail("https://example.test/api/broken", "boom")
	if _, err := harness.Fetch("https://example.test/api/message"); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if _, err := harness.Fetch("https://example.test/api/broken"); err == nil {
		t.Fatalf("Fetch() broken error = nil, want failure")
	}

	calls := harness.Debug().FetchCalls()
	if len(calls) != 2 || calls[0].URL != "https://example.test/api/message" || calls[1].URL != "https://example.test/api/broken" {
		t.Fatalf("Debug().FetchCalls() = %#v, want two captured requests", calls)
	}

	calls[0].URL = "mutated"
	if fresh := harness.Debug().FetchCalls(); len(fresh) != 2 || fresh[0].URL != "https://example.test/api/message" || fresh[1].URL != "https://example.test/api/broken" {
		t.Fatalf("Debug().FetchCalls() reread = %#v, want original request", fresh)
	}

	responses := harness.Debug().FetchResponseRules()
	if len(responses) != 1 || responses[0].URL != "https://example.test/api/message" || responses[0].Status != 200 || responses[0].Body != "ok" {
		t.Fatalf("Debug().FetchResponseRules() = %#v, want one response rule", responses)
	}
	responses[0].URL = "mutated"
	if fresh := harness.Debug().FetchResponseRules(); len(fresh) != 1 || fresh[0].URL != "https://example.test/api/message" || fresh[0].Body != "ok" {
		t.Fatalf("Debug().FetchResponseRules() reread = %#v, want original response rule", fresh)
	}

	errors := harness.Debug().FetchErrorRules()
	if len(errors) != 1 || errors[0].URL != "https://example.test/api/broken" || errors[0].Message != "boom" {
		t.Fatalf("Debug().FetchErrorRules() = %#v, want one error rule", errors)
	}
	errors[0].URL = "mutated"
	if fresh := harness.Debug().FetchErrorRules(); len(fresh) != 1 || fresh[0].URL != "https://example.test/api/broken" || fresh[0].Message != "boom" {
		t.Fatalf("Debug().FetchErrorRules() reread = %#v, want original error rule", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().FetchCalls(); got != nil {
		t.Fatalf("nil Debug().FetchCalls() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().FetchResponseRules(); got != nil {
		t.Fatalf("nil Debug().FetchResponseRules() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().FetchErrorRules(); got != nil {
		t.Fatalf("nil Debug().FetchErrorRules() = %#v, want nil", got)
	}
}

func TestDebugViewReportsActionCalls(t *testing.T) {
	harness, err := NewHarnessBuilder().
		HTML(`<main></main>`).
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := harness.Open("https://example.test/popup"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := harness.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := harness.Print(); err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if err := harness.ScrollTo(4, 5); err != nil {
		t.Fatalf("ScrollTo() error = %v", err)
	}
	if err := harness.ScrollBy(2, -1); err != nil {
		t.Fatalf("ScrollBy() error = %v", err)
	}
	if got, err := harness.MatchMedia("(prefers-reduced-motion: reduce)"); err != nil || !got {
		t.Fatalf("MatchMedia() = (%v, %v), want (true, nil)", got, err)
	}

	openCalls := harness.Debug().OpenCalls()
	if len(openCalls) != 1 || openCalls[0].URL != "https://example.test/popup" {
		t.Fatalf("Debug().OpenCalls() = %#v, want one open call", openCalls)
	}
	openCalls[0].URL = "mutated"
	if fresh := harness.Debug().OpenCalls(); len(fresh) != 1 || fresh[0].URL != "https://example.test/popup" {
		t.Fatalf("Debug().OpenCalls() reread = %#v, want original open call", fresh)
	}

	if closeCalls := harness.Debug().CloseCalls(); len(closeCalls) != 1 {
		t.Fatalf("Debug().CloseCalls() = %#v, want one close call", closeCalls)
	}
	if printCalls := harness.Debug().PrintCalls(); len(printCalls) != 1 {
		t.Fatalf("Debug().PrintCalls() = %#v, want one print call", printCalls)
	}
	scrollCalls := harness.Debug().ScrollCalls()
	if len(scrollCalls) != 2 {
		t.Fatalf("Debug().ScrollCalls() = %#v, want two scroll calls", scrollCalls)
	}
	scrollCalls[0].X = 99
	if fresh := harness.Debug().ScrollCalls(); len(fresh) != 2 || fresh[0].X != 4 || fresh[1].Method != ScrollMethodBy {
		t.Fatalf("Debug().ScrollCalls() reread = %#v, want original scroll calls", fresh)
	}
	if matchMediaCalls := harness.Debug().MatchMediaCalls(); len(matchMediaCalls) != 1 || matchMediaCalls[0].Query != "(prefers-reduced-motion: reduce)" {
		t.Fatalf("Debug().MatchMediaCalls() = %#v, want one query call", matchMediaCalls)
	}
	if fresh := harness.Debug().MatchMediaCalls(); len(fresh) != 1 || fresh[0].Query != "(prefers-reduced-motion: reduce)" {
		t.Fatalf("Debug().MatchMediaCalls() reread = %#v, want original query call", fresh)
	}

	harness.Mocks().MatchMedia().RecordListenerCall("(prefers-reduced-motion: reduce)", "change")
	listeners := harness.Debug().MatchMediaListenerCalls()
	if len(listeners) != 1 || listeners[0].Query != "(prefers-reduced-motion: reduce)" || listeners[0].Method != "change" {
		t.Fatalf("Debug().MatchMediaListenerCalls() = %#v, want one listener call", listeners)
	}
	listeners[0].Method = "mutated"
	if fresh := harness.Debug().MatchMediaListenerCalls(); len(fresh) != 1 || fresh[0].Method != "change" {
		t.Fatalf("Debug().MatchMediaListenerCalls() reread = %#v, want original listener call", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().OpenCalls(); got != nil {
		t.Fatalf("nil Debug().OpenCalls() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().CloseCalls(); got != nil {
		t.Fatalf("nil Debug().CloseCalls() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().PrintCalls(); got != nil {
		t.Fatalf("nil Debug().PrintCalls() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().ScrollCalls(); got != nil {
		t.Fatalf("nil Debug().ScrollCalls() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().MatchMediaCalls(); got != nil {
		t.Fatalf("nil Debug().MatchMediaCalls() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().MatchMediaListenerCalls(); got != nil {
		t.Fatalf("nil Debug().MatchMediaListenerCalls() = %#v, want nil", got)
	}
}

func TestDebugViewReportsDialogMessages(t *testing.T) {
	harness, err := FromHTML(`<main></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	harness.Mocks().Dialogs().QueueConfirm(true)
	harness.Mocks().Dialogs().QueuePromptText("Ada")

	if err := harness.Alert("hello"); err != nil {
		t.Fatalf("Alert() error = %v", err)
	}
	if _, err := harness.Confirm("Continue?"); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if _, _, err := harness.Prompt("Your name?"); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	alerts := harness.Debug().DialogAlerts()
	if len(alerts) != 1 || alerts[0] != "hello" {
		t.Fatalf("Debug().DialogAlerts() = %#v, want one alert", alerts)
	}
	alerts[0] = "mutated"
	if fresh := harness.Debug().DialogAlerts(); len(fresh) != 1 || fresh[0] != "hello" {
		t.Fatalf("Debug().DialogAlerts() reread = %#v, want original alert", fresh)
	}

	confirms := harness.Debug().DialogConfirmMessages()
	if len(confirms) != 1 || confirms[0] != "Continue?" {
		t.Fatalf("Debug().DialogConfirmMessages() = %#v, want one confirm message", confirms)
	}
	confirms[0] = "mutated"
	if fresh := harness.Debug().DialogConfirmMessages(); len(fresh) != 1 || fresh[0] != "Continue?" {
		t.Fatalf("Debug().DialogConfirmMessages() reread = %#v, want original confirm message", fresh)
	}

	prompts := harness.Debug().DialogPromptMessages()
	if len(prompts) != 1 || prompts[0] != "Your name?" {
		t.Fatalf("Debug().DialogPromptMessages() = %#v, want one prompt message", prompts)
	}
	prompts[0] = "mutated"
	if fresh := harness.Debug().DialogPromptMessages(); len(fresh) != 1 || fresh[0] != "Your name?" {
		t.Fatalf("Debug().DialogPromptMessages() reread = %#v, want original prompt message", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DialogAlerts(); got != nil {
		t.Fatalf("nil Debug().DialogAlerts() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().DialogConfirmMessages(); got != nil {
		t.Fatalf("nil Debug().DialogConfirmMessages() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().DialogPromptMessages(); got != nil {
		t.Fatalf("nil Debug().DialogPromptMessages() = %#v, want nil", got)
	}
}

func TestDebugViewReportsDownloadAndFileInputCaptures(t *testing.T) {
	harness, err := FromHTML(`<main><input id="upload" type="file"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.CaptureDownload("report.csv", []byte("downloaded bytes")); err != nil {
		t.Fatalf("CaptureDownload() error = %v", err)
	}
	if err := harness.SetFiles("#upload", []string{"report.csv", "archive.zip"}); err != nil {
		t.Fatalf("SetFiles() error = %v", err)
	}

	artifacts := harness.Debug().DownloadArtifacts()
	if len(artifacts) != 1 || artifacts[0].FileName != "report.csv" || string(artifacts[0].Bytes) != "downloaded bytes" {
		t.Fatalf("Debug().DownloadArtifacts() = %#v, want one download capture", artifacts)
	}
	artifacts[0].FileName = "mutated"
	artifacts[0].Bytes[0] = 'X'
	if fresh := harness.Debug().DownloadArtifacts(); len(fresh) != 1 || fresh[0].FileName != "report.csv" || string(fresh[0].Bytes) != "downloaded bytes" {
		t.Fatalf("Debug().DownloadArtifacts() reread = %#v, want original capture", fresh)
	}

	selections := harness.Debug().FileInputSelections()
	if len(selections) != 1 || selections[0].Selector != "#upload" || len(selections[0].Files) != 2 {
		t.Fatalf("Debug().FileInputSelections() = %#v, want one file-input selection", selections)
	}
	selections[0].Selector = "mutated"
	selections[0].Files[0] = "mutated"
	if fresh := harness.Debug().FileInputSelections(); len(fresh) != 1 || fresh[0].Selector != "#upload" || fresh[0].Files[0] != "report.csv" {
		t.Fatalf("Debug().FileInputSelections() reread = %#v, want original selection", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().DownloadArtifacts(); got != nil {
		t.Fatalf("nil Debug().DownloadArtifacts() = %#v, want nil", got)
	}
	if got := nilHarness.Debug().FileInputSelections(); got != nil {
		t.Fatalf("nil Debug().FileInputSelections() = %#v, want nil", got)
	}
}

func TestDebugViewReportsStorageEvents(t *testing.T) {
	harness, err := NewHarnessBuilder().
		HTML(`<main><script>host:localStorageSetItem("accent", "blue"); host:sessionStorageRemoveItem("tab"); host:localStorageClear()</script></main>`).
		LocalStorage(map[string]string{"theme": "dark"}).
		SessionStorage(map[string]string{"tab": "main"}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := harness.Debug().DumpDOM(); got == "" {
		t.Fatalf("Debug().DumpDOM() = %q, want non-empty", got)
	}

	events := harness.Debug().StorageEvents()
	if len(events) != 5 {
		t.Fatalf("Debug().StorageEvents() = %#v, want five events", events)
	}
	if events[0].Scope != "local" || events[0].Op != "seed" || events[0].Key != "theme" || events[0].Value != "dark" {
		t.Fatalf("Debug().StorageEvents()[0] = %#v, want local seed", events[0])
	}
	if events[1].Scope != "session" || events[1].Op != "seed" || events[1].Key != "tab" || events[1].Value != "main" {
		t.Fatalf("Debug().StorageEvents()[1] = %#v, want session seed", events[1])
	}
	if events[2].Scope != "local" || events[2].Op != "set" || events[2].Key != "accent" || events[2].Value != "blue" {
		t.Fatalf("Debug().StorageEvents()[2] = %#v, want local set", events[2])
	}
	if events[3].Scope != "session" || events[3].Op != "remove" || events[3].Key != "tab" {
		t.Fatalf("Debug().StorageEvents()[3] = %#v, want session remove", events[3])
	}
	if events[4].Scope != "local" || events[4].Op != "clear" {
		t.Fatalf("Debug().StorageEvents()[4] = %#v, want local clear", events[4])
	}

	events[0].Value = "mutated"
	if fresh := harness.Debug().StorageEvents(); len(fresh) != 5 || fresh[0].Value != "dark" {
		t.Fatalf("Debug().StorageEvents() reread = %#v, want original events", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().StorageEvents(); got != nil {
		t.Fatalf("nil Debug().StorageEvents() = %#v, want nil", got)
	}
}

func TestOtherMockSnapshotsReturnCopies(t *testing.T) {
	harness, err := NewHarnessBuilder().
		HTML(`<main></main>`).
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := harness.Open("https://example.test/popup"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := harness.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := harness.Print(); err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if err := harness.ScrollTo(7, 9); err != nil {
		t.Fatalf("ScrollTo() error = %v", err)
	}
	if err := harness.ScrollBy(2, -1); err != nil {
		t.Fatalf("ScrollBy() error = %v", err)
	}
	if got, err := harness.MatchMedia("(prefers-reduced-motion: reduce)"); err != nil {
		t.Fatalf("MatchMedia() error = %v", err)
	} else if !got {
		t.Fatalf("MatchMedia() = false, want true")
	}

	openCalls := harness.Mocks().Open().Calls()
	if len(openCalls) != 1 {
		t.Fatalf("Open().Calls() = %#v, want 1 entry", openCalls)
	}
	openCalls[0].URL = "mutated"
	openCalls = append(openCalls, OpenCall{URL: "extra"})

	closeCalls := harness.Mocks().Close().Calls()
	if len(closeCalls) != 1 {
		t.Fatalf("Close().Calls() = %#v, want 1 entry", closeCalls)
	}
	closeCalls = append(closeCalls, CloseCall{})

	printCalls := harness.Mocks().Print().Calls()
	if len(printCalls) != 1 {
		t.Fatalf("Print().Calls() = %#v, want 1 entry", printCalls)
	}
	printCalls = append(printCalls, PrintCall{})

	scrollCalls := harness.Mocks().Scroll().Calls()
	if len(scrollCalls) != 2 {
		t.Fatalf("Scroll().Calls() = %#v, want 2 entries", scrollCalls)
	}
	scrollCalls[0].Method = ScrollMethod("mutated")
	scrollCalls[0].X = 100
	scrollCalls[0].Y = 200
	scrollCalls = append(scrollCalls, ScrollCall{Method: ScrollMethod("extra"), X: 3, Y: 4})

	matchMediaCalls := harness.Mocks().MatchMedia().Calls()
	if len(matchMediaCalls) != 1 {
		t.Fatalf("MatchMedia().Calls() = %#v, want 1 entry", matchMediaCalls)
	}
	matchMediaCalls[0].Query = "mutated"
	matchMediaCalls = append(matchMediaCalls, MatchMediaCall{Query: "extra"})

	freshOpenCalls := harness.Mocks().Open().Calls()
	if len(freshOpenCalls) != 1 || freshOpenCalls[0].URL != "https://example.test/popup" {
		t.Fatalf("Open().Calls() reread = %#v, want original open call", freshOpenCalls)
	}
	freshCloseCalls := harness.Mocks().Close().Calls()
	if len(freshCloseCalls) != 1 {
		t.Fatalf("Close().Calls() reread = %#v, want 1 entry", freshCloseCalls)
	}
	freshPrintCalls := harness.Mocks().Print().Calls()
	if len(freshPrintCalls) != 1 {
		t.Fatalf("Print().Calls() reread = %#v, want 1 entry", freshPrintCalls)
	}
	freshScrollCalls := harness.Mocks().Scroll().Calls()
	if len(freshScrollCalls) != 2 || freshScrollCalls[0].Method != ScrollMethodTo || freshScrollCalls[0].X != 7 || freshScrollCalls[0].Y != 9 || freshScrollCalls[1].Method != ScrollMethodBy || freshScrollCalls[1].X != 2 || freshScrollCalls[1].Y != -1 {
		t.Fatalf("Scroll().Calls() reread = %#v, want original scroll calls", freshScrollCalls)
	}
	freshMatchMediaCalls := harness.Mocks().MatchMedia().Calls()
	if len(freshMatchMediaCalls) != 1 || freshMatchMediaCalls[0].Query != "(prefers-reduced-motion: reduce)" {
		t.Fatalf("MatchMedia().Calls() reread = %#v, want original query", freshMatchMediaCalls)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().Open(); got != nil {
		t.Fatalf("nil Harness.Mocks().Open() = %#v, want nil", got)
	}
	if got := nilHarness.Mocks().Close(); got != nil {
		t.Fatalf("nil Harness.Mocks().Close() = %#v, want nil", got)
	}
	if got := nilHarness.Mocks().Print(); got != nil {
		t.Fatalf("nil Harness.Mocks().Print() = %#v, want nil", got)
	}
	if got := nilHarness.Mocks().Scroll(); got != nil {
		t.Fatalf("nil Harness.Mocks().Scroll() = %#v, want nil", got)
	}
	if got := nilHarness.Mocks().MatchMedia(); got != nil {
		t.Fatalf("nil Harness.Mocks().MatchMedia() = %#v, want nil", got)
	}
}

func TestClipboardDownloadAndFileInputSnapshotsReturnCopies(t *testing.T) {
	harness, err := FromHTML(`<main><input id="upload" type="file"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.WriteClipboard("copied text"); err != nil {
		t.Fatalf("WriteClipboard() error = %v", err)
	}
	if err := harness.WriteClipboard("more text"); err != nil {
		t.Fatalf("WriteClipboard() second error = %v", err)
	}
	if err := harness.CaptureDownload("report.csv", []byte("downloaded bytes")); err != nil {
		t.Fatalf("CaptureDownload() error = %v", err)
	}
	if err := harness.SetFiles("#upload", []string{"report.csv", "archive.zip"}); err != nil {
		t.Fatalf("SetFiles(#upload) error = %v", err)
	}

	clipboardWrites := harness.Mocks().Clipboard().Writes()
	if len(clipboardWrites) != 2 {
		t.Fatalf("Clipboard().Writes() = %#v, want 2 entries", clipboardWrites)
	}
	clipboardWrites[0] = "mutated"
	clipboardWrites = append(clipboardWrites, "extra")

	downloadArtifacts := harness.Mocks().Downloads().Artifacts()
	if len(downloadArtifacts) != 1 {
		t.Fatalf("Downloads().Artifacts() = %#v, want 1 entry", downloadArtifacts)
	}
	downloadArtifacts[0].FileName = "mutated.csv"
	if len(downloadArtifacts[0].Bytes) > 0 {
		downloadArtifacts[0].Bytes[0] ^= 0xff
	}
	downloadArtifacts = append(downloadArtifacts, DownloadCapture{FileName: "extra.csv"})

	fileSelections := harness.Mocks().FileInput().Selections()
	if len(fileSelections) != 1 {
		t.Fatalf("FileInput().Selections() = %#v, want 1 entry", fileSelections)
	}
	fileSelections[0].Selector = "mutated"
	fileSelections[0].Files[0] = "mutated.txt"
	fileSelections = append(fileSelections, FileInputSelection{Selector: "extra"})

	freshClipboardWrites := harness.Mocks().Clipboard().Writes()
	if len(freshClipboardWrites) != 2 || freshClipboardWrites[0] != "copied text" || freshClipboardWrites[1] != "more text" {
		t.Fatalf("Clipboard().Writes() reread = %#v, want original writes", freshClipboardWrites)
	}
	freshDownloadArtifacts := harness.Mocks().Downloads().Artifacts()
	if len(freshDownloadArtifacts) != 1 || freshDownloadArtifacts[0].FileName != "report.csv" || string(freshDownloadArtifacts[0].Bytes) != "downloaded bytes" {
		t.Fatalf("Downloads().Artifacts() reread = %#v, want original download capture", freshDownloadArtifacts)
	}
	freshFileSelections := harness.Mocks().FileInput().Selections()
	if len(freshFileSelections) != 1 || freshFileSelections[0].Selector != "#upload" || len(freshFileSelections[0].Files) != 2 || freshFileSelections[0].Files[0] != "report.csv" || freshFileSelections[0].Files[1] != "archive.zip" {
		t.Fatalf("FileInput().Selections() reread = %#v, want original file selection", freshFileSelections)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().Clipboard(); got != nil {
		t.Fatalf("nil Harness.Mocks().Clipboard() = %#v, want nil", got)
	}
	if got := nilHarness.Mocks().Downloads(); got != nil {
		t.Fatalf("nil Harness.Mocks().Downloads() = %#v, want nil", got)
	}
	if got := nilHarness.Mocks().FileInput(); got != nil {
		t.Fatalf("nil Harness.Mocks().FileInput() = %#v, want nil", got)
	}
}

func TestDebugViewReportsInteractionsCopy(t *testing.T) {
	harness, err := FromHTML(`<main><button id="cta">Go</button><input id="name"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Focus("#name"); err != nil {
		t.Fatalf("Focus(#name) error = %v", err)
	}
	if err := harness.Click("#cta"); err != nil {
		t.Fatalf("Click(#cta) error = %v", err)
	}

	interactions := harness.Debug().Interactions()
	if len(interactions) != 2 {
		t.Fatalf("Debug().Interactions() len = %d, want 2", len(interactions))
	}
	if interactions[0].Kind != InteractionKindFocus || interactions[0].Selector != "#name" {
		t.Fatalf("Debug().Interactions()[0] = %#v, want focus #name", interactions[0])
	}
	if interactions[1].Kind != InteractionKindClick || interactions[1].Selector != "#cta" {
		t.Fatalf("Debug().Interactions()[1] = %#v, want click #cta", interactions[1])
	}

	interactions[0].Selector = "mutated"
	interactions[1].Kind = InteractionKindBlur
	fresh := harness.Debug().Interactions()
	if fresh[0].Selector != "#name" {
		t.Fatalf("Debug().Interactions()[0].Selector after mutation = %q, want %q", fresh[0].Selector, "#name")
	}
	if fresh[1].Kind != InteractionKindClick {
		t.Fatalf("Debug().Interactions()[1].Kind after mutation = %q, want %q", fresh[1].Kind, InteractionKindClick)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().Interactions(); got != nil {
		t.Fatalf("nil Debug().Interactions() = %#v, want nil", got)
	}
}

func TestDebugViewReportsEventListenersCopy(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="out"></div><script>host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#out", "beforeend", "<span>once</span>")', "capture", true); host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#out", "beforeend", "<span>stay</span>")', "bubble")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	listeners := harness.Debug().EventListeners()
	if len(listeners) != 2 {
		t.Fatalf("Debug().EventListeners() len = %d, want 2", len(listeners))
	}
	if listeners[0].NodeID == 0 || listeners[0].NodeID != listeners[1].NodeID {
		t.Fatalf("Debug().EventListeners() node ids = %#v, want same non-zero node id", listeners)
	}
	if listeners[0].Event != "click" || listeners[0].Phase != "capture" || !listeners[0].Once {
		t.Fatalf("Debug().EventListeners()[0] = %#v, want capture once click listener", listeners[0])
	}
	if listeners[1].Event != "click" || listeners[1].Phase != "bubble" || listeners[1].Once {
		t.Fatalf("Debug().EventListeners()[1] = %#v, want bubble persistent click listener", listeners[1])
	}

	listeners[0].Source = "mutated"
	fresh := harness.Debug().EventListeners()
	if fresh[0].Source != `host:insertAdjacentHTML("#out", "beforeend", "<span>once</span>")` {
		t.Fatalf("Debug().EventListeners() reread source = %q, want original source", fresh[0].Source)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	after := harness.Debug().EventListeners()
	if len(after) != 1 {
		t.Fatalf("Debug().EventListeners() after click len = %d, want 1", len(after))
	}
	if after[0].Phase != "bubble" || after[0].Once {
		t.Fatalf("Debug().EventListeners() after click = %#v, want persistent bubble listener", after[0])
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().EventListeners(); got != nil {
		t.Fatalf("nil Debug().EventListeners() = %#v, want nil", got)
	}
}

func TestDebugViewReportsPendingMicrotasksCopy(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">seed</div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().PendingMicrotasks(); len(got) != 0 {
		t.Fatalf("Debug().PendingMicrotasks() = %#v, want empty", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().PendingMicrotasks(); got != nil {
		t.Fatalf("nil Debug().PendingMicrotasks() = %#v, want nil", got)
	}
}

func TestDebugViewReportsCookieJarCopy(t *testing.T) {
	harness, err := FromHTML(`<main><script>host:setDocumentCookie("theme=dark"); host:setDocumentCookie("lang=en")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}
	jar := harness.Debug().CookieJar()
	if len(jar) != 2 || jar["theme"] != "dark" || jar["lang"] != "en" {
		t.Fatalf("Debug().CookieJar() = %#v, want theme/lang snapshot", jar)
	}
	jar["theme"] = "mutated"
	fresh := harness.Debug().CookieJar()
	if fresh["theme"] != "dark" {
		t.Fatalf("Debug().CookieJar() reread = %#v, want original cookie jar", fresh)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().CookieJar(); got != nil {
		t.Fatalf("nil Debug().CookieJar() = %#v, want nil", got)
	}
}

func TestDebugViewReportsDOMReadinessAndErrors(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got := harness.Debug().DOMReady(); got {
		t.Fatalf("Debug().DOMReady() before DOM access = %v, want false", got)
	}
	if got := harness.Debug().DOMError(); got != "" {
		t.Fatalf("Debug().DOMError() before DOM access = %q, want empty", got)
	}

	if err := harness.AssertExists("#btn"); err != nil {
		t.Fatalf("AssertExists(#btn) error = %v", err)
	}
	if got := harness.Debug().DOMReady(); !got {
		t.Fatalf("Debug().DOMReady() after DOM access = %v, want true", got)
	}
	if got := harness.Debug().DOMError(); got != "" {
		t.Fatalf("Debug().DOMError() after DOM access = %q, want empty", got)
	}
}

func TestDebugViewReportsDOMFailureDetails(t *testing.T) {
	harness, err := FromHTML(`<main><div id="broken"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("#broken"); err == nil {
		t.Fatalf("AssertExists(#broken) error = nil, want DOM parse failure")
	}
	if got := harness.Debug().DOMReady(); got {
		t.Fatalf("Debug().DOMReady() after parse failure = %v, want false", got)
	}
	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() after parse failure = %q, want parse error", got)
	}
}

func TestDebugViewReportsLastInlineScriptHTML(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script id="boot">host:setInnerHTML("#out", "<span>new</span>")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().LastInlineScriptHTML(), `<script id="boot">host:setInnerHTML("#out", "<span>new</span>")</script>`; got != want {
		t.Fatalf("Debug().LastInlineScriptHTML() = %q, want %q", got, want)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().LastInlineScriptHTML(); got != "" {
		t.Fatalf("nil Debug().LastInlineScriptHTML() = %q, want empty", got)
	}
}

func TestDebugViewReportsInitialHTMLWithoutBootstrapping(t *testing.T) {
	harness, err := FromHTML(`<main><script>host:setInnerHTML("#out", "ok")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().InitialHTML(), `<main><script>host:setInnerHTML("#out", "ok")</script></main>`; got != want {
		t.Fatalf("Debug().InitialHTML() = %q, want %q", got, want)
	}
	if got := harness.Debug().DOMReady(); got {
		t.Fatalf("Debug().DOMReady() after InitialHTML() = %v, want false", got)
	}

	var nilHarness *Harness
	if got := nilHarness.Debug().InitialHTML(); got != "" {
		t.Fatalf("nil Debug().InitialHTML() = %q, want empty", got)
	}
}

func TestConstructorHelpersCaptureExpectedState(t *testing.T) {
	t.Run("FromHTML", func(t *testing.T) {
		harness, err := FromHTML("<main>one</main>")
		if err != nil {
			t.Fatalf("FromHTML() error = %v", err)
		}
		if got, want := harness.URL(), "https://app.local/"; got != want {
			t.Fatalf("URL() = %q, want %q", got, want)
		}
		if got, want := harness.HTML(), "<main>one</main>"; got != want {
			t.Fatalf("HTML() = %q, want %q", got, want)
		}
		if got := harness.Mocks().Storage().Local(); len(got) != 0 {
			t.Fatalf("Storage().Local() = %#v, want empty", got)
		}
		if got := harness.Mocks().Storage().Session(); len(got) != 0 {
			t.Fatalf("Storage().Session() = %#v, want empty", got)
		}
	})

	t.Run("FromHTMLWithURL", func(t *testing.T) {
		harness, err := FromHTMLWithURL("https://example.test/from-url", "<main>two</main>")
		if err != nil {
			t.Fatalf("FromHTMLWithURL() error = %v", err)
		}
		if got, want := harness.URL(), "https://example.test/from-url"; got != want {
			t.Fatalf("URL() = %q, want %q", got, want)
		}
		if got, want := harness.HTML(), "<main>two</main>"; got != want {
			t.Fatalf("HTML() = %q, want %q", got, want)
		}
	})

	t.Run("FromHTMLWithLocalStorage", func(t *testing.T) {
		entries := map[string]string{"token": "abc"}
		harness, err := FromHTMLWithLocalStorage("<main>three</main>", entries)
		if err != nil {
			t.Fatalf("FromHTMLWithLocalStorage() error = %v", err)
		}
		entries["token"] = "mutated"
		if got, want := harness.Mocks().Storage().Local()["token"], "abc"; got != want {
			t.Fatalf("Storage().Local()[\"token\"] = %q, want %q", got, want)
		}
		if got := harness.Mocks().Storage().Session(); len(got) != 0 {
			t.Fatalf("Storage().Session() = %#v, want empty", got)
		}
	})

	t.Run("FromHTMLWithURLAndLocalStorage", func(t *testing.T) {
		entries := map[string]string{"token": "xyz"}
		harness, err := FromHTMLWithURLAndLocalStorage(
			"https://example.test/local",
			"<main>four</main>",
			entries,
		)
		if err != nil {
			t.Fatalf("FromHTMLWithURLAndLocalStorage() error = %v", err)
		}
		entries["token"] = "mutated"
		if got, want := harness.URL(), "https://example.test/local"; got != want {
			t.Fatalf("URL() = %q, want %q", got, want)
		}
		if got, want := harness.Mocks().Storage().Local()["token"], "xyz"; got != want {
			t.Fatalf("Storage().Local()[\"token\"] = %q, want %q", got, want)
		}
	})

	t.Run("FromHTMLWithSessionStorage", func(t *testing.T) {
		entries := map[string]string{"tab": "main"}
		harness, err := FromHTMLWithSessionStorage("<main>five</main>", entries)
		if err != nil {
			t.Fatalf("FromHTMLWithSessionStorage() error = %v", err)
		}
		entries["tab"] = "mutated"
		if got, want := harness.Mocks().Storage().Session()["tab"], "main"; got != want {
			t.Fatalf("Storage().Session()[\"tab\"] = %q, want %q", got, want)
		}
		if got := harness.Mocks().Storage().Local(); len(got) != 0 {
			t.Fatalf("Storage().Local() = %#v, want empty", got)
		}
	})

	t.Run("FromHTMLWithURLAndSessionStorage", func(t *testing.T) {
		entries := map[string]string{"tab": "detail"}
		harness, err := FromHTMLWithURLAndSessionStorage(
			"https://example.test/session",
			"<main>six</main>",
			entries,
		)
		if err != nil {
			t.Fatalf("FromHTMLWithURLAndSessionStorage() error = %v", err)
		}
		entries["tab"] = "mutated"
		if got, want := harness.URL(), "https://example.test/session"; got != want {
			t.Fatalf("URL() = %q, want %q", got, want)
		}
		if got, want := harness.Mocks().Storage().Session()["tab"], "detail"; got != want {
			t.Fatalf("Storage().Session()[\"tab\"] = %q, want %q", got, want)
		}
	})
}

func TestPromptCancelContract(t *testing.T) {
	harness, err := FromHTML("<main></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	harness.Mocks().Dialogs().QueuePromptCancel()

	got, submitted, err := harness.Prompt("Cancel?")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if got != "" || submitted {
		t.Fatalf("Prompt() = (%q, %v), want (\"\", false)", got, submitted)
	}
	if messages := harness.Mocks().Dialogs().PromptMessages(); len(messages) != 1 || messages[0] != "Cancel?" {
		t.Fatalf("PromptMessages() = %#v, want [\"Cancel?\"]", messages)
	}
}

func TestDialogMockSnapshotsReturnCopies(t *testing.T) {
	harness, err := FromHTML("<main></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	harness.Mocks().Dialogs().QueueConfirm(true)
	harness.Mocks().Dialogs().QueuePromptText("typed")
	harness.Mocks().Dialogs().QueuePromptCancel()

	if err := harness.Alert("alert"); err != nil {
		t.Fatalf("Alert() error = %v", err)
	}
	if got, err := harness.Confirm("confirm?"); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	} else if !got {
		t.Fatalf("Confirm() = false, want true")
	}
	if got, submitted, err := harness.Prompt("prompt?"); err != nil {
		t.Fatalf("Prompt() #1 error = %v", err)
	} else if got != "typed" || !submitted {
		t.Fatalf("Prompt() #1 = (%q, %v), want (%q, true)", got, submitted, "typed")
	}
	if got, submitted, err := harness.Prompt("cancel?"); err != nil {
		t.Fatalf("Prompt() #2 error = %v", err)
	} else if got != "" || submitted {
		t.Fatalf("Prompt() #2 = (%q, %v), want (\"\", false)", got, submitted)
	}

	alerts := harness.Mocks().Dialogs().Alerts()
	if len(alerts) != 1 || alerts[0] != "alert" {
		t.Fatalf("Dialogs().Alerts() = %#v, want [\"alert\"]", alerts)
	}
	alerts[0] = "mutated"
	alerts = append(alerts, "extra")

	confirms := harness.Mocks().Dialogs().ConfirmMessages()
	if len(confirms) != 1 || confirms[0] != "confirm?" {
		t.Fatalf("Dialogs().ConfirmMessages() = %#v, want [\"confirm?\"]", confirms)
	}
	confirms[0] = "mutated"
	confirms = append(confirms, "extra")

	prompts := harness.Mocks().Dialogs().PromptMessages()
	if len(prompts) != 2 || prompts[0] != "prompt?" || prompts[1] != "cancel?" {
		t.Fatalf("Dialogs().PromptMessages() = %#v, want [\"prompt?\", \"cancel?\"]", prompts)
	}
	prompts[0] = "mutated"
	prompts = append(prompts, "extra")

	freshAlerts := harness.Mocks().Dialogs().Alerts()
	if len(freshAlerts) != 1 || freshAlerts[0] != "alert" {
		t.Fatalf("Dialogs().Alerts() reread = %#v, want [\"alert\"]", freshAlerts)
	}
	freshConfirms := harness.Mocks().Dialogs().ConfirmMessages()
	if len(freshConfirms) != 1 || freshConfirms[0] != "confirm?" {
		t.Fatalf("Dialogs().ConfirmMessages() reread = %#v, want [\"confirm?\"]", freshConfirms)
	}
	freshPrompts := harness.Mocks().Dialogs().PromptMessages()
	if len(freshPrompts) != 2 || freshPrompts[0] != "prompt?" || freshPrompts[1] != "cancel?" {
		t.Fatalf("Dialogs().PromptMessages() reread = %#v, want [\"prompt?\", \"cancel?\"]", freshPrompts)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().Dialogs(); got != nil {
		t.Fatalf("nil Harness.Mocks().Dialogs() = %#v, want nil", got)
	}
}

func TestInlineScriptsCanDriveHistoryThroughPublicFacade(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/app",
		`<main><script>host:historyPushState("step-1", "", "/step-1"); host:historyReplaceState("step-2", "", "step-2"); host:historyBack(); host:historyForward(); host:historyGo(-1)</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got := harness.Debug().DumpDOM(); got == "" {
		t.Fatalf("Debug().DumpDOM() after history script = empty string, want DOM snapshot")
	}
	if got, want := harness.URL(), "https://example.test/app"; got != want {
		t.Fatalf("URL() after history script = %q, want %q", got, want)
	}
	if got, want := harness.Mocks().Location().CurrentURL(), "https://example.test/app"; got != want {
		t.Fatalf("Location().CurrentURL() after history script = %q, want %q", got, want)
	}
	if got := harness.Mocks().Location().Navigations(); len(got) != 5 || got[0] != "https://example.test/step-1" || got[1] != "https://example.test/step-2" || got[2] != "https://example.test/app" || got[3] != "https://example.test/step-2" || got[4] != "https://example.test/app" {
		t.Fatalf("Location().Navigations() after history script = %#v, want ordered history navigations", got)
	}
}

func TestInlineScriptsCanDriveWebStorageThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().
		URL("https://example.test/app").
		LocalStorage(map[string]string{"theme": "dark"}).
		SessionStorage(map[string]string{"tab": "main"}).
		HTML(`<main><div id="out"></div><script>host:setTextContent("#out", expr(host:localStorageGetItem("theme"))); host:localStorageSetItem("accent", "blue"); host:sessionStorageRemoveItem("tab"); host:localStorageClear()</script></main>`).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := harness.AssertText("#out", "dark"); err != nil {
		t.Fatalf("AssertText(#out, dark) error = %v", err)
	}
	if got := harness.Mocks().Storage().Local(); len(got) != 0 {
		t.Fatalf("Storage().Local() after bootstrap web storage script = %#v, want empty", got)
	}
	if got := harness.Mocks().Storage().Session(); len(got) != 0 {
		t.Fatalf("Storage().Session() after bootstrap web storage script = %#v, want empty", got)
	}
	events := harness.Mocks().Storage().Events()
	if len(events) != 5 || events[0].Op != "seed" || events[1].Op != "seed" || events[2].Op != "set" || events[3].Op != "remove" || events[4].Op != "clear" {
		t.Fatalf("Storage().Events() after web storage script = %#v, want seed/seed/set/remove/clear", events)
	}
}

func TestInlineScriptsCanUseTreeMutationHelpersThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main><div id="src"><span>old</span></div><div id="out">before</div><script>host:replaceChildren("#out", "<em>fresh</em>"); host:cloneNode("#src", true)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="src"><span>old</span></div><div id="src"><span>old</span></div><div id="out"><em>fresh</em></div><script>host:replaceChildren("#out", "<em>fresh</em>"); host:cloneNode("#src", true)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after tree mutation host helpers = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanConstructAndReorderNodesThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main></main><script>const root = host:querySelector("main"); const span = host:createElement("span"); const em = host:createElement("em"); const strong = host:createElement("strong"); host:appendChild(expr(root), expr(span)); host:appendChild(expr(root), expr(em)); host:appendChild(expr(root), expr(strong)); host:insertBefore(expr(root), expr(span), expr(strong)); const removed = host:removeChild(expr(root), expr(span)); host:appendChild(expr(root), expr(removed))</script>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><em></em><strong></strong><span></span></main><script>const root = host:querySelector("main"); const span = host:createElement("span"); const em = host:createElement("em"); const strong = host:createElement("strong"); host:appendChild(expr(root), expr(span)); host:appendChild(expr(root), expr(em)); host:appendChild(expr(root), expr(strong)); host:insertBefore(expr(root), expr(span), expr(strong)); const removed = host:removeChild(expr(root), expr(span)); host:appendChild(expr(root), expr(removed))</script>`; got != want {
		t.Fatalf("Debug().DumpDOM() after node construction host helpers = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanCreateTextNodesReplaceChildrenAndInsertAdjacentNodesThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main id="wrap"><div id="target"><span id="keep">keep</span></div><p id="tail">tail</p></main><script>const target = host:querySelector("#target"); const keep = host:querySelector("#keep"); const seed = host:createTextNode("seed"); host:replaceChild(expr(target), expr(seed), expr(keep)); const em = host:createElement("em"); const strong = host:createElement("strong"); host:insertAdjacentElement(expr(target), "afterbegin", expr(em)); host:insertAdjacentText(expr(target), "beforeend", " tail"); host:insertAdjacentElement(expr(target), "beforebegin", expr(strong)); host:insertAdjacentText(expr(target), "afterend", "!")</script>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main id="wrap"><strong></strong><div id="target"><em></em>seed tail</div>!<p id="tail">tail</p></main><script>const target = host:querySelector("#target"); const keep = host:querySelector("#keep"); const seed = host:createTextNode("seed"); host:replaceChild(expr(target), expr(seed), expr(keep)); const em = host:createElement("em"); const strong = host:createElement("strong"); host:insertAdjacentElement(expr(target), "afterbegin", expr(em)); host:insertAdjacentText(expr(target), "beforeend", " tail"); host:insertAdjacentElement(expr(target), "beforebegin", expr(strong)); host:insertAdjacentText(expr(target), "afterend", "!")</script>`; got != want {
		t.Fatalf("Debug().DumpDOM() after low-level node construction host helpers = %q, want %q", got, want)
	}
}

func TestInlineScriptsRejectInvalidLowLevelNodeConstructionPositionsThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	err = harness.WriteHTML(`<main id="wrap"><div id="target"><span id="keep"></span></div></main><script>const target = host:querySelector("#target"); const em = host:createElement("em"); host:insertAdjacentElement(expr(target), "sideways", expr(em))</script>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want invalid-position error")
	}
	if harness == nil {
		t.Fatalf("WriteHTML() harness = nil, want harness instance")
	}
}

func TestInlineScriptsCanReadTextContentThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main><div id="src">seed</div><div id="out"></div><script>host:setTextContent("#out", expr(host:textContent("#src")))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="src">seed</div><div id="out">seed</div><script>host:setTextContent("#out", expr(host:textContent("#src")))</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after textContent getter = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanReadElementReflectionSurfacesThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main><div id="box" class="alpha beta" style="color: green; background: transparent" data-x="1">Hello <strong>world</strong></div><div id="probe"></div><script>const box = document.querySelector("#box"); const firstAttr = box.attributes.item(0); const styleAttr = box.attributes.namedItem("style"); host:setTextContent("#probe", expr(box.className + "|" + box.innerText + "|" + box.outerText + "|" + box.style.cssText + "|" + box.style.length + "|" + box.style.item(0) + "|" + box.style.getPropertyValue("background") + "|" + box.attributes.length + "|" + firstAttr.name + "=" + firstAttr.value + "|" + styleAttr.value + "|" + box.attributes.namedItem("data-x").value))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="box" class="alpha beta" style="color: green; background: transparent" data-x="1">Hello <strong>world</strong></div><div id="probe">alpha beta|Hello world|Hello world|color: green; background: transparent|2|color|transparent|4|id=box|color: green; background: transparent|1</div><script>const box = document.querySelector("#box"); const firstAttr = box.attributes.item(0); const styleAttr = box.attributes.namedItem("style"); host:setTextContent("#probe", expr(box.className + "|" + box.innerText + "|" + box.outerText + "|" + box.style.cssText + "|" + box.style.length + "|" + box.style.item(0) + "|" + box.style.getPropertyValue("background") + "|" + box.attributes.length + "|" + firstAttr.name + "=" + firstAttr.value + "|" + styleAttr.value + "|" + box.attributes.namedItem("data-x").value))</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after element reflection getters = %q, want %q", got, want)
	}
}

func TestInlineScriptsSupportStandardEventListenersThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="out"></div><script>const out = document.querySelector("#out"); const add = (label) => { const base = out.textContent; const next = base ? base + "|" + label : label; host.setTextContent("#out", next); }; const btn = document.querySelector("#btn"); btn.addEventListener("click", () => add("click")); btn.addEventListener("keydown", (event) => { if (event.key === "Escape") { add("element"); } }); document.addEventListener("keydown", (event) => { if (event.key === "Escape") { add("document"); } }); window.addEventListener("keydown", (event) => { if (event.key === "Escape") { add("window"); } });</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}
	if err := harness.DispatchKeyboard("#btn"); err != nil {
		t.Fatalf("DispatchKeyboard(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="out">click|element|document|window</div><script>const out = document.querySelector("#out"); const add = (label) => { const base = out.textContent; const next = base ? base + "|" + label : label; host.setTextContent("#out", next); }; const btn = document.querySelector("#btn"); btn.addEventListener("click", () => add("click")); btn.addEventListener("keydown", (event) => { if (event.key === "Escape") { add("element"); } }); document.addEventListener("keydown", (event) => { if (event.key === "Escape") { add("document"); } }); window.addEventListener("keydown", (event) => { if (event.key === "Escape") { add("window"); } });</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after standard event listeners = %q, want %q", got, want)
	}
}

func TestInlineScriptsSupportClassListAndDetailsOpenThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main><details id="panel" class="beta"><summary>More</summary><div>Body</div></details><div id="box" class="base"></div><div id="state"></div><script>const panel = document.querySelector("#panel"); panel.classList.add("expanded"); panel.classList.remove("beta"); panel.classList.toggle("ready"); panel.setAttribute("open", ""); host.setTextContent("#state", panel.open ? "open" : "closed"); const box = document.querySelector("#box"); box.classList.toggle("active");</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><details id="panel" class="expanded ready" open=""><summary>More</summary><div>Body</div></details><div id="box" class="base active"></div><div id="state">open</div><script>const panel = document.querySelector("#panel"); panel.classList.add("expanded"); panel.classList.remove("beta"); panel.classList.toggle("ready"); panel.setAttribute("open", ""); host.setTextContent("#state", panel.open ? "open" : "closed"); const box = document.querySelector("#box"); box.classList.toggle("active");</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after classList/open mutation = %q, want %q", got, want)
	}
}

func TestInlineScriptsSupportStandardNodeConstructionThroughPublicFacade(t *testing.T) {
	harness, err := FromHTML(`<main id="root"></main><script>const root = document.querySelector("#root"); const span = document.createElement("span"); span.setAttribute("data-x", "1"); root.appendChild(span); root.removeChild(span); root.appendChild(span);</script>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main id="root"><span data-x="1"></span></main><script>const root = document.querySelector("#root"); const span = document.createElement("span"); span.setAttribute("data-x", "1"); root.appendChild(span); root.removeChild(span); root.appendChild(span);</script>`; got != want {
		t.Fatalf("Debug().DumpDOM() after standard node construction = %q, want %q", got, want)
	}
}

func TestInlineScriptsSupportSelectAndExecCommandCopyThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	harness.Mocks().Clipboard().SeedText("initial")
	err = harness.WriteHTML(`<main><input id="copy" value="seed"><script>const field = document.querySelector("#copy"); field.select(); document.execCommand("copy");</script></main>`)
	if err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	got, err := harness.ReadClipboard()
	if err != nil {
		t.Fatalf("ReadClipboard() error = %v", err)
	}
	if got != "seed" {
		t.Fatalf("ReadClipboard() = %q, want seed", got)
	}
}

func TestInlineScriptsSupportConfirmThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	harness.Mocks().Dialogs().QueueConfirm(true)
	err = harness.WriteHTML(`<main><div id="out"></div><script>const ok = window.confirm("Continue?"); host.setTextContent("#out", ok ? "yes" : "no")</script></main>`)
	if err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">yes</div><script>const ok = window.confirm("Continue?"); host.setTextContent("#out", ok ? "yes" : "no")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after confirm = %q, want %q", got, want)
	}
}

func TestInlineScriptsRejectConfirmWithoutQueuedResponseThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	err = harness.WriteHTML(`<main><script>window.confirm("missing")</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want queued response error")
	}
	if !strings.Contains(err.Error(), "queued response") {
		t.Fatalf("WriteHTML() error = %q, want queued response error", err)
	}
}

func TestInlineScriptsSupportPromptThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	harness.Mocks().Dialogs().QueuePromptText("Ada")
	err = harness.WriteHTML(`<main><div id="out"></div><script>const value = window.prompt("Your name?"); host.setTextContent("#out", value === null ? "canceled" : value)</script></main>`)
	if err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">Ada</div><script>const value = window.prompt("Your name?"); host.setTextContent("#out", value === null ? "canceled" : value)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after prompt = %q, want %q", got, want)
	}
}

func TestInlineScriptsRejectPromptWithoutQueuedResponseThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	err = harness.WriteHTML(`<main><script>window.prompt("missing")</script></main>`)
	if err == nil {
		t.Fatalf("WriteHTML() error = nil, want queued response error")
	}
	if !strings.Contains(err.Error(), "queued response") {
		t.Fatalf("WriteHTML() error = %q, want queued response error", err)
	}
}

func TestInlineScriptsSupportMatchMediaMatchesThroughPublicFacade(t *testing.T) {
	harness, err := NewHarnessBuilder().
		MatchMedia(map[string]bool{"(max-width: 1079px)": true}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	err = harness.WriteHTML(`<main><div id="mode"></div><script>const mobile = window.matchMedia("(max-width: 1079px)").matches; host.setTextContent("#mode", mobile ? "mobile" : "desktop")</script></main>`)
	if err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="mode">mobile</div><script>const mobile = window.matchMedia("(max-width: 1079px)").matches; host.setTextContent("#mode", mobile ? "mobile" : "desktop")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after matchMedia.matches = %q, want %q", got, want)
	}
	if got := harness.Debug().MatchMediaCalls(); len(got) != 1 || got[0].Query != "(max-width: 1079px)" {
		t.Fatalf("Debug().MatchMediaCalls() = %#v, want one max-width query", got)
	}
}

func TestInlineScriptsCanRegisterTemplateWindowLifecycleListeners(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script>window.addEventListener("online", () => {}); window.addEventListener("offline", () => {}); window.addEventListener("resize", () => {}); host.setTextContent("#out", navigator.onLine ? "online" : "offline")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">online</div><script>window.addEventListener("online", () => {}); window.addEventListener("offline", () => {}); window.addEventListener("resize", () => {}); host.setTextContent("#out", navigator.onLine ? "online" : "offline")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after template window listeners = %q, want %q", got, want)
	}

	listeners := harness.Debug().EventListeners()
	if len(listeners) != 3 {
		t.Fatalf("Debug().EventListeners() len = %d, want 3", len(listeners))
	}
	if listeners[0].NodeID == 0 || listeners[0].NodeID != listeners[1].NodeID || listeners[1].NodeID != listeners[2].NodeID {
		t.Fatalf("Debug().EventListeners() node ids = %#v, want same non-zero node id", listeners)
	}
	if listeners[0].Event != "online" || listeners[0].Phase != "bubble" {
		t.Fatalf("Debug().EventListeners()[0] = %#v, want bubble online listener", listeners[0])
	}
	if listeners[1].Event != "offline" || listeners[1].Phase != "bubble" {
		t.Fatalf("Debug().EventListeners()[1] = %#v, want bubble offline listener", listeners[1])
	}
	if listeners[2].Event != "resize" || listeners[2].Phase != "bubble" {
		t.Fatalf("Debug().EventListeners()[2] = %#v, want bubble resize listener", listeners[2])
	}
}

func TestStorageViewReturnsCopies(t *testing.T) {
	harness, err := NewHarnessBuilder().
		URL("https://example.test/").
		HTML("<main></main>").
		LocalStorage(map[string]string{"local": "value"}).
		SessionStorage(map[string]string{"session": "value"}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	local := harness.Mocks().Storage().Local()
	local["local"] = "mutated"
	local["extra"] = "added"

	session := harness.Mocks().Storage().Session()
	session["session"] = "mutated"

	harness.Mocks().Storage().SeedLocal("theme", "dark")
	harness.Mocks().Storage().SeedSession("tab", "main")

	events := harness.Mocks().Storage().Events()
	if len(events) != 4 || events[0].Scope != "local" || events[0].Op != "seed" || events[0].Key != "local" || events[0].Value != "value" || events[1].Scope != "session" || events[1].Op != "seed" || events[1].Key != "session" || events[1].Value != "value" || events[2].Scope != "local" || events[2].Op != "seed" || events[2].Key != "theme" || events[2].Value != "dark" || events[3].Scope != "session" || events[3].Op != "seed" || events[3].Key != "tab" || events[3].Value != "main" {
		t.Fatalf("Storage().Events() = %#v, want four storage events", events)
	}
	events[0].Value = "mutated"
	events = append(events, StorageEvent{Scope: "extra"})

	freshLocal := harness.Mocks().Storage().Local()
	if got, want := freshLocal["local"], "value"; got != want {
		t.Fatalf("Storage().Local()[\"local\"] = %q, want %q", got, want)
	}
	if _, ok := freshLocal["extra"]; ok {
		t.Fatalf("Storage().Local()[\"extra\"] should not exist")
	}

	freshSession := harness.Mocks().Storage().Session()
	if got, want := freshSession["session"], "value"; got != want {
		t.Fatalf("Storage().Session()[\"session\"] = %q, want %q", got, want)
	}
	freshEvents := harness.Mocks().Storage().Events()
	if len(freshEvents) != 4 || freshEvents[0].Op != "seed" || freshEvents[0].Value != "value" || freshEvents[1].Op != "seed" || freshEvents[1].Value != "value" || freshEvents[2].Op != "seed" || freshEvents[2].Value != "dark" || freshEvents[3].Op != "seed" || freshEvents[3].Value != "main" {
		t.Fatalf("Storage().Events() reread = %#v, want original storage events", freshEvents)
	}

	var nilHarness *Harness
	if got := nilHarness.Mocks().Storage(); got != nil {
		t.Fatalf("nil Harness.Mocks().Storage() = %#v, want nil", got)
	}
}

func TestInteractionSliceReportsFocusAndLog(t *testing.T) {
	harness, err := FromHTML(`<main><button id="cta">Go</button><input id="name"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Focus(" #name "); err != nil {
		t.Fatalf("Focus(#name) error = %v", err)
	}
	if got, want := harness.Debug().FocusedSelector(), "#name"; got != want {
		t.Fatalf("Debug().FocusedSelector() after Focus = %q, want %q", got, want)
	}
	if err := harness.AssertExists("input:focus"); err != nil {
		t.Fatalf("AssertExists(input:focus) after Focus error = %v", err)
	}
	if err := harness.AssertExists("input:focus-visible"); err != nil {
		t.Fatalf("AssertExists(input:focus-visible) after Focus error = %v", err)
	}
	if err := harness.AssertExists("main:focus-within"); err != nil {
		t.Fatalf("AssertExists(main:focus-within) after Focus error = %v", err)
	}

	if err := harness.Click("#cta"); err != nil {
		t.Fatalf("Click(#cta) error = %v", err)
	}
	if err := harness.Blur(); err != nil {
		t.Fatalf("Blur() error = %v", err)
	}
	if got := harness.Debug().FocusedSelector(); got != "" {
		t.Fatalf("Debug().FocusedSelector() after Blur = %q, want empty", got)
	}
	if err := harness.AssertExists("input:focus"); err == nil {
		t.Fatalf("AssertExists(input:focus) after Blur error = nil, want no match")
	}
	if err := harness.AssertExists("input:focus-visible"); err == nil {
		t.Fatalf("AssertExists(input:focus-visible) after Blur error = nil, want no match")
	}

	log := harness.Debug().Interactions()
	if len(log) != 3 {
		t.Fatalf("Debug().Interactions() len = %d, want 3", len(log))
	}
	if log[0].Kind != InteractionKindFocus || log[0].Selector != "#name" {
		t.Fatalf("Debug().Interactions()[0] = %#v, want focus #name", log[0])
	}
	if log[1].Kind != InteractionKindClick || log[1].Selector != "#cta" {
		t.Fatalf("Debug().Interactions()[1] = %#v, want click #cta", log[1])
	}
	if log[2].Kind != InteractionKindBlur || log[2].Selector != "#name" {
		t.Fatalf("Debug().Interactions()[2] = %#v, want blur #name", log[2])
	}

	log[0].Selector = "mutated"
	if fresh := harness.Debug().Interactions(); fresh[0].Selector != "#name" {
		t.Fatalf("Debug().Interactions() should return copies, got %#v", fresh[0])
	}
}

func TestInteractionSliceRejectsMissingTargets(t *testing.T) {
	harness, err := FromHTML(`<main><button id="cta">Go</button></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("main[item="); err == nil {
		t.Fatalf("Click(main[item=) error = nil, want selector syntax error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("Click(main[item=) error = %#v, want event error", err)
	}

	if err := harness.Focus("#missing"); err == nil {
		t.Fatalf("Focus(#missing) error = nil, want missing-element error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("Focus(#missing) error = %#v, want event error", err)
	}

	if err := harness.Blur(); err != nil {
		t.Fatalf("Blur() error = %v", err)
	}

	if got := harness.Debug().Interactions(); len(got) != 1 || got[0].Kind != InteractionKindBlur {
		t.Fatalf("Debug().Interactions() = %#v, want one blur event after rejected interactions", got)
	}
}

func TestInteractionSliceSupportsBoundedCombinators(t *testing.T) {
	harness, err := FromHTML(`<main><section><button id="cta">Go</button></section><input id="name"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("main section > button"); err != nil {
		t.Fatalf("Click(main section > button) error = %v", err)
	}
	if err := harness.Focus("section + input"); err != nil {
		t.Fatalf("Focus(section + input) error = %v", err)
	}
	if err := harness.AssertExists("section + input"); err != nil {
		t.Fatalf("AssertExists(section + input) error = %v", err)
	}

	if got, want := harness.Debug().FocusedSelector(), "section + input"; got != want {
		t.Fatalf("Debug().FocusedSelector() = %q, want %q", got, want)
	}

	log := harness.Debug().Interactions()
	if len(log) != 2 {
		t.Fatalf("Debug().Interactions() len = %d, want 2", len(log))
	}
	if log[0].Kind != InteractionKindClick || log[0].Selector != "main section > button" {
		t.Fatalf("Debug().Interactions()[0] = %#v, want click main section > button", log[0])
	}
	if log[1].Kind != InteractionKindFocus || log[1].Selector != "section + input" {
		t.Fatalf("Debug().Interactions()[1] = %#v, want focus section + input", log[1])
	}
}

func TestInteractionSliceSupportsBoundedPseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><input id="enabled" type="text"><input id="flag" type="checkbox" checked><input id="off" type="text" disabled><div id="empty"></div><p id="last">two</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists(":root"); err != nil {
		t.Fatalf("AssertExists(:root) error = %v", err)
	}
	if err := harness.AssertExists("input:checked"); err != nil {
		t.Fatalf("AssertExists(input:checked) error = %v", err)
	}
	if err := harness.AssertExists("input:default"); err != nil {
		t.Fatalf("AssertExists(input:default) error = %v", err)
	}
	if err := harness.AssertExists("input:disabled"); err != nil {
		t.Fatalf("AssertExists(input:disabled) error = %v", err)
	}
	if err := harness.AssertExists("div:empty"); err != nil {
		t.Fatalf("AssertExists(div:empty) error = %v", err)
	}
	if err := harness.Focus("input:first-child"); err != nil {
		t.Fatalf("Focus(input:first-child) error = %v", err)
	}
	if got, want := harness.Debug().FocusedSelector(), "input:first-child"; got != want {
		t.Fatalf("Debug().FocusedSelector() = %q, want %q", got, want)
	}
	if err := harness.AssertExists("p:last-child"); err != nil {
		t.Fatalf("AssertExists(p:last-child) error = %v", err)
	}
}

func TestInteractionSliceSupportsIndeterminatePseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><input id="mixed" type="checkbox" indeterminate><input id="radio-a" type="radio" name="size"><input id="radio-b" type="radio" name="size"><progress id="task"></progress><progress id="done" value="42"></progress></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("input:indeterminate"); err != nil {
		t.Fatalf("AssertExists(input:indeterminate) error = %v", err)
	}
	if err := harness.AssertExists("progress:indeterminate"); err != nil {
		t.Fatalf("AssertExists(progress:indeterminate) error = %v", err)
	}
	if err := harness.AssertExists("#radio-a:indeterminate"); err != nil {
		t.Fatalf("AssertExists(#radio-a:indeterminate) error = %v", err)
	}
	if err := harness.AssertExists("#radio-b:indeterminate"); err != nil {
		t.Fatalf("AssertExists(#radio-b:indeterminate) error = %v", err)
	}

	if err := harness.SetChecked("#radio-a", true); err != nil {
		t.Fatalf("SetChecked(#radio-a) error = %v", err)
	}
	if err := harness.AssertExists("#radio-a:indeterminate"); err == nil {
		t.Fatalf("AssertExists(#radio-a:indeterminate) after SetChecked error = nil, want no match")
	}
	if err := harness.AssertExists("#radio-b:indeterminate"); err == nil {
		t.Fatalf("AssertExists(#radio-b:indeterminate) after SetChecked error = nil, want no match")
	}
}

func TestInteractionSliceSupportsLinkAndPlaceholderPseudoClasses(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/base/",
		`<main><a id="nav" href="/next">Go</a><map><area id="area" href="/popup" alt="Open"></map><form id="profile"><button id="submit-1" type="submit">Save</button><input id="placeholder" type="text" placeholder="Name"><textarea id="story" placeholder="Story"></textarea></form></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.AssertExists("a:link"); err != nil {
		t.Fatalf("AssertExists(a:link) error = %v", err)
	}
	if err := harness.AssertExists("area:link"); err != nil {
		t.Fatalf("AssertExists(area:link) error = %v", err)
	}
	if err := harness.AssertExists("a:any-link"); err != nil {
		t.Fatalf("AssertExists(a:any-link) error = %v", err)
	}
	if err := harness.AssertExists("area:any-link"); err != nil {
		t.Fatalf("AssertExists(area:any-link) error = %v", err)
	}
	if err := harness.AssertExists("input:placeholder-shown"); err != nil {
		t.Fatalf("AssertExists(input:placeholder-shown) error = %v", err)
	}
	if err := harness.AssertExists("textarea:placeholder-shown"); err != nil {
		t.Fatalf("AssertExists(textarea:placeholder-shown) error = %v", err)
	}
	if err := harness.AssertExists("input:blank"); err != nil {
		t.Fatalf("AssertExists(input:blank) error = %v", err)
	}
	if err := harness.AssertExists("textarea:blank"); err != nil {
		t.Fatalf("AssertExists(textarea:blank) error = %v", err)
	}
	if err := harness.AssertExists("button:default"); err != nil {
		t.Fatalf("AssertExists(button:default) error = %v", err)
	}
}

func TestInteractionSliceSupportsLocalLinkPseudoClass(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/page#top",
		`<main><a id="self" href="#top">Self</a><a id="next" href="/next">Next</a><map><area id="area-self" href="#top" alt="Self"></map></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.AssertExists("a:local-link"); err != nil {
		t.Fatalf("AssertExists(a:local-link) error = %v", err)
	}
	if err := harness.AssertExists("area:local-link"); err != nil {
		t.Fatalf("AssertExists(area:local-link) error = %v", err)
	}
	if err := harness.AssertExists("#self:local-link"); err != nil {
		t.Fatalf("AssertExists(#self:local-link) error = %v", err)
	}
	if err := harness.AssertExists("#next:local-link"); err == nil {
		t.Fatalf("AssertExists(#next:local-link) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsBlankPseudoClassForCheckableAndSelectControls(t *testing.T) {
	harness, err := FromHTML(`<main><input id="checkbox-off" type="checkbox"><input id="checkbox-on" type="checkbox" checked><input id="radio-off" type="radio" name="choice"><input id="radio-on" type="radio" name="choice" checked><select id="empty-select"><option value="a">A</option></select><select id="filled-select"><option value="b" selected>B</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("input:blank"); err != nil {
		t.Fatalf("AssertExists(input:blank) error = %v", err)
	}
	if err := harness.AssertExists("select:blank"); err != nil {
		t.Fatalf("AssertExists(select:blank) error = %v", err)
	}
	if err := harness.AssertExists("#checkbox-off:blank"); err != nil {
		t.Fatalf("AssertExists(#checkbox-off:blank) error = %v", err)
	}
	if err := harness.AssertExists("#radio-off:blank"); err != nil {
		t.Fatalf("AssertExists(#radio-off:blank) error = %v", err)
	}
	if err := harness.AssertExists("#checkbox-on:blank"); err == nil {
		t.Fatalf("AssertExists(#checkbox-on:blank) error = nil, want no match")
	}
	if err := harness.AssertExists("#radio-on:blank"); err == nil {
		t.Fatalf("AssertExists(#radio-on:blank) error = nil, want no match")
	}
	if err := harness.AssertExists("#empty-select:blank"); err != nil {
		t.Fatalf("AssertExists(#empty-select:blank) error = %v", err)
	}
	if err := harness.AssertExists("#filled-select:blank"); err == nil {
		t.Fatalf("AssertExists(#filled-select:blank) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsVisitedPseudoClass(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/page",
		`<main><a id="nav" href="https://example.test/visited">Go</a><a id="other" href="https://example.test/other">Other</a><map><area id="area" href="https://example.test/visited" alt="Visited"></map></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.Navigate("https://example.test/visited"); err != nil {
		t.Fatalf("Navigate() error = %v", err)
	}

	if err := harness.AssertExists("a:visited"); err != nil {
		t.Fatalf("AssertExists(a:visited) error = %v", err)
	}
	if err := harness.AssertExists("area:visited"); err != nil {
		t.Fatalf("AssertExists(area:visited) error = %v", err)
	}
	if err := harness.AssertExists("#nav:visited"); err != nil {
		t.Fatalf("AssertExists(#nav:visited) error = %v", err)
	}
	if err := harness.AssertExists("#other:visited"); err == nil {
		t.Fatalf("AssertExists(#other:visited) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsAttributeSelectors(t *testing.T) {
	harness, err := FromHTML(`<main><div id="panel" data-kind="panel"><a id="nav" href="/next" data-role="nav">Go</a><input id="name" type="text"><p id="flag" hidden></p><span id="meta" data-tags="alpha beta gamma" data-locale="en-US" data-note="prefix-middle-suffix" data-code="abc123"></span></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("div[data-kind]"); err != nil {
		t.Fatalf("AssertExists(div[data-kind]) error = %v", err)
	}
	if err := harness.AssertExists("a[href]"); err != nil {
		t.Fatalf("AssertExists(a[href]) error = %v", err)
	}
	if err := harness.AssertExists("a[href=\"/next\"]"); err != nil {
		t.Fatalf("AssertExists(a[href=\"/next\"]) error = %v", err)
	}
	if err := harness.AssertExists("a[data-role=nav]"); err != nil {
		t.Fatalf("AssertExists(a[data-role=nav]) error = %v", err)
	}
	if err := harness.AssertExists("input[type=text]"); err != nil {
		t.Fatalf("AssertExists(input[type=text]) error = %v", err)
	}
	if err := harness.AssertExists("p[hidden]"); err != nil {
		t.Fatalf("AssertExists(p[hidden]) error = %v", err)
	}
	if err := harness.AssertExists("span[data-tags~=beta]"); err != nil {
		t.Fatalf("AssertExists(span[data-tags~=beta]) error = %v", err)
	}
	if err := harness.AssertExists("span[data-locale|=en]"); err != nil {
		t.Fatalf("AssertExists(span[data-locale|=en]) error = %v", err)
	}
	if err := harness.AssertExists("span[data-note^=prefix]"); err != nil {
		t.Fatalf("AssertExists(span[data-note^=prefix]) error = %v", err)
	}
	if err := harness.AssertExists("span[data-note$=suffix]"); err != nil {
		t.Fatalf("AssertExists(span[data-note$=suffix]) error = %v", err)
	}
	if err := harness.AssertExists("span[data-note*=middle]"); err != nil {
		t.Fatalf("AssertExists(span[data-note*=middle]) error = %v", err)
	}
	if err := harness.AssertExists("span[data-tags~=BETA i]"); err != nil {
		t.Fatalf("AssertExists(span[data-tags~=BETA i]) error = %v", err)
	}
	if err := harness.AssertExists("input[type=TEXT i]"); err != nil {
		t.Fatalf("AssertExists(input[type=TEXT i]) error = %v", err)
	}
	if err := harness.AssertExists("a[data-role=missing]"); err == nil {
		t.Fatalf("AssertExists(a[data-role=missing]) error = nil, want no match")
	}
	if err := harness.AssertExists("span[data-tags~=delta]"); err == nil {
		t.Fatalf("AssertExists(span[data-tags~=delta]) error = nil, want no match")
	}
	if err := harness.AssertExists("span[data-tags~=BETA s]"); err == nil {
		t.Fatalf("AssertExists(span[data-tags~=BETA s]) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsMoreBoundedPseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><h1 id="title">Title</h1><details id="details" open><summary>Sum</summary></details><dialog id="dialog" open></dialog><form id="profile"><input id="required" type="text" required><input id="optional" type="text"><input id="readonly" type="text" readonly><textarea id="editable"></textarea><textarea id="readonly-ta" readonly>Locked</textarea></form></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("input:required"); err != nil {
		t.Fatalf("AssertExists(input:required) error = %v", err)
	}
	if err := harness.AssertExists("input:optional"); err != nil {
		t.Fatalf("AssertExists(input:optional) error = %v", err)
	}
	if err := harness.AssertExists("input:read-write"); err != nil {
		t.Fatalf("AssertExists(input:read-write) error = %v", err)
	}
	if err := harness.AssertExists("input:read-only"); err != nil {
		t.Fatalf("AssertExists(input:read-only) error = %v", err)
	}
	if err := harness.AssertExists("textarea:read-write"); err != nil {
		t.Fatalf("AssertExists(textarea:read-write) error = %v", err)
	}
	if err := harness.AssertExists("textarea:read-only"); err != nil {
		t.Fatalf("AssertExists(textarea:read-only) error = %v", err)
	}
	if err := harness.AssertExists("h1:heading"); err != nil {
		t.Fatalf("AssertExists(h1:heading) error = %v", err)
	}
	if err := harness.AssertExists("details:open"); err != nil {
		t.Fatalf("AssertExists(details:open) error = %v", err)
	}
	if err := harness.AssertExists("dialog:open"); err != nil {
		t.Fatalf("AssertExists(dialog:open) error = %v", err)
	}
}

func TestInteractionSliceSupportsDisabledFieldsetAndOptgroupPseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><form id="profile"><fieldset id="outer" disabled><legend id="legend"><span><input id="legend-input" type="text"></span></legend><input id="disabled-required" type="text" required><input id="disabled-optional" type="text"><textarea id="disabled-textarea"></textarea><select id="mode"><optgroup id="disabled-group" disabled label="Disabled"><option id="disabled-option" value="a">A</option></optgroup><optgroup id="enabled-group" label="Enabled"><option id="enabled-option" value="b">B</option></optgroup></select><fieldset id="inner"><input id="inner-input" type="text"></fieldset></fieldset><fieldset id="plain-fieldset"><input id="plain-input" type="text"></fieldset><input id="outside-required" type="text" required value="Ada"><input id="outside-optional" type="text"><textarea id="outside-textarea"></textarea></form></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("fieldset:disabled"); err != nil {
		t.Fatalf("AssertExists(fieldset:disabled) error = %v", err)
	}
	if err := harness.AssertExists("fieldset:enabled"); err != nil {
		t.Fatalf("AssertExists(fieldset:enabled) error = %v", err)
	}
	if err := harness.AssertExists("input:required"); err != nil {
		t.Fatalf("AssertExists(input:required) error = %v", err)
	}
	if err := harness.AssertExists("option:disabled"); err != nil {
		t.Fatalf("AssertExists(option:disabled) error = %v", err)
	}
	if err := harness.AssertExists("option:enabled"); err != nil {
		t.Fatalf("AssertExists(option:enabled) error = %v", err)
	}
	if err := harness.AssertExists("select:disabled"); err != nil {
		t.Fatalf("AssertExists(select:disabled) error = %v", err)
	}
	if err := harness.AssertExists("textarea:read-only"); err != nil {
		t.Fatalf("AssertExists(textarea:read-only) error = %v", err)
	}
	if err := harness.AssertExists("form:valid"); err != nil {
		t.Fatalf("AssertExists(form:valid) error = %v", err)
	}
	if err := harness.AssertExists("#legend-input:disabled"); err == nil {
		t.Fatalf("AssertExists(#legend-input:disabled) error = nil, want no match")
	}
	if err := harness.AssertExists("#disabled-required:required"); err == nil {
		t.Fatalf("AssertExists(#disabled-required:required) error = nil, want no match")
	}
	if err := harness.AssertExists("form:invalid"); err == nil {
		t.Fatalf("AssertExists(form:invalid) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsContentEditablePseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><section id="editable" contenteditable><p id="inherited">Editable</p><div id="false" contenteditable="false"><span id="blocked">Blocked</span></div><div id="plaintext" contenteditable="plaintext-only"><em id="plain-child">Plain</em></div></section><input id="name" type="text"><textarea id="story"></textarea><input id="readonly" type="text" readonly><div id="plain">Plain</div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("section:read-write"); err != nil {
		t.Fatalf("AssertExists(section:read-write) error = %v", err)
	}
	if err := harness.AssertExists("div:read-write"); err != nil {
		t.Fatalf("AssertExists(div:read-write) error = %v", err)
	}
	if err := harness.AssertExists("input:read-write"); err != nil {
		t.Fatalf("AssertExists(input:read-write) error = %v", err)
	}
	if err := harness.AssertExists("textarea:read-write"); err != nil {
		t.Fatalf("AssertExists(textarea:read-write) error = %v", err)
	}
	if err := harness.AssertExists("#blocked:read-only"); err != nil {
		t.Fatalf("AssertExists(#blocked:read-only) error = %v", err)
	}
	if err := harness.AssertExists("#plain-child:read-write"); err != nil {
		t.Fatalf("AssertExists(#plain-child:read-write) error = %v", err)
	}
	if err := harness.AssertExists("#plain:read-only"); err != nil {
		t.Fatalf("AssertExists(#plain:read-only) error = %v", err)
	}
	if err := harness.AssertExists("#blocked:read-write"); err == nil {
		t.Fatalf("AssertExists(#blocked:read-write) error = nil, want no match")
	}
	if err := harness.AssertExists("#plain:read-write"); err == nil {
		t.Fatalf("AssertExists(#plain:read-write) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsModalPseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><dialog id="dialog" modal></dialog><video id="player" fullscreen></video><div id="other" open></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("dialog:modal"); err != nil {
		t.Fatalf("AssertExists(dialog:modal) error = %v", err)
	}
	if err := harness.AssertExists("video:modal"); err != nil {
		t.Fatalf("AssertExists(video:modal) error = %v", err)
	}
	if err := harness.AssertExists("#other:modal"); err == nil {
		t.Fatalf("AssertExists(#other:modal) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsPopoverOpenPseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><div id="menu" popover popover-open></div><div id="closed" popover></div><dialog id="dialog" open></dialog></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("div:popover-open"); err != nil {
		t.Fatalf("AssertExists(div:popover-open) error = %v", err)
	}
	if err := harness.AssertExists("#menu:popover-open"); err != nil {
		t.Fatalf("AssertExists(#menu:popover-open) error = %v", err)
	}
	if err := harness.AssertExists("#closed:popover-open"); err == nil {
		t.Fatalf("AssertExists(#closed:popover-open) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsOpenPseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><select id="dropdown" open><option id="dropdown-option" value="a">A</option></select><select id="listbox" size="2" open><option id="listbox-option" value="b">B</option></select><input id="file" type="file" open><input id="text" type="text" open><div id="other" open></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("select:open"); err != nil {
		t.Fatalf("AssertExists(select:open) error = %v", err)
	}
	if err := harness.AssertExists("#dropdown:open"); err != nil {
		t.Fatalf("AssertExists(#dropdown:open) error = %v", err)
	}
	if err := harness.AssertExists("input:open"); err != nil {
		t.Fatalf("AssertExists(input:open) error = %v", err)
	}
	if err := harness.AssertExists("#listbox:open"); err == nil {
		t.Fatalf("AssertExists(#listbox:open) error = nil, want no match")
	}
	if err := harness.AssertExists("#text:open"); err == nil {
		t.Fatalf("AssertExists(#text:open) error = nil, want no match")
	}
	if err := harness.AssertExists("#other:open"); err == nil {
		t.Fatalf("AssertExists(#other:open) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsDefinedPseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><div id="known"></div><x-widget id="widget" defined></x-widget><x-ghost id="ghost"></x-ghost></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("div:defined"); err != nil {
		t.Fatalf("AssertExists(div:defined) error = %v", err)
	}
	if err := harness.AssertExists("x-widget:defined"); err != nil {
		t.Fatalf("AssertExists(x-widget:defined) error = %v", err)
	}
	if err := harness.AssertExists("#ghost:defined"); err == nil {
		t.Fatalf("AssertExists(#ghost:defined) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsStatePseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><x-widget id="widget"></x-widget><div id="plain" state="checked"></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.SetAttribute("#widget", "state", "checked pressed"); err != nil {
		t.Fatalf("SetAttribute(#widget, state, checked pressed) error = %v", err)
	}

	if err := harness.AssertExists("#widget:state(checked)"); err != nil {
		t.Fatalf("AssertExists(#widget:state(checked)) error = %v", err)
	}
	if err := harness.AssertExists("#widget:state(checked):state(pressed)"); err != nil {
		t.Fatalf("AssertExists(#widget:state(checked):state(pressed)) error = %v", err)
	}
	if err := harness.AssertExists("div:state(checked)"); err == nil {
		t.Fatalf("AssertExists(div:state(checked)) error = nil, want no match")
	}

	if err := harness.RemoveAttribute("#widget", "state"); err != nil {
		t.Fatalf("RemoveAttribute(#widget, state) error = %v", err)
	}
	if err := harness.AssertExists("#widget:state(checked)"); err == nil {
		t.Fatalf("AssertExists(#widget:state(checked)) after RemoveAttribute error = nil, want no match")
	}
}

func TestInteractionSliceSupportsAutofillPseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><input id="name" autofill value="Ada"><input id="other" value="Bob"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("input:autofill"); err != nil {
		t.Fatalf("AssertExists(input:autofill) error = %v", err)
	}
	if err := harness.AssertExists("input:-webkit-autofill"); err != nil {
		t.Fatalf("AssertExists(input:-webkit-autofill) error = %v", err)
	}

	if err := harness.TypeText("#name", "Zed"); err != nil {
		t.Fatalf("TypeText(#name) error = %v", err)
	}
	if err := harness.AssertValue("#name", "Zed"); err != nil {
		t.Fatalf("AssertValue(#name) after TypeText error = %v", err)
	}
	if err := harness.AssertExists("#name:autofill"); err == nil {
		t.Fatalf("AssertExists(#name:autofill) after TypeText error = nil, want no match")
	}
}

func TestInteractionSliceSupportsActiveHoverPseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><div id="wrap"><button id="btn" active>Go</button><span id="hovered" hover>Hover</span></div><label id="active-label" for="active-field" active>Field</label><input id="active-field" type="text"><label id="hover-label" hover><input id="hover-field" type="text"></label><label id="secret-label" for="secret" active>Secret</label><input id="secret" type="hidden"><p id="plain">Text</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("button:active"); err != nil {
		t.Fatalf("AssertExists(button:active) error = %v", err)
	}
	if err := harness.AssertExists("div:active"); err != nil {
		t.Fatalf("AssertExists(div:active) error = %v", err)
	}
	if err := harness.AssertExists("span:hover"); err != nil {
		t.Fatalf("AssertExists(span:hover) error = %v", err)
	}
	if err := harness.AssertExists("div:hover"); err != nil {
		t.Fatalf("AssertExists(div:hover) error = %v", err)
	}
	if err := harness.AssertExists("input:active"); err != nil {
		t.Fatalf("AssertExists(input:active) error = %v", err)
	}
	if err := harness.AssertExists("input:hover"); err != nil {
		t.Fatalf("AssertExists(input:hover) error = %v", err)
	}
	if err := harness.AssertExists("#active-field:active"); err != nil {
		t.Fatalf("AssertExists(#active-field:active) error = %v", err)
	}
	if err := harness.AssertExists("#hover-field:hover"); err != nil {
		t.Fatalf("AssertExists(#hover-field:hover) error = %v", err)
	}
	if err := harness.AssertExists("#secret:active"); err == nil {
		t.Fatalf("AssertExists(#secret:active) error = nil, want no match")
	}
	if err := harness.AssertExists("#plain:active"); err == nil {
		t.Fatalf("AssertExists(#plain:active) error = nil, want no match")
	}
}

func TestInteractionSlicePreservesDefaultPseudoClassAcrossControlUpdates(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><form id="profile"><input id="flag" type="checkbox" checked><button id="submit-1" type="submit">Save</button><button id="submit-2" type="submit">Extra</button><select id="mode"><option id="opt-a" value="a" selected>A</option><option id="opt-b" value="b">B</option></select></form></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("input:default"); err != nil {
		t.Fatalf("AssertExists(input:default) before updates error = %v", err)
	}
	if err := harness.AssertExists("option:default"); err != nil {
		t.Fatalf("AssertExists(option:default) before updates error = %v", err)
	}

	if err := harness.SetChecked("#flag", false); err != nil {
		t.Fatalf("SetChecked(#flag) error = %v", err)
	}
	if err := harness.SetSelectValue("#mode", "b"); err != nil {
		t.Fatalf("SetSelectValue(#mode, b) error = %v", err)
	}

	if err := harness.AssertExists("input:default"); err != nil {
		t.Fatalf("AssertExists(input:default) after updates error = %v", err)
	}
	if err := harness.AssertExists("option:default"); err != nil {
		t.Fatalf("AssertExists(option:default) after updates error = %v", err)
	}
	if err := harness.AssertExists("#opt-b:default"); err == nil {
		t.Fatalf("AssertExists(#opt-b:default) after updates error = nil, want no match")
	}
}

func TestInteractionSliceSupportsHeadingLevelPseudoClass(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><h1 id="title">Title</h1><section><h2 id="sub">Sub</h2><div><h4 id="deep">Deep</h4></div></section><article><h6 id="final">Final</h6></article><p id="plain">Body</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists(":heading(1)"); err != nil {
		t.Fatalf("AssertExists(:heading(1)) error = %v", err)
	}
	if err := harness.AssertExists(":heading(2, 4)"); err != nil {
		t.Fatalf("AssertExists(:heading(2, 4)) error = %v", err)
	}
	if err := harness.AssertExists("h4:heading(4)"); err != nil {
		t.Fatalf("AssertExists(h4:heading(4)) error = %v", err)
	}
	if err := harness.AssertExists("h6:heading(6)"); err != nil {
		t.Fatalf("AssertExists(h6:heading(6)) error = %v", err)
	}
}

func TestInteractionSliceSupportsMediaPseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><audio id="song" src="song.mp3"></audio><video id="film"></video><video id="paused" paused></video><video id="seeking" seeking></video><video id="muted" muted></video><video id="buffering" networkstate="loading" readystate="2"></video><video id="stalled" networkstate="loading" readystate="1" stalled volume-locked></video><div id="other" paused muted></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("audio:playing"); err != nil {
		t.Fatalf("AssertExists(audio:playing) error = %v", err)
	}
	if err := harness.AssertExists("video:paused"); err != nil {
		t.Fatalf("AssertExists(video:paused) error = %v", err)
	}
	if err := harness.AssertExists("video:seeking"); err != nil {
		t.Fatalf("AssertExists(video:seeking) error = %v", err)
	}
	if err := harness.AssertExists("video:muted"); err != nil {
		t.Fatalf("AssertExists(video:muted) error = %v", err)
	}
	if err := harness.AssertExists("video:buffering"); err != nil {
		t.Fatalf("AssertExists(video:buffering) error = %v", err)
	}
	if err := harness.AssertExists("video:stalled"); err != nil {
		t.Fatalf("AssertExists(video:stalled) error = %v", err)
	}
	if err := harness.AssertExists("video:volume-locked"); err != nil {
		t.Fatalf("AssertExists(video:volume-locked) error = %v", err)
	}
	if err := harness.AssertExists("#other:paused"); err == nil {
		t.Fatalf("AssertExists(#other:paused) error = nil, want no match")
	}
}

func TestInteractionSliceSupportsOfTypePseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><section id="single"><em id="only-child">one</em></section><div id="mixed"><p id="para-a">A</p><span id="only-of-type">S</span><p id="para-b">B</p></div><details id="details" open><summary id="summary-a">A</summary><div id="middle">M</div><summary id="summary-b">B</summary></details></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("em:only-child"); err != nil {
		t.Fatalf("AssertExists(em:only-child) error = %v", err)
	}
	if err := harness.AssertExists("em:only-of-type"); err != nil {
		t.Fatalf("AssertExists(em:only-of-type) error = %v", err)
	}
	if err := harness.AssertExists("span:only-of-type"); err != nil {
		t.Fatalf("AssertExists(span:only-of-type) error = %v", err)
	}
	if err := harness.AssertExists("summary:first-of-type"); err != nil {
		t.Fatalf("AssertExists(summary:first-of-type) error = %v", err)
	}
	if err := harness.AssertExists("summary:last-of-type"); err != nil {
		t.Fatalf("AssertExists(summary:last-of-type) error = %v", err)
	}
}

func TestInteractionSliceSupportsConstraintValidationPseudoClasses(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><form id="valid-form"><input id="name" type="text" required value="Ada"><input id="age" type="number" min="1" max="10" value="5"><select id="mode"><option value="a" selected>A</option><option value="b">B</option></select></form><form id="invalid-form"><input id="missing" type="text" required><input id="low" type="number" min="1" max="10" value="0"><input id="high" type="number" min="1" max="10" value="11"></form></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("input:valid"); err != nil {
		t.Fatalf("AssertExists(input:valid) error = %v", err)
	}
	if err := harness.AssertExists("input:invalid"); err != nil {
		t.Fatalf("AssertExists(input:invalid) error = %v", err)
	}
	if err := harness.AssertExists("input:in-range"); err != nil {
		t.Fatalf("AssertExists(input:in-range) error = %v", err)
	}
	if err := harness.AssertExists("input:out-of-range"); err != nil {
		t.Fatalf("AssertExists(input:out-of-range) error = %v", err)
	}
	if err := harness.AssertExists("select:valid"); err != nil {
		t.Fatalf("AssertExists(select:valid) error = %v", err)
	}
	if err := harness.AssertExists("form:valid"); err != nil {
		t.Fatalf("AssertExists(form:valid) error = %v", err)
	}
	if err := harness.AssertExists("form:invalid"); err != nil {
		t.Fatalf("AssertExists(form:invalid) error = %v", err)
	}
}

func TestAttributeReflectionContracts(t *testing.T) {
	harness, err := FromHTML(`<main><div id="root" data-x="1" class="alpha beta"></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, ok, err := harness.GetAttribute(" #root ", "DATA-X"); err != nil || !ok || got != "1" {
		t.Fatalf("GetAttribute(DATA-X) = (%q, %v, %v), want (\"1\", true, nil)", got, ok, err)
	}
	if ok, err := harness.HasAttribute("#root", "data-x"); err != nil || !ok {
		t.Fatalf("HasAttribute(data-x) = (%v, %v), want (true, nil)", ok, err)
	}

	if err := harness.SetAttribute("#root", "data-x", "2"); err != nil {
		t.Fatalf("SetAttribute(data-x) error = %v", err)
	}
	if got, ok, err := harness.GetAttribute("#root", "data-x"); err != nil || !ok || got != "2" {
		t.Fatalf("GetAttribute(data-x) after SetAttribute = (%q, %v, %v), want (\"2\", true, nil)", got, ok, err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="root" data-x="2" class="alpha beta"></div></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after SetAttribute = %q, want %q", got, want)
	}

	if err := harness.RemoveAttribute("#root", "data-x"); err != nil {
		t.Fatalf("RemoveAttribute(data-x) error = %v", err)
	}
	if got, ok, err := harness.GetAttribute("#root", "data-x"); err != nil || ok || got != "" {
		t.Fatalf("GetAttribute(data-x) after RemoveAttribute = (%q, %v, %v), want (\"\", false, nil)", got, ok, err)
	}
	if ok, err := harness.HasAttribute("#root", "data-x"); err != nil || ok {
		t.Fatalf("HasAttribute(data-x) after RemoveAttribute = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestClassListAndDatasetContracts(t *testing.T) {
	harness, err := FromHTML(`<main><div id="root" class="alpha beta" data-foo-bar="1"></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	classList, err := harness.ClassList("#root")
	if err != nil {
		t.Fatalf("ClassList(#root) error = %v", err)
	}
	if got := classList.Values(); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("ClassList.Values() = %#v, want [alpha beta]", got)
	}
	if !classList.Contains("beta") {
		t.Fatalf("ClassList.Contains(beta) = false, want true")
	}

	classes := classList.Values()
	classes[0] = "mutated"
	if got := classList.Values(); got[0] != "alpha" {
		t.Fatalf("ClassList.Values() should return copies, got %#v", got)
	}

	if err := classList.Add("gamma"); err != nil {
		t.Fatalf("ClassList.Add(gamma) error = %v", err)
	}
	if err := classList.Remove("alpha"); err != nil {
		t.Fatalf("ClassList.Remove(alpha) error = %v", err)
	}

	dataset, err := harness.Dataset("#root")
	if err != nil {
		t.Fatalf("Dataset(#root) error = %v", err)
	}
	if got := dataset.Values(); got["fooBar"] != "1" || len(got) != 1 {
		t.Fatalf("Dataset.Values() = %#v, want fooBar=1", got)
	}
	if got, ok := dataset.Get("fooBar"); !ok || got != "1" {
		t.Fatalf("Dataset.Get(fooBar) = (%q, %v), want (\"1\", true)", got, ok)
	}

	values := dataset.Values()
	values["fooBar"] = "mutated"
	if got, ok := dataset.Get("fooBar"); !ok || got != "1" {
		t.Fatalf("Dataset.Get(fooBar) after Values mutation = (%q, %v), want (\"1\", true)", got, ok)
	}

	if err := dataset.Set("shipId", "92432"); err != nil {
		t.Fatalf("Dataset.Set(shipId) error = %v", err)
	}
	if err := dataset.Remove("fooBar"); err != nil {
		t.Fatalf("Dataset.Remove(fooBar) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="root" class="beta gamma" data-ship-id="92432"></div></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after class/dataset view mutation = %q, want %q", got, want)
	}
}

func TestClassListAndDatasetContractsRejectInvalidInputs(t *testing.T) {
	harness, err := FromHTML(`<main><div id="root" class="alpha" data-foo="1"></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.ClassList("main[item="); err == nil {
		t.Fatalf("ClassList(main[item=) error = nil, want selector error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("ClassList(main[item=) error = %#v, want DOM error", err)
	}
	if _, err := harness.Dataset("#missing"); err == nil {
		t.Fatalf("Dataset(#missing) error = nil, want missing-element error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("Dataset(#missing) error = %#v, want DOM error", err)
	}

	classList, err := harness.ClassList("#root")
	if err != nil {
		t.Fatalf("ClassList(#root) error = %v", err)
	}
	if err := classList.Add(" "); err == nil {
		t.Fatalf("ClassList.Add(empty) error = nil, want validation error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("ClassList.Add(empty) error = %#v, want DOM error", err)
	}

	dataset, err := harness.Dataset("#root")
	if err != nil {
		t.Fatalf("Dataset(#root) error = %v", err)
	}
	if err := dataset.Set("foo-bar", "x"); err == nil {
		t.Fatalf("Dataset.Set(foo-bar) error = nil, want validation error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("Dataset.Set(foo-bar) error = %#v, want DOM error", err)
	}

	var nilHarness *Harness
	if _, err := nilHarness.ClassList("#root"); err == nil {
		t.Fatalf("nil Harness.ClassList() error = nil, want DOM error")
	}
	if _, err := nilHarness.Dataset("#root"); err == nil {
		t.Fatalf("nil Harness.Dataset() error = nil, want DOM error")
	}

	var emptyClassList ClassListView
	if got := emptyClassList.Values(); len(got) != 0 {
		t.Fatalf("zero ClassListView.Values() = %#v, want empty", got)
	}
	if emptyClassList.Contains("alpha") {
		t.Fatalf("zero ClassListView.Contains(alpha) = true, want false")
	}
	if err := emptyClassList.Add("alpha"); err == nil {
		t.Fatalf("zero ClassListView.Add() error = nil, want DOM error")
	}

	var emptyDataset DatasetView
	if got := emptyDataset.Values(); len(got) != 0 {
		t.Fatalf("zero DatasetView.Values() = %#v, want empty", got)
	}
	if got, ok := emptyDataset.Get("fooBar"); ok || got != "" {
		t.Fatalf("zero DatasetView.Get(fooBar) = (%q, %v), want (\"\", false)", got, ok)
	}
	if err := emptyDataset.Set("shipId", "1"); err == nil {
		t.Fatalf("zero DatasetView.Set() error = nil, want DOM error")
	}
}

func TestAttributeReflectionContractsRejectInvalidInputs(t *testing.T) {
	harness, err := FromHTML(`<main><div id="root"></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, _, err := harness.GetAttribute("#missing", "id"); err == nil {
		t.Fatalf("GetAttribute(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("GetAttribute(#missing) error = %#v, want DOM error", err)
	}

	if err := harness.SetAttribute("#root", " ", "x"); err == nil {
		t.Fatalf("SetAttribute(empty name) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetAttribute(empty name) error = %#v, want DOM error", err)
	}

	if err := harness.RemoveAttribute("#root", " "); err == nil {
		t.Fatalf("RemoveAttribute(empty name) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("RemoveAttribute(empty name) error = %#v, want DOM error", err)
	}
}

func TestFormControlActionsUpdateDebugDom(t *testing.T) {
	harness, err := FromHTML(`<main><input id="name"><input id="flag" type="checkbox"><textarea id="bio">Base</textarea><select id="mode"><option value="a" selected>A</option><option>B</option><option value="c">C</option></select><form id="profile"><button id="submit" type="submit">Save</button></form></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.TypeText("#name", "Ada"); err != nil {
		t.Fatalf("TypeText(#name) error = %v", err)
	}
	if err := harness.SetChecked("#flag", true); err != nil {
		t.Fatalf("SetChecked(#flag) error = %v", err)
	}
	if err := harness.SetSelectValue("#mode", "B"); err != nil {
		t.Fatalf("SetSelectValue(#mode) error = %v", err)
	}
	if err := harness.Submit("#profile"); err != nil {
		t.Fatalf("Submit(#profile) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><input id="name" value="Ada"><input id="flag" type="checkbox" checked><textarea id="bio">Base</textarea><select id="mode"><option value="a">A</option><option selected>B</option><option value="c">C</option></select><form id="profile"><button id="submit" type="submit">Save</button></form></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), `<main><input id="name"><input id="flag" type="checkbox"><textarea id="bio">Base</textarea><select id="mode"><option value="a" selected>A</option><option>B</option><option value="c">C</option></select><form id="profile"><button id="submit" type="submit">Save</button></form></main>`; got != want {
		t.Fatalf("HTML() = %q, want original source snapshot %q", got, want)
	}

	log := harness.Debug().Interactions()
	if len(log) != 4 {
		t.Fatalf("Debug().Interactions() len = %d, want 4", len(log))
	}
	if log[0].Kind != InteractionKindTypeText || log[0].Selector != "#name" {
		t.Fatalf("Debug().Interactions()[0] = %#v, want type_text #name", log[0])
	}
	if log[1].Kind != InteractionKindSetChecked || log[1].Selector != "#flag" {
		t.Fatalf("Debug().Interactions()[1] = %#v, want set_checked #flag", log[1])
	}
	if log[2].Kind != InteractionKindSetSelectValue || log[2].Selector != "#mode" {
		t.Fatalf("Debug().Interactions()[2] = %#v, want set_select_value #mode", log[2])
	}
	if log[3].Kind != InteractionKindSubmit || log[3].Selector != "#profile" {
		t.Fatalf("Debug().Interactions()[3] = %#v, want submit #profile", log[3])
	}
}

func TestFormControlValueAssignmentUpdatesDebugDom(t *testing.T) {
	harness, err := FromHTML(`<main><select id="mode"><option value="a" selected>A</option><option value="b">B</option></select><div id="out"></div><script>const select = document.querySelector("#mode"); select.value = "b"; host:setTextContent("#out", expr(select.value))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "b" {
		t.Fatalf("TextContent(#out) = %q, want b", got)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><select id="mode"><option value="a">A</option><option value="b" selected>B</option></select><div id="out">b</div><script>const select = document.querySelector("#mode"); select.value = "b"; host:setTextContent("#out", expr(select.value))</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() = %q, want %q", got, want)
	}
}

func TestClickAppliesDefaultActions(t *testing.T) {
	harness, err := FromHTML(`<form id="profile"><input id="agree" type="checkbox"><button id="submit" type="submit">Save</button></form>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#agree"); err != nil {
		t.Fatalf("Click(#agree) error = %v", err)
	}
	if err := harness.Click("#submit"); err != nil {
		t.Fatalf("Click(#submit) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<form id="profile"><input id="agree" type="checkbox" checked><button id="submit" type="submit">Save</button></form>`; got != want {
		t.Fatalf("Debug().DumpDOM() = %q, want %q", got, want)
	}

	log := harness.Debug().Interactions()
	if len(log) != 3 {
		t.Fatalf("Debug().Interactions() len = %d, want 3", len(log))
	}
	if log[0].Kind != InteractionKindClick || log[0].Selector != "#agree" {
		t.Fatalf("Debug().Interactions()[0] = %#v, want click #agree", log[0])
	}
	if log[1].Kind != InteractionKindClick || log[1].Selector != "#submit" {
		t.Fatalf("Debug().Interactions()[1] = %#v, want click #submit", log[1])
	}
	if log[2].Kind != InteractionKindSubmit || log[2].Selector != "#submit" {
		t.Fatalf("Debug().Interactions()[2] = %#v, want submit #submit", log[2])
	}
}

func TestSetFilesMarksFileInputAsUserValid(t *testing.T) {
	harness, err := FromHTML(`<main><input id="upload" type="file"></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.SetFiles("#upload", []string{"report.csv"}); err != nil {
		t.Fatalf("SetFiles(#upload) error = %v", err)
	}
	if err := harness.AssertValue("#upload", "report.csv"); err != nil {
		t.Fatalf("AssertValue(#upload) error = %v", err)
	}
	if err := harness.AssertExists("#upload:user-valid"); err != nil {
		t.Fatalf("AssertExists(#upload:user-valid) error = %v", err)
	}
}

func TestClickAppliesHyperlinkDefaultActions(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/base/",
		`<main><a id="nav" href="/next">Go</a><map name="hot"><area id="popup" href="https://example.test/popup" target="_blank" alt="Open"></map><a id="download" href="https://example.test/files/report.csv" download="report.csv">Download</a></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.Click("#nav"); err != nil {
		t.Fatalf("Click(#nav) error = %v", err)
	}
	if got, want := harness.URL(), "https://example.test/next"; got != want {
		t.Fatalf("URL() after anchor click = %q, want %q", got, want)
	}
	if got := harness.Mocks().Location().Navigations(); len(got) != 1 || got[0] != "https://example.test/next" {
		t.Fatalf("Location().Navigations() = %#v, want one navigation to https://example.test/next", got)
	}

	if err := harness.Click("#popup"); err != nil {
		t.Fatalf("Click(#popup) error = %v", err)
	}
	if got, want := harness.URL(), "https://example.test/next"; got != want {
		t.Fatalf("URL() after target=_blank click = %q, want %q", got, want)
	}
	if got := harness.Mocks().Open().Calls(); len(got) != 1 || got[0].URL != "https://example.test/popup" {
		t.Fatalf("Open().Calls() = %#v, want one open call to popup", got)
	}

	if err := harness.Click("#download"); err != nil {
		t.Fatalf("Click(#download) error = %v", err)
	}
	if got, want := harness.URL(), "https://example.test/next"; got != want {
		t.Fatalf("URL() after download click = %q, want %q", got, want)
	}
	downloads := harness.Mocks().Downloads().Artifacts()
	if len(downloads) != 1 || downloads[0].FileName != "report.csv" || string(downloads[0].Bytes) != "https://example.test/files/report.csv" {
		t.Fatalf("Downloads().Artifacts() = %#v, want one captured download", downloads)
	}
}

func TestClickAppliesResetDefaultAction(t *testing.T) {
	harness, err := FromHTML(`<form id="profile"><input id="name"><input id="flag" type="checkbox"><input id="radio-a" type="radio" name="size" checked><input id="radio-b" type="radio" name="size"><textarea id="bio">Base</textarea><select id="mode"><option value="a" selected>A</option><option>B</option><option value="c">C</option></select><button id="reset" type="reset">Reset</button></form>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.TypeText("#name", "Ada"); err != nil {
		t.Fatalf("TypeText(#name) error = %v", err)
	}
	if err := harness.SetChecked("#flag", true); err != nil {
		t.Fatalf("SetChecked(#flag) error = %v", err)
	}
	if err := harness.SetChecked("#radio-b", true); err != nil {
		t.Fatalf("SetChecked(#radio-b) error = %v", err)
	}
	if err := harness.TypeText("#bio", "Line 1\nLine 2"); err != nil {
		t.Fatalf("TypeText(#bio) error = %v", err)
	}
	if err := harness.SetSelectValue("#mode", "B"); err != nil {
		t.Fatalf("SetSelectValue(#mode) error = %v", err)
	}

	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<form id="profile"><input id="name"><input id="flag" type="checkbox"><input id="radio-a" type="radio" name="size" checked><input id="radio-b" type="radio" name="size"><textarea id="bio">Base</textarea><select id="mode"><option value="a" selected>A</option><option>B</option><option value="c">C</option></select><button id="reset" type="reset">Reset</button></form>`; got != want {
		t.Fatalf("Debug().DumpDOM() after reset click = %q, want %q", got, want)
	}

	log := harness.Debug().Interactions()
	if len(log) != 6 {
		t.Fatalf("Debug().Interactions() len = %d, want 6", len(log))
	}
	if log[5].Kind != InteractionKindClick || log[5].Selector != "#reset" {
		t.Fatalf("Debug().Interactions()[5] = %#v, want click #reset", log[5])
	}
}

func TestWriteHTMLContract(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">old</button><div id="out">before</div><script>host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", "old-listener")')</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Focus("#btn"); err != nil {
		t.Fatalf("Focus(#btn) error = %v", err)
	}
	if err := harness.ScrollTo(7, 9); err != nil {
		t.Fatalf("ScrollTo(7, 9) error = %v", err)
	}

	markup := `<main><button id="btn">new</button><div id="out">fresh</div><script>host:setInnerHTML("#out", "written")</script></main>`
	if err := harness.WriteHTML(markup); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">new</button><div id="out">written</div><script>host:setInnerHTML("#out", "written")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after WriteHTML = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), markup; got != want {
		t.Fatalf("HTML() after WriteHTML = %q, want %q", got, want)
	}
	if got := harness.Debug().FocusedSelector(); got != "" {
		t.Fatalf("Debug().FocusedSelector() after WriteHTML = %q, want empty", got)
	}
	if gotX, gotY := harness.Debug().ScrollPosition(); gotX != 0 || gotY != 0 {
		t.Fatalf("Debug().ScrollPosition() after WriteHTML = (%d, %d), want (0, 0)", gotX, gotY)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) after WriteHTML error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">new</button><div id="out">written</div><script>host:setInnerHTML("#out", "written")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after Click on rewritten document = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanWriteHTMLThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host:writeHTML('<main><div id="out">new</div></main>'); host:setInnerHTML("#out", "after")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">after</div></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after host writeHTML = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), `<main><div id="out">new</div></main>`; got != want {
		t.Fatalf("HTML() after host writeHTML = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanDriveLocationMockThroughPublicActions(t *testing.T) {
	markup := `<main><div id="out"></div><script>host:locationAssign("/assign"); host:locationReplace("replace"); host:locationReload()</script></main>`
	harness, err := FromHTMLWithURL("https://example.test/start", markup)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), markup; got != want {
		t.Fatalf("Debug().DumpDOM() after location host bridge = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://example.test/replace"; got != want {
		t.Fatalf("URL() after location host bridge = %q, want %q", got, want)
	}
	if got, want := harness.Mocks().Location().CurrentURL(), "https://example.test/replace"; got != want {
		t.Fatalf("Mocks().Location().CurrentURL() = %q, want %q", got, want)
	}
	if got, want := harness.Mocks().Location().Navigations(), []string{
		"https://example.test/assign",
		"https://example.test/replace",
		"https://example.test/replace",
	}; len(got) != len(want) {
		t.Fatalf("Mocks().Location().Navigations() = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Mocks().Location().Navigations()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	}
}

func TestInlineScriptsCanSetLocationPropertiesThroughPublicActions(t *testing.T) {
	markup := `<main><div id="out"></div><script>host:locationSet("hash", "#step1"); host:locationSet("pathname", "next"); host:locationSet("search", "?mode=full")</script></main>`
	harness, err := FromHTMLWithURL("https://example.test/start?old=1", markup)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), markup; got != want {
		t.Fatalf("Debug().DumpDOM() after locationSet host bridge = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://example.test/next?mode=full#step1"; got != want {
		t.Fatalf("URL() after locationSet host bridge = %q, want %q", got, want)
	}
	if got, want := harness.Mocks().Location().Navigations(), []string{
		"https://example.test/start?old=1#step1",
		"https://example.test/next?old=1#step1",
		"https://example.test/next?mode=full#step1",
	}; len(got) != len(want) {
		t.Fatalf("Mocks().Location().Navigations() = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Mocks().Location().Navigations()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	}
}

func TestInlineScriptsCanSetLocationUsernameAndPasswordThroughPublicActions(t *testing.T) {
	markup := `<main><script>host:locationSet("username", "alice"); host:locationSet("password", "secret")</script></main>`
	harness, err := FromHTMLWithURL("https://example.test/start", markup)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), markup; got != want {
		t.Fatalf("Debug().DumpDOM() after location credential updates = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://alice:secret@example.test/start"; got != want {
		t.Fatalf("URL() after location credential updates = %q, want %q", got, want)
	}
	if got, want := harness.Mocks().Location().Navigations(), []string{
		"https://alice@example.test/start",
		"https://alice:secret@example.test/start",
	}; len(got) != len(want) {
		t.Fatalf("Mocks().Location().Navigations() = %#v, want %#v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Mocks().Location().Navigations()[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	}
}

func TestInlineScriptsCanReadLocationPropertiesThroughPublicActions(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{
			name:   "href",
			script: `host:setTextContent("#out", expr(host:locationHref()))`,
			want:   "https://example.test:8443/path/name?mode=full#step-1",
		},
		{
			name:   "origin",
			script: `host:setTextContent("#out", expr(host:locationOrigin()))`,
			want:   "https://example.test:8443",
		},
		{
			name:   "protocol",
			script: `host:setTextContent("#out", expr(host:locationProtocol()))`,
			want:   "https:",
		},
		{
			name:   "host",
			script: `host:setTextContent("#out", expr(host:locationHost()))`,
			want:   "example.test:8443",
		},
		{
			name:   "hostname",
			script: `host:setTextContent("#out", expr(host:locationHostname()))`,
			want:   "example.test",
		},
		{
			name:   "port",
			script: `host:setTextContent("#out", expr(host:locationPort()))`,
			want:   "8443",
		},
		{
			name:   "pathname",
			script: `host:setTextContent("#out", expr(host:locationPathname()))`,
			want:   "/path/name",
		},
		{
			name:   "search",
			script: `host:setTextContent("#out", expr(host:locationSearch()))`,
			want:   "?mode=full",
		},
		{
			name:   "hash",
			script: `host:setTextContent("#out", expr(host:locationHash()))`,
			want:   "#step-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			markup := `<main><div id="out"></div><script>` + tc.script + `</script></main>`
			harness, err := FromHTMLWithURL("https://example.test:8443/path/name?mode=full#step-1", markup)
			if err != nil {
				t.Fatalf("FromHTMLWithURL() error = %v", err)
			}

			if err := harness.AssertText("#out", tc.want); err != nil {
				t.Fatalf("AssertText(#out, %q) error = %v", tc.want, err)
			}
			if got, want := harness.URL(), "https://example.test:8443/path/name?mode=full#step-1"; got != want {
				t.Fatalf("URL() after %s getter = %q, want %q", tc.name, got, want)
			}
		})
	}
}

func TestInlineScriptsRejectLocationGetterArgumentsThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.WriteHTML(`<main><script>host:locationHref("extra")</script></main>`); err == nil {
		t.Fatalf("WriteHTML(locationHref extra) error = nil, want validation failure")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("WriteHTML(locationHref extra) error = %#v, want DOM error", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">old</div></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after rejected locationHref getter = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanDriveWindowNameThroughPublicActions(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/start",
		`<main><div id="out">old</div><script>host:setWindowName("alpha"); host:locationAssign("/next"); host:setInnerHTML("#out", "done")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">done</div><script>host:setWindowName("alpha"); host:locationAssign("/next"); host:setInnerHTML("#out", "done")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after window name host bridge = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://example.test/next"; got != want {
		t.Fatalf("URL() after window name host bridge = %q, want %q", got, want)
	}
	if got, want := harness.Debug().WindowName(), "alpha"; got != want {
		t.Fatalf("Debug().WindowName() after window name host bridge = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanBootstrapRawHtmlWithBrowserGlobals(t *testing.T) {
	const initialURL = "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=initial"
	const rawHTML = `<main><div id="agri-unit-converter-root">root</div><div id="result"></div><div id="formatted"></div><div id="href"></div><script>const root = document.getElementById("agri-unit-converter-root"); const current = new URL(window.location.href); const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23); window.location.search.length; sessionStorage.setItem("mode", navigator.onLine && "search"); window.history.replaceState({}, "", "?mode=raw#ready"); localStorage.setItem("format", formatted); matchMedia("(prefers-reduced-motion: reduce)"); clipboard.writeText(root.textContent); setTimeout("noop", 5); queueMicrotask("noop"); host:setTextContent("#result", expr(root.textContent)); host:setTextContent("#formatted", expr(formatted)); host:setTextContent("#href", expr(current.href))</script></main>`

	harness, err := NewHarnessBuilder().
		URL(initialURL).
		MatchMedia(map[string]bool{"(prefers-reduced-motion: reduce)": true}).
		HTML(rawHTML).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got, err := harness.TextContent("#result"); err != nil {
		t.Fatalf("TextContent(#result) error = %v", err)
	} else if got != "root" {
		t.Fatalf("TextContent(#result) = %q, want root", got)
	}
	if got, err := harness.TextContent("#formatted"); err != nil {
		t.Fatalf("TextContent(#formatted) error = %v", err)
	} else if got != "1.23" {
		t.Fatalf("TextContent(#formatted) = %q, want 1.23", got)
	}
	if got, err := harness.TextContent("#href"); err != nil {
		t.Fatalf("TextContent(#href) error = %v", err)
	} else if got != initialURL {
		t.Fatalf("TextContent(#href) = %q, want %q", got, initialURL)
	}
	if got, want := harness.URL(), "https://finitefield.org/en/tools/agri/agri-unit-converter/?mode=raw#ready"; got != want {
		t.Fatalf("URL() after raw browser-global bootstrap = %q, want %q", got, want)
	}
	if !harness.Debug().DOMReady() {
		t.Fatalf("Debug().DOMReady() = false, want true after raw browser-global bootstrap")
	}
	if got := harness.Debug().DOMError(); got != "" {
		t.Fatalf("Debug().DOMError() = %q, want empty after raw browser-global bootstrap", got)
	}
	if got, ok := harness.Debug().HistoryState(); !ok || got != "[object Object]" {
		t.Fatalf("Debug().HistoryState() = (%q, %v), want ([object Object], true)", got, ok)
	}
	if got := harness.Debug().LocalStorage()["format"]; got != "1.23" {
		t.Fatalf("Debug().LocalStorage()[format] = %q, want 1.23", got)
	}
	if got := harness.Debug().SessionStorage()["mode"]; got != "search" {
		t.Fatalf("Debug().SessionStorage()[mode] = %q, want search", got)
	}
	if got := harness.Debug().MatchMediaCalls(); len(got) != 1 || got[0].Query != "(prefers-reduced-motion: reduce)" {
		t.Fatalf("Debug().MatchMediaCalls() = %#v, want one prefers-reduced-motion query", got)
	}
	if got := harness.Debug().ClipboardWrites(); len(got) != 1 || got[0] != "root" {
		t.Fatalf("Debug().ClipboardWrites() = %#v, want one root write", got)
	}
	if got := harness.Debug().PendingTimers(); len(got) != 1 || got[0].Source != "noop" {
		t.Fatalf("Debug().PendingTimers() = %#v, want one noop timer", got)
	}
	if got := harness.Debug().PendingMicrotasks(); len(got) != 0 {
		t.Fatalf("Debug().PendingMicrotasks() = %#v, want empty after bootstrap drain", got)
	}
}

func TestInlineScriptsSkipNonClassicScriptTypes(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script type="application/ld+json">{"@context":"https://schema.org","@type":"WebPage"}</script><script>host.setTextContent("#out", "ok")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "ok" {
		t.Fatalf("TextContent(#out) = %q, want ok", got)
	}
	if got := harness.Debug().DOMError(); got != "" {
		t.Fatalf("Debug().DOMError() = %q, want empty after non-classic script skip", got)
	}
}

func TestInlineScriptsCanReadDocumentPropertiesThroughPublicActions(t *testing.T) {
	const initialURL = "https://example.test/app?mode=preview#section"
	const rawHTML = `<html dir="rtl"><head><title>Doc Title</title></head><body><main id="root"><div id="title"></div><div id="ready"></div><div id="view"></div><div id="base"></div><div id="url"></div><div id="uri"></div><div id="compat"></div><div id="content"></div><div id="design"></div><div id="dir"></div><div id="doctype"></div><script>host:setTextContent("#title", expr(document.title)); host:setTextContent("#ready", expr(document.readyState)); host:setTextContent("#view", expr(document.defaultView.location.href)); host:setTextContent("#base", expr(document.baseURI)); host:setTextContent("#url", expr(document.URL)); host:setTextContent("#uri", expr(document.documentURI)); host:setTextContent("#compat", expr(document.compatMode)); host:setTextContent("#content", expr(document.contentType)); host:setTextContent("#design", expr(document.designMode)); host:setTextContent("#dir", expr(document.dir)); host:setTextContent("#doctype", expr(document.doctype === null ? "true" : "false"))</script></main></body></html>`

	harness, err := FromHTMLWithURL(initialURL, rawHTML)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got, err := harness.TextContent("#title"); err != nil {
		t.Fatalf("TextContent(#title) error = %v", err)
	} else if got != "Doc Title" {
		t.Fatalf("TextContent(#title) = %q, want Doc Title", got)
	}
	if got, err := harness.TextContent("#ready"); err != nil {
		t.Fatalf("TextContent(#ready) error = %v", err)
	} else if got != "loading" {
		t.Fatalf("TextContent(#ready) = %q, want loading", got)
	}
	if got, err := harness.TextContent("#view"); err != nil {
		t.Fatalf("TextContent(#view) error = %v", err)
	} else if got != initialURL {
		t.Fatalf("TextContent(#view) = %q, want %q", got, initialURL)
	}
	if got, err := harness.TextContent("#base"); err != nil {
		t.Fatalf("TextContent(#base) error = %v", err)
	} else if got != initialURL {
		t.Fatalf("TextContent(#base) = %q, want %q", got, initialURL)
	}
	if got, err := harness.TextContent("#url"); err != nil {
		t.Fatalf("TextContent(#url) error = %v", err)
	} else if got != initialURL {
		t.Fatalf("TextContent(#url) = %q, want %q", got, initialURL)
	}
	if got, err := harness.TextContent("#uri"); err != nil {
		t.Fatalf("TextContent(#uri) error = %v", err)
	} else if got != initialURL {
		t.Fatalf("TextContent(#uri) = %q, want %q", got, initialURL)
	}
	if got, err := harness.TextContent("#compat"); err != nil {
		t.Fatalf("TextContent(#compat) error = %v", err)
	} else if got != "CSS1Compat" {
		t.Fatalf("TextContent(#compat) = %q, want CSS1Compat", got)
	}
	if got, err := harness.TextContent("#content"); err != nil {
		t.Fatalf("TextContent(#content) error = %v", err)
	} else if got != "text/html" {
		t.Fatalf("TextContent(#content) = %q, want text/html", got)
	}
	if got, err := harness.TextContent("#design"); err != nil {
		t.Fatalf("TextContent(#design) error = %v", err)
	} else if got != "off" {
		t.Fatalf("TextContent(#design) = %q, want off", got)
	}
	if got, err := harness.TextContent("#dir"); err != nil {
		t.Fatalf("TextContent(#dir) error = %v", err)
	} else if got != "rtl" {
		t.Fatalf("TextContent(#dir) = %q, want rtl", got)
	}
	if got, err := harness.TextContent("#doctype"); err != nil {
		t.Fatalf("TextContent(#doctype) error = %v", err)
	} else if got != "true" {
		t.Fatalf("TextContent(#doctype) = %q, want true", got)
	}
	if !harness.Debug().DOMReady() {
		t.Fatalf("Debug().DOMReady() = false, want true after document property bootstrap")
	}
	if got := harness.Debug().DOMError(); got != "" {
		t.Fatalf("Debug().DOMError() = %q, want empty after document property bootstrap", got)
	}
}

func TestInlineScriptsCanReadNodeTreeNavigationThroughPublicActions(t *testing.T) {
	const rawHTML = `<html><body><section id="out"><div id="doc-node-type"></div><div id="doc-node-name"></div><div id="doc-parent-node"></div><div id="doc-parent-element"></div><div id="doc-next-sibling"></div><div id="doc-previous-sibling"></div><div id="doc-owner"></div><div id="doc-child-count"></div><div id="doc-first-element"></div><div id="doc-last-element"></div><div id="doc-root-parent"></div><div id="doc-root-owner"></div><div id="doc-root-parent-element"></div><div id="mixed-node-type"></div><div id="mixed-node-name"></div><div id="mixed-node-value"></div><div id="mixed-owner"></div><div id="mixed-first-child"></div><div id="mixed-last-child"></div><div id="mixed-child-count"></div><div id="first-next-sibling"></div><div id="first-next-element-sibling"></div><div id="second-previous-sibling"></div><div id="second-previous-element-sibling"></div></section><main id="root"><div id="mixed">alpha<span id="child"></span>omega</div><div id="siblings"><span id="first"></span>middle<em id="second"></em></div></main><script>host:setTextContent("#doc-node-type", expr(document.nodeType)); host:setTextContent("#doc-node-name", expr(document.nodeName)); host:setTextContent("#doc-parent-node", expr(document.parentNode === null)); host:setTextContent("#doc-parent-element", expr(document.parentElement === null)); host:setTextContent("#doc-next-sibling", expr(document.nextSibling === null)); host:setTextContent("#doc-previous-sibling", expr(document.previousSibling === null)); host:setTextContent("#doc-owner", expr(document.ownerDocument === null)); host:setTextContent("#doc-child-count", expr(document.childElementCount)); host:setTextContent("#doc-first-element", expr(document.firstElementChild.nodeName)); host:setTextContent("#doc-last-element", expr(document.lastElementChild.nodeName)); host:setTextContent("#doc-root-parent", expr(document.documentElement.parentNode === document)); host:setTextContent("#doc-root-owner", expr(document.documentElement.ownerDocument === document)); host:setTextContent("#doc-root-parent-element", expr(document.documentElement.parentElement === null)); host:setTextContent("#mixed-node-type", expr(document.getElementById("mixed").nodeType)); host:setTextContent("#mixed-node-name", expr(document.getElementById("mixed").nodeName)); host:setTextContent("#mixed-node-value", expr(document.getElementById("mixed").nodeValue === null)); host:setTextContent("#mixed-owner", expr(document.getElementById("mixed").ownerDocument === document)); host:setTextContent("#mixed-first-child", expr(document.getElementById("mixed").firstChild.nodeValue)); host:setTextContent("#mixed-last-child", expr(document.getElementById("mixed").lastChild.nodeValue)); host:setTextContent("#mixed-child-count", expr(document.getElementById("mixed").childElementCount)); host:setTextContent("#first-next-sibling", expr(document.getElementById("first").nextSibling.nodeValue)); host:setTextContent("#first-next-element-sibling", expr(document.getElementById("first").nextElementSibling.id)); host:setTextContent("#second-previous-sibling", expr(document.getElementById("second").previousSibling.nodeValue)); host:setTextContent("#second-previous-element-sibling", expr(document.getElementById("second").previousElementSibling.id))</script></body></html>`

	harness, err := FromHTML(rawHTML)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	checks := map[string]string{
		"#doc-node-type":                   "9",
		"#doc-node-name":                   "#document",
		"#doc-parent-node":                 "true",
		"#doc-parent-element":              "true",
		"#doc-next-sibling":                "true",
		"#doc-previous-sibling":            "true",
		"#doc-owner":                       "true",
		"#doc-child-count":                 "1",
		"#doc-first-element":               "HTML",
		"#doc-last-element":                "HTML",
		"#doc-root-parent":                 "true",
		"#doc-root-owner":                  "true",
		"#doc-root-parent-element":         "true",
		"#mixed-node-type":                 "1",
		"#mixed-node-name":                 "DIV",
		"#mixed-node-value":                "true",
		"#mixed-owner":                     "true",
		"#mixed-first-child":               "alpha",
		"#mixed-last-child":                "omega",
		"#mixed-child-count":               "1",
		"#first-next-sibling":              "middle",
		"#first-next-element-sibling":      "second",
		"#second-previous-sibling":         "middle",
		"#second-previous-element-sibling": "first",
	}

	for selector, want := range checks {
		if got, err := harness.TextContent(selector); err != nil {
			t.Fatalf("TextContent(%s) error = %v", selector, err)
		} else if got != want {
			t.Fatalf("TextContent(%s) = %q, want %q", selector, got, want)
		}
	}
}

func TestInlineScriptsCanUseNodeListCollectionParityThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><ul><li id="a"></li><li id="b"></li></ul><div id="probe"></div><script>const nodes = document.querySelectorAll("li"); let forEachOut = ""; nodes.forEach((node, index, list) => { forEachOut += (forEachOut === "" ? "" : "|") + index + ":" + node.id + ":" + list.length; }); let keysOut = ""; for (let key of nodes.keys()) { keysOut += (keysOut === "" ? "" : "|") + key; }; let valuesOut = ""; for (let node of nodes.values()) { valuesOut += (valuesOut === "" ? "" : "|") + node.id; }; let entriesOut = ""; for (let entry of nodes.entries()) { entriesOut += (entriesOut === "" ? "" : "|") + entry[0] + ":" + entry[1].id; }; host:setTextContent("#probe", expr(forEachOut + ";" + keysOut + ";" + valuesOut + ";" + entriesOut))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#probe"); err != nil {
		t.Fatalf("TextContent(#probe) error = %v", err)
	} else if want := "0:a:2|1:b:2;0|1;a|b;0:a|1:b"; got != want {
		t.Fatalf("TextContent(#probe) = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanExposeCurrentScriptThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script id="boot">host:setInnerHTML("#out", expr(host:documentCurrentScript()))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out"><script id="boot">host:setInnerHTML("#out", expr(host:documentCurrentScript()))</script></div><script id="boot">host:setInnerHTML("#out", expr(host:documentCurrentScript()))</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after documentCurrentScript bootstrap = %q, want %q", got, want)
	}
}

func TestInlineScriptsTreatEventHandlersAsNonScriptContextsForCurrentScript(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="out">old</div><script>host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", expr(host:documentCurrentScript()))')</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="out"></div><script>host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", expr(host:documentCurrentScript()))')</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after event handler currentScript = %q, want %q", got, want)
	}
}

func TestNavigateResolvesRelativeURLs(t *testing.T) {
	harness, err := FromHTMLWithURL("https://example.test/start", "<main></main>")
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.Navigate("next"); err != nil {
		t.Fatalf("Navigate(next) error = %v", err)
	}

	if got, want := harness.URL(), "https://example.test/next"; got != want {
		t.Fatalf("URL() after relative Navigate = %q, want %q", got, want)
	}
	if got, want := harness.Debug().URL(), "https://example.test/next"; got != want {
		t.Fatalf("Debug().URL() after relative Navigate = %q, want %q", got, want)
	}
}

func TestHasPseudoClassMatchesDescendantSubtrees(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><section id="wrap"><article id="a1"><span class="hit">Hit</span></article><article id="a2"><span class="miss">Miss</span></article></section><aside id="adjacent" class="hit"><span class="hit">Sibling</span></aside><aside id="plain"><span class="hit">Outside</span></aside></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("section:has(.hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(.hit)) error = %v", err)
	}
	if err := harness.AssertExists("section:has(article > .hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(article > .hit)) error = %v", err)
	}
	if err := harness.AssertExists("section:has(:bogus, .hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(:bogus, .hit)) error = %v", err)
	}
	if err := harness.AssertExists("section:has(> article > .hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(> article > .hit)) error = %v", err)
	}
	if err := harness.AssertExists("section:has(+ aside.hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(+ aside.hit)) error = %v", err)
	}
	if err := harness.AssertExists("section:has(~ aside.hit)"); err != nil {
		t.Fatalf("AssertExists(section:has(~ aside.hit)) error = %v", err)
	}
	if err := harness.AssertExists("article:has(.hit, .miss)"); err != nil {
		t.Fatalf("AssertExists(article:has(.hit, .miss)) error = %v", err)
	}
	if err := harness.AssertExists("section:has(.missing)"); err == nil {
		t.Fatalf("AssertExists(section:has(.missing)) error = nil, want no match")
	}
}

func TestNotPseudoClassFiltersCurrentNodes(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><section id="wrap"><article id="a1" class="match"><span class="hit">Hit</span></article><article id="a2"><span class="miss">Miss</span></article></section><aside id="plain"><span class="hit">Outside</span></aside></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("section:not(.missing)"); err != nil {
		t.Fatalf("AssertExists(section:not(.missing)) error = %v", err)
	}
	if err := harness.AssertExists("section:not(:bogus)"); err != nil {
		t.Fatalf("AssertExists(section:not(:bogus)) error = %v", err)
	}
	if err := harness.AssertExists("article:not(.match, .other)"); err != nil {
		t.Fatalf("AssertExists(article:not(.match, .other)) error = %v", err)
	}
	if err := harness.AssertExists("section:not(:bogus, #wrap)"); err == nil {
		t.Fatalf("AssertExists(section:not(:bogus, #wrap)) error = nil, want no match")
	}
	if err := harness.AssertExists("#a1:not(.match)"); err == nil {
		t.Fatalf("AssertExists(#a1:not(.match)) error = nil, want no match")
	}
}

func TestIsAndWherePseudoClassesMatchCurrentNodes(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><section id="wrap" class="match"><article id="a1" class="hit">One</article><article id="a2" class="miss">Two</article></section><aside id="plain"><span class="hit">Outside</span></aside></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists("section:is(#wrap, .missing)"); err != nil {
		t.Fatalf("AssertExists(section:is(#wrap, .missing)) error = %v", err)
	}
	if err := harness.AssertExists("section:is(:bogus, #wrap)"); err != nil {
		t.Fatalf("AssertExists(section:is(:bogus, #wrap)) error = %v", err)
	}
	if err := harness.AssertExists("section:where(#wrap)"); err != nil {
		t.Fatalf("AssertExists(section:where(#wrap)) error = %v", err)
	}
	if err := harness.AssertExists("section:where(, #wrap, )"); err != nil {
		t.Fatalf("AssertExists(section:where(, #wrap, )) error = %v", err)
	}
	if err := harness.AssertExists("article:where(.hit, .miss)"); err != nil {
		t.Fatalf("AssertExists(article:where(.hit, .miss)) error = %v", err)
	}
	if err := harness.AssertExists("article:is(.hit)"); err != nil {
		t.Fatalf("AssertExists(article:is(.hit)) error = %v", err)
	}
	if err := harness.AssertExists("#plain:is(.hit)"); err == nil {
		t.Fatalf("AssertExists(#plain:is(.hit)) error = nil, want no match")
	}
}

func TestScopePseudoClassMatchesDocumentRootContext(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><section id="panel"><p id="child">one</p></section><p id="sibling">two</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertExists(":scope"); err != nil {
		t.Fatalf("AssertExists(:scope) error = %v", err)
	}
	if err := harness.AssertExists(":scope > section"); err != nil {
		t.Fatalf("AssertExists(:scope > section) error = %v", err)
	}
	if err := harness.AssertExists(":scope > p"); err != nil {
		t.Fatalf("AssertExists(:scope > p) error = %v", err)
	}
	if err := harness.AssertExists("section :scope"); err == nil {
		t.Fatalf("AssertExists(section :scope) error = nil, want no match")
	}
}

func TestNthPseudoClassMatchesChildPositions(t *testing.T) {
	harness, err := FromHTML(`<main id="root"><ul id="list"><li id="one" class="selected">1</li><li id="two">2</li><li id="three" class="selected">3</li><li id="four" class="selected">4</li><li id="five">5</li></ul><div id="mixed"><p id="para-a">A</p><span id="mid">M</span><p id="para-b">B</p><p id="para-c">C</p></div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("li:nth-child(3)", "3"); err != nil {
		t.Fatalf("AssertText(li:nth-child(3)) error = %v", err)
	}
	if err := harness.AssertExists("li:nth-child(odd)"); err != nil {
		t.Fatalf("AssertExists(li:nth-child(odd)) error = %v", err)
	}
	if err := harness.AssertText("li:nth-child(2 of .selected)", "3"); err != nil {
		t.Fatalf("AssertText(li:nth-child(2 of .selected)) error = %v", err)
	}
	if err := harness.AssertText("li:nth-child(2 of .selected, #two)", "2"); err != nil {
		t.Fatalf("AssertText(li:nth-child(2 of .selected, #two)) error = %v", err)
	}
	if err := harness.AssertText("p:nth-of-type(3)", "C"); err != nil {
		t.Fatalf("AssertText(p:nth-of-type(3)) error = %v", err)
	}
	if err := harness.AssertText("li:nth-last-child(1)", "5"); err != nil {
		t.Fatalf("AssertText(li:nth-last-child(1)) error = %v", err)
	}
	if err := harness.AssertText("li:nth-last-child(1 of .selected)", "4"); err != nil {
		t.Fatalf("AssertText(li:nth-last-child(1 of .selected)) error = %v", err)
	}
	if err := harness.AssertText("p:nth-last-of-type(2)", "B"); err != nil {
		t.Fatalf("AssertText(p:nth-last-of-type(2)) error = %v", err)
	}
	if err := harness.AssertExists("span:nth-of-type(2)"); err == nil {
		t.Fatalf("AssertExists(span:nth-of-type(2)) error = nil, want no match")
	}
	if err := harness.AssertExists("li:nth-last-child(6)"); err == nil {
		t.Fatalf("AssertExists(li:nth-last-child(6)) error = nil, want no match")
	}
}

func TestTargetPseudoClassTracksLocationFragments(t *testing.T) {
	harness, err := FromHTMLWithURL("https://example.test/page#legacy", `<main id="root"><a name="legacy">legacy</a><div id="space target">space</div><p id="tail">tail</p></main>`)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.AssertText("a:target", "legacy"); err != nil {
		t.Fatalf("AssertText(a:target) error = %v", err)
	}
	if err := harness.AssertExists("main:target-within"); err != nil {
		t.Fatalf("AssertExists(main:target-within) after bootstrap error = %v", err)
	}
	if err := harness.Navigate("#space%20target"); err != nil {
		t.Fatalf("Navigate(#space%%20target) error = %v", err)
	}
	if err := harness.AssertText("div:target", "space"); err != nil {
		t.Fatalf("AssertText(div:target) error = %v", err)
	}
	if err := harness.AssertExists("main:target-within"); err != nil {
		t.Fatalf("AssertExists(main:target-within) after encoded fragment error = %v", err)
	}
	if err := harness.Navigate("#missing"); err != nil {
		t.Fatalf("Navigate(#missing) error = %v", err)
	}
	if err := harness.AssertExists(":target"); err == nil {
		t.Fatalf("AssertExists(:target) after missing fragment error = nil, want no match")
	}
	if err := harness.AssertExists(":target-within"); err == nil {
		t.Fatalf("AssertExists(:target-within) after missing fragment error = nil, want no match")
	}
}

func TestLangPseudoClassTracksInheritedLanguage(t *testing.T) {
	harness, err := FromHTML(`<main id="root" lang="en-US"><section id="panel"><p id="inherited">Hello</p></section><article id="french" lang="fr"><span id="direct">Salut</span><div id="unknown" lang=""><em id="blank">Nada</em></div></article></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("p:lang(en)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:lang(en)) error = %v", err)
	}
	if err := harness.AssertText("span:lang(fr)", "Salut"); err != nil {
		t.Fatalf("AssertText(span:lang(fr)) error = %v", err)
	}

	if err := harness.SetAttribute("#root", "lang", "fr"); err != nil {
		t.Fatalf("SetAttribute(#root, lang, fr) error = %v", err)
	}
	if err := harness.AssertText("p:lang(fr)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:lang(fr)) after SetAttribute error = %v", err)
	}
	if err := harness.AssertExists("p:lang(en)"); err == nil {
		t.Fatalf("AssertExists(p:lang(en)) after SetAttribute error = nil, want no match")
	}
}

func TestDirPseudoClassTracksInheritedDirection(t *testing.T) {
	harness, err := FromHTML(`<main id="root" dir="rtl"><section id="panel"><p id="inherited">Hello</p><div id="auto-ltr" dir="auto">abc</div><div id="auto-rtl" dir="auto">مرحبا</div></section><article id="ltr" dir="ltr"><span id="nested">Salut</span></article></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("p:dir(rtl)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:dir(rtl)) error = %v", err)
	}
	if err := harness.AssertText("div:dir(ltr)", "abc"); err != nil {
		t.Fatalf("AssertText(div:dir(ltr)) error = %v", err)
	}
	if err := harness.AssertText("div:dir(rtl)", "مرحبا"); err != nil {
		t.Fatalf("AssertText(div:dir(rtl)) error = %v", err)
	}
	if err := harness.AssertText("span:dir(ltr)", "Salut"); err != nil {
		t.Fatalf("AssertText(span:dir(ltr)) error = %v", err)
	}

	if err := harness.SetAttribute("#root", "dir", "ltr"); err != nil {
		t.Fatalf("SetAttribute(#root, dir, ltr) error = %v", err)
	}
	if err := harness.AssertText("p:dir(ltr)", "Hello"); err != nil {
		t.Fatalf("AssertText(p:dir(ltr)) after SetAttribute error = %v", err)
	}
	if err := harness.AssertExists("p:dir(rtl)"); err == nil {
		t.Fatalf("AssertExists(p:dir(rtl)) after SetAttribute error = nil, want no match")
	}
}

func TestWriteHTMLRejectsInvalidMarkupWithoutMutatingDocument(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.WriteHTML(`<main><div id="broken"></main>`); err == nil {
		t.Fatalf("WriteHTML(invalid) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("WriteHTML(invalid) error = %#v, want DOM error", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">old</div></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after failed WriteHTML = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), `<main><div id="out">old</div></main>`; got != want {
		t.Fatalf("HTML() after failed WriteHTML = %q, want %q", got, want)
	}
}

func TestWriteHTMLRestoresSessionStateOnHostFailure(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/app#target",
		`<main><button id="btn">Go</button><div id="target">target</div><script>host:setWindowName("alpha"); host:setDocumentCookie("theme=dark"); host:historySetScrollRestoration("manual"); host:historyPushState("step-1", "", "/step-1#target")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if got := harness.Debug().TargetNodeID(); got == 0 {
		t.Fatalf("Debug().TargetNodeID() before failed WriteHTML = %d, want targeted node", got)
	}
	if err := harness.Focus("#btn"); err != nil {
		t.Fatalf("Focus(#btn) error = %v", err)
	}
	if err := harness.ScrollTo(11, 17); err != nil {
		t.Fatalf("ScrollTo() error = %v", err)
	}

	if err := harness.WriteHTML(`<main><script>host:setWindowName("beta"); host:setDocumentCookie("lang=en"); host:historySetScrollRestoration("auto"); host:locationAssign("/next"); host:doesNotExist()</script></main>`); err == nil {
		t.Fatalf("WriteHTML(host failure) error = nil, want host error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("WriteHTML(host failure) error = %#v, want DOM error", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="target">target</div><script>host:setWindowName("alpha"); host:setDocumentCookie("theme=dark"); host:historySetScrollRestoration("manual"); host:historyPushState("step-1", "", "/step-1#target")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after failed WriteHTML = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), `<main><button id="btn">Go</button><div id="target">target</div><script>host:setWindowName("alpha"); host:setDocumentCookie("theme=dark"); host:historySetScrollRestoration("manual"); host:historyPushState("step-1", "", "/step-1#target")</script></main>`; got != want {
		t.Fatalf("HTML() after failed WriteHTML = %q, want %q", got, want)
	}
	if got := harness.Debug().TargetNodeID(); got == 0 {
		t.Fatalf("Debug().TargetNodeID() after failed WriteHTML = %d, want targeted node", got)
	}
	if got := harness.Debug().FocusedSelector(); got != "#btn" {
		t.Fatalf("Debug().FocusedSelector() after failed WriteHTML = %q, want %q", got, "#btn")
	}
	if gotX, gotY := harness.Debug().ScrollPosition(); gotX != 11 || gotY != 17 {
		t.Fatalf("Debug().ScrollPosition() after failed WriteHTML = (%d, %d), want (11, 17)", gotX, gotY)
	}
	if got, want := harness.Debug().WindowName(), "alpha"; got != want {
		t.Fatalf("Debug().WindowName() after failed WriteHTML = %q, want %q", got, want)
	}
	if got, want := harness.Debug().DocumentCookie(), "theme=dark"; got != want {
		t.Fatalf("Debug().DocumentCookie() after failed WriteHTML = %q, want %q", got, want)
	}
	if got, want := harness.Debug().HistoryScrollRestoration(), "manual"; got != want {
		t.Fatalf("Debug().HistoryScrollRestoration() after failed WriteHTML = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://example.test/step-1#target"; got != want {
		t.Fatalf("URL() after failed WriteHTML = %q, want %q", got, want)
	}
	if got := harness.Debug().HistoryLength(); got != 2 {
		t.Fatalf("Debug().HistoryLength() after failed WriteHTML = %d, want 2", got)
	}
	if got, ok := harness.Debug().HistoryState(); !ok || got != "step-1" {
		t.Fatalf("Debug().HistoryState() after failed WriteHTML = (%q, %v), want (\"step-1\", true)", got, ok)
	}
	if got, want := harness.Debug().NavigationLog(), []string{"https://example.test/step-1#target"}; len(got) != len(want) {
		t.Fatalf("Debug().NavigationLog() after failed WriteHTML = %#v, want %#v", got, want)
	} else if got[0] != want[0] {
		t.Fatalf("Debug().NavigationLog()[0] after failed WriteHTML = %q, want %q", got[0], want[0])
	}
}

func TestInlineScriptsRejectReadOnlyLocationOriginThroughPublicActions(t *testing.T) {
	harness, err := FromHTMLWithURL("https://example.test/start", `<main><div id="out">old</div></main>`)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.WriteHTML(`<main><script>host:locationSet("origin", "https://other.test/")</script></main>`); err == nil {
		t.Fatalf("WriteHTML(locationSet origin) error = nil, want read-only failure")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("WriteHTML(locationSet origin) error = %#v, want DOM error", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">old</div></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after rejected origin assignment = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), `<main><div id="out">old</div></main>`; got != want {
		t.Fatalf("HTML() after rejected origin assignment = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://example.test/start"; got != want {
		t.Fatalf("URL() after rejected origin assignment = %q, want %q", got, want)
	}
	if got, want := harness.Debug().NavigationLog(), []string{}; len(got) != len(want) {
		t.Fatalf("Debug().NavigationLog() after rejected origin assignment = %#v, want %#v", got, want)
	}
}

func TestInlineScriptsRejectInvalidHistoryScrollRestorationThroughPublicActions(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/app",
		`<main><script>host:setWindowName("alpha"); host:setDocumentCookie("theme=dark"); host:historySetScrollRestoration("manual"); host:historyPushState("step-1", "", "/step-1")</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.WriteHTML(`<main><script>host:historySetScrollRestoration("sideways")</script></main>`); err == nil {
		t.Fatalf("WriteHTML(historySetScrollRestoration sideways) error = nil, want validation failure")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("WriteHTML(historySetScrollRestoration sideways) error = %#v, want DOM error", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><script>host:setWindowName("alpha"); host:setDocumentCookie("theme=dark"); host:historySetScrollRestoration("manual"); host:historyPushState("step-1", "", "/step-1")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after rejected history scroll restoration = %q, want %q", got, want)
	}
	if got, want := harness.HTML(), `<main><script>host:setWindowName("alpha"); host:setDocumentCookie("theme=dark"); host:historySetScrollRestoration("manual"); host:historyPushState("step-1", "", "/step-1")</script></main>`; got != want {
		t.Fatalf("HTML() after rejected history scroll restoration = %q, want %q", got, want)
	}
	if got, want := harness.Debug().WindowName(), "alpha"; got != want {
		t.Fatalf("Debug().WindowName() after rejected history scroll restoration = %q, want %q", got, want)
	}
	if got, want := harness.Debug().DocumentCookie(), "theme=dark"; got != want {
		t.Fatalf("Debug().DocumentCookie() after rejected history scroll restoration = %q, want %q", got, want)
	}
	if got, want := harness.Debug().HistoryScrollRestoration(), "manual"; got != want {
		t.Fatalf("Debug().HistoryScrollRestoration() after rejected history scroll restoration = %q, want %q", got, want)
	}
	if got, want := harness.URL(), "https://example.test/step-1"; got != want {
		t.Fatalf("URL() after rejected history scroll restoration = %q, want %q", got, want)
	}
	if got := harness.Debug().HistoryLength(); got != 2 {
		t.Fatalf("Debug().HistoryLength() after rejected history scroll restoration = %d, want 2", got)
	}
	if got, ok := harness.Debug().HistoryState(); !ok || got != "step-1" {
		t.Fatalf("Debug().HistoryState() after rejected history scroll restoration = (%q, %v), want (\"step-1\", true)", got, ok)
	}
	if got, want := harness.Debug().NavigationLog(), []string{"https://example.test/step-1"}; len(got) != len(want) {
		t.Fatalf("Debug().NavigationLog() after rejected history scroll restoration = %#v, want %#v", got, want)
	} else if got[0] != want[0] {
		t.Fatalf("Debug().NavigationLog()[0] after rejected history scroll restoration = %q, want %q", got[0], want[0])
	}
}

func TestNilHarnessWriteHTMLWrapperReturnsError(t *testing.T) {
	var harness *Harness

	if err := harness.WriteHTML("<main></main>"); err == nil {
		t.Fatalf("nil Harness.WriteHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.WriteHTML() error = %#v, want DOM error", err)
	}
}

func TestInlineScriptsDispatchTargetListenersThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="out"></div><script>host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", "clicked"); host:setInnerHTML("#out", "done")')</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="out">done</div><script>host:addEventListener("#btn", "click", 'host:setInnerHTML("#out", "clicked"); host:setInnerHTML("#out", "done")')</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after listener dispatch = %q, want %q", got, want)
	}
}

func TestInlineScriptsDispatchCaptureTargetAndBubbleListenersThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><section id="wrap"><button id="btn">Go</button></section><div id="log"></div><script>host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>capture</span>")', "capture"); host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>target</span>")'); host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><section id="wrap"><button id="btn">Go</button></section><div id="log"><span>capture</span><span>target</span><span>bubble</span></div><script>host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>capture</span>")', "capture"); host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>target</span>")'); host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after capture/target/bubble listeners = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanPreventDefaultActionsThroughPublicActions(t *testing.T) {
	harness, err := FromHTMLWithURL(
		"https://example.test/base/",
		`<main><a id="nav" href="/next">Go</a><div id="out"></div><script>host:addEventListener("#nav", "click", 'host:preventDefault(); host:setInnerHTML("#out", "blocked")')</script></main>`,
	)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.Click("#nav"); err != nil {
		t.Fatalf("Click(#nav) error = %v", err)
	}

	if got, want := harness.URL(), "https://example.test/base/"; got != want {
		t.Fatalf("URL() after prevented click = %q, want %q", got, want)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><a id="nav" href="/next">Go</a><div id="out">blocked</div><script>host:addEventListener("#nav", "click", 'host:preventDefault(); host:setInnerHTML("#out", "blocked")')</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after prevented click = %q, want %q", got, want)
	}
	if got := harness.Mocks().Location().Navigations(); len(got) != 0 {
		t.Fatalf("Location().Navigations() = %#v, want no navigation", got)
	}
}

func TestInlineScriptsCanStopPropagationThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><section id="wrap"><button id="btn">Go</button></section><div id="log"></div><script>host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>capture</span>"); host:stopPropagation()', "capture"); host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>target</span>")'); host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><section id="wrap"><button id="btn">Go</button></section><div id="log"><span>capture</span><span>target</span></div><script>host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>capture</span>"); host:stopPropagation()', "capture"); host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>target</span>")'); host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after stopPropagation click = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanUseOnceListenersThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="log"></div><script>host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>once</span>")', "target", true)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) first error = %v", err)
	}
	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) second error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="log"><span>once</span></div><script>host:addEventListener("#btn", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>once</span>")', "target", true)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after once listener = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanRemoveLaterListenersThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><section id="wrap"><button id="btn">Go</button></section><div id="log"></div><script>host:addEventListener("#wrap", "click", 'host:removeEventListener("#btn", "click", host:removeNode("#btn")); host:insertAdjacentHTML("#log", "beforeend", "<span>remover</span>")', "capture"); host:addEventListener("#btn", "click", 'host:removeNode("#btn")'); host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><section id="wrap"><button id="btn">Go</button></section><div id="log"><span>remover</span><span>bubble</span></div><script>host:addEventListener("#wrap", "click", 'host:removeEventListener("#btn", "click", host:removeNode("#btn")); host:insertAdjacentHTML("#log", "beforeend", "<span>remover</span>")', "capture"); host:addEventListener("#btn", "click", 'host:removeNode("#btn")'); host:addEventListener("#wrap", "click", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after listener removal = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanDispatchCustomEventsThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><section id="wrap"><button id="btn">Go</button></section><div id="log"></div><script>host:addEventListener("#wrap", "custom", 'host:insertAdjacentHTML("#log", "beforeend", "<span>capture</span>")', "capture"); host:addEventListener("#btn", "custom", 'host:insertAdjacentHTML("#log", "beforeend", "<span>target</span>")'); host:addEventListener("#wrap", "custom", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Dispatch("#btn", "custom"); err != nil {
		t.Fatalf("Dispatch(#btn, custom) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><section id="wrap"><button id="btn">Go</button></section><div id="log"><span>capture</span><span>target</span><span>bubble</span></div><script>host:addEventListener("#wrap", "custom", 'host:insertAdjacentHTML("#log", "beforeend", "<span>capture</span>")', "capture"); host:addEventListener("#btn", "custom", 'host:insertAdjacentHTML("#log", "beforeend", "<span>target</span>")'); host:addEventListener("#wrap", "custom", 'host:insertAdjacentHTML("#log", "beforeend", "<span>bubble</span>")', "bubble")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after custom dispatch = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanDispatchKeyboardSequencesThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="log"></div><script>host:addEventListener("#btn", "keydown", 'host:insertAdjacentHTML("#log", "beforeend", "<span>down</span>")'); host:addEventListener("#btn", "keypress", 'host:insertAdjacentHTML("#log", "beforeend", "<span>press</span>")'); host:addEventListener("#btn", "keyup", 'host:insertAdjacentHTML("#log", "beforeend", "<span>up</span>")')</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.DispatchKeyboard("#btn"); err != nil {
		t.Fatalf("DispatchKeyboard(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="log"><span>down</span><span>press</span><span>up</span></div><script>host:addEventListener("#btn", "keydown", 'host:insertAdjacentHTML("#log", "beforeend", "<span>down</span>")'); host:addEventListener("#btn", "keypress", 'host:insertAdjacentHTML("#log", "beforeend", "<span>press</span>")'); host:addEventListener("#btn", "keyup", 'host:insertAdjacentHTML("#log", "beforeend", "<span>up</span>")')</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after keyboard dispatch = %q, want %q", got, want)
	}
}

func TestInlineScriptsQueueMicrotasksDuringBootstrapThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">start</div><script>host:queueMicrotask('host:setInnerHTML(#out, micro)')</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">micro</div><script>host:queueMicrotask('host:setInnerHTML(#out, micro)')</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after bootstrap microtask = %q, want %q", got, want)
	}
}

func TestInlineScriptsQueueMicrotasksAfterClickThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><button id="btn">Go</button><div id="out">start</div><script>host:addEventListener("#btn", "click", "host:queueMicrotask('host:setInnerHTML(#out, micro)')")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Click("#btn"); err != nil {
		t.Fatalf("Click(#btn) error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><button id="btn">Go</button><div id="out">micro</div><script>host:addEventListener("#btn", "click", "host:queueMicrotask('host:setInnerHTML(#out, micro)')")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after click microtask = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanAdvanceTimersThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">start</div><script>host:setTimeout('host:setInnerHTML(#out, micro)', 5)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">start</div><script>host:setTimeout('host:setInnerHTML(#out, micro)', 5)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() before AdvanceTime = %q, want %q", got, want)
	}
	if err := harness.AdvanceTime(4); err != nil {
		t.Fatalf("AdvanceTime(4) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">start</div><script>host:setTimeout('host:setInnerHTML(#out, micro)', 5)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() before timer due = %q, want %q", got, want)
	}
	if err := harness.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">micro</div><script>host:setTimeout('host:setInnerHTML(#out, micro)', 5)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after timer = %q, want %q", got, want)
	}
}

func TestInlineScriptsCanScheduleRepeatingTimersThroughPublicActions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">start</div><script>host:setInterval('host:insertAdjacentHTML("#out", "beforeend", "<span>tick</span>")', 5)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) first error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">start<span>tick</span></div><script>host:setInterval('host:insertAdjacentHTML("#out", "beforeend", "<span>tick</span>")', 5)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after first interval = %q, want %q", got, want)
	}

	if err := harness.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) second error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="out">start<span>tick</span><span>tick</span></div><script>host:setInterval('host:insertAdjacentHTML("#out", "beforeend", "<span>tick</span>")', 5)</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after second interval = %q, want %q", got, want)
	}
}

func TestFormControlActionsRejectUnsupportedTargets(t *testing.T) {
	harness, err := FromHTML(`<main><input id="name"><input id="flag" type="checkbox"><select id="mode"><option>A</option></select></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.TypeText("#flag", "Ada"); err == nil {
		t.Fatalf("TypeText(#flag) error = nil, want unsupported control error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TypeText(#flag) error = %#v, want DOM error", err)
	}
	if err := harness.SetChecked("#name", true); err == nil {
		t.Fatalf("SetChecked(#name) error = nil, want unsupported control error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetChecked(#name) error = %#v, want DOM error", err)
	}
	if err := harness.SetSelectValue("#name", "A"); err == nil {
		t.Fatalf("SetSelectValue(#name) error = nil, want unsupported control error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetSelectValue(#name) error = %#v, want DOM error", err)
	}
	if err := harness.Submit("#name"); err == nil {
		t.Fatalf("Submit(#name) error = nil, want unsupported target error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("Submit(#name) error = %#v, want DOM error", err)
	}
}

func TestNilHarnessFormControlWrappersReturnErrors(t *testing.T) {
	var harness *Harness

	if err := harness.TypeText("#name", "Ada"); err == nil {
		t.Fatalf("nil Harness.TypeText() error = nil, want DOM error")
	}
	if err := harness.SetChecked("#flag", true); err == nil {
		t.Fatalf("nil Harness.SetChecked() error = nil, want DOM error")
	}
	if err := harness.SetSelectValue("#mode", "B"); err == nil {
		t.Fatalf("nil Harness.SetSelectValue() error = nil, want DOM error")
	}
	if err := harness.Submit("#profile"); err == nil {
		t.Fatalf("nil Harness.Submit() error = nil, want DOM error")
	}
}

func TestNilHarnessEventWrappersReturnErrors(t *testing.T) {
	var harness *Harness

	if err := harness.Click("#cta"); err == nil {
		t.Fatalf("nil Harness.Click() error = nil, want event error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("nil Harness.Click() error = %#v, want event error", err)
	}
	if err := harness.Focus("#cta"); err == nil {
		t.Fatalf("nil Harness.Focus() error = nil, want event error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("nil Harness.Focus() error = %#v, want event error", err)
	}
	if err := harness.Blur(); err == nil {
		t.Fatalf("nil Harness.Blur() error = nil, want event error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("nil Harness.Blur() error = %#v, want event error", err)
	}
	if err := harness.Dispatch("#cta", "custom"); err == nil {
		t.Fatalf("nil Harness.Dispatch() error = nil, want event error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("nil Harness.Dispatch() error = %#v, want event error", err)
	}
	if err := harness.DispatchKeyboard("#cta"); err == nil {
		t.Fatalf("nil Harness.DispatchKeyboard() error = nil, want event error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindEvent {
		t.Fatalf("nil Harness.DispatchKeyboard() error = %#v, want event error", err)
	}
}

func TestNilHarnessTimeWrappersReturnErrors(t *testing.T) {
	var harness *Harness

	if err := harness.AdvanceTime(1); err == nil {
		t.Fatalf("nil Harness.AdvanceTime() error = nil, want timer error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindTimer {
		t.Fatalf("nil Harness.AdvanceTime() error = %#v, want timer error", err)
	}
}

func TestNilHarnessMatchMediaWrapperReturnsError(t *testing.T) {
	var harness *Harness

	if _, err := harness.MatchMedia("(prefers-reduced-motion: reduce)"); err == nil {
		t.Fatalf("nil Harness.MatchMedia() error = nil, want mock error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindMock {
		t.Fatalf("nil Harness.MatchMedia() error = %#v, want mock error", err)
	}
}

func TestNilHarnessAttributeWrappersReturnErrors(t *testing.T) {
	var harness *Harness

	if _, _, err := harness.GetAttribute("#root", "id"); err == nil {
		t.Fatalf("nil Harness.GetAttribute() error = nil, want DOM error")
	}
	if _, err := harness.HasAttribute("#root", "id"); err == nil {
		t.Fatalf("nil Harness.HasAttribute() error = nil, want DOM error")
	}
	if err := harness.SetAttribute("#root", "id", "x"); err == nil {
		t.Fatalf("nil Harness.SetAttribute() error = nil, want DOM error")
	}
	if err := harness.RemoveAttribute("#root", "id"); err == nil {
		t.Fatalf("nil Harness.RemoveAttribute() error = nil, want DOM error")
	}
}

func TestInlineScriptsMutateDOMDuringBootstrap(t *testing.T) {
	harness, err := FromHTML(`<main><div id="target">old</div><script>host:setInnerHTML("#target", "<em>updated</em>")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="target"><em>updated</em></div><script>host:setInnerHTML("#target", "<em>updated</em>")</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after inline script bootstrap = %q, want %q", got, want)
	}
}

func TestMutationContractsGettersAndSetters(t *testing.T) {
	harness, err := FromHTML(`<section id="wrap"><div id="target"><p>Hello</p><span>world</span></div><p id="tail">tail</p></section>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.InnerHTML("#target"); err != nil {
		t.Fatalf("InnerHTML(#target) error = %v", err)
	} else if want := `<p>Hello</p><span>world</span>`; got != want {
		t.Fatalf("InnerHTML(#target) = %q, want %q", got, want)
	}

	if got, err := harness.OuterHTML("#target"); err != nil {
		t.Fatalf("OuterHTML(#target) error = %v", err)
	} else if want := `<div id="target"><p>Hello</p><span>world</span></div>`; got != want {
		t.Fatalf("OuterHTML(#target) = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#target"); err != nil {
		t.Fatalf("TextContent(#target) error = %v", err)
	} else if want := `Helloworld`; got != want {
		t.Fatalf("TextContent(#target) = %q, want %q", got, want)
	}

	if err := harness.SetInnerHTML("#target", `<em id="next">updated</em>tail`); err != nil {
		t.Fatalf("SetInnerHTML(#target) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<section id="wrap"><div id="target"><em id="next">updated</em>tail</div><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("Debug().DumpDOM() after SetInnerHTML = %q, want %q", got, want)
	}
	if err := harness.SetTextContent("#target", `plain <text> & more`); err != nil {
		t.Fatalf("SetTextContent(#target) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<section id="wrap"><div id="target">plain &lt;text&gt; &amp; more</div><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("Debug().DumpDOM() after SetTextContent = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#target"); err != nil {
		t.Fatalf("TextContent(#target) after SetTextContent error = %v", err)
	} else if want := `plain <text> & more`; got != want {
		t.Fatalf("TextContent(#target) after SetTextContent = %q, want %q", got, want)
	}

	if err := harness.InsertAdjacentHTML("#target", "beforebegin", `<a id="bb"></a>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(beforebegin) error = %v", err)
	}
	if err := harness.InsertAdjacentHTML("#target", "afterbegin", `<i id="ab">a</i>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(afterbegin) error = %v", err)
	}
	if err := harness.InsertAdjacentHTML("#target", "beforeend", `<i id="be">b</i>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(beforeend) error = %v", err)
	}
	if err := harness.InsertAdjacentHTML("#target", "afterend", `<a id="ae"></a>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(afterend) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<section id="wrap"><a id="bb"></a><div id="target"><i id="ab">a</i>plain &lt;text&gt; &amp; more<i id="be">b</i></div><a id="ae"></a><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("Debug().DumpDOM() after InsertAdjacentHTML = %q, want %q", got, want)
	}

	if err := harness.SetOuterHTML("#target", `<article id="next-outer">n</article><aside id="extra"></aside>`); err != nil {
		t.Fatalf("SetOuterHTML(#target) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<section id="wrap"><a id="bb"></a><article id="next-outer">n</article><aside id="extra"></aside><a id="ae"></a><p id="tail">tail</p></section>`; got != want {
		t.Fatalf("Debug().DumpDOM() after SetOuterHTML = %q, want %q", got, want)
	}
}

func TestMutationContractsCloneNodeDuplicatesSubtree(t *testing.T) {
	harness, err := FromHTML(`<main><div id="source"><span id="child">text</span></div><p id="tail">tail</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.CloneNode("#source", true); err != nil {
		t.Fatalf("CloneNode(#source, true) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="source"><span id="child">text</span></div><div id="source"><span id="child">text</span></div><p id="tail">tail</p></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after CloneNode = %q, want %q", got, want)
	}
	if err := harness.AssertText("main > div + div > span", "text"); err != nil {
		t.Fatalf("AssertText(main > div + div > span) after CloneNode error = %v", err)
	}
}

func TestMutationContractsRemoveNodeRemovesSubtree(t *testing.T) {
	harness, err := FromHTML(`<section id="wrap"><div id="remove"><span id="child">x</span></div><p id="keep">k</p></section>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.RemoveNode("#remove"); err != nil {
		t.Fatalf("RemoveNode(#remove) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<section id="wrap"><p id="keep">k</p></section>`; got != want {
		t.Fatalf("Debug().DumpDOM() after RemoveNode = %q, want %q", got, want)
	}
	if err := harness.RemoveNode("#child"); err == nil {
		t.Fatalf("RemoveNode(#child) error = nil, want DOM error after subtree removal")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("RemoveNode(#child) error = %#v, want DOM error", err)
	}
}

func TestMutationContractsRejectInvalidTargets(t *testing.T) {
	harness, err := FromHTML(`<div id="target"><span>ok</span></div><p id="sibling">tail</p>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.InnerHTML("#missing"); err == nil {
		t.Fatalf("InnerHTML(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("InnerHTML(#missing) error = %#v, want DOM error", err)
	}

	if _, err := harness.OuterHTML("#missing"); err == nil {
		t.Fatalf("OuterHTML(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("OuterHTML(#missing) error = %#v, want DOM error", err)
	}
	if _, err := harness.TextContent("#missing"); err == nil {
		t.Fatalf("TextContent(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(#missing) error = %#v, want DOM error", err)
	}

	if err := harness.ReplaceChildren("#missing", "<p>x</p>"); err == nil {
		t.Fatalf("ReplaceChildren(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("ReplaceChildren(#missing) error = %#v, want DOM error", err)
	}
	if err := harness.SetInnerHTML("#missing", "<p>x</p>"); err == nil {
		t.Fatalf("SetInnerHTML(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetInnerHTML(#missing) error = %#v, want DOM error", err)
	}
	if err := harness.SetTextContent("#missing", "x"); err == nil {
		t.Fatalf("SetTextContent(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetTextContent(#missing) error = %#v, want DOM error", err)
	}

	if err := harness.SetOuterHTML("#missing", "<p>x</p>"); err == nil {
		t.Fatalf("SetOuterHTML(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetOuterHTML(#missing) error = %#v, want DOM error", err)
	}

	if err := harness.InsertAdjacentHTML("#missing", "beforeend", "<p>x</p>"); err == nil {
		t.Fatalf("InsertAdjacentHTML(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("InsertAdjacentHTML(#missing) error = %#v, want DOM error", err)
	}

	if err := harness.RemoveNode("#missing"); err == nil {
		t.Fatalf("RemoveNode(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("RemoveNode(#missing) error = %#v, want DOM error", err)
	}
	if err := harness.CloneNode("#missing", true); err == nil {
		t.Fatalf("CloneNode(#missing) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("CloneNode(#missing) error = %#v, want DOM error", err)
	}
}

func TestMutationContractsRejectDocumentParentRestrictions(t *testing.T) {
	harness, err := FromHTML(`<div id="target"><span>ok</span></div><p id="sibling">tail</p>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.SetOuterHTML("#target", `<section id="new"></section>`); err == nil {
		t.Fatalf("SetOuterHTML(document child) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("SetOuterHTML(document child) error = %#v, want DOM error", err)
	}
	if err := harness.InsertAdjacentHTML("#target", "beforebegin", `<a id="bb"></a>`); err == nil {
		t.Fatalf("InsertAdjacentHTML(beforebegin on document child) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("InsertAdjacentHTML(beforebegin on document child) error = %#v, want DOM error", err)
	}

	if err := harness.InsertAdjacentHTML("#target", "afterend", `<a id="ae"></a>`); err == nil {
		t.Fatalf("InsertAdjacentHTML(afterend on document child) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("InsertAdjacentHTML(afterend on document child) error = %#v, want DOM error", err)
	}
}

func TestMutationContractsSetTextContentPreservesFocusAndClearsTargetDescendants(t *testing.T) {
	harness, err := FromHTML(`<main><div id="target"><span id="child">x</span></div><p id="other">y</p></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.Focus("#target"); err != nil {
		t.Fatalf("Focus(#target) error = %v", err)
	}
	if err := harness.Navigate("#child"); err != nil {
		t.Fatalf("Navigate(#child) error = %v", err)
	}
	if err := harness.SetTextContent("#target", "plain"); err != nil {
		t.Fatalf("SetTextContent(#target) error = %v", err)
	}
	if got, want := harness.Debug().FocusedSelector(), "#target"; got != want {
		t.Fatalf("Debug().FocusedSelector() after SetTextContent = %q, want %q", got, want)
	}
	if err := harness.AssertExists(":target"); err == nil {
		t.Fatalf("AssertExists(:target) after SetTextContent error = nil, want no match")
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="target">plain</div><p id="other">y</p></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after SetTextContent = %q, want %q", got, want)
	}
}

func TestMutationContractsKeepLiveCollectionsCoherentAfterPublicMutations(t *testing.T) {
	harness, err := FromHTML(`<main><div id="target"><span id="first"></span>gap</div><div id="before"></div><div id="after"></div><script>const target = document.querySelector("#target"); host:setTextContent("#before", expr(target.children.length + ":" + target.childNodes.length)); host:replaceChildren("#target", "<em id=\"next\">fresh</em>tail"); host:setTextContent("#after", expr(target.children.length + ":" + target.childNodes.length + ":" + target.children.item(0).id + ":" + target.childNodes.item(1).nodeValue))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#before"); err != nil {
		t.Fatalf("TextContent(#before) error = %v", err)
	} else if want := "1:2"; got != want {
		t.Fatalf("TextContent(#before) = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#after"); err != nil {
		t.Fatalf("TextContent(#after) error = %v", err)
	} else if want := "1:2:next:tail"; got != want {
		t.Fatalf("TextContent(#after) = %q, want %q", got, want)
	}
	if got, want := harness.Debug().DumpDOM(), `<main><div id="target"><em id="next">fresh</em>tail</div><div id="before">1:2</div><div id="after">1:2:next:tail</div><script>const target = document.querySelector("#target"); host:setTextContent("#before", expr(target.children.length + ":" + target.childNodes.length)); host:replaceChildren("#target", "<em id=\"next\">fresh</em>tail"); host:setTextContent("#after", expr(target.children.length + ":" + target.childNodes.length + ":" + target.children.item(0).id + ":" + target.childNodes.item(1).nodeValue))</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after live collection mutation contract = %q, want %q", got, want)
	}
}

func TestMutationContractsSetTextContentUpdatesTextareaDefaultValue(t *testing.T) {
	harness, err := FromHTML(`<form id="profile"><textarea id="bio">Base</textarea><button id="reset" type="reset">Reset</button></form>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.SetTextContent("#bio", "Draft"); err != nil {
		t.Fatalf("SetTextContent(#bio) error = %v", err)
	}
	if got, want := harness.Debug().DumpDOM(), `<form id="profile"><textarea id="bio">Draft</textarea><button id="reset" type="reset">Reset</button></form>`; got != want {
		t.Fatalf("Debug().DumpDOM() after SetTextContent = %q, want %q", got, want)
	}

	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) error = %v", err)
	}
	if got, err := harness.TextContent("#bio"); err != nil {
		t.Fatalf("TextContent(#bio) after reset click error = %v", err)
	} else if got != "Draft" {
		t.Fatalf("TextContent(#bio) after reset click = %q, want %q", got, "Draft")
	}

	if err := harness.TypeText("#bio", "User"); err != nil {
		t.Fatalf("TypeText(#bio) error = %v", err)
	}
	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) after TypeText error = %v", err)
	}
	if got, err := harness.TextContent("#bio"); err != nil {
		t.Fatalf("TextContent(#bio) after reset click following TypeText error = %v", err)
	} else if got != "Draft" {
		t.Fatalf("TextContent(#bio) after reset click following TypeText = %q, want %q", got, "Draft")
	}
}

func TestMutationContractsTextareaChildMutationsUpdateResetDefaultValue(t *testing.T) {
	harness, err := FromHTML(`<form id="profile"><textarea id="bio">Base</textarea><button id="reset" type="reset">Reset</button></form>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.ReplaceChildren("#bio", "Draft"); err != nil {
		t.Fatalf("ReplaceChildren(#bio) error = %v", err)
	}
	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) after ReplaceChildren error = %v", err)
	}
	if got, err := harness.TextContent("#bio"); err != nil {
		t.Fatalf("TextContent(#bio) after ReplaceChildren reset error = %v", err)
	} else if got != "Draft" {
		t.Fatalf("TextContent(#bio) after ReplaceChildren reset = %q, want %q", got, "Draft")
	}

	if err := harness.SetInnerHTML("#bio", "Fresh"); err != nil {
		t.Fatalf("SetInnerHTML(#bio) second update error = %v", err)
	}
	if err := harness.InsertAdjacentHTML("#bio", "beforeend", `<span id="bang">!</span>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(#bio,beforeend) error = %v", err)
	}
	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) after InsertAdjacentHTML error = %v", err)
	}
	if got, err := harness.TextContent("#bio"); err != nil {
		t.Fatalf("TextContent(#bio) after InsertAdjacentHTML reset error = %v", err)
	} else if got != "Fresh!" {
		t.Fatalf("TextContent(#bio) after InsertAdjacentHTML reset = %q, want %q", got, "Fresh!")
	}

	if err := harness.SetInnerHTML("#bio", "Fresh"); err != nil {
		t.Fatalf("SetInnerHTML(#bio) third update error = %v", err)
	}
	if err := harness.InsertAdjacentHTML("#bio", "beforeend", `<span id="bang">!</span>`); err != nil {
		t.Fatalf("InsertAdjacentHTML(#bio,beforeend) second error = %v", err)
	}
	if err := harness.RemoveNode("#bang"); err != nil {
		t.Fatalf("RemoveNode(#bang) error = %v", err)
	}
	if err := harness.Click("#reset"); err != nil {
		t.Fatalf("Click(#reset) after RemoveNode error = %v", err)
	}
	if got, err := harness.TextContent("#bio"); err != nil {
		t.Fatalf("TextContent(#bio) after RemoveNode reset error = %v", err)
	} else if got != "Fresh" {
		t.Fatalf("TextContent(#bio) after RemoveNode reset = %q, want %q", got, "Fresh")
	}
}

func TestHarnessInlineScriptsSupportClassicJSMemberCalls(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script id="boot">host.setInnerHTML("#out", host.documentCurrentScript())</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, want := harness.Debug().DumpDOM(), `<main><div id="out"><script id="boot">host.setInnerHTML("#out", host.documentCurrentScript())</script></div><script id="boot">host.setInnerHTML("#out", host.documentCurrentScript())</script></main>`; got != want {
		t.Fatalf("Debug().DumpDOM() after classic JS inline script = %q, want %q", got, want)
	}
	if got, want := harness.Debug().LastInlineScriptHTML(), `<script id="boot">host.setInnerHTML("#out", host.documentCurrentScript())</script>`; got != want {
		t.Fatalf("Debug().LastInlineScriptHTML() = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNullishCoalescing(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">seed</div><script>host.setTextContent("#out", "kept" ?? host.setTextContent("#side", "changed"))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after nullish coalescing error = %v", err)
	} else if want := "kept"; got != want {
		t.Fatalf("TextContent(#out) after nullish coalescing = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after nullish coalescing error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#side) after nullish coalescing = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNullishCoalescingOnNonScalarValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><div id=\"side\">seed</div><script>let obj = { value: \"kept\" }; let arr = [\"seed\"]; let text = \"go\"; host.setTextContent(\"#out\", `" + "${(obj ?? host.setTextContent(\"#side\", \"changed\")).value}|${(arr ?? host.setTextContent(\"#side\", \"changed\"))[0]}|${text ?? host.setTextContent(\"#side\", \"changed\")}|${null ?? \"fallback\"}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after nullish coalescing on non-scalar values error = %v", err)
	} else if want := "kept|seed|go|fallback"; got != want {
		t.Fatalf("TextContent(#out) after nullish coalescing on non-scalar values = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after nullish coalescing on non-scalar values error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#side) after nullish coalescing on non-scalar values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportOptionalChaining(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">seed</div><script>host.setTextContent("#out", null?.setTextContent("#side", "changed") ?? "fallback")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after optional chaining error = %v", err)
	} else if want := "fallback"; got != want {
		t.Fatalf("TextContent(#out) after optional chaining = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after optional chaining error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#side) after optional chaining = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectPropertyAccessAndOptionalChaining(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><div id=\"side\">seed</div><div id=\"tail\">seed</div><script>let payload = { title: \"ready\", nested: { value: \"changed\" }, items: [1, 2, 3] }; host.setTextContent(\"#out\", payload?.nested?.value); host.setTextContent(\"#side\", `${payload?.items?.length}`); host.setTextContent(\"#tail\", payload?.missing?.value ?? \"fallback\")</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object property access error = %v", err)
	} else if want := "changed"; got != want {
		t.Fatalf("TextContent(#out) after object property access = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after array length access error = %v", err)
	} else if want := "3"; got != want {
		t.Fatalf("TextContent(#side) after array length access = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#tail"); err != nil {
		t.Fatalf("TextContent(#tail) after optional property access error = %v", err)
	} else if want := "fallback"; got != want {
		t.Fatalf("TextContent(#tail) after optional property access = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportOptionalBracketAccessAndOptionalCalls(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><div id=\"side\">seed</div><div id=\"tail\">seed</div><script>let payload = { nested: { value: \"changed\" }, items: [1, 2, 3] }; let ops = { write: x => x }; host?.[\"setTextContent\"](\"#out\", payload?.[\"nested\"]?.[\"value\"]); host?.[\"setTextContent\"](\"#side\", `${payload?.items?.[1]}`); host?.[\"setTextContent\"](\"#tail\", ops.write?.(\"fresh\"))</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after optional bracket access error = %v", err)
	} else if want := "changed"; got != want {
		t.Fatalf("TextContent(#out) after optional bracket access = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after optional array access error = %v", err)
	} else if want := "2"; got != want {
		t.Fatalf("TextContent(#side) after optional array access = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#tail"); err != nil {
		t.Fatalf("TextContent(#tail) after optional call error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#tail) after optional call = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBracketAccessOnStringValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let text = \"go\"; host.setTextContent(\"#out\", `${text[0]}|${text[1]}|${text[\"length\"]}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after string bracket access error = %v", err)
	} else if want := "g|o|2"; got != want {
		t.Fatalf("TextContent(#out) after string bracket access = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportMemberAccessOnPrimitiveStringAndArrayValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let num = 1; let bool = false; let big = 1n; let text = \"go\"; let arr = [1, 2]; host.setTextContent(\"#out\", `${num.foo}|${bool.foo}|${big.foo}|${text.foo}|${arr.foo}|${text.length}|${arr.length}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after member access on string and array values error = %v", err)
	} else if want := "undefined|undefined|undefined|undefined|undefined|2|2"; got != want {
		t.Fatalf("TextContent(#out) after member access on primitive, string, and array values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeleteExpressionsOnPrimitiveAndArrayValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let num = 1; let bool = false; let big = 1n; let arr = [1, 2]; let deletes = `${delete num.foo}|${delete bool.foo}|${delete big.foo}|${delete num?.foo}|${delete arr.foo}`; host.setTextContent(\"#out\", deletes)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after delete on primitive and array values error = %v", err)
	} else if want := "true|true|true|true|true"; got != want {
		t.Fatalf("TextContent(#out) after delete on primitive and array values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBigIntLiterals(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", -123n)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after BigInt literal error = %v", err)
	} else if want := "-123"; got != want {
		t.Fatalf("TextContent(#out) after BigInt literal = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportTemplateLiteralInterpolation(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let name = \"world\"; host.setTextContent(\"#out\", `hello ${name}!`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after template interpolation error = %v", err)
	} else if want := "hello world!"; got != want {
		t.Fatalf("TextContent(#out) after template interpolation = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportTaggedTemplateLiterals(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function tag(strings, left, right) { return strings[0] + left + strings[1] + right + strings[2] }; host.setTextContent(\"#out\", tag`hello ${\"world\"} ${1 + 1}!`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after tagged template literal error = %v", err)
	} else if want := "hello world 2!"; got != want {
		t.Fatalf("TextContent(#out) after tagged template literal = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsReportRuntimeErrorForNonCallableTaggedTemplates(t *testing.T) {
	harness, err := FromHTML("<main><script>(1)`hello`</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want runtime error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "cannot call ") || !strings.Contains(got, "bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want runtime call error text", got)
	}
}

func TestHarnessInlineScriptsReportRuntimeErrorForNonCallableCallExpressions(t *testing.T) {
	harness, err := FromHTML("<main><script>({})()</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want runtime error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "cannot call ") || !strings.Contains(got, "bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want runtime call error text", got)
	}
}

func TestHarnessInlineScriptsSupportCommaOperatorSequenceExpressions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let value = \"left\"; host.setTextContent(\"#out\", (value, \"right\"))</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after comma operator sequence expression error = %v", err)
	} else if want := "right"; got != want {
		t.Fatalf("TextContent(#out) after comma operator sequence expression = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportArrayAndObjectLiterals(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", ["a", null, {kind: "box"}])</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after array/object literals error = %v", err)
	} else if want := `a,,[object Object]`; got != want {
		t.Fatalf("TextContent(#out) after array/object literals = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportArrayLiteralElisions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let array = [1, , \"two\", , null]; host.setTextContent(\"#out\", `" + "${array.length}-${typeof array[1]}-${typeof array[3]}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after array literal elisions error = %v", err)
	} else if want := "5-undefined-undefined"; got != want {
		t.Fatalf("TextContent(#out) after array literal elisions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDestructuringPatterns(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let [first, , third] = [1, 2, 3]; let {kind: label} = {kind: \"box\"}; host.setTextContent(\"#out\", `${first}-${third}-${label}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after destructuring error = %v", err)
	} else if want := "1-3-box"; got != want {
		t.Fatalf("TextContent(#out) after destructuring = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportCatchBindingPatterns(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>try { throw {kind: \"box\", count: 2} } catch ({kind, count}) { host.setTextContent(\"#out\", `${kind}-${count}`) }</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after catch binding patterns error = %v", err)
	} else if want := "box-2"; got != want {
		t.Fatalf("TextContent(#out) after catch binding patterns = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportComputedObjectDestructuringKeys(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let key = \"kind\"; let { [key]: label, [\"count\"]: total } = {kind: \"box\", count: 2}; host.setTextContent(\"#out\", `" + "${label}-${total}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after computed object destructuring keys error = %v", err)
	} else if want := "box-2"; got != want {
		t.Fatalf("TextContent(#out) after computed object destructuring keys = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDestructuringDefaults(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let [first = \"fallback\", second = first] = []; const {kind = \"box\", label: alias = kind} = {}; host.setTextContent(\"#out\", `${first}-${second}-${kind}-${alias}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after destructuring defaults error = %v", err)
	} else if want := "fallback-fallback-box-box"; got != want {
		t.Fatalf("TextContent(#out) after destructuring defaults = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportVarDeclarations(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>var value = 1; var value = value + 1; host.setTextContent(\"#out\", `${value}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after var declarations error = %v", err)
	} else if want := "2"; got != want {
		t.Fatalf("TextContent(#out) after var declarations = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportUsingDeclarations(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>using value = \"seed\"; await using awaited = \"tail\"; host.setTextContent(\"#out\", `" + "${value}-${awaited}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after using declarations error = %v", err)
	} else if want := "seed-tail"; got != want {
		t.Fatalf("TextContent(#out) after using declarations = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportUsingDeclarationsInForHeaders(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>for (using value of [\"seed\"]) { host.setTextContent(\"#out\", value); break }</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after using declarations in for headers error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#out) after using declarations in for headers = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAwaitUsingDeclarationsInForAwaitHeaders(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>for await (await using value of [\"seed\"]) { host.setTextContent(\"#out\", value); break }</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after await using declarations in for await headers error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#out) after await using declarations in for await headers = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNonDecimalNumericLiterals(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>host.setTextContent(\"#out\", `" + "${0x10}|${0b1010}|${0o77}|${0x1_0n}|${0b10_10n}|${0o7_7n}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after non-decimal numeric literals error = %v", err)
	} else if want := "16|10|63|16|10|63"; got != want {
		t.Fatalf("TextContent(#out) after non-decimal numeric literals = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportUnaryPlusAndMinus(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>host.setTextContent(\"#out\", `" + "${+\"0x10\"}|${-\"2\"}|${+true}|${-false}|${-1n}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after unary plus and minus error = %v", err)
	} else if want := "16|-2|1|0|-1"; got != want {
		t.Fatalf("TextContent(#out) after unary plus and minus = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportVoidOperator(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>host.setTextContent(\"#out\", `" + "${void \"seed\"}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after void operator error = %v", err)
	} else if want := "undefined"; got != want {
		t.Fatalf("TextContent(#out) after void operator = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportLogicalNegationAcrossBoundedValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>host.setTextContent(\"#out\", `" + "${!{}}|${![]}|${!host}|${!null}|${!undefined}|${!\"\"}|${!0}|${!1}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after logical negation across bounded values error = %v", err)
	} else if want := "false|false|false|true|true|true|true|false"; got != want {
		t.Fatalf("TextContent(#out) after logical negation across bounded values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportThisExpression(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>host.setTextContent(\"#out\", `" + "${this}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after this expression error = %v", err)
	} else if want := "undefined"; got != want {
		t.Fatalf("TextContent(#out) after this expression = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSpreadRestSyntax(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let more = [2, 3]; let extra = {kind: \"box\"}; let [first, ...rest] = [1, ...more, 4]; let {kind, ...others} = {...extra, count: 2}; let {count} = others; host.setTextContent(\"#out\", `${first}-${rest}-${kind}-${count}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after spread/rest error = %v", err)
	} else if want := "1-2,3,4-box-2"; got != want {
		t.Fatalf("TextContent(#out) after spread/rest = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportArraySpreadAndDestructuringFromIteratorLikeValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function* values() { yield 1; yield 2; yield 3 }; let [first, ...rest] = values(); let spread = [...values()]; host.setTextContent(\"#out\", `" + "${first}-${rest}-${spread}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after array spread/destructuring on iterator-like object error = %v", err)
	} else if want := "1-2,3-1,2,3"; got != want {
		t.Fatalf("TextContent(#out) after array spread/destructuring on iterator-like object = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportArraySpreadAndDestructuringFromStringValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let text = \"go\"; let [first, ...rest] = text; let spread = [...text]; host.setTextContent(\"#out\", `" + "${first}-${rest}-${spread}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after array spread/destructuring on string value error = %v", err)
	} else if want := "g-o-g,o"; got != want {
		t.Fatalf("TextContent(#out) after array spread/destructuring on string value = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectArraySpreadOnScalarValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let spread = [1, ...1]; host.setTextContent(\"#out\", `" + "${spread}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "array spread requires a string, array, or iterator-like object value in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want array spread runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportObjectSpreadFromStringAndArrayValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let text = \"go\"; let array = [1, 2]; let spreadText = {...text}; let spreadArray = {...array}; host.setTextContent(\"#out\", `" + "${spreadText[\"0\"]}${spreadText[\"1\"]}-${spreadArray[\"0\"]}-${spreadArray[\"1\"]}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object spread on string and array values error = %v", err)
	} else if want := "go-1-2"; got != want {
		t.Fatalf("TextContent(#out) after object spread on string and array values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectSpreadOnPrimitiveValuesAsNoOp(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let spread = { seed: "ok", ...1, ...false, ...1n }; host.setTextContent("#out", ` + "`" + `${spread.seed}-${typeof spread["0"]}-${typeof spread["1"]}-${typeof spread["2"]}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object spread on primitive values error = %v", err)
	} else if want := "ok-undefined-undefined-undefined"; got != want {
		t.Fatalf("TextContent(#out) after object spread on primitive values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectSpreadOnNullishValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let spreadNull = {...null}; let spreadUndefined = {...undefined}; host.setTextContent("#out", ` + "`" + `${spreadNull["0"]}-${spreadUndefined["0"]}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object spread on nullish values error = %v", err)
	} else if want := "undefined-undefined"; got != want {
		t.Fatalf("TextContent(#out) after object spread on nullish values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportCallArgumentSpreadOnIteratorLikeValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function write(first, second) { host.setTextContent(\"#out\", `" + "${first}-${second}" + "` ) }; function* values() { yield \"left\"; yield \"right\" }; write(...values())</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after call argument spread error = %v", err)
	} else if want := "left-right"; got != want {
		t.Fatalf("TextContent(#out) after call argument spread = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportArrowFunctions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let write = (value) => { host.setTextContent("#out", value) }; write("fresh")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after arrow functions error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after arrow functions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNewTargetInFunctionBodiesAndArrows(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>function outer() { let inspect = function () { return typeof new.target }; let capture = () => typeof new.target; host.setTextContent("#out", ` + "`" + `${typeof new.target}:${inspect()}-${capture()}` + "`" + `) }; new outer()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after new.target error = %v", err)
	} else if want := "function:undefined-function"; got != want {
		t.Fatalf("TextContent(#out) after new.target = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsReportParseErrorForNewTargetOutsideFunctionBodies(t *testing.T) {
	harness, err := FromHTML(`<main><script>typeof new.target</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "new.target is only supported inside bounded function or constructor bodies") {
		t.Fatalf("Debug().DOMError() = %q, want new.target parse error text", got)
	}
}

func TestHarnessInlineScriptsReportParseErrorForSuperOutsideMethods(t *testing.T) {
	harness, err := FromHTML(`<main><script>let obj = { read: function() { return super.toString() } }; obj.read()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "`super` is only supported inside bounded class and object literal methods in this slice") {
		t.Fatalf("Debug().DOMError() = %q, want super parse error text", got)
	}
}

func TestHarnessInlineScriptsSupportAsyncArrowFunctionsAndAwait(t *testing.T) {
	harness, err := FromHTML(`<main><div id="source">ready</div><div id="out">old</div><script>let read = async () => host?.["textContent"]("#source"); let write = async () => host?.["setTextContent"]("#out", await read()); write()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async arrow functions error = %v", err)
	} else if want := "ready"; got != want {
		t.Fatalf("TextContent(#out) after async arrow functions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAsyncFunctionExpressionsAndAwait(t *testing.T) {
	harness, err := FromHTML(`<main><div id="source">ready</div><div id="out">old</div><script>let read = async function () { return await host?.["textContent"]("#source") }; host.setTextContent("#out", await read())</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async function expressions error = %v", err)
	} else if want := "ready"; got != want {
		t.Fatalf("TextContent(#out) after async function expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAwaitAcrossBoundedValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>async function read() { let obj = { value: \"kept\" }; let arr = [\"seed\"]; let text = \"go\"; host.setTextContent(\"#out\", `" + "${(await obj).value}|${(await arr)[0]}|${await text}|${await null}" + "` ) }; read()</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after await across bounded values error = %v", err)
	} else if want := "kept|seed|go|null"; got != want {
		t.Fatalf("TextContent(#out) after await across bounded values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAsyncClassMethodsAndAwait(t *testing.T) {
	harness, err := FromHTML(`<main><div id="source">ready</div><div id="out">old</div><script>class Example { async read() { return await host?.["textContent"]("#source") } }; let example = new Example(); host.setTextContent("#out", await example.read())</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async class methods error = %v", err)
	} else if want := "ready"; got != want {
		t.Fatalf("TextContent(#out) after async class methods = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAsyncGeneratorFunctionsAndClassMethods(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"source\">ready</div><div id=\"out\">old</div><script>async function* readFunction() { yield await host?.[\"textContent\"](\"#source\"); yield await host?.[\"textContent\"](\"#source\") }; class Example { static async *readStatic() { yield await host?.[\"textContent\"](\"#source\") } async *readInstance() { yield await host?.[\"textContent\"](\"#source\") } }; let fn = readFunction(); let fnFirst = await fn.next(); let fnSecond = await fn.next(); let staticFirst = await Example.readStatic().next(); let instanceFirst = await (new Example()).readInstance().next(); host.setTextContent(\"#out\", `" + "${fnFirst.value}|${fnSecond.value}|${staticFirst.value}|${instanceFirst.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async generator functions and class methods error = %v", err)
	} else if want := "ready|ready|ready|ready"; got != want {
		t.Fatalf("TextContent(#out) after async generator functions and class methods = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAsyncGeneratorYieldDelegation(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>async function* read() { yield \"first\"; yield* [await host?.[\"textContent\"](\"#out\")]; yield \"third\" }; let it = read(); let first = await it.next(); let second = await it.next(); let third = await it.next(); let done = await it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async generator yield delegation error = %v", err)
	} else if want := "first|old|third|true"; got != want {
		t.Fatalf("TextContent(#out) after async generator yield delegation = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAsyncGeneratorYieldDelegationOnString(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>async function* read() { yield* \"go\"; yield \"done\" }; let it = read(); let first = await it.next(); let second = await it.next(); let third = await it.next(); let done = await it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async generator yield delegation on string error = %v", err)
	} else if want := "g|o|done|true"; got != want {
		t.Fatalf("TextContent(#out) after async generator yield delegation on string = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAsyncGeneratorYieldDelegationOnAsyncIteratorLikeObject(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>async function* read() { yield* { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: \"go\", done: false } }; return { done: true } } }; yield \"done\" }; let it = read(); let first = await it.next(); let second = await it.next(); let third = await it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${second.value}|${third.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after async generator yield delegation on async iterator-like object error = %v", err)
	} else if want := "go|done|true"; got != want {
		t.Fatalf("TextContent(#out) after async generator yield delegation on async iterator-like object = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorYieldDelegationImmediateIteratorReturnValue(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function* read() { let result = yield* { next() { return { value: \"finished\", done: true } } }; yield result }; let it = read(); let first = it.next(); let done = it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${done.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator yield delegation immediate iterator return value error = %v", err)
	} else if want := "finished|true"; got != want {
		t.Fatalf("TextContent(#out) after generator yield delegation immediate iterator return value = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNestedAsyncGeneratorYieldDelegationOnString(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>async function* read() { if (true) { yield* \"go\"; }; yield \"done\" }; let it = read(); let first = await it.next(); let second = await it.next(); let third = await it.next(); let done = await it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after nested async generator yield delegation on string error = %v", err)
	} else if want := "g|o|done|true"; got != want {
		t.Fatalf("TextContent(#out) after nested async generator yield delegation on string = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorYieldDelegationOnString(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function* read() { yield* \"go\"; yield \"done\" }; let it = read(); let first = it.next(); let second = it.next(); let third = it.next(); let done = it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator yield delegation on string error = %v", err)
	} else if want := "g|o|done|true"; got != want {
		t.Fatalf("TextContent(#out) after generator yield delegation on string = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNestedGeneratorYieldDelegationOnString(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function* read() { if (true) { yield* \"go\"; }; yield \"done\" }; let it = read(); let first = it.next(); let second = it.next(); let third = it.next(); let done = it.next(); host.setTextContent(\"#out\", `" + "${first.value}|${second.value}|${third.value}|${done.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after nested generator yield delegation on string error = %v", err)
	} else if want := "g|o|done|true"; got != want {
		t.Fatalf("TextContent(#out) after nested generator yield delegation on string = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorClassMethodsAndYield(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>class Example { *read() { yield \"fresh\" } }; let example = new Example(); let it = example.read(); host.setTextContent(\"#out\", `" + "${it.next().value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator class methods error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after generator class methods = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportFunctionDeclarationsAndReturnStatements(t *testing.T) {
	harness, err := FromHTML(`<main><div id="source">fresh</div><div id="out">old</div><script>function choose(flag) { if (flag) { return host?.["textContent"]("#source") } return "fallback" }; host.setTextContent("#out", choose(true))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after function declarations error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after function declarations = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorFunctionExpressions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let sync = function* () { yield \"fresh\" }; let it = sync(); host.setTextContent(\"#out\", `" + "${it.next().value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator function expressions error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after generator function expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectLiteralShorthandPropertiesAndMethods(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let item = { kind: \"box\" }; let list = [1, 2]; let obj = { item, list, read() { return this.item.kind + \"-\" + this.list[1] } }; host.setTextContent(\"#out\", `" + "${obj.item.kind}-${obj.list[1]}-${obj.read()}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object literal shorthand error = %v", err)
	} else if want := "box-2-box-2"; got != want {
		t.Fatalf("TextContent(#out) after object literal shorthand = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsReportParseErrorForMalformedObjectLiteralShorthandSequences(t *testing.T) {
	harness, err := FromHTML(`<main><script>let value = "seed"; let obj = { value other }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "object literals must separate properties with commas") {
		t.Fatalf("Debug().DOMError() = %q, want malformed object literal shorthand parse error text", got)
	}
}

func TestHarnessInlineScriptsSupportComputedObjectLiteralPropertiesAndMethods(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let key = \"value\"; let method = \"read\"; let value = \"seed\"; let obj = { [key]: value, [method]() { return this.value } }; host.setTextContent(\"#out\", `" + "${obj.value}-${obj.read()}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after computed object literal properties error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after computed object literal properties = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectLiteralAsyncAndGeneratorMethods(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { async: \"seed\", async read() { return await this.async }, *spin() { yield this.async }, async *drift() { yield await this.async; yield this.async } }; let asyncValue = await obj.read(); let syncValue = obj.spin().next().value; let asyncIt = obj.drift(); let asyncFirst = await asyncIt.next(); let asyncSecond = await asyncIt.next(); let asyncThird = await asyncIt.next(); host.setTextContent(\"#out\", `" + "${asyncValue}|${syncValue}|${asyncFirst.value}|${asyncSecond.value}|${asyncThird.done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object literal async and generator methods error = %v", err)
	} else if want := "seed|seed|seed|seed|true"; got != want {
		t.Fatalf("TextContent(#out) after object literal async and generator methods = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectLiteralGetterAccessors(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let label = \"score\"; let obj = { get value() { return \"seed\" }, get [label]() { return this.value } }; host.setTextContent(\"#out\", `" + "${obj.value}-${obj.score}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object literal getter accessors error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after object literal getter accessors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectLiteralSuperMethods(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let proto = { read() { return \"proto\" }, label: \"base\" }; let obj = { __proto__: proto, read() { return super.read() + \"-child\" }, get label() { return super.label } }; let plain = { read() { return super.toString() } }; host.setTextContent(\"#out\", `" + "${obj.read()}|${obj.label}|${plain.read()}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object literal super methods error = %v", err)
	} else if want := "proto-child|base|[object Object]"; got != want {
		t.Fatalf("TextContent(#out) after object literal super methods = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNullPrototypeObjectLiteralSuperAccess(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { __proto__: null, read() { return super.label } }; host.setTextContent(\"#out\", `" + "${obj.read()}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after null-prototype object literal super access error = %v", err)
	} else if want := "undefined"; got != want {
		t.Fatalf("TextContent(#out) after null-prototype object literal super access = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNullPrototypeObjectLiteralSuperAssignment(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { __proto__: null, value: \"seed\", write(v) { super.value = v; return this.value }, create(v) { super.extra = v; return this.extra } }; host.setTextContent(\"#out\", `" + "${obj.write(\"updated\")}|${obj.create(\"fresh\")}|${obj.value}|${obj.extra}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after null-prototype object literal super assignment error = %v", err)
	} else if want := "updated|fresh|updated|fresh"; got != want {
		t.Fatalf("TextContent(#out) after null-prototype object literal super assignment = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNullPrototypeObjectLiteralSuperCompoundAssignment(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { __proto__: null, value: \"seed\", write(v) { super.value += v; return this.value } }; host.setTextContent(\"#out\", `" + "${obj.write(\"-updated\")}|${obj.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after null-prototype object literal super compound assignment error = %v", err)
	} else if want := "seed-updated|seed-updated"; got != want {
		t.Fatalf("TextContent(#out) after null-prototype object literal super compound assignment = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsReportRuntimeErrorForNullPrototypeObjectLiteralSuperCalls(t *testing.T) {
	harness, err := FromHTML(`<main><script>let obj = { __proto__: null, read() { return super.toString() } }; obj.read()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want runtime error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "cannot call ") || !strings.Contains(got, "bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want runtime call error text", got)
	}
}

func TestHarnessInlineScriptsSupportDeleteExpressionsOnSuperTargets(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>class Base { static value = \"seed\" }; class Derived extends Base { static value = \"derived\"; static zap() { return delete super.value }; static read() { return super.value } }; let obj = { __proto__: null, value: \"seed\", zap() { return delete super.value }, read() { return this.value } }; host.setTextContent(\"#out\", `" + "${Derived.zap()}|${Derived.read()}|${Base.value}|${Derived.value}|${obj.zap()}|${obj.read()}|${obj.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after delete expressions on super targets error = %v", err)
	} else if want := "true|seed|seed|seed|true|undefined|undefined"; got != want {
		t.Fatalf("TextContent(#out) after delete expressions on super targets = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectLiteralSetterAccessors(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { _value: \"seed\", get value() { return this._value }, set value(v) { this._value = v } }; obj.value = \"updated\"; host.setTextContent(\"#out\", `" + "${obj.value}|${obj._value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after object literal setter accessors error = %v", err)
	} else if want := "updated|updated"; got != want {
		t.Fatalf("TextContent(#out) after object literal setter accessors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeletingObjectLiteralSetterAccessors(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { _value: \"seed\", get value() { return this._value }, set value(v) { this._value = v } }; delete obj.value; host.setTextContent(\"#out\", `" + "${obj.value}|${obj._value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after deleting object literal setter accessors error = %v", err)
	} else if want := "undefined|seed"; got != want {
		t.Fatalf("TextContent(#out) after deleting object literal setter accessors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportObjectPropertyAssignmentAndPrivateFieldMutation(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { value: \"seed\", nested: { count: 1 } }; class Counter { #value = 1; constructor() { this.#value = this.#value + 1 } inc() { this.#value = this.#value + 1 } read() { return this.#value } }; let counter = new Counter(); counter.inc(); obj.value = \"updated\"; obj.nested.count = obj.nested.count + 1; host.setTextContent(\"#out\", `" + "${obj.value}-${obj.nested.count}-${counter.read()}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after property assignment error = %v", err)
	} else if want := "updated-2-3"; got != want {
		t.Fatalf("TextContent(#out) after property assignment = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsReportRuntimeErrorForGetterOnlyObjectPropertyAssignments(t *testing.T) {
	harness, err := FromHTML(`<main><script>let obj = { get value() { return "seed" } }; obj.value = "updated"</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want runtime error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "getter-only property") {
		t.Fatalf("Debug().DOMError() = %q, want getter-only property runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportArrayPropertyAssignment(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let arr = [1, 2]; arr[0]++; ++arr[1]; arr[2] = 5; arr.length = 2; host.setTextContent(\"#out\", `" + "${arr[0]}|${arr[1]}|${arr.length}|${arr[2]}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after array property assignment error = %v", err)
	} else if want := "2|3|2|undefined"; got != want {
		t.Fatalf("TextContent(#out) after array property assignment = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportPrivateInOperator(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>class Example { #secret = 1; has(other) { return #secret in other } }; let example = new Example(); host.setTextContent(\"#out\", example.has(example) + \"-\" + example.has({}))</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after private `in` operator error = %v", err)
	} else if want := "true-false"; got != want {
		t.Fatalf("TextContent(#out) after private `in` operator = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeleteExpressions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { nested: { value: \"seed\" } }; host.setTextContent(\"#out\", `" + "${delete obj}-${delete obj.nested.value}-${obj.nested.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after delete expressions error = %v", err)
	} else if want := "false-true-undefined"; got != want {
		t.Fatalf("TextContent(#out) after delete expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeleteExpressionsOnArrays(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let arr = [1, 2, { value: \"seed\" }]; delete arr[1]; delete arr[2].value; host.setTextContent(\"#out\", `" + "${arr[0]}|${arr[1]}|${arr[2].value}|${arr.length}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after delete expressions on arrays error = %v", err)
	} else if want := "1|undefined|undefined|3"; got != want {
		t.Fatalf("TextContent(#out) after delete expressions on arrays = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeletingArrayLength(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let arr = [1, 2]; host.setTextContent(\"#out\", `" + "${delete arr.length}|${arr.length}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after deleting array length error = %v", err)
	} else if want := "false|2"; got != want {
		t.Fatalf("TextContent(#out) after deleting array length = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeletingStringProperties(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let text = \"go\"; host.setTextContent(\"#out\", `" + "${delete text[0]}|${delete text[\"length\"]}|${delete text.foo}|${text[0]}|${text[\"length\"]}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after deleting string properties error = %v", err)
	} else if want := "false|false|true|g|2"; got != want {
		t.Fatalf("TextContent(#out) after deleting string properties = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDeleteExpressionsWithOptionalChaining(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let maybe = null; let obj = { nested: { value: \"seed\" } }; host.setTextContent(\"#out\", `" + "${delete maybe?.value}|${delete obj?.nested?.value}|${obj.nested.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after delete expressions with optional chaining error = %v", err)
	} else if want := "true|true|undefined"; got != want {
		t.Fatalf("TextContent(#out) after delete expressions with optional chaining = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportWithStatements(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let obj = { value: \"seed\", count: 1 }; with (obj) { count++; value = value + \"-\" + count; } host.setTextContent(\"#out\", obj.value)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after with statements error = %v", err)
	} else if want := "seed-2"; got != want {
		t.Fatalf("TextContent(#out) after with statements = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBreakAndContinueAcrossLoopSwitchAndTry(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">old</div><div id="tail">old</div><script>let run = true; let first = true; let branch = true; while (run ?? false) { try { switch (branch) { case true: first &&= undefined; branch &&= false; host.setTextContent("#out", "first"); continue; case false: host.setTextContent("#side", "second"); break }; break } finally { host.setTextContent("#tail", "finally") } }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after break/continue error = %v", err)
	} else if want := "first"; got != want {
		t.Fatalf("TextContent(#out) after break/continue = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after break/continue error = %v", err)
	} else if want := "second"; got != want {
		t.Fatalf("TextContent(#side) after break/continue = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#tail"); err != nil {
		t.Fatalf("TextContent(#tail) after break/continue error = %v", err)
	} else if want := "finally"; got != want {
		t.Fatalf("TextContent(#tail) after break/continue = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportLabeledBreakAndContinueAcrossLoopSwitchAndTry(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">old</div><div id="tail">old</div><script>let run = true; let first = true; outer: while (run ?? false) { try { switch (first) { case true: first &&= false; host.setTextContent("#out", "first"); continue outer; case false: host.setTextContent("#side", "second"); break outer } } finally { host.setTextContent("#tail", "finally") } }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after labeled break/continue error = %v", err)
	} else if want := "first"; got != want {
		t.Fatalf("TextContent(#out) after labeled break/continue = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after labeled break/continue error = %v", err)
	} else if want := "second"; got != want {
		t.Fatalf("TextContent(#side) after labeled break/continue = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#tail"); err != nil {
		t.Fatalf("TextContent(#tail) after labeled break/continue error = %v", err)
	} else if want := "finally"; got != want {
		t.Fatalf("TextContent(#tail) after labeled break/continue = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportThrowStatementsWithCatchBindings(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let value = "seed"; try { throw value } catch (error) { host.setTextContent("#out", error) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after throw statements error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#out) after throw statements = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDefaultParameterValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="source">ready</div><div id="out">old</div><script>function choose(value = host?.["textContent"]("#source")) { return value }; host.setTextContent("#out", choose())</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after default parameter values error = %v", err)
	} else if want := "ready"; got != want {
		t.Fatalf("TextContent(#out) after default parameter values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDestructuringFunctionParameters(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let arrow = ([first, second = first], {kind: label = "box"} = {}) => first + "-" + second + "-" + label; class Example { read([first, second = first], {kind: label = "box"} = {}) { return first + "-" + second + "-" + label } }; let example = new Example(); host.setTextContent("#out", arrow([1]) + "|" + example.read([1]))</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after destructuring function parameters error = %v", err)
	} else if want := "1-1-box|1-1-box"; got != want {
		t.Fatalf("TextContent(#out) after destructuring function parameters = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportTopLevelAwait(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>await host.setTextContent("#out", "fresh")</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after top-level await error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after top-level await = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportExportDeclarations(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>export const value = host?.["textContent"]("#out"); export { value as alias }; export default value; host.setTextContent("#out", value)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after export declarations error = %v", err)
	} else if want := "old"; got != want {
		t.Fatalf("TextContent(#out) after export declarations = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDefaultClassExports(t *testing.T) {
	harness, err := FromHTML(`<main><div id="seed">seed</div><div id="out">old</div><script type="module" id="box">export default class Box { static value = host?.["textContent"]("#seed"); };</script><script>import Box from "box"; host.setTextContent("#out", Box.value)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after default class exports error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#out) after default class exports = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDefaultModuleSpecifierAliases(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = host?.[\"textContent\"](\"#seed\"); export default value;</script><script type=\"module\" id=\"reexport\">export { default as mirror } from \"math\" with { type: \"json\" }; export { value as default } from \"math\" with { type: \"json\" };</script><script>import { default as seeded } from \"math\"; import { mirror } from \"reexport\"; host.setTextContent(\"#out\", `" + "${seeded}-${mirror}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after default module specifier aliases error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after default module specifier aliases = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDefaultAndNamespaceImports(t *testing.T) {
	harness, err := FromHTML(`<main><div id="seed">seed</div><div id="out">old</div><script type="module" id="math">export const value = host?.["textContent"]("#seed"); export default value;</script><script>import seeded, * as ns from "math"; host.setTextContent("#out", seeded + "-" + ns.value)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after default and namespace imports error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after default and namespace imports = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDefaultGeneratorFunctionExports(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script type=\"module\" id=\"spin\">export default function* read() { yield \"fresh\"; };</script><script>import generator from \"spin\"; let it = generator(); host.setTextContent(\"#out\", `" + "${it.next().value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after default generator function exports error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after default generator function exports = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportAnonymousDefaultFunctionExports(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script type=\"module\" id=\"spin\">export default function () { return \"fresh\"; };</script><script>import fn from \"spin\"; host.setTextContent(\"#out\", fn())</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after anonymous default function exports error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after anonymous default function exports = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorNextArguments(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { yield \"first\"; yield \"second\" }; let it = make(); host.setTextContent(\"#out\", `" + "${it.next(\"seed\").value}|${it.next(\"ignored\").value}|${it.next().done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator next arguments error = %v", err)
	} else if want := "first|second|true"; got != want {
		t.Fatalf("TextContent(#out) after generator next arguments = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorNextArgumentsOnYieldExpressions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function* syncSpin() { let first = yield \"first\"; let value; value = yield first; let box = { value: \"\" }; box.value = yield value; yield `${value}|${box.value}` }; async function* asyncSpin() { let first = yield \"first\"; let value; value = yield first; let box = { value: \"\" }; box.value = yield value; yield `${value}|${box.value}` }; let syncIt = syncSpin(); let syncOne = syncIt.next(); let syncTwo = syncIt.next(\"seed\"); let syncThree = syncIt.next(\"tail\"); let syncFour = syncIt.next(\"final\"); let syncDone = syncIt.next(); async function run() { let asyncIt = asyncSpin(); let asyncOne = await asyncIt.next(); let asyncTwo = await asyncIt.next(\"seed\"); let asyncThree = await asyncIt.next(\"tail\"); let asyncFour = await asyncIt.next(\"final\"); let asyncDone = await asyncIt.next(); host.setTextContent(\"#out\", `" + "${syncOne.value}|${syncTwo.value}|${syncThree.value}|${syncFour.value}|${syncDone.done}|${asyncOne.value}|${asyncTwo.value}|${asyncThree.value}|${asyncFour.value}|${asyncDone.done}" + "`); }; await run()</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator next arguments on yield expressions error = %v", err)
	} else if want := "first|seed|tail|tail|final|true|first|seed|tail|tail|final|true"; got != want {
		t.Fatalf("TextContent(#out) after generator next arguments on yield expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorNextArgumentsOnYieldStarDelegates(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { yield* { index: 0, next(value) { if (this.index === 0) { this.index = 1; return { value: value === undefined ? \"first\" : value, done: false } }; if (this.index === 1) { this.index = 2; return { value: value === undefined ? \"second\" : value, done: false } }; return { done: true } } }; yield \"done\" }; let it = make(); host.setTextContent(\"#out\", `" + "${it.next().value}|${it.next(\"seed\").value}|${it.next().value}|${it.next().done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator next arguments on yield* delegates error = %v", err)
	} else if want := "first|seed|done|true"; got != want {
		t.Fatalf("TextContent(#out) after generator next arguments on yield* delegates = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectYieldStarOnScalarValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>function* spin() { yield* 1 }; let it = spin(); it.next()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "yield* expects a string, array, or iterator-like object in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want `yield*` runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportGeneratorYieldExpressions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { yield \"first\"; (yield \"second\"); yield \"done\" }; let it = make(); host.setTextContent(\"#out\", `" + "${it.next().value}|${it.next(\"ignored\").value}|${it.next().value}|${it.next().done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator yield expressions error = %v", err)
	} else if want := "first|second|done|true"; got != want {
		t.Fatalf("TextContent(#out) after generator yield expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorReturnArguments(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { yield \"first\"; yield \"second\" }; let it = make(); let first = it.next(); let stopped = it.return(\"done\"); host.setTextContent(\"#out\", `" + "${first.value}|${stopped.value}|${stopped.done}|${it.next().done}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator return arguments error = %v", err)
	} else if want := "first|done|true|true"; got != want {
		t.Fatalf("TextContent(#out) after generator return arguments = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorThrowWithoutArguments(t *testing.T) {
	harness, err := FromHTML("<main><script>function* spin() { yield \"first\" }; spin().throw()</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want runtime error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "undefined") {
		t.Fatalf("Debug().DOMError() = %q, want undefined runtime error text", got)
	}
}

func TestHarnessInlineScriptsReportParseErrorForTopLevelReturnStatements(t *testing.T) {
	harness, err := FromHTML(`<main><script>return "boom"</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "return statements are only supported inside bounded function bodies") {
		t.Fatalf("Debug().DOMError() = %q, want return parse error text", got)
	}
}

func TestHarnessInlineModuleScriptsReportParseErrorForInvalidDefaultExportForms(t *testing.T) {
	harness, err := FromHTML(`<main><script type="module" id="bad">export default const value = 1</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "`export default const` declarations are not supported") {
		t.Fatalf("Debug().DOMError() = %q, want invalid default export parse error text", got)
	}
}

func TestHarnessInlineModuleScriptsReportParseErrorForReservedImportAndExportNames(t *testing.T) {
	for _, tc := range []struct {
		name string
		html string
		want string
	}{
		{
			name: "import",
			html: `<main><script type="module" id="math">export default 1</script><script type="module" id="bad">import { this as value } from "math"</script></main>`,
			want: "reserved import specifier name",
		},
		{
			name: "export",
			html: `<main><script type="module" id="bad">export { this as value }</script></main>`,
			want: "reserved export specifier name",
		},
	} {
		harness, err := FromHTML(tc.html)
		if err != nil {
			t.Fatalf("FromHTML() error = %v", err)
		}

		if _, err := harness.TextContent("main"); err == nil {
			t.Fatalf("TextContent(main) error = nil, want parse error (%s)", tc.name)
		} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
			t.Fatalf("TextContent(main) error = %#v, want DOM error (%s)", err, tc.name)
		}

		if got := harness.Debug().DOMError(); got == "" {
			t.Fatalf("Debug().DOMError() = %q, want parse error text (%s)", got, tc.name)
		} else if !strings.Contains(got, tc.want) {
			t.Fatalf("Debug().DOMError() = %q, want %q (%s)", got, tc.want, tc.name)
		}
	}
}

func TestHarnessInlineScriptsReportParseErrorForReservedBindingNames(t *testing.T) {
	harness, err := FromHTML(`<main><script>let { this } = { this: 1 }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "reserved lexical binding name") {
		t.Fatalf("Debug().DOMError() = %q, want reserved binding parse error text", got)
	}
}

func TestHarnessInlineScriptsReportParseErrorForPrivateNamesInObjectLiteralMethods(t *testing.T) {
	harness, err := FromHTML(`<main><script>let obj = { async #read() { return 1 } }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "object literal methods do not support private names") {
		t.Fatalf("Debug().DOMError() = %q, want private-name parse error text", got)
	}
}

func TestHarnessInlineScriptsReportParseErrorForMalformedClassMemberSequences(t *testing.T) {
	for _, tc := range []struct {
		name string
		html string
		want string
	}{
		{
			name: "instance",
			html: `<main><script>class Example { foo bar }</script></main>`,
			want: "unexpected class body element",
		},
		{
			name: "static",
			html: `<main><script>class Example { static foo bar }</script></main>`,
			want: "unexpected class member syntax",
		},
	} {
		harness, err := FromHTML(tc.html)
		if err != nil {
			t.Fatalf("FromHTML() error = %v", err)
		}

		if _, err := harness.TextContent("main"); err == nil {
			t.Fatalf("TextContent(main) error = nil, want parse error (%s)", tc.name)
		} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
			t.Fatalf("TextContent(main) error = %#v, want DOM error (%s)", err, tc.name)
		}

		if got := harness.Debug().DOMError(); got == "" {
			t.Fatalf("Debug().DOMError() = %q, want parse error text (%s)", got, tc.name)
		} else if !strings.Contains(got, tc.want) {
			t.Fatalf("Debug().DOMError() = %q, want %q (%s)", got, tc.want, tc.name)
		}
	}
}

func TestHarnessInlineScriptsReportParseErrorForUnsupportedParameterSyntax(t *testing.T) {
	harness, err := FromHTML(`<main><script>function broken(value: label) {}</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want parse error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want parse error text", got)
	} else if !strings.Contains(got, "function parameter list must separate parameters with commas") {
		t.Fatalf("Debug().DOMError() = %q, want unsupported parameter parse error text", got)
	}
}

func TestHarnessInlineScriptsReportRuntimeErrorForSuperCallInBaseClassConstructor(t *testing.T) {
	harness, err := FromHTML(`<main><script>class Example { constructor() { super() } }; new Example()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want runtime error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "requires a constructor on the base target") {
		t.Fatalf("Debug().DOMError() = %q, want base-target runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportAnonymousDefaultAsyncFunctionAndGeneratorExports(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"asyncFn\">export default async function() { return await host?.[\"textContent\"](\"#seed\"); };</script><script type=\"module\" id=\"asyncGen\">export default async function*() { yield await host?.[\"textContent\"](\"#seed\"); };</script><script>import read from \"asyncFn\"; import spin from \"asyncGen\"; let yielded = await spin().next(); host.setTextContent(\"#out\", `" + "${await read()}|${yielded.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after anonymous default async exports error = %v", err)
	} else if want := "seed|seed"; got != want {
		t.Fatalf("TextContent(#out) after anonymous default async exports = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNamespaceReExports(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = host?.[\"textContent\"](\"#seed\"); export default value;</script><script type=\"module\" id=\"reexport\">export * as ns from \"math\" with { type: \"json\" };</script><script>import { ns } from \"reexport\"; host.setTextContent(\"#out\", `" + "${ns.value}-${ns.default}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after namespace re-exports error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after namespace re-exports = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportStarReExportsWithImportAttributes(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = host?.[\"textContent\"](\"#seed\"); export default value;</script><script type=\"module\" id=\"reexport\">export * from \"math\" with { type: \"json\" };</script><script>import { value } from \"reexport\"; host.setTextContent(\"#out\", value)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after star re-exports with attributes error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#out) after star re-exports with attributes = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportDefaultAndNamespaceImportsWithImportAttributes(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = host?.[\"textContent\"](\"#seed\"); export default value;</script><script>import seeded, * as ns from \"math\" with { type: \"json\" }; host.setTextContent(\"#out\", `" + "${seeded}-${ns.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after default + namespace imports with attributes error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after default + namespace imports with attributes = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSpecifierReExportsWithImportAttributes(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = 7; export default host?.[\"textContent\"](\"#seed\");</script><script type=\"module\" id=\"reexport\">export { default as mirror, value as copy } from \"math\" with { type: \"json\" };</script><script>import { mirror, copy } from \"reexport\"; host.setTextContent(\"#out\", `" + "${mirror}-${copy}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after specifier re-exports with attributes error = %v", err)
	} else if want := "seed-7"; got != want {
		t.Fatalf("TextContent(#out) after specifier re-exports with attributes = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportModuleScriptsAndDynamicImport(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = host?.[\"textContent\"](\"#seed\"); export default value;</script><script type=\"module\" id=\"reexport\">export { value as alias } from \"math\";</script><script>import seed, { value as fromage } from \"math\"; let moduleName = \"re\" + \"export\"; let ns = await import(moduleName); host.setTextContent(\"#out\", `${seed}-${fromage}-${ns.alias}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after module scripts and dynamic import error = %v", err)
	} else if want := "seed-seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after module scripts and dynamic import = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBareDynamicImportExpressions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = 7; export default value;</script><script>let moduleName = \"ma\" + \"th\"; import(moduleName); host.setTextContent(\"#out\", \"ok\")</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after bare dynamic import expressions error = %v", err)
	} else if want := "ok"; got != want {
		t.Fatalf("TextContent(#out) after bare dynamic import expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportImportAttributesAndDynamicImportOptions(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"seed\">seed</div><div id=\"out\">old</div><script type=\"module\" id=\"math\">export const value = host?.[\"textContent\"](\"#seed\"); export default value;</script><script type=\"module\" id=\"consumer\">import { default as seeded } from \"math\" with { type: \"json\" }; export const viaImport = seeded; export default seeded;</script><script>let moduleName = \"con\" + \"sumer\"; let ns = await import(moduleName, { with: { type: \"json\" } }); host.setTextContent(\"#out\", `${ns.viaImport}-${ns.default}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after import attributes and dynamic import options error = %v", err)
	} else if want := "seed-seed"; got != want {
		t.Fatalf("TextContent(#out) after import attributes and dynamic import options = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectDynamicImportOptionsNonObject(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>await import("math", 1)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "dynamic import() optional attributes must be an object in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want dynamic import runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportImportMetaUrlInModuleScripts(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script type="module" id="meta">host.setTextContent("#out", import.meta.url)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after import.meta.url in module scripts error = %v", err)
	} else if want := "inline-module:meta"; got != want {
		t.Fatalf("TextContent(#out) after import.meta.url in module scripts = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBareImportMetaInModuleScripts(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script type="module" id="meta">host.setTextContent("#out", typeof import.meta + ":" + import.meta.url)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after bare import.meta in module scripts error = %v", err)
	} else if want := "object:inline-module:meta"; got != want {
		t.Fatalf("TextContent(#out) after bare import.meta in module scripts = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportGeneratorFunctionsAndYield(t *testing.T) {
	harness, err := FromHTML(`<main><div id="first">alpha</div><div id="second">beta</div><div id="out">old</div><script>let gather = function* () { let first = host?.["textContent"]("#first"); yield first; let second = host?.["textContent"]("#second"); yield second; }; let it = gather(); host?.["setTextContent"]("#out", it.next().value); host?.["setTextContent"]("#out", it.next().value)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after generator functions error = %v", err)
	} else if want := "beta"; got != want {
		t.Fatalf("TextContent(#out) after generator functions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNamedGeneratorFunctionsAndSelfBinding(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* spin() { yield spin; yield 1 }; let it = make(); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after named generator functions error = %v", err)
	} else if want := "1"; got != want {
		t.Fatalf("TextContent(#out) after named generator functions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNestedYieldInIfBranches(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { if (true) { if (true) { yield 1 } }; yield 2 }; let it = make(); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after nested yield in if branches error = %v", err)
	} else if want := "2"; got != want {
		t.Fatalf("TextContent(#out) after nested yield in if branches = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportYieldInsideLoopBodies(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { while (true) { yield 1; yield 2 } }; let it = make(); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after yield inside loop bodies error = %v", err)
	} else if want := "2"; got != want {
		t.Fatalf("TextContent(#out) after yield inside loop bodies = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportYieldInsideSwitchClauses(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { switch (\"b\") { case \"a\": yield 1; break; case \"b\": yield 2; yield 3; break; default: yield 4 }; yield 5 }; let it = make(); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after yield inside switch clauses error = %v", err)
	} else if want := "5"; got != want {
		t.Fatalf("TextContent(#out) after yield inside switch clauses = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportYieldInsideTryCatchFinallyBlocks(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let make = function* () { try { yield 1; undefined.foo; yield 2 } catch (err) { yield 3; yield 4 } finally { yield 5; yield 6 } }; let it = make(); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`); host?.[\"setTextContent\"](\"#out\", `${it.next().value}`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after yield inside try/catch/finally blocks error = %v", err)
	} else if want := "6"; got != want {
		t.Fatalf("TextContent(#out) after yield inside try/catch/finally blocks = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportLetConstAndIfElse(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let target = "#out"; const value = "fresh"; if (value) { host.setTextContent(target, value) } else { host.setTextContent(target, "fallback") }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after let/const and if/else error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after let/const and if/else = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportLogicalAssignmentOperators(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let value = null; value ??= "fresh"; host.setTextContent("#out", value)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after logical assignment error = %v", err)
	} else if want := "fresh"; got != want {
		t.Fatalf("TextContent(#out) after logical assignment = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportLogicalOrAndAndOnNonScalarValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">safe</div><script>let obj = { kind: "box" }; let arr = [1, 2]; let text = "seed"; let objectOr = obj || host.setTextContent("#side", "boom"); let arrayAnd = arr && { kind: "fresh" }; let stringOr = text || host.setTextContent("#side", "boom"); host.setTextContent("#out", ` + "`" + `${objectOr.kind}|${arrayAnd.kind}|${stringOr}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after logical or/and on non-scalar values error = %v", err)
	} else if want := "box|fresh|seed"; got != want {
		t.Fatalf("TextContent(#out) after logical or/and on non-scalar values = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after logical or/and on non-scalar values error = %v", err)
	} else if want := "safe"; got != want {
		t.Fatalf("TextContent(#side) after logical or/and on non-scalar values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportLogicalAssignmentOnObjectProperties(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let obj = { _value: "", get value() { return this._value }, set value(next) { this._value = next }, nested: { count: null } }; obj.value ||= "fresh"; obj.nested.count ??= 7; host.setTextContent("#out", ` + "`" + `${obj.value}-${obj._value}-${obj.nested.count}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after logical assignment on object properties error = %v", err)
	} else if want := "fresh-fresh-7"; got != want {
		t.Fatalf("TextContent(#out) after logical assignment on object properties = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportCompoundAssignmentOperators(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let value = 1; value += 2; value *= 3; let obj = { count: 2, nested: { mask: 3 } }; obj.count -= 1; obj.nested.mask <<= 2; host.setTextContent("#out", ` + "`" + `${value}-${obj.count}-${obj.nested.mask}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after compound assignment operators error = %v", err)
	} else if want := "9-1-12"; got != want {
		t.Fatalf("TextContent(#out) after compound assignment operators = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportIncrementAndDecrementExpressions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let value = 1; let obj = { count: 1 }; host.setTextContent("#out", ` + "`" + `${value++}|${value}|${++value}|${obj.count++}|${obj.count}|${++obj["count"]}|${obj.count}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after increment and decrement expressions error = %v", err)
	} else if want := "1|2|3|1|2|3|3"; got != want {
		t.Fatalf("TextContent(#out) after increment and decrement expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportInOperator(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let obj = { set value(next) { this._value = next }, nested: { count: 1 }, items: [1, 2] }; host.setTextContent("#out", ` + "`" + `${"value" in obj}-${"missing" in obj}-${"count" in obj.nested}-${0 in obj.items}-${2 in obj.items}-${"length" in obj.items}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after in operator error = %v", err)
	} else if want := "true-false-true-true-false-true"; got != want {
		t.Fatalf("TextContent(#out) after in operator = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectInOperatorOnNonObjectRHS(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", ` + "`" + `${1 in 2}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "relational `in` requires an object or array value on the right in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want `in` runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportInstanceofOperator(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>class Base {}; class Derived extends Base {}; let base = new Base(); let derived = new Derived(); let plain = {}; host.setTextContent("#out", ` + "`" + `${base instanceof Base}-${derived instanceof Base}-${derived instanceof Derived}-${plain instanceof Base}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after instanceof operator error = %v", err)
	} else if want := "true-true-true-false"; got != want {
		t.Fatalf("TextContent(#out) after instanceof operator = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectInstanceofOnNonClassObjects(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", ` + "`" + `${({}) instanceof ({})}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "relational `instanceof` requires a class object or constructible function value on the right in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want `instanceof` runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportConditionalOperator(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", ` + "`" + `${({} ? "object" : "no")}-${([] ? "array" : "no")}-${(false ? "left" : true ? "middle" : "right")}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after conditional operator error = %v", err)
	} else if want := "object-array-middle"; got != want {
		t.Fatalf("TextContent(#out) after conditional operator = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportEqualityComparisons(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let obj = { value: 1 }; let alias = obj; let arr = [1, 2]; let arrAlias = arr; host.setTextContent("#out", ` + "`" + `${obj === alias}|${obj === { value: 1 }}|${obj == alias}|${obj == { value: 1 }}|${arr === arrAlias}|${arr == arrAlias}|${1 == "1"}|${1 === "1"}|${1 != "1"}|${1 !== "1"}|${null == undefined}|${null === undefined}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after equality comparisons error = %v", err)
	} else if want := "true|false|true|false|true|true|true|false|false|true|true|false"; got != want {
		t.Fatalf("TextContent(#out) after equality comparisons = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportExponentiationOperators(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let value = 2; value **= 3; let big = 2n; big **= 3n; host.setTextContent("#out", ` + "`" + `${value}-${big}-${2 ** 3 ** 2}-${2 ** -1}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after exponentiation operators error = %v", err)
	} else if want := "8-8-512-0.5"; got != want {
		t.Fatalf("TextContent(#out) after exponentiation operators = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBitwiseAndShiftOperators(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", ` + "`" + `${5 & 3}-${5 | 2}-${5 ^ 1}-${1 << 3}-${8 >> 1}-${8 >>> 1}-${~1}-${1n & 3n}-${1n << 2n}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after bitwise operators error = %v", err)
	} else if want := "1-7-4-8-4-4--2-1-4"; got != want {
		t.Fatalf("TextContent(#out) after bitwise operators = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBlockBodiedWhileLoops(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="step1"></div><div id="step2"></div><script>let step1 = host.querySelector("#step1"); let step2 = host.querySelector("#step2"); while (step1 ?? step2) { if (step1) { host.setTextContent("#out", "first"); host.removeNode("#step1"); step1 &&= undefined } else { host.setTextContent("#out", "second"); host.removeNode("#step2"); step2 &&= undefined } }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after while loop error = %v", err)
	} else if want := "second"; got != want {
		t.Fatalf("TextContent(#out) after while loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSelectorLists(t *testing.T) {
	harness, err := FromHTML(`<main><section id="wrap" data-note="a,b"><article id="a1"></article></section><aside id="plain"></aside><div id="out">old</div><script>let first = host.querySelector('.missing, #plain'); let count = host.querySelectorAll('section[data-note="a,b"], aside'); host.setTextContent("#out", ` + "`" + `${first ? "hit" : "miss"}|${count}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after selector lists error = %v", err)
	} else if want := "hit|2"; got != want {
		t.Fatalf("TextContent(#out) after selector lists = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNthChildOfSelectorList(t *testing.T) {
	harness, err := FromHTML(`<main><ul id="list"><li id="one" class="selected">1</li><li id="two">2</li><li id="three" class="selected">3</li><li id="four" class="selected">4</li></ul><div id="out">old</div><script>let second = host.querySelector('li:nth-child(2 of .selected, #two)'); let last = host.querySelector('li:nth-last-child(1 of .selected)'); host.setTextContent("#out", ` + "`" + `${second === host.querySelector("#two")}-${last === host.querySelector("#four")}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after nth-child selector lists error = %v", err)
	} else if want := "true-true"; got != want {
		t.Fatalf("TextContent(#out) after nth-child selector lists = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportBlockBodiedForLoops(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>for (let keepGoing = true; keepGoing; keepGoing &&= false) { host.setTextContent("#out", "ran") }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for loop error = %v", err)
	} else if want := "ran"; got != want {
		t.Fatalf("TextContent(#out) after for loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSingleStatementLoopBodies(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let count = 0; while (count < 2) count++; let out = ""; for (let value of [1, 2]) out += value; host.setTextContent("#out", ` + "`" + `${count}|${out}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after single-statement loop bodies error = %v", err)
	} else if want := "2|12"; got != want {
		t.Fatalf("TextContent(#out) after single-statement loop bodies = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSingleStatementIfBodies(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let count = 0; if (count > 1) count++; else count += 2; host.setTextContent("#out", ` + "`" + `${count}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after single-statement if bodies error = %v", err)
	} else if want := "2"; got != want {
		t.Fatalf("TextContent(#out) after single-statement if bodies = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForOfLoops(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>for (let [first, second = first] of [[1], [2, 3]]) { host.setTextContent("#out", ` + "`" + `${first}-${second}` + "`" + `) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for...of loop error = %v", err)
	} else if want := "2-3"; got != want {
		t.Fatalf("TextContent(#out) after for...of loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectForOfOverNonIterableValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>for (let value of 1) { host.setTextContent("#out", value) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "for...of loops require a string, array, or iterator-like object value in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want `for...of` runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportForOfLoopsOnIteratorLikeObjects(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let values = { index: 0, next() { if (this.index === 0) { this.index = 1; return { value: "left", done: false } }; if (this.index === 1) { this.index = 2; return { value: "right", done: false } }; return { done: true } } }; let out = ""; for (let value of values) { out += value }; host.setTextContent("#out", out)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for...of iterator-like loop error = %v", err)
	} else if want := "leftright"; got != want {
		t.Fatalf("TextContent(#out) after for...of iterator-like loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForOfLoopsOnStringValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script>for (let value of "go") { host.setTextContent("#out", host?.["textContent"]("#out") + value) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for...of string loop error = %v", err)
	} else if want := "go"; got != want {
		t.Fatalf("TextContent(#out) after for...of string loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForInLoops(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script>for (let key in { alpha: 1, beta: 2 }) { host.setTextContent("#out", host?.["textContent"]("#out") + key) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for...in loop error = %v", err)
	} else if want := "alphabeta"; got != want {
		t.Fatalf("TextContent(#out) after for...in loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForInLoopsOnStringValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script>for (let key in "go") { host.setTextContent("#out", host?.["textContent"]("#out") + key) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for...in string loop error = %v", err)
	} else if want := "01"; got != want {
		t.Fatalf("TextContent(#out) after for...in string loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsRejectForInOverNonObjects(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>for (let key in 1) { host.setTextContent("#out", key) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if _, err := harness.TextContent("main"); err == nil {
		t.Fatalf("TextContent(main) error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("TextContent(main) error = %#v, want DOM error", err)
	}

	if got := harness.Debug().DOMError(); got == "" {
		t.Fatalf("Debug().DOMError() = %q, want runtime error text", got)
	} else if !strings.Contains(got, "for...in loops require a string, object, or array value on the right in this bounded classic-JS slice") {
		t.Fatalf("Debug().DOMError() = %q, want `for...in` runtime error text", got)
	}
}

func TestHarnessInlineScriptsSupportForAwaitOfLoops(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>async function wrap(value) { return value }; for await (let value of [wrap("alpha"), wrap("beta")]) { host.setTextContent("#out", value) }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for await...of loop error = %v", err)
	} else if want := "beta"; got != want {
		t.Fatalf("TextContent(#out) after for await...of loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForAwaitOfLoopsOnStringValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out"></div><script>async function run() { for await (let value of "go") { host.setTextContent("#out", host?.["textContent"]("#out") + value) } }; await run()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for await...of string loop error = %v", err)
	} else if want := "go"; got != want {
		t.Fatalf("TextContent(#out) after for await...of string loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForAwaitOfLoopsOnIteratorLikeObjects(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>async function wrap(value) { return value }; let values = { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: wrap("alpha"), done: false } }; if (this.index === 1) { this.index = 2; return { value: wrap("beta"), done: false } }; return { done: true } } }; let out = ""; for await (let value of values) { out += value }; host.setTextContent("#out", out)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for await...of iterator-like loop error = %v", err)
	} else if want := "alphabeta"; got != want {
		t.Fatalf("TextContent(#out) after for await...of iterator-like loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForAwaitOfLoopsOnAsyncIteratorLikeObjects(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>async function run() { let values = { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: "alpha", done: false } }; if (this.index === 1) { this.index = 2; return { value: "beta", done: false } }; return { done: true } } }; let out = ""; for await (let value of values) { out += value }; host.setTextContent("#out", out) }; await run()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for await...of async iterator-like loop error = %v", err)
	} else if want := "alphabeta"; got != want {
		t.Fatalf("TextContent(#out) after for await...of async iterator-like loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportForAwaitOfLoopsWithArrayBindingPatternsOnAsyncIteratorLikeObjects(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>async function run() { let values = { index: 0, async next() { if (this.index === 0) { this.index = 1; return { value: ["alpha", "beta"], done: false } }; return { done: true } } }; for await (let [first, second] of values) { host.setTextContent("#out", first + second) } }; await run()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after for await...of async iterator-like array binding loop error = %v", err)
	} else if want := "alphabeta"; got != want {
		t.Fatalf("TextContent(#out) after for await...of async iterator-like array binding loop = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportTypeofOperator(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>host.setTextContent("#out", ` + "`" + `${typeof null}-${typeof host}-${typeof host.echo}` + "`" + `)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after typeof operator error = %v", err)
	} else if want := "object-object-function"; got != want {
		t.Fatalf("TextContent(#out) after typeof operator = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassDeclarationsWithStaticBlocks(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>class Example { static { host.setTextContent("#out", "static") } }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class declaration error = %v", err)
	} else if want := "static"; got != want {
		t.Fatalf("TextContent(#out) after class declaration = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportStaticClassFields(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>class Example { static value = host.setTextContent("#out", "field"); static { host.setTextContent("#out", "block") } }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class field error = %v", err)
	} else if want := "block"; got != want {
		t.Fatalf("TextContent(#out) after class field = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperInClassFieldInitializers(t *testing.T) {
	script := "class Base { constructor() {} static get label() { return \"base-static\" } get kind() { return \"base-instance\" } }; class Example extends Base { static label = super.label; value = super.kind; constructor() { super(); host.setTextContent(\"#out\", `" + "${Example.label}|${this.value}" + "`); } }; new Example()"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after super in class field initializers error = %v", err)
	} else if want := "base-static|base-instance"; got != want {
		t.Fatalf("TextContent(#out) after super in class field initializers = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperInBaseClassFieldInitializers(t *testing.T) {
	script := `class Example { static label = super.label; value = super.kind; constructor() { host.setTextContent("#out", ` + "`" + `${Example.label}|${this.value}` + "`" + `) } }; new Example()`
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after base class super field initializers error = %v", err)
	} else if want := "undefined|undefined"; got != want {
		t.Fatalf("TextContent(#out) after base class super field initializers = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperInBaseClassComputedMemberNames(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>class Example { static [super.name] = "value" }; host.setTextContent("#out", Example.undefined)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after base class super computed member names error = %v", err)
	} else if want := "value"; got != want {
		t.Fatalf("TextContent(#out) after base class super computed member names = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassMethods(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">old</div><script>class Example { static value = "field"; static writeStatic() { host.setTextContent("#out", Example.value) } writeInstance() { host.setTextContent("#side", "instance") } }; Example.writeStatic(); Example.prototype.writeInstance()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class methods error = %v", err)
	} else if want := "field"; got != want {
		t.Fatalf("TextContent(#out) after class methods = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after class methods error = %v", err)
	} else if want := "instance"; got != want {
		t.Fatalf("TextContent(#side) after class methods = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportStaticPrototypeMembers(t *testing.T) {
	script := "class Example { static prototype = \"special\"; writeInstance() { return \"instance\" } }; let example = new Example(); host.setTextContent(\"#out\", `" + "${Example.prototype}|${example.writeInstance()}|${example instanceof Example}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after static prototype members error = %v", err)
	} else if want := "special|instance|true"; got != want {
		t.Fatalf("TextContent(#out) after static prototype members = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportStaticPrototypeSetterMembers(t *testing.T) {
	script := "class Example { static set prototype(value) { host.setTextContent(\"#out\", value) } }; Example.prototype = \"special\"; host.setTextContent(\"#side\", `" + "${Example.prototype}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><div id=\"side\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after static prototype setter members error = %v", err)
	} else if want := "special"; got != want {
		t.Fatalf("TextContent(#out) after static prototype setter members = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after static prototype setter members error = %v", err)
	} else if want := "undefined"; got != want {
		t.Fatalf("TextContent(#side) after static prototype setter members = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassExpressions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>class Base { static read() { return "base" } }; let Derived = class extends Base { static read() { return super.read() + "-expr" } }; host.setTextContent("#out", Derived.read())</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class expressions error = %v", err)
	} else if want := "base-expr"; got != want {
		t.Fatalf("TextContent(#out) after class expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassInheritanceFromClassExpressionValues(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>function makeBase() { return class { static read() { return "base" } } }; class Derived extends makeBase() { static read() { return super.read() + "-expr" } }; host.setTextContent("#out", Derived.read())</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class inheritance from class expression values error = %v", err)
	} else if want := "base-expr"; got != want {
		t.Fatalf("TextContent(#out) after class inheritance from class expression values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassInheritanceFromConstructibleFunctionValues(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function Base(value = \"base\") { this.seed = value }; class Derived extends Base {}; host.setTextContent(\"#out\", `" + "${new Derived(\"seed\").seed}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class inheritance from constructible function values error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#out) after class inheritance from constructible function values = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportConstructibleFunctionPrototypeAccess(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function Base() {}; function Other() {}; class Derived extends Base { static read() { return `" + "${typeof super.prototype}|${typeof Base.prototype}|${new Base() instanceof Base}|${new Base() instanceof Other}" + "` } }; host.setTextContent(\"#out\", Derived.read())</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after constructible function prototype access error = %v", err)
	} else if want := "object|object|true|false"; got != want {
		t.Fatalf("TextContent(#out) after constructible function prototype access = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportExtendsNull(t *testing.T) {
	script := "class Example extends null { static read() { return \"ok\" } }; let example = new Example(); host.setTextContent(\"#out\", `" + "${example instanceof Example}|${Example.read()}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after extends null error = %v", err)
	} else if want := "true|ok"; got != want {
		t.Fatalf("TextContent(#out) after extends null = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportNewOnClassExpressions(t *testing.T) {
	harness, err := FromHTML(`<main><div id="named">old</div><div id="anon">old</div><script>let Named = class { constructor() { host.setTextContent("#named", "named") } }; new Named(); new (class { constructor() { host.setTextContent("#anon", "anon") } })()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#named"); err != nil {
		t.Fatalf("TextContent(#named) after new on class expressions error = %v", err)
	} else if want := "named"; got != want {
		t.Fatalf("TextContent(#named) after new on class expressions = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#anon"); err != nil {
		t.Fatalf("TextContent(#anon) after new on class expressions error = %v", err)
	} else if want := "anon"; got != want {
		t.Fatalf("TextContent(#anon) after new on class expressions = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassGetterAccessors(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>let label = \"kind\"; class Example { static prefix = \"static\"; static get [label]() { return this.prefix } get = \"plain\"; value = \"instance\"; get read() { return this.value } }; let example = new Example(); host.setTextContent(\"#out\", `" + "${Example.kind}|${example.read}|${example.get}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class getter accessors error = %v", err)
	} else if want := "static|instance|plain"; got != want {
		t.Fatalf("TextContent(#out) after class getter accessors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportPrivateClassGetterAccessors(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>class Example { static #prefix = \"static\"; static get #kind() { return this.#prefix } static revealed = Example.#kind; #value = \"instance\"; get #read() { return this.#value } snapshot = this.#read }; let example = new Example(); host.setTextContent(\"#out\", `" + "${Example.revealed}|${example.snapshot}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after private class getter accessors error = %v", err)
	} else if want := "static|instance"; got != want {
		t.Fatalf("TextContent(#out) after private class getter accessors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassSetterAccessors(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>class Example { static _value = \"static-seed\"; static get value() { return this._value } static set value(next) { this._value = next } _value = \"instance-seed\"; get value() { return this._value } set value(next) { this._value = next } }; Example.value = \"static-updated\"; let example = new Example(); example.value = \"instance-updated\"; host.setTextContent(\"#out\", `" + "${Example.value}|${example.value}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class setter accessors error = %v", err)
	} else if want := "static-updated|instance-updated"; got != want {
		t.Fatalf("TextContent(#out) after class setter accessors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassInstanceFieldsAndNew(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">old</div><div id="tail">old</div><script>class Example { value = "field"; constructor() { host.setTextContent("#tail", "ctor") } write() { host.setTextContent("#side", "method") } }; let instance = new Example(); host.setTextContent("#out", instance.value); instance.write()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#tail"); err != nil {
		t.Fatalf("TextContent(#tail) after class instantiation error = %v", err)
	} else if want := "ctor"; got != want {
		t.Fatalf("TextContent(#tail) after class instantiation = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class instantiation error = %v", err)
	} else if want := "field"; got != want {
		t.Fatalf("TextContent(#out) after class instantiation = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after class instantiation error = %v", err)
	} else if want := "method"; got != want {
		t.Fatalf("TextContent(#side) after class instantiation = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportConstructibleFunctionConstructorsAndInstanceof(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>function Plain(value) { this.value = value }; class Box { constructor(value) { this.value = value } }; let plain = new Plain(\"seed\"); let box = new Box(\"class\"); host.setTextContent(\"#out\", `" + "${plain.value}|${plain instanceof Plain}|${box.value}|${box instanceof Box}" + "`)</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after constructible function constructors error = %v", err)
	} else if want := "seed|true|class|true"; got != want {
		t.Fatalf("TextContent(#out) after constructible function constructors = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportComputedClassMembers(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><div id=\"side\">old</div><script>let fieldName = \"value\"; let staticName = \"staticValue\"; let methodName = \"write\"; class Example { [fieldName] = \"field\"; static [staticName] = \"static\"; [methodName]() { host.setTextContent(\"#side\", \"method\") } }; let instance = new Example(); host.setTextContent(\"#out\", `${Example.staticValue}-${instance.value}`); instance[methodName]()</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after computed class members error = %v", err)
	} else if want := "static-field"; got != want {
		t.Fatalf("TextContent(#out) after computed class members = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after computed class members error = %v", err)
	} else if want := "method"; got != want {
		t.Fatalf("TextContent(#side) after computed class members = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportComputedClassMembersReadingSuper(t *testing.T) {
	script := "class Base { static get staticKey() { return \"static-name\" } get instanceKey() { return \"instance-name\" } }; class Example extends Base { static [super.staticKey] = \"static\"; [super.instanceKey] = \"instance\" }; let example = new Example(); host.setTextContent(\"#out\", `" + "${Example[\"static-name\"]}|${example[\"instance-name\"]}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after computed class members reading super error = %v", err)
	} else if want := "static|instance"; got != want {
		t.Fatalf("TextContent(#out) after computed class members reading super = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportClassInheritance(t *testing.T) {
	harness, err := FromHTML("<main><div id=\"out\">old</div><div id=\"side\">old</div><div id=\"tail\">old</div><div id=\"extra\">old</div><script>class Base { value = \"base\"; static kind = \"base\"; constructor() { host.setTextContent(\"#tail\", \"baseCtor\") } write() { host.setTextContent(\"#side\", \"baseMethod\") } static ping() { host.setTextContent(\"#extra\", \"ping\") } }; class Derived extends Base { value = \"derived\"; static kind = \"derived\" }; let instance = new Derived(); host.setTextContent(\"#out\", `${Derived.kind}-${instance.value}`); instance.write(); Derived.ping()</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#tail"); err != nil {
		t.Fatalf("TextContent(#tail) after class inheritance error = %v", err)
	} else if want := "baseCtor"; got != want {
		t.Fatalf("TextContent(#tail) after class inheritance = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after class inheritance error = %v", err)
	} else if want := "derived-derived"; got != want {
		t.Fatalf("TextContent(#out) after class inheritance = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after class inheritance error = %v", err)
	} else if want := "baseMethod"; got != want {
		t.Fatalf("TextContent(#side) after class inheritance = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#extra"); err != nil {
		t.Fatalf("TextContent(#extra) after class inheritance error = %v", err)
	} else if want := "ping"; got != want {
		t.Fatalf("TextContent(#extra) after class inheritance = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperPropertyAccess(t *testing.T) {
	script := "class Base { static kind = \"base\"; greet() { return \"base\" } static label() { return \"label\" } }; class Derived extends Base { static seen = super.kind; static describe() { return `" + "${super[\"kind\"]}-${super[\"label\"]()}" + "` } read() { return `" + "${super.greet()}-${Derived.seen}" + "` } }; let instance = new Derived(); host.setTextContent(\"#out\", `" + "${Derived.seen}|${Derived.describe()}|${instance.read()}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after super property access error = %v", err)
	} else if want := "base|base-label|base-base"; got != want {
		t.Fatalf("TextContent(#out) after super property access = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperPropertyAssignment(t *testing.T) {
	script := "class Base { static label = \"base\" }; class Derived extends Base { static label = \"derived\"; kind = \"initial\"; static update() { super.label = \"static-updated\" } write() { super.kind = \"instance-updated\" } }; let instance = new Derived(); Derived.update(); instance.write(); host.setTextContent(\"#out\", `" + "${Derived.label}|${Base.label}|${instance.kind}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after super property assignment error = %v", err)
	} else if want := "static-updated|base|instance-updated"; got != want {
		t.Fatalf("TextContent(#out) after super property assignment = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperPropertyAssignmentOnMissingReceiverProperties(t *testing.T) {
	script := "class Base { static label = \"base\" }; class Derived extends Base { static update() { super.label = \"static-updated\" } write() { super.kind = \"instance-updated\" } }; let instance = new Derived(); Derived.update(); instance.write(); host.setTextContent(\"#out\", `" + "${Derived.label}|${Base.label}|${instance.kind}" + "`)"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after super property assignment on missing receiver properties error = %v", err)
	} else if want := "static-updated|base|instance-updated"; got != want {
		t.Fatalf("TextContent(#out) after super property assignment on missing receiver properties = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSuperCallsInConstructors(t *testing.T) {
	harness, err := FromHTML(`<main><div id="base">old</div><div id="derived">old</div><script>class Base { constructor(value = "base") { host.setTextContent("#base", value) } }; class Derived extends Base { constructor() { super("seed"); host.setTextContent("#derived", "done") } }; new Derived()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#base"); err != nil {
		t.Fatalf("TextContent(#base) after super constructor call error = %v", err)
	} else if want := "seed"; got != want {
		t.Fatalf("TextContent(#base) after super constructor call = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#derived"); err != nil {
		t.Fatalf("TextContent(#derived) after super constructor call error = %v", err)
	} else if want := "done"; got != want {
		t.Fatalf("TextContent(#derived) after super constructor call = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportConstructorArgumentsInNewExpressions(t *testing.T) {
	script := "class Example { constructor(first = \"seed\", second = \"tail\") { host.setTextContent(\"#out\", `" + "${first}-${second}" + "` ) } }; new Example(\"picked\")"
	harness, err := FromHTML("<main><div id=\"out\">old</div><script>" + script + "</script></main>")
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after constructor arguments error = %v", err)
	} else if want := "picked-tail"; got != want {
		t.Fatalf("TextContent(#out) after constructor arguments = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportPrivateClassFields(t *testing.T) {
	harness, err := FromHTML(`<main><div id="mirror">old</div><div id="base">old</div><div id="derived">old</div><div id="count">old</div><script>class Base { #base = "base"; #writeBase() { host.setTextContent("#base", this.#base) } readBase() { this.#writeBase() } }; class Derived extends Base { #derived = "derived"; mirrored = this.#derived; static #count = "7"; static #writeCount() { host.setTextContent("#count", this.#count) } static reveal() { this.#writeCount() } #writeDerived() { host.setTextContent("#derived", this.#derived) } readDerived() { this.#writeDerived() } }; let instance = new Derived(); host.setTextContent("#mirror", instance.mirrored); instance.readBase(); instance.readDerived(); Derived.reveal()</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#mirror"); err != nil {
		t.Fatalf("TextContent(#mirror) after private fields error = %v", err)
	} else if want := "derived"; got != want {
		t.Fatalf("TextContent(#mirror) after private fields = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#base"); err != nil {
		t.Fatalf("TextContent(#base) after private fields error = %v", err)
	} else if want := "base"; got != want {
		t.Fatalf("TextContent(#base) after private fields = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#derived"); err != nil {
		t.Fatalf("TextContent(#derived) after private fields error = %v", err)
	} else if want := "derived"; got != want {
		t.Fatalf("TextContent(#derived) after private fields = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#count"); err != nil {
		t.Fatalf("TextContent(#count) after private fields error = %v", err)
	} else if want := "7"; got != want {
		t.Fatalf("TextContent(#count) after private fields = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportSwitchStatements(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><script>let obj = { kind: "box" }; let arr = [1, 2]; let out = ""; switch (obj) { case { kind: "box" }: out = "bad"; break; case obj: out = "obj"; break; default: out = "default" }; switch (arr) { case [1, 2]: out += "|bad"; break; case arr: out += "|arr"; break; default: out += "|default" }; switch ("seed") { case "seed": out += "|seed"; break; default: out += "|default" }; host.setTextContent("#out", out)</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after switch error = %v", err)
	} else if want := "obj|arr|seed"; got != want {
		t.Fatalf("TextContent(#out) after switch = %q, want %q", got, want)
	}
}

func TestHarnessInlineScriptsSupportTryCatchFinallyStatements(t *testing.T) {
	harness, err := FromHTML(`<main><div id="out">old</div><div id="side">seed</div><script>try { host.setTextContent("#missing", "fail") } catch { host.setTextContent("#out", "caught") } finally { host.setTextContent("#side", "finally") }</script></main>`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after try/catch/finally error = %v", err)
	} else if want := "caught"; got != want {
		t.Fatalf("TextContent(#out) after try/catch/finally = %q, want %q", got, want)
	}
	if got, err := harness.TextContent("#side"); err != nil {
		t.Fatalf("TextContent(#side) after try/catch/finally error = %v", err)
	} else if want := "finally"; got != want {
		t.Fatalf("TextContent(#side) after try/catch/finally = %q, want %q", got, want)
	}
}

func TestNilHarnessMutationWrappersReturnDomErrors(t *testing.T) {
	var nilHarness *Harness

	if _, err := nilHarness.InnerHTML("#target"); err == nil {
		t.Fatalf("nil Harness.InnerHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.InnerHTML() error = %#v, want DOM error", err)
	}
	if _, err := nilHarness.OuterHTML("#target"); err == nil {
		t.Fatalf("nil Harness.OuterHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.OuterHTML() error = %#v, want DOM error", err)
	}
	if _, err := nilHarness.TextContent("#target"); err == nil {
		t.Fatalf("nil Harness.TextContent() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.TextContent() error = %#v, want DOM error", err)
	}
	if err := nilHarness.ReplaceChildren("#target", "<p>x</p>"); err == nil {
		t.Fatalf("nil Harness.ReplaceChildren() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.ReplaceChildren() error = %#v, want DOM error", err)
	}
	if err := nilHarness.SetInnerHTML("#target", "<p>x</p>"); err == nil {
		t.Fatalf("nil Harness.SetInnerHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.SetInnerHTML() error = %#v, want DOM error", err)
	}
	if err := nilHarness.SetTextContent("#target", "x"); err == nil {
		t.Fatalf("nil Harness.SetTextContent() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.SetTextContent() error = %#v, want DOM error", err)
	}
	if err := nilHarness.SetOuterHTML("#target", "<p>x</p>"); err == nil {
		t.Fatalf("nil Harness.SetOuterHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.SetOuterHTML() error = %#v, want DOM error", err)
	}
	if err := nilHarness.InsertAdjacentHTML("#target", "beforeend", "<p>x</p>"); err == nil {
		t.Fatalf("nil Harness.InsertAdjacentHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.InsertAdjacentHTML() error = %#v, want DOM error", err)
	}
	if err := nilHarness.RemoveNode("#target"); err == nil {
		t.Fatalf("nil Harness.RemoveNode() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.RemoveNode() error = %#v, want DOM error", err)
	}
	if err := nilHarness.CloneNode("#target", true); err == nil {
		t.Fatalf("nil Harness.CloneNode() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("nil Harness.CloneNode() error = %#v, want DOM error", err)
	}

	zeroSessionHarness := &Harness{}
	if _, err := zeroSessionHarness.InnerHTML("#target"); err == nil {
		t.Fatalf("Harness{nil session}.InnerHTML() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("Harness{nil session}.InnerHTML() error = %#v, want DOM error", err)
	}
	if _, err := zeroSessionHarness.TextContent("#target"); err == nil {
		t.Fatalf("Harness{nil session}.TextContent() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("Harness{nil session}.TextContent() error = %#v, want DOM error", err)
	}
	if err := zeroSessionHarness.CloneNode("#target", true); err == nil {
		t.Fatalf("Harness{nil session}.CloneNode() error = nil, want DOM error")
	} else if got, ok := err.(Error); !ok || got.Kind != ErrorKindDOM {
		t.Fatalf("Harness{nil session}.CloneNode() error = %#v, want DOM error", err)
	}
}
