package gethbridge

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
)

type gethPublicWakuAPIWrapper struct {
	api *waku.PublicWakuAPI
}

// NewGethPublicWakuAPIWrapper returns an object that wraps Geth's PublicWakuAPI in a types interface
func NewGethPublicWakuAPIWrapper(api *waku.PublicWakuAPI) types.PublicWakuAPI {
	if api == nil {
		panic("PublicWakuAPI cannot be nil")
	}

	return &gethPublicWakuAPIWrapper{
		api: api,
	}
}

// AddPrivateKey imports the given private key.
func (w *gethPublicWakuAPIWrapper) AddPrivateKey(ctx context.Context, privateKey types.HexBytes) (string, error) {
	return w.api.AddPrivateKey(ctx, hexutil.Bytes(privateKey))
}

// GenerateSymKeyFromPassword derives a key from the given password, stores it, and returns its ID.
func (w *gethPublicWakuAPIWrapper) GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error) {
	return w.api.GenerateSymKeyFromPassword(ctx, passwd)
}

// DeleteKeyPair removes the key with the given key if it exists.
func (w *gethPublicWakuAPIWrapper) DeleteKeyPair(ctx context.Context, key string) (bool, error) {
	return w.api.DeleteKeyPair(ctx, key)
}

// NewMessageFilter creates a new filter that can be used to poll for
// (new) messages that satisfy the given criteria.
func (w *gethPublicWakuAPIWrapper) NewMessageFilter(req types.Criteria) (string, error) {
	topics := make([]wakucommon.TopicType, len(req.Topics))
	for index, tt := range req.Topics {
		topics[index] = wakucommon.TopicType(tt)
	}

	criteria := waku.Criteria{
		SymKeyID:     req.SymKeyID,
		PrivateKeyID: req.PrivateKeyID,
		Sig:          req.Sig,
		MinPow:       req.MinPow,
		Topics:       topics,
		AllowP2P:     req.AllowP2P,
	}
	return w.api.NewMessageFilter(criteria)
}

// GetFilterMessages returns the messages that match the filter criteria and
// are received between the last poll and now.
func (w *gethPublicWakuAPIWrapper) GetFilterMessages(id string) ([]*types.Message, error) {
	msgs, err := w.api.GetFilterMessages(id)
	if err != nil {
		return nil, err
	}

	wrappedMsgs := make([]*types.Message, len(msgs))
	for index, msg := range msgs {
		wrappedMsgs[index] = &types.Message{
			Sig:       msg.Sig,
			TTL:       msg.TTL,
			Timestamp: msg.Timestamp,
			Topic:     types.TopicType(msg.Topic),
			Payload:   msg.Payload,
			Padding:   msg.Padding,
			PoW:       msg.PoW,
			Hash:      msg.Hash,
			Dst:       msg.Dst,
			P2P:       msg.P2P,
		}
	}
	return wrappedMsgs, nil
}

// Post posts a message on the network.
// returns the hash of the message in case of success.
func (w *gethPublicWakuAPIWrapper) Post(ctx context.Context, req types.NewMessage) ([]byte, error) {
	msg := waku.NewMessage{
		SymKeyID:   req.SymKeyID,
		PublicKey:  req.PublicKey,
		Sig:        req.SigID, // Sig is really a SigID
		TTL:        req.TTL,
		Topic:      wakucommon.TopicType(req.Topic),
		Payload:    req.Payload,
		Padding:    req.Padding,
		PowTime:    req.PowTime,
		PowTarget:  req.PowTarget,
		TargetPeer: req.TargetPeer,
	}
	return w.api.Post(ctx, msg)
}
