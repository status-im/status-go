package pathprocessor

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/contracts/cryptopunks"
	"github.com/status-im/status-go/rpc"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

const (
	cryptoPunksHandlerFunctionNameTransferPunk = "transferPunk"
)

var (
	// CryptoPunksContractID - CryptoPunks contract ID (mainnet)
	CryptoPunksContractID = thirdparty.ContractID{
		ChainID: walletCommon.ChainID(walletCommon.EthereumMainnet),
		Address: common.HexToAddress("0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB"),
	}
)

// CryptoPunksHandler handles CryptoPunks transfers
type CryptoPunksHandler struct {
	*BaseNFTHandler
}

func NewCryptoPunksHandler(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *CryptoPunksHandler {
	return &CryptoPunksHandler{
		BaseNFTHandler: NewBaseNFTHandler(ethClientGetter, transactor),
	}
}

func (h *CryptoPunksHandler) Name() string {
	return "CryptoPunksTransfer"
}

func (h *CryptoPunksHandler) CanHandle(contractID thirdparty.ContractID) bool {
	return contractID == CryptoPunksContractID
}

func (h *CryptoPunksHandler) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	parsedABI, err := abi.JSON(strings.NewReader(cryptopunks.CryptoPunksMetaData.ABI))
	if err != nil {
		return nil, err
	}

	tokenID, err := walletCommon.GetTokenIdFromSymbol(params.FromToken.Symbol)
	if err != nil {
		return nil, err
	}

	return parsedABI.Pack(cryptoPunksHandlerFunctionNameTransferPunk, params.ToAddr, tokenID)
}

func (h *CryptoPunksHandler) BuildTransactionV2(
	transactor transactions.TransactorIface,
	sendArgs *wallettypes.SendTxArgs,
	lastUsedNonce int64,
) (*ethTypes.Transaction, uint64, error) {
	cryptoPunksContractAddress := types.Address(CryptoPunksContractID.Address)

	fixedSendArgs := *sendArgs
	fixedSendArgs.To = &cryptoPunksContractAddress

	return transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, fixedSendArgs, lastUsedNonce)
}

func (h *CryptoPunksHandler) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return CryptoPunksContractID.Address, nil
}
