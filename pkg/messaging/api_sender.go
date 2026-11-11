package messaging

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-go/pkg/messaging/layers/transport"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/waku-org/sds-go-bindings/sds"
)

func (a *API) SendPublic(ctx context.Context, params types.SendPublicParams) error {
	return a.core.controller.Sender().SendPublic(ctx, params)
}

// EnsureSDSChannel loads persisted state for a channel before periodic sync.
func (a *API) EnsureSDSChannel(channelID string) error {
	return a.core.stack.Reliability.EnsureSDSChannel(channelID)
}

// TickSDS returns sync payloads due for publication.
func (a *API) TickSDS() ([]sds.TickEvent, error) {
	return a.core.stack.Reliability.TickSDS()
}

// SendSDSSync sends a tick-generated metadata payload on the specified topic.
func (a *API) SendSDSSync(ctx context.Context, payload []byte, pubsubTopic, syncTopic string) error {
	return a.core.controller.Sender().SendSDSSync(ctx, payload, pubsubTopic, syncTopic)
}

func (a *API) SendPrivate(ctx context.Context, params types.SendPrivateParams) error {
	return a.core.controller.Sender().SendPrivate(ctx, params)
}

func (a *API) SendPrivateHashRatchetKeys(ctx context.Context, recipients []*ecdsa.PublicKey, groupID []byte) error {
	return a.core.controller.Sender().SendPrivateHashRatchetKeys(ctx, recipients, groupID)
}

func (a *API) GetEphemeralKey() (*ecdsa.PrivateKey, error) {
	return a.core.controller.Processor().GetEphemeralKey()
}

func ContactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return transport.ContactCodeTopic(publicKey)
}
