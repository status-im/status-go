package timers

import (
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/jail/internal/loop"
	"github.com/status-im/status-go/jail/internal/vm"
)

type timerTask struct {
	id       int64
	timer    *time.Timer
	duration time.Duration
	interval bool
	call     otto.FunctionCall
	stopped  bool
}

func (t *timerTask) SetID(id int64) { t.id = id }
func (t *timerTask) GetID() int64   { return t.id }

func (t *timerTask) Execute(vm *vm.VM, l *loop.Loop) error {
	arguments := t.getArguments()
	if _, err := vm.Call(`Function.call.call`, nil, arguments...); err != nil {
		return err
	}

	if !(t.interval && !t.stopped) {
		return nil
	}

	t.timer.Reset(t.duration)
	return l.Add(t)
}

func (t *timerTask) Cancel() {
	t.timer.Stop()
}

func (t *timerTask) getArguments() (arguments []interface{}) {
	arguments = make([]interface{}, 1)
	if len(t.call.ArgumentList) > 2 {
		tmp := t.call.ArgumentList[2:]
		arguments = make([]interface{}, 2+len(tmp))

		for i, value := range tmp {
			arguments[i+2] = value
		}
	}
	arguments[0] = t.call.ArgumentList[0]

	return
}
