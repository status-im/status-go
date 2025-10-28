package wakuv2

import (
	"crypto/ecdsa"
	"database/sql"
	"errors"

	"github.com/status-im/status-go/crypto"
)

type SQLiteProtectedTopicsPersistence struct {
	db *sql.DB
}

var _ ProtectedTopicsPersistence = (*SQLiteProtectedTopicsPersistence)(nil)

func NewSQLiteProtectedTopicsPersistence(db *sql.DB) *SQLiteProtectedTopicsPersistence {
	return &SQLiteProtectedTopicsPersistence{db: db}
}

func (s *SQLiteProtectedTopicsPersistence) Insert(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error {
	var privKeyBytes []byte
	if privKey != nil {
		privKeyBytes = crypto.FromECDSA(privKey)
	}
	pubKeyBytes := crypto.FromECDSAPub(publicKey)

	_, err := s.db.Exec("INSERT OR REPLACE INTO pubsubtopic_signing_key (topic, priv_key, pub_key) VALUES (?, ?, ?)",
		pubsubTopic, privKeyBytes, pubKeyBytes)
	return err
}

func (s *SQLiteProtectedTopicsPersistence) Delete(pubsubTopic string) error {
	_, err := s.db.Exec("DELETE FROM pubsubtopic_signing_key WHERE topic = ?", pubsubTopic)
	return err
}

func (s *SQLiteProtectedTopicsPersistence) FetchPrivateKey(topic string) (*ecdsa.PrivateKey, error) {
	var privKeyBytes []byte
	err := s.db.QueryRow("SELECT priv_key FROM pubsubtopic_signing_key WHERE topic = ?", topic).Scan(&privKeyBytes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return crypto.ToECDSA(privKeyBytes)
}

func (s *SQLiteProtectedTopicsPersistence) All() ([]ProtectedTopic, error) {
	rows, err := s.db.Query("SELECT pub_key, topic FROM pubsubtopic_signing_key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProtectedTopic
	for rows.Next() {
		var pubKeyBytes []byte
		var topic string
		err := rows.Scan(&pubKeyBytes, &topic)
		if err != nil {
			return nil, err
		}

		pubk, err := crypto.UnmarshalPubkey(pubKeyBytes)
		if err != nil {
			return nil, err
		}

		result = append(result, ProtectedTopic{
			PubKey: pubk,
			Topic:  topic,
		})
	}

	return result, nil
}
