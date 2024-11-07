package pathprocessor

import (
	"context"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/paraswap"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type SwapParaswapTxArgs struct {
	transactions.SendTxArgs
	ChainID            uint64  `json:"chainId"`
	ChainIDTo          uint64  `json:"chainIdTo"`
	TokenIDFrom        string  `json:"tokenIdFrom"`
	TokenIDTo          string  `json:"tokenIdTo"`
	SlippagePercentage float32 `json:"slippagePercentage"`
}

type SwapParaswapProcessor struct {
	paraswapClient paraswap.ClientInterface
	transactor     transactions.TransactorIface
	priceRoute     sync.Map // [fromChainName-toChainName-fromTokenSymbol-toTokenSymbol, paraswap.Route]
}

const (
	partnerID = "status.app"
)

func getPartnerAddressAndFeePcnt(chainID uint64) (common.Address, float64) {
	const partnerFeePcnt = 0.7

	switch chainID {
	case walletCommon.EthereumMainnet:
		return common.HexToAddress("0xd9abc564bfabefa88a6C2723d78124579600F568"), partnerFeePcnt
	case walletCommon.OptimismMainnet:
		return common.HexToAddress("0xE9B59dC0b30cd4646430c25de0111D651c395775"), partnerFeePcnt
	case walletCommon.ArbitrumMainnet:
		return common.HexToAddress("0x9a8278e856C0B191B9daa2d7DD1f7B28268E4DA2"), partnerFeePcnt
	}
	return common.Address{}, 0
}

func NewSwapParaswapProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, tokenManager *walletToken.Manager) *SwapParaswapProcessor {
	defaultChainID := walletCommon.EthereumMainnet
	partnerAddress, partnerFeePcnt := getPartnerAddressAndFeePcnt(defaultChainID)

	return &SwapParaswapProcessor{
		paraswapClient: paraswap.NewClientV5(
			defaultChainID,
			partnerID,
			partnerAddress,
			partnerFeePcnt,
		),
		transactor: transactor,
		priceRoute: sync.Map{},
	}
}

func createSwapParaswapErrorResponse(err error) error {
	switch err.Error() {
	case "Price Timeout":
		return ErrPriceTimeout
	case "No routes found with enough liquidity":
		return ErrNotEnoughLiquidity
	case "ESTIMATED_LOSS_GREATER_THAN_MAX_IMPACT":
		return ErrPriceImpactTooHigh
	}
	return createErrorResponse(ProcessorSwapParaswapName, err)
}

func (s *SwapParaswapProcessor) Name() string {
	return ProcessorSwapParaswapName
}

func (s *SwapParaswapProcessor) Clear() {
	s.priceRoute = sync.Map{}
}

func (s *SwapParaswapProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromChain == nil || params.ToChain == nil {
		return false, ErrNoChainSet
	}
	if params.FromToken == nil || params.ToToken == nil {
		return false, ErrToAndFromTokensMustBeSet
	}

	if params.FromChain.ChainID != params.ToChain.ChainID {
		return false, ErrFromAndToChainsMustBeSame
	}

	if params.FromToken.Symbol == params.ToToken.Symbol {
		return false, ErrFromAndToTokensMustBeDifferent
	}

	chainID := params.FromChain.ChainID
	partnerAddress, partnerFeePcnt := getPartnerAddressAndFeePcnt(chainID)
	s.paraswapClient.SetChainID(chainID)
	s.paraswapClient.SetPartnerAddress(partnerAddress)
	s.paraswapClient.SetPartnerFeePcnt(partnerFeePcnt)

	searchForToken := params.FromToken.Address == walletCommon.ZeroAddress()
	searchForToToken := params.ToToken.Address == walletCommon.ZeroAddress()
	if searchForToToken || searchForToken {
		tokensList, err := s.paraswapClient.FetchTokensList(context.Background())
		if err != nil {
			return false, createSwapParaswapErrorResponse(err)
		}

		for _, t := range tokensList {
			if searchForToken && t.Symbol == params.FromToken.Symbol {
				params.FromToken.Address = common.HexToAddress(t.Address)
				params.FromToken.Decimals = t.Decimals
				if !searchForToToken {
					break
				}
			}

			if searchForToToken && t.Symbol == params.ToToken.Symbol {
				params.ToToken.Address = common.HexToAddress(t.Address)
				params.ToToken.Decimals = t.Decimals
				if !searchForToken {
					break
				}
			}
		}
	}

	if params.FromToken.Address == walletCommon.ZeroAddress() || params.ToToken.Address == walletCommon.ZeroAddress() {
		return false, ErrCannotResolveTokens
	}

	return true, nil
}

func calcReceivedAmountAndFee(baseDestAmount *big.Int, feePcnt float64) (destAmount *big.Int, destFee *big.Int) {
	destAmount = new(big.Int).Set(baseDestAmount)
	destFee = new(big.Int).SetUint64(0)

	if feePcnt > 0 {
		baseDestAmountFloat := new(big.Float).SetInt(baseDestAmount)
		feePcntFloat := big.NewFloat(feePcnt / 100.0)

		destFeeFloat := new(big.Float).Set(baseDestAmountFloat)
		destFeeFloat = destFeeFloat.Mul(destFeeFloat, feePcntFloat)
		destFeeFloat.Int(destFee)

		destAmount = destAmount.Sub(destAmount, destFee)
	}
	return
}

func (s *SwapParaswapProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	return walletCommon.ZeroBigIntValue(), walletCommon.ZeroBigIntValue(), nil
}

func (s *SwapParaswapProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	// not sure what we can do here since we're using the api to build the transaction
	return []byte{}, nil
}

func (s *SwapParaswapProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	swapSide := paraswap.SellSide
	if params.AmountOut != nil && params.AmountOut.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
		swapSide = paraswap.BuySide
	}

	priceRoute, err := s.paraswapClient.FetchPriceRoute(context.Background(), params.FromToken.Address, params.FromToken.Decimals,
		params.ToToken.Address, params.ToToken.Decimals, params.AmountIn, params.FromAddr, params.ToAddr, swapSide)
	if err != nil {
		return 0, createSwapParaswapErrorResponse(err)
	}

	key := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol, params.AmountIn)
	s.priceRoute.Store(key, &priceRoute)

	return priceRoute.GasCost.Uint64(), nil
}

func (s *SwapParaswapProcessor) GetContractAddress(params ProcessorInputParams) (address common.Address, err error) {
	key := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol, params.AmountIn)
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		err = ErrPriceRouteNotFound
		return
	}
	priceRoute := priceRouteIns.(*paraswap.Route)

	return priceRoute.TokenTransferProxy, nil
}

// TODO: remove this struct once mobile switches to the new approach
func (s *SwapParaswapProcessor) prepareTransaction(sendArgs *MultipathProcessorTxArgs) error {
	slippageBP := uint(sendArgs.SwapTx.SlippagePercentage * 100) // convert to basis points

	key := makeKey(sendArgs.SwapTx.ChainID, sendArgs.SwapTx.ChainIDTo, sendArgs.SwapTx.TokenIDFrom, sendArgs.SwapTx.TokenIDTo, sendArgs.SwapTx.ValueIn.ToInt())
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		return ErrPriceRouteNotFound
	}
	priceRoute := priceRouteIns.(*paraswap.Route)

	tx, err := s.paraswapClient.BuildTransaction(context.Background(), priceRoute.SrcTokenAddress, priceRoute.SrcTokenDecimals, priceRoute.SrcAmount.Int,
		priceRoute.DestTokenAddress, priceRoute.DestTokenDecimals, priceRoute.DestAmount.Int, slippageBP,
		common.Address(sendArgs.SwapTx.From), common.Address(*sendArgs.SwapTx.To),
		priceRoute.RawPriceRoute, priceRoute.Side)
	if err != nil {
		return createSwapParaswapErrorResponse(err)
	}

	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		return ErrConvertingAmountToBigInt
	}

	gas, err := strconv.ParseUint(tx.Gas, 10, 64)
	if err != nil {
		return createSwapParaswapErrorResponse(err)
	}

	gasPrice, ok := new(big.Int).SetString(tx.GasPrice, 10)
	if !ok {
		return ErrConvertingAmountToBigInt
	}

	sendArgs.ChainID = tx.ChainID
	sendArgs.SwapTx.ChainID = tx.ChainID
	toAddr := types.HexToAddress(tx.To)
	sendArgs.SwapTx.From = types.HexToAddress(tx.From)
	sendArgs.SwapTx.To = &toAddr
	sendArgs.SwapTx.Value = (*hexutil.Big)(value)
	sendArgs.SwapTx.Gas = (*hexutil.Uint64)(&gas)
	sendArgs.SwapTx.GasPrice = (*hexutil.Big)(gasPrice)
	sendArgs.SwapTx.Data = types.Hex2Bytes(tx.Data)

	return nil
}

func (s *SwapParaswapProcessor) prepareTransactionV2(sendArgs *transactions.SendTxArgs) error {
	slippageBP := uint(sendArgs.SlippagePercentage * 100) // convert to basis points

	key := makeKey(sendArgs.FromChainID, sendArgs.ToChainID, sendArgs.FromTokenID, sendArgs.ToTokenID, sendArgs.ValueIn.ToInt())
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		return ErrPriceRouteNotFound
	}
	priceRoute := priceRouteIns.(*paraswap.Route)

	tx, err := s.paraswapClient.BuildTransaction(context.Background(), priceRoute.SrcTokenAddress, priceRoute.SrcTokenDecimals, priceRoute.SrcAmount.Int,
		priceRoute.DestTokenAddress, priceRoute.DestTokenDecimals, priceRoute.DestAmount.Int, slippageBP,
		common.Address(sendArgs.From), common.Address(*sendArgs.To),
		priceRoute.RawPriceRoute, priceRoute.Side)
	if err != nil {
		return createSwapParaswapErrorResponse(err)
	}

	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		return ErrConvertingAmountToBigInt
	}

	gas, err := strconv.ParseUint(tx.Gas, 10, 64)
	if err != nil {
		return createSwapParaswapErrorResponse(err)
	}

	gasPrice, ok := new(big.Int).SetString(tx.GasPrice, 10)
	if !ok {
		return ErrConvertingAmountToBigInt
	}

	sendArgs.FromChainID = tx.ChainID
	toAddr := types.HexToAddress(tx.To)
	sendArgs.From = types.HexToAddress(tx.From)
	sendArgs.To = &toAddr
	sendArgs.Value = (*hexutil.Big)(value)
	sendArgs.Gas = (*hexutil.Uint64)(&gas)
	sendArgs.GasPrice = (*hexutil.Big)(gasPrice)
	sendArgs.Data = types.Hex2Bytes(tx.Data)

	return nil
}

func (s *SwapParaswapProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	err := s.prepareTransaction(sendArgs)
	if err != nil {
		return nil, 0, createSwapParaswapErrorResponse(err)
	}
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, sendArgs.SwapTx.SendTxArgs, lastUsedNonce)
}

func (s *SwapParaswapProcessor) BuildTransactionV2(sendArgs *transactions.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	err := s.prepareTransactionV2(sendArgs)
	if err != nil {
		return nil, 0, createSwapParaswapErrorResponse(err)
	}
	return s.transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (s *SwapParaswapProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (types.Hash, uint64, error) {
	err := s.prepareTransaction(sendArgs)
	if err != nil {
		return types.Hash{}, 0, createSwapParaswapErrorResponse(err)
	}

	return s.transactor.SendTransactionWithChainID(sendArgs.ChainID, sendArgs.SwapTx.SendTxArgs, lastUsedNonce, verifiedAccount)
}

func (s *SwapParaswapProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	key := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol, params.AmountIn)
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	priceRoute := priceRouteIns.(*paraswap.Route)

	_, partnerFeePcnt := getPartnerAddressAndFeePcnt(params.FromChain.ChainID)
	destAmount, _ := calcReceivedAmountAndFee(priceRoute.DestAmount.Int, partnerFeePcnt)

	return destAmount, nil
}
