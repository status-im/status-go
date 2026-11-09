package router

import (
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
)

// refreshActiveRouteGas re-estimates the gas collect picks from the active route. The estimates
// (RPC, and HTTP for some providers) run without activeRoutesMutex, on copies of the paths; the
// results are applied only if the active route wasn't replaced meanwhile.
func (r *Router) refreshActiveRouteGas(now time.Time, collect func(routes.Route) []*gasRefresh) {
	r.activeRoutesMutex.Lock()
	active := r.activeRoutes
	var refreshes []*gasRefresh
	if active != nil {
		refreshes = collect(active.Route)
	}
	r.activeRoutesMutex.Unlock()
	if len(refreshes) == 0 {
		return
	}

	runGasRefreshes(refreshes, r.logger)

	r.activeRoutesMutex.Lock()
	defer r.activeRoutesMutex.Unlock()
	if r.activeRoutes != active {
		// a new route has estimates of its own
		return
	}
	applyGasRefreshes(active.Route, refreshes, now)
}

type gasEstimatorFunc func() (gasEstimate, error)

// gasEstimatorFactory makes the estimator for a path.
type gasEstimatorFactory func(*routes.Path) gasEstimatorFunc

func (r *Router) txGasEstimator(path *routes.Path) gasEstimatorFunc {
	return func() (gasEstimate, error) { return r.estimateTxGasForPath(path) }
}

func (r *Router) approvalGasEstimator(path *routes.Path) gasEstimatorFunc {
	return func() (gasEstimate, error) { return r.estimateApprovalGasForPath(path) }
}

// gasEstimate is a raw node estimate and its buffered limit.
type gasEstimate struct {
	estimated uint64
	limit     uint64
}

// gasRefresh is one re-estimate of a route's path, made on a copy of it.
type gasRefresh struct {
	index    int
	approval bool
	path     *routes.Path
	estimate gasEstimatorFunc
	gas      gasEstimate
	err      error
}

func newGasRefresh(index int, path *routes.Path, approval bool, txEstimator, approvalEstimator gasEstimatorFactory) *gasRefresh {
	refresh := &gasRefresh{index: index, approval: approval, path: path.Copy()}
	if approval {
		refresh.estimate = approvalEstimator(refresh.path)
	} else {
		refresh.estimate = txEstimator(refresh.path)
	}
	return refresh
}

// dueGasRefreshes picks the update loop's re-estimates: paths on chainID, once per
// GasReestimateInterval. Only suggestions change; evaluateAndUpdatePathDetails applies them unless
// gas is custom.
func dueGasRefreshes(route routes.Route, chainID uint64, now time.Time, txEstimator, approvalEstimator gasEstimatorFactory) []*gasRefresh {
	var refreshes []*gasRefresh
	for i, path := range route {
		if path.FromChain == nil || path.FromChain.ChainID != chainID {
			continue
		}
		// the main tx can't be estimated before the approval is mined
		approval := path.ApprovalRequired
		if path.GasEstimateDue(approval, now) {
			refreshes = append(refreshes, newGasRefresh(i, path, approval, txEstimator, approvalEstimator))
		}
	}
	return refreshes
}

// staleGasRefreshes picks the build's re-estimates: estimates older than GasEstimateMaxAge (the
// update loop may have stopped). Confirmed gas is never changed; the build refuses it if it is below
// the new estimate.
func staleGasRefreshes(route routes.Route, approvalPlaced func(processorName string) bool, now time.Time,
	txEstimator, approvalEstimator gasEstimatorFactory) []*gasRefresh {
	var refreshes []*gasRefresh
	for i, path := range route {
		approval := path.ApprovalRequired && !approvalPlaced(path.ProcessorName)
		if !path.GasEstimateFresh(approval, now) {
			refreshes = append(refreshes, newGasRefresh(i, path, approval, txEstimator, approvalEstimator))
		}
	}
	return refreshes
}

func runGasRefreshes(refreshes []*gasRefresh, logger *zap.Logger) {
	for _, refresh := range refreshes {
		refresh.gas, refresh.err = refresh.estimate()
		if refresh.err != nil {
			logger.Warn("runGasRefreshes: gas re-estimation failed, keeping the previous estimate",
				zap.String("processor", refresh.path.ProcessorName),
				zap.Uint64("fromChain", refresh.path.FromChain.ChainID),
				zap.Bool("approval", refresh.approval),
				zap.Time("txGasEstimatedAt", refresh.path.TxGasEstimatedAt),
				zap.Time("approvalGasEstimatedAt", refresh.path.ApprovalGasEstimatedAt),
				zap.Error(refresh.err))
			continue
		}
		logger.Debug("runGasRefreshes: gas re-estimated",
			zap.String("processor", refresh.path.ProcessorName),
			zap.Uint64("fromChain", refresh.path.FromChain.ChainID),
			zap.Bool("approval", refresh.approval),
			zap.Uint64("estimatedGas", refresh.gas.estimated),
			zap.Uint64("gas", refresh.gas.limit))
	}
}

// applyGasRefreshes stores the new estimates in the route they were collected from; a failed one
// keeps the old estimate and time. A main tx estimate keeps the value it was made for, so a value
// changed meanwhile makes it stale.
func applyGasRefreshes(route routes.Route, refreshes []*gasRefresh, now time.Time) {
	for _, refresh := range refreshes {
		if refresh.err != nil || refresh.index >= len(route) {
			continue
		}
		path := route[refresh.index]
		if refresh.approval {
			path.SuggestedApprovalGasAmount = refresh.gas.limit
			path.EstimatedApprovalGasAmount = refresh.gas.estimated
			path.ApprovalGasEstimatedAt = now
			continue
		}
		path.SuggestedTxGasAmount = refresh.gas.limit
		path.EstimatedTxGasAmount = refresh.gas.estimated
		path.TxGasEstimatedAt = now
		path.TxGasEstimatedValue = (*hexutil.Big)(estimationAmountIn(refresh.path))
	}
}

func (r *Router) estimateTxGasForPath(path *routes.Path) (gasEstimate, error) {
	processor, ok := r.pathProcessors[path.ProcessorName]
	if !ok {
		return gasEstimate{}, ErrCannotFindPathProcessorForProvidedIdentity
	}
	processorInputParams, err := r.processorInputParamsForPath(path)
	if err != nil {
		return gasEstimate{}, err
	}
	estimated, err := processor.EstimateGas(processorInputParams, path.TxPackedData)
	if err != nil {
		return gasEstimate{}, err
	}
	return gasEstimate{
		estimated: estimated,
		limit:     pathprocessor.BufferedGasEstimate(processor, processorInputParams, estimated),
	}, nil
}

func (r *Router) estimateApprovalGasForPath(path *routes.Path) (gasEstimate, error) {
	processorInputParams, err := r.processorInputParamsForPath(path)
	if err != nil {
		return gasEstimate{}, err
	}
	estimated, err := r.estimateGasForApproval(processorInputParams, path.ApprovalPackedData)
	if err != nil {
		return gasEstimate{}, err
	}
	return gasEstimate{estimated: estimated, limit: bufferApprovalGas(estimated)}, nil
}

// processorInputParamsForPath rebuilds the processor params the path was built with.
func (r *Router) processorInputParamsForPath(path *routes.Path) (pathprocessor.ProcessorInputParams, error) {
	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()
	input := r.lastInputParams
	if input == nil || input.Uuid != path.RouterInputParamsUuid {
		return pathprocessor.ProcessorInputParams{}, ErrCannotFindPathForProvidedIdentity
	}

	processorInputParams := newProcessorInputParams(input, path.FromChain, path.ToChain, path.FromToken, path.ToToken)
	processorInputParams.CommunityParams = path.GetCommunityParams()
	processorInputParams.AmountIn = estimationAmountIn(path)
	return processorInputParams, nil
}

// estimationAmountIn returns the amount to re-estimate with: the built-for amount when packed data
// encodes it, otherwise the value to sign.
func estimationAmountIn(path *routes.Path) *big.Int {
	amountIn := path.AmountIn
	if len(path.TxPackedData) > 0 && path.AmountInOriginal != nil {
		amountIn = path.AmountInOriginal
	}
	if amountIn == nil {
		return nil
	}
	return new(big.Int).Set(amountIn.ToInt())
}
