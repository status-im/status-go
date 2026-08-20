package pathprocessor

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/internal/contracts/cryptokitties"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/transactions"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
)

const (
	cryptoKittiesFunctionNameTransfer = "transfer"
)

var (
	// CryptoKittiesContractID - CryptoKitties contract ID (mainnet)
	CryptoKittiesContractID = thirdparty.ContractID{
		ChainID: walletCommon.ChainID(walletCommon.EthereumMainnet),
		Address: common.HexToAddress("0x06012c8cf97bead5deae237070f9587f8e7a266d"),
	}
)

// CryptoKittiesHandler handles CryptoKitties transfers
type CryptoKittiesHandler struct {
	*BaseNFTHandler
}

func NewCryptoKittiesHandler(ethClientGetter rpc.EthClientGetter, transactor transactions.TransactorIface) *CryptoKittiesHandler {
	return &CryptoKittiesHandler{
		BaseNFTHandler: NewBaseNFTHandler(ethClientGetter, transactor),
	}
}

func (h *CryptoKittiesHandler) Name() string {
	return "CryptoKittiesTransfer"
}

func (h *CryptoKittiesHandler) CanHandle(contractID thirdparty.ContractID) bool {
	return contractID == CryptoKittiesContractID
}

func (h *CryptoKittiesHandler) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	if params.FromToken == nil || params.FromToken.CollectibleTokenID == nil {
		return nil, ErrNoTokenSet
	}

	parsedABI, err := abi.JSON(strings.NewReader(cryptokitties.CryptoKittiesMetaData.ABI))
	if err != nil {
		return nil, err
	}

	return parsedABI.Pack(cryptoKittiesFunctionNameTransfer, params.ToAddr, params.FromToken.CollectibleTokenID.ToInt())
}

func (h *CryptoKittiesHandler) BuildTransactionV2(
	transactor transactions.TransactorIface,
	sendArgs *wallettypes.SendTxArgs,
	lastUsedNonce int64,
) (*ethTypes.Transaction, uint64, error) {
	cryptoKittiesContractAddress := types.Address(CryptoKittiesContractID.Address)

	fixedSendArgs := *sendArgs
	fixedSendArgs.To = &cryptoKittiesContractAddress

	return transactor.ValidateAndBuildTransaction(sendArgs.FromChainID, fixedSendArgs, lastUsedNonce)
}

func (h *CryptoKittiesHandler) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	return CryptoKittiesContractID.Address, nil
}
