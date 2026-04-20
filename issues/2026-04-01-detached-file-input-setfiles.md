# `SetFiles()` needs to target detached file inputs created on demand

## Summary

- Short summary: The file-input mock can seed and select attached `<input type="file">` elements, but `SetFiles()` does not reach detached file inputs that are created dynamically with `document.createElement("input")` and never appended to the DOM.

## Context

- Owning subsystem: Runtime / file input
- Related capability or gap: dynamic file input selection for import flows
- Related docs:
  - `internal/runtime/file_input.go`
  - `internal/runtime/session.go`
  - `../finitefield-site/web/content/pages/tools/forestry/basal-area-stem-density-calculator/template.html`
- Affected finitefield-site coverage: `web-go/internal/generate/forestry_basal_area_stem_density_calculator_browser_test.go` BRT-006

## Problem

- Current behavior: `SetFiles()` resolves file inputs through DOM tree selection only, so a detached input created by a click handler does not receive seeded files or dispatch `input` / `change` listeners.
- Expected behavior: `SetFiles()` should be able to target any matching file input in the current document state, including detached elements created on demand by import pickers.
- Reproduction steps:
  1. Load a page with a button that creates `<input type="file">` in a click handler.
  2. Call `SetFiles("input[type=file]", []string{"sample.json"})` after clicking the button.
  3. Observe that the change handler does not run because the detached input is not resolved.
- Reproduction code:

```text
document.getElementById("import").addEventListener("click", () => {
  const picker = document.createElement("input");
  picker.type = "file";
  picker.addEventListener("change", () => {
    document.getElementById("out").textContent = "imported";
  });
  picker.click();
});
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestForestryBasalAreaStemDensityCalculatorBRT006ExportJsonAndImportDetachedFilePicker$'
```

- Scope / non-goals: Keep the fix focused on file-input resolution and event dispatch for detached inputs. Do not add a workaround in the page code.

## Acceptance Criteria

- [ ] `SetFiles()` can target a detached file input created on demand.
- [ ] The seeded selection dispatches `input` and `change` listeners on that detached input.
- [ ] The detached input still records file selections in the mock registry.
- [ ] A regression test covers the detached-picker import flow.

## Test Plan

- Suggested test layer: runtime/session test for `SetFiles()`, plus a browser-bootstrap smoke test.
- Regression or failure-path coverage: a click-created import picker should receive seeded files and trigger its change handler.
- Mock or fixture needs: a detached `<input type="file">` created by script.

## Notes

- This gap was confirmed in `browser-tester-go`, not in the finitefield-site app code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
