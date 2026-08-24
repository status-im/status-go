package pathprocessor

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"

	"github.com/stretchr/testify/assert"
)

var mainnet = params.Network{
	ChainID:                walletCommon.EthereumMainnet,
	ChainName:              "Ethereum",
	BlockExplorerURL:       "https://etherscan.io/",
	IconURL:                "network/ethereum",
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
	IconURL:                "network/optimism",
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

// USDC addresses present in the hop bridge contract tables (internal/contracts/hop/address.go).
var (
	usdcMainnet = &tokentypes.Token{Token: &types.Token{
		ChainID: walletCommon.EthereumMainnet,
		Address: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		Symbol:  walletCommon.UsdcSymbolEVM,
	}}
	usdcOptimism = &tokentypes.Token{Token: &types.Token{
		ChainID: walletCommon.OptimismMainnet,
		Address: common.HexToAddress("0x0b2c639c533813f4aa9d7837caf62653d097ff85"),
		Symbol:  walletCommon.UsdcSymbolEVM,
	}}
)

func makeToken(chainID uint64, address string) *tokentypes.Token {
	return &tokentypes.Token{Token: &types.Token{
		ChainID: chainID,
		Address: common.HexToAddress(address),
	}}
}

type expectedResult struct {
	expected      bool
	expectedError error
}

func TestPathProcessors_AvailableFor(t *testing.T) {
	tests := []struct {
		name     string
		input    ProcessorInputParams
		expected map[string]expectedResult
	}{
		{
			name:  "No Tokens Set",
			input: ProcessorInputParams{},
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
			name: "Only FromToken Set",
			input: ProcessorInputParams{
				FromToken: makeToken(walletCommon.EthereumMainnet, "0x1"),
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
			name: "Same Token On Same Chain",
			input: ProcessorInputParams{
				FromToken: makeToken(walletCommon.EthereumMainnet, "0x1"),
				ToToken:   makeToken(walletCommon.EthereumMainnet, "0x1"),
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
			name: "Different Tokens On Same Chain",
			input: ProcessorInputParams{
				FromToken: makeToken(walletCommon.EthereumMainnet, "0x1"),
				ToToken:   makeToken(walletCommon.EthereumMainnet, "0x2"),
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
			name: "Same-Address Tokens On Different Chains - Not Supported By Hop",
			input: ProcessorInputParams{
				FromToken: makeToken(walletCommon.EthereumMainnet, "0x1"),
				ToToken:   makeToken(walletCommon.OptimismMainnet, "0x1"),
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
		{
			name: "USDC Bridged Between Mainnet And Optimism - Supported By Hop",
			input: ProcessorInputParams{
				FromToken: usdcMainnet,
				ToToken:   usdcOptimism,
			},
			expected: map[string]expectedResult{
				pathProcessorCommon.ProcessorTransferName: {
					expected:      false,
					expectedError: ErrFromAndToTokensMustBeSame,
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
	}

	for _, tt := range tests {
		for processorName, expResult := range tt.expected {
			t.Run(fmt.Sprintf("%s[%s]", processorName, tt.name), func(t *testing.T) {
				var processor PathProcessor
				switch processorName {
				case pathProcessorCommon.ProcessorTransferName:
					processor = NewTransferProcessor(nil, nil)
				case pathProcessorCommon.ProcessorBridgeHopName:
					processor = NewHopBridgeProcessor(nil, nil, nil, nil)
				case pathProcessorCommon.ProcessorSwapParaswapName:
					processor = NewSwapParaswapProcessor(nil, nil, nil)
				}

				assert.Equal(t, processorName, processor.Name())
				result, err := processor.AvailableFor(tt.input)
				assert.Equal(t, expResult.expectedError, err)
				assert.Equal(t, expResult.expected, result)
			})
		}
	}
}

func TestHopBridgeProcessor_AvailableFor_FromChainNotSupported(t *testing.T) {
	processor := NewHopBridgeProcessor(nil, nil, nil, nil)

	// ToToken is supported by hop, FromToken is not — the raw hop contracts error is returned.
	result, err := processor.AvailableFor(ProcessorInputParams{
		FromToken: makeToken(walletCommon.EthereumMainnet, "0x1"),
		ToToken:   usdcOptimism,
	})
	assert.Error(t, err)
	assert.False(t, result)
}
