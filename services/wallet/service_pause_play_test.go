package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServicePauseNoopWhenNotStarted(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Pause())
}

func TestServicePauseNoopWhenAlreadyPaused(t *testing.T) {
	svc := &Service{
		started:          true,
		paused: true,
	}
	require.NoError(t, svc.Pause())
}

func TestServiceResumeNoopWhenNotStarted(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Resume())
}

func TestServiceResumeNoopWhenNotPaused(t *testing.T) {
	svc := &Service{
		started:          true,
		paused: false,
	}
	require.NoError(t, svc.Resume())
}
