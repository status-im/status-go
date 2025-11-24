package requests

import (
	"github.com/status-im/status-go/protocol/requests"
)

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

func CreateAccountAdapter(input *CreateAccount, rootDataDir string) *requests.CreateAccount {
	return &requests.CreateAccount{
		RootDataDir:        rootDataDir,
		Password:           input.Password,
		KdfIterations:      input.KdfIterations,
		KeycardInstanceUID: input.KeycardInstanceUID,
		DeviceName:         input.DeviceName,
		DisplayName:        input.DisplayName,
		ImagePath:          input.ImagePath,
		CustomizationColor: input.CustomizationColor,
		WakuV2LightClient:  input.WakuV2LightClient,
		PreviewPrivacy:     input.PreviewPrivacy,
	}
}

//func CreateAccountAdapter(input *requests.CreateAccount) *CreateAccount {
//	out := &CreateAccount{
//		Password:                  input.Password,
//		KdfIterations:             input.KdfIterations,
//		KeycardInstanceUID:        input.KeycardInstanceUID,
//		DeviceName:                input.DeviceName,
//		DisplayName:               input.DisplayName,
//		ImagePath:                 input.ImagePath,
//		CustomizationColor:        input.CustomizationColor,
//		WakuV2LightClient:         input.WakuV2LightClient,
//		PreviewPrivacy:            input.PreviewPrivacy,
//		KeycardPairingKey:         input.KeycardPairingKey,
//		KeycardPairingDataFile:    input.KeycardPairingDataFile,
//		ThirdpartyServicesEnabled: input.ThirdpartyServicesEnabled,
//	}
//
//	if input.ImageCropRectangle != nil {
//		out.ImageCropRectangle = &ImageCropRectangle{
//			Ax: input.ImageCropRectangle.Ax,
//			Ay: input.ImageCropRectangle.Ay,
//			Bx: input.ImageCropRectangle.Bx,
//			By: input.ImageCropRectangle.By,
//		}
//	}
//
//	return out
//}
