package messaging

import (
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/types"
)

func (a *API) Metrics() string {
	return a.core.metrics()
}

func (a *API) MetricsPushReceivedMessages(receivedMessages types.ReceivedMessages) {
	if a.core.wakumetrics != nil {
		a.core.wakumetrics.PushReceivedMessages(adapters.ToWakumetricsReceivedMessages(receivedMessages))
	}
}

func (a *API) MetricsPushSentMessage(pubsubTopic string, contentTopic string, messageType string, messageSize uint32) {
	if a.core.wakumetrics != nil {
		a.core.wakumetrics.PushRawMessageByType(pubsubTopic, contentTopic, messageType, messageSize)
	}
}
