package requests

import (
	"errors"
)

var ErrRestoreAccountInvalidMnemonic = errors.New("restore-account: invalid mnemonic")

type RestoreAccount struct {
	Mnemonic    string `json:"mnemonic"`
	FetchBackup bool   `json:"fetchBackup"`
	CreateAccount
}

func (c *RestoreAccount) Validate() error {
	if len(c.Mnemonic) == 0 {
		return ErrRestoreAccountInvalidMnemonic
	}

	return c.CreateAccount.Validate(&CreateAccountValidation{
		AllowEmptyDisplayName: true,
	})
}

type RestoreKeycardAccount struct {
	FetchBackup bool `json:"fetchBackup"`

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

	CreateAccount
}

func (c *RestoreKeycardAccount) Validate() error {
	return c.CreateAccount.Validate(&CreateAccountValidation{
		AllowEmptyDisplayName: true,
	})
}
