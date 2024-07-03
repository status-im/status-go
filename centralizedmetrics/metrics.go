package centralizedmetrics

import (
	"database/sql"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/centralizedmetrics/common"
	"github.com/status-im/status-go/centralizedmetrics/providers"
)

const defaultPollInterval = 10 * time.Second

type MetricsInfo struct {
	Enabled       bool `json:"enabled"`
	UserConfirmed bool `json:"userConfirmed"`
}

type MetricRepository interface {
	Poll() ([]common.Metric, error)
	Delete(metrics []common.Metric) error
	Add(metric common.Metric) error
	Info() (*MetricsInfo, error)
	ToggleEnabled(isEnabled bool) error
}

type MetricService struct {
	repository MetricRepository
	processor  common.MetricProcessor
	ticker     *time.Ticker
	done       chan bool
	started    bool
	wg         sync.WaitGroup
}

func NewDefaultMetricService(db *sql.DB) *MetricService {
	repository := NewSQLiteMetricRepository(db)
	processor := providers.NewMixpanelMetricProcessor(providers.MixpanelAppID, providers.MixpanelToken, providers.MixpanelBaseURL)
	return NewMetricService(repository, processor, defaultPollInterval)
}

func NewMetricService(repository MetricRepository, processor common.MetricProcessor, interval time.Duration) *MetricService {
	return &MetricService{
		repository: repository,
		processor:  processor,
		ticker:     time.NewTicker(interval),
		done:       make(chan bool),
	}
}

func (s *MetricService) Start() {
	if s.started {
		return
	}
	s.wg.Add(1)
	s.started = true
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.done:
				return
			case <-s.ticker.C:
				s.processMetrics()
			}
		}
	}()
}

func (s *MetricService) Stop() {
	if !s.started {
		return
	}
	s.ticker.Stop()
	s.done <- true
	s.wg.Wait()
	s.started = false
}

func (s *MetricService) EnsureStarted() error {
	info, err := s.Info()
	if err != nil {
		return err
	}
	if info.Enabled {
		s.Start()
	}
	return nil
}

func (s *MetricService) Info() (*MetricsInfo, error) {
	return s.repository.Info()
}

func (s *MetricService) ToggleEnabled(isEnabled bool) error {
	err := s.repository.ToggleEnabled(isEnabled)
	if err != nil {
		return err
	}
	if isEnabled {
		s.Start()
	} else {
		s.Stop()
	}
	return nil
}

func (s *MetricService) AddMetric(metric common.Metric) error {
	return s.repository.Add(metric)
}

func (s *MetricService) processMetrics() {
	log.Info("processing metrics")
	metrics, err := s.repository.Poll()
	if err != nil {
		log.Warn("error polling metrics", "error", err)
		return
	}
	log.Info("polled metrics")

	if len(metrics) == 0 {
		return
	}
	log.Info("processing metrics")

	if err := s.processor.Process(metrics); err != nil {
		log.Warn("error processing metrics", "error", err)
		return
	}

	log.Info("deleting metrics")
	if err := s.repository.Delete(metrics); err != nil {
		log.Warn("error deleting metrics", "error", err)
	}
	log.Info("done metrics")
}
