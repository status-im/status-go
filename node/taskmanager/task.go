package taskmanager

type task struct {
	name      string
	stopCh    chan struct{}
	stoppedCh chan struct{}
}

func newTask(name string) *task {
	return &task{
		name:      name,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

func (t *task) stop() {
	close(t.stopCh)
}

func (t *task) wait() {
	<-t.stoppedCh
}
