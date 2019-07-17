package whisper

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-protocol-go/transport/whisper/filter"
	whisper "github.com/status-im/whisper/whisperv6"
)

// WhisperTransport defines an interface which each Whisper transport
// should conform to.
type WhisperTransport interface {
	JoinPublic(string) error
	LeavePublic(string) error
	JoinPrivate(*ecdsa.PublicKey) error
	LeavePrivate(*ecdsa.PublicKey) error
	RetrievePublicMessages(string) ([]*whisper.ReceivedMessage, error)
	RetrievePrivateMessages(*ecdsa.PublicKey) ([]*whisper.ReceivedMessage, error)
	// LEGACY
	RetrieveAllRaw() (map[filter.Chat][]*whisper.ReceivedMessage, error)
	SendPublic(context.Context, whisper.NewMessage, string) ([]byte, error)
	SendPrivateWithSharedSecret(context.Context, whisper.NewMessage, *ecdsa.PublicKey, []byte) ([]byte, error)
	SendPrivateWithPartitioned(context.Context, whisper.NewMessage, *ecdsa.PublicKey) ([]byte, error)
	SendPrivateOnDiscovery(context.Context, whisper.NewMessage, *ecdsa.PublicKey) ([]byte, error)
	ProcessNegotiatedSecret(filter.NegotiatedSecret) error
	Request(context.Context, RequestOptions) error
}

type RequestOptions struct {
	Topics   []whisper.TopicType
	Password string
	Limit    int
	From     int64 // in seconds
	To       int64 // in seconds
}

const (
	defaultPowTime = 1
)

func DefaultWhisperMessage() whisper.NewMessage {
	msg := whisper.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = defaultPowTime

	return msg
}
