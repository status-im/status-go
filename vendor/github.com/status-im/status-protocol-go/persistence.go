package statusproto

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"

	protocol "github.com/status-im/status-protocol-go/v1"
)

const (
	uniqueIDContstraint = "UNIQUE constraint failed: user_messages.id"
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

func (db sqlitePersistence) SaveMessages(chatID string, messages []*protocol.Message) (last int64, err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return
	}
	stmt, err = tx.Prepare(`INSERT INTO user_messages(
id, chat_id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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

	var rst sql.Result

	for _, msg := range messages {
		pkey := []byte{}
		if msg.SigPubKey != nil {
			pkey, err = marshalECDSAPub(msg.SigPubKey)
		}
		rst, err = stmt.Exec(
			msg.ID, chatID, msg.ContentT, msg.MessageT, msg.Text,
			msg.Clock, msg.Timestamp, msg.Content.ChatID, msg.Content.Text,
			pkey, msg.Flags)
		if err != nil {
			if err.Error() == uniqueIDContstraint {
				// skip duplicated messages
				err = nil
				continue
			}
			return
		}

		last, err = rst.LastInsertId()
		if err != nil {
			return
		}
	}
	return
}

// Messages returns messages for a given contact, in a given period. Ordered by a timestamp.
func (db sqlitePersistence) Messages(chatID string, from, to time.Time) (result []*protocol.Message, err error) {
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags
FROM user_messages WHERE chat_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`,
		chatID, protocol.TimestampInMsFromTime(from), protocol.TimestampInMsFromTime(to))
	if err != nil {
		return nil, err
	}
	var (
		rst = []*protocol.Message{}
	)
	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = unmarshalECDSAPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

func (db sqlitePersistence) NewMessages(chatID string, rowid int64) ([]*protocol.Message, error) {
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags
FROM user_messages WHERE chat_id = ? AND rowid >= ? ORDER BY clock`,
		chatID, rowid)
	if err != nil {
		return nil, err
	}
	var (
		rst = []*protocol.Message{}
	)
	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = unmarshalECDSAPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

// TODO(adam): refactor all message getters in order not to
// repeat the select fields over and over.
func (db sqlitePersistence) UnreadMessages(chatID string) ([]*protocol.Message, error) {
	rows, err := db.db.Query(`
		SELECT
			id,
			content_type,
			message_type,
			text,
			clock,
			timestamp,
			content_chat_id,
			content_text,
			public_key,
			flags
		FROM
			user_messages
		WHERE
			chat_id = ? AND
			flags & ? == 0
		ORDER BY clock`,
		chatID, protocol.MessageRead,
	)
	if err != nil {
		return nil, err
	}

	var result []*protocol.Message

	for rows.Next() {
		msg := protocol.Message{
			Content: protocol.Content{},
		}
		pkey := []byte{}
		err = rows.Scan(
			&msg.ID, &msg.ContentT, &msg.MessageT, &msg.Text, &msg.Clock,
			&msg.Timestamp, &msg.Content.ChatID, &msg.Content.Text, &pkey, &msg.Flags)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			msg.SigPubKey, err = unmarshalECDSAPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, &msg)
	}

	return result, nil
}

func marshalECDSAPub(pub *ecdsa.PublicKey) (rst []byte, err error) {
	switch pub.Curve.(type) {
	case *secp256k1.BitCurve:
		rst = make([]byte, 34)
		rst[0] = 1
		copy(rst[1:], secp256k1.CompressPubkey(pub.X, pub.Y))
		return rst[:], nil
	default:
		return nil, errors.New("unknown curve")
	}
}

func unmarshalECDSAPub(buf []byte) (*ecdsa.PublicKey, error) {
	pub := &ecdsa.PublicKey{}
	if len(buf) < 1 {
		return nil, errors.New("too small")
	}
	switch buf[0] {
	case 1:
		pub.Curve = secp256k1.S256()
		pub.X, pub.Y = secp256k1.DecompressPubkey(buf[1:])
		ok := pub.IsOnCurve(pub.X, pub.Y)
		if !ok {
			return nil, errors.New("not on curve")
		}
		return pub, nil
	default:
		return nil, errors.New("unknown curve")
	}
}
