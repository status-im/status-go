package pairing

import (
	"crypto/ecdsa"
	"crypto/tls"

	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/params"
)

type SenderConfig struct {
	// SenderConfig.KeystorePath must end with keyUID
	KeystorePath string `json:"keystorePath"`
	// DeviceType SendPairInstallation need this information
	DeviceType string `json:"deviceType"`

	KeyUID   string `json:"keyUID"`
	Password string `json:"password"`

	DB *multiaccounts.Database `json:"-"`
}

type ReceiverConfig struct {
	NodeConfig *params.NodeConfig `json:"nodeConfig"`

	// ReceiverConfig.KeystorePath must not end with keyUID (because keyUID is not known yet)
	KeystorePath string `json:"keystorePath"`
	// DeviceType SendPairInstallation need this information
	DeviceType    string `json:"deviceType"`
	KDFIterations int    `json:"kdfIterations"`
	// SettingCurrentNetwork corresponding to field current_network from table settings, so that we can override current network from sender
	SettingCurrentNetwork string `json:"settingCurrentNetwork"`

	DB             *multiaccounts.Database `json:"-"`
	LoggedInKeyUID string                  `json:"-"`
}

type ServerConfig struct {
	// Timeout the number of milliseconds after which the pairing server will automatically terminate
	Timeout uint `json:"timeout"`

	// Connection fields, not json (un)marshalled
	// Required for the server, but MUST NOT come from client

	PK       *ecdsa.PublicKey `json:"-"`
	EK       []byte           `json:"-"`
	Cert     *tls.Certificate `json:"-"`
	Hostname string           `json:"-"`
	Mode     Mode             `json:"-"`
}

type ClientConfig struct{}

type SenderServerConfig struct {
	SenderConfig *SenderConfig `json:"senderConfig"`
	ServerConfig *ServerConfig `json:"serverConfig"`
}

type SenderClientConfig struct {
	SenderConfig *SenderConfig `json:"senderConfig"`
	ClientConfig *ClientConfig `json:"clientConfig"`
}

type ReceiverClientConfig struct {
	ReceiverConfig *ReceiverConfig `json:"receiverConfig"`
	ClientConfig   *ClientConfig   `json:"clientConfig"`
}

type ReceiverServerConfig struct {
	ReceiverConfig *ReceiverConfig `json:"receiverConfig"`
	ServerConfig   *ServerConfig   `json:"serverConfig"`
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
