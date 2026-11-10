package signal

import (
	"encoding/json"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
)

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

	// EventLoggedIn is once node was injected with user account and ready to be used.
	EventLoggedIn = "node.login"
)

// NodeCrashEvent is special kind of error, used to report node crashes
type NodeCrashEvent struct {
	Error string `json:"error"`
}

// NodeLoginEvent returns the result of the login event
type NodeLoginEvent struct {
	Error        string                 `json:"error,omitempty"`
	Settings     *settings.Settings     `json:"settings,omitempty"`
	Account      *multiaccounts.Account `json:"account,omitempty"`
	EnsUsernames json.RawMessage        `json:"ensUsernames,omitempty"`
}

// SendNodeCrashed emits a signal when status node has crashed, and
// provides error description.
func SendNodeCrashed(err error) {
	CurrentSession().SendNodeCrashed(err)
}

// SendNodeCrashed sends in the session s.
func (s Session) SendNodeCrashed(err error) {
	s.send(EventNodeCrashed,
		NodeCrashEvent{
			Error: err.Error(),
		})
}

// SendNodeStarted emits a signal when status node has just started (but not
// finished startup yet).
func SendNodeStarted() {
	CurrentSession().SendNodeStarted()
}

// SendNodeStarted sends in the session s.
func (s Session) SendNodeStarted() {
	s.send(EventNodeStarted, nil)
}

// SendNodeReady emits a signal when status node has started and successfully
// completed startup.
func SendNodeReady() {
	CurrentSession().SendNodeReady()
}

// SendNodeReady sends in the session s.
func (s Session) SendNodeReady() {
	s.send(EventNodeReady, nil)
}

// SendNodeStopped emits a signal when underlying node has stopped.
func SendNodeStopped() {
	CurrentSession().SendNodeStopped()
}

// SendNodeStopped sends in the session s.
func (s Session) SendNodeStopped() {
	s.send(EventNodeStopped, nil)
}

func SendLoggedIn(account *multiaccounts.Account, settings *settings.Settings, ensUsernames json.RawMessage, err error) {
	CurrentSession().SendLoggedIn(account, settings, ensUsernames, err)
}

// SendLoggedIn sends in the session s.
func (s Session) SendLoggedIn(account *multiaccounts.Account, settings *settings.Settings, ensUsernames json.RawMessage, err error) {
	event := NodeLoginEvent{Settings: settings, Account: account, EnsUsernames: ensUsernames}
	if err != nil {
		event.Error = err.Error()
	}
	s.send(EventLoggedIn, event)
}
