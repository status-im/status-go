package localnotifications

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceStartStop(t *testing.T) {
	db, stop := setupAppTestDb(t)
	defer stop()

	s, err := NewService(db)
	require.NoError(t, err)
	require.NoError(t, s.Start())
	require.Equal(t, true, s.IsStarted())

	require.NoError(t, s.Stop())
	require.Equal(t, false, s.IsStarted())
}
