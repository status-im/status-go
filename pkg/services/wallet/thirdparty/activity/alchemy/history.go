package alchemy

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

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
	if stored.Erc1155 != nil {
		t.Erc1155Metadata = []Erc1155Metadata{*stored.Erc1155}
	}
	result := HistoryTransfer{
		Address:     stored.Address,
		ChainID:     stored.ChainID,
		BlockNumber: t.BlockNum.Uint64(),
		Timestamp:   t.Metadata.BlockTimestamp.Unix(),
		Hash:        t.Hash,
		UniqueID:    t.UniqueID,
		From:        t.FromAddress,
		To:          t.ToAddress,
		Category:    t.Category,
		Symbol:      t.Asset,
	}

	switch {
	case t.FromAddress == stored.Address && t.ToAddress != nil && *t.ToAddress == stored.Address:
		result.Direction = HistoryDirectionSelf
	case t.FromAddress == stored.Address:
		result.Direction = HistoryDirectionOut
	default:
		result.Direction = HistoryDirectionIn
	}

	amount := big.NewInt(0)
	if data := extractTransferTokenAndValue(t, stored.ChainID); len(data) > 0 {
		result.TokenAddress = data[0].Token.Address
		if data[0].Token.TokenID != nil {
			tokenID := data[0].Token.TokenID.ToInt().String()
			result.TokenID = &tokenID
		}
		if data[0].Value != nil {
			amount.Set(data[0].Value.ToInt())
		}
	}

	var decimals *uint64
	switch t.Category {
	case TransferCategoryErc721:
		amount.SetInt64(1)
		decimals = new(uint64)
	case TransferCategorySpecialNft:
		amount.SetInt64(1)
		decimals = new(uint64)
		if t.RawContract.Address != nil {
			result.TokenAddress = *t.RawContract.Address
		}
		if t.TokenID != nil {
			tokenID := t.TokenID.String()
			result.TokenID = &tokenID
		}
	case TransferCategoryErc1155:
		decimals = new(uint64)
	default:
		if t.RawContract.Decimal != nil {
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

// formatUnits returns amount / 10^decimals as an exact decimal string without trailing zeros.
func formatUnits(amount *big.Int, decimals uint64) string {
	denominator := new(big.Int).Exp(big.NewInt(10), new(big.Int).SetUint64(decimals), nil)
	value := new(big.Rat).SetFrac(amount, denominator).FloatString(int(decimals))
	if strings.Contains(value, ".") {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	return value
}
