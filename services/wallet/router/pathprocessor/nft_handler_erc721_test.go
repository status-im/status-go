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

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/contracts/erc721"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	mock_transactor "github.com/status-im/status-go/internal/transactions/mock"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

func TestERC721Handler_Comprehensive(t *testing.T) {
	handler := NewERC721Handler(nil, nil)

	// Test basic methods
	assert.Equal(t, pathProcessorCommon.ProcessorERC721Name, handler.Name())
	assert.True(t, handler.CanHandle(thirdparty.ContractID{
		ChainID: walletCommon.ChainID(walletCommon.EthereumMainnet),
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
	}))
	assert.True(t, handler.CanHandle(CryptoKittiesContractID))
	assert.True(t, handler.CanHandle(CryptoPunksContractID))

	// Test GetContractAddress
	contractAddr := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	params := ProcessorInputParams{
		FromToken: &tokentypes.Token{Token: &types.Token{Address: contractAddr}},
	}
	address, err := handler.GetContractAddress(params)
	require.NoError(t, err)
	assert.Equal(t, contractAddr, address)

	// Note: PackTxInputData requires RPC client for checkIfFunctionExists,
	// so we test the internal packing function instead
}

func TestERC721Handler_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	handler := NewERC721Handler(nil, mockTransactor)

	testTx := ethTypes.NewTransaction(1, common.HexToAddress("0xabcd"), big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})
	contractAddress := types2.Address(common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"))

	// Test BuildTransactionV2
	buildArgs := &wallettypes.SendTxArgs{
		FromChainID: walletCommon.EthereumMainnet,
		From:        types2.HexToAddress("0x1234567890123456789012345678901234567890"),
		To:          &contractAddress,
		Gas:         (*hexutil.Uint64)(new(uint64)),
		GasPrice:    (*hexutil.Big)(big.NewInt(1000000000)),
		Value:       (*hexutil.Big)(big.NewInt(0)),
		Data:        types2.HexBytes("test_data"),
	}

	mockTransactor.EXPECT().ValidateAndBuildTransaction(walletCommon.EthereumMainnet, *buildArgs, int64(-1)).Return(testTx, uint64(1), nil)

	tx, nonce, err := handler.BuildTransactionV2(mockTransactor, buildArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx)
	assert.Equal(t, uint64(1), nonce)
}

func TestERC721Handler_PackTxInputDataInternally(t *testing.T) {
	handler := NewERC721Handler(nil, nil)

	testCases := []struct {
		name     string
		tokenID  string
		expected *big.Int
	}{
		{"Simple token ID", "456", big.NewInt(456)},
		{"Zero token ID", "0", big.NewInt(0)},
		{"Large token ID", "999999", big.NewInt(999999)},
		{"Hex token ID", "0x1a", big.NewInt(26)},
	}

	fromAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	toAddr := common.HexToAddress("0x5678901234567890123456789012345678901234")
	contractAddr := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, ok := new(big.Int).SetString(tc.tokenID, 0)
			require.True(t, ok)
			assert.Equal(t, tc.expected, parsed)
			collectibleTokenID := hexutil.Big(*parsed)

			params := ProcessorInputParams{
				FromToken: &tokentypes.Token{
					Token:              &types.Token{Address: contractAddr},
					CollectibleTokenID: &collectibleTokenID,
				},
				FromAddr: fromAddr,
				ToAddr:   toAddr,
			}

			// Test safeTransferFrom
			data, err := handler.packTxInputDataInternally(params, erc721FunctionNameSafeTransferFrom)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Verify packed data correctness
			parsedABI, _ := abi.JSON(strings.NewReader(erc721.Erc721MetaData.ABI))
			expectedData, _ := parsedABI.Pack(erc721FunctionNameSafeTransferFrom, fromAddr, toAddr, tc.expected)
			assert.Equal(t, expectedData, data)

			// Test transferFrom
			data2, err := handler.packTxInputDataInternally(params, erc721FunctionNameTransferFrom)
			require.NoError(t, err)
			assert.NotEmpty(t, data2)

			// Verify packed data correctness
			expectedData2, _ := parsedABI.Pack(erc721FunctionNameTransferFrom, fromAddr, toAddr, tc.expected)
			assert.Equal(t, expectedData2, data2)

			// Data should be different (different function signatures)
			assert.NotEqual(t, data, data2)
		})
	}

	// Test error cases
	_, err := handler.packTxInputDataInternally(ProcessorInputParams{
		FromToken: nil,
		FromAddr:  fromAddr,
		ToAddr:    toAddr,
	}, erc721FunctionNameSafeTransferFrom)
	assert.ErrorIs(t, err, ErrNoTokenSet)

	_, err = handler.packTxInputDataInternally(ProcessorInputParams{
		FromToken: &tokentypes.Token{Token: &types.Token{Address: contractAddr}},
		FromAddr:  fromAddr,
		ToAddr:    toAddr,
	}, erc721FunctionNameSafeTransferFrom)
	assert.ErrorIs(t, err, ErrNoTokenSet)

	// Test with invalid function name
	parsed, ok := new(big.Int).SetString("123", 0)
	require.True(t, ok)
	collectibleTokenID := hexutil.Big(*parsed)
	validParams := ProcessorInputParams{
		FromToken: &tokentypes.Token{
			Token:              &types.Token{Address: contractAddr},
			CollectibleTokenID: &collectibleTokenID,
		},
		FromAddr: fromAddr,
		ToAddr:   toAddr,
	}
	_, err = handler.packTxInputDataInternally(validParams, "invalidFunction")
	assert.Error(t, err) // Should error because function doesn't exist in ABI
}

func TestERC721Handler_Integration(t *testing.T) {
	handler := NewERC721Handler(nil, nil)

	// Test complete flow for generic ERC721 transfer
	contractAddr := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	parsed, ok := new(big.Int).SetString("789", 0)
	require.True(t, ok)
	collectibleTokenID := hexutil.Big(*parsed)
	params := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &tokentypes.Token{Token: &types.Token{
			Address: contractAddr,
		}},
		ToToken: nil,
	}
	params.FromToken.CollectibleTokenID = &collectibleTokenID
	params.FromAddr = common.HexToAddress("0x1234567890123456789012345678901234567890")
	params.ToAddr = common.HexToAddress("0x5678901234567890123456789012345678901234")

	// 1. Verify handler can handle any contract (ERC721 is generic)
	contractID := thirdparty.ContractID{
		ChainID: walletCommon.ChainID(walletCommon.EthereumMainnet),
		Address: contractAddr,
	}
	assert.True(t, handler.CanHandle(contractID))

	// 2. Get contract address
	address, err := handler.GetContractAddress(params)
	require.NoError(t, err)
	assert.Equal(t, contractAddr, address)

	// 3. Test internal packing (since PackTxInputData requires RPC)
	data, err := handler.packTxInputDataInternally(params, erc721FunctionNameSafeTransferFrom)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 4. Verify the data contains valid ERC721 function call
	assert.True(t, len(data) >= 4, "Data should contain at least function selector (4 bytes)")
}
