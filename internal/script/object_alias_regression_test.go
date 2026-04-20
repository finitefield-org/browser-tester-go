package script

import "testing"

func TestDispatchSupportsArrayPushThenPropertyIncrementOnObjectMapValueInCallback(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const map = {};
["Night", "Day", "Night"].forEach((key) => {
  if (!map[key]) {
    map[key] = { records: [], count: 0 };
  }
  const group = map[key];
  group.records.push(key);
  group.count += 1;
});
["Day", "Night"].map((key) => key + ":" + map[key].records.length + ":" + map[key].count).join("|")`,
	})
	if err != nil {
		t.Fatalf("Dispatch(object map callback mutation alias) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(object map callback mutation alias) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "Day:1:1|Night:2:2" {
		t.Fatalf("Dispatch(object map callback mutation alias) value = %q, want Day:1:1|Night:2:2", result.Value.String)
	}
}

func TestDispatchKeepsObjectMapAliasAlignedAfterArrayPushInCallback(t *testing.T) {
	runtime := NewRuntime(nil)

	result, err := runtime.Dispatch(DispatchRequest{
		Source: `const map = {};
let snapshot = "";
["Night"].forEach((key) => {
  if (!map[key]) {
    map[key] = { records: [], count: 0 };
  }
  const group = map[key];
  group.records.push(key);
  snapshot = [
    group.records.length,
    map[key].records.length,
    group.count,
    map[key].count,
    group === map[key]
  ].join("|");
});
snapshot`,
	})
	if err != nil {
		t.Fatalf("Dispatch(object map alias snapshot) error = %v", err)
	}
	if result.Value.Kind != ValueKindString {
		t.Fatalf("Dispatch(object map alias snapshot) kind = %q, want %q", result.Value.Kind, ValueKindString)
	}
	if result.Value.String != "1|1|0|0|true" {
		t.Fatalf("Dispatch(object map alias snapshot) value = %q, want 1|1|0|0|true", result.Value.String)
	}
}
