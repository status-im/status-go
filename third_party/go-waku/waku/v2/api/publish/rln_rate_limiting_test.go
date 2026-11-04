package publish

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestRlnRateLimit(t *testing.T) {
	updateCh := make(chan RlnRateLimitState, 10)
	refillTime := time.Now()
	capacity := 3
	state := RlnRateLimitState{
		RemainingTokens: capacity,
		LastRefill:      refillTime,
	}
	r := NewRlnRateLimiter(capacity, 5*time.Second, state, updateCh)
	l := utils.Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sleepDuration := 6 * time.Second
	var mu sync.Mutex
	go func(ctx context.Context, ch chan RlnRateLimitState) {
		usedToken := 0
		for {
			select {
			case update := <-ch:
				mu.Lock()
				if update.LastRefill != refillTime {
					usedToken = 0
					require.WithinDuration(t, refillTime.Add(sleepDuration), update.LastRefill, time.Second, "Last refill timestamp is incorrect")
					require.Equal(t, update.RemainingTokens, capacity)
					continue
				}
				usedToken++
				require.Equal(t, update.RemainingTokens, capacity-usedToken)
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}(ctx, updateCh)

	for i := 0; i < capacity; i++ {
		require.NoError(t, r.Check(context.Background(), l))
	}
	require.ErrorIs(t, r.Check(context.Background(), l), ErrRateLimited)

	time.Sleep(sleepDuration)

	for i := 0; i < 3; i++ {
		require.NoError(t, r.Check(context.Background(), l))
	}
	require.ErrorIs(t, r.Check(context.Background(), l), ErrRateLimited)

	// wait for goroutine to finish
	time.Sleep(time.Second)
}
