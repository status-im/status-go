package protocol

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"

	"github.com/pkg/errors"
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
	return db.saveChat(nil, chat)
}

func (db sqlitePersistence) SaveChats(chats []*Chat) error {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
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

	// Insert record
	stmt, err := tx.Prepare(`INSERT INTO chats(id, name, color, active, type, timestamp,  deleted_at_clock_value, unviewed_message_count, last_clock_value, last_message, members, membership_updates)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
		chat.LastMessage,
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
			unviewed_message_count,
			last_clock_value,
			last_message,
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
			chat                     Chat
			encodedMembers           []byte
			encodedMembershipUpdates []byte
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
			&chat.LastMessage,
			&encodedMembers,
			&encodedMembershipUpdates,
		)
		if err != nil {
			return
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

		chats = append(chats, &chat)
	}

	return
}

func (db sqlitePersistence) Chat(chatID string) (*Chat, error) {
	var (
		chat                     Chat
		encodedMembers           []byte
		encodedMembershipUpdates []byte
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
			membership_updates
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
		&chat.LastMessage,
		&encodedMembers,
		&encodedMembershipUpdates,
	)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
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
