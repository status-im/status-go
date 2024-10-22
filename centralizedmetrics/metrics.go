package centralizedmetrics

import (
	"database/sql"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/centralizedmetrics/common"
	"github.com/status-im/status-go/centralizedmetrics/providers"
	gocommon "github.com/status-im/status-go/common"
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
	interval   time.Duration

	logger *zap.Logger
}

func NewDefaultMetricService(db *sql.DB, logger *zap.Logger) *MetricService {
	repository := NewSQLiteMetricRepository(db)
	processor := providers.NewMixpanelMetricProcessor(providers.MixpanelAppID, providers.MixpanelToken, providers.MixpanelBaseURL, logger)
	return NewMetricService(repository, processor, defaultPollInterval, logger)
}

func NewMetricService(repository MetricRepository, processor common.MetricProcessor, interval time.Duration, logger *zap.Logger) *MetricService {
	return &MetricService{
		repository: repository,
		processor:  processor,
		interval:   interval,
		done:       make(chan bool),
		logger:     logger.Named("MetricService"),
	}
}

func (s *MetricService) Start() {
	if s.started {
		return
	}
	s.ticker = time.NewTicker(s.interval)
	s.wg.Add(1)
	s.started = true
	go func() {
		defer gocommon.LogOnPanic()
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
	s.logger.Info("processing metrics")
	metrics, err := s.repository.Poll()
	if err != nil {
		s.logger.Warn("error polling metrics", zap.Error(err))
		return
	}
	s.logger.Info("polled metrics")

	if len(metrics) == 0 {
		return
	}
	s.logger.Info("processing metrics")

	if err := s.processor.Process(metrics); err != nil {
		s.logger.Warn("error processing metrics", zap.Error(err))
		return
	}

	s.logger.Info("deleting metrics")
	if err := s.repository.Delete(metrics); err != nil {
		s.logger.Warn("error deleting metrics", zap.Error(err))
	}
	s.logger.Info("done metrics")
}
