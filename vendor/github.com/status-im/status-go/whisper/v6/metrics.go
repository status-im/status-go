package whisper

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	envelopesReceivedCounter = prom.NewCounter(prom.CounterOpts{
		Name: "whisper_envelopes_received_total",
		Help: "Number of envelopes received.",
	})
	envelopesValidatedCounter = prom.NewCounter(prom.CounterOpts{
		Name: "whisper_envelopes_validated_total",
		Help: "Number of envelopes processed successfully.",
	})
	envelopesRejectedCounter = prom.NewCounterVec(prom.CounterOpts{
		Name: "whisper_envelopes_rejected_total",
		Help: "Number of envelopes rejected.",
	}, []string{"reason"})
	envelopesCacheFailedCounter = prom.NewCounterVec(prom.CounterOpts{
		Name: "whisper_envelopes_cache_failures_total",
		Help: "Number of envelopes which failed to be cached.",
	}, []string{"type"})
	envelopesCachedCounter = prom.NewCounterVec(prom.CounterOpts{
		Name: "whisper_envelopes_cached_total",
		Help: "Number of envelopes cached.",
	}, []string{"cache"})
	envelopesSizeMeter = prom.NewHistogram(prom.HistogramOpts{
		Name:    "whisper_envelopes_size_bytes",
		Help:    "Size of processed Waku envelopes in bytes.",
		Buckets: prom.ExponentialBuckets(256, 4, 10),
	})
	// rate limiter metrics
	rateLimitsProcessed = prom.NewCounter(prom.CounterOpts{
		Name: "whisper_rate_limits_processed_total",
		Help: "Number of packets Waku rate limiter processed.",
	})
	rateLimitsExceeded = prom.NewCounterVec(prom.CounterOpts{
		Name: "whisper_rate_limits_exceeded_total",
		Help: "Number of times the Waku rate limits were exceeded",
	}, []string{"type"})
	// bridging
	bridgeSent = prom.NewCounter(prom.CounterOpts{
		Name: "whisper_bridge_sent_total",
		Help: "Number of envelopes bridged from Whisper",
	})
	bridgeReceivedSucceed = prom.NewCounter(prom.CounterOpts{
		Name: "whisper_bridge_received_success_total",
		Help: "Number of envelopes bridged to Whisper and successfully added",
	})
	bridgeReceivedFailed = prom.NewCounter(prom.CounterOpts{
		Name: "whisper_bridge_received_failure_total",
		Help: "Number of envelopes bridged to Whisper and failed to be added",
	})
)

func init() {
	prom.MustRegister(envelopesReceivedCounter)
	prom.MustRegister(envelopesRejectedCounter)
	prom.MustRegister(envelopesCacheFailedCounter)
	prom.MustRegister(envelopesCachedCounter)
	prom.MustRegister(envelopesSizeMeter)
	prom.MustRegister(rateLimitsProcessed)
	prom.MustRegister(rateLimitsExceeded)
	prom.MustRegister(bridgeSent)
	prom.MustRegister(bridgeReceivedSucceed)
	prom.MustRegister(bridgeReceivedFailed)
}
