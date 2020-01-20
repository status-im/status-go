// +build nimbus

package nimbusbridge

import (
	"syscall"
)

// RoutineQueue provides a mechanism for marshalling function calls
// so that they are run in a specific thread (the thread where
// RoutineQueue is initialized).
type RoutineQueue struct {
	tid    int
	events chan event
}

type callReturn struct {
	value interface{}
	err   error
}

// NewRoutineQueue returns a new RoutineQueue object.
func NewRoutineQueue() *RoutineQueue {
	q := &RoutineQueue{
		tid:    syscall.Gettid(),
		events: make(chan event, 20),
	}

	return q
}

// event represents an event triggered by the user.
type event struct {
	f    func(chan<- callReturn)
	done chan callReturn
}

func (q *RoutineQueue) HandleEvent() {
	if syscall.Gettid() != q.tid {
		panic("HandleEvent called from wrong thread")
	}

	select {
	case ev := <-q.events:
		ev.f(ev.done)
	default:
		return
	}
}

// Send executes the passed function. This method can be called safely from a
// goroutine in order to execute a Nimbus function. It is important to note that the
// passed function won't be executed immediately, instead it will be added to
// the user events queue.
func (q *RoutineQueue) Send(f func(chan<- callReturn)) callReturn {
	ev := event{f: f, done: make(chan callReturn, 1)}
	defer close(ev.done)
	if syscall.Gettid() == q.tid {
		f(ev.done)
		return <-ev.done
	}
	q.events <- ev
	return <-ev.done
}
