package browsertester

import "testing"

func TestIssue168ObjectFromEntriesSupportsPageInitLookupTables(t *testing.T) {
	harness, err := FromHTML(`
		<pre id="out"></pre>
		<script>
		  const kanaPairs = [
		    ["full", "アイウ"],
		    ["half", "ｱｲｳ"]
		  ];
		  const normalized = Object.fromEntries(
		    kanaPairs.map(([key, value]) => [key, value.slice(0, 2)])
		  );
		  const aliases = Object.fromEntries(
		    new Map([
		      ["zenkaku", normalized.full],
		      ["hankaku", normalized.half]
		    ])
		  );

		  document.getElementById("out").textContent =
		    aliases.zenkaku + "|" + aliases.hankaku + "|" + Object.keys(aliases).join(",");
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if err := harness.AssertText("#out", "アイ|ｱｲ|zenkaku,hankaku"); err != nil {
		t.Fatalf("AssertText(#out, アイ|ｱｲ|zenkaku,hankaku) error = %v", err)
	}
}
