package peer_exchange

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

// PeerExchangeID_v20alpha1 is the current Waku Peer Exchange protocol identifier
const PeerExchangeID_v20alpha1 = libp2pProtocol.ID("/vac/waku/peer-exchange/2.0.0-alpha1")
const MaxCacheSize = 1000
const CacheCleanWindow = 200

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidId        = errors.New("invalid request id")
)

type peerRecord struct {
	node *enode.Node
	idx  int
}

type WakuPeerExchange struct {
	h    host.Host
	disc *discv5.DiscoveryV5

	log *zap.Logger

	cancel context.CancelFunc

	wg            sync.WaitGroup
	peerConnector PeerConnector
	peerCh        chan peer.AddrInfo
	enrCache      map[enode.ID]peerRecord // todo: next step: ring buffer; future: implement cache satisfying https://rfc.vac.dev/spec/34/
	enrCacheMutex sync.RWMutex
	rng           *rand.Rand
}

type PeerConnector interface {
	PeerChannel() chan<- peer.AddrInfo
}

// NewWakuPeerExchange returns a new instance of WakuPeerExchange struct
func NewWakuPeerExchange(h host.Host, disc *discv5.DiscoveryV5, peerConnector PeerConnector, log *zap.Logger) (*WakuPeerExchange, error) {
	wakuPX := new(WakuPeerExchange)
	wakuPX.h = h
	wakuPX.disc = disc
	wakuPX.log = log.Named("wakupx")
	wakuPX.enrCache = make(map[enode.ID]peerRecord)
	wakuPX.rng = rand.New(rand.NewSource(rand.Int63()))
	wakuPX.peerConnector = peerConnector
	return wakuPX, nil
}

// Start inits the peer exchange protocol
func (wakuPX *WakuPeerExchange) Start(ctx context.Context) error {
	if wakuPX.cancel != nil {
		return errors.New("peer exchange already started")
	}

	wakuPX.wg.Wait() // Waiting for any go routines to stop

	ctx, cancel := context.WithCancel(ctx)
	wakuPX.cancel = cancel
	wakuPX.peerCh = make(chan peer.AddrInfo)

	wakuPX.h.SetStreamHandlerMatch(PeerExchangeID_v20alpha1, protocol.PrefixTextMatch(string(PeerExchangeID_v20alpha1)), wakuPX.onRequest(ctx))
	wakuPX.log.Info("Peer exchange protocol started")

	wakuPX.wg.Add(1)
	go wakuPX.runPeerExchangeDiscv5Loop(ctx)
	return nil
}

func (wakuPX *WakuPeerExchange) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wakuPX.log.With(logging.HostID("peer", s.Conn().RemotePeer()))
		requestRPC := &pb.PeerExchangeRPC{}
		reader := pbio.NewDelimitedReader(s, math.MaxInt32)
		err := reader.ReadMsg(requestRPC)
		if err != nil {
			logger.Error("reading request", zap.Error(err))
			metrics.RecordPeerExchangeError(ctx, "decodeRpcFailure")
			return
		}

		if requestRPC.Query != nil {
			logger.Info("request received")

			records, err := wakuPX.getENRsFromCache(requestRPC.Query.NumPeers)
			if err != nil {
				logger.Error("obtaining enrs from cache", zap.Error(err))
				metrics.RecordPeerExchangeError(ctx, "pxFailure")
				return
			}

			responseRPC := &pb.PeerExchangeRPC{}
			responseRPC.Response = new(pb.PeerExchangeResponse)
			responseRPC.Response.PeerInfos = records

			writer := pbio.NewDelimitedWriter(s)
			err = writer.WriteMsg(responseRPC)
			if err != nil {
				logger.Error("writing response", zap.Error(err))
				metrics.RecordPeerExchangeError(ctx, "pxFailure")
				return
			}
		}
	}
}

// Stop unmounts the peer exchange protocol
func (wakuPX *WakuPeerExchange) Stop() {
	if wakuPX.cancel == nil {
		return
	}
	wakuPX.h.RemoveStreamHandler(PeerExchangeID_v20alpha1)
	wakuPX.cancel()
	close(wakuPX.peerCh)
	wakuPX.wg.Wait()
}

func (wakuPX *WakuPeerExchange) getENRsFromCache(numPeers uint64) ([]*pb.PeerInfo, error) {
	wakuPX.enrCacheMutex.Lock()
	defer wakuPX.enrCacheMutex.Unlock()

	if len(wakuPX.enrCache) == 0 {
		return nil, nil
	}

	numItems := int(numPeers)
	if len(wakuPX.enrCache) < int(numPeers) {
		numItems = len(wakuPX.enrCache)
	}

	perm := wakuPX.rng.Perm(len(wakuPX.enrCache))[0:numItems]
	permSet := make(map[int]int)
	for i, v := range perm {
		permSet[v] = i
	}

	var result []*pb.PeerInfo
	iter := 0
	for k := range wakuPX.enrCache {
		if _, ok := permSet[iter]; ok {
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			enode := wakuPX.enrCache[k]

			err := enode.node.Record().EncodeRLP(writer)
			if err != nil {
				return nil, err
			}

			writer.Flush()

			result = append(result, &pb.PeerInfo{
				ENR: b.Bytes(),
			})
		}
		iter++
	}

	return result, nil
}

func (wakuPX *WakuPeerExchange) cleanCache() {
	if len(wakuPX.enrCache) < MaxCacheSize {
		return
	}

	r := make(map[enode.ID]peerRecord)
	for k, v := range wakuPX.enrCache {
		if v.idx > CacheCleanWindow {
			v.idx -= CacheCleanWindow
			r[k] = v
		}
	}

	wakuPX.enrCache = r
}

func (wakuPX *WakuPeerExchange) iterate(ctx context.Context) error {
	iterator, err := wakuPX.disc.Iterator()
	if err != nil {
		return fmt.Errorf("obtaining iterator: %w", err)
	}

	closeCh := make(chan struct{}, 1)
	defer close(closeCh)

	// Closing iterator when context is cancelled or function is returning
	wakuPX.wg.Add(1)
	go func() {
		defer wakuPX.wg.Done()
		select {
		case <-ctx.Done():
			iterator.Close()
		case <-closeCh:
			iterator.Close()
		}
	}()

	for {
		if ctx.Err() != nil {
			break
		}

		exists := iterator.Next()
		if !exists {
			break
		}

		_, addresses, err := utils.Multiaddress(iterator.Node())
		if err != nil {
			wakuPX.log.Error("extracting multiaddrs from enr", zap.Error(err))
			continue
		}

		if len(addresses) == 0 {
			continue
		}

		wakuPX.log.Debug("Discovered px peers via discv5")

		wakuPX.enrCacheMutex.Lock()
		wakuPX.enrCache[iterator.Node().ID()] = peerRecord{
			idx:  len(wakuPX.enrCache),
			node: iterator.Node(),
		}
		wakuPX.enrCacheMutex.Unlock()
	}

	return nil
}

func (wakuPX *WakuPeerExchange) runPeerExchangeDiscv5Loop(ctx context.Context) {
	defer wakuPX.wg.Done()

	// Runs a discv5 loop adding new peers to the px peer cache
	if wakuPX.disc == nil {
		wakuPX.log.Warn("trying to run discovery v5 (for PX) while it's disabled")
		return
	}

	ch := make(chan struct{}, 1)
	ch <- struct{}{} // Initial execution

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

restartLoop:
	for {
		select {
		case <-ch:
			err := wakuPX.iterate(ctx)
			if err != nil {
				wakuPX.log.Debug("iterating peer exchange", zap.Error(err))
				time.Sleep(2 * time.Second)
			}
			ch <- struct{}{}
		case <-ticker.C:
			wakuPX.cleanCache()
		case <-ctx.Done():
			close(ch)
			break restartLoop
		}
	}
}
