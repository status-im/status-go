package common

import (
	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/types"
)

type MessageConfirmationPersistence interface {
	InsertPendingConfirmation(confirmation *MessageConfirmation) error
	MarkAsConfirmed(dataSyncID []byte, atLeastOne bool) (messageID cryptotypes.HexBytes, err error)
}

type HashRatchetPersistence interface {
	SaveMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error
	GetMessages(keyID []byte) ([]*types.ReceivedMessage, error)
	GetMessagesCountForGroup(groupID []byte) (int, error)
	DeleteMessages(ids [][]byte) error
	DeleteMessagesOlderThan(timestamp int64) error
}

type MessageConfirmation struct {
	// DataSyncID is the ID of the datasync message sent
	DataSyncID []byte
	// MessageID is the message id of the message
	MessageID []byte
	// PublicKey is the compressed receiver public key
	PublicKey []byte
	// ConfirmedAt is the unix timestamp in seconds of when the message was confirmed
	ConfirmedAt int64
}
