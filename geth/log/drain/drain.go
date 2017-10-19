package drain

import (
	"github.com/status-im/status-go/geth/log"
)

// Drain emits all entries into nothingness.
type Drain struct{}

// Emit implements the log.metrics interface and does nothing with the
// provided entry.
func (Drain) Emit(e log.Entry) error {
	return nil
}
