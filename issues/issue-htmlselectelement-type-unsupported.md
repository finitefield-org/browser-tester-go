# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: `HTMLSelectElement.type` is still unsupported during classic-JS bootstrap.

## Context

- Owning subsystem: Runtime
- Related capability or gap: The element-reference bridge still needs to expose the standard `type` property for `select` elements, so bootstrap helpers can inspect form controls without tripping the bounded classic-JS surface.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: Reading `select.type` can fail with an unsupported browser surface error in the bounded classic-JS slice.
- Expected behavior: `HTMLSelectElement.type` should reflect the standard control type, returning `select-one` or `select-multiple`.
- Reproduction steps:
  1. Run a minimal bootstrap case that reads `document.getElementById("field").type` from a `<select>` element.
  2. Observe that the helper fails before initialization completes if the surface is unsupported.
- Reproduction code:

```go
package main

import "browsertester"

func main() {
	h, err := browsertester.FromHTML(`<main>
  <select id="field">
    <option value="one">one</option>
  </select>
  <div id="out"></div>
  <script>
    function report(elm) {
      document.getElementById("out").textContent = elm.type;
    }
    report(document.getElementById("field"));
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
go test ./internal/runtime -run TestSessionInlineScriptsCanReadSelectTypeReflectionSurfaces -count=1
```

- Scope / non-goals: Add `type` reflection for `HTMLSelectElement` in browser-tester-go. Do not work around this in application code by avoiding `select.type`.

## Acceptance Criteria

- [ ] `select.type` reads no longer produce an unsupported-surface error.
- [ ] The reflected value matches `select-one` or `select-multiple`.
- [ ] Regression coverage exists for a minimal `select.type` read.

## Test Plan

- Suggested test layer: `internal/runtime` bootstrap regression test.
- Regression or failure-path coverage: Add a minimal HTML bootstrap case that reads `select.type`.
- Mock or fixture needs: None.

## Notes

- Links, screenshots, logs, or other context:
