# `HTMLInputElement` interactions still fail in bounded classic-JS slices during tolerance-checker browser tests

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and choose the smallest layer that can fix the behavior.
- Review `capability-matrix.md` and confirm the missing capability row.
- Review `roadmap.md` if the fix affects sequencing or rollout order.

## Summary

- Short summary: The browser harness still rejects ordinary `HTMLInputElement` interactions inside bounded classic-JS slices, so the tolerance-checker Go browser tests cannot type into manual measurement cells, change CSV unit selects, or dispatch paste on an input target.

## Context

- Owning subsystem: `bt-runtime` browser interaction / bounded classic-JS slice execution
- Related capability or gap: element-level text entry, select changes, and paste dispatch on `HTMLInputElement`
- Related docs: `browser-tester-go/doc/capability-matrix.md`, `browser-tester-go/doc/implementation-guide.md`

## Problem

- Current behavior: When `finitefield-site/web-go/internal/generate/construction_tolerance_checker_browser_test.go` drives the tolerance-checker page, the harness raises `dom: unsupported: unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice` for:
  - `TypeText()` on manual grid cells
  - `SetSelectValue()` on CSV column unit selects
  - `Dispatch(selector, "paste")` on the manual table input target
- Expected behavior: These are normal browser-facing controls. The harness should allow the page's inline handlers to run without surfacing a browser-surface rejection.
- Reproduction steps:
  1. Load the tolerance-checker page in the browser harness.
  2. Switch to the manual tab and try to type into a measurement cell with `TypeText()`.
  3. Switch to the CSV tab and change a column-unit select with `SetSelectValue()`.
  4. Switch back to manual mode, seed TSV text, and dispatch `paste` on the first input cell.
  5. Observe the unsupported `HTMLInputElement` surface error before the page logic can complete.
- Reproduction code:

```html
<main>
  <input id="value" type="text" value="">
  <select id="unit">
    <option value="mm">mm</option>
    <option value="m">m</option>
  </select>
  <textarea id="log"></textarea>
  <script>
    const value = document.getElementById("value");
    const unit = document.getElementById("unit");
    value.addEventListener("input", (event) => {
      if (!(event.target instanceof HTMLInputElement)) return;
      document.getElementById("log").value = `typed:${event.target.value}`;
    });
    unit.addEventListener("change", (event) => {
      document.getElementById("log").value = `unit:${event.target.value}`;
    });
    value.addEventListener("paste", (event) => {
      if (!(event.target instanceof HTMLInputElement)) return;
      document.getElementById("log").value = `paste:${event.clipboardData.getData("text/plain")}`;
    });
  </script>
</main>
```

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run 'TestLocalizeInternalURL|TestConstructionToleranceChecker' -count=1
```

- Scope / non-goals: This issue is about the browser tester runtime surface gate. It is not about changing `web-go` page code or introducing a test workaround to avoid `HTMLInputElement` usage.

## Evidence

- Failing tests from the rerun:
  - `TestConstructionToleranceCheckerBRT004ManualOKInRange`
  - `TestConstructionToleranceCheckerBRT005ManualAboveUpperShowsDelta`
  - `TestConstructionToleranceCheckerBRT006ManualBelowLowerShowsDelta`
  - `TestConstructionToleranceCheckerBRT007ManualParseErrorMarksRowNG`
  - `TestConstructionToleranceCheckerBRT008AnyOutOfRangePointMarksRowNG`
  - `TestConstructionToleranceCheckerBRT010ManualMixedUnitConversionUsesBaseUnit`
  - `TestConstructionToleranceCheckerBRT011ManualPasteExpandsTSVIntoTable`
  - `TestConstructionToleranceCheckerBRT013CsvColumnUnitSettingAffectsJudgment`
- Representative error text:

```text
dom: unsupported: unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice
```

- The failure occurs before the page's inline handlers can complete, so it is a browser-tester-go runtime limitation rather than a `web-go` page regression.

## Acceptance Criteria

- [ ] `TypeText()` can drive `HTMLInputElement` controls in bounded classic-JS slices.
- [ ] `SetSelectValue()` can change `<select>` controls that page handlers treat as part of the same input flow.
- [ ] Synthetic `paste` dispatch on an input target can run without hitting the `HTMLInputElement` surface gate.
- [ ] The tolerance-checker Go browser tests run without workaround code in `web-go`.

## Test Plan

- Suggested test layer: runtime/session browser interaction test and a page-level regression test that uses ordinary manual inputs and selects.
- Regression coverage: a page with an `input` listener, a `change` listener on a select, and a `paste` listener on the same input should execute all three actions in a bounded classic-JS slice.
- Mock or fixture needs: a minimal HTML page with one text input, one select, and one paste listener.

## Notes

- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- This issue is intentionally separate from the earlier paste-clipboard payload report because the blocker observed here is the `HTMLInputElement` surface gate itself.
- The CSV structural mismatch case in the same `web-go` test run was not included here because it surfaced as a generic page parse failure instead of the browser-surface error above.
