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

// FromAddress converts account address from string to common.Address.
// The function is useful to format "From" field of send transaction struct.
func FromAddress(accountAddress string) common.Address {
	from, err := ParseAccountString(accountAddress)
	if err != nil {
		return common.Address{}
	}

	return from.Address
}

// ToAddress converts account address from string to *common.Address.
// The function is useful to format "To" field of send transaction struct.
func ToAddress(accountAddress string) *common.Address {
	to, err := ParseAccountString(accountAddress)
	if err != nil {
		return nil
	}

	return &to.Address
}
