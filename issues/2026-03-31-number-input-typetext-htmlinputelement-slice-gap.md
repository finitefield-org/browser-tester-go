# `TypeText()` on number inputs still hits unsupported `HTMLInputElement` in the bounded classic-JS slice

## Summary

- Re-running the scaffolding area calculator browser tests after the browser-tester-go fix still fails at the first `TypeText()` call on a polygon-row number input.
- The failure comes from `browser-tester-go`, not from `finitefield-site/web-go`, because the error is raised inside the harness while typing into a plain `<input>`.

## Context

- Affected repository: `finitefield-site`
- Affected test file: `web-go/internal/generate/construction_scaffolding_area_calculator_browser_test.go`
- Failing tests:
  - `TestConstructionScaffoldingAreaCalculatorBRT004PolygonEstimateSumsRows`
  - `TestConstructionScaffoldingAreaCalculatorBRT005PolygonMissingRowShowsError`
- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`

## Problem

- Current behavior: `Harness.TypeText("#scaffolding-side-0", "12")` fails with `dom: unsupported: unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice`.
- Expected behavior: typing into a normal number input should work without requiring any workaround in the page test.
- Why this matters: the scaffolding calculator page uses polygon-row number inputs, so this browser-surface gap blocks exact browser coverage for the test cases.

## Reproduction

1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run:

```bash
go test ./internal/generate -run ScaffoldingAreaCalculator -count=1
```

2. Observe the first failure on BRT-004 or BRT-005 when the harness tries to type into `#scaffolding-side-0`.
3. The page never reaches the polygon total or validation assertions because the harness aborts first.

## Observed Output

```text
--- FAIL: TestConstructionScaffoldingAreaCalculatorBRT004PolygonEstimateSumsRows (0.03s)
    construction_scaffolding_area_calculator_browser_test.go:246: TypeText(#scaffolding-side-0, 12) error = dom: unsupported: unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice
--- FAIL: TestConstructionScaffoldingAreaCalculatorBRT005PolygonMissingRowShowsError (0.03s)
    construction_scaffolding_area_calculator_browser_test.go:292: TypeText(#scaffolding-side-0, 12) error = dom: unsupported: unsupported browser surface "HTMLInputElement" in this bounded classic-JS slice
```

## Why This Is browser-tester-go

- The error is thrown from `Harness.TypeText()`, before any application-specific calculation or DOM assertion runs.
- The page under test is only using standard browser inputs; there is no `web-go`-specific logic involved in the failing step.
- The unsupported surface name is `HTMLInputElement`, which points to a missing browser-tester-go runtime capability in the bounded classic-JS execution slice.

## Impact

- Polygon-row browser tests cannot use the real input path.
- Any test that depends on typing into number inputs and letting page listeners inspect the input target via browser globals will continue to fail.

## Acceptance Criteria

- `TypeText()` works on number inputs in the bounded classic-JS slice.
- `HTMLInputElement` is available where needed for input event handling.
- A regression test covers typing into a number input and reading the resulting value through the page listener.

## Notes

- This issue was re-confirmed on 2026-03-31 after the browser-tester-go fix was applied.
- No workaround was used in the `finitefield-site` test code.
- Earlier related issue: `2026-03-31-html-input-element-constructor-gap.md`
