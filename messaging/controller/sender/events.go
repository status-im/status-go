package sender

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/layers/encryption/multidevice"
)

type ScheduledReliableSend struct {
	Recipient            *ecdsa.PublicKey
	MessageID            []byte
	ReliabilityMessageID []byte
}

type SentMessage struct {
	Private                bool
	Recipient              *ecdsa.PublicKey
	RecipientInstallations []*multidevice.Installation
	MessageIDs             [][]byte
}
