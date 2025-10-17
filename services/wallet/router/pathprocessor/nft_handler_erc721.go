package pathprocessor

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/contracts/erc721"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
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

func NewERC721Handler(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *ERC721Handler {
	return &ERC721Handler{
		BaseNFTHandler: NewBaseNFTHandler(ethClientGetter, transactor),
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

	ethClient, err := h.ethClientGetter.EthClient(params.FromChain.ChainID)
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
