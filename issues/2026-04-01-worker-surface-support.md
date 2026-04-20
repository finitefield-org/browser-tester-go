# Worker surface support in the bounded classic JS slice

## Summary
`browser-tester-go` fails when a page creates a Web Worker from classic JavaScript. The `csv-to-json-converter` page calls `new Worker(...)` unconditionally during conversion setup, and the current bounded classic-JS slice rejects that browser surface.

## Context
This was discovered while adding Go browser tests for:
- `/tools/data/csv-to-json-converter/`

The page implementation legitimately uses a Worker for conversion. The failure is in `browser-tester-go`, not in `finitefield-site`.

## Original failed command
Working directory:
`/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`

Command:
`go test ./internal/generate -run CSVToJSONConverter -count=1`

## Failure
The test run fails with:
`timer: unsupported: unsupported browser surface "Worker" in this bounded classic-JS slice`

## Problem
Pages that use `Worker` cannot be exercised in browser tests even when their data and DOM are otherwise valid. This blocks coverage for tools that intentionally offload work to a Worker, including CSV conversion pages.

## Expected behavior
`browser-tester-go` should support the `Worker` surface in the classic JS slice, or provide an equivalent testing surface that allows Worker-backed pages to execute their normal flow.

## Acceptance criteria
- `new Worker(...)` does not fail in browser tests for classic JS pages.
- Worker-backed pages can run their normal message flow during tests.
- Existing browser tests that do not use Worker remain unaffected.

## Test plan
- Re-run `cd /Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go && go test ./internal/generate -run CSVToJSONConverter -count=1`
- Re-run the broader `go test ./...` coverage in `browser-tester-go`
