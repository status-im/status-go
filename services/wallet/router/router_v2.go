package router

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/wallet/async"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	walletToken "github.com/status-im/status-go/services/wallet/token"
)

var (
	supportedNetworks = map[uint64]bool{
		walletCommon.EthereumMainnet: true,
		walletCommon.OptimismMainnet: true,
		walletCommon.ArbitrumMainnet: true,
	}

	supportedTestNetworks = map[uint64]bool{
		walletCommon.EthereumSepolia: true,
		walletCommon.OptimismSepolia: true,
		walletCommon.ArbitrumSepolia: true,
	}
)

type RouteInputParams struct {
	SendType             SendType                `json:"sendType" validate:"required"`
	AddrFrom             common.Address          `json:"addrFrom" validate:"required"`
	AddrTo               common.Address          `json:"addrTo" validate:"required"`
	AmountIn             *hexutil.Big            `json:"amountIn" validate:"required"`
	AmountOut            *hexutil.Big            `json:"amountOut"`
	TokenID              string                  `json:"tokenID" validate:"required"`
	ToTokenID            string                  `json:"toTokenID"`
	DisabledFromChainIDs []uint64                `json:"disabledFromChainIDs"`
	DisabledToChainIDs   []uint64                `json:"disabledToChainIDs"`
	GasFeeMode           GasFeeMode              `json:"gasFeeMode" validate:"required"`
	FromLockedAmount     map[uint64]*hexutil.Big `json:"fromLockedAmount"`
	TestnetMode          bool                    `json:"testnetMode"`

	// For send types like EnsRegister, EnsRelease, EnsSetPubKey, StickersBuy
	Username  string       `json:"username"`
	PublicKey string       `json:"publicKey"`
	PackID    *hexutil.Big `json:"packID"`
}

type PathV2 struct {
	ProcessorName  string
	FromChain      *params.Network    // Source chain
	ToChain        *params.Network    // Destination chain
	FromToken      *walletToken.Token // Token on the source chain
	AmountIn       *hexutil.Big       // Amount that will be sent from the source chain
	AmountInLocked bool               // Is the amount locked
	AmountOut      *hexutil.Big       // Amount that will be received on the destination chain

	SuggestedLevelsForMaxFeesPerGas *MaxFeesLevels // Suggested max fees for the transaction

	TxBaseFee     *hexutil.Big // Base fee for the transaction
	TxPriorityFee *hexutil.Big // Priority fee for the transaction
	TxGasAmount   uint64       // Gas used for the transaction
	TxBonderFees  *hexutil.Big // Bonder fees for the transaction - used for Hop bridge
	TxTokenFees   *hexutil.Big // Token fees for the transaction - used for bridges (represent the difference between the amount in and the amount out)
	TxL1Fee       *hexutil.Big // L1 fee for the transaction - used for for transactions placed on L2 chains

	ApprovalRequired        bool            // Is approval required for the transaction
	ApprovalAmountRequired  *hexutil.Big    // Amount required for the approval transaction
	ApprovalContractAddress *common.Address // Address of the contract that needs to be approved
	ApprovalBaseFee         *hexutil.Big    // Base fee for the approval transaction
	ApprovalPriorityFee     *hexutil.Big    // Priority fee for the approval transaction
	ApprovalGasAmount       uint64          // Gas used for the approval transaction
	ApprovalL1Fee           *hexutil.Big    // L1 fee for the approval transaction - used for for transactions placed on L2 chains

	EstimatedTime TransactionEstimation

	requiredTokenBalance  *big.Int
	requiredNativeBalance *big.Int
}

func (p *PathV2) Equal(o *PathV2) bool {
	return p.FromChain.ChainID == o.FromChain.ChainID && p.ToChain.ChainID == o.ToChain.ChainID
}

type SuggestedRoutesV2 struct {
	Best                  []*PathV2
	Candidates            []*PathV2
	TokenPrice            float64
	NativeChainTokenPrice float64
}

type GraphV2 = []*NodeV2

type NodeV2 struct {
	Path     *PathV2
	Children GraphV2
}

func newSuggestedRoutesV2(
	amountIn *big.Int,
	candidates []*PathV2,
	fromLockedAmount map[uint64]*hexutil.Big,
	tokenPrice float64,
	nativeChainTokenPrice float64,
) *SuggestedRoutesV2 {
	suggestedRoutes := &SuggestedRoutesV2{
		Candidates:            candidates,
		Best:                  candidates,
		TokenPrice:            tokenPrice,
		NativeChainTokenPrice: nativeChainTokenPrice,
	}
	if len(candidates) == 0 {
		return suggestedRoutes
	}

	node := &NodeV2{
		Path:     nil,
		Children: buildGraphV2(amountIn, candidates, 0, []uint64{}),
	}
	routes := node.buildAllRoutesV2()
	routes = filterRoutesV2(routes, amountIn, fromLockedAmount)
	best := findBestV2(routes, tokenPrice, nativeChainTokenPrice)

	if len(best) > 0 {
		sort.Slice(best, func(i, j int) bool {
			return best[i].AmountInLocked
		})
		rest := new(big.Int).Set(amountIn)
		for _, path := range best {
			diff := new(big.Int).Sub(rest, path.AmountIn.ToInt())
			if diff.Cmp(pathprocessor.ZeroBigIntValue) >= 0 {
				path.AmountIn = (*hexutil.Big)(path.AmountIn.ToInt())
			} else {
				path.AmountIn = (*hexutil.Big)(new(big.Int).Set(rest))
			}
			rest.Sub(rest, path.AmountIn.ToInt())
		}
	}

	suggestedRoutes.Best = best
	return suggestedRoutes
}

func newNodeV2(path *PathV2) *NodeV2 {
	return &NodeV2{Path: path, Children: make(GraphV2, 0)}
}

func buildGraphV2(AmountIn *big.Int, routes []*PathV2, level int, sourceChainIDs []uint64) GraphV2 {
	graph := make(GraphV2, 0)
	for _, route := range routes {
		found := false
		for _, chainID := range sourceChainIDs {
			if chainID == route.FromChain.ChainID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		node := newNodeV2(route)

		newRoutes := make([]*PathV2, 0)
		for _, r := range routes {
			if route.Equal(r) {
				continue
			}
			newRoutes = append(newRoutes, r)
		}

		newAmountIn := new(big.Int).Sub(AmountIn, route.AmountIn.ToInt())
		if newAmountIn.Sign() > 0 {
			newSourceChainIDs := make([]uint64, len(sourceChainIDs))
			copy(newSourceChainIDs, sourceChainIDs)
			newSourceChainIDs = append(newSourceChainIDs, route.FromChain.ChainID)
			node.Children = buildGraphV2(newAmountIn, newRoutes, level+1, newSourceChainIDs)

			if len(node.Children) == 0 {
				continue
			}
		}

		graph = append(graph, node)
	}

	return graph
}

func (n NodeV2) buildAllRoutesV2() [][]*PathV2 {
	res := make([][]*PathV2, 0)

	if len(n.Children) == 0 && n.Path != nil {
		res = append(res, []*PathV2{n.Path})
	}

	for _, node := range n.Children {
		for _, route := range node.buildAllRoutesV2() {
			extendedRoute := route
			if n.Path != nil {
				extendedRoute = append([]*PathV2{n.Path}, route...)
			}
			res = append(res, extendedRoute)
		}
	}

	return res
}

func findBestV2(routes [][]*PathV2, tokenPrice float64, nativeChainTokenPrice float64) []*PathV2 {
	var best []*PathV2
	bestCost := big.NewFloat(math.Inf(1))
	for _, route := range routes {
		currentCost := big.NewFloat(0)
		for _, path := range route {
			tokenDenominator := big.NewFloat(math.Pow(10, float64(path.FromToken.Decimals)))

			path.requiredTokenBalance = new(big.Int).Set(path.AmountIn.ToInt())
			path.requiredNativeBalance = big.NewInt(0)

			// ecaluate the cost of the path
			pathCost := big.NewFloat(0)
			nativeTokenPrice := new(big.Float).SetFloat64(nativeChainTokenPrice)

			if path.TxBaseFee != nil && path.TxPriorityFee != nil {
				feePerGas := new(big.Int).Add(path.TxBaseFee.ToInt(), path.TxPriorityFee.ToInt())
				txFeeInWei := new(big.Int).Mul(feePerGas, big.NewInt(int64(path.TxGasAmount)))
				txFeeInEth := gweiToEth(weiToGwei(txFeeInWei))

				path.requiredNativeBalance.Add(path.requiredNativeBalance, txFeeInWei)
				pathCost = new(big.Float).Mul(txFeeInEth, nativeTokenPrice)
			}

			if path.TxBonderFees != nil && path.TxBonderFees.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
				path.requiredTokenBalance.Add(path.requiredTokenBalance, path.TxBonderFees.ToInt())
				pathCost.Add(pathCost, new(big.Float).Mul(
					new(big.Float).Quo(new(big.Float).SetInt(path.TxBonderFees.ToInt()), tokenDenominator),
					new(big.Float).SetFloat64(tokenPrice)))

			}

			if path.TxL1Fee != nil && path.TxL1Fee.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
				l1FeeInWei := path.TxL1Fee.ToInt()
				l1FeeInEth := gweiToEth(weiToGwei(l1FeeInWei))

				path.requiredNativeBalance.Add(path.requiredNativeBalance, l1FeeInWei)
				pathCost.Add(pathCost, new(big.Float).Mul(l1FeeInEth, nativeTokenPrice))
			}

			if path.TxTokenFees != nil && path.TxTokenFees.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 && path.FromToken != nil {
				path.requiredTokenBalance.Add(path.requiredTokenBalance, path.TxTokenFees.ToInt())
				pathCost.Add(pathCost, new(big.Float).Mul(
					new(big.Float).Quo(new(big.Float).SetInt(path.TxTokenFees.ToInt()), tokenDenominator),
					new(big.Float).SetFloat64(tokenPrice)))
			}

			if path.ApprovalRequired {
				if path.ApprovalBaseFee != nil && path.ApprovalPriorityFee != nil {
					feePerGas := new(big.Int).Add(path.ApprovalBaseFee.ToInt(), path.ApprovalPriorityFee.ToInt())
					txFeeInWei := new(big.Int).Mul(feePerGas, big.NewInt(int64(path.ApprovalGasAmount)))
					txFeeInEth := gweiToEth(weiToGwei(txFeeInWei))

					path.requiredNativeBalance.Add(path.requiredNativeBalance, txFeeInWei)
					pathCost.Add(pathCost, new(big.Float).Mul(txFeeInEth, nativeTokenPrice))
				}

				if path.ApprovalL1Fee != nil {
					l1FeeInWei := path.ApprovalL1Fee.ToInt()
					l1FeeInEth := gweiToEth(weiToGwei(l1FeeInWei))

					path.requiredNativeBalance.Add(path.requiredNativeBalance, l1FeeInWei)
					pathCost.Add(pathCost, new(big.Float).Mul(l1FeeInEth, nativeTokenPrice))
				}
			}

			currentCost = new(big.Float).Add(currentCost, pathCost)
		}

		if currentCost.Cmp(bestCost) == -1 {
			best = route
			bestCost = currentCost
		}
	}

	return best
}

func validateInputData(input *RouteInputParams) error {
	if input.SendType == ENSRegister {
		if input.Username == "" || input.PublicKey == "" {
			return errors.New("username and public key are required for ENSRegister")
		}
		if input.TestnetMode {
			if input.TokenID != pathprocessor.SttSymbol {
				return errors.New("only STT is supported for ENSRegister on testnet")
			}
		} else {
			if input.TokenID != pathprocessor.SntSymbol {
				return errors.New("only SNT is supported for ENSRegister")
			}
		}
		return nil
	}

	if input.SendType == ENSRelease {
		if input.Username == "" {
			return errors.New("username is required for ENSRelease")
		}
	}

	if input.SendType == ENSSetPubKey {
		if input.Username == "" || input.PublicKey == "" || ens.ValidateENSUsername(input.Username) != nil {
			return errors.New("username and public key are required for ENSSetPubKey")
		}
	}

	if input.SendType == StickersBuy {
		if input.PackID == nil {
			return errors.New("packID is required for StickersBuy")
		}
	}

	if input.SendType == Swap {
		if input.ToTokenID == "" {
			return errors.New("toTokenID is required for Swap")
		}
		if input.TokenID == input.ToTokenID {
			return errors.New("tokenID and toTokenID must be different")
		}

		// we can do this check, cause AmountIn is required in `RouteInputParams`
		if input.AmountIn.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 &&
			input.AmountOut != nil &&
			input.AmountOut.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			return errors.New("only one of amountIn or amountOut can be set")
		}

		if input.AmountIn.ToInt().Sign() < 0 {
			return errors.New("amountIn must be positive")
		}

		if input.AmountOut != nil && input.AmountOut.ToInt().Sign() < 0 {
			return errors.New("amountOut must be positive")
		}
	}

	if input.FromLockedAmount != nil && len(input.FromLockedAmount) > 0 {
		for chainID, amount := range input.FromLockedAmount {
			if input.TestnetMode {
				if !supportedTestNetworks[chainID] {
					return errors.New("locked amount is not supported for the selected network")
				}
			} else {
				if !supportedNetworks[chainID] {
					return errors.New("locked amount is not supported for the selected network")
				}
			}

			if amount == nil || amount.ToInt().Sign() < 0 {
				return errors.New("locked amount must be positive")
			}
		}
	}

	return nil
}

func (r *Router) SuggestedRoutesV2(ctx context.Context, input *RouteInputParams) (*SuggestedRoutesV2, error) {
	err := validateInputData(input)
	if err != nil {
		return nil, err
	}

	networks, err := r.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, err
	}

	var (
		group      = async.NewAtomicGroup(ctx)
		mu         sync.Mutex
		candidates = make([]*PathV2, 0)
	)

	for networkIdx := range networks {
		network := networks[networkIdx]
		if network.IsTest != input.TestnetMode {
			continue
		}

		if containsNetworkChainID(network, input.DisabledFromChainIDs) {
			continue
		}

		if !input.SendType.isAvailableFor(network) {
			continue
		}

		var (
			token   *walletToken.Token
			toToken *walletToken.Token
		)

		token = input.SendType.FindToken(r.tokenManager, r.collectiblesService, input.AddrFrom, network, input.TokenID)
		if token == nil {
			continue
		}

		if input.SendType == Swap {
			toToken = input.SendType.FindToken(r.tokenManager, r.collectiblesService, common.Address{}, network, input.ToTokenID)
		}

		amountLocked := false
		amountToSend := input.AmountIn.ToInt()
		if lockedAmount, ok := input.FromLockedAmount[network.ChainID]; ok {
			amountToSend = lockedAmount.ToInt()
			amountLocked = true
		}
		if len(input.FromLockedAmount) > 0 {
			for chainID, lockedAmount := range input.FromLockedAmount {
				if chainID == network.ChainID {
					continue
				}
				amountToSend = new(big.Int).Sub(amountToSend, lockedAmount.ToInt())
			}
		}

		group.Add(func(c context.Context) error {
			if err != nil {
				return err
			}

			for _, pProcessor := range r.pathProcessors {
				// With the condition below we're eliminating `Swap` as potential path that can participate in calculating the best route
				// once we decide to inlcude `Swap` in the calculation we need to update `canUseProcessor` function.
				// This also applies to including another (Celer) bridge in the calculation.
				// TODO:
				// this algorithm, includeing finding the best route, has to be updated to include more bridges and one (for now) or more swap options
				// it means that candidates should not be treated linearly, but improve the logic to have multiple routes with different processors of the same type.
				// Example:
				// Routes for sending SNT from Ethereum to Optimism can be:
				// 1. Swap SNT(mainnet) to ETH(mainnet); then bridge via Hop ETH(mainnet) to ETH(opt); then Swap ETH(opt) to SNT(opt); then send SNT (opt) to the destination
				// 2. Swap SNT(mainnet) to ETH(mainnet); then bridge via Celer ETH(mainnet) to ETH(opt); then Swap ETH(opt) to SNT(opt); then send SNT (opt) to the destination
				// 3. Swap SNT(mainnet) to USDC(mainnet); then bridge via Hop USDC(mainnet) to USDC(opt); then Swap USDC(opt) to SNT(opt); then send SNT (opt) to the destination
				// 4. Swap SNT(mainnet) to USDC(mainnet); then bridge via Celer USDC(mainnet) to USDC(opt); then Swap USDC(opt) to SNT(opt); then send SNT (opt) to the destination
				// 5. ...
				// 6. ...
				//
				// With the current routing algorithm atm we're not able to generate all possible routes.
				if !input.SendType.canUseProcessor(pProcessor) {
					continue
				}

				for _, dest := range networks {
					if dest.IsTest != input.TestnetMode {
						continue
					}

					if !input.SendType.isAvailableFor(network) {
						continue
					}

					if !input.SendType.isAvailableBetween(network, dest) {
						continue
					}

					if containsNetworkChainID(dest, input.DisabledToChainIDs) {
						continue
					}

					processorInputParams := pathprocessor.ProcessorInputParams{
						FromChain: network,
						ToChain:   dest,
						FromToken: token,
						ToToken:   toToken,
						ToAddr:    input.AddrTo,
						FromAddr:  input.AddrFrom,
						AmountIn:  amountToSend,
						AmountOut: input.AmountOut.ToInt(),

						Username:  input.Username,
						PublicKey: input.PublicKey,
						PackID:    input.PackID.ToInt(),
					}

					can, err := pProcessor.AvailableFor(processorInputParams)
					if err != nil || !can {
						continue
					}

					bonderFees, tokenFees, err := pProcessor.CalculateFees(processorInputParams)
					if err != nil {
						continue
					}

					gasLimit, err := pProcessor.EstimateGas(processorInputParams)
					if err != nil {
						continue
					}

					approvalContractAddress, err := pProcessor.GetContractAddress(processorInputParams)
					if err != nil {
						continue
					}
					approvalRequired, approvalAmountRequired, approvalGasLimit, l1ApprovalFee, err := r.requireApproval(ctx, input.SendType, &approvalContractAddress, input.AddrFrom, network, token, amountToSend)
					if err != nil {
						continue
					}

					var l1FeeWei uint64
					if input.SendType.needL1Fee() {

						txInputData, err := pProcessor.PackTxInputData(processorInputParams)
						if err != nil {
							continue
						}

						l1FeeWei, _ = r.feesManager.GetL1Fee(ctx, network.ChainID, txInputData)
					}

					fees, err := r.feesManager.SuggestedFees(ctx, network.ChainID)
					if err != nil {
						continue
					}

					amountOut, err := pProcessor.CalculateAmountOut(processorInputParams)
					if err != nil {
						continue
					}

					maxFeesPerGas := fees.feeFor(input.GasFeeMode)

					estimatedTime := r.feesManager.TransactionEstimatedTime(ctx, network.ChainID, maxFeesPerGas)
					if approvalRequired && estimatedTime < MoreThanFiveMinutes {
						estimatedTime += 1
					}

					mu.Lock()
					candidates = append(candidates, &PathV2{
						ProcessorName:  pProcessor.Name(),
						FromChain:      network,
						ToChain:        dest,
						FromToken:      token,
						AmountIn:       (*hexutil.Big)(amountToSend),
						AmountInLocked: amountLocked,
						AmountOut:      (*hexutil.Big)(amountOut),

						SuggestedLevelsForMaxFeesPerGas: fees.MaxFeesLevels,

						TxBaseFee:     (*hexutil.Big)(fees.BaseFee),
						TxPriorityFee: (*hexutil.Big)(fees.MaxPriorityFeePerGas),
						TxGasAmount:   gasLimit,
						TxBonderFees:  (*hexutil.Big)(bonderFees),
						TxTokenFees:   (*hexutil.Big)(tokenFees),
						TxL1Fee:       (*hexutil.Big)(big.NewInt(int64(l1FeeWei))),

						ApprovalRequired:        approvalRequired,
						ApprovalAmountRequired:  (*hexutil.Big)(approvalAmountRequired),
						ApprovalContractAddress: &approvalContractAddress,
						ApprovalBaseFee:         (*hexutil.Big)(fees.BaseFee),
						ApprovalPriorityFee:     (*hexutil.Big)(fees.MaxPriorityFeePerGas),
						ApprovalGasAmount:       approvalGasLimit,
						ApprovalL1Fee:           (*hexutil.Big)(big.NewInt(int64(l1ApprovalFee))),

						EstimatedTime: estimatedTime,
					})
					mu.Unlock()
				}
			}
			return nil
		})
	}

	group.Wait()

	prices, err := input.SendType.FetchPrices(r.marketManager, input.TokenID)
	if err != nil {
		return nil, err
	}

	suggestedRoutes := newSuggestedRoutesV2(input.AmountIn.ToInt(), candidates, input.FromLockedAmount, prices[input.TokenID], prices["ETH"])

	// check the best route for the required balances
	for _, path := range suggestedRoutes.Best {

		if path.requiredTokenBalance != nil && path.requiredTokenBalance.Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			tokenBalance := big.NewInt(1)
			if input.SendType == ERC1155Transfer {
				tokenBalance, err = r.getERC1155Balance(ctx, path.FromChain, path.FromToken, input.AddrFrom)
				if err != nil {
					return nil, err
				}
			} else if input.SendType != ERC721Transfer {
				tokenBalance, err = r.getBalance(ctx, path.FromChain, path.FromToken, input.AddrFrom)
				if err != nil {
					return nil, err
				}
			}

			if tokenBalance.Cmp(path.requiredTokenBalance) == -1 {
				return suggestedRoutes, errors.New("not enough token balance")
			}
		}

		nativeToken := r.tokenManager.FindToken(path.FromChain, path.FromChain.NativeCurrencySymbol)
		if nativeToken == nil {
			return nil, errors.New("native token not found")
		}

		nativeBalance, err := r.getBalance(ctx, path.FromChain, nativeToken, input.AddrFrom)
		if err != nil {
			return nil, err
		}

		if nativeBalance.Cmp(path.requiredNativeBalance) == -1 {
			return suggestedRoutes, errors.New("not enough native balance")
		}
	}

	return suggestedRoutes, nil
}
