package pathprocessor

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/types"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupTxInputDataDBTest(t *testing.T) (*TransactionInputDataDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewTransactionInputDataDB(db), func() {
		require.NoError(t, db.Close())
	}
}

type testTxInputData struct {
	ChainID   w_common.ChainID
	TxHash    types.Hash
	InputData TransactionInputData
}

func generateTestTxInputData(offset int, count int) []testTxInputData {
	ret := make([]testTxInputData, 0, count)
	for i := offset; i < offset+count; i++ {
		inputData := NewInputData()

		inputDataIdx := i % 3
		inputData.ProcessorName = fmt.Sprintf("processor_name_%d", inputDataIdx)

		fromAsset := fmt.Sprintf("from_asset_%d", inputDataIdx)
		fromAmount := new(big.Int).SetInt64(int64(inputDataIdx))
		toAsset := fmt.Sprintf("to_asset_%d", inputDataIdx)
		toAmount := new(big.Int).SetInt64(int64(inputDataIdx * 2))
		side := SwapSide(i % 2)
		slippageBps := uint16(i % 100)
		approvalAmount := new(big.Int).SetInt64(int64(inputDataIdx * 3))
		approvalSpender := common.HexToAddress(fmt.Sprintf("0x%d", inputDataIdx*4))

		switch inputDataIdx {
		case 0:
			inputData.FromAsset = &fromAsset
			inputData.FromAmount = (*hexutil.Big)(fromAmount)
		case 1:
			inputData.FromAsset = &fromAsset
			inputData.FromAmount = (*hexutil.Big)(fromAmount)
			inputData.ToAsset = &toAsset
			inputData.ToAmount = (*hexutil.Big)(toAmount)
			inputData.Side = &side
			inputData.SlippageBps = &slippageBps
		case 2:
			inputData.FromAsset = &fromAsset
			inputData.FromAmount = (*hexutil.Big)(fromAmount)
			inputData.ApprovalAmount = (*hexutil.Big)(approvalAmount)
			inputData.ApprovalSpender = &approvalSpender
		}
		testInputData := testTxInputData{
			ChainID:   w_common.ChainID(i % 3),
			TxHash:    types.HexToHash(uuid.New().String()),
			InputData: *inputData,
		}
		ret = append(ret, testInputData)
	}
	return ret
}

func TestUpsertTxInputData(t *testing.T) {
	iDB, cleanup := setupTxInputDataDBTest(t)
	defer cleanup()

	testData := generateTestTxInputData(0, 15)
	for _, data := range testData {
		err := iDB.UpsertInputData(data.ChainID, data.TxHash, data.InputData)
		require.NoError(t, err)
	}

	for _, data := range testData {
		readData, err := iDB.ReadInputData(data.ChainID, data.TxHash)
		require.NoError(t, err)
		require.NotNil(t, readData)
		require.Equal(t, data.InputData, *readData)
	}
}
