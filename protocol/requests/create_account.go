package requests

import (
	"errors"
)

var ErrCreateAccountInvalidDisplayName = errors.New("create-account: invalid display name")
var ErrCreateAccountInvalidPassword = errors.New("create-account: invalid password")
var ErrCreateAccountInvalidCustomizationColor = errors.New("create-account: invalid customization color")
var ErrCreateAccountInvalidRootKeystoreDir = errors.New("create-account: invalid root keystore directory")
var ErrCreateAccountInvalidBackupDisabledDataDir = errors.New("create-account: invalid backup disabled data directory")

type CreateAccount struct {
	// BackupDisabledDataDir is the directory where backup is disabled
	// WARNING: This is used as `RootDataDir`. Consider renaming?
	BackupDisabledDataDir string `json:"backupDisabledDataDir"`
	KdfIterations         int    `json:"kdfIterations"`

	DeviceName         string `json:"deviceName"`
	DisplayName        string `json:"displayName"`
	Password           string `json:"password"`
	ImagePath          string `json:"imagePath"`
	CustomizationColor string `json:"customizationColor"`
	Emoji              string `json:"emoji"`

	WakuV2Nameserver  *string `json:"wakuV2Nameserver"`
	WakuV2LightClient bool    `json:"wakuV2LightClient"`

	LogLevel    *string `json:"logLevel"`
	LogFilePath string  `json:"logFilePath"`
	LogEnabled  bool    `json:"logEnabled"`

	PreviewPrivacy bool `json:"previewPrivacy"`

	VerifyTransactionURL     *string `json:"verifyTransactionURL"`
	VerifyENSURL             *string `json:"verifyENSURL"`
	VerifyENSContractAddress *string `json:"verifyENSContractAddress"`
	VerifyTransactionChainID *int64  `json:"verifyTransactionChainID"`
	UpstreamConfig           string  `json:"upstreamConfig"`

	// Deprecated: CurrentNetwork is deprecated
	CurrentNetwork string  `json:"currentNetwork"`
	NetworkID      *uint64 `json:"networkId"`

	TestNetworksEnabled bool `json:"testNetworksEnabled"`

	WalletSecretsConfig

	TorrentConfigEnabled *bool
	TorrentConfigPort    *int
}

type WalletSecretsConfig struct {
	PoktToken            string `json:"poktToken"`
	InfuraToken          string `json:"infuraToken"`
	InfuraSecret         string `json:"infuraSecret"`
	OpenseaAPIKey        string `json:"openseaApiKey"`
	RaribleMainnetAPIKey string `json:"raribleMainnetApiKey"`
	RaribleTestnetAPIKey string `json:"raribleTestnetApiKey"`

	// Testing
	GanacheURL                  string `json:"ganacheURL"`
	AlchemyEthereumMainnetToken string `json:"alchemyEthereumMainnetToken"`
	AlchemyEthereumGoerliToken  string `json:"alchemyEthereumGoerliToken"`
	AlchemyEthereumSepoliaToken string `json:"alchemyEthereumSepoliaToken"`
	AlchemyArbitrumMainnetToken string `json:"alchemyArbitrumMainnetToken"`
	AlchemyArbitrumGoerliToken  string `json:"alchemyArbitrumGoerliToken"`
	AlchemyArbitrumSepoliaToken string `json:"alchemyArbitrumSepoliaToken"`
	AlchemyOptimismMainnetToken string `json:"alchemyOptimismMainnetToken"`
	AlchemyOptimismGoerliToken  string `json:"alchemyOptimismGoerliToken"`
	AlchemyOptimismSepoliaToken string `json:"alchemyOptimismSepoliaToken"`
}

func (c *CreateAccount) Validate() error {
	return ValidateAccountCreationRequest(*c, false)
}

func ValidateAccountCreationRequest(c CreateAccount, allowEmptyDisplayName bool) error {
	// TODO(cammellos): Add proper validation for password/displayname/etc

	// Empty display name is allowed during account restore
	if len(c.DisplayName) == 0 && !allowEmptyDisplayName {
		return ErrCreateAccountInvalidDisplayName
	}

	if len(c.Password) == 0 {
		return ErrCreateAccountInvalidPassword
	}

	if len(c.CustomizationColor) == 0 {
		return ErrCreateAccountInvalidCustomizationColor
	}

	if len(c.BackupDisabledDataDir) == 0 {
		return ErrCreateAccountInvalidBackupDisabledDataDir
	}

	return nil

}
