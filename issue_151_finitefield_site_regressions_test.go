package browsertester

import "testing"

func TestIssue151MapDeleteWithExtraArgumentIsNotDispatchedAsFormData(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const pickMap = new Map();
		  pickMap.set('sku-1', 12);
		  pickMap.set('sku-2', 5);
		  const deleted = pickMap.delete('sku-1', 'extra');
		  const missing = pickMap.delete('missing', 'extra');
		  document.getElementById('out').textContent = [
		    String(deleted),
		    String(missing),
		    String(pickMap.size),
		    String(pickMap.get('sku-2')),
		  ].join('|');
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false|1|5" {
		t.Fatalf("TextContent(#out) = %q, want true|false|1|5", got)
	}
}

func TestIssue151MapHasWithExtraArgumentUsesMapSemantics(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const pickMap = new Map();
		  pickMap.set('sku-1', 12);
		  const hasSku = pickMap.has('sku-1', 'extra');
		  const hasMissing = pickMap.has('missing', 'extra');
		  document.getElementById('out').textContent = [
		    String(hasSku),
		    String(hasMissing),
		  ].join('|');
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "true|false" {
		t.Fatalf("TextContent(#out) = %q, want true|false", got)
	}
}

func TestIssue151PickMapGetOrFallbackDoesNotOverwriteMapBinding(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const pickMap = new Map();
		  pickMap.set('W1', { fixed: 0, unit: 2 });
		  const getUnitCost = (warehouse) => {
		    const pick = pickMap.get(warehouse) || { fixed: 0, unit: 0 };
		    return pick.unit;
		  };
		  const costMissing = getUnitCost('W9');
		  const costExisting = getUnitCost('W1');
		  const mapValue = pickMap.get('W1');
		  document.getElementById('out').textContent = [
		    String(costMissing),
		    String(costExisting),
		    mapValue ? String(mapValue.unit) : 'none',
		  ].join('|');
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "0|2|2" {
		t.Fatalf("TextContent(#out) = %q, want 0|2|2", got)
	}
}

func TestIssue151PickMapGetOrObjectLiteralFallbackKeepsMap(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const pickMap = new Map();
		  pickMap.set('W1', { fixed: 0, unit: 2 });
		  const fallback = pickMap.get('W9') || { fixed: 0, unit: 0 };
		  const mapValue = pickMap.get('W1');
		  document.getElementById('out').textContent = [
		    String(fallback.unit),
		    mapValue ? String(mapValue.unit) : 'none',
		  ].join('|');
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "0|2" {
		t.Fatalf("TextContent(#out) = %q, want 0|2", got)
	}
}

func TestIssue151NestedConstShadowNamedPickDoesNotOverwriteOuterPick(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const pick = new Map();
		  pick.set('W1', { unit: 2 });
		  const run = () => {
		    const inner = () => {
		      const pick = { fixed: 0, unit: 0 };
		      return pick.unit;
		    };
		    inner();
		    const outer = pick.get('W1');
		    return outer ? String(outer.unit) : 'none';
		  };
		  document.getElementById('out').textContent = run();
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "2" {
		t.Fatalf("TextContent(#out) = %q, want 2", got)
	}
}

func TestIssue151PlainObjectGetMethodIsNotHijackedAsFormData(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const store = {
		    get(name) {
		      return 'value:' + name;
		    }
		  };
		  document.getElementById('out').textContent = store.get('W1');
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "value:W1" {
		t.Fatalf("TextContent(#out) = %q, want value:W1", got)
	}
}

func TestIssue151MapGetPropertyAccessIsNotTreatedAsFormDataCall(t *testing.T) {
	harness, err := FromHTML(`
		<p id='out'></p>
		<script>
		  const pickMap = new Map();
		  pickMap.set('W1', { unit: 2 });
		  const getter = pickMap.get;
		  document.getElementById('out').textContent = String(typeof getter);
		</script>
	`)
	if err != nil {
		t.Fatalf("FromHTML() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "function" {
		t.Fatalf("TextContent(#out) = %q, want function", got)
	}
}
