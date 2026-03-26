package datasync

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"
	"github.com/status-im/mvds/transport"
	"go.uber.org/zap"
)

const backoffInterval = 60

var errNotInitialized = errors.New("datasync transport not initialized")
var DatasyncTicker = 300 * time.Millisecond
var datasyncTickerMutex sync.RWMutex

func SetPaused(paused bool) {
	datasyncTickerMutex.Lock()
	defer datasyncTickerMutex.Unlock()
	if paused {
		DatasyncTicker = 2 * time.Second
	} else {
		DatasyncTicker = 300 * time.Millisecond
	}
}

func currentOffsetToSecond() uint64 {
	datasyncTickerMutex.RLock()
	defer datasyncTickerMutex.RUnlock()
	return uint64(math.Ceil(float64(time.Second) / float64(DatasyncTicker)))
}

type NodeTransport struct {
	packets  chan transport.Packet
	logger   *zap.Logger
	dispatch func(state.PeerID, *protobuf.Payload) error
}

var _ transport.Transport = (*NodeTransport)(nil)

func NewNodeTransport() *NodeTransport {
	return &NodeTransport{
		packets: make(chan transport.Packet),
	}
}

func (t *NodeTransport) Init(dispatch func(state.PeerID, *protobuf.Payload) error, logger *zap.Logger) {
	t.dispatch = dispatch
	t.logger = logger
}

func (t *NodeTransport) AddPacket(p transport.Packet) {
	t.packets <- p
}

func (t *NodeTransport) Watch(ctx context.Context) (*transport.Packet, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case p := <-t.packets:
		return &p, true
	}
}

func (t *NodeTransport) Send(_ state.PeerID, peer state.PeerID, payload *protobuf.Payload) error {
	if t.dispatch == nil {
		return errNotInitialized
	}

	// We don't return an error otherwise datasync will keep
	// re-trying sending at each epoch
	err := t.dispatch(peer, payload)
	if err != nil {
		t.logger.Error("failed to send message", zap.Error(err))
		return nil
	}

	return nil
}

// CalculateSendTime calculates the next epoch
// at which a message should be sent.
// We randomize it a bit so that not all messages are sent on the same epoch
func CalculateSendTime(count uint64, time int64) int64 {
	return time + int64(uint64(math.Exp2(float64(count-1)))*backoffInterval*currentOffsetToSecond()) + int64(rand.Intn(30)) // nolint: gosec
}
