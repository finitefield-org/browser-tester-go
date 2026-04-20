# `TextDecoder.decode()` returns NUL bytes for manually populated `Uint8Array`s

## Summary

- Re-running the `roof-area-pitch-calculator` browser tests after the browser-tester-go fix still fails when the page restores its shared state payload from `s=...` in the query string.
- The failure is in `browser-tester-go`, not in `finitefield-site/web-go`, because the same page logic works in a real browser but the harness decodes a manually populated `Uint8Array` into a string full of NUL bytes.

## Context

- Affected repository: `finitefield-site`
- Affected test file: `web-go/internal/generate/construction_roof_area_pitch_calculator_browser_test.go`
- Failing test:
  - `TestConstructionRoofAreaPitchCalculatorBRT013ShareURLRestoresState`
- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`

## Problem

- Current behavior: the share URL contains a valid `s=` payload, but the restored harness cannot recover the original section values because the page’s `fromBase64Url()` path decodes JSON bytes with `new Uint8Array(length)` plus indexed assignment, then passes that array to `new TextDecoder().decode(bytes)`.
- In `browser-tester-go`, that decode path produces a string whose visible content is all NUL characters, so `JSON.parse()` fails with `invalid character '\x00' looking for beginning of value`.
- Expected behavior: `TextDecoder.decode()` should return the original JSON string for any valid `Uint8Array`, including one populated by indexed assignment.

## Reproduction

1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run:

```bash
go test ./internal/generate -run TestConstructionRoofAreaPitchCalculatorBRT013ShareURLRestoresState -count=1 -v
```

2. The test emits a share URL with a valid `s=` payload and then fails on restore because the restored section inputs stay empty.
3. A minimal browser-tester reproduction is:

```go
const rawHTML = `<main><div id="out"></div><script>(function(){const bytes=new Uint8Array(3);bytes[0]=91;bytes[1]=123;bytes[2]=34;const decoded=new TextDecoder().decode(bytes);document.getElementById("out").textContent=JSON.stringify([bytes[0],bytes[1],bytes[2],decoded.length,decoded.charCodeAt(0),decoded.charCodeAt(1),decoded.charCodeAt(2),decoded]);})()</script></main>`
```

```go
harness, _ := browsertester.FromHTML(rawHTML)
```

4. Observed output:

```text
[91,123,34,3,0,0,0,"\u0000\u0000\u0000"]
```

5. The same bytes created with `Uint8Array.from([91,123,34])` decode correctly, so the failure is specific to the indexed-assignment path used by the page’s base64 restore helper.

## Why This Is browser-tester-go

- The page code is standard browser code: it builds a byte array from `atob()`, then uses `TextDecoder` and `JSON.parse()` to restore the state snapshot.
- The typed array contents are visible through direct indexed reads before decoding, but the decoded string contains only NUL bytes in the harness.
- That means the browser-tester runtime is not preserving the written bytes when the array is populated with indexed assignment, which breaks a normal browser restore path.

## Impact

- Shared URLs for tools that serialize state as base64 JSON cannot restore correctly in the harness when they decode into a manually populated `Uint8Array`.
- `roof-area-pitch-calculator` cannot validate share/restoration coverage without a browser-tester fix.

## Acceptance Criteria

- `TextDecoder.decode()` returns the expected UTF-8 string for a `Uint8Array` populated by indexed assignment.
- A regression test covers the manual-assignment case, not just `TextEncoder` round-trips or `Uint8Array.from(...)` inputs.

## Test Plan

- Suggested test layer: `internal/runtime` bootstrap or runtime contract test.
- Regression coverage should verify that bytes written into `new Uint8Array(n)` via `bytes[i] = ...` are visible to `TextDecoder.decode(bytes)`.
- A second assertion should confirm that the `roof-area-pitch-calculator` restore path can parse the `s=` payload and repopulate the section inputs.

## Notes

- The browser-tester-go API surface itself is available: `URLSearchParams`, `TextDecoder`, `TextEncoder`, `atob`, and `Uint8Array.from(...)` all work in this environment.
- The issue reproduces without any workaround in the app test code.
