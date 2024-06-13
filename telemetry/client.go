package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/wakuv2"

	v1protocol "github.com/status-im/status-go/protocol/v1"
	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
)

type TelemetryType string

const (
	ProtocolStatsMetric    TelemetryType = "ProtocolStats"
	ReceivedEnvelopeMetric TelemetryType = "ReceivedEnvelope"
	SentEnvelopeMetric     TelemetryType = "SentEnvelope"
	UpdateEnvelopeMetric   TelemetryType = "UpdateEnvelope"
	ReceivedMessagesMetric TelemetryType = "ReceivedMessages"
)

type TelemetryRequest struct {
	Id            int              `json:"id"`
	TelemetryType TelemetryType    `json:"telemetry_type"`
	TelemetryData *json.RawMessage `json:"telemetry_data"`
}

func (c *Client) PushReceivedMessages(receivedMessages ReceivedMessages) {
	c.receivedMessagesCh <- receivedMessages
}

func (c *Client) PushSentEnvelope(sentEnvelope wakuv2.SentEnvelope) {
	c.sentEnvelopeCh <- sentEnvelope
}

func (c *Client) PushReceivedEnvelope(receivedEnvelope *v2protocol.Envelope) {
	c.receivedEnvelopeCh <- receivedEnvelope
}

type ReceivedMessages struct {
	Filter     transport.Filter
	SSHMessage *types.Message
	Messages   []*v1protocol.StatusMessage
}

type Client struct {
	serverURL          string
	httpClient         *http.Client
	logger             *zap.Logger
	keyUID             string
	nodeName           string
	version            string
	receivedMessagesCh chan ReceivedMessages
	receivedEnvelopeCh chan *v2protocol.Envelope
	sentEnvelopeCh     chan wakuv2.SentEnvelope
	telemetryCh        chan TelemetryRequest
	nextId             int
	sendPeriod         time.Duration
}

func NewClient(logger *zap.Logger, serverURL string, keyUID string, nodeName string, version string) *Client {
	return &Client{
		serverURL:          serverURL,
		httpClient:         &http.Client{Timeout: time.Minute},
		logger:             logger,
		keyUID:             keyUID,
		nodeName:           nodeName,
		version:            version,
		receivedMessagesCh: make(chan ReceivedMessages),
		receivedEnvelopeCh: make(chan *v2protocol.Envelope),
		sentEnvelopeCh:     make(chan wakuv2.SentEnvelope),
		telemetryCh:        make(chan TelemetryRequest),
		nextId:             0,
		sendPeriod:         10 * time.Second,
	}
}

func (c *Client) CollectAndProcessTelemetry(ctx context.Context) {
	go func() {
		for {
			select {
			case receivedMessages := <-c.receivedMessagesCh:
				c.processAndPushTelemetry(receivedMessages)
			case receivedEnvelope := <-c.receivedEnvelopeCh:
				c.processAndPushTelemetry(receivedEnvelope)
			case sentEnvelope := <-c.sentEnvelopeCh:
				c.processAndPushTelemetry(sentEnvelope)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Client) Start(ctx context.Context) {
	go c.CollectAndProcessTelemetry(ctx)
	go func() {
		ticker := time.NewTicker(c.sendPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var telemetryRequests []TelemetryRequest
				collecting := true
				for collecting {
					select {
					case telemetryRequest := <-c.telemetryCh:
						telemetryRequests = append(telemetryRequests, telemetryRequest)
					default:
						collecting = false
					}
				}
				if len(telemetryRequests) > 0 {
					c.pushTelemetryRequest(telemetryRequests)
				}
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
	default:
		c.logger.Error("Unknown telemetry data type")
		return
	}

	c.nextId++
	c.telemetryCh <- telemetryRequest
}

func (c *Client) pushTelemetryRequest(request []TelemetryRequest) {
	url := fmt.Sprintf("%s/record-metrics", c.serverURL)
	body, _ := json.Marshal(request)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending telemetry data", zap.Error(err))
	}
}

func (c *Client) ProcessReceivedMessages(receivedMessages ReceivedMessages) *json.RawMessage {
	c.logger.Debug("Pushing received messages to telemetry server")
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
