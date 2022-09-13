package wallet

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/bridge"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type SendType int

const (
	Transfer SendType = iota
	ENSRegister
	ENSRelease
	ENSSetPubKey
	StickersBuy
)
const EstimateUsername = "RandomUsername"
const EstimatePubKey = "0x04bb2024ce5d72e45d4a4f8589ae657ef9745855006996115a23a1af88d536cf02c0524a585fce7bfa79d6a9669af735eda6205d6c7e5b3cdc2b8ff7b2fa1f0b56"

func (s SendType) isTransfer() bool {
	return s == Transfer
}

func (s SendType) isAvailableFor(network *params.Network) bool {
	if s == Transfer {
		return true
	}

	if network.ChainID == 1 || network.ChainID == 5 {
		return true
	}

	return false
}

func (s SendType) EstimateGas(service *Service, network *params.Network) uint64 {
	from := types.Address(common.HexToAddress("0x5ffa75ce51c3a7ebe23bde37b5e3a0143dfbcee0"))
	tx := transactions.SendTxArgs{
		From:  from,
		Value: (*hexutil.Big)(big.NewInt(0)),
	}
	if s == ENSRegister {
		estimate, err := service.ens.API().RegisterEstimate(context.Background(), network.ChainID, tx, EstimateUsername, EstimatePubKey)
		if err != nil {
			return 400000
		}
		return estimate
	}

	if s == ENSRelease {
		estimate, err := service.ens.API().ReleaseEstimate(context.Background(), network.ChainID, tx, EstimateUsername)
		if err != nil {
			return 200000
		}
		return estimate
	}

	if s == ENSSetPubKey {
		estimate, err := service.ens.API().SetPubKeyEstimate(context.Background(), network.ChainID, tx, fmt.Sprint(EstimateUsername, ".stateofus.eth"), EstimatePubKey)
		if err != nil {
			return 400000
		}
		return estimate
	}

	if s == StickersBuy {
		packId := &bigint.BigInt{Int: big.NewInt(2)}
		estimate, err := service.stickers.API().BuyEstimate(context.Background(), network.ChainID, from, packId)
		if err != nil {
			return 400000
		}
		return estimate
	}

	return 0
}

var zero = big.NewInt(0)

type Path struct {
	BridgeName    string
	From          *params.Network
	To            *params.Network
	MaxAmountIn   *hexutil.Big
	AmountIn      *hexutil.Big
	AmountOut     *hexutil.Big
	GasAmount     uint64
	GasFees       *SuggestedFees
	BonderFees    *hexutil.Big
	TokenFees     *big.Float
	Cost          *big.Float
	Preferred     bool
	EstimatedTime TransactionEstimation
}

type Graph = []*Node

type Node struct {
	Path     *Path
	Children Graph
}

func newNode(path *Path) *Node {
	return &Node{Path: path, Children: make(Graph, 0)}
}

func buildGraph(AmountIn *big.Int, routes []*Path, level int, sourceChainIDs []uint64) Graph {
	graph := make(Graph, 0)
	for _, route := range routes {
		found := false
		for _, chainID := range sourceChainIDs {
			if chainID == route.From.ChainID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		node := newNode(route)

		newRoutes := make([]*Path, 0)
		for _, r := range routes {
			if r.From.ChainID == route.From.ChainID && r.To.ChainID == route.To.ChainID {
				continue
			}
			newRoutes = append(newRoutes, r)
		}

		newAmountIn := new(big.Int).Sub(AmountIn, route.MaxAmountIn.ToInt())
		if newAmountIn.Sign() > 0 {
			newSourceChainIDs := make([]uint64, len(sourceChainIDs))
			copy(newSourceChainIDs, sourceChainIDs)
			newSourceChainIDs = append(newSourceChainIDs, route.From.ChainID)
			node.Children = buildGraph(newAmountIn, newRoutes, level+1, newSourceChainIDs)

			if len(node.Children) == 0 {
				continue
			}
		}

		graph = append(graph, node)
	}

	return graph
}

func (n Node) findBest(level int) ([]*Path, *big.Float) {
	if len(n.Children) == 0 {
		if n.Path == nil {
			return []*Path{}, big.NewFloat(0)
		}
		return []*Path{n.Path}, n.Path.Cost
	}

	var best []*Path
	bestTotalCost := big.NewFloat(math.Inf(1))

	for _, node := range n.Children {
		routes, totalCost := node.findBest(level + 1)
		if totalCost.Cmp(bestTotalCost) < 0 {
			best = routes
			bestTotalCost = totalCost
		}
	}

	if n.Path == nil {
		return best, bestTotalCost
	}

	return append([]*Path{n.Path}, best...), new(big.Float).Add(bestTotalCost, n.Path.Cost)
}

type SuggestedRoutes struct {
	Best                  []*Path
	Candidates            []*Path
	TokenPrice            float64
	NativeChainTokenPrice float64
}

func newSuggestedRoutes(amountIn *big.Int, candidates []*Path) *SuggestedRoutes {
	if len(candidates) == 0 {
		return &SuggestedRoutes{
			Candidates: candidates,
			Best:       candidates,
		}
	}

	node := &Node{
		Path:     nil,
		Children: buildGraph(amountIn, candidates, 0, []uint64{}),
	}
	best, _ := node.findBest(0)

	if len(best) > 0 {
		rest := new(big.Int).Set(amountIn)
		for _, path := range best {
			diff := new(big.Int).Sub(rest, path.MaxAmountIn.ToInt())
			if diff.Cmp(big.NewInt(0)) >= 0 {
				path.AmountIn = path.MaxAmountIn
			} else {
				path.AmountIn = (*hexutil.Big)(new(big.Int).Set(rest))
			}
			rest.Sub(rest, path.AmountIn.ToInt())
		}
	}

	return &SuggestedRoutes{
		Candidates: candidates,
		Best:       best,
	}
}

func NewRouter(s *Service) *Router {
	bridges := make(map[string]bridge.Bridge)
	simple := bridge.NewSimpleBridge(s.transactor)
	hop := bridge.NewHopBridge(s.rpcClient)
	bridges[simple.Name()] = simple
	bridges[hop.Name()] = hop

	return &Router{s, bridges}
}

func containsNetworkChainId(network *params.Network, chainIDs []uint64) bool {
	for _, chainID := range chainIDs {
		if chainID == network.ChainID {
			return true
		}
	}

	return false
}

type Router struct {
	s       *Service
	bridges map[string]bridge.Bridge
}

func (r *Router) getBalance(ctx context.Context, network *params.Network, token *token.Token, account common.Address) (*big.Int, error) {
	clients, err := chain.NewClients(r.s.rpcClient, []uint64{network.ChainID})
	if err != nil {
		return nil, err
	}

	return r.s.tokenManager.GetBalance(ctx, clients[0], account, token.Address)
}

func (r *Router) estimateTimes(ctx context.Context, network *params.Network, gasFees *SuggestedFees, gasFeeMode GasFeeMode) TransactionEstimation {
	if gasFeeMode == GasFeeLow {
		return r.s.feesManager.transactionEstimatedTime(ctx, network.ChainID, gasFees.MaxFeePerGasLow)
	}

	if gasFeeMode == GasFeeMedium {
		return r.s.feesManager.transactionEstimatedTime(ctx, network.ChainID, gasFees.MaxFeePerGasMedium)
	}

	return r.s.feesManager.transactionEstimatedTime(ctx, network.ChainID, gasFees.MaxFeePerGasHigh)
}

func (r *Router) suggestedRoutes(ctx context.Context, sendType SendType, account common.Address, amountIn *big.Int, tokenSymbol string, disabledFromChainIDs, disabledToChaindIDs, preferedChainIDs []uint64, gasFeeMode GasFeeMode) (*SuggestedRoutes, error) {
	areTestNetworksEnabled, err := r.s.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	networks, err := r.s.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	prices, err := fetchCryptoComparePrices([]string{"ETH", tokenSymbol}, "USD")
	if err != nil {
		return nil, err
	}

	var (
		group      = async.NewAtomicGroup(ctx)
		mu         sync.Mutex
		candidates = make([]*Path, 0)
	)
	for networkIdx := range networks {
		network := networks[networkIdx]
		if network.IsTest != areTestNetworksEnabled {
			continue
		}

		if containsNetworkChainId(network, disabledFromChainIDs) {
			continue
		}

		if !sendType.isAvailableFor(network) {
			continue
		}

		token := r.s.tokenManager.FindToken(network, tokenSymbol)
		if token == nil {
			continue
		}

		nativeToken := r.s.tokenManager.FindToken(network, network.NativeCurrencySymbol)
		if nativeToken == nil {
			continue
		}

		group.Add(func(c context.Context) error {
			gasFees, err := r.s.feesManager.suggestedFees(ctx, network.ChainID)
			if err != nil {
				return err
			}

			balance, err := r.getBalance(ctx, network, token, account)
			if err != nil {
				return err
			}

			nativeBalance, err := r.getBalance(ctx, network, nativeToken, account)
			if err != nil {
				return err
			}
			maxFees := gasFees.feeFor(gasFeeMode)

			estimatedTime := r.s.feesManager.transactionEstimatedTime(ctx, network.ChainID, maxFees)

			for _, bridge := range r.bridges {
				for _, dest := range networks {
					if dest.IsTest != areTestNetworksEnabled {
						continue
					}

					if !sendType.isAvailableFor(network) {
						continue
					}

					if len(preferedChainIDs) > 0 && !containsNetworkChainId(network, preferedChainIDs) {
						continue
					}

					if containsNetworkChainId(dest, disabledToChaindIDs) {
						continue
					}

					can, err := bridge.Can(network, dest, token, balance)
					if err != nil || !can {
						continue
					}

					bonderFees, tokenFees, err := bridge.CalculateFees(network, dest, token, amountIn, prices["ETH"], prices[tokenSymbol], gasFees.GasPrice)
					if err != nil {
						continue
					}

					gasLimit := uint64(0)
					if sendType.isTransfer() {
						gasLimit, err = bridge.EstimateGas(network, dest, token, amountIn)
						if err != nil {
							continue
						}
					} else {
						gasLimit = sendType.EstimateGas(r.s, network)
					}
					requiredNativeBalance := new(big.Int).Mul(gweiToWei(maxFees), big.NewInt(int64(gasLimit)))

					if nativeBalance.Cmp(requiredNativeBalance) <= 0 {
						continue
					}

					preferred := containsNetworkChainId(dest, preferedChainIDs)

					gasCost := new(big.Float)
					gasCost.Mul(
						new(big.Float).Mul(gweiToEth(maxFees), big.NewFloat((float64(gasLimit)))),
						big.NewFloat(prices["ETH"]),
					)
					tokenFeesAsFloat := new(big.Float).Quo(
						new(big.Float).SetInt(tokenFees),
						big.NewFloat(math.Pow(10, float64(token.Decimals))),
					)
					tokenCost := new(big.Float)
					tokenCost.Mul(tokenFeesAsFloat, big.NewFloat(prices[tokenSymbol]))
					cost := new(big.Float)
					cost.Add(tokenCost, gasCost)

					mu.Lock()
					candidates = append(candidates, &Path{
						BridgeName:    bridge.Name(),
						From:          network,
						To:            dest,
						MaxAmountIn:   (*hexutil.Big)(balance),
						AmountIn:      (*hexutil.Big)(big.NewInt(0)),
						AmountOut:     (*hexutil.Big)(big.NewInt(0)),
						GasAmount:     gasLimit,
						GasFees:       gasFees,
						BonderFees:    (*hexutil.Big)(bonderFees),
						TokenFees:     tokenFeesAsFloat,
						Preferred:     preferred,
						Cost:          cost,
						EstimatedTime: estimatedTime,
					})
					mu.Unlock()
				}
			}
			return nil
		})
	}

	group.Wait()

	suggestedRoutes := newSuggestedRoutes(amountIn, candidates)
	suggestedRoutes.TokenPrice = prices[tokenSymbol]
	suggestedRoutes.NativeChainTokenPrice = prices["ETH"]
	for _, path := range suggestedRoutes.Best {
		amountOut, err := r.bridges[path.BridgeName].CalculateAmountOut(path.From, path.To, (*big.Int)(path.AmountIn), tokenSymbol)
		if err != nil {
			continue
		}
		path.AmountOut = (*hexutil.Big)(amountOut)
	}
	return suggestedRoutes, nil
}
