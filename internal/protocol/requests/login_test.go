package requests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginUnmarshalLogFilePath(t *testing.T) {
	var request Login
	err := json.Unmarshal([]byte(`{
		"keyUid": "0x1234",
		"password": "pass",
		"logFilePath": "/data/logs"
	}`), &request)
	require.NoError(t, err)
	require.Equal(t, "/data/logs", request.LogFilePath)
	require.NoError(t, request.Validate())
}
