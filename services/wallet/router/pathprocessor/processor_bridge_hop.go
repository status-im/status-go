package pathprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	netUrl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/contracts/hop"
	hopL1CctpImplementation "github.com/status-im/status-go/contracts/hop/l1Contracts/l1CctpImplementation"
	hopL1Erc20Bridge "github.com/status-im/status-go/contracts/hop/l1Contracts/l1Erc20Bridge"
	hopL1EthBridge "github.com/status-im/status-go/contracts/hop/l1Contracts/l1EthBridge"
	hopL1HopBridge "github.com/status-im/status-go/contracts/hop/l1Contracts/l1HopBridge"
	hopL2AmmWrapper "github.com/status-im/status-go/contracts/hop/l2Contracts/l2AmmWrapper"
	hopL2ArbitrumBridge "github.com/status-im/status-go/contracts/hop/l2Contracts/l2ArbitrumBridge"
	hopL2CctpImplementation "github.com/status-im/status-go/contracts/hop/l2Contracts/l2CctpImplementation"
	hopL2OptimismBridge "github.com/status-im/status-go/contracts/hop/l2Contracts/l2OptimismBridge"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

type HopBridgeTxArgs struct {
	transactions.SendTxArgs
	ChainID   uint64         `json:"chainId"`
	ChainIDTo uint64         `json:"chainIdTo"`
	Symbol    string         `json:"symbol"`
	Recipient common.Address `json:"recipient"`
	Amount    *hexutil.Big   `json:"amount"`
	BonderFee *hexutil.Big   `json:"bonderFee"`
}

type BonderFee struct {
	AmountIn                *bigint.BigInt `json:"amountIn"`
	Slippage                float32        `json:"slippage"`
	AmountOutMin            *bigint.BigInt `json:"amountOutMin"`
	DestinationAmountOutMin *bigint.BigInt `json:"destinationAmountOutMin"`
	BonderFee               *bigint.BigInt `json:"bonderFee"`
	EstimatedRecieved       *bigint.BigInt `json:"estimatedRecieved"`
	Deadline                int64          `json:"deadline"`
	DestinationDeadline     int64          `json:"destinationDeadline"`
}

func (bf *BonderFee) UnmarshalJSON(data []byte) error {
	type Alias BonderFee
	aux := &struct {
		AmountIn                string  `json:"amountIn"`
		Slippage                float32 `json:"slippage"`
		AmountOutMin            string  `json:"amountOutMin"`
		DestinationAmountOutMin string  `json:"destinationAmountOutMin"`
		BonderFee               string  `json:"bonderFee"`
		EstimatedRecieved       string  `json:"estimatedRecieved"`
		Deadline                int64   `json:"deadline"`
		DestinationDeadline     *int64  `json:"destinationDeadline"`
		*Alias
	}{
		Alias: (*Alias)(bf),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return createBridgeHopErrorResponse(err)
	}

	bf.AmountIn = &bigint.BigInt{Int: new(big.Int)}
	bf.AmountIn.SetString(aux.AmountIn, 10)

	bf.Slippage = aux.Slippage

	bf.AmountOutMin = &bigint.BigInt{Int: new(big.Int)}
	bf.AmountOutMin.SetString(aux.AmountOutMin, 10)

	bf.DestinationAmountOutMin = &bigint.BigInt{Int: new(big.Int)}
	bf.DestinationAmountOutMin.SetString(aux.DestinationAmountOutMin, 10)

	bf.BonderFee = &bigint.BigInt{Int: new(big.Int)}
	bf.BonderFee.SetString(aux.BonderFee, 10)

	bf.EstimatedRecieved = &bigint.BigInt{Int: new(big.Int)}
	bf.EstimatedRecieved.SetString(aux.EstimatedRecieved, 10)

	bf.Deadline = aux.Deadline

	if aux.DestinationDeadline != nil {
		bf.DestinationDeadline = *aux.DestinationDeadline
	}

	return nil
}

type HopBridgeProcessor struct {
	transactor     transactions.TransactorIface
	httpClient     *thirdparty.HTTPClient
	tokenManager   *token.Manager
	contractMaker  *contracts.ContractMaker
	networkManager network.ManagerInterface
	bonderFee      *sync.Map // [fromChainName-toChainName]BonderFee
}

func NewHopBridgeProcessor(rpcClient rpc.ClientInterface, transactor transactions.TransactorIface, tokenManager *token.Manager, networkManager network.ManagerInterface) *HopBridgeProcessor {
	return &HopBridgeProcessor{
		contractMaker:  &contracts.ContractMaker{RPCClient: rpcClient},
		httpClient:     thirdparty.NewHTTPClient(),
		transactor:     transactor,
		tokenManager:   tokenManager,
		networkManager: networkManager,
		bonderFee:      &sync.Map{},
	}
}

func createBridgeHopErrorResponse(err error) error {
	return createErrorResponse(ProcessorBridgeHopName, err)
}

func (h *HopBridgeProcessor) Name() string {
	return ProcessorBridgeHopName
}

func (h *HopBridgeProcessor) Clear() {
	h.bonderFee = &sync.Map{}
}

func (h *HopBridgeProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromChain == nil || params.ToChain == nil {
		return false, ErrNoChainSet
	}
	if params.FromToken == nil {
		return false, ErrNoTokenSet
	}
	if params.ToToken != nil {
		return false, ErrToTokenShouldNotBeSet
	}
	if params.FromChain.ChainID == params.ToChain.ChainID {
		return false, ErrFromAndToChainsMustBeDifferent
	}
	// We chcek if the contract is available on the network for the token
	_, err := h.GetContractAddress(params)
	// toToken is not nil only if the send type is Swap
	return err == nil, err
}

func (c *HopBridgeProcessor) getAppropriateABI(contractType string, chainID uint64, token *token.Token) (abi.ABI, error) {
	switch contractType {
	case hop.CctpL1Bridge:
		return abi.JSON(strings.NewReader(hopL1CctpImplementation.HopL1CctpImplementationABI))
	case hop.L1Bridge:
		if token.IsNative() {
			return abi.JSON(strings.NewReader(hopL1EthBridge.HopL1EthBridgeABI))
		}
		if token.Symbol == HopSymbol {
			return abi.JSON(strings.NewReader(hopL1HopBridge.HopL1HopBridgeABI))
		}
		return abi.JSON(strings.NewReader(hopL1Erc20Bridge.HopL1Erc20BridgeABI))
	case hop.L2AmmWrapper:
		return abi.JSON(strings.NewReader(hopL2AmmWrapper.HopL2AmmWrapperABI))
	case hop.CctpL2Bridge:
		return abi.JSON(strings.NewReader(hopL2CctpImplementation.HopL2CctpImplementationABI))
	case hop.L2Bridge:
		if chainID == walletCommon.OptimismMainnet ||
			chainID == walletCommon.OptimismSepolia {
			return abi.JSON(strings.NewReader(hopL2OptimismBridge.HopL2OptimismBridgeABI))
		}
		if chainID == walletCommon.ArbitrumMainnet ||
			chainID == walletCommon.ArbitrumSepolia {
			return abi.JSON(strings.NewReader(hopL2ArbitrumBridge.HopL2ArbitrumBridgeABI))
		}
	}

	return abi.ABI{}, ErrNotAvailableForContractType
}

func (h *HopBridgeProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	_, contractType, err := hop.GetContractAddress(params.FromChain.ChainID, params.FromToken.Symbol)
	if err != nil {
		return []byte{}, createBridgeHopErrorResponse(err)
	}

	return h.packTxInputDataInternally(params, contractType)
}

func (h *HopBridgeProcessor) packTxInputDataInternally(params ProcessorInputParams, contractType string) ([]byte, error) {
	abi, err := h.getAppropriateABI(contractType, params.FromChain.ChainID, params.FromToken)
	if err != nil {
		return []byte{}, createBridgeHopErrorResponse(err)
	}

	bonderKey := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, "", "")
	bonderFeeIns, ok := h.bonderFee.Load(bonderKey)
	if !ok {
		return nil, ErrNoBonderFeeFound
	}
	bonderFee := bonderFeeIns.(*BonderFee)

	switch contractType {
	case hop.CctpL1Bridge:
		return h.packCctpL1BridgeTx(abi, params.ToChain.ChainID, params.ToAddr, bonderFee)
	case hop.L1Bridge:
		return h.packL1BridgeTx(abi, params.ToChain.ChainID, params.ToAddr, bonderFee)
	case hop.L2AmmWrapper:
		return h.packL2AmmWrapperTx(abi, params.ToChain.ChainID, params.ToAddr, bonderFee)
	case hop.CctpL2Bridge:
		return h.packCctpL2BridgeTx(abi, params.ToChain.ChainID, params.ToAddr, bonderFee)
	case hop.L2Bridge:
		return h.packL2BridgeTx(abi, params.ToChain.ChainID, params.ToAddr, bonderFee)
	}

	return []byte{}, ErrContractTypeNotSupported
}

func (h *HopBridgeProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[h.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	value := big.NewInt(0)
	if params.FromToken.IsNative() {
		value = params.AmountIn
	}

	contractAddress, contractType, err := hop.GetContractAddress(params.FromChain.ChainID, params.FromToken.Symbol)
	if err != nil {
		return 0, createBridgeHopErrorResponse(err)
	}

	input, err := h.packTxInputDataInternally(params, contractType)
	if err != nil {
		return 0, createBridgeHopErrorResponse(err)
	}

	ethClient, err := h.contractMaker.RPCClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createBridgeHopErrorResponse(err)
	}

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		if !params.FromToken.IsNative() {
			// TODO: this is a temporary solution until we find a better way to estimate the gas
			// hardcoding the estimation for other than ETH, cause we cannot get a proper estimation without having an approval placed first
			// this is an error we're facing otherwise: `execution reverted: ERC20: transfer amount exceeds allowance`
			estimation = 350000
		} else {
			return 0, createBridgeHopErrorResponse(err)
		}
	}

	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (h *HopBridgeProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	address, _, err := hop.GetContractAddress(params.FromChain.ChainID, params.FromToken.Symbol)
	return address, createBridgeHopErrorResponse(err)
}

// TODO: remove this struct once mobile switches to the new approach
func (h *HopBridgeProcessor) sendOrBuild(sendArgs *MultipathProcessorTxArgs, signerFn bind.SignerFn, lastUsedNonce int64) (tx *ethTypes.Transaction, err error) {
	fromChain := h.networkManager.Find(sendArgs.HopTx.ChainID)
	if fromChain == nil {
		return tx, fmt.Errorf("ChainID not supported %d", sendArgs.HopTx.ChainID)
	}

	token := h.tokenManager.FindToken(fromChain, sendArgs.HopTx.Symbol)

	var nonce uint64
	if lastUsedNonce < 0 {
		nonce, err = h.transactor.NextNonce(h.contractMaker.RPCClient, fromChain.ChainID, sendArgs.HopTx.From)
		if err != nil {
			return tx, createBridgeHopErrorResponse(err)
		}
	} else {
		nonce = uint64(lastUsedNonce) + 1
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.HopTx.Nonce = &argNonce

	txOpts := sendArgs.HopTx.ToTransactOpts(signerFn)
	if token.IsNative() {
		txOpts.Value = (*big.Int)(sendArgs.HopTx.Amount)
	}

	ethClient, err := h.contractMaker.RPCClient.EthClient(fromChain.ChainID)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	contractAddress, contractType, err := hop.GetContractAddress(fromChain.ChainID, sendArgs.HopTx.Symbol)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	bonderKey := makeKey(sendArgs.HopTx.ChainID, sendArgs.HopTx.ChainIDTo, "", "")
	bonderFeeIns, ok := h.bonderFee.Load(bonderKey)
	if !ok {
		return nil, ErrNoBonderFeeFound
	}
	bonderFee := bonderFeeIns.(*BonderFee)

	switch contractType {
	case hop.CctpL1Bridge:
		tx, err = h.sendCctpL1BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainIDTo, sendArgs.HopTx.Recipient, txOpts, bonderFee)
	case hop.L1Bridge:
		tx, err = h.sendL1BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainIDTo, sendArgs.HopTx.Recipient, txOpts, token, bonderFee)
	case hop.L2AmmWrapper:
		tx, err = h.sendL2AmmWrapperTx(contractAddress, ethClient, sendArgs.HopTx.ChainIDTo, sendArgs.HopTx.Recipient, txOpts, bonderFee)
	case hop.CctpL2Bridge:
		tx, err = h.sendCctpL2BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainIDTo, sendArgs.HopTx.Recipient, txOpts, bonderFee)
	case hop.L2Bridge:
		tx, err = h.sendL2BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainIDTo, sendArgs.HopTx.Recipient, txOpts, bonderFee)
	default:
		return tx, ErrContractTypeNotSupported
	}
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}
	err = h.transactor.StoreAndTrackPendingTx(txOpts.From, sendArgs.HopTx.Symbol, sendArgs.HopTx.ChainID, sendArgs.HopTx.MultiTransactionID, tx)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}
	return tx, nil
}

func (h *HopBridgeProcessor) sendOrBuildV2(sendArgs *transactions.SendTxArgs, signerFn bind.SignerFn, lastUsedNonce int64) (tx *ethTypes.Transaction, err error) {
	fromChain := h.networkManager.Find(sendArgs.FromChainID)
	if fromChain == nil {
		return tx, fmt.Errorf("ChainID not supported %d", sendArgs.FromChainID)
	}

	token := h.tokenManager.FindToken(fromChain, sendArgs.FromTokenID)

	var nonce uint64
	if lastUsedNonce < 0 {
		nonce, err = h.transactor.NextNonce(h.contractMaker.RPCClient, fromChain.ChainID, sendArgs.From)
		if err != nil {
			return tx, createBridgeHopErrorResponse(err)
		}
	} else {
		nonce = uint64(lastUsedNonce) + 1
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.Nonce = &argNonce

	txOpts := sendArgs.ToTransactOpts(signerFn)
	if token.IsNative() {
		txOpts.Value = (*big.Int)(sendArgs.Value)
	}

	ethClient, err := h.contractMaker.RPCClient.EthClient(fromChain.ChainID)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	contractAddress, contractType, err := hop.GetContractAddress(fromChain.ChainID, sendArgs.FromTokenID)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	bonderKey := makeKey(sendArgs.FromChainID, sendArgs.ToChainID, "", "")
	bonderFeeIns, ok := h.bonderFee.Load(bonderKey)
	if !ok {
		return nil, ErrNoBonderFeeFound
	}
	bonderFee := bonderFeeIns.(*BonderFee)

	switch contractType {
	case hop.CctpL1Bridge:
		tx, err = h.sendCctpL1BridgeTx(contractAddress, ethClient, sendArgs.ToChainID, common.Address(*sendArgs.To), txOpts, bonderFee)
	case hop.L1Bridge:
		tx, err = h.sendL1BridgeTx(contractAddress, ethClient, sendArgs.ToChainID, common.Address(*sendArgs.To), txOpts, token, bonderFee)
	case hop.L2AmmWrapper:
		tx, err = h.sendL2AmmWrapperTx(contractAddress, ethClient, sendArgs.ToChainID, common.Address(*sendArgs.To), txOpts, bonderFee)
	case hop.CctpL2Bridge:
		tx, err = h.sendCctpL2BridgeTx(contractAddress, ethClient, sendArgs.ToChainID, common.Address(*sendArgs.To), txOpts, bonderFee)
	case hop.L2Bridge:
		tx, err = h.sendL2BridgeTx(contractAddress, ethClient, sendArgs.ToChainID, common.Address(*sendArgs.To), txOpts, bonderFee)
	default:
		return tx, ErrContractTypeNotSupported
	}
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}
	err = h.transactor.StoreAndTrackPendingTx(txOpts.From, sendArgs.FromTokenID, sendArgs.FromChainID, sendArgs.MultiTransactionID, tx)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}
	return tx, nil
}

func (h *HopBridgeProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (hash types.Hash, nonce uint64, err error) {
	tx, err := h.sendOrBuild(sendArgs, getSigner(sendArgs.HopTx.ChainID, sendArgs.HopTx.From, verifiedAccount), lastUsedNonce)
	if err != nil {
		return types.Hash{}, 0, createBridgeHopErrorResponse(err)
	}
	return types.Hash(tx.Hash()), tx.Nonce(), nil
}

func (h *HopBridgeProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	tx, err := h.sendOrBuild(sendArgs, nil, lastUsedNonce)
	return tx, tx.Nonce(), createBridgeHopErrorResponse(err)
}

func (h *HopBridgeProcessor) BuildTransactionV2(sendArgs *transactions.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	tx, err := h.sendOrBuildV2(sendArgs, nil, lastUsedNonce)
	if err != nil {
		return nil, 0, createBridgeHopErrorResponse(err)
	}
	return tx, tx.Nonce(), createBridgeHopErrorResponse(err)
}

func (h *HopBridgeProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	bonderKey := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, "", "")
	if params.TestsMode {
		if val, ok := params.TestBonderFeeMap[params.FromToken.Symbol]; ok {
			res := new(big.Int).Sub(params.AmountIn, val)
			bonderFee := &BonderFee{
				AmountIn:                &bigint.BigInt{Int: params.AmountIn},
				Slippage:                5,
				AmountOutMin:            &bigint.BigInt{Int: res},
				DestinationAmountOutMin: &bigint.BigInt{Int: res},
				BonderFee:               &bigint.BigInt{Int: val},
				EstimatedRecieved:       &bigint.BigInt{Int: res},
				Deadline:                time.Now().Add(SevenDaysInSeconds).Unix(),
			}
			h.bonderFee.Store(bonderKey, bonderFee)
			return val, walletCommon.ZeroBigIntValue(), nil
		}
		return nil, nil, ErrNoBonderFeeFound
	}

	hopChainsMap := map[uint64]string{
		walletCommon.EthereumMainnet: "ethereum",
		walletCommon.OptimismMainnet: "optimism",
		walletCommon.ArbitrumMainnet: "arbitrum",
	}

	fromChainName, ok := hopChainsMap[params.FromChain.ChainID]
	if !ok {
		return nil, nil, ErrFromChainNotSupported
	}

	toChainName, ok := hopChainsMap[params.ToChain.ChainID]
	if !ok {
		return nil, nil, ErrToChainNotSupported
	}

	reqParams := netUrl.Values{}
	reqParams.Add("amount", params.AmountIn.String())
	reqParams.Add("token", params.FromToken.Symbol)
	reqParams.Add("fromChain", fromChainName)
	reqParams.Add("toChain", toChainName)
	reqParams.Add("slippage", "0.5") // menas 0.5%

	url := "https://api.hop.exchange/v1/quote"
	response, err := h.httpClient.DoGetRequest(context.Background(), url, reqParams, nil)
	if err != nil {
		return nil, nil, err
	}

	bonderFee := &BonderFee{}
	err = json.Unmarshal(response, bonderFee)
	if err != nil {
		return nil, nil, createBridgeHopErrorResponse(err)
	}

	h.bonderFee.Store(bonderKey, bonderFee)

	// Remove token fee from bonder fee as said here:
	// https://docs.hop.exchange/v/developer-docs/api/api#get-v1-quote
	// `bonderFee` - The suggested bonder fee for the amount in. The bonder fee also includes the cost of the destination transaction fee.
	tokenFee := new(big.Int).Sub(bonderFee.AmountIn.Int, bonderFee.EstimatedRecieved.Int)

	return bonderFee.BonderFee.Int, tokenFee, nil
}

func (h *HopBridgeProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	bonderKey := makeKey(params.FromChain.ChainID, params.ToChain.ChainID, "", "")
	bonderFeeIns, ok := h.bonderFee.Load(bonderKey)
	if !ok {
		return nil, ErrNoBonderFeeFound
	}
	bonderFee := bonderFeeIns.(*BonderFee)
	return bonderFee.AmountOutMin.Int, nil
}

func (h *HopBridgeProcessor) packCctpL1BridgeTx(abi abi.ABI, toChainID uint64, to common.Address, bonderFee *BonderFee) ([]byte, error) {
	return abi.Pack("send",
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int)
}

func (h *HopBridgeProcessor) sendCctpL1BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64,
	to common.Address, txOpts *bind.TransactOpts, bonderFee *BonderFee) (tx *ethTypes.Transaction, err error) {
	contractInstance, err := hopL1CctpImplementation.NewHopL1CctpImplementation(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	return contractInstance.Send(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int)
}

func (h *HopBridgeProcessor) packL1BridgeTx(abi abi.ABI, toChainID uint64, to common.Address, bonderFee *BonderFee) ([]byte, error) {
	return abi.Pack("sendToL2",
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.AmountOutMin.Int,
		big.NewInt(bonderFee.Deadline),
		common.Address{},
		walletCommon.ZeroBigIntValue())
}

func (h *HopBridgeProcessor) sendL1BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64,
	to common.Address, txOpts *bind.TransactOpts, token *token.Token, bonderFee *BonderFee) (tx *ethTypes.Transaction, err error) {
	if token.IsNative() {
		contractInstance, err := hopL1EthBridge.NewHopL1EthBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, createBridgeHopErrorResponse(err)
		}

		return contractInstance.SendToL2(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			bonderFee.AmountIn.Int,
			bonderFee.AmountOutMin.Int,
			big.NewInt(bonderFee.Deadline),
			common.Address{},
			walletCommon.ZeroBigIntValue())
	}

	if token.Symbol == HopSymbol {
		contractInstance, err := hopL1HopBridge.NewHopL1HopBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, createBridgeHopErrorResponse(err)
		}

		return contractInstance.SendToL2(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			bonderFee.AmountIn.Int,
			bonderFee.AmountOutMin.Int,
			big.NewInt(bonderFee.Deadline),
			common.Address{},
			walletCommon.ZeroBigIntValue())
	}

	contractInstance, err := hopL1Erc20Bridge.NewHopL1Erc20Bridge(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	return contractInstance.SendToL2(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.AmountOutMin.Int,
		big.NewInt(bonderFee.Deadline),
		common.Address{},
		walletCommon.ZeroBigIntValue())

}

func (h *HopBridgeProcessor) packCctpL2BridgeTx(abi abi.ABI, toChainID uint64, to common.Address, bonderFee *BonderFee) ([]byte, error) {
	return abi.Pack("send",
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int)
}

func (h *HopBridgeProcessor) sendCctpL2BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64,
	to common.Address, txOpts *bind.TransactOpts, bonderFee *BonderFee) (tx *ethTypes.Transaction, err error) {
	contractInstance, err := hopL2CctpImplementation.NewHopL2CctpImplementation(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	return contractInstance.Send(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int,
	)
}

func (h *HopBridgeProcessor) packL2AmmWrapperTx(abi abi.ABI, toChainID uint64, to common.Address, bonderFee *BonderFee) ([]byte, error) {
	return abi.Pack("swapAndSend",
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int,
		bonderFee.AmountOutMin.Int,
		big.NewInt(bonderFee.Deadline),
		bonderFee.DestinationAmountOutMin.Int,
		big.NewInt(bonderFee.DestinationDeadline))
}

func (h *HopBridgeProcessor) sendL2AmmWrapperTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64,
	to common.Address, txOpts *bind.TransactOpts, bonderFee *BonderFee) (tx *ethTypes.Transaction, err error) {
	contractInstance, err := hopL2AmmWrapper.NewHopL2AmmWrapper(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, createBridgeHopErrorResponse(err)
	}

	return contractInstance.SwapAndSend(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int,
		bonderFee.AmountOutMin.Int,
		big.NewInt(bonderFee.Deadline),
		bonderFee.DestinationAmountOutMin.Int,
		big.NewInt(bonderFee.DestinationDeadline))
}

func (h *HopBridgeProcessor) packL2BridgeTx(abi abi.ABI, toChainID uint64, to common.Address, bonderFee *BonderFee) ([]byte, error) {
	return abi.Pack("send",
		big.NewInt(int64(toChainID)),
		to,
		bonderFee.AmountIn.Int,
		bonderFee.BonderFee.Int,
		bonderFee.AmountOutMin.Int,
		big.NewInt(bonderFee.Deadline))
}

func (h *HopBridgeProcessor) sendL2BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64,
	to common.Address, txOpts *bind.TransactOpts, bonderFee *BonderFee) (tx *ethTypes.Transaction, err error) {
	fromChainID := ethClient.NetworkID()
	if fromChainID == walletCommon.OptimismMainnet ||
		fromChainID == walletCommon.OptimismSepolia {
		contractInstance, err := hopL2OptimismBridge.NewHopL2OptimismBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, createBridgeHopErrorResponse(err)
		}

		return contractInstance.Send(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			bonderFee.AmountIn.Int,
			bonderFee.BonderFee.Int,
			bonderFee.AmountOutMin.Int,
			big.NewInt(bonderFee.Deadline))
	}
	if fromChainID == walletCommon.ArbitrumMainnet ||
		fromChainID == walletCommon.ArbitrumSepolia {
		contractInstance, err := hopL2ArbitrumBridge.NewHopL2ArbitrumBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, createBridgeHopErrorResponse(err)
		}

		return contractInstance.Send(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			bonderFee.AmountIn.Int,
			bonderFee.BonderFee.Int,
			bonderFee.AmountOutMin.Int,
			big.NewInt(bonderFee.Deadline))
	}

	return tx, ErrTxForChainNotSupported
}
