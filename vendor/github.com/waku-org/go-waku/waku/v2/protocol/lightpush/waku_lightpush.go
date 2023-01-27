package lightpush

import (
	"context"
	"encoding/hex"
	"errors"
	"math"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

// LightPushID_v20beta1 is the current Waku Lightpush protocol identifier
const LightPushID_v20beta1 = libp2pProtocol.ID("/vac/waku/lightpush/2.0.0-beta1")

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidId        = errors.New("invalid request id")
)

type WakuLightPush struct {
	h      host.Host
	relay  *relay.WakuRelay
	cancel context.CancelFunc

	log *zap.Logger
}

// NewWakuRelay returns a new instance of Waku Lightpush struct
func NewWakuLightPush(h host.Host, relay *relay.WakuRelay, log *zap.Logger) *WakuLightPush {
	wakuLP := new(WakuLightPush)
	wakuLP.relay = relay
	wakuLP.h = h
	wakuLP.log = log.Named("lightpush")

	return wakuLP
}

// Start inits the lighpush protocol
func (wakuLP *WakuLightPush) Start(ctx context.Context) error {
	if wakuLP.relayIsNotAvailable() {
		return errors.New("relay is required, without it, it is only a client and cannot be started")
	}

	ctx, cancel := context.WithCancel(ctx)

	wakuLP.cancel = cancel
	wakuLP.h.SetStreamHandlerMatch(LightPushID_v20beta1, protocol.PrefixTextMatch(string(LightPushID_v20beta1)), wakuLP.onRequest(ctx))
	wakuLP.log.Info("Light Push protocol started")

	return nil
}

// relayIsNotAvailable determines if this node supports relaying messages for other lightpush clients
func (wakuLp *WakuLightPush) relayIsNotAvailable() bool {
	return wakuLp.relay == nil
}

func (wakuLP *WakuLightPush) onRequest(ctx context.Context) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		logger := wakuLP.log.With(logging.HostID("peer", s.Conn().RemotePeer()))
		requestPushRPC := &pb.PushRPC{}

		writer := protoio.NewDelimitedWriter(s)
		reader := protoio.NewDelimitedReader(s, math.MaxInt32)

		err := reader.ReadMsg(requestPushRPC)
		if err != nil {
			logger.Error("reading request", zap.Error(err))
			metrics.RecordLightpushError(ctx, "decodeRpcFailure")
			return
		}

		logger.Info("request received")
		if requestPushRPC.Query != nil {
			logger.Info("push request")
			response := new(pb.PushResponse)

			pubSubTopic := requestPushRPC.Query.PubsubTopic
			message := requestPushRPC.Query.Message

			// TODO: Assumes success, should probably be extended to check for network, peers, etc
			// It might make sense to use WithReadiness option here?

			_, err := wakuLP.relay.PublishToTopic(ctx, message, pubSubTopic)

			if err != nil {
				logger.Error("publishing message", zap.Error(err))
				response.Info = "Could not publish message"
			} else {
				response.IsSuccess = true
				response.Info = "Totally" // TODO: ask about this
			}

			responsePushRPC := &pb.PushRPC{}
			responsePushRPC.RequestId = requestPushRPC.RequestId
			responsePushRPC.Response = response

			err = writer.WriteMsg(responsePushRPC)
			if err != nil {
				logger.Error("writing response", zap.Error(err))
				_ = s.Reset()
			} else {
				logger.Info("response sent")
			}
		}

		if requestPushRPC.Response != nil {
			if requestPushRPC.Response.IsSuccess {
				logger.Info("request success")
			} else {
				logger.Info("request failure", zap.String("info=", requestPushRPC.Response.Info))
			}
		}
	}
}

func (wakuLP *WakuLightPush) request(ctx context.Context, req *pb.PushRequest, opts ...LightPushOption) (*pb.PushResponse, error) {
	params := new(LightPushParameters)
	params.host = wakuLP.h
	params.log = wakuLP.log

	optList := append(DefaultOptions(wakuLP.h), opts...)
	for _, opt := range optList {
		opt(params)
	}

	if params.selectedPeer == "" {
		metrics.RecordLightpushError(ctx, "dialError")
		return nil, ErrNoPeersAvailable
	}

	if len(params.requestId) == 0 {
		return nil, ErrInvalidId
	}

	logger := wakuLP.log.With(logging.HostID("peer", params.selectedPeer))
	// We connect first so dns4 addresses are resolved (NewStream does not do it)
	err := wakuLP.h.Connect(ctx, wakuLP.h.Peerstore().PeerInfo(params.selectedPeer))
	if err != nil {
		logger.Error("connecting peer", zap.Error(err))
		return nil, err
	}

	connOpt, err := wakuLP.h.NewStream(ctx, params.selectedPeer, LightPushID_v20beta1)
	if err != nil {
		logger.Error("creating stream to peer", zap.Error(err))
		metrics.RecordLightpushError(ctx, "dialError")
		return nil, err
	}

	defer connOpt.Close()
	defer func() {
		err := connOpt.Reset()
		if err != nil {
			metrics.RecordLightpushError(ctx, "dialError")
			logger.Error("resetting connection", zap.Error(err))
		}
	}()

	pushRequestRPC := &pb.PushRPC{RequestId: hex.EncodeToString(params.requestId), Query: req}

	writer := protoio.NewDelimitedWriter(connOpt)
	reader := protoio.NewDelimitedReader(connOpt, math.MaxInt32)

	err = writer.WriteMsg(pushRequestRPC)
	if err != nil {
		logger.Error("writing request", zap.Error(err))
		return nil, err
	}

	pushResponseRPC := &pb.PushRPC{}
	err = reader.ReadMsg(pushResponseRPC)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		metrics.RecordLightpushError(ctx, "decodeRPCFailure")
		return nil, err
	}

	return pushResponseRPC.Response, nil
}

// Stop unmounts the lightpush protocol
func (wakuLP *WakuLightPush) Stop() {
	if wakuLP.cancel == nil {
		return
	}

	wakuLP.cancel()
	wakuLP.h.RemoveStreamHandler(LightPushID_v20beta1)
}

// PublishToTopic is used to broadcast a WakuMessage to a pubsub topic via lightpush protocol
func (wakuLP *WakuLightPush) PublishToTopic(ctx context.Context, message *pb.WakuMessage, topic string, opts ...LightPushOption) ([]byte, error) {
	if message == nil {
		return nil, errors.New("message can't be null")
	}

	req := new(pb.PushRequest)
	req.Message = message
	req.PubsubTopic = topic

	response, err := wakuLP.request(ctx, req, opts...)
	if err != nil {
		return nil, err
	}

	if response.IsSuccess {
		hash, _, _ := message.Hash()
		wakuLP.log.Info("waku.lightpush published", logging.HexString("hash", hash))
		return hash, nil
	} else {
		return nil, errors.New(response.Info)
	}
}

// Publish is used to broadcast a WakuMessage to the default waku pubsub topic via lightpush protocol
func (wakuLP *WakuLightPush) Publish(ctx context.Context, message *pb.WakuMessage, opts ...LightPushOption) ([]byte, error) {
	return wakuLP.PublishToTopic(ctx, message, relay.DefaultWakuTopic, opts...)
}
