package publish

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

var ErrRateLimited = errors.New("rate limit exceeded")

const RlnLimiterCapacity = 100
const RlnLimiterRefillInterval = 10 * time.Minute

// RlnRateLimiter is used to rate limit the outgoing messages,
// The capacity and refillInterval comes from RLN contract configuration.
type RlnRateLimiter struct {
	mu             sync.Mutex
	capacity       int
	tokens         int
	refillInterval time.Duration
	lastRefill     time.Time
}

// NewRlnPublishRateLimiter creates a new rate limiter, starts with a full capacity bucket.
func NewRlnRateLimiter(capacity int, refillInterval time.Duration) *RlnRateLimiter {
	return &RlnRateLimiter{
		capacity:       capacity,
		tokens:         capacity, // Start with a full bucket
		refillInterval: refillInterval,
		lastRefill:     time.Now(),
	}
}

// Allow checks if a token can be consumed, and refills the bucket if necessary
func (rl *RlnRateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens if the refill interval has passed
	now := time.Now()
	if now.Sub(rl.lastRefill) >= rl.refillInterval {
		rl.tokens = rl.capacity // Refill the bucket
		rl.lastRefill = now
	}

	// Check if there are tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func (rl *RlnRateLimiter) Check(ctx context.Context, logger *zap.Logger) error {
	if rl.Allow() {
		return nil
	}
	return ErrRateLimited
}
