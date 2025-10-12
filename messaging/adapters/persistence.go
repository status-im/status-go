package adapters

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/internal"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	wakupersistence "github.com/status-im/status-go/messaging/waku/persistence"
)

type KeysPersistence struct {
	P types.Persistence
}

var _ transport.KeysPersistence = (*KeysPersistence)(nil)

func (kp *KeysPersistence) All() (map[string][]byte, error) {
	return kp.P.WakuStorage().Keys()
}

func (kp *KeysPersistence) Add(chatID string, key []byte) error {
	return kp.P.WakuStorage().AddKey(chatID, key)
}

type ProcessedMessageIDsCache struct {
	P types.Persistence
}

var _ transport.ProcessedMessageIDsCachePersistence = (*ProcessedMessageIDsCache)(nil)

func (pm *ProcessedMessageIDsCache) Clear() error {
	return pm.P.MessageCacheClear()
}
func (pm *ProcessedMessageIDsCache) Hits(ids []string) (map[string]bool, error) {
	return pm.P.MessageCacheHits(ids)
}
func (pm *ProcessedMessageIDsCache) Add(ids []string, timestamp uint64) error {
	return pm.P.MessageCacheAdd(ids, timestamp)
}
func (pm *ProcessedMessageIDsCache) Clean(timestamp uint64) error {
	return pm.P.MessageCacheClearOlderThan(timestamp)
}

type WakuProtectedTopics struct {
	P types.Persistence
}

var _ wakupersistence.ProtectedTopics = (*WakuProtectedTopics)(nil)

func (wpt *WakuProtectedTopics) Insert(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error {
	return wpt.P.WakuStorage().InsertProtectedTopic(pubsubTopic, privKey, publicKey)
}

func (wpt *WakuProtectedTopics) Delete(pubsubTopic string) error {
	return wpt.P.WakuStorage().DeleteProtectedTopic(pubsubTopic)
}

func (wpt *WakuProtectedTopics) FetchPrivateKey(topic string) (*ecdsa.PrivateKey, error) {
	return wpt.P.WakuStorage().FetchPrivateKeyForProtectedTopic(topic)
}

func (wpt *WakuProtectedTopics) ProtectedTopics() ([]wakupersistence.ProtectedTopic, error) {
	pt, err := wpt.P.WakuStorage().ProtectedTopics()
	if err != nil {
		return nil, err
	}
	result := make([]wakupersistence.ProtectedTopic, len(pt))
	for i, p := range pt {
		result[i] = wakupersistence.ProtectedTopic{
			PubKey: p.PubKey,
			Topic:  p.Topic,
		}
	}
	return result, nil
}

func EncryptionPersistence(p types.Persistence) encryption.Persistence {
	encryptionStorage := p.EncryptionStorage()
	if encryptionStorage == nil {
		return nil
	}
	internal, ok := encryptionStorage.(*internal.SQLiteEncryptionPersistence)
	if !ok {
		panic("custom EncryptionPersistence implementations are not supported yet")
	}
	return internal.SQLitePersistence
}
