package requests

type Login struct {
	KeyUID   string `json:"keyUid"`
	Password string `json:"password"`

	RuntimeLogLevel string `json:"runtimeLogLevel"`
	//WakuV2Nameserver      string `json:"wakuV2Nameserver"`
	//BandwidthStatsEnabled bool   `json:"bandwidthStatsEnabled"`

	KeycardWhisperPrivateKey string `json:"keycardWhisperPrivateKey"`

	// Mnemonic allows to log in to an account when password is lost.
	// This is needed for the "Lost keycard -> Start using without keycard" flow, when a keycard account database
	// exists locally, but now the keycard is lost. In this case client is responsible for calling
	// `convertToRegularAccount` after a successful login. This could be improved in the future.
	// When non-empty, mnemonic is used to generate required keypairs and:
	// - Password is ignored and replaced with encryption public key
	// - KeycardWhisperPrivateKey is ignored and replaced with chat private key
	// - KeyUID is ignored and replaced with hash of the master public key
	// FIXME: Move this flow into a separate endpoint
	Mnemonic string `json:"mnemonic"`

	WalletConfig
	WalletSecretsConfig

	APIConfig          *APIConfig `json:"apiConfig"`
	StatusProxyEnabled bool       `json:"statusProxyEnabled"`
}
