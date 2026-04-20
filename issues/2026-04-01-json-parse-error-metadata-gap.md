# Issue Draft

## Summary

- Short summary: `JSON.parse` errors in the Go browser runtime lose `message` / `line` / `column` / `position` fields, which prevents line/column recovery in browser tests.

## Context

- Owning subsystem: Runtime / JS error surface
- Related capability or gap: Built-in `JSON.parse` error metadata, `Error` stringification, parse-error diagnostics
- Related docs:
  - `doc/capability-matrix.md`
  - `doc/implementation-guide.md`

## Problem

- Current behavior: When inline page code catches a `JSON.parse` failure, `error.message`, `error.line`, `error.column`, and `error.position` are all empty in the runtime. Only `String(error)` contains a message.
- Expected behavior: The runtime should expose parse-error metadata in a shape that browser tests can use to recover line and column information, or otherwise mirror browser error surfaces closely enough for parse diagnostics.
- Reproduction steps:
  1. Build a harness with a button that runs `JSON.parse` on `{"a":1,}` and writes `String(error)`, `error.message`, `error.line`, `error.column`, and `error.position` into the DOM.
  2. Click the button.
  3. Observe that only `String(error)` has content and the other fields are empty.
- Reproduction code:

```text
<!doctype html><main><textarea id="input"></textarea><button id="go">Go</button><div id="detail"></div><script>
document.getElementById("go").addEventListener("click", () => {
  try {
    JSON.parse(document.getElementById("input").value);
    document.getElementById("detail").textContent = "ok";
  } catch (error) {
    document.getElementById("detail").textContent = [
      String(error),
      String(error && error.name || ""),
      String(error && error.message || ""),
      String(error && error.line || ""),
      String(error && error.column || ""),
      String(error && error.position || ""),
    ].join("|");
  }
});
</script></main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run JSONYamlToml -count=1
```

- Scope / non-goals: Keep the runtime focused on browser-like `JSON.parse` error metadata for browser tests; do not widen it into a full parser implementation.

## Acceptance Criteria

- [ ] `JSON.parse` failure surfaces usable error metadata for browser tests.
- [ ] `error.message` or an equivalent property contains the parse error message.
- [ ] At least one of `line`, `column`, or `position` is available for line/column recovery.
- [ ] Regression coverage is added for the JSON parse error surface.

## Test Plan

- Suggested test layer: runtime/browser bootstrap tests
- Regression or failure-path coverage: trailing-comma JSON reproduction
- Mock or fixture needs: none

## Notes

- This issue was confirmed from a browser-tester-go-only repro, not from the finitefield-site app code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
