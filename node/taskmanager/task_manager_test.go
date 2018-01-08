package taskmanager

import (
	"fmt"
	"testing"
)

func newTestTask(t *testing.T, name string, notifyTest chan struct{}) initFunc {
	return func(stop <-chan struct{}, stopped chan<- struct{}) {
		go func() {
			<-stop
			notifyTest <- struct{}{}
			t.Logf("%s received stop signal", name)
			stopped <- struct{}{}
			t.Logf("%s ended", name)
		}()
	}
}

func TestTaskManager(t *testing.T) {
	tasksCount := 3
	notifyTest := make(chan struct{}, tasksCount)
	notificationsReceived := make(chan struct{})

	tm := New()
	for i := 0; i < tasksCount; i++ {
		name := fmt.Sprintf("task-%d", i)
		tm.Add(name, newTestTask(t, name, notifyTest))
	}

	go func() {
		for i := 0; i < tasksCount; i++ {
			<-notifyTest
		}
		notificationsReceived <- struct{}{}
	}()

	<-tm.StopTasks()
	<-notificationsReceived
}
