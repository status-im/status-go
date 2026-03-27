package node

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/common"
)

// fakePausable is a minimal Pausable for testing.
type fakePausable struct {
	common.PauseBroadcaster
	name string
}

func newFakePausable(name string) *fakePausable {
	p := &fakePausable{name: name}
	p.MarkStarted()
	return p
}

func (p *fakePausable) PausableName() string { return p.name }

func (p *fakePausable) Pause() error {
	p.MarkPaused()
	return nil
}

func (p *fakePausable) Resume() error {
	p.MarkResumed()
	return nil
}

func TestServiceRegistry_RegisterAndList(t *testing.T) {
	r := newServiceRegistry()
	r.Register(newFakePausable("wallet"))
	r.Register(newFakePausable("messaging"))

	list := r.ListPausable()
	require.Len(t, list, 2)
	names := map[string]bool{}
	for _, info := range list {
		names[info.Name] = true
		require.Equal(t, common.ServiceStateRunning, info.State)
	}
	require.True(t, names["wallet"])
	require.True(t, names["messaging"])
}

func TestServiceRegistry_PauseAndResume(t *testing.T) {
	r := newServiceRegistry()
	p := newFakePausable("wallet")
	r.Register(p)

	require.NoError(t, r.Pause("wallet"))
	require.Equal(t, common.ServiceStatePaused, p.PausableState())

	require.NoError(t, r.Resume("wallet"))
	require.Equal(t, common.ServiceStateRunning, p.PausableState())
}

func TestServiceRegistry_PauseUnknownReturnsError(t *testing.T) {
	r := newServiceRegistry()
	err := r.Pause("nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrServiceNotFound)
	require.Contains(t, err.Error(), "nonexistent")
}

func TestServiceRegistry_ResumeUnknownReturnsError(t *testing.T) {
	r := newServiceRegistry()
	err := r.Resume("nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrServiceNotFound)
	require.Contains(t, err.Error(), "nonexistent")
}

func TestServiceRegistry_PauseMultiple(t *testing.T) {
	r := newServiceRegistry()
	wallet := newFakePausable("wallet")
	connector := newFakePausable("connector")
	r.Register(wallet)
	r.Register(connector)

	require.NoError(t, r.PauseMultiple([]string{"wallet", "connector"}))
	require.Equal(t, common.ServiceStatePaused, wallet.PausableState())
	require.Equal(t, common.ServiceStatePaused, connector.PausableState())
}

func TestServiceRegistry_PauseAll_ResumeAll(t *testing.T) {
	r := newServiceRegistry()
	wallet := newFakePausable("wallet")
	connector := newFakePausable("connector")
	newsfeed := newFakePausable("newsfeed")
	r.Register(wallet)
	r.Register(connector)
	r.Register(newsfeed)

	require.NoError(t, r.PauseAll())
	require.Equal(t, common.ServiceStatePaused, wallet.PausableState())
	require.Equal(t, common.ServiceStatePaused, connector.PausableState())
	require.Equal(t, common.ServiceStatePaused, newsfeed.PausableState())

	require.NoError(t, r.ResumeAll())
	require.Equal(t, common.ServiceStateRunning, wallet.PausableState())
	require.Equal(t, common.ServiceStateRunning, connector.PausableState())
	require.Equal(t, common.ServiceStateRunning, newsfeed.PausableState())
}

func TestServiceRegistry_PauseMultiple_PartialError(t *testing.T) {
	r := newServiceRegistry()
	r.Register(newFakePausable("wallet"))
	// "connector" is not registered

	err := r.PauseMultiple([]string{"wallet", "connector"})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrServiceNotFound)
	require.Contains(t, err.Error(), "connector")
}

// errorPausable returns an error on Pause.
type errorPausable struct {
	common.PauseBroadcaster
}

func (p *errorPausable) PausableName() string { return "broken" }
func (p *errorPausable) Pause() error         { return errors.New("pause failed") }
func (p *errorPausable) Resume() error        { return nil }

func TestServiceRegistry_PauseAll_CollectsErrors(t *testing.T) {
	r := newServiceRegistry()
	r.Register(&errorPausable{})
	r.Register(newFakePausable("wallet"))

	err := r.PauseAll()
	require.Error(t, err)
	require.Contains(t, err.Error(), "pause failed")
}
