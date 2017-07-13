package loop // import "fknsrs.biz/p/ottoext/loop"

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/robertkrimen/otto"
)

func formatTask(t Task) string {
	if t == nil {
		return "<nil>"
	}

	return fmt.Sprintf("<%T> %d", t, t.GetID())
}

// Task represents something that the event loop can schedule and run.
//
// Task describes two operations that will almost always be boilerplate,
// SetID and GetID. They exist so that the event loop can identify tasks
// after they're added.
//
// Execute is called when a task has been pulled from the "ready" queue.
//
// Cancel is called when a task is removed from the loop without being
// finalised.
type Task interface {
	SetID(id int64)
	GetID() int64
	Execute(vm *otto.Otto, l *Loop) error
	Cancel()
}

// Loop encapsulates the event loop's state. This includes the vm on which the
// loop operates, a monotonically incrementing event id, a map of tasks that
// aren't ready yet, keyed by their ID, and a channel of tasks that are ready
// to finalise on the VM. The channel holding the tasks pending finalising can
// be buffered or unbuffered.
type Loop struct {
	vm     *otto.Otto
	id     int64
	lock   sync.RWMutex
	tasks  map[int64]Task
	ready  chan Task
	closed bool
}

// New creates a new Loop with an unbuffered ready queue on a specific VM.
func New(vm *otto.Otto) *Loop {
	return NewWithBacklog(vm, 0)
}

// NewWithBacklog creates a new Loop on a specific VM, giving it a buffered
// queue, the capacity of which being specified by the backlog argument.
func NewWithBacklog(vm *otto.Otto, backlog int) *Loop {
	return &Loop{
		vm:    vm,
		tasks: make(map[int64]Task),
		ready: make(chan Task, backlog),
	}
}

// VM gets the JavaScript interpreter associated with the loop. This will be
// some kind of Otto object, but it's wrapped in an interface so the
// `ottoext` library can work with forks/extensions of otto.
func (l *Loop) VM() *otto.Otto {
	return l.vm
}

// Add puts a task into the loop. This signals to the loop that this task is
// doing something outside of the JavaScript environment, and that at some
// point, it will become ready for finalising.
func (l *Loop) Add(t Task) {
	l.lock.Lock()
	t.SetID(atomic.AddInt64(&l.id, 1))
	l.tasks[t.GetID()] = t
	l.lock.Unlock()
}

// Remove takes a task out of the loop. This should not be called if a task
// has already become ready for finalising. Warranty void if constraint is
// broken.
func (l *Loop) Remove(t Task) {
	l.remove(t)
	go l.Ready(nil)
}

func (l *Loop) remove(t Task) {
	l.removeByID(t.GetID())
}

func (l *Loop) removeByID(id int64) {
	l.lock.Lock()
	delete(l.tasks, id)
	l.lock.Unlock()
}

// Ready signals to the loop that a task is ready to be finalised. This might
// block if the "ready channel" in the loop is at capacity.
func (l *Loop) Ready(t Task) {
	if l.closed {
		return
	}

	l.ready <- t
}

// EvalAndRun is a combination of Eval and Run. Creatively named.
func (l *Loop) EvalAndRun(s interface{}) error {
	if err := l.Eval(s); err != nil {
		return err
	}

	return l.Run()
}

// Eval executes some code in the VM associated with the loop and returns an
// error if that execution fails.
func (l *Loop) Eval(s interface{}) error {
	if _, err := l.vm.Run(s); err != nil {
		return err
	}

	return nil
}

func (l *Loop) processTask(t Task) error {
	id := t.GetID()

	if err := t.Execute(l.vm, l); err != nil {
		l.lock.RLock()
		for _, t := range l.tasks {
			t.Cancel()
		}
		l.lock.RUnlock()

		return err
	}

	l.removeByID(id)

	return nil
}

// Run handles the task scheduling and finalisation. It will block until
// there's no work left to do, or an error occurs.
func (l *Loop) Run() error {
	for {
		l.lock.Lock()
		if len(l.tasks) == 0 {
			// prevent any more tasks entering the ready channel
			l.closed = true

			l.lock.Unlock()

			break
		}
		l.lock.Unlock()

		t := <-l.ready

		if t != nil {
			if err := l.processTask(t); err != nil {
				return err
			}
		}
	}

	// drain ready channel of any existing tasks
outer:
	for {
		select {
		case t := <-l.ready:
			if t != nil {
				if err := l.processTask(t); err != nil {
					return err
				}
			}
		default:
			break outer
		}
	}

	close(l.ready)

	return nil
}
