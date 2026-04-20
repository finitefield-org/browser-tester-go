# `log-bucking-optimizer` page source trips the classic JS parser after `return /.../` inside an arrow body

## Summary

- Short summary: The generated `log-bucking-optimizer` page fails in browser tests with `parse: unterminated quoted string in parenthesized expression` while scanning the `downloadCsv` helper.

## Context

- Owning subsystem: Runtime / classic JS parser
- Related capability or gap: parsing of arrow-function bodies that contain `return /regex/` after a `return` keyword
- Related docs:
  - `internal/script/classic_script.go`
  - `doc/tools/forestry/log-bucking-optimizer.md`
- Affected finitefield-site coverage:
  - `../finitefield-site/web-go/internal/generate/forestry_log_bucking_optimizer_browser_test.go`

## Problem

- Current behavior: `Session.ensureDOM()` fails for the generated `log-bucking-optimizer` HTML when the `downloadCsv` helper includes a `return /[",\n\t]/.test(text)` branch and a template literal escape branch.
- Expected behavior: The page script should parse and bootstrap normally so browser tests can exercise the tool flow.
- Reproduction steps:
  1. Build the finitefield-site page at `/tools/forestry/log-bucking-optimizer/`.
  2. Run the `log-bucking-optimizer` browser tests.
  3. Observe a parse error before the first browser interaction.
- Reproduction source:

```text
Generated HTML file:
/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/build/en/tools/forestry/log-bucking-optimizer/index.html
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestForestryLogBuckingOptimizer' -count=1
```

- Browser-tester repro command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go
go test ./internal/script -run '^TestClassicJSStatementParserPreservesRegexLiteralAfterReturnInParenthesizedSource$' -count=1
```

## Acceptance Criteria

- [ ] `Session.ensureDOM()` succeeds on the generated `log-bucking-optimizer` HTML with the Lucide mock.
- [ ] `go test ./internal/generate -run '^TestForestryLogBuckingOptimizer' -count=1` passes from `finitefield-site/web-go`.
- [ ] Regression coverage is added for the page source at the parser layer.

## Test Plan

- Suggested test layer: classic JS parser regression test.
- Regression or failure-path coverage: one smoke test that exercises `return /regex/` inside a parenthesized arrow body and a block body.
- Mock or fixture needs: none

## Notes

- This issue is separate from the click/event-dispatch gap also uncovered by `log-bucking-optimizer`.
