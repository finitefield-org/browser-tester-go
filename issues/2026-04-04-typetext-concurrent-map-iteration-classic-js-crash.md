# `TypeText()` can trigger a concurrent map iteration crash in the classic JS runtime

## Summary

- Short summary: typing into a normal text input can crash the browser tester with `fatal error: concurrent map iteration and map write` while dispatching the input event.
- The failure is in `browser-tester-go`, not in `finitefield-site`, because the panic happens inside the harness while the page is still processing the typed input.

## Context

- Affected repository: `finitefield-site`
- Affected test file: `web-go/internal/generate/logistics_unit_converter_browser_test.go`
- Failing test:
  - `TestLogisticsUnitConverterBRT005WeightSuffixParsingUpdatesUnitSelectAndShareUrl`
- Command cwd: `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`

## Problem

- Current behavior: `Harness.TypeText("#luc-weight-input", "2204.62 lb")` eventually panics with a runtime crash in the classic JS slice.
- Expected behavior: typing into a plain `<input>` should dispatch normally and let the page update its own state.
- The crash appears to come from concurrent access inside `browsertester/internal/script.(*classicJSEnvironment).replaceArrayBindingsSeen`.

## Reproduction

1. From `/Users/kazuyoshitoshiya/Documents/GitHub/finitefield-site/web-go`, run:

```bash
go test ./internal/generate -run 'TestLogisticsUnitConverter' -count=1
```

2. The failure occurs in `TestLogisticsUnitConverterBRT005WeightSuffixParsingUpdatesUnitSelectAndShareUrl` when the harness calls `TypeText()` on `#luc-weight-input`.
3. The page never reaches the share URL assertions because the harness aborts first.

## Observed Output

```text
fatal error: concurrent map iteration and map write
...
browser-tester-go/internal/script.(*classicJSEnvironment).replaceArrayBindingsSeen(...)
browser-tester-go/internal/runtime.(*Session).TypeText(...)
finitefield-site/web-go/internal/generate.TestLogisticsUnitConverterBRT005WeightSuffixParsingUpdatesUnitSelectAndShareUrl(...)
```

## Why This Is browser-tester-go

- The crash happens before any page-specific assertion runs.
- The page under test uses a standard text input and a normal `input` listener.
- The stack trace points to the runtime implementation, not to the application code.

## Impact

- Browser tests that depend on realistic typing into text inputs can crash nondeterministically.
- Finitefield page coverage can work around the issue with `SetValue()`, but that weakens coverage for input-event behavior.

## Acceptance Criteria

- `TypeText()` no longer triggers `fatal error: concurrent map iteration and map write` in the classic JS runtime.
- A regression test covers typing into a normal text input and dispatching the resulting input event.

## Notes

- If the fix requires changing the runtime synchronization around binding tracking, the regression should stay at the harness/runtime layer.
- The failing finitefield-page test can be switched to `SetValue()` as a temporary workaround, but that does not address the underlying bug.
