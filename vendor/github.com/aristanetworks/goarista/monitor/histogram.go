// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package monitor

import (
	"expvar"
	"strings"
	"time"

	"github.com/aristanetworks/goarista/monitor/stats"
)

// Histogram contains the data needed to properly export itself to expvar
// and provide a pretty printed version
type Histogram struct {
	name      string
	histogram *stats.Histogram
}

// NewHistogram creates a new histogram and registers an HTTP handler for it.
// "name" must end with "Histogram", so that the "/debug/latency" handler can
// pretty print it.
func NewHistogram(name string, numBuckets int, growth float64, smallest float64,
	minValue int64) *Histogram {

	histogramOptions := stats.HistogramOptions{
		NumBuckets:         numBuckets,
		GrowthFactor:       growth,
		SmallestBucketSize: smallest,
		MinValue:           minValue,
	}

	hist := stats.NewHistogram(histogramOptions)
	histogram := &Histogram{
		name:      name,
		histogram: hist,
	}
	expvar.Publish(name, histogram)
	return histogram

}

func (h *Histogram) String() string {
	return h.addUnits(h.histogram.Delta1m().String()) +
		h.addUnits(h.histogram.Delta10m().String()) +
		h.addUnits(h.histogram.Delta1h().String()) +
		h.addUnits(h.histogram.Value().String())
}

// UpdateLatencyValues updates the stats.Histogram's buckets with the new
// datapoint and updates the string associated with the expvar.String
func (h *Histogram) UpdateLatencyValues(t0, t1 time.Time) {
	h.histogram.Add(int64(t1.Sub(t0) / time.Microsecond))
}

func (h *Histogram) addUnits(hist string) string {
	i := strings.Index(hist, "\n")
	return hist[:i] + "µs" + hist[i:]
}
