package runtime

import (
	"fmt"
	"strconv"

	"browsertester/internal/dom"
	"browsertester/internal/script"
)

type browserNodeListIterationMode uint8

const (
	browserNodeListIterationValues browserNodeListIterationMode = iota
	browserNodeListIterationKeys
	browserNodeListIterationEntries
)

func browserNodeListValue(session *Session, store *dom.Store, ids []dom.NodeID) (script.Value, error) {
	return browserNodeListValueFromIDs(session, store, ids)
}

func browserChildNodeListValue(session *Session, store *dom.Store, coll dom.ChildNodeList) (script.Value, error) {
	return browserNodeListValueFromIDs(session, store, coll.IDs())
}

func browserNodeListValueFromIDs(session *Session, store *dom.Store, ids []dom.NodeID) (script.Value, error) {
	var listValue script.Value
	entries := make([]script.ObjectEntry, 0, len(ids)+6)
	for i, nodeID := range ids {
		entries = append(entries, script.ObjectEntry{
			Key:   strconv.Itoa(i),
			Value: browserElementReferenceValue(nodeID),
		})
	}

	entries = append(entries, script.ObjectEntry{
		Key:   "length",
		Value: script.NumberValue(float64(len(ids))),
	})
	entries = append(entries, script.ObjectEntry{
		Key: "item",
		Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) != 1 {
				return script.UndefinedValue(), fmt.Errorf("NodeList.item expects 1 argument")
			}
			index, err := browserInt64Value("NodeList.item", args[0])
			if err != nil {
				return script.UndefinedValue(), err
			}
			if index < 0 || int(index) >= len(ids) {
				return script.NullValue(), nil
			}
			return browserElementReferenceValue(ids[int(index)]), nil
		}),
	})

	entries = append(entries, script.ObjectEntry{
		Key: "forEach",
		Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
			if len(args) == 0 {
				return script.UndefinedValue(), script.ThrowValue(script.StringValue("NodeList.forEach expects 1 argument"))
			}
			callback := args[0]
			thisArg := script.UndefinedValue()
			hasThisArg := false
			if len(args) > 1 {
				thisArg = args[1]
				hasThisArg = true
			}
			for i, nodeID := range ids {
				_, err := script.InvokeCallableValue(
					&inlineScriptHost{session: session, store: store},
					callback,
					[]script.Value{
						browserElementReferenceValue(nodeID),
						script.NumberValue(float64(i)),
						listValue,
					},
					thisArg,
					hasThisArg,
				)
				if err != nil {
					return script.UndefinedValue(), err
				}
			}
			return script.UndefinedValue(), nil
		}),
	})
	entriesValue, err := browserNodeListIteratorMethodValue(ids, browserNodeListIterationEntries)
	if err != nil {
		return script.UndefinedValue(), err
	}
	entries = append(entries, script.ObjectEntry{
		Key:   "entries",
		Value: entriesValue,
	})
	keysValue, err := browserNodeListIteratorMethodValue(ids, browserNodeListIterationKeys)
	if err != nil {
		return script.UndefinedValue(), err
	}
	entries = append(entries, script.ObjectEntry{
		Key:   "keys",
		Value: keysValue,
	})
	valuesValue, err := browserNodeListIteratorMethodValue(ids, browserNodeListIterationValues)
	if err != nil {
		return script.UndefinedValue(), err
	}
	entries = append(entries, script.ObjectEntry{
		Key:   "values",
		Value: valuesValue,
	})
	listValue = script.ObjectValue(entries)
	return listValue, nil
}

func browserNodeListIteratorMethodValue(ids []dom.NodeID, mode browserNodeListIterationMode) (script.Value, error) {
	return script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
		return browserNodeListIteratorValue(ids, mode)
	}), nil
}

func browserNodeListIteratorValue(ids []dom.NodeID, mode browserNodeListIterationMode) (script.Value, error) {
	index := 0
	return script.ObjectValue([]script.ObjectEntry{
		{
			Key: "next",
			Value: script.NativeFunctionValue(func(args []script.Value) (script.Value, error) {
				if index >= len(ids) {
					return browserNodeListIteratorResult(script.UndefinedValue(), true), nil
				}

				currentIndex := index
				currentNodeID := ids[currentIndex]
				index++

				switch mode {
				case browserNodeListIterationKeys:
					return browserNodeListIteratorResult(script.NumberValue(float64(currentIndex)), false), nil
				case browserNodeListIterationEntries:
					return browserNodeListIteratorResult(script.ArrayValue([]script.Value{
						script.NumberValue(float64(currentIndex)),
						browserElementReferenceValue(currentNodeID),
					}), false), nil
				default:
					return browserNodeListIteratorResult(browserElementReferenceValue(currentNodeID), false), nil
				}
			}),
		},
	}), nil
}

func browserNodeListIteratorResult(value script.Value, done bool) script.Value {
	return script.ObjectValue([]script.ObjectEntry{
		{Key: "value", Value: value},
		{Key: "done", Value: script.BoolValue(done)},
	})
}
