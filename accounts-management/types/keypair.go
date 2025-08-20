package types

import (
	"errors"

	"github.com/status-im/status-go/crypto/types"
)

var (
	ErrDbKeypairNotFound = errors.New("keypair is not found")
)

type KeypairType string
type AccountType string
type AccountOperable string

const (
	KeypairTypeProfile KeypairType = "profile"
	KeypairTypeKey     KeypairType = "key"
	KeypairTypeSeed    KeypairType = "seed"
)

const (
	AccountTypeGenerated AccountType = "generated"
	AccountTypeKey       AccountType = "key"
	AccountTypeSeed      AccountType = "seed"
	AccountTypeWatch     AccountType = "watch"
)

const (
	AccountNonOperable       AccountOperable = "no"        // an account is non operable it is not a keycard account and there is no keystore file for it and no keystore file for the address it is derived from
	AccountPartiallyOperable AccountOperable = "partially" // an account is partially operable if it is not a keycard account and there is created keystore file for the address it is derived from
	AccountFullyOperable     AccountOperable = "fully"     // an account is fully operable if it is not a keycard account and there is a keystore file for it

)

type Keypair struct {
	KeyUID                  string      `json:"key-uid"`
	Name                    string      `json:"name"`
	Type                    KeypairType `json:"type"`
	DerivedFrom             string      `json:"derived-from"`
	LastUsedDerivationIndex uint64      `json:"last-used-derivation-index,omitempty"`
	SyncedFrom              string      `json:"synced-from,omitempty"` // keeps an info which device this keypair is added from can be one of two values defined in constants or device name (custom)
	Clock                   uint64      `json:"clock,omitempty"`
	Accounts                []*Account  `json:"accounts,omitempty"`
	Keycards                []*Keycard  `json:"keycards,omitempty"`
	Removed                 bool        `json:"removed,omitempty"`
}

type AccountCreationDetails struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Emoji   string `json:"emoji,omitempty"`
	ColorID string `json:"colorId,omitempty"`
}

type Account struct {
	Address               types.Address   `json:"address"`
	KeyUID                string          `json:"key-uid"`
	Wallet                bool            `json:"wallet"`
	AddressWasNotShown    bool            `json:"address-was-not-shown,omitempty"`
	Chat                  bool            `json:"chat"`
	Type                  AccountType     `json:"type,omitempty"`
	Path                  string          `json:"path,omitempty"`
	PublicKey             types.HexBytes  `json:"public-key,omitempty"`
	Name                  string          `json:"name"`
	Emoji                 string          `json:"emoji"`
	ColorID               string          `json:"colorId,omitempty"`
	Hidden                bool            `json:"hidden"`
	Clock                 uint64          `json:"clock,omitempty"`
	Removed               bool            `json:"removed,omitempty"`
	Operable              AccountOperable `json:"operable"` // describes an account's operability (check AccountOperable type constants for details)
	CreatedAt             int64           `json:"createdAt"`
	Position              int64           `json:"position"`
	ProdPreferredChainIDs string          `json:"prodPreferredChainIds"`
	TestPreferredChainIDs string          `json:"testPreferredChainIds"`
}

type Keycard struct {
	KeycardUID        string          `json:"keycard-uid"`
	KeycardName       string          `json:"keycard-name"`
	KeycardLocked     bool            `json:"keycard-locked"`
	AccountsAddresses []types.Address `json:"accounts-addresses"`
	KeyUID            string          `json:"key-uid"`
	Position          uint64
}

func (a *Keypair) MigratedToKeycard() bool {
	return len(a.Keycards) > 0
}

// Returns operability of a keypair:
// - if any of keypair's account is not operable, then a keyapir is considered as non operable
// - if any of keypair's account is partially operable, then a keyapir is considered as partially operable
// - if all accounts are fully operable, then a keyapir is considered as fully operable
func (a *Keypair) Operability() AccountOperable {
	for _, acc := range a.Accounts {
		if acc.Operable == AccountNonOperable {
			return AccountNonOperable
		}
		if acc.Operable == AccountPartiallyOperable {
			return AccountPartiallyOperable
		}
	}

	return AccountFullyOperable
}
