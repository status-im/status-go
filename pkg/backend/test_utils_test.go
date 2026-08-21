package backend

import (
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/status-im/status-go/internal/platform"
	"github.com/status-im/status-go/params"
	networktestutil "github.com/status-im/status-go/pkg/services/networks/testutil"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

// makeTestNodeConfig defines a function to return a params.NodeConfig
// where specific network addresses are assigned based on provided network id.
func makeTestNodeConfig(t *testing.T) (*params.NodeConfig, error) {
	rootDataDir := t.TempDir()

	networkID := walletcommon.EthereumSepolia
	testDir := filepath.Join(rootDataDir, "StatusChain")

	if platform.OperatingSystemIs(platform.WindowsPlatform) {
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
		"NetworkId": ` + strconv.FormatUint(networkID, 10) + `,
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

	// Node startup always initializes TokenManager (even when WalletConfig.Enabled is false),
	// and TokenManager requires at least one active network.
	nodeConfig.Networks = networktestutil.MinimalActiveNetworks()

	return nodeConfig, nil
}
