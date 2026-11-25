package backend

import (
	"sync"

	"github.com/status-im/status-go/internal/db/multiaccounts"
)

type LoginStateKind string

const (
	LoginStateNone       LoginStateKind = "none"
	LoginStateInProgress LoginStateKind = "inProgress"
	LoginStateLoggedIn   LoginStateKind = "loggedIn"
)

// LoginState is the login state: in progress from an accepted login request until its
// node.login outcome, whoever sent the request, so a client can wait for or adopt a
// login it didn't start. Account is set only when logged in.
type LoginState struct {
	State   LoginStateKind         `json:"state"`
	Account *multiaccounts.Account `json:"account,omitempty"`
}

type loginStateTracker struct {
	mu    sync.Mutex
	state LoginStateKind
}

func (t *loginStateTracker) set(state LoginStateKind) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = state
}

func (t *loginStateTracker) get() LoginStateKind {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == "" {
		return LoginStateNone
	}
	return t.state
}

// LoginRequestAccepted marks a login in progress. Login entry points call it as soon as
// they accept a request, before the login runs asynchronously.
func (b *StatusBackend) LoginRequestAccepted() {
	b.loginState.set(LoginStateInProgress)
}

func (b *StatusBackend) loginFinished(err error) {
	if err != nil {
		b.loginState.set(LoginStateNone)
		return
	}
	b.loginState.set(LoginStateLoggedIn)
}

func (b *StatusBackend) LoginState() LoginState {
	state := b.loginState.get()
	if state != LoginStateLoggedIn {
		return LoginState{State: state}
	}
	acc, err := b.GetActiveAccount()
	if err != nil {
		return LoginState{State: LoginStateNone}
	}
	return LoginState{State: state, Account: acc}
}
