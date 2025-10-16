package messaging

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	datasyncnode "github.com/status-im/mvds/node"
	datasyncproto "github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/connection"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/common"
	datasyncpeer "github.com/status-im/status-go/messaging/datasync/peer"
	"github.com/status-im/status-go/messaging/events"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/layers/encryption/sharedsecret"
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

	persistence types.Persistence

	identity  *ecdsa.PrivateKey
	waku      wakutypes.Waku
	transport *transport.Transport
	sender    *common.MessageSender
	encryptor *encryption.Protocol

	wg   sync.WaitGroup
	quit chan struct{}

	connectionState       connection.State
	mvdsStatusChangeEvent chan datasyncnode.PeerStatusChangeEvent

	publisher *pubsub.Publisher

	wakumetrics *wakumetrics.Client
}

type CoreParams struct {
	Identity       *ecdsa.PrivateKey
	InstallationID string

	DB          *sql.DB // FIXME: This should be removed once the database is not needed in the sender
	Persistence types.Persistence

	NodeKey       *ecdsa.PrivateKey
	WakuConfig    params.WakuV2Config
	ClusterConfig params.ClusterConfig

	TimeSource timesource.TimeSource
}

func newCore(waku wakutypes.Waku, params CoreParams, config *config) (*Core, error) {
	transport, err := transport.NewTransport(
		waku,
		params.Identity,
		adapters.NewKeysPersistence(params.Persistence),
		adapters.NewProcessedMessageIDsCache(params.Persistence),
		config.envelopesMonitorConfig,
		config.logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transport instance")
	}

	encryptor := encryption.New(
		adapters.NewEncryptionPersistence(params.Persistence),
		params.InstallationID,
		config.logger,
	)

	sender, err := common.NewMessageSender(
		params.Identity,
		params.DB,
		params.Persistence.MessageSenderStorage(),
		adapters.NewSegmentationPersistence(params.Persistence),
		transport,
		encryptor,
		config.logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create messageSender")
	}

	return &Core{
		config:                *config,
		persistence:           params.Persistence,
		identity:              params.Identity,
		waku:                  waku,
		transport:             transport,
		sender:                sender,
		encryptor:             encryptor,
		quit:                  make(chan struct{}),
		mvdsStatusChangeEvent: make(chan datasyncnode.PeerStatusChangeEvent, 5),
		publisher:             pubsub.NewPublisher(),
	}, nil
}

func NewCore(params CoreParams, options ...Options) (*Core, error) {
	config := newConfig(options...)

	waku, err := newWaku(wakuParams{
		persistence:                     params.Persistence,
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

	// set shared secret handles
	c.sender.SetHandleSharedSecrets(func(s []*sharedsecret.Secret) error {
		return c.API().HandleSharedSecrets(adapters.FromEncryptionSharedSecrets(s))
	})
	err = c.sender.StartDatasync(c.mvdsStatusChangeEvent, c.sendDataSync)
	if err != nil {
		return err
	}

	subscriptions, err := c.encryptor.Start(c.identity)
	if err != nil {
		return err
	}

	// handle stored shared secrets
	err = c.API().HandleSharedSecrets(adapters.FromEncryptionSharedSecrets(subscriptions.SharedSecrets))
	if err != nil {
		return err
	}

	c.startCleanupLoop("messageSegmentsCleanupLoop", c.sender.CleanupSegments)
	c.startCleanupLoop("hashRatchetEncryptedMessagesCleanupLoop", c.sender.CleanupHashRatchetEncryptedMessages)

	// Forward MessageEvent
	go func() {
		defer gocommon.LogOnPanic()

		c.wg.Add(1)
		defer c.wg.Done()

		s, unsub := pubsub.Subscribe[events.MessageEvent](c.sender.Publisher(), 0)
		defer unsub()

		for {
			select {
			case <-c.quit:
				return
			case event := <-s:
				pubsub.Publish(c.publisher, event)
			}
		}
	}()

	return nil
}

func (c *Core) stop() error {
	close(c.quit)

	c.sender.Stop()

	err := c.transport.Stop()
	if err != nil {
		return err
	}

	func() {
		c.wg.Add(1)
		defer c.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := c.transport.ResetFilters(ctx)
		if err != nil {
			c.logger.Warn("could not reset filters", zap.Error(err))
		}
	}()

	err = c.encryptor.Stop()
	if err != nil {
		return err
	}

	if c.metricsEnabled {
		err = wakumetrics.UnregisterMetrics()
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

func (c *Core) sendDataSync(receiver state.PeerID, payload *datasyncproto.Payload) error {
	ctx := context.Background()
	if !payload.IsValid() {
		c.logger.Error("payload is invalid")
		return errors.New("payload is invalid")
	}

	marshalledPayload, err := proto.Marshal(payload)
	if err != nil {
		c.logger.Error("failed to marshal payload")
		return err
	}

	publicKey, err := datasyncpeer.IDToPublicKey(receiver)
	if err != nil {
		c.logger.Error("failed to convert id to public key", zap.Error(err))
		return err
	}

	// Calculate the messageIDs
	messageIDs := make([][]byte, 0, len(payload.Messages))
	hexMessageIDs := make([]string, 0, len(payload.Messages))
	for _, payload := range payload.Messages {
		mid := types.MessageID(&c.identity.PublicKey, payload.Body)
		messageIDs = append(messageIDs, mid)
		hexMessageIDs = append(hexMessageIDs, mid.String())
	}

	messageSpec, err := c.encryptor.BuildEncryptedMessage(c.identity, publicKey, marshalledPayload)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt message")
	}

	// The shared secret needs to be handle before we send a message
	// otherwise the topic might not be set up before we receive a message
	err = c.API().HandleSharedSecrets([]*types.SharedSecret{adapters.FromEncryptionSharedSecret(messageSpec.SharedSecret)})
	if err != nil {
		return err
	}

	hashes, newMessages, err := c.sender.SendMessageSpec(ctx, publicKey, messageSpec, messageIDs)
	if err != nil {
		c.logger.Error("failed to send a datasync message", zap.Error(err))
		return err
	}

	c.logger.Debug("sent private messages", zap.Any("messageIDs", hexMessageIDs), zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)))
	c.transport.TrackMany(messageIDs, hashes, newMessages)

	pubsub.Publish(c.publisher, events.DatasyncMessagesSentEvent{
		Messages: newMessages,
	})

	return nil
}

func (c *Core) connectionChanged(state connection.State) {
	c.transport.ConnectionChanged(state)

	if !c.connectionState.Offline && state.Offline {
		c.sender.StopDatasync()
	}

	if c.connectionState.Offline && !state.Offline {
		err := c.sender.StartDatasync(c.mvdsStatusChangeEvent, c.sendDataSync)
		if err != nil {
			c.logger.Error("failed to start datasync", zap.Error(err))
		}
	}
}

func (c *Core) resetDatasyncForPeer(publicKey *ecdsa.PublicKey, eventTime uint64) {
	select {
	case c.mvdsStatusChangeEvent <- datasyncnode.PeerStatusChangeEvent{
		PeerID:    datasyncpeer.PublicKeyToPeerID(*publicKey),
		Status:    datasyncnode.OnlineStatus,
		EventTime: eventTime,
	}:
	default:
		c.logger.Debug("mvdsStatusChangeEvent channel is full")
	}
}

func (c *Core) startCleanupLoop(name string, cleanupFunc func() error) {
	logger := c.logger.Named(name)

	go func() {
		defer gocommon.LogOnPanic()

		c.wg.Add(1)
		defer c.wg.Done()

		// Delay by a few minutes to minimize messenger's startup time
		var interval time.Duration = 5 * time.Minute
		for {
			select {
			case <-time.After(interval):
				// Set the regular interval after the first execution
				interval = 1 * time.Hour

				err := cleanupFunc()
				if err != nil {
					logger.Error("failed to cleanup", zap.Error(err))
				}

			case <-c.quit:
				return
			}
		}
	}()
}

type wakuParams struct {
	persistence types.Persistence

	identity *ecdsa.PrivateKey
	nodeKey  *ecdsa.PrivateKey

	wakuConfig     params.WakuV2Config
	clusterConfig  params.ClusterConfig
	metricsEnabled bool

	onHistoricMessagesRequestFailed func([]byte, peer.AddrInfo, error)
	onPeerStats                     func(wakutypes.ConnStatus)

	timeSource timesource.TimeSource

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
		adapters.NewWakuProtectedTopics(params.persistence),
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

	go func() {
		defer gocommon.LogOnPanic()

		c.wg.Add(1)
		defer c.wg.Done()

		sentMessagesSub, unsubSentMessages := pubsub.Subscribe[events.MessageEvent](c.publisher, 100)
		defer unsubSentMessages()

		sentDatasyncSub, unsubSentDatasync := pubsub.Subscribe[events.DatasyncMessagesSentEvent](c.publisher, 100)
		defer unsubSentDatasync()

		for {
			select {
			case sub := <-sentMessagesSub:
				if sub.Type != events.RawMessageSent {
					continue
				}
				msg := sub.RawMessage
				c.wakumetrics.PushRawMessageByType(
					msg.PubsubTopic,
					msg.ContentTopic,
					msg.MessageType.String(),
					uint32(len(msg.Payload)),
				)

			case sub := <-sentDatasyncSub:
				for _, msg := range sub.Messages {
					c.wakumetrics.PushRawMessageByType(
						msg.PubsubTopic,
						msg.Topic.String(),
						"DATASYNC",
						uint32(len(msg.Payload)),
					)
				}

			case <-c.quit:
				return
			}
		}
	}()

	return nil
}

func (c *Core) metrics() string {
	// TODO: Remove type assertion once Waku metrics are fully integrated into the Messaging module.
	return c.waku.(*wakuv2.Waku).Metrics()
}

func (c *Core) generateHashRatchetKey(groupID []byte) error {
	key, err := c.encryptor.GenerateHashRatchetKey(groupID)
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
	encryptedPayload, ratchet, newSeqNo, err := c.encryptor.EncryptWithHashRatchet(groupID, payload)
	if err == encryption.ErrNoEncryptionKey {
		_, err := c.encryptor.GenerateHashRatchetKey(groupID)
		if err != nil {
			return nil, nil, 0, err
		}
		encryptedPayload, ratchet, newSeqNo, err = c.encryptor.EncryptWithHashRatchet(groupID, payload)
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
	messageSpec, err := c.encryptor.BuildHashRatchetMessage(groupID, payload)
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

	decrypted, err := c.encryptor.HandleMessage(myIdentityKey, theirPublicKey, &encryptionMessage, make([]byte, 0))
	if err != nil {
		return nil, err
	}

	return decrypted.DecryptedMessage, nil
}
