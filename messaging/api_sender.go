package messaging

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/layers/transport"
	"github.com/status-im/status-go/messaging/types"
)

func (a *API) SendPublic(ctx context.Context, params types.SendPublicParams) error {
	return a.core.controller.Sender().SendPublic(ctx, params)
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
