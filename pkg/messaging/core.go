package messaging

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/connection"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging/common"
	"github.com/status-im/status-go/pkg/messaging/controller"
	"github.com/status-im/status-go/pkg/messaging/events"
	encryption2 "github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	reliabilitypb "github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation"
	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	wakuv3 "github.com/status-im/status-go/pkg/messaging/waku"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
	wakumetrics2 "github.com/status-im/status-go/pkg/messaging/wakumetrics"
	"github.com/status-im/status-go/pkg/pubsub"
)

type Core struct {
	config

	identity   *ecdsa.PrivateKey
	waku       wakutypes.Waku
	timeSource timesource.Provider
	stack      *common.MessagingStack
	controller *controller.Controller

	publisher *pubsub.Publisher

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	quit   chan struct{}

	connectionState connection.State

	wakumetrics *wakumetrics2.Client
}

type sdsEnvelopeHashesTracker interface {
	TrackedEnvelopeHashes(identifier []byte) ([]string, error)
}

type sdsApplicationMessageIDTracker interface {
	TakeApplicationMessageIDForSDS(sdsMessageID []byte) ([]byte, bool)
}

// Mode selects Core (full/relay) vs Edge (light) operation. It is re-exported
// from the waku layer so callers configure the messaging API without importing
// the transport package directly.
type Mode = wakuv3.Mode

const (
	ModeCore = wakuv3.ModeCore
	ModeEdge = wakuv3.ModeEdge
)

// ModeFromLightClient maps the legacy WakuV2Config.LightClient boolean onto a Mode.
func ModeFromLightClient(lightClient bool) Mode {
	return wakuv3.ModeFromLightClient(lightClient)
}

type CoreParams struct {
	Identity       *ecdsa.PrivateKey
	InstallationID string

	NodeKey    *ecdsa.PrivateKey
	WakuConfig params.WakuV2Config

	// Fleet is the network preset the waku node resolves its peers from; Mode
	// selects Core (full/relay) vs Edge (light). Together they give the
	// messaging API its Start(fleet, mode)-shaped configuration.
	Fleet string
	Mode  Mode

	TimeSource timesource.Provider
}

func newCore(waku wakutypes.Waku, params CoreParams, config *config) (*Core, error) {
	var err error
	stack := &common.MessagingStack{}
	var core *Core

	stack.Transport, err = transport.NewTransport(
		waku,
		params.Identity,
		config.persistence.TransportStorage().KeysStorage(),
		config.persistence.TransportStorage().ProcessedMessageIDsCacheStorage(),
		config.envelopesMonitorConfig,
		config.logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transport instance")
	}

	stack.Segmentation = segmentation.NewSegmenter(
		config.persistence.SegmentationStorage(),
		config.logger,
	)

	stack.Encryption = encryption2.New(
		config.persistence.EncryptionStorage(),
		params.InstallationID,
		config.logger,
		config.tracer,
	)

	missingDepsHandler := func(messageID string, missingDeps []string, channelID string) error {
		if config.missingDepsObserver != nil {
			config.missingDepsObserver(messageID, missingDeps, channelID)
		}
		return core.fetchMissingDependenciesAsync(messageID, missingDeps, channelID)
	}

	stack.Reliability, err = reliability.NewReliability(
		config.persistence.MVDSStorage(),
		params.Identity,
		missingDepsHandler,
		config.logger,
	)
	if err != nil {
		return nil, err
	}

	publisher := pubsub.NewPublisher()

	controller := controller.NewController(
		params.Identity,
		stack,
		config.persistence.MessageConfirmationStorage(),
		config.persistence.HashRatchetStorage(),
		publisher,
		config.logger,
		config.tracer,
	)

	timeSource := params.TimeSource
	if timeSource == nil {
		timeSource = timesource.DefaultService()
	}

	ctx, cancel := context.WithCancel(context.Background())

	core = &Core{
		config:     *config,
		identity:   params.Identity,
		waku:       waku,
		timeSource: timeSource,
		stack:      stack,
		controller: controller,
		publisher:  publisher,
		ctx:        ctx,
		cancel:     cancel,
		quit:       make(chan struct{}),
	}

	stack.Reliability.SetRetrievalHintProvider(core.resolveSDSRetrievalHint)
	stack.Reliability.SetMessageSentHandler(core.handleSDSMessageSent)

	return core, nil
}

func buildSDSRetrievalHint(logger *zap.Logger, tracker sdsEnvelopeHashesTracker, messageID string) []byte {
	decodedMessageID, err := cryptotypes.DecodeHex(messageID)
	if err != nil {
		logger.Debug("failed to decode SDS message ID for retrieval hint",
			zap.String("messageID", messageID),
			zap.Error(err),
		)
		return nil
	}

	hashes, err := tracker.TrackedEnvelopeHashes(decodedMessageID)
	if err != nil {
		logger.Debug("no tracked envelope hash for SDS message ID",
			zap.String("messageID", messageID),
			zap.Error(err),
		)
		return nil
	}

	envelopeHashes := make([][]byte, len(hashes))
	for i, hash := range hashes {
		envelopeHashes[i] = []byte(hash)
	}

	hint, err := proto.Marshal(&reliabilitypb.RetrievalHint{
		EnvelopeHashes: envelopeHashes,
	})
	if err != nil {
		logger.Debug("failed to marshal SDS retrieval hint",
			zap.String("messageID", messageID),
			zap.Error(err),
		)
		return nil
	}

	return hint
}

func (c *Core) resolveSDSRetrievalHint(messageID string) []byte {
	return buildSDSRetrievalHint(c.logger, c.stack.Transport, messageID)
}

func (c *Core) handleSDSMessageSent(sdsMessageID string) {
	publishSDSMessageDelivered(c.logger, c.stack.Transport, c.publisher, sdsMessageID)
}

func publishSDSMessageDelivered(
	logger *zap.Logger,
	tracker sdsApplicationMessageIDTracker,
	publisher *pubsub.Publisher,
	sdsMessageID string,
) {
	decodedSDSMessageID, err := cryptotypes.DecodeHex(sdsMessageID)
	if err != nil {
		logger.Debug("failed to decode SDS delivery confirmation", zap.String("messageID", sdsMessageID), zap.Error(err))
		return
	}

	applicationMessageID, ok := tracker.TakeApplicationMessageIDForSDS(decodedSDSMessageID)
	if !ok {
		logger.Debug("ignoring delivery confirmation for untracked SDS message", zap.String("messageID", sdsMessageID))
		return
	}

	pubsub.Publish(publisher, events.DeliveredMessage{MessageIDs: [][]byte{applicationMessageID}})
}

func NewCore(params CoreParams, options ...Options) (*Core, error) {
	config := newConfig(options...)

	if config.persistence == nil {
		return nil, errors.New("persistence is not configured")
	}

	waku, err := newWaku(wakuParams{
		nodeKey:        params.NodeKey,
		fleet:          params.Fleet,
		mode:           params.Mode,
		port:           params.WakuConfig.Port,
		udpPort:        params.WakuConfig.UDPPort,
		nameserver:     params.WakuConfig.Nameserver,
		metricsEnabled: config.metricsEnabled,
		timeSource:     params.TimeSource,
		logger:         config.logger,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create waku instance")
	}

	return newCore(waku, params, config)
}

func (c *Core) API() *API {
	return NewAPI(c)
}

func (c *Core) start() error {
	err := c.waku.Start()
	if err != nil {
		return err
	}

	if c.metricsEnabled {
		err := c.startWakuMetrics()
		if err != nil {
			return err
		}
	}

	err = c.controller.Start()
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) stop() error {
	close(c.quit)
	c.cancel()
	defer c.wg.Wait()

	err := c.controller.Stop()
	if err != nil {
		return err
	}

	if c.metricsEnabled {
		err := wakumetrics2.UnregisterMetrics()
		if err != nil {
			return err
		}
	}

	err = c.waku.Stop()
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) fetchMissingDependenciesAsync(messageID string, missingDeps []string, channelID string) error {
	if len(missingDeps) == 0 {
		return nil
	}

	select {
	case <-c.ctx.Done():
		return nil
	default:
	}

	c.wg.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer c.wg.Done()

		fetchCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
		defer cancel()

		alreadyProcessed, err := c.stack.Transport.AlreadyProcessed(missingDeps)
		if err != nil {
			c.logger.Debug("failed to check missing dependencies cache",
				zap.String("messageID", messageID),
				zap.String("channelID", channelID),
				zap.Strings("missingDeps", missingDeps),
				zap.Error(err),
			)
			return
		}

		missingDepsToFetch := make([]string, 0, len(missingDeps))
		for _, missingDep := range missingDeps {
			if !alreadyProcessed[missingDep] {
				missingDepsToFetch = append(missingDepsToFetch, missingDep)
			}
		}
		if len(missingDepsToFetch) == 0 {
			return
		}

		err = c.stack.Transport.FetchMessagesByHashes(fetchCtx, missingDepsToFetch)
		if err != nil {
			c.logger.Debug("failed to fetch missing dependencies from storenode",
				zap.String("messageID", messageID),
				zap.String("channelID", channelID),
				zap.Strings("missingDeps", missingDepsToFetch),
				zap.Error(err),
			)
		}
	}()

	return nil
}

type wakuParams struct {
	nodeKey *ecdsa.PrivateKey

	// fleet + mode fully determine the peer configuration and the Core/Edge
	// policy; the waku node builds the rest of its config from them.
	fleet string
	mode  wakuv3.Mode

	// port / udpPort / nameserver are the only node settings a caller configures
	// (zero/empty falls back to the waku defaults). Everything else — host,
	// discovery limit, max message size, etc. — is defaulted
	// by the waku layer or derived from the mode.
	port       int
	udpPort    int
	nameserver string

	metricsEnabled bool

	timeSource timesource.Provider

	logger *zap.Logger
}

func newWaku(params wakuParams) (*wakuv3.Waku, error) {
	cfg := &wakuv3.Config{
		// Fleet + Mode drive peer resolution and the peer-exchange/discv5/
		// light-client policy inside the waku node (see wakuv2.setDefaults). The
		// host, discovery limit, max message size and default shard topic are
		// filled by the waku layer's setDefaults.
		Fleet:      params.fleet,
		Mode:       params.mode,
		Port:       params.port,
		UDPPort:    params.udpPort,
		Nameserver: params.nameserver,
		// Status nodes advertise the ip/port observed by their peers.
		AutoUpdate:          true,
		UseThrottledPublish: true,
		MetricsEnabled:      params.metricsEnabled,
	}

	waku, err := wakuv3.New(
		params.nodeKey,
		cfg,
		params.logger,
		params.timeSource,
	)
	if err != nil {
		return nil, err
	}

	return waku, nil
}

func (c *Core) startWakuMetrics() error {
	if c.wakumetrics == nil {
		options := []wakumetrics2.TelemetryClientOption{
			wakumetrics2.WithPeerID(c.waku.PeerID().String()),
		}

		wakuMetricsHandler, err := wakumetrics2.NewClient(options...)
		if err != nil {
			return err
		}

		err = wakuMetricsHandler.RegisterWithRegistry()
		if err != nil {
			return err
		}

		// TODO: Remove type assertion once Waku metrics are fully integrated into the Messaging module.
		c.waku.(*wakuv3.Waku).SetMetricsHandler(wakuMetricsHandler)

		c.wakumetrics = wakuMetricsHandler
	}

	return nil
}

func (c *Core) metrics() string {
	// TODO: Remove type assertion once Waku metrics are fully integrated into the Messaging module.
	return c.waku.(*wakuv3.Waku).Metrics()
}

func (c *Core) generateHashRatchetKey(groupID []byte) error {
	key, err := c.stack.Encryption.GenerateHashRatchetKey(groupID)
	if err != nil {
		return err
	}

	keyID, err := key.GetKeyID()
	if err != nil {
		return err
	}

	c.logger.Info("generate hash ratchet key",
		zap.String("group-id", cryptotypes.Bytes2Hex(groupID)),
		zap.String("key-id", cryptotypes.Bytes2Hex(keyID)),
	)

	return nil
}

func (c *Core) encryptWithHashRatchet(groupID []byte, payload []byte) ([]byte, []byte, uint32, error) {
	encryptedPayload, ratchet, newSeqNo, err := c.stack.Encryption.EncryptWithHashRatchet(groupID, payload)
	if err == encryption2.ErrNoEncryptionKey {
		_, err := c.stack.Encryption.GenerateHashRatchetKey(groupID)
		if err != nil {
			return nil, nil, 0, err
		}
		encryptedPayload, ratchet, newSeqNo, err = c.stack.Encryption.EncryptWithHashRatchet(groupID, payload)
		if err != nil {
			return nil, nil, 0, err
		}

	} else if err != nil {
		return nil, nil, 0, err
	}

	keyID, err := ratchet.GetKeyID()
	if err != nil {
		return nil, nil, 0, err
	}

	return encryptedPayload, keyID, newSeqNo, nil
}

func (c *Core) buildHashRatchetMessage(groupID []byte, payload []byte) ([]byte, error) {
	messageSpec, err := c.stack.Encryption.BuildHashRatchetMessage(groupID, payload)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(messageSpec.Message)
}

func (c *Core) decryptMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, data []byte) ([]byte, error) {
	var encryptionMessage encryption2.ProtocolMessage
	err := proto.Unmarshal(data, &encryptionMessage)
	if err != nil {
		return nil, err
	}

	decrypted, err := c.stack.Encryption.HandleMessage(context.Background(), myIdentityKey, theirPublicKey, &encryptionMessage, make([]byte, 0))
	if err != nil {
		return nil, err
	}

	return decrypted.DecryptedMessage, nil
}

func (c *Core) connectionChanged(state connection.State) {
	c.stack.Transport.ConnectionChanged(state)

	if !c.connectionState.Offline && state.Offline {
		c.controller.StopReliability()
	}

	if c.connectionState.Offline && !state.Offline {
		err := c.controller.StartReliability()
		if err != nil {
			c.logger.Error("failed to start datasync", zap.Error(err))
		}
	}

	c.connectionState = state
}
