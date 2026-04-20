# Issue Draft

## Summary

- Short summary: `select` elements without an explicit `selected` option report an empty value in the Go browser runtime instead of the browser default first-option value.

## Context

- Owning subsystem: DOM value reflection
- Related capability or gap: `select` value reflection, `AssertValue`, initial form control state
- Related docs:
  - `doc/capability-matrix.md`
  - `doc/implementation-guide.md`

## Problem

- Current behavior: In the browser-tester-go DOM store, a `<select>` with no `selected` attribute on any `<option>` returns `""` from `ValueForNode` and `AssertValue`, even though a real browser exposes the first option as the selected value.
- Expected behavior: The bounded browser runtime should reflect the browser default selection for single-select controls when no explicit `selected` attribute is present.
- Reproduction steps:
  1. Build a harness from the HTML below.
  2. Assert the value of `#mode`.
  3. Observe that the runtime returns an empty string instead of the first option.
- Reproduction code:

```text
<!doctype html><main>
  <select id="mode">
    <option value="auto">Auto</option>
    <option value="yaml">YAML</option>
  </select>
</main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run JSONYamlToml -count=1
```

- Scope / non-goals: Keep the fix focused on bounded DOM value reflection for native `<select>` controls; do not widen the runtime beyond the browser semantics needed by browser tests.

## Acceptance Criteria

- [ ] `ValueForNode` for a single-select without any explicit `selected` attribute returns the first option value.
- [ ] `AssertValue` on such a select matches browser default behavior.
- [ ] Regression coverage is added for the default-selected option case.

## Test Plan

- Suggested test layer: DOM value reflection and runtime assertions
- Regression or failure-path coverage: a select with no explicit `selected` attributes
- Mock or fixture needs: none

## Notes

- This issue was confirmed from a browser-tester-go-only repro, not from the finitefield-site app code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
