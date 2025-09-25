package pathprocessor

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"

	"github.com/stretchr/testify/assert"
)

var mainnet = params.Network{
	ChainID:                walletCommon.EthereumMainnet,
	ChainName:              "Ethereum",
	BlockExplorerURL:       "https://etherscan.io/",
	IconURL:                "network/Network=Ethereum",
	ChainColor:             "#627EEA",
	ShortName:              "eth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  1,
	Enabled:                true,
	RelatedChainID:         walletCommon.EthereumMainnet,
}

var optimism = params.Network{
	ChainID:                walletCommon.OptimismMainnet,
	ChainName:              "Optimism",
	BlockExplorerURL:       "https://optimistic.etherscan.io",
	IconURL:                "network/Network=Optimism",
	ChainColor:             "#E90101",
	ShortName:              "oeth",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  2,
	Enabled:                true,
	RelatedChainID:         walletCommon.OptimismMainnet,
}

var base = params.Network{
	ChainID:                walletCommon.BaseMainnet,
	ChainName:              "Base",
	BlockExplorerURL:       "https://basescan.org",
	IconURL:                "network/Network=Base",
	ChainColor:             "#0052FF",
	ShortName:              "base",
	NativeCurrencyName:     "Ether",
	NativeCurrencySymbol:   "ETH",
	NativeCurrencyDecimals: 18,
	IsTest:                 false,
	Layer:                  2,
	Enabled:                true,
	RelatedChainID:         walletCommon.BaseMainnet,
}

var testEstimationMap = map[string]requests.Estimation{
	pathProcessorCommon.ProcessorTransferName:     {Value: uint64(1000)},
	pathProcessorCommon.ProcessorBridgeHopName:    {Value: uint64(5000)},
	pathProcessorCommon.ProcessorSwapParaswapName: {Value: uint64(2000)},
}

type expectedResult struct {
	expected      bool
	expectedError error
}

func TestPathProcessors(t *testing.T) {
	tests := []struct {
		name          string
		input         ProcessorInputParams
		expectedError error
		expected      map[string]expectedResult
	}{
		{
			name: "Empty Input Params",
			input: ProcessorInputParams{
				TestsMode: true,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrNoChainSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrNoChainSet,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrNoChainSet,
				},
			},
		},
		{
			name: "Same Chains Set - No FormToken - No ToToken",
			input: ProcessorInputParams{
				TestsMode:         true,
				FromChain:         &mainnet,
				ToChain:           &mainnet,
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
			},
		},
		{
			name: "Same Chains Set - FormToken Set - No ToToken",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &mainnet,
				FromToken: &tokentypes.Token{Token: &types.Token{
					Symbol: walletCommon.EthSymbol,
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeDifferent,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
			},
		},
		{
			name: "Same Chains Set - FormToken Set - ToToken Set - Same Tokens",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &mainnet,
				FromToken: &tokentypes.Token{Token: &types.Token{
					Symbol: walletCommon.EthSymbol,
				}},
				ToToken: &tokentypes.Token{Token: &types.Token{
					Symbol: walletCommon.EthSymbol,
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      true,
					expectedError: nil,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeDifferent,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToTokensMustBeDifferent,
				},
			},
		},
		{
			name: "Same Chains Set - FormToken Set - ToToken Set - Different Tokens On Same Chain",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &mainnet,
				FromToken: &tokentypes.Token{Token: &types.Token{
					ChainID: walletCommon.EthereumMainnet,
					Address: common.HexToAddress("0x1"),
				}},
				ToToken: &tokentypes.Token{Token: &types.Token{
					ChainID: walletCommon.EthereumMainnet,
					Address: common.HexToAddress("0x2"),
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrFromAndToTokensMustBeSame,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeDifferent,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      true,
					expectedError: nil,
				},
			},
		},
		{
			name: "Different Chains Set - No FormToken - No ToToken",
			input: ProcessorInputParams{
				TestsMode:         true,
				FromChain:         &mainnet,
				ToChain:           &optimism,
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeSame,
				},
			},
		},
		{
			name: "Different Chains Set - FormToken Set - No ToToken",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &optimism,
				FromToken: &tokentypes.Token{Token: &types.Token{
					Symbol: walletCommon.EthSymbol,
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
			},
		},
		{
			name: "Different Chains Set - FormToken Set - ToToken Set - Same Tokens",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &optimism,
				FromToken: &tokentypes.Token{Token: &types.Token{
					ChainID: walletCommon.EthereumMainnet,
					Address: common.HexToAddress("0x1"),
				}},
				ToToken: &tokentypes.Token{Token: &types.Token{
					ChainID: walletCommon.EthereumMainnet,
					Address: common.HexToAddress("0x1"),
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      true,
					expectedError: nil,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeDifferent,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToTokensMustBeDifferent,
				},
			},
		},
		{
			name: "Different Chains Set - FormToken Set - No ToToken - Token Not Supported On ToChain",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &optimism,
				ToChain:   &base,
				FromToken: &tokentypes.Token{Token: &types.Token{
					Symbol: walletCommon.DaiSymbol,
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToChainNotSupported,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrToAndFromTokensMustBeSet,
				},
			},
		},
		{
			name: "Different Chains Set - FormToken Set - ToToken Set - Different Tokens On Different Chains",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &optimism,
				FromToken: &tokentypes.Token{Token: &types.Token{
					ChainID: walletCommon.EthereumMainnet,
					Address: common.HexToAddress("0x1"),
				}},
				ToToken: &tokentypes.Token{Token: &types.Token{
					ChainID: walletCommon.OptimismMainnet,
					Address: common.HexToAddress("0x1"),
				}},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrFromAndToTokensMustBeSame,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToChainNotSupported,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeSame,
				},
			},
		},
	}

	for _, tt := range tests {
		for processorName, expResult := range tt.expected {
			t.Run(fmt.Sprintf("%s[%s]", processorName, tt.name), func(t *testing.T) {

				var processor PathProcessor
				if processorName == pathProcessorCommon.ProcessorTransferName {
					processor = NewTransferProcessor(nil, nil)
				} else if processorName == pathProcessorCommon.ProcessorBridgeHopName {
					processor = NewHopBridgeProcessor(nil, nil, nil, nil)
				} else if processorName == pathProcessorCommon.ProcessorSwapParaswapName {
					processor = NewSwapParaswapProcessor(nil, nil, nil)
				}

				fmt.Println("\n\nprocessor.Name()", processor.Name())
				fmt.Printf("\n\ninput: %+v", tt.input)

				assert.Equal(t, processorName, processor.Name())
				result, err := processor.AvailableFor(tt.input)
				fmt.Println("\n\nresult", result)
				fmt.Println("\n\nerr", err)
				if expResult.expectedError != nil {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, expResult.expected, result)

				if tt.input.TestEstimationMap != nil {
					inputData, err := processor.PackTxInputData(tt.input)
					assert.NoError(t, err)
					estimatedGas, err := processor.EstimateGas(tt.input, inputData)
					assert.NoError(t, err)
					assert.Greater(t, estimatedGas, uint64(0))

					input := tt.input
					input.TestEstimationMap = map[string]requests.Estimation{
						"randomName": {Value: 10000},
					}
					inputData, err = processor.PackTxInputData(tt.input)
					assert.NoError(t, err)
					estimatedGas, err = processor.EstimateGas(input, inputData)
					assert.Error(t, err)
					assert.Equal(t, ErrNoEstimationFound, err)
					assert.Equal(t, uint64(0), estimatedGas)
				} else {
					inputData, err := processor.PackTxInputData(tt.input)
					assert.NoError(t, err)
					estimatedGas, err := processor.EstimateGas(tt.input, inputData)
					assert.Error(t, err)
					assert.Equal(t, ErrNoEstimationFound, err)
					assert.Equal(t, uint64(0), estimatedGas)
				}
			})
		}
	}
}
