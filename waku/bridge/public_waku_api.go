package bridge

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/wakuv1"

	wakutypes "github.com/status-im/status-go/waku/types"
)

type GethPublicWakuAPIWrapper struct {
	api *wakuv1.PublicWakuAPI
}

// NewGethPublicWakuAPIWrapper returns an object that wraps Geth's PublicWakuAPI in a types interface
func NewGethPublicWakuAPIWrapper(api *wakuv1.PublicWakuAPI) wakutypes.PublicWakuAPI {
	if api == nil {
		panic("PublicWakuAPI cannot be nil")
	}

	return &GethPublicWakuAPIWrapper{
		api: api,
	}
}

// AddPrivateKey imports the given private key.
func (w *GethPublicWakuAPIWrapper) AddPrivateKey(ctx context.Context, privateKey types.HexBytes) (string, error) {
	return w.api.AddPrivateKey(ctx, hexutil.Bytes(privateKey))
}

// GenerateSymKeyFromPassword derives a key from the given password, stores it, and returns its ID.
func (w *GethPublicWakuAPIWrapper) GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error) {
	return w.api.GenerateSymKeyFromPassword(ctx, passwd)
}

// DeleteKeyPair removes the key with the given key if it exists.
func (w *GethPublicWakuAPIWrapper) DeleteKeyPair(ctx context.Context, key string) (bool, error) {
	return w.api.DeleteKeyPair(ctx, key)
}

// NewMessageFilter creates a new filter that can be used to poll for
// (new) messages that satisfy the given criteria.
func (w *GethPublicWakuAPIWrapper) NewMessageFilter(req wakutypes.Criteria) (string, error) {
	return w.api.NewMessageFilter(req)
}

func (w *GethPublicWakuAPIWrapper) BloomFilter() []byte {
	return w.api.BloomFilter()
}

// GetFilterMessages returns the messages that match the filter criteria and
// are received between the last poll and now.
func (w *GethPublicWakuAPIWrapper) GetFilterMessages(id string) ([]*wakutypes.Message, error) {
	return w.api.GetFilterMessages(id)
}

// Post posts a message on the network.
// returns the hash of the message in case of success.
func (w *GethPublicWakuAPIWrapper) Post(ctx context.Context, req wakutypes.NewMessage) ([]byte, error) {
	return w.api.Post(ctx, req)
}
