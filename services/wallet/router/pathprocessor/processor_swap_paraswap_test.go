package pathprocessor

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	gomock "go.uber.org/mock/gomock"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/paraswap"
	mock_paraswap "github.com/status-im/status-go/services/wallet/thirdparty/paraswap/mock"
	"github.com/status-im/status-go/services/wallet/token"

	"github.com/stretchr/testify/require"
)

func TestParaswapWithPartnerFee(t *testing.T) {
	testPriceRoute := &paraswap.Route{
		GasCost:            &bigint.BigInt{Int: big.NewInt(500)},
		SrcAmount:          &bigint.BigInt{Int: big.NewInt(1000)},
		SrcTokenAddress:    common.HexToAddress("0x123"),
		SrcTokenDecimals:   18,
		DestAmount:         &bigint.BigInt{Int: big.NewInt(2000)},
		DestTokenAddress:   common.HexToAddress("0x465"),
		DestTokenDecimals:  6,
		Side:               paraswap.SellSide,
		ContractAddress:    common.HexToAddress("0x789"),
		TokenTransferProxy: common.HexToAddress("0xabc"),
	}

	processor := NewSwapParaswapProcessor(nil, nil, nil)

	fromToken := token.Token{
		Symbol: EthSymbol,
	}
	toToken := token.Token{
		Symbol: UsdcSymbol,
	}
	chainIDs := []uint64{walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet, walletCommon.OptimismMainnet, walletCommon.UnknownChainID}

	for _, chainID := range chainIDs {
		key := makeKey(chainID, chainID, fromToken.Symbol, toToken.Symbol, testPriceRoute.SrcAmount.Int)
		processor.priceRoute.Store(key, testPriceRoute)

		testInputParams := ProcessorInputParams{
			FromChain: &params.Network{ChainID: chainID},
			ToChain:   &params.Network{ChainID: chainID},
			FromToken: &fromToken,
			ToToken:   &toToken,
			AmountIn:  testPriceRoute.SrcAmount.Int,
		}

		partnerAddress, partnerFeePcnt := getPartnerAddressAndFeePcnt(chainID)

		if partnerAddress != walletCommon.ZeroAddress() {
			require.Greater(t, partnerFeePcnt, 0.0)

			expectedFee := uint64(float64(testPriceRoute.DestAmount.Uint64()) * partnerFeePcnt / 100.0)
			expectedDestAmount := testPriceRoute.DestAmount.Uint64() - expectedFee

			amountOut, err := processor.CalculateAmountOut(testInputParams)
			require.NoError(t, err)
			require.NotNil(t, amountOut)
			require.InEpsilon(t, expectedDestAmount, amountOut.Uint64(), 2.0)
		} else {
			require.Equal(t, 0.0, partnerFeePcnt)

			amountOut, err := processor.CalculateAmountOut(testInputParams)
			require.NoError(t, err)
			require.NotNil(t, amountOut)
			require.Equal(t, testPriceRoute.DestAmount.Uint64(), amountOut.Uint64())
		}

		// Check contract address
		contractAddress, err := processor.GetContractAddress(testInputParams)
		require.NoError(t, err)
		require.Equal(t, testPriceRoute.TokenTransferProxy, contractAddress)
	}
}

func expectClientFetchPriceRoute(clientMock *mock_paraswap.MockClientInterface, route paraswap.Route, err error) {
	clientMock.EXPECT().FetchPriceRoute(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(route, err)
}

func TestParaswapErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_paraswap.NewMockClientInterface(ctrl)

	processor := NewSwapParaswapProcessor(nil, nil, nil)
	processor.paraswapClient = client

	fromToken := token.Token{
		Symbol: EthSymbol,
	}
	toToken := token.Token{
		Symbol: UsdcSymbol,
	}
	chainID := walletCommon.EthereumMainnet

	testInputParams := ProcessorInputParams{
		FromChain: &params.Network{ChainID: chainID},
		ToChain:   &params.Network{ChainID: chainID},
		FromToken: &fromToken,
		ToToken:   &toToken,
	}

	// Test Errors
	type testCase struct {
		clientError    string
		processorError error
	}

	testCases := []testCase{
		{"Price Timeout", ErrPriceTimeout},
		{"No routes found with enough liquidity", ErrNotEnoughLiquidity},
		{"ESTIMATED_LOSS_GREATER_THAN_MAX_IMPACT", ErrPriceImpactTooHigh},
	}

	for _, tc := range testCases {
		expectClientFetchPriceRoute(client, paraswap.Route{}, errors.New(tc.clientError))
		_, err := processor.EstimateGas(testInputParams)
		require.Equal(t, tc.processorError.Error(), err.Error())
	}
}
