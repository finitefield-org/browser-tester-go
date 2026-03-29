package runtime

import (
	"strings"
	"testing"

	"browsertester/internal/script"
)

func TestBrowserOpenWithoutArgumentsRecordsBlankPopupCall(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := browserOpen(session, nil); err != nil {
		t.Fatalf("browserOpen() error = %v", err)
	}
	if calls := session.OpenCalls(); len(calls) != 1 || calls[0].URL != "" {
		t.Fatalf("OpenCalls() = %#v, want one blank popup call", calls)
	}
}

func TestBrowserOpenForwardsStringCoercionOnUrlArgument(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := browserOpen(session, []script.Value{
		script.NumberValue(42),
	}); err != nil {
		t.Fatalf("browserOpen() error = %v", err)
	}
	if calls := session.OpenCalls(); len(calls) != 1 || calls[0].URL != "42" {
		t.Fatalf("OpenCalls() = %#v, want one numeric URL call", calls)
	}
}

func TestBrowserOpenReturnsDefensiveCopyThroughOpenCalls(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := browserOpen(session, []script.Value{
		script.StringValue("https://example.test/popup"),
	}); err != nil {
		t.Fatalf("browserOpen() error = %v", err)
	}

	calls := session.OpenCalls()
	if len(calls) != 1 || calls[0].URL != "https://example.test/popup" {
		t.Fatalf("OpenCalls() = %#v, want one popup call", calls)
	}
	calls[0].URL = "mutated"
	if fresh := session.OpenCalls(); len(fresh) != 1 || fresh[0].URL != "https://example.test/popup" {
		t.Fatalf("OpenCalls() reread = %#v, want original popup call", fresh)
	}
}

func TestBrowserOpenForwardsNullCoercionOnUrlArgument(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := browserOpen(session, []script.Value{
		script.NullValue(),
	}); err != nil {
		t.Fatalf("browserOpen() error = %v", err)
	}
	if calls := session.OpenCalls(); len(calls) != 1 || calls[0].URL != "null" {
		t.Fatalf("OpenCalls() = %#v, want one null URL call", calls)
	}
}

func TestBrowserOpenForwardsUndefinedCoercionOnUrlArgument(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := browserOpen(session, []script.Value{
		script.UndefinedValue(),
	}); err != nil {
		t.Fatalf("browserOpen() error = %v", err)
	}
	if calls := session.OpenCalls(); len(calls) != 1 || calls[0].URL != "undefined" {
		t.Fatalf("OpenCalls() = %#v, want one undefined URL call", calls)
	}
}

func TestBrowserOpenReportsOpenFailureWithoutArguments(t *testing.T) {
	session := NewSession(SessionConfig{
		OpenFailure: "open blocked",
	})

	_, err := browserOpen(session, nil)
	if err == nil {
		t.Fatalf("browserOpen() error = nil, want open failure")
	} else if !strings.Contains(err.Error(), "open blocked") {
		t.Fatalf("browserOpen() error = %v, want open blocked message", err)
	}
	if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "" {
		t.Fatalf("OpenCalls() after open failure = %#v, want one blank popup call", got)
	}
}

func TestBrowserOpenReportsOpenFailureForUrlArgument(t *testing.T) {
	session := NewSession(SessionConfig{
		OpenFailure: "open blocked",
	})

	_, err := browserOpen(session, []script.Value{
		script.StringValue("https://example.test/popup"),
	})
	if err == nil {
		t.Fatalf("browserOpen() error = nil, want open failure")
	} else if !strings.Contains(err.Error(), "open blocked") {
		t.Fatalf("browserOpen() error = %v, want open blocked message", err)
	}
	if got := session.OpenCalls(); len(got) != 1 || got[0].URL != "https://example.test/popup" {
		t.Fatalf("OpenCalls() after open failure = %#v, want one popup call", got)
	}
}

func TestBrowserOpenRejectsSymbolUrl(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	_, err := browserOpen(session, []script.Value{
		script.SymbolValue("token"),
	})
	if err == nil {
		t.Fatalf("browserOpen() error = nil, want Symbol coercion failure")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
		t.Fatalf("browserOpen() error = %#v, want runtime error", err)
	}
	if calls := session.OpenCalls(); len(calls) != 0 {
		t.Fatalf("OpenCalls() after rejected browserOpen() = %#v, want empty", calls)
	}
}

func TestBrowserOpenIgnoresExtraArguments(t *testing.T) {
	session := NewSession(DefaultSessionConfig())

	if _, err := browserOpen(session, []script.Value{
		script.StringValue(""),
		script.StringValue("_blank"),
		script.StringValue("noopener,noreferrer"),
		script.StringValue("ignored"),
	}); err != nil {
		t.Fatalf("browserOpen() error = %v", err)
	}
	if calls := session.OpenCalls(); len(calls) != 1 || calls[0].URL != "" {
		t.Fatalf("OpenCalls() = %#v, want one blank popup call", calls)
	}
}

func TestBrowserOpenRejectsSymbolTargetAndFeatures(t *testing.T) {
	tests := []struct {
		name string
		args []script.Value
	}{
		{
			name: "target",
			args: []script.Value{
				script.StringValue(""),
				script.SymbolValue("token"),
			},
		},
		{
			name: "features",
			args: []script.Value{
				script.StringValue(""),
				script.StringValue("_blank"),
				script.SymbolValue("token"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession(DefaultSessionConfig())
			_, err := browserOpen(session, tc.args)
			if err == nil {
				t.Fatalf("browserOpen() error = nil, want Symbol coercion failure")
			}
			if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindRuntime {
				t.Fatalf("browserOpen() error = %#v, want runtime error", err)
			}
			if calls := session.OpenCalls(); len(calls) != 0 {
				t.Fatalf("OpenCalls() after rejected browserOpen() = %#v, want empty", calls)
			}
		})
	}
}

func TestBrowserOpenWithoutArgumentsRejectsNilSession(t *testing.T) {
	_, err := browserOpen(nil, nil)
	if err == nil {
		t.Fatalf("browserOpen(nil, nil) error = nil, want unsupported error")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported {
		t.Fatalf("browserOpen(nil, nil) error = %#v, want unsupported error", err)
	}
}

func TestBrowserOpenRejectsNilSessionBeforeCoercion(t *testing.T) {
	_, err := browserOpen(nil, []script.Value{
		script.SymbolValue("token"),
	})
	if err == nil {
		t.Fatalf("browserOpen(nil, [Symbol]) error = nil, want unsupported error")
	}
	if scriptErr, ok := err.(script.Error); !ok || scriptErr.Kind != script.ErrorKindUnsupported {
		t.Fatalf("browserOpen(nil, [Symbol]) error = %#v, want unsupported error", err)
	}
}
