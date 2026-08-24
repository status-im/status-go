package router

import (
	"context"
	"errors"
	"math/big"
	"slices"

	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/contracts"
	gaspriceproxy "github.com/status-im/status-go/internal/contracts/gas-price-proxy"
	"github.com/status-im/status-go/internal/contracts/hop"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	"github.com/status-im/status-go/pkg/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/market"
	"github.com/status-im/status-go/pkg/services/wallet/requests"
	"github.com/status-im/status-go/pkg/services/wallet/router/fees"
	"github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
	"github.com/status-im/status-go/pkg/services/wallet/router/sendtype"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func (r *Router) requireApproval(ctx context.Context, sendType sendtype.SendType, approvalContractAddress *common.Address, params pathprocessor.ProcessorInputParams) (
	bool, *big.Int, error) {
	if sendType.IsCollectiblesTransfer() || sendType.IsEnsTransfer() || sendType.IsStickersTransfer() {
		r.logger.Debug("requireApproval: not required (collectible/ens/stickers)",
			zap.Int("sendType", int(sendType)),
			zap.Uint64("chainId", params.FromChain.ChainID))
		return false, nil, nil
	}

	if params.FromToken.IsNative() {
		r.logger.Debug("requireApproval: not required (native token)",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.String("token", params.FromToken.Symbol))
		return false, nil, nil
	}

	contractMaker := contracts.NewContractMaker(r.rpcClient)

	contract, err := contractMaker.NewERC20(params.FromChain.ChainID, params.FromToken.Address)
	if err != nil {
		r.logger.Error("requireApproval: failed to instantiate ERC20 contract",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.String("token", params.FromToken.Symbol),
			zap.Stringer("tokenAddress", params.FromToken.Address),
			zap.Error(err))
		return false, nil, err
	}

	if approvalContractAddress == nil || *approvalContractAddress == walletCommon.ZeroAddress() {
		r.logger.Debug("requireApproval: not required (no approval contract)",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.String("token", params.FromToken.Symbol))
		return false, nil, nil
	}

	allowance, err := contract.Allowance(&bind.CallOpts{
		Context: ctx,
	}, params.FromAddr, *approvalContractAddress)

	if err != nil {
		r.logger.Error("requireApproval: failed to read allowance",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.String("token", params.FromToken.Symbol),
			zap.Stringer("spender", approvalContractAddress),
			zap.Error(err))
		return false, nil, err
	}

	if allowance.Cmp(params.AmountIn) >= 0 {
		r.logger.Debug("requireApproval: existing allowance covers amountIn",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.String("token", params.FromToken.Symbol),
			zap.Stringer("allowance", allowance),
			zap.Stringer("amountIn", params.AmountIn))
		return false, nil, nil
	}

	r.logger.Debug("requireApproval: approval required",
		zap.Uint64("chainId", params.FromChain.ChainID),
		zap.String("token", params.FromToken.Symbol),
		zap.Stringer("allowance", allowance),
		zap.Stringer("amountIn", params.AmountIn))
	return true, params.AmountIn, nil
}

func (r *Router) estimateGasForApproval(params pathprocessor.ProcessorInputParams, input []byte) (uint64, error) {
	ethClient, err := r.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		r.logger.Error("estimateGasForApproval: failed to get eth client",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.Error(err))
		return 0, err
	}

	gas, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	})
	if err != nil {
		r.logger.Error("estimateGasForApproval: EstimateGas failed",
			zap.Uint64("chainId", params.FromChain.ChainID),
			zap.String("token", params.FromToken.Symbol),
			zap.Stringer("from", params.FromAddr),
			zap.Error(err))
		return 0, err
	}
	r.logger.Debug("estimateGasForApproval: gas estimated",
		zap.Uint64("chainId", params.FromChain.ChainID),
		zap.String("token", params.FromToken.Symbol),
		zap.Uint64("gas", gas))
	return gas, nil
}

func (r *Router) calculateL1Fee(chainID uint64, data []byte) (*big.Int, error) {
	ethClient, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return CalculateL1Fee(chainID, data, ethClient)
}

func CalculateL1Fee(chainID uint64, data []byte, ethClient ethclient.EthClientInterface) (*big.Int, error) {
	oracleContractAddress, err := gaspriceproxy.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	proxyContract, err := gaspriceproxy.NewGaspriceproxy(oracleContractAddress, ethClient)
	if err != nil {
		return nil, err
	}

	callOpt := &bind.CallOpts{}

	return proxyContract.GetL1Fee(callOpt, data)
}

func (r *Router) getERC1155Balance(ctx context.Context, chainID uint64, token *tokentypes.Token, account common.Address) (*big.Int, error) {
	if token == nil || token.CollectibleTokenID == nil {
		r.logger.Error("getERC1155Balance: token or token ID is nil",
			zap.Uint64("chainId", chainID))
		return nil, errors.New("token or token ID is nil")
	}

	balances, err := r.collectiblesManager.FetchERC1155Balances(
		ctx,
		account,
		walletCommon.ChainID(chainID),
		token.Address,
		[]*bigint.BigInt{&bigint.BigInt{Int: token.CollectibleTokenID.ToInt()}},
	)
	if err != nil {
		r.logger.Error("getERC1155Balance: FetchERC1155Balances failed",
			zap.Uint64("chainId", chainID),
			zap.Stringer("tokenAddress", token.Address),
			zap.Stringer("account", account),
			zap.Error(err))
		return nil, err
	}

	if len(balances) != 1 || balances[0] == nil {
		r.logger.Error("getERC1155Balance: invalid balance response",
			zap.Uint64("chainId", chainID),
			zap.Int("responseLen", len(balances)))
		return nil, errors.New("invalid ERC1155 balance fetch response")
	}

	r.logger.Debug("getERC1155Balance: balance fetched",
		zap.Uint64("chainId", chainID),
		zap.Stringer("tokenAddress", token.Address),
		zap.Stringer("balance", balances[0].Int))
	return balances[0].Int, nil
}

func (r *Router) getBalance(ctx context.Context, chainID uint64, token *tokentypes.Token, account common.Address) (*big.Int, error) {
	balance, err := r.tokenBalancesFetcher.FetchSingle(ctx, chainID, token.Address, account)
	if err != nil {
		r.logger.Error("getBalance: FetchSingle failed",
			zap.Uint64("chainId", chainID),
			zap.String("token", token.Symbol),
			zap.Stringer("tokenAddress", token.Address),
			zap.Stringer("account", account),
			zap.Error(err))
		return nil, err
	}
	r.logger.Debug("getBalance: balance fetched",
		zap.Uint64("chainId", chainID),
		zap.String("token", token.Symbol),
		zap.Stringer("balance", balance))
	return balance, nil
}

func (r *Router) resolveSuggestedNonceForPath(ctx context.Context, path *routes.Path, address common.Address, usedNonces map[uint64]uint64) error {
	var nextNonce uint64
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		nextNonce = nonce + 1
		r.logger.Debug("resolveSuggestedNonceForPath: reusing local nonce counter",
			zap.Uint64("chainId", path.FromChain.ChainID),
			zap.Uint64("nextNonce", nextNonce))
	} else {
		nonce, err := r.transactor.NextNonce(ctx, r.rpcClient, path.FromChain.ChainID, cryptotypes.Address(address))
		if err != nil {
			r.logger.Error("resolveSuggestedNonceForPath: NextNonce failed",
				zap.Uint64("chainId", path.FromChain.ChainID),
				zap.Stringer("address", address),
				zap.Error(err))
			return err
		}
		nextNonce = nonce
		r.logger.Debug("resolveSuggestedNonceForPath: fetched nonce from chain",
			zap.Uint64("chainId", path.FromChain.ChainID),
			zap.Uint64("nextNonce", nextNonce))
	}

	usedNonces[path.FromChain.ChainID] = nextNonce
	if !path.ApprovalRequired {
		path.SuggestedTxNonce = (*hexutil.Uint64)(&nextNonce)
	} else {
		path.SuggestedApprovalTxNonce = (*hexutil.Uint64)(&nextNonce)
		txNonce := nextNonce + 1
		path.SuggestedTxNonce = (*hexutil.Uint64)(&txNonce)

		usedNonces[path.FromChain.ChainID] = txNonce
	}
	return nil
}

// applyCustomFields applies custom fields to the path based on fetched fees and used nonces
func (r *Router) applyCustomFields(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, usedNonces map[uint64]uint64) error {
	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()

	eipP1559EnabledChain := path.FromChain.EIP1559Enabled
	r.logger.Debug("applyCustomFields: applying fields",
		zap.String("processor", path.ProcessorName),
		zap.Uint64("fromChain", path.FromChain.ChainID),
		zap.Bool("eip1559Enabled", eipP1559EnabledChain),
		zap.Int("customParamsEntries", len(r.lastInputParams.PathTxCustomParams)))

	if err := r.setSuggestedFields(ctx, path, fetchedFees, usedNonces, eipP1559EnabledChain); err != nil {
		r.logger.Error("applyCustomFields: setSuggestedFields failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Error(err))
		return err
	}

	if err := r.setPathFields(path, fetchedFees); err != nil {
		r.logger.Error("applyCustomFields: setPathFields failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Error(err))
		return err
	}

	if len(r.lastInputParams.PathTxCustomParams) == 0 {
		r.logger.Debug("applyCustomFields: applying default fee modes",
			zap.String("processor", path.ProcessorName),
			zap.Int("globalFeeMode", int(r.lastInputParams.GasFeeMode)))
		return r.applyDefaultFeeModes(path, fetchedFees, eipP1559EnabledChain)
	}
	r.logger.Debug("applyCustomFields: applying custom fee modes",
		zap.String("processor", path.ProcessorName))
	return r.applyCustomFeeModes(ctx, path, fetchedFees, eipP1559EnabledChain)
}

// setSuggestedFields sets suggested fee fields
func (r *Router) setSuggestedFields(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, usedNonces map[uint64]uint64, eipP1559EnabledChain bool) error {
	if !eipP1559EnabledChain {
		path.SuggestedNonEIP1559Fees = fetchedFees.NonEIP1559Fees
	} else {
		path.SuggestedLevelsForMaxFeesPerGas = fetchedFees.MaxFeesLevels
		if fetchedFees.MaxPriorityFeeSuggestedBounds != nil {
			if fetchedFees.MaxPriorityFeeSuggestedBounds.Lower != nil {
				path.SuggestedMinPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeeSuggestedBounds.Lower)
			}
			if fetchedFees.MaxPriorityFeeSuggestedBounds.Upper != nil {
				path.SuggestedMaxPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeeSuggestedBounds.Upper)
			}
		}
	}

	return r.resolveSuggestedNonceForPath(ctx, path, r.lastInputParams.AddrFrom, usedNonces)
}

// setPathFields sets path fields
func (r *Router) setPathFields(path *routes.Path, fetchedFees *fees.SuggestedFees) error {
	if fetchedFees.CurrentBaseFee != nil {
		path.CurrentBaseFee = (*hexutil.Big)(fetchedFees.CurrentBaseFee)
	}

	path.TxGasAmount = path.SuggestedTxGasAmount
	path.ApprovalGasAmount = path.SuggestedApprovalGasAmount
	path.TxNonce = path.SuggestedTxNonce
	path.ApprovalTxNonce = path.SuggestedApprovalTxNonce

	return nil
}

// applyDefaultFeeModes applies default fee modes to the path
func (r *Router) applyDefaultFeeModes(path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	if !eipP1559EnabledChain {
		return r.applyDefaultNonEIP1559Fees(path, fetchedFees)
	}

	return r.applyDefaultEIP1559Fees(path, fetchedFees)
}

// applyDefaultNonEIP1559Fees applies default non-EIP1559 fees
func (r *Router) applyDefaultNonEIP1559Fees(path *routes.Path, fetchedFees *fees.SuggestedFees) error {
	if path.ApprovalRequired {
		path.ApprovalGasFeeMode = r.lastInputParams.GasFeeMode
		path.ApprovalGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
		path.ApprovalEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
	}

	path.TxGasFeeMode = r.lastInputParams.GasFeeMode
	path.TxGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
	path.TxEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
	return nil
}

// applyDefaultEIP1559Fees applies default EIP1559 fees
func (r *Router) applyDefaultEIP1559Fees(path *routes.Path, fetchedFees *fees.SuggestedFees) error {
	maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(r.lastInputParams.GasFeeMode)
	if err != nil {
		r.logger.Error("applyDefaultEIP1559Fees: FeeFor failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Int("feeMode", int(r.lastInputParams.GasFeeMode)),
			zap.Error(err))
		return err
	}

	if path.ApprovalRequired {
		path.ApprovalGasFeeMode = r.lastInputParams.GasFeeMode
		path.ApprovalMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
		path.ApprovalBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
		path.ApprovalPriorityFee = (*hexutil.Big)(priorityFee)
		path.ApprovalEstimatedTime = estimatedTime
	}

	path.TxGasFeeMode = r.lastInputParams.GasFeeMode
	path.TxMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
	path.TxBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.TxPriorityFee = (*hexutil.Big)(priorityFee)
	path.TxEstimatedTime = estimatedTime
	return nil
}

// applyCustomFeeModes applies custom fee modes to the path
func (r *Router) applyCustomFeeModes(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	if path.ApprovalRequired {
		if err := r.applyCustomApprovalFees(ctx, path, fetchedFees, eipP1559EnabledChain); err != nil {
			return err
		}
	}

	return r.applyCustomTxFees(ctx, path, fetchedFees, eipP1559EnabledChain)
}

// applyCustomApprovalFees applies custom fees for approval transaction
func (r *Router) applyCustomApprovalFees(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	approvalTxIdentityKey := path.TxIdentityKey(true)
	approvalTxCustomParams, ok := r.lastInputParams.PathTxCustomParams[approvalTxIdentityKey]
	if !ok {
		return nil
	}

	path.ApprovalGasFeeMode = approvalTxCustomParams.GasFeeMode
	if approvalTxCustomParams.GasFeeMode != fees.GasFeeCustom {
		return r.applyNonCustomApprovalFees(path, fetchedFees, eipP1559EnabledChain, approvalTxCustomParams)
	}
	return r.applyCustomApprovalFeeMode(ctx, path, fetchedFees, eipP1559EnabledChain, approvalTxCustomParams)
}

// applyNonCustomApprovalFees applies non-custom fees for approval transaction
func (r *Router) applyNonCustomApprovalFees(path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {
	if !eipP1559EnabledChain {
		path.ApprovalGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
		path.ApprovalEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
		return nil
	}

	maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(params.GasFeeMode)
	if err != nil {
		r.logger.Error("applyNonCustomApprovalFees: FeeFor failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Int("feeMode", int(params.GasFeeMode)),
			zap.Error(err))
		return err
	}

	path.ApprovalMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
	path.ApprovalBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.ApprovalPriorityFee = (*hexutil.Big)(priorityFee)
	path.ApprovalEstimatedTime = estimatedTime
	return nil
}

// applyCustomApprovalFeeMode applies custom fee mode for approval transaction
func (r *Router) applyCustomApprovalFeeMode(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {

	path.ApprovalTxNonce = (*hexutil.Uint64)(&params.Nonce)
	path.ApprovalGasAmount = params.GasAmount

	if !eipP1559EnabledChain {
		path.ApprovalGasPrice = params.GasPrice

		estimatedTime, err := r.feesManager.EstimatedTime(ctx, path.FromChain.ChainID, params.GasPrice.ToInt(), nil, nil)
		if err != nil {
			r.logger.Error("applyCustomApprovalFeeMode: EstimatedTime (non-EIP1559) failed",
				zap.String("processor", path.ProcessorName),
				zap.Uint64("fromChain", path.FromChain.ChainID),
				zap.Error(err))
			return err
		}
		path.ApprovalEstimatedTime = estimatedTime
		return nil
	}

	path.ApprovalMaxFeesPerGas = params.MaxFeesPerGas
	path.ApprovalBaseFee = (*hexutil.Big)(new(big.Int).Sub(params.MaxFeesPerGas.ToInt(), params.PriorityFee.ToInt()))
	path.ApprovalPriorityFee = params.PriorityFee

	estimatedTime, err := r.feesManager.EstimatedTime(ctx, path.FromChain.ChainID, nil, path.ApprovalMaxFeesPerGas.ToInt(), path.ApprovalPriorityFee.ToInt())
	if err != nil {
		r.logger.Error("applyCustomApprovalFeeMode: EstimatedTime (EIP1559) failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Error(err))
		return err
	}
	path.ApprovalEstimatedTime = estimatedTime
	return nil
}

// applyCustomTxFees applies custom fees for main transaction
func (r *Router) applyCustomTxFees(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	txIdentityKey := path.TxIdentityKey(false)
	txCustomParams, ok := r.lastInputParams.PathTxCustomParams[txIdentityKey]
	if !ok {
		return nil
	}

	path.TxGasFeeMode = txCustomParams.GasFeeMode
	if txCustomParams.GasFeeMode != fees.GasFeeCustom {
		return r.applyNonCustomTxFees(path, fetchedFees, eipP1559EnabledChain, txCustomParams)
	}
	return r.applyCustomTxFeeMode(ctx, path, fetchedFees, eipP1559EnabledChain, txCustomParams)
}

// applyNonCustomTxFees applies non-custom fees for main transaction
func (r *Router) applyNonCustomTxFees(path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {
	if !eipP1559EnabledChain {
		path.TxGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
		path.TxEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
		return nil
	}

	maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(params.GasFeeMode)
	if err != nil {
		r.logger.Error("applyNonCustomTxFees: FeeFor failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Int("feeMode", int(params.GasFeeMode)),
			zap.Error(err))
		return err
	}

	path.TxMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
	path.TxBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.TxPriorityFee = (*hexutil.Big)(priorityFee)
	path.TxEstimatedTime = estimatedTime
	return nil
}

// applyCustomTxFeeMode applies custom fee mode for main transaction
func (r *Router) applyCustomTxFeeMode(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {

	path.TxNonce = (*hexutil.Uint64)(&params.Nonce)
	path.TxGasAmount = params.GasAmount

	if !eipP1559EnabledChain {
		path.TxGasPrice = params.GasPrice
		estimatedTime, err := r.feesManager.EstimatedTime(ctx, path.FromChain.ChainID, path.TxGasPrice.ToInt(), nil, nil)
		if err != nil {
			r.logger.Error("applyCustomTxFeeMode: EstimatedTime (non-EIP1559) failed",
				zap.String("processor", path.ProcessorName),
				zap.Uint64("fromChain", path.FromChain.ChainID),
				zap.Error(err))
			return err
		}
		path.TxEstimatedTime = estimatedTime
		return nil
	}

	path.TxMaxFeesPerGas = params.MaxFeesPerGas
	path.TxBaseFee = (*hexutil.Big)(new(big.Int).Sub(params.MaxFeesPerGas.ToInt(), params.PriorityFee.ToInt()))
	path.TxPriorityFee = params.PriorityFee

	estimatedTime, err := r.feesManager.EstimatedTime(ctx, path.FromChain.ChainID, nil, path.TxMaxFeesPerGas.ToInt(), path.TxPriorityFee.ToInt())
	if err != nil {
		r.logger.Error("applyCustomTxFeeMode: EstimatedTime (EIP1559) failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Error(err))
		return err
	}
	path.TxEstimatedTime = estimatedTime
	return nil
}

func (r *Router) updatePathFields(path *routes.Path, fetchedFees *fees.SuggestedFees, noBaseFee bool, noPriorityFee bool) {
	path.FromChain.EIP1559Enabled = fetchedFees.EIP1559Enabled
	path.FromChain.NoBaseFee = noBaseFee
	path.FromChain.NoPriorityFee = noPriorityFee
}

func (r *Router) evaluateAndUpdatePathDetails(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees,
	usedNonces map[uint64]uint64, noBaseFee bool, noPriorityFee bool) (err error) {
	r.logger.Debug("evaluateAndUpdatePathDetails: starting",
		zap.String("processor", path.ProcessorName),
		zap.Uint64("fromChain", path.FromChain.ChainID),
		zap.String("token", path.FromToken.Symbol),
		zap.Bool("approvalRequired", path.ApprovalRequired),
		zap.Bool("noBaseFee", noBaseFee),
		zap.Bool("noPriorityFee", noPriorityFee))

	r.updatePathFields(path, fetchedFees, noBaseFee, noPriorityFee)

	l1TxFeeWei := big.NewInt(0)
	l1ApprovalFeeWei := big.NewInt(0)

	needL1Fee := path.FromChain.ChainID == walletCommon.OptimismMainnet ||
		path.FromChain.ChainID == walletCommon.OptimismSepolia ||
		path.FromChain.ChainID == walletCommon.BaseMainnet ||
		path.FromChain.ChainID == walletCommon.BaseSepolia

	if path.ApprovalRequired && needL1Fee {
		l1ApprovalFeeWei, err = r.calculateL1Fee(path.FromChain.ChainID, path.ApprovalPackedData)
		if err != nil {
			r.logger.Error("evaluateAndUpdatePathDetails: calculateL1Fee for approval failed",
				zap.String("processor", path.ProcessorName),
				zap.Uint64("fromChain", path.FromChain.ChainID),
				zap.Error(err))
			return err
		}
		r.logger.Debug("evaluateAndUpdatePathDetails: approval L1 fee calculated",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Stringer("l1ApprovalFeeWei", l1ApprovalFeeWei))
	}

	err = r.applyCustomFields(ctx, path, fetchedFees, usedNonces)
	if err != nil {
		r.logger.Error("evaluateAndUpdatePathDetails: applyCustomFields failed",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Error(err))
		return
	}

	if needL1Fee {
		l1TxFeeWei, err = r.calculateL1Fee(path.FromChain.ChainID, path.TxPackedData)
		if err != nil {
			r.logger.Error("evaluateAndUpdatePathDetails: calculateL1Fee for tx failed",
				zap.String("processor", path.ProcessorName),
				zap.Uint64("fromChain", path.FromChain.ChainID),
				zap.Error(err))
			return err
		}
		r.logger.Debug("evaluateAndUpdatePathDetails: tx L1 fee calculated",
			zap.String("processor", path.ProcessorName),
			zap.Uint64("fromChain", path.FromChain.ChainID),
			zap.Stringer("l1TxFeeWei", l1TxFeeWei))
	}

	// calculate ETH fees
	ethTotalFees := big.NewInt(0)
	var txFeeInWei *big.Int
	if !path.FromChain.EIP1559Enabled {
		txFeeInWei = new(big.Int).Mul(path.TxGasPrice.ToInt(), big.NewInt(int64(path.TxGasAmount)))
	} else {
		txFeeInWei = new(big.Int).Mul(path.TxMaxFeesPerGas.ToInt(), big.NewInt(int64(path.TxGasAmount)))
	}
	ethTotalFees.Add(ethTotalFees, txFeeInWei)
	ethTotalFees.Add(ethTotalFees, l1TxFeeWei)

	approvalFeeInWei := big.NewInt(0)
	if path.ApprovalRequired {
		if !path.FromChain.EIP1559Enabled {
			approvalFeeInWei.Mul(path.ApprovalGasPrice.ToInt(), big.NewInt(int64(path.ApprovalGasAmount)))
		} else {
			approvalFeeInWei.Mul(path.ApprovalMaxFeesPerGas.ToInt(), big.NewInt(int64(path.ApprovalGasAmount)))
		}
		ethTotalFees.Add(ethTotalFees, approvalFeeInWei)
		ethTotalFees.Add(ethTotalFees, l1ApprovalFeeWei)
	}

	// calculate required balances (bonder and token fees are already included in the amountIn by Hop bridge (once we include Celar we need to check how they handle the fees))
	requiredNativeBalance := big.NewInt(0)
	requiredTokenBalance := big.NewInt(0)

	if path.FromToken.IsNative() {
		requiredNativeBalance.Add(requiredNativeBalance, path.AmountIn.ToInt())
		if !path.SubtractFees {
			requiredNativeBalance.Add(requiredNativeBalance, ethTotalFees)
		}
	} else {
		requiredTokenBalance.Add(requiredTokenBalance, path.AmountIn.ToInt())
		requiredNativeBalance.Add(requiredNativeBalance, ethTotalFees)
	}

	// set the values
	path.TxFee = (*hexutil.Big)(txFeeInWei)
	path.TxL1Fee = (*hexutil.Big)(l1TxFeeWei)

	path.ApprovalFee = (*hexutil.Big)(approvalFeeInWei)
	path.ApprovalL1Fee = (*hexutil.Big)(l1ApprovalFeeWei)

	path.TxTotalFee = (*hexutil.Big)(ethTotalFees)

	path.RequiredTokenBalance = requiredTokenBalance
	path.RequiredNativeBalance = requiredNativeBalance

	r.logger.Debug("evaluateAndUpdatePathDetails: completed",
		zap.String("processor", path.ProcessorName),
		zap.Uint64("fromChain", path.FromChain.ChainID),
		zap.Stringer("txFee", txFeeInWei),
		zap.Stringer("approvalFee", approvalFeeInWei),
		zap.Stringer("ethTotalFees", ethTotalFees),
		zap.Stringer("requiredNativeBalance", requiredNativeBalance),
		zap.Stringer("requiredTokenBalance", requiredTokenBalance))
	return
}

func findToken(sendType sendtype.SendType, tokenManager TokenManager, collectiblesManager *collectibles.Manager,
	tokenKey string) *tokentypes.Token {
	if !sendType.IsCollectiblesTransfer() {
		token, err := tokenManager.GetTokenByKey(tokenKey)
		if err != nil {
			return nil
		}
		return token
	}

	if sendType.IsCommunityRelatedTransfer() {
		// TODO: optimize tokens to handle community tokens
		return nil
	}

	chainID, contractAddress, collectibleTokenID, success := tokentypes.ParseCollectibleKey(tokenKey)
	if !success {
		return nil
	}

	name := "" // need it for display only, transfer itself needs just the contract address and token id
	collectibleData, err := collectiblesManager.GetCacheCollectibleData(thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: walletCommon.ChainID(chainID),
			Address: contractAddress,
		},
		TokenID: &bigint.BigInt{Int: collectibleTokenID},
	})
	if err == nil {
		name = collectibleData.Name
	}

	return &tokentypes.Token{
		Token: &types.Token{
			Address:  contractAddress,
			Decimals: 0,
			ChainID:  chainID,
			Name:     name,
		},
		CollectibleTokenID: (*hexutil.Big)(collectibleTokenID),
	}
}

func (r *Router) fetchPrices(sendType sendtype.SendType, tokenKeys []string) (map[string]float64, error) {
	r.logger.Debug("fetchPrices: requesting prices",
		zap.Int("sendType", int(sendType)),
		zap.Strings("tokenKeys", tokenKeys))

	pricesMap, err := r.marketManager.GetOrFetchPrices(tokenKeys, []string{"USD"}, market.MaxAgeInSecondsForFresh)

	if err != nil {
		r.logger.Error("fetchPrices: GetOrFetchPrices failed",
			zap.Strings("tokenKeys", tokenKeys),
			zap.Error(err))
		return nil, err
	}
	prices := make(map[string]float64, 0)
	for tokenKey, pricePerCurrency := range pricesMap {
		prices[tokenKey] = pricePerCurrency["USD"].Price
	}
	if sendType.IsCollectiblesTransfer() {
		for _, tokenKey := range tokenKeys {
			prices[tokenKey] = 0
		}
	}
	r.logger.Debug("fetchPrices: prices fetched", zap.Int("count", len(prices)))
	return prices, nil
}

func (r *Router) TokenAvailableForBridgingViaHop(chainID uint64, address common.Address) bool {
	hopContracts := hop.GetTokenContractsAvailableOnChain(chainID)
	return slices.Contains(hopContracts, address)
}

// IsChainSupportedForSwapViaParaswap returns true if the chain is supported for swap via Paraswap, false otherwise.
func (r *Router) IsChainSupportedForSwapViaParaswap(chainID uint64) (bool, error) {
	paraswapClient := r.paraswapClientFactory(chainID)
	tokens, err := paraswapClient.FetchTokensList(context.Background())
	if err != nil {
		return false, err
	}

	return len(tokens) > 0, nil
}

// IsChainSupportedForSwapViaLiFi returns true if the chain is supported for swap via LI.FI, false otherwise.
func (r *Router) IsChainSupportedForSwapViaLiFi(chainID uint64) (bool, error) {
	lifiClient := r.lifiClientFactory(chainID)
	tokens, err := lifiClient.FetchTokensList(context.Background())
	if err != nil {
		return false, err
	}

	return len(tokens) > 0, nil
}
