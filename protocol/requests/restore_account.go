package requests

import (
	"errors"
)

var (
	ErrRestoreAccountInvalidMnemonic    = errors.New("restore-account: no mnemonic or keycard is set")
	ErrRestoreAccountMnemonicAndKeycard = errors.New("restore-account: both mnemonic and keycard info are set")
)

type RestoreAccount struct {
	Mnemonic string `json:"mnemonic"`
	BIP39Passphrase string `json:"BIP39Passphrase"`

	// Keycard info can be set instead of Mnemonic.
	// This is to log in using a keycard with existing account.
	Keycard *KeycardData `json:"keycard"`

	FetchBackup bool `json:"fetchBackup"`

	CreateAccount
}

func (c *RestoreAccount) Validate() error {
	if len(c.Mnemonic) == 0 && c.Keycard == nil {
		return ErrRestoreAccountInvalidMnemonic
	}

	if len(c.Mnemonic) > 0 && c.Keycard != nil {
		return ErrRestoreAccountMnemonicAndKeycard
	}

	return c.CreateAccount.Validate(&CreateAccountValidation{
		AllowEmptyDisplayName: true,
	})
}

type KeycardData struct {
	KeyUID              string `json:"keyUID"`
	Address             string `json:"address"`
	WhisperPrivateKey   string `json:"whisperPrivateKey"`
	WhisperPublicKey    string `json:"whisperPublicKey"`
	WhisperAddress      string `json:"whisperAddress"`
	WalletPublicKey     string `json:"walletPublicKey"`
	WalletAddress       string `json:"walletAddress"`
	WalletRootAddress   string `json:"walletRootAddress"`
	Eip1581Address      string `json:"eip1581Address"`
	EncryptionPublicKey string `json:"encryptionPublicKey"`
}
