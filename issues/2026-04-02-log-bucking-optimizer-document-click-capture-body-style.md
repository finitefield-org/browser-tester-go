# `log-bucking-optimizer` open-button click fails when a document capture listener and a direct button listener both toggle body overflow

## Summary

- Short summary: The `log-bucking-optimizer` open button throws `assignment to "element:...@body" is unsupported` when a document-level capture listener and the button's own click listener both call the dialog opener.

## Context

- Owning subsystem: Runtime / click dispatch
- Related capability or gap: click event propagation with capture listeners plus DOM mutation in the same turn
- Related docs:
  - `internal/runtime/events.go`
  - `internal/runtime/session.go`
  - `../finitefield-site/web/content/pages/tools/forestry/log-bucking-optimizer/template.html`
- Affected finitefield-site coverage:
  - `../finitefield-site/web-go/internal/generate/forestry_log_bucking_optimizer_browser_test.go`

## Problem

- Current behavior: clicking `#log-bucking-open-button` aborts with `event: unsupported: assignment to "element:60@body" is unsupported in this bounded classic-JS slice` even though `document.body.style.overflow = "hidden"` is otherwise supported.
- Expected behavior: capture-phase document handlers and target-phase button handlers should both be able to run, with the second handler observing the DOM state set by the first handler.
- Reproduction steps:
  1. Run the `log-bucking-optimizer` browser tests.
  2. Click the open button in the generated page.
  3. Observe the unsupported assignment error before the dialog assertion completes.
- Reproduction source:

```text
Generated HTML file:
/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/build/en/tools/forestry/log-bucking-optimizer/index.html
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestForestryLogBuckingOptimizerBRT001OpenButtonShowsFullscreenDialog$' -count=1
```

## Acceptance Criteria

- [ ] Clicking a button with both a document capture listener and a button listener no longer throws an unsupported assignment error.
- [ ] The `log-bucking-optimizer` open button browser test passes.
- [ ] Regression coverage is added for the capture-plus-target click path.

## Test Plan

- Suggested test layer: runtime click-dispatch regression test.
- Regression or failure-path coverage: a minimal page that toggles `document.body.style.overflow` from both a capture listener and a direct button listener.
- Mock or fixture needs: none

## Notes

- This issue is separate from the parser failure above.
