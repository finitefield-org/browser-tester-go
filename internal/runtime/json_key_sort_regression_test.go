package runtime

import (
	"fmt"
	"testing"
)

func runJSONKeySortClick(t *testing.T, scope, order, want string) {
	t.Helper()

	rawHTML := fmt.Sprintf(`<main>
<button id="run" type="button">Run</button>
<pre id="out"></pre>
<script>
const compareKeys = (a, b) => (a < b ? -1 : a > b ? 1 : 0);
function sortJsonValue(value, scope, order, isRoot) {
  if (Array.isArray(value)) {
    if (scope === "all") {
      return value.map((item) => sortJsonValue(item, scope, order, false));
    }
    return value;
  }

  if (value && typeof value === "object") {
    const shouldSort = scope === "all" || (scope === "top" && isRoot);
    if (!shouldSort) return value;

    const keys = Object.keys(value).sort(compareKeys);
    if (order === "desc") keys.reverse();

    const out = {};
    keys.forEach((key) => {
      out[key] = sortJsonValue(value[key], scope, order, false);
    });
    return out;
  }

  return value;
}

document.getElementById("run").addEventListener("click", () => {
  try {
    const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}");
    const sorted = sortJsonValue(parsed, %q, %q, true);
    document.getElementById("out").textContent = JSON.stringify(sorted);
  } catch (error) {
    document.getElementById("out").textContent = "ERR:" + (error && error.message ? error.message : String(error));
  }
});
</script></main>`, scope, order)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func runJSONKeySortClickWithEarlyRootGuard(t *testing.T, scope, order, want string) {
	t.Helper()

	rawHTML := fmt.Sprintf(`<main>
<div id="json-key-sort-tool-root">
  <label><input id="json-key-sort-scope-all" type="radio" name="json-key-sort-scope" value="all" checked>All</label>
  <label><input id="json-key-sort-scope-top" type="radio" name="json-key-sort-scope" value="top">Top</label>
  <label><input id="json-key-sort-order-asc" type="radio" name="json-key-sort-order" value="asc" checked>Asc</label>
  <label><input id="json-key-sort-order-desc" type="radio" name="json-key-sort-order" value="desc">Desc</label>
  <textarea id="json-key-sort-output"></textarea>
  <div id="json-key-sort-status"><span></span></div>
  <select id="json-key-sort-indent">
    <option value="2" selected>2</option>
    <option value="4">4</option>
    <option value="tab">tab</option>
    <option value="minify">minify</option>
  </select>
  <label><input id="json-key-sort-trailing-newline" type="checkbox" checked>Trailing newline</label>
  <button id="json-key-sort-run-button" type="button">Run</button>
  <textarea id="json-key-sort-input"></textarea>
</div>
<script>
(() => {
  const root = document.getElementById("json-key-sort-tool-root");
  if (!root) return;

  const el = {
    input: root.querySelector("#json-key-sort-input"),
    output: root.querySelector("#json-key-sort-output"),
    runButton: root.querySelector("#json-key-sort-run-button"),
    scopeRadios: Array.from(root.querySelectorAll("input[name='json-key-sort-scope']")),
    orderRadios: Array.from(root.querySelectorAll("input[name='json-key-sort-order']")),
    status: root.querySelector("#json-key-sort-status"),
    statusText: root.querySelector("#json-key-sort-status span"),
    indentSelect: root.querySelector("#json-key-sort-indent"),
    trailingNewline: root.querySelector("#json-key-sort-trailing-newline"),
    sampleButton: null,
    clearButton: null,
    copyButton: null,
    downloadButton: null,
    swapButton: null,
    settingsToggle: null,
    settingsPanel: null,
  };

  function setRadioValue(radios, value, fallback) {
    const next = radios.some((radio) => radio.value === value) ? value : fallback;
    radios.forEach((radio) => {
      radio.checked = radio.value === next;
    });
  }

  function getRadioValue(radios, fallback) {
    const node = radios.find((radio) => radio.checked);
    return node ? node.value : fallback;
  }

  function setStatus(type, text) {
    if (!el.status || !el.statusText) return;
    el.statusText.textContent = text || "";
    el.status.dataset.type = type || "";
  }

  function updateOutputActionState() {}

  function initDefaults() {
    setRadioValue(el.scopeRadios, %q, "all");
    setRadioValue(el.orderRadios, %q, "asc");
    if (el.indentSelect) el.indentSelect.value = "2";
    if (el.trailingNewline) el.trailingNewline.checked = true;
    setStatus("idle", "Idle");
    updateOutputActionState();
  }

  const compareKeys = (a, b) => (a < b ? -1 : a > b ? 1 : 0);

  function sortJsonValue(value, scope, order, isRoot) {
    if (Array.isArray(value)) {
      if (scope === "all") {
        return value.map((item) => sortJsonValue(item, scope, order, false));
      }
      return value;
    }

    if (value && typeof value === "object") {
      const shouldSort = scope === "all" || (scope === "top" && isRoot);
      if (!shouldSort) return value;

      const keys = Object.keys(value).sort(compareKeys);
      if (order === "desc") keys.reverse();

      const out = {};
      keys.forEach((key) => {
        out[key] = sortJsonValue(value[key], scope, order, false);
      });
      return out;
    }

    return value;
  }

  function runSort() {
    try {
      const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}");
      const scope = getRadioValue(el.scopeRadios, %q);
      const order = getRadioValue(el.orderRadios, %q);
      const sorted = sortJsonValue(parsed, scope, order, true);
      if (el.output) el.output.textContent = JSON.stringify(sorted);
    } catch (error) {
      if (el.output) el.output.textContent = "ERR:" + (error && error.message ? error.message : String(error));
    }
  }

  setRadioValue(el.scopeRadios, %q, "all");
  setRadioValue(el.orderRadios, %q, "asc");
  initDefaults();
  if (el.runButton) el.runButton.addEventListener("click", runSort);
})();
</script></main>`, scope, order, scope, order, scope, order)

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#json-key-sort-run-button"); err != nil {
		t.Fatalf("Click(#json-key-sort-run-button) error = %v", err)
	}

	if got, err := session.TextContent("#json-key-sort-output"); err != nil {
		t.Fatalf("TextContent(#json-key-sort-output) error = %v", err)
	} else if got != want {
		t.Fatalf("TextContent(#json-key-sort-output) = %q, want %q", got, want)
	}
}

func TestSessionClickRunsRecursiveJsonKeySortAllLevels(t *testing.T) {
	runJSONKeySortClick(t, "all", "asc", `{"a":{"c":3,"d":4},"arr":[{"x":1,"y":2},3],"b":1}`)
}

func TestSessionClickRunsRecursiveJsonKeySortDescending(t *testing.T) {
	runJSONKeySortClick(t, "all", "desc", `{"b":1,"arr":[{"y":2,"x":1},3],"a":{"d":4,"c":3}}`)
}

func TestSessionClickRunsRecursiveJsonKeySortAllLevelsWithEarlyRootGuard(t *testing.T) {
	runJSONKeySortClickWithEarlyRootGuard(t, "all", "asc", `{"a":{"c":3,"d":4},"arr":[{"x":1,"y":2},3],"b":1}`)
}

func TestSessionClickRunsRecursiveJsonKeySortDescendingWithEarlyRootGuard(t *testing.T) {
	runJSONKeySortClickWithEarlyRootGuard(t, "all", "desc", `{"b":1,"arr":[{"y":2,"x":1},3],"a":{"d":4,"c":3}}`)
}

func TestSessionClickRunsRecursiveJsonKeySortWithIdentityCallback(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main>
<button id="run" type="button">Run</button>
<pre id="out"></pre>
<script>
const compareKeys = (a, b) => (a < b ? -1 : a > b ? 1 : 0);
function identity(value) {
  return value;
}
function sortJsonValue(value, scope, order, isRoot) {
  if (Array.isArray(value)) {
    if (scope === "all") {
      return value.map((item) => sortJsonValue(item, scope, order, false));
    }
    return value;
  }

  if (value && typeof value === "object") {
    const shouldSort = scope === "all" || (scope === "top" && isRoot);
    if (!shouldSort) return value;

    const keys = Object.keys(value).sort(compareKeys);
    if (order === "desc") keys.reverse();

    const out = {};
    keys.forEach((key) => {
      out[key] = identity(sortJsonValue(value[key], scope, order, false));
    });
    return out;
  }

  return value;
}

document.getElementById("run").addEventListener("click", () => {
  try {
    const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}");
    const sorted = sortJsonValue(parsed, "all", "asc", true);
    document.getElementById("out").textContent = JSON.stringify(sorted);
  } catch (error) {
    document.getElementById("out").textContent = "ERR:" + (error && error.message ? error.message : String(error));
  }
});
</script></main>`})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"a":{"c":3,"d":4},"arr":[{"x":1,"y":2},3],"b":1}` {
		t.Fatalf("TextContent(#out) = %q, want sorted JSON", got)
	}
}

func TestSessionClickRunsRecursiveJsonKeySortWithSeparateIdentityCallbackCall(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main>
<button id="run" type="button">Run</button>
<pre id="out"></pre>
<script>
const compareKeys = (a, b) => (a < b ? -1 : a > b ? 1 : 0);
function identity(value) {
  return value;
}
function sortJsonValue(value, scope, order, isRoot) {
  if (Array.isArray(value)) {
    if (scope === "all") {
      return value.map((item) => sortJsonValue(item, scope, order, false));
    }
    return value;
  }

  if (value && typeof value === "object") {
    const shouldSort = scope === "all" || (scope === "top" && isRoot);
    if (!shouldSort) return value;

    const keys = Object.keys(value).sort(compareKeys);
    if (order === "desc") keys.reverse();

    const out = {};
    keys.forEach((key) => {
      const child = value[key];
      const echoed = identity(child);
      out[key] = sortJsonValue(echoed, scope, order, false);
    });
    return out;
  }

  return value;
}

document.getElementById("run").addEventListener("click", () => {
  try {
    const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}");
    const sorted = sortJsonValue(parsed, "all", "asc", true);
    document.getElementById("out").textContent = JSON.stringify(sorted);
  } catch (error) {
    document.getElementById("out").textContent = "ERR:" + (error && error.message ? error.message : String(error));
  }
});
</script></main>`})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != `{"a":{"c":3,"d":4},"arr":[{"x":1,"y":2},3],"b":1}` {
		t.Fatalf("TextContent(#out) = %q, want sorted JSON", got)
	}
}

func TestSessionClickRunsTopLevelJsonKeySortWithRadioSelection(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main>
<label><input id="scope-all" type="radio" name="json-key-sort-scope" value="all" checked>All</label>
<label><input id="scope-top" type="radio" name="json-key-sort-scope" value="top">Top</label>
<button id="run" type="button">Run</button>
<pre id="scope"></pre>
<pre id="out"></pre>
<script>
const compareKeys = (a, b) => (a < b ? -1 : a > b ? 1 : 0);
function getRadioValue(radios, fallback) {
  const node = radios.find((radio) => radio.checked);
  return node ? node.value : fallback;
}
function sortJsonValue(value, scope, order, isRoot) {
  if (Array.isArray(value)) {
    if (scope === "all") {
      return value.map((item) => sortJsonValue(item, scope, order, false));
    }
    return value;
  }

  if (value && typeof value === "object") {
    const shouldSort = scope === "all" || (scope === "top" && isRoot);
    if (!shouldSort) return value;

    const keys = Object.keys(value).sort(compareKeys);
    if (order === "desc") keys.reverse();

    const out = {};
    keys.forEach((key) => {
      out[key] = sortJsonValue(value[key], scope, order, false);
    });
    return out;
  }

  return value;
}

const el = {
  out: document.getElementById("out"),
  run: document.getElementById("run"),
  scope: document.getElementById("scope"),
  scopeRadios: Array.from(document.querySelectorAll('input[name="json-key-sort-scope"]')),
};

el.run.addEventListener("click", () => {
  try {
    const parsed = JSON.parse("{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}");
    const scope = getRadioValue(el.scopeRadios, "all");
    el.scope.textContent = scope;
    const sorted = sortJsonValue(parsed, scope, "asc", true);
    el.out.textContent = JSON.stringify(sorted, null, 2);
  } catch (error) {
    el.out.textContent = "ERR:" + (error && error.message ? error.message : String(error));
  }
});
</script></main>`})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.SetChecked("#scope-top", true); err != nil {
		t.Fatalf("SetChecked(#scope-top) error = %v", err)
	}
	if err := session.AssertChecked("#scope-top", true); err != nil {
		t.Fatalf("AssertChecked(#scope-top, true) error = %v", err)
	}
	if err := session.AssertChecked("#scope-all", false); err != nil {
		t.Fatalf("AssertChecked(#scope-all, false) error = %v", err)
	}
	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}

	if got, err := session.TextContent("#scope"); err != nil {
		t.Fatalf("TextContent(#scope) error = %v", err)
	} else if got != "top" {
		t.Fatalf("TextContent(#scope) = %q, want top", got)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if want := "{\n  \"a\": {\n    \"d\": 4,\n    \"c\": 3\n  },\n  \"arr\": [\n    {\n      \"y\": 2,\n      \"x\": 1\n    },\n    3\n  ],\n  \"b\": 1\n}"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func TestSessionClickRunsRecursiveJsonKeySortWithOutputActionState(t *testing.T) {
	session := NewSession(SessionConfig{HTML: `<main>
<div id="root">
  <button id="run" type="button">Run</button>
  <textarea id="input">{"b":1,"a":{"d":4,"c":3},"arr":[{"y":2,"x":1},3]}</textarea>
  <textarea id="output"></textarea>
  <label><input id="all" type="radio" name="scope" value="all" checked></label>
  <label><input id="top" type="radio" name="scope" value="top"></label>
  <label><input id="asc" type="radio" name="order" value="asc" checked></label>
  <label><input id="desc" type="radio" name="order" value="desc"></label>
  <label><input id="trailing" type="checkbox"></label>
  <div id="status"><span></span></div>
</div>
<script>
(() => {
  const root = document.getElementById("root");
  const el = {
    input: root.querySelector("#input"),
    output: root.querySelector("#output"),
    runButton: root.querySelector("#run"),
    scopeRadios: Array.from(root.querySelectorAll("input[name='scope']")),
    orderRadios: Array.from(root.querySelectorAll("input[name='order']")),
    trailingNewline: root.querySelector("#trailing"),
    status: root.querySelector("#status"),
    statusText: root.querySelector("#status span"),
  };

  function setRadioValue(radios, value, fallback) {
    const next = radios.some((radio) => radio.value === value) ? value : fallback;
    radios.forEach((radio) => {
      radio.checked = radio.value === next;
    });
  }

  function getRadioValue(radios, fallback) {
    const node = radios.find((radio) => radio.checked);
    return node ? node.value : fallback;
  }

  function setStatus(type, text) {
    if (!el.status || !el.statusText) return;
    el.statusText.textContent = text || "";
    el.status.dataset.type = type || "";
  }

  function compareKeys(a, b) {
    if (a === b) return 0;
    return a < b ? -1 : 1;
  }

  function sortJsonValue(value, scope, order, isRoot) {
    if (Array.isArray(value)) {
      if (scope === "all") {
        return value.map((item) => sortJsonValue(item, scope, order, false));
      }
      return value;
    }

    if (value && typeof value === "object") {
      const shouldSort = scope === "all" || (scope === "top" && isRoot);
      if (!shouldSort) return value;

      const keys = Object.keys(value).sort(compareKeys);
      if (order === "desc") keys.reverse();

      const out = {};
      keys.forEach((key) => {
        const child = value[key];
        out[key] = scope === "all" ? sortJsonValue(child, scope, order, false) : child;
      });
      return out;
    }

    return value;
  }

  function runSort() {
    const inputText = "{\"b\":1,\"a\":{\"d\":4,\"c\":3},\"arr\":[{\"y\":2,\"x\":1},3]}";
    const scope = getRadioValue(el.scopeRadios, "all");
    const order = getRadioValue(el.orderRadios, "asc");
    const parsed = JSON.parse(inputText);
    const sorted = sortJsonValue(parsed, scope, order, true);
    if (el.output) el.output.value = JSON.stringify(sorted, null, 2);
  }

  setRadioValue(el.scopeRadios, "all", "all");
  setRadioValue(el.orderRadios, "asc", "asc");
  setStatus("idle", "Idle");

  if (el.runButton) el.runButton.addEventListener("click", runSort);
})();
</script></main>`})

  if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.Click("#run"); err != nil {
		t.Fatalf("Click(#run) error = %v", err)
	}
	if got, err := session.TextContent("#output"); err != nil {
		t.Fatalf("TextContent(#output) error = %v", err)
	} else if got != "{\n  \"a\": {\n    \"c\": 3,\n    \"d\": 4\n  },\n  \"arr\": [\n    {\n      \"x\": 1,\n      \"y\": 2\n    },\n    3\n  ],\n  \"b\": 1\n}" {
		t.Fatalf("TextContent(#output) = %q, want sorted JSON", got)
	}
}
