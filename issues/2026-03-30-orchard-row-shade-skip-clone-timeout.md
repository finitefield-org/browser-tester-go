# Issue Template

Use this file as the starting point for a new issue draft.
Keep the scope to one capability or one concrete bug when possible.

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: The orchard row shade simulator still times out because skipped classic-JS evaluation clones nested bindings too deeply while handling `TypeText()`.

## Context

- Owning subsystem: `internal/script` classic-JS parser/runtime, reached through `internal/runtime` event dispatch.
- Related capability or gap: short-circuit parsing and skipped-branch evaluation in `cloneForSkipping()` / `cloneSkipped()`.
- Related docs: `doc/subsystem-map.md`, `doc/implementation-guide.md`, `doc/capability-matrix.md`

## Problem

- Current behavior: `TestAgriOrchardRowShadeSimulatorBRT004RepresentativeInputsRenderHeatmapAndMetrics` still times out after `Harness.TypeText()` in `finitefield-site/web-go`. The stack is now in `sanitizeSkippedMapState`, `sanitizeSkippedValueShallow`, `cloneSkipped`, `cloneForSkipping`, `parseLogicalAnd`, `parseLogicalOr`, `invokeArrowFunction`, and `runtime.(*Session).TypeText`.
- Expected behavior: `TypeText()` should finish quickly, even when the page's `input` listener contains short-circuit expressions that trigger skipped evaluation.
- Reproduction steps:
  1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run the orchard row shade simulator browser test.
  2. Wait for `BRT004` to reach `Harness.TypeText()`.
  3. Let the 60s timeout expire and inspect the stack trace.
- Reproduction code:

```text
The failure is reproduced by the orchard row shade simulator page itself, not by a finitefield-site workaround.
The observed flow is:
`fillOrchardRowShadeCoreInputs(...)` -> `Harness.TypeText(...)` -> `Session.TypeText(...)`
-> classic-JS skipped-branch parsing via `cloneForSkipping(...)`
-> deep value sanitization in `sanitizeSkippedMapState(...)`.

The remaining blocker is different from the previously fixed self-referential array binding case; that regression test passes, but this browser flow still hangs in skipped-clone machinery.
```

- Original failed command:

```bash
go test -v -timeout 60s ./internal/generate -run 'TestAgriOrchardRowShadeSimulatorBRT004RepresentativeInputsRenderHeatmapAndMetrics' -count=1
```

- Scope / non-goals: Fix the skipped-classic-JS clone/evaluation path in `browser-tester-go`. Do not change the finitefield-site page to avoid the browser engine path.

## Acceptance Criteria

- [ ] `TypeText()` no longer spends the full timeout in skipped classic-JS clone/evaluation paths.
- [ ] The orchard row shade simulator BRT004 test completes successfully.
- [ ] Regression coverage exists for a minimal skipped short-circuit / input-listener case.

## Test Plan

- Suggested test layer: `internal/script` regression test, with an `internal/runtime` integration test if needed.
- Regression or failure-path coverage: Add a focused test that exercises a short-circuit expression inside a listener and verifies the skip-evaluation clone path does not recurse through nested map/set state.
- Mock or fixture needs: A minimal HTML fixture with a text input, an `input` listener, and a short-circuit expression that forces `cloneForSkipping()`.

## Notes

- Links, screenshots, logs, or other context:
- Working directory for the failed command: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
- The top of the timeout stack repeatedly points to `sanitizeSkippedMapState(0x0)` and `cloneSkipped`, which is why this is still a `browser-tester-go` issue rather than a finitefield-site assertion failure.
