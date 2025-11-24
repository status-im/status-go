package requests

import (
	"github.com/status-im/status-go/protocol/requests"
)

// Login represents a request to log in to a non-keycard account.
type Login struct {
	// KeyUID is the UID of the account to log in to.
	// Can be obtained from `listAccounts` RPC.
	KeyUID string `json:"keyUid"`

	// Password is the password of the account to log in to.
	// It is applied as is. Hashing should be done by the client if needed.
	Password string `json:"password"`

	// RuntimeConfig contains configuration parameters for the services.
	// This parameter contains required configuration parameters for some services. Services will fail to start
	// if they are missing. This parameter is required (if a corresponding service is enabled).
	// Values are only applied for a single run and not persisted.
	RuntimeConfig RuntimeConfig `json:"runtimeConfig,omitempty"`

	// RuntimeOverrides allows overriding some default and persisted options.
	// This parameter is optional. Values are only applied for a single run and not persisted.
	RuntimeOverrides RuntimeOverrides `json:"runtimeOverrides,omitempty"`

	// FIXME: Move to KeycardLogin request
	//KeycardWhisperPrivateKey string `json:"keycardWhisperPrivateKey"`

	// Mnemonic allows to log in to an account when password is lost.
	// This is needed for the "Lost keycard -> Start using without keycard" flow, when a keycard account database
	// exists locally, but now the keycard is lost. In this case client is responsible for calling
	// `convertToRegularAccount` after a successful login. This could be improved in the future.
	// When non-empty, mnemonic is used to generate required keypairs and:
	// - Password is ignored and replaced with encryption public key
	// - KeycardWhisperPrivateKey is ignored and replaced with chat private key
	// - KeyUID is ignored and replaced with hash of the master public key
	// FIXME: Move this flow into a separate endpoint
	//Mnemonic string `json:"mnemonic"`
}

type RuntimeConfig struct {
	// WARNING: make sure this structure only contain required configuration parameters. WalletConfig definitely mixes approaches.
	// TODO: Group by services
	WalletConfig       requests.WalletConfig
	WalletSecrets      requests.WalletSecretsConfig
	APIConfig          *requests.APIConfig `json:"apiConfig"`
	StatusProxyEnabled bool                `json:"statusProxyEnabled"`
}

type RuntimeOverrides struct {
	LogLevel              *string `json:"logLevel"`
	WakuV2Nameserver      *string `json:"wakuV2Nameserver"`
	BandwidthStatsEnabled *bool   `json:"bandwidthStatsEnabled"`
}
