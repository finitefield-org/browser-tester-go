# File-input seeds need byte-preserving content for invalid UTF-8 cases

## Summary

- Short summary: The file-input mock can seed text files, but it does not preserve raw invalid UTF-8 bytes well enough to exercise browser code that reads uploads through `TextDecoder(..., { fatal: true })`.

## Context

- Owning subsystem: Runtime / file input
- Related capability or gap: byte-preserving file seeds for upload flows
- Related docs:
  - `doc/mock-guide.md`
  - `internal/runtime/file_input.go`
  - `../finitefield-site/web/content/pages/tools/data/csv-tsv-converter/template.html`

## Problem

- Current behavior: Seeding a file through `SeedFileText()` does not let the finitefield-site CSV/TSV converter reach its invalid-UTF-8 error path.
- Expected behavior: Browser tests should be able to seed arbitrary file bytes so a page can call `await file.arrayBuffer()` and fail decoding on invalid UTF-8.
- Reproduction steps:
  1. Seed a file with bytes that are not valid UTF-8.
  2. Select the file in a page that decodes uploads with `TextDecoder("utf-8", { fatal: true })`.
  3. Observe that the invalid-UTF-8 path is not reproducible with the current text-only seed surface.
- Reproduction code:

```text
invalidUTF8 := string([]byte{0xff, 0xfe, 0xfd})
mocks.FileInput().SeedFileText("#csv-tsv-file-input", "broken.csv", invalidUTF8)
if err := harness.SetFiles("#csv-tsv-file-input", []string{"broken.csv"}); err != nil {
  panic(err)
}
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate -run '^TestCSVTSVConverterFileSelectionRejectsInvalidUTF8$'
```

- Scope / non-goals: Keep the fix focused on file-input seeding. Do not widen unrelated drag-and-drop or clipboard behavior.

## Acceptance Criteria

- [ ] File-input mocks can seed arbitrary byte payloads, not only valid UTF-8 text.
- [ ] `input.files[0]` byte content survives selection and can be decoded by browser code.
- [ ] A regression test covers the UTF-8 decode failure path.

## Test Plan

- Suggested test layer: runtime/session test for file-input seeds, plus a browser-bootstrap smoke test
- Regression or failure-path coverage: invalid UTF-8 seed should drive the app into the `TextDecoder` fatal error path
- Mock or fixture needs: a byte-backed file seed API or equivalent

## Notes

- This gap was confirmed in `browser-tester-go`, not in the finitefield-site app code.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
