# `Worker` support is missing from the bounded browser surface

## Summary

- Short summary: Blob-backed `new Worker(...)` fails in browser tests because the bounded runtime rejects the `Worker` surface.

## Context

- Owning subsystem: Runtime / browser globals
- Related capability or gap: classic-JS browser surface does not include `Worker`
- Related docs:
  - `doc/capability-matrix.md`
  - `doc/implementation-guide.md`
  - `../finitefield-site/web/content/pages/tools/data/csv-tsv-converter/template.html`

## Problem

- Current behavior: The CSV/TSV converter browser test throws `unsupported browser surface "Worker" in this bounded classic-JS slice` when it clicks Convert.
- Expected behavior: `new Worker(URL.createObjectURL(blob))` should construct a worker and allow the page to receive worker messages.
- Reproduction steps:
  1. Load a minimal page that creates a blob-backed worker from a click handler.
  2. Click the button.
  3. Observe the unsupported surface error.
- Reproduction code:

```text
<!doctype html><main>
  <button id="run">Run</button>
  <div id="status"></div>
  <script>
    document.getElementById("run").addEventListener("click", () => {
      const blob = new Blob([
        "self.onmessage = () => self.postMessage('ok');"
      ], { type: "text/javascript" });
      const worker = new Worker(URL.createObjectURL(blob));
      worker.onmessage = () => {
        document.getElementById("status").textContent = "ok";
      };
      worker.postMessage("start");
    });
  </script>
</main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestCSVTSVConverter'
```

- Scope / non-goals: Keep the fix focused on the bounded worker surface needed by browser tests. Do not widen the runtime into a full worker implementation.

## Acceptance Criteria

- [ ] `new Worker(...)` works for blob URLs in the bounded browser surface.
- [ ] Worker message round-trips work for browser tests.
- [ ] Regression tests are added at the runtime/browser-bootstrap layer.

## Test Plan

- Suggested test layer: runtime/browser bootstrap test
- Regression or failure-path coverage: one blob-backed worker smoke test that posts a message
- Mock or fixture needs: none

## Notes

- This issue was confirmed from the finitefield-site CSV/TSV converter browser test, not from app code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
