package requests

import (
	"errors"
)

var (
	ErrRestoreRegularAccountKeycardSet      = errors.New("restore-account: keycard details set for regular account")
	ErrRestoreRegularAccountMnemonicMissing = errors.New("restore-account: mnemonic is not set")

	ErrRestoreKeycardAccountMnemonicSet           = errors.New("restore-account: mnemonic is set for keycard account")
	ErrRestoreKeycardAccountKecardDetatilsMissing = errors.New("restore-account: keycard details are not set")
	ErrRestoreKeycardAccountNoWhisperPrivateKey   = errors.New("restore-account: chat private key is not set")
)

type RestoreAccount struct {
	Mnemonic string `json:"mnemonic"`

	// Keycard info can be set instead of Mnemonic.
	// This is to log in using a keycard with existing account.
	Keycard *KeycardData `json:"keycard"`

	FetchBackup bool `json:"fetchBackup"`

	CreateAccount
}

func (c *RestoreAccount) Validate(restoreViaKeycard bool) error {
	if restoreViaKeycard {
		if len(c.Mnemonic) > 0 {
			return ErrRestoreKeycardAccountMnemonicSet
		}

		if c.Keycard == nil {
			return ErrRestoreKeycardAccountKecardDetatilsMissing
		}

		if c.Keycard.WhisperPrivateKey == "" {
			return ErrRestoreKeycardAccountNoWhisperPrivateKey
		}
	} else {
		if c.Keycard != nil {
			return ErrRestoreRegularAccountKeycardSet
		}

		if len(c.Mnemonic) == 0 {
			return ErrRestoreRegularAccountMnemonicMissing
		}
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
