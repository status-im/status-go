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
)

type Client struct {
	serverURL  string
	httpClient *http.Client
	logger     *zap.Logger
	keyUID     string
	nodeName   string
}

func NewClient(logger *zap.Logger, serverURL string, keyUID string, nodeName string) *Client {
	return &Client{
		serverURL:  serverURL,
		httpClient: &http.Client{Timeout: time.Minute},
		logger:     logger,
		keyUID:     keyUID,
		nodeName:   nodeName,
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
		})
	}
	body, _ := json.Marshal(postBody)
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending message to telemetry server", zap.Error(err))
	}
}
