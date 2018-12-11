package helpers

import (
	"errors"
	"time"
)

// Retry tries to execute the give function `maxRetries` and timeouts after `timeout`.
func Retry(fn func() error, maxRetries int, timeout time.Duration) error {
	tries := 0
	tt := time.NewTimer(timeout)
	defer tt.Stop()

	for {
		tries++

		if err := fn(); err == nil {
			return nil
		}

		if tries > maxRetries {
			return errors.New("too manny retries")
		}

		select {
		case <-tt.C:
			return errors.New("timed out")
		case <-time.After(time.Second):
		}
	}
}
