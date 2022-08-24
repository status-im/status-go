package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	gowakuPersistence "github.com/status-im/go-waku/waku/persistence"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/utils"

	"go.uber.org/zap"
)

var ErrInvalidCursor = errors.New("invalid cursor")

// DBStore is a MessageProvider that has a *sql.DB connection
type DBStore struct {
	db  *sql.DB
	log *zap.Logger

	maxMessages int
	maxDuration time.Duration

	wg   sync.WaitGroup
	quit chan struct{}
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

// WithRetentionPolicy is a DBOption that specifies the max number of messages
// to be stored and duration before they're removed from the message store
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
func NewDBStore(log *zap.Logger, options ...DBOption) (*DBStore, error) {
	result := new(DBStore)
	result.log = log.Named("dbstore")
	result.quit = make(chan struct{})

	for _, opt := range options {
		err := opt(result)
		if err != nil {
			return nil, err
		}
	}

	err := result.cleanOlderRecords()
	if err != nil {
		return nil, err
	}

	result.wg.Add(1)
	go result.checkForOlderRecords(10 * time.Second) // is 10s okay?

	return result, nil
}

func (d *DBStore) cleanOlderRecords() error {
	d.log.Debug("Cleaning older records...")

	// Delete older messages
	if d.maxDuration > 0 {
		start := time.Now()
		sqlStmt := `DELETE FROM store_messages WHERE receiverTimestamp < ?`
		_, err := d.db.Exec(sqlStmt, utils.GetUnixEpochFrom(time.Now().Add(-d.maxDuration)))
		if err != nil {
			return err
		}
		elapsed := time.Since(start)
		d.log.Debug("deleting older records from the DB", zap.Duration("duration", elapsed))
	}

	// Limit number of records to a max N
	if d.maxMessages > 0 {
		start := time.Now()
		sqlStmt := `DELETE FROM store_messages WHERE id IN (SELECT id FROM store_messages ORDER BY receiverTimestamp DESC LIMIT -1 OFFSET ?)`
		_, err := d.db.Exec(sqlStmt, d.maxMessages)
		if err != nil {
			return err
		}
		elapsed := time.Since(start)
		d.log.Debug("deleting excess records from the DB", zap.Duration("duration", elapsed))
	}

	return nil
}

func (d *DBStore) checkForOlderRecords(t time.Duration) {
	defer d.wg.Done()

	ticker := time.NewTicker(t)
	defer ticker.Stop()

	for {
		select {
		case <-d.quit:
			return
		case <-ticker.C:
			err := d.cleanOlderRecords()
			if err != nil {
				d.log.Error("cleaning older records", zap.Error(err))
			}
		}
	}
}

// Stop closes a DB connection
func (d *DBStore) Stop() {
	d.quit <- struct{}{}
	d.wg.Wait()
	d.db.Close()
}

// Put inserts a WakuMessage into the DB
func (d *DBStore) Put(env *protocol.Envelope) error {
	stmt, err := d.db.Prepare("INSERT INTO store_messages (id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	cursor := env.Index()
	dbKey := gowakuPersistence.NewDBKey(uint64(cursor.SenderTime), env.PubsubTopic(), env.Index().Digest)
	_, err = stmt.Exec(dbKey.Bytes(), cursor.ReceiverTime, env.Message().Timestamp, env.Message().ContentTopic, env.PubsubTopic(), env.Message().Payload, env.Message().Version)
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return nil
}

// Query retrieves messages from the DB
func (d *DBStore) Query(query *pb.HistoryQuery) ([]gowakuPersistence.StoredMessage, error) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		d.log.Info(fmt.Sprintf("Loading records from the DB took %s", elapsed))
	}()

	sqlQuery := `SELECT id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version 
					 FROM store_messages 
					 %s
					 ORDER BY senderTimestamp %s, pubsubTopic, id %s
					 LIMIT ?`

	var conditions []string
	var parameters []interface{}

	if query.PubsubTopic != "" {
		conditions = append(conditions, "pubsubTopic = ?")
		parameters = append(parameters, query.PubsubTopic)
	}

	if query.StartTime != 0 {
		conditions = append(conditions, "id >= ?")
		startTimeDBKey := gowakuPersistence.NewDBKey(uint64(query.StartTime), "", []byte{})
		parameters = append(parameters, startTimeDBKey.Bytes())

	}

	if query.EndTime != 0 {
		conditions = append(conditions, "id <= ?")
		endTimeDBKey := gowakuPersistence.NewDBKey(uint64(query.EndTime), "", []byte{})
		parameters = append(parameters, endTimeDBKey.Bytes())
	}

	if len(query.ContentFilters) != 0 {
		var ctPlaceHolder []string
		for _, ct := range query.ContentFilters {
			if ct.ContentTopic != "" {
				ctPlaceHolder = append(ctPlaceHolder, "?")
				parameters = append(parameters, ct.ContentTopic)
			}
		}
		conditions = append(conditions, "contentTopic IN ("+strings.Join(ctPlaceHolder, ", ")+")")
	}

	if query.PagingInfo.Cursor != nil {
		var exists bool
		cursorDBKey := gowakuPersistence.NewDBKey(uint64(query.PagingInfo.Cursor.SenderTime), query.PagingInfo.Cursor.PubsubTopic, query.PagingInfo.Cursor.Digest)

		err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM store_messages WHERE id = ?)",
			cursorDBKey.Bytes(),
		).Scan(&exists)

		if err != nil {
			return nil, err
		}

		if exists {
			eqOp := ">"
			if query.PagingInfo.Direction == pb.PagingInfo_BACKWARD {
				eqOp = "<"
			}
			conditions = append(conditions, fmt.Sprintf("id %s ?", eqOp))

			parameters = append(parameters, cursorDBKey.Bytes())
		} else {
			return nil, ErrInvalidCursor
		}
	}

	conditionStr := ""
	if len(conditions) != 0 {
		conditionStr = "WHERE " + strings.Join(conditions, " AND ")
	}

	orderDirection := "ASC"
	if query.PagingInfo.Direction == pb.PagingInfo_BACKWARD {
		orderDirection = "DESC"
	}

	sqlQuery = fmt.Sprintf(sqlQuery, conditionStr, orderDirection, orderDirection)

	stmt, err := d.db.Prepare(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	parameters = append(parameters, query.PagingInfo.PageSize)
	rows, err := stmt.Query(parameters...)
	if err != nil {
		return nil, err
	}

	var result []gowakuPersistence.StoredMessage
	for rows.Next() {
		record, err := d.GetStoredMessage(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}

	defer rows.Close()

	return result, nil
}

// MostRecentTimestamp returns an unix timestamp with the most recent senderTimestamp
// in the message table
func (d *DBStore) MostRecentTimestamp() (int64, error) {
	result := sql.NullInt64{}

	err := d.db.QueryRow(`SELECT max(senderTimestamp) FROM store_messages`).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return result.Int64, nil
}

// Count returns the number of rows in the message table
func (d *DBStore) Count() (int, error) {
	var result int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM store_messages`).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return result, nil
}

// GetAll returns all the stored WakuMessages
func (d *DBStore) GetAll() ([]gowakuPersistence.StoredMessage, error) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		d.log.Info("loading records from the DB", zap.Duration("duration", elapsed))
	}()

	rows, err := d.db.Query("SELECT id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version FROM store_messages ORDER BY senderTimestamp ASC")
	if err != nil {
		return nil, err
	}

	var result []gowakuPersistence.StoredMessage

	defer rows.Close()

	for rows.Next() {
		record, err := d.GetStoredMessage(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}

	d.log.Info("DB returned records", zap.Int("count", len(result)))

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetStoredMessage is a helper function used to convert a `*sql.Rows` into a `StoredMessage`
func (d *DBStore) GetStoredMessage(row *sql.Rows) (gowakuPersistence.StoredMessage, error) {
	var id []byte
	var receiverTimestamp int64
	var senderTimestamp int64
	var contentTopic string
	var payload []byte
	var version uint32
	var pubsubTopic string

	err := row.Scan(&id, &receiverTimestamp, &senderTimestamp, &contentTopic, &pubsubTopic, &payload, &version)
	if err != nil {
		d.log.Error("scanning messages from db", zap.Error(err))
		return gowakuPersistence.StoredMessage{}, err
	}

	msg := new(pb.WakuMessage)
	msg.ContentTopic = contentTopic
	msg.Payload = payload
	msg.Timestamp = senderTimestamp
	msg.Version = version

	record := gowakuPersistence.StoredMessage{
		ID:           id,
		PubsubTopic:  pubsubTopic,
		ReceiverTime: receiverTimestamp,
		Message:      msg,
	}

	return record, nil
}
