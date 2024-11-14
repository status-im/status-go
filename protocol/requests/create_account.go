package requests

import (
	"github.com/pkg/errors"

	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
)

var ErrCreateAccountInvalidDisplayName = errors.New("create-account: invalid display name")
var ErrCreateAccountInvalidPassword = errors.New("create-account: invalid password")
var ErrCreateAccountInvalidCustomizationColor = errors.New("create-account: invalid customization color")
var ErrCreateAccountInvalidRootKeystoreDir = errors.New("create-account: invalid root keystore directory")
var ErrCreateAccountInvalidRootDataDir = errors.New("create-account: invalid root data directory")

type ImageCropRectangle struct {
	Ax int `json:"ax"`
	Ay int `json:"ay"`
	Bx int `json:"bx"`
	By int `json:"by"`
}

type APIConfig struct {
	APIModules       string   `json:"apiModules"`
	ConnectorEnabled bool     `json:"connectorEnabled"`
	HTTPEnabled      bool     `json:"httpEnabled"`
	HTTPHost         string   `json:"httpHost"`
	HTTPPort         int      `json:"httpPort"`
	HTTPVirtualHosts []string `json:"httpVirtualHosts"`
	WSEnabled        bool     `json:"wsEnabled"`
	WSHost           string   `json:"wsHost"`
	WSPort           int      `json:"wsPort"`
}

type CreateAccount struct {
	// RootDataDir is an absolute path to the root directory where all data will be stored.
	RootDataDir   string `json:"rootDataDir"`
	KdfIterations int    `json:"kdfIterations"`

	DeviceName         string              `json:"deviceName"`
	DisplayName        string              `json:"displayName"`
	Password           string              `json:"password"`
	ImagePath          string              `json:"imagePath"`
	ImageCropRectangle *ImageCropRectangle `json:"imageCropRectangle"`
	CustomizationColor string              `json:"customizationColor"`

	WakuV2Nameserver                             *string `json:"wakuV2Nameserver"`
	WakuV2LightClient                            bool    `json:"wakuV2LightClient"`
	WakuV2EnableStoreConfirmationForMessagesSent bool    `json:"wakuV2EnableStoreConfirmationForMessagesSent"`
	WakuV2EnableMissingMessageVerification       bool    `json:"wakuV2EnableMissingMessageVerification"`
	WakuV2Fleet                                  string  `json:"wakuV2Fleet"`

	LogLevel    *string `json:"logLevel"`
	LogFilePath string  `json:"logFilePath"` // absolute path
	LogEnabled  bool    `json:"logEnabled"`

	PreviewPrivacy bool `json:"previewPrivacy"`

	VerifyTransactionURL     *string `json:"verifyTransactionURL"`
	VerifyENSURL             *string `json:"verifyENSURL"`
	VerifyENSContractAddress *string `json:"verifyENSContractAddress"`
	VerifyTransactionChainID *int64  `json:"verifyTransactionChainID"`
	UpstreamConfig           string  `json:"upstreamConfig"`

	// Deprecated: CurrentNetwork is deprecated. It was passed and not used, so nothing should be passed instead.
	// If you want to use non-default network, use NetworkID.
	CurrentNetwork       string           `json:"currentNetwork"`
	NetworkID            *uint64          `json:"networkId"`
	TestOverrideNetworks []params.Network `json:"networksOverride"` // This is used for testing purposes only

	TestNetworksEnabled bool `json:"testNetworksEnabled"`

	WalletSecretsConfig

	TorrentConfigEnabled *bool
	TorrentConfigPort    *int

	TelemetryServerURL string `json:"telemetryServerURL"`

	APIConfig *APIConfig `json:"apiConfig"`

	KeycardInstanceUID string `json:"keycardInstanceUID"`

	// on mobile there is no KeycardPairingDataFile, so for now KeycardPairingKey will be used
	// for recovering account
	KeycardPairingKey      string  `json:"keycardPairingKey"`
	KeycardPairingDataFile *string `json:"keycardPairingDataFile"`
	StatusProxyEnabled     bool    `json:"statusProxyEnabled"`
}

type WalletSecretsConfig struct {
	PoktToken            string `json:"poktToken"`
	InfuraToken          string `json:"infuraToken"`
	InfuraSecret         string `json:"infuraSecret"`
	OpenseaAPIKey        string `json:"openseaApiKey"`
	RaribleMainnetAPIKey string `json:"raribleMainnetApiKey"`
	RaribleTestnetAPIKey string `json:"raribleTestnetApiKey"`

	AlchemyEthereumMainnetToken string `json:"alchemyEthereumMainnetToken"`
	AlchemyEthereumSepoliaToken string `json:"alchemyEthereumSepoliaToken"`
	AlchemyArbitrumMainnetToken string `json:"alchemyArbitrumMainnetToken"`
	AlchemyArbitrumSepoliaToken string `json:"alchemyArbitrumSepoliaToken"`
	AlchemyOptimismMainnetToken string `json:"alchemyOptimismMainnetToken"`
	AlchemyOptimismSepoliaToken string `json:"alchemyOptimismSepoliaToken"`

	StatusProxyStageName          string `json:"statusProxyStageName"`
	StatusProxyMarketUser         string `json:"statusProxyMarketUser"`
	StatusProxyMarketPassword     string `json:"statusProxyMarketPassword"`
	StatusProxyBlockchainUser     string `json:"statusProxyBlockchainUser"`
	StatusProxyBlockchainPassword string `json:"statusProxyBlockchainPassword"`

	// Testing
	GanacheURL string `json:"ganacheURL"`
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

	if len(c.Password) == 0 && !validation.AllowEmptyPassword {
		return ErrCreateAccountInvalidPassword
	}

	if len(c.CustomizationColor) == 0 && !validation.AllowEmptyCustomizationColor {
		return ErrCreateAccountInvalidCustomizationColor
	}

	if len(c.RootDataDir) == 0 {
		return ErrCreateAccountInvalidRootDataDir
	}

	return nil
}

// NOTE: Reasoning for this struct here: https://github.com/status-im/status-go/pull/4980#discussion_r1539219099
type CreateAccountValidation struct {
	AllowEmptyDisplayName        bool
	AllowEmptyPassword           bool
	AllowEmptyCustomizationColor bool
}
