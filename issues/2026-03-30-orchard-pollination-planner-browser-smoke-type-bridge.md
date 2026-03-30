# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: `TestAgriOrchardPollinationPlannerBrowserSmoke` is blocked by missing `elm.type` reflection in the classic-JS runtime slice.

## Context

- Owning subsystem: Runtime
- Related capability or gap: The element-reference bridge needs to expose the standard `type` property for form controls so page bootstrap helpers can distinguish checkbox inputs from text-like inputs without tripping the bounded classic-JS surface.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: The orchard pollination planner browser smoke test fails during bootstrap, before any browser assertions run. The shared `setValue()` helper in `web/content/pages/tools/agri/orchard-pollination-planner/template.html:1207` branches on `elm.type === "checkbox"`, and the runtime reports an unsupported surface error for that property access.
- Expected behavior: Reading `element.type` on standard form controls should succeed and return the reflected control type, so bootstrap helpers can branch on checkbox inputs without throwing.
- Reproduction steps:
  1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run `go test ./internal/generate -run TestAgriOrchardPollinationPlannerBrowserSmoke -count=1`.
  2. The page bootstrap reaches the shared `setValue()` helper before any test assertions run.
  3. The helper reads `elm.type` and the browser tester reports an unsupported browser surface for that element property.
- Reproduction code:

```go
package main

import "browsertester"

func main() {
	h, err := browsertester.FromHTML(`<main>
  <input id="field" type="checkbox" checked>
  <div id="out"></div>
  <script>
    function setValue(elm, value) {
      if (elm.type === "checkbox") {
        elm.checked = Boolean(value);
        return;
      }
      elm.value = value === null || value === undefined ? "" : String(value);
    }
    setValue(document.getElementById("field"), false);
    document.getElementById("out").textContent = "done";
  </script>
</main>`)
	if err != nil {
		panic(err)
	}
	_ = h
}
```

- Original failed command:

```bash
go test ./internal/generate -run TestAgriOrchardPollinationPlannerBrowserSmoke -count=1
```
- Scope / non-goals: Add `type` reflection for standard form-control element references in `browser-tester-go`. Do not work around this in finitefield-site by replacing the `elm.type` guard.

## Acceptance Criteria

- [ ] `element.type` reads on standard form controls no longer produce an unsupported-surface error.
- [ ] The orchard pollination planner browser smoke test reaches its assertions.
- [ ] Regression coverage exists for a minimal `elm.type` read in a bootstrap helper.

## Test Plan

- Suggested test layer: `internal/runtime` bootstrap regression test.
- Regression or failure-path coverage: Add a minimal HTML bootstrap case that calls a helper using `elm.type`.
- Mock or fixture needs: None.

## Notes

- Links, screenshots, logs, or other context:
- Working directory for the failed command: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
