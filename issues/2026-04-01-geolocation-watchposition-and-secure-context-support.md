# `navigator.geolocation.watchPosition()` and `window.isSecureContext` support are needed for GPS browser tests

## Summary

- Short summary: The bounded browser runtime does not expose `navigator.geolocation` or `window.isSecureContext`, so browser tests for GPS-driven flows cannot start, receive mocked fixes, or verify insecure-context behavior.

## Context

- Owning subsystem: Runtime / browser globals
- Related capability or gap: geolocation watch callbacks and secure-context detection
- Related docs: `browser-tester-go/internal/runtime/browser_globals.go`, `browser-tester-go/internal/runtime/session.go`, `browser-tester-go/doc/capability-matrix.md`
- Affected finitefield-site coverage: `web-go/internal/generate/area-boundary-calculator_browser_test.go`

## Problem

- Current behavior: `navigator.geolocation` is not resolved by the runtime, there is no way for tests to inject position updates into active watchers, and `window.isSecureContext` is not exposed as a browser-global value.
- Expected behavior: Classic-JS browser tests should be able to start GPS watch sessions, emit mock positions into `watchPosition()` callbacks, clear watches, and branch on secure-context availability the same way the page does in production.
- Reproduction steps:
  1. Load a page that calls `navigator.geolocation.watchPosition(...)` from a button click handler.
  2. Attempt to drive the callback from a harness or test.
  3. Observe that no geolocation surface exists to hook into, and insecure-context logic cannot be exercised through `window.isSecureContext`.
- Reproduction code:

```text
<!doctype html><main>
  <button id="start">start</button>
  <div id="out"></div>
  <script>
    document.getElementById("start").addEventListener("click", () => {
      if (!window.isSecureContext) {
        document.getElementById("out").textContent = "insecure";
        return;
      }
      navigator.geolocation.watchPosition((position) => {
        document.getElementById("out").textContent = String(position.coords.latitude);
      });
    });
  </script>
</main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate
```

- Scope / non-goals: Keep the fix focused on browser-global geolocation and secure-context support for the bounded runtime. Do not workaround by rewriting the finitefield-site tests to avoid GPS coverage.

## Acceptance Criteria

- [ ] `window.isSecureContext` resolves in the runtime and reflects the configured page URL.
- [ ] `navigator.geolocation.watchPosition()` registers active watchers.
- [ ] Tests can inject mock geolocation positions into active watchers.
- [ ] `navigator.geolocation.clearWatch()` removes registered watches.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/session tests for browser globals and a public harness-level smoke test for position injection.
- Regression or failure-path coverage: verify secure vs insecure URLs and ensure injected positions reach registered watch callbacks.
- Mock or fixture needs: a lightweight geolocation emitter API exposed from `browsertester`.

## Notes

- This gap blocks the new `area-boundary-calculator` GPS coverage in `finitefield-site`.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
