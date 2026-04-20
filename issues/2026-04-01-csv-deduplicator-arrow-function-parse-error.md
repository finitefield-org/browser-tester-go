# `csv-deduplicator` page source trips the classic JS parser after an arrow function

## Summary

- Short summary: The generated `csv-deduplicator` page fails in browser tests with `parse: unterminated parenthesized expression after \`arrow function\`` during DOM bootstrap.

## Context

- Owning subsystem: Runtime / classic JS parser
- Related capability or gap: parsing of the generated `csv-deduplicator` inline classic script
- Related docs:
  - `internal/script/classic_script.go`
  - `internal/runtime/browser_globals_bootstrap_test.go`
- Affected finitefield-site coverage:
  - `../finitefield-site/web-go/internal/generate/data_csv_deduplicator_browser_test.go`

## Problem

- Current behavior: `Session.ensureDOM()` fails for the generated `csv-deduplicator` HTML with a parse error after an arrow function, even when the standard Lucide external JS mock is seeded.
- Expected behavior: The page script should parse and bootstrap normally so browser tests can exercise the tool flow.
- Reproduction steps:
  1. Build the finitefield-site page at `/tools/data/csv-deduplicator/`.
  2. Seed `https://unpkg.com/lucide@latest` with the same stub used by `web-go`.
  3. Run `ensureDOM()` on the generated HTML.
  4. Observe the parse error.
- Reproduction source:

```text
Generated HTML file:
/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/build/en/tools/data/csv-deduplicator/index.html
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestCSVDeduplicator' -count=1
```

- Browser-tester repro command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go
go test ./internal/runtime -run '^TestCSVDeduplicatorEnsureDOM$' -count=1
```

- Scope / non-goals: Keep the fix focused on the classic JS parser or bootstrap path that blocks this page. Do not change the finitefield-site page just to work around the harness.

## Acceptance Criteria

- [ ] `Session.ensureDOM()` succeeds on the generated `csv-deduplicator` HTML with the Lucide mock.
- [ ] `go test ./internal/generate -run '^TestCSVDeduplicator' -count=1` passes from `finitefield-site/web-go`.
- [ ] Regression coverage is added for the page HTML at the browser-bootstrap layer.

## Test Plan

- Suggested test layer: runtime/browser bootstrap test.
- Regression or failure-path coverage: one smoke test that loads the generated `csv-deduplicator` HTML and verifies DOM bootstrap succeeds.
- Mock or fixture needs: the standard Lucide external JS mock used by `web-go`.

## Notes

- This issue was confirmed in `browser-tester-go`, not in the finitefield-site page code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
