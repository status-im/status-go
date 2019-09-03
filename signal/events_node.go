package signal

const (
	// EventNodeStarted is triggered when underlying node is started
	EventNodeStarted = "node.started"

	// EventNodeReady is triggered when underlying node is fully ready
	// (consider backend to be fully registered)
	EventNodeReady = "node.ready"

	// EventNodeStopped is triggered when underlying node is fully stopped
	EventNodeStopped = "node.stopped"

	// EventNodeCrashed is triggered when node crashes
	EventNodeCrashed = "node.crashed"

	// EventChainDataRemoved is triggered when node's chain data is removed
	EventChainDataRemoved = "chaindata.removed"

	// EventLoggedIn is once node was injected with user account and ready to be used.
	EventLoggedIn = "node.login"
)

// NodeCrashEvent is special kind of error, used to report node crashes
type NodeCrashEvent struct {
	Error string `json:"error"`
}

// NodeLoginEvent returns the result of the login event
type NodeLoginEvent struct {
	Error string `json:"error,omitempty"`
}

// SendNodeCrashed emits a signal when status node has crashed, and
// provides error description.
func SendNodeCrashed(err error) {
	send(EventNodeCrashed,
		NodeCrashEvent{
			Error: err.Error(),
		})
}

// SendNodeStarted emits a signal when status node has just started (but not
// finished startup yet).
func SendNodeStarted() {
	send(EventNodeStarted, nil)
}

// SendNodeReady emits a signal when status node has started and successfully
// completed startup.
func SendNodeReady() {
	send(EventNodeReady, nil)
}

// SendNodeStopped emits a signal when underlying node has stopped.
func SendNodeStopped() {
	send(EventNodeStopped, nil)
}

// SendChainDataRemoved emits a signal when node's chain data has been removed.
func SendChainDataRemoved() {
	send(EventChainDataRemoved, nil)
}

func SendLoggedIn(err error) {
	event := NodeLoginEvent{}
	if err != nil {
		event.Error = err.Error()
	}
	send(EventLoggedIn, event)
}
