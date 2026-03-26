# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- [x] Review `doc/subsystem-map.md` and identify the owning subsystem.
- [x] Review `doc/implementation-guide.md` and pick the test layer first.
- [x] Review `doc/capability-matrix.md` and confirm the capability row or gap.
- [x] Review `doc/roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: `agri-unit-converter` cannot run from raw HTML because the classic-JS runtime still stops at browser globals like `document`.

## Context

- Owning subsystem: `internal/script`
- Related capability or gap: Raw HTML inline bootstrap execution against browser globals without stripping `<script>` tags first
- Related docs: `doc/implementation-guide.md`, `doc/capability-matrix.md`, `doc/roadmap.md`

## Problem

- Current behavior: The `web-go` regression for `/tools/agri/agri-unit-converter/` only passes after removing every `<script>` block from the generated HTML. That means browser-tester-go can verify the DOM shell, but it cannot exercise the page's production bootstrap as generated.
- Root cause: The current classic-JS runtime exposes the `host` bridge, but not the browser-global surface that this page's inline script expects. The first concrete failure is `unsupported identifier "document"` from `document.getElementById(...)`. The same bootstrap also depends on `window.location`, `navigator.onLine`, `new URL(...)`, `Intl.NumberFormat(...)`, `history.replaceState(...)`, storage, media-query, clipboard, and timer APIs, so a raw-HTML path cannot work until those globals are bridged or explicitly shimmed.
- Expected behavior: `FromHTMLWithURL(...)` should be able to load the generated page HTML as-is and run the bootstrap deterministically. If a browser surface is still out of scope, the runtime should fail with a typed unsupported error that names that surface directly, instead of forcing the caller to strip scripts as a workaround.
- Reproduction steps:
  1. Render the generated HTML for `/tools/agri/agri-unit-converter/`.
  2. Call `FromHTMLWithURL(...)` on the raw HTML without removing any `<script>` tags.
  3. Observe `ErrorKindDOM` with `unsupported identifier "document"` when the bootstrap reaches `document.getElementById(...)`.
- Reproduction code:

```text
const root = document.getElementById("agri-unit-converter-root");
const hasSearch = window.location.search && window.location.search.length > 1;
const url = new URL(window.location.href);
const isOffline = !navigator.onLine;
const formatted = new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(1.23);
window.history.replaceState({}, "", url.toString());
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run TestAgriUnitConverter -count=1
```

- Scope / non-goals: Do not widen the whole browser platform in one step. Start with the smallest browser-global bridge needed to make the raw HTML bootstrap work, and keep unsupported surfaces explicit.

## Acceptance Criteria

- [ ] The agri-unit-converter page can be loaded through `FromHTMLWithURL(...)` without stripping scripts first.
- [ ] The first missing browser-global surface is reported as a typed unsupported error, not as a silent fallback.
- [ ] `web-go` can replace the script-stripping workaround with a raw HTML regression test.
- [ ] Regression or failure-path tests are added at the chosen layer.

## Test Plan

- Suggested test layer: `internal/script` first, then `internal/runtime`
- Regression or failure-path coverage: Add a raw-HTML bootstrap regression for `agri-unit-converter`
- Mock or fixture needs: A minimal HTML fixture with `document`, `window.location`, `navigator.onLine`, `new URL(...)`, and `Intl.NumberFormat(...)`

## Notes

- Links, screenshots, logs, or other context: The current `web-go` workaround lives in [`web-go/internal/generate/agri_unit_converter_browser_test.go`](../../finitefield-site/web-go/internal/generate/agri_unit_converter_browser_test.go). It strips every `<script>` block before constructing the harness, which hides this gap rather than fixing it.
