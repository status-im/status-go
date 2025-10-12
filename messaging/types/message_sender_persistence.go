package types

import (
	cryptotypes "github.com/status-im/status-go/crypto/types"
)

type MessageSenderPersistence interface {
	InsertPendingConfirmation(confirmation *RawMessageConfirmation) error
	MarkAsConfirmed(dataSyncID []byte, atLeastOne bool) (messageID cryptotypes.HexBytes, err error)
	SaveHashRatchetMessage(groupID []byte, keyID []byte, m *ReceivedMessage) error
	GetHashRatchetMessages(keyID []byte) ([]*ReceivedMessage, error)
	DeleteHashRatchetMessages(ids [][]byte) error
	DeleteHashRatchetMessagesOlderThan(timestamp int64) error
}
