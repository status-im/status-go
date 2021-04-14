package anonmetrics

import (
	"crypto/ecdsa"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/appmetrics"
)

type ClientConfig struct {
	ShouldSend bool
	SendAddress *ecdsa.PublicKey
}

type Client struct {
	Config *ClientConfig
	DB *appmetrics.Database
	Identity *ecdsa.PrivateKey
	Logger *zap.Logger

	IntervalInc *FibonacciIntervalIncrementer

	// mainLoopQuit is a channel that concurrently orchestrates that the main loop that should be terminated
	mainLoopQuit chan struct{}
}

func NewClient() *Client {
	c := new(Client)

	// Set default fibonacci start values
	fii := &FibonacciIntervalIncrementer{
		Last:    0,
		Current: 1,
	}
	c.IntervalInc = fii

	return c
}

func (c *Client) mainLoop() error {
	for {
		// Get all unsent metrics grouped by session id

		// Convert the metrics into protobuf

		// Generate an ephemeral key per session id

		// Send the protobuf message

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

func (c *Client) Start() error {
	c.startMainLoop()
	return nil
}

func (c *Client) stopMainLoop() {
	if c.mainLoopQuit != nil {
		close(c.mainLoopQuit)
		c.mainLoopQuit = nil
	}
}

func (c *Client) Stop() error {
	c.stopMainLoop()
	return nil
}

type FibonacciIntervalIncrementer struct {
	Last    int64
	Current int64
}

func (f *FibonacciIntervalIncrementer) Next() int64 {
	out := f.Last + f.Current

	f.Last = f.Current
	f.Current = out

	return out
}
