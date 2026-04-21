package generator

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/status-im/extkeys"

	"github.com/status-im/status-go/internal/accounts-management/common"
	"github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

// CreateAccountFromMnemonic creates an account from a mnemonic phrase.
func CreateAccountFromMnemonic(mnemonicPhrase string, bip39Passphrase string) (*Account, error) {
	masterExtendedKey, err := common.CreateExtendedKeyFromMnemonic(mnemonicPhrase, bip39Passphrase)
	if err != nil {
		return nil, ErrAccountCreationFromMnemonicFailed(err)
	}

	return NewAccount(masterExtendedKey.ToECDSA(), masterExtendedKey), nil
}

// CreateAccountFromPrivateKey creates an account from a private key.
func CreateAccountFromPrivateKey(privateKeyHex string) (*Account, error) {
	privateKeyWithoutPrefix := strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyWithoutPrefix)
	if err != nil {
		return nil, ErrAccountCreationFromPrivateKeyFailed(err)
	}

	return NewAccount(privateKey, nil), nil
}

// CreateAccountFromKey creates an account from a key.
func CreateAccountFromKey(key *types.Key) (*Account, error) {
	account := NewAccount(key.PrivateKey, key.ExtendedKey)
	if err := account.ValidateExtendedKey(); err != nil {
		return nil, ErrAccountCreationFromKeyFailed(err)
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
			return nil, nil, ErrMnemonicCreationFailed(err)
		}

		acc, err := CreateAccountFromMnemonic(mnemonicPhrase, bip39Passphrase)
		if err != nil {
			return nil, nil, ErrAccountCreationFromMnemonicFailed(err)
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
		return nil, ErrPathDecodingFailed(err)
	}

	if acc.extendedKey.IsZeroed() && len(path) == 0 {
		return acc, nil
	}

	if acc.extendedKey.IsZeroed() {
		return nil, ErrZeroedExtendedKey
	}

	childExtendedKey, err := acc.extendedKey.Derive(path)
	if err != nil {
		return nil, ErrExtendedKeyDerivationFailed(err)
	}

	return NewAccount(childExtendedKey.ToECDSA(), childExtendedKey), nil
}

// DeriveChildrenFromAccount derives multiple child accounts from an account.
func DeriveChildrenFromAccount(acc *Account, pathStrings []string) (map[string]*Account, error) {
	pathAccounts := make(map[string]*Account)

	for _, pathString := range pathStrings {
		childAccount, err := DeriveChildFromAccount(acc, pathString)
		if err != nil {
			return pathAccounts, ErrChildAccountDerivationFailed(err)
		}

		pathAccounts[pathString] = childAccount
	}

	return pathAccounts, nil
}

// derivePublicChildKey performs BIP32 non-hardened public child key derivation using go-ethereum's secp256k1 primitives
func derivePublicChildKey(parentPubKeyBytes, chainCode []byte, index uint32) (childPubKeyBytes, childChainCode []byte, err error) {
	if index >= 0x80000000 {
		return nil, nil, errors.New("hardened child derivation not supported from public key")
	}
	data := make([]byte, 37)
	copy(data[:33], parentPubKeyBytes)
	binary.BigEndian.PutUint32(data[33:], index)

	mac := hmac.New(sha512.New, chainCode)
	_, _ = mac.Write(data)
	I := mac.Sum(nil)
	IL, IR := I[:32], I[32:]

	curve := crypto.S256()
	x1, y1 := curve.ScalarBaseMult(IL)
	parentKey, err := crypto.DecompressPubkey(parentPubKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	x, y := curve.Add(x1, y1, parentKey.X, parentKey.Y)
	if x.Sign() == 0 && y.Sign() == 0 {
		return nil, nil, errors.New("derived key is point at infinity")
	}
	childPubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
	return crypto.CompressPubkey(childPubKey), IR, nil
}

func DeriveExtendedPublicKeyAtPath(mnemonic, passphrase, path string) (string, error) {
	acc, err := CreateAccountFromMnemonic(mnemonic, passphrase)
	if err != nil {
		return "", err
	}
	derived, err := DeriveChildFromAccount(acc, path)
	if err != nil {
		return "", err
	}
	xpub, err := derived.ExtendedKey().Neuter()
	if err != nil {
		return "", err
	}
	return xpub.String(), nil
}

// DeriveAccountsPublicInfoFromExtendedPublicKeyForPaths derives public accounts from an extended public key (xpub) for the given paths.
// Paths are relative to the provided xpub.
func DeriveAccountsPublicInfoFromExtendedPublicKeyForPaths(xpubString string, paths []string) (map[string]AccountPublicInfo, error) {
	extKey, err := extkeys.NewKeyFromString(xpubString)
	if err != nil {
		return nil, ErrExtendedKeyParsingFailed(err)
	}

	if extKey.IsPrivate {
		return nil, ErrExtendedPrivateKeyNotAllowed
	}

	result := make(map[string]AccountPublicInfo)
	for _, pathString := range paths {
		_, path, err := decodePath(pathString)
		if err != nil {
			return nil, ErrPathDecodingFailed(err)
		}

		pubKeyBytes := extKey.KeyData
		chainCode := extKey.ChainCode
		for _, index := range path {
			pubKeyBytes, chainCode, err = derivePublicChildKey(pubKeyBytes, chainCode, index)
			if err != nil {
				return nil, ErrExtendedKeyDerivationFailed(err)
			}
		}

		pubKey, err := crypto.DecompressPubkey(pubKeyBytes)
		if err != nil {
			return nil, ErrPublicKeyDerivationFailed(err)
		}

		result[pathString] = AccountPublicInfo{
			PublicKey: cryptotypes.EncodeHex(crypto.FromECDSAPub(pubKey)),
			Address:   crypto.PubkeyToAddress(*pubKey).Hex(),
		}
	}

	return result, nil
}

func CreateAndDeriveAccountsFromMnemonic(mnemonic string, paths []string, bip39Passphrase string) (*Account, map[string]*Account, error) {
	account, err := CreateAccountFromMnemonic(mnemonic, bip39Passphrase)
	if err != nil {
		return nil, nil, err
	}

	derivedAccounts, err := DeriveChildrenFromAccount(account, paths)
	if err != nil {
		return nil, nil, err
	}

	return account, derivedAccounts, nil
}
