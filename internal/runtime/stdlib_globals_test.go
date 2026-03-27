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
			Object.keys(date).length + "|" + Object.keys(Object.assign({}, date)).length,
			JSON.stringify(parsed),
			JSON.stringify(date),
			date.toISOString(),
			Math.abs(-4) + "|" + Math.min(3, 1, 2) + "|" + Math.max(3, 1, 2),
			Number.isFinite(1) + "|" + Number.isFinite(Number.NaN),
			String(true) + "|" + Boolean(0).valueOf(),
			Number(15).toString(16) + "|" + (15).valueOf(),
			"  Go  ".trim().toLowerCase() + "|" + "go".replace("g", "n") + "|" + "ab".lastIndexOf("b") + "|" + "go".slice(1),
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
		"0|0",
		`{"a":1,"b":[2,3]}`,
		strconv.Quote(wantDate),
		wantDate,
		"4|1|3",
		"true|false",
		"true|false",
		"f|15",
		"go|no|1|o",
		"2,3|2,3|2,4,6|true|true|1,2,3|2",
		"123",
	}, "~")

	if result.String != want {
		t.Fatalf("runScriptOnStore() value = %q, want %q", result.String, want)
	}
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

func TestRunScriptRejectsInvalidJSONParse(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `JSON.parse("{")`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want parse error")
	}
}

func TestRunScriptRejectsInvalidObjectEntriesArity(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := session.runScriptOnStore(dom.NewStore(), `Object.entries(1)`)
	if err == nil {
		t.Fatalf("runScriptOnStore() error = nil, want object entries error")
	}
	if !strings.Contains(err.Error(), "Object.entries expects an object") {
		t.Fatalf("runScriptOnStore() error = %v, want object entries message", err)
	}
}
