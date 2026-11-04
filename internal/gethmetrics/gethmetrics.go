// Package gethmetrics tunes go-ethereum's process-wide metrics registry.
//
// go-ethereum's RPC server registers a ~16KB histogram per method and outcome
// ("rpc/duration/<method>/<success|failure>") on every call, whether or not
// metrics are enabled. status-go never exports go-ethereum metrics, so those
// histograms are dropped here; all other metrics keep their default behaviour.
package gethmetrics

import (
	"strings"

	"github.com/ethereum/go-ethereum/metrics"
)

const (
	rpcServeTimePrefix = "rpc/duration/"
	successSuffix      = "/success"
	failureSuffix      = "/failure"
)

var emptyHistogramSnapshot = metrics.NewHistogram(metrics.NewUniformSample(1)).Snapshot()

func init() {
	metrics.DefaultRegistry = &registry{Registry: metrics.DefaultRegistry}
}

type registry struct {
	metrics.Registry
}

func (r *registry) GetOrRegister(name string, ctor func() interface{}) interface{} {
	if isRPCServeTimeHistogram(name) {
		return discardHistogram{}
	}
	return r.Registry.GetOrRegister(name, ctor)
}

func isRPCServeTimeHistogram(name string) bool {
	return strings.HasPrefix(name, rpcServeTimePrefix) &&
		(strings.HasSuffix(name, successSuffix) || strings.HasSuffix(name, failureSuffix))
}

type discardHistogram struct{}

func (discardHistogram) Clear()                              {}
func (discardHistogram) Update(int64)                        {}
func (discardHistogram) Snapshot() metrics.HistogramSnapshot { return emptyHistogramSnapshot }
