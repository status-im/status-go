package wakumetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MessagesSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_messages_sent_total",
			Help: "Frequency of Waku messages sent by this node",
		},
		[]string{"publish_method"},
	)

	EnvelopeSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_envelope_sent_total",
			Help: "Total number of envelopes sent by this node",
		},
		[]string{"pubsub_topic", "content_topic", "publish_method"},
	)

	MessagesReceivedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_messages_received_total",
			Help: "Frequency of Status messages received",
		},
		[]string{"pubsub_topic", "content_topic", "chat_id"},
	)

	WakuMessagesSizeBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_message_size_bytes",
			Help: "Size of each Waku message in bytes sent by this node",
		},
		[]string{"publish_method"},
	)

	EnvelopeSentErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_envelope_sent_errors_total",
			Help: "Frequency of errors occurred when sending an envelope",
		},
		[]string{"pubsub_topic", "content_topic"},
	)

	MessageDeliveryConfirmations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_message_delivery_confirmations_total",
			Help: "Frequency of message delivery confirmations",
		},
	)

	ConnectedPeers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_connected_peers",
			Help: "Current number of peers connected",
		},
	)

	PeersByOrigin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_peers_by_origin",
			Help: "Number of peers by discovery origin",
		},
		[]string{"origin"},
	)

	PeersByShard = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_peers_by_shard",
			Help: "Number of peers by shard",
		},
		[]string{"shard"},
	)

	PeerConnectionFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_peer_connection_failures_total",
			Help: "Total number of peer connection failures",
		},
	)

	PeerDialFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_peer_dial_failures_total",
			Help: "Total number of peer dial failures by error type",
		},
		[]string{"error_type", "protocols"},
	)

	StoreQuerySuccesses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_store_query_successes_total",
			Help: "Frequency of successful store confirmation queries",
		},
	)

	StoreQueryFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_store_query_failures_total",
			Help: "Frequency of failed store confirmation queries",
		},
	)

	MissedMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_missed_messages_total",
			Help: "Frequency of missed messages detected by store query",
		},
		[]string{"pubsub_topic", "content_topic"},
	)

	PeerId = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_peer_id",
			Help: "Peer ID",
		},
		[]string{"peer_id"},
	)
)

// RegisterMetrics registers all metrics with the provided registry
func RegisterMetrics() error {
	collectors := []prometheus.Collector{
		MessagesSentTotal,
		MessagesReceivedTotal,
		WakuMessagesSizeBytes,
		EnvelopeSentErrors,
		MessageDeliveryConfirmations,
		ConnectedPeers,
		PeersByOrigin,
		PeersByShard,
		PeerConnectionFailures,
		PeerDialFailures,
		StoreQuerySuccesses,
		StoreQueryFailures,
		MissedMessages,
		PeerId,
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			return err
		}
	}

	return nil
}

func UnregisterMetrics() error {
	collectors := []prometheus.Collector{
		MessagesSentTotal,
		MessagesReceivedTotal,
		WakuMessagesSizeBytes,
		EnvelopeSentErrors,
		MessageDeliveryConfirmations,
		ConnectedPeers,
		PeersByOrigin,
		PeersByShard,
		PeerConnectionFailures,
		PeerDialFailures,
		StoreQuerySuccesses,
		StoreQueryFailures,
		MissedMessages,
		PeerId,
	}

	for _, collector := range collectors {
		prometheus.Unregister(collector)
	}

	return nil
}
