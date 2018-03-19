package account

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

// SelectedExtKey is a container for currently selected (logged in) account
type SelectedExtKey struct {
	Address     common.Address
	AccountKey  *keystore.Key
	SubAccounts []accounts.Account
}

// Hex dumps address of a given extended key as hex string
func (k *SelectedExtKey) Hex() string {
	if k == nil {
		return "0x0"
	}

	return k.Address.Hex()
}

// ParseAccountString parses hex encoded string and returns is as accounts.Account.
func ParseAccountString(account string) (accounts.Account, error) {
	// valid address, convert to account
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}

	return accounts.Account{}, ErrInvalidAccountAddressOrKey
}
