package pathprocessor

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	gomock "go.uber.org/mock/gomock"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/lifi"
	mock_lifi "github.com/status-im/status-go/pkg/services/wallet/thirdparty/lifi/mock"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLiFiTokens() (tokentypes.Token, tokentypes.Token) {
	fromToken := tokentypes.Token{Token: &types.Token{
		Symbol:  walletCommon.EthSymbol,
		ChainID: walletCommon.EthereumMainnet,
	}}
	toToken := tokentypes.Token{Token: &types.Token{
		Symbol:  walletCommon.UsdcSymbol,
		Address: common.HexToAddress("0x465"),
		ChainID: walletCommon.EthereumMainnet,
	}}
	return fromToken, toToken
}

func TestLiFiQuote(t *testing.T) {
	testQuote := lifi.Quote{
		Estimate: lifi.Estimate{
			FromAmount:      &bigint.BigInt{Int: big.NewInt(1000)},
			ToAmount:        &bigint.BigInt{Int: big.NewInt(2000)},
			ToAmountMin:     &bigint.BigInt{Int: big.NewInt(1990)},
			ApprovalAddress: common.HexToAddress("0xabc"),
		},
		TransactionRequest: lifi.TransactionRequest{
			From:     "0x111",
			To:       "0x222",
			Value:    "0x3e8",
			Data:     "0xabcd",
			GasPrice: "0x64",
			GasLimit: "0x3e8",
			ChainID:  walletCommon.EthereumMainnet,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_lifi.NewMockClientInterface(ctrl)
	client.EXPECT().SetChainID(gomock.Any()).AnyTimes()

	processor := NewLiFiProcessor(nil, nil, nil)
	processor.lifiClient = client

	fromToken, toToken := testLiFiTokens()
	amountIn := testQuote.Estimate.FromAmount.Int

	testInputParams := ProcessorInputParams{
		FromAddr:  common.HexToAddress("0x111"),
		ToAddr:    common.HexToAddress("0x222"),
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &fromToken,
		ToToken:   &toToken,
		AmountIn:  amountIn,
	}

	available, err := processor.AvailableFor(testInputParams)
	require.NoError(t, err)
	require.True(t, available)

	key := pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn)
	processor.quotes.Store(key, &testQuote)

	amountOut, err := processor.CalculateAmountOut(testInputParams)
	require.NoError(t, err)
	require.NotNil(t, amountOut)
	require.Equal(t, testQuote.Estimate.ToAmount.Uint64(), amountOut.Uint64())

	client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(testQuote, nil)
	contractAddress, err := processor.GetContractAddress(testInputParams)
	require.NoError(t, err)
	require.Equal(t, testQuote.Estimate.ApprovalAddress, contractAddress)

	client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(testQuote, nil)
	inputData, err := processor.PackTxInputData(testInputParams)
	assert.NoError(t, err)
	assert.Equal(t, testQuote.TransactionRequest.Data, hexutil.Encode(inputData))
}

func TestLiFiBridgeAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_lifi.NewMockClientInterface(ctrl)
	client.EXPECT().SetChainID(gomock.Any()).AnyTimes()

	processor := NewLiFiProcessor(nil, nil, nil)
	processor.lifiClient = client

	// Bridge the same asset (USDC) across two different chains.
	fromToken := tokentypes.Token{Token: &types.Token{
		Symbol:  walletCommon.UsdcSymbol,
		Address: common.HexToAddress("0x465"),
		ChainID: walletCommon.EthereumMainnet,
	}}
	toToken := tokentypes.Token{Token: &types.Token{
		Symbol:  walletCommon.UsdcSymbol,
		Address: common.HexToAddress("0x999"),
		ChainID: walletCommon.OptimismMainnet,
	}}

	available, err := processor.AvailableFor(ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.OptimismMainnet},
		FromToken: &fromToken,
		ToToken:   &toToken,
		AmountIn:  big.NewInt(1000),
	})
	require.NoError(t, err)
	require.True(t, available)
}

func TestLiFiBuySideUnsupported(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_lifi.NewMockClientInterface(ctrl)
	processor := NewLiFiProcessor(nil, nil, nil)
	processor.lifiClient = client

	fromToken, toToken := testLiFiTokens()

	available, err := processor.AvailableFor(ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &fromToken,
		ToToken:   &toToken,
		AmountOut: big.NewInt(2000),
	})
	require.NoError(t, err)
	require.False(t, available)
}

func TestLiFiErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_lifi.NewMockClientInterface(ctrl)
	client.EXPECT().SetChainID(gomock.Any()).AnyTimes()

	processor := NewLiFiProcessor(nil, nil, nil)
	processor.lifiClient = client

	fromToken, toToken := testLiFiTokens()

	testInputParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
		ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
		FromToken: &fromToken,
		ToToken:   &toToken,
		AmountIn:  big.NewInt(1000),
	}

	testCases := []struct {
		clientError    string
		processorError error
	}{
		{"No available quotes for the requested transfer", ErrNotEnoughLiquidity},
		{"price impact too high", ErrPriceImpactTooHigh},
	}

	for _, tc := range testCases {
		client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(lifi.Quote{}, errors.New(tc.clientError))
		_, err := processor.GetContractAddress(testInputParams)
		require.Equal(t, tc.processorError.Error(), err.Error())
	}
}

func TestLiFiEstimateGas(t *testing.T) {
	testQuote := lifi.Quote{
		Estimate: lifi.Estimate{
			FromAmount:      &bigint.BigInt{Int: big.NewInt(1000)},
			ToAmount:        &bigint.BigInt{Int: big.NewInt(2000)},
			ToAmountMin:     &bigint.BigInt{Int: big.NewInt(1990)},
			ApprovalAddress: common.HexToAddress("0xabc"),
		},
		TransactionRequest: lifi.TransactionRequest{
			Data:     "0xabcd",
			GasLimit: "0x3e8",
			ChainID:  walletCommon.EthereumMainnet,
		},
	}
	amountIn := testQuote.Estimate.FromAmount.Int

	t.Run("native token estimates through the eth client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_lifi.NewMockClientInterface(ctrl)
		client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)

		processor := NewLiFiProcessor(mockRPCClient, nil, nil)
		processor.lifiClient = client

		fromToken, toToken := testLiFiTokens()
		testInputParams := ProcessorInputParams{
			FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
			ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
			FromToken: &fromToken,
			ToToken:   &toToken,
			AmountIn:  amountIn,
		}

		// Pre-store the quote so no fetch is needed.
		key := pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn)
		processor.quotes.Store(key, &testQuote)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(80000), nil)

		estimation, err := processor.EstimateGas(testInputParams, []byte{})
		require.NoError(t, err)
		require.Equal(t, uint64(float64(80000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("erc20 estimation error falls back to the quoted gas limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_lifi.NewMockClientInterface(ctrl)
		client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)

		processor := NewLiFiProcessor(mockRPCClient, nil, nil)
		processor.lifiClient = client

		// Non-native from token: estimation reverts before approval, the quote's gas limit is used.
		fromToken := tokentypes.Token{Token: &types.Token{
			Symbol:  walletCommon.UsdcSymbol,
			Address: common.HexToAddress("0x465"),
			ChainID: walletCommon.EthereumMainnet,
		}}
		toToken := tokentypes.Token{Token: &types.Token{
			Symbol:  walletCommon.EthSymbol,
			Address: common.HexToAddress("0x999"),
			ChainID: walletCommon.EthereumMainnet,
		}}
		testInputParams := ProcessorInputParams{
			FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
			ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
			FromToken: &fromToken,
			ToToken:   &toToken,
			AmountIn:  amountIn,
		}

		key := pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn)
		processor.quotes.Store(key, &testQuote)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			Return(uint64(0), errors.New("execution reverted: ERC20: transfer amount exceeds allowance"))

		estimation, err := processor.EstimateGas(testInputParams, []byte{})
		require.NoError(t, err)
		// GasLimit 0x3e8 = 1000
		require.Equal(t, uint64(float64(1000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("native estimation error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_lifi.NewMockClientInterface(ctrl)
		client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)

		processor := NewLiFiProcessor(mockRPCClient, nil, nil)
		processor.lifiClient = client

		fromToken, toToken := testLiFiTokens()
		testInputParams := ProcessorInputParams{
			FromChain: &params.Network{ChainID: walletCommon.EthereumMainnet},
			ToChain:   &params.Network{ChainID: walletCommon.EthereumMainnet},
			FromToken: &fromToken,
			ToToken:   &toToken,
			AmountIn:  amountIn,
		}

		key := pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn)
		processor.quotes.Store(key, &testQuote)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			Return(uint64(0), errors.New("estimation failed"))

		_, err := processor.EstimateGas(testInputParams, []byte{})
		require.Error(t, err)
	})
}
