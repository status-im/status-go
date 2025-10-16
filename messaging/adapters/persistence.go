package adapters

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/internal"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	waku "github.com/status-im/status-go/messaging/waku"
)

func NewKeysPersistence(p types.Persistence) transport.KeysPersistence {
	transportStorage := p.TransportStorage()
	if transportStorage == nil {
		return nil
	}

	internal, ok := transportStorage.(*internal.SQLiteTransportPersistence)
	if ok {
		return internal.SQLiteKeysPersistence
	}

	return &keysPersistence{P: p}
}

type keysPersistence struct {
	P types.Persistence
}

var _ transport.KeysPersistence = (*keysPersistence)(nil)

func (kp *keysPersistence) All() (map[string][]byte, error) {
	return kp.P.TransportStorage().Keys()
}

func (kp *keysPersistence) Add(chatID string, key []byte) error {
	return kp.P.TransportStorage().AddKey(chatID, key)
}

func NewProcessedMessageIDsCache(p types.Persistence) transport.ProcessedMessageIDsCachePersistence {
	transportStorage := p.TransportStorage()
	if transportStorage == nil {
		return nil
	}

	internal, ok := transportStorage.(*internal.SQLiteTransportPersistence)
	if ok {
		return internal.SQLiteProcessedMessageIDsCachePersistence
	}

	return &processedMessageIDsCache{P: p}
}

type processedMessageIDsCache struct {
	P types.Persistence
}

var _ transport.ProcessedMessageIDsCachePersistence = (*processedMessageIDsCache)(nil)

func (pm *processedMessageIDsCache) Clear() error {
	return pm.P.TransportStorage().MessageCacheClear()
}
func (pm *processedMessageIDsCache) Hits(ids []string) (map[string]bool, error) {
	return pm.P.TransportStorage().MessageCacheHits(ids)
}
func (pm *processedMessageIDsCache) Add(ids []string, timestamp uint64) error {
	return pm.P.TransportStorage().MessageCacheAdd(ids, timestamp)
}
func (pm *processedMessageIDsCache) Clean(timestamp uint64) error {
	return pm.P.TransportStorage().MessageCacheClearOlderThan(timestamp)
}

func NewWakuProtectedTopics(p types.Persistence) waku.ProtectedTopicsPersistence {
	wakuStorage := p.WakuStorage()
	if wakuStorage == nil {
		return nil
	}

	internal, ok := wakuStorage.(*internal.SQLiteWakuPersistence)
	if ok {
		return internal.SQLiteProtectedTopicsPersistence
	}

	return &wakuProtectedTopics{P: p}
}

type wakuProtectedTopics struct {
	P types.Persistence
}

var _ waku.ProtectedTopicsPersistence = (*wakuProtectedTopics)(nil)

func (wpt *wakuProtectedTopics) Insert(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error {
	return wpt.P.WakuStorage().InsertProtectedTopic(pubsubTopic, privKey, publicKey)
}

func (wpt *wakuProtectedTopics) Delete(pubsubTopic string) error {
	return wpt.P.WakuStorage().DeleteProtectedTopic(pubsubTopic)
}

func (wpt *wakuProtectedTopics) FetchPrivateKey(topic string) (*ecdsa.PrivateKey, error) {
	return wpt.P.WakuStorage().FetchPrivateKeyForProtectedTopic(topic)
}

func (wpt *wakuProtectedTopics) All() ([]waku.ProtectedTopic, error) {
	pt, err := wpt.P.WakuStorage().ProtectedTopics()
	if err != nil {
		return nil, err
	}
	result := make([]waku.ProtectedTopic, len(pt))
	for i, p := range pt {
		result[i] = waku.ProtectedTopic{
			PubKey: p.PubKey,
			Topic:  p.Topic,
		}
	}
	return result, nil
}

func NewSegmentationPersistence(p types.Persistence) segmentation.Persistence {
	segmentationStorage := p.SegmentationStorage()
	if segmentationStorage == nil {
		return nil
	}
	internal, ok := segmentationStorage.(*internal.SQLiteSegmentationPersistence)
	if !ok {
		panic("custom SegmentationPersistence implementations are not supported yet")
	}
	return internal.SQLitePersistence
}

func NewEncryptionPersistence(p types.Persistence) encryption.Persistence {
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
