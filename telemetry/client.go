package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"

	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"

	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"

	v1protocol "github.com/status-im/status-go/protocol/v1"

	telemetrytypes "github.com/status-im/telemetry/pkg/types"

	telemetry "github.com/status-im/status-go/telemetry/common"
)

const (
	MaxRetryCache = 5000
)

func (c *Client) PushReceivedMessages(ctx context.Context, receivedMessages ReceivedMessages) {
	c.processAndPushTelemetry(ctx, receivedMessages)
}

func (c *Client) PushSentEnvelope(ctx context.Context, sentEnvelope telemetry.SentEnvelope) {
	c.processAndPushTelemetry(ctx, sentEnvelope)
}

func (c *Client) PushReceivedEnvelope(ctx context.Context, receivedEnvelope *v2protocol.Envelope) {
	c.processAndPushTelemetry(ctx, receivedEnvelope)
}

func (c *Client) PushErrorSendingEnvelope(ctx context.Context, errorSendingEnvelope telemetry.ErrorSendingEnvelope) {
	c.processAndPushTelemetry(ctx, errorSendingEnvelope)
}

func (c *Client) PushProtocolStats(ctx context.Context, stats telemetry.ProtocolStatsMap) {
	c.processAndPushTelemetry(ctx, stats)
}

func (c *Client) PushPeerCount(ctx context.Context, peerCount int) {
	if peerCount != c.lastPeerCount {
		c.lastPeerCount = peerCount
		c.processAndPushTelemetry(ctx, PeerCount{PeerCount: peerCount})
	}
}

func (c *Client) PushPeerConnFailures(ctx context.Context, peerConnFailures map[string]int) {
	for peerID, failures := range peerConnFailures {
		if lastFailures, exists := c.lastPeerConnFailures[peerID]; exists {
			if failures == lastFailures {
				continue
			}
		}
		c.lastPeerConnFailures[peerID] = failures
		c.processAndPushTelemetry(ctx, PeerConnFailure{FailedPeerId: peerID, FailureCount: failures})
	}
}

type ReceivedMessages struct {
	Filter     transport.Filter
	SSHMessage *types.Message
	Messages   []*v1protocol.StatusMessage
}

type PeerCount struct {
	PeerCount int
}

type PeerConnFailure struct {
	FailedPeerId string
	FailureCount int
}

type Client struct {
	telemetry.ITelemetryClient
	serverURL            string
	httpClient           *http.Client
	logger               *zap.Logger
	keyUID               string
	nodeName             string
	peerId               string
	version              string
	telemetryCh          chan telemetrytypes.TelemetryRequest
	telemetryCacheLock   sync.Mutex
	telemetryCache       []telemetrytypes.TelemetryRequest
	telemetryRetryCache  []telemetrytypes.TelemetryRequest
	nextIdLock           sync.Mutex
	nextId               int
	sendPeriod           time.Duration
	lastPeerCount        int
	lastPeerConnFailures map[string]int
}

type TelemetryClientOption func(*Client)

func WithSendPeriod(sendPeriod time.Duration) TelemetryClientOption {
	return func(c *Client) {
		c.sendPeriod = sendPeriod
	}
}

func WithPeerID(peerId string) TelemetryClientOption {
	return func(c *Client) {
		c.peerId = peerId
	}
}

func NewClient(logger *zap.Logger, serverURL string, keyUID string, nodeName string, version string, opts ...TelemetryClientOption) *Client {
	serverURL = strings.TrimRight(serverURL, "/")
	client := &Client{
		serverURL:            serverURL,
		httpClient:           &http.Client{Timeout: time.Minute},
		logger:               logger,
		keyUID:               keyUID,
		nodeName:             nodeName,
		version:              version,
		telemetryCh:          make(chan telemetrytypes.TelemetryRequest),
		telemetryCacheLock:   sync.Mutex{},
		telemetryCache:       make([]telemetrytypes.TelemetryRequest, 0),
		telemetryRetryCache:  make([]telemetrytypes.TelemetryRequest, 0),
		nextId:               0,
		nextIdLock:           sync.Mutex{},
		sendPeriod:           10 * time.Second, // default value
		lastPeerCount:        0,
		lastPeerConnFailures: make(map[string]int),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func (c *Client) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case telemetryRequest := <-c.telemetryCh:
				c.telemetryCacheLock.Lock()
				c.telemetryCache = append(c.telemetryCache, telemetryRequest)
				c.telemetryCacheLock.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		sendPeriod := c.sendPeriod
		timer := time.NewTimer(sendPeriod)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				c.telemetryCacheLock.Lock()
				telemetryRequests := make([]telemetrytypes.TelemetryRequest, len(c.telemetryCache))
				copy(telemetryRequests, c.telemetryCache)
				c.telemetryCache = nil
				c.telemetryCacheLock.Unlock()

				if len(telemetryRequests) > 0 {
					err := c.pushTelemetryRequest(telemetryRequests)
					if err != nil {
						if sendPeriod < 60*time.Second { //Stop the growing if the timer is > 60s to at least retry every minute
							sendPeriod = sendPeriod * 2
						}
					} else {
						sendPeriod = c.sendPeriod
					}
				}
				timer.Reset(sendPeriod)
			case <-ctx.Done():
				return
			}
		}

	}()
}

func (c *Client) processAndPushTelemetry(ctx context.Context, data interface{}) {
	var telemetryRequest telemetrytypes.TelemetryRequest
	switch v := data.(type) {
	case ReceivedMessages:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.ReceivedMessagesMetric,
			TelemetryData: c.ProcessReceivedMessages(v),
		}
	case *v2protocol.Envelope:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.ReceivedEnvelopeMetric,
			TelemetryData: c.ProcessReceivedEnvelope(v),
		}
	case telemetry.SentEnvelope:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.SentEnvelopeMetric,
			TelemetryData: c.ProcessSentEnvelope(v),
		}
	case telemetry.ErrorSendingEnvelope:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.ErrorSendingEnvelopeMetric,
			TelemetryData: c.ProcessErrorSendingEnvelope(v),
		}
	case PeerCount:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.PeerCountMetric,
			TelemetryData: c.ProcessPeerCount(v),
		}
	case PeerConnFailure:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.PeerConnFailureMetric,
			TelemetryData: c.ProcessPeerConnFailure(v),
		}
	case telemetry.ProtocolStatsMap:
		telemetryRequest = telemetrytypes.TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: telemetrytypes.ProtocolStatsMetric,
			TelemetryData: c.ProcessProtocolStats(v),
		}
	default:
		c.logger.Error("Unknown telemetry data type")
		return
	}

	select {
	case <-ctx.Done():
		return
	case c.telemetryCh <- telemetryRequest:
	}

	c.nextIdLock.Lock()
	c.nextId++
	c.nextIdLock.Unlock()
}

// This is assuming to not run concurrently as we are not locking the `telemetryRetryCache`
func (c *Client) pushTelemetryRequest(request []telemetrytypes.TelemetryRequest) error {
	if len(c.telemetryRetryCache) > MaxRetryCache { //Limit the size of the cache to not grow the slice indefinitely in case the Telemetry server is gone for longer time
		removeNum := len(c.telemetryRetryCache) - MaxRetryCache
		c.telemetryRetryCache = c.telemetryRetryCache[removeNum:]
	}
	c.telemetryRetryCache = append(c.telemetryRetryCache, request...)

	url := fmt.Sprintf("%s/record-metrics", c.serverURL)
	body, err := json.Marshal(c.telemetryRetryCache)
	if err != nil {
		c.logger.Error("Error marshaling telemetry data", zap.Error(err))
		return err
	}
	res, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending telemetry data", zap.Error(err))
		return err
	}
	defer res.Body.Close()
	var responseBody []map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&responseBody); err != nil {
		c.logger.Error("Error decoding response body", zap.Error(err))
		return err
	}
	if res.StatusCode != http.StatusCreated {
		c.logger.Error("Error sending telemetry data", zap.Int("statusCode", res.StatusCode), zap.Any("responseBody", responseBody))
		return fmt.Errorf("status code %d, response body: %v", res.StatusCode, responseBody)
	}

	c.telemetryRetryCache = nil
	return nil
}

func (c *Client) ProcessReceivedMessages(receivedMessages ReceivedMessages) *json.RawMessage {
	var postBody []telemetrytypes.ReceivedMessage
	for _, message := range receivedMessages.Messages {
		postBody = append(postBody, telemetrytypes.ReceivedMessage{
			ChatID:         receivedMessages.Filter.ChatID,
			MessageHash:    types.EncodeHex(receivedMessages.SSHMessage.Hash),
			MessageID:      message.ApplicationLayer.ID.String(),
			SentAt:         int64(receivedMessages.SSHMessage.Timestamp),
			PubsubTopic:    receivedMessages.Filter.PubsubTopic,
			Topic:          receivedMessages.Filter.ContentTopic.String(),
			MessageType:    message.ApplicationLayer.Type.String(),
			ReceiverKeyUID: c.keyUID,
			PeerID:         c.peerId,
			NodeName:       c.nodeName,
			MessageSize:    len(receivedMessages.SSHMessage.Payload),
			StatusVersion:  c.version,
		})
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessReceivedEnvelope(envelope *v2protocol.Envelope) *json.RawMessage {
	postBody := telemetrytypes.ReceivedEnvelope{
		MessageHash:    envelope.Hash().String(),
		SentAt:         envelope.Message().GetTimestamp() / int64(time.Second),
		PubsubTopic:    envelope.PubsubTopic(),
		Topic:          envelope.Message().ContentTopic,
		ReceiverKeyUID: c.keyUID,
		PeerID:         c.peerId,
		NodeName:       c.nodeName,
		StatusVersion:  c.version,
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessSentEnvelope(sentEnvelope telemetry.SentEnvelope) *json.RawMessage {
	postBody := telemetrytypes.SentEnvelope{
		MessageHash:   sentEnvelope.Envelope.Hash().String(),
		SentAt:        sentEnvelope.Envelope.Message().GetTimestamp() / int64(time.Second),
		PubsubTopic:   sentEnvelope.Envelope.PubsubTopic(),
		Topic:         sentEnvelope.Envelope.Message().ContentTopic,
		SenderKeyUID:  c.keyUID,
		PeerID:        c.peerId,
		NodeName:      c.nodeName,
		PublishMethod: sentEnvelope.PublishMethod.String(),
		StatusVersion: c.version,
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessErrorSendingEnvelope(errorSendingEnvelope telemetry.ErrorSendingEnvelope) *json.RawMessage {
	postBody := telemetrytypes.ErrorSendingEnvelope{
		SentEnvelope: telemetrytypes.SentEnvelope{
			MessageHash:   errorSendingEnvelope.SentEnvelope.Envelope.Hash().String(),
			SentAt:        errorSendingEnvelope.SentEnvelope.Envelope.Message().GetTimestamp() / int64(time.Second),
			PubsubTopic:   errorSendingEnvelope.SentEnvelope.Envelope.PubsubTopic(),
			Topic:         errorSendingEnvelope.SentEnvelope.Envelope.Message().ContentTopic,
			SenderKeyUID:  c.keyUID,
			PeerID:        c.peerId,
			NodeName:      c.nodeName,
			PublishMethod: errorSendingEnvelope.SentEnvelope.PublishMethod.String(),
			StatusVersion: c.version,
		},
		Error: errorSendingEnvelope.Error.Error(),
	}

	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessPeerCount(peerCount PeerCount) *json.RawMessage {
	postBody := telemetrytypes.PeerCount{
		PeerCount:     peerCount.PeerCount,
		NodeName:      c.nodeName,
		NodeKeyUid:    c.keyUID,
		PeerID:        c.peerId,
		StatusVersion: c.version,
		Timestamp:     time.Now().Unix(),
	}

	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessPeerConnFailure(peerConnFailure PeerConnFailure) *json.RawMessage {
	postBody := telemetrytypes.PeerConnFailure{
		FailedPeerId:  peerConnFailure.FailedPeerId,
		FailureCount:  peerConnFailure.FailureCount,
		NodeName:      c.nodeName,
		NodeKeyUid:    c.keyUID,
		PeerId:        c.peerId,
		StatusVersion: c.version,
		Timestamp:     time.Now().Unix(),
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) UpdateEnvelopeProcessingError(shhMessage *types.Message, processingError error) {
	c.logger.Debug("Pushing envelope update to telemetry server", zap.String("hash", types.EncodeHex(shhMessage.Hash)))
	url := fmt.Sprintf("%s/update-envelope", c.serverURL)
	var errorString = ""
	if processingError != nil {
		errorString = processingError.Error()
	}
	postBody := telemetrytypes.ReceivedEnvelope{
		MessageHash:     types.EncodeHex(shhMessage.Hash),
		SentAt:          int64(shhMessage.Timestamp),
		PubsubTopic:     shhMessage.PubsubTopic,
		Topic:           shhMessage.Topic.String(),
		ReceiverKeyUID:  c.keyUID,
		PeerID:          c.peerId,
		NodeName:        c.nodeName,
		ProcessingError: errorString,
	}
	body, _ := json.Marshal(postBody)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending envelope update to telemetry server", zap.Error(err))
	}
}

func (c *Client) ProcessProtocolStats(stats telemetry.ProtocolStatsMap) *json.RawMessage {
	getStatsPerProtocol := func(protocolID protocol.ID, stats telemetry.ProtocolStatsMap) telemetrytypes.Metric {
		return telemetrytypes.Metric{
			RateIn:   stats[protocolID].RateIn,
			RateOut:  stats[protocolID].RateOut,
			TotalIn:  stats[protocolID].TotalIn,
			TotalOut: stats[protocolID].TotalOut,
		}
	}

	postBody := telemetrytypes.ProtocolStats{
		PeerID:          c.peerId,
		Relay:           getStatsPerProtocol(relay.WakuRelayID_v200, stats),
		Store:           getStatsPerProtocol(legacy_store.StoreID_v20beta4, stats),
		FilterPush:      getStatsPerProtocol(filter.FilterPushID_v20beta1, stats),
		FilterSubscribe: getStatsPerProtocol(filter.FilterSubscribeID_v20beta1, stats),
		Lightpush:       getStatsPerProtocol(lightpush.LightPushID_v20beta1, stats),
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}
