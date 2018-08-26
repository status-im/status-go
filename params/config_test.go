package params_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/go-playground/validator.v9"

	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/require"
)

var clusterConfigData = []byte(`{
	"staticnodes": [
		"enode://7ab298cedc4185a894d21d8a4615262ec6bdce66c9b6783878258e0d5b31013d30c9038932432f70e5b2b6a5cd323bf820554fcb22fbc7b45367889522e9c449@10.1.1.1:30303",
		"enode://f59e8701f18c79c5cbc7618dc7bb928d44dc2f5405c7d693dad97da2d8585975942ec6fd36d3fe608bfdc7270a34a4dd00f38cfe96b2baa24f7cd0ac28d382a1@10.1.1.2:30303"
	]
}`)

func TestLoadNodeConfigFromNonExistingFile(t *testing.T) {
	_, err := params.LoadNodeConfig(`{
		"NetworkId": 3,
		"DataDir": "/tmp/statusgo",
		"ClusterConfigFile": "/file/does/not.exist"
	}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestLoadNodeConfigFromFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	// create cluster config file
	clusterFile := filepath.Join(tmpDir, "cluster.json")
	err = ioutil.WriteFile(clusterFile, clusterConfigData, os.ModePerm)
	require.NoError(t, err)

	c, err := params.LoadNodeConfig(`{
		"NetworkId": 3,
		"DataDir": "` + tmpDir + `",
		"ClusterConfigFile": "` + clusterFile + `"
	}`)
	require.NoError(t, err)
	require.True(t, c.ClusterConfig.Enabled)
	require.Len(t, c.ClusterConfig.StaticNodes, 2)
}

// TestGenerateAndLoadNodeConfig tests creating and loading config
// exactly as it's done by status-react.
func TestGenerateAndLoadNodeConfig(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	var testCases = []struct {
		Name      string
		Fleet     string // optional; if omitted all fleets will be tested
		NetworkID int    // optional; if omitted all networks will be checked
		Update    func(*params.NodeConfig)
		Validate  func(t *testing.T, dataDir string, c *params.NodeConfig)
	}{
		{
			Name:   "default KeyStoreDir",
			Update: func(config *params.NodeConfig) {},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.Equal(t, dataDir, c.DataDir)
				keyStoreDir := filepath.Join(dataDir, params.KeyStoreDir)
				require.Equal(t, keyStoreDir, c.KeyStoreDir)
			},
		},
		{
			Name: "non-default KeyStoreDir",
			Update: func(c *params.NodeConfig) {
				c.KeyStoreDir = "/foo/bar"
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.Equal(t, "/foo/bar", c.KeyStoreDir)
			},
		},
		{
			Name:      "custom network and upstream",
			NetworkID: 333,
			Update: func(c *params.NodeConfig) {
				c.UpstreamConfig.Enabled = true
				c.UpstreamConfig.URL = "http://custom.local"
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.Equal(t, uint64(333), c.NetworkID)
				require.True(t, c.UpstreamConfig.Enabled)
				require.Equal(t, "http://custom.local", c.UpstreamConfig.URL)
			},
		},
		{
			Name:      "upstream config",
			NetworkID: params.RopstenNetworkID,
			Update: func(c *params.NodeConfig) {
				c.UpstreamConfig.Enabled = true
				c.UpstreamConfig.URL = params.RopstenEthereumNetworkURL
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.UpstreamConfig.Enabled)
				require.Equal(t, params.RopstenEthereumNetworkURL, c.UpstreamConfig.URL)
			},
		},
		{
			Name:      "loading LES config",
			NetworkID: params.MainNetworkID,
			Update:    func(c *params.NodeConfig) {},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				var genesis core.Genesis
				err := json.Unmarshal([]byte(c.LightEthConfig.Genesis), &genesis)
				require.NoError(t, err)

				require.Zero(t, genesis.Config.ChainID.Cmp(gethparams.MainnetChainConfig.ChainID))
				require.Zero(t, genesis.Config.HomesteadBlock.Cmp(gethparams.MainnetChainConfig.HomesteadBlock))
				require.Zero(t, genesis.Config.EIP150Block.Cmp(gethparams.MainnetChainConfig.EIP150Block))
				require.Zero(t, genesis.Config.EIP155Block.Cmp(gethparams.MainnetChainConfig.EIP155Block))
				require.Zero(t, genesis.Config.EIP158Block.Cmp(gethparams.MainnetChainConfig.EIP158Block))
			},
		},
		{
			Name:   "cluster nodes setup",
			Update: func(c *params.NodeConfig) {},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.ClusterConfig.Enabled)
				require.NotEmpty(t, c.ClusterConfig.BootNodes)
				require.NotEmpty(t, c.ClusterConfig.StaticNodes)
				require.NotEmpty(t, c.ClusterConfig.TrustedMailServers)
			},
		},
		{
			Name: "custom bootnodes",
			Update: func(c *params.NodeConfig) {
				c.ClusterConfig.BootNodes = []string{"a", "b", "c"}
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.ClusterConfig.Enabled)
				require.Equal(t, []string{"a", "b", "c"}, c.ClusterConfig.BootNodes)
			},
		},
		{
			Name: "disabled ClusterConfiguration",
			Update: func(c *params.NodeConfig) {
				c.ClusterConfig.Enabled = false
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.False(t, c.ClusterConfig.Enabled)
			},
		},
		{
			Name:   "peers discovery and topics",
			Update: func(c *params.NodeConfig) {},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.NotNil(t, c.RequireTopics)
				require.False(t, c.NoDiscovery)
				require.Contains(t, c.RequireTopics, params.WhisperDiscv5Topic)
				require.Equal(t, params.WhisperDiscv5Limits, c.RequireTopics[params.WhisperDiscv5Topic])
			},
		},
		{
			Name: "verify NoDiscovery preserved",
			Update: func(c *params.NodeConfig) {
				c.NoDiscovery = true
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.NoDiscovery)
			},
		},
		{
			Name:   "staging fleet",
			Fleet:  params.FleetStaging,
			Update: func(c *params.NodeConfig) {},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				staging, ok := params.ClusterForFleet("eth.staging")
				require.True(t, ok)
				beta, ok := params.ClusterForFleet("eth.beta")
				require.True(t, ok)

				require.NotEqual(t, staging, beta)

				// test case asserts
				require.Equal(t, "eth.staging", c.ClusterConfig.Fleet)
				require.Equal(t, staging.BootNodes, c.ClusterConfig.BootNodes)
			},
		},
		{
			Name: "Whisper light client",
			Update: func(c *params.NodeConfig) {
				c.WhisperConfig.LightClient = true
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.WhisperConfig.LightClient)
			},
		},
	}

	for _, tc := range testCases {
		fleets := []string{params.FleetBeta, params.FleetStaging}
		if tc.Fleet != params.FleetUndefined {
			fleets = []string{tc.Fleet}
		}

		networks := []int{params.MainNetworkID, params.RinkebyNetworkID, params.RopstenNetworkID}
		if tc.NetworkID != 0 {
			networks = []int{tc.NetworkID}
		}

		for _, fleet := range fleets {
			for _, networkID := range networks {
				name := fmt.Sprintf("%s_%s_%d", tc.Name, fleet, networkID)
				t.Run(name, func(t *testing.T) {
					// Corresponds to GenerateConfig() binding.
					config, err := params.NewNodeConfig(tmpDir, "", fleet, uint64(networkID))
					require.NoError(t, err)

					// Corresponds to config update in status-react.
					tc.Update(config)
					configBytes, err := json.Marshal(config)
					require.NoError(t, err)

					// Corresponds to starting node and loading config from JSON blob.
					loadedConfig, err := params.LoadNodeConfig(string(configBytes))
					require.NoError(t, err)
					tc.Validate(t, tmpDir, loadedConfig)
				})
			}
		}
	}
}

func TestConfigWriteRead(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	nodeConfig, err := params.NewNodeConfig(tmpDir, "", params.FleetBeta, params.RopstenNetworkID)
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
