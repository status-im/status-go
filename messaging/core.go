package messaging

import (
	"context"
	"crypto/ecdsa"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/common"
	"github.com/status-im/status-go/messaging/controller"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/reliability"
	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	wakuv2 "github.com/status-im/status-go/messaging/waku"
	wakuv2common "github.com/status-im/status-go/messaging/waku/common"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
	"github.com/status-im/status-go/messaging/wakumetrics"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
)

var (
	ErrWakuIdentityInjectionFailure = errors.New("failed to inject identity into waku")
)

type Core struct {
	config

	identity   *ecdsa.PrivateKey
	waku       wakutypes.Waku
	stack      *common.MessagingStack
	controller *controller.Controller

	publisher *pubsub.Publisher

	wg   sync.WaitGroup
	quit chan struct{}

	wakumetrics *wakumetrics.Client
}

type CoreParams struct {
	Identity       *ecdsa.PrivateKey
	InstallationID string

	NodeKey       *ecdsa.PrivateKey
	WakuConfig    params.WakuV2Config
	ClusterConfig params.ClusterConfig

	TimeSource timesource.Provider
}

func newCore(waku wakutypes.Waku, params CoreParams, config *config) (*Core, error) {
	var err error
	stack := &common.MessagingStack{}

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

	stack.Encryption = encryption.New(
		config.persistence.EncryptionStorage(),
		params.InstallationID,
		config.logger,
		config.tracer,
	)

	stack.Reliability = reliability.NewReliability(
		config.persistence.MVDSStorage(),
		params.Identity,
		config.logger,
	)

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

	return &Core{
		config:     *config,
		identity:   params.Identity,
		waku:       waku,
		stack:      stack,
		controller: controller,
		publisher:  publisher,
		quit:       make(chan struct{}),
	}, nil
}

func NewCore(params CoreParams, options ...Options) (*Core, error) {
	config := newConfig(options...)

	if config.persistence == nil {
		return nil, errors.New("persistence is not configured")
	}

	waku, err := newWaku(wakuParams{
		identity:                        params.Identity,
		nodeKey:                         params.NodeKey,
		wakuConfig:                      params.WakuConfig,
		clusterConfig:                   params.ClusterConfig,
		metricsEnabled:                  config.metricsEnabled,
		onHistoricMessagesRequestFailed: config.onHistoricMessagesRequestFailed,
		onPeerStats: func(status wakutypes.ConnStatus) {
			config.onPeerStats(types.ConnStatus{
				IsOnline: status.IsOnline,
				Peers:    adapters.FromWakuPeerStats(status.Peers),
			})
		},
		timeSource: params.TimeSource,
		logger:     config.logger,
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

	err := c.controller.Stop()
	if err != nil {
		return err
	}

	if c.metricsEnabled {
		err := wakumetrics.UnregisterMetrics()
		if err != nil {
			return err
		}
	}

	err = c.waku.Stop()
	if err != nil {
		return err
	}

	c.wg.Wait()

	return nil
}

type wakuParams struct {
	identity *ecdsa.PrivateKey
	nodeKey  *ecdsa.PrivateKey

	wakuConfig     params.WakuV2Config
	clusterConfig  params.ClusterConfig
	metricsEnabled bool

	onHistoricMessagesRequestFailed func([]byte, peer.AddrInfo, error)
	onPeerStats                     func(wakutypes.ConnStatus)

	timeSource timesource.Provider

	logger *zap.Logger
}

func newWaku(params wakuParams) (*wakuv2.Waku, error) {
	cfg := &wakuv2.Config{
		MaxMessageSize:                         wakuv2common.DefaultMaxMessageSize,
		Host:                                   params.wakuConfig.Host,
		Port:                                   params.wakuConfig.Port,
		LightClient:                            params.wakuConfig.LightClient,
		WakuNodes:                              params.clusterConfig.WakuNodes,
		DiscoveryLimit:                         params.wakuConfig.DiscoveryLimit,
		DiscV5BootstrapNodes:                   params.clusterConfig.DiscV5BootstrapNodes,
		Nameserver:                             params.wakuConfig.Nameserver,
		UDPPort:                                params.wakuConfig.UDPPort,
		AutoUpdate:                             params.wakuConfig.AutoUpdate,
		DefaultShardPubsubTopic:                wakuv2.DefaultShardPubsubTopic(),
		ClusterID:                              params.clusterConfig.ClusterID,
		EnableMissingMessageVerification:       params.wakuConfig.EnableMissingMessageVerification,
		EnableStoreConfirmationForMessagesSent: params.wakuConfig.EnableStoreConfirmationForMessagesSent,
		UseThrottledPublish:                    true,
		MetricsEnabled:                         params.metricsEnabled,
	}

	// Configure peer exchange and discv5 settings based on node type
	if cfg.LightClient {
		cfg.EnablePeerExchangeServer = false
		cfg.EnablePeerExchangeClient = true
		cfg.EnableDiscV5 = false
	} else {
		cfg.EnablePeerExchangeServer = true
		cfg.EnablePeerExchangeClient = false
		cfg.EnableDiscV5 = true
	}

	if params.wakuConfig.MaxMessageSize > 0 {
		cfg.MaxMessageSize = params.wakuConfig.MaxMessageSize
	}

	waku, err := wakuv2.New(
		params.nodeKey,
		cfg,
		params.logger,
		params.timeSource,
		params.onHistoricMessagesRequestFailed,
		params.onPeerStats,
	)
	if err != nil {
		return nil, err
	}

	// Inject the identity into Waku
	err = waku.DeleteKeyPairs()
	if err != nil {
		return nil, err
	}
	_, err = waku.AddKeyPair(params.identity)
	if err != nil {
		return nil, ErrWakuIdentityInjectionFailure
	}

	return waku, nil
}

func (c *Core) startWakuMetrics() error {
	if c.wakumetrics == nil {
		options := []wakumetrics.TelemetryClientOption{
			wakumetrics.WithPeerID(c.waku.PeerID().String()),
		}

		wakuMetricsHandler, err := wakumetrics.NewClient(options...)
		if err != nil {
			return err
		}

		err = wakuMetricsHandler.RegisterWithRegistry()
		if err != nil {
			return err
		}

		// TODO: Remove type assertion once Waku metrics are fully integrated into the Messaging module.
		c.waku.(*wakuv2.Waku).SetMetricsHandler(wakuMetricsHandler)

		c.wakumetrics = wakuMetricsHandler
	}

	return nil
}

func (c *Core) metrics() string {
	// TODO: Remove type assertion once Waku metrics are fully integrated into the Messaging module.
	return c.waku.(*wakuv2.Waku).Metrics()
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
	if err == encryption.ErrNoEncryptionKey {
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
	var encryptionMessage encryption.ProtocolMessage
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
