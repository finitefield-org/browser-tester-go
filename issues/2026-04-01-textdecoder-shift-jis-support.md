# `TextDecoder("shift_jis")` support is missing from the bounded browser runtime

## Summary

- Short summary: The runtime only accepts UTF-8 in `TextDecoder`, so browser tests that load Shift_JIS CSV files cannot decode upload bytes correctly.

## Context

- Owning subsystem: Runtime / text codec
- Related capability or gap: `TextDecoder` label support for file-import flows
- Related docs: `browser-tester-go/internal/runtime/text_codec.go`, `browser-tester-go/internal/runtime/browser_globals_bootstrap_test.go`
- Affected finitefield-site coverage: `web-go/internal/generate/data_csv_to_json_converter_browser_test.go` BRT-023

## Problem

- Current behavior: `new TextDecoder("shift_jis")` is rejected, so browser code that reads a Shift_JIS upload and decodes it in the client cannot be exercised in the harness.
- Expected behavior: The bounded runtime should accept `shift_jis` and decode the same byte sequences a browser would decode for CSV import tests.
- Reproduction steps:
  1. Load a page that calls `new TextDecoder("shift_jis").decode(bytes)`.
  2. Provide a valid Shift_JIS byte sequence.
  3. Observe that the constructor is rejected before decoding can happen.
- Reproduction code:

```text
<!doctype html><main><div id="out"></div><script>
  const decoder = new TextDecoder("shift_jis");
  const bytes = new Uint8Array([0x82, 0xa0, 0x82, 0xa2]);
  document.getElementById("out").textContent = decoder.decode(bytes);
</script></main>
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go
go test ./internal/generate
```

- Scope / non-goals: Add Shift_JIS support only. Do not widen the codec surface beyond what the file-import browser tests need.

## Acceptance Criteria

- [ ] `TextDecoder("shift_jis")` constructs successfully.
- [ ] `decoder.encoding` reports `shift_jis`.
- [ ] `decoder.decode()` returns the expected UTF-8 string for valid Shift_JIS bytes.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/browser bootstrap test.
- Regression or failure-path coverage: confirm UTF-8 still works and Shift_JIS decoding produces the expected Japanese text.
- Mock or fixture needs: a short valid Shift_JIS byte sequence.

## Notes

- This is a browser-tester-go runtime gap, not a finitefield-site page bug.
- Command execution folder: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`
