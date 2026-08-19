package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/pausable"
)

func TestPauseResumeBackgroundLifecycleHooks(t *testing.T) {
	p := newFakePausable("wallet")
	reg := newServiceRegistry()
	reg.Register(p)
	node := &StatusNode{
		serviceRegistry: reg,
	}

	require.NoError(t, node.Pause())
	require.Equal(t, pausable.ServiceStatePaused, p.PausableState())
	require.NoError(t, node.Resume())
	require.Equal(t, pausable.ServiceStateRunning, p.PausableState())
}
