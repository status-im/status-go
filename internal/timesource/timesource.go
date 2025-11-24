package timesource

import (
	"context"
	"time"
)

type Provider interface {
	Now() time.Time
}

type Service interface {
	Provider
	Start(ctx context.Context) error
	Stop()
}

// DefaultService initializes time source with default config values.
func DefaultService() Service {
	return defaultNTPTimeSource
}

func LocalService() Service {
	return &localTimeSource{}
}

