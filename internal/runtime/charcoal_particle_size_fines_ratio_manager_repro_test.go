package runtime

import (
	"os"
	"strings"
	"testing"
)

func TestSessionCharcoalParticleSizeFinesRatioManagerExternalHTML(t *testing.T) {
	path := os.Getenv("CHARCOAL_HTML_PATH")
	if path == "" {
		t.Skip("CHARCOAL_HTML_PATH is not set")
	}

	html, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	rawHTML := string(html)
	if old := os.Getenv("CHARCOAL_HTML_REPLACE_OLD"); old != "" {
		rawHTML = strings.Replace(rawHTML, old, os.Getenv("CHARCOAL_HTML_REPLACE_NEW"), 1)
	}

	s := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty for external charcoal HTML", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproInitialSyncDelegatedInputListeners(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="sampleName" type="text" value="seed"><input data-bind="showChart" type="checkbox" checked><select data-bind="massBasis"><option value="as_received">As received</option><option value="dry">Dry</option></select><textarea data-bind="sampleNote">seed</textarea></section><div id="out"></div><script>const root = document.getElementById("root"); const state = { sampleName: "Lot 1", showChart: true, massBasis: "dry", sampleNote: "hello" }; function syncAllControls() { root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); } function updateFromInput(target) { const value = target.type === "checkbox" ? target.checked : target.value; document.getElementById("out").textContent = String(value); } root.addEventListener("input", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); root.addEventListener("change", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); syncAllControls();</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := s.AdvanceTime(500); err != nil {
		t.Fatalf("AdvanceTime(500) error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after delegated input sync repro", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproRowEditorInitialRender(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="sampleName" type="text" value="seed"><input data-bind="showChart" type="checkbox" checked><select data-bind="massBasis"><option value="as_received">As received</option><option value="dry">Dry</option></select><textarea data-bind="sampleNote">seed</textarea></section><div id="rows"></div><div id="out"></div><script>const root = document.getElementById("root"); const rowsRoot = document.getElementById("rows"); const out = document.getElementById("out"); const state = { sampleName: "Lot 1", showChart: true, massBasis: "dry", sampleNote: "hello", rows: [{ id: "row-1", rowType: "sieve", label: "50 mm", openingMm: "50", retainedMass: "1.0", retainedPct: "", included: true }] }; let syncTimer = 0; function syncAllControls() { root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); } function renderRows() { rowsRoot.innerHTML = state.rows.map((row) => '<div data-row-id="' + row.id + '"><select data-row-field="rowType"><option value="sieve">sieve</option><option value="pan">pan</option></select><input data-row-field="label" type="text" value="' + row.label + '"><input data-row-field="openingMm" type="text" value="' + row.openingMm + '"><input data-row-field="retainedMass" type="text" value="' + row.retainedMass + '"><input data-row-field="included" type="checkbox"' + (row.included ? ' checked' : '') + '></div>').join(""); } function renderResults() { out.textContent = String(state.rows.length); } function scheduleRefresh() { if (syncTimer) { clearTimeout(syncTimer); } syncTimer = setTimeout(() => { syncTimer = 0; renderResults(); }, 0); } function updateFromInput(target) { const value = target.type === "checkbox" ? target.checked : target.value; state[target.dataset.bind] = value; scheduleRefresh(); } function updateRowInput(target) { const row = target.closest("[data-row-id]"); const rowId = row && row.dataset.rowId; const item = state.rows.find((entry) => entry.id === rowId); if (!item) return; const value = target.type === "checkbox" ? target.checked : target.value; item[target.dataset.rowField] = value; scheduleRefresh(); } root.addEventListener("input", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.rowField) { updateRowInput(target); return; } if (target.dataset.bind) { updateFromInput(target); } }); root.addEventListener("change", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.rowField) { updateRowInput(target); return; } if (target.dataset.bind) { updateFromInput(target); } }); syncAllControls(); renderRows(); renderResults();</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := s.AdvanceTime(500); err != nil {
		t.Fatalf("AdvanceTime(500) error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after row editor repro", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproReentrantSyncAllControls(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="displayPrecision" type="number" value="1"><input data-bind="massUnit" type="text" value="kg"><select data-bind="openingUnit"><option value="mm">mm</option><option value="mesh">mesh</option></select></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { displayPrecision: "1", massUnit: "kg", openingUnit: "mm" }; function syncAllControls() { root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); } function scheduleRefresh() { out.textContent = String(Date.now()); } function updateFromInput(target) { const path = target.dataset.bind; const value = target.type === "checkbox" ? target.checked : target.value; state[path] = value; if (path === "displayPrecision" || path === "massUnit" || path === "openingUnit") { syncAllControls(); } scheduleRefresh(); } root.addEventListener("input", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); root.addEventListener("change", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); syncAllControls();</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := s.AdvanceTime(500); err != nil {
		t.Fatalf("AdvanceTime(500) error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after reentrant syncAllControls repro", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproProfileControls(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><select data-bind="profilePreset"><option value="bbq">bbq</option><option value="custom">custom</option></select><input data-bind="profileName" type="text" value=""><input data-bind="customProfile.softLimits.finesPct" type="text" value="4"><input data-bind="customProfile.hardLimits.finesPct" type="text" value="8"><input data-bind="customProfile.weights.finesPct" type="text" value="40"><input data-bind="customProfile.useD50" type="checkbox"></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { profilePreset: "bbq", profileName: "", customProfile: { softLimits: { finesPct: "4" }, hardLimits: { finesPct: "8" }, weights: { finesPct: "40" }, useD50: false } }; function presetToStateProfile() { return { softLimits: { finesPct: "6" }, hardLimits: { finesPct: "10" }, weights: { finesPct: "30" }, useD50: true }; } function syncProfileControls() { const isCustom = state.profilePreset === "custom"; const current = isCustom ? state.customProfile : presetToStateProfile(); root.querySelectorAll("[data-bind^='customProfile.']").forEach((el) => { const path = el.dataset.bind.replace(/^customProfile\./, ""); const value = path === "useD50" ? current.useD50 : current.softLimits.finesPct; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } el.disabled = !isCustom; }); } function syncAllControls() { root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); syncProfileControls(); } function scheduleRefresh() { out.textContent = state.profilePreset; } function updateFromInput(target) { const path = target.dataset.bind; const value = target.type === "checkbox" ? target.checked : target.value; if (path === "profilePreset") { state.profilePreset = value; if (value !== "custom") { state.customProfile = presetToStateProfile(); } syncProfileControls(); } else { state[path] = value; } scheduleRefresh(); } root.addEventListener("input", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); root.addEventListener("change", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); syncAllControls();</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := s.AdvanceTime(500); err != nil {
		t.Fatalf("AdvanceTime(500) error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after profile controls repro", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproLongControlListTextareaType(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="shipOkThreshold" type="text" value="75"><input data-bind="reviewThreshold" type="text" value="55"><select data-bind="finesDefinitionMode"><option value="pan">pan</option><option value="below_cutoff">below_cutoff</option></select><input data-bind="finesCutoffMm" type="text" value="5"><input data-bind="displayPrecision" type="number" value="1"><input data-bind="showChart" type="checkbox" checked><input data-bind="autoSort" type="checkbox" checked><input data-bind="querySync" type="checkbox" checked><input data-bind="localSave" type="checkbox" checked><input data-bind="localOnlyNotes" type="checkbox" checked><input data-bind="autoOpenQuery" type="checkbox" checked><input data-bind="mobileSummaryBar" type="checkbox" checked><select data-bind="profilePreset"><option value="bbq">bbq</option><option value="custom">custom</option></select><input data-bind="profileName" type="text" value=""><input data-bind="sampleName" type="text" value=""><input data-bind="lotCode" type="text" value=""><input data-bind="sampleDate" type="date" value=""><input data-bind="specName" type="text" value=""><input data-bind="sieveStackName" type="text" value=""><select data-bind="massBasis"><option value="as_received">as_received</option><option value="dry">dry</option></select><input data-bind="sampleMass" type="text" value=""><textarea data-bind="sampleNote"></textarea><input data-bind="directFinesPct" type="text" value=""><input data-bind="customProfile.softLimits.finesPct" type="text" value="4"><input data-bind="customProfile.hardLimits.finesPct" type="text" value="8"><input data-bind="customProfile.weights.finesPct" type="text" value="40"><input data-bind="customProfile.useD50" type="checkbox"></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { shipOkThreshold: "75", reviewThreshold: "55", finesDefinitionMode: "pan", finesCutoffMm: "5", displayPrecision: "1", showChart: true, autoSort: true, querySync: true, localSave: true, localOnlyNotes: true, autoOpenQuery: true, mobileSummaryBar: true, profilePreset: "bbq", profileName: "", sampleName: "", lotCode: "", sampleDate: "", specName: "", sieveStackName: "", massBasis: "as_received", sampleMass: "", sampleNote: "", directFinesPct: "", customProfile: { softLimits: { finesPct: "4" }, hardLimits: { finesPct: "8" }, weights: { finesPct: "40" }, useD50: false } }; function presetToStateProfile() { return { softLimits: { finesPct: "6" }, hardLimits: { finesPct: "10" }, weights: { finesPct: "30" }, useD50: true }; } function syncProfileControls() { const isCustom = state.profilePreset === "custom"; const current = isCustom ? state.customProfile : presetToStateProfile(); root.querySelectorAll("[data-bind^='customProfile.']").forEach((el) => { const path = el.dataset.bind.replace(/^customProfile\./, ""); const value = path === "useD50" ? current.useD50 : current.softLimits.finesPct; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } el.disabled = !isCustom; }); } function syncAllControls() { root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); syncProfileControls(); } function updateFromInput(target) { const path = target.dataset.bind; const value = target.type === "checkbox" ? target.checked : target.value; if (path === "profilePreset") { state.profilePreset = value; if (value !== "custom") { state.customProfile = presetToStateProfile(); } syncProfileControls(); } else { state[path] = value; } out.textContent = "done"; } root.addEventListener("input", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); root.addEventListener("change", (event) => { const target = event.target; if (!(target instanceof HTMLElement)) return; if (target.dataset.bind) updateFromInput(target); }); syncAllControls(); out.textContent = "done";</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after long control list textarea.type repro", got)
	}
	if got, err := s.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproTextareaInLabelWithPlaceholder(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="sampleMass" type="text" value="10"><label class="field"><span>Sample note</span><textarea class="input" data-bind="sampleNote" placeholder="Memo"></textarea></label></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { sampleMass: "12", sampleNote: "hello" }; root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); out.textContent = "done";</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after textarea-in-label repro", got)
	}
	if got, err := s.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproLongListWithLabelTextarea(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><input data-bind="shipOkThreshold" type="text" value="75"><input data-bind="reviewThreshold" type="text" value="55"><select data-bind="finesDefinitionMode"><option value="pan">pan</option><option value="below_cutoff">below_cutoff</option></select><input data-bind="finesCutoffMm" type="text" value="5"><input data-bind="displayPrecision" type="number" value="1"><input data-bind="showChart" type="checkbox" checked><input data-bind="autoSort" type="checkbox" checked><input data-bind="querySync" type="checkbox" checked><input data-bind="localSave" type="checkbox" checked><input data-bind="localOnlyNotes" type="checkbox" checked><input data-bind="autoOpenQuery" type="checkbox" checked><input data-bind="mobileSummaryBar" type="checkbox" checked><select data-bind="profilePreset"><option value="bbq">bbq</option><option value="custom">custom</option></select><input data-bind="profileName" type="text" value=""><input data-bind="sampleName" type="text" value=""><input data-bind="lotCode" type="text" value=""><input data-bind="sampleDate" type="date" value=""><input data-bind="specName" type="text" value=""><input data-bind="sieveStackName" type="text" value=""><select data-bind="massBasis"><option value="as_received">as_received</option><option value="dry">dry</option></select><input data-bind="sampleMass" type="text" value=""><label class="field"><span>Sample note</span><textarea class="input" data-bind="sampleNote" placeholder="Memo"></textarea></label><input data-bind="directFinesPct" type="text" value=""><input data-bind="customProfile.softLimits.finesPct" type="text" value="4"><input data-bind="customProfile.hardLimits.finesPct" type="text" value="8"><input data-bind="customProfile.weights.finesPct" type="text" value="40"><input data-bind="customProfile.useD50" type="checkbox"></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { shipOkThreshold: "75", reviewThreshold: "55", finesDefinitionMode: "pan", finesCutoffMm: "5", displayPrecision: "1", showChart: true, autoSort: true, querySync: true, localSave: true, localOnlyNotes: true, autoOpenQuery: true, mobileSummaryBar: true, profilePreset: "bbq", profileName: "", sampleName: "", lotCode: "", sampleDate: "", specName: "", sieveStackName: "", massBasis: "as_received", sampleMass: "", sampleNote: "", directFinesPct: "", customProfile: { softLimits: { finesPct: "4" }, hardLimits: { finesPct: "8" }, weights: { finesPct: "40" }, useD50: false } }; function presetToStateProfile() { return { softLimits: { finesPct: "6" }, hardLimits: { finesPct: "10" }, weights: { finesPct: "30" }, useD50: true }; } function syncProfileControls() { const isCustom = state.profilePreset === "custom"; const current = isCustom ? state.customProfile : presetToStateProfile(); root.querySelectorAll("[data-bind^='customProfile.']").forEach((el) => { const path = el.dataset.bind.replace(/^customProfile\./, ""); const value = path === "useD50" ? current.useD50 : current.softLimits.finesPct; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } el.disabled = !isCustom; }); } function syncAllControls() { root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); syncProfileControls(); } out.textContent = "done";</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after long list with label textarea repro", got)
	}
	if got, err := s.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproSettingsGridTextareaPair(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><div class="settings-grid"><label class="field"><span>Sample mass</span><input class="input" data-bind="sampleMass" type="text" inputmode="decimal" placeholder="Mass"></label><label class="field"><span>Sample note</span><textarea class="input" data-bind="sampleNote" placeholder="Memo"></textarea></label></div></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { sampleMass: "12", sampleNote: "hello" }; root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); out.textContent = "done";</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after settings-grid textarea pair repro", got)
	}
	if got, err := s.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproJapaneseTextareaValue(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><label class="field"><span>Sample note</span><textarea class="input" data-bind="sampleNote" placeholder="メモ"></textarea></label></section><div id="out"></div><script>const root = document.getElementById("root"); const out = document.getElementById("out"); const state = { sampleNote: "BBQ 向けのサンプル" }; root.querySelectorAll("[data-bind]").forEach((el) => { const value = state[el.dataset.bind]; if (el.type === "checkbox") { el.checked = Boolean(value); } else { el.value = value ?? ""; } }); out.textContent = "done";</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after Japanese textarea value repro", got)
	}
	if got, err := s.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
}

func TestSessionCharcoalParticleSizeFinesRatioManagerReproTextareaProgrammaticValueDoesNotDispatchInput(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><section id="root"><textarea id="note"></textarea></section><div id="out"></div><script>const root = document.getElementById("root"); const note = document.getElementById("note"); const out = document.getElementById("out"); root.addEventListener("input", (event) => { const target = event.target; out.textContent = target.type; }); note.value = "BBQ 向けのサンプル"; out.textContent = "done";</script></main>`,
	})

	if _, err := s.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if got := s.DOMError(); got != "" {
		t.Fatalf("DOMError() = %q, want empty after programmatic textarea value repro", got)
	}
	if got, err := s.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "done" {
		t.Fatalf("TextContent(#out) = %q, want done", got)
	}
}
