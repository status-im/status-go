package centralizedmetrics

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/centralizedmetrics/common"
	"github.com/status-im/status-go/protocol/tt"
)

var testMetric = common.Metric{ID: "user-id", EventName: "test-name", EventValue: map[string]interface{}{"test-name": "test-value"}, Platform: "android", AppVersion: "2.30.0"}

func newMetricService(t *testing.T, repository MetricRepository, processor common.MetricProcessor, interval time.Duration) *MetricService {
	return NewMetricService(repository, processor, interval, tt.MustCreateTestLogger())
}

// TestMetricService covers the main functionalities of MetricService
func TestMetricService(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	// Start the service
	service.Start()
	defer service.Stop()

	// Test adding a metric
	if err := service.AddMetric(testMetric); err != nil {
		t.Fatalf("failed to add metric: %v", err)
	}

	err := tt.RetryWithBackOff(func() error {
		// Verify metrics were processed and deleted
		if len(processor.processedMetrics) != 1 {
			return fmt.Errorf("expected 1 processed metric, got %d", len(processor.processedMetrics))
		}

		if len(repository.metrics) != 0 {
			return fmt.Errorf("expected 0 metrics in repository, got %d", len(repository.metrics))
		}
		return nil
	})
	require.NoError(t, err)
}

// TestMetricRepository is a mock implementation of MetricRepository for testing
type TestMetricRepository struct {
	metrics       []common.Metric
	enabled       bool
	userConfirmed bool
	mutex         sync.Mutex
}

func (r *TestMetricRepository) Poll() ([]common.Metric, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	polledMetrics := r.metrics
	r.metrics = []common.Metric{}
	return polledMetrics, nil
}

func (r *TestMetricRepository) ToggleEnabled(enabled bool) error {
	r.enabled = enabled
	return nil
}

func (r *TestMetricRepository) Info() (*MetricsInfo, error) {
	return &MetricsInfo{Enabled: r.enabled, UserConfirmed: r.userConfirmed}, nil
}

func (r *TestMetricRepository) Delete(metrics []common.Metric) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Simulate deleting from the repository
	for _, metric := range metrics {
		for i, m := range r.metrics {
			if m.ID == metric.ID {
				r.metrics = append(r.metrics[:i], r.metrics[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (r *TestMetricRepository) Add(metric common.Metric) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.metrics = append(r.metrics, metric)
	return nil
}

// TestMetricProcessor is a mock implementation of MetricProcessor for testing
type TestMetricProcessor struct {
	processedMetrics []common.Metric
	mutex            sync.Mutex
}

func (p *TestMetricProcessor) Process(metrics []common.Metric) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.processedMetrics = append(p.processedMetrics, metrics...)
	return nil
}

func TestAddMetric(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	err := service.AddMetric(testMetric)
	if err != nil {
		t.Fatalf("failed to add metric: %v", err)
	}

	// Verify metric was added to the repository
	if len(repository.metrics) != 1 {
		t.Fatalf("expected 1 metric in repository, got %d", len(repository.metrics))
	}

	require.Equal(t, testMetric.ID, repository.metrics[0].ID)
	require.Equal(t, testMetric.EventValue, repository.metrics[0].EventValue)
	require.Equal(t, testMetric.Platform, repository.metrics[0].Platform)
	require.Equal(t, testMetric.AppVersion, repository.metrics[0].AppVersion)
}

func TestProcessMetrics(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	// Add metrics directly to repository for polling
	require.NoError(t, repository.Add(common.Metric{ID: "3", EventValue: map[string]interface{}{"price": 6.28}}))
	require.NoError(t, repository.Add(common.Metric{ID: "4", EventValue: map[string]interface{}{"price": 2.71}}))

	service.processMetrics()

	// Verify metrics were processed
	if len(processor.processedMetrics) != 2 {
		t.Fatalf("expected 2 processed metrics, got %d", len(processor.processedMetrics))
	}

	// Verify metrics were deleted from repository
	if len(repository.metrics) != 0 {
		t.Fatalf("expected 0 metrics in repository, got %d", len(repository.metrics))
	}
}

func TestStartStop(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	service.Start()
	require.True(t, service.started)
	service.Stop()

	err := tt.RetryWithBackOff(func() error {
		if service.started {
			return errors.New("expected service to be stopped, but it is still running")
		}
		return nil

	})
	require.NoError(t, err)
}

func TestServiceWithoutMetrics(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	service.Start()
	defer service.Stop()

	// Verify no metrics were processed
	if len(processor.processedMetrics) != 0 {
		t.Fatalf("expected 0 processed metrics, got %d", len(processor.processedMetrics))
	}
}

func TestServiceEnabled(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	err := service.ToggleEnabled(true)
	require.NoError(t, err)
	require.True(t, service.started)

	err = service.ToggleEnabled(false)
	require.NoError(t, err)
	require.False(t, service.started)
}

func TestServiceEnsureStarted(t *testing.T) {
	repository := &TestMetricRepository{}
	processor := &TestMetricProcessor{}
	service := newMetricService(t, repository, processor, 1*time.Second)

	err := service.EnsureStarted()
	require.NoError(t, err)
	require.False(t, service.started)

	repository.enabled = true

	err = service.EnsureStarted()
	require.NoError(t, err)
	require.True(t, service.started)
}
