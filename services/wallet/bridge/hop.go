package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	netUrl "net/url"
	"strings"

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
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

const (
	SevenDaysInSeconds = 604800
	hopSymbol          = "HOP"
)

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
	// We chcek if the contract is available on the network for the token
	_, err := h.GetContractAddress(from, token)
	// toToken is not nil only if the send type is Swap
	return err == nil && toToken == nil, nil
}

func (c *HopBridge) getAppropriateABI(contractType string, chainID uint64, token *token.Token) (abi.ABI, error) {
	switch contractType {
	case hop.CctpL1Bridge:
		return abi.JSON(strings.NewReader(hopL1CctpImplementation.HopL1CctpImplementationABI))
	case hop.L1Bridge:
		if token.IsNative() {
			return abi.JSON(strings.NewReader(hopL1EthBridge.HopL1EthBridgeABI))
		}
		if token.Symbol == hopSymbol {
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

	return abi.ABI{}, errors.New("not available for contract type")
}

func (h *HopBridge) PackTxInputData(contractType string, fromNetwork *params.Network, toNetwork *params.Network, from common.Address, to common.Address, token *token.Token, amountIn *big.Int) ([]byte, error) {
	abi, err := h.getAppropriateABI(contractType, fromNetwork.ChainID, token)
	if err != nil {
		return []byte{}, err
	}

	switch contractType {
	case hop.CctpL1Bridge:
		return h.packCctpL1BridgeTx(abi, toNetwork.ChainID, to)
	case hop.L1Bridge:
		return h.packL1BridgeTx(abi, toNetwork.ChainID, to)
	case hop.L2AmmWrapper:
		return h.packL2AmmWrapperTx(abi, toNetwork.ChainID, to)
	case hop.CctpL2Bridge:
		return h.packCctpL2BridgeTx(abi, toNetwork.ChainID, to)
	case hop.L2Bridge:
		return h.packL2BridgeTx(abi, toNetwork.ChainID, to)
	}

	return []byte{}, errors.New("contract type not supported yet")
}

func (h *HopBridge) EstimateGas(fromNetwork *params.Network, toNetwork *params.Network, from common.Address, to common.Address, token *token.Token, toToken *token.Token, amountIn *big.Int) (uint64, error) {
	value := big.NewInt(0)
	if token.IsNative() {
		value = amountIn
	}

	contractAddress, contractType, err := hop.GetContractAddress(fromNetwork.ChainID, token.Symbol)
	if err != nil {
		return 0, err
	}

	input, err := h.PackTxInputData(contractType, fromNetwork, toNetwork, from, to, token, amountIn)
	if err != nil {
		return 0, err
	}

	ethClient, err := h.contractMaker.RPCClient.EthClient(fromNetwork.ChainID)
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		From:  from,
		To:    &contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(context.Background(), msg)
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

func (h *HopBridge) GetContractAddress(network *params.Network, token *token.Token) (common.Address, error) {
	address, _, err := hop.GetContractAddress(network.ChainID, token.Symbol)
	return address, err
}

func (h *HopBridge) sendOrBuild(sendArgs *TransactionBridge, signerFn bind.SignerFn) (tx *ethTypes.Transaction, err error) {
	fromNetwork := h.contractMaker.RPCClient.NetworkManager.Find(sendArgs.ChainID)
	if fromNetwork == nil {
		return tx, fmt.Errorf("ChainID not supported %d", sendArgs.ChainID)
	}

	token := h.tokenManager.FindToken(fromNetwork, sendArgs.HopTx.Symbol)

	nonce, err := h.transactor.NextNonce(h.contractMaker.RPCClient, fromNetwork.ChainID, sendArgs.HopTx.From)
	if err != nil {
		return tx, err
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.HopTx.Nonce = &argNonce

	txOpts := sendArgs.HopTx.ToTransactOpts(signerFn)
	if token.IsNative() {
		txOpts.Value = (*big.Int)(sendArgs.HopTx.Amount)
	}

	ethClient, err := h.contractMaker.RPCClient.EthClient(fromNetwork.ChainID)
	if err != nil {
		return tx, err
	}

	contractAddress, contractType, err := hop.GetContractAddress(fromNetwork.ChainID, sendArgs.HopTx.Symbol)
	if err != nil {
		return tx, err
	}

	switch contractType {
	case hop.CctpL1Bridge:
		return h.sendCctpL1BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainID, sendArgs.HopTx.Recipient, txOpts)
	case hop.L1Bridge:
		return h.sendL1BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainID, sendArgs.HopTx.Recipient, txOpts, token)
	case hop.L2AmmWrapper:
		return h.sendL2AmmWrapperTx(contractAddress, ethClient, sendArgs.HopTx.ChainID, sendArgs.HopTx.Recipient, txOpts)
	case hop.CctpL2Bridge:
		return h.sendCctpL2BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainID, sendArgs.HopTx.Recipient, txOpts)
	case hop.L2Bridge:
		return h.sendL2BridgeTx(contractAddress, ethClient, sendArgs.HopTx.ChainID, sendArgs.HopTx.Recipient, txOpts)
	}

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

	// Remove token fee from bonder fee as said here:
	// https://docs.hop.exchange/v/developer-docs/api/api#get-v1-quote
	// `bonderFee` - The suggested bonder fee for the amount in. The bonder fee also includes the cost of the destination transaction fee.
	tokenFee := big.NewInt(0) //new(big.Int).Sub(h.bonderFee.AmountIn.Int, h.bonderFee.EstimatedRecieved.Int)

	return h.bonderFee.BonderFee.Int, tokenFee, nil
}

func (h *HopBridge) CalculateAmountOut(from, to *params.Network, amountIn *big.Int, symbol string) (*big.Int, error) {
	return h.bonderFee.EstimatedRecieved.Int, nil
}

func (h *HopBridge) packCctpL1BridgeTx(abi abi.ABI, toChainID uint64, to common.Address) ([]byte, error) {
	return abi.Pack("send",
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int)
}

func (h *HopBridge) sendCctpL1BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64, to common.Address, txOpts *bind.TransactOpts) (tx *ethTypes.Transaction, err error) {
	contractInstance, err := hopL1CctpImplementation.NewHopL1CctpImplementation(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, err
	}

	return contractInstance.Send(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int)
}

func (h *HopBridge) packL1BridgeTx(abi abi.ABI, toChainID uint64, to common.Address) ([]byte, error) {
	return abi.Pack("sendToL2",
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.AmountOutMin.Int,
		big.NewInt(h.bonderFee.Deadline),
		common.Address{},
		big.NewInt(0))
}

func (h *HopBridge) sendL1BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64, to common.Address, txOpts *bind.TransactOpts, token *token.Token) (tx *ethTypes.Transaction, err error) {
	if token.IsNative() {
		contractInstance, err := hopL1EthBridge.NewHopL1EthBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, err
		}

		return contractInstance.SendToL2(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			h.bonderFee.AmountIn.Int,
			h.bonderFee.AmountOutMin.Int,
			big.NewInt(h.bonderFee.Deadline),
			common.Address{},
			big.NewInt(0))
	}

	if token.Symbol == hopSymbol {
		contractInstance, err := hopL1HopBridge.NewHopL1HopBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, err
		}

		return contractInstance.SendToL2(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			h.bonderFee.AmountIn.Int,
			h.bonderFee.AmountOutMin.Int,
			big.NewInt(h.bonderFee.Deadline),
			common.Address{},
			big.NewInt(0))
	}

	contractInstance, err := hopL1Erc20Bridge.NewHopL1Erc20Bridge(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, err
	}

	return contractInstance.SendToL2(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.AmountOutMin.Int,
		big.NewInt(h.bonderFee.Deadline),
		common.Address{},
		big.NewInt(0))

}

func (h *HopBridge) packCctpL2BridgeTx(abi abi.ABI, toChainID uint64, to common.Address) ([]byte, error) {
	return abi.Pack("send",
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int)
}

func (h *HopBridge) sendCctpL2BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64, to common.Address, txOpts *bind.TransactOpts) (tx *ethTypes.Transaction, err error) {
	contractInstance, err := hopL2CctpImplementation.NewHopL2CctpImplementation(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, err
	}

	return contractInstance.Send(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int,
	)
}

func (h *HopBridge) packL2AmmWrapperTx(abi abi.ABI, toChainID uint64, to common.Address) ([]byte, error) {
	return abi.Pack("swapAndSend",
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int,
		h.bonderFee.AmountOutMin.Int,
		big.NewInt(h.bonderFee.Deadline),
		h.bonderFee.DestinationAmountOutMin.Int,
		big.NewInt(h.bonderFee.DestinationDeadline))
}

func (h *HopBridge) sendL2AmmWrapperTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64, to common.Address, txOpts *bind.TransactOpts) (tx *ethTypes.Transaction, err error) {
	contractInstance, err := hopL2AmmWrapper.NewHopL2AmmWrapper(
		contractAddress,
		ethClient,
	)
	if err != nil {
		return tx, err
	}

	return contractInstance.SwapAndSend(
		txOpts,
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int,
		h.bonderFee.AmountOutMin.Int,
		big.NewInt(h.bonderFee.Deadline),
		h.bonderFee.DestinationAmountOutMin.Int,
		big.NewInt(h.bonderFee.DestinationDeadline))
}

func (h *HopBridge) packL2BridgeTx(abi abi.ABI, toChainID uint64, to common.Address) ([]byte, error) {
	return abi.Pack("send",
		big.NewInt(int64(toChainID)),
		to,
		h.bonderFee.AmountIn.Int,
		h.bonderFee.BonderFee.Int,
		h.bonderFee.AmountOutMin.Int,
		big.NewInt(h.bonderFee.Deadline))
}

func (h *HopBridge) sendL2BridgeTx(contractAddress common.Address, ethClient chain.ClientInterface, toChainID uint64, to common.Address, txOpts *bind.TransactOpts) (tx *ethTypes.Transaction, err error) {
	fromChainID := ethClient.NetworkID()
	if fromChainID == walletCommon.OptimismMainnet ||
		fromChainID == walletCommon.OptimismSepolia {
		contractInstance, err := hopL2OptimismBridge.NewHopL2OptimismBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, err
		}

		return contractInstance.Send(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			h.bonderFee.AmountIn.Int,
			h.bonderFee.BonderFee.Int,
			h.bonderFee.AmountOutMin.Int,
			big.NewInt(h.bonderFee.Deadline))
	}
	if fromChainID == walletCommon.ArbitrumMainnet ||
		fromChainID == walletCommon.ArbitrumSepolia {
		contractInstance, err := hopL2ArbitrumBridge.NewHopL2ArbitrumBridge(
			contractAddress,
			ethClient,
		)
		if err != nil {
			return tx, err
		}

		return contractInstance.Send(
			txOpts,
			big.NewInt(int64(toChainID)),
			to,
			h.bonderFee.AmountIn.Int,
			h.bonderFee.BonderFee.Int,
			h.bonderFee.AmountOutMin.Int,
			big.NewInt(h.bonderFee.Deadline))
	}

	return tx, errors.New("tx for chain not supported yet")
}
