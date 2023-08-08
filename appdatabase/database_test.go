package appdatabase

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/appdatabase/migrations"
	migrationsprevnodecfg "github.com/status-im/status-go/appdatabase/migrationsprevnodecfg"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/sqlite"
)

func Test_GetDBFilename(t *testing.T) {
	// Test with a temp file instance
	db, stop, err := SetupTestSQLDB("test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, stop())
	}()

	fn, err := GetDBFilename(db)
	require.NoError(t, err)
	require.True(t, len(fn) > 0)

	// Test with in memory instance
	mdb, err := InitializeDB(":memory:", "test", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, mdb.Close())
	}()

	fn, err = GetDBFilename(mdb)
	require.NoError(t, err)
	require.Equal(t, "", fn)
}

const (
	erc20ReceiptTestDataTemplate = `{"type":"0x2","root":"0x","status":"0x%d","cumulativeGasUsed":"0x10f8d2c","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000001008000000000000000000000000000000000000002000000000020000000000000000000800000000000000000000000010000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000800000000000000000000","logs":[{"address":"0x98339d8c260052b7ad81c28c16c0b98420f2b46a","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000000000000000000000000000000000000000000000","0x000000000000000000000000e2d622c817878da5143bbe06866ca8e35273ba8a"],"data":"0x0000000000000000000000000000000000000000000000000000000000989680","blockNumber":"0x825527","transactionHash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a","transactionIndex":"0x6c","blockHash":"0x69e0f829a557052c134cd7e21c220507d91bc35c316d3c47217e9bd362270274","logIndex":"0xcd","removed":false}],"transactionHash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x8623","blockHash":"0x69e0f829a557052c134cd7e21c220507d91bc35c316d3c47217e9bd362270274","blockNumber":"0x825527","transactionIndex":"0x6c"}`
	erc20TxTestData              = `{"type":"0x2","nonce":"0x3d","gasPrice":"0x0","maxPriorityFeePerGas":"0x8c347c90","maxFeePerGas":"0x45964d43a4","gas":"0x8623","value":"0x0","input":"0x40c10f19000000000000000000000000e2d622c817878da5143bbe06866ca8e35273ba8a0000000000000000000000000000000000000000000000000000000000989680","v":"0x0","r":"0xbcac4bb290d48b467bb18ac67e98050b5f316d2c66b2f75dcc1d63a45c905d21","s":"0x10c15517ea9cabd7fe134b270daabf5d2e8335e935d3e021f54a4efaffb37cd2","to":"0x98339d8c260052b7ad81c28c16c0b98420f2b46a","chainId":"0x5","accessList":[],"hash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a"}`

	erc20LogTestData   = `{"address":"0x98339d8c260052b7ad81c28c16c0b98420f2b46a","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000000000000000000000000000000000000000000000","0x000000000000000000000000e2d622c817878da5143bbe06866ca8e35273ba8a"],"data":"0x0000000000000000000000000000000000000000000000000000000000989680","blockNumber":"0x825527","transactionHash":"0xdcaa0fc7fe2e0d1f1343d1f36807344bb4fd26cda62ad8f9d8700e2c458cc79a","transactionIndex":"0x6c","blockHash":"0x69e0f829a557052c134cd7e21c220507d91bc35c316d3c47217e9bd362270274","logIndex":"0xcd","removed":false}`
	ethReceiptTestData = `{
		"type": "0x2",
		"root": "0x",
		"status": "0x1",
		"cumulativeGasUsed": "0x2b461",
		"logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"logs": [],
		"transactionHash": "0x4ac700ee2a1702f82b3cfdc88fd4d91f767b87fea9b929bd6223c6471a5e05b4",
		"contractAddress": "0x0000000000000000000000000000000000000000",
		"gasUsed": "0x5208",
		"blockHash": "0x25fe164361c1cb4ed1b46996f7b5236d3118144529b31fca037fcda1d8ee684d",
		"blockNumber": "0x5e3294",
		"transactionIndex": "0x3"
	}`
	ethTxTestData = `{
		"type": "0x2",
		"nonce": "0x1",
		"gasPrice": "0x0",
		"maxPriorityFeePerGas": "0x33",
		"maxFeePerGas": "0x3b9aca00",
		"gas": "0x55f0",
		"value": "0x%s",
		"input": "0x",
		"v": "0x0",
		"r": "0xacc277ce156382d6f333cc8d75a56250778b17f1c6d1676af63cf68d53713986",
		"s": "0x32417261484e9796390abb8db13f993965d917836be5cd96df25b9b581de91ec",
		"to": "0xbd54a96c0ae19a220c8e1234f54c940dfab34639",
		"chainId": "0x1a4",
		"accessList": [],
		"hash": "0x4ac700ee2a1702f82b3cfdc88fd4d91f767b87fea9b929bd6223c6471a5e05b4"
	}`

	erc721TxTestData      = `{"type":"0x2","nonce":"0x2f","gasPrice":"0x0","maxPriorityFeePerGas":"0x3b9aca00","maxFeePerGas":"0x2f691e609","gas":"0x1abc3","value":"0x0","input":"0x42842e0e000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c630000000000000000000000001c9751e0fbf5081849b56b522d50fb7f163b8080000000000000000000000000000000000000000000000000000000003ba7b95e360c6ebe","v":"0x1","r":"0xead469c32ffda3aa933f9aed814df411fb07893153c775b50596660036bbb5da","s":"0x73edadd4e4a7f0895f686b68e16101d195c0bb1b5f248f16b21557800b95bdf8","to":"0x85f0e02cb992aa1f9f47112f815f519ef1a59e2d","chainId":"0x1","accessList":[],"hash":"0x1dd936499e35ece8747bc481e476ac43eb4555a3a82e8cb93b7e429219bdd371"}`
	erc721ReceiptTestData = `{"type":"0x2","root":"0x","status":"0x1","cumulativeGasUsed":"0x54cadb","logsBloom":"0x00000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200200000000000000000000000000008000000000000080000000000000000000000002000200000020000000200000000000800000000000000000000000010000000000000000000000040000000000000000000000000000000000000000000000000020000000000000000000040010000000000000000000000000000000800000000000002020000000000000000000000000000000000000000000000000020000010000000000000000000004000000000000000000000000000000000000000","logs":[{"address":"0x85f0e02cb992aa1f9f47112f815f519ef1a59e2d","topics":["0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63","0x0000000000000000000000000000000000000000000000000000000000000000","0x000000000000000000000000000000000000000000000000000000003ba7b95e"],"data":"0x","blockNumber":"0xf57974","transactionHash":"0x1dd936499e35ece8747bc481e476ac43eb4555a3a82e8cb93b7e429219bdd371","transactionIndex":"0x44","blockHash":"0x9228724ff5c19f9b1586e19b13102f94798d1ee32b5f14d5cbcdf74cc32eb732","logIndex":"0x86","removed":false},{"address":"0x85f0e02cb992aa1f9f47112f815f519ef1a59e2d","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63","0x0000000000000000000000001c9751e0fbf5081849b56b522d50fb7f163b8080","0x000000000000000000000000000000000000000000000000000000003ba7b95e"],"data":"0x","blockNumber":"0xf57974","transactionHash":"0x1dd936499e35ece8747bc481e476ac43eb4555a3a82e8cb93b7e429219bdd371","transactionIndex":"0x44","blockHash":"0x9228724ff5c19f9b1586e19b13102f94798d1ee32b5f14d5cbcdf74cc32eb732","logIndex":"0x87","removed":false}],"transactionHash":"0x1dd936499e35ece8747bc481e476ac43eb4555a3a82e8cb93b7e429219bdd371","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x18643","blockHash":"0x9228724ff5c19f9b1586e19b13102f94798d1ee32b5f14d5cbcdf74cc32eb732","blockNumber":"0xf57974","transactionIndex":"0x44"}`
	erc721LogTestData     = `{"address":"0x85f0e02cb992aa1f9f47112f815f519ef1a59e2d","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63","0x0000000000000000000000001c9751e0fbf5081849b56b522d50fb7f163b8080","0x000000000000000000000000000000000000000000000000000000003ba7b95e"],"data":"0x","blockNumber":"0xf57974","transactionHash":"0x1dd936499e35ece8747bc481e476ac43eb4555a3a82e8cb93b7e429219bdd371","transactionIndex":"0x44","blockHash":"0x9228724ff5c19f9b1586e19b13102f94798d1ee32b5f14d5cbcdf74cc32eb732","logIndex":"0x87","removed":false}`

	uniswapV2TxTestData      = `{"type":"0x2","nonce":"0x42","gasPrice":"0x0","maxPriorityFeePerGas":"0x3b9aca00","maxFeePerGas":"0x13c6f691f2","gas":"0x2ed0d","value":"0xa688906bd8b0000","input":"0x3593564c000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000006440875700000000000000000000000000000000000000000000000000000000000000020b080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000a688906bd8b0000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000a688906bd8b000000000000000000000000000000000000000000001188be846e642b0ae4ae055e00000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc20000000000000000000000006982508145454ce325ddbe47a25d4ec3d2311933","v":"0x1","r":"0xeb7b527c2bfd3d26ea8e21951f537f4603867a11532081ba77fde9465696c20a","s":"0x5c120e64973a3b83a80d8b045a2228b9d1421065c1d480d2c1e322dad3b76c0f","to":"0xef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","chainId":"0x1","accessList":[],"hash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa"}`
	uniswapV2ReceiptTestData = `{"type":"0x2","root":"0x","status":"0x1","cumulativeGasUsed":"0x1ba15e","logsBloom":"0x00200000000000000000000080400000000000000000000000000000000000000000000000000000000000000000000002000000080000000000000200000000000000080000000000000008000000200000000000000000000000008000000000000000000000000000000000000000000000000000000000000010000000000000000000008000000000000000040000000001000000080000004200000000000800000000000000000000008000000000000000000000000000000800000001000012000000000000000000000000400000000000001000000000000000000000200001000000020000000000000000000000000000400000000080000000","logs":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b"],"data":"0x0000000000000000000000000000000000000000000000000a688906bd8b0000","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x2b","removed":false},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b"],"data":"0x0000000000000000000000000000000000000000000000000a688906bd8b0000","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x2c","removed":false},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","0x000000000000000000000000a43fe16908251ee70ef74718545e4fe6c5ccec9f"],"data":"0x0000000000000000000000000000000000000000000000000a688906bd8b0000","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x2d","removed":false},{"address":"0x6982508145454ce325ddbe47a25d4ec3d2311933","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000a43fe16908251ee70ef74718545e4fe6c5ccec9f","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63"],"data":"0x000000000000000000000000000000000000000011b2e784030a3a65a3559087","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x2e","removed":false},{"address":"0xa43fe16908251ee70ef74718545e4fe6c5ccec9f","topics":["0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"],"data":"0x000000000000000000000000000000000000003bdd991fe0c766723fa956e323000000000000000000000000000000000000000000000023240d303bb8bbb575","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x2f","removed":false},{"address":"0xa43fe16908251ee70ef74718545e4fe6c5ccec9f","topics":["0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63"],"data":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a688906bd8b0000000000000000000000000000000000000000000011b2e784030a3a65a35590870000000000000000000000000000000000000000000000000000000000000000","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x30","removed":false}],"transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x1ec85","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","blockNumber":"0x104ae90","transactionIndex":"0x4"}`
	uniswapV2LogTestData     = `{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b"],"data":"0x0000000000000000000000000000000000000000000000000a688906bd8b0000","blockNumber":"0x104ae90","transactionHash":"0x6d70a0b14e2fe1ba28d6cb910ffc4aa787264dff6c273e20509136461ac587aa","transactionIndex":"0x4","blockHash":"0x49e3ef5a17eb5563b327fffdf315dd9269c5a5676eec1f5c15897c4ef61623df","logIndex":"0x2b","removed":false}`

	uniswapV3TxTestData      = `{"type":"0x2","nonce":"0x41","gasPrice":"0x0","maxPriorityFeePerGas":"0x3b9aca00","maxFeePerGas":"0x92abb2610","gas":"0x34389","value":"0x1f161421c8e0000","input":"0x3593564c000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000643e278300000000000000000000000000000000000000000000000000000000000000020b000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000001f161421c8e00000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000001f161421c8e000000000000000000000000000000000000000000000002488dd50cfbb0a2a15abb00000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002bc02aaa39b223fe8d0a0e5c4f27ead9083c756cc20027105026f006b85729a8b14553fae6af249ad16c9aab000000000000000000000000000000000000000000","v":"0x1","r":"0x4fca68a439e7f841bdbe6d108bebd3d4c405f739cae203e61422152c4a0a057c","s":"0x597bd6d3848d31357207df1f92df77580310b7ad31f178575f0dae7f36934b39","to":"0xef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","chainId":"0x1","accessList":[],"hash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b"}`
	uniswapV3ReceiptTestData = `{"type":"0x2","root":"0x","status":"0x1","cumulativeGasUsed":"0x6c1b8a","logsBloom":"0x00000000000000000000000000400000000000000000000200000000000000000000000000000000000000000000000002000000080020000000000200000000000004000000000800000028000000000000000000000000240000008000000000000000000000800000000000000000000000000000000000000010000800000000000000000000000000080000000000000001000000000000000000000000000800000000000000000000000000000000000000000000000000000800002000000002000000000000000000000000400000000000000000000000000000000000200000000000000000000000000000000400000000400000000080000000","logs":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b"],"data":"0x00000000000000000000000000000000000000000000000001f161421c8e0000","blockNumber":"0x1047cc4","transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","transactionIndex":"0x4a","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","logIndex":"0xd8","removed":false},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b"],"data":"0x00000000000000000000000000000000000000000000000001f161421c8e0000","blockNumber":"0x1047cc4","transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","transactionIndex":"0x4a","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","logIndex":"0xd9","removed":false},{"address":"0x5026f006b85729a8b14553fae6af249ad16c9aab","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000007316f8dd242974f0fd7b16dbcc68920b96bc4db1","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63"],"data":"0x000000000000000000000000000000000000000000024cc783fc216d1e77f90d","blockNumber":"0x1047cc4","transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","transactionIndex":"0x4a","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","logIndex":"0xda","removed":false},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","0x0000000000000000000000007316f8dd242974f0fd7b16dbcc68920b96bc4db1"],"data":"0x00000000000000000000000000000000000000000000000001f161421c8e0000","blockNumber":"0x1047cc4","transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","transactionIndex":"0x4a","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","logIndex":"0xdb","removed":false},{"address":"0x7316f8dd242974f0fd7b16dbcc68920b96bc4db1","topics":["0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b","0x000000000000000000000000165eeecc32dcb623f51fc6c1ddd9e2aea1575c63"],"data":"0xfffffffffffffffffffffffffffffffffffffffffffdb3387c03de92e18806f300000000000000000000000000000000000000000000000001f161421c8e00000000000000000000000000000000000000000000000ea9ed3658a1ccb7e6d1cc000000000000000000000000000000000000000000001e5ab304463cab4cd155fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd6f54","blockNumber":"0x1047cc4","transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","transactionIndex":"0x4a","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","logIndex":"0xdc","removed":false}],"transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x22478","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","blockNumber":"0x1047cc4","transactionIndex":"0x4a"}`
	uniswapV3LogTestData     = `{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","topics":["0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c","0x000000000000000000000000ef1c6e67703c7bd7107eed8303fbe6ec2554bf6b"],"data":"0x00000000000000000000000000000000000000000000000001f161421c8e0000","blockNumber":"0x1047cc4","transactionHash":"0x5c5bca1291d1f09c07a9b66e56e78cc23da41b3e69e330dcd46a71ef6176df8b","transactionIndex":"0x4a","blockHash":"0x95c685d5165471e878aea2aaaa719bf4357cdbcd22722df4338e3e54f4e6c5d5","logIndex":"0xd8","removed":false}`
)

func TestMigrateWalletJsonBlobs(t *testing.T) {
	openDB := func() (*sql.DB, error) {
		return sqlite.OpenDB(sqlite.InMemoryPath, "1234567890", sqlite.ReducedKDFIterationsNumber)
	}
	db, err := openDB()
	require.NoError(t, err)

	// Execute the old migrations
	err = migrationsprevnodecfg.Migrate(db)
	require.NoError(t, err)

	err = nodecfg.MigrateNodeConfig(db)
	require.NoError(t, err)

	// Migrate until 1682393575_sync_ens_name.up
	err = migrations.MigrateTo(db, customSteps, 1682393575)
	require.NoError(t, err)

	// Validate that transfers table has no status column
	exists, err := ColumnExists(db, "transfers", "status")
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = ColumnExists(db, "transfers", "status")
	require.NoError(t, err)
	require.False(t, exists)

	insertTestTransaction := func(index int, txBlob string, receiptBlob string, logBlob string, ethType bool) error {
		indexStr := strconv.Itoa(index)
		senderStr := strconv.Itoa(index + 1)

		var txValue *string
		if txBlob != "" {
			txValue = &txBlob
		}
		var receiptValue *string
		if receiptBlob != "" {
			receiptValue = &receiptBlob
		}
		var logValue *string
		if logBlob != "" {
			logValue = &logBlob
		}
		entryType := "eth"
		if !ethType {
			entryType = "erc20"
		}
		_, err = db.Exec(`INSERT OR IGNORE INTO blocks(network_id, address, blk_number, blk_hash) VALUES (?, ?, ?, ?);
			INSERT INTO transfers (hash, address, sender, network_id, tx, receipt, log, blk_hash, type,  blk_number, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			index, common.HexToAddress(indexStr), index, common.HexToHash(indexStr),
			common.HexToHash(indexStr), common.HexToAddress(indexStr), common.HexToAddress(senderStr), index, txValue, receiptValue, logValue, common.HexToHash(indexStr), entryType, index, index)
		return err
	}

	// Empty transaction, found the usecase in the test DB
	err = insertTestTransaction(1, "", "", "", true)
	require.NoError(t, err)

	erc20FailReceiptJSON := fmt.Sprintf(erc20ReceiptTestDataTemplate, 0)
	erc20SuccessReceiptJSON := fmt.Sprintf(erc20ReceiptTestDataTemplate, 1)
	err = insertTestTransaction(2, erc20TxTestData, erc20FailReceiptJSON, erc20LogTestData, false)
	require.NoError(t, err)

	err = insertTestTransaction(3, erc20TxTestData, erc20SuccessReceiptJSON, erc20LogTestData, false)
	require.NoError(t, err)

	err = insertTestTransaction(4, erc721TxTestData, erc721ReceiptTestData, erc721LogTestData, false)
	require.NoError(t, err)

	ethZeroValueTxTestData := fmt.Sprintf(ethTxTestData, "0")
	ethVeryBigValueTxTestData := fmt.Sprintf(ethTxTestData, "12345678901234567890")
	ethOriginalTxTestData := fmt.Sprintf(ethTxTestData, "2386f26fc10000")

	err = insertTestTransaction(5, ethZeroValueTxTestData, ethReceiptTestData, "", true)
	require.NoError(t, err)
	err = insertTestTransaction(6, ethVeryBigValueTxTestData, "", "", true)
	require.NoError(t, err)
	err = insertTestTransaction(7, ethOriginalTxTestData, ethReceiptTestData, "", true)
	require.NoError(t, err)

	err = insertTestTransaction(8, uniswapV2TxTestData, uniswapV2ReceiptTestData, uniswapV2LogTestData, false)
	require.NoError(t, err)

	err = insertTestTransaction(9, uniswapV3TxTestData, uniswapV3ReceiptTestData, uniswapV3LogTestData, false)
	require.NoError(t, err)

	failMigrationSteps := []*sqlite.PostStep{
		{
			Version: customSteps[1].Version,
			CustomMigration: func(sqlTx *sql.Tx) error {
				return errors.New("failed to run custom migration")
			},
			RollBackVersion: customSteps[1].RollBackVersion,
		},
	}

	// Attempt to run test migration 1686048341 and fail in custom step
	err = migrations.MigrateTo(db, failMigrationSteps, customSteps[1].Version)
	require.Error(t, err)

	exists, err = ColumnExists(db, "transfers", "status")
	require.NoError(t, err)
	require.False(t, exists)

	// Run test migration 1686048341_transfers_receipt_json_blob_out.<up/down>.sql
	err = migrations.MigrateTo(db, customSteps, customSteps[2].Version)
	require.NoError(t, err)

	// Validate that the migration was run and transfers table has now status column
	exists, err = ColumnExists(db, "transfers", "status")
	require.NoError(t, err)
	require.True(t, exists)

	// Run test migration 1687193315.<up/down>.sql
	err = migrations.MigrateTo(db, customSteps, customSteps[1].Version)
	require.NoError(t, err)

	// Validate that the migration was run and transfers table has now txFrom column
	exists, err = ColumnExists(db, "transfers", "tx_from_address")
	require.NoError(t, err)
	require.True(t, exists)

	var (
		status, receiptType, cumulativeGasUsed, gasUsed, txIndex sql.NullInt64
		gasLimit, gasPriceClamped64, gasTipCapClamped64          sql.NullInt64
		gasFeeCapClamped64, accountNonce, size, logIndex, txType sql.NullInt64

		protected                     sql.NullBool
		amount128Hex                  sql.NullString
		contractAddress, tokenAddress *common.Address
		txFrom, txTo                  *common.Address
		txHash, blockHash             []byte
		entryType                     string
		isTokenIDNull                 bool
	)

	tokenID := new(big.Int)
	rows, err := db.Query(`SELECT status, receipt_type, tx_hash, log_index, block_hash, cumulative_gas_used, contract_address, gas_used, tx_index,
		tx_type, protected, gas_limit, gas_price_clamped64, gas_tip_cap_clamped64, gas_fee_cap_clamped64, amount_padded128hex, account_nonce, size, token_address, token_id, type,
		tx_from_address, tx_to_address,

		CASE
			WHEN token_id IS NULL THEN 1
			ELSE 0
		END as token_id_status

		FROM transfers ORDER BY timestamp ASC`)
	require.NoError(t, err)

	scanNextData := func() error {
		rows.Next()
		if rows.Err() != nil {
			return rows.Err()
		}
		err := rows.Scan(&status, &receiptType, &txHash, &logIndex, &blockHash, &cumulativeGasUsed, &contractAddress, &gasUsed, &txIndex,
			&txType, &protected, &gasLimit, &gasPriceClamped64, &gasTipCapClamped64, &gasFeeCapClamped64, &amount128Hex, &accountNonce, &size, &tokenAddress, (*bigint.SQLBigIntBytes)(tokenID), &entryType, &txFrom, &txTo, &isTokenIDNull)
		if err != nil {
			return err
		}
		return nil
	}

	validateTransaction := func(tt *types.Transaction, expectedEntryType w_common.Type, tl *types.Log) {
		if tt == nil {
			require.False(t, txType.Valid)
			require.False(t, protected.Valid)
			require.False(t, gasLimit.Valid)
			require.False(t, gasPriceClamped64.Valid)
			require.False(t, gasTipCapClamped64.Valid)
			require.False(t, gasFeeCapClamped64.Valid)
			require.False(t, amount128Hex.Valid)
			require.False(t, accountNonce.Valid)
			require.False(t, size.Valid)
			require.Empty(t, tokenAddress)
			require.True(t, isTokenIDNull)
			require.Equal(t, string(w_common.EthTransfer), entryType)
		} else {
			require.True(t, txType.Valid)
			require.Equal(t, tt.Type(), uint8(txType.Int64))
			require.True(t, protected.Valid)
			require.Equal(t, tt.Protected(), protected.Bool)
			require.True(t, gasLimit.Valid)
			require.Equal(t, tt.Gas(), uint64(gasLimit.Int64))
			require.True(t, gasPriceClamped64.Valid)
			require.Equal(t, *sqlite.BigIntToClampedInt64(tt.GasPrice()), gasPriceClamped64.Int64)
			require.True(t, gasTipCapClamped64.Valid)
			require.Equal(t, *sqlite.BigIntToClampedInt64(tt.GasTipCap()), gasTipCapClamped64.Int64)
			require.True(t, gasFeeCapClamped64.Valid)
			require.Equal(t, *sqlite.BigIntToClampedInt64(tt.GasFeeCap()), gasFeeCapClamped64.Int64)
			require.True(t, accountNonce.Valid)
			require.Equal(t, tt.Nonce(), uint64(accountNonce.Int64))
			require.True(t, size.Valid)
			require.Equal(t, int64(tt.Size()), size.Int64)

			if expectedEntryType == w_common.EthTransfer {
				require.True(t, amount128Hex.Valid)
				require.Equal(t, *sqlite.BigIntToPadded128BitsStr(tt.Value()), amount128Hex.String)
				require.True(t, isTokenIDNull)
			} else {
				actualEntryType, expectedTokenAddress, expectedTokenID, expectedValue, expectedFrom, expectedTo := w_common.ExtractTokenIdentity(expectedEntryType, tl, tt)
				if actualEntryType == w_common.Erc20Transfer {
					require.True(t, amount128Hex.Valid)
					require.Equal(t, *sqlite.BigIntToPadded128BitsStr(expectedValue), amount128Hex.String)
					require.True(t, isTokenIDNull)
					require.Equal(t, *expectedTokenAddress, *tokenAddress)
					require.Equal(t, *expectedFrom, *txFrom)
					require.Equal(t, *expectedTo, *txTo)
				} else if actualEntryType == w_common.Erc721Transfer {
					require.False(t, amount128Hex.Valid)
					require.False(t, isTokenIDNull)
					require.Equal(t, expectedTokenID, expectedTokenID)
					require.Equal(t, *expectedTokenAddress, *tokenAddress)
					require.Equal(t, *expectedFrom, *txFrom)
					require.Equal(t, *expectedTo, *txTo)
				} else {
					require.False(t, amount128Hex.Valid)
					require.True(t, isTokenIDNull)
					require.Empty(t, tokenAddress)
					require.Empty(t, txFrom)
					require.Empty(t, txTo)
				}

				require.Equal(t, expectedEntryType, actualEntryType)
			}
		}
	}

	validateReceipt := func(tr *types.Receipt, tl *types.Log) {
		if tr == nil {
			require.False(t, status.Valid)
			require.False(t, receiptType.Valid)
			require.Equal(t, []byte(nil), txHash)
			require.Equal(t, []byte(nil), blockHash)
			require.False(t, cumulativeGasUsed.Valid)
			require.Empty(t, contractAddress)
			require.False(t, gasUsed.Valid)
			require.False(t, txIndex.Valid)
		} else {
			require.True(t, status.Valid)
			require.Equal(t, tr.Status, uint64(status.Int64))
			require.True(t, receiptType.Valid)
			require.Equal(t, int64(tr.Type), receiptType.Int64)
			require.Equal(t, tr.TxHash, common.BytesToHash(txHash))
			require.Equal(t, tr.BlockHash, common.BytesToHash(blockHash))
			require.True(t, cumulativeGasUsed.Valid)
			require.Equal(t, int64(tr.CumulativeGasUsed), cumulativeGasUsed.Int64)
			require.Equal(t, tr.ContractAddress, *contractAddress)
			require.True(t, gasUsed.Valid)
			require.Equal(t, int64(tr.GasUsed), gasUsed.Int64)
			require.True(t, txIndex.Valid)
			require.Equal(t, int64(tr.TransactionIndex), txIndex.Int64)
		}
		if tl == nil {
			require.False(t, logIndex.Valid)
		} else {
			require.True(t, logIndex.Valid)
			require.Equal(t, uint(logIndex.Int64), tl.Index)
		}
	}

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(nil, w_common.EthTransfer, nil)
	validateReceipt(nil, nil)

	var successReceipt types.Receipt
	err = json.Unmarshal([]byte(erc20SuccessReceiptJSON), &successReceipt)
	require.NoError(t, err)

	var failReceipt types.Receipt
	err = json.Unmarshal([]byte(erc20FailReceiptJSON), &failReceipt)
	require.NoError(t, err)

	var erc20Log types.Log
	err = json.Unmarshal([]byte(erc20LogTestData), &erc20Log)
	require.NoError(t, err)

	var erc20Tx types.Transaction
	err = json.Unmarshal([]byte(erc20TxTestData), &erc20Tx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&erc20Tx, w_common.Erc20Transfer, &erc20Log)
	validateReceipt(&failReceipt, &erc20Log)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&erc20Tx, w_common.Erc20Transfer, &erc20Log)
	validateReceipt(&successReceipt, &erc20Log)

	var erc721Receipt types.Receipt
	err = json.Unmarshal([]byte(erc721ReceiptTestData), &erc721Receipt)
	require.NoError(t, err)

	var erc721Log types.Log
	err = json.Unmarshal([]byte(erc721LogTestData), &erc721Log)
	require.NoError(t, err)

	var erc721Tx types.Transaction
	err = json.Unmarshal([]byte(erc721TxTestData), &erc721Tx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&erc721Tx, w_common.Erc721Transfer, &erc721Log)
	validateReceipt(&erc721Receipt, &erc721Log)

	var zeroTestTx types.Transaction
	err = json.Unmarshal([]byte(ethZeroValueTxTestData), &zeroTestTx)
	require.NoError(t, err)

	var ethReceipt types.Receipt
	err = json.Unmarshal([]byte(ethReceiptTestData), &ethReceipt)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&zeroTestTx, w_common.EthTransfer, nil)
	validateReceipt(&ethReceipt, nil)

	var bigTestTx types.Transaction
	err = json.Unmarshal([]byte(ethVeryBigValueTxTestData), &bigTestTx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&bigTestTx, w_common.EthTransfer, nil)
	validateReceipt(nil, nil)

	var ethOriginalTestTx types.Transaction
	err = json.Unmarshal([]byte(ethOriginalTxTestData), &ethOriginalTestTx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&ethOriginalTestTx, w_common.EthTransfer, nil)
	validateReceipt(&ethReceipt, nil)

	var uniswapV2Receipt types.Receipt
	err = json.Unmarshal([]byte(uniswapV2ReceiptTestData), &uniswapV2Receipt)
	require.NoError(t, err)

	var uniswapV2Log types.Log
	err = json.Unmarshal([]byte(uniswapV2LogTestData), &uniswapV2Log)
	require.NoError(t, err)

	var uniswapV2Tx types.Transaction
	err = json.Unmarshal([]byte(uniswapV2TxTestData), &uniswapV2Tx)
	require.NoError(t, err)

	var uniswapV3Receipt types.Receipt
	err = json.Unmarshal([]byte(uniswapV3ReceiptTestData), &uniswapV3Receipt)
	require.NoError(t, err)

	var uniswapV3Log types.Log
	err = json.Unmarshal([]byte(uniswapV3LogTestData), &uniswapV3Log)
	require.NoError(t, err)

	var uniswapV3Tx types.Transaction
	err = json.Unmarshal([]byte(uniswapV3TxTestData), &uniswapV3Tx)
	require.NoError(t, err)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&uniswapV2Tx, w_common.UniswapV2Swap, &uniswapV2Log)
	validateReceipt(&uniswapV2Receipt, &uniswapV2Log)

	err = scanNextData()
	require.NoError(t, err)
	validateTransaction(&uniswapV3Tx, w_common.UniswapV3Swap, &uniswapV3Log)
	validateReceipt(&uniswapV3Receipt, &uniswapV3Log)

	err = scanNextData()
	// Validate that we processed all data (no more rows expected)
	require.Error(t, err)

	db.Close()
}
