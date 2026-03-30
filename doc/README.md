# browser-tester go

This directory is the Go rewrite workspace for `browser-tester`. It is still a plan and
specification set, but the phase 0 scaffold now exists and the module builds with skeleton tests.
The detailed support map lives in [`capability-matrix.md`](capability-matrix.md), and the remaining
implementation gaps are tracked in [`../TODO.md`](../TODO.md).

The design follows the lessons captured in [`../next.md`](../../next.md) and
[`../next-reflection.md`](../../next-reflection.md):

- keep `Harness` thin
- keep state in explicit subsystems
- make deterministic mocks first-class
- separate debug views from the public action surface
- keep configuration explicit instead of hiding seeds in unrelated state

## Document Index

- [Subsystem Map](subsystem-map.md)
- [Capability Matrix](capability-matrix.md)
- [Implementation Guide](implementation-guide.md)
- [Mock Guide](mock-guide.md)
- [Release Checklist](release-checklist.md)
- [Roadmap](roadmap.md)
- [Native RegExp Engine ADR](adr/0001-native-regexp-engine.md)

## Core Rules

- Check `../html-standard/` before adding HTML, DOM, selector, or serialization behavior.
- Add a new public `Harness` method only after deciding whether it belongs on `Harness`,
  `DebugView`, or a mock family.
- Add a new test-only mock through the runtime registry, then expose it through the public facade.
- Prefer explicit configuration structs over hidden encodings or seed keys.
- Keep the Go implementation deterministic. Avoid background goroutines unless a subsystem
  explicitly requires them.

## Current Status

- Phase 0 scaffold is present, and the module builds with skeleton tests.
- The public facade remains thin, with the main behavior split across DOM, runtime, script, and
  mock subsystems.
- The mock registry now covers fetch, external JS dependency loads, dialogs, clipboard,
  navigator, location, open/close/print/scroll, matchMedia, downloads, file input, and storage.
- `DebugView` is read-only and exists for DOM, runtime, and trace inspection rather than as an
  action surface.
- The native RegExp engine design lives in `adr/0001-native-regexp-engine.md`, and the native
  engine is the production path.
- The detailed capability map is maintained in `capability-matrix.md`, and the remaining
  implementation gaps live in `../TODO.md`.

## Target Shape

- Public package: `browsertester`
- Internal packages: `internal/dom`, `internal/runtime`, `internal/script`, `internal/mocks`
- Public facade types: `Harness`, `HarnessBuilder`, `DebugView`, `MockRegistryView`, `Error`,
  `OptionLabel`, `OptionValue`, `OptgroupLabel`

## When Code Lands

The intended quick check is:

```bash
go test ./...
```
