package pairing

import (
	"crypto/ecdsa"
	"crypto/tls"
	"encoding/json"
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
	SenderConfig
	ServerConfig
}

type SenderClientConfig struct {
	SenderConfig
	ClientConfig
}

type ReceiverClientConfig struct {
	ReceiverConfig
	ClientConfig
}

type ReceiverServerConfig struct {
	ReceiverConfig
	ServerConfig
}

// VVV Old code below this line VVV

type PayloadSourceSenderConfig struct {
	KeyUID   string `json:"keyUID"`
	Password string `json:"password"`
}

type PayloadSourceReceiverConfig struct {
	KDFIterations int `json:"kdfIterations"`
	NodeConfig    *params.NodeConfig
	// this field already exists within params.NodeConfig, but it doesn't support json marshalling, so we need to duplicate it here
	RootDataDir string
	// corresponding to field current_network from table settings, so that we can override current network from sender
	SettingCurrentNetwork string
}

// PayloadSourceConfig represents location and access data of the pairing payload
// ONLY available from the application client
type PayloadSourceConfig struct {
	// required for sender and receiver, there are some different cases:
	// 1. for sender, KeystorePath must end with keyUID
	// 2. for receiver, KeystorePath must not end with keyUID (because keyUID is not known yet)
	KeystorePath string `json:"keystorePath"`
	// required for sender and receiver, SendPairInstallation need this information
	DeviceType string `json:"deviceType"`
	*PayloadSourceSenderConfig
	*PayloadSourceReceiverConfig
	// Timeout the number of milliseconds after which the pairing server will automatically terminate
	Timeout uint `json:"timeout"`
}

type Config struct {
	// Connection fields
	PK       *ecdsa.PublicKey
	EK       []byte
	Cert     *tls.Certificate
	Hostname string
	Mode     Mode

	// AccountPayload management fields
	*AccountPayloadManagerConfig
}

// AccountPayloadManagerConfig represents the initialisation parameters required for a AccountPayloadManager
type AccountPayloadManagerConfig struct {
	DB *multiaccounts.Database
	*PayloadSourceConfig
	// only used for the receiver side
	LoggedInKeyUID string
}

// Superfluous below this comment, specific config types makes this redundant.
// VVV I like the idea here though VVV

func (a *AccountPayloadManagerConfig) GetNodeConfig() *params.NodeConfig {
	if a.PayloadSourceConfig != nil && a.PayloadSourceConfig.PayloadSourceReceiverConfig != nil {
		return a.NodeConfig
	}
	return nil
}

func (a *AccountPayloadManagerConfig) GetSettingCurrentNetwork() string {
	if a.PayloadSourceConfig != nil && a.PayloadSourceConfig.PayloadSourceReceiverConfig != nil {
		return a.SettingCurrentNetwork
	}
	return ""
}

func (a *AccountPayloadManagerConfig) GetDeviceType() string {
	if a.PayloadSourceConfig != nil {
		return a.DeviceType
	}
	return ""
}

func (a *AccountPayloadManagerConfig) GetPayloadSourceSenderConfig() *PayloadSourceSenderConfig {
	if a.PayloadSourceConfig != nil && a.PayloadSourceConfig.PayloadSourceSenderConfig != nil {
		return a.PayloadSourceSenderConfig
	}
	return nil
}

func (a *AccountPayloadManagerConfig) GetPayloadSourceReceiverConfig() *PayloadSourceReceiverConfig {
	if a.PayloadSourceConfig != nil && a.PayloadSourceConfig.PayloadSourceReceiverConfig != nil {
		return a.PayloadSourceReceiverConfig
	}
	return nil
}

func (a *AccountPayloadManagerConfig) GetKeystorePath() string {
	if a.PayloadSourceConfig != nil {
		return a.KeystorePath
	}
	return ""
}

func (a *AccountPayloadManagerConfig) GetTimeout() uint {
	if a.PayloadSourceConfig != nil {
		return a.Timeout
	}
	return 0
}

type payloadSourceUnmarshalCallback func(conf *PayloadSourceConfig) (*PayloadSourceConfig, error)

func NewPayloadSourceForClient(configJSON string, mode Mode) (*PayloadSourceConfig, error) {
	return unmarshalPayloadSourceConfig(configJSON, func(conf *PayloadSourceConfig) (*PayloadSourceConfig, error) {
		if mode == Sending && conf.NodeConfig == nil {
			return nil, ErrNodeConfigNilAsReceiver
		}
		if mode == Receiving && conf.KeyUID == "" {
			return nil, ErrKeyUIDEmptyAsSender
		}
		return updateRootDataDirToNodeConfig(conf)
	})
}

func NewPayloadSourceForServer(configJSON string, mode Mode) (*PayloadSourceConfig, error) {
	return unmarshalPayloadSourceConfig(configJSON, func(conf *PayloadSourceConfig) (*PayloadSourceConfig, error) {
		if mode == Sending && conf.KeyUID == "" {
			return nil, ErrKeyUIDEmptyAsSender
		}
		if mode == Receiving && conf.NodeConfig == nil {
			return nil, ErrNodeConfigNilAsReceiver
		}
		return updateRootDataDirToNodeConfig(conf)
	})
}

func updateRootDataDirToNodeConfig(conf *PayloadSourceConfig) (*PayloadSourceConfig, error) {
	if conf.PayloadSourceReceiverConfig != nil && conf.PayloadSourceReceiverConfig.NodeConfig != nil {
		conf.NodeConfig.RootDataDir = conf.RootDataDir
	}
	return conf, nil
}

func unmarshalPayloadSourceConfig(configJSON string, successCallback payloadSourceUnmarshalCallback) (*PayloadSourceConfig, error) {
	var conf = PayloadSourceConfig{}
	err := json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return nil, err
	}
	return successCallback(&conf)
}
