package backend

import (
	"errors"
	"sync"

	"github.com/status-im/status-go/internal/db/multiaccounts"
)

type LoginStateKind string

const (
	LoginStateNone       LoginStateKind = "none"
	LoginStateInProgress LoginStateKind = "inProgress"
	LoginStateLoggedIn   LoginStateKind = "loggedIn"
)

var (
	ErrLoginInProgress = errors.New("a login is already in progress")
	ErrAlreadyLoggedIn = errors.New("already logged in")
)

// LoginState is the login state: in progress from an accepted login request until its
// node.login outcome, whoever sent the request, so a client can wait for or adopt a
// login it didn't start. Seq is the login sequence number of the current or last login,
// also stamped on its node.login, so a client can tell which login a node.login belongs
// to. Account is set only when logged in.
type LoginState struct {
	State   LoginStateKind         `json:"state"`
	Seq     uint64                 `json:"seq"`
	Account *multiaccounts.Account `json:"account,omitempty"`
}

type loginStateTracker struct {
	mu    sync.Mutex
	state LoginStateKind
	seq   uint64
}

func (t *loginStateTracker) accept() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch t.state {
	case LoginStateInProgress:
		return ErrLoginInProgress
	case LoginStateLoggedIn:
		return ErrAlreadyLoggedIn
	}
	t.seq++
	t.state = LoginStateInProgress
	return nil
}

// begin is accept for the backend entry points, which also run inside an accepted
// request: a login already in progress keeps its seq.
func (t *loginStateTracker) begin() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == LoginStateInProgress {
		return
	}
	t.seq++
	t.state = LoginStateInProgress
}

func (t *loginStateTracker) set(state LoginStateKind) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = state
	return t.seq
}

func (t *loginStateTracker) get() LoginStateKind {
	state, _ := t.snapshot()
	return state
}

func (t *loginStateTracker) snapshot() (LoginStateKind, uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == "" {
		return LoginStateNone, t.seq
	}
	return t.state, t.seq
}

// AcceptLoginRequest marks a login in progress with the next login sequence number.
// Login API entry points call it before the login runs asynchronously. It rejects the
// request, changing nothing, while a login is in progress or a session is logged in.
func (b *StatusBackend) AcceptLoginRequest() error {
	return b.loginState.accept()
}

func (b *StatusBackend) loginStarted() {
	b.loginState.begin()
}

// loginFinished sets the outcome and returns the seq of the login it belongs to.
func (b *StatusBackend) loginFinished(err error) uint64 {
	if err != nil {
		return b.loginState.set(LoginStateNone)
	}
	return b.loginState.set(LoginStateLoggedIn)
}

func (b *StatusBackend) LoginState() LoginState {
	state, seq := b.loginState.snapshot()
	if state != LoginStateLoggedIn {
		return LoginState{State: state, Seq: seq}
	}
	acc, err := b.GetActiveAccount()
	if err != nil {
		return LoginState{State: LoginStateNone, Seq: seq}
	}
	return LoginState{State: state, Seq: seq, Account: acc}
}
