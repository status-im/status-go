package lightpush

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/relay"
	utils "github.com/status-im/go-waku/waku/v2/utils"
)

var log = logging.Logger("waku_lightpush")

const WakuLightPushProtocolId = libp2pProtocol.ID("/vac/waku/lightpush/2.0.0-beta1")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidId        = errors.New("invalid request id")
)

type WakuLightPush struct {
	h     host.Host
	relay *relay.WakuRelay
	ctx   context.Context
}

func NewWakuLightPush(ctx context.Context, h host.Host, relay *relay.WakuRelay) *WakuLightPush {
	wakuLP := new(WakuLightPush)
	wakuLP.relay = relay
	wakuLP.ctx = ctx
	wakuLP.h = h

	wakuLP.h.SetStreamHandler(WakuLightPushProtocolId, wakuLP.onRequest)
	log.Info("Light Push protocol started")

	return wakuLP
}

func (wakuLP *WakuLightPush) onRequest(s network.Stream) {
	defer s.Close()

	requestPushRPC := &pb.PushRPC{}

	writer := protoio.NewDelimitedWriter(s)
	reader := protoio.NewDelimitedReader(s, 64*1024)

	err := reader.ReadMsg(requestPushRPC)
	if err != nil {
		log.Error("error reading request", err)
		return
	}

	log.Info(fmt.Sprintf("%s: lightpush message received from %s", s.Conn().LocalPeer(), s.Conn().RemotePeer()))

	if requestPushRPC.Query != nil {
		log.Info("lightpush push request")
		pubSubTopic := relay.Topic(requestPushRPC.Query.PubsubTopic)
		message := requestPushRPC.Query.Message

		response := new(pb.PushResponse)
		if wakuLP.relay != nil {
			// XXX Assumes success, should probably be extended to check for network, peers, etc
			_, err := wakuLP.relay.Publish(wakuLP.ctx, message, &pubSubTopic)

			if err != nil {
				response.IsSuccess = false
				response.Info = "Could not publish message"
			} else {
				response.IsSuccess = true
				response.Info = "Totally" // TODO: ask about this
			}
		} else {
			log.Debug("no relay protocol present, unsuccessful push")
			response.IsSuccess = false
			response.Info = "No relay protocol"
		}

		responsePushRPC := &pb.PushRPC{}
		responsePushRPC.RequestId = requestPushRPC.RequestId
		responsePushRPC.Response = response

		err = writer.WriteMsg(responsePushRPC)
		if err != nil {
			log.Error("error writing response", err)
			s.Reset()
		} else {
			log.Info(fmt.Sprintf("%s: response sent  to %s", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String()))
		}
	}

	if requestPushRPC.Response != nil {
		if requestPushRPC.Response.IsSuccess {
			log.Info("lightpush message success")
		} else {
			log.Info(fmt.Sprintf("lightpush message failure. info=%s", requestPushRPC.Response.Info))
		}
	}
}

type LightPushParameters struct {
	selectedPeer peer.ID
	requestId    []byte

	lp *WakuLightPush
}

type LightPushOption func(*LightPushParameters)

func WithPeer(p peer.ID) LightPushOption {
	return func(params *LightPushParameters) {
		params.selectedPeer = p
	}
}

func WithAutomaticPeerSelection() LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeer(params.lp.h, string(WakuLightPushProtocolId))
		if err == nil {
			params.selectedPeer = *p
		} else {
			log.Info("Error selecting peer: ", err)
		}
	}
}

func WithRequestId(requestId []byte) LightPushOption {
	return func(params *LightPushParameters) {
		params.requestId = requestId
	}
}

func WithAutomaticRequestId() LightPushOption {
	return func(params *LightPushParameters) {
		params.requestId = protocol.GenerateRequestId()
	}
}

func DefaultOptions() []LightPushOption {
	return []LightPushOption{
		WithAutomaticRequestId(),
		WithAutomaticPeerSelection(),
	}
}

func (wakuLP *WakuLightPush) Request(ctx context.Context, req *pb.PushRequest, opts ...LightPushOption) (*pb.PushResponse, error) {
	params := new(LightPushParameters)
	params.lp = wakuLP

	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		return nil, ErrNoPeersAvailable
	}

	if len(params.requestId) == 0 {
		return nil, ErrInvalidId
	}

	connOpt, err := wakuLP.h.NewStream(ctx, params.selectedPeer, WakuLightPushProtocolId)
	if err != nil {
		log.Info("failed to connect to remote peer", err)
		return nil, err
	}

	defer connOpt.Close()
	defer connOpt.Reset()

	pushRequestRPC := &pb.PushRPC{RequestId: hex.EncodeToString(params.requestId), Query: req}

	writer := protoio.NewDelimitedWriter(connOpt)
	reader := protoio.NewDelimitedReader(connOpt, 64*1024)

	err = writer.WriteMsg(pushRequestRPC)
	if err != nil {
		log.Error("could not write request", err)
		return nil, err
	}

	pushResponseRPC := &pb.PushRPC{}
	err = reader.ReadMsg(pushResponseRPC)
	if err != nil {
		log.Error("could not read response", err)
		return nil, err
	}

	return pushResponseRPC.Response, nil
}
