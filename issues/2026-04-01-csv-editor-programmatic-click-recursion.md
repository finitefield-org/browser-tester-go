# Issue Draft

## Summary

- Short summary: `csv-editor` browser tests hit a stack overflow when a click handler calls `element.click()` on another element, especially the hidden file input opened from the main CTA.

## Context

- Owning subsystem: Runtime
- Related capability or gap: event dispatch / programmatic click handling on nested browser surfaces
- Related docs:
  - `../finitefield-site/doc/tools/data/csv-editor.md`
  - `doc/implementation-guide.md`

## Problem

- Current behavior: The `csv-editor` open button handler runs `setDialogOpen(true)` and then calls `el.fileInput?.click()`. In browser-tester-go, that path recurses through `dispatchEventListenersWithPropagation -> browserElementClick -> InvokeCallableValue` until the Go runtime overflows the goroutine stack.
- Expected behavior: Calling `click()` from inside a click handler should dispatch one additional synthetic click on the target element and then return. It should not re-enter the originating handler recursively.
- Reproduction steps:
  1. Run the `csv-editor` browser test suite.
  2. Let the first test click `#csv-editor-open-button`.
  3. Observe an infinite recursion / stack overflow before the dialog assertion can complete.
- Reproduction command:

```bash
go test ./internal/generate -run 'Test.*CSVEditor|Test.*csvEditor|TestCSVEditor' -count=1
```

- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- Observed failure:
  - `runtime: goroutine stack exceeds 1000000000-byte limit`
  - recursive frames alternate between `dispatchEventListenersWithPropagation`, `browserElementClick`, and `invokeArrowFunction`
- Scope / non-goals:
  - Scope is the bounded click / event-dispatch path used by browser tests.
  - Non-goal: changing the `csv-editor` page to avoid `element.click()`.

## Acceptance Criteria

- [ ] A button click handler can call `otherElement.click()` without recursively re-entering the originating handler.
- [ ] Clicking the `csv-editor` open button no longer overflows the stack.
- [ ] The fix is covered by a regression test in browser-tester-go.

## Test Plan

- Suggested test layer: runtime/browser event-dispatch regression test
- Regression or failure-path coverage: a minimal page with `button.addEventListener("click", () => input.click())`
- Mock or fixture needs: none

## Notes

- This issue is separate from the existing file/DataTransfer/drag-drop gap for `csv-editor`.
- The failure was observed from the `csv-editor` browser test run, not from finitefield-site application code.
