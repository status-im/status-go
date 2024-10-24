package statusgo

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/logutils/requestlog"
	"github.com/status-im/status-go/protocol/requests"
)

func TestInitLogging(t *testing.T) {
	tempDir := t.TempDir()
	t.Logf("temp dir: %s", tempDir)
	gethLogFile := path.Join(tempDir, "geth.log")
	requestsLogFile := path.Join(tempDir, "requests.log")
	logSettings := fmt.Sprintf(`{"LogRequestGo": true, "LogRequestFile": "%s", "File": "%s", "Level": "INFO", "Enabled": true, "MobileSystem": false}`, requestsLogFile, gethLogFile)
	response := InitLogging(logSettings)
	require.Equal(t, `{"error":""}`, response)
	_, err := os.Stat(gethLogFile)
	require.NoError(t, err)
	require.NotNil(t, requestlog.GetRequestLogger())

	// requests log file should not be created yet
	_, err = os.Stat(requestsLogFile)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	createAccountRequest := &requests.CreateAccount{
		DisplayName:        "some-display-name",
		CustomizationColor: "#ffffff",
		Password:           "some-password",
		RootDataDir:        tempDir,
		LogFilePath:        gethLogFile,
	}
	_, err = statusBackend.CreateAccountAndLogin(createAccountRequest)
	require.NoError(t, err)
	result := CallPrivateRPC(`{"jsonrpc":"2.0","method":"settings_getSettings","params":[],"id":1}`)
	require.NotContains(t, result, "error")
	// Check if request log file exists now
	_, err = os.Stat(requestsLogFile)
	require.NoError(t, err)
	require.FileExists(t, requestsLogFile)
}
