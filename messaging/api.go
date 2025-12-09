package messaging

import (
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/pkg/pubsub"
)

type API struct {
	core *Core
}

func NewAPI(core *Core) *API {
	return &API{
		core: core,
	}
}

func (a *API) Start() error {
	return a.core.start()
}

func (a *API) Stop() error {
	return a.core.stop()
}

func (a *API) Publisher() *pubsub.Publisher {
	return a.core.publisher
}

// GetCurrentTime satisfies the common.TimeSource interface.
func (a *API) GetCurrentTime() uint64 {
	return a.core.stack.Transport.GetCurrentTime()
}

func (a *API) ConnectionChanged(state connection.State) {
	a.core.connectionChanged(state)
}
