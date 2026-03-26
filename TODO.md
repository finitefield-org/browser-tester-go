# Go Workspace Backlog

This file tracks the remaining work for the `go/` rewrite workspace.
The current codebase builds and `go test ./...` passes; the remaining items are bounded expansion, hardening, and documentation upkeep.

## P0 - Keep The Current Surface Stable

- [ ] Keep `go test ./...` and the release checklist green before merging any new slice.
- [ ] Keep unsupported behavior explicit. If a slice is incomplete, return a typed unsupported error instead of silently falling back.
- [ ] Keep the public facade thin. Do not add new `Harness` setters when a debug view or mock family is a better fit.
- [ ] Keep public API changes synchronized with `README.md`, `go/doc/capability-matrix.md`, `go/doc/subsystem-map.md`, and `go/doc/mock-guide.md`.

## P1 - Hardening

- [ ] Add or refresh internal subsystem tests for any change under `internal/dom`, `internal/runtime`, `internal/script`, or `internal/mocks`.
- [ ] Add or refresh public contract tests for any new or changed facade behavior.
- [ ] Add regression tests for every bug fix that changes observable behavior.
- [ ] Keep fuzz and property coverage current for parser, selector, scheduler, location/history, cookie/window.name, and mock-registry boundaries.

## P2 - Script Syntax Backlog

- [x] Object literals and array literals.
- [x] Object literal shorthand properties and methods.
- [x] Object literal computed property names and methods.
- [x] Object literal getter accessors.
- [x] Object literal setter accessors.
- [x] `throw` statements with catch-bound values.
- [x] Array/object destructuring patterns in `let` / `const` declarations.
- [x] Default binding values in array/object destructuring patterns.
- [x] Spread/rest syntax in array/object literals and `let` / `const` binding patterns.
- [x] `var` declarations.
- [x] Unary `typeof` operator.
- [x] Relational `in` operator on bounded object and array values.
- [x] Relational `instanceof` operator on bounded class objects.
- [x] Conditional `?:` operator.
- [x] Exponentiation operator `**` and assignment `**=`.
- [x] Bitwise and shift operators.
- [x] Logical assignment operators on local bindings and object property chains.
- [x] Arrow functions.
- [x] Plain `function` declarations and `return` statements.
- [x] `async` / `await`.
- [x] Plain `async function` declarations and expressions with `await` statements.
- [x] Async class methods with `await` statements.
- [x] Generator class methods with `yield` statements.
- [x] Default parameter values in function, arrow, and class method parameters.
- [x] Generator functions and standalone `yield` statements.
- [x] Named generator expressions and self-binding.
- [x] `yield*` delegation.
- [x] Unlabeled `break` / `continue` statements.
- [x] `delete` expressions on local object bindings.
- [x] `yield` inside nested block-bodied `if` / `else` branches and other simple block bodies.
- [x] `yield` inside loop bodies.
- [x] `yield` inside `switch` clauses and `try` / `catch` / `finally` blocks.
- [x] Labeled `break` / `continue` statements.
- [x] Bounded array `for...of` loops.
- [x] Bounded array `for await...of` loops.
- [x] Bounded object/array `for...in` loops.
- [x] Bounded `new Class()` instantiation for class objects.
- [x] Bounded `extends` inheritance for class objects.
- [x] Property assignment on existing object bindings and private class fields.
- [x] Module-style `export` declarations and export specifier lists.
- [x] Module syntax: `import` declarations, re-export syntax with `from`, and dynamic `import()`.
- [x] Top-level `await` at the dispatch entrypoint.
- [x] Static and prototype class methods.
- [x] Class instance fields.
- [x] Class computed fields and methods.
- [x] Class syntax beyond the current slice: private fields.
- [x] Class `super` property, method, and constructor calls in class bodies.
- [x] Template literal interpolation.
- [x] Optional chaining across bounded object-property chains, bracket access, and nullish bases, including `host?.method(...)` and `host?.["method"](...)`.
- [x] Optional call syntax `?.()` and bracket access `?.[expr]`.
- [ ] Any still-unsupported syntax that continues to throw `ErrorKindUnsupported`.

## P2 - Selector And Query Expansion

- [ ] Add new selector slices only when a user-visible gap appears and the HTML standard calls for it.
- [ ] Keep document-level and element-level query semantics aligned with the shared selector engine.
- [ ] Add additional live collection slices only when a concrete gap justifies the cost.
- [ ] Extend beyond the current bounded pseudo-class slice only when a specific scenario requires it.

## P2 - Reflection, Mutation, And Serialization

- [ ] Fill any missing bounded reflection helpers, `classList` / `dataset` behavior, or tree-mutation slices only when tests expose a gap.
- [ ] Keep `textarea` reset-default synchronization covered by tests.
- [ ] Keep `WriteHTML()` rollback behavior covered by tests.
- [ ] Verify live collections remain coherent after mutation.

## P3 - Mock Families And Debug Surfaces

- [ ] Add new mock families only through the runtime registry and expose them through the typed facade.
- [ ] Document seed state, capture behavior, failure injection, reset semantics, and a minimal example for every new mock family.
- [ ] Add new debug snapshots only when they explain a real regression or user-visible gap.
- [ ] Avoid growing `Harness` into a bag of `set_*` methods.

## Explicit Non-Goals

- Renderer and layout engine.
- External network I/O.
- Legacy or deprecated spec branches unless the capability matrix explicitly adds them.
