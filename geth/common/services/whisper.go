package services

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
)

// Whisper interface for Whisper service.
type Whisper interface {
	MinPow() float64
	MaxMessageSize() uint32
	Overflow() bool
	APIs() []rpc.API
	RegisterServer(server whisperv5.MailServer)
	RegisterNotificationServer(server whisperv5.NotificationServer)
	Protocols() []p2p.Protocol
	Version() uint
	SetMaxMessageSize(size uint32) error
	SetMinimumPoW(val float64) error
	AllowP2PMessagesFromPeer(peerID []byte) error
	RequestHistoricMessages(peerID []byte, envelope *whisperv5.Envelope) error
	SendP2PMessage(peerID []byte, envelope *whisperv5.Envelope) error
	SendP2PDirect(peer *whisperv5.Peer, envelope *whisperv5.Envelope) error
	NewKeyPair() (string, error)
	AddKeyPair(key *ecdsa.PrivateKey) (string, error)
	SelectKeyPair(key *ecdsa.PrivateKey) error
	DeleteKeyPairs() error
	DeleteKeyPair(id string) bool
	HasKeyPair(id string) bool
	GetPrivateKey(id string) (*ecdsa.PrivateKey, error)
	GenerateSymKey() (string, error)
	AddSymKey(id string, key []byte) (string, error)
	AddSymKeyDirect(key []byte) (string, error)
	AddSymKeyFromPassword(password string) (string, error)
	HasSymKey(id string) bool
	DeleteSymKey(id string) bool
	GetSymKey(id string) ([]byte, error)
	Subscribe(f *whisperv5.Filter) (string, error)
	GetFilter(id string) *whisperv5.Filter
	Unsubscribe(id string) error
	Send(envelope *whisperv5.Envelope) error
	Start(stack *p2p.Server) error
	Stop() error
	HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error
	Stats() whisperv5.Statistics
	Envelopes() []*whisperv5.Envelope
	Messages(id string) []*whisperv5.ReceivedMessage
}

type WhisperAPI interface {
	Version(ctx context.Context) string
	Info(ctx context.Context) whisperv5.Info
	SetMaxMessageSize(ctx context.Context, size uint32) (bool, error)
	SetMinPoW(ctx context.Context, pow float64) (bool, error)
	MarkTrustedPeer(ctx context.Context, enode string) (bool, error)
	NewKeyPair(ctx context.Context) (string, error)
	AddPrivateKey(ctx context.Context, privateKey hexutil.Bytes) (string, error)
	DeleteKeyPair(ctx context.Context, key string) (bool, error)
	HasKeyPair(ctx context.Context, id string) bool
	GetPublicKey(ctx context.Context, id string) (hexutil.Bytes, error)
	GetPrivateKey(ctx context.Context, id string) (hexutil.Bytes, error)
	NewSymKey(ctx context.Context) (string, error)
	AddSymKey(ctx context.Context, key hexutil.Bytes) (string, error)
	GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error)
	HasSymKey(ctx context.Context, id string) bool
	GetSymKey(ctx context.Context, id string) (hexutil.Bytes, error)
	DeleteSymKey(ctx context.Context, id string) bool
	Post(ctx context.Context, req whisperv5.NewMessage) (bool, error)
	UninstallFilter(id string)
	Unsubscribe(id string)
	Messages(ctx context.Context, crit whisperv5.Criteria) (*rpc.Subscription, error)
	GetFilterMessages(id string) ([]*whisperv5.Message, error)
	DeleteMessageFilter(id string) (bool, error)
	NewMessageFilter(req whisperv5.Criteria) (string, error)
}
