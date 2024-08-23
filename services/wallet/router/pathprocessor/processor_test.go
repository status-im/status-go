package pathprocessor

import (
	"fmt"
	"testing"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/token"

	"github.com/stretchr/testify/assert"
)

var mainnet = params.Network{
	ChainID:                walletCommon.EthereumMainnet,
	ChainName:              "Mainnet",
	RPCURL:                 "https://eth-archival.rpc.grove.city/v1/",
	FallbackURL:            "https://mainnet.infura.io/v3/",
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
	RPCURL:                 "https://optimism-mainnet.rpc.grove.city/v1/",
	FallbackURL:            "https://optimism-mainnet.infura.io/v3/",
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

var testEstimationMap = map[string]Estimation{
	ProcessorTransferName:     {uint64(1000), nil},
	ProcessorBridgeHopName:    {uint64(5000), nil},
	ProcessorSwapParaswapName: {uint64(2000), nil},
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
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrNoChainSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrNoChainSet,
				},
				ProcessorSwapParaswapName: {
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
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				ProcessorSwapParaswapName: {
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
					Symbol: EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				ProcessorTransferName: {
					expected:      true,
					expectedError: nil,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrFromAndToChainsMustBeDifferent,
				},
				ProcessorSwapParaswapName: {
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
					Symbol: EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorSwapParaswapName: {
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
					Symbol: EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: UsdcSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorSwapParaswapName: {
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
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrNoTokenSet,
				},
				ProcessorSwapParaswapName: {
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
					Symbol: EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				ProcessorTransferName: {
					expected:      false,
					expectedError: nil,
				},
				ProcessorBridgeHopName: {
					expected:      true,
					expectedError: nil,
				},
				ProcessorSwapParaswapName: {
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
					Symbol: EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: EthSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorSwapParaswapName: {
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
					Symbol: EthSymbol,
				},
				ToToken: &token.Token{
					Symbol: UsdcSymbol,
				},
				TestEstimationMap: testEstimationMap,
			},
			expected: map[string]expectedResult{
				ProcessorTransferName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorBridgeHopName: {
					expected:      false,
					expectedError: ErrToTokenShouldNotBeSet,
				},
				ProcessorSwapParaswapName: {
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
				if processorName == ProcessorTransferName {
					processor = NewTransferProcessor(nil, nil)
				} else if processorName == ProcessorBridgeHopName {
					processor = NewHopBridgeProcessor(nil, nil, nil, nil)
				} else if processorName == ProcessorSwapParaswapName {
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
					input.TestEstimationMap = map[string]Estimation{
						"randomName": {10000, nil},
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
