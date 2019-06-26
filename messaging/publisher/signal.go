package publisher

import (
	"github.com/status-im/status-go/signal"
)

// SignalHandler sends signals on protocol events
type SignalHandler struct{}

func (h SignalHandler) DecryptMessageFailed(pubKey string) {
	signal.SendDecryptMessageFailed(pubKey)
}

func (h SignalHandler) BundleAdded(identity string, installationID string) {
	signal.SendBundleAdded(identity, installationID)
}

func (h SignalHandler) WhisperFilterAdded(filters []*signal.Filter) {
	signal.SendWhisperFilterAdded(filters)
}
