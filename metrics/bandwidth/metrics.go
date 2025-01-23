package bandwidth

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	BandwidthIn = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_bandwidth_in",
			Help: "Incoming bandwidth rate per protocol",
		},
		[]string{"protocol"},
	)

	BandwidthOut = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_bandwidth_out",
			Help: "Outgoing bandwidth rate per protocol",
		},
		[]string{"protocol"},
	)

	BandwidthTotalIn = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_bandwidth_total_in",
			Help: "Total incoming bandwidth per protocol",
		},
		[]string{"protocol"},
	)

	BandwidthTotalOut = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_bandwidth_total_out",
			Help: "Total outgoing bandwidth per protocol",
		},
		[]string{"protocol"},
	)
)

// RegisterMetrics registers all bandwidth metrics with the provided registry
func RegisterMetrics() error {
	collectors := []prometheus.Collector{
		BandwidthIn,
		BandwidthOut,
		BandwidthTotalIn,
		BandwidthTotalOut,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

// UnregisterMetrics unregisters all bandwidth metrics
func UnregisterMetrics() error {
	collectors := []prometheus.Collector{
		BandwidthIn,
		BandwidthOut,
		BandwidthTotalIn,
		BandwidthTotalOut,
	}

	for _, collector := range collectors {
		prometheus.Unregister(collector)
	}

	return nil
}
