package messaging

import (
	"time"

	"github.com/status-im/status-go/pkg/messaging/types"
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
	return uint64(a.core.timeSource.Now().UnixNano() / int64(time.Millisecond))
}

// Online reports whether the transport currently has connectivity. It derives
// from the three-state ConnectionStatus (online == status != Disconnected),
// matching the logos-delivery Messaging API's online semantics.
func (a *API) Online() bool {
	return a.core.stack.Transport.ConnectionState().IsOnline()
}

// ConnectionStatus returns the transport's current three-state connection status
// (Disconnected / PartiallyConnected / Connected).
func (a *API) ConnectionStatus() types.ConnectionStatus {
	state := a.core.stack.Transport.ConnectionState()
	return types.ConnectionStatus{
		IsOnline: state.IsOnline(),
		State:    state,
	}
}

// SubscribeFilterMatched returns a channel that is notified whenever an incoming
// envelope matches at least one installed filter. bufSize should be 1.
// Callers must call UnsubscribeFilterMatched when done.
func (a *API) SubscribeFilterMatched() chan struct{} {
	if a.core.stack.Transport == nil {
		return nil
	}
	return a.core.stack.Transport.SubscribeFilterMatched()
}

func (a *API) UnsubscribeFilterMatched(ch chan struct{}) {
	if a.core.stack.Transport == nil || ch == nil {
		return
	}
	a.core.stack.Transport.UnsubscribeFilterMatched(ch)
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

// PauseDataSync idles (paused==true) or re-arms (paused==false) the reliability
// layer's data-sync node so its outbound loop performs no work while the host is
// backgrounded.
func (a *API) PauseDataSync(paused bool) error {
	if a.core.stack.Reliability == nil {
		return nil
	}
	return a.core.stack.Reliability.SetPaused(paused)
}
