package params_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/go-playground/validator.v9"

	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/require"
)

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
			require.False(t, nodeConfig.WSEnabled)
			require.Equal(t, 4242, nodeConfig.WSPort)
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
		`default boot cluster (Ropsten Dev)`,
		`{
			"NetworkId": 3,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			// Bootnodes for dev and prod modes are the same so no need for a separate Ropsten Prod test.

			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")
			require.Equal(t, "673363514b866bbbaf1b94462203650e3b1a2d4fb498df33361458b465b61ca3", nodeConfig.BootClusterConfig.RootHash)
			require.Equal(t, 395, nodeConfig.BootClusterConfig.RootNumber)

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
			require.Equal(t, expectedEnodes, enodes)
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
			require.NoError(t, err)
			require.False(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be disabled")
			require.Empty(t, nodeConfig.BootClusterConfig.RootHash)
			require.Empty(t, nodeConfig.BootClusterConfig.RootNumber)
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
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")
			require.Equal(t, "673363514b866bbbaf1b94462203650e3b1a2d4fb498df33361458b465b61ca3", nodeConfig.BootClusterConfig.RootHash)
			require.Equal(t, 395, nodeConfig.BootClusterConfig.RootNumber)

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
			require.Equal(t, expectedEnodes, enodes)
		},
	},
	{
		`select boot cluster (Rinkeby Dev)`,
		`{
			"NetworkId": 4,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")
			require.Equal(t, "f946d8f4d46d5b8a68006485cef0d1583db4a45feab83661323d4d52000aa2db", nodeConfig.BootClusterConfig.RootHash)
			require.Equal(t, 205, nodeConfig.BootClusterConfig.RootNumber)

			enodes := nodeConfig.BootClusterConfig.BootNodes
			expectedEnodes := []string{
				"enode://7512c8f6e7ffdcc723cf77e602a1de9d8cc2e8ad35db309464819122cd773857131aee390fec33894db13da730c8432bb248eed64039e3810e156e979b2847cb@51.15.78.243:30303",
				"enode://1cc27a5a41130a5c8b90db5b2273dc28f7b56f3edfc0dcc57b665d451274b26541e8de49ea7a074281906a82209b9600239c981163b6ff85c3038a8e2bc5d8b8@51.15.68.93:30303",
				"enode://798d17064141b8f88df718028a8272b943d1cb8e696b3dab56519c70b77b1d3469b56b6f4ce3788457646808f5c7299e9116626f2281f30b959527b969a71e4f@51.15.75.244:30303",
			}
			require.Equal(t, expectedEnodes, enodes)
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
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")
			require.Equal(t, "f946d8f4d46d5b8a68006485cef0d1583db4a45feab83661323d4d52000aa2db", nodeConfig.BootClusterConfig.RootHash)
			require.Equal(t, 205, nodeConfig.BootClusterConfig.RootNumber)

			enodes := nodeConfig.BootClusterConfig.BootNodes
			expectedEnodes := []string{
				"enode://fda3f6273a0f2da4ac5858d1f52e5afaf9def281121be3d37558c67d4d9ca26c6ad7a0520b2cd7454120fb770e86d5760487c9924b2166e65485f606e56d60fc@51.15.69.144:30303",
				"enode://ba41aa829287a0a9076d9bffed97c8ce2e491b99873288c9e886f16fd575306ac6c656db4fbf814f5a9021aec004ffa9c0ae8650f92fd10c12eeb7c364593eb3@51.15.69.147:30303",
				"enode://28ecf5272b560ca951f4cd7f1eb8bd62da5853b026b46db432c4b01797f5b0114819a090a72acd7f32685365ecd8e00450074fa0673039aefe10f3fb666e0f3f@51.15.76.249:30303",
			}
			require.Equal(t, expectedEnodes, enodes)
		},
	},
	{
		`select boot cluster (Homestead Dev)`,
		`{
			"NetworkId": 1,
			"DataDir": "$TMPDIR"
		}`,
		func(t *testing.T, dataDir string, nodeConfig *params.NodeConfig, err error) {
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")
			require.Equal(t, "85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f", nodeConfig.BootClusterConfig.RootHash)
			require.True(t, nodeConfig.BootClusterConfig.RootNumber >= 805)

			enodes := nodeConfig.BootClusterConfig.BootNodes
			expectedEnodes := []string{
				"enode://93833be81c3d1bdb2ae5cde258c8f82ad1011a1bea8eb49fe50b0af394d4f7f7e45974356870552f36744efd732692a64865d1e8b64114eaf89a1bad0a1903a2@51.15.64.29:30303",
				"enode://d76854bc54144b2269c5316d5f00f0a194efee2fb8d31e7b1939effd7e17f25773f8dc7fda8c4eb469450799da7f39b4e364e2a278d91b53539dcbb10b139635@51.15.73.37:30303",
				"enode://57874205931df976079e4ff8ebb5756461030fb00f73486bd5ec4ae6ed6ba98e27d09f58e59bd85281d24084a6062bc8ab514dbcdaa9678fc3001d47772e626e@51.15.75.213:30303",
			}
			require.Equal(t, expectedEnodes, enodes)
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
			require.NoError(t, err)
			require.True(t, nodeConfig.BootClusterConfig.Enabled, "boot cluster is expected to be enabled by default")
			require.Equal(t, "85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f", nodeConfig.BootClusterConfig.RootHash)
			require.True(t, nodeConfig.BootClusterConfig.RootNumber >= 805)

			enodes := nodeConfig.BootClusterConfig.BootNodes
			expectedEnodes := []string{
				"enode://f3b0e5dca730962bae814f3402b8f8a296644c33e8d7a95bd1ab313143a752c77076a03bcb76263570f2f34d4eb530f1daf5054c0990921a872a34eb505dcedf@51.15.73.129:30303",
				"enode://fce0d1c2292829b0eccce444f8943f88087ce00a5e910b157972ee1658a948d23c7a046f26567f73b2b18d126811509d7ef1de5be9b1decfcbb14738a590c477@51.15.75.187:30303",
				"enode://3b4b9fa02ae8d54c2db51a674bc93d85649b4775f22400f74ae25e9f1c665baa3bcdd33cadd2c1a93cd08a6af984cb605fbb61ec0d750a11d48d4080298af008@51.15.77.193:30303",
			}
			require.Equal(t, expectedEnodes, enodes)
		},
	},
	{
		`default DevMode (true)`,
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
		testCase.configJSON = strings.Replace(testCase.configJSON, "$TMPDIR", tmpDir, -1)
		nodeConfig, err := params.LoadNodeConfig(testCase.configJSON)
		testCase.validator(t, tmpDir, nodeConfig, err)
	}
}

func TestConfigWriteRead(t *testing.T) {
	configReadWrite := func(networkId uint64, refFile string) {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "geth-config-tests")
		require.Nil(t, err)
		defer os.RemoveAll(tmpDir) // nolint: errcheck

		nodeConfig, err := params.NewNodeConfig(tmpDir, networkId, true)
		require.Nil(t, err, "cannot create new config object")

		err = nodeConfig.Save()
		require.Nil(t, err, "cannot persist configuration")

		loadedConfigData, err := ioutil.ReadFile(filepath.Join(nodeConfig.DataDir, "config.json"))
		require.Nil(t, err, "cannot read configuration from disk")

		refConfigData := LoadFromFile(refFile)

		// Ease updating new config data.
		//ioutil.WriteFile(fmt.Sprintf("/tmp/chainId.%d.json", networkId), []byte(loadedConfigData), 0777)

		refConfigData = strings.Replace(refConfigData, "$TMPDIR", nodeConfig.DataDir, -1)
		refConfigData = strings.Replace(refConfigData, "$VERSION", params.Version, -1)

		require.EqualValues(t, refConfigData, loadedConfigData)
	}

	configReadWrite(params.RinkebyNetworkID, "testdata/config.rinkeby.json")
	configReadWrite(params.RopstenNetworkID, "testdata/config.ropsten.json")
	configReadWrite(params.MainNetworkID, "testdata/config.mainnet.json")
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
