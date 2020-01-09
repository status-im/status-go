package transport

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/types"
)

type Transport interface {
	Track(identifiers [][]byte, hash []byte, newMessage *types.NewMessage)
	SendPublic(ctx context.Context, newMessage *types.NewMessage, chatName string) ([]byte, error)
	SendPrivateWithSharedSecret(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error)
	SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error)
}
