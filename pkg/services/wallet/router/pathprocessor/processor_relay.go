package pathprocessor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	"github.com/status-im/status-go/pkg/security"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/relay"
	walletToken "github.com/status-im/status-go/pkg/services/wallet/token"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

// RelayProcessor handles both same-chain swaps and cross-chain bridges through Relay (relay.link)
type RelayProcessor struct {
	ethClientGetter rpc.EthClientGetter
	relayClient     relay.ClientInterface
	tokenManager    *walletToken.Manager
	transactor      transactions.TransactorIface
	quotes          sync.Map // [fromTokenKey-toTokenKey-amountIn, *relay.Quote]
}

func NewRelayProcessor(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface, tokenManager *walletToken.Manager,
	apiKey security.SensitiveString, referrer string) *RelayProcessor {
	return &RelayProcessor{
		ethClientGetter: ethClientGetter,
		relayClient:     relay.NewClient(walletCommon.EthereumMainnet, referrer, apiKey.Reveal()),
		tokenManager:    tokenManager,
		transactor:      transactor,
		quotes:          sync.Map{},
	}
}

func createRelayErrorResponse(err error) error {
	var apiErr *relay.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "NO_SWAP_ROUTES_FOUND", "NO_INTERNAL_SWAP_ROUTES_FOUND", "UNSUPPORTED_ROUTE":
			return ErrNoRoutesFound
		case "INSUFFICIENT_LIQUIDITY", "INSUFFICIENT_POOL_LIQUIDITY":
			return ErrNotEnoughLiquidity
		case "NO_QUOTES", "GENERATE_SWAP_FAILED", "REVERSE_SWAP_FAILED":
			return ErrNoQuotesAvailable
		case "SWAP_IMPACT_TOO_HIGH":
			return ErrPriceImpactTooHigh
		case "SLIPPAGE", "TOO_LITTLE_RECEIVED":
			return ErrSlippageExceeded
		case "AMOUNT_TOO_LOW", "DEPOSITED_AMOUNT_TOO_LOW_TO_FILL":
			return ErrAmountTooLow
		case "AMOUNT_TOO_HIGH":
			return ErrAmountTooHigh
		}
	}
	return createErrorResponse(pathProcessorCommon.ProcessorRelayName, err)
}

func (s *RelayProcessor) Name() string {
	return pathProcessorCommon.ProcessorRelayName
}

func (s *RelayProcessor) Clear() {
	s.quotes = sync.Map{}
}

func isRelayBridge(params ProcessorInputParams) bool {
	return params.FromToken.ChainID != params.ToToken.ChainID
}

func (s *RelayProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromToken == nil || params.ToToken == nil {
		return false, ErrToAndFromTokensMustBeSet
	}

	if !isRelayBridge(params) && strings.EqualFold(params.FromToken.Address.Hex(), params.ToToken.Address.Hex()) {
		return false, ErrFromAndToTokensMustBeDifferent
	}

	if params.AmountOut != nil && params.AmountOut.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
		return false, nil
	}

	s.relayClient.SetChainID(params.FromToken.ChainID)

	return true, nil
}

func (s *RelayProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func getRelayFromAndToTokenAddresses(params ProcessorInputParams) (common.Address, common.Address) {
	fromTokenAddress := params.FromToken.Address
	toTokenAddress := params.ToToken.Address
	if params.FromToken.IsNative() {
		fromTokenAddress = walletCommon.ZeroAddress()
	}
	if params.ToToken.IsNative() {
		toTokenAddress = walletCommon.ZeroAddress()
	}
	return fromTokenAddress, toTokenAddress
}

// relayApproveStep returns the quote's ERC20 approval step, if any.
func relayApproveStep(quote *relay.Quote) *relay.Step {
	for i := range quote.Steps {
		step := &quote.Steps[i]
		if step.Kind == relay.StepKindTransaction && step.ID == relay.StepIDApprove {
			return step
		}
	}
	return nil
}

// relayMainStep returns the transaction that actually moves the funds (deposit/swap/send).
func relayMainStep(quote *relay.Quote) (*relay.Step, error) {
	for i := range quote.Steps {
		step := &quote.Steps[i]
		if step.Kind == relay.StepKindTransaction && step.ID != relay.StepIDApprove {
			return step, nil
		}
	}
	return nil, fmt.Errorf("unsupported relay quote: no transaction step")
}

// validateRelayQuote rejects quotes this processor can't execute: anything requiring an
// off-chain signature, more than one approval, an approval after the main step, more than
// one main transaction (only one is ever built and sent, so the quoted route would be left
// half executed), or a main step that doesn't run on the origin chain.
func validateRelayQuote(quote *relay.Quote, params ProcessorInputParams) error {
	approvals := 0
	mains := 0
	for _, step := range quote.Steps {
		if step.Kind != relay.StepKindTransaction {
			return fmt.Errorf("unsupported relay quote: %s step of kind %q", step.ID, step.Kind)
		}
		if len(step.Items) != 1 {
			return fmt.Errorf("unsupported relay quote: %s step has %d items", step.ID, len(step.Items))
		}
		if step.ID == relay.StepIDApprove {
			approvals++
			if mains > 0 {
				return fmt.Errorf("unsupported relay quote: approve step after the main step")
			}
			continue
		}
		mains++
	}
	if approvals > 1 {
		return fmt.Errorf("unsupported relay quote: %d approve steps", approvals)
	}
	if mains > 1 {
		return fmt.Errorf("unsupported relay quote: %d transaction steps, only one can be executed", mains)
	}

	main, err := relayMainStep(quote)
	if err != nil {
		return err
	}
	if main.Items[0].Data.ChainID != params.FromToken.ChainID {
		return fmt.Errorf("unsupported relay quote: main step on chain %d, expected %d", main.Items[0].Data.ChainID, params.FromToken.ChainID)
	}
	return nil
}

func (s *RelayProcessor) fetchAndStoreQuote(params ProcessorInputParams) (*relay.Quote, error) {
	fromTokenAddress, toTokenAddress := getRelayFromAndToTokenAddresses(params)

	quote, err := s.relayClient.FetchQuote(context.Background(), relay.QuoteParams{
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
		return nil, createRelayErrorResponse(err)
	}

	if err := validateRelayQuote(&quote, params); err != nil {
		return nil, createRelayErrorResponse(err)
	}

	key := pathProcessorCommon.MakeKey(params.FromToken.Key(), params.ToToken.Key(), params.AmountIn)
	s.quotes.Store(key, &quote)
	return &quote, nil
}

func (s *RelayProcessor) getQuote(key string) (*relay.Quote, error) {
	quoteIns, ok := s.quotes.Load(key)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	quote, ok := quoteIns.(*relay.Quote)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	return quote, nil
}

func (s *RelayProcessor) getOrFetchQuote(params ProcessorInputParams) (*relay.Quote, error) {
	key := pathProcessorCommon.MakeKey(params.FromToken.Key(), params.ToToken.Key(), params.AmountIn)
	if quote, err := s.getQuote(key); err == nil {
		return quote, nil
	}
	return s.fetchAndStoreQuote(params)
}

// GetContractAddress returns the spender the router must approve.
func (s *RelayProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	quote, err := s.getOrFetchQuote(params)
	if err != nil {
		return common.Address{}, err
	}

	if approve := relayApproveStep(quote); approve != nil {
		data := approve.Items[0].Data
		if !strings.EqualFold(data.To, params.FromToken.Address.Hex()) {
			return common.Address{}, createRelayErrorResponse(fmt.Errorf("unsupported relay quote: approve step targets %s, expected the token %s", data.To, params.FromToken.Address.Hex()))
		}
		spender, _, err := walletCommon.UnpackApprovalInputData(types.Hex2Bytes(data.Data))
		if err != nil {
			return common.Address{}, createRelayErrorResponse(err)
		}
		return spender, nil
	}

	main, err := relayMainStep(quote)
	if err != nil {
		return common.Address{}, createRelayErrorResponse(err)
	}
	to := common.HexToAddress(main.Items[0].Data.To)
	if !params.FromToken.IsNative() && to == params.FromToken.Address {
		return walletCommon.ZeroAddress(), nil
	}
	return to, nil
}

// RelayOwnRouter is the route value for Relay's own solver (no underlying tool).
const RelayOwnRouter = "relay"

// relayProviderTool returns the swap source Relay routes through (e.g. "0x", "kyberswap").
func relayProviderTool(quote *relay.Quote) string {
	for _, router := range []string{quote.Details.Route.Origin.Router, quote.Details.Route.Destination.Router} {
		if router != "" && router != RelayOwnRouter {
			return router
		}
	}
	return ""
}

// GetProviderTool returns the underlying swap source for the current quote, or "" if none.
func (s *RelayProcessor) GetProviderTool(params ProcessorInputParams) string {
	quote, err := s.getOrFetchQuote(params)
	if err != nil || quote == nil {
		return ""
	}
	return relayProviderTool(quote)
}

// GetRouteExecutionDuration returns Relay's estimated time in seconds once the tx is included.
func (s *RelayProcessor) GetRouteExecutionDuration(params ProcessorInputParams) uint {
	quote, err := s.getOrFetchQuote(params)
	if err != nil || quote == nil || quote.Details.TimeEstimate <= 0 {
		return 0
	}
	return uint(quote.Details.TimeEstimate)
}

func (s *RelayProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	key := pathProcessorCommon.MakeKey(params.FromToken.Key(), params.ToToken.Key(), params.AmountIn)
	quote, err := s.getQuote(key)
	if err != nil {
		return nil, createRelayErrorResponse(err)
	}
	amountOut := quote.Details.CurrencyOut.Amount
	if amountOut == nil || amountOut.Int == nil {
		return walletCommon.ZeroBigIntValue(), nil
	}
	return amountOut.Int, nil
}

func (s *RelayProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	quote, err := s.getOrFetchQuote(params)
	if err != nil {
		return []byte{}, err
	}
	main, err := relayMainStep(quote)
	if err != nil {
		return []byte{}, createRelayErrorResponse(err)
	}
	return types.Hex2Bytes(main.Items[0].Data.Data), nil
}

// relayStepValue is the native value the main step sends: the quoted one, else the amount for a native input and zero for an ERC20 one.
func relayStepValue(data relay.StepTxData, params ProcessorInputParams) *big.Int {
	if data.Value != nil && data.Value.Int != nil {
		return data.Value.Int
	}
	if params.FromToken.IsNative() {
		return params.AmountIn
	}
	return big.NewInt(0)
}

func relayStepGas(data relay.StepTxData) (uint64, bool) {
	if data.Gas == nil || data.Gas.Int == nil || !data.Gas.IsUint64() || data.Gas.Uint64() == 0 {
		return 0, false
	}
	return data.Gas.Uint64(), true
}

func (s *RelayProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	isNative := params.FromToken.IsNative()

	quote, err := s.getOrFetchQuote(params)
	if err != nil {
		return 0, err
	}
	main, err := relayMainStep(quote)
	if err != nil {
		return 0, createRelayErrorResponse(err)
	}
	data := main.Items[0].Data
	to := common.HexToAddress(data.To)

	ethClient, err := s.ethClientGetter.EthClient(params.FromToken.ChainID)
	if err != nil {
		return 0, createRelayErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &to,
		Value: relayStepValue(data, params),
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		// ERC20 estimation reverts before approval; fall back to the gas Relay quoted (when it does).
		if isNative {
			return 0, createRelayErrorResponse(err)
		}
		quotedGas, ok := relayStepGas(data)
		if !ok {
			return 0, createRelayErrorResponse(err)
		}
		estimation = quotedGas
	}

	gasFactor := pathProcessorCommon.IncreaseEstimatedGasFactor
	if isRelayBridge(params) {
		gasFactor = pathProcessorCommon.IncreaseEstimatedGasFactorForBridge
	}

	increasedEstimation := float64(estimation) * gasFactor

	return uint64(increasedEstimation), nil
}

func (s *RelayProcessor) fetchAndStoreQuoteFromSendTxArgs(sendArgs *wallettypes.SendTxArgs) (*relay.Quote, error) {
	return s.fetchAndStoreQuote(ProcessorInputParams{
		FromToken:          sendArgs.FromToken,
		ToToken:            sendArgs.ToToken,
		AmountIn:           sendArgs.ValueIn.ToInt(),
		FromAddr:           common.Address(sendArgs.From),
		ToAddr:             common.Address(*sendArgs.To),
		SlippagePercentage: sendArgs.SlippagePercentage,
	})
}

// BuildTransactionV2 maps the main step onto sendArgs. Fee fields (gas price / EIP-1559 caps) are left as chosen by
// the router and the user; the quoted gas is only used when the router didn't estimate one.
func (s *RelayProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	key := pathProcessorCommon.MakeKey(sendArgs.FromToken.Key(), sendArgs.ToToken.Key(), sendArgs.ValueIn.ToInt())
	quote, err := s.getQuote(key)
	if err != nil {
		quote, err = s.fetchAndStoreQuoteFromSendTxArgs(sendArgs)
		if err != nil {
			return nil, 0, err
		}
	}

	main, err := relayMainStep(quote)
	if err != nil {
		return nil, 0, createRelayErrorResponse(err)
	}
	data := main.Items[0].Data

	if data.Value == nil || data.Value.Int == nil {
		return nil, 0, ErrConvertingAmountToBigInt
	}

	sendArgs.FromChainID = data.ChainID
	toAddr := types.HexToAddress(data.To)
	sendArgs.To = &toAddr
	sendArgs.Value = (*hexutil.Big)(data.Value.Int)
	sendArgs.Data = types.Hex2Bytes(data.Data)

	if sendArgs.Gas == nil || *sendArgs.Gas == 0 {
		if gas, ok := relayStepGas(data); ok {
			sendArgs.Gas = (*hexutil.Uint64)(&gas)
		}
	}

	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}
