package router

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	routerTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

type amountOption struct {
	amount       *big.Int
	locked       bool
	subtractFees bool
}

func makeBalanceKey(chainID uint64, symbol string) string {
	return fmt.Sprintf("%d-%s", chainID, symbol)
}

type ProcessorError struct {
	ProcessorName string
	Error         error
}

type SuggestedRoutes struct {
	Uuid          string
	Best          routes.Route
	Candidates    routes.Route
	UpdatedPrices map[string]float64
}

type Router struct {
	rpcClient           *rpc.Client
	tokenManager        *token.Manager
	marketManager       *market.Manager
	collectiblesService *collectibles.Service
	collectiblesManager *collectibles.Manager
	ensService          *ens.Service
	stickersService     *stickers.Service
	feesManager         *fees.FeeManager
	pathProcessors      map[string]pathprocessor.PathProcessor
	scheduler           *async.Scheduler

	activeBalanceMap sync.Map // map[string]*big.Int

	activeRoutesMutex sync.Mutex
	activeRoutes      *SuggestedRoutes

	routeCanceledMutex sync.Mutex
	routeCanceled      bool

	lastInputParamsMutex sync.Mutex
	lastInputParams      *requests.RouteInputParams

	clientsForUpdatesPerChains sync.Map
}

func NewRouter(rpcClient *rpc.Client, transactor *transactions.Transactor, tokenManager *token.Manager, marketManager *market.Manager,
	collectibles *collectibles.Service, collectiblesManager *collectibles.Manager, ensService *ens.Service, stickersService *stickers.Service) *Router {
	processors := make(map[string]pathprocessor.PathProcessor)

	return &Router{
		rpcClient:           rpcClient,
		tokenManager:        tokenManager,
		marketManager:       marketManager,
		collectiblesService: collectibles,
		collectiblesManager: collectiblesManager,
		ensService:          ensService,
		stickersService:     stickersService,
		feesManager: &fees.FeeManager{
			RPCClient: rpcClient,
		},
		pathProcessors: processors,
		scheduler:      async.NewScheduler(),
	}
}

func (r *Router) AddPathProcessor(processor pathprocessor.PathProcessor) {
	r.pathProcessors[processor.Name()] = processor
}

func (r *Router) Stop() {
	r.scheduler.Stop()
}

func (r *Router) GetFeesManager() *fees.FeeManager {
	return r.feesManager
}

func (r *Router) GetPathProcessors() map[string]pathprocessor.PathProcessor {
	return r.pathProcessors
}

func (r *Router) GetBestRouteAndAssociatedInputParams() (routes.Route, requests.RouteInputParams) {
	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()
	if r.activeRoutes == nil {
		return nil, requests.RouteInputParams{}
	}

	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()
	ip := *r.lastInputParams

	return r.activeRoutes.Best.Copy(), ip
}

func (r *Router) SetTestBalanceMap(balanceMap map[string]*big.Int) {
	for k, v := range balanceMap {
		r.activeBalanceMap.Store(k, v)
	}
}

func newSuggestedRoutes(
	input *requests.RouteInputParams,
	candidates routes.Route,
	updatedPrices map[string]float64,
) (*SuggestedRoutes, []routes.Route) {
	suggestedRoutes := &SuggestedRoutes{
		Uuid:          input.Uuid,
		Candidates:    candidates,
		UpdatedPrices: updatedPrices,
	}
	if len(candidates) == 0 {
		return suggestedRoutes, nil
	}

	node := &routes.Node{
		Path:     nil,
		Children: routes.BuildGraph(input.AmountIn.ToInt(), candidates, 0, []uint64{}),
	}
	allRoutes := node.BuildAllRoutes()
	allRoutes = filterRoutes(allRoutes, input.AmountIn.ToInt(), input.FromLockedAmount)

	if len(allRoutes) > 0 {
		sort.Slice(allRoutes, func(i, j int) bool {
			iRoute := getRoutePriority(allRoutes[i])
			jRoute := getRoutePriority(allRoutes[j])
			return iRoute <= jRoute
		})
	}

	return suggestedRoutes, allRoutes
}

func sendRouterResult(uuid string, result interface{}, err error) {
	routesResponse := responses.RouterSuggestedRoutes{
		Uuid: uuid,
	}

	if err != nil {
		errorResponse := errors.CreateErrorResponseFromError(err)
		routesResponse.ErrorResponse = errorResponse.(*errors.ErrorResponse)
	}

	if suggestedRoutes, ok := result.(*SuggestedRoutes); ok && suggestedRoutes != nil {
		routesResponse.Best = suggestedRoutes.Best
		routesResponse.Candidates = suggestedRoutes.Candidates
		routesResponse.UpdatedPrices = suggestedRoutes.UpdatedPrices
	}

	signal.SendWalletEvent(signal.SuggestedRoutes, routesResponse)
}

func (r *Router) SuggestedRoutesAsync(input *requests.RouteInputParams) {
	r.scheduler.Enqueue(routerTask, func(ctx context.Context) (interface{}, error) {
		return r.SuggestedRoutes(ctx, input)
	}, func(result interface{}, taskType async.TaskType, err error) {
		sendRouterResult(input.Uuid, result, err)
	})
}

func (r *Router) clearActiveRoute() {
	r.activeRoutesMutex.Lock()
	r.activeRoutes = nil
	r.activeRoutesMutex.Unlock()
}

func (r *Router) markRouteCanceled(value bool) {
	r.routeCanceledMutex.Lock()
	r.routeCanceled = value
	r.routeCanceledMutex.Unlock()
}

func (r *Router) abortUpdates() {
	r.markRouteCanceled(true)
	r.unsubscribeFeesUpdateAccrossAllChains()
}

func (r *Router) StopSuggestedRoutesAsyncCalculation() {
	r.abortUpdates()
	r.scheduler.Stop()
}

func (r *Router) StopSuggestedRoutesCalculation() {
	r.abortUpdates()
}

func (r *Router) SuggestedRoutes(ctx context.Context, input *requests.RouteInputParams) (suggestedRoutes *SuggestedRoutes, err error) {
	r.clearActiveRoute()
	r.abortUpdates()
	r.markRouteCanceled(false)

	// clear all processors
	for _, processor := range r.pathProcessors {
		if clearable, ok := processor.(pathprocessor.PathProcessorClearable); ok {
			clearable.Clear()
		}
	}

	r.lastInputParamsMutex.Lock()
	r.lastInputParams = input
	r.lastInputParamsMutex.Unlock()

	defer func() {
		r.activeRoutesMutex.Lock()
		r.activeRoutes = suggestedRoutes
		r.activeRoutesMutex.Unlock()
		r.routeCanceledMutex.Lock()
		if suggestedRoutes != nil && err == nil && !r.routeCanceled {
			// subscribe for updates
			for _, path := range suggestedRoutes.Best {
				err = r.subscribeForUdates(path.FromChain.ChainID)
			}
		}
		r.routeCanceledMutex.Unlock()
	}()

	testnetMode, err := r.rpcClient.NetworkManager.GetTestNetworksEnabled()
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	input.TestnetMode = testnetMode

	err = input.Validate()
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	selectedFromChains, selectedToChains, err := r.getSelectedChains(input)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	err = r.prepareBalanceMapForTokenOnChains(ctx, input, selectedFromChains)
	// return only if there are no balances, otherwise try to resolve the candidates for chains we know the balances for
	noBalanceOnAnyChain := true
	r.activeBalanceMap.Range(func(key, value interface{}) bool {
		if value.(*big.Int).Cmp(walletCommon.ZeroBigIntValue()) > 0 {
			noBalanceOnAnyChain = false
			return false
		}
		return true
	})
	if noBalanceOnAnyChain {
		if err != nil {
			return nil, errors.CreateErrorResponseFromError(err)
		}
		return nil, ErrNoPositiveBalance
	}

	candidates, processorErrors, err := r.resolveCandidates(ctx, input, selectedFromChains, selectedToChains)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	suggestedRoutes, err = r.resolveRoutes(ctx, input, candidates)

	if err == nil && (suggestedRoutes == nil || len(suggestedRoutes.Best) == 0) {
		// No best route found, but no error given.
		if len(processorErrors) > 0 {
			// Return one of the path processor errors if present.
			// Give precedence to the custom error message.
			for _, processorError := range processorErrors {
				if processorError.Error != nil && pathprocessor.IsCustomError(processorError.Error) {
					err = processorError.Error
					break
				}
			}
			if err == nil {
				err = errors.CreateErrorResponseFromError(processorErrors[0].Error)
			}
		} else {
			err = ErrNoBestRouteFound
		}
	}

	mapError := func(err error) error {
		if err == nil {
			return nil
		}
		pattern := "insufficient funds for gas * price + value: address "
		addressIndex := strings.Index(errors.DetailsFromError(err), pattern)
		if addressIndex != -1 {
			addressIndex += len(pattern) + walletCommon.HexAddressLength
			return errors.CreateErrorResponseFromError(&errors.ErrorResponse{
				Code:    errors.ErrorCodeFromError(err),
				Details: errors.DetailsFromError(err)[:addressIndex],
			})
		}
		return err
	}
	// map some errors to more user-friendly messages
	return suggestedRoutes, mapError(err)
}

// prepareBalanceMapForTokenOnChains prepares the balance map for passed address, where the key is in format "chainID-tokenSymbol" and
// value is the balance of the token. Native token (EHT) is always added to the balance map.
func (r *Router) prepareBalanceMapForTokenOnChains(ctx context.Context, input *requests.RouteInputParams, selectedFromChains []*params.Network) (err error) {
	// clear the active balance map
	r.activeBalanceMap = sync.Map{}

	if input.TestsMode {
		for k, v := range input.TestParams.BalanceMap {
			r.activeBalanceMap.Store(k, v)
		}
		return nil
	}

	chainError := func(chainId uint64, token string, intErr error) {
		if err == nil {
			err = fmt.Errorf("chain %d, token %s: %w", chainId, token, intErr)
		} else {
			err = fmt.Errorf("%s; chain %d, token %s: %w", err.Error(), chainId, token, intErr)
		}
	}

	for _, chain := range selectedFromChains {
		// check token existence
		token := input.SendType.FindToken(r.tokenManager, r.collectiblesService, input.AddrFrom, chain, input.TokenID)
		if token == nil {
			chainError(chain.ChainID, input.TokenID, ErrTokenNotFound)
			continue
		}
		// check native token existence
		nativeToken := r.tokenManager.FindToken(chain, chain.NativeCurrencySymbol)
		if nativeToken == nil {
			chainError(chain.ChainID, chain.NativeCurrencySymbol, ErrNativeTokenNotFound)
			continue
		}

		// add token balance for the chain
		var tokenBalance *big.Int
		if input.SendType == sendtype.ERC721Transfer {
			tokenBalance = big.NewInt(1)
		} else if input.SendType == sendtype.ERC1155Transfer {
			tokenBalance, err = r.getERC1155Balance(ctx, chain, token, input.AddrFrom)
			if err != nil {
				chainError(chain.ChainID, token.Symbol, errors.CreateErrorResponseFromError(err))
			}
		} else {
			tokenBalance, err = r.getBalance(ctx, chain.ChainID, token, input.AddrFrom)
			if err != nil {
				chainError(chain.ChainID, token.Symbol, errors.CreateErrorResponseFromError(err))
			}
		}
		// add only if balance is not nil
		if tokenBalance != nil {
			r.activeBalanceMap.Store(makeBalanceKey(chain.ChainID, token.Symbol), tokenBalance)
		}

		if token.IsNative() {
			continue
		}

		// add native token balance for the chain
		nativeBalance, err := r.getBalance(ctx, chain.ChainID, nativeToken, input.AddrFrom)
		if err != nil {
			chainError(chain.ChainID, token.Symbol, errors.CreateErrorResponseFromError(err))
		}
		// add only if balance is not nil
		if nativeBalance != nil {
			r.activeBalanceMap.Store(makeBalanceKey(chain.ChainID, nativeToken.Symbol), nativeBalance)
		}
	}

	return
}

func (r *Router) getSelectedUnlockedChains(input *requests.RouteInputParams, processingChain *params.Network, selectedFromChains []*params.Network) []*params.Network {
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

func (r *Router) getOptionsForAmoutToSplitAccrossChainsForProcessingChain(input *requests.RouteInputParams, amountToSplit *big.Int, processingChain *params.Network,
	selectedFromChains []*params.Network) map[uint64][]amountOption {
	selectedButNotLockedChains := r.getSelectedUnlockedChains(input, processingChain, selectedFromChains)

	crossChainAmountOptions := make(map[uint64][]amountOption)
	for _, chain := range selectedButNotLockedChains {
		var (
			ok           bool
			tokenBalance *big.Int
		)

		value, ok := r.activeBalanceMap.Load(makeBalanceKey(chain.ChainID, input.TokenID))
		if !ok {
			continue
		}
		tokenBalance, ok = value.(*big.Int)
		if !ok {
			continue
		}

		if tokenBalance.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
			if tokenBalance.Cmp(amountToSplit) <= 0 {
				crossChainAmountOptions[chain.ChainID] = append(crossChainAmountOptions[chain.ChainID], amountOption{
					amount:       tokenBalance,
					locked:       false,
					subtractFees: true, // for chains where we're taking the full balance, we want to subtract the fees
				})
				amountToSplit = new(big.Int).Sub(amountToSplit, tokenBalance)
			} else if amountToSplit.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
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

func (r *Router) getCrossChainsOptionsForSendingAmount(input *requests.RouteInputParams, selectedFromChains []*params.Network) map[uint64][]amountOption {
	// All we do in this block we're free to do, because of the validateInputData function which checks if the locked amount
	// was properly set and if there is something unexpected it will return an error and we will not reach this point
	finalCrossChainAmountOptions := make(map[uint64][]amountOption) // represents all possible amounts that can be sent from the "from" chain

	for _, selectedFromChain := range selectedFromChains {

		amountLocked := false
		amountToSend := input.AmountIn.ToInt()

		if amountToSend.Cmp(walletCommon.ZeroBigIntValue()) == 0 {
			finalCrossChainAmountOptions[selectedFromChain.ChainID] = append(finalCrossChainAmountOptions[selectedFromChain.ChainID], amountOption{
				amount: amountToSend,
				locked: false,
			})
			continue
		}

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

		if amountToSend.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
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
			if input.SendType == sendtype.Transfer && len(selectedFromChains) > 1 {
				// All we do in this block we're free to do, because of the validateInputData function which checks if the locked amount
				// was properly set and if there is something unexpected it will return an error and we will not reach this point
				amountToSplitAccrossChains := new(big.Int).Set(amountToSend)

				crossChainAmountOptions := r.getOptionsForAmoutToSplitAccrossChainsForProcessingChain(input, amountToSend, selectedFromChain, selectedFromChains)

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

func (r *Router) findOptionsForSendingAmount(input *requests.RouteInputParams, selectedFromChains []*params.Network) (map[uint64][]amountOption, error) {

	crossChainAmountOptions := r.getCrossChainsOptionsForSendingAmount(input, selectedFromChains)

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

func (r *Router) getSelectedChains(input *requests.RouteInputParams) (selectedFromChains []*params.Network, selectedToChains []*params.Network, err error) {
	var networks []*params.Network
	networks, err = r.rpcClient.NetworkManager.Get(false)
	if err != nil {
		return nil, nil, errors.CreateErrorResponseFromError(err)
	}

	for _, network := range networks {
		if network.IsTest != input.TestnetMode {
			continue
		}

		if !walletCommon.ArrayContainsElement(network.ChainID, input.DisabledFromChainIDs) {
			selectedFromChains = append(selectedFromChains, network)
		}

		if !walletCommon.ArrayContainsElement(network.ChainID, input.DisabledToChainIDs) {
			selectedToChains = append(selectedToChains, network)
		}
	}

	return selectedFromChains, selectedToChains, nil
}

func (r *Router) resolveCandidates(ctx context.Context, input *requests.RouteInputParams, selectedFromChains []*params.Network,
	selectedToChains []*params.Network) (candidates routes.Route, processorErrors []*ProcessorError, err error) {
	var (
		testsMode = input.TestsMode && input.TestParams != nil
		group     = async.NewAtomicGroup(ctx)
		mu        sync.Mutex
	)

	crossChainAmountOptions, err := r.findOptionsForSendingAmount(input, selectedFromChains)
	if err != nil {
		return nil, nil, errors.CreateErrorResponseFromError(err)
	}

	appendProcessorErrorFn := func(processorName string, sendType sendtype.SendType, fromChainID uint64, toChainID uint64, amount *big.Int, err error) {
		logutils.ZapLogger().Error("router.resolveCandidates error",
			zap.String("processor", processorName),
			zap.Int("sendType", int(sendType)),
			zap.Uint64("fromChainId", fromChainID),
			zap.Uint64("toChainId", toChainID),
			zap.Stringer("amount", amount),
			zap.Error(err))
		mu.Lock()
		defer mu.Unlock()
		processorErrors = append(processorErrors, &ProcessorError{
			ProcessorName: processorName,
			Error:         err,
		})
	}

	appendPathFn := func(path *routes.Path) {
		mu.Lock()
		defer mu.Unlock()
		candidates = append(candidates, path)
	}

	for networkIdx := range selectedFromChains {
		network := selectedFromChains[networkIdx]

		if !input.SendType.IsAvailableFor(network) {
			continue
		}

		var (
			token   *walletToken.Token
			toToken *walletToken.Token
		)

		if testsMode {
			token = input.TestParams.TokenFrom
		} else {
			token = input.SendType.FindToken(r.tokenManager, r.collectiblesService, input.AddrFrom, network, input.TokenID)
		}
		if token == nil {
			continue
		}

		if input.SendType == sendtype.Swap {
			toToken = input.SendType.FindToken(r.tokenManager, r.collectiblesService, common.Address{}, network, input.ToTokenID)
		}

		var fetchedFees *fees.SuggestedFees
		if testsMode {
			fetchedFees = input.TestParams.SuggestedFees
		} else {
			fetchedFees, err = r.feesManager.SuggestedFees(ctx, network.ChainID)
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
					if !input.SendType.CanUseProcessor(pProcessor) {
						continue
					}

					// if we're doing a single chain operation, we can skip bridge processors
					if walletCommon.IsSingleChainOperation(selectedFromChains, selectedToChains) && pathprocessor.IsProcessorBridge(pProcessor.Name()) {
						continue
					}

					if !input.SendType.ProcessZeroAmountInProcessor(amountOption.amount, input.AmountOut.ToInt(), pProcessor.Name()) {
						continue
					}

					for _, dest := range selectedToChains {

						if !input.SendType.IsAvailableFor(network) {
							continue
						}

						if !input.SendType.IsAvailableBetween(network, dest) {
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
						if input.TestsMode {
							processorInputParams.TestsMode = input.TestsMode
							processorInputParams.TestEstimationMap = input.TestParams.EstimationMap
							processorInputParams.TestBonderFeeMap = input.TestParams.BonderFeeMap
							processorInputParams.TestApprovalGasEstimation = input.TestParams.ApprovalGasEstimation
							processorInputParams.TestApprovalL1Fee = input.TestParams.ApprovalL1Fee
						}

						can, err := pProcessor.AvailableFor(processorInputParams)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}
						if !can {
							continue
						}

						bonderFees, tokenFees, err := pProcessor.CalculateFees(processorInputParams)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}

						gasLimit, err := pProcessor.EstimateGas(processorInputParams)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}

						approvalContractAddress, err := pProcessor.GetContractAddress(processorInputParams)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}
						approvalRequired, approvalAmountRequired, err := r.requireApproval(ctx, input.SendType, &approvalContractAddress, processorInputParams)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}

						var approvalGasLimit uint64
						if approvalRequired {
							if processorInputParams.TestsMode {
								approvalGasLimit = processorInputParams.TestApprovalGasEstimation
							} else {
								approvalGasLimit, err = r.estimateGasForApproval(processorInputParams, &approvalContractAddress)
								if err != nil {
									appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
									continue
								}
							}
						}

						amountOut, err := pProcessor.CalculateAmountOut(processorInputParams)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}

						maxFeesPerGas := fetchedFees.FeeFor(input.GasFeeMode)

						estimatedTime := r.feesManager.TransactionEstimatedTime(ctx, network.ChainID, maxFeesPerGas)
						if approvalRequired && estimatedTime < fees.MoreThanFiveMinutes {
							estimatedTime += 1
						}

						path := &routes.Path{
							ProcessorName:  pProcessor.Name(),
							FromChain:      network,
							ToChain:        dest,
							FromToken:      token,
							ToToken:        toToken,
							AmountIn:       (*hexutil.Big)(amountOption.amount),
							AmountInLocked: amountOption.locked,
							AmountOut:      (*hexutil.Big)(amountOut),

							// set params that we don't want to be recalculated with every new block creation
							TxGasAmount:  gasLimit,
							TxBonderFees: (*hexutil.Big)(bonderFees),
							TxTokenFees:  (*hexutil.Big)(tokenFees),

							ApprovalRequired:        approvalRequired,
							ApprovalAmountRequired:  (*hexutil.Big)(approvalAmountRequired),
							ApprovalContractAddress: &approvalContractAddress,
							ApprovalGasAmount:       approvalGasLimit,

							EstimatedTime: estimatedTime,

							SubtractFees: amountOption.subtractFees,
						}

						err = r.cacluateFees(ctx, path, fetchedFees, processorInputParams.TestsMode, processorInputParams.TestApprovalL1Fee)
						if err != nil {
							appendProcessorErrorFn(pProcessor.Name(), input.SendType, processorInputParams.FromChain.ChainID, processorInputParams.ToChain.ChainID, processorInputParams.AmountIn, err)
							continue
						}

						appendPathFn(path)
					}
				}
			}
			return nil
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		iChain := getChainPriority(candidates[i].FromChain.ChainID)
		jChain := getChainPriority(candidates[j].FromChain.ChainID)
		return iChain <= jChain
	})

	group.Wait()
	return candidates, processorErrors, nil
}

func (r *Router) checkBalancesForTheBestRoute(ctx context.Context, bestRoute routes.Route) (hasPositiveBalance bool, err error) {
	// make a copy of the active balance map
	balanceMapCopy := make(map[string]*big.Int)
	r.activeBalanceMap.Range(func(k, v interface{}) bool {
		balanceMapCopy[k.(string)] = new(big.Int).Set(v.(*big.Int))
		return true
	})
	if balanceMapCopy == nil {
		return false, ErrCannotCheckBalance
	}

	// check the best route for the required balances
	for _, path := range bestRoute {
		tokenKey := makeBalanceKey(path.FromChain.ChainID, path.FromToken.Symbol)
		if tokenBalance, ok := balanceMapCopy[tokenKey]; ok {
			if tokenBalance.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
				hasPositiveBalance = true
			}
		}

		if path.ProcessorName == pathprocessor.ProcessorBridgeHopName {
			if path.TxBonderFees.ToInt().Cmp(path.AmountOut.ToInt()) > 0 {
				return hasPositiveBalance, ErrLowAmountInForHopBridge
			}
		}

		if path.RequiredTokenBalance != nil && path.RequiredTokenBalance.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
			if tokenBalance, ok := balanceMapCopy[tokenKey]; ok {
				if tokenBalance.Cmp(path.RequiredTokenBalance) == -1 {
					err := &errors.ErrorResponse{
						Code:    ErrNotEnoughTokenBalance.Code,
						Details: fmt.Sprintf(ErrNotEnoughTokenBalance.Details, path.FromToken.Symbol, path.FromChain.ChainID),
					}
					return hasPositiveBalance, err
				}
				balanceMapCopy[tokenKey].Sub(tokenBalance, path.RequiredTokenBalance)
			} else {
				return hasPositiveBalance, ErrTokenNotFound
			}
		}

		ethKey := makeBalanceKey(path.FromChain.ChainID, pathprocessor.EthSymbol)
		if nativeBalance, ok := balanceMapCopy[ethKey]; ok {
			if nativeBalance.Cmp(path.RequiredNativeBalance) == -1 {
				err := &errors.ErrorResponse{
					Code:    ErrNotEnoughNativeBalance.Code,
					Details: fmt.Sprintf(ErrNotEnoughNativeBalance.Details, pathprocessor.EthSymbol, path.FromChain.ChainID),
				}
				return hasPositiveBalance, err
			}
			balanceMapCopy[ethKey].Sub(nativeBalance, path.RequiredNativeBalance)
		} else {
			return hasPositiveBalance, ErrNativeTokenNotFound
		}
	}

	return hasPositiveBalance, nil
}

func (r *Router) resolveRoutes(ctx context.Context, input *requests.RouteInputParams, candidates routes.Route) (suggestedRoutes *SuggestedRoutes, err error) {
	var prices map[string]float64
	if input.TestsMode {
		prices = input.TestParams.TokenPrices
	} else {
		prices, err = input.SendType.FetchPrices(r.marketManager, []string{input.TokenID, input.ToTokenID})
		if err != nil {
			return nil, errors.CreateErrorResponseFromError(err)
		}
	}

	tokenPrice := prices[input.TokenID]
	nativeTokenPrice := prices[pathprocessor.EthSymbol]

	var allRoutes []routes.Route
	suggestedRoutes, allRoutes = newSuggestedRoutes(input, candidates, prices)

	defer func() {
		if suggestedRoutes.Best != nil && len(suggestedRoutes.Best) > 0 {
			sort.Slice(suggestedRoutes.Best, func(i, j int) bool {
				iChain := getChainPriority(suggestedRoutes.Best[i].FromChain.ChainID)
				jChain := getChainPriority(suggestedRoutes.Best[j].FromChain.ChainID)
				return iChain <= jChain
			})
		}
	}()

	var (
		bestRoute                        routes.Route
		lastBestRouteWithPositiveBalance routes.Route
		lastBestRouteErr                 error
	)

	for len(allRoutes) > 0 {
		bestRoute = routes.FindBestRoute(allRoutes, tokenPrice, nativeTokenPrice)
		var hasPositiveBalance bool
		hasPositiveBalance, err = r.checkBalancesForTheBestRoute(ctx, bestRoute)

		if err != nil {
			// If it's about transfer or bridge and there is more routes, but on the best (cheapest) one there is not enugh balance
			// we shold check other routes even though there are not the cheapest ones
			if input.SendType == sendtype.Transfer ||
				input.SendType == sendtype.Bridge {
				if hasPositiveBalance {
					lastBestRouteWithPositiveBalance = bestRoute
					lastBestRouteErr = err
				}

				if len(allRoutes) > 1 {
					allRoutes = removeBestRouteFromAllRouters(allRoutes, bestRoute)
					continue
				} else {
					break
				}
			}
		}

		break
	}

	// if none of the routes have positive balance, we should return the last best route with positive balance
	if err != nil && lastBestRouteWithPositiveBalance != nil {
		bestRoute = lastBestRouteWithPositiveBalance
		err = lastBestRouteErr
	}

	if len(bestRoute) > 0 {
		// At this point we have to do the final check and update the amountIn (subtracting fees) if complete balance is going to be sent for native token (ETH)
		for _, path := range bestRoute {
			if path.SubtractFees && path.FromToken.IsNative() {
				path.AmountIn.ToInt().Sub(path.AmountIn.ToInt(), path.TxFee.ToInt())
				if path.TxL1Fee.ToInt().Cmp(walletCommon.ZeroBigIntValue()) > 0 {
					path.AmountIn.ToInt().Sub(path.AmountIn.ToInt(), path.TxL1Fee.ToInt())
				}
				if path.ApprovalRequired {
					path.AmountIn.ToInt().Sub(path.AmountIn.ToInt(), path.ApprovalFee.ToInt())
					if path.ApprovalL1Fee.ToInt().Cmp(walletCommon.ZeroBigIntValue()) > 0 {
						path.AmountIn.ToInt().Sub(path.AmountIn.ToInt(), path.ApprovalL1Fee.ToInt())
					}
				}
			}
		}
	}
	suggestedRoutes.Best = bestRoute

	return suggestedRoutes, err
}
