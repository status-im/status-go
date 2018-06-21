package debug

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

var (
	postTimeout = 60 * time.Second
	errTimeout  = errors.New("timed out waiting for the envelope to be sent")
	errExpired  = errors.New("envelope expired before being sent")
)

// PublicAPI represents a set of APIs from the `web3.debug` namespace.
type PublicAPI struct {
	s *Service
}

// NewAPI creates an instance of the debug API.
func NewAPI(s *Service) *PublicAPI {
	return &PublicAPI{s: s}
}

// PostSync sends an envelope through shhext_post and waits until the related
// envelope event is sent.
func (api *PublicAPI) PostSync(ctx context.Context, req whisper.NewMessage) (hash hexutil.Bytes, err error) {
	hash, err = api.s.p.Post(ctx, req)
	if err != nil {
		return
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, postTimeout)
	defer cancel()
	err = api.waitForHash(ctxTimeout, hash)
	return
}

// waitForHash waits for a specific hash to be sent
func (api *PublicAPI) waitForHash(ctx context.Context, hash hexutil.Bytes) error {
	h := common.BytesToHash(hash)
	events := make(chan whisper.EnvelopeEvent, 100)
	sub := api.s.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-events:
			if ev.Hash == h {
				if ev.Event == whisper.EventEnvelopeSent {
					return nil
				}
				if ev.Event == whisper.EventEnvelopeExpired {
					return errExpired
				}
			}
		case <-ctx.Done():
			return errTimeout
		}
	}
}
