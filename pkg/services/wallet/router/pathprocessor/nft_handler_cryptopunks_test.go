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

	"github.com/status-im/status-go/internal/contracts/cryptopunks"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	mock_transactor "github.com/status-im/status-go/internal/transactions/mock"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

func TestCryptoPunksHandler_Comprehensive(t *testing.T) {
	handler := NewCryptoPunksHandler(nil, nil)

	// Test basic methods
	assert.Equal(t, "CryptoPunksTransfer", handler.Name())
	assert.True(t, handler.CanHandle(CryptoPunksContractID))
	assert.False(t, handler.CanHandle(thirdparty.ContractID{
		ChainID: walletCommon.ChainID(walletCommon.OptimismMainnet),
		Address: CryptoPunksContractID.Address,
	}))

	// Test GetContractAddress
	address, err := handler.GetContractAddress(ProcessorInputParams{})
	require.NoError(t, err)
	assert.Equal(t, CryptoPunksContractID.Address, address)

	// Test PackTxInputData - valid case
	parsed, ok := new(big.Int).SetString("123", 0)
	require.True(t, ok)
	hb := hexutil.Big(*parsed)
	validParams := ProcessorInputParams{
		FromToken: &tokentypes.Token{
			Token:              &wsdktypes.Token{Symbol: "PUNK"},
			CollectibleTokenID: &hb,
		},
		ToAddr: common.HexToAddress("0x5678901234567890123456789012345678901234"),
	}
	data, err := handler.PackTxInputData(validParams)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify packed data correctness
	parsedABI, _ := abi.JSON(strings.NewReader(cryptopunks.CryptoPunksMetaData.ABI))
	expectedData, _ := parsedABI.Pack(cryptoPunksHandlerFunctionNameTransferPunk, validParams.ToAddr, big.NewInt(123))
	assert.Equal(t, expectedData, data)

	// Test PackTxInputData - error cases
	_, err = handler.PackTxInputData(ProcessorInputParams{
		FromToken: nil,
		ToAddr:    validParams.ToAddr,
	})
	assert.ErrorIs(t, err, ErrNoTokenSet)

	_, err = handler.PackTxInputData(ProcessorInputParams{
		FromToken: &tokentypes.Token{Token: &wsdktypes.Token{Symbol: "PUNK"}},
		ToAddr:    validParams.ToAddr,
	})
	assert.ErrorIs(t, err, ErrNoTokenSet)
}

func TestCryptoPunksHandler_WithMocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
	mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
	handler := NewCryptoPunksHandler(mockRPCClient, mockTransactor)

	testTx := ethTypes.NewTransaction(1, CryptoPunksContractID.Address, big.NewInt(0), 21000, big.NewInt(1000000000), []byte{})

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
			expectedAddress := cryptotypes.Address(CryptoPunksContractID.Address)
			assert.Equal(t, &expectedAddress, args.To)
			return testTx, uint64(1), nil
		})

	tx2, nonce, err := handler.BuildTransactionV2(mockTransactor, buildArgs, -1)
	require.NoError(t, err)
	assert.Equal(t, testTx, tx2)
	assert.Equal(t, uint64(1), nonce)
}
