package peer_exchange

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

// PeerExchangeID_v20alpha1 is the current Waku Peer Exchange protocol identifier
const PeerExchangeID_v20alpha1 = libp2pProtocol.ID("/vac/waku/peer-exchange/2.0.0-alpha1")
const MaxCacheSize = 1000
const CacheCleanWindow = 200
const dialTimeout = 7 * time.Second

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidId        = errors.New("invalid request id")
)

type peerRecord struct {
	node enode.Node
	idx  int
}

type WakuPeerExchange struct {
	h    host.Host
	disc *discv5.DiscoveryV5

	log *zap.Logger

	cancel  context.CancelFunc
	started bool
	wg      sync.WaitGroup

	enrCache      map[enode.ID]peerRecord // todo: next step: ring buffer; future: implement cache satisfying https://rfc.vac.dev/spec/34/
	enrCacheMutex sync.RWMutex
	rng           *rand.Rand
}

// NewWakuPeerExchange returns a new instance of WakuPeerExchange struct
func NewWakuPeerExchange(h host.Host, disc *discv5.DiscoveryV5, log *zap.Logger) *WakuPeerExchange {
	wakuPX := new(WakuPeerExchange)
	wakuPX.h = h
	wakuPX.disc = disc
	wakuPX.log = log.Named("wakupx")
	wakuPX.enrCache = make(map[enode.ID]peerRecord)
	wakuPX.rng = rand.New(rand.NewSource(rand.Int63()))
	return wakuPX
}

// Start inits the peer exchange protocol
func (wakuPX *WakuPeerExchange) Start(ctx context.Context) error {
	if wakuPX.started {
		return errors.New("peer exchange already started")
	}

	wakuPX.wg.Wait() // Waiting for any go routines to stop

	ctx, cancel := context.WithCancel(ctx)
	wakuPX.cancel = cancel
	wakuPX.started = true

	wakuPX.h.SetStreamHandlerMatch(PeerExchangeID_v20alpha1, protocol.PrefixTextMatch(string(PeerExchangeID_v20alpha1)), wakuPX.onRequest(ctx))
	wakuPX.log.Info("Peer exchange protocol started")

	wakuPX.wg.Add(1)
	go wakuPX.runPeerExchangeDiscv5Loop(ctx)

	return nil
}

func (wakuPX *WakuPeerExchange) handleResponse(ctx context.Context, response *pb.PeerExchangeResponse) error {
	var peers []peer.AddrInfo
	for _, p := range response.PeerInfos {
		enrRecord := &enr.Record{}
		buf := bytes.NewBuffer(p.ENR)

		err := enrRecord.DecodeRLP(rlp.NewStream(buf, uint64(len(p.ENR))))
		if err != nil {
			wakuPX.log.Error("converting bytes to enr", zap.Error(err))
			return err
		}

		enodeRecord, err := enode.New(enode.ValidSchemes, enrRecord)
		if err != nil {
			wakuPX.log.Error("creating enode record", zap.Error(err))

			return err
		}

		peerInfo, err := utils.EnodeToPeerInfo(enodeRecord)
		if err != nil {
			return err
		}

		if wakuPX.h.Network().Connectedness(peerInfo.ID) != network.Connected {
			peers = append(peers, *peerInfo)
		}
	}

	if len(peers) != 0 {
		log.Info("connecting to newly discovered peers", zap.Int("count", len(peers)))
		for _, p := range peers {
			func(p peer.AddrInfo) {
				ctx, cancel := context.WithTimeout(ctx, dialTimeout)
				defer cancel()
				err := wakuPX.h.Connect(ctx, p)
				if err != nil {
					log.Info("connecting to peer", zap.String("peer", p.ID.Pretty()), zap.Error(err))
				}
			}(p)
		}
	}

	return nil
}

func (wakuPX *WakuPeerExchange) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wakuPX.log.With(logging.HostID("peer", s.Conn().RemotePeer()))
		requestRPC := &pb.PeerExchangeRPC{}
		reader := protoio.NewDelimitedReader(s, math.MaxInt32)
		err := reader.ReadMsg(requestRPC)
		if err != nil {
			logger.Error("reading request", zap.Error(err))
			metrics.RecordPeerExchangeError(ctx, "decodeRpcFailure")
			return
		}

		if requestRPC.Query != nil {
			logger.Info("request received")
			err := wakuPX.respond(ctx, requestRPC.Query.NumPeers, s.Conn().RemotePeer())
			if err != nil {
				logger.Error("responding", zap.Error(err))
				metrics.RecordPeerExchangeError(ctx, "pxFailure")
				return
			}
		}

		if requestRPC.Response != nil {
			logger.Info("response received")
			err := wakuPX.handleResponse(ctx, requestRPC.Response)
			if err != nil {
				logger.Error("handling response", zap.Error(err))
				metrics.RecordPeerExchangeError(ctx, "pxFailure")
				return
			}
		}
	}
}

func (wakuPX *WakuPeerExchange) Request(ctx context.Context, numPeers int, opts ...PeerExchangeOption) error {
	params := new(PeerExchangeParameters)
	params.host = wakuPX.h
	params.log = wakuPX.log

	optList := DefaultOptions(wakuPX.h)
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		metrics.RecordPeerExchangeError(ctx, "dialError")
		return ErrNoPeersAvailable
	}

	requestRPC := &pb.PeerExchangeRPC{
		Query: &pb.PeerExchangeQuery{
			NumPeers: uint64(numPeers),
		},
	}

	return wakuPX.sendPeerExchangeRPCToPeer(ctx, requestRPC, params.selectedPeer)
}

// Stop unmounts the peer exchange protocol
func (wakuPX *WakuPeerExchange) Stop() {
	if wakuPX.cancel == nil {
		return
	}
	wakuPX.cancel()
	wakuPX.h.RemoveStreamHandler(PeerExchangeID_v20alpha1)
	wakuPX.wg.Wait()
}

func (wakuPX *WakuPeerExchange) sendPeerExchangeRPCToPeer(ctx context.Context, rpc *pb.PeerExchangeRPC, peerID peer.ID) error {
	logger := wakuPX.log.With(logging.HostID("peer", peerID))

	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := wakuPX.h.Connect(ctx, wakuPX.h.Peerstore().PeerInfo(peerID))
	if err != nil {
		logger.Error("connecting peer", zap.Error(err))
		return err
	}

	connOpt, err := wakuPX.h.NewStream(ctx, peerID, PeerExchangeID_v20alpha1)
	if err != nil {
		logger.Error("creating stream to peer", zap.Error(err))
		return err
	}
	defer connOpt.Close()

	writer := protoio.NewDelimitedWriter(connOpt)
	err = writer.WriteMsg(rpc)
	if err != nil {
		logger.Error("writing response", zap.Error(err))
		return err
	}

	return nil
}

func (wakuPX *WakuPeerExchange) respond(ctx context.Context, numPeers uint64, peerID peer.ID) error {
	records, err := wakuPX.getENRsFromCache(numPeers)
	if err != nil {
		return err
	}

	responseRPC := &pb.PeerExchangeRPC{}
	responseRPC.Response = new(pb.PeerExchangeResponse)
	responseRPC.Response.PeerInfos = records

	return wakuPX.sendPeerExchangeRPCToPeer(ctx, responseRPC, peerID)
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

func (wakuPX *WakuPeerExchange) findPeers(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	peerRecords, err := wakuPX.disc.FindNodes(ctx, "")
	if err != nil {
		wakuPX.log.Error("finding peers", zap.Error(err))
	}

	cnt := 0
	wakuPX.enrCacheMutex.Lock()
	for _, p := range peerRecords {
		cnt++
		wakuPX.enrCache[p.Node.ID()] = peerRecord{
			idx:  len(wakuPX.enrCache),
			node: p.Node,
		}
	}
	wakuPX.enrCacheMutex.Unlock()

	wakuPX.log.Info("discovered px peers via discv5", zap.Int("count", cnt))

	wakuPX.cleanCache()
}

func (wakuPX *WakuPeerExchange) runPeerExchangeDiscv5Loop(ctx context.Context) {
	defer wakuPX.wg.Done()

	// Runs a discv5 loop adding new peers to the px peer cache
	if wakuPX.disc == nil {
		wakuPX.log.Warn("trying to run discovery v5 (for PX) while it's disabled")
		return
	}

	wakuPX.log.Info("starting peer exchange discovery v5 loop")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// This loop "competes" with the loop in wakunode2
	// For the purpose of collecting px peers, 30 sec intervals should be enough

	wakuPX.findPeers(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			wakuPX.findPeers(ctx)
		}

	}
}
