package script

var currentInvokeHost HostBindings

// CurrentInvokeHost returns the host bindings currently active for callback invocation.
func CurrentInvokeHost() HostBindings {
	return currentInvokeHost
}

func setCurrentInvokeHost(host HostBindings) func() {
	prev := currentInvokeHost
	currentInvokeHost = host
	return func() {
		currentInvokeHost = prev
	}
}

// InvokeCallableValue invokes a bounded callable value with an optional receiver.
// The host is used for host-reference resolution during the call.
func InvokeCallableValue(host HostBindings, callee Value, args []Value, receiver Value, hasReceiver bool) (Value, error) {
	restoreHost := setCurrentInvokeHost(host)
	defer restoreHost()

	parser := &classicJSStatementParser{
		host:      host,
		env:       newClassicJSEnvironment(),
		stepLimit: DefaultRuntimeConfig().StepLimit,
	}
	callable := scalarJSValue(callee)
	if hasReceiver {
		callable.receiver = receiver
		callable.hasReceiver = true
	}
	result, err := parser.invoke(callable, args)
	if err != nil {
		return UndefinedValue(), err
	}
	if result.kind != jsValueScalar {
		return UndefinedValue(), NewError(ErrorKindRuntime, "callable did not return a scalar value in this bounded classic-JS slice")
	}
	return result.value, nil
}

// ThrowValue returns a bounded throw signal that can be caught by classic-JS try/catch.
func ThrowValue(value Value) error {
	return classicJSThrowSignal{value: value}
}

// ThrowValueFromError extracts a bounded JS throw value from an error signal.
func ThrowValueFromError(err error) (Value, bool) {
	return classicJSThrowSignalValue(err)
}
