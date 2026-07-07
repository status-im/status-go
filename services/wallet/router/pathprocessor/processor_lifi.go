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
	quotes          sync.Map // [fromTokenKey-toTokenKey-amountIn, *lifi.Quote]
}

func NewLiFiProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface, tokenManager *walletToken.Manager) *LiFiProcessor {
	return &LiFiProcessor{
		ethClientGetter: ethClientGetter,
		lifiClient:      lifi.NewClient(walletCommon.EthereumMainnet, lifi.Integrator, ""),
		tokenManager:    tokenManager,
		transactor:      transactor,
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

	return uint64(increasedEstimation), nil
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

	sendArgs.FromChainID = txReq.ChainID
	toAddr := types2.HexToAddress(txReq.To)
	sendArgs.From = types2.HexToAddress(txReq.From)
	sendArgs.To = &toAddr
	sendArgs.Value = (*hexutil.Big)(value)
	sendArgs.Gas = (*hexutil.Uint64)(&gas)
	sendArgs.GasPrice = (*hexutil.Big)(gasPrice)
	sendArgs.Data = types2.Hex2Bytes(txReq.Data)

	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}
