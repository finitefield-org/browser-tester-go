# Issue Draft

## Summary

- Short summary: `DataTransfer` / `File` / drag-and-drop support is missing from the Go browser runtime, which blocks `csv-editor` sample loading and column reorder tests.

## Context

- Owning subsystem: Runtime
- Related capability or gap: Event dispatch and default actions; file input and drag-and-drop surfaces
- Related docs:
  - `doc/capability-matrix.md`
  - `doc/implementation-guide.md`
  - `../finitefield-site/doc/tools/data/csv-editor.md`

## Problem

- Current behavior: A minimal browser-tester-go harness that executes `new DataTransfer()` inside inline script fails, so the runtime cannot exercise browser code paths that depend on `DataTransfer`, `File`, `input.files` re-assignment from a `FileList`, or drag-and-drop reordering payloads.
- Expected behavior: The bounded browser runtime should expose enough of `DataTransfer`, `File`, and the `files` surface for file-selection and drag-and-drop flows used by `csv-editor`.
- Reproduction steps:
  1. Create a minimal harness with a button handler that runs `new DataTransfer()`, adds a `File`, and assigns `input.files = dt.files`.
  2. Click the button.
  3. Observe that the script fails before the file can be assigned.
- Reproduction code:

```text
<!doctype html><main><button id="go">Go</button><input id="file" type="file"><div id="out"></div><script>
document.getElementById("go").addEventListener("click", () => {
  try {
    const dt = new DataTransfer();
    const file = new File(["hello"], "sample.csv", { type: "text/csv" });
    dt.items.add(file);
    document.getElementById("file").files = dt.files;
    document.getElementById("out").textContent = "ok:" + document.getElementById("file").files.length;
  } catch (error) {
    document.getElementById("out").textContent = "ERR:" + error.name + ":" + error.message;
  }
});
</script></main>
```

- Original failed command:

```bash
cd /tmp/btprobe_csv_editor
go mod tidy
go run .
```

- Scope / non-goals: Keep the fix focused on the bounded file-input and drag-and-drop surfaces needed by `csv-editor`; do not widen the runtime into a full browser drag-and-drop implementation.

## Acceptance Criteria

- [ ] `new DataTransfer()` is available in the bounded runtime slice needed by browser tests.
- [ ] `new File()` is available in the bounded runtime slice needed by browser tests.
- [ ] `input.files` can be populated from the resulting file list for file-selection flows.
- [ ] Drag-and-drop event payloads expose a usable `dataTransfer` object for tests that exercise column reorder.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/browser bootstrap tests
- Regression or failure-path coverage: file-selection bootstrap and drag-and-drop bootstrap reproductions
- Mock or fixture needs: none

## Notes

- This issue was confirmed from a browser-tester-go-only repro, not from the finitefield-site app code.
- Command execution folder: `/tmp/btprobe_csv_editor`
