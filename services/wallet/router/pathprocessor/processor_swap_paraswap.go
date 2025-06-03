package pathprocessor

import (
	"context"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty/paraswap"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

type SwapParaswapTxArgs struct {
	wallettypes.SendTxArgs
	ChainID            uint64  `json:"chainId"`
	ChainIDTo          uint64  `json:"chainIdTo"`
	TokenIDFrom        string  `json:"tokenIdFrom"`
	TokenIDTo          string  `json:"tokenIdTo"`
	SlippagePercentage float32 `json:"slippagePercentage"`
}

type SwapParaswapProcessor struct {
	rpcClient      *rpc.Client
	paraswapClient paraswap.ClientInterface
	tokenManager   *walletToken.Manager
	transactor     transactions.TransactorIface
	priceRoute     sync.Map // [fromChainName-toChainName-fromTokenSymbol-toTokenSymbol, paraswap.Route]
	transactions   sync.Map // [fromChainName-toChainName-fromTokenSymbol-toTokenSymbol, paraswap.Transaction]
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
	case walletCommon.BaseMainnet:
		return common.HexToAddress("0x107E3208A27e2A56D420fE6f8c5B88c821052f89"), partnerFeePcnt
	case walletCommon.BSCMainnet:
		return common.HexToAddress("0xEF693aCC26e7fb24B96056b33472D89d7dA5bAC9"), partnerFeePcnt
	}
	return common.Address{}, 0
}

func NewSwapParaswapProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, tokenManager *walletToken.Manager) *SwapParaswapProcessor {
	defaultChainID := walletCommon.EthereumMainnet
	partnerAddress, partnerFeePcnt := getPartnerAddressAndFeePcnt(defaultChainID)

	return &SwapParaswapProcessor{
		rpcClient: rpcClient,
		paraswapClient: paraswap.NewClientV5(
			defaultChainID,
			partnerID,
			partnerAddress,
			partnerFeePcnt,
		),
		tokenManager: tokenManager,
		transactor:   transactor,
		priceRoute:   sync.Map{},
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
	return createErrorResponse(pathProcessorCommon.ProcessorSwapParaswapName, err)
}

func (s *SwapParaswapProcessor) Name() string {
	return pathProcessorCommon.ProcessorSwapParaswapName
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

	if !params.TestsMode {
		fromToken := s.tokenManager.FindToken(params.FromChain, params.FromToken.Symbol)
		params.FromToken = fromToken

		toToken := s.tokenManager.FindToken(params.ToChain, params.ToToken.Symbol)
		params.ToToken = toToken
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

func (s *SwapParaswapProcessor) fetchAndStorePriceRoute(params ProcessorInputParams) (*paraswap.Route, error) {
	swapSide := paraswap.SellSide
	if params.AmountOut != nil && params.AmountOut.Cmp(walletCommon.ZeroBigIntValue()) > 0 {
		swapSide = paraswap.BuySide
	}

	// TODO: this is an extra check, we should remove it once we set the proper address for the native (ETH/BNB) token
	if params.FromToken.IsNative() {
		params.FromToken.Address = common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee") // ETH address across all chains that we support
	}
	if params.ToToken.IsNative() {
		params.ToToken.Address = common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee") // ETH address across all chains that we support
	}

	priceRoute, err := s.paraswapClient.FetchPriceRoute(context.Background(), params.FromToken.Address, params.FromToken.Decimals,
		params.ToToken.Address, params.ToToken.Decimals, params.AmountIn, params.FromAddr, params.ToAddr, swapSide)
	if err != nil {
		return nil, createSwapParaswapErrorResponse(err)
	}

	key := pathProcessorCommon.MakeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol, params.AmountIn)
	s.storePriceRoute(key, &priceRoute)
	return &priceRoute, nil
}

func (s *SwapParaswapProcessor) fetchAndStoreTransaction(params ProcessorInputParams) (*paraswap.Transaction, error) {
	slippageBP := uint(params.SlippagePercentage * 100) // convert to basis points

	key := pathProcessorCommon.MakeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol, params.AmountIn)
	priceRoute, err := s.getPriceRoute(key)
	if err != nil {
		return nil, createSwapParaswapErrorResponse(err)
	}

	tx, newPriceRoute, err := s.paraswapClient.BuildTransactionWithRetry(context.Background(), priceRoute.SrcTokenAddress, priceRoute.SrcTokenDecimals, priceRoute.SrcAmount.Int,
		priceRoute.DestTokenAddress, priceRoute.DestTokenDecimals, priceRoute.DestAmount.Int, slippageBP,
		params.FromAddr, params.ToAddr, priceRoute.RawPriceRoute, priceRoute.Side)
	if err != nil {
		return nil, createSwapParaswapErrorResponse(err)
	}

	if newPriceRoute != nil {
		s.storePriceRoute(key, newPriceRoute)
	}

	s.storeTransaction(key, &tx)
	return &tx, nil
}

func (s *SwapParaswapProcessor) fetchAndStoreTransactionFromSendTxArgs(sendArgs *wallettypes.SendTxArgs) (*paraswap.Transaction, error) {
	return s.fetchAndStoreTransaction(ProcessorInputParams{
		FromChain:          &params.Network{ChainID: sendArgs.FromChainID},
		ToChain:            &params.Network{ChainID: sendArgs.ToChainID},
		FromToken:          &tokenTypes.Token{Symbol: sendArgs.FromTokenID},
		ToToken:            &tokenTypes.Token{Symbol: sendArgs.ToTokenID},
		AmountIn:           sendArgs.ValueIn.ToInt(),
		FromAddr:           common.Address(sendArgs.From),
		ToAddr:             common.Address(*sendArgs.To),
		SlippagePercentage: sendArgs.SlippagePercentage,
	})
}

func (s *SwapParaswapProcessor) storePriceRoute(key string, priceRoute *paraswap.Route) {
	s.priceRoute.Store(key, priceRoute)
}

func (s *SwapParaswapProcessor) getPriceRoute(key string) (*paraswap.Route, error) {
	priceRouteIns, ok := s.priceRoute.Load(key)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	priceRoute, ok := priceRouteIns.(*paraswap.Route)
	if !ok {
		return nil, ErrPriceRouteNotFound
	}
	return priceRoute.Copy(), nil
}

func (s *SwapParaswapProcessor) storeTransaction(key string, tx *paraswap.Transaction) {
	s.transactions.Store(key, tx)
}

func (s *SwapParaswapProcessor) getTransaction(key string) (*paraswap.Transaction, error) {
	txIns, ok := s.transactions.Load(key)
	if !ok {
		return nil, ErrTransactionNotFound
	}
	tx, ok := txIns.(*paraswap.Transaction)
	if !ok {
		return nil, ErrTransactionNotFound
	}
	return tx, nil
}

func (s *SwapParaswapProcessor) GetContractAddress(params ProcessorInputParams) (address common.Address, err error) {
	priceRoute, err := s.fetchAndStorePriceRoute(params)
	if err != nil {
		return common.Address{}, createSwapParaswapErrorResponse(err)
	}
	return priceRoute.TokenTransferProxy, nil
}

func (s *SwapParaswapProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	if params.TestsMode {
		return []byte{}, nil
	}

	tx, err := s.fetchAndStoreTransaction(params)
	if err != nil {
		return []byte{}, createSwapParaswapErrorResponse(err)
	}
	return types.Hex2Bytes(tx.Data), nil
}

func (s *SwapParaswapProcessor) EstimateGas(params ProcessorInputParams, input []byte) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	value := big.NewInt(0)
	if params.FromToken.IsNative() {
		value = params.AmountIn
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, createENSRegisterProcessorErrorResponse(err)
	}

	increasedEstimation := float64(estimation) * pathProcessorCommon.IncreaseEstimatedGasFactor

	return uint64(increasedEstimation), nil
}

// TODO: remove this struct once mobile switches to the new approach
func (s *SwapParaswapProcessor) prepareTransaction(sendArgs *MultipathProcessorTxArgs) error {
	slippageBP := uint(sendArgs.SwapTx.SlippagePercentage * 100) // convert to basis points

	key := pathProcessorCommon.MakeKey(sendArgs.SwapTx.ChainID, sendArgs.SwapTx.ChainIDTo, sendArgs.SwapTx.TokenIDFrom, sendArgs.SwapTx.TokenIDTo, sendArgs.SwapTx.ValueIn.ToInt())
	priceRoute, err := s.getPriceRoute(key)
	if err != nil {
		return createSwapParaswapErrorResponse(err)
	}

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

func (s *SwapParaswapProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	err := s.prepareTransaction(sendArgs)
	if err != nil {
		return nil, 0, createSwapParaswapErrorResponse(err)
	}
	return s.transactor.ValidateAndBuildTransaction(sendArgs.ChainID, sendArgs.SwapTx.SendTxArgs, lastUsedNonce)
}

func (s *SwapParaswapProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	key := pathProcessorCommon.MakeKey(sendArgs.FromChainID, sendArgs.ToChainID, sendArgs.FromTokenID, sendArgs.ToTokenID, sendArgs.ValueIn.ToInt())
	tx, err := s.getTransaction(key)
	if err != nil {
		tx, err = s.fetchAndStoreTransactionFromSendTxArgs(sendArgs)
		if err != nil {
			return nil, 0, createSwapParaswapErrorResponse(err)
		}
	}
	value, ok := new(big.Int).SetString(tx.Value, 10)
	if !ok {
		return nil, 0, ErrConvertingAmountToBigInt
	}

	gas, err := strconv.ParseUint(tx.Gas, 10, 64)
	if err != nil {
		return nil, 0, createSwapParaswapErrorResponse(err)
	}

	gasPrice, ok := new(big.Int).SetString(tx.GasPrice, 10)
	if !ok {
		return nil, 0, ErrConvertingAmountToBigInt
	}

	sendArgs.FromChainID = tx.ChainID
	toAddr := types.HexToAddress(tx.To)
	sendArgs.From = types.HexToAddress(tx.From)
	sendArgs.To = &toAddr
	sendArgs.Value = (*hexutil.Big)(value)
	sendArgs.Gas = (*hexutil.Uint64)(&gas)
	sendArgs.GasPrice = (*hexutil.Big)(gasPrice)
	sendArgs.Data = types.Hex2Bytes(tx.Data)

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
	key := pathProcessorCommon.MakeKey(params.FromChain.ChainID, params.ToChain.ChainID, params.FromToken.Symbol, params.ToToken.Symbol, params.AmountIn)
	priceRoute, err := s.getPriceRoute(key)
	if err != nil {
		return nil, createSwapParaswapErrorResponse(err)
	}

	_, partnerFeePcnt := getPartnerAddressAndFeePcnt(params.FromChain.ChainID)
	destAmount, _ := calcReceivedAmountAndFee(priceRoute.DestAmount.Int, partnerFeePcnt)
	if destAmount.Cmp(walletCommon.ZeroBigIntValue()) == -1 {
		return walletCommon.ZeroBigIntValue(), nil
	}

	return destAmount, nil
}
