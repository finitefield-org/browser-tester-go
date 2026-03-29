# Roadmap

This is the recommended build order for the Go workspace. The phases are intentionally sequential at
the start so the public facade stays small and the implementation stays testable.

## Phase 0: Scaffold

- Create the module, package root, internal packages, and CI-friendly tests.
- Define the public facade, error taxonomy, and explicit builder config.
- Land `Subsystem Map`, `Capability Matrix`, `Implementation Guide`, and `Mock Guide` before adding
  behavior.
- The scaffold is present now; later phases should extend it behind the thin facade rather than
  widening the API early.

Exit criteria:

- the package compiles
- the facade is thin
- the docs are in place

## Phase 1: DOM Core

- Parse HTML into the internal DOM store.
- Implement the initial selector subset, bounded attribute selectors, and a bounded combinator
  slice.
- Implement DOM dump and assertion helpers. The current workspace already has the initial assertion
  slice; later work should expand it only when a bounded user-visible gap appears.

Exit criteria:

- HTML round-trips deterministically in tests
- selectors work for the first supported slice

## Phase 2: Script Core

- Implement the minimum classic-JS script parser/evaluator slice.
- Bounded equality operators `==`, `!=`, `===`, and `!==` now work across bounded values, including
  object and array alias identity plus scalar coercion.
- Bounded relational `in` operators now fail with runtime errors on non-object right-hand values,
  bounded relational `instanceof` operators now fail with runtime errors on non-class right-hand
  values, bounded `for...of` loops now fail with runtime errors on non-iterable right-hand values,
  bounded `for...in` loops now fail with runtime errors on non-object right-hand values, and dynamic
  `import()` optional attributes now fail with runtime errors when the attributes value is not an
  object instead of generic unsupported errors.
- Classic scripts now reject `import.meta` syntax outside module scripts with a parse error instead
  of a generic unsupported error, while bounded module scripts expose `import.meta` as an object
  with a `url` property.
- Reserved declaration names such as `this`, `void`, `function`, and `class` now reject with parse
  errors in declaration positions and binding patterns instead of generic unsupported errors.
- Private names inside object literal methods now reject with parse errors in this bounded
  classic-JS slice instead of generic unsupported errors.
- Malformed object literal shorthand sequences now reject with parse errors in this bounded
  classic-JS slice instead of generic unsupported errors.
- Static class members named `prototype` are supported in the bounded class slice without breaking
  class instantiation, including setter-only `prototype` members that read back as `undefined` while
  keeping the hidden class prototype slot intact.
- Unary `+` / `-` now coerce bounded scalar values in the same classic-JS slice, logical negation
  `!` and logical `&&` / `||` now work across bounded values, `void` still discards any bounded
  expression result, while `+BigInt` remains an explicit unsupported error.
- `extends null` class inheritance is accepted in the bounded class slice and still instantiates
  with bounded class object semantics.
- Add host bindings needed for inline bootstrap, including bounded `innerHTML` / `outerHTML` /
  `textContent` helpers, bounded `dataset` reads, writes, and deletes for template-driven controls, a bounded
  `documentCurrentScript` helper for classic inline scripts, an explicit `expr(...)` wrapper for
  nested host expressions, and a bounded browser-global bridge for raw HTML bootstrap (`window`,
  `document`, `location`, `history`, `navigator`, `URL`, `DOMParser` (including parsererror fallbacks with `getElementsByTagName()`), `XMLSerializer`, `Intl.NumberFormat`,
  `Intl.Collator`, storage, `matchMedia`, `clipboard`, dynamic session-backed `window.<custom>`
  object properties, timers, and `console`).
- The current workspace already supports object literal shorthand properties and methods with any
  bound value, object literal computed property names and methods, object literal getter/setter
  accessors, bounded `throw` statements with catch-bound values, optional catch binding (`catch {}`),
  and catch binding patterns, bounded
  `delete` expressions on local object, array, string, and primitive number/boolean/bigint bindings
  including optional chaining and array `length`, bounded property assignment on local
  object/array bindings, including creating missing plain object properties on write, with getter-only property assignments failing with runtime errors, private
  class fields, bounded private `in` operator on bounded class private fields, bounded `super`
  property assignment, and bounded `super` deletion on bounded prototype targets and bounded
  null-prototype object literals, while `super` outside bounded class and object literal methods
  rejects with parse errors, bounded prefix/postfix increment and decrement expressions on local
  bindings and object/array property chains, bounded logical assignment operators and other bounded
  compound assignment operators on local bindings and object/array property chains, bounded
  relational `in` operator on bounded object and array values, bounded relational `instanceof`
  operator on bounded class objects and bounded constructible plain function values, bounded
  conditional `?:` operator, bounded exponentiation `**` / `**=` operators, bounded bitwise and
  shift operators, nullish coalescing with `??` across bounded values, bounded break / continue
  statements across loop, switch, and try bodies, including labeled loop / switch / try forms, class
  declarations with static blocks, public `static` fields, malformed class body member sequences
  that reject with parse errors, class getter/setter accessors including private and computed names,
  private fields and methods, computed fields and methods, instance fields, bounded `extends`
  inheritance, bounded `super` property, method, and constructor calls, bounded `new Class()`
  instantiation, and bounded `new (class {...})()` / `new (class extends Base {...})()`
  class-expression instantiation, including class-valued expression bases, plus plain `function`
  declarations with `return` statements and default parameter values plus bounded constructible
  plain function constructors with `this` property creation and `instanceof`, bounded `new.target`
  inside function and constructor bodies, bounded `this` expressions using the current receiver
  binding inside function and method bodies and `undefined` at top level, bounded function-like
  parameter destructuring patterns with default values and rest identifiers, plain `async function`
  declarations and expressions with `await` across bounded values, bounded async generator functions
  and async class methods with `await` / `yield` / `yield*` statements, bounded generator class
  methods with standalone `yield` statements, top-level `await` at the dispatch entrypoint, bounded
  module syntax through inline module scripts, including `import` declarations, optional import
  attributes / options objects on bounded import syntax, `export` declarations and specifier lists
  with `default` aliasing such as `import { default as name }` and `export { value as default }`,
  `export default class` declarations, re-export syntax with `from`, including namespace re-exports
  like `export * as ns from ...`, dynamic `import()` with string-compatible specifiers and optional
  attributes objects, and bounded `import.meta.url` module metadata, plus array/object destructuring
  patterns in `let` / `const` / `var` declarations with default binding values, plus bounded array
  destructuring assignment expressions on assignable member-expression targets, including computed
  object binding keys, bounded `using` / `await using` declarations, bounded array and iterator-like
  object `for...of` loops over string, array, and iterator-like object values, bounded array and
  iterator-like object `for await...of` loops over string, array, and iterator-like object values,
  including async iterator-like objects whose `next()` returns a promise value, bounded
  string/object/array `for...in` loops, bounded loop bodies with explicit terminators, plus
  single-statement loop bodies with explicit terminators, and bounded single-statement `if` / `else`
  control flow with explicit terminators, and bounded tagged template literals with function tags;
  the parser now also accepts numeric literals across decimal, hexadecimal, binary, and octal forms
  including numeric separators, plus `BigInt` literals; object literal spread now also accepts
  string and array values; bounded generator `next(arg)` calls accept arguments in this slice, with
  sent values forwarded into declaration initializers, object and array property-assignment RHSs,
  and currently suspended `yield*` delegated iterators; plain binding assignments are supported in
  this slice, while undeclared assignments remain unsupported in this slice; `yield` expressions are
  also supported in this slice, `yield*` delegation now also surfaces final return values from
  iterator-like objects when delegation completes, `return(value)` closes the iterator in this
  slice, and `throw(value)` remains an explicit runtime error in this slice, with an omitted
  argument treated as `undefined`; keep unsupported syntax explicit and list the remaining gaps in
  `TODO.md`.
- The current workspace already supports object literal shorthand properties and methods with any
  bound value, object literal computed property names and methods, object literal getter/setter
  accessors, bounded `throw` statements with catch-bound values, bounded `delete` expressions on
  local object, array, and string bindings including optional chaining and array `length`, bounded
  property assignment on local object/array bindings, including creating missing plain object
  properties on write, private class fields, and bounded
  `super` property assignment, bounded prefix/postfix increment and decrement expressions on local
  bindings and object/array property chains, bounded logical assignment operators and other bounded
  compound assignment operators on local bindings and object/array property chains, bounded
  relational `in` operator on bounded object and array values, bounded relational `instanceof`
  operator on bounded class objects, bounded conditional `?:` operator, bounded exponentiation `**`
  / `**=` operators, bounded bitwise and shift operators, bounded break / continue statements across
  loop, switch, and try bodies, including labeled loop / switch / try forms, class declarations with
  static blocks, public `static` fields, class getter/setter accessors including private and
  computed names, private fields and methods, computed fields and methods, instance fields, bounded
  `extends` inheritance, bounded `super` property, method, and constructor calls, and bounded `new
  Class()` instantiation, plus plain `function` declarations with `return` statements and default
  parameter values, bounded `new.target` inside function and constructor bodies, bounded
  function-like parameter destructuring patterns with default values and rest identifiers, plain
  `async function` declarations and expressions with `await` across bounded values, bounded async
  generator functions and async class methods with `await` / `yield` / `yield*` statements, bounded
  generator class methods with standalone `yield` statements, top-level `await` at the dispatch
  entrypoint, bounded module syntax through inline module scripts, including `import` declarations,
  `export` declarations and specifier lists with `default` aliasing such as `import { default as
  name }` and `export { value as default }`, `export default class` declarations, re-export syntax
  with `from`, including namespace re-exports like `export * as ns from ...`, dynamic `import()`
  with string-compatible specifiers, and bounded `import.meta.url` module metadata, plus
  array/object destructuring patterns in `let` / `const` / `var` declarations with default binding
  values, bounded `using` / `await using` declarations, and bounded array and iterator-like object
  `for...of` loops over string, array, and iterator-like object values, bounded array and
  iterator-like object `for await...of` loops over string, array, and iterator-like object values,
  including async iterator-like objects whose `next()` returns a promise value, bounded
  string/object/array `for...in` loops, and bounded tagged template literals with function tags;
  object literal async / generator methods are also supported in the same bounded slice; class
  expressions are also supported in the same bounded class slice without leaking the class name into
  the outer scope, can be instantiated directly with bounded `new` expressions, and can be used as
  bounded `extends` bases, including class-valued expression bases; array spread and array
  destructuring also accept string values and iterator-like objects with a `next()` method; keep
  unsupported syntax explicit and list the remaining gaps in `TODO.md`.
- Tagged template literals with bounded function tags are supported in the same bounded classic-JS
  slice, and non-callable tags now fail with runtime errors instead of generic unsupported errors.
- `new Class(...args)` constructor argument passing is also supported in the bounded class
  instantiation slice.
- Anonymous default-exported async function / async generator forms are also supported in the
  bounded module syntax slice.
- Anonymous default-exported plain function forms are also supported in the bounded module syntax
  slice.
- The bounded module syntax slice also supports default import plus namespace import combinations
  such as `import seeded, * as ns from ...`.
- The bounded loop slice also supports `using` declarations in bounded `for...of` / `for...in`
  headers, plus `await using` declarations in bounded `for await...of` headers.
- Module re-export syntax with `from` also accepts optional import attributes / options objects on
  bounded module scripts, including namespace re-exports and default + namespace import
  combinations.
- The classic-JS comma operator / sequence expressions are also supported, while commas inside
  array/object literals and call arguments remain separators.
- Object literal methods can read `super` through bounded prototype targets.
- Object literal methods can also read `super` through bounded null-prototype object literals.
- Object literal methods can also write `super` through bounded null-prototype object literals,
  including compound assignments.
- Class expressions are also supported in the same bounded class slice without leaking the class
  name into the outer scope, can be instantiated directly with bounded `new` expressions, and can be
  used as bounded `extends` bases, including class-valued expressions returned from bounded
  functions and bounded constructible function values with bounded `.prototype` access and class
  field initializers / computed member names that can read `super`.
- Call argument spread now accepts string values, bounded array values, and iterator-like objects
  with a `next()` method.
- Object spread now treats `null` and `undefined` as no-op spreads while still accepting string,
  array, object, and other primitive values as no-op spreads.
- Array literal elisions are accepted as bounded undefined slots, so `[ , 1, , ]` now parses in the
  same classic-JS slice.
- yield* delegation now also accepts string values in addition to arrays and iterator-like objects,
  now surfaces final return values from iterator-like objects when delegation completes, now fails
  with runtime errors for scalar inputs, and now resumes correctly from nested block bodies.
- Bare `import(...)` expression statements also use the same bounded dynamic-import path as `await
  import(...)`.

Exit criteria:

- inline scripts can mutate the DOM
- errors are classified and repeatable

## Phase 3: Events and Form Controls

- Add bounded capture/target/bubble event dispatch for click/input/change/submit, target-only
  focus/blur/reset behavior, `preventDefault`/`stopPropagation`-style event control, bounded
  listener removal, bounded `once` listeners, default actions, and user-like actions.
- Add input, checkbox, select, focus, blur, and submit behavior.
- Keep broader event semantics out of scope until a bounded need appears.

Exit criteria:

- deterministic event order
- deterministic form-control state updates

## Phase 4: Deterministic Runtime and Mocks

- Add fake clock, scheduler, bounded timers, bounded animation-frame callbacks, a bounded microtask
  queue, bounded history entries/state/scroll restoration, a bounded cookie jar, bounded web-storage
  state, and bounded `window.name` state. The current workspace already has `host:queueMicrotask()`
  draining at script/action boundaries.
- Add typed mock families and thin `Harness` actions.
- Keep capture and failure injection explicit.

Exit criteria:

- mock families are inspectable
- time-based behavior is deterministic
- timer-driven behavior is driven through `AdvanceTime()` and stays bounded
- history-driven behavior is deterministic and bounded
- cookie-state behavior is deterministic and bounded
- web-storage behavior is deterministic and bounded
- window-name behavior is deterministic and bounded

## Phase 5: Hardening

- Add subsystem tests, public contract tests, regression tests, and property tests.
- The current workspace already has seeded fuzz/property tests for the script, selector,
  timer/scheduler, location/history, location getters, cookie/window.name, and mock registry
  boundaries, plus read-only history stack snapshots through `DebugView.HistoryEntries()`, pending
  timer / animation-frame snapshots through `DebugView.PendingTimers()` /
  `DebugView.PendingAnimationFrames()`, pending microtask snapshots through
  `DebugView.PendingMicrotasks()`, cookie-jar snapshots through `DebugView.CookieJar()`, and
  registered event listener snapshots through `DebugView.EventListeners()`; keep extending those
  before widening parser surfaces.
- Read-only size snapshots such as `DebugView.SelectCount()`, `DebugView.TemplateCount()`,
  `DebugView.TableCount()`, `DebugView.ButtonCount()`, `DebugView.TextAreaCount()`,
  `DebugView.InputCount()`, `DebugView.FieldsetCount()`, `DebugView.LegendCount()`,
  `DebugView.OutputCount()`, `DebugView.LabelCount()`, `DebugView.ProgressCount()`,
  `DebugView.MeterCount()`, `DebugView.AudioCount()`, `DebugView.VideoCount()`,
  `DebugView.IframeCount()`, `DebugView.EmbedCount()`, and `DebugView.TrackCount()` should be kept
  in sync with the same DOM slices when they help explain parser or collection regressions.
- `DebugView.PictureCount()` should be kept in sync with the same DOM slice when it helps explain
  picture-collection regressions.
- `DebugView.SourceCount()` should be kept in sync with the same DOM slice when it helps explain
  source-collection regressions.
- `DebugView.DialogCount()` / `DebugView.DetailsCount()` / `DebugView.SummaryCount()` /
  `DebugView.SectionCount()` / `DebugView.MainCount()` / `DebugView.ArticleCount()` /
  `DebugView.NavCount()` / `DebugView.AsideCount()` / `DebugView.FigureCount()` /
  `DebugView.FigcaptionCount()` / `DebugView.HeaderCount()` / `DebugView.FooterCount()` /
  `DebugView.AddressCount()` / `DebugView.BlockquoteCount()` / `DebugView.ParagraphCount()` /
  `DebugView.PreCount()` / `DebugView.MarkCount()` / `DebugView.QCount()` / `DebugView.CiteCount()`
  / `DebugView.AbbrCount()` / `DebugView.StrongCount()` / `DebugView.SpanCount()` /
  `DebugView.DataCount()` / `DebugView.DfnCount()` / `DebugView.KbdCount()` / `DebugView.VarCount()`
  / `DebugView.CodeCount()` / `DebugView.SmallCount()` / `DebugView.TimeCount()` should be kept in
  sync with the same DOM slice when they help explain
  dialog/details/summary/section/main/article/nav/aside/figure/figcaption/header/footer/address/
  blockquote/paragraph/pre/mark/q/cite/abbr/strong/span/data/dfn/kbd/var/code/small/time collection
  regressions.
- `DebugView.SampCount()` should be kept in sync with the same DOM slice when it helps explain samp
  collection regressions.
- `DebugView.RubyCount()` should be kept in sync with the same DOM slice when it helps explain ruby
  collection regressions.
- `DebugView.RtCount()` should be kept in sync with the same DOM slice when it helps explain rt
  collection regressions.
- Define the release checklist. The checklist now lives in `release-checklist.md`.

Exit criteria:

- behavior is covered at the public boundary and the internal boundary

## Phase 6: Selector and Query Expansion

- Expand selector support in bounded slices, including bounded attribute selectors such as `[attr]`,
  `[attr=value]`, `[attr~=value]`, `[attr|=value]`, `[attr^=value]`, `[attr$=value]`, and
  `[attr*=value]`, descendant, child, sibling combinators, and a bounded pseudo-class slice for
  `:root`, `:scope`, `:defined`, `:state(identifier)`, `:active`, `:hover`, `:empty`, `:checked`,
  `:indeterminate`, `:autofill`, `:-webkit-autofill`, `:default`, `:enabled`, `:disabled`,
  `:required`, `:optional`, `:read-only`, `:read-write`, `:valid`, `:invalid`, `:user-valid`,
  `:user-invalid`, `:in-range`, `:out-of-range`, `:first-child`, `:last-child`, `:first-of-type`,
  `:last-of-type`, `:only-child`, `:only-of-type`, `:nth-child()`, `:nth-of-type()`,
  `:nth-last-child()`, `:nth-last-of-type()`, `:link`, `:any-link`, `:visited`, `:local-link`,
  `:lang()`, `:dir()`, `:placeholder-shown`, `:blank`, `:heading`, `:heading(integer#)`, `:playing`,
  `:paused`, `:seeking`, `:buffering`, `:stalled`, `:muted`, `:volume-locked`,
  `:picture-in-picture`, `:fullscreen`, `:modal`,
  `:popover-open`, `:open`, `:focus`, `:focus-visible`, `:focus-within`, `:target`,
  `:target-within`, `:is()`, `:where()`, `:not()`, and `:has()` first.
- Add script-side query APIs that reuse the same selector engine, plus a minimal snapshot `NodeList`
  and bounded live `HTMLCollection` / `NodeList` slices for `children`, `document.images`,
  `document.forms`, `form.elements`, `fieldset.elements`, `select.options`,
  `select.selectedOptions`, `datalist.options`, `table.rows`, `table.tBodies`,
  `HTMLTableSectionElement.rows`, `tr.cells`, `document.scripts`, `document.links`,
  `document.anchors`, `childNodes`, `template.content.childNodes`, and script/image/form-count
  inspection.
- Expand live collections only as needed by user-visible gaps.

Exit criteria:

- DOM and script querying use the same core selector logic

## Phase 7: Reflection, Mutation, Serialization

- Add bounded attribute reflection helpers, then bounded `classList` / `dataset` views, tree
  mutation primitives, and HTML serialization/insertion surfaces.
- The current workspace already has public `classList` / `dataset` views plus the public
  selector-based tree-mutation wrappers, including `TextContent()` / `SetTextContent()`,
  `ReplaceChildren()`, `CloneNode()`, and `WriteHTML()` for bounded document-write-style replay; on
  `textarea`, content mutations that change its contents keep the reset default value in sync; keep
  expanding only the slices that are still missing.
- Keep each slice bounded and documented.

Exit criteria:

- mutation updates the DOM deterministically
- live collections stay coherent after mutation

## Working Rules

- Do not move to a later phase until the earlier phase is covered by tests.
- Do not add a new public `Harness` method until the capability matrix has a row for it.
- Use `../html-standard/` when adding or changing HTML behavior.
- Prefer small slices over large parity pushes.
