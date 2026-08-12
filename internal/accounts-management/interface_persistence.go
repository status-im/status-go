package accountsmanagement

//go:generate go tool mockgen -package=mock_persistence -source=interface_persistence.go -destination=./mock/persistence.go

import (
	"github.com/status-im/status-go/internal/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

type Persistence interface {
	// AddressExists checks if the address exists in the persistence
	AddressExists(address cryptotypes.Address) (bool, error)
	// GetProfileKeypair returns the profile keypair
	GetProfileKeypair() (*types.Keypair, error)
	// GetWalletRootAddress returns the root address of the wallet
	GetWalletRootAddress() (cryptotypes.Address, error)
	// GetKeypairByKeyUID returns a keypair by its keyUID
	GetKeypairByKeyUID(keyUID string) (*types.Keypair, error)
	// GetActiveKeypairs returns all active keypairs
	GetActiveKeypairs() ([]*types.Keypair, error)
	// GetAllKeypairs returns all keypairs
	GetAllKeypairs() ([]*types.Keypair, error)
	// SaveOrUpdateKeypair saves or updates a keypair and its accounts
	SaveOrUpdateKeypair(keypair *types.Keypair) error
	// SaveOrUpdateAccounts saves or updates accounts
	SaveOrUpdateAccounts(accounts []*types.Account, updateKeypairClock bool) error
	// UpdateKeypairXPub updates the xpub and cold-wallet flag of a keypair
	UpdateKeypairXPub(keyUID string, xpub string, coldWallet types.ColdWalletType, clock uint64) error
	// MarkKeypairFullyOperable marks a keypair as fully operable
	MarkKeypairFullyOperable(keyUID string, clock uint64, updateKeypairClock bool) (err error)
	// MarkAccountFullyOperable marks an account as fully operable
	MarkAccountFullyOperable(address cryptotypes.Address) (err error)
	// GetPositionForNextNewAccount returns the position for the next new account
	GetPositionForNextNewAccount() (int64, error)
	// GetAccountByAddress returns an account by its address
	GetAccountByAddress(address cryptotypes.Address) (*types.Account, error)
	// RemoveAccount removes an account
	RemoveAccount(address cryptotypes.Address, clock uint64) error
	// RemoveKeypair removes a keypair
	RemoveKeypair(keyUID string, clock uint64) error
}
