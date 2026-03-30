# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: Classic-JS bootstrap still throws when a standard button assigns `tabIndex`.

## Context

- Owning subsystem: Runtime
- Related capability or gap: The Runtime element-reflection mutation bridge does not handle `tabIndex` writes on standard button controls, so common segmented-control bootstrap code aborts before the page can finish initializing.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: The finitefield-site browser smoke test for `spray-application-rate` fails before any assertions run. `DOMError()` reports `unsupported: assignment to "element:798@button.tabIndex" is unsupported in this bounded classic-JS slice`. The page bootstrap hits this path in `web/content/pages/tools/agri/spray-application-rate/template.html:2116`, where `renderSegments()` sets `btn.tabIndex = isActive ? 0 : -1;`.
- Expected behavior: Assigning `tabIndex` on standard button controls should succeed in the bounded classic-JS slice so segmented controls can initialize without throwing and maintain keyboard focus order metadata.
- Reproduction steps:
  1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run `go test ./internal/generate -run 'TestAgriSprayApplicationRate' -count=1`.
  2. The test fails during bootstrap, before any browser assertions run.
  3. A minimal inline script that assigns `button.tabIndex` reproduces the same unsupported-assignment error.
- Reproduction code:

```go
package main

import "browsertester"

func main() {
	h, err := browsertester.FromHTML(`<main>
  <button id="seg" type="button" aria-selected="false">Segment</button>
  <script>
    const btn = document.getElementById("seg");
    btn.tabIndex = -1;
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
go test ./internal/generate -run 'TestAgriSprayApplicationRate' -count=1
```
- Scope / non-goals: Add support for `tabIndex` writes on standard button controls in the Runtime element-reflection bridge. Do not work around this in finitefield-site by removing the `tabIndex` assignment; the site relies on standard keyboard-navigation semantics for segmented buttons.

## Acceptance Criteria

- [ ] `button.tabIndex` assignments on standard controls no longer produce an unsupported-assignment error.
- [ ] The spray application rate browser smoke test reaches its assertions.
- [ ] Regression coverage exists for a minimal `button.tabIndex = ...` bootstrap case.

## Test Plan

- Suggested test layer: `internal/runtime` bootstrap regression test.
- Regression or failure-path coverage: Add a minimal HTML bootstrap case that assigns `tabIndex` on a button.
- Mock or fixture needs: None.

## Notes

- Links, screenshots, logs, or other context:
- Working directory for the failed command: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
