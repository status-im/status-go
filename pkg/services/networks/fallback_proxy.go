package networks

import (
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/common"
)

// TODO(status-im/status-app#23177): remove with the test proxy fallback.

// withFallbackProxy adds the eth-rpc-proxy of every fallback stage as the last
// provider of each network the main proxy serves. It is enabled when the main
// proxy is.
func withFallbackProxy(networks []params.Network, walletConfig *params.WalletConfig) []params.Network {
	stages := params.FallbackProxyStages(walletConfig.StatusProxyStageName, walletConfig.EthRpcProxyUrl.Reveal())
	if len(stages) == 0 {
		return networks
	}

	updatedNetworks := deepCopyNetworks(networks)
	for i := range updatedNetworks {
		network := &updatedNetworks[i]
		mainProxy := findProvider(network.RpcProviders, common.StatusSmartProxy)
		if mainProxy == nil {
			continue
		}
		chainName, networkName := GetProxyChainAndNetworkName(network.ChainID)
		for _, stage := range stages {
			fallback := params.NewEthRpcProxyProvider(network.ChainID, common.StatusSmartProxyFallback,
				smartProxyUrl(getProxyHost("", stage), chainName, networkName), false, true)
			fallback.Enabled = mainProxy.Enabled
			network.RpcProviders = append(network.RpcProviders, *fallback)
		}
	}
	return updatedNetworks
}

func findProvider(providers []params.RpcProvider, name string) *params.RpcProvider {
	for i := range providers {
		if providers[i].Name == name {
			return &providers[i]
		}
	}
	return nil
}
