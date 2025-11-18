package common

import (
	"crypto/ecdsa"

	messagingevents "github.com/status-im/status-go/messaging/events"
)

type MessageEvent struct {
	ScheduledMessage *ScheduledMessageEvent
	SentMessage      *messagingevents.SentMessage
}

type ScheduledMessageEvent struct {
	Recipient  *ecdsa.PublicKey
	RawMessage *RawMessage
}
