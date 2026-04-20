# `HTMLInputElement`-guarded input handlers block tolerance-checker browser tests

## Summary

- Short summary: The bounded classic-JS slice rejects `HTMLInputElement` at runtime when the tolerance-checker page handles manual table input and CSV mapping changes, so `TypeText()` / `SetSelectValue()`-driven browser tests cannot proceed.

## Context

- Owning subsystem: `bt-runtime` browser-global bridge / constructor globals
- Related capability or gap: `HTMLInputElement` support inside bounded classic-JS slices that execute inline `input`/`change` handlers
- Related docs: `browser-tester-go/README.md`, `browser-tester-go/doc/capability-matrix.md`

## Problem

- Current behavior: When `finitefield-site/web-go/internal/generate/construction_tolerance_checker_browser_test.go` types into the manual grid or changes CSV mapping controls, the page's inline handlers hit `unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice`.
- Expected behavior: The page should be able to process normal input and change events in the browser harness without constructor-gating failures.
- Reproduction steps:
  1. Load the tolerance-checker page in the browser harness.
  2. Focus a manual measurement field and call `TypeText()`, or change a CSV mapping unit select with `SetSelectValue()`.
  3. Observe the unsupported browser surface error.
- Reproduction code:

```html
<main>
  <input id="side-0" type="text" value="">
  <script>
    document.getElementById("side-0").addEventListener("input", (event) => {
      if (!(event.target instanceof HTMLInputElement)) return;
      event.target.value = event.target.value.trim();
    });
  </script>
</main>
```

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run 'TestConstructionToleranceChecker' -count=1
```

- Scope / non-goals: This issue is about the browser harness/runtime constructor gap, not about changing the site code to avoid `instanceof HTMLInputElement`.

## Acceptance Criteria

- [ ] `HTMLInputElement` is available in the bounded classic-JS slice used by browser tests.
- [ ] `input` and `change` handlers can inspect `event.target` as an `HTMLInputElement` without throwing.
- [ ] `TypeText()` and `SetSelectValue()` can drive tolerance-checker input flows without runtime surface errors.

## Test Plan

- Suggested test layer: runtime/browser-global contract test plus a page-level regression test.
- Regression coverage: verify a page with an `input` listener guarded by `instanceof HTMLInputElement` can accept typed text and continue updating state.
- Mock or fixture needs: a minimal HTML page with an `input` listener and one `<select>`-driven change handler if select coverage is added.

## Notes

- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- This is a separate report from the earlier HTMLInputElement issue because it was observed while exercising the tolerance-checker browser tests.
