package dedup

import (
	"github.com/ethereum/go-ethereum/log"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
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

// NewDeduplicator creates a new deduplicator
func NewDeduplicator(keyPairProvider keyPairProvider) *Deduplicator {
	return &Deduplicator{
		log:             log.New("package", "status-go/services/sshext.deduplicator"),
		keyPairProvider: keyPairProvider,
	}
}

// Start enabled deduplication.
func (d *Deduplicator) Start(db *leveldb.DB) error {
	d.cache = newCache(db)
	return nil
}

// Deduplicate receives a list of whisper messages and
// returns the list of the messages that weren't filtered previously for the
// specified filter.
func (d *Deduplicator) Deduplicate(messages []*whisper.Message) []*whisper.Message {
	if d.cache == nil {
		d.log.Info("Deduplication wasn't started. Returning all the messages.")
		return messages
	}
	result := make([]*whisper.Message, 0)

	for _, message := range messages {
		if has, err := d.cache.Has(d.keyPairProvider.SelectedKeyPairID(), message); !has {
			if err != nil {
				d.log.Error("error while deduplicating messages: search cache failed", "err", err)
			}
			result = append(result, message)
		}
	}

	// Put all the messages there, for simplicity.
	// That way, we will always have repeating messages in the current day.
	// Performance implications seem negligible on 30000 messages/day
	err := d.cache.Put(d.keyPairProvider.SelectedKeyPairID(), messages)
	if err != nil {
		d.log.Error("error while deduplicating messages: cache update failed", "err", err)
	}

	return result
}
