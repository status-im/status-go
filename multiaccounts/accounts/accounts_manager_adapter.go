package accounts

import (
	"github.com/status-im/status-go/accounts-management/types"
	ethtypes "github.com/status-im/status-go/eth-node/types"
)

type AccountsManagerPersistenceAdapter struct {
	db *Database
}

func NewAccountsManagerPersistenceAdapter(db *Database) *AccountsManagerPersistenceAdapter {
	return &AccountsManagerPersistenceAdapter{db: db}
}

func (a *AccountsManagerPersistenceAdapter) AddressExists(address ethtypes.Address) (bool, error) {
	return a.db.AddressExists(address)
}

func (a *AccountsManagerPersistenceAdapter) GetPath(address ethtypes.Address) (string, error) {
	return a.db.GetPath(address)
}

func (a *AccountsManagerPersistenceAdapter) GetWalletRootAddress() (ethtypes.Address, error) {
	address, err := a.db.GetWalletRootAddress()
	if err != nil {
		return ethtypes.ZeroAddress(), err
	}
	return address, nil
}

func (a *AccountsManagerPersistenceAdapter) GetProfileKeypair() (*types.Keypair, error) {
	dbKeypair, err := a.db.GetProfileKeypair()
	if err != nil {
		return nil, err
	}
	return KeypairToAccountsManagerKeypair(dbKeypair), nil
}

func (a *AccountsManagerPersistenceAdapter) GetKeypairByKeyUID(keyUID string) (*types.Keypair, error) {
	dbKeypair, err := a.db.GetKeypairByKeyUID(keyUID)
	if err != nil {
		if err == ErrDbKeypairNotFound {
			return nil, types.ErrDbKeypairNotFound
		}
		return nil, err
	}
	return KeypairToAccountsManagerKeypair(dbKeypair), nil
}

func (a *AccountsManagerPersistenceAdapter) GetActiveKeypairs() ([]*types.Keypair, error) {
	dbKeypairs, err := a.db.GetActiveKeypairs()
	if err != nil {
		return nil, err
	}
	return KeypairsToAccountsManagerKeypairs(dbKeypairs), nil
}

func (a *AccountsManagerPersistenceAdapter) SaveOrUpdateKeypair(keypair *types.Keypair) error {
	dbKeypair := AccountsManagerKeypairToKeypair(keypair)
	return a.db.SaveOrUpdateKeypair(dbKeypair)
}

func (a *AccountsManagerPersistenceAdapter) SaveOrUpdateKeycard(keycard *types.Keycard, clock uint64, updateKeypairClock bool) error {
	dbKeycard := AccountsManagerKeycardToKeycard(keycard)
	return a.db.SaveOrUpdateKeycard(*dbKeycard, clock, updateKeypairClock)
}

func (a *AccountsManagerPersistenceAdapter) MarkKeypairFullyOperable(keyUID string, clock uint64, updateKeypairClock bool) (err error) {
	return a.db.MarkKeypairFullyOperable(keyUID, clock, updateKeypairClock)
}

func (a *AccountsManagerPersistenceAdapter) MarkAccountFullyOperable(address ethtypes.Address) (err error) {
	return a.db.MarkAccountFullyOperable(address)
}

func (a *AccountsManagerPersistenceAdapter) DeleteAllKeycardsWithKeyUID(keyUID string, clock uint64) (err error) {
	return a.db.DeleteAllKeycardsWithKeyUID(keyUID, clock)
}

func (a *AccountsManagerPersistenceAdapter) SaveOrUpdateAccounts(accounts []*types.Account, updateKeypairClock bool) error {
	dbAccounts := AccountsManagerAccountsToAccounts(accounts)
	return a.db.SaveOrUpdateAccounts(dbAccounts, updateKeypairClock)
}

func (a *AccountsManagerPersistenceAdapter) GetPositionForNextNewAccount() (int64, error) {
	return a.db.GetPositionForNextNewAccount()
}

func (a *AccountsManagerPersistenceAdapter) GetAccountByAddress(address ethtypes.Address) (*types.Account, error) {
	dbAccount, err := a.db.GetAccountByAddress(address)
	if err != nil {
		return nil, err
	}
	return AccountToAccountsManagerAccount(dbAccount), nil
}

func (a *AccountsManagerPersistenceAdapter) RemoveAccount(address ethtypes.Address, clock uint64) error {
	return a.db.RemoveAccount(address, clock)
}

func (a *AccountsManagerPersistenceAdapter) RemoveKeypair(keyUID string, clock uint64) error {
	return a.db.RemoveKeypair(keyUID, clock)
}
