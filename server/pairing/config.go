package pairing

import (
	"crypto/ecdsa"
	"crypto/tls"
	"net"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/requests"
)

type SenderConfig struct {
	// SenderConfig.KeystorePath must end with keyUID
	KeystorePath string `json:"keystorePath" validate:"required,keystorepath"`
	// DeviceType SendPairInstallation need this information
	DeviceType string `json:"deviceType" validate:"required"`

	KeyUID   string `json:"keyUID" validate:"required,keyuid"`
	Password string `json:"password" validate:"required"`
	ChatKey  string `json:"chatKey"` // set only in case of a Keycard user, otherwise empty

	DB *multiaccounts.Database `json:"-"`
}

type ReceiverConfig struct {
	CreateAccount *requests.CreateAccount `json:"createAccount" validate:"required"`

	// DeviceType SendPairInstallation need this information
	// Deprecated: This field will be automatically overridden with runtime.GOOS and can be omitted by client.
	// The field will be removed in https://github.com/status-im/status-go/issues/3351 is fully implemented.
	DeviceType string `json:"-"`

	DB             *multiaccounts.Database `json:"-"`
	LoggedInKeyUID string                  `json:"-"`
}

type KeystoreFilesConfig struct {
	KeystorePath   string `json:"keystorePath" validate:"required,keystorepath"`
	LoggedInKeyUID string `json:"loggedInKeyUid" validate:"required,keyuid"`
	Password       string `json:"password" validate:"required"`
}

type KeystoreFilesSenderConfig struct {
	KeystoreFilesConfig
	KeypairsToExport []string `json:"keypairsToExport" validate:"required"`
}

type KeystoreFilesReceiverConfig struct {
	KeystoreFilesConfig
	KeypairsToImport []string `json:"keypairsToImport" validate:"required"`
}

type ServerConfig struct {
	// Timeout the number of milliseconds after which the pairing server will automatically terminate
	Timeout uint `json:"timeout" validate:"omitempty,gte=0"`

	// Connection fields, not json (un)marshalled
	// Required for the server, but MUST NOT come from client

	PK             *ecdsa.PublicKey `json:"-"`
	EK             []byte           `json:"-"`
	Cert           *tls.Certificate `json:"-"`
	ListenIP       net.IP           `json:"-"`
	IPAddresses    []net.IP         `json:"-"`
	InstallationID string           `json:"-"`
	KeyUID         string           `json:"-"`
}

type ClientConfig struct{}

type SenderServerConfig struct {
	SenderConfig *SenderConfig `json:"senderConfig" validate:"required"`
	ServerConfig *ServerConfig `json:"serverConfig" validate:"omitempty,dive"`
}

type KeystoreFilesSenderServerConfig struct {
	SenderConfig *KeystoreFilesSenderConfig `json:"senderConfig" validate:"required"`
	ServerConfig *ServerConfig              `json:"serverConfig" validate:"omitempty,dive"`
}

type SenderClientConfig struct {
	SenderConfig *SenderConfig `json:"senderConfig" validate:"required"`
	ClientConfig *ClientConfig `json:"clientConfig"`
}

type ReceiverClientConfig struct {
	ReceiverConfig *ReceiverConfig `json:"receiverConfig" validate:"required"`
	ClientConfig   *ClientConfig   `json:"clientConfig"`
}

type KeystoreFilesReceiverClientConfig struct {
	ReceiverConfig *KeystoreFilesReceiverConfig `json:"receiverConfig" validate:"required"`
	ClientConfig   *ClientConfig                `json:"clientConfig"`
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

func NewKeystoreFilesSenderServerConfig() *KeystoreFilesSenderServerConfig {
	return &KeystoreFilesSenderServerConfig{
		SenderConfig: new(KeystoreFilesSenderConfig),
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

func NewKeystoreFilesReceiverClientConfig() *KeystoreFilesReceiverClientConfig {
	return &KeystoreFilesReceiverClientConfig{
		ReceiverConfig: new(KeystoreFilesReceiverConfig),
		ClientConfig:   new(ClientConfig),
	}
}

func NewReceiverServerConfig() *ReceiverServerConfig {
	return &ReceiverServerConfig{
		ReceiverConfig: new(ReceiverConfig),
		ServerConfig:   new(ServerConfig),
	}
}

func (c *ReceiverConfig) AbsoluteKeystorePath() string {
	// Follow the same path as in InitKeyStoreDirWithAccount
	// Keep keyUID empty as it's unknown yet
	_, path := api.DefaultKeystorePath(c.CreateAccount.RootDataDir, "")
	return path
}
