package backend

import (
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
)

// makeTestNodeConfig defines a function to return a params.NodeConfig
// where specific network addresses are assigned based on provided network id.
func makeTestNodeConfig(t *testing.T) (*params.NodeConfig, error) {
	rootDataDir := t.TempDir()

	networkID := params.StatusChainNetworkID
	testDir := filepath.Join(rootDataDir, "StatusChain")

	if common.OperatingSystemIs(common.WindowsPlatform) {
		testDir = filepath.ToSlash(testDir)
	}

	// run tests with "INFO" log level only
	// when `go test` invoked with `-v` flag
	errorLevel := "ERROR"
	if testing.Verbose() {
		errorLevel = "INFO"
	}

	configJSON := `{
		"Name": "test",
		"NetworkId": ` + strconv.Itoa(networkID) + `,
		"RootDataDir": "` + testDir + `",
		"KeycardPairingDataFile": "` + path.Join(testDir, "keycard/pairings.json") + `",
		"HTTPPort": 8645,
		"WSPort": 8646,
		"LogLevel": "` + errorLevel + `",
		"NoDiscovery": true,
		"LightEthConfig": {
			"Enabled": true
		}
	}`

	nodeConfig, err := params.NewConfigFromJSON(configJSON)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}
