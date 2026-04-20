package runtime

import "testing"

func TestSessionMatchMediaMaxWidth899Fallback(t *testing.T) {
	s := NewSession(SessionConfig{})

	matches, err := s.MatchMedia("(max-width: 899px)")
	if err != nil {
		t.Fatalf("MatchMedia() error = %v", err)
	}
	if matches {
		t.Fatalf("MatchMedia() = %v, want false for default inner width", matches)
	}
}

