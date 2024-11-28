package mailservers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/protocol/transport"
)

func MustDecodeENR(enrStr string) *enode.Node {
	node, err := enode.Parse(enode.ValidSchemes, enrStr)
	if err != nil || node == nil {
		panic("could not decode enr: " + enrStr)
	}
	return node
}

func MustDecodeMultiaddress(multiaddrsStr string) *multiaddr.Multiaddr {
	maddr, err := multiaddr.NewMultiaddr(multiaddrsStr)
	if err != nil || maddr == nil {
		panic("could not decode multiaddr: " + multiaddrsStr)
	}
	return &maddr
}

type Mailserver struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Custom bool                 `json:"custom"`
	ENR    *enode.Node          `json:"enr"`
	Addr   *multiaddr.Multiaddr `json:"addr"`

	// Deprecated: only used with WakuV1
	Password       string `json:"password,omitempty"`
	Fleet          string `json:"fleet"`
	FailedRequests uint   `json:"-"`
}

func (m Mailserver) PeerInfo() (peer.AddrInfo, error) {
	var maddrs []multiaddr.Multiaddr

	if m.ENR != nil {
		addrInfo, err := enr.EnodeToPeerInfo(m.ENR)
		if err != nil {
			return peer.AddrInfo{}, err
		}
		addrInfo.Addrs = utils.EncapsulatePeerID(addrInfo.ID, addrInfo.Addrs...)
		maddrs = append(maddrs, addrInfo.Addrs...)
	}

	if m.Addr != nil {
		maddrs = append(maddrs, *m.Addr)
	}

	p, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	if len(p) != 1 {
		return peer.AddrInfo{}, errors.New("invalid mailserver setup")
	}

	return p[0], nil
}

func (m Mailserver) PeerID() (peer.ID, error) {
	p, err := m.PeerInfo()
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

func (m Mailserver) nullablePassword() (val sql.NullString) {
	if m.Password != "" {
		val.String = m.Password
		val.Valid = true
	}
	return
}

type MailserverRequestGap struct {
	ID     string `json:"id"`
	ChatID string `json:"chatId"`
	From   uint64 `json:"from"`
	To     uint64 `json:"to"`
}

type MailserverTopic struct {
	PubsubTopic  string   `json:"pubsubTopic"`
	ContentTopic string   `json:"topic"`
	Discovery    bool     `json:"discovery?"`
	Negotiated   bool     `json:"negotiated?"`
	ChatIDs      []string `json:"chat-ids"`
	LastRequest  int      `json:"last-request"` // default is 1
}

type ChatRequestRange struct {
	ChatID            string `json:"chat-id"`
	LowestRequestFrom int    `json:"lowest-request-from"`
	HighestRequestTo  int    `json:"highest-request-to"`
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

func (d *Database) Add(mailserver Mailserver) error {
	// TODO: we are only storing the multiaddress.
	// In a future PR we must allow storing multiple multiaddresses and ENR
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailservers(
			id,
			name,
			address,
			password,
			fleet
		) VALUES (?, ?, ?, ?, ?)`,
		mailserver.ID,
		mailserver.Name,
		(*mailserver.Addr).String(),
		mailserver.nullablePassword(),
		mailserver.Fleet,
	)
	return err
}

func (d *Database) Mailservers() ([]Mailserver, error) {
	rows, err := d.db.Query(`SELECT id, name, address, password, fleet FROM mailservers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return toMailservers(rows)
}

func toMailservers(rows *sql.Rows) ([]Mailserver, error) {
	var result []Mailserver

	for rows.Next() {
		var (
			m        Mailserver
			addrStr  string
			password sql.NullString
		)
		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&addrStr,
			&password,
			&m.Fleet,
		); err != nil {
			return nil, err
		}
		m.Custom = true
		if password.Valid {
			m.Password = password.String
		}

		// TODO: we are only storing the multiaddress.
		// In a future PR we must allow storing multiple multiaddresses and ENR
		maddr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			return nil, err
		}
		m.Addr = &maddr

		result = append(result, m)
	}

	return result, nil
}

func (d *Database) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM mailservers WHERE id = ?`, id)
	return err
}

func (d *Database) AddGaps(gaps []MailserverRequestGap) error {
	tx, err := d.db.Begin()
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

	for _, gap := range gaps {

		_, err = tx.Exec(`INSERT OR REPLACE INTO mailserver_request_gaps(
				id,
				chat_id,
				gap_from,
				gap_to
			) VALUES (?, ?, ?, ?)`,
			gap.ID,
			gap.ChatID,
			gap.From,
			gap.To,
		)
		if err != nil {
			return err
		}

	}
	return nil
}

func (d *Database) RequestGaps(chatID string) ([]MailserverRequestGap, error) {
	var result []MailserverRequestGap

	rows, err := d.db.Query(`SELECT id, chat_id, gap_from, gap_to FROM mailserver_request_gaps WHERE chat_id = ?`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m MailserverRequestGap
		if err := rows.Scan(
			&m.ID,
			&m.ChatID,
			&m.From,
			&m.To,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return result, nil
}

func (d *Database) DeleteGaps(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	inVector := strings.Repeat("?, ", len(ids)-1) + "?"
	query := fmt.Sprintf(`DELETE FROM mailserver_request_gaps WHERE id IN (%s)`, inVector) // nolint: gosec
	idsArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		idsArgs = append(idsArgs, id)
	}

	_, err := d.db.Exec(query, idsArgs...)
	return err
}

func (d *Database) DeleteGapsByChatID(chatID string) error {
	_, err := d.db.Exec(`DELETE FROM mailserver_request_gaps WHERE chat_id = ?`, chatID)
	return err
}

func (d *Database) AddTopic(topic MailserverTopic) error {

	chatIDs := sqlStringSlice(topic.ChatIDs)
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailserver_topics(
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
	return err
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

func (d *Database) DeleteTopic(pubsubTopic, contentTopic string) error {
	_, err := d.db.Exec(`DELETE FROM mailserver_topics WHERE pubsub_topic = ? AND topic = ?`, pubsubTopic, contentTopic)
	return err
}

// SetTopics deletes all topics excepts the one set, or upsert those if
// missing
func (d *Database) SetTopics(filters []*transport.Filter) (err error) {
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
		contentTopics, ok := contentTopicsPerPubsubTopic[filter.PubsubTopic]
		if !ok {
			contentTopics = make(map[string]struct{})
		}
		contentTopics[filter.ContentTopic.String()] = struct{}{}
		contentTopicsPerPubsubTopic[filter.PubsubTopic] = contentTopics
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
		err = tx.QueryRow(`SELECT topic FROM mailserver_topics WHERE topic = ? AND pubsub_topic = ?`, filter.ContentTopic.String(), filter.PubsubTopic).Scan(&topic)
		if err != nil && err != sql.ErrNoRows {
			return
		} else if err == sql.ErrNoRows {
			// we insert the topic
			_, err = tx.Exec(`INSERT INTO mailserver_topics(topic,pubsub_topic,last_request,discovery,negotiated) VALUES (?,?,?,?,?)`, filter.ContentTopic.String(), filter.PubsubTopic, lastRequest, filter.Discovery, filter.Negotiated)
		}
		if err != nil {
			return
		}
	}

	return
}

func (d *Database) AddChatRequestRange(req ChatRequestRange) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO mailserver_chat_request_ranges(
			chat_id,
			lowest_request_from,
			highest_request_to
		) VALUES (?, ?, ?)`,
		req.ChatID,
		req.LowestRequestFrom,
		req.HighestRequestTo,
	)
	return err
}

func (d *Database) AddChatRequestRanges(reqs []ChatRequestRange) (err error) {
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
	for _, req := range reqs {

		_, err = tx.Exec(`INSERT OR REPLACE INTO mailserver_chat_request_ranges(
			chat_id,
			lowest_request_from,
			highest_request_to
		) VALUES (?, ?, ?)`,
			req.ChatID,
			req.LowestRequestFrom,
			req.HighestRequestTo,
		)
		if err != nil {
			return
		}
	}
	return
}

func (d *Database) ChatRequestRanges() ([]ChatRequestRange, error) {
	var result []ChatRequestRange

	rows, err := d.db.Query(`SELECT chat_id, lowest_request_from, highest_request_to FROM mailserver_chat_request_ranges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var req ChatRequestRange
		if err := rows.Scan(
			&req.ChatID,
			&req.LowestRequestFrom,
			&req.HighestRequestTo,
		); err != nil {
			return nil, err
		}
		result = append(result, req)
	}

	return result, nil
}

func (d *Database) DeleteChatRequestRange(chatID string) error {
	_, err := d.db.Exec(`DELETE FROM mailserver_chat_request_ranges WHERE chat_id = ?`, chatID)
	return err
}
