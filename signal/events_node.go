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
)

// NodeCrashEvent is special kind of error, used to report node crashes
type NodeCrashEvent struct {
	Error string `json:"error"`
}

// SendNodeStarted emits a signal when status node has crashed, and
// provides error description.
func SendNodeCrashed(err error) {
	sendSignal(Envelope{
		Type: EventNodeCrashed,
		Event: NodeCrashEvent{
			Error: err.Error(),
		},
	})
}

// SendNodeStarted emits a signal when status node has just started (but not
// finished startup yet).
func SendNodeStarted() {
	sendSignal(Envelope{
		Type: EventNodeStarted,
	})
}

// SendNodeReady emits a signal when status node has started and successfully
// completed startup.
func SendNodeReady() {
	sendSignal(Envelope{
		Type: EventNodeReady,
	})
}

// SendNodeStopped emits a signal when underlying node has stopped.
func SendNodeStopped() {
	sendSignal(Envelope{
		Type: EventNodeStopped,
	})
}

// SendChainDataRemoved emits a signal when node's chain data has been removed.
func SendChainDataRemoved() {
	sendSignal(Envelope{
		Type: EventChainDataRemoved,
	})
}
