package requests

import (
	"github.com/pkg/errors"

	utils "github.com/status-im/status-go/common"
)

var ErrCreateAccountInvalidDisplayName = errors.New("create-account: invalid display name")
var ErrCreateAccountInvalidPassword = errors.New("create-account: invalid password")
var ErrCreateAccountInvalidCustomizationColor = errors.New("create-account: invalid customization color")
var ErrCreateAccountInvalidRootKeystoreDir = errors.New("create-account: invalid root keystore directory")
var ErrCreateAccountInvalidBackupDisabledDataDir = errors.New("create-account: invalid backup disabled data directory")

type ImageCropRectangle struct {
	Ax int `json:"ax"`
	Ay int `json:"ay"`
	Bx int `json:"bx"`
	By int `json:"by"`
}

type CreateAccount struct {
	// BackupDisabledDataDir is the directory where backup is disabled
	// WARNING: This is used as `RootDataDir`. Consider renaming?
	BackupDisabledDataDir string `json:"backupDisabledDataDir"`
	KdfIterations         int    `json:"kdfIterations"`

	DeviceName         string              `json:"deviceName"`
	DisplayName        string              `json:"displayName"`
	Password           string              `json:"password"`
	ImagePath          string              `json:"imagePath"`
	ImageCropRectangle *ImageCropRectangle `json:"imageCropRectangle"`
	CustomizationColor string              `json:"customizationColor"`
	Emoji              string              `json:"emoji"`

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

	// Deprecated: CurrentNetwork is deprecated. It was passed and not used, so nothing should be passed instead.
	// If you want to use non-default network, use NetworkID.
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

func (c *CreateAccount) Validate(validation *CreateAccountValidation) error {
	// TODO(cammellos): Add proper validation for password/displayname/etc
	// Empty display name is allowed during account restore
	if len(c.DisplayName) == 0 && !validation.AllowEmptyDisplayName {
		return ErrCreateAccountInvalidDisplayName
	}

	if err := utils.ValidateDisplayName(&c.DisplayName); err != nil {
		return errors.Wrap(ErrCreateAccountInvalidDisplayName, err.Error())
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

// NOTE: Reasoning for this struct here: https://github.com/status-im/status-go/pull/4980#discussion_r1539219099
type CreateAccountValidation struct {
	AllowEmptyDisplayName bool
}
