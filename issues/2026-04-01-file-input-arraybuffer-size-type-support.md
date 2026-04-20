# `input.files` file objects need `arrayBuffer()`, `size`, and `type` for browser tests

## Summary

- Short summary: The file-input mock can seed `File.text()` through `SetFiles()`, but it does not expose `File.arrayBuffer()`, `File.size`, or `File.type`, so browser tests that read raw bytes from uploads cannot run in the bounded runtime.

## Context

- Owning subsystem: Runtime / file input
- Related capability or gap: byte-backed file objects for upload flows
- Related docs: `browser-tester-go/doc/mock-guide.md`, `browser-tester-go/internal/runtime/file_input.go`
- Affected finitefield-site coverage: `web-go/internal/generate/data_csv_to_json_converter_browser_test.go` BRT-022

## Problem

- Current behavior: `SetFiles()` exposes `input.files[0]` objects with `name` and `File.text()`, but browser code that calls `await file.arrayBuffer()` or reads `file.size` / `file.type` hits unsupported surfaces or missing metadata.
- Expected behavior: File objects in the harness should behave like browser `File` objects closely enough for upload flows that inspect raw bytes, file size, and file type.
- Reproduction steps:
  1. Load a page with `<input type="file">` and a change handler that reads `file.arrayBuffer()`, `file.size`, and `file.type`.
  2. Seed a file and trigger `SetFiles()`.
  3. Observe that the current runtime cannot satisfy the byte-reading path.
- Reproduction code:

```text
<!doctype html><main>
  <input id="upload" type="file">
  <div id="out"></div>
  <script>
    document.getElementById("upload").addEventListener("change", async () => {
      const file = document.getElementById("upload").files[0];
      const buffer = await file.arrayBuffer();
      document.getElementById("out").textContent = [
        file.name,
        file.size,
        file.type,
        Array.from(new Uint8Array(buffer)).join(","),
      ].join("|");
    });
  </script>
</main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate
```

- Scope / non-goals: Keep the fix focused on the file-input mock surface used by browser tests. This is not a request to add drag-and-drop or a full `FileList` implementation.

## Acceptance Criteria

- [ ] File objects returned from `input.files` expose `arrayBuffer()`.
- [ ] File objects expose `size` and `type`.
- [ ] Seeded file contents remain available through `File.text()`.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/session test for `SetFiles()`, plus a browser-bootstrap smoke test.
- Regression or failure-path coverage: verify seeded files can be read as bytes and that the metadata is present and stable.
- Mock or fixture needs: a small text file seed and a byte-backed seed.

## Notes

- This gap was confirmed in `browser-tester-go`, not in the finitefield-site app code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
