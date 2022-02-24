package persistence

import (
	"database/sql"
	"time"

	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type MessageProvider interface {
	GetAll() ([]StoredMessage, error)
	Put(cursor *pb.Index, pubsubTopic string, message *pb.WakuMessage) error
	Stop()
}

// DBStore is a MessageProvider that has a *sql.DB connection
type DBStore struct {
	MessageProvider
	db  *sql.DB
	log *zap.SugaredLogger

	maxMessages int
	maxDuration time.Duration
}

type StoredMessage struct {
	ID           []byte
	PubsubTopic  string
	ReceiverTime int64
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

func WithRetentionPolicy(maxMessages int, maxDuration time.Duration) DBOption {
	return func(d *DBStore) error {
		d.maxDuration = maxDuration
		d.maxMessages = maxMessages
		return nil
	}
}

// Creates a new DB store using the db specified via options.
// It will create a messages table if it does not exist and
// clean up records according to the retention policy used
func NewDBStore(log *zap.SugaredLogger, options ...DBOption) (*DBStore, error) {
	result := new(DBStore)
	result.log = log.Named("dbstore")

	for _, opt := range options {
		err := opt(result)
		if err != nil {
			return nil, err
		}
	}

	err := result.createTable()
	if err != nil {
		return nil, err
	}

	err = result.cleanOlderRecords()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DBStore) createTable() error {
	sqlStmt := `CREATE TABLE IF NOT EXISTS message (
		id BLOB PRIMARY KEY,
		receiverTimestamp INTEGER NOT NULL,
		senderTimestamp INTEGER NOT NULL,
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

func (d *DBStore) cleanOlderRecords() error {
	// Delete older messages
	if d.maxDuration > 0 {
		sqlStmt := `DELETE FROM message WHERE receiverTimestamp < ?`
		_, err := d.db.Exec(sqlStmt, utils.GetUnixEpochFrom(time.Now().Add(-d.maxDuration)))
		if err != nil {
			return err
		}
	}

	// Limit number of records to a max N
	if d.maxMessages > 0 {
		sqlStmt := `DELETE FROM message WHERE id IN (SELECT id FROM message ORDER BY receiverTimestamp DESC LIMIT -1 OFFSET 5)`
		_, err := d.db.Exec(sqlStmt, d.maxMessages)
		if err != nil {
			return err
		}
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
		var receiverTimestamp int64
		var senderTimestamp int64
		var contentTopic string
		var payload []byte
		var version uint32
		var pubsubTopic string

		err = rows.Scan(&id, &receiverTimestamp, &senderTimestamp, &contentTopic, &pubsubTopic, &payload, &version)
		if err != nil {
			d.log.Fatal(err)
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
