package params_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	validator "gopkg.in/go-playground/validator.v9"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/utils"
)

func TestNewNodeConfigWithDefaults(t *testing.T) {
	c, err := params.NewNodeConfigWithDefaults(
		"/some/data/path",
		params.GoerliNetworkID,
		params.WithFleet(params.FleetProd),
		params.WithLES(),
		params.WithMailserver(),
	)
	require.NoError(t, err)
	assert.Equal(t, "/some/data/path", c.DataDir)
	assert.Equal(t, "/some/data/path/keystore", c.KeyStoreDir)
	// assert Whisper
	assert.Equal(t, true, c.WakuConfig.Enabled)
	assert.Equal(t, "/some/data/path/waku", c.WakuConfig.DataDir)
	// assert MailServer
	assert.Equal(t, false, c.WakuConfig.EnableMailServer)
	// assert cluster
	assert.Equal(t, false, c.NoDiscovery)
	assert.Equal(t, params.FleetProd, c.ClusterConfig.Fleet)
	assert.NotEmpty(t, c.ClusterConfig.BootNodes)
	assert.NotEmpty(t, c.ClusterConfig.StaticNodes)
	assert.NotEmpty(t, c.ClusterConfig.PushNotificationsServers)
	// assert LES
	assert.Equal(t, true, c.LightEthConfig.Enabled)
	// assert other
	assert.Equal(t, false, c.HTTPEnabled)
	assert.Equal(t, false, c.IPCEnabled)

	assert.Equal(t, "/some/data/path/archivedata", c.TorrentConfig.DataDir)
	assert.Equal(t, "/some/data/path/torrents", c.TorrentConfig.TorrentDir)
	assert.Equal(t, 9025, c.TorrentConfig.Port)
	assert.Equal(t, false, c.TorrentConfig.Enabled)

	assert.NoError(t, c.UpdateWithDefaults())
	assert.NotEmpty(t, c.ShhextConfig.DefaultPushNotificationsServers)
}

func TestNewConfigFromJSON(t *testing.T) {
	tmpDir := t.TempDir()
	json := `{
		"NetworkId": 3,
		"DataDir": "` + tmpDir + `",
		"KeyStoreDir": "` + tmpDir + `",
		"NoDiscovery": true,
    "TorrentConfig": {
      "Port": 9025,
      "Enabled": false,
      "DataDir": "` + tmpDir + `/archivedata",
      "TorrentDir": "` + tmpDir + `/torrents"
    }
	}`
	c, err := params.NewConfigFromJSON(json)
	require.NoError(t, err)
	require.Equal(t, uint64(3), c.NetworkID)
	require.Equal(t, tmpDir, c.DataDir)
	require.Equal(t, tmpDir, c.KeyStoreDir)
	require.Equal(t, false, c.TorrentConfig.Enabled)
	require.Equal(t, 9025, c.TorrentConfig.Port)
	require.Equal(t, tmpDir+"/archivedata", c.TorrentConfig.DataDir)
	require.Equal(t, tmpDir+"/torrents", c.TorrentConfig.TorrentDir)
}

func TestConfigWriteRead(t *testing.T) {
	tmpDir := t.TempDir()

	nodeConfig, err := utils.MakeTestNodeConfigWithDataDir("", tmpDir, params.GoerliNetworkID)
	require.Nil(t, err, "cannot create new config object")

	err = nodeConfig.Save()
	require.Nil(t, err, "cannot persist configuration")

	loadedConfigData, err := ioutil.ReadFile(filepath.Join(nodeConfig.DataDir, "config.json"))
	require.Nil(t, err, "cannot read configuration from disk")
	loadedConfig := string(loadedConfigData)
	require.Contains(t, loadedConfig, fmt.Sprintf(`"NetworkId": %d`, params.GoerliNetworkID))
	require.Contains(t, loadedConfig, fmt.Sprintf(`"DataDir": "%s"`, tmpDir))
	require.Contains(t, loadedConfig, fmt.Sprintf(`"BackupDisabledDataDir": "%s"`, tmpDir))
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
				"BackupDisabledDataDir": "/tmp/data",
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
			Name: "Validate that PFSEnabled & InstallationID are checked for validity",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"NoDiscovery": true,
				"WakuConfig": {
					"Enabled": true,
					"DataDir": "/foo"
				},
				"ShhextConfig": {
					"BackupDisabledDataDir": "/some/dir",
					"PFSEnabled": true
				}
			}`,
			Error: "PFSEnabled is true, but InstallationID is empty",
		},
		{
			Name: "Default HTTP virtual hosts is localhost and CORS is empty",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir"
			}`,
			CheckFunc: func(t *testing.T, config *params.NodeConfig) {
				require.Equal(t, []string{"localhost"}, config.HTTPVirtualHosts)
				require.Nil(t, config.HTTPCors)
			},
		},
		{
			Name: "Set HTTP virtual hosts and CORS",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"HTTPVirtualHosts": ["my.domain.com"],
				"HTTPCors": ["http://my.domain.com:8080"]
			}`,
			CheckFunc: func(t *testing.T, config *params.NodeConfig) {
				require.Equal(t, []string{"my.domain.com"}, config.HTTPVirtualHosts)
				require.Equal(t, []string{"http://my.domain.com:8080"}, config.HTTPCors)
			},
		},
		{
			Name: "ShhextConfig is not required",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir"
			}`,
		},
		{
			Name: "BackupDisabledDataDir must be set if PFSEnabled is true",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
				"ShhextConfig": {
					"PFSEnabled": true
				}
			}`,
			Error: "field BackupDisabledDataDir is required if PFSEnabled is true",
		},
		{
			Name:   "Missing APIModules",
			Config: `{"NetworkId": 1, "DataDir": "/tmp/data", "KeyStoreDir": "/tmp/data", "APIModules" :""}`,
			FieldErrors: map[string]string{
				"APIModules": "required",
			},
		},
		{
			Name: "Validate that TorrentConfig.DataDir and TorrentConfig.TorrentDir can't be empty strings",
			Config: `{
				"NetworkId": 1,
				"DataDir": "/some/dir",
				"KeyStoreDir": "/some/dir",
        "TorrentConfig": {
          "Enabled": true,
          "Port": 9025,
          "DataDir": "",
          "TorrentDir": ""
        }
      }`,
			Error: `TorrentConfig.DataDir and TorrentConfig.TorrentDir cannot be ""`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
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
					fmt.Println(tc.Error)
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
		})
	}
}
