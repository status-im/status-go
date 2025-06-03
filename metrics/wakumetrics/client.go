package wakumetrics

import (
	"fmt"
	"strconv"

	messagingtypes "github.com/status-im/status-go/messaging/types"

	"github.com/status-im/status-go/wakuv2"

	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"

	v1protocol "github.com/status-im/status-go/protocol/v1"
	v2common "github.com/status-im/status-go/wakuv2/common"
)

type ReceivedMessages struct {
	Filter     messagingtypes.ChatFilter
	SHHMessage *messagingtypes.ReceivedMessage
	Messages   []*v1protocol.StatusMessage
}

type Client struct {
	peerId               string
	deviceType           string
	version              string
	lastPeerConnFailures map[string]int
}

type TelemetryClientOption func(*Client)

func WithPeerID(peerId string) TelemetryClientOption {
	return func(c *Client) {
		c.peerId = peerId
		metrics.NodePeerId.WithLabelValues(peerId).Set(1)
	}
}

func WithDeviceType(deviceType string) TelemetryClientOption {
	return func(c *Client) {
		c.deviceType = deviceType
	}
}

func WithVersion(version string) TelemetryClientOption {
	return func(c *Client) {
		c.version = version
	}
}

func NewClient(opts ...TelemetryClientOption) (*Client, error) {
	client := &Client{
		lastPeerConnFailures: make(map[string]int),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// RegisterWithRegistry registers all metrics with the provided registry
func (c *Client) RegisterWithRegistry() error {
	if err := RegisterMetrics(); err != nil {
		return fmt.Errorf("failed to register metrics: %v", err)
	}
	return nil
}

func (c *Client) SetDeviceType(deviceType string) {
	c.deviceType = deviceType
}

func (c *Client) PushReceivedMessages(receivedMessages ReceivedMessages) {
	metrics.MessagesReceivedTotal.WithLabelValues(
		receivedMessages.Filter.PubsubTopic,
		receivedMessages.Filter.ContentTopic.String(),
		receivedMessages.Filter.ChatID,
	).Add(float64(len(receivedMessages.Messages)))
}

func (c *Client) PushSentEnvelope(sentEnvelope wakuv2.SentEnvelope) {
	metrics.EnvelopeSentTotal.WithLabelValues(
		sentEnvelope.Envelope.PubsubTopic(),
		sentEnvelope.Envelope.Message().ContentTopic,
		sentEnvelope.PublishMethod.String(),
	).Inc()
}

func (c *Client) PushErrorSendingEnvelope(errorSendingEnvelope wakuv2.ErrorSendingEnvelope) {
	metrics.EnvelopeSentErrors.WithLabelValues(
		errorSendingEnvelope.SentEnvelope.Envelope.PubsubTopic(),
		errorSendingEnvelope.SentEnvelope.Envelope.Message().ContentTopic,
	).Inc()
}

func (c *Client) PushPeerConnFailures(peerConnFailures map[string]int) {
	for peerID, failures := range peerConnFailures {
		if lastFailures, exists := c.lastPeerConnFailures[peerID]; exists {
			if failures == lastFailures {
				continue
			}
		}
		c.lastPeerConnFailures[peerID] = failures
		metrics.PeerConnectionFailures.Add(float64(failures))
	}
}

func (c *Client) PushMessageCheckSuccess() {
	metrics.StoreQuerySuccesses.Inc()
}

func (c *Client) PushMessageCheckFailure() {
	metrics.StoreQueryFailures.Inc()
}

func (c *Client) PushPeerCountByShard(peerCountByShard map[uint16]uint) {
	for shard, count := range peerCountByShard {
		metrics.PeersByShard.WithLabelValues(strconv.FormatUint(uint64(shard), 10)).Set(float64(count))
	}
}

func (c *Client) PushPeerCountByOrigin(peerCountByOrigin map[wps.Origin]uint) {
	for origin, count := range peerCountByOrigin {
		metrics.PeersByOrigin.WithLabelValues(getOriginString(origin)).Set(float64(count))
	}
}

func (c *Client) PushDialFailure(dialFailure v2common.DialError) {
	metrics.PeerDialFailures.WithLabelValues(
		dialFailure.ErrType.String(),
		dialFailure.Protocols,
	).Inc()
}

func (c *Client) PushMissedMessage(envelope *v2protocol.Envelope) {
	metrics.MissedMessages.WithLabelValues(
		envelope.PubsubTopic(),
		envelope.Message().ContentTopic,
	).Inc()
}

func (c *Client) PushMissedRelevantMessage(receivedMessage *v2common.ReceivedMessage) {
	metrics.MissedMessages.WithLabelValues(
		receivedMessage.PubsubTopic,
		receivedMessage.ContentTopic.String(),
	).Inc()
}

func (c *Client) PushMessageDeliveryConfirmed() {
	metrics.MessageDeliveryConfirmations.Inc()
}

func (c *Client) PushSentMessageTotal(messageSize uint32, publishMethod string) {
	metrics.WakuMessagesSizeBytes.WithLabelValues(publishMethod).Add(float64(messageSize))
	metrics.MessagesSentTotal.WithLabelValues(publishMethod).Inc()
}

func (c *Client) PushRawMessageByType(pubsubTopic string, contentTopic string, messageType string, messageSize uint32) {
	metrics.RawMessagesSizeBytes.WithLabelValues(messageType, pubsubTopic, contentTopic).Add(float64(messageSize))
	metrics.RawMessagesSentTotal.WithLabelValues(messageType, pubsubTopic, contentTopic).Inc()
}

func getOriginString(origin wps.Origin) string {
	switch origin {
	case wps.Unknown:
		return "unknown"
	case wps.Discv5:
		return "discv5"
	case wps.Static:
		return "static"
	case wps.PeerExchange:
		return "peer_exchange"
	case wps.DNSDiscovery:
		return "dns_discovery"
	case wps.Rendezvous:
		return "rendezvous"
	case wps.PeerManager:
		return "peer_manager"
	default:
		return "unknown"
	}
}
