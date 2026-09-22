package pathprocessor

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	gomock "go.uber.org/mock/gomock"

	sdkTypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/crypto/types"
	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	mock_transactions "github.com/status-im/status-go/internal/transactions/mock"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/relay"
	mock_relay "github.com/status-im/status-go/pkg/services/wallet/thirdparty/relay/mock"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"

	"github.com/stretchr/testify/require"
)

var (
	relayTestUsdc    = common.HexToAddress("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913")
	relayTestRouter  = common.HexToAddress("0xaaaaaaae92cc1ceef79a038017889fdd26d23d4d")
	relayTestSpender = common.HexToAddress("0xf70da97812cb96acdf810712aa562db8dfa3dbef")
)

func relayNativeToken(chainID uint64) tokentypes.Token {
	return tokentypes.Token{Token: &sdkTypes.Token{Symbol: walletCommon.EthSymbol, ChainID: chainID}}
}

func relayUsdcToken(chainID uint64) tokentypes.Token {
	return tokentypes.Token{Token: &sdkTypes.Token{Symbol: walletCommon.UsdcSymbol, Address: relayTestUsdc, ChainID: chainID}}
}

func relayTxStep(id string, to common.Address, data string, value *big.Int, gas *big.Int, chainID uint64) relay.Step {
	txData := relay.StepTxData{
		From:    "0x0000000000000000000000000000000000000003",
		To:      to.Hex(),
		Data:    data,
		Value:   &bigint.BigInt{Int: value},
		ChainID: chainID,
	}
	if gas != nil {
		txData.Gas = &bigint.BigInt{Int: gas}
	}
	return relay.Step{ID: id, Kind: relay.StepKindTransaction, Items: []relay.StepItem{{Status: "incomplete", Data: txData}}}
}

func relayQuoteWith(steps ...relay.Step) relay.Quote {
	return relay.Quote{
		RequestID: "0x1234",
		Steps:     steps,
		Details: relay.Details{
			Operation:    "swap",
			TimeEstimate: 15,
			CurrencyIn:   relay.CurrencyAmount{Amount: &bigint.BigInt{Int: big.NewInt(1000)}},
			CurrencyOut:  relay.CurrencyAmount{Amount: &bigint.BigInt{Int: big.NewInt(2000)}, MinimumAmount: &bigint.BigInt{Int: big.NewInt(1990)}},
		},
	}
}

func newRelayTestProcessor(t *testing.T, ctrl *gomock.Controller) (*RelayProcessor, *mock_relay.MockClientInterface) {
	t.Helper()
	client := mock_relay.NewMockClientInterface(ctrl)
	client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
	processor := NewRelayProcessor(nil, nil, nil, security.SensitiveString{}, relay.ReferrerDev)
	processor.relayClient = client
	return processor, client
}

func relayInputParams(fromToken, toToken *tokentypes.Token, amountIn *big.Int) ProcessorInputParams {
	return ProcessorInputParams{
		FromAddr:  common.HexToAddress("0x0000000000000000000000000000000000000003"),
		ToAddr:    common.HexToAddress("0x0000000000000000000000000000000000000004"),
		FromChain: &params.Network{ChainID: fromToken.ChainID},
		ToChain:   &params.Network{ChainID: toToken.ChainID},
		FromToken: fromToken,
		ToToken:   toToken,
		AmountIn:  amountIn,
	}
}

func TestRelayQuote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processor, client := newRelayTestProcessor(t, ctrl)
	require.Equal(t, pathProcessorCommon.ProcessorRelayName, processor.Name())

	fromToken := relayNativeToken(walletCommon.EthereumMainnet)
	toToken := relayUsdcToken(walletCommon.EthereumMainnet)
	amountIn := big.NewInt(1000)
	testInputParams := relayInputParams(&fromToken, &toToken, amountIn)

	testQuote := relayQuoteWith(relayTxStep("swap", relayTestRouter, "0xabcd", amountIn, nil, walletCommon.EthereumMainnet))

	available, err := processor.AvailableFor(testInputParams)
	require.NoError(t, err)
	require.True(t, available)

	bonderFees, tokenFees, err := processor.CalculateFees(testInputParams)
	require.NoError(t, err)
	require.Equal(t, int64(0), bonderFees.Int64())
	require.Equal(t, int64(0), tokenFees.Int64())

	key := pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn)
	processor.quotes.Store(key, &testQuote)

	amountOut, err := processor.CalculateAmountOut(testInputParams)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), amountOut.Uint64())

	// warm cache: no re-quote within a routing round
	contractAddress, err := processor.GetContractAddress(testInputParams)
	require.NoError(t, err)
	require.Equal(t, relayTestRouter, contractAddress)

	inputData, err := processor.PackTxInputData(testInputParams)
	require.NoError(t, err)
	require.Equal(t, "0xabcd", hexutil.Encode(inputData))

	require.Equal(t, uint(15), processor.GetRouteExecutionDuration(testInputParams))

	// a cleared processor fetches exactly once and the round shares that fetch
	processor.Clear()
	client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(testQuote, nil).Times(1)

	contractAddress, err = processor.GetContractAddress(testInputParams)
	require.NoError(t, err)
	require.Equal(t, relayTestRouter, contractAddress)

	inputData, err = processor.PackTxInputData(testInputParams)
	require.NoError(t, err)
	require.Equal(t, "0xabcd", hexutil.Encode(inputData))

	amountOut, err = processor.CalculateAmountOut(testInputParams)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), amountOut.Uint64())
}

func TestRelayQuoteRequestParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processor, client := newRelayTestProcessor(t, ctrl)

	fromToken := relayNativeToken(walletCommon.EthereumMainnet)
	toToken := relayUsdcToken(walletCommon.OptimismMainnet)
	amountIn := big.NewInt(1000)
	p := relayInputParams(&fromToken, &toToken, amountIn)
	p.SlippagePercentage = 0.5

	client.EXPECT().FetchQuote(gomock.Any(), relay.QuoteParams{
		FromChainID:        walletCommon.EthereumMainnet,
		ToChainID:          walletCommon.OptimismMainnet,
		FromToken:          walletCommon.ZeroAddress(),
		ToToken:            relayTestUsdc,
		FromAddress:        p.FromAddr,
		ToAddress:          p.ToAddr,
		AmountIn:           amountIn,
		SlippagePercentage: 0.5,
	}).Return(relayQuoteWith(relayTxStep("deposit", relayTestRouter, "0xabcd", amountIn, nil, walletCommon.EthereumMainnet)), nil)

	_, err := processor.GetContractAddress(p)
	require.NoError(t, err)
}

func TestRelayBridgeAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processor, _ := newRelayTestProcessor(t, ctrl)

	fromToken := relayUsdcToken(walletCommon.EthereumMainnet)
	toToken := relayUsdcToken(walletCommon.OptimismMainnet)

	available, err := processor.AvailableFor(relayInputParams(&fromToken, &toToken, big.NewInt(1000)))
	require.NoError(t, err)
	require.True(t, available)
}

func TestRelaySameChainSameTokenUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processor, _ := newRelayTestProcessor(t, ctrl)

	fromToken := relayUsdcToken(walletCommon.EthereumMainnet)
	toToken := relayUsdcToken(walletCommon.EthereumMainnet)

	_, err := processor.AvailableFor(relayInputParams(&fromToken, &toToken, big.NewInt(1000)))
	require.Equal(t, ErrFromAndToTokensMustBeDifferent, err)

	_, err = processor.AvailableFor(ProcessorInputParams{FromToken: &fromToken})
	require.Equal(t, ErrToAndFromTokensMustBeSet, err)
}

func TestRelayBuySideUnsupported(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processor, _ := newRelayTestProcessor(t, ctrl)

	fromToken := relayNativeToken(walletCommon.EthereumMainnet)
	toToken := relayUsdcToken(walletCommon.EthereumMainnet)
	p := relayInputParams(&fromToken, &toToken, nil)
	p.AmountOut = big.NewInt(2000)

	available, err := processor.AvailableFor(p)
	require.NoError(t, err)
	require.False(t, available)
}

func TestRelayErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processor, client := newRelayTestProcessor(t, ctrl)

	fromToken := relayNativeToken(walletCommon.EthereumMainnet)
	toToken := relayUsdcToken(walletCommon.EthereumMainnet)
	p := relayInputParams(&fromToken, &toToken, big.NewInt(1000))

	testCases := []struct {
		clientError    error
		processorError error
	}{
		// (quote errors and the swap routing & pricing execution errors, see docs.relay.link)
		{&relay.APIError{Code: "NO_SWAP_ROUTES_FOUND", Message: "No routes found"}, ErrNoRoutesFound},
		{&relay.APIError{Code: "NO_INTERNAL_SWAP_ROUTES_FOUND", Message: "x"}, ErrNoRoutesFound},
		{&relay.APIError{Code: "UNSUPPORTED_ROUTE", Message: "x"}, ErrNoRoutesFound},
		{&relay.APIError{Code: "INSUFFICIENT_LIQUIDITY", Message: "x"}, ErrNotEnoughLiquidity},
		{&relay.APIError{Code: "INSUFFICIENT_POOL_LIQUIDITY", Message: "x"}, ErrNotEnoughLiquidity},
		{&relay.APIError{Code: "NO_QUOTES", Message: "x"}, ErrNoQuotesAvailable},
		{&relay.APIError{Code: "GENERATE_SWAP_FAILED", Message: "x"}, ErrNoQuotesAvailable},
		{&relay.APIError{Code: "REVERSE_SWAP_FAILED", Message: "x"}, ErrNoQuotesAvailable},
		{&relay.APIError{Code: "SWAP_IMPACT_TOO_HIGH", Message: "x"}, ErrPriceImpactTooHigh},
		{&relay.APIError{Code: "SLIPPAGE", Message: "x"}, ErrSlippageExceeded},
		{&relay.APIError{Code: "TOO_LITTLE_RECEIVED", Message: "x"}, ErrSlippageExceeded},
		{&relay.APIError{Code: "AMOUNT_TOO_LOW", Message: "x"}, ErrAmountTooLow},
		{&relay.APIError{Code: "DEPOSITED_AMOUNT_TOO_LOW_TO_FILL", Message: "x"}, ErrAmountTooLow},
		{&relay.APIError{Code: "AMOUNT_TOO_HIGH", Message: "x"}, ErrAmountTooHigh},
	}

	for _, tc := range testCases {
		client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(relay.Quote{}, tc.clientError)
		_, err := processor.GetContractAddress(p)
		require.Equal(t, tc.processorError.Error(), err.Error())
	}

	// anything else surfaces as the Relay custom error carrying the API message
	client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(relay.Quote{}, &relay.APIError{Code: "SANCTIONED_CURRENCY", Message: "blocked"})
	_, err := processor.GetContractAddress(p)
	require.Equal(t, ErrRelayCustomError, err)
	require.Contains(t, err.Error(), "SANCTIONED_CURRENCY")

	client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(relay.Quote{}, errors.New("network down"))
	_, err = processor.GetContractAddress(p)
	require.Equal(t, ErrRelayCustomError, err)
}

func TestRelayGetContractAddress(t *testing.T) {
	approveData, err := walletCommon.PackApprovalInputData(big.NewInt(1000), &relayTestSpender)
	require.NoError(t, err)
	approveHex := hexutil.Encode(approveData)

	usdc := relayUsdcToken(walletCommon.BaseMainnet)
	eth := relayNativeToken(walletCommon.BaseMainnet)
	amountIn := big.NewInt(1000)

	testCases := []struct {
		name      string
		fromToken tokentypes.Token
		toToken   tokentypes.Token
		quote     relay.Quote
		expected  common.Address
		wantErr   bool
	}{
		{
			name:      "approve step yields the decoded spender",
			fromToken: usdc, toToken: eth,
			quote: relayQuoteWith(
				relayTxStep("approve", relayTestUsdc, approveHex, big.NewInt(0), nil, walletCommon.BaseMainnet),
				relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.BaseMainnet)),
			expected: relayTestSpender,
		},
		{
			name:      "approve step for another contract is rejected",
			fromToken: usdc, toToken: eth,
			quote: relayQuoteWith(
				relayTxStep("approve", relayTestRouter, approveHex, big.NewInt(0), nil, walletCommon.BaseMainnet),
				relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.BaseMainnet)),
			wantErr: true,
		},
		{
			name:      "erc20 without approve step uses the main step target",
			fromToken: usdc, toToken: eth,
			quote:    relayQuoteWith(relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.BaseMainnet)),
			expected: relayTestRouter,
		},
		{
			name:      "erc20 direct transfer deposit needs no allowance",
			fromToken: usdc, toToken: eth,
			quote:    relayQuoteWith(relayTxStep("deposit", relayTestUsdc, "0xa9059cbb", big.NewInt(0), nil, walletCommon.BaseMainnet)),
			expected: walletCommon.ZeroAddress(),
		},
		{
			name:      "native uses the main step target",
			fromToken: eth, toToken: usdc,
			quote:    relayQuoteWith(relayTxStep("deposit", relayTestSpender, "0x", amountIn, nil, walletCommon.BaseMainnet)),
			expected: relayTestSpender,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			processor, client := newRelayTestProcessor(t, ctrl)
			client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(tc.quote, nil)

			fromToken, toToken := tc.fromToken, tc.toToken
			got, err := processor.GetContractAddress(relayInputParams(&fromToken, &toToken, amountIn))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestRelayUnsupportedQuote(t *testing.T) {
	usdc := relayUsdcToken(walletCommon.BaseMainnet)
	eth := relayNativeToken(walletCommon.BaseMainnet)
	amountIn := big.NewInt(1000)

	signatureStep := relay.Step{ID: "authorize", Kind: relay.StepKindSignature, Items: []relay.StepItem{{}}}

	testCases := []struct {
		name  string
		quote relay.Quote
	}{
		{"signature step", relayQuoteWith(signatureStep, relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.BaseMainnet))},
		{"no transaction step", relayQuoteWith()},
		{"only an approve step", relayQuoteWith(relayTxStep("approve", relayTestUsdc, "0x095ea7b3", big.NewInt(0), nil, walletCommon.BaseMainnet))},
		{"approve after the main step", relayQuoteWith(
			relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.BaseMainnet),
			relayTxStep("approve", relayTestUsdc, "0x095ea7b3", big.NewInt(0), nil, walletCommon.BaseMainnet))},
		{"main step on another chain", relayQuoteWith(relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.OptimismMainnet))},
		// only the first main step would be built and sent, leaving the quoted route half executed
		{"two main transaction steps", relayQuoteWith(
			relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.BaseMainnet),
			relayTxStep("send", relayTestSpender, "0xef01", big.NewInt(0), nil, walletCommon.BaseMainnet))},
		{"step without items", relayQuoteWith(relay.Step{ID: "swap", Kind: relay.StepKindTransaction})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			processor, client := newRelayTestProcessor(t, ctrl)
			client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(tc.quote, nil)

			fromToken, toToken := usdc, eth
			_, err := processor.GetContractAddress(relayInputParams(&fromToken, &toToken, amountIn))
			require.Error(t, err)
			require.Equal(t, ErrRelayCustomError, err)
		})
	}
}

func TestRelayEstimateGas(t *testing.T) {
	amountIn := big.NewInt(1000)

	newProcessorWithEthClient := func(t *testing.T, ctrl *gomock.Controller) (*RelayProcessor, *mock_ethclient.MockEthClientInterface) {
		client := mock_relay.NewMockClientInterface(ctrl)
		client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
		mockRPCClient := mock_rpcclient.NewMockClientInterface(ctrl)
		mockEthClient := mock_ethclient.NewMockEthClientInterface(ctrl)
		mockRPCClient.EXPECT().EthClient(gomock.Any()).Return(mockEthClient, nil).AnyTimes()

		processor := NewRelayProcessor(mockRPCClient, nil, nil, security.SensitiveString{}, relay.ReferrerDev)
		processor.relayClient = client
		return processor, mockEthClient
	}

	t.Run("native token estimates against the main step target and value", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		processor, mockEthClient := newProcessorWithEthClient(t, ctrl)

		fromToken := relayNativeToken(walletCommon.EthereumMainnet)
		toToken := relayUsdcToken(walletCommon.EthereumMainnet)
		p := relayInputParams(&fromToken, &toToken, amountIn)
		quote := relayQuoteWith(relayTxStep("deposit", relayTestSpender, "0xabcd", amountIn, nil, walletCommon.EthereumMainnet))
		processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ interface{}, msg interface{}) (uint64, error) {
				callMsg := msg.(ethereum.CallMsg)
				require.Equal(t, relayTestSpender, *callMsg.To)
				require.Equal(t, 0, amountIn.Cmp(callMsg.Value))
				require.Equal(t, []byte{0xab, 0xcd}, callMsg.Data)
				return uint64(80000), nil
			})

		estimation, err := processor.EstimateGas(p, []byte{0xab, 0xcd})
		require.NoError(t, err)
		require.Equal(t, uint64(float64(80000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("bridge uses the bridge gas factor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		processor, mockEthClient := newProcessorWithEthClient(t, ctrl)

		fromToken := relayNativeToken(walletCommon.EthereumMainnet)
		toToken := relayNativeToken(walletCommon.OptimismMainnet)
		p := relayInputParams(&fromToken, &toToken, amountIn)
		quote := relayQuoteWith(relayTxStep("deposit", relayTestSpender, "0x", amountIn, nil, walletCommon.EthereumMainnet))
		processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(21000), nil)

		estimation, err := processor.EstimateGas(p, []byte{})
		require.NoError(t, err)
		require.Equal(t, uint64(float64(21000)*pathProcessorCommon.IncreaseEstimatedGasFactorForBridge), estimation)
	})

	t.Run("erc20 estimation error falls back to the quoted gas", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		processor, mockEthClient := newProcessorWithEthClient(t, ctrl)

		fromToken := relayUsdcToken(walletCommon.EthereumMainnet)
		toToken := relayNativeToken(walletCommon.EthereumMainnet)
		p := relayInputParams(&fromToken, &toToken, amountIn)
		quote := relayQuoteWith(relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), big.NewInt(150000), walletCommon.EthereumMainnet))
		processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).
			Return(uint64(0), errors.New("execution reverted: ERC20: transfer amount exceeds allowance"))

		estimation, err := processor.EstimateGas(p, []byte{})
		require.NoError(t, err)
		require.Equal(t, uint64(float64(150000)*pathProcessorCommon.IncreaseEstimatedGasFactor), estimation)
	})

	t.Run("erc20 estimation error without quoted gas is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		processor, mockEthClient := newProcessorWithEthClient(t, ctrl)

		fromToken := relayUsdcToken(walletCommon.EthereumMainnet)
		toToken := relayNativeToken(walletCommon.EthereumMainnet)
		p := relayInputParams(&fromToken, &toToken, amountIn)
		quote := relayQuoteWith(relayTxStep("swap", relayTestRouter, "0xabcd", big.NewInt(0), nil, walletCommon.EthereumMainnet))
		processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(0), errors.New("execution reverted"))

		_, err := processor.EstimateGas(p, []byte{})
		require.Error(t, err)
	})

	t.Run("native estimation error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		processor, mockEthClient := newProcessorWithEthClient(t, ctrl)

		fromToken := relayNativeToken(walletCommon.EthereumMainnet)
		toToken := relayUsdcToken(walletCommon.EthereumMainnet)
		p := relayInputParams(&fromToken, &toToken, amountIn)
		quote := relayQuoteWith(relayTxStep("deposit", relayTestSpender, "0x", amountIn, big.NewInt(150000), walletCommon.EthereumMainnet))
		processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

		mockEthClient.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(uint64(0), errors.New("estimation failed"))

		_, err := processor.EstimateGas(p, []byte{})
		require.Error(t, err)
	})
}

func TestRelayBuildTransactionV2(t *testing.T) {
	amountIn := big.NewInt(1000)
	fromToken := relayNativeToken(walletCommon.BaseMainnet)
	toToken := relayUsdcToken(walletCommon.BaseMainnet)
	quote := relayQuoteWith(relayTxStep("deposit", relayTestSpender, "0xabcd", amountIn, big.NewInt(150000), walletCommon.BaseMainnet))

	maxFee := (*hexutil.Big)(big.NewInt(30))
	priorityFee := (*hexutil.Big)(big.NewInt(2))

	baseArgs := func() *wallettypes.SendTxArgs {
		return &wallettypes.SendTxArgs{
			From:                 types.HexToAddress("0x0000000000000000000000000000000000000003"),
			ValueIn:              (*hexutil.Big)(amountIn),
			FromToken:            &fromToken,
			ToToken:              &toToken,
			MaxFeePerGas:         maxFee,
			MaxPriorityFeePerGas: priorityFee,
		}
	}

	run := func(t *testing.T, sendArgs *wallettypes.SendTxArgs) wallettypes.SendTxArgs {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := mock_relay.NewMockClientInterface(ctrl)
		client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
		transactor := mock_transactions.NewMockTransactorIface(ctrl)

		processor := NewRelayProcessor(nil, transactor, nil, security.SensitiveString{}, relay.ReferrerDev)
		processor.relayClient = client
		processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

		var built wallettypes.SendTxArgs
		transactor.EXPECT().ValidateAndBuildTransaction(walletCommon.BaseMainnet, gomock.Any(), int64(-1)).
			DoAndReturn(func(_ uint64, args wallettypes.SendTxArgs, _ int64) (*ethTypes.Transaction, uint64, error) {
				built = args
				return &ethTypes.Transaction{}, 7, nil
			})

		_, nonce, err := processor.BuildTransactionV2(sendArgs, -1)
		require.NoError(t, err)
		require.Equal(t, uint64(7), nonce)
		return built
	}

	t.Run("maps the main step and keeps the router fee mode", func(t *testing.T) {
		built := run(t, baseArgs())
		require.Equal(t, walletCommon.BaseMainnet, built.FromChainID)
		require.Equal(t, types.Address(relayTestSpender), *built.To)
		require.Equal(t, 0, amountIn.Cmp(built.Value.ToInt()))
		require.Equal(t, types.HexBytes{0xab, 0xcd}, built.Data)
		require.Equal(t, maxFee, built.MaxFeePerGas)
		require.Equal(t, priorityFee, built.MaxPriorityFeePerGas)
		require.Nil(t, built.GasPrice)
		// gas comes from the quote only when the router didn't set it
		require.Equal(t, uint64(150000), uint64(*built.Gas))
	})

	t.Run("keeps the gas amount chosen by the router", func(t *testing.T) {
		args := baseArgs()
		gas := hexutil.Uint64(99000)
		args.Gas = &gas
		built := run(t, args)
		require.Equal(t, uint64(99000), uint64(*built.Gas))
	})
}

func TestRelayBuildTransactionV2RefetchesQuote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	amountIn := big.NewInt(1000)
	fromToken := relayNativeToken(walletCommon.BaseMainnet)
	toToken := relayUsdcToken(walletCommon.BaseMainnet)
	quote := relayQuoteWith(relayTxStep("deposit", relayTestSpender, "0xabcd", amountIn, nil, walletCommon.BaseMainnet))

	client := mock_relay.NewMockClientInterface(ctrl)
	client.EXPECT().SetChainID(gomock.Any()).AnyTimes()
	client.EXPECT().FetchQuote(gomock.Any(), gomock.Any()).Return(quote, nil).Times(1)
	transactor := mock_transactions.NewMockTransactorIface(ctrl)
	transactor.EXPECT().ValidateAndBuildTransaction(walletCommon.BaseMainnet, gomock.Any(), int64(-1)).Return(&ethTypes.Transaction{}, uint64(0), nil)

	processor := NewRelayProcessor(nil, transactor, nil, security.SensitiveString{}, relay.ReferrerDev)
	processor.relayClient = client

	to := types.HexToAddress("0x0000000000000000000000000000000000000004")
	_, _, err := processor.BuildTransactionV2(&wallettypes.SendTxArgs{
		From:      types.HexToAddress("0x0000000000000000000000000000000000000003"),
		To:        &to,
		ValueIn:   (*hexutil.Big)(amountIn),
		FromToken: &fromToken,
		ToToken:   &toToken,
	}, -1)
	require.NoError(t, err)
}

func TestRelayProviderTool(t *testing.T) {
	fromToken := relayNativeToken(walletCommon.BaseMainnet)
	toToken := relayUsdcToken(walletCommon.BaseMainnet)
	amountIn := big.NewInt(1000)

	testCases := []struct {
		name        string
		origin      string
		destination string
		expected    string
	}{
		{"same-chain swap through a dex", "0x", "0x", "0x"},
		{"swap executed on the destination leg", "relay", "kyberswap", "kyberswap"},
		{"origin dex wins over destination", "uniswap", "kyberswap", "uniswap"},
		{"pure bridge on relay liquidity", "relay", "relay", ""},
		{"no route information", "", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			processor, _ := newRelayTestProcessor(t, ctrl)
			quote := relayQuoteWith(relayTxStep("swap", relayTestRouter, "0xabcd", amountIn, nil, walletCommon.BaseMainnet))
			quote.Details.Route = relay.Route{
				Origin:      relay.RouteLeg{Router: tc.origin},
				Destination: relay.RouteLeg{Router: tc.destination},
			}
			processor.quotes.Store(pathProcessorCommon.MakeKey(fromToken.Key(), toToken.Key(), amountIn), &quote)

			require.Equal(t, tc.expected, processor.GetProviderTool(relayInputParams(&fromToken, &toToken, amountIn)))
		})
	}
}
