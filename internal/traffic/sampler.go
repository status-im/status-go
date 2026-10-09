package traffic

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/httptraffic"

	"github.com/status-im/status-go/internal/panics"
)

// A log line lists the heaviest few of each, compactly, to stay one short line.
const (
	loggedSources   = 8
	loggedHosts     = 5
	loggedEndpoints = 5
)

// Sampler runs a recorder's Report loop as a service of its own. It is not
// pausable on purpose: it has to keep sampling while the other services are
// paused for the background, which is the traffic it is there to show.
type Sampler struct {
	rec           *httptraffic.Recorder
	interval      time.Duration
	samplesPerLog int
	logger        *zap.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewSampler samples rec every interval and logs the traffic at debug every
// samplesPerLog samples.
func NewSampler(rec *httptraffic.Recorder, interval time.Duration, samplesPerLog int, logger *zap.Logger) *Sampler {
	return &Sampler{rec: rec, interval: interval, samplesPerLog: samplesPerLog, logger: logger}
}

// Start begins sampling; it does nothing when already sampling.
func (s *Sampler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go func(done chan struct{}) {
		defer panics.LogOnPanic()
		defer close(done)
		s.rec.Report(ctx, s.interval, s.samplesPerLog, s.log)
	}(s.done)
}

// Stop ends sampling and waits for the loop to return.
func (s *Sampler) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func (s *Sampler) log(d httptraffic.Snapshot) {
	ce := s.logger.Check(zap.DebugLevel, "HTTP traffic")
	if ce == nil {
		return
	}
	summary := d.Summary(loggedSources, loggedHosts, loggedEndpoints)
	ce.Write(
		zap.Duration("interval", d.Until.Sub(d.Since)),
		zap.Uint64("bytesSent", d.Totals.BytesSent),
		zap.Uint64("bytesReceived", d.Totals.BytesReceived),
		zap.Uint64("requests", d.Totals.Requests),
		zap.Bool("inBackground", d.Totals.InBackground),
		zap.Strings("sources", summary.Sources),
		zap.Strings("hosts", summary.Hosts),
		zap.Strings("endpoints", summary.Endpoints),
	)
}
