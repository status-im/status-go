package signal

import (
	"os"

	"github.com/status-im/status-go/pkg/messaging/diagnostics"
)

const EventMessageTrace = "message.trace"

func init() {
	if os.Getenv("STATUS_MESSAGE_TRACE") == "1" {
		diagnostics.Enable(func(event diagnostics.Event) {
			send(EventMessageTrace, event)
		})
	}
}
