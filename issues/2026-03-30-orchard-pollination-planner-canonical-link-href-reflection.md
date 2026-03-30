# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: Classic-JS bootstrap still throws on `link[rel="canonical"].href` for canonical links.

## Context

- Owning subsystem: Runtime
- Related capability or gap: The element-reference bridge in `internal/runtime` exposes `href` for `a` and `area`, but not for `link` elements. The shared base template reads `canonicalLink.href` during bootstrap, so canonical links need the same reflected URL surface.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: The orchard pollination planner browser smoke test fails during bootstrap before any assertions run. The shared helper in `web/content/shared/base/template.html:69-71` executes `const canonicalHref = canonicalLink && canonicalLink.href ? canonicalLink.href : ""`, and the runtime reports `unsupported browser surface "element:31.href" in this bounded classic-JS slice`.
- Expected behavior: Reading `href` from a standard canonical `<link rel="canonical">` element should succeed and return the resolved absolute URL, just like browser behavior.
- Reproduction steps:
  1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run `go test ./internal/generate -run TestAgriOrchardPollinationPlanner -count=1`.
  2. Bootstrap reaches the shared base template script before any browser assertions run.
  3. The `canonicalLink.href` read throws, so the smoke test never reaches the tool assertions.
- Reproduction code:

```go
package main

import "browsertester"

func main() {
	h, err := browsertester.FromHTML(`<!doctype html>
<html>
  <head>
    <link rel="canonical" href="/tools/agri/orchard-pollination-planner/">
  </head>
  <body>
    <div id="out"></div>
    <script>
      const link = document.querySelector('link[rel="canonical"]');
      document.getElementById("out").textContent = link.href;
    </script>
  </body>
</html>`)
	if err != nil {
		panic(err)
	}
	_ = h
}
```

- Original failed command:

```bash
go test ./internal/generate -run TestAgriOrchardPollinationPlanner -count=1
```
- Scope / non-goals: Add `href` reflection for `HTMLLinkElement` in `browser-tester-go`. Do not work around this in `finitefield-site` by removing the canonical-link read or rewriting the shared base template.

## Acceptance Criteria

- [ ] `link[rel="canonical"].href` reads no longer produce an unsupported-surface error.
- [ ] The orchard pollination planner browser smoke test reaches its assertions.
- [ ] Regression coverage exists for a minimal canonical-link `href` read.

## Test Plan

- Suggested test layer: `internal/runtime` bootstrap regression test.
- Regression or failure-path coverage: Add a minimal HTML bootstrap case that reads `link[rel="canonical"].href`.
- Mock or fixture needs: None.

## Notes

- Links, screenshots, logs, or other context:
- Working directory for the failed command: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
