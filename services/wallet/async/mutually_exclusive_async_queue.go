package async

import (
	"context"
	"sort"
	"sync"
)

type TaskPriority int

type CancelableTask func(context.Context)

type taskEntry struct {
	task     CancelableTask
	priority TaskPriority
}

// PriorityBasedAsyncQueue is a task queue that allows only one task to be executed at a time
// It is useful for long-running interruptible tasks that check the following conditions:
// - Only one task should be executed at a time and that should always be the highest priority task.
// - A higher queued task should interrupt a lower priority task until finished
// - A previously interrupted task should start again after the higher priority task is finished
// - Same priority tasks should be executed in the order they were queued
// - A task should be able to cancel itself
type PriorityBasedAsyncQueue struct {
	// queue is a list of sorted tasks based on priority, where the first element is the currently running task
	queue          []*taskEntry
	mutex          sync.Mutex
	rootContext    context.Context
	currentContext context.Context
	cancelFn       context.CancelFunc
}

func NewPriorityBasedAsyncQueue(ctx context.Context) *PriorityBasedAsyncQueue {
	q := &PriorityBasedAsyncQueue{}
	q.queue = make([]*taskEntry, 0, 2)
	q.rootContext = context.Background()
	return q
}

func insertTask(queue []*taskEntry, task *taskEntry) []*taskEntry {
	for i, t := range queue {
		if t.priority < task.priority {
			return append(queue[:i], append([]*taskEntry{task}, queue[i:]...)...)
		}
	}
	return append(queue, task)
}

func (q *PriorityBasedAsyncQueue) RunTask(task CancelableTask, priority TaskPriority) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	newTask := &taskEntry{task: task, priority: priority}
	if len(q.queue) == 0 {
		q.queue = insertTask(q.queue, newTask)
		q.currentContext, q.cancelFn = context.WithCancel(q.rootContext)
		q.run()
	} else {
		q.queue = insertTask(q.queue, newTask)
		if q.queue[0] == newTask {
			q.cancelFn()
		}
	}
}

// Execute all tasks in the queue, one at a time, in order of priority
func (q *PriorityBasedAsyncQueue) run() {
	go func() {
		finish := false
		for !finish {
			q.mutex.Lock()
			var task *taskEntry
			if len(q.queue) > 0 {
				task = q.queue[0]
			} else {
				finish = true
			}
			q.mutex.Unlock()

			if !finish {
				task.task(q.currentContext)

				q.mutex.Lock()
				select {
				case <-q.rootContext.Done():
					// The root context was canceled, so we abort everything
					finish = true
				case <-q.currentContext.Done():
					q.currentContext, q.cancelFn = context.WithCancel(q.rootContext)
					// The task was canceled, so we need to re-enqueue it
				default:
					// The task completed, so we can safely remove it from the queue
					idx := sort.Search(len(q.queue), func(i int) bool {
						return q.queue[i].priority <= task.priority
					})
					// We know that the task is in the queue and all enqueued having the same priority were appended, so we can safely remove it
					q.queue = append(q.queue[:idx], q.queue[idx+1:]...)
				}

				q.mutex.Unlock()
			}
		}
	}()
}
