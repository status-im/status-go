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
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			if nodeConfig.BootClusterConfig.RootHash == "" {
				t.Fatal("empty CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootHash != "91825fffecb5678167273955deaddbf03c26ae04287cfda61403c0bad5ceab8d" {
				t.Fatal("invalid CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootNumber != 259 {
				t.Fatal("empty CHT number")
			}

			enodes := nodeConfig.BootClusterConfig.BootNodes
			expectedEnodes := []string{
				"enode://7ab298cedc4185a894d21d8a4615262ec6bdce66c9b6783878258e0d5b31013d30c9038932432f70e5b2b6a5cd323bf820554fcb22fbc7b45367889522e9c449@51.15.63.93:30303",
				"enode://f59e8701f18c79c5cbc7618dc7bb928d44dc2f5405c7d693dad97da2d8585975942ec6fd36d3fe608bfdc7270a34a4dd00f38cfe96b2baa24f7cd0ac28d382a1@51.15.79.88:30303",
				"enode://e2a3587b7b41acfc49eddea9229281905d252efba0baf565cf6276df17faf04801b7879eead757da8b5be13b05f25e775ab6d857ff264bc53a89c027a657dd10@51.15.45.114:30303",
				"enode://fe991752c4ceab8b90608fbf16d89a5f7d6d1825647d4981569ebcece1b243b2000420a5db721e214231c7a6da3543fa821185c706cbd9b9be651494ec97f56a@51.15.67.119:30303",
				"enode://482484b9198530ee2e00db89791823244ca41dcd372242e2e1297dd06f6d8dd357603960c5ad9cc8dc15fcdf0e4edd06b7ad7db590e67a0b54f798c26581ebd7@51.15.75.138:30303",
				"enode://9e99e183b5c71d51deb16e6b42ac9c26c75cfc95fff9dfae828b871b348354cbecf196dff4dd43567b26c8241b2b979cb4ea9f8dae2d9aacf86649dafe19a39a@51.15.79.176:30303",
				"enode://12d52c3796700fb5acff2c7d96df7bbb6d7109b67f3442ee3d99ac1c197016cddb4c3568bbeba05d39145c59c990cd64f76bc9b00d4b13f10095c49507dd4cf9@51.15.63.110:30303",
				"enode://0f7c65277f916ff4379fe520b875082a56e587eb3ce1c1567d9ff94206bdb05ba167c52272f20f634cd1ebdec5d9dfeb393018bfde1595d8e64a717c8b46692f@51.15.54.150:30303",
				"enode://e006f0b2dc98e757468b67173295519e9b6d5ff4842772acb18fd055c620727ab23766c95b8ee1008dea9e8ef61e83b1515ddb3fb56dbfb9dbf1f463552a7c9f@212.47.237.127:30303",
				"enode://d40871fc3e11b2649700978e06acd68a24af54e603d4333faecb70926ca7df93baa0b7bf4e927fcad9a7c1c07f9b325b22f6d1730e728314d0e4e6523e5cebc2@51.15.132.235:30303",
				"enode://ea37c9724762be7f668e15d3dc955562529ab4f01bd7951f0b3c1960b75ecba45e8c3bb3c8ebe6a7504d9a40dd99a562b13629cc8e5e12153451765f9a12a61d@163.172.189.205:30303",
				"enode://88c2b24429a6f7683fbfd06874ae3f1e7c8b4a5ffb846e77c705ba02e2543789d66fc032b6606a8d8888eb6239a2abe5897ce83f78dcdcfcb027d6ea69aa6fe9@163.172.157.61:30303",
				"enode://ce6854c2c77a8800fcc12600206c344b8053bb90ee3ba280e6c4f18f3141cdc5ee80bcc3bdb24cbc0e96dffd4b38d7b57546ed528c00af6cd604ab65c4d528f6@163.172.153.124:30303",
				"enode://00ae60771d9815daba35766d463a82a7b360b3a80e35ab2e0daa25bdc6ca6213ff4c8348025e7e1a908a8f58411a364fe02a0fb3c2aa32008304f063d8aaf1a2@163.172.132.85:30303",
				"enode://86ebc843aa51669e08e27400e435f957918e39dc540b021a2f3291ab776c88bbda3d97631639219b6e77e375ab7944222c47713bdeb3251b25779ce743a39d70@212.47.254.155:30303",
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

			if nodeConfig.BootClusterConfig.Enabled != false {
				t.Fatal("boot cluster is expected to be disabled")
			}

			if nodeConfig.BootClusterConfig.RootHash != "" {
				t.Fatal("empty CHT hash is expected")
			}

			if nodeConfig.BootClusterConfig.RootNumber != 0 {
				t.Fatal("empty CHT number is expected")
			}
		},
	},
	{
		`select boot cluster (Ropsten Prod)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			if nodeConfig.BootClusterConfig.RootHash == "" {
				t.Fatal("empty CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootHash != "91825fffecb5678167273955deaddbf03c26ae04287cfda61403c0bad5ceab8d" {
				t.Fatal("invalid CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootNumber != 259 {
				t.Fatal("empty CHT number")
			}

			enodes := nodeConfig.BootClusterConfig.BootNodes
			expectedEnodes := []string{
				"enode://7ab298cedc4185a894d21d8a4615262ec6bdce66c9b6783878258e0d5b31013d30c9038932432f70e5b2b6a5cd323bf820554fcb22fbc7b45367889522e9c449@51.15.63.93:30303",
				"enode://f59e8701f18c79c5cbc7618dc7bb928d44dc2f5405c7d693dad97da2d8585975942ec6fd36d3fe608bfdc7270a34a4dd00f38cfe96b2baa24f7cd0ac28d382a1@51.15.79.88:30303",
				"enode://e2a3587b7b41acfc49eddea9229281905d252efba0baf565cf6276df17faf04801b7879eead757da8b5be13b05f25e775ab6d857ff264bc53a89c027a657dd10@51.15.45.114:30303",
				"enode://fe991752c4ceab8b90608fbf16d89a5f7d6d1825647d4981569ebcece1b243b2000420a5db721e214231c7a6da3543fa821185c706cbd9b9be651494ec97f56a@51.15.67.119:30303",
				"enode://482484b9198530ee2e00db89791823244ca41dcd372242e2e1297dd06f6d8dd357603960c5ad9cc8dc15fcdf0e4edd06b7ad7db590e67a0b54f798c26581ebd7@51.15.75.138:30303",
				"enode://9e99e183b5c71d51deb16e6b42ac9c26c75cfc95fff9dfae828b871b348354cbecf196dff4dd43567b26c8241b2b979cb4ea9f8dae2d9aacf86649dafe19a39a@51.15.79.176:30303",
				"enode://12d52c3796700fb5acff2c7d96df7bbb6d7109b67f3442ee3d99ac1c197016cddb4c3568bbeba05d39145c59c990cd64f76bc9b00d4b13f10095c49507dd4cf9@51.15.63.110:30303",
				"enode://0f7c65277f916ff4379fe520b875082a56e587eb3ce1c1567d9ff94206bdb05ba167c52272f20f634cd1ebdec5d9dfeb393018bfde1595d8e64a717c8b46692f@51.15.54.150:30303",
				"enode://e006f0b2dc98e757468b67173295519e9b6d5ff4842772acb18fd055c620727ab23766c95b8ee1008dea9e8ef61e83b1515ddb3fb56dbfb9dbf1f463552a7c9f@212.47.237.127:30303",
				"enode://d40871fc3e11b2649700978e06acd68a24af54e603d4333faecb70926ca7df93baa0b7bf4e927fcad9a7c1c07f9b325b22f6d1730e728314d0e4e6523e5cebc2@51.15.132.235:30303",
				"enode://ea37c9724762be7f668e15d3dc955562529ab4f01bd7951f0b3c1960b75ecba45e8c3bb3c8ebe6a7504d9a40dd99a562b13629cc8e5e12153451765f9a12a61d@163.172.189.205:30303",
				"enode://88c2b24429a6f7683fbfd06874ae3f1e7c8b4a5ffb846e77c705ba02e2543789d66fc032b6606a8d8888eb6239a2abe5897ce83f78dcdcfcb027d6ea69aa6fe9@163.172.157.61:30303",
				"enode://ce6854c2c77a8800fcc12600206c344b8053bb90ee3ba280e6c4f18f3141cdc5ee80bcc3bdb24cbc0e96dffd4b38d7b57546ed528c00af6cd604ab65c4d528f6@163.172.153.124:30303",
				"enode://00ae60771d9815daba35766d463a82a7b360b3a80e35ab2e0daa25bdc6ca6213ff4c8348025e7e1a908a8f58411a364fe02a0fb3c2aa32008304f063d8aaf1a2@163.172.132.85:30303",
				"enode://86ebc843aa51669e08e27400e435f957918e39dc540b021a2f3291ab776c88bbda3d97631639219b6e77e375ab7944222c47713bdeb3251b25779ce743a39d70@212.47.254.155:30303",
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
			"NetworkId": 4,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			if nodeConfig.BootClusterConfig.RootHash == "" {
				t.Fatal("empty CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootHash != "rinkeby-dev" {
				t.Fatal("invalid CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootNumber < 66 {
				t.Fatal("empty CHT number")
			}

			enodes := nodeConfig.BootClusterConfig.BootNodes
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
			"NetworkId": 4,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			if nodeConfig.BootClusterConfig.RootHash == "" {
				t.Fatal("empty CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootHash != "rinkeby-prod" {
				t.Fatal("invalid CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootNumber < 66 {
				t.Fatal("empty CHT number")
			}

			enodes := nodeConfig.BootClusterConfig.BootNodes
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
			"NetworkId": 1,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			if nodeConfig.BootClusterConfig.RootHash == "" {
				t.Fatal("empty CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootHash != "85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f" {
				t.Fatal("invalid CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootNumber < 805 {
				t.Fatal("empty CHT number")
			}
			enodes := nodeConfig.BootClusterConfig.BootNodes
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
			"NetworkId": 1,
			"DataDir": "$TMPDIR",
			"DevMode": false
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("boot cluster is expected to be enabled by default")
			}

			if nodeConfig.BootClusterConfig.RootHash == "" {
				t.Fatal("empty CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootHash != "85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f" {
				t.Fatal("invalid CHT hash")
			}

			if nodeConfig.BootClusterConfig.RootNumber < 805 {
				t.Fatal("empty CHT number")
			}
			enodes := nodeConfig.BootClusterConfig.BootNodes
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
		`default DevMode (true)`,
		`{
			"NetworkId": 311,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nodeConfig.DevMode != true {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", true, nodeConfig.DevMode)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("expected boot cluster to be enabled")
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

			if nodeConfig.DevMode != false {
				t.Fatalf("unexpected dev mode: expected: %v, got: %v", false, nodeConfig.DevMode)
			}

			if nodeConfig.BootClusterConfig.Enabled != true {
				t.Fatal("expected boot cluster to be enabled")
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

		refConfigData := geth.LoadFromFile(refFile)

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
