package pathprocessor

import (
	"fmt"
	"testing"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/token"

	"github.com/stretchr/testify/assert"
)

var mainnet = params.Network{
	ChainID:                walletCommon.EthereumMainnet,
	ChainName:              "Mainnet",
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
				FromToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
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
				FromToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToTokensMustBeDifferent,
				},
			},
		},
		{
			name: "Same Chains Set - FormToken Set - ToToken Set - Different Tokens",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &mainnet,
				FromToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: walletCommon.UsdcSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
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
				FromToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: nil,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      true,
					expectedError: nil,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeSame,
				},
			},
		},
		{
			name: "Different Chains Set - FormToken Set - ToToken Set - Same Tokens",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &optimism,
				FromToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				pathProcessorCommon.ProcessorSwapParaswapName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeSame,
				},
			},
		},
		{
			name: "Different Chains Set - FormToken Set - ToToken Set - Different Tokens",
			input: ProcessorInputParams{
				TestsMode: true,
				FromChain: &mainnet,
				ToChain:   &optimism,
				FromToken: &token.Token{
					Symbol: walletCommon.EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: walletCommon.UsdcSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				pathProcessorCommon.ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
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

				assert.Equal(t, processorName, processor.Name())
				result, err := processor.AvailableFor(tt.input)
				if expResult.expectedError != nil {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, expResult.expected, result)

				if tt.input.TestEstimationMap != nil {
					estimatedGas, err := processor.EstimateGas(tt.input)
					assert.NoError(t, err)
					assert.Greater(t, estimatedGas, uint64(0))

					input := tt.input
					input.TestEstimationMap = map[string]requests.Estimation{
						"randomName": {Value: 10000},
					}
					estimatedGas, err = processor.EstimateGas(input)
					assert.Error(t, err)
					assert.Equal(t, ErrNoEstimationFound, err)
					assert.Equal(t, uint64(0), estimatedGas)
				} else {
					estimatedGas, err := processor.EstimateGas(tt.input)
					assert.Error(t, err)
					assert.Equal(t, ErrNoEstimationFound, err)
					assert.Equal(t, uint64(0), estimatedGas)
				}
			})
		}
	}
}
