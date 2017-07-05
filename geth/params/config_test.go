package params_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
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
			"HTTPEnabledMode": false,
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
			"HTTPEnabledMode": false,
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
			"HTTPEnabledMode": false,
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
			"HTTPEnabledMode": false,
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			networkId := uint64(311)
			if nodeConfig.NetworkID != networkId {
				t.Fatalf("unexpected NetworkID, expected: %v, got: %v", networkId, nodeConfig.NetworkID)
			}
		},
	},
	{
		`default boot cluster (Ropsten Dev)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.ConfigFile != params.BootClusterConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					params.BootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://da3bf389a031f33fb55c9f5f54fde8473912402d27fffaa50efd74c0d0515f3a61daf6d52151f2876b19c15828e6f670352bff432b5ec457652e74755e8c864f@51.15.62.116:30303",
				"enode://584c0db89b00719e9e7b1b5c32a4a8942f379f4d5d66bb69f9c7fa97fa42f64974e7b057b35eb5a63fd7973af063f9a1d32d8c60dbb4854c64cb8ab385470258@51.15.35.2:30303",
				"enode://e71ba996b923e4756783375e0ed1f29963b7f759305926315d3cf9a71364d2980dbc86b1c1549812c720b38f60d347149f1ba33681c5cd4c8fca4ee8116efec6@51.15.35.70:30303",
				"enode://829f09a946bcac67afbd7b8face82c48106af6a6b2507c007de6c79b2dbdf5544368afdf93cf5218d6d75a1f0219e6312db48bcc2f0fdfacb1d82c81f061d623@51.15.54.229:30303",
				"enode://e80276aabb7682a4a659f4341c1199de79d91a2e500a6ee9bed16ed4ce927ba8d32ba5dea357739ffdf2c5bcc848d3064bb6f149f0b4249c1f7e53f8bf02bfc8@51.15.39.57:30303",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`disabled boot cluster`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"Enabled": false
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("boot cluster is expected to be disabled")
			}
		},
	},
	{
		`select boot cluster (Ropsten Prod)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"ConfigFile": "ropsten.prod.json"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedConfigFile := "ropsten.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					expectedConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://fbddff478e18292dc32b90f139bf773a08da89ffe29208e4de0091f6c589e60fccfaf16d4f4a76be49f57782c061ec8ea97078601c6f367feabda740f5ce8246@51.15.55.219:30303",
				"enode://4e5ee0487a4d8349ab9a9925b00eed0f976d98972c5a22f43fd50d1424897757032c36f273b434a4d3e013a2544eca74a9d1a0419f9f07f7bb43182a73df3690@51.15.35.110:30303",
				"enode://18efd9afb60443e00fed602cc0df526cd1d8543d2f6037df9380eb973d30b5fd04ac9f221053f82034581051bfd6e54356a99af2255f1a674d71d17440a6c95b@51.15.34.3:30303",
				"enode://5b99c0cb372299fd3f2d94612a682990722eb7c3a252dacefc8270eb7f172fc699c1ddfad826fbfc979270538e8d89bd6919703eb9ef526eac0a45e9fb455123@51.15.56.154:30303",
				"enode://0e1d4d0fcfe888bf8a478b0fd89760a47733a5c04cd47de353295a6eb8dde8f54821b31196527d0c5c73a7024dc9ff34127692d237840fc09c312b3a19cd28fe@51.15.60.23:30303",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`select boot cluster (Rinkeby Dev)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"ConfigFile": "rinkeby.dev.json"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedConfigFile := "rinkeby.dev.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					expectedConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://7512c8f6e7ffdcc723cf77e602a1de9d8cc2e8ad35db309464819122cd773857131aee390fec33894db13da730c8432bb248eed64039e3810e156e979b2847cb@51.15.78.243:30303",
				"enode://1cc27a5a41130a5c8b90db5b2273dc28f7b56f3edfc0dcc57b665d451274b26541e8de49ea7a074281906a82209b9600239c981163b6ff85c3038a8e2bc5d8b8@51.15.68.93:30303",
				"enode://798d17064141b8f88df718028a8272b943d1cb8e696b3dab56519c70b77b1d3469b56b6f4ce3788457646808f5c7299e9116626f2281f30b959527b969a71e4f@51.15.75.244:30303",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`select boot cluster (Rinkeby Prod)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"ConfigFile": "rinkeby.prod.json"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedConfigFile := "rinkeby.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					expectedConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://fda3f6273a0f2da4ac5858d1f52e5afaf9def281121be3d37558c67d4d9ca26c6ad7a0520b2cd7454120fb770e86d5760487c9924b2166e65485f606e56d60fc@51.15.69.144:30303",
				"enode://ba41aa829287a0a9076d9bffed97c8ce2e491b99873288c9e886f16fd575306ac6c656db4fbf814f5a9021aec004ffa9c0ae8650f92fd10c12eeb7c364593eb3@51.15.69.147:30303",
				"enode://28ecf5272b560ca951f4cd7f1eb8bd62da5853b026b46db432c4b01797f5b0114819a090a72acd7f32685365ecd8e00450074fa0673039aefe10f3fb666e0f3f@51.15.76.249:30303",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`select boot cluster (Homestead Dev)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"ConfigFile": "homestead.dev.json"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedConfigFile := "homestead.dev.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					expectedConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://93833be81c3d1bdb2ae5cde258c8f82ad1011a1bea8eb49fe50b0af394d4f7f7e45974356870552f36744efd732692a64865d1e8b64114eaf89a1bad0a1903a2@51.15.64.29:30303",
				"enode://d76854bc54144b2269c5316d5f00f0a194efee2fb8d31e7b1939effd7e17f25773f8dc7fda8c4eb469450799da7f39b4e364e2a278d91b53539dcbb10b139635@51.15.73.37:30303",
				"enode://57874205931df976079e4ff8ebb5756461030fb00f73486bd5ec4ae6ed6ba98e27d09f58e59bd85281d24084a6062bc8ab514dbcdaa9678fc3001d47772e626e@51.15.75.213:30303",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`select boot cluster (Homestead Prod)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"ConfigFile": "homestead.prod.json"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedConfigFile := "homestead.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					expectedConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://f3b0e5dca730962bae814f3402b8f8a296644c33e8d7a95bd1ab313143a752c77076a03bcb76263570f2f34d4eb530f1daf5054c0990921a872a34eb505dcedf@51.15.73.129:30303",
				"enode://fce0d1c2292829b0eccce444f8943f88087ce00a5e910b157972ee1658a948d23c7a046f26567f73b2b18d126811509d7ef1de5be9b1decfcbb14738a590c477@51.15.75.187:30303",
				"enode://3b4b9fa02ae8d54c2db51a674bc93d85649b4775f22400f74ae25e9f1c665baa3bcdd33cadd2c1a93cd08a6af984cb605fbb61ec0d750a11d48d4080298af008@51.15.77.193:30303",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`select boot cluster (custom JSON, via absolute path)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR",
			"BootClusterConfig": {
				"ConfigFile": "$TMPDIR/bootstrap-cluster.json"
			}
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedConfigFile := filepath.Join(dataDir, "bootstrap-cluster.json")
			if nodeConfig.BootClusterConfig.ConfigFile != expectedConfigFile {
				t.Fatalf("unexpected BootClusterConfigFile, expected: %v, got: %v",
					expectedConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://foobar@41.41.41.41:30300",
				"enode://foobaz@42.42.42.42:30302",
			}
			if len(enodes) != len(expectedEnodes) {
				t.Fatalf("wrong number of enodes, expected: %d, got: %d", len(expectedEnodes), len(enodes))
			}
			if !reflect.DeepEqual(enodes, expectedEnodes) {
				t.Fatalf("wrong list of enodes, expected: \n%v,\n\ngot:\n%v", expectedEnodes, enodes)
			}
		},
	},
	{
		`default DevMode (true)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", true, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			if nodeConfig.BootClusterConfig.ConfigFile != params.BootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					params.BootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`explicit DevMode = false`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", false, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "ropsten.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`populate bootstrap config (Homestead/Dev)`,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR",
			"DevMode": true
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", true, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "homestead.dev.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`populate bootstrap config (Homestead/Prod)`,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", false, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "homestead.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`populate bootstrap config (Ropsten/Dev)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", true, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "ropsten.dev.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`populate bootstrap config (Ropsten/Prod)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", false, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "ropsten.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`populate bootstrap config (Rinkeby/Dev)`,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", true, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "rinkeby.dev.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
			}
		},
	},
	{
		`populate bootstrap config (Rinkeby/Prod)`,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.DevMode {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", false, nodeConfig.DevMode)
			}

			if !nodeConfig.BootClusterConfig.Enabled {
				t.Fatal("expected boot cluster to be enabled")
			}

			expectedBootClusterConfigFile := "rinkeby.prod.json"
			if nodeConfig.BootClusterConfig.ConfigFile != expectedBootClusterConfigFile {
				t.Fatalf("unexpected bootcluster config file, expected: %v, got: %v",
					expectedBootClusterConfigFile, nodeConfig.BootClusterConfig.ConfigFile)
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

	// create sample Bootstrap Cluster Config
	bootstrapConfig := []byte(`["enode://foobar@41.41.41.41:30300", "enode://foobaz@42.42.42.42:30302"]`)
	if err = ioutil.WriteFile(filepath.Join(tmpDir, "bootstrap-cluster.json"), bootstrapConfig, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	t.Log(tmpDir)

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

		nodeConfig, err := params.NewNodeConfig(tmpDir, networkId, true)
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

		refConfigData := LoadFromFile(refFile)

		refConfigData = strings.Replace(refConfigData, "$TMPDIR", nodeConfig.DataDir, -1)
		refConfigData = strings.Replace(refConfigData, "$VERSION", params.Version, -1)
		if string(loadedConfigData) != refConfigData {
			t.Fatalf("configuration mismatch,\nexpected: %v\ngot: %v", refConfigData, string(loadedConfigData))
		}
	}

	configReadWrite(params.RinkebyNetworkID, "testdata/config.rinkeby.json")
	configReadWrite(params.RopstenNetworkID, "testdata/config.ropsten.json")
	configReadWrite(params.MainNetworkID, "testdata/config.mainnet.json")
}
