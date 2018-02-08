package loop

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/status-im/status-go/geth/jail/internal/vm"
)

// ErrClosed represents the error returned when we try to add or ready
// a task on a closed loop.
var ErrClosed = errors.New("The loop is closed and no longer accepting tasks")

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
	Execute(vm *vm.VM, l *Loop) error
	Cancel()
}

// Loop encapsulates the event loop's state. This includes the vm on which the
// loop operates, a monotonically incrementing event id, a map of tasks that
// aren't ready yet, keyed by their ID, a channel of tasks that are ready
// to finalise on the VM, and a boolean that indicates if the loop is still
// accepting tasks. The channel holding the tasks pending finalising can be
// buffered or unbuffered.
//
// Warning: id must be the first field in this struct as it's accessed atomically.
// Otherwise, on ARM and x86-32 it will panic.
// More information: https://golang.org/pkg/sync/atomic/#pkg-note-BUG.
type Loop struct {
	id        int64
	vm        *vm.VM
	lock      sync.RWMutex
	tasks     map[int64]Task
	ready     chan Task
	closeChan chan struct{}
}

// New creates a new Loop with an unbuffered ready queue on a specific VM.
func New(vm *vm.VM) *Loop {
	return NewWithBacklog(vm, 0)
}

// NewWithBacklog creates a new Loop on a specific VM, giving it a buffered
// queue, the capacity of which being specified by the backlog argument.
func NewWithBacklog(vm *vm.VM, backlog int) *Loop {
	return &Loop{
		vm:        vm,
		tasks:     make(map[int64]Task),
		ready:     make(chan Task, backlog),
		closeChan: make(chan struct{}),
	}
}

// close the loop so that it no longer accepts tasks.
func (l *Loop) close() {
	close(l.closeChan)
}

// VM gets the JavaScript interpreter associated with the loop.
func (l *Loop) VM() *vm.VM {
	return l.vm
}

// Add puts a task into the loop. This signals to the loop that this task is
// doing something outside of the JavaScript environment, and that at some
// point, it will become ready for finalising.
func (l *Loop) Add(t Task) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	select {
	case <-l.closeChan:
		return ErrClosed
	default:
	}
	t.SetID(atomic.AddInt64(&l.id, 1))
	l.tasks[t.GetID()] = t
	return nil
}

// Remove takes a task out of the loop. This should not be called if a task
// has already become ready for finalising. Warranty void if constraint is
// broken.
func (l *Loop) Remove(t Task) {
	l.remove(t)
	go l.Ready(nil) // nolint: errcheck
}

func (l *Loop) remove(t Task) {
	l.removeByID(t.GetID())
}

func (l *Loop) removeByID(id int64) {
	l.lock.Lock()
	delete(l.tasks, id)
	l.lock.Unlock()
}

func (l *Loop) removeAll() {
	l.lock.Lock()
	for _, t := range l.tasks {
		t.Cancel()
	}
	l.tasks = make(map[int64]Task)
	l.lock.Unlock()
}

// Ready signals to the loop that a task is ready to be finalised. This might
// block if the "ready channel" in the loop is at capacity.
func (l *Loop) Ready(t Task) error {
	select {
	case <-l.closeChan:
		t.Cancel()
		return ErrClosed
	case l.ready <- t:
		return nil
	}
}

// AddAndExecute combines Add and Ready for immediate execution.
func (l *Loop) AddAndExecute(t Task) error {
	if err := l.Add(t); err != nil {
		return err
	}
	return l.Ready(t)
}

// Eval executes some code in the VM associated with the loop and returns an
// error if that execution fails.
func (l *Loop) Eval(s interface{}) error {
	_, err := l.vm.Run(s)
	return err
}

func (l *Loop) processTask(t Task) error {
	id := t.GetID()

	if err := t.Execute(l.vm, l); err != nil {
		l.lock.RLock()
		t.Cancel()
		l.lock.RUnlock()

		return err
	}

	l.removeByID(id)

	return nil
}

// Run handles the task scheduling and finalisation.
// It runs infinitely waiting for new tasks.
func (l *Loop) Run(ctx context.Context) error {
	defer l.close()
	defer l.removeAll()

	for {
		select {
		case t := <-l.ready:
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if t == nil {
				continue
			}

			err := l.processTask(t)
			if err != nil {
				// TODO(divan): do we need to report
				// errors up to the caller?
				// Ignoring for now.
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
