package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/wakuv2"

	v2protocol "github.com/waku-org/go-waku/waku/v2/protocol"
)

type Client struct {
	serverURL  string
	httpClient *http.Client
	logger     *zap.Logger
	keyUID     string
	nodeName   string
	version    string
}

func NewClient(logger *zap.Logger, serverURL string, keyUID string, nodeName string, version string) *Client {
	return &Client{
		serverURL:  serverURL,
		httpClient: &http.Client{Timeout: time.Minute},
		logger:     logger,
		keyUID:     keyUID,
		nodeName:   nodeName,
		version:    version,
	}
}

func (c *Client) PushReceivedMessages(filter transport.Filter, sshMessage *types.Message, messages []*v1protocol.StatusMessage) {
	c.logger.Debug("Pushing received messages to telemetry server")
	url := fmt.Sprintf("%s/received-messages", c.serverURL)
	var postBody []map[string]interface{}
	for _, message := range messages {
		postBody = append(postBody, map[string]interface{}{
			"chatId":         filter.ChatID,
			"messageHash":    types.EncodeHex(sshMessage.Hash),
			"messageId":      message.ApplicationLayer.ID,
			"sentAt":         sshMessage.Timestamp,
			"pubsubTopic":    filter.PubsubTopic,
			"topic":          filter.ContentTopic.String(),
			"messageType":    message.ApplicationLayer.Type.String(),
			"receiverKeyUID": c.keyUID,
			"nodeName":       c.nodeName,
			"messageSize":    len(sshMessage.Payload),
			"statusVersion":  c.version,
		})
	}
	body, _ := json.Marshal(postBody)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending message to telemetry server", zap.Error(err))
	} else {
		c.logger.Debug("Successfully pushed received messages to telemetry server")
	}
}

func (c *Client) PushReceivedEnvelope(envelope *v2protocol.Envelope) {
	url := fmt.Sprintf("%s/received-envelope", c.serverURL)
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
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending received envelope to telemetry server", zap.Error(err))
	} else {
		c.logger.Debug("Successfully pushed received envelope to telemetry server", zap.String("hash", envelope.Hash().String()))
	}
}

func (c *Client) PushSentEnvelope(envelope *v2protocol.Envelope, publishMethod wakuv2.PublishMethod) {
	url := fmt.Sprintf("%s/sent-envelope", c.serverURL)
	postBody := map[string]interface{}{
		"messageHash":   envelope.Hash().String(),
		"sentAt":        uint32(envelope.Message().GetTimestamp() / int64(time.Second)),
		"pubsubTopic":   envelope.PubsubTopic(),
		"topic":         envelope.Message().ContentTopic,
		"senderKeyUID":  c.keyUID,
		"nodeName":      c.nodeName,
		"publishMethod": publishMethod.String(),
		"statusVersion": c.version,
	}
	body, _ := json.Marshal(postBody)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending pushed envelope to telemetry server", zap.Error(err))
	} else {
		c.logger.Debug("Successfully pushed sent envelope to telemetry server", zap.String("hash", envelope.Hash().String()))
	}
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
	} else {
		c.logger.Debug("Successfully pushed envelope processing error to telemetry server", zap.String("hash", types.EncodeHex(shhMessage.Hash)))
	}
}
