package waku

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishWithLightpushRetryRetriesRateLimitedPublication(t *testing.T) {
	calls := 0
	var retryAttempt int
	var retryDelay time.Duration
	var retryCause error
	quotaErr := errors.New("lightpush errorexceeds the rate limit")

	attempts, err := publishWithLightpushRetry(
		context.Background(),
		func(context.Context) error {
			calls++
			if calls == 1 {
				return quotaErr
			}
			return nil
		},
		func(attempt int, delay time.Duration, cause error) {
			retryAttempt = attempt
			retryDelay = delay
			retryCause = cause
		},
	)

	require.NoError(t, err)
	require.Equal(t, 2, attempts)
	require.Equal(t, 2, calls)
	require.Equal(t, 2, retryAttempt)
	require.GreaterOrEqual(t, retryDelay, lightpushRateLimitBaseDelay)
	require.LessOrEqual(t, retryDelay, lightpushRateLimitBaseDelay*3/2)
	require.ErrorIs(t, retryCause, quotaErr)
}

func TestPublishWithLightpushRetryDoesNotRetryOtherErrors(t *testing.T) {
	expectedErr := errors.New("lightpush errorCould not publish message: invalid request")
	calls := 0

	attempts, err := publishWithLightpushRetry(
		context.Background(),
		func(context.Context) error {
			calls++
			return expectedErr
		},
		nil,
	)

	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, 1, attempts)
	require.Equal(t, 1, calls)
	require.False(t, isLightpushRateLimited(err))
}

func TestPublishWithLightpushRetryStopsAfterRetryLimit(t *testing.T) {
	calls := 0
	attempts, err := publishWithLightpushRetry(
		context.Background(),
		func(context.Context) error {
			calls++
			return errors.New("lightpush errorexceeds the rate limit")
		},
		nil,
	)

	require.Error(t, err)
	require.True(t, isLightpushRateLimited(err))
	require.Contains(t, err.Error(), "after 4 publication attempts")
	require.Equal(t, lightpushRateLimitMaxRetries+1, attempts)
	require.Equal(t, lightpushRateLimitMaxRetries+1, calls)
}
