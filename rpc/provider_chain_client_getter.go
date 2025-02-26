package rpc

import "github.com/status-im/status-go/rpc/chain/ethclient"

type ProviderChainClientGetter struct {
	provider string
	client   ClientInterface
}

func NewProviderChainClientGetter(provider string, client ClientInterface) *ProviderChainClientGetter {
	return &ProviderChainClientGetter{
		provider: provider,
		client:   client,
	}
}

func (c *ProviderChainClientGetter) GetEthClient(chainID uint64) (ethclient.EthClientInterface, error) {
	return c.client.EthClientWithProvider(chainID, c.provider)
}
