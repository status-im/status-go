package params_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/go-playground/validator.v9"

	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/require"
)

var loadConfigTestCases = []struct {
	name           string
	useMainnetFlag bool // nolint
	configJSON     string
	validator      func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error)
}{
	{
		`invalid input JSON (missing comma at the end of key:value pair)`,
		false,
		`{
			"NetworkId": 3
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.Error(t, err, "error is expected, not thrown")
		},
	},
	{
		`check static DataDir passing`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "/storage/emulated/0/ethereum/"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.Equal(t, "/storage/emulated/0/ethereum/", nodeConfig.DataDir)
		},
	},
	{
		`use default KeyStoreDir`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)

			_, err = os.Stat(dataDir)
			require.False(t, os.IsNotExist(err), "data directory doesn't exist")
			require.Equal(t, dataDir, nodeConfig.DataDir)

			require.Equal(t, filepath.Join(dataDir, params.KeyStoreDir), filepath.Join(dataDir, params.KeyStoreDir))
		},
	},
	{
		`use non-default KeyStoreDir`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"KeyStoreDir": "/foo/bar"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.Equal(t, dataDir, nodeConfig.DataDir)
			require.Equal(t, "/foo/bar", nodeConfig.KeyStoreDir)
		},
	},
	{
		`test Upstream config setting`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"Name": "TestStatusNode",
			"WSPort": 4242,
			"IPCEnabled": true,
			"WSEnabled": false,
			"UpstreamConfig": {
				"Enabled": true,
				"URL": "http://upstream.loco.net/nodes"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.NetworkID != 3 {
				t.Fatal("wrong NetworkId")
			}

			if !nodeConfig.UpstreamConfig.Enabled {
				t.Fatal("wrong UpstreamConfig.Enabled state")
			}

			if nodeConfig.UpstreamConfig.URL != "http://upstream.loco.net/nodes" {
				t.Fatal("wrong UpstreamConfig.URL value")
			}
		},
	},
	{
		`test parameter overriding`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"Name": "TestStatusNode",
			"WSPort": 4242,
			"IPCEnabled": true,
			"WSEnabled": false,
			"RPCEnabled": true,
			"LightEthConfig": {
				"DatabaseCache": 64
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)

			require.EqualValues(t, 3, nodeConfig.NetworkID)
			require.Equal(t, "TestStatusNode", nodeConfig.Name)
			require.Equal(t, params.HTTPPort, nodeConfig.HTTPPort)
			require.Equal(t, params.HTTPHost, nodeConfig.HTTPHost)
			require.True(t, nodeConfig.RPCEnabled)
			require.False(t, nodeConfig.WSEnabled)
			require.Equal(t, 4242, nodeConfig.WSPort)
			require.True(t, nodeConfig.IPCEnabled)
			require.Equal(t, 64, nodeConfig.LightEthConfig.DatabaseCache)
		},
	},
	{
		`test loading Testnet config`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"Name": "TestStatusNode",
			"WSPort": 8546,
			"IPCEnabled": true,
			"WSEnabled": false,
			"LightEthConfig": {
				"DatabaseCache": 64
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)

			genesis := new(core.Genesis)
			err = json.Unmarshal([]byte(nodeConfig.LightEthConfig.Genesis), genesis)
			require.NoError(t, err)

			chainConfig := genesis.Config
			refChainConfig := gethparams.TestnetChainConfig

			require.Empty(t, chainConfig.HomesteadBlock.Cmp(refChainConfig.HomesteadBlock), "invalid chainConfig.HomesteadBlock")
			require.Nil(t, chainConfig.DAOForkBlock)
			require.Equal(t, refChainConfig.DAOForkSupport, chainConfig.DAOForkSupport)

			require.Empty(t, chainConfig.EIP150Block.Cmp(refChainConfig.EIP150Block))
			require.Equal(t, refChainConfig.EIP150Hash, chainConfig.EIP150Hash)

			require.Empty(t, chainConfig.EIP155Block.Cmp(refChainConfig.EIP155Block))
			require.Empty(t, chainConfig.EIP158Block.Cmp(refChainConfig.EIP158Block))
			require.Empty(t, chainConfig.ChainId.Cmp(refChainConfig.ChainId))
		},
	},
	{
		`test loading Mainnet config`,
		true,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR",
			"Name": "TestStatusNode",
			"WSPort": 8546,
			"IPCEnabled": true,
			"WSEnabled": false,
			"LightEthConfig": {
				"DatabaseCache": 64
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)

			genesis := new(core.Genesis)
			err = json.Unmarshal([]byte(nodeConfig.LightEthConfig.Genesis), genesis)
			require.NoError(t, err)

			chainConfig := genesis.Config

			require.Empty(t, chainConfig.HomesteadBlock.Cmp(gethparams.MainnetChainConfig.HomesteadBlock))
			require.Empty(t, chainConfig.DAOForkBlock.Cmp(gethparams.MainnetChainConfig.DAOForkBlock))
			require.True(t, chainConfig.DAOForkSupport)
			require.Empty(t, chainConfig.EIP150Block.Cmp(gethparams.MainnetChainConfig.EIP150Block))
			require.Equal(t, gethparams.MainnetChainConfig.EIP150Hash, chainConfig.EIP150Hash)
			require.Empty(t, chainConfig.EIP155Block.Cmp(gethparams.MainnetChainConfig.EIP155Block))
			require.Empty(t, chainConfig.EIP158Block.Cmp(gethparams.MainnetChainConfig.EIP158Block))
			require.Empty(t, chainConfig.ChainId.Cmp(gethparams.MainnetChainConfig.ChainId))
		},
	},
	{
		`test loading Privatenet config`,
		false,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"Name": "TestStatusNode",
			"WSPort": 8546,
			"IPCEnabled": true,
			"WSEnabled": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.EqualValues(t, 311, nodeConfig.NetworkID)
		},
	},
	{
		`default boot cluster (Ropsten Dev)`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")

			enodes := nodeConfig.BootClusterConfig.BootNodes
			require.True(t, len(enodes) >= 3)
		},
	},
	{
		`default boot cluster (Ropsten Prod)`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")

			enodes := nodeConfig.BootClusterConfig.BootNodes
			require.True(t, len(enodes) >= 3)
		},
	},
	{
		`disabled boot cluster`,
		false,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"Enabled": false
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.False(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be disabled")
		},
	},
	{
		`select boot cluster (Rinkeby Dev)`,
		false,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")

			enodes := nodeConfig.BootClusterConfig.BootNodes
			require.True(t, len(enodes) >= 3)
		},
	},
	{
		`select boot cluster (Rinkeby Prod)`,
		false,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")

			enodes := nodeConfig.BootClusterConfig.BootNodes
			require.True(t, len(enodes) >= 3)
		},
	},
	{
		`select boot cluster (Mainnet dev)`,
		true,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")

			enodes := nodeConfig.BootClusterConfig.BootNodes
			require.True(t, len(enodes) >= 2)
		},
	},
	{
		`select boot cluster (Mainnet Prod)`,
		true,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")

			enodes := nodeConfig.BootClusterConfig.BootNodes
			require.True(t, len(enodes) >= 2)
		},
	},
	{
		`select Mainnet without flag`,
		false,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.EqualError(t, err, "code not compiled for use of mainnet")
		},
	},
	{
		`default DevMode (true)`,
		false,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.DevMode)
			require.True(t, nodeConfig.BootClusterConfig.Enabled)
		},
	},
	{
		`explicit DevMode = false`,
		false,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.False(t, nodeConfig.DevMode)
			require.True(t, nodeConfig.BootClusterConfig.Enabled)
		},
	},
}

// TestLoadNodeConfig tests loading JSON configuration and setting default values.
func TestLoadNodeConfig(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	// create sample Bootstrap Cluster Config
	bootstrapConfig := []byte(`["enode://foobar@41.41.41.41:30300", "enode://foobaz@42.42.42.42:30302"]`)
	err = ioutil.WriteFile(filepath.Join(tmpDir, "bootstrap-cluster.json"), bootstrapConfig, os.ModePerm)
	require.NoError(t, err)
	t.Log(tmpDir)

	for _, testCase := range loadConfigTestCases {
		t.Log("test: " + testCase.name)
		params.UseMainnet = testCase.useMainnetFlag
		testCase.configJSON = strings.Replace(testCase.configJSON, "$TMPDIR", tmpDir, -1)
		nodeConfig, err := params.LoadNodeConfig(testCase.configJSON)
		testCase.validator(t, tmpDir, nodeConfig, err)
	}
}

func TestConfigWriteRead(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	nodeConfig, err := params.NewNodeConfig(tmpDir, params.RopstenNetworkID, true)
	require.Nil(t, err, "cannot create new config object")

	err = nodeConfig.Save()
	require.Nil(t, err, "cannot persist configuration")

	loadedConfigData, err := ioutil.ReadFile(filepath.Join(nodeConfig.DataDir, "config.json"))
	require.Nil(t, err, "cannot read configuration from disk")
	require.Contains(t, string(loadedConfigData), fmt.Sprintf(`"DevMode": %t`, true))
	require.Contains(t, string(loadedConfigData), fmt.Sprintf(`"NetworkId": %d`, params.RopstenNetworkID))
	require.Contains(t, string(loadedConfigData), fmt.Sprintf(`"DataDir": "%s"`, tmpDir))
}

// TestNodeConfigValidate checks validation of individual fields.
func TestNodeConfigValidate(t *testing.T) {
	testCases := []struct {
		Name        string
		Config      string
		Error       string
		FieldErrors map[string]string // map[Field]Tag
	}{
		{
			Name: "Valid JSON config",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/tmp/data"
			}`,
			Error:       "",
			FieldErrors: nil,
		},
		{
			Name:        "Invalid JSON config",
			Config:      `{"NetworkId": }`,
			Error:       "invalid character '}'",
			FieldErrors: nil,
		},
		{
			Name:        "Invalid field type",
			Config:      `{"NetworkId": "abc"}`,
			Error:       "json: cannot unmarshal string into Go struct field",
			FieldErrors: nil,
		},
		{
			Name:   "Validate all required fields",
			Config: `{}`,
			Error:  "",
			FieldErrors: map[string]string{
				"NetworkID": "required",
				"DataDir":   "required",
			},
		},
		{
			Name: "Validate Name does not contain slash",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"Name": "invalid/name"
			}`,
			Error: "",
			FieldErrors: map[string]string{
				"Name": "excludes",
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("Test Case %s", tc.Name)

		_, err := params.LoadNodeConfig(tc.Config)

		switch err := err.(type) {
		case validator.ValidationErrors:
			for _, ve := range err {
				require.Contains(t, tc.FieldErrors, ve.Field())
				require.Equal(t, tc.FieldErrors[ve.Field()], ve.Tag())
			}
		case error:
			require.Contains(t, err.Error(), tc.Error)
		case nil:
			require.Empty(t, tc.Error)
			require.Nil(t, tc.FieldErrors)
		}
	}
}
