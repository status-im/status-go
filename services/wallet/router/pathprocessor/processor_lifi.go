package pathprocessor

import (
	"context"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/permit2"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/lifi"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

// LiFiProcessor handles both same-chain swaps and cross-chain bridges through the
// LI.FI aggregator: same /quote flow for both, the from/to chain (derived from the
// tokens) is the only difference.
type LiFiProcessor struct {
	ethClientGetter rpc.EthClientGetter
	lifiClient      lifi.ClientInterface
	tokenManager    *walletToken.Manager
	transactor      transactions.TransactorIface
	permitResolver  *permit2.Resolver
	quotes          sync.Map // [fromTokenKey-toTokenKey-amountIn, *lifi.Quote]
}

func NewLiFiProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface, tokenManager *walletToken.Manager) *LiFiProcessor {
	return &LiFiProcessor{
		ethClientGetter: ethClientGetter,
		lifiClient:      lifi.NewClient(walletCommon.EthereumMainnet, lifi.Integrator, ""),
		tokenManager:    tokenManager,
		transactor:      transactor,
		permitResolver:  permit2.NewResolver(ethClientGetter),
		quotes:          sync.Map{},
	}
}

func createLiFiErrorResponse(err error) error {
	switch {
	case strings.Contains(err.Error(), "No available quotes"):
		return ErrNotEnoughLiquidity
	case strings.Contains(err.Error(), "price impact"):
		return ErrPriceImpactTooHigh
	}
	return createErrorResponse(pathProcessorCommon.ProcessorLiFiName, err)
}

func (s *LiFiProcessor) Name() string {
	return pathProcessorCommon.ProcessorLiFiName
}

func (s *LiFiProcessor) Clear() {
	s.quotes = sync.Map{}
}

// isLiFiBridge reports whether the operation crosses chains (bridge vs swap).
func isLiFiBridge(params ProcessorInputParams) bool {
	return params.FromToken.ChainID != params.ToToken.ChainID
}

func (s *LiFiProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromToken == nil || params.ToToken == nil {
		return false, ErrToAndFromTokensMustBeSet
	}

	// For a same-chain swap the two tokens must differ; across chains it's a bridge.
	if !isLiFiBridge(params) && strings.EqualFold(params.FromToken.Address.Hex(), params.ToToken.Address.Hex()) {
		return false, ErrFromAndToTokensMustBeDifferent
	}

	if params.AmountOut != nil && params.AmountOut.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
		return false, nil
	}

	s.lifiClient.SetChainID(params.FromToken.ChainID)

	return true, nil
}

func (s *LiFiProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func getLiFiFromAndToTokenAddresses(params ProcessorInputParams) (common.Address, common.Address) {
	fromTokenAddress := params.FromToken.Address
	toTokenAddress := params.ToToken.Address
	if params.FromToken.IsNative() {
		fromTokenAddress = lifi.NativeTokenAddress
	}
	if params.ToToken.IsNative() {
		toTokenAddress = lifi.NativeTokenAddress
	}
	return fromTokenAddress, toTokenAddress
}

func (s *LiFiProcessor) fetchAndStoreQuote(params ProcessorInputParams) (*lifi.Quote, error) {
	fromTokenAddress, toTokenAddress := getLiFiFromAndToTokenAddresses(params)

	quote, err := s.lifiClient.FetchQuote(context.Background(), lifi.QuoteParams{
		FromChainID:        params.FromToken.ChainID,
		ToChainID:          params.ToToken.ChainID,
		FromToken:          fromTokenAddress,
		ToToken:            toTokenAddress,
		FromAddress:        params.FromAddr,
		ToAddress:          params.ToAddr,
		AmountIn:           params.AmountIn,
		SlippagePercentage: params.SlippagePercentage,
	})
	if err != nil {
		return nil, createLiFiErrorResponse(err)
	}

	key := pathProcessorCommon.MakeKey(params.FromToken.Key(), params.ToToken.Key(), params.AmountIn)
	s.quotes.Store(key, &quote)
	return &quote, nil
}

func (s *LiFiProcessor) getQuote(key string) (*lifi.Quote, error) {
	quoteIns, ok := s.quotes.Load(key)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	quote, ok := quoteIns.(*lifi.Quote)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	return quote, nil
}

func (s *LiFiProcessor) getOrFetchQuote(params ProcessorInputParams) (*lifi.Quote, error) {
	key := pathProcessorCommon.MakeKey(params.FromToken.Key(), params.ToToken.Key(), params.AmountIn)
	if quote, err := s.getQuote(key); err == nil {
		return quote, nil
	}
	return s.fetchAndStoreQuote(params)
}

func (s *LiFiProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	quote, err := s.fetchAndStoreQuote(params)
	if err != nil {
		return common.Address{}, err
	}
	return quote.Estimate.ApprovalAddress, nil
}

// ResolvePermit reports whether this swap can pull its tokens with an off-chain permit.
// A nil plan and nil error means the regular approve-then-swap flow applies: native token,
// chain not enabled or without a Permit2Proxy, or unreadable LI.FI chain metadata. None of
// those fail the route.
func (s *LiFiProcessor) ResolvePermit(ctx context.Context, params ProcessorInputParams) (*permit2.Plan, error) {
	if params.FromToken == nil || params.FromToken.IsNative() || params.TestsMode {
		return nil, nil
	}

	chainID := params.FromToken.ChainID
	deployment, enabled := permit2.DeploymentForChain(chainID)
	if !enabled {
		return nil, nil
	}

	// LI.FI has to publish the same contracts we pinned. A mismatch means a redeploy left
	// our pins stale, or the response isn't LI.FI's: either way the swap takes the regular
	// approve-then-swap flow rather than the pinned addresses being used anyway.
	chainInfo, err := s.lifiClient.GetChainInfo(ctx, chainID)
	if err != nil || chainInfo == nil ||
		!permit2.TrustedAddresses(chainID, chainInfo.Permit2, chainInfo.Permit2Proxy) {
		return nil, nil //nolint:nilerr // no permit support is a valid outcome, not a failure
	}

	return s.permitResolver.Resolve(ctx, permit2.ResolveParams{
		ChainID:      chainID,
		Owner:        params.FromAddr,
		Token:        params.FromToken.Address,
		Amount:       params.AmountIn,
		Permit2:      deployment.Permit2,
		Permit2Proxy: deployment.Proxy,
	})
}

// GetProviderTool returns the underlying tool/exchange LI.FI routes through (e.g. "1inch")
// for the current quote, or an empty string if it can't be resolved.
func (s *LiFiProcessor) GetProviderTool(params ProcessorInputParams) string {
	quote, err := s.getOrFetchQuote(params)
	if err != nil || quote == nil {
		return ""
	}
	return quote.Tool
}

func (s *LiFiProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	key := pathProcessorCommon.MakeKey(params.FromToken.Key(), params.ToToken.Key(), params.AmountIn)
	quote, err := s.getQuote(key)
	if err != nil {
		return nil, createLiFiErrorResponse(err)
	}
	if quote.Estimate.ToAmount == nil || quote.Estimate.ToAmount.Int == nil {
		return walletCommon.ZeroBigIntValue(), nil
	}
	return quote.Estimate.ToAmount.Int, nil
}

func (s *LiFiProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	if params.TestsMode {
		return []byte{}, nil
	}

	quote, err := s.fetchAndStoreQuote(params)
	if err != nil {
		return []byte{}, err
	}
	return types2.Hex2Bytes(quote.TransactionRequest.Data), nil
}

func (s *LiFiProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	isNative := params.FromToken.IsNative()
	value := big.NewInt(0)
	if isNative {
		value = params.AmountIn
	}

	quote, err := s.getOrFetchQuote(params)
	if err != nil {
		return 0, createLiFiErrorResponse(err)
	}
	contractAddress := quote.Estimate.ApprovalAddress

	ethClient, err := s.ethClientGetter.EthClient(params.FromToken.ChainID)
	if err != nil {
		return 0, createLiFiErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		// ERC20 estimation reverts before approval; fall back to the quote's gas limit.
		if isNative {
			return 0, createLiFiErrorResponse(err)
		}
		quotedGas, decErr := hexutil.DecodeUint64(quote.TransactionRequest.GasLimit)
		if decErr != nil || quotedGas == 0 {
			return 0, createLiFiErrorResponse(err)
		}
		estimation = quotedGas
	}

	gasFactor := pathProcessorCommon.IncreaseEstimatedGasFactor
	if isLiFiBridge(params) {
		gasFactor = pathProcessorCommon.IncreaseEstimatedGasFactorForBridge
	}

	increasedEstimation := float64(estimation) * gasFactor

	// The permit path can't be estimated on-chain: permitTransferFrom reverts until the
	// signature exists, which is after the route is priced. Use a fixed overhead instead so
	// the fee shown to the user matches the gas limit on the tx.
	if params.PermitPlan != nil && params.PermitPlan.Details != nil {
		increasedEstimation += float64(permitGasOverhead(params.PermitPlan.Details.Type))
	}

	return uint64(increasedEstimation), nil
}

// estimatePermitGas prices the wrapped proxy call. Once the permit is signed the tx can be
// simulated for real, which is the only reliable number: the quote's gas limit was priced
// for the user calling the diamond directly and covers none of the permit, the transfer
// into the proxy or the diamond approval. Falls back to the quote plus the fixed overhead,
// and never returns less than that.
func (s *LiFiProcessor) estimatePermitGas(chainID uint64, from, to types2.Address, data []byte,
	quotedGas uint64, permitType permit2.Type) uint64 {
	fallback := quotedGas + permitGasOverhead(permitType)

	ethClient, err := s.ethClientGetter.EthClient(chainID)
	if err != nil {
		return fallback
	}

	toAddress := common.Address(to)
	estimation, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  common.Address(from),
		To:    &toAddress,
		Value: big.NewInt(0),
		Data:  data,
	})
	if err != nil {
		return fallback
	}

	withMargin := uint64(float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor)
	if withMargin < fallback {
		return fallback
	}
	return withMargin
}

// permitGasOverhead is the extra gas the Permit2Proxy call costs on top of the swap: the
// permit, the token transfer into the proxy and the diamond approval.
//
// Measured on a mainnet USDC swap: permit 67,712 + transferFrom 16,149 + allowance 3,448,
// so ~87k before the diamond is reached. The values below carry margin on top because the
// 63/64 rule amplifies any shortfall at every level of the swap's call stack: an
// underestimate runs the deepest call out of gas rather than failing cleanly. The Permit2
// number is still a guess, permitTransferFrom hasn't been measured against a real tx yet.
func permitGasOverhead(permitType permit2.Type) uint64 {
	switch permitType {
	case permit2.TypeEIP2612:
		return 120_000
	case permit2.TypePermit2:
		return 130_000
	}
	return 0
}

func (s *LiFiProcessor) fetchAndStoreQuoteFromSendTxArgs(sendArgs *wallettypes.SendTxArgs) (*lifi.Quote, error) {
	return s.fetchAndStoreQuote(ProcessorInputParams{
		FromToken:          sendArgs.FromToken,
		ToToken:            sendArgs.ToToken,
		AmountIn:           sendArgs.ValueIn.ToInt(),
		FromAddr:           common.Address(sendArgs.From),
		ToAddr:             common.Address(*sendArgs.To),
		SlippagePercentage: sendArgs.SlippagePercentage,
	})
}

func (s *LiFiProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	key := pathProcessorCommon.MakeKey(sendArgs.FromToken.Key(), sendArgs.ToToken.Key(), sendArgs.ValueIn.ToInt())
	quote, err := s.getQuote(key)
	if err != nil {
		quote, err = s.fetchAndStoreQuoteFromSendTxArgs(sendArgs)
		if err != nil {
			return nil, 0, createLiFiErrorResponse(err)
		}
	}

	txReq := quote.TransactionRequest

	value, err := hexutil.DecodeBig(txReq.Value)
	if err != nil {
		return nil, 0, ErrConvertingAmountToBigInt
	}

	gas, err := hexutil.DecodeUint64(txReq.GasLimit)
	if err != nil {
		return nil, 0, createLiFiErrorResponse(err)
	}

	gasPrice, err := hexutil.DecodeBig(txReq.GasPrice)
	if err != nil {
		return nil, 0, ErrConvertingAmountToBigInt
	}

	toAddr := types2.HexToAddress(txReq.To)
	data := types2.Hex2Bytes(txReq.Data)

	// With a permit, the transaction goes to the Permit2Proxy rather than straight to the
	// LI.FI diamond: the proxy pulls the tokens using the user's signature and forwards
	// this calldata on. That is what removes the separate approval transaction.
	if details := sendArgs.PermitDetails; details != nil {
		// The quote was refetched after the route was built; refuse if it moved the target.
		// Falling back is no longer possible here, the route already told the client no
		// approval was needed.
		if err = permit2.ValidateSwapTarget(txReq.ChainID, details, common.Address(toAddr)); err != nil {
			return nil, 0, createLiFiErrorResponse(err)
		}

		data, err = permit2.PackSwapCalldata(details, data)
		if err != nil {
			return nil, 0, createLiFiErrorResponse(err)
		}
		toAddr = types2.Address(details.Spender)
		gas = s.estimatePermitGas(txReq.ChainID, types2.HexToAddress(txReq.From), toAddr, data, gas, details.Type)
	}

	sendArgs.FromChainID = txReq.ChainID
	sendArgs.From = types2.HexToAddress(txReq.From)
	sendArgs.To = &toAddr
	sendArgs.Value = (*hexutil.Big)(value)
	sendArgs.Gas = (*hexutil.Uint64)(&gas)
	sendArgs.GasPrice = (*hexutil.Big)(gasPrice)
	sendArgs.Data = data

	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}
