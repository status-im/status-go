package wakumetrics

import (
	"fmt"
	"strconv"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/wakuv2"

	v1protocol "github.com/status-im/status-go/protocol/v1"
	v2common "github.com/status-im/status-go/wakuv2/common"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
)

type ReceivedMessages struct {
	Filter     transport.Filter
	SSHMessage *types.Message
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
	MessagesReceivedTotal.WithLabelValues(
		receivedMessages.Filter.PubsubTopic,
		receivedMessages.Filter.ContentTopic.String(),
		receivedMessages.Filter.ChatID,
	).Observe(float64(len(receivedMessages.Messages)))
}

func (c *Client) PushSentEnvelope(sentEnvelope wakuv2.SentEnvelope) {
	EnvelopeSentTotal.WithLabelValues(
		sentEnvelope.Envelope.PubsubTopic(),
		sentEnvelope.Envelope.Message().ContentTopic,
		sentEnvelope.PublishMethod.String(),
	).Observe(1)
}

func (c *Client) PushErrorSendingEnvelope(errorSendingEnvelope wakuv2.ErrorSendingEnvelope) {
	EnvelopeSentErrors.WithLabelValues(
		errorSendingEnvelope.SentEnvelope.Envelope.PubsubTopic(),
		errorSendingEnvelope.SentEnvelope.Envelope.Message().ContentTopic,
	).Inc()
}

func (c *Client) PushPeerCount(peerCount int) {
	ConnectedPeers.Set(float64(peerCount))
}

func (c *Client) PushPeerConnFailures(peerConnFailures map[string]int) {
	for peerID, failures := range peerConnFailures {
		if lastFailures, exists := c.lastPeerConnFailures[peerID]; exists {
			if failures == lastFailures {
				continue
			}
		}
		c.lastPeerConnFailures[peerID] = failures
		PeerConnectionFailures.Add(float64(failures))
	}
}

func (c *Client) PushMessageCheckSuccess() {
	StoreQuerySuccesses.Observe(1)
}

func (c *Client) PushMessageCheckFailure() {
	StoreQueryFailures.Observe(1)
}

func (c *Client) PushPeerCountByShard(peerCountByShard map[uint16]uint) {
	for shard, count := range peerCountByShard {
		PeersByShard.WithLabelValues(strconv.FormatUint(uint64(shard), 10)).Set(float64(count))
	}
}

func (c *Client) PushPeerCountByOrigin(peerCountByOrigin map[wps.Origin]uint) {
	for origin, count := range peerCountByOrigin {
		PeersByOrigin.WithLabelValues(getOriginString(origin)).Set(float64(count))
	}
}

func (c *Client) PushDialFailure(dialFailure v2common.DialError) {
	PeerDialFailures.WithLabelValues(
		dialFailure.ErrType.String(),
		dialFailure.Protocols,
	).Inc()
}

func (c *Client) PushMissedMessage(envelope *v2protocol.Envelope) {
	MissedMessages.WithLabelValues(
		envelope.PubsubTopic(),
		envelope.Message().ContentTopic,
	).Inc()
}

func (c *Client) PushMissedRelevantMessage(receivedMessage *v2common.ReceivedMessage) {
	MissedMessages.WithLabelValues(
		receivedMessage.PubsubTopic,
		receivedMessage.ContentTopic.String(),
	).Inc()
}

func (c *Client) PushMessageDeliveryConfirmed() {
	MessageDeliveryConfirmations.Inc()
}

func (c *Client) PushSentMessageTotal(messageSize uint32, publishMethod string) {
	WakuMessagesSizeBytes.WithLabelValues(publishMethod).Observe(float64(messageSize))
	MessagesSentTotal.WithLabelValues(publishMethod).Observe(1)
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
