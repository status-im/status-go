package wallet

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
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
	Bridge
)
const EstimateUsername = "RandomUsername"
const EstimatePubKey = "0x04bb2024ce5d72e45d4a4f8589ae657ef9745855006996115a23a1af88d536cf02c0524a585fce7bfa79d6a9669af735eda6205d6c7e5b3cdc2b8ff7b2fa1f0b56"

func (s SendType) isTransfer() bool {
	return s == Transfer
}

func (s SendType) isAvailableBetween(from, to *params.Network) bool {
	if s != Bridge {
		return true
	}

	return from.ChainID != to.ChainID
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
		Value: (*hexutil.Big)(zero),
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
		packID := &bigint.BigInt{Int: big.NewInt(2)}
		estimate, err := service.stickers.API().BuyEstimate(context.Background(), network.ChainID, from, packID)
		if err != nil {
			return 400000
		}
		return estimate
	}

	return 0
}

var zero = big.NewInt(0)

type Path struct {
	BridgeName              string
	From                    *params.Network
	To                      *params.Network
	MaxAmountIn             *hexutil.Big
	AmountIn                *hexutil.Big
	AmountInLocked          bool
	AmountOut               *hexutil.Big
	GasAmount               uint64
	GasFees                 *SuggestedFees
	BonderFees              *hexutil.Big
	TokenFees               *big.Float
	Cost                    *big.Float
	EstimatedTime           TransactionEstimation
	ApprovalRequired        bool
	ApprovalGasFees         *big.Float
	ApprovalAmountRequired  *hexutil.Big
	ApprovalContractAddress *common.Address
}

func (p *Path) Equal(o *Path) bool {
	return p.From.ChainID == o.From.ChainID && p.To.ChainID == o.To.ChainID
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
			if route.Equal(r) {
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

func (n Node) buildAllRoutes() [][]*Path {
	res := make([][]*Path, 0)

	if len(n.Children) == 0 && n.Path != nil {
		res = append(res, []*Path{n.Path})
	}

	for _, node := range n.Children {
		for _, route := range node.buildAllRoutes() {
			extendedRoute := route
			if n.Path != nil {
				extendedRoute = append([]*Path{n.Path}, route...)
			}
			res = append(res, extendedRoute)
		}
	}

	return res
}

func filterRoutes(routes [][]*Path, amountIn *big.Int, fromLockedAmount map[uint64]*hexutil.Big) [][]*Path {
	if len(fromLockedAmount) == 0 {
		return routes
	}

	filteredRoutesLevel1 := make([][]*Path, 0)
	for _, route := range routes {
		routeOk := true
		fromIncluded := make(map[uint64]bool)
		fromExcluded := make(map[uint64]bool)
		for chainID, amount := range fromLockedAmount {
			if amount.ToInt().Cmp(zero) == 0 {
				fromExcluded[chainID] = false
			} else {
				fromIncluded[chainID] = false
			}

		}
		for _, path := range route {
			if _, ok := fromExcluded[path.From.ChainID]; ok {
				routeOk = false
				break
			}
			if _, ok := fromIncluded[path.From.ChainID]; ok {
				fromIncluded[path.From.ChainID] = true
			}
		}
		for _, value := range fromIncluded {
			if !value {
				routeOk = false
				break
			}
		}

		if routeOk {
			filteredRoutesLevel1 = append(filteredRoutesLevel1, route)
		}
	}

	filteredRoutesLevel2 := make([][]*Path, 0)
	for _, route := range filteredRoutesLevel1 {
		routeOk := true
		for _, path := range route {
			if amount, ok := fromLockedAmount[path.From.ChainID]; ok {
				requiredAmountIn := new(big.Int).Sub(amountIn, amount.ToInt())
				restAmountIn := big.NewInt(0)

				for _, otherPath := range route {
					if path.Equal(otherPath) {
						continue
					}
					restAmountIn = new(big.Int).Add(otherPath.MaxAmountIn.ToInt(), restAmountIn)
				}
				if restAmountIn.Cmp(requiredAmountIn) >= 0 {
					path.AmountIn = amount
					path.AmountInLocked = true
				} else {
					routeOk = false
					break
				}
			}
		}
		if routeOk {
			filteredRoutesLevel2 = append(filteredRoutesLevel2, route)
		}
	}

	return filteredRoutesLevel2
}

func findBest(routes [][]*Path) []*Path {
	var best []*Path
	bestCost := big.NewFloat(math.Inf(1))
	for _, route := range routes {
		currentCost := big.NewFloat(0)
		for _, path := range route {
			currentCost = new(big.Float).Add(currentCost, path.Cost)
		}

		if currentCost.Cmp(bestCost) == -1 {
			best = route
			bestCost = currentCost
		}
	}

	return best
}

type SuggestedRoutes struct {
	Best                  []*Path
	Candidates            []*Path
	TokenPrice            float64
	NativeChainTokenPrice float64
}

func newSuggestedRoutes(
	amountIn *big.Int,
	candidates []*Path,
	fromLockedAmount map[uint64]*hexutil.Big,
) *SuggestedRoutes {
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
	routes := node.buildAllRoutes()
	routes = filterRoutes(routes, amountIn, fromLockedAmount)
	best := findBest(routes)

	if len(best) > 0 {
		sort.Slice(best, func(i, j int) bool {
			return best[i].AmountInLocked
		})
		rest := new(big.Int).Set(amountIn)
		for _, path := range best {
			diff := new(big.Int).Sub(rest, path.MaxAmountIn.ToInt())
			if diff.Cmp(zero) >= 0 {
				path.AmountIn = (*hexutil.Big)(path.MaxAmountIn.ToInt())
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
	cbridge := bridge.NewCbridge(s.rpcClient, s.transactor, s.tokenManager)
	hop := bridge.NewHopBridge(s.rpcClient, s.transactor, s.tokenManager)
	bridges[simple.Name()] = simple
	bridges[hop.Name()] = hop
	bridges[cbridge.Name()] = cbridge

	return &Router{s, bridges, s.rpcClient}
}

func containsNetworkChainID(network *params.Network, chainIDs []uint64) bool {
	for _, chainID := range chainIDs {
		if chainID == network.ChainID {
			return true
		}
	}

	return false
}

type Router struct {
	s         *Service
	bridges   map[string]bridge.Bridge
	rpcClient *rpc.Client
}

func (r *Router) requireApproval(ctx context.Context, bridge bridge.Bridge, account common.Address, network *params.Network, token *token.Token, amountIn *big.Int) (bool, *big.Int, uint64, *common.Address, error) {
	if token.IsNative() {
		return false, nil, 0, nil, nil
	}
	contractMaker := &contracts.ContractMaker{RPCClient: r.rpcClient}

	bridgeAddress := bridge.GetContractAddress(network, token)
	if bridgeAddress == nil {
		return false, nil, 0, nil, nil
	}

	contract, err := contractMaker.NewERC20(network.ChainID, token.Address)
	if err != nil {
		return false, nil, 0, nil, err
	}

	allowance, err := contract.Allowance(&bind.CallOpts{
		Context: ctx,
	}, account, *bridgeAddress)

	if err != nil {
		return false, nil, 0, nil, err
	}

	if allowance.Cmp(amountIn) >= 0 {
		return false, nil, 0, nil, nil
	}

	ethClient, err := r.rpcClient.EthClient(network.ChainID)
	if err != nil {
		return false, nil, 0, nil, err
	}

	erc20ABI, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
	if err != nil {
		return false, nil, 0, nil, err
	}

	data, err := erc20ABI.Pack("approve", bridgeAddress, amountIn)
	if err != nil {
		return false, nil, 0, nil, err
	}

	estimate, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  account,
		To:    &token.Address,
		Value: big.NewInt(0),
		Data:  data,
	})
	if err != nil {
		return false, nil, 0, nil, err
	}

	return true, amountIn, estimate, bridgeAddress, nil

}

func (r *Router) getBalance(ctx context.Context, network *params.Network, token *token.Token, account common.Address) (*big.Int, error) {
	clients, err := chain.NewClients(r.s.rpcClient, []uint64{network.ChainID})
	if err != nil {
		return nil, err
	}

	return r.s.tokenManager.GetBalance(ctx, clients[0], account, token.Address)
}

func (r *Router) suggestedRoutes(
	ctx context.Context,
	sendType SendType,
	account common.Address,
	amountIn *big.Int,
	tokenSymbol string,
	disabledFromChainIDs,
	disabledToChaindIDs,
	preferedChainIDs []uint64,
	gasFeeMode GasFeeMode,
	fromLockedAmount map[uint64]*hexutil.Big,
) (*SuggestedRoutes, error) {
	areTestNetworksEnabled, err := r.s.accountsDB.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	networks, err := r.s.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	pricesMap, err := r.s.priceManager.FetchPrices([]string{"ETH", tokenSymbol}, []string{"USD"})
	if err != nil {
		return nil, err
	}
	prices := make(map[string]float64, 0)
	for symbol, pricePerCurrency := range pricesMap {
		prices[symbol] = pricePerCurrency["USD"]
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

		if containsNetworkChainID(network, disabledFromChainIDs) {
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

			maxAmountIn := (*hexutil.Big)(balance)
			if amount, ok := fromLockedAmount[network.ChainID]; ok {
				if amount.ToInt().Cmp(balance) == 1 {
					return errors.New("locked amount cannot be bigger than balance")
				}
				maxAmountIn = amount
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

					if !sendType.isAvailableBetween(network, dest) {
						continue
					}

					if len(preferedChainIDs) > 0 && !containsNetworkChainID(dest, preferedChainIDs) {
						continue
					}

					if containsNetworkChainID(dest, disabledToChaindIDs) {
						continue
					}

					can, err := bridge.Can(network, dest, token, maxAmountIn.ToInt())
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

					// Removed the required fees from maxAMount in case of native token tx
					if token.IsNative() {
						maxAmountIn = (*hexutil.Big)(new(big.Int).Sub(maxAmountIn.ToInt(), requiredNativeBalance))
					}

					if nativeBalance.Cmp(requiredNativeBalance) <= 0 {
						continue
					}

					approvalRequired, approvalAmountRequired, approvalGasLimit, approvalContractAddress, err := r.requireApproval(ctx, bridge, account, network, token, amountIn)
					if err != nil {
						continue
					}
					approvalGasFees := new(big.Float).Mul(gweiToEth(maxFees), big.NewFloat((float64(approvalGasLimit))))

					approvalGasCost := new(big.Float)
					approvalGasCost.Mul(
						approvalGasFees,
						big.NewFloat(prices["ETH"]),
					)

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
					cost.Add(cost, approvalGasCost)

					mu.Lock()
					candidates = append(candidates, &Path{
						BridgeName:              bridge.Name(),
						From:                    network,
						To:                      dest,
						MaxAmountIn:             maxAmountIn,
						AmountIn:                (*hexutil.Big)(zero),
						AmountOut:               (*hexutil.Big)(zero),
						GasAmount:               gasLimit,
						GasFees:                 gasFees,
						BonderFees:              (*hexutil.Big)(bonderFees),
						TokenFees:               tokenFeesAsFloat,
						Cost:                    cost,
						EstimatedTime:           estimatedTime,
						ApprovalRequired:        approvalRequired,
						ApprovalGasFees:         approvalGasFees,
						ApprovalAmountRequired:  (*hexutil.Big)(approvalAmountRequired),
						ApprovalContractAddress: approvalContractAddress,
					})
					mu.Unlock()
				}
			}
			return nil
		})
	}

	group.Wait()

	suggestedRoutes := newSuggestedRoutes(amountIn, candidates, fromLockedAmount)
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
