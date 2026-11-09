package sender

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// retrievalHintWaitTimeout bounds how long a send waits for the wire hashes of
// the previous message in its SDS channel.
const retrievalHintWaitTimeout = 10 * time.Second

// sdsChannel serializes the sends of one SDS channel, so that each message is
// wrapped once the wire hashes of the previous one are known: they are its
// retrieval hint in the next message's causal history.
type sdsChannel struct {
	mu sync.Mutex
	// last is the SDS message ID of the last message sent on the channel.
	last []byte
}

type sdsChannels struct {
	mu       sync.Mutex
	channels map[string]*sdsChannel
}

func (c *sdsChannels) get(channelID string) *sdsChannel {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channels == nil {
		c.channels = make(map[string]*sdsChannel)
	}
	channel, ok := c.channels[channelID]
	if !ok {
		channel = &sdsChannel{}
		c.channels[channelID] = channel
	}
	return channel
}

// waitForRetrievalHint waits until the wire hashes of sdsMessageID are known,
// at most retrievalHintWaitTimeout. A message wrapped earlier carries a
// partial hint, and the following ones carry the full one.
func (s *Sender) waitForRetrievalHint(ctx context.Context, logger *zap.Logger, sdsMessageID []byte) {
	if len(sdsMessageID) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, retrievalHintWaitTimeout)
	defer cancel()
	if err := s.stack.Transport.WaitForEnvelopeHashes(ctx, sdsMessageID); err != nil {
		logger.Debug("previous SDS message has no wire hash yet",
			zap.String("sdsMessageID", cryptotypes.EncodeHex(sdsMessageID)),
			zap.Error(err))
	}
}
