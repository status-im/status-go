package statusproto

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	protocol "github.com/status-im/status-protocol-go/v1"
)

var (
	// ErrMsgAlreadyExist returned if msg already exist.
	ErrMsgAlreadyExist = errors.New("message with given ID already exist")
)

// sqlitePersistence wrapper around sql db with operations common for a client.
type sqlitePersistence struct {
	db *sql.DB
}

func (db sqlitePersistence) LastMessageClock(chatID string) (int64, error) {
	if chatID == "" {
		return 0, errors.New("chat ID is empty")
	}

	var last sql.NullInt64
	err := db.db.QueryRow("SELECT max(clock) FROM user_messages WHERE chat_id = ?", chatID).Scan(&last)
	if err != nil {
		return 0, err
	}
	return last.Int64, nil
}

func (db sqlitePersistence) SaveChat(chat Chat) error {
	var err error

	pkey := []byte{}
	// For one to one chatID is an encoded public key
	if chat.ChatType == ChatTypeOneToOne {
		pkey, err = hex.DecodeString(chat.ID[2:])
		if err != nil {
			return err
		}
		// Safety check, make sure is well formed
		_, err := crypto.UnmarshalPubkey(pkey)
		if err != nil {
			return err
		}

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

	// Insert record
	stmt, err := db.db.Prepare(`INSERT INTO chats(id, name, color, active, type, timestamp,  deleted_at_clock_value, public_key, unviewed_message_count, last_clock_value, last_message_content_type, last_message_content, last_message_timestamp, last_message_clock_value, members, membership_updates)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
		pkey,
		chat.UnviewedMessagesCount,
		chat.LastClockValue,
		chat.LastMessageContentType,
		chat.LastMessageContent,
		chat.LastMessageTimestamp,
		chat.LastMessageClockValue,
		encodedMembers.Bytes(),
		encodedMembershipUpdates.Bytes(),
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
			id,
			name,
			color,
			active,
			type,
			timestamp,
			deleted_at_clock_value,
			public_key,
			unviewed_message_count,
			last_clock_value,
			last_message_content_type,
			last_message_content,
			last_message_timestamp,
			last_message_clock_value,
			members,
			membership_updates
		FROM chats
		ORDER BY chats.timestamp DESC
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			lastMessageContentType sql.NullString
			lastMessageContent     sql.NullString
			lastMessageTimestamp   sql.NullInt64
			lastMessageClockValue  sql.NullInt64

			chat                     Chat
			encodedMembers           []byte
			encodedMembershipUpdates []byte
			pkey                     []byte
		)
		err = rows.Scan(
			&chat.ID,
			&chat.Name,
			&chat.Color,
			&chat.Active,
			&chat.ChatType,
			&chat.Timestamp,
			&chat.DeletedAtClockValue,
			&pkey,
			&chat.UnviewedMessagesCount,
			&chat.LastClockValue,
			&lastMessageContentType,
			&lastMessageContent,
			&lastMessageTimestamp,
			&lastMessageClockValue,
			&encodedMembers,
			&encodedMembershipUpdates,
		)
		if err != nil {
			return
		}
		chat.LastMessageContent = lastMessageContent.String
		chat.LastMessageContentType = lastMessageContentType.String
		chat.LastMessageTimestamp = lastMessageTimestamp.Int64
		chat.LastMessageClockValue = lastMessageClockValue.Int64

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

		if len(pkey) != 0 {
			chat.PublicKey, err = crypto.UnmarshalPubkey(pkey)
			if err != nil {
				return
			}
		}
		chats = append(chats, &chat)
	}

	return
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
			tribute_to_talk
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
		)
		if err != nil {
			return nil, err
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

func (db sqlitePersistence) SetContactsENSData(contacts []Contact) error {
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

	// Ensure contacts exists

	err = db.SetContactsGeneratedData(contacts, tx)
	if err != nil {
		return err
	}

	// Update ens data
	for _, contact := range contacts {
		_, err := tx.Exec(`UPDATE contacts SET name = ?, ens_verified = ? , ens_verified_at = ? WHERE id = ?`, contact.Name, contact.ENSVerified, contact.ENSVerifiedAt, contact.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetContactsGeneratedData sets a contact generated data if not existing already
// in the database
func (db sqlitePersistence) SetContactsGeneratedData(contacts []Contact, tx *sql.Tx) (err error) {
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

	for _, contact := range contacts {
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO contacts(
				id,
				address,
				name,
				alias,
				identicon,
				photo,
				last_updated,
				tribute_to_talk
			) VALUES (?, ?, "", ?, ?, "", 0, "")`,
			contact.ID,
			contact.Address,
			contact.Alias,
			contact.Identicon,
		)
		if err != nil {
			return
		}
	}

	return
}

func (db sqlitePersistence) SaveContact(contact Contact, tx *sql.Tx) (err error) {
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
			tribute_to_talk
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	)
	return
}

// Messages returns messages for a given contact, in a given period. Ordered by a timestamp.
func (db sqlitePersistence) Messages(from, to time.Time) (result []*protocol.Message, err error) {
	rows, err := db.db.Query(`
		SELECT
			id,
			chat_id,
			content_type, 
			message_type, 
			text,
			clock,
			timestamp,
			content_chat_id,
			content_text,
			public_key,
			flags
		FROM user_messages
		WHERE timestamp >= ? AND timestamp <= ? 
		ORDER BY timestamp`,
		protocol.TimestampInMsFromTime(from),
		protocol.TimestampInMsFromTime(to),
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		var pkey []byte
		err = rows.Scan(
			&msg.ID, &msg.ChatID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags,
		)
		if err != nil {
			return
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = crypto.UnmarshalPubkey(pkey)
			if err != nil {
				return
			}
		}
		result = append(result, &msg)
	}
	return
}

func (db sqlitePersistence) SaveMessages(messages []*protocol.Message) (last int64, err error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO user_messages(
			id,
			chat_id, 
			content_type, 
			message_type,
			text,
			clock,
			timestamp,
			content_chat_id,
			content_text,
			public_key,
			flags
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return
	}

	var rst sql.Result

	for _, msg := range messages {
		var pkey []byte
		if msg.SigPubKey != nil {
			pkey = crypto.FromECDSAPub(msg.SigPubKey)
		}
		rst, err = stmt.Exec(
			msg.ID, msg.ChatID, msg.ContentT, msg.MessageT, msg.Text, msg.Clock, msg.Timestamp,
			msg.Content.ChatID, msg.Content.Text, pkey, msg.Flags,
		)
		if err != nil {
			return
		}

		last, err = rst.LastInsertId()
		if err != nil {
			return
		}
	}

	return
}
