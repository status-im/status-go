package persistence

//go:generate mockgen -package=mock_persistence -source=interface.go -destination=../mock/persistence.go

import (
	"github.com/status-im/status-go/accounts-management/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type Persistence interface {
	// AddressExists checks if the address exists in the persistence
	AddressExists(address ethtypes.Address) (bool, error)
	// GetProfileKeypair returns the profile keypair
	GetProfileKeypair() (*types.Keypair, error)
	// GetWalletRootAddress returns the root address of the wallet
	GetWalletRootAddress() (ethtypes.Address, error)
	// GetPath returns the derivation path of the address
	GetPath(address ethtypes.Address) (string, error)
	// GetKeypairByKeyUID returns a keypair by its keyUID
	GetKeypairByKeyUID(keyUID string) (*types.Keypair, error)
	// GetActiveKeypairs returns all active keypairs
	GetActiveKeypairs() ([]*types.Keypair, error)
	// SaveOrUpdateKeypair saves or updates a keypair and its accounts
	SaveOrUpdateKeypair(keypair *types.Keypair) error
	// SaveOrUpdateAccounts saves or updates accounts
	SaveOrUpdateAccounts(accounts []*types.Account, updateKeypairClock bool) error
	// SaveOrUpdateKeycard saves or updates a keycard and its accounts
	SaveOrUpdateKeycard(keycard *types.Keycard, clock uint64, updateKeypairClock bool) error
	// MarkKeypairFullyOperable marks a keypair as fully operable
	MarkKeypairFullyOperable(keyUID string, clock uint64, updateKeypairClock bool) (err error)
	// MarkAccountFullyOperable marks an account as fully operable
	MarkAccountFullyOperable(address ethtypes.Address) (err error)
	// DeleteAllKeycardsWithKeyUID deletes all keycards with a given keyUID
	DeleteAllKeycardsWithKeyUID(keyUID string, clock uint64) (err error)
	// GetPositionForNextNewAccount returns the position for the next new account
	GetPositionForNextNewAccount() (int64, error)
	// GetAccountByAddress returns an account by its address
	GetAccountByAddress(address ethtypes.Address) (*types.Account, error)
	// RemoveAccount removes an account
	RemoveAccount(address ethtypes.Address, clock uint64) error
	// RemoveKeypair removes a keypair
	RemoveKeypair(keyUID string, clock uint64) error
}
