package onramp

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"
)

type Manager struct {
	providers []Provider
}

func NewManager(providers []Provider) *Manager {
	return &Manager{
		providers: providers,
	}
}

func (c *Manager) GetProviders(ctx context.Context) ([]CryptoOnRamp, error) {
	ret := make([]CryptoOnRamp, 0, len(c.providers))
	for _, provider := range c.providers {
		cryptoOnRamp, err := provider.GetCryptoOnRamp(ctx)
		if err != nil {
			log.Error("failed to get crypto on ramp", "id", provider.ID(), "error", err)
			continue
		}

		ret = append(ret, cryptoOnRamp)
	}

	return ret, nil
}

func (c *Manager) GetURL(ctx context.Context, providerID string, parameters Parameters) (string, error) {
	for _, provider := range c.providers {
		if provider.ID() != providerID {
			continue
		}

		return provider.GetURL(ctx, parameters)
	}

	return "", errors.New("provider not found")
}
