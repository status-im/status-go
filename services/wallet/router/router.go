package router

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	routerTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

func makeBalanceKey(chainID uint64, symbol string) string {
	return fmt.Sprintf("%d-%s", chainID, symbol)
}

type ProcessorError struct {
	ProcessorName string
	Error         error
}

type SuggestedRoutes struct {
	Uuid          string
	Route         routes.Route
	UpdatedPrices map[string]float64
}

type Router struct {
	rpcClient           *rpc.Client
	transactor          *transactions.Transactor
	tokenManager        *token.Manager
	marketManager       *market.Manager
	collectiblesService *collectibles.Service
	collectiblesManager *collectibles.Manager
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
	collectibles *collectibles.Service, collectiblesManager *collectibles.Manager) *Router {
	processors := make(map[string]pathprocessor.PathProcessor)

	return &Router{
		rpcClient:           rpcClient,
		transactor:          transactor,
		tokenManager:        tokenManager,
		marketManager:       marketManager,
		collectiblesService: collectibles,
		collectiblesManager: collectiblesManager,
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

	return r.activeRoutes.Route.Copy(), ip
}

func (r *Router) SetTestBalanceMap(balanceMap map[string]*big.Int) {
	for k, v := range balanceMap {
		r.activeBalanceMap.Store(k, v)
	}
}

func (r *Router) setCustomTxDetails(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, pathTxCustomParams *requests.PathTxCustomParams) error {
	if pathTxIdentity == nil {
		return ErrTxIdentityNotProvided
	}
	err := pathTxIdentity.Validate()
	if err != nil {
		return err
	}
	if pathTxCustomParams == nil {
		return ErrTxCustomParamsNotProvided
	}
	err = pathTxCustomParams.Validate()
	if err != nil {
		return err
	}

	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()
	if r.activeRoutes == nil || len(r.activeRoutes.Route) == 0 {
		return ErrCannotCustomizeIfNoRoute
	}

	var addrFrom common.Address
	r.lastInputParamsMutex.Lock()
	addrFrom = r.lastInputParams.AddrFrom
	r.lastInputParamsMutex.Unlock()

	fetchedFees, noBaseFee, noPriorityFee, err := r.feesManager.SuggestedFees(ctx, pathTxIdentity.ChainID, addrFrom)
	if err != nil {
		return err
	}

	for _, path := range r.activeRoutes.Route {
		if path.PathIdentity() != pathTxIdentity.PathIdentity() {
			continue
		}

		// update the custom params
		r.lastInputParamsMutex.Lock()
		if r.lastInputParams.PathTxCustomParams == nil {
			r.lastInputParams.PathTxCustomParams = make(map[string]*requests.PathTxCustomParams)
		}
		r.lastInputParams.PathTxCustomParams[pathTxIdentity.TxIdentityKey()] = pathTxCustomParams
		r.lastInputParamsMutex.Unlock()

		// update the path details
		usedNonces := make(map[uint64]uint64)
		err = r.evaluateAndUpdatePathDetails(ctx, path, fetchedFees, usedNonces, noBaseFee, noPriorityFee, false, 0)
		if err != nil {
			return err
		}
		err = r.checkBalancesForTheBestRoute(r.activeRoutes.Route)
		// inform the client about the changes
		sendRouterResult(pathTxIdentity.RouterInputParamsUuid, r.activeRoutes, err)

		return nil
	}

	return ErrCannotFindPathForProvidedIdentity
}

func (r *Router) SetFeeMode(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, feeMode fees.GasFeeMode) error {
	if feeMode == fees.GasFeeCustom {
		return ErrCustomFeeModeCannotBeSetThisWay
	}

	return r.setCustomTxDetails(ctx, pathTxIdentity, &requests.PathTxCustomParams{GasFeeMode: feeMode})
}

func (r *Router) SetCustomTxDetails(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, pathTxCustomParams *requests.PathTxCustomParams) error {
	if pathTxCustomParams != nil && pathTxCustomParams.GasFeeMode != fees.GasFeeCustom {
		return ErrOnlyCustomFeeModeCanBeSetThisWay
	}
	return r.setCustomTxDetails(ctx, pathTxIdentity, pathTxCustomParams)
}

// ReevaluateRouterPath reevaluates the tx-fields from the router path that matches the provided pathTxIdentity and sends signal.SuggestedRoutes.
func (r *Router) ReevaluateRouterPath(ctx context.Context, pathTxIdentity *requests.PathTxIdentity) error {
	if pathTxIdentity == nil {
		return ErrTxIdentityNotProvided
	}
	err := pathTxIdentity.Validate()
	if err != nil {
		return err
	}

	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()
	if r.activeRoutes == nil || len(r.activeRoutes.Route) == 0 {
		return ErrNoBestRouteFound
	}

	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()

	fetchedFees, noBaseFee, noPriorityFee, err := r.feesManager.SuggestedFees(ctx, pathTxIdentity.ChainID, r.lastInputParams.AddrFrom)
	if err != nil {
		return err
	}

	for _, path := range r.activeRoutes.Route {
		if path.PathIdentity() != pathTxIdentity.PathIdentity() {
			continue
		}

		for _, pProcessor := range r.pathProcessors {
			if pProcessor.Name() != path.ProcessorName {
				continue
			}

			processorInputParams, err := r.CreateProcessorInputParams(r.lastInputParams, path.FromChain, path.ToChain, path.FromToken, path.ToToken, 0)
			if err != nil {
				return err
			}

			txPackedData, err := pProcessor.PackTxInputData(processorInputParams)
			if err != nil {
				return err
			}

			gasLimit, err := pProcessor.EstimateGas(processorInputParams, txPackedData)
			if err != nil {
				return err
			}

			path.SuggestedTxGasAmount = gasLimit
			path.TxGasAmount = gasLimit
			path.TxPackedData = txPackedData

			// update the path details
			usedNonces := make(map[uint64]uint64)
			if path.ApprovalRequired {
				usedNonces[path.FromChain.ChainID] = uint64(*path.ApprovalTxNonce) - 1
			} else {
				usedNonces[path.FromChain.ChainID] = uint64(*path.TxNonce - 1)
			}
			err = r.evaluateAndUpdatePathDetails(ctx, path, fetchedFees, usedNonces, noBaseFee, noPriorityFee, false, 0)
			if err != nil {
				return err
			}

			// inform the client about the changes
			sendRouterResult(pathTxIdentity.RouterInputParamsUuid, r.activeRoutes, err)

			return nil
		}

		return ErrCannotFindPathProcessorForProvidedIdentity
	}

	return ErrCannotFindPathForProvidedIdentity
}

func sendRouterResult(uuid string, result interface{}, err error) {
	routesResponse := responses.RouterSuggestedRoutes{
		Uuid: uuid,
	}

	emptySignal := true
	if err != nil {
		errorResponse := errors.CreateErrorResponseFromError(err)
		routesResponse.ErrorResponse = errorResponse.(*errors.ErrorResponse)
		emptySignal = false
	}

	if suggestedRoutes, ok := result.(*SuggestedRoutes); ok && suggestedRoutes != nil {
		routesResponse.Route = suggestedRoutes.Route
		emptySignal = false
	}

	if emptySignal {
		return
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
			for _, path := range suggestedRoutes.Route {
				err = r.subscribeForUdates(path.FromChain.ChainID, input.AddrFrom)
			}
		}
		r.routeCanceledMutex.Unlock()
	}()

	testnetMode, err := r.rpcClient.GetNetworkManager().GetTestNetworksEnabled()
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	input.TestnetMode = testnetMode

	err = input.Validate()
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	fromChain := r.rpcClient.GetNetworkManager().Find(input.FromChainID)
	if fromChain == nil {
		return nil, errors.CreateErrorResponseFromError(fmt.Errorf("from chain %d not found", input.FromChainID))
	}

	toChain := r.rpcClient.GetNetworkManager().Find(input.ToChainID)
	if toChain == nil {
		return nil, errors.CreateErrorResponseFromError(fmt.Errorf("to chain %d not found", input.ToChainID))
	}

	err = r.prepareBalanceMapForTokenOnChain(ctx, input, fromChain)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	// return only if there are no balances, otherwise try to resolve the candidates for chains we know the balances for
	// an exception is Status chain, which is gasless
	if input.FromChainID != walletCommon.StatusNetworkSepolia {
		noBalanceOnAnyChain := true
		r.activeBalanceMap.Range(func(key, value interface{}) bool {
			if value.(*big.Int).Cmp(walletCommon.ZeroBigIntValue()) > 0 {
				noBalanceOnAnyChain = false
				return false
			}
			return true
		})
		if noBalanceOnAnyChain {
			return nil, ErrNoPositiveBalance
		}
	}

	route, processorErrors, err := r.resolveRoute(ctx, input, fromChain, toChain)
	if err != nil {
		return nil, errors.CreateErrorResponseFromError(err)
	}

	err = r.checkBalancesForRouteAndAdjustAmountIn(route)
	if err != nil {
		// don't return here, cause we have to return the route anywaye, even there are balance issues
		logutils.ZapLogger().Error("router.checkBalancesForRouteAndAdjustAmountIn error", zap.Error(err))
	}

	if err == nil && len(route) == 0 {
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

	suggestedRoutes = r.makeSuggestedRoute(input, route)

	// map some errors to more user-friendly messages
	return suggestedRoutes, mapError(err)
}

// prepareBalanceMapForTokenOnChain prepares the balance map for passed address, where the key is in format "chainID-tokenSymbol" and
// value is the balance of the token. Native token (EHT) is always added to the balance map.
func (r *Router) prepareBalanceMapForTokenOnChain(ctx context.Context, input *requests.RouteInputParams, fromChain *params.Network) (err error) {
	// clear the active balance map
	r.activeBalanceMap = sync.Map{}

	if input.TestsMode {
		for k, v := range input.TestParams.BalanceMap {
			r.activeBalanceMap.Store(k, v)
		}
		return
	}

	// check token existence
	token := findToken(input.SendType, r.tokenManager, r.collectiblesService, input.AddrFrom, fromChain, input.TokenID)
	if token == nil {
		err = errors.CreateErrorResponseFromError(ErrTokenNotFound)
		return
	}
	// check native token existence
	nativeToken := r.tokenManager.FindToken(fromChain, fromChain.NativeCurrencySymbol)
	if nativeToken == nil {
		err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", fromChain.ChainID, token.Symbol, ErrNativeTokenNotFound))
		return
	}

	// add token balance for the chain
	var tokenBalance *big.Int
	if input.SendType == sendtype.ERC721Transfer {
		tokenBalance = big.NewInt(1)
	} else if input.SendType == sendtype.ERC1155Transfer {
		tokenBalance, err = r.getERC1155Balance(ctx, fromChain, token, input.AddrFrom)
		if err != nil {
			err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", fromChain.ChainID, token.Symbol, err))
			return
		}
	} else {
		tokenBalance, err = r.getBalance(ctx, fromChain.ChainID, token, input.AddrFrom)
		if err != nil {
			err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", fromChain.ChainID, token.Symbol, err))
			return
		}
	}
	// add only if balance is not nil
	if tokenBalance != nil {
		r.activeBalanceMap.Store(makeBalanceKey(fromChain.ChainID, token.Symbol), tokenBalance)
	}

	if token.IsNative() {
		return
	}

	// add native token balance for the chain
	nativeBalance, err := r.getBalance(ctx, fromChain.ChainID, nativeToken, input.AddrFrom)
	if err != nil {
		err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", fromChain.ChainID, token.Symbol, err))
		return
	}
	// add only if balance is not nil
	if nativeBalance != nil {
		r.activeBalanceMap.Store(makeBalanceKey(fromChain.ChainID, nativeToken.Symbol), nativeBalance)
	}

	return
}

func (r *Router) CreateProcessorInputParams(input *requests.RouteInputParams, fromNetwork *params.Network, toNetwork *params.Network,
	fromToken *tokenTypes.Token, toToken *tokenTypes.Token, useCommunityTokenTransferDetailsAtIndex int) (pathprocessor.ProcessorInputParams, error) {
	var err error
	processorInputParams := pathprocessor.ProcessorInputParams{
		FromChain:          fromNetwork,
		ToChain:            toNetwork,
		FromToken:          fromToken,
		ToToken:            toToken,
		ToAddr:             input.AddrTo,
		FromAddr:           input.AddrFrom,
		AmountIn:           input.AmountIn.ToInt(),
		SlippagePercentage: input.SlippagePercentage,

		Username:  input.Username,
		PublicKey: input.PublicKey,
		PackID:    input.PackID.ToInt(),
	}

	if input.AmountOut != nil {
		processorInputParams.AmountOut = input.AmountOut.ToInt()
	}

	if input.PackID != nil {
		processorInputParams.PackID = input.PackID.ToInt()
	}

	if input.SendType.IsCommunityRelatedTransfer() {
		processorInputParams.CommunityParams = input.CommunityRouteInputParams

		if input.CommunityRouteInputParams.UseTransferDetails() && fromNetwork != nil {
			tokenContractAddress := input.CommunityRouteInputParams.TransferDetails[useCommunityTokenTransferDetailsAtIndex].TokenContractAddress
			tokenType, err := r.tokenManager.GetCommunityTokenType(fromNetwork.ChainID, tokenContractAddress.String())
			if err != nil {
				return processorInputParams, err
			}

			privilegeLevel, err := r.tokenManager.GetCommunityTokenPrivilegesLevel(fromNetwork.ChainID, tokenContractAddress.String())
			if err != nil {
				return processorInputParams, err
			}

			input.CommunityRouteInputParams.TransferDetails[useCommunityTokenTransferDetailsAtIndex].TokenType = tokenType
			input.CommunityRouteInputParams.TransferDetails[useCommunityTokenTransferDetailsAtIndex].PrivilegeLevel = privilegeLevel

			err = input.CommunityRouteInputParams.SetInternalParams(useCommunityTokenTransferDetailsAtIndex)
			if err != nil {
				return processorInputParams, err
			}
		}
	}

	if input.TestsMode {
		processorInputParams.TestsMode = input.TestsMode
		processorInputParams.TestEstimationMap = input.TestParams.EstimationMap
		processorInputParams.TestBonderFeeMap = input.TestParams.BonderFeeMap
		processorInputParams.TestApprovalGasEstimation = input.TestParams.ApprovalGasEstimation
		processorInputParams.TestApprovalL1Fee = input.TestParams.ApprovalL1Fee
	}

	return processorInputParams, err
}

func (r *Router) findFromAndToTokens(testsMode bool, input *requests.RouteInputParams, network *params.Network) (fromToken *tokenTypes.Token, toToken *tokenTypes.Token) {
	if testsMode {
		fromToken = input.TestParams.TokenFrom
	} else {
		fromToken = findToken(input.SendType, r.tokenManager, r.collectiblesService, input.AddrFrom, network, input.TokenID)
	}
	if fromToken == nil {
		return
	}

	if input.SendType == sendtype.Swap {
		toToken = findToken(input.SendType, r.tokenManager, r.collectiblesService, common.Address{}, network, input.ToTokenID)
	}
	return
}

func (r *Router) resolveRoute(ctx context.Context, input *requests.RouteInputParams, fromChain *params.Network, toChain *params.Network) (route routes.Route, processorErrors []*ProcessorError, err error) {
	var (
		testsMode = input.TestsMode && input.TestParams != nil

		usedNonces   = make(map[uint64]uint64)
		usedNoncesMu sync.Mutex
	)

	appendProcessorErrorFn := func(processorName string, sendType sendtype.SendType, fromChainID uint64, toChainID uint64, amount *big.Int, err error) {
		logutils.ZapLogger().Error("router.resolveRoute error",
			zap.String("processor", processorName),
			zap.Int("sendType", int(sendType)),
			zap.Uint64("fromChainId", fromChainID),
			zap.Uint64("toChainId", toChainID),
			zap.Stringer("amount", amount),
			zap.Error(err))
		processorErrors = append(processorErrors, &ProcessorError{
			ProcessorName: processorName,
			Error:         err,
		})
	}

	if !input.SendType.IsAvailableFor(fromChain) {
		err = errors.CreateErrorResponseFromError(fmt.Errorf("send type %d not available for from chain %d", input.SendType, fromChain.ChainID))
		return
	}

	fromToken, toToken := r.findFromAndToTokens(testsMode, input, fromChain)
	if fromToken == nil {
		err = errors.CreateErrorResponseFromError(fmt.Errorf("from token not found for send type %d on chain %d", input.SendType, fromChain.ChainID))
		return
	}

	var (
		fetchedFees   *fees.SuggestedFees
		noBaseFee     bool
		noPriorityFee bool
	)
	if testsMode {
		fetchedFees = input.TestParams.SuggestedFees
	} else {
		fetchedFees, noBaseFee, noPriorityFee, err = r.feesManager.SuggestedFees(ctx, fromChain.ChainID, r.lastInputParams.AddrFrom)
		if err != nil {
			err = errors.CreateErrorResponseFromError(fmt.Errorf("failed to fetch fees for from chain %d", fromChain.ChainID))
			return
		}
	}

	for _, pProcessor := range r.pathProcessors {
		// check if the processor is available for the send type
		if !input.SendType.CanUseProcessor(pProcessor.Name()) {
			continue
		}

		// if we're doing a single chain operation, we can skip bridge processors
		if walletCommon.IsSingleChainOperation(fromChain, toChain) && walletCommon.IsProcessorBridge(pProcessor.Name()) {
			continue
		}

		if !input.SendType.ProcessZeroAmountInProcessor(input.AmountIn.ToInt(), input.AmountOut.ToInt(), pProcessor.Name()) {
			continue
		}

		if input.UseCommunityTransferDetails() {
			for i := 0; i < len(input.CommunityRouteInputParams.TransferDetails); i++ {
				usedNoncesMu.Lock()
				path, err := r.buildPath(ctx, input, fromChain, toChain, fromToken, toToken, pProcessor, fetchedFees, usedNonces, noBaseFee, noPriorityFee, i)
				usedNoncesMu.Unlock()
				if err != nil {
					appendProcessorErrorFn(pProcessor.Name(), input.SendType, fromChain.ChainID, toChain.ChainID, input.AmountIn.ToInt(), err)
					continue
				}

				route = append(route, path)
			}
		} else {
			usedNoncesMu.Lock()
			path, err := r.buildPath(ctx, input, fromChain, toChain, fromToken, toToken, pProcessor, fetchedFees, usedNonces, noBaseFee, noPriorityFee, 0)
			usedNoncesMu.Unlock()
			if err != nil {
				appendProcessorErrorFn(pProcessor.Name(), input.SendType, fromChain.ChainID, toChain.ChainID, input.AmountIn.ToInt(), err)
				continue
			}

			route = append(route, path)
		}
	}

	return
}

func (r *Router) buildPath(ctx context.Context, input *requests.RouteInputParams, fromNetwork *params.Network,
	toNetwork *params.Network, fromToken *tokenTypes.Token, toToken *tokenTypes.Token,
	pathProcessor pathprocessor.PathProcessor, fetchedFees *fees.SuggestedFees, usedNonces map[uint64]uint64,
	noBaseFee bool, noPriorityFee bool, useCommunityTokenTransferDetailsAtIndex int) (*routes.Path, error) {
	if !input.SendType.IsAvailableFor(fromNetwork) {
		return nil, ErrPathNotSupportedForProvidedChain
	}

	if !input.SendType.IsAvailableBetween(fromNetwork, toNetwork) {
		return nil, ErrPathNotSupportedBetweenProvidedChains
	}

	processorInputParams, err := r.CreateProcessorInputParams(input, fromNetwork, toNetwork, fromToken, toToken, useCommunityTokenTransferDetailsAtIndex)
	if err != nil {
		return nil, err
	}

	can, err := pathProcessor.AvailableFor(processorInputParams)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrPathNotAvaliableForProvidedParameters
	}

	bonderFees, tokenFees, err := pathProcessor.CalculateFees(processorInputParams)
	if err != nil {
		return nil, err
	}

	contractAddress, err := pathProcessor.GetContractAddress(processorInputParams)
	if err != nil {
		return nil, err
	}
	approvalRequired, approvalAmountRequired, err := r.requireApproval(ctx, input.SendType, &contractAddress, processorInputParams)
	if err != nil {
		return nil, err
	}

	var (
		approvalGasLimit   uint64
		approvalPackedData []byte
		gasLimit           uint64
		txPackedData       []byte
	)
	if approvalRequired {
		if processorInputParams.TestsMode {
			approvalGasLimit = processorInputParams.TestApprovalGasEstimation
		} else {
			approvalPackedData, err = walletCommon.PackApprovalInputData(processorInputParams.AmountIn, &contractAddress)
			if err != nil {
				return nil, err
			}
			approvalGasLimit, err = r.estimateGasForApproval(processorInputParams, approvalPackedData)
			if err != nil {
				return nil, err
			}
		}
	}

	// Until we change the logic for Bridge to follow the same logic as for Swap (meaning first approval, then bridge tx) we have to provide txPackedData
	// otherwise we could do the logic below in the else block of `if approvalRequired` codition.
	if input.SendType != sendtype.Swap || !approvalRequired {
		txPackedData, err = pathProcessor.PackTxInputData(processorInputParams)
		if err != nil {
			return nil, err
		}

		gasLimit, err = pathProcessor.EstimateGas(processorInputParams, txPackedData)
		if err != nil {
			return nil, err
		}
	}

	amountOut, err := pathProcessor.CalculateAmountOut(processorInputParams)
	if err != nil {
		return nil, err
	}

	path := &routes.Path{
		RouterInputParamsUuid: input.Uuid,
		ProcessorName:         pathProcessor.Name(),
		FromChain:             fromNetwork,
		ToChain:               toNetwork,
		FromToken:             fromToken,
		ToToken:               toToken,
		AmountIn:              (*hexutil.Big)(processorInputParams.AmountIn),
		AmountOut:             (*hexutil.Big)(amountOut),

		// set params that we don't want to be recalculated with every new block creation
		SuggestedTxGasAmount:       gasLimit,
		SuggestedApprovalGasAmount: approvalGasLimit,

		UsedContractAddress: &contractAddress,
		TxPackedData:        txPackedData,
		TxGasAmount:         gasLimit,
		TxBonderFees:        (*hexutil.Big)(bonderFees),
		TxTokenFees:         (*hexutil.Big)(tokenFees),

		ApprovalRequired:        approvalRequired,
		ApprovalAmountRequired:  (*hexutil.Big)(approvalAmountRequired),
		ApprovalContractAddress: &contractAddress,
		ApprovalPackedData:      approvalPackedData,
		ApprovalGasAmount:       approvalGasLimit,
	}

	tokenBalance, ok := r.activeBalanceMap.Load(makeBalanceKey(path.FromChain.ChainID, path.FromToken.Symbol))
	if ok {
		tokenBalanceBigInt, ok := tokenBalance.(*big.Int)
		if ok &&
			processorInputParams.AmountIn.Cmp(walletCommon.ZeroBigIntValue()) > 0 &&
			tokenBalanceBigInt.Cmp(processorInputParams.AmountIn) == 0 {
			path.SubtractFees = true
		}
	}

	if input.SendType.IsCommunityRelatedTransfer() {
		// set community params copy as community params for the path instance
		communityParams := processorInputParams.CommunityParams.Copy()
		if input.UseCommunityTransferDetails() {
			// in case of multi token community transfer we need to set the internal params to refer to the correct token
			err = communityParams.SetInternalParams(useCommunityTokenTransferDetailsAtIndex)
			if err != nil {
				return nil, err
			}
		}
		path.SetCommunityParams(communityParams)
	}

	err = r.evaluateAndUpdatePathDetails(ctx, path, fetchedFees, usedNonces, noBaseFee, noPriorityFee, processorInputParams.TestsMode, processorInputParams.TestApprovalL1Fee)
	if err != nil {
		return nil, err
	}

	return path, nil
}

func (r *Router) checkBalancesForTheBestRoute(bestRoute routes.Route) (err error) {
	// make a copy of the active balance map
	balanceMapCopy := make(map[string]*big.Int)
	r.activeBalanceMap.Range(func(k, v interface{}) bool {
		balanceMapCopy[k.(string)] = new(big.Int).Set(v.(*big.Int))
		return true
	})
	if balanceMapCopy == nil {
		err = ErrCannotCheckBalance
		return
	}

	// check the best route for the required balances
	for _, path := range bestRoute {
		tokenKey := makeBalanceKey(path.FromChain.ChainID, path.FromToken.Symbol)
		if tokenBalance, ok := balanceMapCopy[tokenKey]; ok {
			if tokenBalance.Cmp(walletCommon.ZeroBigIntValue()) <= 0 && path.AmountIn.ToInt().Cmp(walletCommon.ZeroBigIntValue()) > 0 {
				err = &errors.ErrorResponse{
					Code:    ErrNotEnoughTokenBalance.Code,
					Details: fmt.Sprintf(ErrNotEnoughTokenBalance.Details, path.FromToken.Symbol, path.FromChain.ChainID),
				}
				return
			}
		}

		if path.ProcessorName == pathProcessorCommon.ProcessorBridgeHopName {
			if path.TxBonderFees.ToInt().Cmp(path.AmountOut.ToInt()) > 0 {
				err = ErrLowAmountInForHopBridge
				return
			}
		}

		if path.RequiredTokenBalance != nil && path.RequiredTokenBalance.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
			if tokenBalance, ok := balanceMapCopy[tokenKey]; ok {
				if tokenBalance.Cmp(path.RequiredTokenBalance) == -1 {
					err = &errors.ErrorResponse{
						Code:    ErrNotEnoughTokenBalance.Code,
						Details: fmt.Sprintf(ErrNotEnoughTokenBalance.Details, path.FromToken.Symbol, path.FromChain.ChainID),
					}
					return
				}
				balanceMapCopy[tokenKey].Sub(tokenBalance, path.RequiredTokenBalance)
			} else {
				err = ErrTokenNotFound
				return
			}
		}

		nativeTokenKey := makeBalanceKey(path.FromChain.ChainID, path.FromChain.NativeCurrencySymbol)
		if nativeBalance, ok := balanceMapCopy[nativeTokenKey]; ok {
			if nativeBalance.Cmp(path.RequiredNativeBalance) == -1 {
				err = &errors.ErrorResponse{
					Code:    ErrNotEnoughNativeBalance.Code,
					Details: fmt.Sprintf(ErrNotEnoughNativeBalance.Details, path.FromChain.NativeCurrencySymbol, path.FromChain.ChainID),
				}
				return
			}
			balanceMapCopy[nativeTokenKey].Sub(nativeBalance, path.RequiredNativeBalance)
		} else {
			err = ErrNativeTokenNotFound
			return
		}
	}

	return
}

func (r *Router) makeSuggestedRoute(input *requests.RouteInputParams, route routes.Route) (suggestedRoutes *SuggestedRoutes) {
	var prices map[string]float64
	if input.TestsMode {
		prices = input.TestParams.TokenPrices
	} else {
		var errPrices error
		prices, errPrices = fetchPrices(input.SendType, r.marketManager, []string{input.TokenID, input.ToTokenID})
		// error while fetching prices should not block the route evaluation, don't return, just log the error
		if errPrices != nil {
			logutils.ZapLogger().Error("router.checkRoute error fetching prices",
				zap.String("input.TokenID", input.TokenID),
				zap.String("input.ToTokenID", input.ToTokenID),
				zap.Error(errPrices))
		}
	}

	suggestedRoutes = &SuggestedRoutes{
		Uuid:          input.Uuid,
		Route:         route,
		UpdatedPrices: prices,
	}

	return
}

func (r *Router) checkBalancesForRouteAndAdjustAmountIn(route routes.Route) (err error) {
	if len(route) == 0 {
		return
	}

	err = r.checkBalancesForTheBestRoute(route)
	if err != nil {
		err = errors.CreateErrorResponseFromError(err)
		return
	}

	// At this point we have to do the final check and update the amountIn (subtracting fees) if complete balance is going to be sent for native token (ETH/BNB)
	for _, path := range route {
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

	return
}
