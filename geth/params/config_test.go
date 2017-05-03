package params_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/params"
)

var loadConfigTestCases = []struct {
	name       string
	configJSON string
	validator  func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error)
}{
	{
		`invalid input configuration`,
		`{
			"NetworkId": 3
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
			if err == nil {
				t.Fatal("error is expected, not thrown")
			}
		},
	},
	{
		`missing required field (DataDir)`,
		`{
			"NetworkId": 3,
			"Name": "TestStatusNode"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != params.ErrMissingDataDir {
				t.Fatalf("expected error not thrown, expected: %v, thrown: %v", params.ErrMissingDataDir, err)
			}
		},
	},
	{
		`missing required field (NetworkId)`,
		`{
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != params.ErrMissingNetworkID {
				t.Fatalf("expected error not thrown, expected: %v, thrown: %v", params.ErrMissingNetworkID, err)
			}
		},
	},
	{
		`check static DataDir passing`,
		`{
			"NetworkId": 3,
			"DataDir": "/storage/emulated/0/ethereum/"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedDataDir := "/storage/emulated/0/ethereum/"
			if nodeConfig.DataDir != expectedDataDir {
				t.Fatalf("incorrect DataDir used, expected: %v, got: %v", expectedDataDir, nodeConfig.DataDir)
			}
		},
	},
	{
		`use default KeyStoreDir`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := os.Stat(dataDir); os.IsNotExist(err) {
				t.Fatalf("data directory doesn't exist: %s", dataDir)
			}

			expectedDataDir := dataDir
			if nodeConfig.DataDir != expectedDataDir {
				t.Fatalf("incorrect DataDir used, expected: %v, got: %v", expectedDataDir, nodeConfig.DataDir)
			}

			expectedKeyStoreDir := filepath.Join(dataDir, params.KeyStoreDir)
			if nodeConfig.KeyStoreDir != expectedKeyStoreDir {
				t.Fatalf("incorrect KeyStoreDir used, expected: %v, got: %v", expectedKeyStoreDir, nodeConfig.KeyStoreDir)
			}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedDataDir := dataDir
			if nodeConfig.DataDir != expectedDataDir {
				t.Fatalf("incorrect DataDir used, expected: %v, got: %v", expectedDataDir, nodeConfig.DataDir)
			}

			expectedKeyStoreDir := "/foo/bar"
			if nodeConfig.KeyStoreDir != expectedKeyStoreDir {
				t.Fatalf("incorrect KeyStoreDir used, expected: %v, got: %v", expectedKeyStoreDir, nodeConfig.KeyStoreDir)
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
			"LightEthConfig": {
				"DatabaseCache": 64
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.NetworkID != 3 {
				t.Fatal("wrong NetworkId")
			}

			if nodeConfig.Name != "TestStatusNode" {
				t.Fatal("wrong Name")
			}

			if nodeConfig.HTTPPort != params.HTTPPort {
				t.Fatal("wrong HTTPPort")
			}

			if nodeConfig.HTTPHost != params.HTTPHost {
				t.Fatal("wrong HTTPHost")
			}

			if nodeConfig.WSPort != 4242 {
				t.Fatal("wrong WSPort")
			}

			if nodeConfig.WSEnabled {
				t.Fatal("wrong WSEnabled")
			}

			if !nodeConfig.IPCEnabled {
				t.Fatal("wrong IPCEnabled")
			}
			if nodeConfig.LightEthConfig.DatabaseCache != 64 {
				t.Fatal("wrong LightEthConfig.DatabaseCache")
			}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			genesis := new(core.Genesis)
			if err := json.Unmarshal([]byte(nodeConfig.LightEthConfig.Genesis), genesis); err != nil {
				t.Fatal(err)
			}
			chainConfig := genesis.Config
			refChainConfig := gethparams.TestnetChainConfig

			if chainConfig.HomesteadBlock.Cmp(refChainConfig.HomesteadBlock) != 0 {
				t.Fatal("invalid chainConfig.HomesteadBlock")
			}
			if chainConfig.DAOForkBlock != nil { // already forked
				t.Fatal("invalid chainConfig.DAOForkBlock")
			}
			if chainConfig.DAOForkSupport != refChainConfig.DAOForkSupport {
				t.Fatal("invalid chainConfig.DAOForkSupport")
			}
			if chainConfig.EIP150Block.Cmp(refChainConfig.EIP150Block) != 0 {
				t.Fatal("invalid chainConfig.EIP150Block")
			}
			if chainConfig.EIP150Hash != refChainConfig.EIP150Hash {
				t.Fatal("invalid chainConfig.EIP150Hash")
			}
			if chainConfig.EIP155Block.Cmp(refChainConfig.EIP155Block) != 0 {
				t.Fatal("invalid chainConfig.EIP155Block")
			}
			if chainConfig.EIP158Block.Cmp(refChainConfig.EIP158Block) != 0 {
				t.Fatal("invalid chainConfig.EIP158Block")
			}
			if chainConfig.ChainId.Cmp(refChainConfig.ChainId) != 0 {
				t.Fatal("invalid chainConfig.ChainId")
			}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			genesis := new(core.Genesis)
			if err := json.Unmarshal([]byte(nodeConfig.LightEthConfig.Genesis), genesis); err != nil {
				t.Fatal(err)
			}
			chainConfig := genesis.Config
			if chainConfig.HomesteadBlock.Cmp(gethparams.MainNetHomesteadBlock) != 0 {
				t.Fatal("invalid chainConfig.HomesteadBlock")
			}
			if chainConfig.DAOForkBlock.Cmp(gethparams.MainNetDAOForkBlock) != 0 {
				t.Fatal("invalid chainConfig.DAOForkBlock")
			}
			if !chainConfig.DAOForkSupport {
				t.Fatal("invalid chainConfig.DAOForkSupport")
			}
			if chainConfig.EIP150Block.Cmp(gethparams.MainNetHomesteadGasRepriceBlock) != 0 {
				t.Fatal("invalid chainConfig.EIP150Block")
			}
			if chainConfig.EIP150Hash != gethparams.MainNetHomesteadGasRepriceHash {
				t.Fatal("invalid chainConfig.EIP150Hash")
			}
			if chainConfig.EIP155Block.Cmp(gethparams.MainNetSpuriousDragon) != 0 {
				t.Fatal("invalid chainConfig.EIP155Block")
			}
			if chainConfig.EIP158Block.Cmp(gethparams.MainNetSpuriousDragon) != 0 {
				t.Fatal("invalid chainConfig.EIP158Block")
			}
			if chainConfig.ChainId.Cmp(gethparams.MainNetChainID) != 0 {
				t.Fatal("invalid chainConfig.ChainId")
			}
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
			//nodeConfig.LightEthConfig.Genesis = nodeConfig.LightEthConfig.Genesis[:125]
			//fmt.Println(nodeConfig)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			networkId := uint64(311)
			if nodeConfig.NetworkID != networkId {
				t.Fatalf("unexpected NetworkID, expected: %v, got: %v", networkId, nodeConfig.NetworkID)
			}
		},
	},
}

func TestLoadNodeConfig(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	for _, testCase := range loadConfigTestCases {
		t.Log("test: " + testCase.name)
		testCase.configJSON = strings.Replace(testCase.configJSON, "$TMPDIR", tmpDir, -1)
		nodeConfig, err := params.LoadNodeConfig(testCase.configJSON)
		testCase.validator(t, tmpDir, nodeConfig, err)
	}
}

func TestConfigWriteRead(t *testing.T) {
	configReadWrite := func(networkId uint64, refFile string) {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir) // nolint: errcheck

		nodeConfig, err := params.NewNodeConfig(tmpDir, networkId)
		if err != nil {
			t.Fatalf("cannot create new config object: %v", err)
		}

		if err := nodeConfig.Save(); err != nil {
			t.Fatalf("cannot persist configuration: %v", err)
		}

		loadedConfigData, err := ioutil.ReadFile(filepath.Join(nodeConfig.DataDir, "config.json"))
		if err != nil {
			t.Fatalf("cannot read configuration from disk: %v", err)
		}

		refConfigData := geth.LoadFromFile(refFile)

		refConfigData = strings.Replace(refConfigData, "$TMPDIR", nodeConfig.DataDir, -1)
		refConfigData = strings.Replace(refConfigData, "$VERSION", params.Version, -1)
		if string(loadedConfigData) != refConfigData {
			t.Fatalf("configuration mismatch,\nexpected: %v\ngot: %v", refConfigData, string(loadedConfigData))
		}
	}

	configReadWrite(params.TestNetworkID, "testdata/config.testnet.json")
	configReadWrite(params.MainNetworkID, "testdata/config.mainnet.json")
}
