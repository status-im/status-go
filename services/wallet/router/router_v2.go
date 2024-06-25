package router

import (
	"context"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
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
	TestnetMode          bool

	// For send types like EnsRegister, EnsRelease, EnsSetPubKey, StickersBuy
	Username  string       `json:"username"`
	PublicKey string       `json:"publicKey"`
	PackID    *hexutil.Big `json:"packID"`

	// TODO: Remove two fields below once we implement a better solution for tests
	// Currently used for tests only
	testsMode  bool
	testParams *routerTestParams
}

type routerTestParams struct {
	tokenFrom             *walletToken.Token
	tokenPrices           map[string]float64
	estimationMap         map[string]uint64   // [processor-name, estimated-value]
	bonderFeeMap          map[string]*big.Int // [token-symbol, bonder-fee]
	suggestedFees         *SuggestedFees
	baseFee               *big.Int
	balanceMap            map[string]*big.Int // [token-symbol, balance]
	approvalGasEstimation uint64
	approvalL1Fee         uint64
}

type processorRouteCalculations struct {
	can                     bool
	bonderFees              *big.Int
	tokenFees               *big.Int
	gasLimit                uint64
	approvalContractAddress common.Address
	txInputData             []byte
	amountOut               *big.Int
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

			path.requiredTokenBalance = big.NewInt(0)
			path.requiredNativeBalance = big.NewInt(0)
			if path.FromToken.IsNative() {
				path.requiredNativeBalance.Add(path.requiredNativeBalance, path.AmountIn.ToInt())
			} else {
				path.requiredTokenBalance.Add(path.requiredTokenBalance, path.AmountIn.ToInt())
			}

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
				if path.FromToken.IsNative() {
					path.requiredNativeBalance.Add(path.requiredNativeBalance, path.TxBonderFees.ToInt())
				} else {
					path.requiredTokenBalance.Add(path.requiredTokenBalance, path.TxBonderFees.ToInt())
				}
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
				if path.FromToken.IsNative() {
					path.requiredNativeBalance.Add(path.requiredNativeBalance, path.TxTokenFees.ToInt())
				} else {
					path.requiredTokenBalance.Add(path.requiredTokenBalance, path.TxTokenFees.ToInt())
				}
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
			return ErrUsernameAndPubKeyRequiredForENSRegister
		}
		if input.TestnetMode {
			if input.TokenID != pathprocessor.SttSymbol {
				return ErrOnlySTTSupportedForENSRegisterOnTestnet
			}
		} else {
			if input.TokenID != pathprocessor.SntSymbol {
				return ErrOnlySTTSupportedForENSReleaseOnTestnet
			}
		}
		return nil
	}

	if input.SendType == ENSRelease {
		if input.Username == "" {
			return ErrUsernameRequiredForENSRelease
		}
	}

	if input.SendType == ENSSetPubKey {
		if input.Username == "" || input.PublicKey == "" || ens.ValidateENSUsername(input.Username) != nil {
			return ErrUsernameAndPubKeyRequiredForENSSetPubKey
		}
	}

	if input.SendType == StickersBuy {
		if input.PackID == nil {
			return ErrPackIDRequiredForStickersBuy
		}
	}

	if input.SendType == Swap {
		if input.ToTokenID == "" {
			return ErrToTokenIDRequiredForSwap
		}
		if input.TokenID == input.ToTokenID {
			return ErrTokenIDAndToTokenIDDifferent
		}

		// we can do this check, cause AmountIn is required in `RouteInputParams`
		if input.AmountIn.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 &&
			input.AmountOut != nil &&
			input.AmountOut.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			return ErrOnlyOneOfAmountInOrOutSet
		}

		if input.AmountIn.ToInt().Sign() < 0 {
			return ErrAmountInMustBePositive
		}

		if input.AmountOut != nil && input.AmountOut.ToInt().Sign() < 0 {
			return ErrAmountOutMustBePositive
		}
	}

	if input.FromLockedAmount != nil && len(input.FromLockedAmount) > 0 {
		suppNetworks := copyMap(supportedNetworks)
		if input.TestnetMode {
			suppNetworks = copyMap(supportedTestNetworks)
		}

		totalLockedAmount := big.NewInt(0)

		for chainID, amount := range input.FromLockedAmount {
			if input.TestnetMode {
				if !supportedTestNetworks[chainID] {
					return ErrLockedAmountNotSupportedForNetwork
				}
			} else {
				if !supportedNetworks[chainID] {
					return ErrLockedAmountNotSupportedForNetwork
				}
			}

			delete(suppNetworks, chainID)

			totalLockedAmount = new(big.Int).Add(totalLockedAmount, amount.ToInt())

			if amount == nil || amount.ToInt().Sign() < 0 {
				return ErrLockedAmountMustBePositive
			}
		}

		if totalLockedAmount.Cmp(input.AmountIn.ToInt()) > 0 {
			return ErrLockedAmountExceedsTotalSendAmount
		} else if totalLockedAmount.Cmp(input.AmountIn.ToInt()) < 0 && len(suppNetworks) == 0 {
			return ErrLockedAmountLessThanSendAmountAllNetworks
		}
	}

	return nil
}

func (r *Router) SuggestedRoutesV2(ctx context.Context, input *RouteInputParams, networks []*params.Network) (*SuggestedRoutesV2, error) {
	// clear all processors
	for _, processor := range r.pathProcessors {
		if clearable, ok := processor.(pathprocessor.PathProcessorClearable); ok {
			clearable.Clear()
		}
	}

	err := validateInputData(input)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	candidates, err := r.resolveCandidates(ctx, input, networks)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	return r.resolveRoutes(ctx, input, candidates)
}

func (r *Router) resolveCandidates(ctx context.Context, input *RouteInputParams, networks []*params.Network) (candidates []*PathV2, err error) {
	var (
		group = async.NewAtomicGroup(ctx)
		mu    sync.Mutex
	)

	for networkIdx := range networks {
		network := networks[networkIdx]
		group.Add(func(c context.Context) error {
			candidatesForNetwork := r.resolveCandidatesForNetwork(ctx, input, network, networks)
			mu.Lock()
			candidates = append(candidates, candidatesForNetwork...)
			mu.Unlock()
			return nil
		})
	}

	group.Wait()
	return candidates, nil
}

func validateInput(input *RouteInputParams, network *params.Network) bool {
	if network.IsTest != input.TestnetMode {
		return false
	}

	if containsNetworkChainID(network, input.DisabledFromChainIDs) {
		return false
	}

	if !input.SendType.isAvailableFor(network) {
		return false
	}

	return true
}

func (r *Router) resolveCandidatesForNetwork(ctx context.Context, input *RouteInputParams, network *params.Network, networks []*params.Network) []*PathV2 {
	candidates := make([]*PathV2, 0)

	if !validateInput(input, network) {
		return nil
	}

	var (
		token   *walletToken.Token
		toToken *walletToken.Token
	)

	token = input.SendType.FindToken(r.tokenManager, r.collectiblesService, input.AddrFrom, network, input.TokenID)
	if token == nil {
		return nil
	}

	if input.SendType == Swap {
		toToken = input.SendType.FindToken(r.tokenManager, r.collectiblesService, common.Address{}, network, input.ToTokenID)
	}

	amountLocked := false
	amountToSend := input.AmountIn.ToInt()
	if lockedAmount, ok := input.FromLockedAmount[network.ChainID]; ok {
		amountToSend = lockedAmount.ToInt()
		amountLocked = true
	} else if len(input.FromLockedAmount) > 0 {
		for chainID, lockedAmount := range input.FromLockedAmount {
			if chainID == network.ChainID {
				continue
			}
			amountToSend = new(big.Int).Sub(amountToSend, lockedAmount.ToInt())
		}
	}

	for _, pProcessor := range r.pathProcessors {
		// With the current routing algorithm atm we're not able to generate all possible routes.
		if !input.SendType.canUseProcessor(pProcessor) {
			continue
		}

		rCandidates := r.CandidateResolver.resolveCandidatesForProcessor(ctx, input, network, token, toToken, amountToSend, amountLocked, pProcessor, networks)
		candidates = append(candidates, rCandidates...)
	}

	return candidates
}

func (r *DefaultCandidateResolver) resolveCandidatesForProcessor(ctx context.Context, input *RouteInputParams, network *params.Network, token, toToken *walletToken.Token, amountToSend *big.Int, amountLocked bool, pProcessor pathprocessor.PathProcessor, networks []*params.Network) []*PathV2 {
	candidates := make([]*PathV2, 0)
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

	for _, dest := range networks {
		if !validateInput(input, dest) {
			continue
		}

		if !input.SendType.isAvailableBetween(network, dest) {
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
			AmountOut: amountToSend,

			Username:  input.Username,
			PublicKey: input.PublicKey,
			PackID:    input.PackID.ToInt(),
		}

		calc, err := resolveCandidatesForProcessorParams(processorInputParams, pProcessor, input.SendType.needL1Fee())
		if err != nil {
			continue
		}

		approvalRequired, approvalAmountRequired, approvalGasLimit, l1ApprovalFee, err := r.requireApproval(ctx, input.SendType, &calc.approvalContractAddress, processorInputParams)
		if err != nil {
			continue
		}

		fees, l1FeeWei, estimatedTime, err := r.Estimator.Estimate(ctx, network.ChainID, calc.txInputData, input.SendType.needL1Fee(), input.GasFeeMode)
		if err != nil {
			continue
		}

		if approvalRequired && estimatedTime < MoreThanFiveMinutes {
			estimatedTime += 1
		}

		candidates = append(candidates, &PathV2{
			ProcessorName:  pProcessor.Name(),
			FromChain:      network,
			ToChain:        dest,
			FromToken:      token,
			AmountIn:       (*hexutil.Big)(amountToSend),
			AmountInLocked: amountLocked,
			AmountOut:      (*hexutil.Big)(calc.amountOut),

			SuggestedLevelsForMaxFeesPerGas: fees.MaxFeesLevels,

			TxBaseFee:     (*hexutil.Big)(fees.BaseFee),
			TxPriorityFee: (*hexutil.Big)(fees.MaxPriorityFeePerGas),
			TxGasAmount:   calc.gasLimit,
			TxBonderFees:  (*hexutil.Big)(calc.bonderFees),
			TxTokenFees:   (*hexutil.Big)(calc.tokenFees),
			TxL1Fee:       (*hexutil.Big)(big.NewInt(int64(l1FeeWei))),

			ApprovalRequired:        approvalRequired,
			ApprovalAmountRequired:  (*hexutil.Big)(approvalAmountRequired),
			ApprovalContractAddress: &calc.approvalContractAddress,
			ApprovalBaseFee:         (*hexutil.Big)(fees.BaseFee),
			ApprovalPriorityFee:     (*hexutil.Big)(fees.MaxPriorityFeePerGas),
			ApprovalGasAmount:       approvalGasLimit,
			ApprovalL1Fee:           (*hexutil.Big)(big.NewInt(int64(l1ApprovalFee))),

			EstimatedTime: estimatedTime,
		})
	}

	return candidates
}

func resolveCandidatesForProcessorParams(processorInputParams pathprocessor.ProcessorInputParams, pProcessor pathprocessor.PathProcessor, needL1Fee bool) (*processorRouteCalculations, error) {
	processorCalculations := processorRouteCalculations{}
	can, err := pProcessor.AvailableFor(processorInputParams)
	if err != nil || !can {
		return nil, err
	}

	processorCalculations.can = can

	bonderFees, tokenFees, err := pProcessor.CalculateFees(processorInputParams)
	if err != nil {
		return nil, err
	}

	processorCalculations.bonderFees = bonderFees
	processorCalculations.tokenFees = tokenFees

	gasLimit, err := pProcessor.EstimateGas(processorInputParams)
	if err != nil {
		return nil, err
	}

	processorCalculations.gasLimit = gasLimit

	approvalContractAddress, err := pProcessor.GetContractAddress(processorInputParams)
	if err != nil {
		return nil, err
	}

	processorCalculations.approvalContractAddress = approvalContractAddress

	if needL1Fee {
		txInputData, err := pProcessor.PackTxInputData(processorInputParams)
		if err != nil {
			return nil, err
		}

		processorCalculations.txInputData = txInputData
	}

	amountOut, err := pProcessor.CalculateAmountOut(processorInputParams)
	if err != nil {
		return nil, err
	}

	processorCalculations.amountOut = amountOut

	return &processorCalculations, nil
}

func (r *Router) resolveRoutes(ctx context.Context, input *RouteInputParams, candidates []*PathV2) (suggestedRoutes *SuggestedRoutesV2, err error) {
	var prices map[string]float64
	if input.testsMode {
		prices = input.testParams.tokenPrices
	} else {
		prices, err = input.SendType.FetchPrices(r.marketManager, input.TokenID)
		if err != nil {
			return nil, errors.CreateErrorResponseFromError(err)
		}
	}

	suggestedRoutes = newSuggestedRoutesV2(input.AmountIn.ToInt(), candidates, input.FromLockedAmount, prices[input.TokenID], prices[pathprocessor.EthSymbol])

	// check the best route for the required balances
	for _, path := range suggestedRoutes.Best {
		if path.requiredTokenBalance != nil && path.requiredTokenBalance.Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			tokenBalance := big.NewInt(1)
			if input.testsMode {
				if val, ok := input.testParams.balanceMap[path.FromToken.Symbol]; ok {
					tokenBalance = val
				}
			} else {
				if input.SendType == ERC1155Transfer {
					tokenBalance, err = r.getERC1155Balance(ctx, path.FromChain, path.FromToken, input.AddrFrom)
					if err != nil {
						return nil, errors.CreateErrorResponseFromError(err)
					}
				} else if input.SendType != ERC721Transfer {
					tokenBalance, err = r.getBalance(ctx, path.FromChain, path.FromToken, input.AddrFrom)
					if err != nil {
						return nil, errors.CreateErrorResponseFromError(err)
					}
				}
			}

			if tokenBalance.Cmp(path.requiredTokenBalance) == -1 {
				return suggestedRoutes, ErrNotEnoughTokenBalance
			}
		}

		nativeBalance := big.NewInt(0)
		if input.testsMode {
			if val, ok := input.testParams.balanceMap[pathprocessor.EthSymbol]; ok {
				nativeBalance = val
			}
		} else {
			nativeToken := r.tokenManager.FindToken(path.FromChain, path.FromChain.NativeCurrencySymbol)
			if nativeToken == nil {
				return nil, ErrNativeTokenNotFound
			}

			nativeBalance, err = r.getBalance(ctx, path.FromChain, nativeToken, input.AddrFrom)
			if err != nil {
				return nil, errors.CreateErrorResponseFromError(err)
			}
		}

		if nativeBalance.Cmp(path.requiredNativeBalance) == -1 {
			return suggestedRoutes, ErrNotEnoughNativeBalance
		}
	}

	return suggestedRoutes, nil
}
