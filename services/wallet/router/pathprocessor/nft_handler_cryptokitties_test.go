package pathprocessor

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	wsdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/contracts/cryptokitties"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	mock_transactor "github.com/status-im/status-go/internal/transactions/mock"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

func TestCryptoKittiesHandler_Comprehensive(t *testing.T) {
	handler := NewCryptoKittiesHandler(nil, nil)

	// Test basic methods
	assert.Equal(t, "CryptoKittiesTransfer", handler.Name())
	assert.True(t, handler.CanHandle(CryptoKittiesContractID))
	assert.False(t, handler.CanHandle(thirdparty.ContractID{
		ChainID: walletCommon.ChainID(walletCommon.OptimismMainnet),
		Address: CryptoKittiesContractID.Address,
	}))
	assert.False(t, handler.CanHandle(CryptoPunksContractID))

	// Test GetContractAddress
	address, err := handler.GetContractAddress(ProcessorInputParams{})
	require.NoError(t, err)
	assert.Equal(t, CryptoKittiesContractID.Address, address)

	// Test PackTxInputData - valid cases
	toAddr := common.HexToAddress("0x5678901234567890123456789012345678901234")
	testCases := []struct {
		tokenID  string
		expected *big.Int
	}{
		{"123", big.NewInt(123)},
		{"0", big.NewInt(0)},
		{"0xff", big.NewInt(255)},
		{"999999", big.NewInt(999999)},
	}

	for _, tc := range testCases {
		parsed, ok := new(big.Int).SetString(tc.tokenID, 0)
		require.True(t, ok)
		hb := hexutil.Big(*parsed)

		params := ProcessorInputParams{
			FromToken: &tokentypes.Token{
				Token:              &wsdktypes.Token{Symbol: "CK"},
				CollectibleTokenID: &hb,
			},
			ToAddr: toAddr,
		}
		data, err := handler.PackTxInputData(params)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify packed data correctness
		parsedABI, _ := abi.JSON(strings.NewReader(cryptokitties.CryptoKittiesMetaData.ABI))
		expectedData, _ := parsedABI.Pack(cryptoKittiesFunctionNameTransfer, toAddr, tc.expected)
		assert.Equal(t, expectedData, data)
	}

	// Test PackTxInputData - error cases
	_, err = handler.PackTxInputData(ProcessorInputParams{
		FromToken: nil,
		ToAddr:    toAddr,
	})
	assert.ErrorIs(t, err, ErrNoTokenSet)

	_, err = handler.PackTxInputData(ProcessorInputParams{
		FromToken: &tokentypes.Token{Token: &wsdktypes.Token{Symbol: "CK"}},
		ToAddr:    toAddr,
	})
	assert.ErrorIs(t, err, ErrNoTokenSet)
}

func TestCryptoKittiesHandler_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
	handler := NewCryptoKittiesHandler(mockRPCClient, mockTransactor)

	testTx := ethTypes.NewTransaction(1, CryptoKittiesContractID.Address, big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})

	// Test BuildTransactionV2
	buildArgs := &wallettypes.SendTxArgs{
		FromChainID: walletCommon.EthereumMainnet,
		From:        cryptotypes.HexToAddress("0x1234567890123456789012345678901234567890"),
		To:          &cryptotypes.Address{},
		Gas:         (*hexutil.Uint64)(new(uint64)),
		GasPrice:    (*hexutil.Big)(big.NewInt(1000000000)),
		Value:       (*hexutil.Big)(big.NewInt(0)),
		Data:        cryptotypes.HexBytes("test_data"),
	}

	mockTransactor.EXPECT().ValidateAndBuildTransaction(walletCommon.EthereumMainnet, gomock.Any(), int64(-1)).DoAndReturn(
		func(chainID uint64, args wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
			expectedAddress := cryptotypes.Address(CryptoKittiesContractID.Address)
			assert.Equal(t, &expectedAddress, args.To)
			return testTx, uint64(1), nil
		})

	tx2, nonce, err := handler.BuildTransactionV2(mockTransactor, buildArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx2)
	assert.Equal(t, uint64(1), nonce)
}
