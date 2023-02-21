package identify

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/metricshelper"

	"github.com/prometheus/client_golang/prometheus"
)

const metricNamespace = "libp2p_identify"

var (
	pushesTriggered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "identify_pushes_triggered_total",
			Help:      "Pushes Triggered",
		},
		[]string{"trigger"},
	)
	identify = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "identify_total",
			Help:      "Identify",
		},
		[]string{"dir"},
	)
	identifyPush = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Name:      "identify_push_total",
			Help:      "Identify Push",
		},
		[]string{"dir"},
	)
)

var initMetricsOnce sync.Once

func initMetrics() {
	prometheus.MustRegister(pushesTriggered, identify, identifyPush)
}

type MetricsTracer interface {
	TriggeredPushes(event any)
	Identify(network.Direction)
	IdentifyPush(network.Direction)
}

type metricsTracer struct{}

var _ MetricsTracer = &metricsTracer{}

func NewMetricsTracer() MetricsTracer {
	initMetricsOnce.Do(initMetrics)
	return &metricsTracer{}
}

func (t *metricsTracer) TriggeredPushes(ev any) {
	tags := metricshelper.GetStringSlice()
	defer metricshelper.PutStringSlice(tags)

	typ := "unknown"
	switch ev.(type) {
	case event.EvtLocalProtocolsUpdated:
		typ = "protocols_updated"
	case event.EvtLocalAddressesUpdated:
		typ = "addresses_updated"
	}
	*tags = append(*tags, typ)
	pushesTriggered.WithLabelValues(*tags...).Inc()
}

func (t *metricsTracer) Identify(dir network.Direction) {
	tags := metricshelper.GetStringSlice()
	defer metricshelper.PutStringSlice(tags)

	*tags = append(*tags, metricshelper.GetDirection(dir))
	identify.WithLabelValues(*tags...).Inc()
}

func (t *metricsTracer) IdentifyPush(dir network.Direction) {
	tags := metricshelper.GetStringSlice()
	defer metricshelper.PutStringSlice(tags)

	*tags = append(*tags, metricshelper.GetDirection(dir))
	identifyPush.WithLabelValues(*tags...).Inc()
}
