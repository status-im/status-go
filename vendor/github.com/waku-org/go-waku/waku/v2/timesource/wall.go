package timesource

import "time"

type WallClockTimeSource struct {
}

func NewDefaultClock() *WallClockTimeSource {
	return &WallClockTimeSource{}
}

func (t *WallClockTimeSource) Now() time.Time {
	return time.Now()
}

func (t *WallClockTimeSource) Start() error {
	// Do nothing
	return nil
}

func (t *WallClockTimeSource) Stop() error {
	// Do nothing
	return nil
}
