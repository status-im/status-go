package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	netUrl "net/url"
	"strings"
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
	hopBridge "github.com/status-im/status-go/contracts/hop/bridge"
	hopWrapper "github.com/status-im/status-go/contracts/hop/wrapper"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

const SevenDaysInSeconds = 604800

type HopTxArgs struct {
	transactions.SendTxArgs
	ChainID   uint64         `json:"chainId"`
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
		return err
	}

	bf.AmountIn = &bigint.BigInt{Int: new(big.Int)}
	bf.AmountIn.SetString(aux.AmountIn, 10)

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

type HopBridge struct {
	transactor    *transactions.Transactor
	httpClient    *thirdparty.HTTPClient
	tokenManager  *token.Manager
	contractMaker *contracts.ContractMaker
	bonderFee     *BonderFee
}

func NewHopBridge(rpcClient *rpc.Client, transactor *transactions.Transactor, tokenManager *token.Manager) *HopBridge {
	return &HopBridge{
		contractMaker: &contracts.ContractMaker{RPCClient: rpcClient},
		httpClient:    thirdparty.NewHTTPClient(),
		transactor:    transactor,
		tokenManager:  tokenManager,
	}
}

func (h *HopBridge) Name() string {
	return "Hop"
}

func (h *HopBridge) AvailableFor(from, to *params.Network, token *token.Token, toToken *token.Token) (bool, error) {
	if from.ChainID == to.ChainID || toToken != nil {
		return false, nil
	}

	// currently Hop bridge is not available for testnets
	if from.IsTest || to.IsTest {
		return false, nil
	}

	return true, nil
}

func (h *HopBridge) PackTxInputData(fromNetwork *params.Network, toNetwork *params.Network, from common.Address, to common.Address, token *token.Token, amountIn *big.Int) ([]byte, error) {
	if fromNetwork.Layer == 1 {
		ABI, err := abi.JSON(strings.NewReader(hopBridge.HopBridgeABI))
		if err != nil {
			return []byte{}, err
		}

		return ABI.Pack("sendToL2",
			big.NewInt(int64(toNetwork.ChainID)),
			to,
			h.bonderFee.AmountIn.Int,
			h.bonderFee.AmountOutMin.Int,
			big.NewInt(h.bonderFee.Deadline),
			common.HexToAddress("0x0"),
			big.NewInt(0))
	} else {
		ABI, err := abi.JSON(strings.NewReader(hopWrapper.HopWrapperABI))
		if err != nil {
			return []byte{}, err
		}

		return ABI.Pack("swapAndSend",
			big.NewInt(int64(toNetwork.ChainID)),
			to,
			h.bonderFee.AmountIn.Int,
			h.bonderFee.BonderFee.Int,
			h.bonderFee.AmountOutMin.Int,
			big.NewInt(h.bonderFee.Deadline),
			h.bonderFee.DestinationAmountOutMin.Int,
			big.NewInt(h.bonderFee.DestinationDeadline))
	}
}

func (h *HopBridge) EstimateGas(fromNetwork *params.Network, toNetwork *params.Network, from common.Address, to common.Address, token *token.Token, toToken *token.Token, amountIn *big.Int) (uint64, error) {
	value := new(big.Int)

	if token.IsNative() {
		value = amountIn
	}

	contractAddress := h.GetContractAddress(fromNetwork, token)
	if contractAddress == nil {
		return 0, errors.New("contract not found")
	}

	ctx := context.Background()

	input, err := h.PackTxInputData(fromNetwork, toNetwork, from, to, token, amountIn)
	if err != nil {
		return 0, err
	}

	ethClient, err := h.contractMaker.RPCClient.EthClient(fromNetwork.ChainID)
	if err != nil {
		return 0, err
	}

	if code, err := ethClient.PendingCodeAt(ctx, *contractAddress); err != nil {
		return 0, err
	} else if len(code) == 0 {
		return 0, bind.ErrNoCode
	}

	msg := ethereum.CallMsg{
		From:  from,
		To:    contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(ctx, msg)
	if err != nil {
		return 0, err
	}
	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (h *HopBridge) BuildTx(fromNetwork, toNetwork *params.Network, fromAddress common.Address, toAddress common.Address, token *token.Token, amountIn *big.Int, bonderFee *big.Int) (*ethTypes.Transaction, error) {
	toAddr := types.Address(toAddress)
	sendArgs := &TransactionBridge{
		HopTx: &HopTxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:  types.Address(fromAddress),
				To:    &toAddr,
				Value: (*hexutil.Big)(amountIn),
				Data:  types.HexBytes("0x0"),
			},
			Symbol:    token.Symbol,
			Recipient: toAddress,
			Amount:    (*hexutil.Big)(amountIn),
			BonderFee: (*hexutil.Big)(bonderFee),
			ChainID:   toNetwork.ChainID,
		},
		ChainID: fromNetwork.ChainID,
	}

	return h.BuildTransaction(sendArgs)
}

func (h *HopBridge) GetContractAddress(network *params.Network, token *token.Token) *common.Address {
	var address common.Address
	if network.Layer == 1 {
		address, _ = hop.L1BridgeContractAddress(network.ChainID, token.Symbol)
	} else {
		address, _ = hop.L2AmmWrapperContractAddress(network.ChainID, token.Symbol)
	}

	return &address
}

func (h *HopBridge) sendOrBuild(sendArgs *TransactionBridge, signerFn bind.SignerFn) (tx *ethTypes.Transaction, err error) {
	fromNetwork := h.contractMaker.RPCClient.NetworkManager.Find(sendArgs.ChainID)
	if fromNetwork == nil {
		return tx, fmt.Errorf("ChainID not supported %d", sendArgs.ChainID)
	}

	nonce, err := h.transactor.NextNonce(h.contractMaker.RPCClient, fromNetwork.ChainID, sendArgs.HopTx.From)
	if err != nil {
		return tx, err
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.HopTx.Nonce = &argNonce

	token := h.tokenManager.FindToken(fromNetwork, sendArgs.HopTx.Symbol)
	if fromNetwork.Layer == 1 {
		tx, err = h.sendToL2(sendArgs.ChainID, sendArgs.HopTx, signerFn, token)
		return tx, err
	}
	tx, err = h.swapAndSend(sendArgs.ChainID, sendArgs.HopTx, signerFn, token)
	return tx, err
}

func (h *HopBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (hash types.Hash, err error) {
	tx, err := h.sendOrBuild(sendArgs, getSigner(sendArgs.ChainID, sendArgs.HopTx.From, verifiedAccount))
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(tx.Hash()), nil
}

func (h *HopBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return h.sendOrBuild(sendArgs, nil)
}

func (h *HopBridge) sendToL2(chainID uint64, hopArgs *HopTxArgs, signerFn bind.SignerFn, token *token.Token) (tx *ethTypes.Transaction, err error) {
	bridge, err := h.contractMaker.NewHopL1Bridge(chainID, hopArgs.Symbol)
	if err != nil {
		return tx, err
	}
	txOpts := hopArgs.ToTransactOpts(signerFn)
	if token.IsNative() {
		txOpts.Value = (*big.Int)(hopArgs.Amount)
	}

	var (
		deadline     *big.Int
		amountOutMin *big.Int
	)

	if h.bonderFee != nil {
		deadline = big.NewInt(h.bonderFee.Deadline)
		amountOutMin = h.bonderFee.AmountOutMin.Int
	} else {
		now := time.Now()
		deadline = big.NewInt(now.Unix() + SevenDaysInSeconds)
	}

	tx, err = bridge.SendToL2(
		txOpts,
		big.NewInt(int64(hopArgs.ChainID)),
		hopArgs.Recipient,
		hopArgs.Amount.ToInt(),
		amountOutMin,
		deadline,
		common.HexToAddress("0x0"),
		big.NewInt(0),
	)

	return tx, err
}

func (h *HopBridge) swapAndSend(chainID uint64, hopArgs *HopTxArgs, signerFn bind.SignerFn, token *token.Token) (tx *ethTypes.Transaction, err error) {
	ammWrapper, err := h.contractMaker.NewHopL2AmmWrapper(chainID, hopArgs.Symbol)
	if err != nil {
		return tx, err
	}

	toNetwork := h.contractMaker.RPCClient.NetworkManager.Find(hopArgs.ChainID)
	if toNetwork == nil {
		return tx, err
	}

	txOpts := hopArgs.ToTransactOpts(signerFn)
	if token.IsNative() {
		txOpts.Value = (*big.Int)(hopArgs.Amount)
	}

	var deadline *big.Int
	amountOutMin := big.NewInt(0)
	destinationDeadline := big.NewInt(0)
	destinationAmountOutMin := big.NewInt(0)

	// https://docs.hop.exchange/v/developer-docs/smart-contracts/integration#l2-greater-than-l1-and-l2-greater-than-l2
	// Do not set `destinationAmountOutMin` and `destinationDeadline` when sending to L1 because there is no AMM on L1,
	// otherwise the computed transferId will be invalid and the transfer will be unbondable. These parameters should be set to 0 when sending to L1.
	if h.bonderFee != nil {
		deadline = big.NewInt(h.bonderFee.Deadline)
		amountOutMin = h.bonderFee.AmountOutMin.Int
		destinationDeadline = big.NewInt(h.bonderFee.DestinationDeadline)
		destinationAmountOutMin = h.bonderFee.DestinationAmountOutMin.Int
	} else {
		now := time.Now()
		deadline = big.NewInt(now.Unix() + SevenDaysInSeconds)
		if toNetwork.Layer != 1 {
			destinationDeadline = big.NewInt(now.Unix() + SevenDaysInSeconds)
		}
	}

	tx, err = ammWrapper.SwapAndSend(
		txOpts,
		new(big.Int).SetUint64(hopArgs.ChainID),
		hopArgs.Recipient,
		hopArgs.Amount.ToInt(),
		hopArgs.BonderFee.ToInt(),
		amountOutMin,
		deadline,
		destinationAmountOutMin,
		destinationDeadline,
	)

	return tx, err
}

func (h *HopBridge) CalculateFees(from, to *params.Network, token *token.Token, amountIn *big.Int) (*big.Int, *big.Int, error) {
	hopChainsMap := map[uint64]string{
		walletCommon.EthereumMainnet: "ethereum",
		walletCommon.OptimismMainnet: "optimism",
		walletCommon.ArbitrumMainnet: "arbitrum",
	}

	fromChainName, ok := hopChainsMap[from.ChainID]
	if !ok {
		return nil, nil, errors.New("from chain not supported")
	}

	toChainName, ok := hopChainsMap[to.ChainID]
	if !ok {
		return nil, nil, errors.New("to chain not supported")
	}

	params := netUrl.Values{}
	params.Add("amount", amountIn.String())
	params.Add("token", token.Symbol)
	params.Add("fromChain", fromChainName)
	params.Add("toChain", toChainName)
	params.Add("slippage", "0.5") // menas 0.5%

	url := "https://api.hop.exchange/v1/quote"
	response, err := h.httpClient.DoGetRequest(context.Background(), url, params)
	if err != nil {
		return nil, nil, err
	}

	h.bonderFee = &BonderFee{}
	err = json.Unmarshal(response, h.bonderFee)
	if err != nil {
		return nil, nil, err
	}

	tokenFee := new(big.Int).Sub(h.bonderFee.AmountIn.Int, h.bonderFee.EstimatedRecieved.Int)

	return h.bonderFee.BonderFee.Int, tokenFee, nil
}

func (h *HopBridge) CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error) {
	return h.bonderFee.EstimatedRecieved.Int, nil
}
