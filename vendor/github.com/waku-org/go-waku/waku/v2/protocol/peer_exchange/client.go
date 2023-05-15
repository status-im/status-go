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
	"github.com/waku-org/go-waku/waku/v2/metrics"
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

	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := wakuPX.h.Connect(ctx, wakuPX.h.Peerstore().PeerInfo(params.selectedPeer))
	if err != nil {
		return err
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

		peerInfo, err := wenr.EnodeToPeerInfo(enodeRecord)
		if err != nil {
			return err
		}

		peers = append(peers, *peerInfo)
	}

	if len(peers) != 0 {
		wakuPX.log.Info("connecting to newly discovered peers", zap.Int("count", len(peers)))
		wakuPX.wg.Add(1)
		go func() {
			defer wakuPX.wg.Done()
			for _, p := range peers {
				select {
				case <-ctx.Done():
					return
				case wakuPX.peerConnector.PeerChannel() <- p:
				}
			}
		}()
	}

	return nil
}
