package wakumetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	messagesSentTotal = prometheus.NewCounterVec(
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

	messagesReceivedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_messages_received_total",
			Help: "Frequency of Status messages received",
		},
		[]string{"pubsub_topic", "content_topic", "chat_id"},
	)

	wakuMessagesSizeBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_message_size_bytes",
			Help: "Size of each Waku message in bytes sent by this node",
		},
		[]string{"publish_method"},
	)

	envelopeSentErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_envelope_sent_errors_total",
			Help: "Frequency of errors occurred when sending an envelope",
		},
		[]string{"pubsub_topic", "content_topic"},
	)

	messageDeliveryConfirmations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_message_delivery_confirmations_total",
			Help: "Frequency of message delivery confirmations",
		},
	)

	connectedPeers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_connected_peers",
			Help: "Current number of peers connected",
		},
	)

	peersByOrigin = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_peers_by_origin",
			Help: "Number of peers by discovery origin",
		},
		[]string{"origin"},
	)

	peersByShard = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statusgo_waku_peers_by_shard",
			Help: "Number of peers by shard",
		},
		[]string{"shard"},
	)

	peerConnectionFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_peer_connection_failures_total",
			Help: "Total number of peer connection failures",
		},
	)

	peerDialFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_peer_dial_failures_total",
			Help: "Total number of peer dial failures by error type",
		},
		[]string{"error_type", "protocols"},
	)

	storeQuerySuccesses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_store_query_successes_total",
			Help: "Frequency of successful store confirmation queries",
		},
	)

	storeQueryFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "statusgo_waku_store_query_failures_total",
			Help: "Frequency of failed store confirmation queries",
		},
	)

	missedMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "statusgo_waku_missed_messages_total",
			Help: "Frequency of missed messages detected by store query",
		},
		[]string{"pubsub_topic", "content_topic"},
	)

	nodePeerId = prometheus.NewGaugeVec(
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
		messagesSentTotal,
		messagesReceivedTotal,
		wakuMessagesSizeBytes,
		envelopeSentErrors,
		messageDeliveryConfirmations,
		connectedPeers,
		peersByOrigin,
		peersByShard,
		peerConnectionFailures,
		peerDialFailures,
		storeQuerySuccesses,
		storeQueryFailures,
		missedMessages,
		nodePeerId,
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
		messagesSentTotal,
		messagesReceivedTotal,
		wakuMessagesSizeBytes,
		envelopeSentErrors,
		messageDeliveryConfirmations,
		connectedPeers,
		peersByOrigin,
		peersByShard,
		peerConnectionFailures,
		peerDialFailures,
		storeQuerySuccesses,
		storeQueryFailures,
		missedMessages,
		nodePeerId,
	}

	for _, collector := range collectors {
		prometheus.Unregister(collector)
	}

	return nil
}
