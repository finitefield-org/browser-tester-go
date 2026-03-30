# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: Classic scripts reject `var` declarations with an unsupported browser surface error.

## Context

- Owning subsystem: Script
- Related capability or gap: Classic-JS parsing/evaluation should accept `var` declarations, but the runtime rejects them during browser bootstrap.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: A classic script that starts with `var root = ...` fails with `unsupported browser surface "var" in this bounded classic-JS slice`.
- Expected behavior: `var` declarations should parse and execute like they do in a browser, so existing pages can bootstrap normally.
- Reproduction steps:
  1. `cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
  2. Run `go test ./internal/generate -run SeedingRateCalculator -count=1`
  3. The page fails before interaction, on the first click/open action, because the bootstrap script is rejected at the `var` declaration.
- Reproduction code:

```text
package main

import "browsertester"

func main() {
	h, err := browsertester.FromHTML(`<main><div id="out"></div><script>
var value = 1;
document.getElementById("out").textContent = String(value);
</script></main>`)
	if err != nil {
		panic(err)
	}
	_, _ = h, h
}
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run SeedingRateCalculator -count=1
```

- Scope / non-goals: Fix classic-JS `var` declaration handling in browser-tester-go. Do not change finitefield-site page code to work around the missing feature.

## Acceptance Criteria

- [ ] Primary behavior is implemented or fixed.
- [ ] Failure paths are explicit and do not silently fall back.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: `internal/script` parser/evaluator tests plus a bootstrap regression test.
- Regression or failure-path coverage: Add a minimal classic-script case containing `var` and verify it executes.
- Mock or fixture needs: None.

## Notes

- Links, screenshots, logs, or other context:
- Working directory for the failed command: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
