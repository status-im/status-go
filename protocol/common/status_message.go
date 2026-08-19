package common

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/internal/crypto/types"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// ApplicationLayer is the topmost layer and represents the application message.
type ApplicationLayer struct {
	// Payload after having been unwrapped from the application layer
	Payload   []byte                                   `json:"-"`
	ID        types.HexBytes                           `json:"id"`
	SigPubKey *ecdsa.PublicKey                         `json:"-"`
	Type      protobuf.ApplicationMetadataMessage_Type `json:"-"`
}

// StatusMessage adds application layer on top of the messaging
type StatusMessage struct {
	messagingtypes.Message
	ApplicationLayer ApplicationLayer `json:"applicationLayer"`
}

// SigPubKey returns the most important signature, from the application layer to transport
func (m *StatusMessage) SigPubKey() *ecdsa.PublicKey {
	if m.ApplicationLayer.SigPubKey != nil {
		return m.ApplicationLayer.SigPubKey
	}

	return m.TransportLayer.SigPubKey
}

func NewStatusMessages(messages []*messagingtypes.Message) []*StatusMessage {
	var statusMessages []*StatusMessage
	for _, m := range messages {
		statusMessage := &StatusMessage{
			Message: *m,
		}
		// Ignoring error as message might not be wrapped into app layer
		_ = processMessageApplicationLayer(statusMessage)
		statusMessages = append(statusMessages, statusMessage)
	}
	return statusMessages
}

func processMessageApplicationLayer(m *StatusMessage) error {
	message, err := protobuf.Unmarshal(m.EncryptionLayer.Payload)
	if err != nil {
		return err
	}

	recoveredKey, err := RecoverKey(message)
	if err != nil {
		return err
	}

	m.ApplicationLayer.SigPubKey = recoveredKey
	// Calculate ID using the wrapped record
	m.ApplicationLayer.ID = messagingtypes.MessageID(recoveredKey, m.EncryptionLayer.Payload)
	m.ApplicationLayer.Payload = message.Payload
	m.ApplicationLayer.Type = message.Type
	return nil
}
