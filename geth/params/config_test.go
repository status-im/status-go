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
			"NetworkId": 311,
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

			enodes, err := nodeConfig.LoadBootClusterNodes()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expectedEnodes := []string{
				"enode://bc4a130219ae94c2a66e3ec3377c2a460e006d56b877d5e8edb0d0f8064cf400f117a53a5389f552c9e1a122b9a07eeaa41e7ed885268ee825b6a788188fb52e@51.15.55.219:30379",
				"enode://fbddff478e18292dc32b90f139bf773a08da89ffe29208e4de0091f6c589e60fccfaf16d4f4a76be49f57782c061ec8ea97078601c6f367feabda740f5ce8246@51.15.55.219:30303",
				"enode://df20352d07924ffe93e67ee7e81105d7b250f7932ff4e0351da2cfa17282e53f765895c32cda36b70dfb98def0b472b29d79321737035641bf3bcf595dcc1041@51.15.35.110:30379",
				"enode://4e5ee0487a4d8349ab9a9925b00eed0f976d98972c5a22f43fd50d1424897757032c36f273b434a4d3e013a2544eca74a9d1a0419f9f07f7bb43182a73df3690@51.15.35.110:30303",
				"enode://2cbcc7b3d067581072066143e0fade0d007e80ecc8f86eb475200c3da3a6d81dd4e1e7051fc3dfaee337f110ceec61594a901b09e36eb367629ddff6e1dfd955@51.15.34.3:30379",
				"enode://18efd9afb60443e00fed602cc0df526cd1d8543d2f6037df9380eb973d30b5fd04ac9f221053f82034581051bfd6e54356a99af2255f1a674d71d17440a6c95b@51.15.34.3:30303",
				"enode://cdb95f3d866472a74195342979ffea4ed7f9b68cd1e8c6f9a25b3197c221f01bc076ccba760341d8b69bb6bfbc9bf4fdeabd0caa99ee0bf4e79917fa1f42423c@51.15.56.154:30379",
				"enode://5b99c0cb372299fd3f2d94612a682990722eb7c3a252dacefc8270eb7f172fc699c1ddfad826fbfc979270538e8d89bd6919703eb9ef526eac0a45e9fb455123@51.15.56.154:30303",
				"enode://5ce8e96d9589671767a7b1c6b9a34bcf532587387eb062de712a9f716a66f05f412126121ce4d97330bc5dc7a4938ff1ecc22306b0b8b97a7f748c6f5f59c620@51.15.60.23:30379",
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
				"enode://6853f434735e540f0fcd85ffebcaa75280d1171ca9a205e8c41d87428d71b07ad14ab266236b64268467ccc462679edc888f76326418d18d7bcfe8d1159391aa@51.15.61.194:30379",
				"enode://1fa2dfe6b925ca753496ea197c973b66ef889ef4de2bf52acd5b8665c0cc2e8b95fbd192e764f10735f589297f1ae533f350f004e403063e8d4ad979aae44c12@51.15.61.194:30303",
				"enode://a8f1dae49f665c566734e002f89c1feb9b01e3ed09bdea6199aa6093f25085c4777fd553d2d1d14457286c24aaa48eaf6db99315e0caf62d97ea8bce801ae7c1@51.15.35.2:30379",
				"enode://44b91c043bcd96bc5279524f1bfe32df0670374135435ebacb29ba5d0e18192623e63ead711c9c363afdf2500fac423116ac28bdd2d700bd70e096326f95c63f@51.15.35.2:30303",
				"enode://64278f1e4224a5ff4608da54b7b045ae0b875a332c57e6f9b4cbb3e9ac1e56a1d5b91ff2def2662c767146b3f7f08924c15f66d41352a18ebe71832c35f6a0cf@51.15.54.229:30379",
				"enode://fb7622d3a50dc603f5c76919dd99c4112e5925cb891a67086b9dce581166fbdad361fd0bfb7ff128ab8f5e24e209e0b923668fbddb7e8b99edb82c1e3d782a80@51.15.54.229:30303",
				"enode://e1fcf9e7a47ab43a546d1c1633c511d98d13646bbf5c82d74ff98a1c88e54567b0be6574e977164e1b4c997ef8e79b19f1e12e85a6230c746dd74206fe37cfa0@51.15.35.70:30379",
				"enode://14c2960f57f6d63ed541cf64226aafbc7a21c40c6e4935a2e58fd2466fa7d06ec32082734c64d32f7c4692f4b90f26d019f472ba55cdda6d624ef4d7d8441285@51.15.35.70:30303",
				"enode://a8512bcaae1245fda71d400291dd22937d89947b6fc31283945557abe1281c5a9325ffc11e363cfed6362e4d2d9b941c5b325270662ba43ac8c424168e6567a6@51.15.39.57:30379",
				"enode://02cfa2b02b5431bfdc1bad0f575de8ea151029fe9a9c689074793d704d1b428255bd111bf578f0b4fcaa18267da7c335db9557e1012434f4a8ab2c25f4b3da4d@51.15.39.57:30303",
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
			expectedEnodes := []string{}
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
			expectedEnodes := []string{}
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
			expectedEnodes := []string{}
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
			expectedEnodes := []string{}
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
