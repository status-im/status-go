package timesource

import "time"

type Timesource interface {
	Now() time.Time
	Start() error
	Stop() error
}
