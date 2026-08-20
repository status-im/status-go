package mailservers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"

	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

type MailserverTopic struct {
	PubsubTopic  string   `json:"pubsubTopic"`
	ContentTopic string   `json:"topic"`
	Discovery    bool     `json:"discovery?"`
	Negotiated   bool     `json:"negotiated?"`
	ChatIDs      []string `json:"chat-ids"`
	// LastRequest is the timestamp through which this topic is known complete.
	// Zero means the topic has not completed its initial history fetch yet.
	LastRequest int `json:"last-request"`
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
		_, err = tx.Exec(`INSERT INTO mailserver_topics(
			  pubsub_topic,
			  topic,
			  chat_ids,
			  last_request,
			  discovery,
			  negotiated
		  ) VALUES (?, ?, ?, ?, ?, ?)
		  ON CONFLICT(topic, pubsub_topic) DO UPDATE SET
			  chat_ids = excluded.chat_ids,
			  last_request = MAX(mailserver_topics.last_request, excluded.last_request),
			  discovery = excluded.discovery,
			  negotiated = excluded.negotiated`,
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

// AdvanceHistoryCursors marks every initialized topic complete through the
// given timestamp. Topics with a zero cursor are deliberately left untouched
// so a reliable live connection cannot suppress their initial history fetch.
func (d *Database) AdvanceHistoryCursors(through int) error {
	_, err := d.db.Exec(`
		UPDATE mailserver_topics
		SET last_request = ?
		WHERE last_request > 0 AND last_request < ?
	`, through, through)
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

	// Insert if not existing
	for _, filter := range filters {
		// fetch
		var topic string
		err = tx.QueryRow(`SELECT topic FROM mailserver_topics WHERE topic = ? AND pubsub_topic = ?`, filter.ContentTopic().String(), filter.PubsubTopic()).Scan(&topic)
		if err != nil && err != sql.ErrNoRows {
			return
		} else if err == sql.ErrNoRows {
			// we insert the topic
			_, err = tx.Exec(`INSERT INTO mailserver_topics(topic,pubsub_topic,last_request,discovery,negotiated) VALUES (?,?,?,?,?)`, filter.ContentTopic().String(), filter.PubsubTopic(), 0, filter.IsDiscovery(), filter.IsNegotiated())
		}
		if err != nil {
			return
		}
	}

	return
}
