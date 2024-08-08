package publish

import (
	"context"
	"errors"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// PublishRateLimiter is used to decorate publish functions to limit the
// number of messages per second that can be published
type PublishRateLimiter struct {
	limiter *rate.Limiter
}

// NewPublishRateLimiter will create a new instance of PublishRateLimiter.
// You can specify an rate.Inf value to in practice ignore the rate limiting
func NewPublishRateLimiter(r rate.Limit, b int) *PublishRateLimiter {
	return &PublishRateLimiter{
		limiter: rate.NewLimiter(r, b),
	}
}

// ThrottlePublishFn is used to decorate a PublishFn so rate limiting is applied
func (p *PublishRateLimiter) ThrottlePublishFn(ctx context.Context, publishFn PublishFn) PublishFn {
	return func(envelope *protocol.Envelope, logger *zap.Logger) error {
		if err := p.Check(ctx, logger); err != nil {
			return err
		}
		return publishFn(envelope, logger)
	}
}

func (p *PublishRateLimiter) Check(ctx context.Context, logger *zap.Logger) error {
	if err := p.limiter.Wait(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("could not send message (limiter)", zap.Error(err))
		}
		return err
	}
	return nil
}
