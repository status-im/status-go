package account

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/extkeys"
)

// errors
var (
	ErrInvalidAccountAddressOrKey  = errors.New("cannot parse address or key to valid account address")
	ErrInvalidMnemonicPhraseLength = errors.New("invalid mnemonic phrase length; valid lengths are 12, 15, 18, 21, and 24")
)

// Info contains wallet and chat addresses and public keys of an account.
type Info struct {
	WalletAddress string
	WalletPubKey  string
	ChatAddress   string
	ChatPubKey    string
}

// SelectedExtKey is a container for the selected (logged in) external account.
type SelectedExtKey struct {
	Address     common.Address
	AccountKey  *keystore.Key
	SubAccounts []accounts.Account
}

// Hex dumps address of a given extended key as hex string.
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

// MnemonicPhraseLengthToEntropyStrength returns the entropy strength for a given mnemonic length
func MnemonicPhraseLengthToEntropyStrength(length int) (extkeys.EntropyStrength, error) {
	if length < 12 || length > 24 || length%3 != 0 {
		return 0, ErrInvalidMnemonicPhraseLength
	}

	bitsLength := length * 11
	checksumLength := bitsLength % 32

	return extkeys.EntropyStrength(bitsLength - checksumLength), nil
}
