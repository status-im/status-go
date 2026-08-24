package networks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetworksServiceAPIsArePublic(t *testing.T) {
	apis := NewService(&Manager{}).APIs()
	require.NotEmpty(t, apis)
	require.Equal(t, "networks", apis[0].Namespace)
	require.True(t, apis[0].Public, "networks RPC must be public like the wallet methods it replaces")
}

func TestServiceStartAndStopTolerateNilManager(t *testing.T) {
	t.Run("Start", func(t *testing.T) {
		s := NewService(nil)
		require.NotPanics(t, func() {
			require.NoError(t, s.Start())
		})
	})
	t.Run("Stop", func(t *testing.T) {
		s := NewService(nil)
		require.NotPanics(t, func() {
			require.NoError(t, s.Stop())
		})
	})
}
