package runtime

import (
	"testing"
	"time"
)

func TestSessionBootstrapsSecureContextReflectsURL(t *testing.T) {
	const rawHTML = `<main><div id="out"></div><script>document.getElementById("out").textContent = String(window.isSecureContext)</script></main>`

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "https",
			url:  "https://example.test/tools/forestry/area-boundary-calculator/",
			want: "true",
		},
		{
			name: "http-example",
			url:  "http://example.test/tools/forestry/area-boundary-calculator/",
			want: "false",
		},
		{
			name: "http-localhost",
			url:  "http://localhost/tools/forestry/area-boundary-calculator/",
			want: "true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(SessionConfig{URL: tc.url, HTML: rawHTML})
			if _, err := session.ensureDOM(); err != nil {
				t.Fatalf("ensureDOM() error = %v", err)
			}

			if got, err := session.TextContent("#out"); err != nil {
				t.Fatalf("TextContent(#out) error = %v", err)
			} else if got != tc.want {
				t.Fatalf("TextContent(#out) = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSessionBootstrapsGeolocationWatchPositionAndClearWatch(t *testing.T) {
	const rawHTML = `<main><button id="start">start</button><button id="stop">stop</button><div id="out">idle</div><script>let watchId = null; document.getElementById("start").addEventListener("click", () => { watchId = navigator.geolocation.watchPosition((position) => { document.getElementById("out").textContent = [position.coords.latitude, position.coords.longitude, position.coords.accuracy, position.coords.altitude, new Date(position.timestamp).toISOString()].join("|"); }, (error) => { document.getElementById("out").textContent = ["error", error.code, error.PERMISSION_DENIED].join("|"); }); }); document.getElementById("stop").addEventListener("click", () => { navigator.geolocation.clearWatch(watchId); watchId = null; });</script></main>`

	session := NewSession(SessionConfig{
		URL:  "https://example.test/tools/forestry/area-boundary-calculator/",
		HTML: rawHTML,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.Click("#start"); err != nil {
		t.Fatalf("Click(#start) error = %v", err)
	}

	timestamp := time.Date(2026, time.April, 1, 9, 30, 45, 123000000, time.UTC)
	accuracy := 4.5
	altitude := 12.5
	if err := session.EmitGeolocationPosition(GeolocationPosition{
		Latitude:  35.123456,
		Longitude: 137.654321,
		Accuracy:  &accuracy,
		Altitude:  &altitude,
		Timestamp: &timestamp,
	}); err != nil {
		t.Fatalf("EmitGeolocationPosition() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "35.123456|137.654321|4.5|12.5|2026-04-01T09:30:45.123Z" {
		t.Fatalf("TextContent(#out) = %q, want position payload", got)
	}

	if err := session.Click("#stop"); err != nil {
		t.Fatalf("Click(#stop) error = %v", err)
	}
	if err := session.EmitGeolocationPosition(GeolocationPosition{
		Latitude:  36.5,
		Longitude: 138.5,
		Accuracy:  &accuracy,
		Altitude:  &altitude,
		Timestamp: &timestamp,
	}); err != nil {
		t.Fatalf("EmitGeolocationPosition() after clearWatch error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after clearWatch error = %v", err)
	} else if got != "35.123456|137.654321|4.5|12.5|2026-04-01T09:30:45.123Z" {
		t.Fatalf("TextContent(#out) after clearWatch = %q, want unchanged position payload", got)
	}
}

func TestSessionBootstrapsGeolocationPermissionDeniedError(t *testing.T) {
	const rawHTML = `<main><button id="start">start</button><div id="out">idle</div><script>document.getElementById("start").addEventListener("click", () => { navigator.geolocation.watchPosition((position) => { document.getElementById("out").textContent = String(position.coords.latitude); }, (error) => { document.getElementById("out").textContent = ["error", error.code, error.PERMISSION_DENIED].join("|"); }); });</script></main>`

	session := NewSession(SessionConfig{
		URL:  "https://example.test/tools/forestry/area-boundary-calculator/",
		HTML: rawHTML,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.Click("#start"); err != nil {
		t.Fatalf("Click(#start) error = %v", err)
	}
	if err := session.EmitGeolocationError(GeolocationErrorPermissionDenied, "permission denied"); err != nil {
		t.Fatalf("EmitGeolocationError() error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "error|1|1" {
		t.Fatalf("TextContent(#out) = %q, want permission denied error payload", got)
	}
}

func TestSessionWriteHTMLClearsGeolocationWatches(t *testing.T) {
	const firstHTML = `<main><button id="start">start</button><div id="out">idle</div><script>document.getElementById("start").addEventListener("click", () => { navigator.geolocation.watchPosition((position) => { document.getElementById("out").textContent = [position.coords.latitude, position.coords.longitude].join("|"); }); });</script></main>`
	const secondHTML = `<main><div id="out">fresh</div></main>`

	session := NewSession(SessionConfig{
		URL:  "https://example.test/tools/forestry/area-boundary-calculator/",
		HTML: firstHTML,
	})

	if _, err := session.ensureDOM(); err != nil {
		t.Fatalf("ensureDOM() error = %v", err)
	}
	if err := session.Click("#start"); err != nil {
		t.Fatalf("Click(#start) error = %v", err)
	}
	if err := session.EmitGeolocationPosition(GeolocationPosition{
		Latitude:  35.5,
		Longitude: 137.5,
	}); err != nil {
		t.Fatalf("EmitGeolocationPosition() error = %v", err)
	}
	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) error = %v", err)
	} else if got != "35.5|137.5" {
		t.Fatalf("TextContent(#out) = %q, want initial position payload", got)
	}

	if err := session.WriteHTML(secondHTML); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}
	if err := session.EmitGeolocationPosition(GeolocationPosition{
		Latitude:  36.5,
		Longitude: 138.5,
	}); err != nil {
		t.Fatalf("EmitGeolocationPosition() after WriteHTML error = %v", err)
	}

	if got, err := session.TextContent("#out"); err != nil {
		t.Fatalf("TextContent(#out) after WriteHTML error = %v", err)
	} else if got != "fresh" {
		t.Fatalf("TextContent(#out) after WriteHTML = %q, want fresh", got)
	}
}
