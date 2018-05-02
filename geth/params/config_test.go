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

var clusterConfigData = []byte(`
[
  {
    "networkID": 3,
    "staticnodes": [
      "enode://7ab298cedc4185a894d21d8a4615262ec6bdce66c9b6783878258e0d5b31013d30c9038932432f70e5b2b6a5cd323bf820554fcb22fbc7b45367889522e9c449@10.1.1.1:30303",
      "enode://f59e8701f18c79c5cbc7618dc7bb928d44dc2f5405c7d693dad97da2d8585975942ec6fd36d3fe608bfdc7270a34a4dd00f38cfe96b2baa24f7cd0ac28d382a1@10.1.1.2:30303"
    ]
  }
]
`)

var loadConfigTestCases = []struct {
	name       string
	configJSON string
	validator  func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error)
}{
	{
		`invalid input JSON (missing comma at the end of key:value pair)`,
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
			require.True(t, nodeConfig.IPCEnabled)
			require.Equal(t, 64, nodeConfig.LightEthConfig.DatabaseCache)
		},
	},
	{
		`test loading Testnet config`,
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
		`default static nodes (Ropsten Dev)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.ClusterConfig.Enabled, "static nodes are expected to be enabled by default")

			enodes := nodeConfig.ClusterConfig.StaticNodes
			t.Logf("LEN SN %d", len(enodes))
			require.Len(t, enodes, 2)
		},
	},
	{
		`illegal cluster configuration file`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"ClusterConfigFile": "/file/does/not.exist"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.Error(t, err, "error is expected, not thrown")
		},
	},
	{
		`valid cluster configuration file`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"ClusterConfigFile": "$TMPDIR/cluster.json"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.ClusterConfig.Enabled, "cluster configuration is expected to be enabled after loading file")

			enodes := nodeConfig.ClusterConfig.StaticNodes
			require.Len(t, enodes, 2)
		},
	},
	{
		`default cluster configuration (Ropsten Prod)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.ClusterConfig.Enabled, "cluster configuration is expected to be enabled by default")

			enodes := nodeConfig.ClusterConfig.StaticNodes
			require.Len(t, enodes, 2)
		},
	},
	{
		`disabled cluster configuration`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"ClusterConfig": {
				"Enabled": false
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.False(t, nodeConfig.ClusterConfig.Enabled, "cluster configuration is expected to be disabled")
		},
	},
	{
		`select cluster configuration (Rinkeby Dev)`,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.ClusterConfig.Enabled, "cluster configuration is expected to be enabled by default")
			require.False(t, nodeConfig.NoDiscovery)
			require.True(t, len(nodeConfig.ClusterConfig.BootNodes) >= 2)
		},
	},
	{
		`select cluster configuration (Mainnet dev)`,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.ClusterConfig.Enabled, "cluster configuration is expected to be enabled by default")

			enodes := nodeConfig.ClusterConfig.StaticNodes
			require.True(t, len(enodes) >= 2)
		},
	},
	{
		`explicit WhisperConfig.LightClient = true`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"WhisperConfig": {
				"LightClient": true
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.WhisperConfig.LightClient)
		},
	},
	{
		`default peer limits`,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.NotNil(t, nodeConfig.RequireTopics)
			require.False(t, nodeConfig.NoDiscovery)
			require.Contains(t, nodeConfig.RequireTopics, params.WhisperDiscv5Topic)
			require.Equal(t, params.WhisperDiscv5Limits, nodeConfig.RequireTopics[params.WhisperDiscv5Topic])
		},
	},
	{
		`no discovery preserved`,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR",
			"NoDiscovery": true
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.NoDiscovery)
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

	// create sample bootnodes config
	err = ioutil.WriteFile(filepath.Join(tmpDir, "cluster.json"), clusterConfigData, os.ModePerm)
	require.NoError(t, err)
	t.Log(tmpDir)

	for _, testCase := range loadConfigTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.configJSON = strings.Replace(testCase.configJSON, "$TMPDIR", tmpDir, -1)
			nodeConfig, err := params.LoadNodeConfig(testCase.configJSON)
			testCase.validator(t, tmpDir, nodeConfig, err)
		})
	}
}

func TestConfigWriteRead(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	nodeConfig, err := params.NewNodeConfig(tmpDir, "", params.RopstenNetworkID)
	require.Nil(t, err, "cannot create new config object")

	err = nodeConfig.Save()
	require.Nil(t, err, "cannot persist configuration")

	loadedConfigData, err := ioutil.ReadFile(filepath.Join(nodeConfig.DataDir, "config.json"))
	require.Nil(t, err, "cannot read configuration from disk")
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
