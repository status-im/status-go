package wallet

import (
	"context"
	"math"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/chain"
)

type SuggestedRoutes struct {
	Networks []params.Network `json:"networks"`
}

func NewRouter(s *Service) *Router {
	return &Router{s}
}

type Router struct {
	s *Service
}

func (r *Router) suitableTokenExists(ctx context.Context, network *params.Network, tokens []*Token, account common.Address, amount float64, tokenSymbol string) (bool, error) {
	for _, token := range tokens {
		if token.Symbol != tokenSymbol {
			continue
		}

		clients, err := chain.NewClients(r.s.rpcClient, []uint64{network.ChainID})
		if err != nil {
			return false, err
		}

		balance, err := r.s.tokenManager.getBalance(ctx, clients[0], account, token.Address)
		if err != nil {
			return false, err
		}

		amountForToken, _ := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(math.Pow10(int(token.Decimals)))).Int(nil)
		if balance.Cmp(amountForToken) >= 0 {
			return true, nil
		}
	}

	return false, nil
}

func (r *Router) suggestedRoutes(ctx context.Context, account common.Address, amount float64, tokenSymbol string, disabledChainIDs []uint64) (*SuggestedRoutes, error) {
	areTestNetworksEnabled, err := r.s.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	networks, err := r.s.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	var (
		group      = async.NewAtomicGroup(ctx)
		mu         sync.Mutex
		candidates = make([]params.Network, 0)
	)
	for networkIdx := range networks {
		network := networks[networkIdx]
		if network.IsTest != areTestNetworksEnabled {
			continue
		}

		networkFound := false
		for _, chainID := range disabledChainIDs {
			networkFound = false
			if chainID == network.ChainID {
				networkFound = true
				break
			}
		}
		// This is network cannot be used as a suggestedRoute as the user has disabled it
		if networkFound {
			continue
		}

		group.Add(func(c context.Context) error {
			if tokenSymbol == network.NativeCurrencySymbol {
				tokens := []*Token{&Token{
					Address:  common.HexToAddress("0x"),
					Symbol:   network.NativeCurrencySymbol,
					Decimals: uint(network.NativeCurrencyDecimals),
					Name:     network.NativeCurrencyName,
				}}
				ok, _ := r.suitableTokenExists(c, network, tokens, account, amount, tokenSymbol)
				if ok {
					mu.Lock()
					candidates = append(candidates, *network)
					mu.Unlock()
					return nil
				}
			}

			tokens, err := r.s.tokenManager.getTokens(network.ChainID)
			if err == nil {
				ok, _ := r.suitableTokenExists(c, network, tokens, account, amount, tokenSymbol)

				if ok {
					mu.Lock()
					candidates = append(candidates, *network)
					mu.Unlock()
					return nil
				}
			}

			customTokens, err := r.s.tokenManager.getCustomsByChainID(network.ChainID)
			if err == nil {
				ok, _ := r.suitableTokenExists(c, network, customTokens, account, amount, tokenSymbol)

				if ok {
					mu.Lock()
					candidates = append(candidates, *network)
					mu.Unlock()
				}
			}

			return nil
		})
	}

	group.Wait()
	return &SuggestedRoutes{
		Networks: candidates,
	}, nil
}
