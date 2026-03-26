package ext

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol"
)

func TestServicePauseResumeBackgroundWithNilMessenger(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Pause())
	require.NoError(t, svc.Resume())
}

func TestServicePauseResumeBackgroundWithMessenger(t *testing.T) {
	svc := &Service{
		messenger: &protocol.Messenger{},
	}
	require.NoError(t, svc.Pause())
	require.NoError(t, svc.Resume())
}
