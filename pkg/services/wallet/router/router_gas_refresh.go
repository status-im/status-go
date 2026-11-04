package router

import (
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
)

// refreshGasEstimates re-estimates the gas of paths on chainID, once per GasReestimateInterval.
// Only suggestions change; evaluateAndUpdatePathDetails applies them unless gas is custom.
func (r *Router) refreshGasEstimates(chainID uint64, route routes.Route, now time.Time) {
	for _, path := range route {
		if path.FromChain == nil || path.FromChain.ChainID != chainID {
			continue
		}
		refreshPathGasEstimates(path, now,
			func() (gasEstimate, error) { return r.estimateTxGasForPath(path) },
			func() (gasEstimate, error) { return r.estimateApprovalGasForPath(path) },
			r.logger)
	}
}

// gasEstimate is a raw node estimate and its buffered limit.
type gasEstimate struct {
	estimated uint64
	limit     uint64
}

// refreshPathGasEstimates re-estimates the path's due gas; a failure keeps the old estimate and time.
func refreshPathGasEstimates(path *routes.Path, now time.Time, estimateTx func() (gasEstimate, error),
	estimateApproval func() (gasEstimate, error), logger *zap.Logger) {
	if path.ApprovalRequired {
		if path.GasEstimateDue(true, now) {
			gas, err := estimateApproval()
			if err != nil {
				logger.Warn("refreshPathGasEstimates: approval gas re-estimation failed, keeping the previous estimate",
					zap.String("processor", path.ProcessorName),
					zap.Uint64("fromChain", path.FromChain.ChainID),
					zap.Time("estimatedAt", path.ApprovalGasEstimatedAt),
					zap.Error(err))
			} else {
				logger.Debug("refreshPathGasEstimates: approval gas re-estimated",
					zap.String("processor", path.ProcessorName),
					zap.Uint64("fromChain", path.FromChain.ChainID),
					zap.Uint64("previousGas", path.SuggestedApprovalGasAmount),
					zap.Uint64("estimatedGas", gas.estimated),
					zap.Uint64("gas", gas.limit))
				path.SuggestedApprovalGasAmount = gas.limit
				path.EstimatedApprovalGasAmount = gas.estimated
				path.ApprovalGasEstimatedAt = now
			}
		}
		// the main tx can't be estimated before the approval is mined; swaps re-estimate it then
		// (ReevaluateRouterPath)
		return
	}

	if !path.GasEstimateDue(false, now) {
		return
	}
	gas, err := estimateTx()
	if err != nil {
		logger.Warn("refreshPathGasEstimates: tx gas re-estimation failed, keeping the previous estimate",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Time("estimatedAt", path.TxGasEstimatedAt),
			zap.Error(err))
		return
	}
	logger.Debug("refreshPathGasEstimates: tx gas re-estimated",
		zap.String("processor", path.ProcessorName),
		zap.Uint64("fromChain", path.FromChain.ChainID),
		zap.Uint64("previousGas", path.SuggestedTxGasAmount),
		zap.Uint64("estimatedGas", gas.estimated),
		zap.Uint64("gas", gas.limit))
	path.SuggestedTxGasAmount = gas.limit
	path.EstimatedTxGasAmount = gas.estimated
	path.TxGasEstimatedAt = now
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
