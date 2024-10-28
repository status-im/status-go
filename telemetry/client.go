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

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/wakuv2"

	wps "github.com/waku-org/go-waku/waku/v2/peerstore"

	v1protocol "github.com/status-im/status-go/protocol/v1"
	v2common "github.com/status-im/status-go/wakuv2/common"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
)

type TelemetryType string

const (
	// Bandwidth as reported by libp2p
	ProtocolStatsMetric TelemetryType = "ProtocolStats"
	// Envelopes sent by this node
	SentEnvelopeMetric TelemetryType = "SentEnvelope"
	// Change in status of a sent envelope (usually processing errors)
	UpdateEnvelopeMetric TelemetryType = "UpdateEnvelope"
	// Messages received by this node
	ReceivedMessagesMetric TelemetryType = "ReceivedMessages"
	// Errors encountered when sending envelopes
	ErrorSendingEnvelopeMetric TelemetryType = "ErrorSendingEnvelope"
	// Total connections for this node at a given time
	PeerCountMetric TelemetryType = "PeerCount"
	// Number of failed peer connections for this node at a given time
	PeerConnFailuresMetric TelemetryType = "PeerConnFailure"
	// Store confirmation for a sent message successful
	MessageCheckSuccessMetric TelemetryType = "MessageCheckSuccess"
	// Store confirmation for a sent message failed
	MessageCheckFailureMetric TelemetryType = "MessageCheckFailure"
	// Total connections for this node per shard at a given time
	PeerCountByShardMetric TelemetryType = "PeerCountByShard"
	// Total connections for this node per discovery origin at a given time
	PeerCountByOriginMetric TelemetryType = "PeerCountByOrigin"
	// Error encountered when attempting to dial a peer
	DialFailureMetric TelemetryType = "DialFailure"
	// Missed message as detected by periodic store query
	MissedMessageMetric TelemetryType = "MissedMessages"
	// Missed message with a relevant filter
	MissedRelevantMessageMetric TelemetryType = "MissedRelevantMessages"
	// MVDS ack received for a sent message
	MessageDeliveryConfirmedMetric TelemetryType = "MessageDeliveryConfirmed"
)

const MaxRetryCache = 5000

type TelemetryRequest struct {
	Id            int              `json:"id"`
	TelemetryType TelemetryType    `json:"telemetry_type"`
	TelemetryData *json.RawMessage `json:"telemetry_data"`
}

func (c *Client) PushReceivedMessages(ctx context.Context, receivedMessages ReceivedMessages) {
	c.processAndPushTelemetry(ctx, receivedMessages)
}

func (c *Client) PushSentEnvelope(ctx context.Context, sentEnvelope wakuv2.SentEnvelope) {
	c.processAndPushTelemetry(ctx, sentEnvelope)
}

func (c *Client) PushReceivedEnvelope(ctx context.Context, receivedEnvelope *v2protocol.Envelope) {
	c.processAndPushTelemetry(ctx, receivedEnvelope)
}

func (c *Client) PushErrorSendingEnvelope(ctx context.Context, errorSendingEnvelope wakuv2.ErrorSendingEnvelope) {
	c.processAndPushTelemetry(ctx, errorSendingEnvelope)
}

func (c *Client) PushPeerCount(ctx context.Context, peerCount int) {
	now := time.Now()
	if peerCount != c.lastPeerCount && now.Sub(c.lastPeerCountTime) > 1*time.Second {
		c.lastPeerCount = peerCount
		c.lastPeerCountTime = now
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

func (c *Client) PushMessageCheckSuccess(ctx context.Context, messageHash string) {
	c.processAndPushTelemetry(ctx, MessageCheckSuccess{MessageHash: messageHash})
}

func (c *Client) PushMessageCheckFailure(ctx context.Context, messageHash string) {
	c.processAndPushTelemetry(ctx, MessageCheckFailure{MessageHash: messageHash})
}

func (c *Client) PushPeerCountByShard(ctx context.Context, peerCountByShard map[uint16]uint) {
	for shard, count := range peerCountByShard {
		c.processAndPushTelemetry(ctx, PeerCountByShard{Shard: shard, Count: count})
	}
}

func (c *Client) PushPeerCountByOrigin(ctx context.Context, peerCountByOrigin map[wps.Origin]uint) {
	for origin, count := range peerCountByOrigin {
		c.processAndPushTelemetry(ctx, PeerCountByOrigin{Origin: origin, Count: count})
	}
}

func (c *Client) PushDialFailure(ctx context.Context, dialFailure v2common.DialError) {
	var errorMessage string = ""
	if dialFailure.ErrType == v2common.ErrorUnknown {
		errorMessage = dialFailure.ErrMsg
	}
	c.processAndPushTelemetry(ctx, DialFailure{ErrorType: dialFailure.ErrType, ErrorMsg: errorMessage, Protocols: dialFailure.Protocols})
}

func (c *Client) PushMissedMessage(ctx context.Context, envelope *v2protocol.Envelope) {
	c.processAndPushTelemetry(ctx, MissedMessage{Envelope: envelope})
}

func (c *Client) PushMissedRelevantMessage(ctx context.Context, receivedMessage *v2common.ReceivedMessage) {
	c.processAndPushTelemetry(ctx, MissedRelevantMessage{ReceivedMessage: receivedMessage})
}

func (c *Client) PushMessageDeliveryConfirmed(ctx context.Context, messageHash string) {
	c.processAndPushTelemetry(ctx, MessageDeliveryConfirmed{MessageHash: messageHash})
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

type MessageCheckSuccess struct {
	MessageHash string
}

type MessageCheckFailure struct {
	MessageHash string
}

type PeerCountByShard struct {
	Shard uint16
	Count uint
}

type PeerCountByOrigin struct {
	Origin wps.Origin
	Count  uint
}

type DialFailure struct {
	ErrorType v2common.DialErrorType
	ErrorMsg  string
	Protocols string
}

type MissedMessage struct {
	Envelope *v2protocol.Envelope
}

type MissedRelevantMessage struct {
	ReceivedMessage *v2common.ReceivedMessage
}

type MessageDeliveryConfirmed struct {
	MessageHash string
}

type Client struct {
	serverURL            string
	httpClient           *http.Client
	logger               *zap.Logger
	keyUID               string
	nodeName             string
	peerId               string
	version              string
	telemetryCh          chan TelemetryRequest
	telemetryCacheLock   sync.Mutex
	telemetryCache       []TelemetryRequest
	telemetryRetryCache  []TelemetryRequest
	nextIdLock           sync.Mutex
	nextId               int
	sendPeriod           time.Duration
	lastPeerCount        int
	lastPeerCountTime    time.Time
	lastPeerConnFailures map[string]int
	deviceType           string
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
		telemetryCh:          make(chan TelemetryRequest),
		telemetryCacheLock:   sync.Mutex{},
		telemetryCache:       make([]TelemetryRequest, 0),
		telemetryRetryCache:  make([]TelemetryRequest, 0),
		nextId:               0,
		nextIdLock:           sync.Mutex{},
		sendPeriod:           10 * time.Second, // default value
		lastPeerCount:        0,
		lastPeerCountTime:    time.Time{},
		lastPeerConnFailures: make(map[string]int),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func (c *Client) SetDeviceType(deviceType string) {
	c.deviceType = deviceType
}

func (c *Client) Start(ctx context.Context) {
	go func() {
		defer common.LogOnPanic()
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
		defer common.LogOnPanic()
		sendPeriod := c.sendPeriod
		timer := time.NewTimer(sendPeriod)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				c.telemetryCacheLock.Lock()
				telemetryRequests := make([]TelemetryRequest, len(c.telemetryCache))
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
	var telemetryRequest TelemetryRequest
	switch v := data.(type) {
	case ReceivedMessages:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: ReceivedMessagesMetric,
			TelemetryData: c.ProcessReceivedMessages(v),
		}
	case wakuv2.SentEnvelope:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: SentEnvelopeMetric,
			TelemetryData: c.ProcessSentEnvelope(v),
		}
	case wakuv2.ErrorSendingEnvelope:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: ErrorSendingEnvelopeMetric,
			TelemetryData: c.ProcessErrorSendingEnvelope(v),
		}
	case PeerCount:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: PeerCountMetric,
			TelemetryData: c.ProcessPeerCount(v),
		}
	case PeerConnFailure:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: PeerConnFailuresMetric,
			TelemetryData: c.ProcessPeerConnFailure(v),
		}
	case MessageCheckSuccess:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: MessageCheckSuccessMetric,
			TelemetryData: c.ProcessMessageCheckSuccess(v),
		}
	case MessageCheckFailure:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: MessageCheckFailureMetric,
			TelemetryData: c.ProcessMessageCheckFailure(v),
		}
	case PeerCountByShard:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: PeerCountByShardMetric,
			TelemetryData: c.ProcessPeerCountByShard(v),
		}
	case PeerCountByOrigin:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: PeerCountByOriginMetric,
			TelemetryData: c.ProcessPeerCountByOrigin(v),
		}
	case DialFailure:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: DialFailureMetric,
			TelemetryData: c.ProcessDialFailure(v),
		}
	case MissedMessage:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: MissedMessageMetric,
			TelemetryData: c.ProcessMissedMessage(v),
		}
	case MissedRelevantMessage:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: MissedRelevantMessageMetric,
			TelemetryData: c.ProcessMissedRelevantMessage(v),
		}
	case MessageDeliveryConfirmed:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: MessageDeliveryConfirmedMetric,
			TelemetryData: c.ProcessMessageDeliveryConfirmed(v),
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
func (c *Client) pushTelemetryRequest(request []TelemetryRequest) error {
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

func (c *Client) commonPostBody() map[string]interface{} {
	return map[string]interface{}{
		"nodeName":      c.nodeName,
		"peerId":        c.peerId,
		"statusVersion": c.version,
		"deviceType":    c.deviceType,
		"timestamp":     time.Now().Unix(),
	}
}

func (c *Client) ProcessReceivedMessages(receivedMessages ReceivedMessages) *json.RawMessage {
	var postBody []map[string]interface{}
	for _, message := range receivedMessages.Messages {
		messageBody := c.commonPostBody()
		messageBody["chatId"] = receivedMessages.Filter.ChatID
		messageBody["messageHash"] = types.EncodeHex(receivedMessages.SSHMessage.Hash)
		messageBody["messageId"] = message.ApplicationLayer.ID
		messageBody["sentAt"] = receivedMessages.SSHMessage.Timestamp
		messageBody["pubsubTopic"] = receivedMessages.Filter.PubsubTopic
		messageBody["topic"] = receivedMessages.Filter.ContentTopic.String()
		messageBody["messageType"] = message.ApplicationLayer.Type.String()
		messageBody["receiverKeyUID"] = c.keyUID
		messageBody["messageSize"] = len(receivedMessages.SSHMessage.Payload)
		postBody = append(postBody, messageBody)
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessSentEnvelope(sentEnvelope wakuv2.SentEnvelope) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = sentEnvelope.Envelope.Hash().String()
	postBody["sentAt"] = uint32(sentEnvelope.Envelope.Message().GetTimestamp() / int64(time.Second))
	postBody["pubsubTopic"] = sentEnvelope.Envelope.PubsubTopic()
	postBody["topic"] = sentEnvelope.Envelope.Message().ContentTopic
	postBody["senderKeyUID"] = c.keyUID
	postBody["publishMethod"] = sentEnvelope.PublishMethod.String()
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessErrorSendingEnvelope(errorSendingEnvelope wakuv2.ErrorSendingEnvelope) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = errorSendingEnvelope.SentEnvelope.Envelope.Hash().String()
	postBody["sentAt"] = uint32(errorSendingEnvelope.SentEnvelope.Envelope.Message().GetTimestamp() / int64(time.Second))
	postBody["pubsubTopic"] = errorSendingEnvelope.SentEnvelope.Envelope.PubsubTopic()
	postBody["topic"] = errorSendingEnvelope.SentEnvelope.Envelope.Message().ContentTopic
	postBody["senderKeyUID"] = c.keyUID
	postBody["publishMethod"] = errorSendingEnvelope.SentEnvelope.PublishMethod.String()
	postBody["error"] = errorSendingEnvelope.Error.Error()
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessPeerCount(peerCount PeerCount) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["peerCount"] = peerCount.PeerCount
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessPeerConnFailure(peerConnFailure PeerConnFailure) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["failedPeerId"] = peerConnFailure.FailedPeerId
	postBody["failureCount"] = peerConnFailure.FailureCount
	postBody["nodeKeyUID"] = c.keyUID
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessMessageCheckSuccess(messageCheckSuccess MessageCheckSuccess) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = messageCheckSuccess.MessageHash
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessPeerCountByShard(peerCountByShard PeerCountByShard) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["shard"] = peerCountByShard.Shard
	postBody["count"] = peerCountByShard.Count
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessMessageCheckFailure(messageCheckFailure MessageCheckFailure) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = messageCheckFailure.MessageHash
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessPeerCountByOrigin(peerCountByOrigin PeerCountByOrigin) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["origin"] = peerCountByOrigin.Origin
	postBody["count"] = peerCountByOrigin.Count
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessDialFailure(dialFailure DialFailure) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["errorType"] = dialFailure.ErrorType
	postBody["errorMsg"] = dialFailure.ErrorMsg
	postBody["protocols"] = dialFailure.Protocols
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessMissedMessage(missedMessage MissedMessage) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = missedMessage.Envelope.Hash().String()
	postBody["sentAt"] = uint32(missedMessage.Envelope.Message().GetTimestamp() / int64(time.Second))
	postBody["pubsubTopic"] = missedMessage.Envelope.PubsubTopic()
	postBody["contentTopic"] = missedMessage.Envelope.Message().ContentTopic
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessMissedRelevantMessage(missedMessage MissedRelevantMessage) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = missedMessage.ReceivedMessage.Envelope.Hash().String()
	postBody["sentAt"] = missedMessage.ReceivedMessage.Sent
	postBody["pubsubTopic"] = missedMessage.ReceivedMessage.PubsubTopic
	postBody["contentTopic"] = missedMessage.ReceivedMessage.ContentTopic
	return c.marshalPostBody(postBody)
}

func (c *Client) ProcessMessageDeliveryConfirmed(messageDeliveryConfirmed MessageDeliveryConfirmed) *json.RawMessage {
	postBody := c.commonPostBody()
	postBody["messageHash"] = messageDeliveryConfirmed.MessageHash
	return c.marshalPostBody(postBody)
}

// Helper function to marshal post body and handle errors
func (c *Client) marshalPostBody(postBody map[string]interface{}) *json.RawMessage {
	body, err := json.Marshal(postBody)
	if err != nil {
		c.logger.Error("Error marshaling post body", zap.Error(err))
		return nil
	}
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) UpdateEnvelopeProcessingError(shhMessage *types.Message, processingError error) {
	defer common.LogOnPanic()
	c.logger.Debug("Pushing envelope update to telemetry server", zap.String("hash", types.EncodeHex(shhMessage.Hash)))
	url := fmt.Sprintf("%s/update-envelope", c.serverURL)
	var errorString = ""
	if processingError != nil {
		errorString = processingError.Error()
	}
	postBody := map[string]interface{}{
		"messageHash":     types.EncodeHex(shhMessage.Hash),
		"sentAt":          shhMessage.Timestamp,
		"pubsubTopic":     shhMessage.PubsubTopic,
		"topic":           shhMessage.Topic,
		"receiverKeyUID":  c.keyUID,
		"peerId":          c.peerId,
		"nodeName":        c.nodeName,
		"processingError": errorString,
		"deviceType":      c.deviceType,
	}
	body, _ := json.Marshal(postBody)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending envelope update to telemetry server", zap.Error(err))
	}
}
