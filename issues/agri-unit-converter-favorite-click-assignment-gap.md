# Issue Template

## Summary

- Short summary: `#favorite-current-button` click can fail with an unsupported assignment error in the bounded classic-JS slice.

## Context

- Owning subsystem: `Runtime` / event dispatch
- Related capability or gap: bounded click dispatch on JS-driven controls
- Related docs: `doc/subsystem-map.md`, `doc/capability-matrix.md`, `doc/roadmap.md`

## Problem

- Current behavior: Clicking `#favorite-current-button` during the agri-unit-converter regression can return `event: unsupported: assignment only works on object, array, or host surface values in this bounded classic-JS slice`.
- Expected behavior: The click should complete and the page should update the favorite state without surfacing a harness runtime error.
- Reproduction steps:
  1. Build a harness for `/tools/agri/agri-unit-converter/` with `?cat=spray&v=250&from=L_ha&to=gal_acre&gal=us&fo=1`.
  2. Click `#favorite-current-button`.
  3. Observe the unsupported assignment error from the click path.
- Reproduction code:

```go
harness := mustAgriUnitConverterHarness(t, agriUnitConverterPageURL+"?cat=spray&v=250&from=L_ha&to=gal_acre&gal=us&fo=1")
_ = harness.Click("#favorite-current-button")
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run '^TestAgriUnitConverterBRT012FavoriteFilterShowsOnlyStarredPairs$' -count=1
```
- Scope / non-goals: This is a browser-tester-go control-flow gap. Do not change the page behavior here; fix the harness so the click path can run cleanly.

## Acceptance Criteria

- [ ] Clicking `#favorite-current-button` completes without the unsupported assignment error.
- [ ] The event dispatch path still preserves click semantics for the page.
- [ ] The Go regression can stop allowing this error as a known failure.

## Test Plan

- Suggested test layer: `web-go/internal/generate` regression test.
- Regression or failure-path coverage: keep the current Go regression and remove the error allowance once fixed.
- Mock or fixture needs: none beyond the existing harness.

## Notes

- This same control path may affect other JS-driven action buttons if they share the same bounded assignment path.
