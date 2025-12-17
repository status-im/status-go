package common

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidEventName = errors.New("centralized-metric: invalid-event-name")
	ErrInvalidPlatform  = errors.New("centralized-metric: invalid-platform")
	ErrInvalidVersion   = errors.New("centralized-metric: invalid-version")
)

type Metric struct {
	ID         string         `json:"id"`
	UserID     string         `json:"userId"`
	EventName  string         `json:"eventName"`
	EventValue map[string]any `json:"eventValue"`
	Timestamp  int64          `json:"timestamp"`
	Platform   string         `json:"platform"`
	AppVersion string         `json:"appVersion"`
}

type MetricProcessor interface {
	Process(metrics []Metric) error
}

func (m *Metric) Validate() error {
	if len(m.EventName) == 0 {
		return ErrInvalidEventName
	}
	if len(m.Platform) == 0 {
		return ErrInvalidPlatform
	}
	if len(m.AppVersion) == 0 {
		return ErrInvalidVersion
	}
	return nil
}

func (m *Metric) EnsureID() {

	if len(m.ID) != 0 {
		return
	}
	m.ID = uuid.New().String()
}
