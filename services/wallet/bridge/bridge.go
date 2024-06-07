package bridge

import (
	"encoding/hex"
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

var (
	ZeroAddress     = common.Address{}
	ZeroBigIntValue = big.NewInt(0)
)

const (
	IncreaseEstimatedGasFactor = 1.1

	EthSymbol = "ETH"
	SntSymbol = "SNT"
	SttSymbol = "STT"

	TransferName        = "Transfer"
	HopName             = "Hop"
	CBridgeName         = "CBridge"
	SwapParaswapName    = "Paraswap"
	ERC721TransferName  = "ERC721Transfer"
	ERC1155TransferName = "ERC1155Transfer"
	ENSRegisterName     = "ENSRegister"
	ENSReleaseName      = "ENSRelease"
)

func getSigner(chainID uint64, from types.Address, verifiedAccount *account.SelectedExtKey) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, verifiedAccount.AccountKey.PrivateKey)
	}
}

type TransactionBridge struct {
	BridgeName        string
	ChainID           uint64
	TransferTx        *transactions.SendTxArgs
	HopTx             *HopTxArgs
	CbridgeTx         *CBridgeTxArgs
	ERC721TransferTx  *ERC721TransferTxArgs
	ERC1155TransferTx *ERC1155TransferTxArgs
	SwapTx            *SwapTxArgs
}

func (t *TransactionBridge) Value() *big.Int {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return t.TransferTx.Value.ToInt()
	} else if t.HopTx != nil {
		return t.HopTx.Amount.ToInt()
	} else if t.CbridgeTx != nil {
		return t.CbridgeTx.Amount.ToInt()
	} else if t.ERC721TransferTx != nil {
		return big.NewInt(1)
	} else if t.ERC1155TransferTx != nil {
		return t.ERC1155TransferTx.Amount.ToInt()
	}

	return big.NewInt(0)
}

func (t *TransactionBridge) From() types.Address {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return t.TransferTx.From
	} else if t.HopTx != nil {
		return t.HopTx.From
	} else if t.CbridgeTx != nil {
		return t.CbridgeTx.From
	} else if t.ERC721TransferTx != nil {
		return t.ERC721TransferTx.From
	} else if t.ERC1155TransferTx != nil {
		return t.ERC1155TransferTx.From
	}

	return types.HexToAddress("0x0")
}

func (t *TransactionBridge) To() types.Address {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return *t.TransferTx.To
	} else if t.HopTx != nil {
		return types.Address(t.HopTx.Recipient)
	} else if t.CbridgeTx != nil {
		return types.Address(t.HopTx.Recipient)
	} else if t.ERC721TransferTx != nil {
		return types.Address(t.ERC721TransferTx.Recipient)
	} else if t.ERC1155TransferTx != nil {
		return types.Address(t.ERC1155TransferTx.Recipient)
	}

	return types.HexToAddress("0x0")
}

func (t *TransactionBridge) Data() types.HexBytes {
	if t.TransferTx != nil && t.TransferTx.To != nil {
		return t.TransferTx.Data
	} else if t.HopTx != nil {
		return types.HexBytes("")
	} else if t.CbridgeTx != nil {
		return types.HexBytes("")
	} else if t.ERC721TransferTx != nil {
		return types.HexBytes("")
	} else if t.ERC1155TransferTx != nil {
		return types.HexBytes("")
	}

	return types.HexBytes("")
}

type BridgeParams struct {
	FromChain *params.Network
	ToChain   *params.Network
	FromAddr  common.Address
	ToAddr    common.Address
	FromToken *token.Token
	ToToken   *token.Token
	AmountIn  *big.Int

	// extra params
	BonderFee *big.Int
	Username  string
	PublicKey string
}

type Bridge interface {
	// returns the name of the bridge
	Name() string
	// checks if the bridge is available for the given networks/tokens
	AvailableFor(params BridgeParams) (bool, error)
	// calculates the fees for the bridge and returns the amount BonderFee and TokenFee (used for bridges)
	CalculateFees(params BridgeParams) (*big.Int, *big.Int, error)
	// Pack the method for sending tx and method call's data
	PackTxInputData(params BridgeParams, contractType string) ([]byte, error)
	EstimateGas(params BridgeParams) (uint64, error)
	CalculateAmountOut(params BridgeParams) (*big.Int, error)
	Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error)
	GetContractAddress(params BridgeParams) (common.Address, error)
	BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error)
	BuildTx(params BridgeParams) (*ethTypes.Transaction, error)
}

func extractCoordinates(pubkey string) ([32]byte, [32]byte) {
	x, _ := hex.DecodeString(pubkey[4:68])
	y, _ := hex.DecodeString(pubkey[68:132])

	var xByte [32]byte
	copy(xByte[:], x)

	var yByte [32]byte
	copy(yByte[:], y)

	return xByte, yByte
}

func usernameToLabel(username string) [32]byte {
	usernameHashed := crypto.Keccak256([]byte(username))
	var label [32]byte
	copy(label[:], usernameHashed)

	return label
}
