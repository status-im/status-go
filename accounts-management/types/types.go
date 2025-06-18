package types

import (
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type PublicKeyData struct {
	CompressedKey string   `json:"compressedKey"`
	EmojiHash     []string `json:"emojiHash"`
}

// SelectedExtKey is a container for the selected (logged in) external account.
type SelectedExtKey struct {
	Address     ethtypes.Address
	AccountKey  *ethtypes.Key
	SubAccounts []Account
}

// Hex dumps address of a given extended key as hex string.
func (k *SelectedExtKey) Hex() string {
	if k == nil {
		return "0x0"
	}

	return k.Address.Hex()
}
