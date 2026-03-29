package runtime

import (
	"strings"
	"testing"
)

func TestSessionWindowNameDefaultsAndPersists(t *testing.T) {
	s := NewSession(DefaultSessionConfig())

	if got := s.WindowName(); got != "" {
		t.Fatalf("windowName() = %q, want empty", got)
	}

	if err := s.setWindowName("alpha"); err != nil {
		t.Fatalf("setWindowName() error = %v", err)
	}
	if got := s.WindowName(); got != "alpha" {
		t.Fatalf("windowName() after set = %q, want alpha", got)
	}

	if err := s.Navigate("https://example.test/next"); err != nil {
		t.Fatalf("Navigate() error = %v", err)
	}
	if got := s.WindowName(); got != "alpha" {
		t.Fatalf("windowName() after Navigate = %q, want alpha", got)
	}

	if err := s.WriteHTML("<main></main>"); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}
	if got := s.WindowName(); got != "alpha" {
		t.Fatalf("windowName() after WriteHTML = %q, want alpha", got)
	}
}

func TestSessionWriteHTMLRestoresWindowNameOnFailure(t *testing.T) {
	s := NewSession(DefaultSessionConfig())

	if err := s.setWindowName("alpha"); err != nil {
		t.Fatalf("setWindowName() error = %v", err)
	}

	if err := s.WriteHTML(`<main><script>host:setWindowName("beta"); host:doesNotExist()</script></main>`); err == nil {
		t.Fatalf("WriteHTML() error = nil, want host failure")
	}

	if got := s.WindowName(); got != "alpha" {
		t.Fatalf("windowName() after failed WriteHTML = %q, want alpha", got)
	}
}

func TestSessionRejectsWindowNameSymbolInputFromInlineScript(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><script>host:setWindowName(expr(Symbol("token")))</script></main>`,
	})

	if _, err := s.ensureDOM(); err == nil {
		t.Fatalf("ensureDOM() error = nil, want Symbol coercion failure")
	} else if !strings.Contains(err.Error(), "Cannot convert a Symbol value to a string") {
		t.Fatalf("ensureDOM() error = %v, want Symbol coercion failure message", err)
	}
	if got := s.WindowName(); got != "" {
		t.Fatalf("WindowName() after rejected Symbol input = %q, want empty", got)
	}
}
