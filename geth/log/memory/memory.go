package memory

import (
	"github.com/status-im/status-go/geth/log"
)

// Memory defines a struct which implements a memory collector for metricss.
type Memory struct {
	Data []log.Entry
}

// Emit adds the giving SentryJSON into the internal slice.
func (m *Memory) Emit(en log.Entry) error {
	m.Data = append(m.Data, en)
	return nil
}
