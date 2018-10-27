package shhext

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/services"
	whisper "github.com/status-im/whisper/whisperv6"
)

var (
	postSyncTimeout        = 60 * time.Second
	errEnvelopeExpired     = errors.New("envelope expired before being sent")
	errNoShhextAttachedAPI = errors.New("No shhext attached")
)

// DebugAPI represents a set of APIs from the `web3.debug` namespace.
type DebugAPI struct {
	s *Service
}

// NewDebugAPI creates an instance of the debug API.
func NewDebugAPI(s *Service) *DebugAPI {
	return &DebugAPI{s: s}
}

// PostSync sends an envelope through shhext_post and waits until it's sent.
func (api *DebugAPI) PostSync(ctx context.Context, req whisper.NewMessage) (hash hexutil.Bytes, err error) {
	shhAPI := services.APIByNamespace(api.s.APIs(), "shhext")
	if shhAPI == nil {
		err = errNoShhextAttachedAPI
		return
	}
	s, ok := shhAPI.(*PublicAPI)
	if !ok {
		err = errNoShhextAttachedAPI
		return
	}
	hash, err = s.Post(ctx, req)
	if err != nil {
		return
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, postSyncTimeout)
	defer cancel()
	err = api.waitForHash(ctxTimeout, hash)
	return
}

// waitForHash waits for a specific hash to be sent
func (api *DebugAPI) waitForHash(ctx context.Context, hash hexutil.Bytes) error {
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
					return errEnvelopeExpired
				}
			}
		case <-ctx.Done():
			return fmt.Errorf("wait for hash canceled: %v", ctx.Err())
		}
	}
}
