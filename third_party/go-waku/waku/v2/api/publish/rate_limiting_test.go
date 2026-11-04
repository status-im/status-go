package publish

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func TestRateLimit(t *testing.T) {
	r := NewDefaultRateLimiter(rate.Limit(1), 1)
	l := utils.Logger()

	var counter atomic.Int32
	fn := r.ThrottlePublishFn(context.Background(), func(envelope *protocol.Envelope, logger *zap.Logger) error {
		counter.Add(1)
		return nil
	})

	go func() {
		for i := 0; i <= 10; i++ {
			err := fn(nil, l)
			require.NoError(t, err)
		}
	}()

	<-time.After(2 * time.Second)

	require.LessOrEqual(t, counter.Load(), int32(3))
}
