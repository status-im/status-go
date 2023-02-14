package server

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// timeoutManager represents a discrete encapsulation of timeout functionality.
// this struct expose 3 functions:
//  - SetTimeout
//  - StartTimeout
//  - StopTimeout
type timeoutManager struct {
	// timeout number of milliseconds the timeout operation will run before executing the `terminate` func()
	// 0 represents an inactive timeout
	timeout uint

	// exitQueue handles the cancel signal channels that circumvent timeout operations and prevent the
	// execution of any `terminate` func()
	exitQueue *exitQueueManager

	logger *zap.Logger
}

// newTimeoutManager returns a fully qualified and initialised timeoutManager
func newTimeoutManager(logger *zap.Logger) *timeoutManager {
	return &timeoutManager{
		exitQueue: &exitQueueManager{queue: []chan struct{}{}, logger: logger},
		logger:    logger,
	}
}

// SetTimeout sets the value of the timeoutManager.timeout
func (t *timeoutManager) SetTimeout(milliseconds uint) {
	t.timeout = milliseconds
}

// StartTimeout starts a timeout operation based on the set timeoutManager.timeout value
// the given terminate func() will be executed once the timeout duration has passed
func (t *timeoutManager) StartTimeout(terminate func()) {
	t.logger.Debug("fired")
	if t.timeout == 0 {
		t.logger.Debug("t.timeout == 0")
		return
	}

	t.logger.Debug("pre StopTimeout()")
	t.StopTimeout()
	t.logger.Debug("post StopTimeout()")

	t.logger.Debug("pre t.run(terminate, exit)")
	exit := make(chan struct{}, 1)
	t.exitQueue.add(exit)
	go t.run(terminate, exit)
	t.logger.Debug("post t.run(terminate, exit)")

	t.logger.Debug("end")
}

// StopTimeout terminates a timeout operation and exits gracefully
func (t *timeoutManager) StopTimeout() {
	t.logger.Debug("fired")
	if t.timeout == 0 {
		t.logger.Debug("t.timeout == 0")
		return
	}

	t.logger.Debug("pre t.exitQueue.empty()")
	t.exitQueue.empty()
	t.logger.Debug("post t.exitQueue.empty()")

	t.logger.Debug("end")
}

// run inits the main timeout run function that awaits for the exit command to be triggered or for the
// timeout duration to elapse and trigger the parameter terminate function.
func (t *timeoutManager) run(terminate func(), exit chan struct{}) {
	t.logger.Debug("run fired", zap.Uint("t.timeout", t.timeout))

	select {
	case <-exit:
		t.logger.Debug("<-exit sig")
		return
	case <-time.After(time.Duration(t.timeout) * time.Millisecond):
		terminate()
		t.logger.Debug("post terminate()")
		return
	}
}

// exitQueueManager
type exitQueueManager struct {
	logger    *zap.Logger
	queue     []chan struct{}
	queueLock sync.Mutex
}

// add handles new exit channels adding them to the exit queue
func (e *exitQueueManager) add(exit chan struct{}) {
	e.logger.Debug("fired")
	e.queueLock.Lock()
	defer e.queueLock.Unlock()
	e.logger.Debug("", zap.Int("e.queue.len()", len(e.queue)))

	e.queue = append(e.queue, exit)
	e.logger.Debug("end", zap.Int("e.queue.len()", len(e.queue)))
}

// empty sends a signal to evert exit channel in the queue and then resets the queue
func (e *exitQueueManager) empty() {
	e.logger.Debug("fired")
	e.queueLock.Lock()
	defer e.queueLock.Unlock()
	e.logger.Debug("", zap.Int("e.queue.len()", len(e.queue)))

	for i := range e.queue {
		e.queue[i] <- struct{}{}
	}

	e.queue = []chan struct{}{}
	e.logger.Debug("end", zap.Int("e.queue.len()", len(e.queue)))
}
