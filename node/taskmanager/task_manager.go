package taskmanager

import (
	"sync"
)

type initFunc func(stop <-chan struct{}, stopped chan<- struct{})

type TaskManager interface {
	Add(name string, f initFunc)
	StopTasks() <-chan struct{}
}

type taskManager struct {
	sync.Mutex
	tasks    []*task
	stopping chan struct{}
}

func New() TaskManager {
	return &taskManager{
		tasks: make([]*task, 0),
	}
}

func (tm *taskManager) Add(name string, f initFunc) {
	t := newTask(name)
	tm.Lock()
	tm.tasks = append(tm.tasks, t)
	tm.Unlock()
	f(t.stopCh, t.stoppedCh)
}

func (tm *taskManager) StopTasks() <-chan struct{} {
	tm.Lock()
	if tm.stopping != nil {
		tm.Unlock()
		return tm.stopping
	}

	tm.stopping = make(chan struct{})
	tm.Unlock()

	go func() {
		var wg sync.WaitGroup

		for _, t := range tm.tasks {
			wg.Add(1)
			go func(t *task) {
				t.stop()
				t.wait()
				wg.Done()
			}(t)
		}

		wg.Wait()
		close(tm.stopping)
	}()

	return tm.stopping
}
