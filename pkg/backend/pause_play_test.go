package backend

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/testutils"
)

func TestAppStateChangeStoppedNoop(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	b.lifecycleState = AppLifecycleStopped

	b.AppStateChange(AppStateBackground)
	require.Equal(t, AppLifecycleStopped, b.LifecycleState())

	b.AppStateChange(AppStateForeground)
	require.Equal(t, AppLifecycleStopped, b.LifecycleState())
}

func TestAppStateChangeBackgroundWithoutRunningNodeForcesStopped(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	b.lifecycleState = AppLifecycleRunning
	b.statusNode = nil

	b.AppStateChange(AppStateBackground)
	require.Equal(t, AppLifecycleStopped, b.LifecycleState())
}

func TestAppStateChangeForegroundWithoutRunningNodeForcesStopped(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	b.lifecycleState = AppLifecyclePausedBackground
	b.statusNode = nil

	b.AppStateChange(AppStateForeground)
	require.Equal(t, AppLifecycleStopped, b.LifecycleState())
}
