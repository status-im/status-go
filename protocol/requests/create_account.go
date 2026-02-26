package requests

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/pkg/errors"

	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
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
	APIModules       string `json:"apiModules"`
	ConnectorEnabled bool   `json:"connectorEnabled"`
	HTTPEnabled      bool   `json:"httpEnabled"`
	HTTPHost         string `json:"httpHost"`
	HTTPPort         int    `json:"httpPort"`
	// Deprecated: HTTPVirtualHosts is deprecated. It was passed and not used, so nothing should be passed instead.
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
	LogToStderr bool    `json:"logToStderr"`

	PreviewPrivacy bool `json:"previewPrivacy"`

	VerifyENSContractAddress *string `json:"verifyENSContractAddress"`
	UpstreamConfig           string  `json:"upstreamConfig"`

	// Deprecated: CurrentNetwork is deprecated. It was passed and not used, so nothing should be passed instead.
	// If you want to use non-default network, use NetworkID.
	CurrentNetwork       string           `json:"currentNetwork"`
	NetworkID            *uint64          `json:"networkId"`
	TestOverrideNetworks []params.Network `json:"networksOverride"` // This is used for testing purposes only

	TestNetworksEnabled bool `json:"testNetworksEnabled"`

	AutoRefreshTokensEnabled bool `json:"autoRefreshTokensEnabled"`

	WalletConfig
	WalletSecretsConfig

	TorrentConfigEnabled *bool
	TorrentConfigPort    *int

	APIConfig *APIConfig `json:"apiConfig"`

	KeycardInstanceUID string `json:"keycardInstanceUID"`

	// on mobile there is no KeycardPairingDataFile, so for now KeycardPairingKey will be used
	// for recovering account
	KeycardPairingKey      string  `json:"keycardPairingKey"`
	KeycardPairingDataFile *string `json:"keycardPairingDataFile"`

	ThirdpartyServicesEnabled bool `json:"thirdpartyServicesEnabled"`
}
type WalletConfig struct {
	TokensListsAutoRefreshInterval      int `json:"tokensListsAutoRefreshInterval"`      // in seconds
	TokensListsAutoRefreshCheckInterval int `json:"tokensListsAutoRefreshCheckInterval"` // in seconds
	MarketDataFullDataRefreshInterval   int `json:"marketDataFullDataRefreshInterval"`   // in seconds
	MarketDataPriceRefreshInterval      int `json:"marketDataPriceRefreshInterval"`      // in seconds

	MulticallOverrides              map[uint64]common.Address `json:"multicallOverrides"` // map[chainID]multicall3 contract address
	CommunityTokenDeployerOverrides map[uint64]common.Address `json:"communityTokenDeployerOverrides"` // map[chainID]CommunityTokenDeployer contract address

	CustomTokens []*tokentypes.Token `json:"customTokens"` // custom tokens, mainly used for registering custom tokens for functional tests
}
type WalletSecretsConfig struct {
	PoktToken            security.SensitiveString `json:"poktToken"`
	InfuraToken          security.SensitiveString `json:"infuraToken"`
	InfuraSecret         security.SensitiveString `json:"infuraSecret"`
	RaribleMainnetAPIKey security.SensitiveString `json:"raribleMainnetApiKey"`
	RaribleTestnetAPIKey security.SensitiveString `json:"raribleTestnetApiKey"`
	AlchemyAPIKey        security.SensitiveString `json:"alchemyApiKey"`

	StatusProxyStageName      string                   `json:"statusProxyStageName"`
	StatusProxyMarketUser     security.SensitiveString `json:"statusProxyMarketUser"`
	StatusProxyMarketPassword security.SensitiveString `json:"statusProxyMarketPassword"`

	MarketDataProxyUrl      security.SensitiveString `json:"marketDataProxyUrl"`
	MarketDataProxyUser     security.SensitiveString `json:"marketDataProxyUser"`
	MarketDataProxyPassword security.SensitiveString `json:"marketDataProxyPassword"`

	StatusProxyUser     security.SensitiveString `json:"statusProxyBlockchainUser"`
	StatusProxyPassword security.SensitiveString `json:"statusProxyBlockchainPassword"`

	EthRpcProxyUrl      security.SensitiveString `json:"ethRpcProxyUrl"`
	EthRpcProxyUser     security.SensitiveString `json:"ethRpcProxyUser"`
	EthRpcProxyPassword security.SensitiveString `json:"ethRpcProxyPassword"`

	NftProxyUrl           security.SensitiveString `json:"nftProxyUrl"`
	NftProxyStageName     string                   `json:"nftProxyStageName"`
	NftProxyUser          security.SensitiveString `json:"nftProxyUser"`
	NftProxyPassword      security.SensitiveString `json:"nftProxyPassword"`
	NftProxyUsePuzzleAuth bool                     `json:"nftProxyUsePuzzleAuth"`
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
