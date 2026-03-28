package runtime

import (
	"fmt"
	"math"
	"testing"

	"browsertester/internal/script"
)

func TestSessionAdvanceTimeRunsTimersInDueOrder(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	if _, err := s.scheduleTimeout(`host:insertAdjacentHTML("#log", "beforeend", "<span>one</span>")`, 10); err != nil {
		t.Fatalf("scheduleTimeout(one) error = %v", err)
	}
	if _, err := s.scheduleTimeout(`host:insertAdjacentHTML("#log", "beforeend", "<span>two</span>")`, 10); err != nil {
		t.Fatalf("scheduleTimeout(two) error = %v", err)
	}
	if _, err := s.scheduleTimeout(`host:insertAdjacentHTML("#log", "beforeend", "<span>early</span>")`, 5); err != nil {
		t.Fatalf("scheduleTimeout(early) error = %v", err)
	}

	if err := s.AdvanceTime(10); err != nil {
		t.Fatalf("AdvanceTime(10) error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><div id="log"><span>early</span><span>one</span><span>two</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after timer delivery = %q, want %q", got, want)
	}
	if got, want := s.NowMs(), int64(10); got != want {
		t.Fatalf("NowMs() after timer delivery = %d, want %d", got, want)
	}
}

func TestSessionAdvanceTimeRunsMicrotasksQueuedByTimers(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	if _, err := s.scheduleTimeout(`host:queueMicrotask('host:insertAdjacentHTML("#log", "beforeend", "<span>micro</span>")')`, 3); err != nil {
		t.Fatalf("scheduleTimeout(microtask) error = %v", err)
	}

	if err := s.AdvanceTime(3); err != nil {
		t.Fatalf("AdvanceTime(3) error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><div id="log"><span>micro</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after microtask timer = %q, want %q", got, want)
	}
}

func TestSessionAdvanceTimeRunsRepeatingTimersOncePerAdvance(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	id, err := s.scheduleInterval(`host:insertAdjacentHTML("#log", "beforeend", "<span>tick</span>")`, 5)
	if err != nil {
		t.Fatalf("scheduleInterval(tick) error = %v", err)
	}

	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) first error = %v", err)
	}
	if got, want := s.DumpDOM(), `<main><div id="log"><span>tick</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after first interval = %q, want %q", got, want)
	}

	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) second error = %v", err)
	}
	if got, want := s.DumpDOM(), `<main><div id="log"><span>tick</span><span>tick</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after second interval = %q, want %q", got, want)
	}

	s.clearInterval(id)
	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) after clearInterval error = %v", err)
	}
	if got, want := s.DumpDOM(), `<main><div id="log"><span>tick</span><span>tick</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after cleared interval = %q, want %q", got, want)
	}
}

func TestSessionAdvanceTimeRunsTimerCallbacksWithArguments(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	var calls []string
	callback := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if len(args) != 2 {
			return script.UndefinedValue(), fmt.Errorf("timer callback args = %#v, want 2 arguments", args)
		}
		calls = append(calls, script.ToJSString(args[0])+"|"+script.ToJSString(args[1]))
		return script.UndefinedValue(), nil
	})

	if _, err := s.scheduleTimeoutCallback(callback, []script.Value{script.StringValue("timeout"), script.StringValue("one")}, 5); err != nil {
		t.Fatalf("scheduleTimeoutCallback(timeout) error = %v", err)
	}
	intervalID, err := s.scheduleIntervalCallback(callback, []script.Value{script.StringValue("interval"), script.StringValue("two")}, 5)
	if err != nil {
		t.Fatalf("scheduleIntervalCallback(interval) error = %v", err)
	}

	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) first error = %v", err)
	}
	if got, want := len(calls), 2; got != want {
		t.Fatalf("calls after first advance = %#v, want %d entries", calls, want)
	}
	if calls[0] != "timeout|one" || calls[1] != "interval|two" {
		t.Fatalf("calls after first advance = %#v, want timeout and interval callbacks", calls)
	}

	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) second error = %v", err)
	}
	if got, want := len(calls), 3; got != want {
		t.Fatalf("calls after second advance = %#v, want %d entries", calls, want)
	}
	if calls[2] != "interval|two" {
		t.Fatalf("calls after second advance = %#v, want interval callback to repeat", calls)
	}

	s.clearInterval(intervalID)
	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) after clearInterval error = %v", err)
	}
	if got, want := len(calls), 3; got != want {
		t.Fatalf("calls after clearInterval = %#v, want %d entries", calls, want)
	}
}

func TestSessionAdvanceTimeRunsTimersScheduledAtMaxNowWithoutOverflow(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	if err := s.AdvanceTime(math.MaxInt64); err != nil {
		t.Fatalf("AdvanceTime(MaxInt64) error = %v", err)
	}
	if got, want := s.NowMs(), int64(math.MaxInt64); got != want {
		t.Fatalf("NowMs() after saturating advance = %d, want %d", got, want)
	}

	if _, err := s.scheduleTimeout(`host:insertAdjacentHTML("#log", "beforeend", "<span>t</span>")`, 1); err != nil {
		t.Fatalf("scheduleTimeout(max-now) error = %v", err)
	}

	var intervalID int64
	intervalCallback := script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		if err := s.InsertAdjacentHTML("#log", "beforeend", "<span>i</span>"); err != nil {
			return script.UndefinedValue(), err
		}
		s.clearInterval(intervalID)
		return script.UndefinedValue(), nil
	})
	intervalID, err := s.scheduleIntervalCallback(intervalCallback, nil, 1)
	if err != nil {
		t.Fatalf("scheduleIntervalCallback(max-now) error = %v", err)
	}

	timers := s.PendingTimers()
	if len(timers) != 2 {
		t.Fatalf("PendingTimers() before delivery = %#v, want 2 entries", timers)
	}
	if timers[0].DueAtMs != math.MaxInt64 || timers[1].DueAtMs != math.MaxInt64 {
		t.Fatalf("PendingTimers() before delivery = %#v, want due-at max saturation", timers)
	}

	if err := s.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) at MaxInt64 error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><div id="log"><span>t</span><span>i</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after max-now timers = %q, want %q", got, want)
	}
	if got := s.PendingTimers(); len(got) != 0 {
		t.Fatalf("PendingTimers() after delivery = %#v, want empty", got)
	}
}

func TestSessionAdvanceTimeRunsAnimationFrames(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	if _, err := s.requestAnimationFrame(`host:insertAdjacentHTML("#log", "beforeend", "<span>frame</span>")`); err != nil {
		t.Fatalf("requestAnimationFrame(frame) error = %v", err)
	}

	if err := s.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><div id="log"><span>frame</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after animation frame = %q, want %q", got, want)
	}
}

func TestSessionCancelAnimationFrameRemovesPendingFrame(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	id, err := s.requestAnimationFrame(`host:insertAdjacentHTML("#log", "beforeend", "<span>frame</span>")`)
	if err != nil {
		t.Fatalf("requestAnimationFrame(frame) error = %v", err)
	}
	s.cancelAnimationFrame(id)

	if err := s.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><div id="log"></div></main>`; got != want {
		t.Fatalf("DumpDOM() after cancelAnimationFrame = %q, want %q", got, want)
	}
}

func TestSessionClearTimeoutCancelsTimer(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	id, err := s.scheduleTimeout(`host:insertAdjacentHTML("#log", "beforeend", "<span>cancelled</span>")`, 5)
	if err != nil {
		t.Fatalf("scheduleTimeout(cancelled) error = %v", err)
	}
	s.clearTimeout(id)

	if err := s.AdvanceTime(5); err != nil {
		t.Fatalf("AdvanceTime(5) error = %v", err)
	}

	if got, want := s.DumpDOM(), `<main><div id="log"></div></main>`; got != want {
		t.Fatalf("DumpDOM() after clearTimeout = %q, want %q", got, want)
	}
}

func TestSessionAdvanceTimeDefersTimersCreatedDuringCallbacks(t *testing.T) {
	s := NewSession(SessionConfig{
		HTML: `<main><div id="log"></div></main>`,
	})

	if _, err := s.scheduleTimeout(`host:setTimeout('host:insertAdjacentHTML("#log", "beforeend", "<span>late</span>")', 0)`, 1); err != nil {
		t.Fatalf("scheduleTimeout(nested) error = %v", err)
	}

	if err := s.AdvanceTime(1); err != nil {
		t.Fatalf("AdvanceTime(1) error = %v", err)
	}
	if got, want := s.DumpDOM(), `<main><div id="log"></div></main>`; got != want {
		t.Fatalf("DumpDOM() after first advance = %q, want %q", got, want)
	}

	if err := s.AdvanceTime(0); err != nil {
		t.Fatalf("AdvanceTime(0) error = %v", err)
	}
	if got, want := s.DumpDOM(), `<main><div id="log"><span>late</span></div></main>`; got != want {
		t.Fatalf("DumpDOM() after second advance = %q, want %q", got, want)
	}
}

func TestSessionPendingTimersAndFramesInspection(t *testing.T) {
	s := NewSession(DefaultSessionConfig())
	if err := s.WriteHTML(`<main><script>host:setTimeout('host:queueMicrotask("noop")', 5); host:setInterval('host:queueMicrotask("noop")', 9); host:requestAnimationFrame('host:queueMicrotask("noop")')</script></main>`); err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}

	timers := s.PendingTimers()
	if len(timers) != 2 {
		t.Fatalf("PendingTimers() = %#v, want 2 entries", timers)
	}
	if timers[0].DueAtMs != 5 || timers[0].IntervalMs != 5 || timers[0].Repeat {
		t.Fatalf("PendingTimers()[0] = %#v, want one-shot timer due at 5", timers[0])
	}
	if timers[1].DueAtMs != 9 || timers[1].IntervalMs != 9 || !timers[1].Repeat {
		t.Fatalf("PendingTimers()[1] = %#v, want repeating timer due at 9", timers[1])
	}

	timers[0].Source = "mutated"
	if fresh := s.PendingTimers(); len(fresh) != 2 || fresh[0].Source == "mutated" {
		t.Fatalf("PendingTimers() reread = %#v, want original timers", fresh)
	}

	frames := s.PendingAnimationFrames()
	if len(frames) != 1 {
		t.Fatalf("PendingAnimationFrames() = %#v, want 1 entry", frames)
	}
	if frames[0].Source != `host:queueMicrotask("noop")` {
		t.Fatalf("PendingAnimationFrames()[0] = %#v, want queued frame source", frames[0])
	}
	frames[0].Source = "mutated"
	if fresh := s.PendingAnimationFrames(); len(fresh) != 1 || fresh[0].Source != `host:queueMicrotask("noop")` {
		t.Fatalf("PendingAnimationFrames() reread = %#v, want original frame", fresh)
	}

	var nilSession *Session
	if got := nilSession.PendingTimers(); got != nil {
		t.Fatalf("nil PendingTimers() = %#v, want nil", got)
	}
	if got := nilSession.PendingAnimationFrames(); got != nil {
		t.Fatalf("nil PendingAnimationFrames() = %#v, want nil", got)
	}
}
