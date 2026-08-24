package pathprocessor

import (
	"context"
	"errors"
	"math/big"
	netUrl "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	sdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	mock_transactor "github.com/status-im/status-go/internal/transactions/mock"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/paraswap"
	mock_paraswap "github.com/status-im/status-go/pkg/services/wallet/thirdparty/paraswap/mock"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

var (
	testFromAddr = common.HexToAddress("0x0000000000000000000000000000000000000A01")
	testToAddr   = common.HexToAddress("0x0000000000000000000000000000000000000A02")
)

// makeNativeToken returns a token with the zero address, which Token.IsNative treats as native.
func makeNativeToken(chainID uint64) *tokentypes.Token {
	return &tokentypes.Token{Token: &sdktypes.Token{
		ChainID: chainID,
		Symbol:  walletCommon.EthSymbol,
	}}
}

// fakeHopHTTPClient satisfies hopHTTPClient and records the last requested URL.
type fakeHopHTTPClient struct {
	resp    []byte
	err     error
	lastURL string
}

func (f *fakeHopHTTPClient) DoGetRequest(_ context.Context, url string, _ netUrl.Values, _ ...thirdparty.RequestOption) ([]byte, error) {
	f.lastURL = url
	return f.resp, f.err
}

func TestTransferProcessor_PackTxInputDataAndEstimateGas(t *testing.T) {
	amountIn := big.NewInt(1000)

	t.Run("native token estimates through the transactor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
		processor := NewTransferProcessor(nil, mockTransactor)

		params := ProcessorInputParams{
			FromChain: &mainnet,
			FromAddr:  testFromAddr,
			ToAddr:    testToAddr,
			FromToken: makeNativeToken(walletCommon.EthereumMainnet),
			AmountIn:  amountIn,
		}

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		assert.Empty(t, input)

		mockTransactor.EXPECT().
			EstimateGas(walletCommon.EthereumMainnet, testFromAddr, testToAddr, amountIn, input).
			Return(uint64(21000), nil)

		estimation, err := processor.EstimateGas(params, input)
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(21000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("erc20 token estimates through the eth client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewTransferProcessor(mockRPCClient, nil)

		params := ProcessorInputParams{
			FromChain: &mainnet,
			FromAddr:  testFromAddr,
			ToAddr:    testToAddr,
			FromToken: usdcMainnet,
			ToToken:   usdcMainnet,
			AmountIn:  amountIn,
		}

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		// ERC20 transfer(address,uint256) selector
		assert.Equal(t, common.FromHex("0xa9059cbb"), input[:4])

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().
			EstimateGas(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, msg ethereum.CallMsg) (uint64, error) {
				assert.Equal(t, testFromAddr, msg.From)
				assert.Equal(t, &usdcMainnet.Address, msg.To)
				assert.Equal(t, input, msg.Data)
				return uint64(50000), nil
			})

		estimation, err := processor.EstimateGas(params, input)
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(50000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("transactor error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTransactor := mock_transactor.NewMockTransactorIface(ctrl)
		processor := NewTransferProcessor(nil, mockTransactor)

		mockTransactor.EXPECT().
			EstimateGas(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(uint64(0), errors.New("estimation failed"))

		_, err := processor.EstimateGas(ProcessorInputParams{
			FromChain: &mainnet,
			FromToken: makeNativeToken(walletCommon.EthereumMainnet),
			AmountIn:  amountIn,
		}, []byte{})
		assert.Error(t, err)
	})

	t.Run("eth client getter error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		processor := NewTransferProcessor(mockRPCClient, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(nil, errors.New("no client"))

		_, err := processor.EstimateGas(ProcessorInputParams{
			FromChain: &mainnet,
			FromToken: usdcMainnet,
			AmountIn:  amountIn,
		}, []byte{})
		assert.Error(t, err)
	})
}

func TestHopBridgeProcessor_CalculateFees(t *testing.T) {
	amountIn := big.NewInt(100000000)
	params := ProcessorInputParams{
		FromChain: &mainnet,
		ToChain:   &optimism,
		FromAddr:  testFromAddr,
		ToAddr:    testToAddr,
		FromToken: usdcMainnet,
		ToToken:   usdcOptimism,
		AmountIn:  amountIn,
	}

	t.Run("fees are fetched from the hop quote API and cached", func(t *testing.T) {
		processor := NewHopBridgeProcessor(nil, nil, nil, nil)
		fake := &fakeHopHTTPClient{resp: []byte(`{
			"amountIn": "100000000",
			"slippage": 0.5,
			"amountOutMin": "99000000",
			"destinationAmountOutMin": "98000000",
			"bonderFee": "150000",
			"estimatedRecieved": "99850000",
			"deadline": 1700000000,
			"destinationDeadline": 1700000000
		}`)}
		processor.httpClient = fake

		bonderFee, tokenFee, err := processor.CalculateFees(params)
		require.NoError(t, err)
		assert.Equal(t, "https://api.hop.exchange/v1/quote", fake.lastURL)
		assert.Equal(t, big.NewInt(150000), bonderFee)
		// tokenFee = amountIn - estimatedRecieved
		assert.Equal(t, big.NewInt(150000), tokenFee)

		// The fetched quote must land in the bonder-fee cache used by PackTxInputData/CalculateAmountOut.
		amountOut, err := processor.CalculateAmountOut(params)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(99000000), amountOut)

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		assert.NotEmpty(t, input)
	})

	t.Run("pack without a cached bonder fee fails", func(t *testing.T) {
		processor := NewHopBridgeProcessor(nil, nil, nil, nil)

		_, err := processor.PackTxInputData(params)
		assert.Equal(t, ErrNoBonderFeeFound, err)
	})

	t.Run("http error is propagated", func(t *testing.T) {
		processor := NewHopBridgeProcessor(nil, nil, nil, nil)
		processor.httpClient = &fakeHopHTTPClient{err: errors.New("hop api unavailable")}

		_, _, err := processor.CalculateFees(params)
		assert.Error(t, err)
	})

	t.Run("malformed response is an error", func(t *testing.T) {
		processor := NewHopBridgeProcessor(nil, nil, nil, nil)
		processor.httpClient = &fakeHopHTTPClient{resp: []byte(`{invalid`)}

		_, _, err := processor.CalculateFees(params)
		assert.Error(t, err)
	})

	t.Run("unsupported chains are rejected before any request", func(t *testing.T) {
		processor := NewHopBridgeProcessor(nil, nil, nil, nil)
		processor.httpClient = &fakeHopHTTPClient{}

		_, _, err := processor.CalculateFees(ProcessorInputParams{
			FromToken: makeToken(walletCommon.BSCMainnet, "0x1"),
			ToToken:   usdcOptimism,
			AmountIn:  amountIn,
		})
		assert.Equal(t, ErrFromChainNotSupported, err)

		_, _, err = processor.CalculateFees(ProcessorInputParams{
			FromToken: usdcMainnet,
			ToToken:   makeToken(walletCommon.BSCMainnet, "0x1"),
			AmountIn:  amountIn,
		})
		assert.Equal(t, ErrToChainNotSupported, err)
	})
}

func TestHopBridgeProcessor_EstimateGas(t *testing.T) {
	amountIn := big.NewInt(100000000)
	params := ProcessorInputParams{
		FromChain: &mainnet,
		ToChain:   &optimism,
		FromAddr:  testFromAddr,
		FromToken: usdcMainnet,
		ToToken:   usdcOptimism,
		AmountIn:  amountIn,
	}

	t.Run("estimates through the eth client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewHopBridgeProcessor(mockRPCClient, nil, nil, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(100000), nil)

		estimation, err := processor.EstimateGas(params, []byte{})
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(100000)*pathProcessorCommon.IncreaseEstimatedGasFactorForBridge), estimation)
	})

	t.Run("erc20 estimation error falls back to the hardcoded estimation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewHopBridgeProcessor(mockRPCClient, nil, nil, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			Return(uint64(0), errors.New("execution reverted: ERC20: transfer amount exceeds allowance"))

		estimation, err := processor.EstimateGas(params, []byte{})
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(600000)*pathProcessorCommon.IncreaseEstimatedGasFactorForBridge), estimation)
	})

	t.Run("eth client getter error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		processor := NewHopBridgeProcessor(mockRPCClient, nil, nil, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(nil, errors.New("no client"))

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})
}

func TestSwapParaswapProcessor_PackTxInputDataAndEstimateGas(t *testing.T) {
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
	testTransaction := paraswap.Transaction{
		From:  testFromAddr.Hex(),
		To:    testToAddr.Hex(),
		Value: "1000",
		Data:  "0xabcd",
	}

	fromToken := makeToken(walletCommon.EthereumMainnet, "0x123")
	toToken := makeToken(walletCommon.EthereumMainnet, "0x465")
	amountIn := testPriceRoute.SrcAmount.Int

	params := ProcessorInputParams{
		FromChain: &mainnet,
		ToChain:   &mainnet,
		FromAddr:  testFromAddr,
		ToAddr:    testToAddr,
		FromToken: fromToken,
		ToToken:   toToken,
		AmountIn:  amountIn,
	}
	key := pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn)

	t.Run("pack builds the transaction from the stored price route", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_paraswap.NewMockClientInterface(ctrl)
		processor := NewSwapParaswapProcessor(nil, nil, nil)
		processor.paraswapClient = client
		processor.storePriceRoute(key, testPriceRoute)

		expectClientBuildTransaction(client, testTransaction, nil, nil)

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		assert.Equal(t, common.FromHex("0xabcd"), input)
	})

	t.Run("pack without a stored price route fails", func(t *testing.T) {
		processor := NewSwapParaswapProcessor(nil, nil, nil)

		_, err := processor.PackTxInputData(params)
		assert.Equal(t, ErrPriceRouteNotFound, err)
	})

	t.Run("estimate gas fetches the price route and estimates through the eth client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_paraswap.NewMockClientInterface(ctrl)
		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)

		processor := NewSwapParaswapProcessor(mockRPCClient, nil, nil)
		processor.paraswapClient = client

		expectClientFetchPriceRoute(client, *testPriceRoute, nil)
		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, msg ethereum.CallMsg) (uint64, error) {
				assert.Equal(t, &testPriceRoute.TokenTransferProxy, msg.To)
				return uint64(30000), nil
			})

		estimation, err := processor.EstimateGas(params, []byte{})
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(30000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("price route fetch error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_paraswap.NewMockClientInterface(ctrl)
		processor := NewSwapParaswapProcessor(nil, nil, nil)
		processor.paraswapClient = client

		expectClientFetchPriceRoute(client, paraswap.Route{}, errors.New("paraswap unavailable"))

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})
}

func TestERC1155Processor_PackTxInputDataAndEstimateGas(t *testing.T) {
	amountIn := big.NewInt(2)
	collectible := makeToken(walletCommon.EthereumMainnet, "0x1155")
	collectible.CollectibleTokenID = (*hexutil.Big)(big.NewInt(123))

	params := ProcessorInputParams{
		FromChain: &mainnet,
		FromAddr:  testFromAddr,
		ToAddr:    testToAddr,
		FromToken: collectible,
		AmountIn:  amountIn,
	}

	t.Run("pack and estimate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewERC1155Processor(mockRPCClient, nil)

		input, err := processor.PackTxInputData(params)
		require.NoError(t, err)
		// ERC1155 safeTransferFrom(address,address,uint256,uint256,bytes) selector
		assert.Equal(t, common.FromHex("0xf242432a"), input[:4])

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, msg ethereum.CallMsg) (uint64, error) {
				assert.Equal(t, &collectible.Address, msg.To)
				assert.Equal(t, input, msg.Data)
				return uint64(40000), nil
			})

		estimation, err := processor.EstimateGas(params, input)
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(40000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("pack without a collectible token id fails", func(t *testing.T) {
		processor := NewERC1155Processor(nil, nil)

		_, err := processor.PackTxInputData(ProcessorInputParams{
			FromToken: makeToken(walletCommon.EthereumMainnet, "0x1155"),
		})
		assert.Equal(t, ErrNoTokenSet, err)
	})

	t.Run("eth client getter error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		processor := NewERC1155Processor(mockRPCClient, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(nil, errors.New("no client"))

		_, err := processor.EstimateGas(params, []byte{})
		assert.Error(t, err)
	})
}

func TestStickersBuyProcessor_EstimateGas(t *testing.T) {
	params := ProcessorInputParams{
		FromChain: &mainnet,
		FromAddr:  testFromAddr,
		PackID:    big.NewInt(1),
	}

	t.Run("estimates through the eth client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		processor := NewStickersBuyProcessor(mockRPCClient, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(mockEthClient, nil)
		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(25000), nil)

		estimation, err := processor.EstimateGas(params, []byte{0x01})
		require.NoError(t, err)
		assert.Equal(t, uint64(float64(25000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("eth client getter error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		processor := NewStickersBuyProcessor(mockRPCClient, nil)

		mockRPCClient.EXPECT().EthClient(walletCommon.EthereumMainnet).Return(nil, errors.New("no client"))

		_, err := processor.EstimateGas(params, []byte{0x01})
		assert.Error(t, err)
	})
}
