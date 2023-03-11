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
	NodeConfig *params.NodeConfig

	// ReceiverConfig.KeystorePath must not end with keyUID (because keyUID is not known yet)
	KeystorePath string `json:"keystorePath"`
	// DeviceType SendPairInstallation need this information
	DeviceType    string `json:"deviceType"`
	KDFIterations int    `json:"kdfIterations"`
	// SettingCurrentNetwork corresponding to field current_network from table settings, so that we can override current network from sender
	SettingCurrentNetwork string

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
	Sender *SenderConfig
	Server *ServerConfig
}

type SenderClientConfig struct {
	Sender *SenderConfig
	Client *ClientConfig
}

type ReceiverClientConfig struct {
	Receiver *ReceiverConfig
	Client   *ClientConfig
}

type ReceiverServerConfig struct {
	Receiver *ReceiverConfig
	Server   *ServerConfig
}

func NewSenderServerConfig() *SenderServerConfig {
	return &SenderServerConfig{
		Sender: new(SenderConfig),
		Server: new(ServerConfig),
	}
}

func NewSenderClientConfig() *SenderClientConfig {
	return &SenderClientConfig{
		Sender: &SenderConfig{},
		Client: &ClientConfig{},
	}
}

func NewReceiverClientConfig() *ReceiverClientConfig {
	return &ReceiverClientConfig{
		Receiver: &ReceiverConfig{},
		Client:   &ClientConfig{},
	}
}

func NewReceiverServerConfig() *ReceiverServerConfig {
	return &ReceiverServerConfig{
		Receiver: new(ReceiverConfig),
		Server:   new(ServerConfig),
	}
}
