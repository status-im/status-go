package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	"github.com/status-im/status-go/contracts/erc721"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

const (
	erc721FunctionNameSafeTransferFrom = "safeTransferFrom"
	erc721FunctionNameTransferFrom     = "transferFrom"
)

// ERC721Handler handles standard ERC721 transfers
type ERC721Handler struct {
	*BaseNFTHandler
}

func NewERC721Handler(rpcClient rpc.ClientInterface, transactor transactions.TransactorIface) *ERC721Handler {
	return &ERC721Handler{
		BaseNFTHandler: NewBaseNFTHandler(rpcClient, transactor),
	}
}

func (h *ERC721Handler) Name() string {
	return pathProcessorCommon.ProcessorERC721Name
}

func (h *ERC721Handler) CanHandle(contractID thirdparty.ContractID) bool {
	return true
}

func (h *ERC721Handler) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	// For standard ERC721 contracts, try safeTransferFrom first, then transferFrom
	err := h.checkIfFunctionExists(params, erc721FunctionNameSafeTransferFrom)
	if err == nil {
		return h.packTxInputDataInternally(params, erc721FunctionNameSafeTransferFrom)
	}

	return h.packTxInputDataInternally(params, erc721FunctionNameTransferFrom)
}

func (h *ERC721Handler) packTxInputDataInternally(params ProcessorInputParams, functionName string) ([]byte, error) {
	abi, err := abi.JSON(strings.NewReader(erc721.Erc721MetaData.ABI))
	if err != nil {
		return []byte{}, err
	}

	id, err := walletCommon.GetTokenIdFromSymbol(params.FromToken.Symbol)
	if err != nil {
		return []byte{}, err
	}

	return abi.Pack(functionName,
		params.FromAddr,
		params.ToAddr,
		id,
	)
}

func (h *ERC721Handler) checkIfFunctionExists(params ProcessorInputParams, functionName string) error {
	data, err := h.packTxInputDataInternally(params, functionName)
	if err != nil {
		return err
	}

	ethClient, err := h.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return err
	}

	value := new(big.Int)
	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: value,
		Data:  data,
	}

	_, err = ethClient.CallContract(context.Background(), msg, nil)
	return err
}

func (h *ERC721Handler) SendOrBuild(
	transactor transactions.TransactorIface,
	rpcClient rpc.ClientInterface,
	sendArgs *MultipathProcessorTxArgs,
	signerFn bind.SignerFn,
	lastUsedNonce int64,
) (*ethTypes.Transaction, error) {
	from := common.Address(sendArgs.ERC721TransferTx.From)

	useSafeTransferFrom := true
	inputParams := ProcessorInputParams{
		FromChain: &params.Network{
			ChainID: sendArgs.ChainID,
		},
		FromAddr: from,
		ToAddr:   sendArgs.ERC721TransferTx.Recipient,
		FromToken: &tokenTypes.Token{
			Symbol:  sendArgs.ERC721TransferTx.TokenID.String(),
			Address: common.Address(*sendArgs.ERC721TransferTx.To),
		},
	}
	err := h.checkIfFunctionExists(inputParams, erc721FunctionNameSafeTransferFrom)
	if err != nil {
		useSafeTransferFrom = false
	}

	ethClient, err := rpcClient.EthClient(sendArgs.ChainID)
	if err != nil {
		return nil, err
	}

	contract, err := collectibles.NewCollectibles(common.Address(*sendArgs.ERC721TransferTx.To), ethClient)
	if err != nil {
		return nil, err
	}

	var nonce uint64
	if lastUsedNonce < 0 {
		nonce, err = transactor.NextNonce(context.Background(), rpcClient, sendArgs.ChainID, sendArgs.ERC721TransferTx.From)
		if err != nil {
			return nil, err
		}
	} else {
		nonce = uint64(lastUsedNonce) + 1
	}

	argNonce := hexutil.Uint64(nonce)
	sendArgs.ERC721TransferTx.Nonce = &argNonce
	txOpts := sendArgs.ERC721TransferTx.ToTransactOpts(signerFn)

	var tx *ethTypes.Transaction
	if useSafeTransferFrom {
		tx, err = contract.SafeTransferFrom(txOpts, from,
			sendArgs.ERC721TransferTx.Recipient,
			sendArgs.ERC721TransferTx.TokenID.ToInt())
	} else {
		tx, err = contract.TransferFrom(txOpts, from,
			sendArgs.ERC721TransferTx.Recipient,
			sendArgs.ERC721TransferTx.TokenID.ToInt())
	}
	if err != nil {
		return nil, err
	}

	err = transactor.StoreAndTrackPendingTx(from, sendArgs.ERC721TransferTx.Symbol, sendArgs.ChainID, sendArgs.ERC721TransferTx.MultiTransactionID, tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (h *ERC721Handler) BuildTransactionV2(
	transactor transactions.TransactorIface,
	sendArgs *wallettypes.SendTxArgs,
	lastUsedNonce int64,
) (*ethTypes.Transaction, uint64, error) {
	return transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, *sendArgs, lastUsedNonce)
}

func (h *ERC721Handler) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return params.FromToken.Address, nil
}
