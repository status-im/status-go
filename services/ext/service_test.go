package ext

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/protocol"
)

func TestServicePauseResumeBackgroundWithNilMessenger(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Pause())
	require.Equal(t, common.ServiceStatePaused, svc.PausableState())
	require.NoError(t, svc.Resume())
	require.Equal(t, common.ServiceStateRunning, svc.PausableState())
}

func TestServicePauseResumeBackgroundWithMessenger(t *testing.T) {
	svc := &Service{
		messenger: &protocol.Messenger{},
	}
	require.NoError(t, svc.Pause())
	require.Equal(t, common.ServiceStatePaused, svc.PausableState())
	require.NoError(t, svc.Resume())
	require.Equal(t, common.ServiceStateRunning, svc.PausableState())
}
