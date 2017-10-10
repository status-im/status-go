package notification

import (
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/notification/message"
)

// Manager of push notifications.
type Manager struct {
	messaging common.MessagingProvider
}

// New notifications manager.
func New(messaging common.MessagingProvider) *Manager {
	return &Manager{
		messaging,
	}
}

// Notify makes send message and notification.
func (n *Manager) Notify(token string, msg *message.Message) (string, error) {
	log.Debug("Notify", "token", token)

	n.messaging.SetMessage([]string{token}, msg.Body)
	n.messaging.SetPayload(msg.Payload)

	err := n.messaging.Send()
	if err != nil {
		log.Error("Notify failed:", err)
	}

	return token, err
}
