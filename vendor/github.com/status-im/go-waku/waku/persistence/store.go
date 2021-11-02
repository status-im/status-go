package persistence

import (
	"database/sql"
	"log"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
)

type MessageProvider interface {
	GetAll() ([]StoredMessage, error)
	Put(cursor *pb.Index, pubsubTopic string, message *pb.WakuMessage) error
	Stop()
}

// DBStore is a MessageProvider that has a *sql.DB connection
type DBStore struct {
	MessageProvider
	db *sql.DB
}

type StoredMessage struct {
	ID           []byte
	PubsubTopic  string
	ReceiverTime float64
	Message      *pb.WakuMessage
}

// DBOption is an optional setting that can be used to configure the DBStore
type DBOption func(*DBStore) error

// WithDB is a DBOption that lets you use any custom *sql.DB with a DBStore.
func WithDB(db *sql.DB) DBOption {
	return func(d *DBStore) error {
		d.db = db
		return nil
	}
}

// WithDriver is a DBOption that will open a *sql.DB connection
func WithDriver(driverName string, datasourceName string) DBOption {
	return func(d *DBStore) error {
		db, err := sql.Open(driverName, datasourceName)
		if err != nil {
			return err
		}
		d.db = db
		return nil
	}
}

// Creates a new DB store using the db specified via options.
// It will create a messages table if it does not exist
func NewDBStore(opt DBOption) (*DBStore, error) {
	result := new(DBStore)

	err := opt(result)
	if err != nil {
		return nil, err
	}

	err = result.createTable()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DBStore) createTable() error {
	sqlStmt := `CREATE TABLE IF NOT EXISTS message (
		id BLOB PRIMARY KEY,
		receiverTimestamp REAL NOT NULL,
		senderTimestamp REAL NOT NULL,
		contentTopic BLOB NOT NULL,
		pubsubTopic BLOB NOT NULL,
		payload BLOB,
		version INTEGER NOT NULL DEFAULT 0
	) WITHOUT ROWID;`
	_, err := d.db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}

// Closes a DB connection
func (d *DBStore) Stop() {
	d.db.Close()
}

// Inserts a WakuMessage into the DB
func (d *DBStore) Put(cursor *pb.Index, pubsubTopic string, message *pb.WakuMessage) error {
	stmt, err := d.db.Prepare("INSERT INTO message (id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(cursor.Digest, cursor.ReceiverTime, message.Timestamp, message.ContentTopic, pubsubTopic, message.Payload, message.Version)
	if err != nil {
		return err
	}

	return nil
}

// Returns all the stored WakuMessages
func (d *DBStore) GetAll() ([]StoredMessage, error) {
	rows, err := d.db.Query("SELECT id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version FROM message ORDER BY senderTimestamp ASC")
	if err != nil {
		return nil, err
	}

	var result []StoredMessage

	defer rows.Close()

	for rows.Next() {
		var id []byte
		var receiverTimestamp float64
		var senderTimestamp float64
		var contentTopic string
		var payload []byte
		var version uint32
		var pubsubTopic string

		err = rows.Scan(&id, &receiverTimestamp, &senderTimestamp, &contentTopic, &pubsubTopic, &payload, &version)
		if err != nil {
			log.Fatal(err)
		}

		msg := new(pb.WakuMessage)
		msg.ContentTopic = contentTopic
		msg.Payload = payload
		msg.Timestamp = senderTimestamp
		msg.Version = version

		record := StoredMessage{
			ID:           id,
			PubsubTopic:  pubsubTopic,
			ReceiverTime: receiverTimestamp,
			Message:      msg,
		}

		result = append(result, record)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}
