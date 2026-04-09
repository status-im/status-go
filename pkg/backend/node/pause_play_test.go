package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	common2 "github.com/status-im/status-go/common"
)

func TestPauseResumeBackgroundLifecycleHooks(t *testing.T) {
	p := newFakePausable("wallet")
	reg := newServiceRegistry()
	reg.Register(p)
	node := &StatusNode{
		serviceRegistry: reg,
	}

	require.NoError(t, node.Pause())
	require.Equal(t, common2.ServiceStatePaused, p.PausableState())
	require.NoError(t, node.Resume())
	require.Equal(t, common2.ServiceStateRunning, p.PausableState())
}
