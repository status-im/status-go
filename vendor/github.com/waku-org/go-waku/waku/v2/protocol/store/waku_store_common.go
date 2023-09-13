package store

import (
	"context"
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"go.uber.org/zap"
)

// StoreID_v20beta4 is the current Waku Store protocol identifier
const StoreID_v20beta4 = libp2pProtocol.ID("/vac/waku/store/2.0.0-beta4")

// MaxPageSize is the maximum number of waku messages to return per page
const MaxPageSize = 100

// MaxContentFilters is the maximum number of allowed content filters in a query
const MaxContentFilters = 10

var (
	// ErrMaxContentFilters is returned when the number of content topics in the query
	// exceeds the limit
	ErrMaxContentFilters = errors.New("exceeds the maximum number of content filters allowed")

	// ErrNoPeersAvailable is returned when there are no store peers in the peer store
	// that could be used to retrieve message history
	ErrNoPeersAvailable = errors.New("no suitable remote peers")

	// ErrInvalidID is returned when no RequestID is given
	ErrInvalidID = errors.New("invalid request id")

	// ErrFailedToResumeHistory is returned when the node attempted to retrieve historic
	// messages to fill its own message history but for some reason it failed
	ErrFailedToResumeHistory = errors.New("failed to resume the history")

	// ErrFailedQuery is emitted when the query fails to return results
	ErrFailedQuery = errors.New("failed to resolve the query")
)

type WakuSwap interface {
	// TODO: add functions
}

type WakuStore struct {
	ctx        context.Context
	cancel     context.CancelFunc
	timesource timesource.Timesource
	metrics    Metrics
	MsgC       relay.Subscription
	wg         *sync.WaitGroup

	log *zap.Logger

	started bool

	msgProvider MessageProvider
	h           host.Host
	pm          *peermanager.PeerManager
}

// NewWakuStore creates a WakuStore using an specific MessageProvider for storing the messages
// Takes an optional peermanager if WakuStore is being created along with WakuNode.
// If using libp2p host, then pass peermanager as nil
func NewWakuStore(p MessageProvider, pm *peermanager.PeerManager, timesource timesource.Timesource, reg prometheus.Registerer, log *zap.Logger) *WakuStore {
	wakuStore := new(WakuStore)
	wakuStore.msgProvider = p
	wakuStore.wg = &sync.WaitGroup{}
	wakuStore.log = log.Named("store")
	wakuStore.timesource = timesource
	wakuStore.pm = pm
	wakuStore.metrics = newMetrics(reg)

	return wakuStore
}
