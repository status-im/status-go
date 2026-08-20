package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmit(t *testing.T) {
	// Signal ids
	signal1ID := 10
	signal2ID := 15

	m := NewSubscriptionManager()
	chn := m.Subscribe()
	require.NotNil(t, chn)

	received := make(chan int, 3) // buffer for the 3 emits

	go func() {
		for sig := range chn {
			received <- sig.Source()
		}
	}()

	m.Emit(context.Background(), signal1ID)
	m.Emit(context.Background(), signal2ID)

	got := make(map[int]int)
	timeout := time.After(1 * time.Second)

	// Verify signal are received once
	for i := 0; i < 2; i++ {
		select {
		case src := <-received:
			got[src]++
			require.Equal(t, got[src], 1, "received signal from source %d more than once", src)
		case <-timeout:
			t.Fatal("timeout waiting for signals")
		}
	}

	// Verify all expected signals were received exactly once
	for _, expected := range []int{signal1ID, signal2ID} {
		require.Equal(t, 1, got[expected], "expected signal from source %d once, got %d", expected, got[expected])
	}
	require.Equal(t, 2, len(got))
}
