package runtime

import (
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"browsertester/internal/script"
)

const (
	GeolocationErrorPermissionDenied  = 1
	GeolocationErrorPositionUnavailable = 2
	GeolocationErrorTimeout           = 3
)

type GeolocationPosition struct {
	Latitude  float64
	Longitude float64
	Accuracy  *float64
	Altitude  *float64
	Timestamp *time.Time
}

type browserGeolocationWatch struct {
	id      int64
	success script.Value
	failure script.Value
	oneShot bool
}

func (s *Session) isSecureContext() bool {
	if s == nil {
		return false
	}
	parsed, err := s.currentLocationURL()
	if err != nil {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(parsed.Scheme)) {
	case "https":
		return true
	case "http":
		host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
		if host == "localhost" {
			return true
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return true
		}
	}
	return false
}

func (s *Session) clearGeolocationWatches() {
	if s == nil {
		return
	}
	s.geolocationWatches = nil
	s.nextGeolocationWatchID = 0
}

func cloneBrowserGeolocationWatchMap(watches map[int64]*browserGeolocationWatch) map[int64]*browserGeolocationWatch {
	if len(watches) == 0 {
		return nil
	}
	out := make(map[int64]*browserGeolocationWatch, len(watches))
	for id, watch := range watches {
		if watch == nil {
			out[id] = nil
			continue
		}
		cloned := *watch
		out[id] = &cloned
	}
	return out
}

func (s *Session) registerGeolocationWatch(success script.Value, failure script.Value, oneShot bool) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session is unavailable")
	}
	if !isCallableEventHandlerValue(success) {
		return 0, fmt.Errorf("geolocation success callback must be callable")
	}
	if failure.Kind != script.ValueKindUndefined && failure.Kind != script.ValueKindNull && !isCallableEventHandlerValue(failure) {
		return 0, fmt.Errorf("geolocation error callback must be callable or null")
	}
	if s.nextGeolocationWatchID == math.MaxInt64 {
		return 0, fmt.Errorf("geolocation watch id space exhausted")
	}
	s.nextGeolocationWatchID++
	id := s.nextGeolocationWatchID
	if s.geolocationWatches == nil {
		s.geolocationWatches = make(map[int64]*browserGeolocationWatch)
	}
	s.geolocationWatches[id] = &browserGeolocationWatch{
		id:      id,
		success: success,
		failure: failure,
		oneShot: oneShot,
	}
	return id, nil
}

func (s *Session) removeGeolocationWatch(id int64) {
	if s == nil || id <= 0 || len(s.geolocationWatches) == 0 {
		return
	}
	delete(s.geolocationWatches, id)
}

func (s *Session) EmitGeolocationPosition(position GeolocationPosition) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if len(s.geolocationWatches) == 0 {
		return nil
	}
	store := s.domStore
	if store == nil {
		return fmt.Errorf("geolocation watches require an active DOM store")
	}

	watches := make([]browserGeolocationWatch, 0, len(s.geolocationWatches))
	for _, watch := range s.geolocationWatches {
		if watch == nil {
			continue
		}
		watches = append(watches, *watch)
	}

	positionValue := browserGeolocationPositionValue(position)
	var firstErr error
	for _, watch := range watches {
		if watch.oneShot {
			s.removeGeolocationWatch(watch.id)
		}
		if !isCallableEventHandlerValue(watch.success) {
			continue
		}
		if _, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, watch.success, []script.Value{positionValue}, script.UndefinedValue(), false); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Session) EmitGeolocationError(code int, message string) error {
	if s == nil {
		return fmt.Errorf("session is unavailable")
	}
	if len(s.geolocationWatches) == 0 {
		return nil
	}
	store := s.domStore
	if store == nil {
		return fmt.Errorf("geolocation watches require an active DOM store")
	}

	watches := make([]browserGeolocationWatch, 0, len(s.geolocationWatches))
	for _, watch := range s.geolocationWatches {
		if watch == nil {
			continue
		}
		watches = append(watches, *watch)
	}

	errorValue := browserGeolocationErrorValue(code, message)
	var firstErr error
	for _, watch := range watches {
		if watch.oneShot || code == GeolocationErrorPermissionDenied {
			s.removeGeolocationWatch(watch.id)
		}
		callback := watch.failure
		if !isCallableEventHandlerValue(callback) {
			continue
		}
		if _, err := script.InvokeCallableValue(&inlineScriptHost{session: s, store: store}, callback, []script.Value{errorValue}, script.UndefinedValue(), false); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func resolveGeolocationReference(session *Session, path string) (script.Value, error) {
	rest := strings.TrimPrefix(strings.TrimSpace(path), ".")
	if rest == "" {
		if session == nil || !session.isSecureContext() {
			return script.UndefinedValue(), nil
		}
		return script.HostObjectReference("geolocation"), nil
	}
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "navigator.geolocation is unavailable in this bounded classic-JS slice")
	}
	if !session.isSecureContext() {
		return script.UndefinedValue(), nil
	}

	switch rest {
	case "watchPosition":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserGeolocationStartWatcher(session, false, args)
		}), nil
	case "getCurrentPosition":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserGeolocationStartWatcher(session, true, args)
		}), nil
	case "clearWatch":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			return browserGeolocationClearWatch(session, args)
		}), nil
	case "PERMISSION_DENIED":
		return script.NumberValue(float64(GeolocationErrorPermissionDenied)), nil
	case "POSITION_UNAVAILABLE":
		return script.NumberValue(float64(GeolocationErrorPositionUnavailable)), nil
	case "TIMEOUT":
		return script.NumberValue(float64(GeolocationErrorTimeout)), nil
	case "toString", "valueOf":
		return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) > 0 {
				return script.UndefinedValue(), fmt.Errorf("geolocation.%s accepts no arguments", rest)
			}
			return script.StringValue("[object Geolocation]"), nil
		}), nil
	}

	return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, fmt.Sprintf("unsupported browser surface %q in this bounded classic-JS slice", "geolocation."+rest))
}

func browserGeolocationStartWatcher(session *Session, oneShot bool, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "navigator.geolocation is unavailable in this bounded classic-JS slice")
	}
	if !session.isSecureContext() {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "navigator.geolocation is unavailable in this bounded classic-JS slice")
	}
	if len(args) < 1 || len(args) > 3 {
		return script.UndefinedValue(), fmt.Errorf("geolocation.%s expects 1 to 3 arguments", geolocationWatchMode(oneShot))
	}
	success := args[0]
	if !isCallableEventHandlerValue(success) {
		return script.UndefinedValue(), fmt.Errorf("geolocation success callback must be callable")
	}

	failure := script.UndefinedValue()
	if len(args) >= 2 {
		failure = args[1]
		if failure.Kind != script.ValueKindUndefined && failure.Kind != script.ValueKindNull && !isCallableEventHandlerValue(failure) {
			return script.UndefinedValue(), fmt.Errorf("geolocation error callback must be callable or null")
		}
	}

	id, err := session.registerGeolocationWatch(success, failure, oneShot)
	if err != nil {
		return script.UndefinedValue(), err
	}
	if oneShot {
		return script.UndefinedValue(), nil
	}
	return script.NumberValue(float64(id)), nil
}

func browserGeolocationClearWatch(session *Session, args []script.Value) (script.Value, error) {
	if session == nil {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "navigator.geolocation is unavailable in this bounded classic-JS slice")
	}
	if !session.isSecureContext() {
		return script.UndefinedValue(), script.NewError(script.ErrorKindUnsupported, "navigator.geolocation is unavailable in this bounded classic-JS slice")
	}
	if len(args) == 0 {
		return script.UndefinedValue(), nil
	}
	if len(args) > 1 {
		return script.UndefinedValue(), fmt.Errorf("geolocation.clearWatch accepts at most 1 argument")
	}
	id, err := browserInt64Value("geolocation.clearWatch", args[0])
	if err != nil {
		return script.UndefinedValue(), err
	}
	session.removeGeolocationWatch(id)
	return script.UndefinedValue(), nil
}

func geolocationWatchMode(oneShot bool) string {
	if oneShot {
		return "getCurrentPosition"
	}
	return "watchPosition"
}

func browserGeolocationPositionValue(position GeolocationPosition) script.Value {
	ts := position.Timestamp
	if ts == nil {
		now := time.Now().UTC()
		ts = &now
	}
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "coords", Value: script.ObjectValue([]script.ObjectEntry{
			{Key: "latitude", Value: script.NumberValue(position.Latitude)},
			{Key: "longitude", Value: script.NumberValue(position.Longitude)},
			{Key: "accuracy", Value: browserGeolocationNullableNumber(position.Accuracy)},
			{Key: "altitude", Value: browserGeolocationNullableNumber(position.Altitude)},
			{Key: "altitudeAccuracy", Value: script.NullValue()},
			{Key: "heading", Value: script.NullValue()},
			{Key: "speed", Value: script.NullValue()},
		})},
		{Key: "timestamp", Value: script.NumberValue(float64(ts.UnixMilli()))},
	})
}

func browserGeolocationErrorValue(code int, message string) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "code", Value: script.NumberValue(float64(code))},
		{Key: "message", Value: script.StringValue(message)},
		{Key: "PERMISSION_DENIED", Value: script.NumberValue(float64(GeolocationErrorPermissionDenied))},
		{Key: "POSITION_UNAVAILABLE", Value: script.NumberValue(float64(GeolocationErrorPositionUnavailable))},
		{Key: "TIMEOUT", Value: script.NumberValue(float64(GeolocationErrorTimeout))},
	})
}

func browserGeolocationNullableNumber(value *float64) script.Value {
	if value == nil {
		return script.NullValue()
	}
	if math.IsNaN(*value) {
		return script.NullValue()
	}
	return script.NumberValue(*value)
}
