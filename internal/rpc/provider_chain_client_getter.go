package rpc

import (
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
)

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

func (c *ProviderChainClientGetter) EthClient(chainID uint64) (ethclient.EthClientInterface, error) {
	return c.client.EthClientWithProvider(chainID, c.provider)
}

// PreferredProviderChainClientGetter returns the client of the first provider in the list that the chain has enabled.
type PreferredProviderChainClientGetter struct {
	providers []string
	client    ClientInterface
}

func NewPreferredProviderChainClientGetter(client ClientInterface, providers ...string) *PreferredProviderChainClientGetter {
	return &PreferredProviderChainClientGetter{
		providers: providers,
		client:    client,
	}
}

func (c *PreferredProviderChainClientGetter) EthClient(chainID uint64) (ethclient.EthClientInterface, error) {
	for _, provider := range c.providers {
		ethClient, err := c.client.EthClientWithProvider(chainID, provider)
		if err != nil {
			return nil, err
		}
		if ethClient != nil {
			return ethClient, nil
		}
	}
	return nil, nil
}
