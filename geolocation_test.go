package browsertester

import (
	"testing"
	"time"
)

func TestHarnessGeolocationEmitPosition(t *testing.T) {
	const rawHTML = `<main><button id="start">start</button><div id="out">idle</div><script>document.getElementById("start").addEventListener("click", () => { navigator.geolocation.watchPosition((position) => { document.getElementById("out").textContent = [position.coords.latitude, position.coords.longitude, new Date(position.timestamp).toISOString()].join("|"); }); });</script></main>`

	harness, err := FromHTMLWithURL("https://example.test/tools/forestry/area-boundary-calculator/", rawHTML)
	if err != nil {
		t.Fatalf("FromHTMLWithURL() error = %v", err)
	}

	if err := harness.Click("#start"); err != nil {
		t.Fatalf("Click(#start) error = %v", err)
	}

	timestamp := time.Date(2026, time.April, 1, 9, 45, 0, 0, time.UTC)
	if err := harness.Geolocation().EmitPosition(GeolocationPosition{
		Latitude:  35.25,
		Longitude: 137.75,
		Timestamp: &timestamp,
	}); err != nil {
		t.Fatalf("Geolocation().EmitPosition() error = %v", err)
	}

	if got, err := harness.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "35.25|137.75|2026-04-01T09:45:00.000Z" {
		t.Fatalf("TextContent(#out) = %q, want geolocation payload", got)
	}
}
