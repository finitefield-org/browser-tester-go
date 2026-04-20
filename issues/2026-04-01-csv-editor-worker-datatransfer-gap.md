# Issue Draft

## Summary

- Short summary: `csv-editor` browser tests are blocked by missing `Worker` support and missing `DataTransfer` / drag payload surfaces in browser-tester-go.

## Context

- Owning subsystem: Runtime
- Related capability or gap: classic-JS browser globals, worker bootstrap, and drag-and-drop/file-selection surfaces
- Related docs:
  - `doc/capability-matrix.md`
  - `doc/implementation-guide.md`
  - `../finitefield-site/doc/tools/data/csv-editor.md`

## Problem

- Current behavior: The `csv-editor` page creates a blob-backed `Worker` for parsing/building CSV output, creates `File`/`DataTransfer` instances for the sample loader, and reads `event.dataTransfer` during drag-and-drop. Those browser surfaces are not available in the browser-tester-go runtime slice, so the documented browser paths cannot run.
- Expected behavior: The bounded runtime should support the minimum `Worker` / `DataTransfer` / drag payload surfaces needed by `csv-editor` browser tests.
- Reproduction steps:
  1. Create a minimal harness or page-level test that clicks the `csv-editor` sample button.
  2. Observe that the script reaches `new Worker(...)` and/or `new DataTransfer()` / drag payload access.
  3. Observe the unsupported-surface failure.
- Reproduction code:

```text
<!doctype html><main>
  <button id="go" type="button">Go</button>
  <input id="file" type="file">
  <div id="out"></div>
  <script>
    document.getElementById("go").addEventListener("click", () => {
      const file = new File(["a,b\n1,2"], "sample.csv", { type: "text/csv" });
      const dt = new DataTransfer();
      dt.items.add(file);
      document.getElementById("file").files = dt.files;

      const blob = new Blob(["self.onmessage = (event) => postMessage(event.data);"], { type: "text/javascript" });
      const worker = new Worker(URL.createObjectURL(blob));
      worker.onmessage = (event) => {
        document.getElementById("out").textContent = event.data;
      };
      worker.postMessage("ok");
    });
  </script>
</main>
```

- Original failed command:

```bash
cd /tmp/btprobe_csv_editor
go test -v -mod=mod -run TestCSVEditorMissingBrowserSurfaces -count=1
```

- Command cwd: `/tmp/btprobe_csv_editor`
- Scope / non-goals:
  - Scope is the bounded browser-global, worker bootstrap, and drag-and-drop/file-selection surfaces needed by `csv-editor`.
  - Non-goal: rewiring `csv-editor` to avoid `Worker`, `DataTransfer`, or `event.dataTransfer` just for tests.

## Acceptance Criteria

- [ ] `new Worker(...)` works for blob URLs in the bounded browser surface.
- [ ] Worker message round-trips work for browser tests.
- [ ] `new File(...)` / `new DataTransfer()` support the sample-loading path used by `csv-editor`.
- [ ] `input.files = dt.files` works for file-selection flows.
- [ ] Drag-and-drop event payloads expose a usable `dataTransfer` object for `dragstart` / `dragover` / `drop`.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/browser bootstrap tests plus one page-level browser regression test
- Regression or failure-path coverage: a minimal worker bootstrapping repro, a file-selection repro, and a drag-payload repro
- Mock or fixture needs: none

## Notes

- This issue was confirmed from the `csv-editor` page design and the browser-tester-go runtime surface, not from finitefield-site app code.
- Command execution folder: `/tmp/btprobe_csv_editor`
