package ethclient_test

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupDBTest(t *testing.T) (*ethclient.DBChain, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return ethclient.NewDBChain(ethclient.NewDB(db), 1), func() {
		require.NoError(t, db.Close())
	}
}

func TestPutBlock(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	blkJSON, blkNumber, blkHash := getTestBlockJSONWithoutTxDetails()
	err := db.PutBlockJSON(blkJSON, false)
	require.NoError(t, err)

	retrievedBlkJSON, err := db.GetBlockJSONByNumber(blkNumber, false)
	require.NoError(t, err)
	require.Equal(t, blkJSON, retrievedBlkJSON)

	retrievedBlkJSON, err = db.GetBlockJSONByHash(blkHash, false)
	require.NoError(t, err)
	require.Equal(t, blkJSON, retrievedBlkJSON)

	blkJSON, blkNumber, blkHash = getTestBlockJSONWithTxDetails()
	err = db.PutBlockJSON(blkJSON, true)
	require.NoError(t, err)

	retrievedBlkJSON, err = db.GetBlockJSONByNumber(blkNumber, true)
	require.NoError(t, err)
	require.Equal(t, blkJSON, retrievedBlkJSON)

	retrievedBlkJSON, err = db.GetBlockJSONByHash(blkHash, true)
	require.NoError(t, err)
	require.Equal(t, blkJSON, retrievedBlkJSON)
}

func TestPutBlockUncles(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	blkHash := common.HexToHash("0x1234")
	uncles := getTestBlockUnclesJSON()

	err := db.PutBlockUnclesJSON(blkHash, uncles)
	require.NoError(t, err)

	retrievedUncles, err := db.GetBlockUncleJSONByHashAndIndex(blkHash, 0)
	require.NoError(t, err)
	require.Equal(t, uncles[0], retrievedUncles)
}

func TestPutTransactions(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	txJSON, txHash := getTestTransactionJSON()
	err := db.PutTransactionsJSON([]json.RawMessage{txJSON})
	require.NoError(t, err)

	retrievedTxJSON, err := db.GetTransactionJSONByHash(txHash)
	require.NoError(t, err)
	require.Equal(t, txJSON, retrievedTxJSON)
}

func TestPutTransactionReceipts(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	receiptJSON, txHash := getTestReceiptJSON()
	err := db.PutTransactionReceiptsJSON([]json.RawMessage{receiptJSON})
	require.NoError(t, err)

	retrievedReceiptJSON, err := db.GetTransactionReceiptJSONByHash(txHash)
	require.NoError(t, err)
	require.Equal(t, receiptJSON, retrievedReceiptJSON)
}
