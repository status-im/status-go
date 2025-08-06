package accountsmanagement

//go:generate mockgen -package=mock_persistence -source=persistence.go -destination=mock/persistence.go

import (
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

type Persistence interface {
	// AddressExists checks if the address exists in the persistence
	AddressExists(address ethtypes.Address) (bool, error)
	// GetProfileKeypair returns the profile keypair
	GetProfileKeypair() (*accounts.Keypair, error)
	// GetWalletRootAddress returns the root address of the wallet
	GetWalletRootAddress() (ethtypes.Address, error)
	// GetPath returns the derivation path of the address
	GetPath(address ethtypes.Address) (string, error)
}
