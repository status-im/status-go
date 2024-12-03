package lightpush

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush/pb"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

// LightPushID_v20beta1 is the current Waku LightPush protocol identifier
const LightPushID_v20beta1 = libp2pProtocol.ID("/vac/waku/lightpush/2.0.0-beta1")
const LightPushENRField = uint8(1 << 3)

var (
	ErrNoPeersAvailable = errors.New("no suitable remote peers")
	ErrInvalidID        = errors.New("invalid request id")
)

// WakuLightPush is the implementation of the Waku LightPush protocol
type WakuLightPush struct {
	h       host.Host
	relay   *relay.WakuRelay
	limiter *utils.RateLimiter
	cancel  context.CancelFunc
	pm      *peermanager.PeerManager
	metrics Metrics

	log *zap.Logger
}

// NewWakuLightPush returns a new instance of Waku Lightpush struct
// Takes an optional peermanager if WakuLightPush is being created along with WakuNode.
// If using libp2p host, then pass peermanager as nil
func NewWakuLightPush(relay *relay.WakuRelay, pm *peermanager.PeerManager, reg prometheus.Registerer, log *zap.Logger, opts ...Option) *WakuLightPush {
	wakuLP := new(WakuLightPush)
	wakuLP.relay = relay
	wakuLP.log = log.Named("lightpush")
	wakuLP.pm = pm
	wakuLP.metrics = newMetrics(reg)

	params := &LightpushParameters{}
	opts = append(DefaultLightpushOptions(), opts...)
	for _, opt := range opts {
		opt(params)
	}

	wakuLP.limiter = utils.NewRateLimiter(params.limitR, params.limitB)

	return wakuLP
}

// Sets the host to be able to mount or consume a protocol
func (wakuLP *WakuLightPush) SetHost(h host.Host) {
	wakuLP.h = h
}

// Start inits the lighpush protocol
func (wakuLP *WakuLightPush) Start(ctx context.Context) error {
	if wakuLP.relayIsNotAvailable() {
		return errors.New("relay is required, without it, it is only a client and cannot be started")
	}

	if wakuLP.pm != nil {
		wakuLP.pm.RegisterWakuProtocol(LightPushID_v20beta1, LightPushENRField)
	}

	ctx, cancel := context.WithCancel(ctx)

	wakuLP.cancel = cancel
	wakuLP.h.SetStreamHandlerMatch(LightPushID_v20beta1, protocol.PrefixTextMatch(string(LightPushID_v20beta1)), wakuLP.onRequest(ctx))
	wakuLP.log.Info("Light Push protocol started")

	return nil
}

// relayIsNotAvailable determines if this node supports relaying messages for other lightpush clients
func (wakuLP *WakuLightPush) relayIsNotAvailable() bool {
	return wakuLP.relay == nil
}

func (wakuLP *WakuLightPush) onRequest(ctx context.Context) func(network.Stream) {
	return func(stream network.Stream) {
		logger := wakuLP.log.With(logging.HostID("peer", stream.Conn().RemotePeer()))
		requestPushRPC := &pb.PushRpc{}

		responsePushRPC := &pb.PushRpc{
			Response: &pb.PushResponse{},
		}

		if !wakuLP.limiter.Allow(stream.Conn().RemotePeer()) {
			wakuLP.metrics.RecordError(rateLimitFailure)
			responseMsg := "exceeds the rate limit"
			responsePushRPC.Response.Info = &responseMsg
			responsePushRPC.RequestId = pb.REQUESTID_RATE_LIMITED
			wakuLP.reply(stream, responsePushRPC, logger)
			return
		}

		reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

		err := reader.ReadMsg(requestPushRPC)
		if err != nil {
			logger.Error("reading request", zap.Error(err))
			wakuLP.metrics.RecordError(decodeRPCFailure)
			if err := stream.Reset(); err != nil {
				wakuLP.log.Error("resetting connection", zap.Error(err))
			}
			return
		}

		responsePushRPC.RequestId = requestPushRPC.RequestId
		if err := requestPushRPC.ValidateRequest(); err != nil {
			responseMsg := "invalid request: " + err.Error()
			responsePushRPC.Response.Info = &responseMsg
			wakuLP.metrics.RecordError(requestBodyFailure)
			wakuLP.reply(stream, responsePushRPC, logger)
			return
		}

		logger = logger.With(zap.String("requestID", requestPushRPC.RequestId))

		logger.Info("push request")

		pubSubTopic := requestPushRPC.Request.PubsubTopic
		message := requestPushRPC.Request.Message

		wakuLP.metrics.RecordMessage()

		// TODO: Assumes success, should probably be extended to check for network, peers, etc
		// It might make sense to use WithReadiness option here?

		_, err = wakuLP.relay.Publish(ctx, message, relay.WithPubSubTopic(pubSubTopic))
		if err != nil {
			logger.Error("publishing message", zap.Error(err))
			wakuLP.metrics.RecordError(messagePushFailure)
			responseMsg := fmt.Sprintf("Could not publish message: %s", err.Error())
			responsePushRPC.Response.Info = &responseMsg
		} else {
			responsePushRPC.Response.IsSuccess = true
			responseMsg := "OK"
			responsePushRPC.Response.Info = &responseMsg
		}

		wakuLP.reply(stream, responsePushRPC, logger)

		logger.Info("response sent")

		stream.Close()

		if responsePushRPC.Response.IsSuccess {
			logger.Info("request success")
		} else {
			logger.Info("request failure", zap.String("info", responsePushRPC.GetResponse().GetInfo()))
		}
	}
}

func (wakuLP *WakuLightPush) reply(stream network.Stream, responsePushRPC *pb.PushRpc, logger *zap.Logger) {
	writer := pbio.NewDelimitedWriter(stream)
	err := writer.WriteMsg(responsePushRPC)
	if err != nil {
		wakuLP.metrics.RecordError(writeResponseFailure)
		logger.Error("writing response", zap.Error(err))
		if err := stream.Reset(); err != nil {
			wakuLP.log.Error("resetting connection", zap.Error(err))
		}
		return
	}
	stream.Close()
}

// request sends a message via lightPush protocol to either a specified peer or peer that is selected.
func (wakuLP *WakuLightPush) request(ctx context.Context, req *pb.PushRequest, params *lightPushRequestParameters, peerID peer.ID) (*pb.PushResponse, error) {

	logger := wakuLP.log.With(logging.HostID("peer", peerID))

	stream, err := wakuLP.h.NewStream(ctx, peerID, LightPushID_v20beta1)
	if err != nil {
		wakuLP.metrics.RecordError(dialFailure)
		if wakuLP.pm != nil {
			wakuLP.pm.HandleDialError(err, peerID)
		}
		return nil, err
	}
	pushRequestRPC := &pb.PushRpc{RequestId: hex.EncodeToString(params.requestID), Request: req}
	if err = pushRequestRPC.ValidateRequest(); err != nil {
		return nil, err
	}

	writer := pbio.NewDelimitedWriter(stream)
	reader := pbio.NewDelimitedReader(stream, math.MaxInt32)

	err = writer.WriteMsg(pushRequestRPC)
	if err != nil {
		wakuLP.metrics.RecordError(writeRequestFailure)
		logger.Error("writing request", zap.Error(err))
		if err := stream.Reset(); err != nil {
			wakuLP.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	pushResponseRPC := &pb.PushRpc{}
	err = reader.ReadMsg(pushResponseRPC)
	if err != nil {
		logger.Error("reading response", zap.Error(err))
		wakuLP.metrics.RecordError(decodeRPCFailure)
		if err := stream.Reset(); err != nil {
			wakuLP.log.Error("resetting connection", zap.Error(err))
		}
		return nil, err
	}

	stream.Close()

	if err = pushResponseRPC.ValidateResponse(pushRequestRPC.RequestId); err != nil {
		wakuLP.metrics.RecordError(responseBodyFailure)
		return nil, fmt.Errorf("invalid response: %w", err)
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

func (wakuLP *WakuLightPush) handleOpts(ctx context.Context, message *wpb.WakuMessage, opts ...RequestOption) (*lightPushRequestParameters, error) {
	params := new(lightPushRequestParameters)
	params.host = wakuLP.h
	params.log = wakuLP.log
	params.pm = wakuLP.pm
	var err error

	optList := append(DefaultOptions(wakuLP.h), opts...)
	for _, opt := range optList {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	if params.pubsubTopic == "" {
		params.pubsubTopic, err = protocol.GetPubSubTopicFromContentTopic(message.ContentTopic)
		if err != nil {
			return nil, err
		}
	}

	if params.pm != nil && params.peerAddr != nil {
		pData, err := wakuLP.pm.AddPeer(params.peerAddr, peerstore.Static, []string{params.pubsubTopic}, LightPushID_v20beta1)
		if err != nil {
			return nil, err
		}
		wakuLP.pm.Connect(pData)
		params.selectedPeers = append(params.selectedPeers, pData.AddrInfo.ID)
	}
	reqPeerCount := params.maxPeers - len(params.selectedPeers)
	if params.pm != nil && reqPeerCount > 0 {
		var selectedPeers peer.IDSlice
		//TODO: update this to work with multiple peer selection
		selectedPeers, err = wakuLP.pm.SelectPeers(
			peermanager.PeerSelectionCriteria{
				SelectionType: params.peerSelectionType,
				Proto:         LightPushID_v20beta1,
				PubsubTopics:  []string{params.pubsubTopic},
				SpecificPeers: params.preferredPeers,
				MaxPeers:      reqPeerCount,
				Ctx:           ctx,
			},
		)
		if err == nil {
			params.selectedPeers = append(params.selectedPeers, selectedPeers...)
		}
	}
	if len(params.selectedPeers) == 0 {
		if err != nil {
			params.log.Error("selecting peers", zap.Error(err))
			wakuLP.metrics.RecordError(peerNotFoundFailure)
			return nil, ErrNoPeersAvailable
		}
	}
	return params, nil
}

// Publish is used to broadcast a WakuMessage to the pubSubTopic (which is derived from the
// contentTopic) via lightpush protocol. If auto-sharding is not to be used, then the
// `WithPubSubTopic` option should be provided to publish the message to an specific pubSubTopic
func (wakuLP *WakuLightPush) Publish(ctx context.Context, message *wpb.WakuMessage, opts ...RequestOption) (wpb.MessageHash, error) {
	if message == nil {
		return wpb.MessageHash{}, errors.New("message can't be null")
	}

	params, err := wakuLP.handleOpts(ctx, message, opts...)
	if err != nil {
		return wpb.MessageHash{}, err
	}
	req := new(pb.PushRequest)
	req.Message = message
	req.PubsubTopic = params.pubsubTopic

	logger := message.Logger(wakuLP.log, params.pubsubTopic)

	logger.Debug("publishing message", zap.Stringers("peers", params.selectedPeers))
	var wg sync.WaitGroup
	responses := make([]*pb.PushResponse, params.selectedPeers.Len())
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for i, peerID := range params.selectedPeers {
		wg.Add(1)
		go func(index int, id peer.ID) {
			defer utils.LogOnPanic()
			paramsValue := *params
			paramsValue.requestID = protocol.GenerateRequestID()
			defer wg.Done()
			response, err := wakuLP.request(reqCtx, req, &paramsValue, id)
			if err != nil {
				logger.Error("could not publish message", zap.Error(err), zap.Stringer("peer", id))
			}
			responses[index] = response
		}(i, peerID)
	}
	wg.Wait()
	var successCount int
	errMsg := "lightpush error"

	for _, response := range responses {
		if response.GetIsSuccess() {
			successCount++
		} else {
			if response.GetInfo() != "" {
				errMsg += *response.Info
			}
		}
	}

	//in case of partial failure, should we retry here or build a layer above that takes care of these things?
	if successCount > 0 {
		hash := message.Hash(params.pubsubTopic)
		utils.MessagesLogger("lightpush").Debug("waku.lightpush published", logging.HexBytes("hash", hash[:]), zap.Int("num-peers", len(responses)))
		return hash, nil
	}

	return wpb.MessageHash{}, errors.New(errMsg)
}
