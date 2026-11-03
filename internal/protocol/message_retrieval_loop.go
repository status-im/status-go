package protocol

import "time"

// retrieveMessagesBatchInterval coalesces matched envelopes into a fixed window
// starting at the first notification. Later arrivals do not extend the window,
// so sustained traffic cannot postpone processing until the stream goes quiet.
const retrieveMessagesBatchInterval = time.Second

// runRetrieveMessagesLoop serializes drains of the receive buffer. A coalesced
// notification that arrives during process remains available to schedule the
// next batch once that call returns. No polling runs while the receiver is idle.
func runRetrieveMessagesLoop(matched <-chan struct{}, cancel <-chan struct{}, quit <-chan struct{}, process func()) {
	var batchTimer *time.Timer
	var batchCh <-chan time.Time
	defer func() {
		if batchTimer != nil {
			batchTimer.Stop()
		}
	}()

	for {
		select {
		case _, ok := <-matched:
			if !ok {
				return
			}
			if batchCh != nil {
				continue // Already scheduled; keep the original processing deadline.
			}
			if batchTimer == nil {
				batchTimer = time.NewTimer(retrieveMessagesBatchInterval)
			} else {
				// The previous timer value was consumed before batchCh was cleared.
				batchTimer.Reset(retrieveMessagesBatchInterval)
			}
			batchCh = batchTimer.C

		case <-batchCh:
			batchCh = nil
			process()

		case <-cancel:
			return
		case <-quit:
			return
		}
	}
}
