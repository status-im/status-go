package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/wakuv2"

	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"

	v1protocol "github.com/status-im/status-go/protocol/v1"
)

type TelemetryType string

const (
	ProtocolStatsMetric        TelemetryType = "ProtocolStats"
	ReceivedEnvelopeMetric     TelemetryType = "ReceivedEnvelope"
	SentEnvelopeMetric         TelemetryType = "SentEnvelope"
	UpdateEnvelopeMetric       TelemetryType = "UpdateEnvelope"
	ReceivedMessagesMetric     TelemetryType = "ReceivedMessages"
	ErrorSendingEnvelopeMetric TelemetryType = "ErrorSendingEnvelope"

	MaxRetryCache = 5000
)

type TelemetryRequest struct {
	Id            int              `json:"id"`
	TelemetryType TelemetryType    `json:"telemetry_type"`
	TelemetryData *json.RawMessage `json:"telemetry_data"`
}

func (c *Client) PushReceivedMessages(receivedMessages ReceivedMessages) {
	c.processAndPushTelemetry(receivedMessages)
}

func (c *Client) PushSentEnvelope(sentEnvelope wakuv2.SentEnvelope) {
	c.processAndPushTelemetry(sentEnvelope)
}

func (c *Client) PushReceivedEnvelope(receivedEnvelope *v2protocol.Envelope) {
	c.processAndPushTelemetry(receivedEnvelope)
}

func (c *Client) PushErrorSendingEnvelope(errorSendingEnvelope wakuv2.ErrorSendingEnvelope) {
	c.processAndPushTelemetry(errorSendingEnvelope)
}

type ReceivedMessages struct {
	Filter     transport.Filter
	SSHMessage *types.Message
	Messages   []*v1protocol.StatusMessage
}

type Client struct {
	serverURL           string
	httpClient          *http.Client
	logger              *zap.Logger
	keyUID              string
	nodeName            string
	version             string
	telemetryCh         chan TelemetryRequest
	telemetryCacheLock  sync.Mutex
	telemetryCache      []TelemetryRequest
	telemetryRetryCache []TelemetryRequest
	nextIdLock          sync.Mutex
	nextId              int
	sendPeriod          time.Duration
}

type TelemetryClientOption func(*Client)

func WithSendPeriod(sendPeriod time.Duration) TelemetryClientOption {
	return func(c *Client) {
		c.sendPeriod = sendPeriod
	}
}

func NewClient(logger *zap.Logger, serverURL string, keyUID string, nodeName string, version string, opts ...TelemetryClientOption) *Client {
	client := &Client{
		serverURL:           serverURL,
		httpClient:          &http.Client{Timeout: time.Minute},
		logger:              logger,
		keyUID:              keyUID,
		nodeName:            nodeName,
		version:             version,
		telemetryCh:         make(chan TelemetryRequest),
		telemetryCacheLock:  sync.Mutex{},
		telemetryCache:      make([]TelemetryRequest, 0),
		telemetryRetryCache: make([]TelemetryRequest, 0),
		nextId:              0,
		nextIdLock:          sync.Mutex{},
		sendPeriod:          10 * time.Second, // default value
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
				telemetryRequests := make([]TelemetryRequest, len(c.telemetryCache))
				copy(telemetryRequests, c.telemetryCache)
				c.telemetryCache = nil
				c.telemetryCacheLock.Unlock()

				if len(telemetryRequests) > 0 {
					err := c.pushTelemetryRequest(telemetryRequests)
					if err != nil {
						if sendPeriod < 60 { //Stop the growing if the timer is > 60s to at least retry every minute
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

func (c *Client) processAndPushTelemetry(data interface{}) {
	var telemetryRequest TelemetryRequest
	switch v := data.(type) {
	case ReceivedMessages:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: ReceivedMessagesMetric,
			TelemetryData: c.ProcessReceivedMessages(v),
		}
	case *v2protocol.Envelope:
		telemetryRequest = TelemetryRequest{
			Id:            c.nextId,
			TelemetryType: ReceivedEnvelopeMetric,
			TelemetryData: c.ProcessReceivedEnvelope(v),
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
	default:
		c.logger.Error("Unknown telemetry data type")
		return
	}

	c.telemetryCh <- telemetryRequest
	c.nextIdLock.Lock()
	c.nextId++
	c.nextIdLock.Unlock()
}

// This is assuming to not run concurrently as we are not locking the `telemetryRetryCache`
func (c *Client) pushTelemetryRequest(request []TelemetryRequest) error {
	if len(c.telemetryRetryCache)+len(request) > MaxRetryCache { //Limit the size of the cache to not grow the slice indefinitely in case the Telemetry server is gone for longer time
		removeNum := len(c.telemetryRetryCache) + len(request) - MaxRetryCache
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
	if res.StatusCode != http.StatusOK {
		c.logger.Error("Error sending telemetry data", zap.Int("statusCode", res.StatusCode))
		return fmt.Errorf("status code %d", res.StatusCode)
	}

	c.telemetryRetryCache = nil
	return nil
}

func (c *Client) ProcessReceivedMessages(receivedMessages ReceivedMessages) *json.RawMessage {
	var postBody []map[string]interface{}
	for _, message := range receivedMessages.Messages {
		postBody = append(postBody, map[string]interface{}{
			"chatId":         receivedMessages.Filter.ChatID,
			"messageHash":    types.EncodeHex(receivedMessages.SSHMessage.Hash),
			"messageId":      message.ApplicationLayer.ID,
			"sentAt":         receivedMessages.SSHMessage.Timestamp,
			"pubsubTopic":    receivedMessages.Filter.PubsubTopic,
			"topic":          receivedMessages.Filter.ContentTopic.String(),
			"messageType":    message.ApplicationLayer.Type.String(),
			"receiverKeyUID": c.keyUID,
			"nodeName":       c.nodeName,
			"messageSize":    len(receivedMessages.SSHMessage.Payload),
			"statusVersion":  c.version,
		})
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessReceivedEnvelope(envelope *v2protocol.Envelope) *json.RawMessage {
	postBody := map[string]interface{}{
		"messageHash":    envelope.Hash().String(),
		"sentAt":         uint32(envelope.Message().GetTimestamp() / int64(time.Second)),
		"pubsubTopic":    envelope.PubsubTopic(),
		"topic":          envelope.Message().ContentTopic,
		"receiverKeyUID": c.keyUID,
		"nodeName":       c.nodeName,
		"statusVersion":  c.version,
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessSentEnvelope(sentEnvelope wakuv2.SentEnvelope) *json.RawMessage {
	postBody := map[string]interface{}{
		"messageHash":   sentEnvelope.Envelope.Hash().String(),
		"sentAt":        uint32(sentEnvelope.Envelope.Message().GetTimestamp() / int64(time.Second)),
		"pubsubTopic":   sentEnvelope.Envelope.PubsubTopic(),
		"topic":         sentEnvelope.Envelope.Message().ContentTopic,
		"senderKeyUID":  c.keyUID,
		"nodeName":      c.nodeName,
		"publishMethod": sentEnvelope.PublishMethod.String(),
		"statusVersion": c.version,
	}
	body, _ := json.Marshal(postBody)
	jsonRawMessage := json.RawMessage(body)
	return &jsonRawMessage
}

func (c *Client) ProcessErrorSendingEnvelope(errorSendingEnvelope wakuv2.ErrorSendingEnvelope) *json.RawMessage {
	postBody := map[string]interface{}{
		"messageHash":   errorSendingEnvelope.SentEnvelope.Envelope.Hash().String(),
		"sentAt":        uint32(errorSendingEnvelope.SentEnvelope.Envelope.Message().GetTimestamp() / int64(time.Second)),
		"pubsubTopic":   errorSendingEnvelope.SentEnvelope.Envelope.PubsubTopic(),
		"topic":         errorSendingEnvelope.SentEnvelope.Envelope.Message().ContentTopic,
		"senderKeyUID":  c.keyUID,
		"nodeName":      c.nodeName,
		"publishMethod": errorSendingEnvelope.SentEnvelope.PublishMethod.String(),
		"statusVersion": c.version,
		"error":         errorSendingEnvelope.Error.Error(),
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
	postBody := map[string]interface{}{
		"messageHash":     types.EncodeHex(shhMessage.Hash),
		"sentAt":          shhMessage.Timestamp,
		"pubsubTopic":     shhMessage.PubsubTopic,
		"topic":           shhMessage.Topic,
		"receiverKeyUID":  c.keyUID,
		"nodeName":        c.nodeName,
		"processingError": errorString,
	}
	body, _ := json.Marshal(postBody)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending envelope update to telemetry server", zap.Error(err))
	}
}
