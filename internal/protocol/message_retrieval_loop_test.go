package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunRetrieveMessagesLoopDoesNotStarveUnderContinuousTraffic(t *testing.T) {
	matched := make(chan struct{}, 1)
	cancel := make(chan struct{})
	quit := make(chan struct{})
	processed := make(chan time.Time, 1)
	started := time.Now()
	stopTraffic := make(chan struct{})

	go runRetrieveMessagesLoopWithInterval(matched, cancel, quit, func() {
		processed <- time.Now()
	}, 200*time.Millisecond)
	t.Cleanup(func() { close(cancel) })

	matched <- struct{}{}
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopTraffic:
				return
			case <-ticker.C:
				select {
				case matched <- struct{}{}:
				default:
				}
			}
		}
	}()
	t.Cleanup(func() { close(stopTraffic) })

	select {
	case processedAt := <-processed:
		require.Less(t, processedAt.Sub(started), 600*time.Millisecond)
	case <-time.After(700 * time.Millisecond):
		t.Fatal("continuous matching traffic postponed message processing")
	}
}

func TestRunRetrieveMessagesLoopProcessesMatchQueuedDuringProcessing(t *testing.T) {
	matched := make(chan struct{}, 1)
	cancel := make(chan struct{})
	quit := make(chan struct{})
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	processed := make(chan struct{}, 2)
	processCalls := 0

	go runRetrieveMessagesLoopWithInterval(matched, cancel, quit, func() {
		processCalls++
		if processCalls == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		processed <- struct{}{}
	}, 40*time.Millisecond)
	t.Cleanup(func() { close(cancel) })

	matched <- struct{}{}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first batch was not processed")
	}

	matched <- struct{}{}
	close(releaseFirst)

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("first batch did not finish")
	}
	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("match received during processing was lost")
	}
}
