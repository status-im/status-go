package tt

import (
	"time"

	"github.com/cenkalti/backoff/v3"
)

func RetryWithBackOff(o func() error) error {
	b := backoff.ExponentialBackOff{
		InitialInterval:     time.Millisecond * 100,
		RandomizationFactor: 0.1,
		Multiplier:          1,
		MaxInterval:         time.Second,
		MaxElapsedTime:      time.Second * 10,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return backoff.Retry(o, &b)
}
