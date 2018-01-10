package timers

import (
	"time"

	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

// Define jail timers
func Define(vm *vm.VM, l *loop.Loop) error {
	if v, err := vm.Get("setTimeout"); err != nil {
		return err
	} else if !v.IsUndefined() {
		return nil
	}

	timeHandlers := map[string]func(call otto.FunctionCall) otto.Value{
		"setInterval":    newTimerHandler(l, true),
		"setTimeout":     newTimerHandler(l, false),
		"setImmediate":   newImmediateTimerHandler(l),
		"clearTimeout":   newClearTimeoutHandler(l),
		"clearInterval":  newClearTimeoutHandler(l),
		"clearImmediate": newClearTimeoutHandler(l),
	}

	for k, handler := range timeHandlers {
		if err := vm.Set(k, handler); err != nil {
			return err
		}
	}

	return nil
}

func getDelayWithMin(call otto.FunctionCall, interval bool) int64 {
	var minDelay = map[bool]int64{
		true:  10,
		false: 4,
	}

	delay, _ := call.Argument(1).ToInteger()
	if delay < minDelay[interval] {
		return minDelay[interval]
	}
	return delay
}

func newTimerHandler(l *loop.Loop, interval bool) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		delay := getDelayWithMin(call, interval)

		t := &timerTask{
			duration: time.Duration(delay) * time.Millisecond,
			call:     call,
			interval: interval,
		}
		// If err is non-nil, then the loop is closed and should not
		// be used anymore.
		if err := l.Add(t); err != nil {
			return otto.UndefinedValue()
		}

		t.timer = time.AfterFunc(t.duration, func() {
			l.Ready(t) // nolint: errcheck
		})

		value, newTimerErr := call.Otto.ToValue(t)
		if newTimerErr != nil {
			panic(newTimerErr)
		}

		return value
	}
}

func newImmediateTimerHandler(l *loop.Loop) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		t := &timerTask{
			duration: time.Millisecond,
			call:     call,
		}

		// If err is non-nil, then the loop is closed and should not
		// be used anymore.
		if err := l.Add(t); err != nil {
			return otto.UndefinedValue()
		}

		t.timer = time.AfterFunc(t.duration, func() {
			l.Ready(t) // nolint: errcheck
		})

		value, setImmediateErr := call.Otto.ToValue(t)
		if setImmediateErr != nil {
			panic(setImmediateErr)
		}

		return value
	}
}

func newClearTimeoutHandler(l *loop.Loop) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		v, _ := call.Argument(0).Export()
		if t, ok := v.(*timerTask); ok {
			t.stopped = true
			t.timer.Stop()
			l.Remove(t)
		}

		return otto.UndefinedValue()
	}
}
