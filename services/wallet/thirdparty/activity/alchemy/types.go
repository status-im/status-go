package alchemy

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/services/wallet/bigint"
)

const getAssetTransfersMethod = "alchemy_getAssetTransfers"
const MaxAssetTransfersCount = 1000

// AlchemyBigInt wraps VarHexBigInt to handle Alchemy API inconsistency
// it sometimes returns plain decimal numbers instead of hexadecimal
type AlchemyBigInt struct {
	bigint.VarHexBigInt
}

func (a *AlchemyBigInt) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// alchemy sometimes returns plain numbers like "0" or "1" without "0x" prefix
	// so we need to take this into account
	if len(str) >= 2 && str[0:2] == "0x" {
		// hex string case - unmarshal directly into embedded struct
		return a.VarHexBigInt.UnmarshalJSON(data)
	} else {
		// plain decimal number
		val := new(big.Int)
		if _, ok := val.SetString(str, 10); !ok {
			return fmt.Errorf("invalid decimal number: %s", str)
		}
		a.VarHexBigInt.Int = val
		return nil
	}
}

type TransferCategory string

const (
	TransferCategoryExternal   TransferCategory = "external"
	TransferCategoryInternal   TransferCategory = "internal"
	TransferCategoryErc20      TransferCategory = "erc20"
	TransferCategoryErc721     TransferCategory = "erc721"
	TransferCategoryErc1155    TransferCategory = "erc1155"
	TransferCategorySpecialNft TransferCategory = "specialnft"
)

type TransferOrder string

const (
	TransferOrderOldToNew TransferOrder = "asc"
	TransferOrderNewToOld TransferOrder = "desc"
)

type GetAssetTransfersParams struct {
	FromBlock         *rpc.BlockNumber   `json:"fromBlock,omitempty"` // Defaults to "0x0"
	ToBlock           *rpc.BlockNumber   `json:"toBlock,omitempty"`   // Defaults to "latest"
	FromAddress       *common.Address    `json:"fromAddress,omitempty"`
	ToAddress         *common.Address    `json:"toAddress,omitempty"`
	ContractAddresses []common.Address   `json:"contractAddresses,omitempty"`
	Category          []TransferCategory `json:"category"`
	Order             TransferOrder      `json:"order"`
	WithMetadata      bool               `json:"withMetadata"`
	ExcludeZeroValue  bool               `json:"excludeZeroValue"`
	MaxCount          *hexutil.Big       `json:"maxCount"`
	PageKey           string             `json:"pageKey,omitempty"`
}

type GetAssetTranfersResponse struct {
	Transfers []Transfer `json:"transfers"`
	PageKey   string     `json:"pageKey"`
}

type Transfer struct {
	Category        TransferCategory     `json:"category"`
	BlockNum        *bigint.VarHexBigInt `json:"blockNum"`
	FromAddress     common.Address       `json:"from"`
	ToAddress       *common.Address      `json:"to,omitempty"`
	Value           float64              `json:"value,omitempty"`
	Erc1155Metadata []Erc1155Metadata    `json:"erc1155Metadata,omitempty"`
	TokenID         *AlchemyBigInt       `json:"tokenId"`
	Asset           string               `json:"asset"`
	UniqueID        string               `json:"uniqueId"`
	Hash            common.Hash          `json:"hash"`
	RawContract     RawContract          `json:"rawContract"`
	Metadata        Metadata             `json:"metadata"`
}

func (t Transfer) String() string {
	return fmt.Sprintf("%s %8s %5s %13.7f %s->%s, %s", t.Hash.TerminalString(), t.Category, t.Asset, t.Value, t.FromAddress.Hex(), addressToKnown(t.ToAddress.Hex()), t.RawContract.Address)
}

func (t Transfer) IsContractDeployment() bool {
	return t.Category == TransferCategoryExternal && t.ToAddress == nil
}

func (t Transfer) IsIncoming(accountAddress common.Address) bool {
	return t.FromAddress != accountAddress
}

type Erc1155Metadata struct {
	TokenID *AlchemyBigInt       `json:"tokenId"`
	Value   *bigint.VarHexBigInt `json:"value"`
}

type RawContract struct {
	Value   *bigint.VarHexBigInt `json:"value"`   // nil if ERC721 or ERC1155 transfer
	Address *common.Address      `json:"address"` // nil if external or internal transfer
	Decimal *bigint.VarHexBigInt `json:"decimal"` // nil if not available in the contract
}

type Metadata struct {
	BlockTimestamp time.Time `json:"blockTimestamp"`
}
