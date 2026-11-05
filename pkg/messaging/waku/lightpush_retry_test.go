package waku

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishWithLightpushRetry(t *testing.T) {
	rateLimited := errors.New("lightpush errorexceeds the rate limit")
	otherFailure := errors.New("lightpush errorpeer unavailable")

	t.Run("retries rate limit and succeeds", func(t *testing.T) {
		calls := 0
		var retryAttempts []int
		attempts, err := publishWithLightpushRetry(context.Background(), func(context.Context) error {
			calls++
			if calls == 1 {
				return rateLimited
			}
			return nil
		}, func(attempt int, delay time.Duration, cause error) {
			retryAttempts = append(retryAttempts, attempt)
			require.GreaterOrEqual(t, delay, lightpushRateLimitBaseDelay)
			require.ErrorIs(t, cause, rateLimited)
		})

		require.NoError(t, err)
		require.Equal(t, 2, attempts)
		require.Equal(t, 2, calls)
		require.Equal(t, []int{2}, retryAttempts)
	})

	t.Run("does not retry unrelated errors", func(t *testing.T) {
		calls := 0
		attempts, err := publishWithLightpushRetry(context.Background(), func(context.Context) error {
			calls++
			return otherFailure
		}, nil)

		require.ErrorIs(t, err, otherFailure)
		require.Equal(t, 1, attempts)
		require.Equal(t, 1, calls)
	})

	t.Run("does not retry mixed quota and non-quota failures", func(t *testing.T) {
		mixedFailure := errors.New("lightpush errorexceeds the rate limitpeer unavailable")
		calls := 0
		attempts, err := publishWithLightpushRetry(context.Background(), func(context.Context) error {
			calls++
			return mixedFailure
		}, nil)

		require.ErrorIs(t, err, mixedFailure)
		require.Equal(t, 1, attempts)
		require.Equal(t, 1, calls)
	})

	t.Run("stops after the configured retry count", func(t *testing.T) {
		calls := 0
		attempts, err := publishWithLightpushRetry(context.Background(), func(context.Context) error {
			calls++
			return rateLimited
		}, nil)

		require.Error(t, err)
		require.ErrorContains(t, err, "rate limit persisted")
		require.ErrorIs(t, err, rateLimited)
		require.Equal(t, lightpushRateLimitMaxRetries+1, attempts)
		require.Equal(t, lightpushRateLimitMaxRetries+1, calls)
	})

	t.Run("cancellation stops a scheduled retry", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		attempts, err := publishWithLightpushRetry(ctx, func(context.Context) error {
			calls++
			return rateLimited
		}, func(_ int, _ time.Duration, _ error) {
			cancel()
		})

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, attempts)
		require.Equal(t, 1, calls)
	})
}
