package account

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/eth-node/types"
)

// errors
var (
	ErrInvalidAccountAddressOrKey  = errors.New("cannot parse address or key to valid account address")
	ErrInvalidMnemonicPhraseLength = errors.New("invalid mnemonic phrase length; valid lengths are 12, 15, 18, 21, and 24")
)

type LoginParams struct {
	ChatAddress    types.Address   `json:"chatAddress"`
	Password       string          `json:"password"`
	MainAccount    types.Address   `json:"mainAccount"`
	WatchAddresses []types.Address `json:"watchAddresses"`
}

type ErrZeroAddress struct {
	field string
}

func (e *ErrZeroAddress) Error() string {
	return fmt.Sprintf("%s contains an empty address", e.field)
}

func newErrZeroAddress(field string) *ErrZeroAddress {
	return &ErrZeroAddress{
		field: field,
	}
}

func ParseLoginParams(paramsJSON string) (LoginParams, error) {
	var (
		params      LoginParams
		zeroAddress types.Address
	)
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return params, err
	}

	if params.ChatAddress == zeroAddress {
		return params, newErrZeroAddress("ChatAddress")
	}

	if params.MainAccount == zeroAddress {
		return params, newErrZeroAddress("MainAccount")
	}

	for _, address := range params.WatchAddresses {
		if address == zeroAddress {
			return params, newErrZeroAddress("WatchAddresses")
		}
	}
	return params, nil
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
	Address     types.Address
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
	if types.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}

	return accounts.Account{}, ErrInvalidAccountAddressOrKey
}

// GethFromAddress converts account address from string to common.Address.
// The function is useful to format "From" field of send transaction struct.
func GethFromAddress(accountAddress string) common.Address {
	from, err := ParseAccountString(accountAddress)
	if err != nil {
		return common.Address{}
	}

	return from.Address
}

// FromAddress converts account address from string to types.Address.
// The function is useful to format "From" field of send transaction struct.
func FromAddress(accountAddress string) types.Address {
	from, err := ParseAccountString(accountAddress)
	if err != nil {
		return types.Address{}
	}

	return types.Address(from.Address)
}

// GethToAddress converts account address from string to *common.Address.
// The function is useful to format "To" field of send transaction struct.
func GethToAddress(accountAddress string) *common.Address {
	to, err := ParseAccountString(accountAddress)
	if err != nil {
		return nil
	}

	return &to.Address
}
