// +build !nimbus

package shhext

import (
	"context"
	"fmt"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
)

type Service struct {
	*ext.Service
	w types.Whisper
}

func New(config params.ShhextConfig, n types.Node, ctx interface{}, handler ext.EnvelopeEventsHandler, ldb *leveldb.DB) *Service {
	w, err := n.GetWhisper(ctx)
	if err != nil {
		panic(err)
	}
	delay := ext.DefaultRequestsDelay
	if config.RequestsDelay != 0 {
		delay = config.RequestsDelay
	}
	requestsRegistry := ext.NewRequestsRegistry(delay)
	mailMonitor := ext.NewMailRequestMonitor(w, handler, requestsRegistry)
	return &Service{
		Service: ext.New(config, n, ldb, mailMonitor, requestsRegistry, w),
		w:       w,
	}
}

func (s *Service) PublicWhisperAPI() types.PublicWhisperAPI {
	return s.w.PublicWhisperAPI()
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "shhext",
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    false,
		},
	}
	return apis
}

func (s *Service) SyncMessages(ctx context.Context, mailServerID []byte, r types.SyncMailRequest) (resp types.SyncEventResponse, err error) {
	err = s.w.SyncMessages(mailServerID, r)
	if err != nil {
		return
	}

	// Wait for the response which is received asynchronously as a p2p packet.
	// This packet handler will send an event which contains the response payload.
	events := make(chan types.EnvelopeEvent, 1024)
	sub := s.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	// Add explicit timeout context, otherwise the request
	// can hang indefinitely if not specified by the sender.
	// Sender is usually through netcat or some bash tool
	// so it's not really possible to specify the timeout.
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	for {
		select {
		case event := <-events:
			if event.Event != types.EventMailServerSyncFinished {
				continue
			}

			log.Info("received EventMailServerSyncFinished event", "data", event.Data)

			var ok bool

			resp, ok = event.Data.(types.SyncEventResponse)
			if !ok {
				err = fmt.Errorf("did not understand the response event data")
				return
			}
			return
		case <-timeoutCtx.Done():
			err = timeoutCtx.Err()
			return
		}
	}
}
