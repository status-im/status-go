package node

import (
	"context"

	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type Service interface {
	Start(ctx context.Context) error
	Stop()
}

type ReceptorService interface {
	Service
	MessageChannel() chan *protocol.Envelope
}
