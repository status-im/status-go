package gethbridge

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

type gethPublicWhisperAPIWrapper struct {
	publicWhisperAPI *whisper.PublicWhisperAPI
}

// NewGethPublicWhisperAPIWrapper returns an object that wraps Geth's PublicWhisperAPI in a whispertypes interface
func NewGethPublicWhisperAPIWrapper(publicWhisperAPI *whisper.PublicWhisperAPI) whispertypes.PublicWhisperAPI {
	if publicWhisperAPI == nil {
		panic("publicWhisperAPI cannot be nil")
	}

	return &gethPublicWhisperAPIWrapper{
		publicWhisperAPI: publicWhisperAPI,
	}
}

// AddPrivateKey imports the given private key.
func (w *gethPublicWhisperAPIWrapper) AddPrivateKey(ctx context.Context, privateKey protocol.HexBytes) (string, error) {
	return w.publicWhisperAPI.AddPrivateKey(ctx, hexutil.Bytes(privateKey))
}

// GenerateSymKeyFromPassword derives a key from the given password, stores it, and returns its ID.
func (w *gethPublicWhisperAPIWrapper) GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error) {
	return w.publicWhisperAPI.GenerateSymKeyFromPassword(ctx, passwd)
}

// DeleteKeyPair removes the key with the given key if it exists.
func (w *gethPublicWhisperAPIWrapper) DeleteKeyPair(ctx context.Context, key string) (bool, error) {
	return w.publicWhisperAPI.DeleteKeyPair(ctx, key)
}

// NewMessageFilter creates a new filter that can be used to poll for
// (new) messages that satisfy the given criteria.
func (w *gethPublicWhisperAPIWrapper) NewMessageFilter(req whispertypes.Criteria) (string, error) {
	topics := make([]whisper.TopicType, len(req.Topics))
	for index, tt := range req.Topics {
		topics[index] = whisper.TopicType(tt)
	}

	criteria := whisper.Criteria{
		SymKeyID:     req.SymKeyID,
		PrivateKeyID: req.PrivateKeyID,
		Sig:          req.Sig,
		MinPow:       req.MinPow,
		Topics:       topics,
		AllowP2P:     req.AllowP2P,
	}
	return w.publicWhisperAPI.NewMessageFilter(criteria)
}

// GetFilterMessages returns the messages that match the filter criteria and
// are received between the last poll and now.
func (w *gethPublicWhisperAPIWrapper) GetFilterMessages(id string) ([]*whispertypes.Message, error) {
	msgs, err := w.publicWhisperAPI.GetFilterMessages(id)
	if err != nil {
		return nil, err
	}

	wrappedMsgs := make([]*whispertypes.Message, len(msgs))
	for index, msg := range msgs {
		wrappedMsgs[index] = &whispertypes.Message{
			Sig:       msg.Sig,
			TTL:       msg.TTL,
			Timestamp: msg.Timestamp,
			Topic:     whispertypes.TopicType(msg.Topic),
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

// Post posts a message on the Whisper network.
// returns the hash of the message in case of success.
func (w *gethPublicWhisperAPIWrapper) Post(ctx context.Context, req whispertypes.NewMessage) ([]byte, error) {
	msg := whisper.NewMessage{
		SymKeyID:   req.SymKeyID,
		PublicKey:  req.PublicKey,
		Sig:        req.SigID, // Sig is really a SigID
		TTL:        req.TTL,
		Topic:      whisper.TopicType(req.Topic),
		Payload:    req.Payload,
		Padding:    req.Padding,
		PowTime:    req.PowTime,
		PowTarget:  req.PowTarget,
		TargetPeer: req.TargetPeer,
	}
	return w.publicWhisperAPI.Post(ctx, msg)
}
