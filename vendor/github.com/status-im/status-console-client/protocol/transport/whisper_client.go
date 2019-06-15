package transport

import (
	"context"

	"github.com/status-im/status-console-client/protocol/subscription"
	whisper "github.com/status-im/whisper/whisperv6"
)

type WhisperClientTransport struct{}

var _ WhisperTransport = (*WhisperClientTransport)(nil)

func (w *WhisperClientTransport) KeysManager() *WhisperServiceKeysManager { return nil }

func (w *WhisperClientTransport) Subscribe(context.Context, chan<- *whisper.ReceivedMessage, *whisper.Filter) (*subscription.Subscription, error) {
	return nil, nil
}

func (w *WhisperClientTransport) Send(context.Context, whisper.NewMessage) ([]byte, error) {
	return nil, nil
}

func (w *WhisperClientTransport) Request(context.Context, RequestOptions) error { return nil }
