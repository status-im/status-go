package common

import (
	"errors"
	"fmt"

	"github.com/status-im/extkeys"
)

var (
	ErrInvalidMnemonicPhraseLength = errors.New("invalid mnemonic phrase length; valid lengths are 12, 15, 18, 21, and 24")
)

// CreateRandomMnemonic generates a random mnemonic phrase with the specified length
func CreateRandomMnemonic(length int) (string, error) {
	entropyStrength, err := LengthToEntropyStrength(length)
	if err != nil {
		return "", err
	}

	mnemonic := extkeys.NewMnemonic()
	return mnemonic.MnemonicPhrase(entropyStrength, extkeys.EnglishLanguage)
}

// CreateRandomMnemonicWithDefaultLength generates a random mnemonic phrase with default length (12 words)
func CreateRandomMnemonicWithDefaultLength() (string, error) {
	const defaultLength = 12
	return CreateRandomMnemonic(defaultLength)
}

// CreateExtendedKeyFromMnemonic creates an extended key from a mnemonic phrase
func CreateExtendedKeyFromMnemonic(phrase, passphrase string) (*extkeys.ExtendedKey, error) {
	mnemonic := extkeys.NewMnemonic()
	seed := mnemonic.MnemonicSeed(phrase, passphrase)
	masterKey, err := extkeys.NewMaster(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}
	return masterKey, nil
}

// LengthToEntropyStrength converts a mnemonic phrase length to its corresponding entropy strength
func LengthToEntropyStrength(length int) (extkeys.EntropyStrength, error) {
	if length < 12 || length > 24 || length%3 != 0 {
		return 0, ErrInvalidMnemonicPhraseLength
	}

	bitsLength := length * 11
	checksumLength := bitsLength % 32

	return extkeys.EntropyStrength(bitsLength - checksumLength), nil
}
