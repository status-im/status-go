package timesource

import "time"

type TimeSource interface {
	Now() time.Time
}
