package ext

import (
	"testing"

	"github.com/stretchr/testify/require"

	messaginglifecycle "github.com/status-im/status-go/pkg/messaging/lifecycle"
	"github.com/status-im/status-go/protocol"
)

func TestServicePauseResumeBackgroundWithNilMessenger(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.PauseBackground())
	require.NoError(t, svc.ResumeForeground())
}

func TestServicePauseResumeBackgroundUpdatesLifecycleState(t *testing.T) {
	messaginglifecycle.SetPausedBackground(false)
	defer messaginglifecycle.SetPausedBackground(false)

	svc := &Service{
		messenger: &protocol.Messenger{},
	}

	require.NoError(t, svc.PauseBackground())
	require.True(t, messaginglifecycle.IsPausedBackground())

	require.NoError(t, svc.ResumeForeground())
	require.False(t, messaginglifecycle.IsPausedBackground())
}
