package multistandardbalance

import (
	"context"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/balance/refresh"

	"go.uber.org/zap"
)

// pairRead is what one decision reads of an account on a chain.
type pairRead struct {
	native bool
	// listed is true when every token of the lists is read.
	listed  bool
	tokens  map[common.Address]struct{}
	erc721  bool
	erc1155 bool
}

func (p *pairRead) addTokens(tokens ...common.Address) {
	for _, token := range tokens {
		if token == (common.Address{}) {
			continue
		}
		if p.tokens == nil {
			p.tokens = make(map[common.Address]struct{})
		}
		p.tokens[token] = struct{}{}
	}
}

// readPlan gathers the reads of one decision by account and chain.
type readPlan struct {
	pairs map[BalancesKey]*pairRead
	// lists are the token lists by chain, asked for once per plan.
	lists map[uint64][]common.Address
}

func newReadPlan() *readPlan {
	return &readPlan{pairs: make(map[BalancesKey]*pairRead), lists: make(map[uint64][]common.Address)}
}

func (p *readPlan) pair(key BalancesKey) *pairRead {
	r, ok := p.pairs[key]
	if !ok {
		r = &pairRead{}
		p.pairs[key] = r
	}
	return r
}

func (c *Controller) tokenList(plan *readPlan, chainID uint64) ([]common.Address, bool) {
	if list, ok := plan.lists[chainID]; ok {
		return list, true
	}
	list, err := c.tokenListProvider.GetTokenContractAddresses(chainID)
	if err != nil {
		c.logger.Error("failed to get token contract addresses", zap.Uint64("chainID", chainID), zap.Error(err))
		return nil, false
	}
	plan.lists[chainID] = list
	return list, true
}

func hasKind(kinds refresh.Kinds, kind refresh.Kinds) bool {
	return kinds == 0 || kinds&kind != 0
}

func (c *Controller) addRead(plan *readPlan, r refresh.Read, accounts []common.Address, chains []uint64) {
	if r.Scope.OlderThan > 0 {
		c.logger.Warn("unsupported balance read", zap.Any("read", r))
		return
	}
	if r.Scope.Pair.Account != (common.Address{}) {
		accounts = []common.Address{r.Scope.Pair.Account}
	}
	if r.Scope.Pair.Chain != 0 {
		chains = []uint64{r.Scope.Pair.Chain}
	}
	for _, account := range accounts {
		for _, chain := range chains {
			key := BalancesKey{Account: account, ChainID: chain}
			switch r.Tokens {
			case refresh.Full:
				if r.Scope.Unread && c.wasRead(key) {
					if tokens := c.unreadTokens(plan, key); len(tokens) > 0 {
						plan.pair(key).addTokens(tokens...)
					}
					continue
				}
				p := plan.pair(key)
				p.native = p.native || hasKind(r.Kinds, refresh.Native)
				p.listed = p.listed || hasKind(r.Kinds, refresh.ERC20)
				p.erc721 = p.erc721 || hasKind(r.Kinds, refresh.ERC721)
				p.erc1155 = p.erc1155 || hasKind(r.Kinds, refresh.ERC1155)
			case refresh.Held:
				p := plan.pair(key)
				p.native = p.native || hasKind(r.Kinds, refresh.Native)
				if hasKind(r.Kinds, refresh.ERC20) {
					p.addTokens(c.heldTokens(key)...)
				}
				p.erc721 = p.erc721 || hasKind(r.Kinds, refresh.ERC721)
				p.erc1155 = p.erc1155 || hasKind(r.Kinds, refresh.ERC1155)
			case refresh.Transfer:
				p := plan.pair(key)
				p.native = p.native || hasKind(r.Kinds, refresh.Native)
				if hasKind(r.Kinds, refresh.ERC20) {
					p.addTokens(r.Contracts...)
				}
			}
		}
	}
}

// wasRead reports whether the balances of key were ever read; the native coin
// is read with every refresh.
func (c *Controller) wasRead(key BalancesKey) bool {
	_, state, err := c.storage.GetNativeBalance(context.Background(), key)
	return err == nil && state.FetchedAt != NeverFetched
}

// unreadTokens are the tokens of the lists that key has no balance of.
func (c *Controller) unreadTokens(plan *readPlan, key BalancesKey) []common.Address {
	list, ok := c.tokenList(plan, key.ChainID)
	if !ok {
		return nil
	}
	stored, _, err := c.storage.GetERC20Balances(context.Background(), key)
	if err != nil {
		return nil
	}
	var unread []common.Address
	for _, token := range list {
		if _, ok := stored[token]; !ok {
			unread = append(unread, token)
		}
	}
	return unread
}

// heldTokens are the tokens of the held set of key: the ones with a balance,
// those transfer detection found, and those of heldTokensProvider.
func (c *Controller) heldTokens(key BalancesKey) []common.Address {
	held := c.detectedTokens(key)
	if stored, _, err := c.storage.GetERC20Balances(context.Background(), key); err == nil {
		for token, balance := range stored {
			if balance != nil && balance.Sign() != 0 {
				held = append(held, token)
			}
		}
	}
	provided, err := c.heldTokensProvider.GetHeldTokens(key.ChainID, key.Account)
	if err != nil {
		c.logger.Error("failed to get held tokens", zap.Uint64("chainID", key.ChainID), zap.Error(err))
	}
	return append(held, provided...)
}

// chainFetch is one fetch of a chain.
type chainFetch struct {
	config multistandardfetcher.FetchConfig
	// listed are the accounts whose ERC20 balances are read for every token
	// of the lists; when it has any, the fetch is a full refresh of the chain.
	listed map[AccountAddress]bool
}

func (c *Controller) buildChainFetches(plan *readPlan) map[uint64]*chainFetch {
	fetches := make(map[uint64]*chainFetch)
	for key, p := range plan.pairs {
		fetch, exists := fetches[key.ChainID]
		if !exists {
			fetch = &chainFetch{
				config: multistandardfetcher.FetchConfig{
					Native:  make([]AccountAddress, 0),
					ERC20:   make(map[AccountAddress][]ContractAddress),
					ERC721:  make(map[AccountAddress][]ContractAddress),
					ERC1155: make(map[AccountAddress][]CollectibleID),
				},
				listed: make(map[AccountAddress]bool),
			}
			fetches[key.ChainID] = fetch
		}

		if p.native {
			fetch.config.Native = append(fetch.config.Native, key.Account)
		}
		if p.listed {
			list, ok := c.tokenList(plan, key.ChainID)
			if !ok {
				continue
			}
			p.addTokens(list...)
			fetch.listed[key.Account] = true
		}
		if len(p.tokens) > 0 {
			fetch.config.ERC20[key.Account] = slices.Collect(maps.Keys(p.tokens))
		}
		if p.erc721 || p.erc1155 {
			erc721List, erc1155List, err := c.collectibleListProvider.GetCollectiblesList(key.ChainID, key.Account)
			if err != nil {
				c.logger.Error("failed to get owned collectibles", zap.Error(err))
				continue
			}
			if p.erc721 {
				for _, erc721ID := range erc721List {
					fetch.config.ERC721[key.Account] = append(fetch.config.ERC721[key.Account], erc721ID.ContractAddress)
				}
			}
			if p.erc1155 {
				fetch.config.ERC1155[key.Account] = append(fetch.config.ERC1155[key.Account], erc1155List...)
			}
		}
	}
	for chainID, fetch := range fetches {
		if isEmpty(fetch.config) {
			delete(fetches, chainID)
		}
	}
	return fetches
}

func isEmpty(config multistandardfetcher.FetchConfig) bool {
	return len(config.Native) == 0 && len(config.ERC20) == 0 && len(config.ERC721) == 0 && len(config.ERC1155) == 0
}

// handleFetchResult stores fetchResult and reports whether it succeeded.
func (c *Controller) handleFetchResult(ctx context.Context, chainID uint64, fetchResult multistandardfetcher.FetchResult, fetch *chainFetch) bool {
	c.logger.Debug("processing result", zap.Uint64("chainID", chainID), zap.String("fetchResult.ResultType", string(fetchResult.ResultType)))
	switch fetchResult.ResultType {
	case multistandardfetcher.ResultTypeNative:
		result, ok := fetchResult.Result.(multistandardfetcher.NativeResult)
		if !ok {
			c.logger.Error("failed to parse native result", zap.Any("result", fetchResult.Result))
			return false
		}
		return c.handleNativeResult(ctx, chainID, result)
	case multistandardfetcher.ResultTypeERC20:
		result, ok := fetchResult.Result.(multistandardfetcher.ERC20Result)
		if !ok {
			c.logger.Error("failed to parse ERC20 result", zap.Any("result", fetchResult.Result))
			return false
		}
		return c.handleERC20Result(ctx, chainID, result, fetch.listed[result.Account])
	case multistandardfetcher.ResultTypeERC721:
		result, ok := fetchResult.Result.(multistandardfetcher.ERC721Result)
		if !ok {
			c.logger.Error("failed to parse ERC721 result", zap.Any("result", fetchResult.Result))
			return false
		}
		return c.handleERC721Result(ctx, chainID, result)
	case multistandardfetcher.ResultTypeERC1155:
		result, ok := fetchResult.Result.(multistandardfetcher.ERC1155Result)
		if !ok {
			c.logger.Error("failed to parse ERC1155 result", zap.Any("result", fetchResult.Result))
			return false
		}
		return c.handleERC1155Result(ctx, chainID, result)
	}
	return true
}
