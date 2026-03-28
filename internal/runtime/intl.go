package runtime

import "browsertester/internal/script"

func (s *Session) intlOverrideValue() (script.Value, bool) {
	if s == nil || !s.hasIntlOverride {
		return script.Value{}, false
	}
	return s.intlOverride, true
}

func (s *Session) setIntlOverride(value script.Value) {
	if s == nil {
		return
	}
	s.intlOverride = value
	s.hasIntlOverride = true
}
