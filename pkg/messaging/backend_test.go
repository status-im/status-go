package messaging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBackend(t *testing.T) {
	for name, want := range map[string]Backend{
		"":                       BackendDefault,
		BackendNameGoWaku:        BackendGoWaku,
		BackendNameLogosDelivery: BackendLogosDelivery,
	} {
		backend, err := ParseBackend(name)
		require.NoError(t, err, name)
		require.Equal(t, want, backend, name)
	}

	_, err := ParseBackend("waku")
	require.Error(t, err)
}

func TestBackendResolve(t *testing.T) {
	require.Equal(t, DefaultBackend, BackendDefault.Resolve())
	require.Equal(t, BackendGoWaku, BackendGoWaku.Resolve())
	require.Equal(t, BackendLogosDelivery, BackendLogosDelivery.Resolve())
	require.Equal(t, BackendNameLogosDelivery, BackendLogosDelivery.String())
	require.Equal(t, BackendNameGoWaku, BackendGoWaku.String())
}
