package ethclient_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"

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
