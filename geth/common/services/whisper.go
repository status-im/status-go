package services

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
)

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
