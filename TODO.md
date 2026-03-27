# Go Workspace Backlog

This file tracks the remaining work for the `go/` rewrite workspace. The current codebase builds and
`go test ./...` passes; the remaining items are bounded expansion, hardening, and documentation
upkeep.

## P0 - Keep The Current Surface Stable

- [ ] Keep `go test ./...` and the release checklist green before merging any new slice.
- [ ] Keep unsupported behavior explicit. If a slice is incomplete, return a typed unsupported error
  instead of silently falling back.
- [ ] Keep the public facade thin. Do not add new `Harness` setters when a debug view or mock family
  is a better fit.
- [ ] Keep public API changes synchronized with `README.md`, `go/doc/capability-matrix.md`,
  `go/doc/subsystem-map.md`, and `go/doc/mock-guide.md`.

## P1 - Hardening

- [ ] Add or refresh internal subsystem tests for any change under `internal/dom`,
  `internal/runtime`, `internal/script`, or `internal/mocks`.
- [ ] Add or refresh public contract tests for any new or changed facade behavior.
- [ ] Add regression tests for every bug fix that changes observable behavior.
- [ ] Keep fuzz and property coverage current for parser, selector, scheduler, location/history,
  cookie/window.name, and mock-registry boundaries.

## P2 - Script Syntax Backlog

- [x] Object literals and array literals.
- [x] Object literal shorthand properties and methods.
- [x] Object literal computed property names and methods.
- [x] Object literal getter accessors.
- [x] Object literal setter accessors.
- [x] Object literal async and generator methods.
- [x] Object literal and class methods can read, write, and delete `super` through bounded prototype
  targets and bounded null-prototype object literals, including compound assignments.
- [x] `throw` statements with catch-bound values.
- [x] `debugger` statements.
- [x] Array/object destructuring patterns in `let` / `const` declarations.
- [x] Default binding values in array/object destructuring patterns.
- [x] Spread/rest syntax in array/object literals and `let` / `const` binding patterns.
- [x] Function-like parameter destructuring patterns with default values and rest identifiers.
- [x] `var` declarations.
- [x] Unary `typeof` operator.
- [x] Relational `in` operator on bounded object and array values.
- [x] Relational `instanceof` operator on bounded class objects.
- [x] Conditional `?:` operator.
- [x] Comma operator / sequence expressions.
- [x] Exponentiation operator `**` and assignment `**=`.
- [x] Bitwise and shift operators.
- [x] Prefix/postfix increment and decrement expressions on local bindings and object/array property
  chains.
- [x] Logical assignment operators on local bindings and object/array property chains.
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
- [x] `delete` expressions on local object and array bindings.
- [x] `yield` inside nested block-bodied `if` / `else` branches and other simple block bodies.
- [x] `yield` inside loop bodies.
- [x] `yield` inside `switch` clauses and `try` / `catch` / `finally` blocks.
- [x] Labeled `break` / `continue` statements.
- [x] Bounded array and iterator-like object `for...of` loops.
- [x] Bounded array and iterator-like object `for await...of` loops.
- [x] Bounded object/array `for...in` loops.
- [x] Bounded `new Class()` instantiation for class objects.
- [x] Bounded `extends` inheritance for class objects.
- [x] Property assignment on existing object/array bindings and private class fields.
- [x] Module-style `export` declarations and export specifier lists.
- [x] Module syntax: `import` declarations, re-export syntax with `from`, and dynamic `import()`.
- [x] `using` / `await using` declarations.
- [x] `import.meta.url` inside bounded module scripts.
- [x] Top-level `await` at the dispatch entrypoint.
- [x] Static and prototype class methods.
- [x] Class instance fields.
- [x] Class computed fields and methods.
- [x] Numeric literals across decimal, hexadecimal, binary, and octal forms, including numeric
  separators and `BigInt` suffixes.
- [x] Class expressions.
- [x] Class syntax beyond the current slice: private fields.
- [x] Private `in` operator on bounded class private fields.
- [x] Class `super` property, method, and constructor calls in class bodies.
- [x] Class `super` property assignment when the receiver does not already expose the property.
- [x] Template literal interpolation.
- [x] Tagged template literals with bounded function tags and interpolation.
- [x] Optional chaining across bounded object-property chains, bracket access, and nullish bases,
  including `host?.method(...)` and `host?.["method"](...)`.
- [x] Optional call syntax `?.()` and bracket access `?.[expr]`.
- [x] Import attributes / options objects on bounded import syntax.
- [x] `with` statements.
- [x] Standalone block statements.
- [x] Regular expression literals.

## P2 - Selector And Query Expansion

- [x] Comma-separated selector lists are accepted by the shared selector engine and by document /
  element query helpers.
- [x] Element-bound `querySelector` / `querySelectorAll` search descendants only instead of
  including the element itself.
- [x] Document-level and element-level query semantics stay aligned on the shared selector engine.
- [x] `:has()` accepts sibling-relative selectors (`+` / `~`) via `:scope` anchoring.
- [x] `:nth-child()` / `:nth-last-child()` accept bounded `of selector-list` filters.
- [x] `:is()` / `:where()` use forgiving selector lists and ignore invalid items.
- [x] `:not()` / `:has()` also use forgiving selector lists and ignore invalid items.
- [x] `:enabled` / `:disabled` respect disabled fieldset and optgroup ancestry, and disabled
  controls are ignored by `:required` / `:optional` / `:valid` / `:invalid` / `:user-valid` /
  `:user-invalid`.
- [x] `:read-only` / `:read-write` honor inherited `contenteditable` on non-input/textarea elements.
- [x] `:open` now covers select drop-down boxes and picker-capable inputs through the bounded
  open sentinel.
- [x] `:blank` now covers unchecked checkable controls and empty selects in addition to text-like
  inputs and textareas.
- [x] `:active` / `:hover` also include labeled controls via bounded label lookup.
- [x] `:default` keeps initial checked/selected snapshots for checkable controls and options.
- [ ] Add new selector slices only when a user-visible gap appears and the HTML standard calls for
  it.
- [ ] Add additional live collection slices only when a concrete gap justifies the cost.
- [ ] Extend beyond the current bounded pseudo-class slice only when a specific scenario requires
  it.

## P2 - DOM Surface Backlog

The DOM store and selector layers are already in place. The remaining backlog is mostly script-visible
DOM surface exposure through the runtime bridge; keep this list bounded and add to it only when a
failing test proves the surface is user-visible.

- [x] Expose the remaining `document` properties through the runtime bridge: `title`, `readyState`,
  `activeElement`, `baseURI`, `URL`, `doctype`, `documentURI`, `defaultView`, `compatMode`,
  `contentType`, `designMode`, and `dir`.
- [x] Expose the remaining `Node` / `Element` tree-navigation properties through the runtime bridge:
  `nodeType`, `nodeName`, `nodeValue`, `ownerDocument`, `parentNode`, `parentElement`,
  `firstChild`, `lastChild`, `firstElementChild`, `lastElementChild`, `nextSibling`,
  `previousSibling`, `nextElementSibling`, `previousElementSibling`, and `childElementCount`.
- [x] Expose the bounded element reflection surfaces through the runtime bridge:
  `className`, `innerText`, `outerText`, `style`, and `attributes`.
- [x] Expose the remaining template-driven standard DOM surfaces through the runtime bridge:
  standard `window` / `document` / `element` `addEventListener`, `details.open`,
  `element.classList`, `input.select()`, `document.createElement()`, `setAttribute()`,
  `appendChild()` / `removeChild()`, `document.execCommand("copy")`, and `window.confirm()`.
- [x] Add the remaining DOM construction and low-level mutation methods only when a test needs
  them: `createTextNode`, `replaceChild`, `insertAdjacentElement`, and `insertAdjacentText`.
- [x] Add collection/member parity only as bounded slices: `NodeList.forEach()`, `NodeList.entries()`,
  `NodeList.keys()`, `NodeList.values()`, and any other live-collection helper that a failing test
  proves visible.
- [ ] Keep legacy or deprecated DOM branches such as `document.all` out of scope unless the
  capability matrix explicitly adds them.

## P2 - Reflection, Mutation, And Serialization

- [ ] Fill any missing bounded reflection helpers or tree-mutation slices only when tests expose a
  gap.
- [x] Keep `textarea` reset-default synchronization covered by tests.
- [x] Keep `WriteHTML()` rollback behavior covered by tests.
- [x] Verify live collections remain coherent after mutation.

## P3 - Mock Families And Debug Surfaces

- [ ] Add new mock families only through the runtime registry and expose them through the typed
  facade.
- [ ] Document seed state, capture behavior, failure injection, reset semantics, and a minimal
  example for every new mock family.
- [ ] Add new debug snapshots only when they explain a real regression or user-visible gap.
- [ ] Avoid growing `Harness` into a bag of `set_*` methods.

## Explicit Non-Goals

- Renderer and layout engine.
- External network I/O.
- Legacy or deprecated spec branches unless the capability matrix explicitly adds them.
