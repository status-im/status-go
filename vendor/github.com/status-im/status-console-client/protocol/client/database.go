package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/status-im/status-console-client/protocol/client/migrations"
	"github.com/status-im/status-console-client/protocol/client/sqlite"
	"github.com/status-im/status-console-client/protocol/v1"
)

func marshalEcdsaPub(pub *ecdsa.PublicKey) (rst []byte, err error) {
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

func unmarshalEcdsaPub(buf []byte) (*ecdsa.PublicKey, error) {
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

const (
	uniqueIDContstraint = "UNIQUE constraint failed: user_messages.id"
)

var (
	// ErrMsgAlreadyExist returned if msg already exist.
	ErrMsgAlreadyExist = errors.New("message with given ID already exist")
)

// Database is an interface for all db operations.
type Database interface {
	Close() error
	Messages(c Contact, from, to time.Time) ([]*protocol.Message, error)
	NewMessages(c Contact, rowid int64) ([]*protocol.Message, error)
	UnreadMessages(c Contact) ([]*protocol.Message, error)
	SaveMessages(c Contact, messages []*protocol.Message) (int64, error)
	LastMessageClock(Contact) (int64, error)
	Contacts() ([]Contact, error)
	SaveContacts(contacts []Contact) error
	DeleteContact(Contact) error
	ContactExist(Contact) (bool, error)
	PublicContactExist(Contact) (bool, error)
	Histories() ([]History, error)
	UpdateHistories([]History) error
}

// Migrate applies migrations.
func Migrate(db *sql.DB) error {
	resources := bindata.Resource(
		migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	log.Printf("[Migrate] applying migrations %s", migrations.AssetNames())

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// InitializeTmpDB creates database in temporary directory with a random key.
// Used for tests.
func InitializeTmpDB() (TmpDatabase, error) {
	tmpfile, err := ioutil.TempFile("", "client-tests-")
	if err != nil {
		return TmpDatabase{}, err
	}
	pass := make([]byte, 4)
	_, err = rand.Read(pass)
	if err != nil {
		return TmpDatabase{}, err
	}
	db, err := InitializeDB(tmpfile.Name(), hex.EncodeToString(pass))
	if err != nil {
		return TmpDatabase{}, err
	}
	return TmpDatabase{
		SQLLiteDatabase: db,
		file:            tmpfile,
	}, nil
}

// TmpDatabase wraps SQLLiteDatabase and removes temporary file after db was closed.
type TmpDatabase struct {
	SQLLiteDatabase
	file *os.File
}

// Close closes sqlite database and removes temporary file.
func (db TmpDatabase) Close() error {
	_ = db.SQLLiteDatabase.Close()
	return os.Remove(db.file.Name())
}

// InitializeDB opens encrypted sqlite database from provided path and applies migrations.
func InitializeDB(path, key string) (SQLLiteDatabase, error) {
	db, err := sqlite.OpenDB(path, key)
	if err != nil {
		return SQLLiteDatabase{}, err
	}
	err = Migrate(db)
	if err != nil {
		return SQLLiteDatabase{}, err
	}
	return SQLLiteDatabase{db: db}, nil
}

// SQLLiteDatabase wrapper around sql db with operations common for a client.
type SQLLiteDatabase struct {
	db *sql.DB
}

// Close closes internal sqlite database.
func (db SQLLiteDatabase) Close() error {
	return db.db.Close()
}

// SaveContacts inserts or replaces provided contacts.
// TODO should it delete all previous contacts?
func (db SQLLiteDatabase) SaveContacts(contacts []Contact) (err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	stmt, err = tx.Prepare("INSERT INTO user_contacts(id, name, type, state, topic, public_key) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	history, err := tx.Prepare("INSERT INTO history_user_contact_topic(contact_id) VALUES (?)")
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()
	for i := range contacts {
		pkey := []byte{}
		if contacts[i].PublicKey != nil {
			pkey, err = marshalEcdsaPub(contacts[i].PublicKey)
			if err != nil {
				return err
			}
		}
		id := contactID(contacts[i])
		_, err = stmt.Exec(id, contacts[i].Name, contacts[i].Type, contacts[i].State, contacts[i].Topic, pkey)
		if err != nil {
			return err
		}
		// to avoid unmarshalling null into sql.NullInt64
		_, err = history.Exec(id)
		if err != nil {
			return err
		}
	}
	return err
}

// Contacts returns all available contacts.
func (db SQLLiteDatabase) Contacts() ([]Contact, error) {
	rows, err := db.db.Query("SELECT name, type, state, topic, public_key FROM user_contacts")
	if err != nil {
		return nil, err
	}

	var (
		rst = []Contact{}
	)
	for rows.Next() {
		// do not reuse same gob instance. same instance marshalls two same objects differently
		// if used repetitively.
		contact := Contact{}
		pkey := []byte{}
		err = rows.Scan(&contact.Name, &contact.Type, &contact.State, &contact.Topic, &pkey)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			contact.PublicKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, contact)
	}
	return rst, nil
}

func (db SQLLiteDatabase) Histories() ([]History, error) {
	rows, err := db.db.Query("SELECT synced, u.name, u.type, u.state, u.topic, u.public_key FROM history_user_contact_topic JOIN user_contacts u ON contact_id = u.id")
	if err != nil {
		return nil, err
	}
	rst := []History{}
	for rows.Next() {
		h := History{
			Contact: Contact{},
		}
		pkey := []byte{}
		err = rows.Scan(&h.Synced, &h.Contact.Name, &h.Contact.Type, &h.Contact.State, &h.Contact.Topic, &pkey)
		if err != nil {
			return nil, err
		}
		if len(pkey) != 0 {
			h.Contact.PublicKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, h)
	}
	return rst, nil
}

func (db SQLLiteDatabase) UpdateHistories(histories []History) (err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	stmt, err = tx.Prepare("UPDATE history_user_contact_topic SET synced = ? WHERE contact_Id = ?")
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}()
	for i := range histories {
		_, err = stmt.Exec(histories[i].Synced, contactID(histories[i].Contact))
		if err != nil {
			return err
		}
	}
	return nil
}

func (db SQLLiteDatabase) DeleteContact(c Contact) error {
	_, err := db.db.Exec("DELETE FROM user_contacts WHERE id = ?", fmt.Sprintf("%s:%d", c.Name, c.Type))
	if err != nil {
		return errors.Wrap(err, "error deleting contact from db")
	}
	return nil
}

func (db SQLLiteDatabase) ContactExist(c Contact) (exists bool, err error) {
	err = db.db.QueryRow("SELECT EXISTS(SELECT id FROM user_contacts WHERE id = ?)", contactID(c)).Scan(&exists)
	return
}

func (db SQLLiteDatabase) PublicContactExist(c Contact) (exists bool, err error) {
	var pkey []byte
	if c.PublicKey != nil {
		pkey, err = marshalEcdsaPub(c.PublicKey)
		if err != nil {
			return false, err
		}
	} else {
		return false, errors.New("no public key")
	}
	err = db.db.QueryRow("SELECT EXISTS(SELECT id FROM user_contacts WHERE public_key = ?)", pkey).Scan(&exists)
	return
}

func (db SQLLiteDatabase) LastMessageClock(c Contact) (int64, error) {
	var last sql.NullInt64
	err := db.db.QueryRow("SELECT max(clock) FROM user_messages WHERE contact_id = ?", contactID(c)).Scan(&last)
	if err != nil {
		return 0, err
	}
	return last.Int64, nil
}

func (db SQLLiteDatabase) SaveMessages(c Contact, messages []*protocol.Message) (last int64, err error) {
	var (
		tx   *sql.Tx
		stmt *sql.Stmt
	)
	tx, err = db.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return
	}
	stmt, err = tx.Prepare(`INSERT INTO user_messages(
id, contact_id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		} else {
			// don't shadow original error
			_ = tx.Rollback()
			return
		}
	}()

	var (
		contactID = fmt.Sprintf("%s:%d", c.Name, c.Type)
		rst       sql.Result
	)
	for _, msg := range messages {
		pkey := []byte{}
		if msg.SigPubKey != nil {
			pkey, err = marshalEcdsaPub(msg.SigPubKey)
		}
		rst, err = stmt.Exec(
			msg.ID, contactID, msg.ContentT, msg.MessageT, msg.Text,
			msg.Clock, msg.Timestamp, msg.Content.ChatID, msg.Content.Text,
			pkey, msg.Flags)
		if err != nil {
			if err.Error() == uniqueIDContstraint {
				err = ErrMsgAlreadyExist
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
func (db SQLLiteDatabase) Messages(c Contact, from, to time.Time) (result []*protocol.Message, err error) {
	contactID := fmt.Sprintf("%s:%d", c.Name, c.Type)
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags
FROM user_messages WHERE contact_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp`,
		contactID, protocol.TimestampInMsFromTime(from), protocol.TimestampInMsFromTime(to))
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
			msg.SigPubKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		rst = append(rst, &msg)
	}
	return rst, nil
}

func (db SQLLiteDatabase) NewMessages(c Contact, rowid int64) ([]*protocol.Message, error) {
	contactID := contactID(c)
	rows, err := db.db.Query(`SELECT
id, content_type, message_type, text, clock, timestamp, content_chat_id, content_text, public_key, flags
FROM user_messages WHERE contact_id = ? AND rowid >= ? ORDER BY clock`,
		contactID, rowid)
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
			msg.SigPubKey, err = unmarshalEcdsaPub(pkey)
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
func (db SQLLiteDatabase) UnreadMessages(c Contact) ([]*protocol.Message, error) {
	contactID := contactID(c)
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
			contact_id = ? AND
			flags & ? == 0
		ORDER BY clock`,
		contactID, protocol.MessageRead,
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
			msg.SigPubKey, err = unmarshalEcdsaPub(pkey)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, &msg)
	}

	return result, nil
}

func contactID(c Contact) string {
	return fmt.Sprintf("%s:%d", c.Name, c.Type)
}
