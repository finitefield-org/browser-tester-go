package runtime

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

func TestRunScriptSupportsBrowserStdlibSlice(t *testing.T) {
	session := NewSession(SessionConfig{})
	session.SetNowMs(1700000000000)
	wantLocaleDate := time.UnixMilli(1700000000000).UTC().Format("1/2/2006")

	result, err := session.runScriptOnStore(dom.NewStore(), `
		let date = new Date(Date.now());
		let assigned = Object.assign({ first: "a" }, { second: "b" });
		let parsed = JSON.parse("{\"a\":1,\"b\":[2,3]}");
		let items = [1, 2, 3];
		let pushed = [1, 2];
		pushed.push(3);
		let seen = "";
		[1, 2, 3].forEach(v => seen = seen + v);
		[
			Array.from("go").join("|"),
			Array.isArray(items) ? "true" : "false",
			Object.keys(assigned).join(","),
			Object.entries(assigned).map(entry => entry.join("=")).join(","),
			Object.values(assigned).join(","),
			new Intl.DateTimeFormat("en-US-u-nu-latn", { timeZone: "America/Chicago", year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false }).formatToParts(new Date(Date.UTC(2026, 0, 21, 8, 45, 0, 0))).find(part => part.type === "hour").value,
			Object.fromEntries([["left", "l"], ["right", "r"]]).left + "|" + Object.fromEntries(new Map([["first", 1], ["second", 2]])).second,
			Object.keys(date).length + "|" + Object.keys(Object.assign({}, date)).length,
			JSON.stringify(parsed),
			JSON.stringify(date),
			date.toISOString(),
			date.toLocaleDateString("en-US"),
			Math.abs(-4) + "|" + Math.min(3, 1, 2) + "|" + Math.max(3, 1, 2) + "|" + Math.floor(1.9) + "|" + Math.floor(-1.1),
			Number.isFinite(1) + "|" + Number.isFinite(Number.NaN),
			String(true) + "|" + Boolean(0).valueOf(),
			Number(15).toString(16) + "|" + (15).valueOf(),
			"  Go  ".trim().toLowerCase() + "|" + "go".replace("g", "n") + "|" + "ab".lastIndexOf("b") + "|" + "go".slice(1),
			["alpha", "beta", "gamma", "beta"].indexOf("beta") + "|" + ["alpha", "beta", "gamma", "beta"].indexOf("beta", 2) + "|" + ["alpha", "beta", "gamma", "beta"].indexOf("beta", -2) + "|" + ["alpha", "beta", "gamma", "beta"].lastIndexOf("beta") + "|" + ["alpha", "beta", "gamma", "beta"].lastIndexOf("beta", 2) + "|" + ["alpha", "beta", "gamma", "beta"].lastIndexOf("beta", -3),
			items.slice(1).join(",") + "|" + items.filter(v => v > 1).join(",") + "|" + items.map(v => v * 2).join(",") + "|" + items.some(v => v === 3) + "|" + items.includes(2) + "|" + pushed.join(",") + "|" + items.find(v => v > 1),
			seen
		].join("~");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}

	wantDate := time.UnixMilli(1700000000000).UTC().Format("2006-01-02T15:04:05.000Z")
	want := strings.Join([]string{
		"g|o",
		"true",
		"first,second",
		"first=a,second=b",
		"a,b",
		"02",
		"l|2",
		"0|0",
		`{"a":1,"b":[2,3]}`,
		strconv.Quote(wantDate),
		wantDate,
		wantLocaleDate,
		"4|1|3|1|-2",
		"true|false",
		"true|false",
		"f|15",
		"go|no|1|o",
		"1|3|3|3|1|1",
		"2,3|2,3|2,4,6|true|true|1,2,3|2",
		"123",
	}, "~")

	if result.String != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", result.String, want)
	}
}

func TestRunScriptSupportsIntlNumberFormatGrouping(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `new Intl.NumberFormat("en", { maximumFractionDigits: 0 }).format(1198.88)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "1,199"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlNumberFormatCurrencyStyle(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `new Intl.NumberFormat("ja-JP", { style: "currency", currency: "JPY", maximumFractionDigits: 0 }).format(1200)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "￥1,200"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlCollatorNumericAndSwedishSorting(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const values = ["item 10", "item 2", "item 1"];
		const collator = new Intl.Collator("en", {
			usage: "sort",
			numeric: true,
			sensitivity: "variant",
		});
		const asc = values.slice().sort(collator.compare).join(",");
		const desc = values.slice().sort((left, right) => collator.compare(right, left)).join(",");
		const zeroPadded = collator.compare("item 02", "item 2");
		const numeric = String(collator.resolvedOptions().numeric);
		const sv = new Intl.Collator("sv", { usage: "sort", sensitivity: "variant" });
		const swedish = ["Öga", "Zebra", "Äpple", "Ål"].slice().sort(sv.compare).join(",");
		asc + "|" + desc + "|" + zeroPadded + "|" + numeric + "|" + swedish
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "item 1,item 2,item 10|item 10,item 2,item 1|0|true|Zebra,Ål,Äpple,Öga"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsUint8ArrayConstruction(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Array.from(new Uint8Array([65, 66, 67])).join(",")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "65,66,67"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsPromiseResolve(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Promise.resolve(new Uint8Array([1, 2, 3]).buffer)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindPromise {
		t.Fatalf("runScriptOnStore() kind = %q, want promise", result.Kind)
	}
	if result.Promise == nil {
		t.Fatalf("runScriptOnStore() promise payload = nil, want object buffer")
	}
	if result.Promise.Kind != script.ValueKindObject {
		t.Fatalf("runScriptOnStore() promise payload kind = %q, want object", result.Promise.Kind)
	}
	if got, ok := objectProperty(*result.Promise, "byteLength"); !ok || got.Kind != script.ValueKindNumber || got.Number != 3 {
		t.Fatalf("runScriptOnStore() promise payload byteLength = %#v, want 3", got)
	}
}

func TestRunScriptRejectsUint8ArrayFromNonArrayLikeValue(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := session.runScriptOnStore(dom.NewStore(), `new Uint8Array({})`); err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want unsupported array-like input")
	} else if !strings.Contains(err.Error(), "array-like") {
		t.Fatalf("runScriptOnStore() error = %v, want array-like validation failure", err)
	}
}

func TestRunScriptSupportsArrayFromOnHostArrayLikeAttributeReferences(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const parsed = new DOMParser().parseFromString(
			'<svg xmlns="http://www.w3.org/2000/svg"><image href="https://example.com/p.png" width="20" height="20" /></svg>',
			"image/svg+xml"
		);
		const safeRoot = parsed.documentElement.cloneNode(true);
		const image = safeRoot.querySelector("image");
		const attrCount = image ? String(image.attributes.length) : "missing";
		const href = image ? String(image.getAttribute("href")) : "missing";
		let snapshot = [];
		if (image) {
			snapshot = Array.from(image.attributes);
		}
		const snapshotLength = String(snapshot.length);
		const attrs = image
			? snapshot
				.map((attr) => attr.name + "=" + attr.value)
				.sort()
				.join(",")
			: "missing";
		const firstAttr = snapshot[0] ? snapshot[0].name + "=" + snapshot[0].value : "missing";
		if (image) {
			image.removeAttribute("href");
		}
		const hrefAfterRemoval = image ? String(image.getAttribute("href")) : "missing";
		[
			String(!!image),
			attrCount,
			href,
			snapshotLength,
			firstAttr,
			attrs,
			hrefAfterRemoval
		].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "true|3|https://example.com/p.png|3|height=20|height=20,href=https://example.com/p.png,width=20|null"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptParsesDimensionSuffixesToMillimeters(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		function safeString(value) { return String(value == null ? "" : value); }
		function normalizeDigits(value) {
			return safeString(value)
				.replace(/[\uFF10-\uFF19]/g, (s) => String.fromCharCode(s.charCodeAt(0) - 65248))
				.replace(/[\uFF0E\u3002]/g, ".")
				.replace(/[\uFF0C\u3001]/g, ",")
				.replace(/[\uFF0B]/g, "+")
				.replace(/[\u30FC\uFF0D\u2015]/g, "-")
				.trim();
		}
		function parseFlexibleNumber(value) {
			const normalized = normalizeDigits(value).replace(/[\s_\u00A0]/g, "");
			if (!normalized) return null;
			const sign = normalized.startsWith("-") ? -1 : 1;
			const unsigned = normalized.replace(/^[+-]/, "");
			if (!unsigned) return null;
			const commaCount = (unsigned.match(/,/g) || []).length;
			const dotCount = (unsigned.match(/\./g) || []).length;
			let candidate = unsigned;
			if (commaCount > 0 && dotCount > 0) {
				const lastComma = unsigned.lastIndexOf(",");
				const lastDot = unsigned.lastIndexOf(".");
				const decimalIndex = Math.max(lastComma, lastDot);
				const intPart = unsigned.slice(0, decimalIndex).replace(/[.,]/g, "");
				const fracPart = unsigned.slice(decimalIndex + 1).replace(/[.,]/g, "");
				candidate = fracPart ? intPart + "." + fracPart : intPart;
			} else if (commaCount === 1) {
				const parts = unsigned.split(",");
				candidate = parts[1].length === 3 ? parts.join("") : parts[0] + "." + parts[1];
			} else if (dotCount === 1) {
				const parts = unsigned.split(".");
				candidate = parts[1].length === 3 ? parts.join("") : parts[0] + "." + parts[1];
			}
			const parsed = Number(candidate);
			if (!Number.isFinite(parsed)) return null;
			return parsed * sign;
		}
		function parseDimensionToMm(value, fallbackUnit) {
			const raw = normalizeDigits(value);
			if (!raw) return { mm: null, unit: fallbackUnit, error: null };
			const compact = raw.replace(/[\s_\u00A0]/g, "");
			const match = compact.match(/^([+-]?[0-9.,]+)(mm|cm|m|in|inch|ft|["'])?$/i);
			if (!match) return { mm: null, unit: fallbackUnit, error: "format" };
			const numeric = parseFlexibleNumber(match[1]);
			if (numeric == null) return { mm: null, unit: fallbackUnit, error: "format" };
			const unit = (match[2] || fallbackUnit).toLowerCase();
			const factor =
				unit === "in" || unit === "inch" ? 25.4 :
				unit === "cm" ? 10 :
				unit === "m" ? 1000 :
				unit === "ft" ? 304.8 :
				1;
			const mm = numeric * factor;
			if (!Number.isFinite(mm) || mm <= 0) return { mm: null, unit, error: "positive" };
			return { mm, unit, error: null };
		}
		const first = parseDimensionToMm("47.2in", "mm");
		const second = parseDimensionToMm("35.4in", "mm");
		[String(first.mm), first.unit, String(second.mm), second.unit].join("|");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "1198.88|in|899.16|in"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsStringFromCharCode(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `String.fromCharCode(0x41, 0x42, 0x43, 0x20, 0x30)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "ABC 0"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsBrowserObjectAssignSymbolsAndToFixed(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const symA = Symbol("token");
		const symB = Symbol("token");
		const assigned = Object.assign({ plain: "a" }, "go", null, undefined, { extra: "b" }, { [symA]: "symbol" });
		const symbols = Object.getOwnPropertySymbols(assigned);
		[
			symA === symB,
			assigned.plain,
			assigned[0],
			assigned[1],
			assigned.extra,
			symbols.length,
			symbols[0].toString(),
			assigned[symbols[0]],
			(1.2).toFixed(2)
		].join("|");
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "false|a|g|o|b|1|Symbol(token)|symbol|1.20"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsNullishObjectHelpers(t *testing.T) {
	t.Run("assign target", func(t *testing.T) {
		session := NewSession(DefaultSessionConfig())

		_, err := session.runScriptOnStore(dom.NewStore(), `Object.assign(null, { a: 1 })`)
		if err == nil {
			t.Fatalf("runScriptOnStore() error = nil, want Object.assign target failure")
		}
		if !strings.Contains(err.Error(), "Cannot convert undefined or null to object") {
			t.Fatalf("runScriptOnStore() error = %v, want nullish conversion failure", err)
		}
	})

	t.Run("getOwnPropertySymbols target", func(t *testing.T) {
		session := NewSession(DefaultSessionConfig())

		_, err := session.runScriptOnStore(dom.NewStore(), `Object.getOwnPropertySymbols(null)`)
		if err == nil {
			t.Fatalf("runScriptOnStore() error = nil, want Object.getOwnPropertySymbols failure")
		}
		if !strings.Contains(err.Error(), "Cannot convert undefined or null to object") {
			t.Fatalf("runScriptOnStore() error = %v, want nullish conversion failure", err)
		}
	})
}

func TestRunScriptUsesSeededMathRandom(t *testing.T) {
	sessionA := NewSession(SessionConfig{
		RandomSeed:    7,
		HasRandomSeed: true,
	})
	sessionB := NewSession(SessionConfig{
		RandomSeed:    7,
		HasRandomSeed: true,
	})
	source := `Math.random() + "|" + Math.random()`

	resultA, err := sessionA.runScriptOnStore(dom.NewStore(), source)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	resultB, err := sessionB.runScriptOnStore(dom.NewStore(), source)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}

	rng := rand.New(rand.NewSource(7))
	want := strconv.FormatFloat(rng.Float64(), 'f', -1, 64) + "|" + strconv.FormatFloat(rng.Float64(), 'f', -1, 64)

	if resultA.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", resultA.Kind)
	}
	if resultB.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", resultB.Kind)
	}
	if resultA.String != want {
		t.Fatalf("runScriptOnStore() seed 7 value = %q, want %q", resultA.String, want)
	}
	if resultB.String != want {
		t.Fatalf("runScriptOnStore() seed 7 repeat value = %q, want %q", resultB.String, want)
	}
}

func TestRunScriptSupportsCSSEscapeOnGlobalCSSObject(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `CSS.escape("0") + "|" + CSS.escape("alpha-beta")`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, `\30 |alpha-beta`; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsUnsupportedCSSReference(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `CSS.supports("color", "red")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want unsupported CSS reference error")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported || !strings.Contains(scriptErr.Message, "CSS.supports") {
		t.Fatalf("runScriptOnStore() error = %#v, want unsupported CSS reference error", err)
	}
}

func TestRunScriptRejectsInvalidJSONParse(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `JSON.parse("{")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want parse error")
	}
}

func TestRunScriptRejectsInvalidObjectEntriesArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.entries()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object entries error")
	}
	if !strings.Contains(err.Error(), "Object.entries expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want object entries message", err)
	}
}

func TestRunScriptSupportsObjectFromEntries(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const table = Object.fromEntries([
			["full", "アイ"],
			["half", "ｱｲ"]
		]);
		const aliasTable = Object.fromEntries(new Map([
			["zenkaku", table.full],
			["hankaku", table.half]
		]));
		aliasTable.zenkaku + "|" + aliasTable.hankaku + "|" + Object.keys(aliasTable).join(",")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "アイ|ｱｲ|zenkaku,hankaku"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsSetConstructorAndMethods(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		const seen = new Set(["alpha", "alpha", "beta"]);
		seen.add("gamma");
		seen.delete("alpha");
		[String(seen.size), String(seen.has("alpha")), String(seen.has("gamma"))].join("|")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "2|false|true"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptSupportsIntlDateTimeFormatTimeZoneAndFormatToParts(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `
		function zonedText(instantMs, zone) {
			const formatter = new Intl.DateTimeFormat("en-US-u-nu-latn", {
				timeZone: zone,
				year: "numeric",
				month: "2-digit",
				day: "2-digit",
				hour: "2-digit",
				minute: "2-digit",
				second: "2-digit",
				hour12: false,
			});
			const parts = formatter.formatToParts(new Date(instantMs));
			const get = (type) => parts.find((part) => part.type === type)?.value || "?";
			return get("year") + "-" + get("month") + "-" + get("day") + " " + get("hour") + ":" + get("minute") + ":" + get("second");
		}
		const arrivalInstant = Date.UTC(2026, 0, 21, 8, 45, 0, 0);
		zonedText(arrivalInstant, "America/Chicago") + "|" + zonedText(arrivalInstant, "America/New_York")
	`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "2026-01-21 02:45:00|2026-01-21 03:45:00"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsInvalidIntlDateTimeFormatTimeZone(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `new Intl.DateTimeFormat("en-US", { timeZone: "Mars/Phobos" }).formatToParts(new Date(0))`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want invalid timeZone failure")
	}
	if !strings.Contains(err.Error(), "timeZone") {
		t.Fatalf("runScriptOnStore() error = %v, want timeZone failure message", err)
	}
}

func TestRunScriptRejectsIntlCollatorNumericTypeMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `new Intl.Collator("en-US", { numeric: "true" })`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want collator numeric type failure")
	}
	if !strings.Contains(err.Error(), "Intl.Collator numeric must be a boolean") {
		t.Fatalf("runScriptOnStore() error = %v, want collator numeric failure message", err)
	}
}

func TestRunScriptRejectsSetCallWithoutNew(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Set()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Set call failure")
	}
	if !strings.Contains(err.Error(), "Set constructor must be called with `new`") {
		t.Fatalf("runScriptOnStore() error = %v, want Set constructor call message", err)
	}
}

func TestRunScriptSupportsDateUTC(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Date.UTC(2026, 0, 21, 8, 45, 0, 0)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindNumber {
		t.Fatalf("runScriptOnStore() kind = %q, want number", result.Kind)
	}
	want := time.Date(2026, time.January, 21, 8, 45, 0, 0, time.UTC).UnixMilli()
	if got := int64(result.Number); got != want {
		t.Fatalf("runScriptOnStore() value = %d, want %d", got, want)
	}
}

func TestRunScriptRejectsInvalidObjectFromEntriesPair(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.fromEntries([["full"]])`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object.fromEntries pair failure")
	}
	if !strings.Contains(err.Error(), "Object.fromEntries pair 0 must be a two-item array") {
		t.Fatalf("runScriptOnStore() error = %v, want object.fromEntries pair message", err)
	}
}

func TestRunScriptRejectsInvalidObjectFromEntriesArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.fromEntries()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object.fromEntries arity failure")
	}
	if !strings.Contains(err.Error(), "Object.fromEntries expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want object.fromEntries arity message", err)
	}
}

func TestRunScriptRejectsObjectAssignGetterOnlyTarget(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `
		Object.assign({
			get a() { return 1; }
		}, {
			a: 2
		})
	`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want getter-only target error")
	}
	if !strings.Contains(err.Error(), "getter-only property") {
		t.Fatalf("runScriptOnStore() error = %v, want getter-only target message", err)
	}
}

func TestRunScriptRejectsMathRoundArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.round()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.round arity failure")
	}
	if !strings.Contains(err.Error(), "Math.round expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.round arity message", err)
	}
}

func TestRunScriptSupportsMathFloor(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	result, err := session.runScriptOnStore(dom.NewStore(), `Math.floor(1.9) + "|" + Math.floor(-1.1)`)
	if err != nil {
		t.Fatalf("runScriptOnStore() error = %v", err)
	}
	if result.Kind != script.ValueKindString {
		t.Fatalf("runScriptOnStore() kind = %q, want string", result.Kind)
	}
	if got, want := result.String, "1|-2"; got != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", got, want)
	}
}

func TestRunScriptRejectsMathFloorArityMismatch(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Math.floor()`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want Math.floor arity failure")
	}
	if !strings.Contains(err.Error(), "Math.floor expects 1 argument") {
		t.Fatalf("runScriptOnStore() error = %v, want Math.floor arity message", err)
	}
}
