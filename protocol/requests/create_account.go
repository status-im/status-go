package requests

import (
	"errors"
)

var ErrCreateAccountInvalidDisplayName = errors.New("create-account: invalid display name")
var ErrCreateAccountInvalidPassword = errors.New("create-account: invalid password")
var ErrCreateAccountInvalidColor = errors.New("create-account: invalid color")
var ErrCreateAccountInvalidRootKeystoreDir = errors.New("create-account: invalid root keystore directory")
var ErrCreateAccountInvalidBackupDisabledDataDir = errors.New("create-account: invalid backup disabled data directory")
var ErrCreateAccountInvalidLogFilePath = errors.New("create-account: invalid log file path")

type CreateAccount struct {
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	ImagePath   string `json:"imagePath"`
	Color       string `json:"color"`
	// RootKeystoreDir is the directory where keys are stored
	RootKeystoreDir string `json:"rootKeystoreDir"`
	// BackupDisabledDataDir is the directory where backup is disabled
	BackupDisabledDataDir string `json:"backupDisabledDataDir"`

	VerifyTransactionURL     *string `json:"verifyTransactionURL"`
	VerifyENSURL             *string `json:"verifyENSURL"`
	VerifyENSContractAddress *string `json:"verifyENSContractAddress"`
	VerifyTransactionChainID *int64  `json:"verifyTransactionChainID"`
	WakuV2Nameserver         *string `json:"wakuV2Nameserver"`
	LogLevel                 *string `json:"logLevel"`
	LogFilePath              string  `json:"logFilePath"`
	LogEnabled               bool    `json:"logEnabled"`
}

func (c *CreateAccount) Validate() error {
	return ValidateAccountCreationRequest(*c)
}

func ValidateAccountCreationRequest(c CreateAccount) error {
	// TODO(cammellos): Add proper validation for password/displayname/etc
	if len(c.DisplayName) == 0 {
		return ErrCreateAccountInvalidDisplayName
	}

	if len(c.Password) == 0 {
		return ErrCreateAccountInvalidPassword
	}

	if len(c.Color) == 0 {
		return ErrCreateAccountInvalidColor
	}

	if len(c.RootKeystoreDir) == 0 {
		return ErrCreateAccountInvalidRootKeystoreDir
	}

	if len(c.BackupDisabledDataDir) == 0 {
		return ErrCreateAccountInvalidBackupDisabledDataDir
	}

	if len(c.LogFilePath) == 0 {
		return ErrCreateAccountInvalidLogFilePath
	}

	return nil

}
