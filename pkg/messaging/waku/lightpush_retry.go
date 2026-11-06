package waku

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	lightpushRateLimitMaxRetries  = 3
	lightpushRateLimitRetryBudget = 5 * time.Second
	lightpushRateLimitBaseDelay   = 250 * time.Millisecond
)

// isLightpushRateLimited adapts go-waku v0.10.3's untyped error format. Publish
// concatenates the response Info strings after "lightpush error" and exposes no
// structured status. Accept only one or more exact quota responses: other visible
// failures, including mixed responses, must not enable this retry policy.
func isLightpushRateLimited(err error) bool {
	for err != nil {
		details, ok := strings.CutPrefix(err.Error(), "lightpush error")
		if ok && details != "" && strings.ReplaceAll(details, "exceeds the rate limit", "") == "" {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// publishWithLightpushRetry keeps transient quota rejection below MVDS. send must
// reuse the prepared envelope and apply the shared outgoing publication limiter
// on every call. The initial send retains its existing timeout; the additional
// retry budget starts only after an explicit rate-limit response.
//
// attempts counts publisher calls, including a call canceled while waiting for
// its outgoing token. onRetry announces a scheduled call, which may subsequently
// be canceled before it starts.
func publishWithLightpushRetry(
	ctx context.Context,
	send func(context.Context) error,
	onRetry func(attempt int, delay time.Duration, cause error),
) (attempts int, err error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	attempts = 1
	err = send(ctx)
	if !isLightpushRateLimited(err) {
		return attempts, err
	}

	retryCtx, cancel := context.WithTimeout(ctx, lightpushRateLimitRetryBudget)
	defer cancel()

	for retry := 0; retry < lightpushRateLimitMaxRetries; retry++ {
		if contextErr := retryCtx.Err(); contextErr != nil {
			return attempts, errors.Join(err, contextErr)
		}

		// 250–375 ms, 500–750 ms, then 1–1.5 s. The shared sender limiter
		// also paces retries among all other data and protocol publications.
		delay := lightpushRateLimitBaseDelay << retry
		jitter, jitterErr := rand.Int(rand.Reader, big.NewInt(int64(delay/2)+1))
		if jitterErr != nil {
			return attempts, errors.Join(err, fmt.Errorf("generate lightpush retry jitter: %w", jitterErr))
		}
		delay += time.Duration(jitter.Int64())
		if onRetry != nil {
			onRetry(attempts+1, delay, err)
		}
		timer := time.NewTimer(delay)
		select {
		case <-retryCtx.Done():
			timer.Stop()
			return attempts, errors.Join(err, retryCtx.Err())
		case <-timer.C:
		}
		if contextErr := retryCtx.Err(); contextErr != nil {
			return attempts, errors.Join(err, contextErr)
		}

		attempts++
		err = send(retryCtx)
		if err == nil {
			return attempts, nil
		}
		if contextErr := retryCtx.Err(); contextErr != nil {
			return attempts, errors.Join(err, contextErr)
		}
		if !isLightpushRateLimited(err) {
			return attempts, err
		}
	}

	return attempts, fmt.Errorf("lightpush rate limit persisted after %d publication attempts: %w", attempts, err)
}
