package common

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"time"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type RawMessageConfirmation struct {
	// DataSyncID is the ID of the datasync message sent
	DataSyncID []byte
	// MessageID is the message id of the message
	MessageID []byte
	// PublicKey is the compressed receiver public key
	PublicKey []byte
	// ConfirmedAt is the unix timestamp in seconds of when the message was confirmed
	ConfirmedAt int64
}

type RawMessagesPersistence struct {
	db *sql.DB
}

func NewRawMessagesPersistence(db *sql.DB) *RawMessagesPersistence {
	return &RawMessagesPersistence{db: db}
}

func (db RawMessagesPersistence) SaveRawMessage(message *RawMessage) error {
	var pubKeys [][]byte
	for _, pk := range message.Recipients {
		pubKeys = append(pubKeys, crypto.CompressPubkey(pk))
	}
	// Encode recipients
	var encodedRecipients bytes.Buffer
	encoder := gob.NewEncoder(&encodedRecipients)

	if err := encoder.Encode(pubKeys); err != nil {
		return err
	}

	_, err := db.db.Exec(`
		 INSERT INTO
		 raw_messages
		 (
		   id,
		   local_chat_id,
		   last_sent,
		   send_count,
		   sent,
		   message_type,
		   resend_automatically,
		   recipients,
		   skip_encryption,
	           send_push_notification,
		   skip_group_message_wrap,
		   send_on_personal_topic,
		   payload
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.ID,
		message.LocalChatID,
		message.LastSent,
		message.SendCount,
		message.Sent,
		message.MessageType,
		message.ResendAutomatically,
		encodedRecipients.Bytes(),
		message.SkipEncryption,
		message.SendPushNotification,
		message.SkipGroupMessageWrap,
		message.SendOnPersonalTopic,
		message.Payload)
	return err
}

func (db RawMessagesPersistence) RawMessageByID(id string) (*RawMessage, error) {
	var rawPubKeys [][]byte
	var encodedRecipients []byte
	var skipGroupMessageWrap sql.NullBool
	var sendOnPersonalTopic sql.NullBool
	message := &RawMessage{}

	err := db.db.QueryRow(`
			SELECT
			  id,
			  local_chat_id,
			  last_sent,
			  send_count,
			  sent,
			  message_type,
			  resend_automatically,
			  recipients,
			  skip_encryption,
		          send_push_notification,
			  skip_group_message_wrap,
			  send_on_personal_topic,
		          payload
			FROM
				raw_messages
			WHERE
				id = ?`,
		id,
	).Scan(
		&message.ID,
		&message.LocalChatID,
		&message.LastSent,
		&message.SendCount,
		&message.Sent,
		&message.MessageType,
		&message.ResendAutomatically,
		&encodedRecipients,
		&message.SkipEncryption,
		&message.SendPushNotification,
		&skipGroupMessageWrap,
		&sendOnPersonalTopic,
		&message.Payload,
	)
	if err != nil {
		return nil, err
	}

	if rawPubKeys != nil {
		// Restore recipients
		decoder := gob.NewDecoder(bytes.NewBuffer(encodedRecipients))
		err = decoder.Decode(&rawPubKeys)
		if err != nil {
			return nil, err
		}
		for _, pkBytes := range rawPubKeys {
			pubkey, err := crypto.UnmarshalPubkey(pkBytes)
			if err != nil {
				return nil, err
			}
			message.Recipients = append(message.Recipients, pubkey)
		}
	}

	if skipGroupMessageWrap.Valid {
		message.SkipGroupMessageWrap = skipGroupMessageWrap.Bool
	}

	if sendOnPersonalTopic.Valid {
		message.SendOnPersonalTopic = sendOnPersonalTopic.Bool
	}

	return message, nil
}

func (db RawMessagesPersistence) RawMessagesIDsByType(t protobuf.ApplicationMetadataMessage_Type) ([]string, error) {
	ids := []string{}

	rows, err := db.db.Query(`
			SELECT
			  id
			FROM
				raw_messages
			WHERE
			message_type = ?`,
		t)
	if err != nil {
		return ids, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// MarkAsConfirmed marks all the messages with dataSyncID as confirmed and returns
// the messageIDs that can be considered confirmed.
// If atLeastOne is set it will return messageid if at least once of the messages
// sent has been confirmed
func (db RawMessagesPersistence) MarkAsConfirmed(dataSyncID []byte, atLeastOne bool) (messageID types.HexBytes, err error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
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

func (db RawMessagesPersistence) InsertPendingConfirmation(confirmation *RawMessageConfirmation) error {

	_, err := db.db.Exec(`INSERT INTO raw_message_confirmations
		 (datasync_id, message_id, public_key)
		 VALUES
		 (?,?,?)`,
		confirmation.DataSyncID,
		confirmation.MessageID,
		confirmation.PublicKey,
	)
	return err
}
