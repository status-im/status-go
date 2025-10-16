package protocol

import (
	"context"
	"database/sql"
	"strings"
	"time"

	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/messaging"
	messagingtypes "github.com/status-im/status-go/messaging/types"
)

type messagingPersistence struct {
	db *sql.DB
}

var _ messagingtypes.Persistence = (*messagingPersistence)(nil)

func NewMessagingPersistence(db *sql.DB) *messagingPersistence {
	return &messagingPersistence{db: db}
}

func (m *messagingPersistence) WakuStorage() messagingtypes.WakuPersistence {
	return messaging.NewWakuSQLitePersistence(m.db)
}

func (m *messagingPersistence) TransportStorage() messagingtypes.TransportPersistence {
	return messaging.NewTransportSQLitePersistence(m.db)
}

func (m *messagingPersistence) SegmentationStorage() messagingtypes.SegmentationPersistence {
	return messaging.NewSegmentationSQLitePersistence(m.db)
}

func (m *messagingPersistence) EncryptionStorage() messagingtypes.EncryptionPersistence {
	return messaging.NewEncryptionSQLitePersistence(m.db)
}

func (m *messagingPersistence) MessageSenderStorage() messagingtypes.MessageSenderPersistence {
	return m
}

func (m *messagingPersistence) InsertPendingConfirmation(confirmation *messagingtypes.RawMessageConfirmation) error {
	_, err := m.db.Exec(`INSERT INTO raw_message_confirmations
		 (datasync_id, message_id, public_key)
		 VALUES
		 (?,?,?)`,
		confirmation.DataSyncID,
		confirmation.MessageID,
		confirmation.PublicKey,
	)
	return err
}

func (c *messagingPersistence) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *messagingtypes.ReceivedMessage) error {
	_, err := c.db.Exec(`INSERT INTO hash_ratchet_encrypted_messages(hash, sig, timestamp, topic, payload, dst, padding, group_id, key_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, m.Hash, m.Sig, m.Timestamp, m.Topic.Bytes(), m.Payload, m.Dst, m.Padding, groupID, keyID)
	return err
}

func (m *messagingPersistence) GetHashRatchetMessages(keyID []byte) ([]*messagingtypes.ReceivedMessage, error) {
	var messages []*messagingtypes.ReceivedMessage

	rows, err := m.db.Query(`SELECT hash, sig, timestamp, topic, payload, dst, padding FROM hash_ratchet_encrypted_messages WHERE key_id = ?`, keyID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var topic []byte
		message := &messagingtypes.ReceivedMessage{}

		err := rows.Scan(&message.Hash, &message.Sig, &message.Timestamp, &topic, &message.Payload, &message.Dst, &message.Padding)
		if err != nil {
			return nil, err
		}

		message.Topic = messagingtypes.BytesToContentTopic(topic)
		messages = append(messages, message)
	}

	return messages, nil
}

func (m *messagingPersistence) DeleteHashRatchetMessages(ids [][]byte) error {
	if len(ids) == 0 {
		return nil
	}

	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}
	inVector := strings.Repeat("?, ", len(ids)-1) + "?"

	_, err := m.db.Exec("DELETE FROM hash_ratchet_encrypted_messages WHERE hash IN ("+inVector+")", idsArgs...) // nolint: gosec

	return err
}

func (m *messagingPersistence) DeleteHashRatchetMessagesOlderThan(timestamp int64) error {
	_, err := m.db.Exec("DELETE FROM hash_ratchet_encrypted_messages WHERE timestamp < ?", timestamp)
	return err
}

// MarkAsConfirmed marks all the messages with dataSyncID as confirmed and returns
// the messageIDs that can be considered confirmed.
// If atLeastOne is set it will return messageid if at least once of the messages
// sent has been confirmed
func (m *messagingPersistence) MarkAsConfirmed(dataSyncID []byte, atLeastOne bool) (messageID cryptotypes.HexBytes, err error) {
	tx, err := m.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	confirmedAt := time.Now().Unix()
	_, err = tx.Exec(`UPDATE raw_message_confirmations SET confirmed_at = ? WHERE datasync_id = ? AND confirmed_at = 0`, confirmedAt, dataSyncID)
	if err != nil {
		return
	}

	// Select any tuple that has a message_id with a datasync_id = ? and that has just been confirmed
	rows, err := tx.Query(`SELECT message_id,confirmed_at FROM raw_message_confirmations WHERE message_id = (SELECT message_id FROM raw_message_confirmations WHERE datasync_id = ? LIMIT 1)`, dataSyncID)
	if err != nil {
		return
	}
	defer rows.Close()

	confirmedResult := true

	for rows.Next() {
		var confirmedAt int64
		err = rows.Scan(&messageID, &confirmedAt)
		if err != nil {
			return
		}
		confirmed := confirmedAt > 0

		if atLeastOne && confirmed {
			// We return, as at least one was confirmed
			return
		}

		confirmedResult = confirmedResult && confirmed
	}

	if !confirmedResult {
		messageID = nil
		return
	}

	return
}
