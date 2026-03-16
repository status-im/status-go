package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServicePauseBackgroundNoopWhenNotStarted(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.PauseBackground())
}

func TestServicePauseBackgroundNoopWhenAlreadyPaused(t *testing.T) {
	svc := &Service{
		started:          true,
		pausedBackground: true,
	}
	require.NoError(t, svc.PauseBackground())
}

func TestServiceResumeForegroundNoopWhenNotStarted(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.ResumeForeground())
}

func TestServiceResumeForegroundNoopWhenNotPaused(t *testing.T) {
	svc := &Service{
		started:          true,
		pausedBackground: false,
	}
	require.NoError(t, svc.ResumeForeground())
}
