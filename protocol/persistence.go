package protocol

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
)

var (
	// ErrMsgAlreadyExist returned if msg already exist.
	ErrMsgAlreadyExist = errors.New("message with given ID already exist")
)

// sqlitePersistence wrapper around sql db with operations common for a client.
type sqlitePersistence struct {
	db *sql.DB
}

func (db sqlitePersistence) SaveChat(chat Chat) error {
	err := chat.Validate()
	if err != nil {
		return err
	}
	return db.saveChat(nil, chat)
}

func (db sqlitePersistence) SaveChats(chats []*Chat) error {
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

	for _, chat := range chats {
		err := db.saveChat(tx, *chat)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db sqlitePersistence) SaveContacts(contacts []*Contact) error {
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

	for _, contact := range contacts {
		err := db.SaveContact(contact, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db sqlitePersistence) saveChat(tx *sql.Tx, chat Chat) error {
	var err error
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	}

	// Encode members
	var encodedMembers bytes.Buffer
	memberEncoder := gob.NewEncoder(&encodedMembers)

	if err := memberEncoder.Encode(chat.Members); err != nil {
		return err
	}

	// Encode membership updates
	var encodedMembershipUpdates bytes.Buffer
	membershipUpdatesEncoder := gob.NewEncoder(&encodedMembershipUpdates)

	if err := membershipUpdatesEncoder.Encode(chat.MembershipUpdates); err != nil {
		return err
	}

	// encode last message
	var encodedLastMessage []byte
	if chat.LastMessage != nil {
		encodedLastMessage, err = json.Marshal(chat.LastMessage)
		if err != nil {
			return err
		}
	}

	// Insert record
	stmt, err := tx.Prepare(`INSERT INTO chats(id, name, color, active, type, timestamp,  deleted_at_clock_value, unviewed_message_count, last_clock_value, last_message, members, membership_updates, muted, invitation_admin)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		chat.ID,
		chat.Name,
		chat.Color,
		chat.Active,
		chat.ChatType,
		chat.Timestamp,
		chat.DeletedAtClockValue,
		chat.UnviewedMessagesCount,
		chat.LastClockValue,
		encodedLastMessage,
		encodedMembers.Bytes(),
		encodedMembershipUpdates.Bytes(),
		chat.Muted,
		chat.InvitationAdmin,
	)
	if err != nil {
		return err
	}

	return err
}

func (db sqlitePersistence) DeleteChat(chatID string) error {
	_, err := db.db.Exec("DELETE FROM chats WHERE id = ?", chatID)
	return err
}

func (db sqlitePersistence) MuteChat(chatID string) error {
	_, err := db.db.Exec("UPDATE chats SET muted = 1 WHERE id = ?", chatID)
	return err
}

func (db sqlitePersistence) UnmuteChat(chatID string) error {
	_, err := db.db.Exec("UPDATE chats SET muted = 0 WHERE id = ?", chatID)
	return err
}

func (db sqlitePersistence) Chats() ([]*Chat, error) {
	return db.chats(nil)
}

func (db sqlitePersistence) chats(tx *sql.Tx) (chats []*Chat, err error) {
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			return
		}
		defer func() {
			if err == nil {
				err = tx.Commit()
				return
			}
			// don't shadow original error
			_ = tx.Rollback()
		}()
	}

	rows, err := tx.Query(`
		SELECT
			chats.id,
			chats.name,
			chats.color,
			chats.active,
			chats.type,
			chats.timestamp,
			chats.deleted_at_clock_value,
			chats.unviewed_message_count,
			chats.last_clock_value,
			chats.last_message,
			chats.members,
			chats.membership_updates,
			chats.muted,
			chats.invitation_admin,
			contacts.identicon,
			contacts.alias
		FROM chats LEFT JOIN contacts ON chats.id = contacts.id
		ORDER BY chats.timestamp DESC
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			alias                    sql.NullString
			identicon                sql.NullString
			invitationAdmin          sql.NullString
			chat                     Chat
			encodedMembers           []byte
			encodedMembershipUpdates []byte
			lastMessageBytes         []byte
		)
		err = rows.Scan(
			&chat.ID,
			&chat.Name,
			&chat.Color,
			&chat.Active,
			&chat.ChatType,
			&chat.Timestamp,
			&chat.DeletedAtClockValue,
			&chat.UnviewedMessagesCount,
			&chat.LastClockValue,
			&lastMessageBytes,
			&encodedMembers,
			&encodedMembershipUpdates,
			&chat.Muted,
			&invitationAdmin,
			&identicon,
			&alias,
		)

		if err != nil {
			return
		}

		if invitationAdmin.Valid {
			chat.InvitationAdmin = invitationAdmin.String
		}

		// Restore members
		membersDecoder := gob.NewDecoder(bytes.NewBuffer(encodedMembers))
		err = membersDecoder.Decode(&chat.Members)
		if err != nil {
			return
		}

		// Restore membership updates
		membershipUpdatesDecoder := gob.NewDecoder(bytes.NewBuffer(encodedMembershipUpdates))
		err = membershipUpdatesDecoder.Decode(&chat.MembershipUpdates)
		if err != nil {
			return
		}

		// Restore last message
		if lastMessageBytes != nil {
			message := &Message{}
			if err = json.Unmarshal(lastMessageBytes, message); err != nil {
				return
			}
			chat.LastMessage = message
		}
		chat.Alias = alias.String
		chat.Identicon = identicon.String

		chats = append(chats, &chat)
	}

	return
}

func (db sqlitePersistence) Chat(chatID string) (*Chat, error) {
	var (
		chat                     Chat
		encodedMembers           []byte
		encodedMembershipUpdates []byte
		lastMessageBytes         []byte
		invitationAdmin          sql.NullString
	)

	err := db.db.QueryRow(`
		SELECT
			id,
			name,
			color,
			active,
			type,
			timestamp,
			deleted_at_clock_value,
			unviewed_message_count,
			last_clock_value,
			last_message,
			members,
			membership_updates,
			muted,
			invitation_admin
		FROM chats
		WHERE id = ?
	`, chatID).Scan(&chat.ID,
		&chat.Name,
		&chat.Color,
		&chat.Active,
		&chat.ChatType,
		&chat.Timestamp,
		&chat.DeletedAtClockValue,
		&chat.UnviewedMessagesCount,
		&chat.LastClockValue,
		&lastMessageBytes,
		&encodedMembers,
		&encodedMembershipUpdates,
		&chat.Muted,
		&invitationAdmin,
	)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		if invitationAdmin.Valid {
			chat.InvitationAdmin = invitationAdmin.String
		}
		// Restore members
		membersDecoder := gob.NewDecoder(bytes.NewBuffer(encodedMembers))
		err = membersDecoder.Decode(&chat.Members)
		if err != nil {
			return nil, err
		}

		// Restore membership updates
		membershipUpdatesDecoder := gob.NewDecoder(bytes.NewBuffer(encodedMembershipUpdates))
		err = membershipUpdatesDecoder.Decode(&chat.MembershipUpdates)
		if err != nil {
			return nil, err
		}

		// Restore last message
		if lastMessageBytes != nil {
			message := &Message{}
			if err = json.Unmarshal(lastMessageBytes, message); err != nil {
				return nil, err
			}
			chat.LastMessage = message
		}

		return &chat, nil
	}

	return nil, err

}

func (db sqlitePersistence) Contacts() ([]*Contact, error) {
	rows, err := db.db.Query(`
		SELECT
			id,
			address,
			name,
			alias,
			identicon,
			photo,
			last_updated,
			system_tags,
			device_info,
			ens_verified,
			ens_verified_at,
			tribute_to_talk,
			local_nickname
		FROM contacts
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var response []*Contact

	for rows.Next() {
		var (
			contact           Contact
			encodedDeviceInfo []byte
			encodedSystemTags []byte
			nickname          sql.NullString
		)
		err := rows.Scan(
			&contact.ID,
			&contact.Address,
			&contact.Name,
			&contact.Alias,
			&contact.Identicon,
			&contact.Photo,
			&contact.LastUpdated,
			&encodedSystemTags,
			&encodedDeviceInfo,
			&contact.ENSVerified,
			&contact.ENSVerifiedAt,
			&contact.TributeToTalk,
			&nickname,
		)
		if err != nil {
			return nil, err
		}

		if nickname.Valid {
			contact.LocalNickname = nickname.String
		}

		if encodedDeviceInfo != nil {
			// Restore device info
			deviceInfoDecoder := gob.NewDecoder(bytes.NewBuffer(encodedDeviceInfo))
			if err := deviceInfoDecoder.Decode(&contact.DeviceInfo); err != nil {
				return nil, err
			}
		}

		if encodedSystemTags != nil {
			// Restore system tags
			systemTagsDecoder := gob.NewDecoder(bytes.NewBuffer(encodedSystemTags))
			if err := systemTagsDecoder.Decode(&contact.SystemTags); err != nil {
				return nil, err
			}
		}

		response = append(response, &contact)
	}

	return response, nil
}

func (db sqlitePersistence) SaveRawMessage(message *common.RawMessage) error {
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
		   payload
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		message.Payload)
	return err
}

func (db sqlitePersistence) RawMessageByID(id string) (*common.RawMessage, error) {
	var rawPubKeys [][]byte
	var encodedRecipients []byte
	message := &common.RawMessage{}

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
		&message.Payload,
	)
	if err != nil {
		return nil, err
	}

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

	return message, nil
}

func (db sqlitePersistence) SaveContact(contact *Contact, tx *sql.Tx) (err error) {
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			return
		}
		defer func() {
			if err == nil {
				err = tx.Commit()
				return
			}
			// don't shadow original error
			_ = tx.Rollback()
		}()
	}

	// Encode device info
	var encodedDeviceInfo bytes.Buffer
	deviceInfoEncoder := gob.NewEncoder(&encodedDeviceInfo)
	err = deviceInfoEncoder.Encode(contact.DeviceInfo)
	if err != nil {
		return
	}

	// Encoded system tags
	var encodedSystemTags bytes.Buffer
	systemTagsEncoder := gob.NewEncoder(&encodedSystemTags)
	err = systemTagsEncoder.Encode(contact.SystemTags)
	if err != nil {
		return
	}

	// Insert record
	stmt, err := tx.Prepare(`
		INSERT INTO contacts(
			id,
			address,
			name,
			alias,
			identicon,
			photo,
			last_updated,
			system_tags,
			device_info,
			ens_verified,
			ens_verified_at,
			tribute_to_talk,
			local_nickname
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?)
	`)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		contact.ID,
		contact.Address,
		contact.Name,
		contact.Alias,
		contact.Identicon,
		contact.Photo,
		contact.LastUpdated,
		encodedSystemTags.Bytes(),
		encodedDeviceInfo.Bytes(),
		contact.ENSVerified,
		contact.ENSVerifiedAt,
		contact.TributeToTalk,
		contact.LocalNickname,
	)
	return
}

func (db sqlitePersistence) SaveTransactionToValidate(transaction *TransactionToValidate) error {
	compressedKey := crypto.CompressPubkey(transaction.From)

	_, err := db.db.Exec(`INSERT INTO messenger_transactions_to_validate(
		command_id,
                message_id,
		transaction_hash,
		retry_count,
		first_seen,
		public_key,
		signature,
		to_validate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		transaction.CommandID,
		transaction.MessageID,
		transaction.TransactionHash,
		transaction.RetryCount,
		transaction.FirstSeen,
		compressedKey,
		transaction.Signature,
		transaction.Validate,
	)

	return err
}

func (db sqlitePersistence) UpdateTransactionToValidate(transaction *TransactionToValidate) error {
	_, err := db.db.Exec(`UPDATE messenger_transactions_to_validate
			      SET retry_count = ?, to_validate = ?
			      WHERE transaction_hash = ?`,
		transaction.RetryCount,
		transaction.Validate,
		transaction.TransactionHash,
	)
	return err
}

func (db sqlitePersistence) TransactionsToValidate() ([]*TransactionToValidate, error) {
	var transactions []*TransactionToValidate
	rows, err := db.db.Query(`
		SELECT
		command_id,
			message_id,
			transaction_hash,
			retry_count,
			first_seen,
			public_key,
			signature,
			to_validate
		FROM messenger_transactions_to_validate
		WHERE to_validate = 1;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t TransactionToValidate
		var pkBytes []byte
		err = rows.Scan(
			&t.CommandID,
			&t.MessageID,
			&t.TransactionHash,
			&t.RetryCount,
			&t.FirstSeen,
			&pkBytes,
			&t.Signature,
			&t.Validate,
		)
		if err != nil {
			return nil, err
		}

		publicKey, err := crypto.DecompressPubkey(pkBytes)
		if err != nil {
			return nil, err
		}
		t.From = publicKey

		transactions = append(transactions, &t)
	}

	return transactions, nil
}
