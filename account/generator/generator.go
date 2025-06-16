package generator

import (
	"fmt"
	"strings"

	"github.com/status-im/status-go/account/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

// CreateAccountFromMnemonic creates an account from a mnemonic phrase.
func CreateAccountFromMnemonic(mnemonicPhrase string, bip39Passphrase string) (*Account, error) {
	masterExtendedKey, err := common.CreateExtendedKeyFromMnemonic(mnemonicPhrase, bip39Passphrase)
	if err != nil {
		return nil, fmt.Errorf("can not create account from mnemonic: %v", err)
	}

	return NewAccount(masterExtendedKey.ToECDSA(), masterExtendedKey), nil
}

// CreateAccountFromPrivateKey creates an account from a private key.
func CreateAccountFromPrivateKey(privateKeyHex string) (*Account, error) {
	privateKeyWithoutPrefix := strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyWithoutPrefix)
	if err != nil {
		return nil, fmt.Errorf("can not create account from private key: %v", err)
	}

	return NewAccount(privateKey, nil), nil
}

// CreateAccountFromKey creates an account from a key.
func CreateAccountFromKey(key *types.Key) (*Account, error) {
	account := NewAccount(key.PrivateKey, key.ExtendedKey)
	if err := account.ValidateExtendedKey(); err != nil {
		return nil, fmt.Errorf("can not create account from key: %v", err)
	}
	return account, nil
}

// CreateAccountsOfMnemonicLength creates n accounts from a mnemonic phrase of length mnemonicPhraseLength.
// It returns the accounts and the mnemonic phrases, the order matches.
func CreateAccountsOfMnemonicLength(mnemonicPhraseLength int, n int, bip39Passphrase string) ([]*Account, []string, error) {
	accounts := make([]*Account, 0)
	mnemonicPhrases := make([]string, 0)

	for i := 0; i < n; i++ {
		mnemonicPhrase, err := common.CreateRandomMnemonic(mnemonicPhraseLength)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create mnemonic seed: %w", err)
		}

		acc, err := CreateAccountFromMnemonic(mnemonicPhrase, bip39Passphrase)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create account from mnemonic: %w", err)
		}

		accounts = append(accounts, acc)
		mnemonicPhrases = append(mnemonicPhrases, mnemonicPhrase)
	}

	return accounts, mnemonicPhrases, nil
}

// DeriveChildFromAccount derives a child account from an account.
func DeriveChildFromAccount(acc *Account, pathString string) (*Account, error) {
	_, path, err := decodePath(pathString)
	if err != nil {
		return nil, fmt.Errorf("can not decode path: %v", err)
	}

	if acc.extendedKey.IsZeroed() && len(path) == 0 {
		return acc, nil
	}

	if acc.extendedKey.IsZeroed() {
		return nil, fmt.Errorf("can not derive child account from zeroed extended key")
	}

	childExtendedKey, err := acc.extendedKey.Derive(path)
	if err != nil {
		return nil, fmt.Errorf("can not derive child account from extended key: %v", err)
	}

	return NewAccount(childExtendedKey.ToECDSA(), childExtendedKey), nil
}

// DeriveChildrenFromAccount derives multiple child accounts from an account.
func DeriveChildrenFromAccount(acc *Account, pathStrings []string) (map[string]*Account, error) {
	pathAccounts := make(map[string]*Account)

	for _, pathString := range pathStrings {
		childAccount, err := DeriveChildFromAccount(acc, pathString)
		if err != nil {
			return pathAccounts, fmt.Errorf("can not derive child account from path: %v", err)
		}

		pathAccounts[pathString] = childAccount
	}

	return pathAccounts, nil
}
