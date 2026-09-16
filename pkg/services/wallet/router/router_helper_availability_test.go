package router

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/contracts/hop"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/lifi"
	mock_lifi "github.com/status-im/status-go/pkg/services/wallet/thirdparty/lifi/mock"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/paraswap"
	mock_paraswap "github.com/status-im/status-go/pkg/services/wallet/thirdparty/paraswap/mock"
)

func TestTokenAvailableForBridgingViaHop(t *testing.T) {
	r := &Router{}

	contracts := hop.GetTokenContractsAvailableOnChain(walletCommon.EthereumMainnet)
	require.NotEmpty(t, contracts, "expected hop to have at least one token contract for ethereum mainnet")

	require.True(t, r.TokenAvailableForBridgingViaHop(walletCommon.EthereumMainnet, contracts[0]))
	require.True(t, r.TokenAvailableForBridgingViaHop(walletCommon.EthereumMainnet, walletCommon.ZeroAddress()))
	require.False(t, r.TokenAvailableForBridgingViaHop(uint64(999999), contracts[0]))
}

// registeredProcessors builds the processor map the router consults to decide
// whether a swap provider is enabled. Only presence matters for these tests.
func registeredProcessors(names ...string) map[string]pathprocessor.PathProcessor {
	processors := make(map[string]pathprocessor.PathProcessor, len(names))
	for _, name := range names {
		processors[name] = &pathprocessor.SwapParaswapProcessor{}
	}
	return processors
}

func TestIsChainSupportedForSwapViaParaswap_ProviderNotRegistered(t *testing.T) {
	r := &Router{
		pathProcessors: registeredProcessors(),
		paraswapClientFactory: func(chainID uint64) paraswap.ClientInterface {
			t.Fatal("paraswap client must not be created when the provider is not registered")
			return nil
		},
	}

	supported, err := r.IsChainSupportedForSwapViaParaswap(walletCommon.EthereumMainnet)
	require.NoError(t, err)
	require.False(t, supported)
}

func TestIsChainSupportedForSwapViaParaswap(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	requestedChainID := walletCommon.EthereumMainnet

	tokens := []paraswap.Token{
		{Address: "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE", Network: int(requestedChainID)},
		{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Network: int(requestedChainID)},
	}

	mockClient := mock_paraswap.NewMockClientInterface(ctrl)
	mockClient.EXPECT().
		FetchTokensList(gomock.Any()).
		Return(tokens, nil)

	r := &Router{
		pathProcessors: registeredProcessors(pathProcessorCommon.ProcessorSwapParaswapName),
		paraswapClientFactory: func(chainID uint64) paraswap.ClientInterface {
			require.Equal(t, requestedChainID, chainID)
			return mockClient
		},
	}
	supported, err := r.IsChainSupportedForSwapViaParaswap(requestedChainID)
	require.NoError(t, err)
	require.True(t, supported)

	mockClient.EXPECT().
		FetchTokensList(gomock.Any()).
		Return([]paraswap.Token{}, nil)

	r = &Router{
		pathProcessors: registeredProcessors(pathProcessorCommon.ProcessorSwapParaswapName),
		paraswapClientFactory: func(chainID uint64) paraswap.ClientInterface {
			return mockClient
		},
	}

	supported, err = r.IsChainSupportedForSwapViaParaswap(101)
	require.NoError(t, err)
	require.False(t, supported)
}

func TestIsChainSupportedForSwapViaParaswap_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_paraswap.NewMockClientInterface(ctrl)
	mockClient.EXPECT().
		FetchTokensList(gomock.Any()).
		Return(nil, errors.New("error fetching tokens list"))

	r := &Router{
		pathProcessors: registeredProcessors(pathProcessorCommon.ProcessorSwapParaswapName),
		paraswapClientFactory: func(chainID uint64) paraswap.ClientInterface {
			return mockClient
		},
	}
	_, err := r.IsChainSupportedForSwapViaParaswap(walletCommon.EthereumMainnet)
	require.Error(t, err)
}

func TestIsChainSupportedForSwapViaLiFi_ProviderNotRegistered(t *testing.T) {
	r := &Router{
		pathProcessors: registeredProcessors(),
		lifiClientFactory: func(chainID uint64) lifi.ClientInterface {
			t.Fatal("lifi client must not be created when the provider is not registered")
			return nil
		},
	}

	supported, err := r.IsChainSupportedForSwapViaLiFi(walletCommon.EthereumMainnet)
	require.NoError(t, err)
	require.False(t, supported)
}

func TestIsChainSupportedForSwapViaLiFi(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	requestedChainID := walletCommon.EthereumMainnet

	mockClient := mock_lifi.NewMockClientInterface(ctrl)
	mockClient.EXPECT().
		FetchTokensList(gomock.Any()).
		Return([]lifi.Token{{}, {}}, nil)

	r := &Router{
		pathProcessors: registeredProcessors(pathProcessorCommon.ProcessorLiFiName),
		lifiClientFactory: func(chainID uint64) lifi.ClientInterface {
			require.Equal(t, requestedChainID, chainID)
			return mockClient
		},
	}
	supported, err := r.IsChainSupportedForSwapViaLiFi(requestedChainID)
	require.NoError(t, err)
	require.True(t, supported)

	mockClient.EXPECT().
		FetchTokensList(gomock.Any()).
		Return([]lifi.Token{}, nil)

	supported, err = r.IsChainSupportedForSwapViaLiFi(requestedChainID)
	require.NoError(t, err)
	require.False(t, supported)
}

func TestIsChainSupportedForSwapViaLiFi_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_lifi.NewMockClientInterface(ctrl)
	mockClient.EXPECT().
		FetchTokensList(gomock.Any()).
		Return(nil, errors.New("error fetching tokens list"))

	r := &Router{
		pathProcessors: registeredProcessors(pathProcessorCommon.ProcessorLiFiName),
		lifiClientFactory: func(chainID uint64) lifi.ClientInterface {
			return mockClient
		},
	}
	_, err := r.IsChainSupportedForSwapViaLiFi(walletCommon.EthereumMainnet)
	require.Error(t, err)
}
