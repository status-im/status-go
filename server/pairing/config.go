package pairing

import (
	"crypto/ecdsa"
	"crypto/tls"

	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
)

type SenderConfig struct {
	// SenderConfig.KeystorePath must end with keyUID
	KeystorePath string `json:"keystorePath" validate:"required,keystorepath"`
	// DeviceType SendPairInstallation need this information
	DeviceType string `json:"deviceType" validate:"required"`

	KeyUID             string   `json:"keyUID" validate:"required,keyuid"`
	Password           string   `json:"password" validate:"required"`
	ChatKey            string   `json:"chatKey"` // set only in case of a Keycard user, otherwise empty
	KeycardPairings    string   `json:"keycardPairings"`
	UnimportedKeypairs []string `json:"unimportedKeypairs"` // used to select keypairs we're transferring keystore files for

	DB *multiaccounts.Database `json:"-"`
}

type ReceiverConfig struct {
	// nodeConfig is required, but we'll validate it separately
	NodeConfig *params.NodeConfig `json:"nodeConfig"`

	// ReceiverConfig.KeystorePath must not end with keyUID (because keyUID is not known yet)
	KeystorePath string `json:"keystorePath" validate:"required,not_end_keyuid"`

	// DeviceType SendPairInstallation need this information
	DeviceType    string `json:"deviceType" validate:"required"`
	KDFIterations int    `json:"kdfIterations" validate:"gte=0"`

	// SettingCurrentNetwork corresponding to field current_network from table settings, so that we can override current network from sender
	SettingCurrentNetwork string `json:"settingCurrentNetwork" validate:"required"`

	DeviceName     string                  `json:"deviceName"`
	DB             *multiaccounts.Database `json:"-"`
	LoggedInKeyUID string                  `json:"-"`

	TransferringKeystoreFiles bool   `json:"transferringKeystoreFiles"` // informs receiver that only keystore files are expected
	Password                  string `json:"password"`                  // necessary in case of transferring keystores
	KeyUID                    string `json:"keyUID"`                    // necessary in case of transferring keystores
}

type ServerConfig struct {
	// Timeout the number of milliseconds after which the pairing server will automatically terminate
	Timeout uint `json:"timeout" validate:"omitempty,gte=0"`

	// Connection fields, not json (un)marshalled
	// Required for the server, but MUST NOT come from client

	PK       *ecdsa.PublicKey `json:"-"`
	EK       []byte           `json:"-"`
	Cert     *tls.Certificate `json:"-"`
	Hostname string           `json:"-"`
}

type ClientConfig struct{}

type SenderServerConfig struct {
	SenderConfig *SenderConfig `json:"senderConfig" validate:"required"`
	ServerConfig *ServerConfig `json:"serverConfig" validate:"omitempty,dive"`
}

type SenderClientConfig struct {
	SenderConfig *SenderConfig `json:"senderConfig" validate:"required"`
	ClientConfig *ClientConfig `json:"clientConfig"`
}

type ReceiverClientConfig struct {
	ReceiverConfig *ReceiverConfig `json:"receiverConfig" validate:"required"`
	ClientConfig   *ClientConfig   `json:"clientConfig"`
}

type ReceiverServerConfig struct {
	ReceiverConfig *ReceiverConfig `json:"receiverConfig" validate:"required"`
	ServerConfig   *ServerConfig   `json:"serverConfig" validate:"omitempty,dive"`
}

func NewSenderServerConfig() *SenderServerConfig {
	return &SenderServerConfig{
		SenderConfig: new(SenderConfig),
		ServerConfig: new(ServerConfig),
	}
}

func NewSenderClientConfig() *SenderClientConfig {
	return &SenderClientConfig{
		SenderConfig: new(SenderConfig),
		ClientConfig: new(ClientConfig),
	}
}

func NewReceiverClientConfig() *ReceiverClientConfig {
	return &ReceiverClientConfig{
		ReceiverConfig: new(ReceiverConfig),
		ClientConfig:   new(ClientConfig),
	}
}

func NewReceiverServerConfig() *ReceiverServerConfig {
	return &ReceiverServerConfig{
		ReceiverConfig: new(ReceiverConfig),
		ServerConfig:   new(ServerConfig),
	}
}
