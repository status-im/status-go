package loop

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/status-im/status-go/geth/jail/internal/vm"
)

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

// state encapsulates the loop's open and closedness. This is a boolean variable
// indicating whether the loop is open for new tasks and a lock to control
// access to this boolean.
type state struct {
	accept bool
	mu     sync.RWMutex
}

// isAccepting indicates whether the loop is currently accepting tasks.
func (s *state) isAccepting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accept
}

// close changes the loop's state to not accept new tasks.
func (s *state) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accept = false
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
	id    int64
	vm    *vm.VM
	lock  sync.RWMutex
	tasks map[int64]Task
	ready chan Task
	state
}

// New creates a new Loop with an unbuffered ready queue on a specific VM.
func New(vm *vm.VM) *Loop {
	return NewWithBacklog(vm, 0)
}

// NewWithBacklog creates a new Loop on a specific VM, giving it a buffered
// queue, the capacity of which being specified by the backlog argument.
func NewWithBacklog(vm *vm.VM, backlog int) *Loop {
	return &Loop{
		vm:    vm,
		tasks: make(map[int64]Task),
		ready: make(chan Task, backlog),
		state: state{
			accept: true,
		},
	}
}

// close the loop so that it no longer accepts tasks.
func (l *Loop) close() {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.isAccepting() {
		l.state.close()
		close(l.ready)
	}
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
	if !l.isAccepting() {
		return errors.New("The loop is closed and no longer accepting tasks")
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
	l.lock.Lock()
	defer l.lock.Unlock()
	if !l.isAccepting() {
		return errors.New("The loop is closed and no longer accepting tasks")
	}
	l.ready <- t
	return nil
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
	for {
		select {
		case t := <-l.ready:
			if ctx.Err() != nil {
				l.removeAll()
				return ctx.Err()
			}

			if t == nil {
				continue
			}

			err := l.processTask(t)
			if err != nil {
				// TODO(divan): do we need to report
				// errors up to the caller?
				// Ignoring for now, as loop
				// should keep running.
				continue
			}
		case <-ctx.Done():
			l.removeAll()
			return ctx.Err()
		}
	}
}
