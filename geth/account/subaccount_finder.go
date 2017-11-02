package account

import (
	"github.com/ethereum/go-ethereum/accounts"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/extkeys"
)

//subAccountFinder finds sub-accounts by existing extended key
type subAccountFinder interface {
	Find(keyStore accountKeyStorer, extKey *extkeys.ExtendedKey, subAccountIndex uint32) ([]accounts.Account, error)
}

type subAccountFinderBase struct{}

// Find traverses cached accounts and adds as a sub-accounts any
// that belong to the currently selected account.
// The extKey is CKD#2 := root of sub-accounts of the main account
func (m *subAccountFinderBase) Find(keyStore accountKeyStorer, extKey *extkeys.ExtendedKey, subAccountIndex uint32) ([]accounts.Account, error) {
	subAccounts := make([]accounts.Account, 0)
	if extKey.Depth == 5 { // CKD#2 level
		// gather possible sub-account addresses
		subAccountAddresses := make([]gethcommon.Address, 0)
		for i := uint32(0); i < subAccountIndex; i++ {
			childKey, err := extKey.Child(i)
			if err != nil {
				return []accounts.Account{}, err
			}
			subAccountAddresses = append(subAccountAddresses, crypto.PubkeyToAddress(childKey.ToECDSA().PublicKey))
		}

		// see if any of the gathered addresses actually exist in cached accounts list
		for _, cachedAccount := range keyStore.Accounts() {
			for _, possibleAddress := range subAccountAddresses {
				if possibleAddress.Hex() == cachedAccount.Address.Hex() {
					subAccounts = append(subAccounts, cachedAccount)
				}
			}
		}
	}

	return subAccounts, nil
}
