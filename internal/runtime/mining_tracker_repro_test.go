package runtime

import "testing"

func TestSessionAssertValueReadsDefaultSelectedOptionInHiddenSection(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><section class="hidden"><select id="mode"><option value="t" selected>t</option><option value="kg">kg</option></select></section></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.AssertValue("#mode", "t"); err != nil {
		t.Fatalf("AssertValue(#mode, t) error = %v", err)
	}
}

func TestSessionAssertValueReadsDefaultSelectedOptionAfterHiddenSectionToggle(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><button id="toggle" type="button">toggle</button><section id="panel" class="hidden"><select id="mode"><option value="t" selected>t</option><option value="kg">kg</option></select></section><script>document.getElementById("toggle").addEventListener("click", function () { document.getElementById("panel").classList.toggle("hidden"); });</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if err := session.Click("#toggle"); err != nil {
		t.Fatalf("Click(#toggle) error = %v", err)
	}

	if err := session.AssertValue("#mode", "t"); err != nil {
		t.Fatalf("AssertValue(#mode, t) after toggle error = %v", err)
	}
}

func TestSessionInlineScriptsCanRenderGroupedTotals(t *testing.T) {
	const rawHTML = `<main><table><tbody id="shift-body"></tbody></table><script>(() => {
  var sampleRecords = [
    {
      recordDate: "2026-03-26",
      shiftLabel: "Night",
      productionValue: 1250,
      productionUnit: "t",
      fuelValue: 420,
      fuelUnit: "L",
      electricityValue: 19800,
      electricityUnit: "kWh"
    },
    {
      recordDate: "2026-03-27",
      shiftLabel: "Day",
      productionValue: 1100,
      productionUnit: "t",
      fuelValue: 330,
      fuelUnit: "L",
      electricityValue: 16500,
      electricityUnit: "kWh"
    },
    {
      recordDate: "2026-04-02",
      shiftLabel: "Night",
      productionValue: 900,
      productionUnit: "t",
      fuelValue: 310,
      fuelUnit: "L",
      electricityValue: 14400,
      electricityUnit: "kWh"
    }
  ];

  function normalizeProduction(value, unit) {
    if (value === null || value === undefined) {
      return null;
    }
    if (unit === "kg") {
      return value / 1000;
    }
    if (unit === "st") {
      return value * 0.90718474;
    }
    if (unit === "lb") {
      return value * 0.00045359237;
    }
    return value;
  }

  function normalizeFuel(value, unit) {
    if (value === null || value === undefined) {
      return null;
    }
    if (unit === "US gal") {
      return value * 3.785411784;
    }
    if (unit === "Imp gal") {
      return value * 4.54609;
    }
    return value;
  }

  function normalizeElectricity(value, unit) {
    if (value === null || value === undefined) {
      return null;
    }
    if (unit === "MWh") {
      return value * 1000;
    }
    return value;
  }

  function groupRecords(records, keyFn, labelFn) {
    var map = {};
    records.forEach(function (record) {
      var key = keyFn(record);
      if (!map[key]) {
        map[key] = {
          key: key,
          label: labelFn(key),
          records: [],
          productionT: 0,
          fuelL: 0,
          electricityKWh: 0
        };
      }
      var group = map[key];
      group.records.push(record);
      var productionT = normalizeProduction(record.productionValue, record.productionUnit);
      if (productionT !== null) {
        group.productionT += productionT;
      }
      var fuelL = normalizeFuel(record.fuelValue, record.fuelUnit);
      if (fuelL !== null) {
        group.fuelL += fuelL;
      }
      var electricityKWh = normalizeElectricity(record.electricityValue, record.electricityUnit);
      if (electricityKWh !== null) {
        group.electricityKWh += electricityKWh;
      }
    });

    return Object.keys(map).sort().map(function (key) {
      var group = map[key];
      group.fuelIntensity = group.productionT > 0 ? group.fuelL / group.productionT : null;
      group.electricityIntensity = group.productionT > 0 ? group.electricityKWh / group.productionT : null;
      return group;
    });
  }

  var directFirst = normalizeProduction(sampleRecords[0].productionValue, sampleRecords[0].productionUnit);
  var directSecondGroup = normalizeProduction(sampleRecords[2].productionValue, sampleRecords[2].productionUnit);
  var rows = groupRecords(sampleRecords, function (record) {
    return record.shiftLabel || "Unlabeled shift";
  }, function (key) {
    return key || "Unlabeled shift";
  });

  document.getElementById("shift-body").innerHTML = [
    "direct=" + String(directFirst) + "," + String(directSecondGroup),
    rows.map(function (group) {
    return '<tr><td>' + [
      group.label,
      String(group.records.length) + "件",
      String(group.productionT) + " t",
      String(group.fuelIntensity),
      String(group.electricityIntensity)
    ].join(" | ") + '</td></tr>';
    }).join("")
  ].join("\n");
})();</script></main>`

	session := NewSession(SessionConfig{HTML: rawHTML})
	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#shift-body"); err != nil {
		t.Fatalf("TextContent(#shift-body) error = %v", err)
	} else if want := "direct=1250,900\nDay | 1件 | 1100 t | 0.3 | 15Night | 2件 | 2150 t | 0.3395348837209302 | 15.906976744186046"; got != want {
		t.Fatalf("TextContent(#shift-body) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanAccumulateDynamicObjectMapValues(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="out"></div><script>(() => {
  const map = {};
  [
    { key: "Night", productionT: 1250, fuelL: 420, electricityKWh: 19800 },
    { key: "Day", productionT: 1100, fuelL: 330, electricityKWh: 16500 },
    { key: "Night", productionT: 900, fuelL: 310, electricityKWh: 14400 }
  ].forEach((record) => {
    if (!map[record.key]) {
      map[record.key] = {
        key: record.key,
        records: [],
        productionT: 0,
        fuelL: 0,
        electricityKWh: 0
      };
    }
    const group = map[record.key];
    group.records.push(record);
    group.productionT += record.productionT;
    group.fuelL += record.fuelL;
    group.electricityKWh += record.electricityKWh;
  });

  document.getElementById("out").textContent = Object.keys(map).sort().map((key) => {
    const group = map[key];
    return [group.key, String(group.records.length), String(group.productionT), String(group.fuelL), String(group.electricityKWh)].join(":");
  }).join("|");
})();</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if want := "Day:1:1100:330:16500|Night:2:2150:730:34200"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanIncrementDynamicObjectMapProperty(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="out"></div><script>(() => {
  const map = {};
  ["Night", "Day", "Night"].forEach((key) => {
    if (!map[key]) {
      map[key] = { count: 0 };
    }
    map[key].count += 1;
  });
  document.getElementById("out").textContent = Object.keys(map).sort().map((key) => key + ":" + map[key].count).join("|");
})();</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if want := "Day:1|Night:2"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanMixArrayPushAndNumericAccumulationOnMapValues(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="out"></div><script>(() => {
  const map = {};
  ["Night", "Day", "Night"].forEach((key) => {
    if (!map[key]) {
      map[key] = { records: [], count: 0 };
    }
    const group = map[key];
    group.records.push(key);
    group.count += 1;
  });
  document.getElementById("out").textContent = Object.keys(map).sort().map((key) => key + ":" + map[key].records.length + ":" + map[key].count).join("|");
})();</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if want := "Day:1:1|Night:2:2"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanAccumulateBeforePushingIntoArrayOnMapValues(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="out"></div><script>(() => {
  const map = {};
  ["Night", "Day", "Night"].forEach((key) => {
    if (!map[key]) {
      map[key] = { records: [], count: 0 };
    }
    const group = map[key];
    group.count += 1;
    group.records.push(key);
  });
  document.getElementById("out").textContent = Object.keys(map).sort().map((key) => key + ":" + map[key].records.length + ":" + map[key].count).join("|");
})();</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if want := "Day:1:1|Night:2:2"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}

func TestSessionInlineScriptsCanMixArrayPushAndNumericAccumulationOnPlainObject(t *testing.T) {
	session := NewSession(SessionConfig{
		HTML: `<main><div id="out"></div><script>(() => {
  const group = { records: [], count: 0 };
  group.records.push("Night");
  group.count += 1;
  document.getElementById("out").textContent = [group.records.length, group.count].join("|");
})();</script></main>`,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if want := "1|1"; got != want {
		t.Fatalf("TextContent(#out) = %q, want %q", got, want)
	}
}
