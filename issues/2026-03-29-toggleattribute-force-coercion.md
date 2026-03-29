# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: `toggleAttribute(name, force)` rejects truthy non-boolean values, so pages that pass a standard JS value into `force` fail even though browsers coerce the argument.

## Context

- Owning subsystem: Runtime
- Related capability or gap: Element attribute toggling and force-argument coercion
- Related docs:
  - `/Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go/doc/subsystem-map.md`
  - `/Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go/doc/capability-matrix.md`
  - `/Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go/doc/implementation-guide.md`

## Problem

- Current behavior: `button.toggleAttribute("data-active", "yes")` fails or validates too strictly instead of treating the string as truthy.
- Expected behavior: The `force` argument should be coerced with normal JS truthiness rules, matching browser DOM behavior.
- Reproduction steps:
  1. Run the failed command below from `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`.
  2. Load the `irrigation-water-guide` browser test harness.
  3. The inline script calls `toggleAttribute` with a truthy string during page bootstrap and the session aborts.
- Reproduction code:

```go
package main

import "browsertester"

func main() {
	h := browsertester.FromHTML(`<!doctype html><main><button id="mode"></button><script>const button = document.querySelector("#mode"); button.toggleAttribute("data-active", "yes");</script></main>`)
	_ = h.TextContent("#mode")
}
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run IrrigationWaterGuide -count=1 -v
```
- Scope / non-goals:
  - Scope: `toggleAttribute` should coerce `force` values instead of requiring a literal boolean.
  - Non-goals: changing finitefield-site to double-negate every `toggleAttribute` call, or adding a page-side workaround.

## Acceptance Criteria

- [ ] `toggleAttribute("data-active", "yes")` treats the string as truthy and adds the attribute.
- [ ] `toggleAttribute("data-active", 0)` treats the number as falsy and removes the attribute.
- [ ] Tests are added or updated at the runtime layer.

## Test Plan

- Suggested test layer: `internal/runtime`
- Regression or failure-path coverage: bootstrap a page that toggles an attribute with a string `force` value
- Mock or fixture needs: none

## Notes

- This blocked `finitefield-site/web-go` while trying to build browser tests from `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
