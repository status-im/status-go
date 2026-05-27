package types

import (
	"encoding/json"
	"errors"

	types "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	ErrDbKeypairNotFound = errors.New("keypair is not found")
)

type KeypairType string
type AccountType string
type AccountOperable string
type ColdWalletType string

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

const (
	ColdWalletTypeNone          ColdWalletType = ""
	ColdWalletTypeStatusKeycard ColdWalletType = "status-keycard"
	ColdWalletTypeLedger        ColdWalletType = "ledger"
	ColdWalletTypeTrezor        ColdWalletType = "trezor"
)

type Keypair struct {
	KeyUID                  string         `json:"key-uid"`
	Name                    string         `json:"name"`
	Type                    KeypairType    `json:"type"`
	DerivedFrom             string         `json:"derived-from"`
	LastUsedDerivationIndex uint64         `json:"last-used-derivation-index,omitempty"`
	SyncedFrom              string         `json:"synced-from,omitempty"` // keeps an info which device this keypair is added from can be one of two values defined in constants or device name (custom)
	Clock                   uint64         `json:"clock,omitempty"`
	Accounts                []*Account     `json:"accounts,omitempty"`
	Keycards                []*Keycard     `json:"keycards,omitempty"`
	Removed                 bool           `json:"removed,omitempty"`
	XPub                    string         `json:"xpub,omitempty"`
	ColdWallet              ColdWalletType `json:"cold-wallet,omitempty"`
}

type AccountCreationDetails struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Emoji   string `json:"emoji,omitempty"`
	ColorID string `json:"colorId,omitempty"`
}

type Account struct {
	Address               types.Address             `json:"address"`
	KeyUID                string                    `json:"key-uid"`
	Wallet                bool                      `json:"wallet"`
	AddressWasNotShown    bool                      `json:"address-was-not-shown,omitempty"`
	Chat                  bool                      `json:"chat"`
	Type                  AccountType               `json:"type,omitempty"`
	Path                  string                    `json:"path,omitempty"`
	PublicKey             types.HexBytes            `json:"public-key,omitempty"`
	Name                  string                    `json:"name"`
	Emoji                 string                    `json:"emoji"`
	ColorID               common.CustomizationColor `json:"colorId,omitempty"`
	Hidden                bool                      `json:"hidden"`
	Clock                 uint64                    `json:"clock,omitempty"`
	Removed               bool                      `json:"removed,omitempty"`
	Operable              AccountOperable           `json:"operable"` // describes an account's operability (check AccountOperable type constants for details)
	CreatedAt             int64                     `json:"createdAt"`
	Position              int64                     `json:"position"`
	ProdPreferredChainIDs string                    `json:"prodPreferredChainIds"`
	TestPreferredChainIDs string                    `json:"testPreferredChainIds"`
}

type Keycard struct {
	KeycardUID        string          `json:"keycard-uid"`
	KeycardName       string          `json:"keycard-name"`
	KeycardLocked     bool            `json:"keycard-locked"`
	AccountsAddresses []types.Address `json:"accounts-addresses"`
	KeyUID            string          `json:"key-uid"`
	Position          uint64
}

// Returns true if an account is a wallet account that logged in user has a control over, otherwise returns false.
func (a *Account) IsWalletNonWatchOnlyAccount() bool {
	return !a.Chat && len(a.Type) > 0 && a.Type != AccountTypeWatch
}

// Returns true if an account is a wallet account that is ready for sending transactions, otherwise returns false.
func (a *Account) IsWalletAccountReadyForTransaction() bool {
	return a.IsWalletNonWatchOnlyAccount() && a.Operable != AccountNonOperable
}

func (a *Account) MarshalJSON() ([]byte, error) {
	item := struct {
		Address               types.Address             `json:"address"`
		MixedcaseAddress      string                    `json:"mixedcase-address"`
		KeyUID                string                    `json:"key-uid"`
		Wallet                bool                      `json:"wallet"`
		Chat                  bool                      `json:"chat"`
		Type                  AccountType               `json:"type"`
		Path                  string                    `json:"path"`
		PublicKey             types.HexBytes            `json:"public-key"`
		Name                  string                    `json:"name"`
		Emoji                 string                    `json:"emoji"`
		ColorID               common.CustomizationColor `json:"colorId"`
		Hidden                bool                      `json:"hidden"`
		Clock                 uint64                    `json:"clock"`
		Removed               bool                      `json:"removed"`
		Operable              AccountOperable           `json:"operable"`
		CreatedAt             int64                     `json:"createdAt"`
		Position              int64                     `json:"position"`
		ProdPreferredChainIDs string                    `json:"prodPreferredChainIds"`
		TestPreferredChainIDs string                    `json:"testPreferredChainIds"`
	}{
		Address:               a.Address,
		MixedcaseAddress:      a.Address.Hex(),
		KeyUID:                a.KeyUID,
		Wallet:                a.Wallet,
		Chat:                  a.Chat,
		Type:                  a.Type,
		Path:                  a.Path,
		PublicKey:             a.PublicKey,
		Name:                  a.Name,
		Emoji:                 a.Emoji,
		ColorID:               a.ColorID,
		Hidden:                a.Hidden,
		Clock:                 a.Clock,
		Removed:               a.Removed,
		Operable:              a.Operable,
		CreatedAt:             a.CreatedAt,
		Position:              a.Position,
		ProdPreferredChainIDs: a.ProdPreferredChainIDs,
		TestPreferredChainIDs: a.TestPreferredChainIDs,
	}

	return json.Marshal(item)
}

func (a *Keypair) MarshalJSON() ([]byte, error) {
	item := struct {
		KeyUID                  string         `json:"key-uid"`
		Name                    string         `json:"name"`
		Type                    KeypairType    `json:"type"`
		DerivedFrom             string         `json:"derived-from"`
		LastUsedDerivationIndex uint64         `json:"last-used-derivation-index"`
		SyncedFrom              string         `json:"synced-from"`
		Clock                   uint64         `json:"clock"`
		Accounts                []*Account     `json:"accounts"`
		Keycards                []*Keycard     `json:"keycards"`
		Removed                 bool           `json:"removed"`
		XPub                    string         `json:"xpub"`
		ColdWallet              ColdWalletType `json:"cold-wallet"`
	}{
		KeyUID:                  a.KeyUID,
		Name:                    a.Name,
		Type:                    a.Type,
		DerivedFrom:             a.DerivedFrom,
		LastUsedDerivationIndex: a.LastUsedDerivationIndex,
		SyncedFrom:              a.SyncedFrom,
		Clock:                   a.Clock,
		Accounts:                a.Accounts,
		Keycards:                a.Keycards,
		Removed:                 a.Removed,
		XPub:                    a.XPub,
		ColdWallet:              a.ColdWallet,
	}

	return json.Marshal(item)
}

func (a *Keypair) CopyKeypair() *Keypair {
	kp := &Keypair{
		Clock:                   a.Clock,
		KeyUID:                  a.KeyUID,
		Name:                    a.Name,
		Type:                    a.Type,
		DerivedFrom:             a.DerivedFrom,
		LastUsedDerivationIndex: a.LastUsedDerivationIndex,
		SyncedFrom:              a.SyncedFrom,
		Accounts:                make([]*Account, len(a.Accounts)),
		Keycards:                make([]*Keycard, len(a.Keycards)),
		Removed:                 a.Removed,
		XPub:                    a.XPub,
		ColdWallet:              a.ColdWallet,
	}

	for i, acc := range a.Accounts {
		kp.Accounts[i] = &Account{
			Address:               acc.Address,
			KeyUID:                acc.KeyUID,
			Wallet:                acc.Wallet,
			Chat:                  acc.Chat,
			Type:                  acc.Type,
			Path:                  acc.Path,
			PublicKey:             acc.PublicKey,
			Name:                  acc.Name,
			Emoji:                 acc.Emoji,
			ColorID:               acc.ColorID,
			Hidden:                acc.Hidden,
			Clock:                 acc.Clock,
			Removed:               acc.Removed,
			Operable:              acc.Operable,
			CreatedAt:             acc.CreatedAt,
			Position:              acc.Position,
			ProdPreferredChainIDs: acc.ProdPreferredChainIDs,
			TestPreferredChainIDs: acc.TestPreferredChainIDs,
		}
	}

	for i, kc := range a.Keycards {
		kp.Keycards[i] = &Keycard{
			KeycardUID:        kc.KeycardUID,
			KeycardName:       kc.KeycardName,
			KeycardLocked:     kc.KeycardLocked,
			AccountsAddresses: kc.AccountsAddresses,
			KeyUID:            kc.KeyUID,
		}
	}

	return kp
}

func (a *Keypair) GetChatAccount() *Account {
	for _, acc := range a.Accounts {
		if acc.Chat {
			return acc
		}
	}
	return nil
}

func (a *Keypair) MigratedToColdWallet() bool {
	return a.ColdWallet != ColdWalletTypeNone
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

func GetAccountTypeForKeypairType(kpType KeypairType) AccountType {
	switch kpType {
	case KeypairTypeProfile:
		return AccountTypeGenerated
	case KeypairTypeKey:
		return AccountTypeKey
	case KeypairTypeSeed:
		return AccountTypeSeed
	default:
		return AccountTypeWatch
	}
}

func (kp *Keycard) ToSyncKeycard() *protobuf.SyncKeycard {
	kc := &protobuf.SyncKeycard{
		Uid:      kp.KeycardUID,
		Name:     kp.KeycardName,
		Locked:   kp.KeycardLocked,
		KeyUid:   kp.KeyUID,
		Position: kp.Position,
	}

	for _, addr := range kp.AccountsAddresses {
		kc.Addresses = append(kc.Addresses, addr.Bytes())
	}

	return kc
}
