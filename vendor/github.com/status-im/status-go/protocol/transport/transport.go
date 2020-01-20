package transport

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/types"
)

type Transport interface {
	Stop() error

	JoinPrivate(publicKey *ecdsa.PublicKey) error
	LeavePrivate(publicKey *ecdsa.PublicKey) error
	JoinGroup(publicKeys []*ecdsa.PublicKey) error
	LeaveGroup(publicKeys []*ecdsa.PublicKey) error
	JoinPublic(chatID string) error
	LeavePublic(chatID string) error
	GetCurrentTime() uint64

	SendPublic(ctx context.Context, newMessage *types.NewMessage, chatName string) ([]byte, error)
	SendPrivateWithSharedSecret(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error)
	SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error)
	SendMessagesRequest(
		ctx context.Context,
		peerID []byte,
		from, to uint32,
		previousCursor []byte,
	) (cursor []byte, err error)

	Track(identifiers [][]byte, hash []byte, newMessage *types.NewMessage)

	InitFilters(chatIDs []string, publicKeys []*ecdsa.PublicKey) ([]*Filter, error)
	LoadFilters(filters []*Filter) ([]*Filter, error)
	RemoveFilters(filters []*Filter) error
	ResetFilters() error
	Filters() []*Filter
	ProcessNegotiatedSecret(secret types.NegotiatedSecret) (*Filter, error)
	RetrieveRawAll() (map[Filter][]*types.Message, error)
}
