# browser-tester Go Workspace

This directory is the Go implementation track for `browser-tester`. It is intentionally
conservative: a thin public facade, explicit builder config, typed mock families, and bounded
runtime slices. The detailed support map lives in `doc/capability-matrix.md`.

## Public Surface

- `Harness`
- `HarnessBuilder`
- `Error` and `ErrorKind`
- `DebugView`
- `Interaction` and `InteractionKind`
- `MockRegistryView`
- `OptionLabel`
- `OptionValue`
- `OptgroupLabel`
- user-like actions that delegate into runtime or mock families:
  - `Fetch`
  - `Alert`
  - `Confirm`
  - `Prompt`
  - `Click`
  - `TypeText`
  - `SetValue`
  - `SetChecked`
  - `SetSelectValue`
  - `Focus`
  - `Blur`
  - `Submit`
  - `ReadClipboard`
  - `WriteClipboard`
  - `MatchMedia` (including listener capture injection through the mock registry)
  - `Open`
  - `Close`
  - `Print`
  - `ScrollTo`
  - `ScrollBy`
  - `Navigate`
  - `AdvanceTime`
  - `SetFiles`
  - `CaptureDownload`
- bounded `window.history` host helpers for inline scripts:
  - `historyPushState` / `historyReplaceState` / `historySetScrollRestoration` use browser-style string coercion for string inputs, including runtime rejection of `Symbol` inputs
  - `historyBack`
  - `historyForward`
  - `historyGo`
  - `historyLength`
  - `historyState`
  - `historyScrollRestoration`
  - `historySetScrollRestoration`
- bounded location read helpers for inline scripts:
  - `locationHref`
  - `locationOrigin`
  - `locationProtocol`
  - `locationHost`
  - `locationHostname`
  - `locationPort`
  - `locationPathname`
  - `locationSearch`
  - `locationHash`
- bounded cookie helpers for inline scripts:
  - `documentCookie`
  - `setDocumentCookie`
  - `navigatorCookieEnabled`
- bounded web-storage helpers for inline scripts:
  - `localStorageGetItem`
  - `localStorageSetItem`
  - `localStorageRemoveItem`
  - `localStorageClear`
  - `localStorageLength`
  - `localStorageKey`
  - `sessionStorageGetItem`
  - `sessionStorageSetItem`
  - `sessionStorageRemoveItem`
  - `sessionStorageClear`
  - `sessionStorageLength`
  - `sessionStorageKey`
- bounded current-script helper for inline scripts:
  - `documentCurrentScript`
- bounded browser-global bridge for raw HTML bootstrap: `window` / `self` / `globalThis` / `top` /
  `parent` / `frames`, `document` (including `title`, `readyState`, `activeElement`, `baseURI`,
  `URL` / `URLSearchParams`, `doctype`, `documentURI`, `defaultView`, `compatMode`, `contentType`,
  `designMode`, and
  `dir`, plus bounded `Node` / `Element` tree-navigation reads on `nodeType`, `nodeName`,
  `nodeValue`, `ownerDocument`, `parentNode`, `parentElement`, `firstChild`, `lastChild`,
  `firstElementChild`, `lastElementChild`, `nextSibling`, `previousSibling`, `nextElementSibling`,
  `previousElementSibling`, `childElementCount`, `contains()`, `isConnected()`, `getRootNode()`, `compareDocumentPosition()`, and `hasChildNodes()`, plus text-node
  `nodeValue` / `data` reads and writes, `wholeText` reads, `splitText()` mutation, `before()` /
  `after()` / `append()` / `prepend()` / `replaceChildren()` / `replaceWith()` / `remove()` node mutation helpers, and
  `normalize()` mutation), `location` (including `assign()` / `replace()` / `reload()` and property setters with browser-style string coercion and runtime rejection of `Symbol` inputs), `history`, `navigator`, `URL` /
  `URLSearchParams`, `Blob`, `URL.createObjectURL()` / `revokeObjectURL()` with `window.URL` alias parity, `DOMParser.parseFromString()` for
  `image/svg+xml` documents, including parsererror fallbacks with `getElementsByTagName()`, `namespaceURI` reads on parsed SVG nodes, `XMLSerializer.serializeToString()` for bounded SVG element nodes, `element.cloneNode()` on bounded element refs, `element.scrollIntoView()` as a bounded no-op scroll helper, `href` on `a` / `area` / `link`, `download` on `a` / `area`, and `tabIndex` on standard form controls, `Intl.NumberFormat` / `Intl.NumberFormat.prototype.resolvedOptions()` / `Intl.NumberFormat.prototype.formatToParts()` / `Intl.NumberFormat.supportedLocalesOf()` / `Intl.Collator.supportedLocalesOf()` / `Intl.Collator`, `CSS.escape()`, `localStorage`, `sessionStorage`,
  `matchMedia`, `fetch()`, `console`, `clipboard`, `window.open()` / bare `open()` string inputs use browser-style string coercion, open a blank popup when called without a URL, ignore extra arguments, and reject `Symbol` inputs at runtime, dynamic session-backed `window.<custom>` object properties such as
  `window.crypto` / `window.hashApi` (unset reads through `window` / `self` / `globalThis` /
  `top` / `parent` / `frames` return `undefined` like browser feature detection), `element.dir`, and
  bounded constructor globals for `HTMLElement` / `HTMLButtonElement` / `HTMLSelectElement` / `Image` / `HTMLImageElement` / `HTMLCanvasElement` / `Uint8Array` element checks, plus inline-script `toggleAttribute(name, force)` and `classList.toggle(token, force)` calls that use browser-style truthiness coercion for the optional force argument, plus blob-backed `Image` / `HTMLImageElement` `load` / `error` callbacks and bounded `canvas.getContext("2d")` / `drawImage()` / `toBlob()` / `toDataURL()` PNG export, and
  bounded timer globals (`setTimeout`, `setInterval`, `clearTimeout`, `clearInterval`,
  `requestAnimationFrame`, `cancelAnimationFrame`, `queueMicrotask`), plus a bounded browser
  stdlib slice for inline scripts: `Array` / `Object` / `JSON` / `Map` / `Set` / `Number` / `String`
  / `Boolean` / `Math` / `Date` / `Symbol` / `Uint8Array`, including the constructible `Array(...)` /
  `new Array(...)` / `instanceof Array` shape plus the template-facing `Array.from()` /
  `Array.isArray()`, and `Map.prototype.clear()` / `Set.prototype.clear()` / `Map.prototype.keys()` / `Map.prototype.values()` / `Map.prototype.entries()`
  plus `Set.prototype.keys()` / `Set.prototype.values()` / `Set.prototype.entries()`,
  `Object.assign()` / `Object.keys()` / `Object.getOwnPropertyNames()` / `Object.getOwnPropertySymbols()` / `Object.prototype.hasOwnProperty.call()` / `Object.hasOwn()`, `JSON.parse()` / `JSON.stringify(value, replacer, space)`,
  `Number.parseInt()` / `Number.parseFloat()` / global `parseInt()` / global `parseFloat()` / `encodeURI()` / `decodeURI()` / `encodeURIComponent()` / `decodeURIComponent()` / `Number.isInteger()` / `Number.isNaN()` / `Number.isFinite()` / `Number.isSafeInteger()` / `Number.EPSILON` / `Number.MAX_VALUE` / `Number.MIN_VALUE` / `Number.MAX_SAFE_INTEGER` / `Number.MIN_SAFE_INTEGER` / `Number.NaN` / `Number.POSITIVE_INFINITY` / `Number.NEGATIVE_INFINITY` / global `NaN` / global `Infinity`, `Date` constructor / `new Date()` / `instanceof Date` / `Date.now()` / `Date.UTC()`, `Math.E` / `Math.LN10` / `Math.LN2` / `Math.LOG10E` / `Math.LOG2E` / `Math.PI` / `Math.SQRT1_2` / `Math.SQRT2` / `Math.abs()` / `Math.pow()` / `Math.ceil()` / `Math.floor()` / `Math.min()` / `Math.max()` /
  `Math.round()` / `Math.trunc()` / `Math.random()` / `Math.acos()` / `Math.acosh()` / `Math.asin()` / `Math.asinh()` / `Math.atan()` / `Math.atan2()` / `Math.atanh()` / `Math.cbrt()` / `Math.clz32()` / `Math.cos()` / `Math.cosh()` / `Math.exp()` / `Math.expm1()` / `Math.fround()` / `Math.hypot()` / `Math.imul()` / `Math.log()` / `Math.log10()` / `Math.log1p()` / `Math.log2()` / `Math.sign()` / `Math.sin()` / `Math.sinh()` / `Math.sqrt()` / `Math.tan()` / `Math.tanh()`, `Date.now()` / `Date.UTC()` / `Intl.DateTimeFormat()` / `Intl.DateTimeFormat.supportedLocalesOf()` / `Intl.Collator()`, `String.fromCharCode()` / `String.fromCodePoint()` / `String.raw()` /
  `String.prototype.charAt()` / `String.prototype.charCodeAt()` / `String.prototype.at()` / `String.prototype.codePointAt()` / `String.prototype.normalize()` / `String.prototype.indexOf()` / `String.prototype.substring()` / `String.prototype.replace()` / `String.prototype.replaceAll()` /
  `String.prototype.matchAll()` / `String.prototype.search()` / `String.prototype.includes()` /
  `String.prototype.split()` / `String.prototype.trim()` / `String.prototype.trimStart()` /
  `String.prototype.trimEnd()` / `String.prototype.padStart()` / `String.prototype.padEnd()` /
  `String.prototype.repeat()` / `String.prototype.toLowerCase()` / `String.prototype.toUpperCase()` / `String.prototype.isWellFormed()` / `String.prototype.toWellFormed()` / `String.prototype.toLocaleLowerCase()` / `String.prototype.toLocaleUpperCase()` / `String.prototype.concat()` / `String.prototype.localeCompare(locale, options)` / `String.prototype.replaceAll(callback replacers)` / `String.prototype[@@iterator]` /
  `String.prototype.startsWith()` / `String.prototype.endsWith()`,
  `Array.prototype.at()` / `Array.prototype.includes()` / `Array.prototype.indexOf()` / `Array.prototype.lastIndexOf()` / `Array.prototype.findIndex()` /
  `Array.prototype.findLast()` / `Array.prototype.findLastIndex()` /
  `Array.prototype.every()` / `Array.prototype.fill()` / `Array.prototype.copyWithin()` /
  `Array.prototype.reduce()` / `Array.prototype.reduceRight()` / `Array.prototype.reverse()` /
  `Array.prototype.sort()` / `Array.prototype.shift()` / `Array.prototype.entries()` /
  `Array.prototype.keys()` / `Array.prototype.values()` / `Array.prototype.toLocaleString()` /
  `Array.prototype.toSorted()` / `Array.prototype.toReversed()` / `Array.from()` /
  `Array.of()` / `Array.isArray()` /
  `flatMap()` / `splice()` / `unshift()`,
  `Number.prototype.toPrecision()` /
  `toExponential()` / `toLocaleString(locale, options)` / `Date.prototype.toDateString()` / `Date.prototype.toTimeString()` / `Date.prototype.toLocaleString()` / `Date.prototype.toLocaleTimeString()` / `Date.prototype.toLocaleDateString()` / `Date.prototype.toUTCString()` / `Date.parse()` / `Date(string)` / `Date.prototype.getFullYear()` / `Date.prototype.getUTCFullYear()` / `Date.prototype.getMonth()` / `Date.prototype.getUTCMonth()` / `Date.prototype.getDate()` / `Date.prototype.getUTCDate()` / `Date.prototype.getDay()` / `Date.prototype.getUTCDay()` / `Date.prototype.getHours()` / `Date.prototype.getUTCHours()` / `Date.prototype.getMinutes()` / `Date.prototype.getUTCMinutes()` / `Date.prototype.getSeconds()` / `Date.prototype.getUTCSeconds()` / `Date.prototype.getMilliseconds()` / `Date.prototype.getUTCMilliseconds()` / `Date.prototype.getTimezoneOffset()` / `Date.prototype.setTime()` / `Date.prototype.setDate()` / `Date.prototype.setUTCDate()` / `Date.prototype.setMonth()` / `Date.prototype.setUTCMonth()` / `Date.prototype.setFullYear()` / `Date.prototype.setUTCFullYear()` / `Date.prototype.setMilliseconds()` / `Date.prototype.setUTCMilliseconds()` / `Date.prototype.setSeconds()` / `Date.prototype.setUTCSeconds()` / `Date.prototype.setMinutes()` / `Date.prototype.setUTCMinutes()` / `Date.prototype.setHours()` / `Date.prototype.setUTCHours()`, and the bounded array/string/number/date prototype helpers used by
  template-driven bootstrap, plus the live `URL` / `URLSearchParams` query-state bridge
  (`search`, `searchParams.set()`, `searchParams.getAll()`, `searchParams.entries()`,
  `searchParams.values()`, `searchParams.sort()`, `searchParams.keys()`, `forEach()`) for template query handling,
  plus `Object.entries()` / `Object.values()` / `Object.fromEntries()` for plain-object enumeration,
  and bounded `Intl.DateTimeFormat()` time-zone formatting with `formatToParts()` / `formatRange()` / `formatRangeToParts()` / `resolvedOptions()` / `supportedLocalesOf()`, bounded `Uint8Array`
  construction from array-like / buffer values, plus `Uint8Array.from()` with map-function support,
  `JSON.stringify(value, replacer, space)`, bounded `Promise.resolve()`, and bounded promise-style
  `then()` / `catch()` chains on browser promises such as `clipboard.writeText()` and `fetch()`,
  including executor `reject(...)` paths and rejected-promise propagation
- bounded event-target helper for inline event listeners:
  - `eventTargetValue`
- nested expression wrapper for inline scripts:
  - `expr(...)`
  - classic-JS inline script calls, with top-level function declarations hoisted before earlier
    statements in the same script, bounded array and object literals, including array literal
    elisions, bounded object literal shorthand properties and methods with any bound value, bounded
    object literal computed property names and methods, bounded object literal getter/setter
    accessors, bounded `throw` statements with catch-bound values, optional catch binding
    (`catch {}`), and catch binding patterns,
    bounded `debugger` statements as no-op statements,
    bounded `delete` expressions on local object, array, string, and primitive number/boolean/bigint
    bindings including optional chaining and array `length`, unary `typeof`, logical negation `!`,
    and logical `&&`/`||`, bounded relational `in` operators on bounded object and array values,
    bounded relational `instanceof` operators on bounded class objects and bounded constructible
    plain function values, bounded equality operators `==`, `!=`, `===`, and `!==` across bounded
    values, bounded conditional `?:` operator, bounded exponentiation `**` / `**=` operators,
    bounded bitwise and shift operators, bounded property assignment on local object/array
    bindings, plain function values, and host-reference property chains such as
    `document.getElementById(...).textContent = ...`, with nested helper calls preserving array
    binding updates across invocation frames so mutations like `push()` remain visible to the
    caller binding and cyclic array/object graphs are skipped safely during replacement, including
    creating missing plain object properties on write, with getter-only property
    assignments failing with runtime errors, private class
    fields, bounded private `in` operator on bounded class private fields, and bounded `super`
    property assignment, including class field initializers and computed class member names that can
    read `super`, even in base classes without `extends`, while `super` outside bounded class/object
    methods rejects with parse errors, bounded prefix/postfix increment and decrement expressions on
    local bindings and object/array property chains, bounded spread/rest syntax in array/object
    literals and `let` / `const` / `var` binding patterns, with object literal spread also accepting
    string and array values, array/object destructuring patterns in `let` / `const` / `var`
    declarations with default binding values, plus bounded array destructuring assignment
    expressions on assignable member-expression targets, including computed object binding keys,
    bounded
    `using` / `await using` declarations, bounded function-like parameter destructuring patterns
    with default values and rest identifiers, with malformed parameter syntax now rejecting with
    parse errors, bounded arrow functions with simple identifier/rest parameters and concise or
    block bodies, bounded async arrow functions with `await` expressions, plain `async function`
    declarations and expressions with `await` across bounded values, bounded async generator
    functions and async class methods with `await` / `yield` / `yield*` statements, plain `function`
    declarations with `return` statements and default parameter values plus bounded constructible
    plain function constructors with `this` property creation and `instanceof`, bounded `new.target`
    inside function and constructor bodies, top-level `await` at the dispatch entrypoint, bounded
    generator functions and expressions with standalone `yield` statements, `yield` expressions, and
    `yield*` delegation, including string values and final return values from iterator-like objects,
    with scalar inputs failing with runtime errors in this slice, plus nested `yield` in bounded
    block bodies, bounded loop bodies with explicit terminators, plus single-statement loop bodies
    with explicit terminators, bounded standalone block statements, bounded single-statement `if` /
    `else` control flow with explicit terminators, bounded `for...of` loops over arrays, bounded
    `for await...of` loops over arrays of
    awaited values inside bounded async bodies, bounded `for...in` loops over object/array keys,
    bounded `switch` clauses, bounded `try` / `catch` / `finally` blocks, bounded `break` /
    `continue` statements across those control-flow bodies, including labeled loop / switch / try
    forms, named self-binding, and iterator result objects from `next()`, bounded module syntax
    through inline module scripts (`<script type="module" id="...">`), including `import`
    declarations, optional import attributes / options objects on bounded import syntax, `export`
    declarations and export specifier lists with `default` aliasing such as `import { default as
    name }` and `export { value as default }`, `export default class` declarations and anonymous
    default-exported plain function / async function / async generator forms, re-export syntax with
    `from`, including optional import attributes / options objects and namespace re-exports like
    `export * as ns from ...`, dynamic `import()` with string-compatible specifiers and optional
    attributes objects, including bare expression statements, against the bounded module registry,
    nullish coalescing with `??` across bounded values, numeric literals across decimal,
    hexadecimal, binary, and octal forms including numeric separators, plus `BigInt` literals, local
    `let` / `const` / `var` bindings, logical assignment operators and other bounded compound
    assignment operators on local bindings and object/array property chains, class declarations with
    `static` blocks, public `static` fields, malformed class body member sequences that reject with
    parse errors, class getter/setter accessors including private and computed names, async class
    methods, generator class methods, private fields and methods, computed fields and methods,
    instance fields, bounded static/prototype methods, bounded extends inheritance for class and
    constructible function values, bounded super property, method, and constructor calls, and
    bounded `new Class()` / `new (class {...})()` / `new (class extends Base {...})()`
    instantiation, template literals with bounded `${...}` interpolation plus tagged template
    literals with bounded function tags and interpolation, bounded regular expression literals with
    bounded `.test()` / `.exec()` helpers and browser-style lookahead / backreference support, with
    non-callable tags failing explicitly at runtime,
    bounded object-property access, bounded bracket access on object, array, string, and primitive
    number/boolean/bigint values, dot access on number/boolean/bigint/string/array values yielding
    `undefined` for unknown properties, array `length` lookups, and optional chaining plus optional
    calls across those bounded chains:
  - bounded comma operator / sequence expressions are supported in classic-JS expressions, while
    commas inside array/object literals and call arguments still act as separators.
  - unary `+` / `-` and `void` now work on bounded scalar values, with `+BigInt` still rejected
    explicitly in this slice.
  - object literal async / generator methods are supported in the same bounded object-literal slice.
  - object literal methods with private names reject with parse errors in this bounded
    object-literal slice.
  - malformed object literal shorthand sequences reject with parse errors in this bounded
    object-literal slice.
  - object literal methods can read `super` through bounded prototype targets in the same bounded
    object-literal slice.
- object literal methods can also read `super` through bounded null-prototype object literals in the
  same bounded object-literal slice.
- object literal methods can also write `super` through bounded null-prototype object literals in
  the same bounded object-literal slice, including compound assignments.
- object literal and class methods can also delete `super` through bounded prototype targets and
  bounded null-prototype object literals in the same bounded object-literal/class-method slice.
- class expressions are supported in the same bounded class slice, can be instantiated directly with
  bounded `new` expressions, and can be used as bounded `extends` bases, including class-valued
  expressions returned from bounded functions and bounded constructible function values with bounded
  `.prototype` access.
  - static class members named `prototype` are supported in the same bounded class slice without
    breaking class instantiation, including setter-only `prototype` members that read back as
    `undefined` while keeping the hidden class prototype slot intact.
  - `extends null` is accepted as a bounded class inheritance form.
  - classic-script `import.meta` syntax outside module scripts is rejected with a parse error, while
    bounded module scripts expose `import.meta` as an object with a `url` property.
  - module import attributes / options objects also apply to default + namespace import combinations
    such as `import seeded, * as ns from ... with { ... }`.
  - reserved declaration names such as `this`, `void`, `function`, and `class` reject with parse
    errors in declaration positions and binding patterns.
  - `new.target` outside function or constructor bodies is rejected with a parse error.
  - `host.method(...)`
  - `host?.method(...)`
  - `host?.["method"](...)`
  - bounded `new Class(...args)` constructor arguments in the class instantiation slice
  - bounded `this` expressions resolve to the current receiver inside function and method bodies and
    to `undefined` at top level
  - `obj.prop`
  - `obj["prop"]`
  - `str[0]`
  - `obj?.prop`
  - `obj.items.length`
  - `obj?.["prop"]`
  - `obj.write?.()`
  - plain backtick-delimited string literals
- remaining JS syntax gaps are tracked in `TODO.md` until the bounded classic-JS slice expands
  further.
- bounded module syntax also supports default import plus namespace import combinations such as
  `import seeded, * as ns from ...`.
- bare `import(...)` expression statements also use the bounded dynamic-import path in classic
  scripts.
- anonymous default-exported plain function forms are also supported in the bounded module syntax
  slice.
- bounded loop headers also support `using` declarations inside `for...of` and `for...in` loops,
  plus `await using` declarations inside `for await...of` loops.
- bounded call argument spread accepts string values, bounded array values, and iterator-like
  objects with a `next()` method.
- bounded array spread and array destructuring also accept string values and iterator-like objects
  with a `next()` method.
- bounded `for...of` loops accept bounded string values, bounded array values, and iterator-like
  objects with a `next()` method, and bounded `for await...of` loops also accept async iterator-like
  objects whose `next()` returns a promise value, including destructuring binding patterns.
- bounded `for...in` loops accept bounded string values, bounded object values, and bounded array
  values.
- bounded object spread treats `null` and `undefined` as no-op spreads, while still accepting
  string, array, object, and other primitive values as no-op spreads.
- bounded bracket access also reads string values by index and exposes string `length`, and dot
  access on string and array values yields `undefined` for unknown properties.
- bounded equality comparisons (`==`, `!=`, `===`, `!==`) now work across bounded values, including
  object and array alias identity and scalar coercion.
- bounded relational `in` operators now report runtime errors on non-object right-hand values
  instead of generic unsupported errors, bounded relational `instanceof` operators now report
  runtime errors on non-class right-hand values instead of generic unsupported errors, bounded
  `for...of` and `for...in` loops now report runtime errors on non-iterable and non-object
  right-hand values respectively instead of generic unsupported errors, and dynamic `import()`
  optional attributes now report runtime errors when the attributes value is not an object.
- nested `yield*` delegation inside bounded generator block bodies is supported for both sync and
  async generators, including async iterator-like objects whose `next()` returns a promise value and
  the final return value from iterator-like objects when delegation completes.
- bounded generator `next(arg)` calls accept arguments in this slice, with sent values forwarded
  into declaration initializers, plain binding assignments, and object and array property-assignment
  RHSs, plus currently suspended `yield*` delegated iterators; undeclared assignments remain
  unsupported in this slice; `return(value)` closes the iterator in this slice, and `throw(value)`
  remains an explicit runtime error in this slice, with an omitted argument treated as `undefined`.
- bounded `window.name` helpers for inline scripts:
  - `window.name` string inputs use browser-style string coercion, including runtime rejection of `Symbol` inputs
  - `windowName`
  - `setWindowName`
- `Prompt` returns the submitted text plus a boolean that is `false` when the prompt is canceled.
- `DebugView` is read-only and exposes inspection state such as `URL`, location parts
  (`LocationOrigin`, `LocationProtocol`, `LocationHost`, `LocationHostname`, `LocationPort`,
  `LocationPathname`, `LocationSearch`, `LocationHash`), `HTML`, `InitialHTML`, `DOMReady`,
  `DOMError`, `NowMs`, `DumpDOM`, `NodeCount`, `ScriptCount`, `ImageCount`, `FormCount`,
  `FocusedSelector`, `FocusedNodeID`, `TargetNodeID`, `HistoryLength`, `HistoryState`,
  `HistoryIndex`, `HistoryEntries`, `VisitedURLs`, `HistoryScrollRestoration`, `PendingTimers`,
  `PendingAnimationFrames`, `PendingMicrotasks`, `ScrollPosition`, `WindowName`, `Clipboard`,
  `MatchMediaRules`, `EventListeners`, `LocalStorage`, `SessionStorage`, `DocumentCookie`,
  `CookieJar`, `NavigationLog`, `Interactions`, and the configured `RandomSeed` when one was set on
  the builder.
- `DebugView.NavigatorOnLine()` exposes the effective `navigator.onLine` state and whether it was
  explicitly seeded on the builder.
- `DebugView.NavigatorLanguage()` exposes the seeded `navigator.language` locale read and whether
  it was explicitly configured through the navigator mock family.
- `DebugView` also exposes the configured failure seed readouts `OpenFailure`, `CloseFailure`,
  `PrintFailure`, and `ScrollFailure` when those builder fields are set.
- `DebugView.NodeCount()` exposes the current DOM node count as a read-only inspection integer after
  the DOM has been bootstrapped.
- `DebugView.ScriptCount()` exposes the current script element count as a read-only inspection
  integer after the DOM has been bootstrapped, `DebugView.ImageCount()` does the same for the
  current image element count, `DebugView.FormCount()` does the same for the current form element
  count, and `DebugView.SelectCount()` does the same for the current select element count.
- `DebugView.TemplateCount()` exposes the current template element count as a read-only inspection
  integer, `DebugView.TableCount()` does the same for the current table element count,
  `DebugView.ButtonCount()` does the same for the current button element count,
  `DebugView.TextAreaCount()` does the same for the current textarea element count,
  `DebugView.InputCount()` does the same for the current input element count,
  `DebugView.FieldsetCount()` does the same for the current fieldset element count,
  `DebugView.LegendCount()` does the same for the current legend element count,
  `DebugView.OutputCount()` does the same for the current output element count,
  `DebugView.LabelCount()` does the same for the current label element count,
  `DebugView.ProgressCount()` does the same for the current progress element count,
  `DebugView.MeterCount()` does the same for the current meter element count,
  `DebugView.AudioCount()` / `DebugView.VideoCount()` do the same for the current audio and video
  element counts, `DebugView.IframeCount()` does the same for the current iframe element count,
  `DebugView.EmbedCount()` does the same for the current embed element count, and
  `DebugView.TrackCount()` does the same for the current track element count.
- `DebugView.PictureCount()` exposes the current picture element count as a read-only inspection
  integer.
- `DebugView.SourceCount()` exposes the current source element count as a read-only inspection
  integer.
- `DebugView.DialogCount()` exposes the current dialog element count as a read-only inspection
  integer.
- `DebugView.DetailsCount()` exposes the current details element count as a read-only inspection
  integer.
- `DebugView.SummaryCount()` exposes the current summary element count as a read-only inspection
  integer.
- `DebugView.SectionCount()` exposes the current section element count as a read-only inspection
  integer.
- `DebugView.MainCount()` exposes the current main element count as a read-only inspection integer.
- `DebugView.ArticleCount()` exposes the current article element count as a read-only inspection
  integer.
- `DebugView.NavCount()` exposes the current nav element count as a read-only inspection integer.
- `DebugView.AsideCount()` exposes the current aside element count as a read-only inspection
  integer.
- `DebugView.FigureCount()` exposes the current figure element count as a read-only inspection
  integer.
- `DebugView.FigcaptionCount()` exposes the current figcaption element count as a read-only
  inspection integer.
- `DebugView.HeaderCount()` exposes the current header element count as a read-only inspection
  integer.
- `DebugView.FooterCount()` exposes the current footer element count as a read-only inspection
  integer.
- `DebugView.AddressCount()` exposes the current address element count as a read-only inspection
  integer.
- `DebugView.BlockquoteCount()` exposes the current blockquote element count as a read-only
  inspection integer.
- `DebugView.ParagraphCount()` exposes the current paragraph element count as a read-only inspection
  integer.
- `DebugView.PreCount()` exposes the current pre element count as a read-only inspection integer.
- `DebugView.MarkCount()` exposes the current mark element count as a read-only inspection integer.
- `DebugView.QCount()` exposes the current q element count as a read-only inspection integer.
- `DebugView.CiteCount()` exposes the current cite element count as a read-only inspection integer.
- `DebugView.AbbrCount()` exposes the current abbr element count as a read-only inspection integer.
- `DebugView.StrongCount()` exposes the current strong element count as a read-only inspection
  integer.
- `DebugView.SpanCount()` exposes the current span element count as a read-only inspection integer.
- `DebugView.DataCount()` exposes the current data element count as a read-only inspection integer.
- `DebugView.DfnCount()` exposes the current dfn element count as a read-only inspection integer.
- `DebugView.KbdCount()` exposes the current kbd element count as a read-only inspection integer.
- `DebugView.SampCount()` exposes the current samp element count as a read-only inspection integer.
- `DebugView.RubyCount()` exposes the current ruby element count as a read-only inspection integer.
- `DebugView.RtCount()` exposes the current rt element count as a read-only inspection integer.
- `DebugView.VarCount()` exposes the current var element count as a read-only inspection integer.
- `DebugView.CodeCount()` exposes the current code element count as a read-only inspection integer.
- `DebugView.SmallCount()` exposes the current small element count as a read-only inspection
  integer.
- `DebugView.TimeCount()` exposes the current time element count as a read-only inspection integer.
- `DebugView.OptionCount()` exposes the current option count as a read-only inspection integer, and
  `DebugView.SelectedOptionCount()` does the same for the selected option count.
- `DebugView.OptgroupCount()` exposes the current optgroup count as a read-only inspection integer.
- `DebugView.LinkCount()` exposes the current link count as a read-only inspection integer, and
  `DebugView.AnchorCount()` does the same for the current anchor count.
- `DebugView.OptionLabels()` exposes the current option labels as a read-only inspection slice.
- `DebugView.SelectedOptionLabels()` exposes the current selected option labels as a read-only
  inspection slice.
- `DebugView.OptionValues()` exposes the current option values as a read-only inspection slice.
- `DebugView.SelectedOptionValues()` exposes the current selected option values as a read-only
  inspection slice.
- `DebugView.OptgroupLabels()` exposes the current optgroup labels as a read-only inspection slice.
- `DebugView.HistoryEntries()` exposes the current history stack as a read-only inspection slice.
- `DebugView.HistoryIndex()` exposes the current history cursor as a read-only inspection integer.
- `DebugView.VisitedURLs()` exposes the current visited URL snapshot as a read-only inspection slice
  derived from the session history.
- `DebugView.DOMReady()` and `DOMError()` expose DOM initialization readiness and the latest DOM
  parse/runtime failure text as read-only inspection data.
- `DebugView.LastInlineScriptHTML()` exposes the most recently executed classic inline script
  outerHTML as read-only inspection data.
- `DebugView.InitialHTML()` exposes the original builder HTML input as read-only inspection data
  without bootstrapping the DOM.
- `DebugView.PendingTimers()` and `PendingAnimationFrames()` expose scheduled timer and
  animation-frame snapshots as read-only inspection slices.
- `DebugView.PendingMicrotasks()` exposes the current queued microtask sources as a read-only
  inspection slice.
- `DebugView.FetchCalls()` exposes the captured fetch call trace as a read-only inspection slice.
- `DebugView.CookieJar()` exposes the current cookie jar as a read-only inspection map.
- `DebugView.DialogAlerts()`, `DialogConfirmMessages()`, and `DialogPromptMessages()` expose
  captured dialog messages as read-only inspection slices.
- `DebugView.DownloadArtifacts()` and `FileInputSelections()` expose captured download and
  file-input traces as read-only inspection slices.
- `DebugView.StorageEvents()` exposes the captured storage change trace as a read-only inspection
  slice.
- `DebugView.OpenCalls()`, `CloseCalls()`, `PrintCalls()`, `ScrollCalls()`, and `MatchMediaCalls()`
  expose browser-action and matchMedia call traces as read-only inspection slices.
- `DebugView.MatchMediaListenerCalls()` exposes the captured matchMedia listener trace as a
  read-only inspection slice.
- `DebugView.ClipboardWrites()` exposes the captured clipboard write trace as a read-only inspection
  slice.
- `DebugView.EventListeners()` exposes the captured DOM event listener registrations as a read-only
  inspection slice.
- `DebugView.FetchResponseRules()` and `FetchErrorRules()` expose configured fetch response/error
  rules as read-only inspection slices.
- assertion helpers:
  - `AssertText`
  - `AssertValue`
  - `AssertChecked`
  - `AssertExists`
- tree-navigation helper:
  - `Contains`
  - `CompareDocumentPosition`
  - `IsConnected`
  - `HasChildNodes`
- attribute reflection helpers:
  - `GetAttribute`
  - `GetAttributeNames`
  - `GetAttributeNode`
  - `HasAttribute`
  - `HasAttributes`
  - `SetAttribute`
  - `ToggleAttribute`
  - `RemoveAttribute`
- form-control reflection helpers:
  - `type` on `button` / `input` / `select`
- live class/dataset views:
  - `ClassList` (`Values`, `Contains`, `Item`, `Add`, `Remove`)
  - `Dataset` (`Values`, `Get`, `Set`, `Remove`)
- tree mutation helpers:
  - `InnerHTML`
  - `TextContent`
  - `OuterHTML`
  - `SetInnerHTML`
  - `ReplaceChildren`
  - `Before`
  - `After`
  - `CloneNode`
  - `SetTextContent`
  - `SetOuterHTML`
  - `InsertAdjacentHTML`
  - `ReplaceWith`
  - `RemoveNode`
  - `WriteHTML`
- typed mock families for:
  - `Fetch`
  - `ExternalJS` (including seeded source loads for `<script src>` and external module `src`
    dependencies; globals defined by seeded classic scripts stay visible to later classic scripts
    in the same session)
  - `Dialogs`
  - `Clipboard`
  - `Navigator` (including seeded `navigator.language` reads)
  - `Location`
  - `Open`
  - `Close`
  - `Print`
  - `Scroll`
  - `MatchMedia` (including listener capture injection through the mock registry)
  - `Downloads`
  - `FileInput` (including seeded file contents for `input.files[0].text()` reads and empty-string clears via `value = ""`)
  - `Storage` (including change capture through `Events()`)

## Current Scope

- DOM bootstrap, selector/query, attribute reflection, live collections, and tree mutation are
  implemented in bounded slices; the detailed support map lives in
  `doc/capability-matrix.md`.
- Inline classic scripts and module scripts run through the bounded script slice, with host
  bindings for browser globals, timers, history/location, storage, clipboard, matchMedia, and
  other runtime helpers. External JS dependencies are mock-driven through the typed registry and
  do not reach the network.
- Event dispatch, focus, form controls, navigation/default actions, download capture, and typed
  mock families are wired through the public facade.
- `DebugView` exposes read-only snapshots for DOM, history, timers, microtasks, storage, cookies,
  matchMedia, listeners, and interaction traces.
- Remaining gaps are tracked in `TODO.md`.

## Docs

- `doc/README.md`
- `doc/subsystem-map.md`
- `doc/capability-matrix.md`
- `doc/implementation-guide.md`
- `doc/mock-guide.md`
- `doc/roadmap.md`

## Issues

- Start new issue drafts from [`doc/issue-template.md`](doc/issue-template.md).
- When you fill it out, keep the owning subsystem, test layer, reproduction steps, reproduction
  code, original failed command, and acceptance criteria explicit.
