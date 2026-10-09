//go:build logos_delivery

package delivery

import (
	"context"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"

	"github.com/status-im/status-go/internal/panics"
)

// Available reports whether this build links liblogosdelivery.
const Available = true

// bindingsClient adapts the bindings' MessagingClient to Client, translating
// its events into this package's.
type bindingsClient struct {
	mc     *messaging.MessagingClient
	events chan Event
}

// NewClient creates a node from cfg. It is not started.
func NewClient(cfg Config) (Client, error) {
	mc, err := messaging.New(bindingsConfig(cfg))
	if err != nil {
		return nil, err
	}
	c := &bindingsClient{mc: mc, events: make(chan Event, cap(mc.Events()))}
	go c.forward()
	return c, nil
}

func bindingsConfig(cfg Config) messaging.Config {
	overrides := messaging.Overrides{}
	if cfg.NodeKey != nil {
		overrides["nodekey"] = hex.EncodeToString(crypto.FromECDSA(cfg.NodeKey))
	}
	if cfg.TCPPort != 0 {
		overrides["tcp-port"] = cfg.TCPPort
	}
	if cfg.UDPPort != 0 {
		overrides["discv5-udp-port"] = cfg.UDPPort
	}
	if cfg.DataDir != "" {
		overrides["local-storage-path"] = cfg.DataDir
	}
	for k, v := range cfg.Overrides {
		overrides[k] = v
	}
	if len(overrides) == 0 {
		overrides = nil
	}
	return messaging.Config{
		Mode:               messaging.Mode(cfg.Mode),
		Preset:             cfg.Preset,
		MessagingOverrides: overrides,
	}
}

func (c *bindingsClient) forward() {
	defer panics.LogOnPanic()
	defer close(c.events)
	for ev := range c.mc.Events() {
		if e := translate(ev); e != nil {
			c.events <- e
		}
	}
}

func translate(ev messaging.Event) Event {
	switch e := ev.(type) {
	case messaging.MessageReceivedEvent:
		return ReceivedEvent{
			MessageHash:  e.MessageHash,
			ContentTopic: e.Message.ContentTopic,
			Payload:      e.Message.Payload,
			Meta:         e.Message.Meta,
			Version:      e.Message.Version,
			Timestamp:    e.Message.Timestamp,
			Ephemeral:    e.Message.Ephemeral,
		}
	case messaging.MessageSentEvent:
		return SentEvent{RequestID: string(e.RequestID), MessageHash: e.MessageHash}
	case messaging.MessagePropagatedEvent:
		return PropagatedEvent{RequestID: string(e.RequestID), MessageHash: e.MessageHash}
	case messaging.MessageErrorEvent:
		return ErrorEvent{RequestID: string(e.RequestID), MessageHash: e.MessageHash, Err: e.Err}
	case messaging.ConnectionStatusEvent:
		return ConnectionEvent{Status: ConnectionStatus(e.Status)}
	}
	return nil
}

func (c *bindingsClient) Start() error         { return c.mc.Start() }
func (c *bindingsClient) Stop() error          { return c.mc.Stop() }
func (c *bindingsClient) Close() error         { return c.mc.Close() }
func (c *bindingsClient) Events() <-chan Event { return c.events }

func (c *bindingsClient) Subscribe(contentTopic string) error {
	return c.mc.Subscribe(contentTopic)
}

func (c *bindingsClient) Unsubscribe(contentTopic string) error {
	return c.mc.Unsubscribe(contentTopic)
}

func (c *bindingsClient) Send(ctx context.Context, contentTopic string, payload []byte, ephemeral bool) (string, error) {
	id, err := c.mc.Send(ctx, contentTopic, payload, ephemeral)
	return string(id), err
}
