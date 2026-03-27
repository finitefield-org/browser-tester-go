# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- [x] Review `doc/subsystem-map.md` and identify the owning subsystem.
- [x] Review `doc/implementation-guide.md` and pick the test layer first.
- [x] Review `doc/capability-matrix.md` and confirm the capability row or gap.
- [x] Review `doc/roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: `element.dataset` is not available to the bounded browser surface, so `chill-hours-portions-calculator` cannot run its inline bootstrap on full generated HTML.

## Context

- Owning subsystem: `internal/dom`
- Related capability or gap: Bounded `element.dataset` reads for inline browser bootstrap on generated pages that use `data-*` driven controls
- Related docs: `doc/implementation-guide.md`, `doc/capability-matrix.md`, `doc/roadmap.md`

## Problem

- Current behavior: The `web-go` regression for `/tools/agri/chill-hours-portions-calculator/` fails as soon as the inline bootstrap touches `button.dataset` on elements such as `data-model-mode`, `data-source-mode`, and `data-round-mode` controls. The runtime reports `unsupported browser surface "element:62.dataset"` from the classic-JS slice.
- Root cause: The browser bridge exposes classes, text, and basic attributes, but not the `dataset` view that generated pages use heavily for control wiring.
- Expected behavior: Raw generated HTML should be able to load without stripping scripts first, or unsupported `dataset` access should be surfaced as a first-class, explicit gap before page bootstrap starts.
- Reproduction steps:
  1. Render the generated HTML for `/tools/agri/chill-hours-portions-calculator/`.
  2. Load the raw HTML into `FromHTMLWithURL(...)` without removing any `<script>` tags.
  3. Observe the unsupported `element:62.dataset` error during bootstrap.
- Reproduction code:

```text
button.dataset.modelMode
button.dataset.sourceMode
button.dataset.roundMode
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run TestAgriChillHoursPortionsCalculator -count=1
```

- Scope / non-goals: Do not widen the whole browser bridge in one step. Start with `dataset` reads on element surfaces used by template-driven controls, and keep the unsupported surface explicit if it remains out of scope.

## Acceptance Criteria

- [ ] The chill-hours page can load through `FromHTMLWithURL(...)` without stripping scripts first.
- [ ] `element.dataset` reads are available for the page's inline bootstrap controls.
- [ ] `web-go` can replace the script-stripping workaround with a raw HTML regression test.
- [ ] Regression or failure-path tests are added at the chosen layer.

## Test Plan

- Suggested test layer: `internal/dom` first, then `internal/runtime`
- Regression or failure-path coverage: Add a raw-HTML bootstrap regression for `chill-hours-portions-calculator`
- Mock or fixture needs: A minimal generated HTML fixture that exercises `button.dataset` access on `data-*` controls

## Notes

- Links, screenshots, logs, or other context: The current `web-go` workaround lives in [`finitefield-site/web-go/internal/generate/agri_chill_hours_portions_calculator_browser_test.go`](../../finitefield-site/web-go/internal/generate/agri_chill_hours_portions_calculator_browser_test.go), which strips every `<script>` block before constructing the harness so the DOM shell can still be tested.
