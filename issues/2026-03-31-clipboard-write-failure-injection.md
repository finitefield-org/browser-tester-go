# Clipboard write failure injection is missing for browser tests

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: The clipboard mock can seed reads and capture writes, but it cannot force `navigator.clipboard.writeText()` to reject, so browser tests cannot cover share-failure paths without a workaround.

## Context

- Owning subsystem: `bt-runtime` clipboard / user-like actions
- Related capability or gap: clipboard write failure injection for browser test coverage
- Related docs: `browser-tester-go/doc/mock-guide.md`, `browser-tester-go/doc/capability-matrix.md`

## Problem

- Current behavior: `Session.WriteClipboard(text)` always succeeds and only records the write, and `navigator.clipboard.writeText()` resolves through that same path. The clipboard family exposes seed/read/write capture only, with no failure hook.
- Expected behavior: Browser tests should be able to force `clipboard.writeText()` to reject so pages can exercise failure UX such as `shareFail` toasts or retry messaging.
- Reproduction steps:
  1. Load a page whose share button calls `navigator.clipboard.writeText(...)` and shows a failure toast on rejection.
  2. Try to configure the harness so the clipboard write rejects.
  3. Observe that the current clipboard family has no failure seed and the write always succeeds.
- Reproduction code:

```go
package main

import "browsertester"

func main() error {
	h, err := browsertester.NewHarnessBuilder().Build()
	if err != nil {
		return err
	}

	// The current clipboard family can seed read text and capture writes,
	// but it cannot make this call fail.
	return h.WriteClipboard("share URL")
}
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go
go test ./...
```

- Scope / non-goals: This is limited to clipboard write failure injection. It is not a request to work around the missing capability in page tests.

## Acceptance Criteria

- [ ] Clipboard write failure can be seeded explicitly.
- [ ] Failure paths are explicit and do not silently fall back to success.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/session test for `WriteClipboard`, plus a public contract test if a new harness-facing seed is added.
- Regression or failure-path coverage: verify `navigator.clipboard.writeText()` rejects when the failure seed is set, and still records writes only when the seed is absent.
- Mock or fixture needs: clipboard seed text, a minimal HTML page with a share button, and a browser test that can assert the failure toast.

## Notes

- Working directory: `/Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go`
- This gap blocks browser coverage for share-failure UX in `finitefield-site` tools that rely on clipboard copy.
