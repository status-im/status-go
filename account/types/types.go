package types

import (
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
)

type LoginParams struct {
	ChatAddress  ethtypes.Address       `json:"chatAddress"`
	Password     string                 `json:"password"`
	MainAccount  ethtypes.Address       `json:"mainAccount"`
	MultiAccount *multiaccounts.Account `json:"multiAccount"`
}

type PublicKeyData struct {
	CompressedKey string   `json:"compressedKey"`
	EmojiHash     []string `json:"emojiHash"`
}

// Info contains wallet and chat addresses and public keys of an account.
type Info struct {
	WalletAddress string
	WalletPubKey  string
	ChatAddress   string
	ChatPubKey    string
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
