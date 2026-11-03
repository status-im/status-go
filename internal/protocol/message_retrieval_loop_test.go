package protocol

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/panics"
)

func TestRunRetrieveMessagesLoopProcessesDuringContinuousMatches(t *testing.T) {
	matched := make(chan struct{}, 1)
	cancel := make(chan struct{})
	quit := make(chan struct{})
	done := make(chan struct{})
	processedDuringTraffic := make(chan struct{}, 1)
	stopProducer := make(chan struct{})
	producerDone := make(chan struct{})

	var lastMatch atomic.Int64
	go func() {
		defer panics.LogOnPanic()
		defer close(producerDone)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()

		notify := func() {
			lastMatch.Store(time.Now().UnixNano())
			select {
			case matched <- struct{}{}:
			default:
			}
		}
		notify()

		for {
			select {
			case <-stopProducer:
				return
			case <-ticker.C:
				notify()
			}
		}
	}()

	go func() {
		defer panics.LogOnPanic()
		defer close(done)
		runRetrieveMessagesLoop(matched, cancel, quit, func() {
			last := time.Unix(0, lastMatch.Load())
			if time.Since(last) < 100*time.Millisecond {
				select {
				case processedDuringTraffic <- struct{}{}:
				default:
				}
			}
		})
	}()

	defer func() {
		close(stopProducer)
		close(cancel)
		<-producerDone
		<-done
	}()

	select {
	case <-processedDuringTraffic:
	case <-time.After(3 * retrieveMessagesBatchInterval):
		require.FailNow(t, "receive processing did not start while matches kept arriving")
	}
}

func TestRunRetrieveMessagesLoopSerializesProcessingAndKeepsPendingMatch(t *testing.T) {
	matched := make(chan struct{}, 1)
	cancel := make(chan struct{})
	quit := make(chan struct{})
	done := make(chan struct{})
	firstProcessStarted := make(chan struct{})
	releaseFirstProcess := make(chan struct{})
	secondProcessStarted := make(chan struct{})

	var calls atomic.Int32
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			close(releaseFirstProcess)
		})
	}
	go func() {
		defer panics.LogOnPanic()
		defer close(done)
		runRetrieveMessagesLoop(matched, cancel, quit, func() {
			call := calls.Add(1)
			if call == 1 {
				close(firstProcessStarted)
				<-releaseFirstProcess
				return
			}
			if call == 2 {
				close(secondProcessStarted)
			}
		})
	}()

	defer func() {
		release()
		close(cancel)
		<-done
	}()

	matched <- struct{}{}
	select {
	case <-firstProcessStarted:
	case <-time.After(3 * retrieveMessagesBatchInterval):
		require.FailNow(t, "receive processing did not start")
	}

	matched <- struct{}{}
	select {
	case <-secondProcessStarted:
		require.FailNow(t, "receive processing ran concurrently")
	case <-time.After(100 * time.Millisecond):
	}

	release()
	select {
	case <-secondProcessStarted:
	case <-time.After(3 * retrieveMessagesBatchInterval):
		require.FailNow(t, "match received during processing did not schedule another drain")
	}
}
