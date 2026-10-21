package alchemy

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/pkg/services/wallet/bigint"
)

// Larger decimals are treated as unknown.
const maxDecimals = 255

type HistoryDirection string

const (
	HistoryDirectionIn   HistoryDirection = "in"
	HistoryDirectionOut  HistoryDirection = "out"
	HistoryDirectionSelf HistoryDirection = "self"
)

// HistoryTransfer is one transfer of one token, seen from Address.
type HistoryTransfer struct {
	Address     common.Address   `json:"address"`
	ChainID     uint64           `json:"chainId"`
	BlockNumber uint64           `json:"blockNumber"`
	Timestamp   int64            `json:"timestamp"`
	Hash        common.Hash      `json:"hash"`
	UniqueID    string           `json:"uniqueId"`
	From        common.Address   `json:"from"`
	To          *common.Address  `json:"to"`
	Direction   HistoryDirection `json:"direction"`
	Category    TransferCategory `json:"category"`
	// Zero address for the native token.
	TokenAddress common.Address `json:"tokenAddress"`
	// Decimal string, NFTs only.
	TokenID  *string `json:"tokenId"`
	Symbol   string  `json:"symbol"`
	Decimals *uint64 `json:"decimals"`
	// Signed integer in base units: negative when the token leaves Address. Unsigned for self transfers.
	Amount string `json:"amount"`
	// Exact decimal of Amount, nil when the decimals are unknown.
	Value *string `json:"value"`
}

func ToHistoryTransfer(stored StoredTransfer) HistoryTransfer {
	t := stored.Transfer
	result := HistoryTransfer{
		Address:   stored.Address,
		ChainID:   stored.ChainID,
		Hash:      t.Hash,
		UniqueID:  t.UniqueID,
		From:      t.FromAddress,
		To:        t.ToAddress,
		Category:  t.Category,
		Symbol:    t.Asset,
		Timestamp: t.Metadata.BlockTimestamp.Unix(),
	}
	if t.BlockNum != nil && t.BlockNum.Int != nil {
		result.BlockNumber = t.BlockNum.Uint64()
	}

	switch {
	case t.FromAddress == stored.Address && t.ToAddress != nil && *t.ToAddress == stored.Address:
		result.Direction = HistoryDirectionSelf
	case t.FromAddress == stored.Address:
		result.Direction = HistoryDirectionOut
	default:
		result.Direction = HistoryDirectionIn
	}

	if t.Category != TransferCategoryExternal && t.RawContract.Address != nil {
		result.TokenAddress = *t.RawContract.Address
	}

	amount := big.NewInt(0)
	var decimals *uint64
	switch t.Category {
	case TransferCategoryErc721, TransferCategorySpecialNft:
		amount.SetInt64(1)
		decimals = new(uint64)
		result.TokenID = decimalString(t.TokenID)
	case TransferCategoryErc1155:
		decimals = new(uint64)
		if stored.Erc1155 != nil {
			if stored.Erc1155.Value != nil && stored.Erc1155.Value.Int != nil {
				amount.Set(stored.Erc1155.Value.Int)
			}
			result.TokenID = decimalString(stored.Erc1155.TokenID)
		}
	default:
		if t.RawContract.Value != nil && t.RawContract.Value.Int != nil {
			amount.Set(t.RawContract.Value.Int)
		}
		if t.RawContract.Decimal != nil && t.RawContract.Decimal.Int != nil && t.RawContract.Decimal.IsUint64() && t.RawContract.Decimal.Uint64() <= maxDecimals {
			d := t.RawContract.Decimal.Uint64()
			decimals = &d
		}
	}

	if result.Direction == HistoryDirectionOut {
		amount.Neg(amount)
	}
	result.Amount = amount.String()
	result.Decimals = decimals
	if decimals != nil {
		value := formatUnits(amount, *decimals)
		result.Value = &value
	}
	return result
}

func decimalString(v *bigint.VarHexBigInt) *string {
	if v == nil || v.Int == nil {
		return nil
	}
	s := v.String()
	return &s
}

// formatUnits returns amount / 10^decimals as an exact decimal string without trailing zeros.
func formatUnits(amount *big.Int, decimals uint64) string {
	abs := new(big.Int).Abs(amount)
	digits := abs.String()
	sign := ""
	if amount.Sign() < 0 {
		sign = "-"
	}
	if decimals == 0 {
		return sign + digits
	}
	if uint64(len(digits)) <= decimals {
		digits = strings.Repeat("0", int(decimals)-len(digits)+1) + digits
	}
	split := len(digits) - int(decimals)
	fraction := strings.TrimRight(digits[split:], "0")
	if fraction == "" {
		return sign + digits[:split]
	}
	return sign + digits[:split] + "." + fraction
}
