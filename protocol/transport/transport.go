package transport

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

type Transport interface {
	Stop() error

	JoinPrivate(publicKey *ecdsa.PublicKey) (*Filter, error)
	LeavePrivate(publicKey *ecdsa.PublicKey) error
	JoinGroup(publicKeys []*ecdsa.PublicKey) ([]*Filter, error)
	LeaveGroup(publicKeys []*ecdsa.PublicKey) error
	JoinPublic(chatID string) (*Filter, error)
	LeavePublic(chatID string) error
	GetCurrentTime() uint64
	MaxMessageSize() uint32

	SendPublic(ctx context.Context, newMessage *types.NewMessage, chatName string) ([]byte, error)
	SendPrivateWithSharedSecret(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error)
	SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error)
	SendPrivateOnPersonalTopic(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error)
	SendCommunityMessage(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error)
	SendMessagesRequest(
		ctx context.Context,
		peerID []byte,
		from, to uint32,
		previousCursor []byte,
		waitForResponse bool,
	) (cursor []byte, err error)

	SendMessagesRequestForFilter(
		ctx context.Context,
		peerID []byte,
		from, to uint32,
		previousCursor []byte,
		filter *Filter,
		waitForResponse bool,
	) (cursor []byte, err error)
	FilterByChatID(string) *Filter

	Track(identifiers [][]byte, hash []byte, newMessage *types.NewMessage)

	InitFilters(chatIDs []string, publicKeys []*ecdsa.PublicKey) ([]*Filter, error)
	InitPublicFilters(chatIDs []string) ([]*Filter, error)
	InitCommunityFilters(pks []*ecdsa.PrivateKey) ([]*Filter, error)
	LoadFilters(filters []*Filter) ([]*Filter, error)
	RemoveFilters(filters []*Filter) error
	RemoveFilterByChatID(string) (*Filter, error)
	ResetFilters() error
	Filters() []*Filter
	LoadKeyFilters(*ecdsa.PrivateKey) (*Filter, error)
	ProcessNegotiatedSecret(secret types.NegotiatedSecret) (*Filter, error)
	RetrieveRawAll() (map[Filter][]*types.Message, error)

	ConfirmMessagesProcessed([]string, uint64) error
	CleanMessagesProcessed(uint64) error

	SetEnvelopeEventsHandler(handler EnvelopeEventsHandler) error
}

func PubkeyToHex(key *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(key))
}
