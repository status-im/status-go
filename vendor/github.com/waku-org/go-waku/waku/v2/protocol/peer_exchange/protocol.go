package peer_exchange

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/logging"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb"
	"go.uber.org/zap"
)

// PeerExchangeID_v20alpha1 is the current Waku Peer Exchange protocol identifier
const PeerExchangeID_v20alpha1 = libp2pProtocol.ID("/vac/waku/peer-exchange/2.0.0-alpha1")
const MaxCacheSize = 1000

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidId        = errors.New("invalid request id")
)

type PeerConnector interface {
	Subscribe(context.Context, <-chan v2.PeerData)
}

type WakuPeerExchange struct {
	h    host.Host
	disc *discv5.DiscoveryV5

	log *zap.Logger

	cancel context.CancelFunc

	wg            sync.WaitGroup
	peerConnector PeerConnector
	enrCache      *enrCache
}

// NewWakuPeerExchange returns a new instance of WakuPeerExchange struct
func NewWakuPeerExchange(disc *discv5.DiscoveryV5, peerConnector PeerConnector, log *zap.Logger) (*WakuPeerExchange, error) {
	newEnrCache, err := newEnrCache(MaxCacheSize)
	if err != nil {
		return nil, err
	}
	wakuPX := new(WakuPeerExchange)
	wakuPX.disc = disc
	wakuPX.log = log.Named("wakupx")
	wakuPX.enrCache = newEnrCache
	wakuPX.peerConnector = peerConnector

	return wakuPX, nil
}

// Sets the host to be able to mount or consume a protocol
func (wakuPX *WakuPeerExchange) SetHost(h host.Host) {
	wakuPX.h = h
}

// Start inits the peer exchange protocol
func (wakuPX *WakuPeerExchange) Start(ctx context.Context) error {
	if wakuPX.cancel != nil {
		return errors.New("peer exchange already started")
	}

	wakuPX.wg.Wait() // Waiting for any go routines to stop

	ctx, cancel := context.WithCancel(ctx)
	wakuPX.cancel = cancel

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

			records, err := wakuPX.enrCache.getENRs(int(requestRPC.Query.NumPeers))
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
	wakuPX.wg.Wait()
}

func (wakuPX *WakuPeerExchange) iterate(ctx context.Context) error {
	iterator, err := wakuPX.disc.Iterator()
	if err != nil {
		return fmt.Errorf("obtaining iterator: %w", err)
	}
	// Closing iterator
	defer iterator.Close()

	for iterator.Next() {
		_, addresses, err := enr.Multiaddress(iterator.Node())
		if err != nil {
			wakuPX.log.Error("extracting multiaddrs from enr", zap.Error(err))
			continue
		}

		if len(addresses) == 0 {
			continue
		}

		wakuPX.log.Debug("Discovered px peers via discv5")
		wakuPX.enrCache.updateCache(iterator.Node())

		select {
		case <-ctx.Done():
			return nil
		default:
		}
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

	for {
		err := wakuPX.iterate(ctx)
		if err != nil {
			wakuPX.log.Debug("iterating peer exchange", zap.Error(err))
			time.Sleep(2 * time.Second)
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
