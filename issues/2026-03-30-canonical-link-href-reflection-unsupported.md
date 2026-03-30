# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: Classic-JS bootstrap still throws when the shared base template reads `link[rel="canonical"].href`.

## Context

- Owning subsystem: Runtime
- Related capability or gap: The Runtime element-reflection bridge only exposes `href` for `a` / `area`, but the shared base template reads the canonical `<link>` element's `href` during bootstrap. That standard reflection is missing, so every tool page that includes the base template can fail before assertions run.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: The finitefield-site browser smoke test for `density-mass-volume-calculator` fails before any assertions run. `DOMError()` reports `unsupported browser surface "element:31.href" in this bounded classic-JS slice`. The page bootstrap hits this path in `web/content/shared/base/template.html:71`, where `suppressInitialToolUrlWrites()` reads `const canonicalHref = canonicalLink && canonicalLink.href ? canonicalLink.href : "";`.
- Expected behavior: Reading `href` on the canonical `<link rel="canonical">` element should succeed and return the resolved URL, so the shared base bootstrap can decide whether the page is a tool page without throwing.
- Reproduction steps:
  1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run `go test ./internal/generate -run 'DensityMassVolume|ChemDensityMassVolume' -count=1`.
  2. The test fails during bootstrap, before any browser assertions run.
  3. A minimal inline script that reads `document.querySelector('link[rel="canonical"]').href` reproduces the same unsupported-surface error.
- Reproduction code:

```go
package main

import "browsertester"

func main() {
	h, err := browsertester.FromHTML(`<html>
  <head>
    <link rel="canonical" href="https://example.test/tools/density-mass-volume-calculator/">
  </head>
  <body>
    <main id="root"></main>
    <script>
      const canonicalLink = document.querySelector('link[rel="canonical"]');
      const canonicalHref = canonicalLink && canonicalLink.href ? canonicalLink.href : "";
      document.getElementById("root").textContent = canonicalHref;
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
go test ./internal/generate -run 'DensityMassVolume|ChemDensityMassVolume' -count=1
```
- Scope / non-goals: Add `href` reflection for the canonical `<link>` element in `browser-tester-go`. Do not work around this in finitefield-site by replacing the standard `link.href` read in the shared base template.

## Acceptance Criteria

- [ ] `link[rel="canonical"].href` reads no longer produce an unsupported-surface error.
- [ ] The density mass volume browser smoke test reaches its assertions.
- [ ] Regression coverage exists for a minimal canonical-link `href` read during bootstrap.

## Test Plan

- Suggested test layer: `internal/runtime` bootstrap regression test.
- Regression or failure-path coverage: Add a minimal HTML bootstrap case that reads `link[rel="canonical"].href`.
- Mock or fixture needs: None.

## Notes

- Links, screenshots, logs, or other context:
- Working directory for the failed command: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
