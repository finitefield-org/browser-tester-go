# Issue Template

## Summary

- Short summary: `Object.defineProperty` on `navigator.onLine` is unsupported in the bounded classic-JS slice, which blocks the offline badge regression.

## Context

- Owning subsystem: `Script`
- Related capability or gap: bounded browser stdlib slice / `navigator.onLine` bootstrap override
- Related docs: `doc/subsystem-map.md`, `doc/capability-matrix.md`, `doc/roadmap.md`

## Problem

- Current behavior: The offline bootstrap used by the agri-unit-converter regression calls `Object.defineProperty(window.navigator, "onLine", ...)`, but browser-tester-go rejects that surface with `unsupported browser surface "Object.defineProperty" in this bounded classic-JS slice`.
- Expected behavior: The harness should provide a supported way to seed `navigator.onLine = false` or another equivalent offline state for tests.
- Reproduction steps:
  1. Inject the offline bootstrap into the harness for `/tools/agri/agri-unit-converter/`.
  2. Use `Object.defineProperty(window.navigator, "onLine", { configurable: true, get: function () { return false; } })`.
  3. Open the page and observe the unsupported surface error.
- Reproduction code:

```text
Object.defineProperty(window.navigator, "onLine", {
  configurable: true,
  get: function () { return false; }
});
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run '^TestAgriUnitConverterBRT020OfflineBadgeIsVisibleWhenNavigatorReportsOffline$' -count=1
```
- Scope / non-goals: This issue is about the test harness capability. Do not change the page implementation; expose a supported offline-seed mechanism instead.

## Acceptance Criteria

- [ ] Tests can seed `navigator.onLine=false` without using unsupported `Object.defineProperty`.
- [ ] The offline badge regression can run without `t.Skip`.
- [ ] The bootstrap path fails explicitly only when the requested mock is genuinely unsupported.

## Test Plan

- Suggested test layer: `web-go/internal/generate` regression test.
- Regression or failure-path coverage: keep the offline badge regression enabled after the harness fix.
- Mock or fixture needs: a supported navigator offline seed or runtime mock.

## Notes

- The same limitation affects any other test that needs a deterministic offline navigator state.
