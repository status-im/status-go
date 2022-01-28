package protocol

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	// ErrMsgAlreadyExist returned if msg already exist.
	ErrMsgAlreadyExist = errors.New("message with given ID already exist")
	HoursInTwoWeeks    = 336
)

// sqlitePersistence wrapper around sql db with operations common for a client.
type sqlitePersistence struct {
	*common.RawMessagesPersistence
	db *sql.DB
}

func newSQLitePersistence(db *sql.DB) *sqlitePersistence {
	return &sqlitePersistence{common.NewRawMessagesPersistence(db), db}
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
	stmt, err := tx.Prepare(`INSERT INTO chats(id, name, color, emoji, active, type, timestamp,  deleted_at_clock_value, unviewed_message_count, unviewed_mentions_count, last_clock_value, last_message, members, membership_updates, muted, invitation_admin, profile, community_id, joined, synced_from, synced_to, description, highlight, read_messages_at_clock_value, received_invitation_admin)
	    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		chat.ID,
		chat.Name,
		chat.Color,
		chat.Emoji,
		chat.Active,
		chat.ChatType,
		chat.Timestamp,
		chat.DeletedAtClockValue,
		chat.UnviewedMessagesCount,
		chat.UnviewedMentionsCount,
		chat.LastClockValue,
		encodedLastMessage,
		encodedMembers.Bytes(),
		encodedMembershipUpdates.Bytes(),
		chat.Muted,
		chat.InvitationAdmin,
		chat.Profile,
		chat.CommunityID,
		chat.Joined,
		chat.SyncedFrom,
		chat.SyncedTo,
		chat.Description,
		chat.Highlight,
		chat.ReadMessagesAtClockValue,
		chat.ReceivedInvitationAdmin,
	)

	if err != nil {
		return err
	}

	return err
}

func (db sqlitePersistence) SetSyncTimestamps(syncedFrom, syncedTo uint32, chatID string) error {
	_, err := db.db.Exec(`UPDATE chats SET synced_from = ?, synced_to = ? WHERE id = ?`, syncedFrom, syncedTo, chatID)
	return err
}

func (db sqlitePersistence) DeleteChat(chatID string) (err error) {
	var tx *sql.Tx
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

	_, err = tx.Exec("DELETE FROM chats WHERE id = ?", chatID)
	if err != nil {
		return
	}

	_, err = tx.Exec(`DELETE FROM user_messages WHERE local_chat_id = ?`, chatID)
	return
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
			chats.emoji,
			chats.active,
			chats.type,
			chats.timestamp,
			chats.deleted_at_clock_value,
                        chats.read_messages_at_clock_value,
			chats.unviewed_message_count,
			chats.unviewed_mentions_count,
			chats.last_clock_value,
			chats.last_message,
			chats.members,
			chats.membership_updates,
			chats.muted,
			chats.invitation_admin,
			chats.profile,
			chats.community_id,
			chats.joined,
			chats.synced_from,
			chats.synced_to,
		    chats.description,
			contacts.alias,
                        chats.highlight,
                        chats.received_invitation_admin
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
			invitationAdmin          sql.NullString
			profile                  sql.NullString
			syncedFrom               sql.NullInt64
			syncedTo                 sql.NullInt64
			chat                     Chat
			encodedMembers           []byte
			encodedMembershipUpdates []byte
			lastMessageBytes         []byte
		)
		err = rows.Scan(
			&chat.ID,
			&chat.Name,
			&chat.Color,
			&chat.Emoji,
			&chat.Active,
			&chat.ChatType,
			&chat.Timestamp,
			&chat.DeletedAtClockValue,
			&chat.ReadMessagesAtClockValue,
			&chat.UnviewedMessagesCount,
			&chat.UnviewedMentionsCount,
			&chat.LastClockValue,
			&lastMessageBytes,
			&encodedMembers,
			&encodedMembershipUpdates,
			&chat.Muted,
			&invitationAdmin,
			&profile,
			&chat.CommunityID,
			&chat.Joined,
			&syncedFrom,
			&syncedTo,
			&chat.Description,
			&alias,
			&chat.Highlight,
			&chat.ReceivedInvitationAdmin,
		)

		if err != nil {
			return
		}

		if invitationAdmin.Valid {
			chat.InvitationAdmin = invitationAdmin.String
		}

		if profile.Valid {
			chat.Profile = profile.String
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

		if syncedFrom.Valid {
			chat.SyncedFrom = uint32(syncedFrom.Int64)
		}

		if syncedTo.Valid {
			chat.SyncedTo = uint32(syncedTo.Int64)
		}

		// Restore last message
		if lastMessageBytes != nil {
			message := &common.Message{}
			if err = json.Unmarshal(lastMessageBytes, message); err != nil {
				return
			}
			chat.LastMessage = message
		}
		chat.Alias = alias.String

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
		profile                  sql.NullString
		syncedFrom               sql.NullInt64
		syncedTo                 sql.NullInt64
	)

	err := db.db.QueryRow(`
		SELECT
			id,
			name,
			color,
			emoji,
			active,
			type,
			timestamp,
			read_messages_at_clock_value,
			deleted_at_clock_value,
			unviewed_message_count,
			unviewed_mentions_count,
			last_clock_value,
			last_message,
			members,
			membership_updates,
			muted,
			invitation_admin,
			profile,
			community_id,
            joined,
		    description,
                    highlight,
                    received_invitation_admin,
                    synced_from,
                    synced_to
		FROM chats
		WHERE id = ?
	`, chatID).Scan(&chat.ID,
		&chat.Name,
		&chat.Color,
		&chat.Emoji,
		&chat.Active,
		&chat.ChatType,
		&chat.Timestamp,
		&chat.ReadMessagesAtClockValue,
		&chat.DeletedAtClockValue,
		&chat.UnviewedMessagesCount,
		&chat.UnviewedMentionsCount,
		&chat.LastClockValue,
		&lastMessageBytes,
		&encodedMembers,
		&encodedMembershipUpdates,
		&chat.Muted,
		&invitationAdmin,
		&profile,
		&chat.CommunityID,
		&chat.Joined,
		&chat.Description,
		&chat.Highlight,
		&chat.ReceivedInvitationAdmin,
		&syncedFrom,
		&syncedTo,
	)
	switch err {
	case sql.ErrNoRows:
		return nil, nil
	case nil:
		if syncedFrom.Valid {
			chat.SyncedFrom = uint32(syncedFrom.Int64)
		}
		if syncedTo.Valid {
			chat.SyncedTo = uint32(syncedTo.Int64)
		}
		if invitationAdmin.Valid {
			chat.InvitationAdmin = invitationAdmin.String
		}
		if profile.Valid {
			chat.Profile = profile.String
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
			message := &common.Message{}
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
	allContacts := make(map[string]*Contact)

	rows, err := db.db.Query(`
		SELECT
			c.id,
			c.address,
			v.name,
			v.verified,
			c.alias,
			c.display_name,
			c.identicon,
			c.last_updated,
			c.last_updated_locally,
			c.added,
			c.blocked,
			c.removed,
			c.has_added_us,
			c.local_nickname,
			i.image_type,
			i.payload
		FROM contacts c 
		LEFT JOIN chat_identity_contacts i ON c.id = i.contact_id 
		LEFT JOIN ens_verification_records v ON c.id = v.public_key;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var (
			contact            Contact
			nickname           sql.NullString
			displayName        sql.NullString
			imageType          sql.NullString
			ensName            sql.NullString
			ensVerified        sql.NullBool
			added              sql.NullBool
			blocked            sql.NullBool
			removed            sql.NullBool
			hasAddedUs         sql.NullBool
			lastUpdatedLocally sql.NullInt64
			imagePayload       []byte
		)

		contact.Images = make(map[string]images.IdentityImage)

		err := rows.Scan(
			&contact.ID,
			&contact.Address,
			&ensName,
			&ensVerified,
			&contact.Alias,
			&displayName,
			&contact.Identicon,
			&contact.LastUpdated,
			&lastUpdatedLocally,
			&added,
			&blocked,
			&removed,
			&hasAddedUs,
			&nickname,
			&imageType,
			&imagePayload,
		)
		if err != nil {
			return nil, err
		}

		if nickname.Valid {
			contact.LocalNickname = nickname.String
		}

		if displayName.Valid {
			contact.DisplayName = displayName.String
		}

		if ensName.Valid {
			contact.EnsName = ensName.String
		}

		if ensVerified.Valid {
			contact.ENSVerified = ensVerified.Bool
		}

		if added.Valid {
			contact.Added = added.Bool
		}

		if blocked.Valid {
			contact.Blocked = blocked.Bool
		}

		if removed.Valid {
			contact.Removed = removed.Bool
		}

		if lastUpdatedLocally.Valid {
			contact.LastUpdatedLocally = uint64(lastUpdatedLocally.Int64)
		}

		if hasAddedUs.Valid {
			contact.HasAddedUs = hasAddedUs.Bool
		}

		previousContact, ok := allContacts[contact.ID]
		if !ok {
			if imageType.Valid {
				contact.Images[imageType.String] = images.IdentityImage{Name: imageType.String, Payload: imagePayload}
			}

			allContacts[contact.ID] = &contact

		} else if imageType.Valid {
			previousContact.Images[imageType.String] = images.IdentityImage{Name: imageType.String, Payload: imagePayload}
			allContacts[contact.ID] = previousContact

		}
	}

	var response []*Contact
	for key := range allContacts {
		response = append(response, allContacts[key])

	}
	return response, nil
}

func (db sqlitePersistence) SaveContactChatIdentity(contactID string, chatIdentity *protobuf.ChatIdentity) (updated bool, err error) {
	if chatIdentity.Clock == 0 {
		return false, errors.New("clock value unset")
	}

	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		// don't shadow original error
		_ = tx.Rollback()
	}()

	for imageType, image := range chatIdentity.Images {
		var exists bool
		err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM chat_identity_contacts WHERE contact_id = ? AND image_type = ? AND clock_value >= ?)`, contactID, imageType, chatIdentity.Clock).Scan(&exists)
		if err != nil {
			return false, err
		}

		if exists {
			continue
		}

		stmt, err := tx.Prepare(`INSERT INTO chat_identity_contacts (contact_id, image_type, clock_value, payload) VALUES (?, ?, ?, ?)`)
		if err != nil {
			return false, err
		}
		defer stmt.Close()
		if image.Payload == nil {
			continue
		}

		// TODO implement something that doesn't reject all images if a single image fails validation
		// Validate image URI to make sure it's serializable
		_, err = images.GetPayloadDataURI(image.Payload)
		if err != nil {
			return false, err
		}

		_, err = stmt.Exec(
			contactID,
			imageType,
			chatIdentity.Clock,
			image.Payload,
		)
		if err != nil {
			return false, err
		}
		updated = true
	}

	return
}

func (db sqlitePersistence) ExpiredMessagesIDs(maxSendCount int) ([]string, error) {
	ids := []string{}

	rows, err := db.db.Query(`
			SELECT
			  id
			FROM
				raw_messages
			WHERE
			message_type IN (?, ?) AND sent = ? AND send_count <= ?`,
		protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		false,
		maxSendCount)
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

	// Insert record
	// NOTE: name, photo and tribute_to_talk are not used anymore, but it's not nullable
	// Removing it requires copying over the table which might be expensive
	// when there are many contacts, so best avoiding it
	stmt, err := tx.Prepare(`
		INSERT INTO contacts(
			id,
			address,
			alias,
			display_name,
			identicon,
			last_updated,
			last_updated_locally,
			local_nickname,
			added,
			blocked,
			removed,
			has_added_us,
			name,
			photo,
			tribute_to_talk
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		contact.ID,
		contact.Address,
		contact.Alias,
		contact.DisplayName,
		contact.Identicon,
		contact.LastUpdated,
		contact.LastUpdatedLocally,
		contact.LocalNickname,
		contact.Added,
		contact.Blocked,
		contact.Removed,
		contact.HasAddedUs,
		//TODO we need to drop these columns
		"",
		"",
		"",
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

func (db sqlitePersistence) GetWhenChatIdentityLastPublished(chatID string) (t int64, hash []byte, err error) {
	rows, err := db.db.Query("SELECT clock_value, hash FROM chat_identity_last_published WHERE chat_id = ?", chatID)
	if err != nil {
		return t, nil, err
	}
	defer func() {
		err = rows.Close()
	}()

	for rows.Next() {
		err = rows.Scan(&t, &hash)
		if err != nil {
			return t, nil, err
		}
	}

	return t, hash, nil
}

func (db sqlitePersistence) SaveWhenChatIdentityLastPublished(chatID string, hash []byte) (err error) {
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

	stmt, err := tx.Prepare("INSERT INTO chat_identity_last_published (chat_id, clock_value, hash) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, time.Now().Unix(), hash)
	if err != nil {
		return err
	}

	return nil
}

func (db sqlitePersistence) ResetWhenChatIdentityLastPublished(chatID string) (err error) {
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

	stmt, err := tx.Prepare("INSERT INTO chat_identity_last_published (chat_id, clock_value, hash) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chatID, 0, []byte("."))
	if err != nil {
		return err
	}

	return nil
}

func (db sqlitePersistence) InsertStatusUpdate(userStatus UserStatus) error {
	_, err := db.db.Exec(`INSERT INTO status_updates(
		public_key,
		status_type,
		clock,
		custom_text)
		VALUES (?, ?, ?, ?)`,
		userStatus.PublicKey,
		userStatus.StatusType,
		userStatus.Clock,
		userStatus.CustomText,
	)

	return err
}

func (db sqlitePersistence) CleanOlderStatusUpdates() error {
	now := time.Now()
	twoWeeksAgo := now.Add(time.Duration(-1*HoursInTwoWeeks) * time.Hour)
	_, err := db.db.Exec(`DELETE FROM status_updates WHERE clock < ?`,
		uint64(twoWeeksAgo.Unix()),
	)

	return err
}

func (db sqlitePersistence) StatusUpdates() (statusUpdates []UserStatus, err error) {
	rows, err := db.db.Query(`
		SELECT
			public_key,
			status_type,
			clock,
			custom_text
		FROM status_updates
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var userStatus UserStatus
		err = rows.Scan(
			&userStatus.PublicKey,
			&userStatus.StatusType,
			&userStatus.Clock,
			&userStatus.CustomText,
		)
		if err != nil {
			return
		}
		statusUpdates = append(statusUpdates, userStatus)
	}

	return
}

func (db sqlitePersistence) getReceivedContactRequest(tx *sql.Tx, pk string) (*ContactRequest, error) {
	var err error
	if tx == nil {
		tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
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
	}
	contactRequest := &ContactRequest{
		SigningKey: pk,
	}
	err = tx.QueryRow(`
		SELECT
			signature,
			timestamp
		FROM contact_requests
		WHERE signing_key = ?
	`, pk).Scan(&contactRequest.Signature,
		&contactRequest.Timestamp,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contactRequest, nil
}

func (db sqlitePersistence) GetReceivedContactRequest(pk string) (*ContactRequest, error) {
	return db.getReceivedContactRequest(nil, pk)
}

func (db sqlitePersistence) SaveReceivedContactRequest(cr *ContactRequest) error {
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

	contactRequest, err := db.getReceivedContactRequest(tx, cr.SigningKey)
	if err != nil {
		return err
	}
	// Nothing to do
	if contactRequest != nil && contactRequest.Timestamp >= cr.Timestamp {
		return nil
	}

	_, err = tx.Exec(`INSERT INTO contact_requests(signing_key, contact_key, signature, timestamp) VALUES(?,?,?,?)`, cr.SigningKey, cr.ContactKey, cr.Signature, cr.Timestamp)

	return err
}
