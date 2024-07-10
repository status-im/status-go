package router

import (
	"context"
	"fmt"
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
	"github.com/status-im/status-go/signal"
)

var (
	routerTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
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
	Uuid                 string                  `json:"uuid"`
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
	testnetMode          bool

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

type amountOption struct {
	amount *big.Int
	locked bool
}

func makeBalanceKey(chainID uint64, symbol string) string {
	return fmt.Sprintf("%d-%s", chainID, symbol)
}

type PathV2 struct {
	ProcessorName  string
	FromChain      *params.Network    // Source chain
	ToChain        *params.Network    // Destination chain
	FromToken      *walletToken.Token // Source token
	ToToken        *walletToken.Token // Destination token, set if applicable
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
	Uuid                  string
	Best                  []*PathV2
	Candidates            []*PathV2
	TokenPrice            float64
	NativeChainTokenPrice float64
}

type SuggestedRoutesV2Response struct {
	Uuid                  string                `json:"Uuid"`
	Best                  []*PathV2             `json:"Best,omitempty"`
	Candidates            []*PathV2             `json:"Candidates,omitempty"`
	TokenPrice            *float64              `json:"TokenPrice,omitempty"`
	NativeChainTokenPrice *float64              `json:"NativeChainTokenPrice,omitempty"`
	ErrorResponse         *errors.ErrorResponse `json:"ErrorResponse,omitempty"`
}

type GraphV2 []*NodeV2

type NodeV2 struct {
	Path     *PathV2
	Children GraphV2
}

func newSuggestedRoutesV2(
	uuid string,
	amountIn *big.Int,
	candidates []*PathV2,
	fromLockedAmount map[uint64]*hexutil.Big,
	tokenPrice float64,
	nativeChainTokenPrice float64,
) (*SuggestedRoutesV2, [][]*PathV2) {
	suggestedRoutes := &SuggestedRoutesV2{
		Uuid:                  uuid,
		Candidates:            candidates,
		Best:                  candidates,
		TokenPrice:            tokenPrice,
		NativeChainTokenPrice: nativeChainTokenPrice,
	}
	if len(candidates) == 0 {
		return suggestedRoutes, nil
	}

	node := &NodeV2{
		Path:     nil,
		Children: buildGraphV2(amountIn, candidates, 0, []uint64{}),
	}
	allRoutes := node.buildAllRoutesV2()
	allRoutes = filterRoutesV2(allRoutes, amountIn, fromLockedAmount)

	return suggestedRoutes, allRoutes
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
			return ErrENSRegisterRequiresUsernameAndPubKey
		}
		if input.testnetMode {
			if input.TokenID != pathprocessor.SttSymbol {
				return ErrENSRegisterTestnetSTTOnly
			}
		} else {
			if input.TokenID != pathprocessor.SntSymbol {
				return ErrENSRegisterMainnetSNTOnly
			}
		}
		return nil
	}

	if input.SendType == ENSRelease {
		if input.Username == "" {
			return ErrENSReleaseRequiresUsername
		}
	}

	if input.SendType == ENSSetPubKey {
		if input.Username == "" || input.PublicKey == "" {
			return ErrENSSetPubKeyRequiresUsernameAndPubKey
		}

		if ens.ValidateENSUsername(input.Username) != nil {
			return ErrENSSetPubKeyInvalidUsername
		}
	}

	if input.SendType == StickersBuy {
		if input.PackID == nil {
			return ErrStickersBuyRequiresPackID
		}
	}

	if input.SendType == Swap {
		if input.ToTokenID == "" {
			return ErrSwapRequiresToTokenID
		}
		if input.TokenID == input.ToTokenID {
			return ErrSwapTokenIDMustBeDifferent
		}

		if input.AmountIn != nil &&
			input.AmountOut != nil &&
			input.AmountIn.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 &&
			input.AmountOut.ToInt().Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			return ErrSwapAmountInAmountOutMustBeExclusive
		}

		if input.AmountIn != nil && input.AmountIn.ToInt().Sign() < 0 {
			return ErrSwapAmountInMustBePositive
		}

		if input.AmountOut != nil && input.AmountOut.ToInt().Sign() < 0 {
			return ErrSwapAmountOutMustBePositive
		}
	}

	return validateFromLockedAmount(input)
}

func validateFromLockedAmount(input *RouteInputParams) error {
	if input.FromLockedAmount == nil || len(input.FromLockedAmount) == 0 {
		return nil
	}

	var suppNetworks map[uint64]bool
	if input.testnetMode {
		suppNetworks = copyMap(supportedTestNetworks)
	} else {
		suppNetworks = copyMap(supportedNetworks)
	}

	totalLockedAmount := big.NewInt(0)
	excludedChainCount := 0

	for chainID, amount := range input.FromLockedAmount {
		if containsNetworkChainID(chainID, input.DisabledFromChainIDs) {
			return ErrDisabledChainFoundAmongLockedNetworks
		}

		if input.testnetMode {
			if !supportedTestNetworks[chainID] {
				return ErrLockedAmountNotSupportedForNetwork
			}
		} else {
			if !supportedNetworks[chainID] {
				return ErrLockedAmountNotSupportedForNetwork
			}
		}

		if amount == nil || amount.ToInt().Sign() < 0 {
			return ErrLockedAmountNotNegative
		}

		if !(amount.ToInt().Sign() > 0) {
			excludedChainCount++
		}
		delete(suppNetworks, chainID)
		totalLockedAmount = new(big.Int).Add(totalLockedAmount, amount.ToInt())
	}

	if (!input.testnetMode && excludedChainCount == len(supportedNetworks)) ||
		(input.testnetMode && excludedChainCount == len(supportedTestNetworks)) {
		return ErrLockedAmountExcludesAllSupported
	}

	if totalLockedAmount.Cmp(input.AmountIn.ToInt()) > 0 {
		return ErrLockedAmountExceedsTotalSendAmount
	} else if totalLockedAmount.Cmp(input.AmountIn.ToInt()) < 0 && len(suppNetworks) == 0 {
		return ErrLockedAmountLessThanSendAmountAllNetworks
	}
	return nil
}

func (r *Router) SuggestedRoutesV2Async(input *RouteInputParams) {
	r.scheduler.Enqueue(routerTask, func(ctx context.Context) (interface{}, error) {
		return r.SuggestedRoutesV2(ctx, input)
	}, func(result interface{}, taskType async.TaskType, err error) {
		routesResponse := SuggestedRoutesV2Response{
			Uuid: input.Uuid,
		}

		if err != nil {
			errorResponse := errors.CreateErrorResponseFromError(err)
			routesResponse.ErrorResponse = errorResponse.(*errors.ErrorResponse)
		}

		if suggestedRoutes, ok := result.(*SuggestedRoutesV2); ok && suggestedRoutes != nil {
			routesResponse.Best = suggestedRoutes.Best
			routesResponse.Candidates = suggestedRoutes.Candidates
			routesResponse.TokenPrice = &suggestedRoutes.TokenPrice
			routesResponse.NativeChainTokenPrice = &suggestedRoutes.NativeChainTokenPrice
		}

		signal.SendWalletEvent(signal.SuggestedRoutes, routesResponse)
	})
}

func (r *Router) StopSuggestedRoutesV2AsyncCalcualtion() {
	r.scheduler.Stop()
}

func (r *Router) SuggestedRoutesV2(ctx context.Context, input *RouteInputParams) (*SuggestedRoutesV2, error) {
	testnetMode, err := r.rpcClient.NetworkManager.GetTestNetworksEnabled()
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	input.testnetMode = testnetMode

	// clear all processors
	for _, processor := range r.pathProcessors {
		if clearable, ok := processor.(pathprocessor.PathProcessorClearable); ok {
			clearable.Clear()
		}
	}

	err = validateInputData(input)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	selectedFromChains, selectedTohains, err := r.getSelectedChains(input)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	balanceMap, err := r.getBalanceMapForTokenOnChains(ctx, input, selectedFromChains)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	candidates, err := r.resolveCandidates(ctx, input, selectedFromChains, selectedTohains, balanceMap)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	return r.resolveRoutes(ctx, input, candidates, balanceMap)
}

// getBalanceMapForTokenOnChains returns the balance map for passed address, where the key is in format "chainID-tokenSymbol" and
// value is the balance of the token. Native token (EHT) is always added to the balance map.
func (r *Router) getBalanceMapForTokenOnChains(ctx context.Context, input *RouteInputParams, selectedFromChains []*params.Network) (balanceMap map[string]*big.Int, err error) {
	if input.testsMode {
		return input.testParams.balanceMap, nil
	}

	balanceMap = make(map[string]*big.Int)

	for _, chain := range selectedFromChains {
		token := input.SendType.FindToken(r.tokenManager, r.collectiblesService, input.AddrFrom, chain, input.TokenID)
		if token == nil {
			continue
		}

		// add token balance for the chain
		tokenBalance := big.NewInt(1)
		if input.SendType == ERC1155Transfer {
			tokenBalance, err = r.getERC1155Balance(ctx, chain, token, input.AddrFrom)
			if err != nil {
				return nil, errors.CreateErrorResponseFromError(err)
			}
		} else if input.SendType != ERC721Transfer {
			tokenBalance, err = r.getBalance(ctx, chain.ChainID, token, input.AddrFrom)
			if err != nil {
				return nil, errors.CreateErrorResponseFromError(err)
			}
		}
		balanceMap[makeBalanceKey(chain.ChainID, token.Symbol)] = tokenBalance

		// add native token balance for the chain
		nativeToken := r.tokenManager.FindToken(chain, chain.NativeCurrencySymbol)
		if nativeToken == nil {
			return nil, ErrNativeTokenNotFound
		}

		nativeBalance, err := r.getBalance(ctx, chain.ChainID, nativeToken, input.AddrFrom)
		if err != nil {
			return nil, errors.CreateErrorResponseFromError(err)
		}
		balanceMap[makeBalanceKey(chain.ChainID, nativeToken.Symbol)] = nativeBalance
	}

	return
}

func (r *Router) getSelectedUnlockedChains(input *RouteInputParams, processingChain *params.Network, selectedFromChains []*params.Network) []*params.Network {
	selectedButNotLockedChains := []*params.Network{processingChain} // always add the processing chain at the beginning
	for _, net := range selectedFromChains {
		if net.ChainID == processingChain.ChainID {
			continue
		}
		if _, ok := input.FromLockedAmount[net.ChainID]; !ok {
			selectedButNotLockedChains = append(selectedButNotLockedChains, net)
		}
	}
	return selectedButNotLockedChains
}

func (r *Router) getOptionsForAmoutToSplitAccrossChainsForProcessingChain(input *RouteInputParams, amountToSplit *big.Int, processingChain *params.Network,
	selectedFromChains []*params.Network, balanceMap map[string]*big.Int) map[uint64][]amountOption {
	selectedButNotLockedChains := r.getSelectedUnlockedChains(input, processingChain, selectedFromChains)

	crossChainAmountOptions := make(map[uint64][]amountOption)
	for _, chain := range selectedButNotLockedChains {
		var (
			ok           bool
			tokenBalance *big.Int
		)

		if tokenBalance, ok = balanceMap[makeBalanceKey(chain.ChainID, input.TokenID)]; !ok {
			continue
		}

		if tokenBalance.Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			if tokenBalance.Cmp(amountToSplit) <= 0 {
				crossChainAmountOptions[chain.ChainID] = append(crossChainAmountOptions[chain.ChainID], amountOption{
					amount: tokenBalance,
					locked: false,
				})
				amountToSplit = new(big.Int).Sub(amountToSplit, tokenBalance)
			} else if amountToSplit.Cmp(pathprocessor.ZeroBigIntValue) > 0 {
				crossChainAmountOptions[chain.ChainID] = append(crossChainAmountOptions[chain.ChainID], amountOption{
					amount: amountToSplit,
					locked: false,
				})
				// break since amountToSplit is fully addressed and the rest is 0
				break
			}
		}
	}

	return crossChainAmountOptions
}

func (r *Router) getCrossChainsOptionsForSendingAmount(input *RouteInputParams, selectedFromChains []*params.Network,
	balanceMap map[string]*big.Int) map[uint64][]amountOption {
	// All we do in this block we're free to do, because of the validateInputData function which checks if the locked amount
	// was properly set and if there is something unexpected it will return an error and we will not reach this point
	finalCrossChainAmountOptions := make(map[uint64][]amountOption) // represents all possible amounts that can be sent from the "from" chain

	for _, selectedFromChain := range selectedFromChains {

		amountLocked := false
		amountToSend := input.AmountIn.ToInt()
		lockedAmount, fromChainLocked := input.FromLockedAmount[selectedFromChain.ChainID]
		if fromChainLocked {
			amountToSend = lockedAmount.ToInt()
			amountLocked = true
		} else if len(input.FromLockedAmount) > 0 {
			for chainID, lockedAmount := range input.FromLockedAmount {
				if chainID == selectedFromChain.ChainID {
					continue
				}
				amountToSend = new(big.Int).Sub(amountToSend, lockedAmount.ToInt())
			}
		}

		if amountToSend.Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			// add full amount always, cause we want to check for balance errors at the end of the routing algorithm
			// TODO: once we introduce bettwer error handling and start checking for the balance at the beginning of the routing algorithm
			// we can remove this line and optimize the routing algorithm more
			finalCrossChainAmountOptions[selectedFromChain.ChainID] = append(finalCrossChainAmountOptions[selectedFromChain.ChainID], amountOption{
				amount: amountToSend,
				locked: amountLocked,
			})

			if amountLocked {
				continue
			}

			// If the amount that need to be send is bigger than the balance on the chain, then we want to check options if that
			// amount can be splitted and sent across multiple chains.
			if input.SendType == Transfer && len(selectedFromChains) > 1 {
				// All we do in this block we're free to do, because of the validateInputData function which checks if the locked amount
				// was properly set and if there is something unexpected it will return an error and we will not reach this point
				amountToSplitAccrossChains := new(big.Int).Set(amountToSend)

				crossChainAmountOptions := r.getOptionsForAmoutToSplitAccrossChainsForProcessingChain(input, amountToSend, selectedFromChain, selectedFromChains, balanceMap)

				// sum up all the allocated amounts accorss all chains
				allocatedAmount := big.NewInt(0)
				for _, amountOptions := range crossChainAmountOptions {
					for _, amountOption := range amountOptions {
						allocatedAmount = new(big.Int).Add(allocatedAmount, amountOption.amount)
					}
				}

				// if the allocated amount is the same as the amount that need to be sent, then we can add the options to the finalCrossChainAmountOptions
				if allocatedAmount.Cmp(amountToSplitAccrossChains) == 0 {
					for cID, amountOptions := range crossChainAmountOptions {
						finalCrossChainAmountOptions[cID] = append(finalCrossChainAmountOptions[cID], amountOptions...)
					}
				}
			}
		}
	}

	return finalCrossChainAmountOptions
}

func (r *Router) findOptionsForSendingAmount(input *RouteInputParams, selectedFromChains []*params.Network,
	balanceMap map[string]*big.Int) (map[uint64][]amountOption, error) {

	crossChainAmountOptions := r.getCrossChainsOptionsForSendingAmount(input, selectedFromChains, balanceMap)

	// filter out duplicates values for the same chain
	for chainID, amountOptions := range crossChainAmountOptions {
		uniqueAmountOptions := make(map[string]amountOption)
		for _, amountOption := range amountOptions {
			uniqueAmountOptions[amountOption.amount.String()] = amountOption
		}

		crossChainAmountOptions[chainID] = make([]amountOption, 0)
		for _, amountOption := range uniqueAmountOptions {
			crossChainAmountOptions[chainID] = append(crossChainAmountOptions[chainID], amountOption)
		}
	}

	return crossChainAmountOptions, nil
}

func (r *Router) getSelectedChains(input *RouteInputParams) (selectedFromChains []*params.Network, selectedTohains []*params.Network, err error) {
	var networks []*params.Network
	networks, err = r.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, nil, errors.CreateErrorResponseFromError(err)
	}

	for _, network := range networks {
		if network.IsTest != input.testnetMode {
			continue
		}

		if !containsNetworkChainID(network.ChainID, input.DisabledFromChainIDs) {
			selectedFromChains = append(selectedFromChains, network)
		}

		if !containsNetworkChainID(network.ChainID, input.DisabledToChainIDs) {
			selectedTohains = append(selectedTohains, network)
		}
	}

	return selectedFromChains, selectedTohains, nil
}

func (r *Router) resolveCandidates(ctx context.Context, input *RouteInputParams, selectedFromChains []*params.Network,
	selectedTohains []*params.Network, balanceMap map[string]*big.Int) (candidates []*PathV2, err error) {
	var (
		testsMode = input.testsMode && input.testParams != nil
		group     = async.NewAtomicGroup(ctx)
		mu        sync.Mutex
	)

	crossChainAmountOptions, err := r.findOptionsForSendingAmount(input, selectedFromChains, balanceMap)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	for networkIdx := range selectedFromChains {
		network := selectedFromChains[networkIdx]

		if !input.SendType.isAvailableFor(network) {
			continue
		}

		var (
			token   *walletToken.Token
			toToken *walletToken.Token
		)

		if testsMode {
			token = input.testParams.tokenFrom
		} else {
			token = input.SendType.FindToken(r.tokenManager, r.collectiblesService, input.AddrFrom, network, input.TokenID)
		}
		if token == nil {
			continue
		}

		if input.SendType == Swap {
			toToken = input.SendType.FindToken(r.tokenManager, r.collectiblesService, common.Address{}, network, input.ToTokenID)
		}

		var fees *SuggestedFees
		if testsMode {
			fees = input.testParams.suggestedFees
		} else {
			fees, err = r.feesManager.SuggestedFees(ctx, network.ChainID)
			if err != nil {
				continue
			}
		}

		group.Add(func(c context.Context) error {
			for _, amountOption := range crossChainAmountOptions[network.ChainID] {
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

					for _, dest := range selectedTohains {

						if !input.SendType.isAvailableFor(network) {
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
							AmountIn:  amountOption.amount,
							AmountOut: input.AmountOut.ToInt(),

							Username:  input.Username,
							PublicKey: input.PublicKey,
							PackID:    input.PackID.ToInt(),
						}
						if input.testsMode {
							processorInputParams.TestsMode = input.testsMode
							processorInputParams.TestEstimationMap = input.testParams.estimationMap
							processorInputParams.TestBonderFeeMap = input.testParams.bonderFeeMap
							processorInputParams.TestApprovalGasEstimation = input.testParams.approvalGasEstimation
							processorInputParams.TestApprovalL1Fee = input.testParams.approvalL1Fee
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
						approvalRequired, approvalAmountRequired, approvalGasLimit, l1ApprovalFee, err := r.requireApproval(ctx, input.SendType, &approvalContractAddress, processorInputParams)
						if err != nil {
							continue
						}

						// TODO: keep l1 fees at 0 until we have the correct algorithm, as we do base fee x 2 that should cover the l1 fees
						var l1FeeWei uint64 = 0
						// if input.SendType.needL1Fee() {
						// 	txInputData, err := pProcessor.PackTxInputData(processorInputParams)
						// 	if err != nil {
						// 		continue
						// 	}

						// 	l1FeeWei, _ = r.feesManager.GetL1Fee(ctx, network.ChainID, txInputData)
						// }

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
							ToToken:        toToken,
							AmountIn:       (*hexutil.Big)(amountOption.amount),
							AmountInLocked: amountOption.locked,
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
			}
			return nil
		})
	}

	group.Wait()
	return candidates, nil
}

func (r *Router) checkBalancesForTheBestRoute(ctx context.Context, bestRoute []*PathV2, input *RouteInputParams, balanceMap map[string]*big.Int) (err error) {
	// check the best route for the required balances
	for _, path := range bestRoute {
		if path.requiredTokenBalance != nil && path.requiredTokenBalance.Cmp(pathprocessor.ZeroBigIntValue) > 0 {
			if tokenBalance, ok := balanceMap[makeBalanceKey(path.FromChain.ChainID, path.FromToken.Symbol)]; ok {
				if tokenBalance.Cmp(path.requiredTokenBalance) == -1 {
					return ErrNotEnoughTokenBalance
				}
			} else {
				return ErrTokenNotFound
			}
		}

		if nativeBalance, ok := balanceMap[makeBalanceKey(path.FromChain.ChainID, pathprocessor.EthSymbol)]; ok {
			if nativeBalance.Cmp(path.requiredNativeBalance) == -1 {
				return ErrNotEnoughNativeBalance
			}
		} else {
			return ErrNativeTokenNotFound
		}
	}

	return nil
}

func removeBestRouteFromAllRouters(allRoutes [][]*PathV2, best []*PathV2) [][]*PathV2 {
	for i, route := range allRoutes {
		routeFound := true
		for _, p := range route {
			found := false
			for _, b := range best {
				if p.ProcessorName == b.ProcessorName &&
					(p.FromChain == nil && b.FromChain == nil || p.FromChain.ChainID == b.FromChain.ChainID) &&
					(p.ToChain == nil && b.ToChain == nil || p.ToChain.ChainID == b.ToChain.ChainID) &&
					(p.FromToken == nil && b.FromToken == nil || p.FromToken.Symbol == b.FromToken.Symbol) {
					found = true
					break
				}
			}
			if !found {
				routeFound = false
				break
			}
		}
		if routeFound {
			return append(allRoutes[:i], allRoutes[i+1:]...)
		}
	}

	return nil
}

func (r *Router) resolveRoutes(ctx context.Context, input *RouteInputParams, candidates []*PathV2, balanceMap map[string]*big.Int) (suggestedRoutes *SuggestedRoutesV2, err error) {
	var prices map[string]float64
	if input.testsMode {
		prices = input.testParams.tokenPrices
	} else {
		prices, err = input.SendType.FetchPrices(r.marketManager, input.TokenID)
		if err != nil {
			return nil, errors.CreateErrorResponseFromError(err)
		}
	}

	tokenPrice := prices[input.TokenID]
	nativeChainTokenPrice := prices[pathprocessor.EthSymbol]

	var allRoutes [][]*PathV2
	suggestedRoutes, allRoutes = newSuggestedRoutesV2(input.Uuid, input.AmountIn.ToInt(), candidates, input.FromLockedAmount, tokenPrice, nativeChainTokenPrice)

	for len(allRoutes) > 0 {
		best := findBestV2(allRoutes, tokenPrice, nativeChainTokenPrice)

		err := r.checkBalancesForTheBestRoute(ctx, best, input, balanceMap)
		if err != nil {
			// If it's about transfer or bridge and there is more routes, but on the best (cheapest) one there is not enugh balance
			// we shold check other routes even though there are not the cheapest ones
			if (input.SendType == Transfer ||
				input.SendType == Bridge) &&
				len(allRoutes) > 1 {
				allRoutes = removeBestRouteFromAllRouters(allRoutes, best)
				continue
			} else {
				return suggestedRoutes, errors.CreateErrorResponseFromError(err)
			}
		}

		if len(best) > 0 {
			sort.Slice(best, func(i, j int) bool {
				return best[i].AmountInLocked
			})
		}
		suggestedRoutes.Best = best
		break
	}

	return suggestedRoutes, nil
}
