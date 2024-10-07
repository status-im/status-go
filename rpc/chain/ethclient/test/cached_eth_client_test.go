package ethclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	ethclient "github.com/status-im/status-go/rpc/chain/ethclient"
	mock_ethclient "github.com/status-im/status-go/rpc/chain/ethclient/mock/client/ethclient"

	gomock "go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"
)

func setupCachedEthClientTest(t *testing.T) (*ethclient.CachedEthClient, *mock_ethclient.MockRPSLimitedEthClientInterface, func()) {
	db, dbCleanup := setupDBTest(t)

	mockCtrl := gomock.NewController(t)

	ethClient := mock_ethclient.NewMockRPSLimitedEthClientInterface(mockCtrl)
	ethClient.EXPECT().GetName().Return("test").AnyTimes()
	ethClient.EXPECT().GetLimiter().Return(nil).AnyTimes()

	cachedEthClient := ethclient.NewCachedEthClient(ethClient, db)

	cleanup := func() {
		dbCleanup()
		mockCtrl.Finish()
	}
	return cachedEthClient, ethClient, cleanup
}

func specialBlockNumbers() []*big.Int {
	return []*big.Int{
		nil,
		big.NewInt(int64(rpc.LatestBlockNumber)),
		big.NewInt(int64(rpc.PendingBlockNumber)),
		big.NewInt(int64(rpc.PendingBlockNumber)),
		big.NewInt(int64(rpc.SafeBlockNumber)),
	}
}

func TestGetBlock(t *testing.T) {
	client, ethClient, cleanup := setupCachedEthClientTest(t)
	defer cleanup()

	ctx := context.Background()

	blkJSON, blkNumber, blkHash := getTestBlockJSONWithTxDetails()

	// First call goes to the chain, through raw endpoint
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = blkJSON
			return nil
		}).Times(1)
	res, err := client.BlockByHash(ctx, blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, res.Hash())
	require.Equal(t, blkNumber, res.Number())

	// Next calls are read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.BlockByHash(ctx, blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, res.Hash())
	require.Equal(t, blkNumber, res.Number())

	// Call by number is also read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.BlockByNumber(ctx, blkNumber)
	require.NoError(t, err)
	require.Equal(t, blkHash, res.Hash())
	require.Equal(t, blkNumber, res.Number())

	// Fetching a different block goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, err = client.BlockByHash(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, err = client.BlockByHash(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)

	// Calls with non-concrete block numbers always go to chain
	for i := 0; i < 3; i++ {
		for _, blockNumber := range specialBlockNumbers() {
			newBlkJSON, _, _ := getTestBlockJSONWithTxDetails()
			ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
					*result.(*json.RawMessage) = newBlkJSON
					return nil
				}).Times(1)
			_, err = client.BlockByNumber(ctx, blockNumber)
			require.NoError(t, err)
		}
	}
}

func TestGetHeader(t *testing.T) {
	client, ethClient, cleanup := setupCachedEthClientTest(t)
	defer cleanup()

	ctx := context.Background()

	blkJSON, blkNumber, blkHash := getTestBlockJSONWithTxDetails()

	// First call goes to the chain, through raw endpoint
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = blkJSON
			return nil
		}).Times(1)
	res, err := client.HeaderByHash(ctx, blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, res.Hash())
	require.Equal(t, blkNumber, res.Number)

	// Next calls are read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.HeaderByHash(ctx, blkHash)
	require.NoError(t, err)
	require.Equal(t, blkHash, res.Hash())
	require.Equal(t, blkNumber, res.Number)

	// Call by number is also read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.HeaderByNumber(ctx, blkNumber)
	require.NoError(t, err)
	require.Equal(t, blkHash, res.Hash())
	require.Equal(t, blkNumber, res.Number)

	// Fetching a different block goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, err = client.BlockByHash(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, err = client.BlockByHash(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)

	// Calls with non-concrete block numbers always go to chain
	for i := 0; i < 3; i++ {
		for _, blockNumber := range specialBlockNumbers() {
			newBlkJSON, _, _ := getTestBlockJSONWithTxDetails()
			ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
					*result.(*json.RawMessage) = newBlkJSON
					return nil
				}).Times(1)
			_, err = client.HeaderByNumber(ctx, blockNumber)
			require.NoError(t, err)
		}
	}
}

func TestGetTransaction(t *testing.T) {
	client, ethClient, cleanup := setupCachedEthClientTest(t)
	defer cleanup()

	ctx := context.Background()

	txJSON, txHash := getTestTransactionJSON()

	// First call goes to the chain, through raw endpoint
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = txJSON
			return nil
		}).Times(1)
	res, _, err := client.TransactionByHash(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, txHash, res.Hash())

	// Next calls are read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, _, err = client.TransactionByHash(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, txHash, res.Hash())

	// Fetching a different tx goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, _, err = client.TransactionByHash(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, _, err = client.TransactionByHash(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)
}

func TestGetReceipt(t *testing.T) {
	client, ethClient, cleanup := setupCachedEthClientTest(t)
	defer cleanup()

	ctx := context.Background()

	rxJSON, txHash := getTestReceiptJSON()

	// First call goes to the chain, through raw endpoint
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = rxJSON
			return nil
		}).Times(1)
	res, err := client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, txHash, res.TxHash)

	// Next calls are read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.TransactionReceipt(ctx, txHash)
	require.NoError(t, err)
	require.Equal(t, txHash, res.TxHash)

	// Fetching a different tx goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, err = client.TransactionReceipt(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*json.RawMessage) = json.RawMessage("Invalid JSON")
			return nil
		}).Times(1)
	_, err = client.TransactionReceipt(ctx, common.HexToHash("0x1234"))
	require.Error(t, err)
}

func TestGetBalance(t *testing.T) {
	client, ethClient, cleanup := setupCachedEthClientTest(t)
	defer cleanup()

	ctx := context.Background()

	account := common.HexToAddress("0x1234")
	blockNumber := big.NewInt(1234)
	valueHex, valueInt := getTestBalance()

	// First call goes to the chain, through raw endpoint
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(**hexutil.Big) = valueHex
			return nil
		}).Times(1)
	res, err := client.BalanceAt(ctx, account, blockNumber)
	require.NoError(t, err)
	require.Equal(t, valueInt, res)

	// Next calls are read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.BalanceAt(ctx, account, blockNumber)
	require.NoError(t, err)
	require.Equal(t, valueInt, res)

	// Fetching a different account goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(**hexutil.Big) = nil
			return errors.New("Some error")
		}).Times(1)
	_, err = client.BalanceAt(ctx, common.HexToAddress("0x4567"), blockNumber)
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(**hexutil.Big) = nil
			return errors.New("Some error")
		}).Times(1)
	_, err = client.BalanceAt(ctx, common.HexToAddress("0x4567"), blockNumber)
	require.Error(t, err)

	// Fetching a different block goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(**hexutil.Big) = nil
			return errors.New("Some error")
		}).Times(1)
	_, err = client.BalanceAt(ctx, account, big.NewInt(5))
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(**hexutil.Big) = nil
			return errors.New("Some error")
		}).Times(1)
	_, err = client.BalanceAt(ctx, account, big.NewInt(5))
	require.Error(t, err)

	// Non-concrete block numbers always go to chain
	for i := 0; i < 3; i++ {
		newValueHex, newValueInt := getTestBalance()
		for _, blockNumber := range specialBlockNumbers() {
			ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
					*result.(**hexutil.Big) = newValueHex
					return nil
				}).Times(1)
			res, err := client.BalanceAt(ctx, account, blockNumber)
			require.NoError(t, err)
			require.Equal(t, newValueInt, res)
		}
	}
}

func TestGetTransactionCount(t *testing.T) {
	client, ethClient, cleanup := setupCachedEthClientTest(t)
	defer cleanup()

	ctx := context.Background()

	account := common.HexToAddress("0x1234")
	blockNumber := big.NewInt(1234)
	valueHex, valueInt := getTestTransactionCount()

	// First call goes to the chain, through raw endpoint
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*hexutil.Uint64) = valueHex
			return nil
		}).Times(1)
	res, err := client.NonceAt(ctx, account, blockNumber)
	require.NoError(t, err)
	require.Equal(t, valueInt, res)

	// Next calls are read from cache
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	res, err = client.NonceAt(ctx, account, blockNumber)
	require.NoError(t, err)
	require.Equal(t, valueInt, res)

	// Fetching a different account goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*hexutil.Uint64) = 0
			return errors.New("Some error")
		}).Times(1)
	_, err = client.NonceAt(ctx, common.HexToAddress("0x4567"), blockNumber)
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*hexutil.Uint64) = 0
			return errors.New("Some error")
		}).Times(1)
	_, err = client.NonceAt(ctx, common.HexToAddress("0x4567"), blockNumber)
	require.Error(t, err)

	// Fetching a different block goes to the chain
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*hexutil.Uint64) = 0
			return errors.New("Some error")
		}).Times(1)
	_, err = client.NonceAt(ctx, account, big.NewInt(5))
	require.Error(t, err)

	// No cache due to error, should hit chain again
	ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			*result.(*hexutil.Uint64) = 0
			return errors.New("Some error")
		}).Times(1)
	_, err = client.NonceAt(ctx, account, big.NewInt(5))
	require.Error(t, err)

	// Non-concrete block numbers always go to chain
	for i := 0; i < 3; i++ {
		newValueHex, newValueInt := getTestTransactionCount()
		for _, blockNumber := range specialBlockNumbers() {
			ethClient.EXPECT().CallContext(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
					*result.(*hexutil.Uint64) = newValueHex
					return nil
				}).Times(1)
			res, err := client.NonceAt(ctx, account, blockNumber)
			require.NoError(t, err)
			require.Equal(t, newValueInt, res)
		}
	}
}
