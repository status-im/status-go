package controller

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/pkg/messaging/adapters"
	common2 "github.com/status-im/status-go/pkg/messaging/common"
	processor2 "github.com/status-im/status-go/pkg/messaging/controller/processor"
	sender2 "github.com/status-im/status-go/pkg/messaging/controller/sender"
	"github.com/status-im/status-go/pkg/messaging/events"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

type Controller struct {
	identity  *ecdsa.PrivateKey
	stack     *common2.MessagingStack
	sender    *sender2.Sender
	processor *processor2.Processor

	messageConfirmationStorage common2.MessageConfirmationPersistence
	hashRatchetStorage         common2.HashRatchetPersistence

	publisher *pubsub.Publisher
	logger    *zap.Logger

	wg   sync.WaitGroup
	quit chan struct{}
}

func NewController(
	identity *ecdsa.PrivateKey,
	stack *common2.MessagingStack,
	messageConfirmationStorage common2.MessageConfirmationPersistence,
	hashRatchetStorage common2.HashRatchetPersistence,
	publisher *pubsub.Publisher,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Controller {
	return &Controller{
		identity:                   identity,
		stack:                      stack,
		sender:                     sender2.NewSender(identity, stack, logger, tracer),
		processor:                  processor2.NewProcessor(identity, stack, messageConfirmationStorage, hashRatchetStorage, logger, tracer),
		messageConfirmationStorage: messageConfirmationStorage,
		hashRatchetStorage:         hashRatchetStorage,
		publisher:                  publisher,
		logger:                     logger.Named("controller"),
		quit:                       make(chan struct{}),
	}
}

func (c *Controller) Start() error {
	subscriptions, err := c.stack.Encryption.Start(c.identity)
	if err != nil {
		return err
	}

	// process stored shared secrets
	err = c.processor.ProcessSharedSecrets(subscriptions.SharedSecrets)
	if err != nil {
		return err
	}

	err = c.StartReliability()
	if err != nil {
		return err
	}

	c.runSubscriptionsLoop()
	c.runSegmentsCleanupLoop()
	c.runHashRatchetCleanupLoop()

	return nil
}

func (c *Controller) Stop() (rerr error) {
	close(c.quit)

	c.StopReliability()

	err := c.stack.Encryption.Stop()
	if err != nil {
		rerr = errors.Wrap(rerr, "failed to stop encryption layer: "+err.Error())
	}

	err = c.stack.Transport.Stop()
	if err != nil {
		rerr = errors.Wrap(rerr, "failed to stop transport layer: "+err.Error())
	}

	c.wg.Wait()

	return
}

func (c *Controller) StartReliability() error {
	return c.stack.Reliability.Start(c.sender.SendPrivateReliability)
}

func (c *Controller) StopReliability() {
	c.stack.Reliability.Stop()
}

func (c *Controller) SaveHashRatchetMessage(groupID []byte, keyID []byte, m *types.ReceivedMessage) error {
	return c.hashRatchetStorage.SaveMessage(groupID, keyID, m)
}

func (c *Controller) GetHashRatchetMessagesCountForGroup(groupID []byte) (int, error) {
	return c.hashRatchetStorage.GetMessagesCountForGroup(groupID)
}

func (c *Controller) Sender() *sender2.Sender {
	return c.sender
}

func (c *Controller) Processor() *processor2.Processor {
	return c.processor
}

func (c *Controller) runSubscriptionsLoop() {
	c.wg.Add(1)
	defer c.wg.Done()

	go func() {
		defer gocommon.LogOnPanic()

		scheduledSendSub, scheduledSendUnsub := pubsub.Subscribe[sender2.ScheduledReliableSend](c.sender.Publisher(), 100)
		defer scheduledSendUnsub()

		sentSub, sentUnsub := pubsub.Subscribe[sender2.SentMessage](c.sender.Publisher(), 100)
		defer sentUnsub()

		unawareOfInstallationSub, unawareOfInstallationUnsub := pubsub.Subscribe[processor2.SenderUnawareOfInstallation](c.processor.Publisher(), 100)
		defer unawareOfInstallationUnsub()

		for {
			select {
			case scheduledSend, ok := <-scheduledSendSub:
				if !ok {
					return
				}
				// We don't need to receive confirmations from our own devices
				if crypto.IsPubKeyEqual(scheduledSend.Recipient, &c.identity.PublicKey) {
					continue
				}

				confirmation := &common2.MessageConfirmation{
					PublicKey:  crypto.CompressPubkey(scheduledSend.Recipient),
					MessageID:  scheduledSend.MessageID,
					DataSyncID: scheduledSend.ReliabilityMessageID,
				}

				err := c.messageConfirmationStorage.InsertPendingConfirmation(confirmation)
				if err != nil {
					c.logger.Error("failed to insert pending confirmation", zap.Error(err))
				}

			case messageSent, ok := <-sentSub:
				if !ok {
					return
				}
				var pubkey *ecdsa.PublicKey
				if messageSent.Private {
					pubkey = messageSent.Recipient
				}
				pubsub.Publish(c.publisher, events.SentMessage{
					PublicKey:     pubkey,
					Installations: adapters.FromEncryptionInstallations(messageSent.RecipientInstallations),
					MessageIDs:    messageSent.MessageIDs,
				})

			case unawareOfInstallation, ok := <-unawareOfInstallationSub:
				if !ok {
					return
				}
				err := c.sender.SendPrivateAdvertiseBundle(context.Background(), unawareOfInstallation.PublicKey)
				if err != nil {
					c.logger.Error("failed to handle ErrDeviceNotFound", zap.Error(err))
				}

			case <-c.quit:
				return
			}
		}
	}()
}

func (c *Controller) cleanupLoop(logger *zap.Logger, cleanupFunc func() error) {
	c.wg.Add(1)
	defer c.wg.Done()

	go func() {
		defer gocommon.LogOnPanic()

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

func (c *Controller) runSegmentsCleanupLoop() {
	c.cleanupLoop(c.logger.Named("segmentsCleanupLoop"), func() error {
		monthAgo := time.Now().AddDate(0, -1, 0)
		return c.stack.Segmentation.CleanupStaleSegments(monthAgo)
	})
}

func (c *Controller) runHashRatchetCleanupLoop() {
	c.cleanupLoop(c.logger.Named("hashRatchetCleanupLoop"), func() error {
		monthAgo := time.Now().AddDate(0, -1, 0).Unix()
		return c.hashRatchetStorage.DeleteMessagesOlderThan(monthAgo)
	})
}
