# Go Workspace Backlog

This file tracks only the remaining implementation work.

## String API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `String` APIs beyond the
  methods already listed in `README.md` and `doc/capability-matrix.md`.
- Schedule the missing current members explicitly:
- Keep legacy and deprecated `String` members out of scope for this slice, including `substr`,
  `trimLeft`, `trimRight`, and the HTML wrapper helpers such as `anchor()`, `big()`, `blink()`,
  `bold()`, `fixed()`, `fontcolor()`, `fontsize()`, `italics()`, `link()`, `small()`, `strike()`,
  `sub()`, and `sup()`.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## Array API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `Array` APIs beyond the
  members already listed in `README.md` and `doc/capability-matrix.md`.
- Schedule the missing current members explicitly:
  - `Array.fromAsync()`
  - `Array[Symbol.species]`
  - `Array.prototype[@@iterator]`
  - `Array.prototype[@@unscopables]`
  - `Array.prototype.toSpliced()`
  - `Array.prototype.with()`
- Keep legacy and deprecated Array compatibility branches out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## Map API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `Map` APIs beyond the
  constructor / `get` / `set` / `has` / `delete` / `clear` / `forEach` / `keys` / `values` / `entries`
  surface already present in the bounded classic-JS slice.
- Schedule the missing current members explicitly:
  - `Map.groupBy()`
  - `Map[Symbol.species]`
  - `Map.prototype.getOrInsert()`
  - `Map.prototype.getOrInsertComputed()`
  - `Map.prototype[@@iterator]`
  - `Map.prototype[@@toStringTag]`
  - `Map iterator` object branding and self-iterability for the iterators returned by
    `keys()` / `values()` / `entries()`
- Keep legacy and deprecated Map compatibility branches out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## Set API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `Set` APIs beyond the
  constructor / `add` / `has` / `delete` / `keys` / `values` / `entries` surface already present
  in the bounded classic-JS slice.
- Schedule the missing current members explicitly:
  - `Set.groupBy()`
  - `Set[Symbol.species]`
  - `Set.prototype.difference()`
  - `Set.prototype.forEach()`
  - `Set.prototype.getOrInsert()`
  - `Set.prototype.getOrInsertComputed()`
  - `Set.prototype.intersection()`
  - `Set.prototype.isDisjointFrom()`
  - `Set.prototype.isSubsetOf()`
  - `Set.prototype.isSupersetOf()`
  - `Set.prototype.symmetricDifference()`
  - `Set.prototype.union()`
  - `Set.prototype[@@iterator]`
  - `Set.prototype[@@toStringTag]`
  - `Set iterator` object branding and self-iterability for the iterators returned by
    `keys()` / `values()` / `entries()`
- Keep legacy and deprecated Set compatibility branches out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## ArrayBuffer API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `ArrayBuffer` APIs beyond
  the typed-array `buffer` placeholder already present in the bounded runtime slice.
- Schedule the missing current members explicitly:
  - `ArrayBuffer` constructor / `instanceof ArrayBuffer` semantics for `new ArrayBuffer(length,
    options)` if the runtime still lacks the full constructible ArrayBuffer shape
  - `ArrayBuffer.prototype`
  - `ArrayBuffer.isView()`
  - `ArrayBuffer[Symbol.species]`
  - `ArrayBuffer.prototype.byteLength`
  - `ArrayBuffer.prototype.constructor`
  - `ArrayBuffer.prototype.detached`
  - `ArrayBuffer.prototype.maxByteLength`
  - `ArrayBuffer.prototype.resizable`
  - `ArrayBuffer.prototype.resize()`
  - `ArrayBuffer.prototype.slice()`
  - `ArrayBuffer.prototype.transfer()`
  - `ArrayBuffer.prototype.transferToFixedLength()`
  - `ArrayBuffer.prototype[@@toStringTag]`
- Keep legacy and deprecated ArrayBuffer compatibility branches out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## JSON API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `JSON` APIs beyond the
  `parse()` / `stringify(value, replacer, space)` slice already present in `README.md` and
  `doc/capability-matrix.md`.
- Schedule the missing current members explicitly:
  - `JSON.parse(text, reviver)` including reviver traversal and property deletion semantics
  - `JSON[Symbol.toStringTag]`
- Keep legacy and deprecated JSON compatibility branches out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## Number API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `Number` APIs beyond the
  members already listed in `README.md` and `doc/capability-matrix.md`.
- Schedule the missing current members explicitly:
  - `Number` constructor / wrapper semantics for `new Number(...)` and `instanceof Number`
  - `Number.prototype[@@toPrimitive]` if the wrapper coercion hook is still missing
- Keep legacy and deprecated `Number` members out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## BigInt API Expansion

- Expand the bounded browser stdlib slice to cover the remaining current `BigInt` APIs beyond the
  literal/arithmetic support already present in the bounded classic-JS slice.
- Schedule the missing current members explicitly:
  - `BigInt()` callable conversion function, not a constructor
  - `BigInt.asIntN()`
  - `BigInt.asUintN()`
  - `BigInt.prototype.toLocaleString()`
  - `BigInt.prototype.toString()`
  - `BigInt.prototype.valueOf()`
  - `BigInt.prototype[@@toStringTag]`
  - BigInt wrapper / instance semantics for `Object(1n)` and `instanceof BigInt` if the runtime
    still lacks the `BigInt` object shape
- Keep legacy and deprecated BigInt-related compatibility branches out of scope for this slice.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.

## RegExp API Expansion

- The native engine design and migration plan live in `doc/adr/0001-native-regexp-engine.md`;
  implement that engine first so the later API rows share one matcher.
- Expand the bounded browser stdlib slice to cover the remaining current `RegExp` APIs beyond the
  regular-expression literal support already present in the bounded classic-JS slice.
- Schedule the missing current members explicitly:
  - `RegExp()` callable / constructible semantics for `RegExp(pattern, flags)` and
    `new RegExp(pattern, flags)`
  - `RegExp.escape()`
  - `RegExp[Symbol.species]`
  - `RegExp.prototype.constructor`
  - `RegExp.prototype.exec()`
  - `RegExp.prototype.test()`
  - `RegExp.prototype.toString()`
  - `RegExp.prototype.lastIndex`
  - `RegExp.prototype.dotAll`
  - `RegExp.prototype.flags`
  - `RegExp.prototype.global`
  - `RegExp.prototype.hasIndices`
  - `RegExp.prototype.ignoreCase`
  - `RegExp.prototype.multiline`
  - `RegExp.prototype.source`
  - `RegExp.prototype.sticky`
  - `RegExp.prototype.unicode`
  - `RegExp.prototype.unicodeSets`
  - `RegExp.prototype[@@match]`
  - `RegExp.prototype[@@matchAll]`
  - `RegExp.prototype[@@replace]`
  - `RegExp.prototype[@@search]`
  - `RegExp.prototype[@@split]`
- Keep legacy and deprecated RegExp compatibility branches out of scope for this slice, including
  `RegExp.prototype.compile()` and the Annex B `RegExp.$1`-style static capture properties.
- When this slice is implemented, add the owning-subsystem tests, failure-path coverage, and any
  doc updates required by the workspace rules.
