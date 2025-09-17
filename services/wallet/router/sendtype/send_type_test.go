package sendtype

import (
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
)

func TestSendType_CategoryMethods(t *testing.T) {
	tests := map[string]struct {
		method    func(SendType) bool
		trueCases []SendType
	}{
		"IsCollectiblesTransfer": {
			method:    SendType.IsCollectiblesTransfer,
			trueCases: []SendType{ERC721Transfer, ERC1155Transfer},
		},
		"IsEnsTransfer": {
			method:    SendType.IsEnsTransfer,
			trueCases: []SendType{ENSRegister, ENSRelease, ENSSetPubKey},
		},
		"IsStickersTransfer": {
			method:    SendType.IsStickersTransfer,
			trueCases: []SendType{StickersBuy},
		},
		"IsCommunityRelatedTransfer": {
			method:    SendType.IsCommunityRelatedTransfer,
			trueCases: []SendType{CommunityDeployOwnerToken, CommunityDeployCollectibles, CommunityDeployAssets, CommunityMintTokens, CommunityRemoteBurn, CommunityBurn, CommunitySetSignerPubKey},
		},
	}

	allTypes := []SendType{Transfer, ENSRegister, ENSRelease, ENSSetPubKey, StickersBuy, Bridge, ERC721Transfer, ERC1155Transfer, Swap, CommunityBurn, CommunityDeployAssets, CommunityDeployCollectibles, CommunityDeployOwnerToken, CommunityMintTokens, CommunityRemoteBurn, CommunitySetSignerPubKey}

	for methodName, test := range tests {
		t.Run(methodName, func(t *testing.T) {
			trueCaseMap := make(map[SendType]bool)
			for _, tc := range test.trueCases {
				trueCaseMap[tc] = true
			}

			for _, sendType := range allTypes {
				expected := trueCaseMap[sendType]
				result := test.method(sendType)
				assert.Equal(t, expected, result, "SendType: %v", sendType)
			}
		})
	}
}

func TestSendType_CanUseProcessor(t *testing.T) {
	tests := map[SendType][]string{
		Transfer:                    {pathProcessorCommon.ProcessorTransferName},
		Bridge:                      {pathProcessorCommon.ProcessorBridgeHopName},
		Swap:                        {pathProcessorCommon.ProcessorSwapParaswapName},
		ERC721Transfer:              {pathProcessorCommon.ProcessorERC721Name},
		ERC1155Transfer:             {pathProcessorCommon.ProcessorERC1155Name},
		ENSRegister:                 {pathProcessorCommon.ProcessorENSRegisterName},
		ENSRelease:                  {pathProcessorCommon.ProcessorENSReleaseName},
		ENSSetPubKey:                {pathProcessorCommon.ProcessorENSPublicKeyName},
		StickersBuy:                 {pathProcessorCommon.ProcessorStickersBuyName},
		CommunityBurn:               {pathProcessorCommon.ProcessorCommunityBurnName},
		CommunityDeployAssets:       {pathProcessorCommon.ProcessorCommunityDeployAssetsName},
		CommunityDeployCollectibles: {pathProcessorCommon.ProcessorCommunityDeployCollectiblesName},
		CommunityDeployOwnerToken:   {pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName},
		CommunityMintTokens:         {pathProcessorCommon.ProcessorCommunityMintTokensName},
		CommunityRemoteBurn:         {pathProcessorCommon.ProcessorCommunityRemoteBurnName},
		CommunitySetSignerPubKey:    {pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName},
	}

	allProcessors := []string{
		pathProcessorCommon.ProcessorTransferName, pathProcessorCommon.ProcessorBridgeHopName,
		pathProcessorCommon.ProcessorSwapParaswapName, pathProcessorCommon.ProcessorERC721Name, pathProcessorCommon.ProcessorERC1155Name,
		pathProcessorCommon.ProcessorENSRegisterName, pathProcessorCommon.ProcessorENSReleaseName, pathProcessorCommon.ProcessorENSPublicKeyName, pathProcessorCommon.ProcessorStickersBuyName,
		pathProcessorCommon.ProcessorCommunityBurnName, pathProcessorCommon.ProcessorCommunityDeployAssetsName,
		pathProcessorCommon.ProcessorCommunityDeployCollectiblesName, pathProcessorCommon.ProcessorCommunityDeployOwnerTokenName,
		pathProcessorCommon.ProcessorCommunityMintTokensName, pathProcessorCommon.ProcessorCommunityRemoteBurnName,
		pathProcessorCommon.ProcessorCommunitySetSignerPubKeyName,
	}

	for sendType, validProcessors := range tests {
		validMap := make(map[string]bool)
		for _, p := range validProcessors {
			validMap[p] = true
		}

		for _, processor := range allProcessors {
			expected := validMap[processor]
			result := sendType.CanUseProcessor(processor)
			assert.Equal(t, expected, result, "SendType: %v, Processor: %s", sendType, processor)
		}
	}

	// Test unknown SendType (default case)
	assert.True(t, SendType(999).CanUseProcessor("any-processor"))
}

func TestSendType_ProcessZeroAmountInProcessor(t *testing.T) {
	zero, nonZero := walletCommon.ZeroBigIntValue(), big.NewInt(100)

	// Non-zero amountIn always returns true
	assert.True(t, Transfer.ProcessZeroAmountInProcessor(nonZero, zero, "any"))

	// Zero amountIn scenarios
	tests := []struct {
		sendType  SendType
		amountOut *big.Int
		processor string
		expected  bool
	}{
		{Transfer, zero, pathProcessorCommon.ProcessorTransferName, true},
		{Transfer, zero, "wrong", false},
		{Swap, nonZero, "any", true},
		{Swap, zero, "any", false},
		{ENSSetPubKey, zero, "any", true},
		{ENSRelease, zero, "any", true},
		{CommunityBurn, zero, "any", true},
		{ENSRegister, zero, "any", false},
		{StickersBuy, zero, "any", false},
	}

	for _, tt := range tests {
		result := tt.sendType.ProcessZeroAmountInProcessor(zero, tt.amountOut, tt.processor)
		assert.Equal(t, tt.expected, result, "SendType: %v", tt.sendType)
	}
}

func TestSendType_IsAvailableBetween(t *testing.T) {
	eth := &params.Network{ChainID: walletCommon.EthereumMainnet}
	opt := &params.Network{ChainID: walletCommon.OptimismMainnet}

	sameChainOnly := []SendType{ERC721Transfer, ENSRegister, StickersBuy, CommunityBurn, Swap}
	diffChainOnly := []SendType{Bridge}
	anyChain := []SendType{Transfer}

	for _, st := range sameChainOnly {
		assert.True(t, st.IsAvailableBetween(eth, eth), "SendType: %v", st)
		assert.False(t, st.IsAvailableBetween(eth, opt), "SendType: %v", st)
	}

	for _, st := range diffChainOnly {
		assert.False(t, st.IsAvailableBetween(eth, eth), "SendType: %v", st)
		assert.True(t, st.IsAvailableBetween(eth, opt), "SendType: %v", st)
	}

	for _, st := range anyChain {
		assert.True(t, st.IsAvailableBetween(eth, eth), "SendType: %v", st)
		assert.True(t, st.IsAvailableBetween(eth, opt), "SendType: %v", st)
	}
}

func TestSendType_IsAvailableFor(t *testing.T) {
	networks := map[uint64]*params.Network{
		walletCommon.EthereumMainnet: {ChainID: walletCommon.EthereumMainnet},
		walletCommon.EthereumSepolia: {ChainID: walletCommon.EthereumSepolia},
		walletCommon.OptimismMainnet: {ChainID: walletCommon.OptimismMainnet},
		walletCommon.ArbitrumMainnet: {ChainID: walletCommon.ArbitrumMainnet},
		walletCommon.BaseMainnet:     {ChainID: walletCommon.BaseMainnet},
		walletCommon.BSCMainnet:      {ChainID: walletCommon.BSCMainnet},
		999999:                       {ChainID: 999999}, // unknown
	}

	swapNetworks := []uint64{walletCommon.EthereumMainnet, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet, walletCommon.BaseMainnet, walletCommon.BSCMainnet}
	bridgeNetworks := []uint64{walletCommon.EthereumMainnet, walletCommon.EthereumSepolia, walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet, walletCommon.BaseMainnet}
	ensNetworks := []uint64{walletCommon.EthereumMainnet, walletCommon.EthereumSepolia}

	// Test Swap
	for chainID, network := range networks {
		expected := slices.Contains(swapNetworks, chainID)
		assert.Equal(t, expected, Swap.IsAvailableFor(network), "Swap on chain %d", chainID)
	}

	// Test Bridge (includes sepolia networks)
	bridgeNetworksExtended := append(bridgeNetworks, walletCommon.OptimismSepolia, walletCommon.ArbitrumSepolia, walletCommon.BaseSepolia)
	for chainID, network := range networks {
		expected := slices.Contains(bridgeNetworksExtended, chainID)
		assert.Equal(t, expected, Bridge.IsAvailableFor(network), "Bridge on chain %d", chainID)
	}

	// Test ENS and Stickers
	for chainID, network := range networks {
		expected := slices.Contains(ensNetworks, chainID)
		assert.Equal(t, expected, ENSRegister.IsAvailableFor(network), "ENS on chain %d", chainID)
		assert.Equal(t, expected, StickersBuy.IsAvailableFor(network), "Stickers on chain %d", chainID)
	}

	// Test others (always true)
	for _, network := range networks {
		assert.True(t, Transfer.IsAvailableFor(network))
		assert.True(t, ERC721Transfer.IsAvailableFor(network))
		assert.True(t, CommunityBurn.IsAvailableFor(network))
	}
}
