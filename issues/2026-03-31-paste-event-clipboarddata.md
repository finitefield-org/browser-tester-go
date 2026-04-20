# Paste events cannot carry clipboard text in synthetic dispatch

## Checklist Before Filing

- Review `subsystem-map.md` and identify the owning subsystem.
- Review `implementation-guide.md` and pick the test layer first.
- Review `capability-matrix.md` and confirm the capability row or gap.
- Review `roadmap.md` if the issue affects build order or sequencing.

## Summary

- Short summary: Synthetic event dispatch cannot reproduce a browser `paste` interaction because the event object has no way to carry `clipboardData` payload text.

## Context

- Owning subsystem: `bt-runtime` event dispatch / user-like actions
- Related capability or gap: synthetic event dispatch needs clipboard-backed paste payloads for browser tests
- Related docs: `browser-tester-go/doc/capability-matrix.md`, `browser-tester-go/doc/implementation-guide.md`

## Problem

- Current behavior: `Harness.Dispatch(selector, "paste")` only dispatches an event name. There is no way to provide `event.clipboardData.getData("text/plain")` contents, so a page-level `paste` handler cannot read pasted TSV/CSV text during a browser test.
- Expected behavior: Synthetic paste tests should be able to supply clipboard text to the dispatched event, or the harness should expose a dedicated paste action that behaves like a real paste interaction.
- Reproduction steps:
  1. Load a page with a `paste` listener that reads `event.clipboardData.getData("text/plain")`.
  2. Seed clipboard text in the harness.
  3. Dispatch `paste` on the target element.
- Reproduction code:

```html
<main>
  <textarea id="target"></textarea>
  <div id="out"></div>
  <script>
    document.getElementById("target").addEventListener("paste", function (event) {
      document.getElementById("out").textContent = event.clipboardData.getData("text/plain");
    });
  </script>
</main>
```

```go
harness, _ := browsertester.FromHTMLWithURL("https://finitefield.org/", html)
_ = harness.WriteClipboard("A\tB\n1\t2")
_ = harness.Dispatch("#target", "paste")
```

- Original failed command:

```bash
cd /Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go
go test ./...
```

- Scope / non-goals: This is limited to synthetic paste interaction support. It is not a request to change page code or to work around the missing payload by typing text directly.

## Acceptance Criteria

- [ ] Primary behavior is implemented or fixed.
- [ ] Failure paths are explicit and do not silently fall back.
- [ ] Tests are added or updated at the chosen layer.

## Test Plan

- Suggested test layer: runtime/session test for event dispatch, plus a public contract test if a new harness action is added.
- Regression or failure-path coverage: verify a dispatched paste event can expose supplied clipboard text to the listener, and verify the failure is explicit when no payload is provided.
- Mock or fixture needs: clipboard seed text, a minimal HTML page with a `paste` listener, and a browser test that exercises TSV paste.

## Notes

- Working directory: `/Users/kazuyoshitoshiya/Documents/GitHub/browser-tester-go`
- This gap blocks browser tests for `tolerance-checker` manual TSV paste coverage without adding a workaround.
