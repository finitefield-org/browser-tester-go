package runtime

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

type timerRecord struct {
	id           int64
	source       string
	callback     script.Value
	callbackArgs []script.Value
	hasCallback  bool
	dueAt        int64
	repeat       bool
	intervalMs   int64
}

type animationFrameRecord struct {
	id          int64
	source      string
	callback    script.Value
	hasCallback bool
}

type TimerSnapshot struct {
	ID         int64
	Source     string
	DueAtMs    int64
	Repeat     bool
	IntervalMs int64
}

type AnimationFrameSnapshot struct {
	ID     int64
	Source string
}

func (s *Session) scheduleTimeout(source string, timeoutMs int64) (int64, error) {
	return s.scheduleTimerRecord(timerRecord{source: source}, timeoutMs, false)
}

func (s *Session) scheduleInterval(source string, timeoutMs int64) (int64, error) {
	return s.scheduleTimerRecord(timerRecord{source: source}, timeoutMs, true)
}

func (s *Session) scheduleTimeoutCallback(callback script.Value, callbackArgs []script.Value, timeoutMs int64) (int64, error) {
	if callback.Kind != script.ValueKindFunction || (callback.NativeFunction == nil && callback.Function == nil) {
		return 0, fmt.Errorf("setTimeout callback must be callable")
	}
	return s.scheduleTimerRecord(timerRecord{
		callback:     callback,
		callbackArgs: append([]script.Value(nil), callbackArgs...),
		hasCallback:  true,
	}, timeoutMs, false)
}

func (s *Session) scheduleIntervalCallback(callback script.Value, callbackArgs []script.Value, timeoutMs int64) (int64, error) {
	if callback.Kind != script.ValueKindFunction || (callback.NativeFunction == nil && callback.Function == nil) {
		return 0, fmt.Errorf("setInterval callback must be callable")
	}
	return s.scheduleTimerRecord(timerRecord{
		callback:     callback,
		callbackArgs: append([]script.Value(nil), callbackArgs...),
		hasCallback:  true,
	}, timeoutMs, true)
}

func (s *Session) scheduleTimerRecord(record timerRecord, timeoutMs int64, repeat bool) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session is unavailable")
	}

	if record.hasCallback {
		if record.callback.Kind != script.ValueKindFunction || (record.callback.NativeFunction == nil && record.callback.Function == nil) {
			return 0, fmt.Errorf("timer callback must be callable")
		}
	} else {
		normalized := strings.TrimSpace(record.source)
		if normalized == "" {
			return 0, fmt.Errorf("timer source must not be empty")
		}
		record.source = normalized
	}
	if timeoutMs < 0 {
		timeoutMs = 0
	}

	if s.nextTimerID == math.MaxInt64 {
		return 0, fmt.Errorf("timer id space exhausted")
	}
	s.nextTimerID++
	id := s.nextTimerID

	if s.timers == nil {
		s.timers = make(map[int64]timerRecord)
	}
	record.id = id
	record.dueAt = saturatingAddInt64(s.scheduler.NowMs(), timeoutMs)
	record.repeat = repeat
	record.intervalMs = timeoutMs
	s.timers[id] = record
	return id, nil
}

func (s *Session) clearTimeout(id int64) {
	if s == nil || id <= 0 || len(s.timers) == 0 {
		if s != nil && s.runningTimerID == id {
			s.runningTimerCancelled = true
		}
		return
	}
	delete(s.timers, id)
	if s.runningTimerID == id {
		s.runningTimerCancelled = true
	}
}

func (s *Session) clearInterval(id int64) {
	s.clearTimeout(id)
}

func (s *Session) requestAnimationFrame(source string) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session is unavailable")
	}

	normalized := strings.TrimSpace(source)
	if normalized == "" {
		return 0, fmt.Errorf("animation frame source must not be empty")
	}
	return s.scheduleAnimationFrame(animationFrameRecord{source: normalized})
}

func (s *Session) requestAnimationFrameCallback(callback script.Value) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session is unavailable")
	}
	if callback.Kind != script.ValueKindFunction || (callback.NativeFunction == nil && callback.Function == nil) {
		return 0, fmt.Errorf("requestAnimationFrame callback must be callable")
	}
	return s.scheduleAnimationFrame(animationFrameRecord{callback: callback, hasCallback: true})
}

func (s *Session) scheduleAnimationFrame(frame animationFrameRecord) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session is unavailable")
	}
	if s.nextAnimationFrameID == math.MaxInt64 {
		return 0, fmt.Errorf("animation frame id space exhausted")
	}
	s.nextAnimationFrameID++
	id := s.nextAnimationFrameID
	if s.animationFrames == nil {
		s.animationFrames = make(map[int64]animationFrameRecord)
	}
	frame.id = id
	s.animationFrames[id] = frame
	return id, nil
}

func (s *Session) cancelAnimationFrame(id int64) {
	if s == nil || id <= 0 || len(s.animationFrames) == 0 {
		return
	}
	delete(s.animationFrames, id)
}

func (s *Session) clearAnimationFrames() {
	if s == nil {
		return
	}
	s.animationFrames = nil
	s.nextAnimationFrameID = 0
}

func (s *Session) clearTimers() {
	if s == nil {
		return
	}
	s.timers = nil
	s.nextTimerID = 0
	s.runningTimerID = 0
	s.runningTimerCancelled = false
}

func cloneTimerMap(timers map[int64]timerRecord) map[int64]timerRecord {
	if len(timers) == 0 {
		return nil
	}
	out := make(map[int64]timerRecord, len(timers))
	for id, timer := range timers {
		if len(timer.callbackArgs) > 0 {
			timer.callbackArgs = append([]script.Value(nil), timer.callbackArgs...)
		}
		out[id] = timer
	}
	return out
}

func cloneAnimationFrameMap(frames map[int64]animationFrameRecord) map[int64]animationFrameRecord {
	if len(frames) == 0 {
		return nil
	}
	out := make(map[int64]animationFrameRecord, len(frames))
	for id, frame := range frames {
		out[id] = frame
	}
	return out
}

func (s *Session) PendingTimers() []TimerSnapshot {
	if s == nil {
		return nil
	}
	if _, err := s.ensureDOM(); err != nil {
		return nil
	}
	if len(s.timers) == 0 {
		return nil
	}
	out := make([]TimerSnapshot, 0, len(s.timers))
	for _, timer := range s.timers {
		out = append(out, TimerSnapshot{
			ID:         timer.id,
			Source:     timer.source,
			DueAtMs:    timer.dueAt,
			Repeat:     timer.repeat,
			IntervalMs: timer.intervalMs,
		})
	}
	if len(out) < 2 {
		return out
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DueAtMs != out[j].DueAtMs {
			return out[i].DueAtMs < out[j].DueAtMs
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *Session) PendingAnimationFrames() []AnimationFrameSnapshot {
	if s == nil {
		return nil
	}
	if _, err := s.ensureDOM(); err != nil {
		return nil
	}
	if len(s.animationFrames) == 0 {
		return nil
	}
	out := make([]AnimationFrameSnapshot, 0, len(s.animationFrames))
	for _, frame := range s.animationFrames {
		out = append(out, AnimationFrameSnapshot{
			ID:     frame.id,
			Source: frame.source,
		})
	}
	if len(out) < 2 {
		return out
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *Session) settlePendingWork(store *dom.Store) (err error) {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return fmt.Errorf("dom store is unavailable")
	}
	defer func() {
		if err != nil {
			s.discardMicrotasks()
		}
	}()

	if err = s.drainMicrotasks(store); err != nil {
		return err
	}
	if s.domStore != nil && s.domStore != store {
		store = s.domStore
	}

	for _, timer := range s.dueTimers() {
		if err = s.runTimer(store, timer); err != nil {
			return err
		}
		if err = s.drainMicrotasks(store); err != nil {
			return err
		}
		if s.domStore != nil && s.domStore != store {
			return nil
		}
	}

	for _, frame := range s.pendingAnimationFrames() {
		if err = s.runAnimationFrame(store, frame); err != nil {
			return err
		}
		if err = s.drainMicrotasks(store); err != nil {
			return err
		}
		if s.domStore != nil && s.domStore != store {
			return nil
		}
	}
	return nil
}

func (s *Session) dueTimers() []timerRecord {
	if s == nil || len(s.timers) == 0 {
		return nil
	}

	now := s.scheduler.NowMs()
	out := make([]timerRecord, 0, len(s.timers))
	for _, timer := range s.timers {
		if timer.dueAt <= now {
			out = append(out, timer)
		}
	}
	if len(out) < 2 {
		return out
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].dueAt != out[j].dueAt {
			return out[i].dueAt < out[j].dueAt
		}
		return out[i].id < out[j].id
	})
	return out
}

func (s *Session) runTimer(store *dom.Store, timer timerRecord) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return fmt.Errorf("dom store is unavailable")
	}
	if len(s.timers) == 0 {
		return nil
	}

	current, ok := s.timers[timer.id]
	if !ok {
		return nil
	}
	delete(s.timers, timer.id)
	prevRunningID := s.runningTimerID
	prevRunningCancelled := s.runningTimerCancelled
	s.runningTimerID = current.id
	s.runningTimerCancelled = false
	defer func() {
		s.runningTimerID = prevRunningID
		s.runningTimerCancelled = prevRunningCancelled
	}()

	if current.hasCallback {
		_, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, current.callback, current.callbackArgs, script.HostObjectReference("window"), true)
		if err != nil {
			return err
		}
	} else {
		if _, err := s.runScriptOnStore(store, current.source); err != nil {
			return err
		}
	}
	if current.repeat && !s.runningTimerCancelled {
		if s.timers == nil {
			s.timers = make(map[int64]timerRecord)
		}
		next := current
		next.dueAt = saturatingAddInt64(s.scheduler.NowMs(), current.intervalMs)
		next.repeat = true
		next.intervalMs = current.intervalMs
		if len(next.callbackArgs) > 0 {
			next.callbackArgs = append([]script.Value(nil), next.callbackArgs...)
		}
		s.timers[current.id] = next
	}
	return nil
}

func (s *Session) pendingAnimationFrames() []animationFrameRecord {
	if s == nil || len(s.animationFrames) == 0 {
		return nil
	}

	out := make([]animationFrameRecord, 0, len(s.animationFrames))
	for _, frame := range s.animationFrames {
		out = append(out, frame)
	}
	if len(out) < 2 {
		return out
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].id < out[j].id
	})
	return out
}

func (s *Session) runAnimationFrame(store *dom.Store, frame animationFrameRecord) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if store == nil {
		return fmt.Errorf("dom store is unavailable")
	}
	if len(s.animationFrames) == 0 {
		return nil
	}

	current, ok := s.animationFrames[frame.id]
	if !ok {
		return nil
	}
	delete(s.animationFrames, frame.id)
	if current.hasCallback {
		_, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, current.callback, nil, script.HostObjectReference("window"), true)
		return err
	}
	_, err := s.runScriptOnStore(store, current.source)
	return err
}

func saturatingAddInt64(base, delta int64) int64 {
	if delta > 0 && base > math.MaxInt64-delta {
		return math.MaxInt64
	}
	if delta < 0 && base < math.MinInt64-delta {
		return math.MinInt64
	}
	return base + delta
}
