package params_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

var clusterConfigData = []byte(`{
	"ClusterConfig": {
		"staticnodes": [
			"enode://7ab298cedc4185a894d21d8a4615262ec6bdce66c9b6783878258e0d5b31013d30c9038932432f70e5b2b6a5cd323bf820554fcb22fbc7b45367889522e9c449@10.1.1.1:30303",
			"enode://f59e8701f18c79c5cbc7618dc7bb928d44dc2f5405c7d693dad97da2d8585975942ec6fd36d3fe608bfdc7270a34a4dd00f38cfe96b2baa24f7cd0ac28d382a1@10.1.1.2:30303"
		]
	}
}`)

var clusters = map[string]func() params.Cluster{
	params.FleetStaging: func() params.Cluster {
		return params.Cluster{
			BootNodes: []string{
				"enode://10a78c17929a7019ef4aa2249d7302f76ae8a06f40b2dc88b7b31ebff4a623fbb44b4a627acba296c1ced3775d91fbe18463c15097a6a36fdb2c804ff3fc5b35@35.238.97.234:30404",   // boot-01.gc-us-central1-a.eth.staging
				"enode://f79fb3919f72ca560ad0434dcc387abfe41e0666201ebdada8ede0462454a13deb05cda15f287d2c4bd85da81f0eb25d0a486bbbc8df427b971ac51533bd00fe@174.138.107.239:30404", // boot-01.do-ams3.eth.staging
			},
			StaticNodes: []string{
				"enode://914c0b30f27bab30c1dfd31dad7652a46fda9370542aee1b062498b1345ee0913614b8b9e3e84622e84a7203c5858ae1d9819f63aece13ee668e4f6668063989@167.99.19.148:30305", // node-01.do-ams3.eth.staging
				"enode://2d897c6e846949f9dcf10279f00e9b8325c18fe7fa52d658520ad7be9607c83008b42b06aefd97cfe1fdab571f33a2a9383ff97c5909ed51f63300834913237e@35.192.0.86:30305",   // "node-01.gc-us-central1-a.eth.staging"
			},
			MailServers: []string{
				"enode://69f72baa7f1722d111a8c9c68c39a31430e9d567695f6108f31ccb6cd8f0adff4991e7fdca8fa770e75bc8a511a87d24690cbc80e008175f40c157d6f6788d48@206.189.240.16:30504", // mail-01.do-ams3.eth.staging
				"enode://e4fc10c1f65c8aed83ac26bc1bfb21a45cc1a8550a58077c8d2de2a0e0cd18e40fd40f7e6f7d02dc6cd06982b014ce88d6e468725ffe2c138e958788d0002a7f@35.239.193.41:30504",  // mail-01.gc-us-central1-a.eth.staging
			},
			RendezvousNodes: []string{
				"/ip4/174.138.107.239/tcp/30703/ethv4/16Uiu2HAkyJHeetQ4DNpd4NZ2ntzxMo25zcdpvGQRqkD5pB9BE6RU",
				"/ip4/35.238.97.234/tcp/30703/ethv4/16Uiu2HAm1sVyXmkMNjdeDWqK2urbyC3oBHi8MDpCdYkns1nYafqz",
			},
		}
	},
	params.FleetBeta: func() params.Cluster {
		return params.Cluster{
			BootNodes: []string{
				"enode://436cc6f674928fdc9a9f7990f2944002b685d1c37f025c1be425185b5b1f0900feaf1ccc2a6130268f9901be4a7d252f37302c8335a2c1a62736e9232691cc3a@174.138.105.243:30404", // boot-01.do-ams3.eth.beta
				"enode://5395aab7833f1ecb671b59bf0521cf20224fe8162fc3d2675de4ee4d5636a75ec32d13268fc184df8d1ddfa803943906882da62a4df42d4fccf6d17808156a87@206.189.243.57:30404",  // boot-02.do-ams3.eth.beta
				"enode://7427dfe38bd4cf7c58bb96417806fab25782ec3e6046a8053370022cbaa281536e8d64ecd1b02e1f8f72768e295d06258ba43d88304db068e6f2417ae8bcb9a6@104.154.88.123:30404",  // boot-01.gc-us-central1-a.eth.beta
				"enode://ebefab39b69bbbe64d8cd86be765b3be356d8c4b24660f65d493143a0c44f38c85a257300178f7845592a1b0332811542e9a58281c835babdd7535babb64efc1@35.202.99.224:30404",   // boot-02.gc-us-central1-a.eth.beta
			},
			StaticNodes: []string{
				"enode://a6a2a9b3a7cbb0a15da74301537ebba549c990e3325ae78e1272a19a3ace150d03c184b8ac86cc33f1f2f63691e467d49308f02d613277754c4dccd6773b95e8@206.189.243.176:30304", // node-01.do-ams3.eth.beta
				"enode://207e53d9bf66be7441e3daba36f53bfbda0b6099dba9a865afc6260a2d253fb8a56a72a48598a4f7ba271792c2e4a8e1a43aaef7f34857f520c8c820f63b44c8@35.224.15.65:30304",    // node-01.gc-us-central1-a.eth.beta
			},
			MailServers: []string{
				"enode://c42f368a23fa98ee546fd247220759062323249ef657d26d357a777443aec04db1b29a3a22ef3e7c548e18493ddaf51a31b0aed6079bd6ebe5ae838fcfaf3a49@206.189.243.162:30504", // mail-01.do-ams3.eth.beta
				"enode://7aa648d6e855950b2e3d3bf220c496e0cae4adfddef3e1e6062e6b177aec93bc6cdcf1282cb40d1656932ebfdd565729da440368d7c4da7dbd4d004b1ac02bf8@206.189.243.169:30504", // mail-02.do-ams3.eth.beta
				"enode://8a64b3c349a2e0ef4a32ea49609ed6eb3364be1110253c20adc17a3cebbc39a219e5d3e13b151c0eee5d8e0f9a8ba2cd026014e67b41a4ab7d1d5dd67ca27427@206.189.243.168:30504", // mail-03.do-ams3.eth.beta
				"enode://7de99e4cb1b3523bd26ca212369540646607c721ad4f3e5c821ed9148150ce6ce2e72631723002210fac1fd52dfa8bbdf3555e05379af79515e1179da37cc3db@35.188.19.210:30504",   // mail-01.gc-us-central1-a.eth.beta
				"enode://015e22f6cd2b44c8a51bd7a23555e271e0759c7d7f52432719665a74966f2da456d28e154e836bee6092b4d686fe67e331655586c57b718be3997c1629d24167@35.226.21.19:30504",    // mail-02.gc-us-central1-a.eth.beta
				"enode://531e252ec966b7e83f5538c19bf1cde7381cc7949026a6e499b6e998e695751aadf26d4c98d5a4eabfb7cefd31c3c88d600a775f14ed5781520a88ecd25da3c6@35.225.227.79:30504",   // mail-03.gc-us-central1-a.eth.beta
			},
			RendezvousNodes: []string{
				"/ip4/174.138.105.243/tcp/30703/ethv4/16Uiu2HAmRHPzF3rQg55PgYPcQkyvPVH9n2hWsYPhUJBZ6kVjJgdV", // boot-01.do-ams3.eth.beta
				"/ip4/206.189.243.57/tcp/30703/ethv4/16Uiu2HAmLqTXuY4Sb6G28HNooaFUXUKzpzKXCcgyJxgaEE2i5vnf",  // boot-02.do-ams3.eth.beta
			},
		}
	},
}

// ClusterForFleet returns a cluster for a given fleet.
func ClusterForFleet(fleet string) (params.Cluster, bool) {
	cluster, ok := clusters[fleet]
	if ok {
		return cluster(), true
	}
	return params.Cluster{}, false
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
	defer os.Remove(clusterFile)

	c, err := params.NewConfigFromJSON(`{
		"NetworkId": 3,
		"DataDir": "` + tmpDir + `",
		"KeyStoreDir": "` + tmpDir + `",
		"NoDiscovery": true
	}`)
	require.NoError(t, err)
	err = params.LoadConfigFromFiles([]string{clusterFile}, c)
	require.NoError(t, err)
	require.Equal(t, uint64(3), c.NetworkID)
	require.Equal(t, tmpDir, c.DataDir)
	require.Equal(t, tmpDir, c.KeyStoreDir)
	require.False(t, c.ClusterConfig.Enabled)
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
			Name:   "DataDir and KeyStoreDir specified",
			Update: func(c *params.NodeConfig) {},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.Equal(t, tmpDir, c.DataDir)
				require.Equal(t, tmpDir, c.KeyStoreDir)
				require.False(t, c.UpstreamConfig.Enabled)
				require.Equal(t, c.ClusterConfig.Fleet != params.FleetUndefined, c.ClusterConfig.Enabled)
				require.True(t, c.WhisperConfig.Enabled)
				require.False(t, c.LightEthConfig.Enabled)
				require.False(t, c.SwarmConfig.Enabled)
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
			Update: func(c *params.NodeConfig) {
				c.LightEthConfig.Enabled = true
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.LightEthConfig.Enabled)
			},
		},
		{
			Name: "cluster nodes setup",
			Update: func(c *params.NodeConfig) {
				cc, _ := ClusterForFleet(c.ClusterConfig.Fleet)
				c.Rendezvous = true
				c.ClusterConfig.BootNodes = cc.BootNodes
				c.ClusterConfig.StaticNodes = cc.StaticNodes
				c.ClusterConfig.RendezvousNodes = cc.RendezvousNodes
				c.ClusterConfig.TrustedMailServers = cc.MailServers
				c.ClusterConfig.Enabled = true
				c.RequireTopics[params.WhisperDiscv5Topic] = params.WhisperDiscv5Limits
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.Rendezvous)
				require.True(t, c.ClusterConfig.Enabled)
				require.NotEmpty(t, c.ClusterConfig.BootNodes)
				require.NotEmpty(t, c.ClusterConfig.StaticNodes)
				require.NotEmpty(t, c.ClusterConfig.TrustedMailServers)
				require.Equal(t, params.WhisperDiscv5Limits, c.RequireTopics[params.WhisperDiscv5Topic])
			},
		},
		{
			Name: "custom bootnodes",
			Update: func(c *params.NodeConfig) {
				c.ClusterConfig.Enabled = true
				c.ClusterConfig.BootNodes = []string{"a", "b", "c"}
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.True(t, c.ClusterConfig.Enabled)
				require.Equal(t, []string{"a", "b", "c"}, c.ClusterConfig.BootNodes)
			},
		},
		{
			Name: "disabled Cluster configuration",
			Update: func(c *params.NodeConfig) {
				c.ClusterConfig.Enabled = false
				c.ClusterConfig.BootNodes = []string{"a", "b", "c"}
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.False(t, c.ClusterConfig.Enabled)
			},
		},
		{
			Name: "peers discovery and topics",
			Update: func(c *params.NodeConfig) {
				c.NoDiscovery = false
				c.ClusterConfig.BootNodes = []string{"a", "b", "c"}
				c.RequireTopics[params.WhisperDiscv5Topic] = params.Limits{2, 2}
			},
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
			Name:  "staging fleet",
			Fleet: params.FleetStaging,
			Update: func(c *params.NodeConfig) {
				c.ClusterConfig.Enabled = true
				c.ClusterConfig.Fleet = "eth.staging"
				cc, _ := ClusterForFleet(c.ClusterConfig.Fleet)
				c.ClusterConfig.BootNodes = cc.BootNodes
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				staging, ok := ClusterForFleet("eth.staging")
				require.True(t, ok)
				beta, ok := ClusterForFleet("eth.beta")
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
				c.WhisperConfig.Enabled = true
				c.WhisperConfig.DataDir = path.Join(tmpDir, "wnode")
				c.WhisperConfig.LightClient = true
			},
			Validate: func(t *testing.T, dataDir string, c *params.NodeConfig) {
				require.Equal(t, path.Join(tmpDir, "wnode"), c.WhisperConfig.DataDir)
				require.True(t, c.WhisperConfig.Enabled)
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
					config, err := utils.MakeTestNodeConfigWithDataDir("", tmpDir, fleet, uint64(networkID))
					require.NoError(t, err)
					config.KeyStoreDir = tmpDir

					// Corresponds to config update in status-react.
					tc.Update(config)
					configBytes, err := json.Marshal(config)
					require.NoError(t, err)

					// Corresponds to starting node and loading config from JSON blob.
					loadedConfig, err := params.NewConfigFromJSON(string(configBytes))
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

	nodeConfig, err := utils.MakeTestNodeConfigWithDataDir("", tmpDir, params.FleetBeta, params.RopstenNetworkID)
	require.Nil(t, err, "cannot create new config object")

	err = nodeConfig.Save()
	require.Nil(t, err, "cannot persist configuration")

	loadedConfigData, err := ioutil.ReadFile(filepath.Join(nodeConfig.DataDir, "config.json"))
	require.Nil(t, err, "cannot read configuration from disk")
	loadedConfig := string(loadedConfigData)
	require.Contains(t, loadedConfig, fmt.Sprintf(`"NetworkId": %d`, params.RopstenNetworkID))
	require.Contains(t, loadedConfig, fmt.Sprintf(`"DataDir": "%s"`, tmpDir))
	require.Contains(t, loadedConfig, fmt.Sprintf(`"Fleet": "%s"`, params.FleetBeta))
}

// TestNodeConfigValidate checks validation of individual fields.
func TestNodeConfigValidate(t *testing.T) {
	testCases := []struct {
		Name        string
		Config      string
		Error       string
		FieldErrors map[string]string // map[Field]Tag
		CheckFunc   func(*testing.T, *params.NodeConfig)
	}{
		{
			Name: "Valid JSON config",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/tmp/data",
				"KeyStoreDir": "/tmp/data",
				"NoDiscovery": true
			}`,
		},
		{
			Name:   "Invalid JSON config",
			Config: `{"NetworkId": }`,
			Error:  "invalid character '}'",
		},
		{
			Name:   "Invalid field type",
			Config: `{"NetworkId": "abc"}`,
			Error:  "json: cannot unmarshal string into Go struct field",
		},
		{
			Name:   "Validate all required fields",
			Config: `{}`,
			FieldErrors: map[string]string{
				"NetworkID":   "required",
				"DataDir":     "required",
				"KeyStoreDir": "required",
			},
		},
		{
			Name: "Validate that Name does not contain slash",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"Name": "invalid/name"
			}`,
			FieldErrors: map[string]string{
				"Name": "excludes",
			},
		},
		{
			Name: "Validate that NodeKey is checked for validity",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"NodeKey": "foo"
			}`,
			Error: "NodeKey is invalid",
		},
		{
			Name: "Validate that UpstreamConfig.URL is validated if UpstreamConfig is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"UpstreamConfig": {
					"Enabled": true,
					"URL": "[bad.url]"
				}
			}`,
			Error: "'[bad.url]' is invalid",
		},
		{
			Name: "Validate that UpstreamConfig.URL is not validated if UpstreamConfig is disabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"UpstreamConfig": {
					"Enabled": false,
					"URL": "[bad.url]"
				}
			}`,
		},
		{
			Name: "Validate that UpstreamConfig.URL validation passes if UpstreamConfig.URL is valid",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"UpstreamConfig": {
					"Enabled": true,
					"URL": "` + params.MainnetEthereumNetworkURL + `"
				}
			}`,
		},
		{
			Name: "Validate that ClusterConfig.Fleet is verified to not be empty if ClusterConfig is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"ClusterConfig": {
					"Enabled": true
				} 
			}`,
			Error: "ClusterConfig.Fleet is empty",
		},
		{
			Name: "Validate that ClusterConfig.BootNodes is verified to not be empty if discovery is disabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": false
			}`,
			Error: "NoDiscovery is false, but ClusterConfig.BootNodes is empty",
		},
		{
			Name: "Validate that ClusterConfig.RendezvousNodes is verified to be empty if Rendezvous is disabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"Rendezvous": true 
			}`,
			Error: "Rendezvous is enabled, but ClusterConfig.RendezvousNodes is empty",
		},
		{
			Name: "Validate that ClusterConfig.RendezvousNodes is verified to contain nodes if Rendezvous is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"Rendezvous": false,
				"ClusterConfig": {
					"RendezvousNodes": ["a"]
				} 
			}`,
			Error: "Rendezvous is disabled, but ClusterConfig.RendezvousNodes is not empty",
		},
		{
			Name: "Validate that WhisperConfig.DataDir is checked to not be empty if mailserver is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WhisperConfig": {
					"Enabled": true,
					"EnableMailServer": true,
					"MailserverPassword": "foo"
				}
			}`,
			Error: "WhisperConfig.DataDir must be specified when WhisperConfig.EnableMailServer is true",
		},
		{
			Name: "Validate that check for WhisperConfig.DataDir passes if it is not empty and mailserver is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WhisperConfig": {
					"Enabled": true,
					"EnableMailServer": true,
					"DataDir": "/foo",
					"MailserverPassword": "foo"
				}
			}`,
			CheckFunc: func(t *testing.T, config *params.NodeConfig) {
				require.Equal(t, "foo", config.WhisperConfig.MailServerPassword)
			},
		},
		{
			Name: "Validate that WhisperConfig.DataDir is checked to not be empty if mailserver is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WhisperConfig": {
					"Enabled": true,
					"EnableMailServer": true,
					"MailserverPassword": "foo"
				}
			}`,
			Error: "WhisperConfig.DataDir must be specified when WhisperConfig.EnableMailServer is true",
		},
		{
			Name: "Validate that WhisperConfig.MailserverPassword and WhisperConfig.MailServerAsymKey are checked to not be empty if mailserver is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WhisperConfig": {
					"Enabled": true,
					"EnableMailServer": true,
					"DataDir": "/foo"
				}
			}`,
			Error: "WhisperConfig.MailServerPassword or WhisperConfig.MailServerAsymKey must be specified when WhisperConfig.EnableMailServer is true",
		},
		{
			Name: "Validate that WhisperConfig.MailServerAsymKey is checked to not be empty if mailserver is enabled",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WhisperConfig": {
					"Enabled": true,
					"EnableMailServer": true,
					"DataDir": "/foo",
					"MailServerAsymKey": "06c365919f1fc8e13ff79a84f1dd14b7e45b869aa5fc0e34940481ee20d32f90"
				}
			}`,
			CheckFunc: func(t *testing.T, config *params.NodeConfig) {
				require.Equal(t, "06c365919f1fc8e13ff79a84f1dd14b7e45b869aa5fc0e34940481ee20d32f90", config.WhisperConfig.MailServerAsymKey)
			},
		},
		{
			Name: "Validate that WhisperConfig.MailServerAsymKey is checked for validity",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WhisperConfig": {
					"Enabled": true,
					"EnableMailServer": true,
					"DataDir": "/foo",
					"MailServerAsymKey": "bar"
				}
			}`,
			Error: "WhisperConfig.MailServerAsymKey is invalid",
		},
	}

	for _, tc := range testCases {
		t.Logf("Test Case %s", tc.Name)

		config, err := params.NewConfigFromJSON(tc.Config)

		switch err := err.(type) {
		case validator.ValidationErrors:
			for _, ve := range err {
				require.Contains(t, tc.FieldErrors, ve.Field())
				require.Equal(t, tc.FieldErrors[ve.Field()], ve.Tag())
			}
		case error:
			if tc.Error == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.Error)
			}
		case nil:
			if tc.Error != "" {
				require.Error(t, err, "Error should be '%v'", tc.Error)
			}
			require.Nil(t, tc.FieldErrors)
			if tc.CheckFunc != nil {
				tc.CheckFunc(t, config)
			}
		}
	}
}
