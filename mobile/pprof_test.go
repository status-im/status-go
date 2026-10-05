package statusgo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireNoAPIError(t *testing.T, response string) {
	t.Helper()
	var r APIResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))
	require.Empty(t, r.Error)
}

func TestStartStopPprof(t *testing.T) {
	requireNoAPIError(t, StartPprof("127.0.0.1:0"))
	requireNoAPIError(t, StopPprof())
}
