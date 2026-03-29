# Set constructor support in bounded classic-JS slice

## Summary

- Short summary: The bounded classic-JS slice does not support `Set` construction well enough for real inline scripts that rely on standard browser JS collection behavior.

## Context

- Owning subsystem: Script
- Related capability or gap: Bounded browser stdlib slice / iterable `Set` support
- Related docs: `doc/capability-matrix.md`, `doc/implementation-guide.md`

## Problem

- Current behavior: The NPK fertilizer calculator page in `finitefield-site` fails in browser-tester-go when its inline solver reaches `new Set()` / `new Set(iterable)`. The failure shows up as `Set constructor requires an array of values in this bounded classic-JS slice` or `cannot call undefined value in this bounded classic-JS slice`.
- Expected behavior: `Set` should be available in the bounded classic-JS slice with browser-standard construction from no arguments and from iterable inputs, so inline scripts can use it for visited-state tracking and deduplication.
- Reproduction steps:
  1. Run the following command from `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`.
  2. Observe the browser test failure when the NPK calculator sample/solver path executes.
- Reproduction code:

```text
const visited = new Set();
const nextSet = new Set(fixedSet);
const initialSet = new Set(base.capacities.map((cap, index) => cap <= EPS ? index : null).filter((value) => value != null));
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run 'TestAgriNpkFertilizerCalculatorBRT' -count=1
```
- Scope / non-goals: Do not change the finitefield-site page to avoid `Set`; the bounded classic-JS slice should support the browser-standard behavior directly.

## Acceptance Criteria

- [ ] `new Set()` works in classic-JS inline scripts.
- [ ] `new Set(iterable)` works for iterable inputs, including Set-to-Set copies.
- [ ] `Array.from(set)` works when `set` is a bounded `Set`.
- [ ] Regression coverage is added at the script/runtime layer.

## Test Plan

- Suggested test layer: Script/runtime regression test covering the classic-JS stdlib slice.
- Regression or failure-path coverage: Add a minimal inline-script reproduction that constructs `Set` with no arguments and from an iterable, then verifies the result is usable by `Array.from()`.
- Mock or fixture needs: None.

## Notes

- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- Related page source: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web/content/pages/tools/agri/npk-fertilizer-calculator/template.html`
