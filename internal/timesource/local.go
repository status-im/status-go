package timesource

import (
	"context"
	"time"
)

type localTimeSource struct {
}

func (l *localTimeSource) Now() time.Time {
	return time.Now()
}

func (l *localTimeSource) Start(ctx context.Context) error {
	return nil
}

func (l *localTimeSource) Stop() {
	return
}
