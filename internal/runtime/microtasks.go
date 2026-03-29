package runtime

import (
	"fmt"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

type microtaskRecord struct {
	source      string
	callback    script.Value
	args        []script.Value
	receiver    script.Value
	hasReceiver bool
}

func (s *Session) enqueueMicrotask(source string) {
	if s == nil {
		return
	}
	normalized := strings.TrimSpace(source)
	if normalized == "" {
		return
	}
	s.microtasks = append(s.microtasks, microtaskRecord{source: normalized})
}

func (s *Session) enqueueCallableMicrotask(callback script.Value, args []script.Value, receiver script.Value, hasReceiver bool) {
	if s == nil {
		return
	}
	clonedArgs := append([]script.Value(nil), args...)
	s.microtasks = append(s.microtasks, microtaskRecord{
		callback:    callback,
		args:        clonedArgs,
		receiver:    receiver,
		hasReceiver: hasReceiver,
	})
}

func (s *Session) discardMicrotasks() {
	if s == nil {
		return
	}
	s.microtasks = nil
}

func (s *Session) PendingMicrotasks() []string {
	if s == nil || len(s.microtasks) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.microtasks))
	for _, microtask := range s.microtasks {
		if microtask.source == "" {
			continue
		}
		out = append(out, microtask.source)
	}
	return out
}

func (s *Session) drainMicrotasks(store *dom.Store) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if store == nil {
		store = s.domStore
	}
	if store == nil {
		return fmt.Errorf("dom store is unavailable")
	}

	for len(s.microtasks) > 0 {
		if s.domStore != nil && s.domStore != store {
			store = s.domStore
		}
		batch := append([]microtaskRecord(nil), s.microtasks...)
		s.microtasks = nil
		for _, microtask := range batch {
			if s.domStore != nil && s.domStore != store {
				store = s.domStore
			}
			if microtask.source != "" {
				if _, err := s.runScriptOnStore(store, microtask.source); err != nil {
					s.microtasks = nil
					return err
				}
				continue
			}
			if microtask.callback.Kind == script.ValueKindUndefined {
				continue
			}
			if _, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, microtask.callback, microtask.args, microtask.receiver, microtask.hasReceiver); err != nil {
				s.microtasks = nil
				return err
			}
		}
	}
	return nil
}

func cloneMicrotaskQueue(records []microtaskRecord) []microtaskRecord {
	if len(records) == 0 {
		return nil
	}
	out := make([]microtaskRecord, len(records))
	for i := range records {
		out[i] = records[i]
		if len(records[i].args) > 0 {
			out[i].args = append([]script.Value(nil), records[i].args...)
		}
	}
	return out
}
