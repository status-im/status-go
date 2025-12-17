package healthmanager

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionManager(t *testing.T) {
	// Create a new SubscriptionManager
	sm := NewSubscriptionManager()

	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Test adding subscribers and receiving notifications
	subscriberCount := 5
	notificationReceived := make([]bool, subscriberCount)

	for i := 0; i < subscriberCount; i++ {
		ch := sm.Subscribe()
		idx := i // Copy the value of i to use inside the goroutine

		wg.Add(1)
		go func(ch chan struct{}, idx int) {
			defer wg.Done()
			select {
			case <-ch:
				notificationReceived[idx] = true
			case <-time.After(1 * time.Second):
				// Timeout waiting for notification
			}
		}(ch, idx)
	}

	// Emit a notification to all subscribers
	sm.Emit(ctx)

	// Wait for all goroutines to finish
	wg.Wait()

	// Verify that all subscribers received the notification
	for i, received := range notificationReceived {
		require.Truef(t, received, "Subscriber %d did not receive notification", i)
	}

	// Test that notifications are not sent to closed channels
	chClosed := sm.Subscribe()
	// Ensure that unsubscribe handles already closed channels properly
	sm.Unsubscribe(chClosed)

	// ensure subscription is removed
	_, exists := sm.subscribers[chClosed]
	require.False(t, exists, "Subscription was not removed")

	sm.Unsubscribe(chClosed)
	// Emit a notification
	sm.Emit(ctx)

	// If no panic occurs, the test is successful
}

func TestSubscriptionManager_MultipleEmitsCollapse(t *testing.T) {
	sm := NewSubscriptionManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := sm.Subscribe()
	defer sm.Unsubscribe(ch)

	var received int
	var mu sync.Mutex

	go func() {
		for range ch {
			mu.Lock()
			received++
			mu.Unlock()
		}
	}()

	// Emit multiple notifications
	sm.Emit(ctx)
	sm.Emit(ctx)
	sm.Emit(ctx)

	// Allow some time for the goroutine to process notifications
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	require.Equalf(t, 1, received, "Expected 1 notifications, but received %d", received)
	mu.Unlock()
}
