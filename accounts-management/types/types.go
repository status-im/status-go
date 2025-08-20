package types

import cryptotypes "github.com/status-im/status-go/crypto/types"

type PublicKeyData struct {
	CompressedKey string   `json:"compressedKey"`
	EmojiHash     []string `json:"emojiHash"`
}

// SelectedExtKey is a container for the selected (logged in) external account.
type SelectedExtKey struct {
	Address     cryptotypes.Address
	AccountKey  *Key
	SubAccounts []Account
}

// Hex dumps address of a given extended key as hex string.
func (k *SelectedExtKey) Hex() string {
	if k == nil {
		return "0x0"
	}

	return k.Address.Hex()
}
