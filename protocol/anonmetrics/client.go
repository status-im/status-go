package anonmetrics

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/appmetrics"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

type ClientConfig struct {
	ShouldSend  bool
	SendAddress *ecdsa.PublicKey
}

type Client struct {
	Config   *ClientConfig
	DB       *appmetrics.Database
	Identity *ecdsa.PrivateKey
	Logger   *zap.Logger

	//messageProcessor is a message processor used to send metric batch messages
	messageProcessor *common.MessageProcessor

	IntervalInc *FibonacciIntervalIncrementer

	// mainLoopQuit is a channel that concurrently orchestrates that the main loop that should be terminated
	mainLoopQuit chan struct{}

	// deleteLoopQuit is a channel that concurrently orchestrates that the delete loop that should be terminated
	deleteLoopQuit chan struct{}
}

func NewClient(processor *common.MessageProcessor) *Client {
	return &Client{
		messageProcessor: processor,
		IntervalInc: &FibonacciIntervalIncrementer{
			Last:    0,
			Current: 1,
		},
	}
}

func (c *Client) mainLoop() error {
	for {
		// Get all unsent metrics grouped by session id
		uam, err := c.DB.GetUnprocessedGroupedBySession()
		if err != nil {
			c.Logger.Error("failed to get unprocessed messages grouped by session", zap.Error(err))
		}

		// Convert the metrics into protobuf
		for _, batch := range uam {
			amb := adaptModelsToProtoBatch(batch, &c.Identity.PublicKey)

			// Generate an ephemeral key per session id
			ephemeralKey, err := crypto.GenerateKey()
			if err != nil {
				c.Logger.Error("failed to generate an ephemeral key", zap.Error(err))
				continue
			}

			// Prepare the protobuf message
			encodedMessage, err := proto.Marshal(amb)
			if err != nil {
				c.Logger.Error("failed to marshal protobuf", zap.Error(err))
				continue
			}
			rawMessage := common.RawMessage{
				Payload:             encodedMessage,
				Sender:              ephemeralKey,
				SkipEncryption:      true,
				SendOnPersonalTopic: true,
				MessageType:         protobuf.ApplicationMetadataMessage_ANONYMOUS_METRIC_BATCH,
			}

			// Send the metrics batch
			_, err = c.messageProcessor.SendPrivate(context.Background(), c.Config.SendAddress, &rawMessage)
			if err != nil {
				c.Logger.Error("failed to send metrics batch message", zap.Error(err))
				continue
			}

			// Mark metrics as processed
			err = c.DB.SetToProcessed(batch)
			if err != nil {
				c.Logger.Error("failed to set metrics as processed in db", zap.Error(err))
			}
		}

		waitFor := time.Duration(c.IntervalInc.Next())
		select {
		case <-time.After(waitFor * time.Second):
		case <-c.mainLoopQuit:
			return nil
		}
	}
}

func (c *Client) startMainLoop() {
	c.stopMainLoop()
	c.mainLoopQuit = make(chan struct{})
	go func() {
		err := c.mainLoop()
		if err != nil {
			c.Logger.Error("main loop exited with an error", zap.Error(err))
		}
	}()
}

func (c *Client) deleteLoop() error {
	for {
		// TODO add a lock on DB from main loop
		oneWeekAgo := time.Now().Add(time.Hour * 24 * 7 * -1)
		err := c.DB.DeleteOlderThan(&oneWeekAgo)
		if err != nil {
			c.Logger.Error("failed to delete metrics older than given time",
				zap.Time("time given", oneWeekAgo),
				zap.Error(err))
		}

		select {
		case <-time.After(time.Hour):
		case <-c.mainLoopQuit:
			return nil
		}
	}
}

func (c *Client) startDeleteLoop() {
	c.stopDeleteLoop()
	c.deleteLoopQuit = make(chan struct{})
	go func() {
		err := c.deleteLoop()
		if err != nil {
			c.Logger.Error("delete loop exited with an error", zap.Error(err))
		}
	}()
}

func (c *Client) Start() error {
	if c.messageProcessor == nil {
		return errors.New("can't start, missing message processor")
	}

	c.startMainLoop()
	c.startDeleteLoop()
	return nil
}

func (c *Client) stopMainLoop() {
	if c.mainLoopQuit != nil {
		close(c.mainLoopQuit)
		c.mainLoopQuit = nil
	}
}

func (c *Client) stopDeleteLoop() {
	if c.deleteLoopQuit != nil {
		close(c.deleteLoopQuit)
		c.deleteLoopQuit = nil
	}
}

func (c *Client) Stop() error {
	c.stopMainLoop()
	c.stopDeleteLoop()
	return nil
}
