package accounts

import (
	"github.com/status-im/status-go/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/multiaccounts/common"
)

func KeypairToAccountsManagerKeypair(keypair *Keypair) *types.Keypair {
	accountsManagerKeypair := &types.Keypair{
		KeyUID:                  keypair.KeyUID,
		Name:                    keypair.Name,
		Type:                    types.KeypairType(keypair.Type),
		DerivedFrom:             keypair.DerivedFrom,
		LastUsedDerivationIndex: keypair.LastUsedDerivationIndex,
		SyncedFrom:              keypair.SyncedFrom,
		Clock:                   keypair.Clock,
		Removed:                 keypair.Removed,
	}

	accountsManagerKeypair.Accounts = make([]*types.Account, len(keypair.Accounts))
	for i, account := range keypair.Accounts {
		accountsManagerKeypair.Accounts[i] = AccountToAccountsManagerAccount(account)
	}

	accountsManagerKeypair.Keycards = make([]*types.Keycard, len(keypair.Keycards))
	for i, keycard := range keypair.Keycards {
		accountsManagerKeypair.Keycards[i] = KeycardToAccountsManagerKeycard(keycard)
	}

	return accountsManagerKeypair
}

func KeypairsToAccountsManagerKeypairs(keypairs []*Keypair) []*types.Keypair {
	accountsManagerKeypairs := make([]*types.Keypair, len(keypairs))
	for i, keypair := range keypairs {
		accountsManagerKeypairs[i] = KeypairToAccountsManagerKeypair(keypair)
	}
	return accountsManagerKeypairs
}

func AccountsManagerKeypairToKeypair(keypair *types.Keypair) *Keypair {
	dbKeypair := &Keypair{
		KeyUID:                  keypair.KeyUID,
		Name:                    keypair.Name,
		Type:                    KeypairType(keypair.Type),
		DerivedFrom:             keypair.DerivedFrom,
		LastUsedDerivationIndex: keypair.LastUsedDerivationIndex,
		SyncedFrom:              keypair.SyncedFrom,
		Clock:                   keypair.Clock,
		Removed:                 keypair.Removed,
	}

	dbKeypair.Accounts = make([]*Account, len(keypair.Accounts))
	for i, account := range keypair.Accounts {
		dbKeypair.Accounts[i] = AccountsManagerAccountToAccount(account)
	}

	dbKeypair.Keycards = make([]*Keycard, len(keypair.Keycards))
	for i, keycard := range keypair.Keycards {
		dbKeypair.Keycards[i] = AccountsManagerKeycardToKeycard(keycard)
	}

	return dbKeypair
}

func AccountToAccountsManagerAccount(account *Account) *types.Account {
	return &types.Account{
		Address:               account.Address,
		KeyUID:                account.KeyUID,
		Wallet:                account.Wallet,
		AddressWasNotShown:    account.AddressWasNotShown,
		Chat:                  account.Chat,
		Type:                  types.AccountType(account.Type),
		Path:                  account.Path,
		PublicKey:             account.PublicKey,
		Name:                  account.Name,
		Emoji:                 account.Emoji,
		ColorID:               string(account.ColorID),
		Hidden:                account.Hidden,
		Clock:                 account.Clock,
		Removed:               account.Removed,
		Operable:              types.AccountOperable(account.Operable),
		CreatedAt:             account.CreatedAt,
		Position:              account.Position,
		ProdPreferredChainIDs: account.ProdPreferredChainIDs,
		TestPreferredChainIDs: account.TestPreferredChainIDs,
	}
}

func AccountsManagerAccountToAccount(account *types.Account) *Account {
	return &Account{
		Address:               account.Address,
		KeyUID:                account.KeyUID,
		Wallet:                account.Wallet,
		AddressWasNotShown:    account.AddressWasNotShown,
		Chat:                  account.Chat,
		Type:                  AccountType(account.Type),
		Path:                  account.Path,
		PublicKey:             account.PublicKey,
		Name:                  account.Name,
		Emoji:                 account.Emoji,
		ColorID:               common.CustomizationColor(account.ColorID),
		Hidden:                account.Hidden,
		Clock:                 account.Clock,
		Removed:               account.Removed,
		Operable:              AccountOperable(account.Operable),
		CreatedAt:             account.CreatedAt,
		Position:              account.Position,
		ProdPreferredChainIDs: account.ProdPreferredChainIDs,
		TestPreferredChainIDs: account.TestPreferredChainIDs,
	}
}

func AccountsManagerAccountsToAccounts(accounts []*types.Account) []*Account {
	dbAccounts := make([]*Account, len(accounts))
	for i, account := range accounts {
		dbAccounts[i] = AccountsManagerAccountToAccount(account)
	}
	return dbAccounts
}

func AccountsToAccountsManagerAccounts(accounts []*Account) []*types.Account {
	accountsManagerAccounts := make([]*types.Account, len(accounts))
	for i, account := range accounts {
		accountsManagerAccounts[i] = AccountToAccountsManagerAccount(account)
	}
	return accountsManagerAccounts
}

func KeycardToAccountsManagerKeycard(keycard *Keycard) *types.Keycard {
	accountsManagerKeycard := &types.Keycard{
		KeycardUID:    keycard.KeycardUID,
		KeycardName:   keycard.KeycardName,
		KeyUID:        keycard.KeyUID,
		Position:      keycard.Position,
		KeycardLocked: keycard.KeycardLocked,
	}

	accountsManagerKeycard.AccountsAddresses = make([]cryptotypes.Address, len(keycard.AccountsAddresses))
	for i, accountAddress := range keycard.AccountsAddresses {
		accountsManagerKeycard.AccountsAddresses[i] = cryptotypes.HexToAddress(accountAddress.Hex())
	}

	return accountsManagerKeycard
}

func AccountsManagerKeycardToKeycard(keycard *types.Keycard) *Keycard {
	dbKeycard := &Keycard{
		KeycardUID:    keycard.KeycardUID,
		KeycardName:   keycard.KeycardName,
		KeyUID:        keycard.KeyUID,
		Position:      keycard.Position,
		KeycardLocked: keycard.KeycardLocked,
	}

	dbKeycard.AccountsAddresses = make([]cryptotypes.Address, len(keycard.AccountsAddresses))
	for i, accountAddress := range keycard.AccountsAddresses {
		dbKeycard.AccountsAddresses[i] = cryptotypes.HexToAddress(accountAddress.Hex())
	}

	return dbKeycard
}
