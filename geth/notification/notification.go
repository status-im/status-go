package notification

import (
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
)

// Manager of push notifications
type Manager struct {
	messaging common.Messaging
}

// New notifications manager
func New(messaging common.Messaging) *Manager {
	return &Manager{
		messaging,
	}
}

// Notify registers notification
func (n *Manager) Notify(token string) string {
	log.Debug("Notify", "token", token)

	n.messaging.NewFcmRegIdsMsg([]string{token}, n.getMessage)

	return token
}

// Send prepared message
func (n *Manager) Send() error {
	_, err := n.messaging.Send()
	if err != nil {
		log.Error("Notify failed:", err)
	}

	return err
}

func (n *Manager) getMessage() interface{} {
	// TODO(oskarth): Experiment with this
	return map[string]string{
		"msg": "Hello World1",
		"sum": "Happy Day",
	}
}
