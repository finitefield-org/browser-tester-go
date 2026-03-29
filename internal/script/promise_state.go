package script

type classicJSPromiseState struct {
	resolved bool
	value    Value
	waiters  []classicJSPromiseWaiter
}

type classicJSPromiseWaiter struct {
	host   HostBindings
	waiter func(Value)
}

type classicJSAwaitSignal struct {
	promise     *classicJSPromiseState
	resumeState classicJSResumeState
}

func (s classicJSAwaitSignal) Error() string {
	return "await suspension"
}

func (s *classicJSPromiseState) cloneDetached(mapping map[*classicJSEnvironment]*classicJSEnvironment) *classicJSPromiseState {
	if s == nil {
		return nil
	}
	cloned := &classicJSPromiseState{
		resolved: s.resolved,
		value:    cloneValueDetached(s.value, mapping),
	}
	return cloned
}

func (s *classicJSPromiseState) resolve(value Value) {
	if s == nil || s.resolved {
		return
	}
	s.resolved = true
	s.value = value
	waiters := append([]classicJSPromiseWaiter(nil), s.waiters...)
	s.waiters = nil
	for _, item := range waiters {
		if item.waiter != nil {
			restoreHost := setCurrentInvokeHost(item.host)
			item.waiter(value)
			restoreHost()
		}
	}
}

func (s *classicJSPromiseState) addWaiter(waiter func(Value)) {
	if s == nil || waiter == nil {
		return
	}
	if s.resolved {
		waiter(s.value)
		return
	}
	s.waiters = append(s.waiters, classicJSPromiseWaiter{
		host:   CurrentInvokeHost(),
		waiter: waiter,
	})
}

func classicJSAwaitSignalDetails(err error) (*classicJSPromiseState, classicJSResumeState, bool) {
	signal, ok := err.(classicJSAwaitSignal)
	if !ok {
		return nil, nil, false
	}
	return signal.promise, signal.resumeState, true
}
