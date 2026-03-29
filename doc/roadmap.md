# Roadmap

This is the recommended build order for the Go workspace. Keep the early phases sequential so the
public facade stays small and the implementation stays testable. Detailed support state lives in
[`capability-matrix.md`](capability-matrix.md), and remaining gaps live in [`../TODO.md`](../TODO.md).

## Phase 0: Scaffold

- Create the module, package root, internal packages, and CI-friendly tests.
- Define the public facade, error taxonomy, explicit builder config, and core docs.
- Keep later behavior behind the thin facade instead of widening the API early.

Exit criteria:

- the package compiles
- the facade is thin
- the docs are in place

## Phase 1: DOM Core

- Parse HTML into the internal DOM store.
- Implement the first selector slice and DOM dump/assertion helpers.

Exit criteria:

- HTML round-trips deterministically in tests
- selectors work for the first supported slice

## Phase 2: Script Core

- Implement the minimum classic-JS parser/evaluator slice and the host bindings needed for inline
  bootstrap.
- Keep expanding only the bounded syntax and runtime helpers required by the capability matrix.

Exit criteria:

- supported inline scripts and modules execute deterministically

## Phase 3: Events and Form Controls

- Wire user-like actions, event propagation, and default actions.
- Add form-control state and selection updates.

Exit criteria:

- interactions and form state are deterministic and observable

## Phase 4: Deterministic Runtime and Mocks

- Add deterministic scheduler support and typed mock families for browser-visible state.
- Cover history, storage, cookies, navigator, clipboard, open/close/print/scroll, matchMedia,
  downloads, and file inputs.

Exit criteria:

- session state and mock capture are deterministic and seedable

## Phase 5: Hardening

- Add subsystem tests, public contract tests, regression tests, and property tests.
- Expand coverage before widening the surface.

Exit criteria:

- behavior is covered at the public boundary and the internal boundary

## Phase 6: Selector and Query Expansion

- Expand selector support and live collections only as needed by user-visible gaps.

Exit criteria:

- DOM and script querying share the same core selector logic

## Phase 7: Reflection, Mutation, Serialization

- Add attribute reflection, classList/dataset views, mutation primitives, and HTML
  serialization/insertion surfaces.

Exit criteria:

- mutation updates the DOM deterministically
- live collections stay coherent after mutation

## Working Rules

- Do not move to a later phase until the earlier phase is covered by tests.
- Do not add a new public `Harness` method until the capability matrix has a row for it.
- Use `../html-standard/` when adding or changing HTML behavior.
- Prefer small slices over large parity pushes.
