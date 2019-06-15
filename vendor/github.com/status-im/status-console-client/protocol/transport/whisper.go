package transport

import (
	"context"

	"github.com/status-im/status-console-client/protocol/subscription"
	whisper "github.com/status-im/whisper/whisperv6"
)

// WhisperTransport defines an interface which each Whisper transport
// should conform to.
type WhisperTransport interface {
	KeysManager() *WhisperServiceKeysManager
	Subscribe(context.Context, chan<- *whisper.ReceivedMessage, *whisper.Filter) (*subscription.Subscription, error)
	Send(context.Context, whisper.NewMessage) ([]byte, error)
	Request(context.Context, RequestOptions) error
}
