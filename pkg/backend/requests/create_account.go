package requests

type ImageCropRectangle struct {
	Ax int `json:"ax"`
	Ay int `json:"ay"`
	Bx int `json:"bx"`
	By int `json:"by"`
}

type CreateAccount struct {
	// RootDataDir is an absolute path to the root directory where all data will be stored.
	//RootDataDir   string `json:"rootDataDir"`

	Password           string `json:"password"`
	KdfIterations      int    `json:"kdfIterations"`
	KeycardInstanceUID string `json:"keycardInstanceUID"`

	DeviceName         string              `json:"deviceName"`
	DisplayName        string              `json:"displayName"`
	ImagePath          string              `json:"imagePath"`
	ImageCropRectangle *ImageCropRectangle `json:"imageCropRectangle"`
	CustomizationColor string              `json:"customizationColor"`

	//WakuV2Nameserver                             *string `json:"wakuV2Nameserver"`
	WakuV2LightClient bool `json:"wakuV2LightClient"`
	//WakuV2EnableStoreConfirmationForMessagesSent bool    `json:"wakuV2EnableStoreConfirmationForMessagesSent"`
	//WakuV2EnableMissingMessageVerification       bool    `json:"wakuV2EnableMissingMessageVerification"`
	//WakuV2Fleet                                  string  `json:"wakuV2Fleet"`

	//LogLevel    *string `json:"logLevel"`
	//LogFilePath string  `json:"logFilePath"` // absolute path
	//LogEnabled  bool    `json:"logEnabled"`
	//LogToStderr bool    `json:"logToStderr"`

	PreviewPrivacy bool `json:"previewPrivacy"`

	//VerifyENSContractAddress *string `json:"verifyENSContractAddress"`
	//UpstreamConfig           string  `json:"upstreamConfig"`

	//NetworkID            *uint64          `json:"networkId"`
	//TestOverrideNetworks []params.Network `json:"networksOverride"` // This is used for testing purposes only

	//TestNetworksEnabled bool `json:"testNetworksEnabled"`

	//AutoRefreshTokensEnabled bool `json:"autoRefreshTokensEnabled"`

	//WalletConfig
	//WalletSecretsConfig

	//TorrentConfigEnabled *bool
	//TorrentConfigPort    *int

	//APIConfig *APIConfig `json:"apiConfig"`

	// on mobile there is no KeycardPairingDataFile, so for now KeycardPairingKey will be used
	// for recovering account
	KeycardPairingKey      string  `json:"keycardPairingKey"`
	KeycardPairingDataFile *string `json:"keycardPairingDataFile"`

	ThirdpartyServicesEnabled bool `json:"thirdpartyServicesEnabled"`
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
