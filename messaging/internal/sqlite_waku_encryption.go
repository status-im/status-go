package internal

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/types"
	wakuv2 "github.com/status-im/status-go/messaging/waku"
)

type SQLiteWakuPersistence struct {
	*wakuv2.SQLiteProtectedTopicsPersistence
}

var _ types.WakuPersistence = (*SQLiteWakuPersistence)(nil)

func (s *SQLiteWakuPersistence) InsertProtectedTopic(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error {
	return nil
}

func (s *SQLiteWakuPersistence) DeleteProtectedTopic(pubsubTopic string) error {
	return nil
}

func (s *SQLiteWakuPersistence) FetchPrivateKeyForProtectedTopic(topic string) (*ecdsa.PrivateKey, error) {
	return nil, nil
}

func (s *SQLiteWakuPersistence) ProtectedTopics() ([]types.ProtectedTopicRecord, error) {
	return nil, nil
}
