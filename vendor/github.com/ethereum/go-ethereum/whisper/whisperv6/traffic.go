package whisperv6

import (
	"sync"
	"time"

	"github.com/juju/ratelimit"
)

// NewTopicTrafficObserver creates new instance.
func NewTopicTrafficObserver(cfg RateLimitConfig) *TopicTrafficObserver {
	return &TopicTrafficObserver{
		cfg:            cfg,
		trafficByTopic: map[TopicType]*ratelimit.Bucket{},
	}
}

// TopicTrafficObserver provides instrumentation for accounting traffic usage by topic.
type TopicTrafficObserver struct {
	cfg            RateLimitConfig
	mu             sync.RWMutex
	trafficByTopic map[TopicType]*ratelimit.Bucket
}

// Observe consumes traffic from a rate limiter associated with a given topic.
func (t *TopicTrafficObserver) Observe(topic TopicType, size int64) {
	t.mu.Lock()
	rl, exist := t.trafficByTopic[topic]
	if !exist {
		rl = ratelimit.NewBucketWithQuantum(time.Duration(t.cfg.Interval), int64(t.cfg.Capacity), int64(t.cfg.Quantum))
		t.trafficByTopic[topic] = rl
	}
	t.mu.Unlock()
	rl.TakeAvailable(size)
}

// Drained returns true if topic doesn't have available capacity for new tokens.
func (t *TopicTrafficObserver) Drained(topic TopicType) (rst bool) {
	t.mu.RLock()
	rl, exist := t.trafficByTopic[topic]
	t.mu.RUnlock()
	if !exist {
		return false
	}
	return rl.Available() == 0
}
