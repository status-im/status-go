package mailservers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	messagingtypes "github.com/status-im/status-go/messaging/types"
)

type MailserverTopic struct {
	PubsubTopic  string   `json:"pubsubTopic"`
	ContentTopic string   `json:"topic"`
	Discovery    bool     `json:"discovery?"`
	Negotiated   bool     `json:"negotiated?"`
	ChatIDs      []string `json:"chat-ids"`
	LastRequest  int      `json:"last-request"` // default is 1
}

// sqlStringSlice helps to serialize a slice of strings into a single column using JSON serialization.
type sqlStringSlice []string

// Scan implements the Scanner interface.
func (ss *sqlStringSlice) Scan(value interface{}) error {
	if value == nil {
		*ss = nil
		return nil
	}
	src, ok := value.([]byte)
	if !ok {
		return errors.New("invalid value type, expected byte slice")
	}
	return json.Unmarshal(src, ss)
}

// Value implements the driver Valuer interface.
func (ss sqlStringSlice) Value() (driver.Value, error) {
	return json.Marshal(ss)
}

// Database sql wrapper for operations with mailserver objects.
type Database struct {
	db *sql.DB
}

func NewDB(db *sql.DB) *Database {
	return &Database{db: db}
}

func (d *Database) AddTopics(topics []MailserverTopic) (err error) {
	var tx *sql.Tx
	tx, err = d.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	for _, topic := range topics {
		chatIDs := sqlStringSlice(topic.ChatIDs)
		_, err = tx.Exec(`INSERT OR REPLACE INTO mailserver_topics(
			  pubsub_topic,
			  topic,
			  chat_ids,
			  last_request,
			  discovery,
			  negotiated
		  ) VALUES (?, ?, ?, ?, ?, ?)`,
			topic.PubsubTopic,
			topic.ContentTopic,
			chatIDs,
			topic.LastRequest,
			topic.Discovery,
			topic.Negotiated,
		)
		if err != nil {
			return
		}
	}
	return
}

func (d *Database) Topics() ([]MailserverTopic, error) {
	var result []MailserverTopic

	rows, err := d.db.Query(`SELECT pubsub_topic, topic, chat_ids, last_request,discovery,negotiated FROM mailserver_topics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			t       MailserverTopic
			chatIDs sqlStringSlice
		)
		if err := rows.Scan(
			&t.PubsubTopic,
			&t.ContentTopic,
			&chatIDs,
			&t.LastRequest,
			&t.Discovery,
			&t.Negotiated,
		); err != nil {
			return nil, err
		}
		t.ChatIDs = chatIDs
		result = append(result, t)
	}

	return result, nil
}

func (d *Database) ResetLastRequest(pubsubTopic, contentTopic string) error {
	_, err := d.db.Exec("UPDATE mailserver_topics SET last_request = 0 WHERE pubsub_topic = ? AND topic = ?", pubsubTopic, contentTopic)
	return err
}

// SetTopics deletes all topics excepts the one set, or upsert those if
// missing
func (d *Database) SetTopics(filters messagingtypes.ChatFilters) (err error) {
	var tx *sql.Tx
	tx, err = d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	if len(filters) == 0 {
		return nil
	}

	contentTopicsPerPubsubTopic := make(map[string]map[string]struct{})
	for _, filter := range filters {
		contentTopics, ok := contentTopicsPerPubsubTopic[filter.PubsubTopic()]
		if !ok {
			contentTopics = make(map[string]struct{})
		}
		contentTopics[filter.ContentTopic().String()] = struct{}{}
		contentTopicsPerPubsubTopic[filter.PubsubTopic()] = contentTopics
	}

	for pubsubTopic, contentTopics := range contentTopicsPerPubsubTopic {
		topicsArgs := make([]interface{}, 0, len(contentTopics)+1)
		topicsArgs = append(topicsArgs, pubsubTopic)
		for ct := range contentTopics {
			topicsArgs = append(topicsArgs, ct)
		}

		inVector := strings.Repeat("?, ", len(contentTopics)-1) + "?"

		// Delete topics
		query := "DELETE FROM mailserver_topics WHERE pubsub_topic = ? AND topic NOT IN (" + inVector + ")" // nolint: gosec
		_, err = tx.Exec(query, topicsArgs...)
	}

	// Default to now - 1.day
	lastRequest := (time.Now().Add(-24 * time.Hour)).Unix()
	// Insert if not existing
	for _, filter := range filters {
		// fetch
		var topic string
		err = tx.QueryRow(`SELECT topic FROM mailserver_topics WHERE topic = ? AND pubsub_topic = ?`, filter.ContentTopic().String(), filter.PubsubTopic()).Scan(&topic)
		if err != nil && err != sql.ErrNoRows {
			return
		} else if err == sql.ErrNoRows {
			// we insert the topic
			_, err = tx.Exec(`INSERT INTO mailserver_topics(topic,pubsub_topic,last_request,discovery,negotiated) VALUES (?,?,?,?,?)`, filter.ContentTopic().String(), filter.PubsubTopic(), lastRequest, filter.IsDiscovery(), filter.IsNegotiated())
		}
		if err != nil {
			return
		}
	}

	return
}
