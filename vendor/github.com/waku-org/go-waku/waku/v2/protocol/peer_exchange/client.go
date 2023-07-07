package peer_exchange

import (
	"bytes"
	"context"
	"math"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio/pbio"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/peers"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb"
	"go.uber.org/zap"
)

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

	connOpt, err := wakuPX.h.NewStream(ctx, params.selectedPeer, PeerExchangeID_v20alpha1)
	if err != nil {
		return err
	}
	defer connOpt.Close()

	writer := pbio.NewDelimitedWriter(connOpt)
	err = writer.WriteMsg(requestRPC)
	if err != nil {
		return err
	}

	reader := pbio.NewDelimitedReader(connOpt, math.MaxInt32)
	responseRPC := &pb.PeerExchangeRPC{}
	err = reader.ReadMsg(responseRPC)
	if err != nil {
		return err
	}

	return wakuPX.handleResponse(ctx, responseRPC.Response)
}

func (wakuPX *WakuPeerExchange) handleResponse(ctx context.Context, response *pb.PeerExchangeResponse) error {
	var discoveredPeers []struct {
		addrInfo peer.AddrInfo
		enr      *enode.Node
	}

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

		addrInfo, err := wenr.EnodeToPeerInfo(enodeRecord)
		if err != nil {
			return err
		}

		discoveredPeers = append(discoveredPeers, struct {
			addrInfo peer.AddrInfo
			enr      *enode.Node
		}{
			addrInfo: *addrInfo,
			enr:      enodeRecord,
		})
	}

	if len(discoveredPeers) != 0 {
		wakuPX.log.Info("connecting to newly discovered peers", zap.Int("count", len(discoveredPeers)))
		wakuPX.wg.Add(1)
		go func() {
			defer wakuPX.wg.Done()

			peerCh := make(chan v2.PeerData)
			defer close(peerCh)
			wakuPX.peerConnector.Subscribe(ctx, peerCh)
			for _, p := range discoveredPeers {
				peer := v2.PeerData{
					Origin:   peers.PeerExchange,
					AddrInfo: p.addrInfo,
					ENR:      p.enr,
				}
				select {
				case <-ctx.Done():
					return
				case peerCh <- peer:
				}
			}
		}()
	}

	return nil
}
