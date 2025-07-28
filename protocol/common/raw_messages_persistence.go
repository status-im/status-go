package common

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"

	"github.com/status-im/status-go/eth-node/crypto"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type RawMessagesPersistence struct {
	db *sql.DB
}

func NewRawMessagesPersistence(db *sql.DB) *RawMessagesPersistence {
	return &RawMessagesPersistence{db: db}
}

func (db RawMessagesPersistence) SaveRawMessage(message *messagingtypes.RawMessage) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

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

	// If the message is not sent, we check whether there's a record
	// in the database already and preserve the state
	if !message.Sent {
		oldMessage, err := db.rawMessageByID(tx, message.ID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if oldMessage != nil {
			message.Sent = oldMessage.Sent
		}
	}
	var sender []byte
	if message.Sender != nil {
		sender = crypto.FromECDSA(message.Sender)
	}
	_, err = tx.Exec(`
		 INSERT INTO
		 raw_messages
		 (
		   id,
		   local_chat_id,
		   last_sent,
		   send_count,
		   sent,
		   message_type,
		   recipients,
		   skip_encryption,
	       send_push_notification,
		   skip_group_message_wrap,
		   send_on_personal_topic,
		   payload,
		   sender,
		   community_id,
		   resend_type,
		   pubsub_topic,
		   hash_ratchet_group_id,
		   community_key_ex_msg_type,
		   resend_method
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		message.ID,
		message.LocalChatID,
		message.LastSent,
		message.SendCount,
		message.Sent,
		message.MessageType,
		encodedRecipients.Bytes(),
		message.SkipEncryptionLayer,
		message.SendPushNotification,
		message.SkipGroupMessageWrap,
		message.SendOnPersonalTopic,
		message.Payload,
		sender,
		message.CommunityID,
		message.ResendType,
		message.PubsubTopic,
		message.HashRatchetGroupID,
		message.CommunityKeyExMsgType,
		message.ResendMethod,
	)
	return err
}

func (db RawMessagesPersistence) RawMessageByID(id string) (*messagingtypes.RawMessage, error) {
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

	return db.rawMessageByID(tx, id)
}

func (db RawMessagesPersistence) rawMessageByID(tx *sql.Tx, id string) (*messagingtypes.RawMessage, error) {
	var rawPubKeys [][]byte
	var encodedRecipients []byte
	var skipGroupMessageWrap, sendOnPersonalTopic sql.NullBool
	var sender []byte
	message := &messagingtypes.RawMessage{}

	err := tx.QueryRow(`
			SELECT
			  id,
			  local_chat_id,
			  last_sent,
			  send_count,
			  sent,
			  message_type,
			  recipients,
			  skip_encryption,
		      send_push_notification,
			  skip_group_message_wrap,
			  send_on_personal_topic,
		      payload,
			  sender,
			  community_id,
			  resend_type,
			  pubsub_topic,
			  hash_ratchet_group_id,
			  community_key_ex_msg_type,
			  resend_method
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
		&encodedRecipients,
		&message.SkipEncryptionLayer,
		&message.SendPushNotification,
		&skipGroupMessageWrap,
		&sendOnPersonalTopic,
		&message.Payload,
		&sender,
		&message.CommunityID,
		&message.ResendType,
		&message.PubsubTopic,
		&message.HashRatchetGroupID,
		&message.CommunityKeyExMsgType,
		&message.ResendMethod,
	)
	if err != nil {
		return nil, err
	}

	if encodedRecipients != nil {
		// Restore recipients
		decoder := gob.NewDecoder(bytes.NewBuffer(encodedRecipients))
		err = decoder.Decode(&rawPubKeys)
		if err != nil {
			return nil, err
		}
		for _, pkBytes := range rawPubKeys {
			pubkey, err := crypto.DecompressPubkey(pkBytes)
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

	if sender != nil {
		message.Sender, err = crypto.ToECDSA(sender)
		if err != nil {
			return nil, err
		}
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

func (db RawMessagesPersistence) GetHashRatchetMessagesCountForGroup(groupID []byte) (int, error) {
	var count int
	err := db.db.QueryRow(`SELECT count(*) FROM hash_ratchet_encrypted_messages WHERE group_id = ?`, groupID).Scan(&count)
	if err == nil {
		return count, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return 0, err
}

func (db RawMessagesPersistence) UpdateRawMessageSent(id string, sent bool) error {
	_, err := db.db.Exec("UPDATE raw_messages SET sent = ? WHERE id = ?", sent, id)
	return err
}

func (db RawMessagesPersistence) UpdateRawMessageLastSent(id string, lastSent uint64) error {
	_, err := db.db.Exec("UPDATE raw_messages SET last_sent = ? WHERE id = ?", lastSent, id)
	return err
}
