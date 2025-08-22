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

	"github.com/status-im/status-go/accounts-management/generator"
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

func TestBaseNFTHandler_NonceAndTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
	handler := NewBaseNFTHandler(mockRPCClient, mockTransactor)

	sendArgs := &MultipathProcessorTxArgs{
		ChainID:          1,
		ERC721TransferTx: &ERC721TxArgs{SendTxArgs: wallettypes.SendTxArgs{From: types.HexToAddress("0x1234")}},
	}

	// Test PrepareNonce with positive lastUsedNonce
	nonce, err := handler.PrepareNonce(sendArgs, 5)
	require.NoError(t, err)
	assert.Equal(t, uint64(6), nonce)

	// Test PrepareNonce with negative lastUsedNonce (calls RPC)
	mockTransactor.EXPECT().NextNonce(gomock.Any(), mockRPCClient, uint64(1), types.HexToAddress("0x1234")).Return(uint64(10), nil)
	nonce, err = handler.PrepareNonce(sendArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, uint64(10), nonce)

	// Test high-level methods with mock NFT handler
	testTx := ethTypes.NewTransaction(1, common.HexToAddress("0x1234"), big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})
	mockNFTHandler := NewMockNFTHandler(ctrl)
	mockNFTHandler.EXPECT().SendOrBuild(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(testTx, nil).Times(2)

	// Test Send method
	hash, sendNonce, err := handler.Send(sendArgs, -1, &generator.Account{}, mockNFTHandler)
	require.NoError(t, err)
	assert.Equal(t, types.Hash(testTx.Hash()), hash)
	assert.Equal(t, testTx.Nonce(), sendNonce)

	// Test BuildTransaction method
	tx, buildNonce, err := handler.BuildTransaction(sendArgs, -1, mockNFTHandler)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
	assert.Equal(t, testTx.Nonce(), buildNonce)
}

func TestBaseNFTHandler_SendOrBuildCollectible(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
	handler := NewBaseNFTHandler(mockRPCClient, mockTransactor)

	testTx := ethTypes.NewTransaction(1, common.HexToAddress("0x1234"), big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})
	contractAddress := types.Address(common.HexToAddress("0xabcd"))
	sendArgs := &MultipathProcessorTxArgs{
		ChainID: 1,
		ERC721TransferTx: &ERC721TxArgs{
			SendTxArgs: wallettypes.SendTxArgs{From: types.HexToAddress("0x1234"), To: &contractAddress},
			TokenID:    (*hexutil.Big)(big.NewInt(123)), Recipient: common.HexToAddress("0x5678"),
		},
	}

	// Setup mock expectations
	mockTransactor.EXPECT().NextNonce(gomock.Any(), mockRPCClient, uint64(1), types.HexToAddress("0x1234")).Return(uint64(1), nil)
	mockTransactor.EXPECT().ValidateAndBuildTransaction(uint64(1), gomock.Any(), int64(0)).Return(testTx, uint64(1), nil)
	mockTransactor.EXPECT().StoreAndTrackPendingTx(gomock.Any(), gomock.Any(), uint64(1), gomock.Any(), testTx).Return(nil)

	packDataFn := func(params ProcessorInputParams) ([]byte, error) { return []byte("test_data"), nil }
	tx, err := handler.SendOrBuildCollectible(sendArgs, -1, packDataFn, nil)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
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

func TestCryptoKittiesHandler_SendOrBuild(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
	handler := NewCryptoKittiesHandler(mockRPCClient, mockTransactor)

	testTx := ethTypes.NewTransaction(1, CryptoKittiesContractID.Address, big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})
	contractAddress := types.Address(CryptoKittiesContractID.Address)
	sendArgs := &MultipathProcessorTxArgs{
		ChainID: walletCommon.EthereumMainnet,
		ERC721TransferTx: &ERC721TxArgs{
			SendTxArgs: wallettypes.SendTxArgs{From: types.HexToAddress("0x1234567890123456789012345678901234567890"), To: &contractAddress},
			TokenID:    (*hexutil.Big)(big.NewInt(123)), Recipient: common.HexToAddress("0x5678901234567890123456789012345678901234"),
		},
	}

	// Setup mock expectations
	mockTransactor.EXPECT().NextNonce(gomock.Any(), mockRPCClient, walletCommon.EthereumMainnet, types.HexToAddress("0x1234567890123456789012345678901234567890")).Return(uint64(1), nil)
	mockTransactor.EXPECT().ValidateAndBuildTransaction(walletCommon.EthereumMainnet, gomock.Any(), int64(0)).Return(testTx, uint64(1), nil)
	mockTransactor.EXPECT().StoreAndTrackPendingTx(gomock.Any(), gomock.Any(), walletCommon.EthereumMainnet, gomock.Any(), testTx).Return(nil)

	tx, err := handler.SendOrBuild(mockTransactor, mockRPCClient, sendArgs, nil, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
}
