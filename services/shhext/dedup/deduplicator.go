package dedup

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
)

type keyPairProvider interface {
	SelectedKeyPairID() string
}

// Deduplicator filters out already received messages for a current filter.
// It keeps a limited cache of the messages.
type Deduplicator struct {
	keyPairProvider keyPairProvider
	cache           *cache
	log             log.Logger
}

type Metadata struct {
	DedupID      []byte        `json:"dedupId"`
	EncryptionID hexutil.Bytes `json:"encryptionId"`
	MessageID    hexutil.Bytes `json:"messageId"`
}

type DeduplicateMessage struct {
	Message  *whisper.Message `json:"message"`
	Metadata Metadata         `json:"metadata"`
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator(keyPairProvider keyPairProvider, db *leveldb.DB) *Deduplicator {
	return &Deduplicator{
		log:             log.New("package", "status-go/services/sshext.deduplicator"),
		keyPairProvider: keyPairProvider,
		cache:           newCache(db),
	}
}

// Deduplicate receives a list of whisper messages and
// returns the list of the messages that weren't filtered previously for the
// specified filter.
func (d *Deduplicator) Deduplicate(messages []*DeduplicateMessage) []*DeduplicateMessage {
	result := make([]*DeduplicateMessage, 0)
	selectedKeyPairID := d.keyPairProvider.SelectedKeyPairID()

	for _, message := range messages {
		if has, err := d.cache.Has(selectedKeyPairID, message.Message); !has {
			if err != nil {
				d.log.Error("error while deduplicating messages: search cache failed", "err", err)
			}
			message.Metadata.DedupID = d.cache.KeyToday(selectedKeyPairID, message.Message)
			result = append(result, message)
		}
	}

	return result
}

// AddMessages adds a message to the deduplicator DB, so it will be filtered
// out.
func (d *Deduplicator) AddMessagesByID(messageIDs [][]byte) error {
	return d.cache.PutIDs(messageIDs)
}

// AddMessageByID adds a message to the deduplicator DB, so it will be filtered
// out.
func (d *Deduplicator) AddMessageByID(messageIDs [][]byte) error {
	return d.cache.PutIDs(messageIDs)
}
