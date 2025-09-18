package pathprocessor

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/crypto/types"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	mock_transactor "github.com/status-im/status-go/transactions/mock"
)

func TestBaseNFTHandler_Comprehensive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
	handler := NewBaseNFTHandler(mockRPCClient, mockTransactor)

	// Test constructor and basic methods
	assert.NotNil(t, handler)
	params := ProcessorInputParams{AmountIn: big.NewInt(1)}

	// Test CalculateFees and CalculateAmountOut
	bonderFee, tokenFee, err := handler.CalculateFees(params)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(0), bonderFee)
	assert.Equal(t, big.NewInt(0), tokenFee)

	amountOut, err := handler.CalculateAmountOut(params)
	require.NoError(t, err)
	assert.Equal(t, params.AmountIn, amountOut)

	// Test EstimateGas with estimation map
	params.TestsMode = true
	params.TestEstimationMap = map[string]requests.Estimation{"TestHandler": {Value: 21000, Err: nil}}
	estimation, err := handler.EstimateGas(params, []byte{}, "TestHandler")
	require.NoError(t, err)
	assert.Equal(t, uint64(21000), estimation)

	// Test EstimateGas without estimation
	params.TestEstimationMap = nil
	_, err = handler.EstimateGas(params, []byte{}, "TestHandler")
	assert.Equal(t, ErrNoEstimationFound, err)
}

func TestSpecificHandlers_BuildTransactionV2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	testTx := ethTypes.NewTransaction(1, common.HexToAddress("0x1234"), big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})

	sendArgs := &wallettypes.SendTxArgs{
		FromChainID: walletCommon.EthereumMainnet, From: types.HexToAddress("0x1234567890123456789012345678901234567890"),
		To: &types.Address{}, Gas: (*hexutil.Uint64)(new(uint64)), GasPrice: (*hexutil.Big)(big.NewInt(1000000000)),
		Value: (*hexutil.Big)(big.NewInt(0)), Data: types.HexBytes("test_data"),
	}

	// Test CryptoKitties handler
	mockTransactor.EXPECT().ValidateAndBuildTransaction(walletCommon.EthereumMainnet, gomock.Any(), int64(-1)).Return(testTx, uint64(1), nil)
	kittiesHandler := NewCryptoKittiesHandler(nil, mockTransactor)
	tx, nonce, err := kittiesHandler.BuildTransactionV2(mockTransactor, sendArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
	assert.Equal(t, uint64(1), nonce)

	// Test CryptoPunks handler
	mockTransactor.EXPECT().ValidateAndBuildTransaction(walletCommon.EthereumMainnet, gomock.Any(), int64(-1)).Return(testTx, uint64(1), nil)
	punksHandler := NewCryptoPunksHandler(nil, mockTransactor)
	tx, nonce, err = punksHandler.BuildTransactionV2(mockTransactor, sendArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
	assert.Equal(t, uint64(1), nonce)

	// Test ERC721 handler
	mockTransactor.EXPECT().ValidateAndBuildTransaction(walletCommon.EthereumMainnet, gomock.Any(), int64(-1)).Return(testTx, uint64(1), nil)
	erc721Handler := NewERC721Handler(nil, mockTransactor)
	tx, nonce, err = erc721Handler.BuildTransactionV2(mockTransactor, sendArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
	assert.Equal(t, uint64(1), nonce)
}
