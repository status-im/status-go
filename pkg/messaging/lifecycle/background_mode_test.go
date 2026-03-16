package lifecycle

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscribePausedBackgroundGetsInitialState(t *testing.T) {
	SetPausedBackground(true)
	sub := SubscribePausedBackground()
	defer sub.Unsubscribe()

	require.Equal(t, true, mustReadPausedState(t, sub.C()))

	SetPausedBackground(false)
}

func TestSetPausedBackgroundPublishesTransitionsOnly(t *testing.T) {
	SetPausedBackground(false)
	sub := SubscribePausedBackground()
	defer sub.Unsubscribe()

	require.Equal(t, false, mustReadPausedState(t, sub.C()))

	SetPausedBackground(true)
	require.Equal(t, true, mustReadPausedState(t, sub.C()))

	SetPausedBackground(true)
	mustNotReadPausedState(t, sub.C())

	SetPausedBackground(false)
	require.Equal(t, false, mustReadPausedState(t, sub.C()))
}

func TestUnsubscribeClosesChannelAndIsIdempotent(t *testing.T) {
	sub := SubscribePausedBackground()
	_ = mustReadPausedState(t, sub.C())

	sub.Unsubscribe()
	sub.Unsubscribe()

	_, ok := <-sub.C()
	require.False(t, ok)
}

func TestSetPausedBackgroundDoesNotBlockSlowSubscriber(t *testing.T) {
	SetPausedBackground(false)
	sub := SubscribePausedBackground()
	defer sub.Unsubscribe()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			SetPausedBackground(i%2 == 0)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SetPausedBackground blocked with slow subscriber")
	}
}

func TestConcurrentSubscribeUnsubscribeAndSet(t *testing.T) {
	SetPausedBackground(false)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sub := SubscribePausedBackground()
			_ = mustReadPausedState(t, sub.C())
			SetPausedBackground(i%2 == 0)
			sub.Unsubscribe()
		}(i)
	}

	wg.Wait()
	SetPausedBackground(false)
}

func mustReadPausedState(t *testing.T, ch <-chan bool) bool {
	t.Helper()
	select {
	case paused, ok := <-ch:
		require.True(t, ok)
		return paused
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle state")
		return false
	}
}

func mustNotReadPausedState(t *testing.T, ch <-chan bool) {
	t.Helper()
	select {
	case paused := <-ch:
		t.Fatalf("unexpected lifecycle state event: %v", paused)
	case <-time.After(50 * time.Millisecond):
	}
}
