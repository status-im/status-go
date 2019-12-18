// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package waku

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	envelopesReceivedCounter = prom.NewCounter(prom.CounterOpts{
		Name: "waku_envelopes_received_total",
		Help: "Number of envelopes received.",
	})
	envelopesValidatedCounter = prom.NewCounter(prom.CounterOpts{
		Name: "waku_envelopes_validated_total",
		Help: "Number of envelopes processed successfully.",
	})
	envelopesRejectedCounter = prom.NewCounterVec(prom.CounterOpts{
		Name: "waku_envelopes_rejected_total",
		Help: "Number of envelopes rejected.",
	}, []string{"reason"})
	envelopesCacheFailedCounter = prom.NewCounterVec(prom.CounterOpts{
		Name: "waku_envelopes_cache_failures_total",
		Help: "Number of envelopes which failed to be cached.",
	}, []string{"type"})
	envelopesCachedCounter = prom.NewCounterVec(prom.CounterOpts{
		Name: "waku_envelopes_cached_total",
		Help: "Number of envelopes cached.",
	}, []string{"cache"})
	envelopesSizeMeter = prom.NewHistogram(prom.HistogramOpts{
		Name:    "waku_envelopes_size_bytes",
		Help:    "Size of processed Waku envelopes in bytes.",
		Buckets: prom.ExponentialBuckets(256, 4, 10),
	})
	// rate limiter metrics
	rateLimitsProcessed = prom.NewCounter(prom.CounterOpts{
		Name: "waku_rate_limits_processed_total",
		Help: "Number of packets Waku rate limiter processed.",
	})
	rateLimitsExceeded = prom.NewCounterVec(prom.CounterOpts{
		Name: "waku_rate_limits_exceeded_total",
		Help: "Number of times the Waku rate limits were exceeded",
	}, []string{"type"})
)

func init() {
	prom.MustRegister(envelopesReceivedCounter)
	prom.MustRegister(envelopesRejectedCounter)
	prom.MustRegister(envelopesCacheFailedCounter)
	prom.MustRegister(envelopesCachedCounter)
	prom.MustRegister(envelopesSizeMeter)
	prom.MustRegister(rateLimitsProcessed)
	prom.MustRegister(rateLimitsExceeded)
}
