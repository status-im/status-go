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

	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/errors"
	"github.com/status-im/status-go/internal/logutils"
	communityToken "github.com/status-im/status-go/internal/protocol/communities/token"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"

	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/pkg/services/wallet/async"
	"github.com/status-im/status-go/pkg/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/market"
	"github.com/status-im/status-go/pkg/services/wallet/requests"
	"github.com/status-im/status-go/pkg/services/wallet/responses"
	"github.com/status-im/status-go/pkg/services/wallet/router/fees"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/lifi"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/paraswap"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

var (
	routerTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

type TokenManager interface {
	GetNativeTokenForChain(chainID uint64) (*tokentypes.Token, error)
	GetTokenByKey(tokenKey string) (*tokentypes.Token, error)
	GetCommunityTokenType(chainID uint64, tokenContractAddress string) (protobuf.CommunityTokenType, error)
	GetCommunityTokenPrivilegesLevel(chainID uint64, tokenContractAddress string) (communityToken.PrivilegesLevel, error)
}

type TokenBalanceFetcher interface {
	FetchSingle(ctx context.Context, chainID uint64, tokenAddress common.Address, accountAddress common.Address) (*big.Int, error)
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
	Route         routes.Route
	UpdatedPrices map[string]float64
}

type Router struct {
	rpcClient            *rpc.Client
	contractMaker        contracts.ContractMakerIface
	transactor           *transactions.Transactor
	tokenManager         TokenManager
	tokenBalancesFetcher TokenBalanceFetcher
	marketManager        *market.Manager
	collectiblesService  *collectibles.Service
	collectiblesManager  *collectibles.Manager
	feesManager          *fees.FeeManager
	pathProcessors       map[string]pathprocessor.PathProcessor
	scheduler            *async.Scheduler

	paraswapClientFactory func(chainID uint64) paraswap.ClientInterface
	lifiClientFactory     func(chainID uint64) lifi.ClientInterface

	activeBalanceMap sync.Map // map[string]*big.Int

	activeRoutesMutex sync.Mutex
	activeRoutes      *SuggestedRoutes

	routeCanceledMutex sync.Mutex
	routeCanceled      bool

	lastInputParamsMutex sync.Mutex
	lastInputParams      *requests.RouteInputParams

	clientsForUpdatesPerChains sync.Map

	logger *zap.Logger
}

func NewRouter(
	rpcClient *rpc.Client,
	transactor *transactions.Transactor,
	tokenManager TokenManager,
	tokenBalancesFetcher TokenBalanceFetcher,
	marketManager *market.Manager,
	collectibles *collectibles.Service, collectiblesManager *collectibles.Manager) *Router {
	processors := make(map[string]pathprocessor.PathProcessor)

	logger := logutils.ZapLogger().Named("router")
	return &Router{
		rpcClient:            rpcClient,
		contractMaker:        contracts.NewContractMaker(rpcClient),
		transactor:           transactor,
		tokenManager:         tokenManager,
		tokenBalancesFetcher: tokenBalancesFetcher,
		marketManager:        marketManager,
		collectiblesService:  collectibles,
		collectiblesManager:  collectiblesManager,
		feesManager:          fees.NewFeeManager(rpcClient, logger.Named("feeManager")),
		pathProcessors:       processors,
		scheduler:            async.NewScheduler(),
		paraswapClientFactory: func(chainID uint64) paraswap.ClientInterface {
			return paraswap.NewClientV5(chainID, pathprocessor.ParaswapPartnerID, walletCommon.ZeroAddress(), 0)
		},
		lifiClientFactory: func(chainID uint64) lifi.ClientInterface {
			return lifi.NewClient(chainID, lifi.Integrator, "")
		},
		logger: logger,
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
		lastUuid := ""
		r.lastInputParamsMutex.Lock()
		if r.lastInputParams != nil {
			lastUuid = r.lastInputParams.Uuid
		}
		r.lastInputParamsMutex.Unlock()
		r.logger.Warn("GetBestRouteAndAssociatedInputParams: no active route (cleared or last calculation failed/was canceled)",
			zap.String("lastInputParamsUuid", lastUuid))
		return nil, requests.RouteInputParams{}
	}

	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()
	if r.lastInputParams == nil {
		r.logger.Warn("GetBestRouteAndAssociatedInputParams: active route present but input params are missing",
			zap.String("activeRoutesUuid", r.activeRoutes.Uuid))
		return nil, requests.RouteInputParams{}
	}
	ip := *r.lastInputParams

	r.logger.Debug("GetBestRouteAndAssociatedInputParams: returning active route",
		zap.String("activeRoutesUuid", r.activeRoutes.Uuid),
		zap.String("inputParamsUuid", ip.Uuid),
		zap.Int("paths", len(r.activeRoutes.Route)))

	return r.activeRoutes.Route.Copy(), ip
}

func (r *Router) SetTestBalanceMap(balanceMap map[string]*big.Int) {
	for k, v := range balanceMap {
		r.activeBalanceMap.Store(k, v)
	}
}

func (r *Router) setCustomTxDetails(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, pathTxCustomParams *requests.PathTxCustomParams) error {
	if pathTxIdentity == nil {
		r.logger.Error("setCustomTxDetails: tx identity not provided")
		return ErrTxIdentityNotProvided
	}
	r.logger.Info("setCustomTxDetails: applying custom tx details",
		zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
		zap.String("pathTxIdentity", pathTxIdentity.TxIdentityKey()),
		zap.Uint64("chainId", pathTxIdentity.ChainID))

	err := pathTxIdentity.Validate()
	if err != nil {
		r.logger.Error("setCustomTxDetails: pathTxIdentity validation failed",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.Error(err))
		return err
	}
	if pathTxCustomParams == nil {
		r.logger.Error("setCustomTxDetails: custom params not provided",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid))
		return ErrTxCustomParamsNotProvided
	}
	err = pathTxCustomParams.Validate()
	if err != nil {
		r.logger.Error("setCustomTxDetails: custom params validation failed",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.Error(err))
		return err
	}

	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()
	if r.activeRoutes == nil || len(r.activeRoutes.Route) == 0 {
		r.logger.Error("setCustomTxDetails: no active route to customize",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid))
		return ErrCannotCustomizeIfNoRoute
	}

	var addrFrom common.Address
	r.lastInputParamsMutex.Lock()
	addrFrom = r.lastInputParams.AddrFrom
	r.lastInputParamsMutex.Unlock()

	fetchedFees, noBaseFee, noPriorityFee, err := r.feesManager.SuggestedFees(ctx, pathTxIdentity.ChainID, addrFrom)
	if err != nil {
		r.logger.Error("setCustomTxDetails: failed to fetch suggested fees",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.Uint64("chainId", pathTxIdentity.ChainID),
			zap.Error(err))
		return err
	}
	r.logger.Debug("setCustomTxDetails: suggested fees fetched",
		zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
		zap.Uint64("chainId", pathTxIdentity.ChainID),
		zap.Bool("noBaseFee", noBaseFee),
		zap.Bool("noPriorityFee", noPriorityFee))

	for _, path := range r.activeRoutes.Route {
		if path.PathIdentity() != pathTxIdentity.PathIdentity() {
			continue
		}
		r.logger.Debug("setCustomTxDetails: matched path",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Uint64("toChain", path.ToChain.ChainID))

		// update the custom params
		r.lastInputParamsMutex.Lock()
		if r.lastInputParams.PathTxCustomParams == nil {
			r.lastInputParams.PathTxCustomParams = make(map[string]*requests.PathTxCustomParams)
		}
		r.lastInputParams.PathTxCustomParams[pathTxIdentity.TxIdentityKey()] = pathTxCustomParams
		r.lastInputParamsMutex.Unlock()

		// update the path details
		usedNonces := make(map[uint64]uint64)
		err = r.evaluateAndUpdatePathDetails(ctx, path, fetchedFees, usedNonces, noBaseFee, noPriorityFee)
		if err != nil {
			r.logger.Error("setCustomTxDetails: evaluateAndUpdatePathDetails failed",
				zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
				zap.String("processor", path.ProcessorName),
				zap.Error(err))
			return err
		}
		err = r.checkBalancesForTheBestRoute(r.activeRoutes.Route)
		// inform the client about the changes
		if err != nil {
			r.logger.Error("setCustomTxDetails: checkBalancesForTheBestRoute returned an error",
				zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
				zap.Error(err))
		}
		r.logger.Info("setCustomTxDetails: path updated, sending result",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.String("processor", path.ProcessorName))
		sendRouterResult(pathTxIdentity.RouterInputParamsUuid, r.activeRoutes, err)

		return nil
	}

	r.logger.Error("setCustomTxDetails: no path matched provided identity",
		zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
		zap.String("pathTxIdentity", pathTxIdentity.TxIdentityKey()))
	return ErrCannotFindPathForProvidedIdentity
}

func (r *Router) SetFeeMode(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, feeMode fees.GasFeeMode) error {
	uuid := ""
	if pathTxIdentity != nil {
		uuid = pathTxIdentity.RouterInputParamsUuid
	}
	r.logger.Info("SetFeeMode: setting fee mode",
		zap.String("uuid", uuid),
		zap.Int("feeMode", int(feeMode)))
	if feeMode == fees.GasFeeCustom {
		r.logger.Error("SetFeeMode: custom fee mode cannot be set via SetFeeMode",
			zap.String("uuid", uuid))
		return ErrCustomFeeModeCannotBeSetThisWay
	}

	return r.setCustomTxDetails(ctx, pathTxIdentity, &requests.PathTxCustomParams{GasFeeMode: feeMode})
}

func (r *Router) SetCustomTxDetails(ctx context.Context, pathTxIdentity *requests.PathTxIdentity, pathTxCustomParams *requests.PathTxCustomParams) error {
	uuid := ""
	if pathTxIdentity != nil {
		uuid = pathTxIdentity.RouterInputParamsUuid
	}
	r.logger.Info("SetCustomTxDetails: applying custom tx details", zap.String("uuid", uuid))
	if pathTxCustomParams != nil && pathTxCustomParams.GasFeeMode != fees.GasFeeCustom {
		r.logger.Error("SetCustomTxDetails: only custom fee mode can be set this way",
			zap.String("uuid", uuid),
			zap.Int("feeMode", int(pathTxCustomParams.GasFeeMode)))
		return ErrOnlyCustomFeeModeCanBeSetThisWay
	}
	return r.setCustomTxDetails(ctx, pathTxIdentity, pathTxCustomParams)
}

// ReevaluateRouterPath reevaluates the tx-fields from the router path that matches the provided pathTxIdentity and sends signal.SuggestedRoutes.
func (r *Router) ReevaluateRouterPath(ctx context.Context, pathTxIdentity *requests.PathTxIdentity) error {
	if pathTxIdentity == nil {
		r.logger.Error("ReevaluateRouterPath: tx identity not provided")
		return ErrTxIdentityNotProvided
	}
	r.logger.Info("ReevaluateRouterPath: reevaluating path",
		zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
		zap.String("pathTxIdentity", pathTxIdentity.TxIdentityKey()),
		zap.Uint64("chainId", pathTxIdentity.ChainID))
	err := pathTxIdentity.Validate()
	if err != nil {
		r.logger.Error("ReevaluateRouterPath: pathTxIdentity validation failed",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.Error(err))
		return err
	}

	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()
	if r.activeRoutes == nil || len(r.activeRoutes.Route) == 0 {
		r.logger.Error("ReevaluateRouterPath: no active route to reevaluate",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid))
		return ErrNoBestRouteFound
	}

	var addrFrom common.Address
	r.lastInputParamsMutex.Lock()
	addrFrom = r.lastInputParams.AddrFrom
	r.lastInputParamsMutex.Unlock()

	fetchedFees, noBaseFee, noPriorityFee, err := r.feesManager.SuggestedFees(ctx, pathTxIdentity.ChainID, addrFrom)
	if err != nil {
		r.logger.Error("ReevaluateRouterPath: failed to fetch suggested fees",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.Uint64("chainId", pathTxIdentity.ChainID),
			zap.Error(err))
		return err
	}
	r.logger.Debug("ReevaluateRouterPath: suggested fees fetched",
		zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
		zap.Uint64("chainId", pathTxIdentity.ChainID),
		zap.Bool("noBaseFee", noBaseFee),
		zap.Bool("noPriorityFee", noPriorityFee))

	for _, path := range r.activeRoutes.Route {
		if path.PathIdentity() != pathTxIdentity.PathIdentity() {
			continue
		}
		r.logger.Debug("ReevaluateRouterPath: matched path",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Uint64("toChain", path.ToChain.ChainID))

		for _, pProcessor := range r.pathProcessors {
			if pProcessor.Name() != path.ProcessorName {
				continue
			}

			r.lastInputParamsMutex.Lock()
			processorInputParams, err := r.CreateProcessorInputParams(r.lastInputParams, path.FromToken, path.ToToken, 0)
			r.lastInputParamsMutex.Unlock()
			if err != nil {
				r.logger.Error("ReevaluateRouterPath: CreateProcessorInputParams failed",
					zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
					zap.String("processor", pProcessor.Name()),
					zap.Error(err))
				return err
			}

			txPackedData, err := pProcessor.PackTxInputData(processorInputParams)
			if err != nil {
				r.logger.Error("ReevaluateRouterPath: PackTxInputData failed",
					zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
					zap.String("processor", pProcessor.Name()),
					zap.Error(err))
				return err
			}

			gasLimit, err := pProcessor.EstimateGas(processorInputParams, txPackedData)
			if err != nil {
				r.logger.Error("ReevaluateRouterPath: EstimateGas failed",
					zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
					zap.String("processor", pProcessor.Name()),
					zap.Error(err))
				return err
			}
			r.logger.Debug("ReevaluateRouterPath: gas estimated",
				zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
				zap.String("processor", pProcessor.Name()),
				zap.Uint64("gasLimit", gasLimit))

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
			err = r.evaluateAndUpdatePathDetails(ctx, path, fetchedFees, usedNonces, noBaseFee, noPriorityFee)
			if err != nil {
				r.logger.Error("ReevaluateRouterPath: evaluateAndUpdatePathDetails failed",
					zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
					zap.String("processor", pProcessor.Name()),
					zap.Error(err))
				return err
			}

			// inform the client about the changes
			r.logger.Info("ReevaluateRouterPath: path reevaluated, sending result",
				zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
				zap.String("processor", pProcessor.Name()))
			sendRouterResult(pathTxIdentity.RouterInputParamsUuid, r.activeRoutes, err)

			return nil
		}

		r.logger.Error("ReevaluateRouterPath: processor not found for matched path",
			zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
			zap.String("processor", path.ProcessorName))
		return ErrCannotFindPathProcessorForProvidedIdentity
	}

	r.logger.Error("ReevaluateRouterPath: no path matched provided identity",
		zap.String("uuid", pathTxIdentity.RouterInputParamsUuid),
		zap.String("pathTxIdentity", pathTxIdentity.TxIdentityKey()))
	return ErrCannotFindPathForProvidedIdentity
}

func sendRouterResult(uuid string, result interface{}, err error) {
	logger := logutils.ZapLogger().Named("router")
	routesResponse := responses.RouterSuggestedRoutes{
		Uuid: uuid,
	}

	emptySignal := true
	if err != nil {
		errorResponse := errors.CreateErrorResponseFromError(err)
		routesResponse.ErrorResponse = errorResponse.(*errors.ErrorResponse)
		emptySignal = false
	}

	pathCount := 0
	if suggestedRoutes, ok := result.(*SuggestedRoutes); ok && suggestedRoutes != nil {
		routesResponse.Route = suggestedRoutes.Route
		pathCount = len(suggestedRoutes.Route)
		emptySignal = false
	}

	if emptySignal {
		logger.Debug("sendRouterResult: no payload to emit", zap.String("uuid", uuid))
		return
	}

	logger.Debug("sendRouterResult: emitting SuggestedRoutes signal",
		zap.String("uuid", uuid),
		zap.Int("paths", pathCount),
		zap.Bool("hasError", err != nil))
	signal.SendWalletEvent(signal.SuggestedRoutes, routesResponse)
}

func (r *Router) SuggestedRoutesAsync(input *requests.RouteInputParams) {
	r.logger.Info("SuggestedRoutesAsync: enqueuing route calculation",
		zap.String("uuid", input.Uuid),
		zap.Int("sendType", int(input.SendType)),
		zap.Uint64("fromChain", input.FromChainID),
		zap.Uint64("toChain", input.ToChainID),
		zap.String("fromToken", input.TokenKey),
		zap.String("toToken", input.ToTokenKey),
		zap.Stringer("amountIn", input.AmountIn.ToInt()),
		zap.Stringer("addrFrom", input.AddrFrom),
		zap.Stringer("addrTo", input.AddrTo))
	r.scheduler.Enqueue(routerTask, func(ctx context.Context) (interface{}, error) {
		return r.SuggestedRoutes(ctx, input)
	}, func(result interface{}, taskType async.TaskType, err error) {
		if err != nil {
			r.logger.Error("SuggestedRoutesAsync: route calculation finished with error",
				zap.String("uuid", input.Uuid),
				zap.Error(err))
		} else if suggestedRoutes, ok := result.(*SuggestedRoutes); ok && suggestedRoutes != nil {
			r.logger.Info("SuggestedRoutesAsync: route calculation finished",
				zap.String("uuid", input.Uuid),
				zap.Int("paths", len(suggestedRoutes.Route)))
		} else {
			r.logger.Info("SuggestedRoutesAsync: route calculation finished with empty result",
				zap.String("uuid", input.Uuid))
		}
		sendRouterResult(input.Uuid, result, err)
	})
}

func (r *Router) clearActiveRoute() {
	r.activeRoutesMutex.Lock()
	clearedUuid := ""
	if r.activeRoutes != nil {
		clearedUuid = r.activeRoutes.Uuid
	}
	r.activeRoutes = nil
	r.activeRoutesMutex.Unlock()
	r.logger.Info("clearActiveRoute: active route cleared", zap.String("clearedUuid", clearedUuid))
}

func (r *Router) markRouteCanceled(value bool) {
	r.routeCanceledMutex.Lock()
	r.routeCanceled = value
	r.routeCanceledMutex.Unlock()
	r.logger.Debug("markRouteCanceled", zap.Bool("canceled", value))
}

func (r *Router) abortUpdates() {
	r.logger.Debug("abortUpdates: aborting active fee/route updates")
	r.markRouteCanceled(true)
	r.unsubscribeFeesUpdateAccrossAllChains()
}

func (r *Router) StopSuggestedRoutesAsyncCalculation() {
	r.logger.Info("StopSuggestedRoutesAsyncCalculation: stopping scheduler and updates")
	r.abortUpdates()
	r.scheduler.Stop()
}

func (r *Router) StopSuggestedRoutesCalculation() {
	r.logger.Info("StopSuggestedRoutesCalculation: stopping updates")
	r.abortUpdates()
}

func (r *Router) SuggestedRoutes(ctx context.Context, input *requests.RouteInputParams) (suggestedRoutes *SuggestedRoutes, err error) {
	r.logger.Info("SuggestedRoutes: starting route calculation",
		zap.String("uuid", input.Uuid),
		zap.Int("sendType", int(input.SendType)),
		zap.Uint64("fromChain", input.FromChainID),
		zap.Uint64("toChain", input.ToChainID),
		zap.String("fromToken", input.TokenKey),
		zap.String("toToken", input.ToTokenKey),
		zap.Stringer("amountIn", input.AmountIn.ToInt()),
		zap.Stringer("addrFrom", input.AddrFrom),
		zap.Stringer("addrTo", input.AddrTo))

	r.clearActiveRoute()
	r.abortUpdates()
	r.markRouteCanceled(false)

	// clear all processors
	for _, processor := range r.pathProcessors {
		if clearable, ok := processor.(pathprocessor.PathProcessorClearable); ok {
			r.logger.Debug("SuggestedRoutes: clearing processor", zap.String("processor", processor.Name()))
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
		if suggestedRoutes == nil {
			r.routeCanceledMutex.Lock()
			canceled := r.routeCanceled
			r.routeCanceledMutex.Unlock()
			// leaves the router without an active route; a subsequent send for an
			// earlier uuid will fail with ErrCannotResolveRouteId
			r.logger.Warn("SuggestedRoutes: finished without active route",
				zap.String("uuid", input.Uuid),
				zap.Bool("canceled", canceled),
				zap.Error(err))
		} else {
			r.logger.Info("SuggestedRoutes: active route stored",
				zap.String("uuid", suggestedRoutes.Uuid),
				zap.Int("paths", len(suggestedRoutes.Route)))
		}
		r.routeCanceledMutex.Lock()
		if suggestedRoutes != nil && err == nil && !r.routeCanceled {
			// subscribe for updates
			r.logger.Debug("SuggestedRoutes: subscribing for fee updates",
				zap.String("uuid", input.Uuid),
				zap.Int("paths", len(suggestedRoutes.Route)))
			for _, path := range suggestedRoutes.Route {
				err = r.subscribeForUdates(path.FromChain.ChainID, input.AddrFrom)
			}
		}
		r.routeCanceledMutex.Unlock()
	}()

	testnetMode, err := r.rpcClient.GetNetworkManager().GetTestNetworksEnabled()
	if err != nil {
		r.logger.Error("SuggestedRoutes: failed to read testnet mode",
			zap.String("uuid", input.Uuid),
			zap.Error(err))
		return nil, errors.CreateErrorResponseFromError(err)
	}
	r.logger.Debug("SuggestedRoutes: testnet mode resolved",
		zap.String("uuid", input.Uuid),
		zap.Bool("testnetMode", testnetMode))

	input.TestnetMode = testnetMode

	err = input.Validate()
	if err != nil {
		r.logger.Error("SuggestedRoutes: input validation failed",
			zap.String("uuid", input.Uuid),
			zap.Error(err))
		return nil, errors.CreateErrorResponseFromError(err)
	}

	err = r.prepareBalanceMapForTokenOnChain(ctx, input)
	if err != nil {
		r.logger.Error("SuggestedRoutes: prepareBalanceMapForTokenOnChain failed",
			zap.String("uuid", input.Uuid),
			zap.Error(err))
		return nil, errors.CreateErrorResponseFromError(err)
	}

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
		r.logger.Info("SuggestedRoutes: no positive balance on any chain",
			zap.String("uuid", input.Uuid))
		return nil, ErrNoPositiveBalance
	}

	route, processorErrors, err := r.resolveRoute(ctx, input)
	if err != nil {
		r.logger.Error("SuggestedRoutes: resolveRoute failed",
			zap.String("uuid", input.Uuid),
			zap.Error(err))
		return nil, errors.CreateErrorResponseFromError(err)
	}
	r.logger.Info("SuggestedRoutes: resolveRoute completed",
		zap.String("uuid", input.Uuid),
		zap.Int("paths", len(route)),
		zap.Int("processorErrors", len(processorErrors)))

	err = r.checkBalancesForRouteAndAdjustAmountIn(route)
	if err != nil {
		// don't return here, cause we have to return the route anywaye, even there are balance issues
		r.logger.Error("SuggestedRoutes: checkBalancesForRouteAndAdjustAmountIn returned an error (route still returned)",
			zap.String("uuid", input.Uuid),
			zap.Error(err))
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
			r.logger.Info("SuggestedRoutes: no best route, surfacing processor error",
				zap.String("uuid", input.Uuid),
				zap.Error(err))
		} else {
			err = ErrNoBestRouteFound
			r.logger.Info("SuggestedRoutes: no best route and no processor errors",
				zap.String("uuid", input.Uuid))
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
func (r *Router) prepareBalanceMapForTokenOnChain(ctx context.Context, input *requests.RouteInputParams) (err error) {
	// clear the active balance map
	r.logger.Debug("prepareBalanceMapForTokenOnChain: preparing balance map",
		zap.String("uuid", input.Uuid),
		zap.Uint64("fromChain", input.FromChainID),
		zap.String("tokenKey", input.TokenKey),
		zap.Stringer("addrFrom", input.AddrFrom))

	r.activeBalanceMap = sync.Map{}

	// check token existence
	token := findToken(input.SendType, r.tokenManager, r.collectiblesManager, input.TokenKey)
	if token == nil {
		r.logger.Error("prepareBalanceMapForTokenOnChain: token not found",
			zap.String("uuid", input.Uuid),
			zap.String("tokenKey", input.TokenKey),
			zap.Int("sendType", int(input.SendType)))
		err = errors.CreateErrorResponseFromError(ErrTokenNotFound)
		return
	}
	// check native token existence
	nativeToken, err := r.tokenManager.GetNativeTokenForChain(input.FromChainID)
	if err != nil {
		r.logger.Error("prepareBalanceMapForTokenOnChain: failed to get native token for chain",
			zap.String("uuid", input.Uuid),
			zap.Uint64("fromChain", input.FromChainID),
			zap.Error(err))
		err = errors.CreateErrorResponseFromError(fmt.Errorf("getting native token for chain %d: %w", input.FromChainID, err))
		return
	}

	// add token balance for the chain
	var tokenBalance *big.Int
	if input.SendType == sendtype.ERC721Transfer {
		tokenBalance = big.NewInt(1)
		r.logger.Debug("prepareBalanceMapForTokenOnChain: ERC721 fixed balance of 1",
			zap.String("uuid", input.Uuid),
			zap.String("token", token.Symbol))
	} else if input.SendType == sendtype.ERC1155Transfer {
		tokenBalance, err = r.getERC1155Balance(ctx, input.FromChainID, token, input.AddrFrom)
		if err != nil {
			r.logger.Error("prepareBalanceMapForTokenOnChain: failed to fetch ERC1155 balance",
				zap.String("uuid", input.Uuid),
				zap.Uint64("fromChain", input.FromChainID),
				zap.String("token", token.Symbol),
				zap.Error(err))
			err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", input.FromChainID, token.Symbol, err))
			return
		}
	} else {
		tokenBalance, err = r.getBalance(ctx, input.FromChainID, token, input.AddrFrom)
		if err != nil {
			r.logger.Error("prepareBalanceMapForTokenOnChain: failed to fetch token balance",
				zap.String("uuid", input.Uuid),
				zap.Uint64("fromChain", input.FromChainID),
				zap.String("token", token.Symbol),
				zap.Error(err))
			err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", input.FromChainID, token.Symbol, err))
			return
		}
	}
	// add only if balance is not nil
	if tokenBalance != nil {
		r.activeBalanceMap.Store(makeBalanceKey(input.FromChainID, token.Symbol), tokenBalance)
		r.logger.Debug("prepareBalanceMapForTokenOnChain: token balance recorded",
			zap.String("uuid", input.Uuid),
			zap.Uint64("fromChain", input.FromChainID),
			zap.String("token", token.Symbol),
			zap.Stringer("balance", tokenBalance))
	}

	if token.IsNative() {
		return
	}

	// add native token balance for the chain
	nativeBalance, err := r.getBalance(ctx, input.FromChainID, nativeToken, input.AddrFrom)
	if err != nil {
		r.logger.Error("prepareBalanceMapForTokenOnChain: failed to fetch native balance",
			zap.String("uuid", input.Uuid),
			zap.Uint64("fromChain", input.FromChainID),
			zap.String("nativeToken", nativeToken.Symbol),
			zap.Error(err))
		err = errors.CreateErrorResponseFromError(fmt.Errorf("chain %d, token %s: %w", input.FromChainID, token.Symbol, err))
		return
	}
	// add only if balance is not nil
	if nativeBalance != nil {
		r.activeBalanceMap.Store(makeBalanceKey(input.FromChainID, nativeToken.Symbol), nativeBalance)
		r.logger.Debug("prepareBalanceMapForTokenOnChain: native token balance recorded",
			zap.String("uuid", input.Uuid),
			zap.Uint64("fromChain", input.FromChainID),
			zap.String("nativeToken", nativeToken.Symbol),
			zap.Stringer("balance", nativeBalance))
	}

	return
}

func (r *Router) CreateProcessorInputParams(input *requests.RouteInputParams, fromToken *tokentypes.Token, toToken *tokentypes.Token,
	useCommunityTokenTransferDetailsAtIndex int) (pathprocessor.ProcessorInputParams, error) {
	var err error

	fromChain := r.rpcClient.GetNetworkManager().Find(input.FromChainID)
	if fromChain == nil {
		// should never be here, input.Validate() ensures that the chain is supported
		panic(fmt.Errorf("from chain %d not found", input.FromChainID))
	}

	toChain := r.rpcClient.GetNetworkManager().Find(input.ToChainID)
	if toChain == nil {
		// should never be here, input.Validate() ensures that the chain is supported
		panic(fmt.Errorf("to chain %d not found", input.ToChainID))
	}

	processorInputParams := pathprocessor.ProcessorInputParams{
		FromChain:          fromChain,
		ToChain:            toChain,
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

		if input.CommunityRouteInputParams.UseTransferDetails() {
			tokenContractAddress := input.CommunityRouteInputParams.TransferDetails[useCommunityTokenTransferDetailsAtIndex].TokenContractAddress
			tokenType, err := r.tokenManager.GetCommunityTokenType(fromChain.ChainID, tokenContractAddress.String())
			if err != nil {
				r.logger.Error("CreateProcessorInputParams: failed to get community token type",
					zap.String("uuid", input.Uuid),
					zap.Uint64("chainId", fromChain.ChainID),
					zap.Stringer("tokenContract", tokenContractAddress),
					zap.Error(err))
				return processorInputParams, err
			}

			privilegeLevel, err := r.tokenManager.GetCommunityTokenPrivilegesLevel(fromChain.ChainID, tokenContractAddress.String())
			if err != nil {
				r.logger.Error("CreateProcessorInputParams: failed to get community token privilege level",
					zap.String("uuid", input.Uuid),
					zap.Uint64("chainId", fromChain.ChainID),
					zap.Stringer("tokenContract", tokenContractAddress),
					zap.Error(err))
				return processorInputParams, err
			}

			input.CommunityRouteInputParams.TransferDetails[useCommunityTokenTransferDetailsAtIndex].TokenType = tokenType
			input.CommunityRouteInputParams.TransferDetails[useCommunityTokenTransferDetailsAtIndex].PrivilegeLevel = privilegeLevel
			r.logger.Debug("CreateProcessorInputParams: community transfer details resolved",
				zap.String("uuid", input.Uuid),
				zap.Uint64("chainId", fromChain.ChainID),
				zap.Stringer("tokenContract", tokenContractAddress),
				zap.Int("transferDetailsIndex", useCommunityTokenTransferDetailsAtIndex),
				zap.Int("tokenType", int(tokenType)),
				zap.Int("privilegeLevel", int(privilegeLevel)))

			err = input.CommunityRouteInputParams.SetInternalParams(useCommunityTokenTransferDetailsAtIndex)
			if err != nil {
				r.logger.Error("CreateProcessorInputParams: failed to set community internal params",
					zap.String("uuid", input.Uuid),
					zap.Error(err))
				return processorInputParams, err
			}
		}
	}

	return processorInputParams, err
}

func (r *Router) findFromAndToTokens(input *requests.RouteInputParams, chainID uint64) (fromToken *tokentypes.Token, toToken *tokentypes.Token) {
	fromToken = findToken(input.SendType, r.tokenManager, r.collectiblesManager, input.TokenKey)
	if fromToken == nil {
		return
	}

	toToken = findToken(input.SendType, r.tokenManager, r.collectiblesManager, input.ToTokenKey)
	return
}

func (r *Router) resolveRoute(ctx context.Context, input *requests.RouteInputParams) (route routes.Route, processorErrors []*ProcessorError, err error) {
	r.logger.Info("resolveRoute: resolving candidate paths",
		zap.String("uuid", input.Uuid),
		zap.Int("sendType", int(input.SendType)),
		zap.Uint64("fromChain", input.FromChainID),
		zap.Uint64("toChain", input.ToChainID))

	var (
		usedNonces   = make(map[uint64]uint64)
		usedNoncesMu sync.Mutex
	)

	appendProcessorErrorFn := func(processorName string, sendType sendtype.SendType, fromChainID uint64, toChainID uint64, amount *big.Int, err error) {
		r.logger.Error("resolveRoute: processor failed to build path",
			zap.String("uuid", input.Uuid),
			zap.String("processor", processorName),
			zap.Int("sendType", int(sendType)),
			zap.Uint64("fromChain", fromChainID),
			zap.Uint64("toChain", toChainID),
			zap.Stringer("amountIn", amount),
			zap.Error(err))
		processorErrors = append(processorErrors, &ProcessorError{
			ProcessorName: processorName,
			Error:         err,
		})
	}

	if !input.SendType.IsAvailableFor(input.FromChainID) {
		r.logger.Error("resolveRoute: send type not available for from chain",
			zap.String("uuid", input.Uuid),
			zap.Int("sendType", int(input.SendType)),
			zap.Uint64("fromChain", input.FromChainID))
		err = errors.CreateErrorResponseFromError(fmt.Errorf("send type %d not available for from chain %d", input.SendType, input.FromChainID))
		return
	}

	fromToken, toToken := r.findFromAndToTokens(input, input.FromChainID)
	if fromToken == nil {
		r.logger.Error("resolveRoute: from token not found",
			zap.String("uuid", input.Uuid),
			zap.Int("sendType", int(input.SendType)),
			zap.Uint64("fromChain", input.FromChainID),
			zap.String("tokenKey", input.TokenKey))
		err = errors.CreateErrorResponseFromError(fmt.Errorf("from token not found for send type %d on chain %d", input.SendType, input.FromChainID))
		return
	}
	if !input.SendType.IsCollectiblesTransfer() && toToken == nil {
		r.logger.Error("resolveRoute: to token not found",
			zap.String("uuid", input.Uuid),
			zap.Int("sendType", int(input.SendType)),
			zap.Uint64("toChain", input.ToChainID),
			zap.String("toTokenKey", input.ToTokenKey))
		err = errors.CreateErrorResponseFromError(fmt.Errorf("to token not found for send type %d on chain %d", input.SendType, input.ToChainID))
		return
	}
	r.logger.Debug("resolveRoute: tokens resolved",
		zap.String("uuid", input.Uuid),
		zap.String("fromToken", fromToken.Symbol),
		zap.String("toTokenKey", input.ToTokenKey))

	var (
		fetchedFees   *fees.SuggestedFees
		noBaseFee     bool
		noPriorityFee bool
	)
	fetchedFees, noBaseFee, noPriorityFee, err = r.feesManager.SuggestedFees(ctx, input.FromChainID, r.lastInputParams.AddrFrom)
	if err != nil {
		r.logger.Error("resolveRoute: failed to fetch suggested fees",
			zap.String("uuid", input.Uuid),
			zap.Uint64("fromChain", input.FromChainID),
			zap.Error(err))
		err = errors.CreateErrorResponseFromError(fmt.Errorf("failed to fetch fees for from chain %d", input.FromChainID))
		return
	}
	r.logger.Debug("resolveRoute: suggested fees fetched",
		zap.String("uuid", input.Uuid),
		zap.Uint64("fromChain", input.FromChainID),
		zap.Bool("noBaseFee", noBaseFee),
		zap.Bool("noPriorityFee", noPriorityFee))

	for _, pProcessor := range r.pathProcessors {
		// check if the processor is available for the send type
		if !input.SendType.CanUseProcessor(pProcessor.Name()) {
			r.logger.Debug("resolveRoute: skipping processor (not usable for send type)",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pProcessor.Name()),
				zap.Int("sendType", int(input.SendType)))
			continue
		}

		// on a single-chain operation, skip bridge-only processors (LI.FI is also a swap)
		if walletCommon.IsSingleChainOperation(input.FromChainID, input.ToChainID) &&
			walletCommon.IsProcessorBridge(pProcessor.Name()) && !walletCommon.IsProcessorSwap(pProcessor.Name()) {
			r.logger.Debug("resolveRoute: skipping bridge processor for single-chain op",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pProcessor.Name()),
				zap.Uint64("chainId", input.FromChainID))
			continue
		}

		if !input.SendType.ProcessZeroAmountInProcessor(input.AmountIn.ToInt(), input.AmountOut.ToInt(), pProcessor.Name()) {
			r.logger.Debug("resolveRoute: skipping processor (zero-amount rule)",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pProcessor.Name()))
			continue
		}

		r.logger.Debug("resolveRoute: considering processor",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pProcessor.Name()))

		if input.UseCommunityTransferDetails() {
			for i := 0; i < len(input.CommunityRouteInputParams.TransferDetails); i++ {
				usedNoncesMu.Lock()
				path, err := r.buildPath(ctx, input, fromToken, toToken, pProcessor, fetchedFees, usedNonces, noBaseFee, noPriorityFee, i)
				usedNoncesMu.Unlock()
				if err != nil {
					appendProcessorErrorFn(pProcessor.Name(), input.SendType, input.FromChainID, input.ToChainID, input.AmountIn.ToInt(), err)
					continue
				}

				r.logger.Debug("resolveRoute: path built (community transfer)",
					zap.String("uuid", input.Uuid),
					zap.String("processor", pProcessor.Name()),
					zap.Int("transferDetailsIndex", i))
				route = append(route, path)
			}
		} else {
			usedNoncesMu.Lock()
			path, err := r.buildPath(ctx, input, fromToken, toToken, pProcessor, fetchedFees, usedNonces, noBaseFee, noPriorityFee, 0)
			usedNoncesMu.Unlock()
			if err != nil {
				appendProcessorErrorFn(pProcessor.Name(), input.SendType, input.FromChainID, input.ToChainID, input.AmountIn.ToInt(), err)
				continue
			}

			r.logger.Debug("resolveRoute: path built",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pProcessor.Name()))
			route = append(route, path)
		}
	}

	r.logger.Info("resolveRoute: completed",
		zap.String("uuid", input.Uuid),
		zap.Int("paths", len(route)),
		zap.Int("processorErrors", len(processorErrors)))
	return
}

func (r *Router) buildPath(ctx context.Context, input *requests.RouteInputParams, fromToken *tokentypes.Token,
	toToken *tokentypes.Token, pathProcessor pathprocessor.PathProcessor, fetchedFees *fees.SuggestedFees,
	usedNonces map[uint64]uint64, noBaseFee bool, noPriorityFee bool, useCommunityTokenTransferDetailsAtIndex int) (*routes.Path, error) {
	r.logger.Debug("buildPath: building path",
		zap.String("uuid", input.Uuid),
		zap.String("processor", pathProcessor.Name()),
		zap.Uint64("fromChain", input.FromChainID),
		zap.Uint64("toChain", input.ToChainID),
		zap.String("fromToken", fromToken.Symbol),
		zap.Int("transferDetailsIndex", useCommunityTokenTransferDetailsAtIndex))

	if !input.SendType.IsAvailableFor(input.FromChainID) {
		return nil, ErrPathNotSupportedForProvidedChain
	}

	if !input.SendType.IsAvailableBetween(input.FromChainID, input.ToChainID) {
		return nil, ErrPathNotSupportedBetweenProvidedChains
	}

	processorInputParams, err := r.CreateProcessorInputParams(input, fromToken, toToken, useCommunityTokenTransferDetailsAtIndex)
	if err != nil {
		r.logger.Error("buildPath: CreateProcessorInputParams failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}

	can, err := pathProcessor.AvailableFor(processorInputParams)
	if err != nil {
		r.logger.Error("buildPath: AvailableFor failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}
	if !can {
		r.logger.Debug("buildPath: processor not available for params",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()))
		return nil, ErrPathNotAvaliableForProvidedParameters
	}

	bonderFees, tokenFees, err := pathProcessor.CalculateFees(processorInputParams)
	if err != nil {
		r.logger.Error("buildPath: CalculateFees failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}
	r.logger.Debug("buildPath: fees calculated",
		zap.String("uuid", input.Uuid),
		zap.String("processor", pathProcessor.Name()),
		zap.Stringer("bonderFees", bonderFees),
		zap.Stringer("tokenFees", tokenFees))

	contractAddress, err := pathProcessor.GetContractAddress(processorInputParams)
	if err != nil {
		r.logger.Error("buildPath: GetContractAddress failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}
	approvalRequired, approvalAmountRequired, err := r.requireApproval(ctx, input.SendType, &contractAddress, processorInputParams)
	if err != nil {
		r.logger.Error("buildPath: requireApproval failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}
	r.logger.Debug("buildPath: approval check completed",
		zap.String("uuid", input.Uuid),
		zap.String("processor", pathProcessor.Name()),
		zap.Bool("approvalRequired", approvalRequired),
		zap.Stringer("approvalAmount", approvalAmountRequired))

	var (
		approvalGasLimit   uint64
		approvalPackedData []byte
		gasLimit           uint64
		txPackedData       []byte
	)
	if approvalRequired {
		approvalPackedData, err = walletCommon.PackApprovalInputData(processorInputParams.AmountIn, &contractAddress)
		if err != nil {
			r.logger.Error("buildPath: PackApprovalInputData failed",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pathProcessor.Name()),
				zap.Error(err))
			return nil, err
		}
		approvalGasLimit, err = r.estimateGasForApproval(processorInputParams, approvalPackedData)
		if err != nil {
			r.logger.Error("buildPath: estimateGasForApproval failed",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pathProcessor.Name()),
				zap.Error(err))
			return nil, err
		}
		r.logger.Debug("buildPath: approval gas estimated",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Uint64("approvalGasLimit", approvalGasLimit))
	}

	// Until we change the logic for Bridge to follow the same logic as for Swap (meaning first approval, then bridge tx) we have to provide txPackedData
	// otherwise we could do the logic below in the else block of `if approvalRequired` codition.
	if input.SendType != sendtype.Swap || !approvalRequired {
		txPackedData, err = pathProcessor.PackTxInputData(processorInputParams)
		if err != nil {
			r.logger.Error("buildPath: PackTxInputData failed",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pathProcessor.Name()),
				zap.Error(err))
			return nil, err
		}

		gasLimit, err = pathProcessor.EstimateGas(processorInputParams, txPackedData)
		if err != nil {
			r.logger.Error("buildPath: EstimateGas failed",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pathProcessor.Name()),
				zap.Error(err))
			return nil, err
		}
		r.logger.Debug("buildPath: tx gas estimated",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Uint64("gasLimit", gasLimit))
	}

	amountOut, err := pathProcessor.CalculateAmountOut(processorInputParams)
	if err != nil {
		r.logger.Error("buildPath: CalculateAmountOut failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}
	r.logger.Debug("buildPath: amount out calculated",
		zap.String("uuid", input.Uuid),
		zap.String("processor", pathProcessor.Name()),
		zap.Stringer("amountOut", amountOut))

	path := &routes.Path{
		RouterInputParamsUuid: input.Uuid,
		ProcessorName:         pathProcessor.Name(),
		FromChain:             processorInputParams.FromChain,
		ToChain:               processorInputParams.ToChain,
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

	// processors that route through an underlying tool/exchange (e.g. LI.FI -> "1inch")
	// can surface it here; others simply leave path.Tool empty
	if tp, ok := pathProcessor.(interface {
		GetProviderTool(pathprocessor.ProcessorInputParams) string
	}); ok {
		path.Tool = tp.GetProviderTool(processorInputParams)
	}

	tokenBalance, ok := r.activeBalanceMap.Load(makeBalanceKey(path.FromChain.ChainID, path.FromToken.Symbol))
	if ok {
		tokenBalanceBigInt, ok := tokenBalance.(*big.Int)
		if ok &&
			processorInputParams.AmountIn.Cmp(walletCommon.ZeroBigIntValue()) > 0 &&
			tokenBalanceBigInt.Cmp(processorInputParams.AmountIn) == 0 {
			path.SubtractFees = true
			r.logger.Debug("buildPath: full balance send detected, subtractFees=true",
				zap.String("uuid", input.Uuid),
				zap.String("processor", pathProcessor.Name()),
				zap.String("token", path.FromToken.Symbol))
		}
	}

	if input.SendType.IsCommunityRelatedTransfer() {
		// set community params copy as community params for the path instance
		communityParams := processorInputParams.CommunityParams.Copy()
		if input.UseCommunityTransferDetails() {
			// in case of multi token community transfer we need to set the internal params to refer to the correct token
			err = communityParams.SetInternalParams(useCommunityTokenTransferDetailsAtIndex)
			if err != nil {
				r.logger.Error("buildPath: community SetInternalParams failed",
					zap.String("uuid", input.Uuid),
					zap.String("processor", pathProcessor.Name()),
					zap.Error(err))
				return nil, err
			}
		}
		path.SetCommunityParams(communityParams)
	}

	err = r.evaluateAndUpdatePathDetails(ctx, path, fetchedFees, usedNonces, noBaseFee, noPriorityFee)
	if err != nil {
		r.logger.Error("buildPath: evaluateAndUpdatePathDetails failed",
			zap.String("uuid", input.Uuid),
			zap.String("processor", pathProcessor.Name()),
			zap.Error(err))
		return nil, err
	}

	r.logger.Debug("buildPath: path built successfully",
		zap.String("uuid", input.Uuid),
		zap.String("processor", pathProcessor.Name()),
		zap.Uint64("fromChain", path.FromChain.ChainID),
		zap.Uint64("toChain", path.ToChain.ChainID),
		zap.String("fromToken", path.FromToken.Symbol),
		zap.Stringer("amountIn", processorInputParams.AmountIn),
		zap.Stringer("amountOut", amountOut),
		zap.Bool("approvalRequired", approvalRequired))
	return path, nil
}

func (r *Router) checkBalancesForTheBestRoute(bestRoute routes.Route) (err error) {
	// make a copy of the active balance map
	r.logger.Debug("checkBalancesForTheBestRoute: checking balances",
		zap.Int("paths", len(bestRoute)))

	balanceMapCopy := make(map[string]*big.Int)
	r.activeBalanceMap.Range(func(k, v interface{}) bool {
		balanceMapCopy[k.(string)] = new(big.Int).Set(v.(*big.Int))
		return true
	})
	if balanceMapCopy == nil {
		r.logger.Error("checkBalancesForTheBestRoute: balance map copy is nil")
		err = ErrCannotCheckBalance
		return
	}

	// check the best route for the required balances
	for _, path := range bestRoute {
		tokenKey := makeBalanceKey(path.FromChain.ChainID, path.FromToken.Symbol)
		if tokenBalance, ok := balanceMapCopy[tokenKey]; ok {
			if tokenBalance.Cmp(walletCommon.ZeroBigIntValue()) <= 0 && path.AmountIn.ToInt().Cmp(walletCommon.ZeroBigIntValue()) > 0 {
				r.logger.Error("checkBalancesForTheBestRoute: zero token balance with non-zero amountIn",
					zap.String("processor", path.ProcessorName),
					zap.String("token", path.FromToken.Symbol),
					zap.Uint64("fromChain", path.FromChain.ChainID),
					zap.Stringer("amountIn", path.AmountIn.ToInt()))
				err = &errors.ErrorResponse{
					Code:    ErrNotEnoughTokenBalance.Code,
					Details: fmt.Sprintf(ErrNotEnoughTokenBalance.Details, path.FromToken.Symbol, path.FromChain.ChainID),
				}
				return
			}
		}

		if path.ProcessorName == pathProcessorCommon.ProcessorBridgeHopName {
			if path.TxBonderFees.ToInt().Cmp(path.AmountOut.ToInt()) > 0 {
				r.logger.Error("checkBalancesForTheBestRoute: hop bonder fee exceeds amountOut",
					zap.Uint64("fromChain", path.FromChain.ChainID),
					zap.Stringer("bonderFees", path.TxBonderFees.ToInt()),
					zap.Stringer("amountOut", path.AmountOut.ToInt()))
				err = ErrLowAmountInForHopBridge
				return
			}
		}

		if path.RequiredTokenBalance != nil && path.RequiredTokenBalance.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
			if tokenBalance, ok := balanceMapCopy[tokenKey]; ok {
				if tokenBalance.Cmp(path.RequiredTokenBalance) == -1 {
					r.logger.Error("checkBalancesForTheBestRoute: insufficient token balance",
						zap.String("processor", path.ProcessorName),
						zap.String("token", path.FromToken.Symbol),
						zap.Uint64("fromChain", path.FromChain.ChainID),
						zap.Stringer("balance", tokenBalance),
						zap.Stringer("required", path.RequiredTokenBalance))
					err = &errors.ErrorResponse{
						Code:    ErrNotEnoughTokenBalance.Code,
						Details: fmt.Sprintf(ErrNotEnoughTokenBalance.Details, path.FromToken.Symbol, path.FromChain.ChainID),
					}
					return
				}
				balanceMapCopy[tokenKey].Sub(tokenBalance, path.RequiredTokenBalance)
			} else {
				r.logger.Error("checkBalancesForTheBestRoute: token entry not found in balance map",
					zap.String("token", path.FromToken.Symbol),
					zap.Uint64("fromChain", path.FromChain.ChainID))
				err = ErrTokenNotFound
				return
			}
		}

		nativeTokenKey := makeBalanceKey(path.FromChain.ChainID, path.FromChain.NativeCurrencySymbol)
		if nativeBalance, ok := balanceMapCopy[nativeTokenKey]; ok {
			if nativeBalance.Cmp(path.RequiredNativeBalance) == -1 {
				r.logger.Error("checkBalancesForTheBestRoute: insufficient native balance",
					zap.String("processor", path.ProcessorName),
					zap.String("nativeToken", path.FromChain.NativeCurrencySymbol),
					zap.Uint64("fromChain", path.FromChain.ChainID),
					zap.Stringer("balance", nativeBalance),
					zap.Stringer("required", path.RequiredNativeBalance))
				err = &errors.ErrorResponse{
					Code:    ErrNotEnoughNativeBalance.Code,
					Details: fmt.Sprintf(ErrNotEnoughNativeBalance.Details, path.FromChain.NativeCurrencySymbol, path.FromChain.ChainID),
				}
				return
			}
			balanceMapCopy[nativeTokenKey].Sub(nativeBalance, path.RequiredNativeBalance)
		} else {
			r.logger.Error("checkBalancesForTheBestRoute: native token entry not found in balance map",
				zap.String("nativeToken", path.FromChain.NativeCurrencySymbol),
				zap.Uint64("fromChain", path.FromChain.ChainID))
			err = ErrNativeTokenNotFound
			return
		}
	}

	r.logger.Debug("checkBalancesForTheBestRoute: balance check passed", zap.Int("paths", len(bestRoute)))
	return
}

func (r *Router) makeSuggestedRoute(input *requests.RouteInputParams, route routes.Route) (suggestedRoutes *SuggestedRoutes) {
	r.logger.Debug("makeSuggestedRoute: assembling result",
		zap.String("uuid", input.Uuid),
		zap.Int("paths", len(route)))

	prices, errPrices := r.fetchPrices(input.SendType, []string{input.TokenKey, input.ToTokenKey})
	// error while fetching prices should not block the route evaluation, don't return, just log the error
	if errPrices != nil {
		r.logger.Error("makeSuggestedRoute: error fetching prices (route still returned)",
			zap.String("uuid", input.Uuid),
			zap.String("fromToken", input.TokenKey),
			zap.String("toToken", input.ToTokenKey),
			zap.Error(errPrices))
	} else {
		r.logger.Debug("makeSuggestedRoute: prices fetched",
			zap.String("uuid", input.Uuid),
			zap.Int("priceEntries", len(prices)))
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
		r.logger.Debug("checkBalancesForRouteAndAdjustAmountIn: empty route, skipping")
		return
	}
	r.logger.Debug("checkBalancesForRouteAndAdjustAmountIn: checking balances and adjusting amountIn",
		zap.Int("paths", len(route)))

	err = r.checkBalancesForTheBestRoute(route)
	if err != nil {
		err = errors.CreateErrorResponseFromError(err)
		return
	}

	// At this point we have to do the final check and update the amountIn (subtracting fees) if complete balance is going to be sent for native token (ETH/BNB)
	for _, path := range route {
		if path.SubtractFees && path.FromToken.IsNative() {
			r.logger.Debug("checkBalancesForRouteAndAdjustAmountIn: subtracting fees from amountIn for full-balance native send",
				zap.String("processor", path.ProcessorName),
				zap.Uint64("fromChain", path.FromChain.ChainID),
				zap.String("token", path.FromToken.Symbol),
				zap.Stringer("amountInBefore", path.AmountIn.ToInt()),
				zap.Stringer("txFee", path.TxFee.ToInt()),
				zap.Stringer("txL1Fee", path.TxL1Fee.ToInt()),
				zap.Bool("approvalRequired", path.ApprovalRequired))
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
			r.logger.Debug("checkBalancesForRouteAndAdjustAmountIn: amountIn adjusted",
				zap.String("processor", path.ProcessorName),
				zap.Stringer("amountInAfter", path.AmountIn.ToInt()))
		}
	}

	return
}
