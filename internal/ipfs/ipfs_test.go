package ipfs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	messaginglifecycle "github.com/status-im/status-go/pkg/messaging/lifecycle"
)

func TestTaskDispatcherPausesAndResumesByLifecycle(t *testing.T) {
	messaginglifecycle.SetPausedBackground(true)
	defer messaginglifecycle.SetPausedBackground(false)

	d := &Downloader{
		inputTaskChan:   make(chan taskRequest, 1),
		rateLimiterChan: make(chan taskRequest, 1),
		quit:            make(chan struct{}, 1),
	}

	go d.taskDispatcher()
	defer close(d.quit)

	req := taskRequest{cid: "cid-1", doneChan: make(chan taskResponse, 1)}
	d.inputTaskChan <- req

	select {
	case <-d.rateLimiterChan:
		t.Fatal("task was dispatched while paused")
	case <-time.After(450 * time.Millisecond):
	}

	messaginglifecycle.SetPausedBackground(false)

	select {
	case got := <-d.rateLimiterChan:
		require.Equal(t, req.cid, got.cid)
	case <-time.After(2 * time.Second):
		t.Fatal("task was not dispatched after resume")
	}
}
