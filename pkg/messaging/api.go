package messaging

import (
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

// PauseTransport signals the transport and its sub-components to idle their goroutines.
func (a *API) PauseTransport() {
	if a.core.stack.Transport != nil {
		a.core.stack.Transport.Pause()
	}
}

// ResumeTransport signals the transport and its sub-components to resume their goroutines.
func (a *API) ResumeTransport() {
	if a.core.stack.Transport != nil {
		a.core.stack.Transport.Resume()
	}
}
