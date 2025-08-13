package messaging

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	datasyncnode "github.com/status-im/mvds/node"
	datasyncproto "github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/connection"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/adapters"
	"github.com/status-im/status-go/messaging/common"
	datasyncpeer "github.com/status-im/status-go/messaging/datasync/peer"
	"github.com/status-im/status-go/messaging/events"
	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	wakutypes "github.com/status-im/status-go/waku/types"
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
}

func NewCore(waku wakutypes.Waku, identity *ecdsa.PrivateKey, db *sql.DB, persistence types.Persistence, encryptionProtocol *encryption.Protocol, options ...Options) (*Core, error) {
	config := config{
		envelopesMonitorConfig: &transport.EnvelopesMonitorConfig{
			IsMailserver: func(ethtypes.EnodeID) bool {
				return false
			},
		},
	}

	for _, option := range options {
		option(&config)
	}

	if config.logger == nil {
		config.logger = zap.NewNop()
	}
	config.envelopesMonitorConfig.Logger = config.logger

	transport, err := transport.NewTransport(
		waku,
		identity,
		&adapters.KeysPersistence{P: persistence},
		&adapters.ProcessedMessageIDsCache{P: persistence},
		config.envelopesMonitorConfig,
		config.logger,
	)
	if err != nil {
		return nil, err
	}

	sender, err := common.NewMessageSender(
		identity,
		db, // FIXME: This should be removed once the database is not needed in the sender
		persistence,
		transport,
		encryptionProtocol,
		config.logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create messageSender")
	}

	return &Core{
		config:                config,
		persistence:           persistence,
		identity:              identity,
		waku:                  waku,
		transport:             transport,
		sender:                sender,
		encryptor:             encryptionProtocol,
		quit:                  make(chan struct{}),
		mvdsStatusChangeEvent: make(chan datasyncnode.PeerStatusChangeEvent, 5),
		publisher:             pubsub.NewPublisher(),
	}, nil
}

func (c *Core) API() *API {
	return NewAPI(c)
}

func (c *Core) start() error {
	// set shared secret handles
	c.sender.SetHandleSharedSecrets(c.API().HandleSharedSecrets)
	err := c.sender.StartDatasync(c.mvdsStatusChangeEvent, c.sendDataSync)
	if err != nil {
		return err
	}

	subscriptions, err := c.encryptor.Start(c.identity)
	if err != nil {
		return err
	}

	// handle stored shared secrets
	err = c.API().HandleSharedSecrets(subscriptions.SharedSecrets)
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

	c.sender.StopDatasync()

	err := c.transport.Stop()
	if err != nil {
		return err
	}

	err = c.encryptor.Stop()
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
	err = c.API().HandleSharedSecrets([]*sharedsecret.Secret{messageSpec.SharedSecret})
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
