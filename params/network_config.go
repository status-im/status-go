package params

import "github.com/ethereum/go-ethereum/common"

// RpcProviderAuthType defines the different types of authentication for RPC providers
type RpcProviderAuthType string

const (
	NoAuth    RpcProviderAuthType = "no-auth"    // No authentication
	BasicAuth RpcProviderAuthType = "basic-auth" // HTTP Header "Authorization: Basic base64(username:password)"
	TokenAuth RpcProviderAuthType = "token-auth" // URL Token-based authentication "https://api.example.com/YOUR_TOKEN"
)

// RpcProviderType defines the type of RPC provider
type RpcProviderType string

const (
	EmbeddedProxyProviderType  RpcProviderType = "embedded-proxy"  // Proxy-based RPC provider
	EmbeddedDirectProviderType RpcProviderType = "embedded-direct" // Direct RPC provider
	UserProviderType           RpcProviderType = "user"            // User-defined RPC provider
)

// RpcProvider represents an RPC provider configuration with various options
type RpcProvider struct {
	ID               int64           `json:"id" validate:"omitempty"`          // Auto-increment ID (for sorting order)
	ChainID          uint64          `json:"chainId" validate:"required,gt=0"` // Chain ID of the network
	Name             string          `json:"name" validate:"required,min=1"`   // Provider name for identification
	URL              string          `json:"url" validate:"required,url"`      // Current Provider URL
	EnableRPSLimiter bool            `json:"enableRpsLimiter"`                 // Enable RPC rate limiting for this provider
	Type             RpcProviderType `json:"type" validate:"required,oneof=embedded-proxy embedded-direct user"`
	Enabled          bool            `json:"enabled"` // Whether the provider is enabled
	// Authentication
	AuthType     RpcProviderAuthType `json:"authType" validate:"required,oneof=no-auth basic-auth token-auth"` // Type of authentication
	AuthLogin    string              `json:"authLogin" validate:"omitempty,min=1"`                             // Login for BasicAuth (empty string if not used)
	AuthPassword string              `json:"authPassword" validate:"omitempty,min=1"`                          // Password for BasicAuth (empty string if not used)
	AuthToken    string              `json:"authToken" validate:"omitempty,min=1"`                             // Token for TokenAuth (empty string if not used)
}

// GetFullURL returns the URL with auth token if TokenAuth is used
func (p RpcProvider) GetFullURL() string {
	if p.AuthType == TokenAuth && p.AuthToken != "" {
		return p.URL + "/" + p.AuthToken
	}
	return p.URL
}

type TokenOverride struct {
	Symbol  string         `json:"symbol"`
	Address common.Address `json:"address"`
}

type Network struct {
	ChainID      uint64        `json:"chainId" validate:"required,gt=0"`
	ChainName    string        `json:"chainName" validate:"required,min=1"`
	RpcProviders []RpcProvider `json:"rpcProviders" validate:"dive,required"` // List of RPC providers, in the order in which they are accessed

	// Deprecated fields (kept for backward compatibility)
	// FIXME: Removal of deprecated fields in integration PR https://github.com/status-im/status-go/pull/6178
	DefaultRPCURL       string `json:"defaultRpcUrl" validate:"omitempty,url"`       // Deprecated: proxy rpc url
	DefaultFallbackURL  string `json:"defaultFallbackURL" validate:"omitempty,url"`  // Deprecated: proxy fallback url
	DefaultFallbackURL2 string `json:"defaultFallbackURL2" validate:"omitempty,url"` // Deprecated: second proxy fallback url
	RPCURL              string `json:"rpcUrl" validate:"omitempty,url"`              // Deprecated: direct rpc url
	OriginalRPCURL      string `json:"originalRpcUrl" validate:"omitempty,url"`      // Deprecated: direct rpc url if user overrides RPCURL
	FallbackURL         string `json:"fallbackURL" validate:"omitempty,url"`         // Deprecated
	OriginalFallbackURL string `json:"originalFallbackURL" validate:"omitempty,url"` // Deprecated

	BlockExplorerURL       string          `json:"blockExplorerUrl,omitempty" validate:"omitempty,url"`
	IconURL                string          `json:"iconUrl,omitempty" validate:"omitempty"`
	NativeCurrencyName     string          `json:"nativeCurrencyName,omitempty" validate:"omitempty,min=1"`
	NativeCurrencySymbol   string          `json:"nativeCurrencySymbol,omitempty" validate:"omitempty,min=1"`
	NativeCurrencyDecimals uint64          `json:"nativeCurrencyDecimals" validate:"omitempty"`
	IsTest                 bool            `json:"isTest"`
	Layer                  uint64          `json:"layer" validate:"omitempty"`
	Enabled                bool            `json:"enabled"`
	ChainColor             string          `json:"chainColor" validate:"omitempty"`
	ShortName              string          `json:"shortName" validate:"omitempty,min=1"`
	TokenOverrides         []TokenOverride `json:"tokenOverrides" validate:"omitempty,dive"`
	RelatedChainID         uint64          `json:"relatedChainId" validate:"omitempty"`
}

func (n *Network) DeepCopy() Network {
	updatedNetwork := *n
	updatedNetwork.RpcProviders = make([]RpcProvider, len(n.RpcProviders))
	copy(updatedNetwork.RpcProviders, n.RpcProviders)
	updatedNetwork.TokenOverrides = make([]TokenOverride, len(n.TokenOverrides))
	copy(updatedNetwork.TokenOverrides, n.TokenOverrides)
	return updatedNetwork
}

func newRpcProvider(chainID uint64, name, url string, enableRpsLimiter bool, providerType RpcProviderType) *RpcProvider {
	return &RpcProvider{
		ChainID:          chainID,
		Name:             name,
		URL:              url,
		EnableRPSLimiter: enableRpsLimiter,
		Type:             providerType,
		Enabled:          true,
		AuthType:         NoAuth,
	}
}

func NewUserProvider(chainID uint64, name, url string, enableRpsLimiter bool) *RpcProvider {
	return newRpcProvider(chainID, name, url, enableRpsLimiter, UserProviderType)
}

func NewProxyProvider(chainID uint64, name, url string, enableRpsLimiter bool) *RpcProvider {
	return newRpcProvider(chainID, name, url, enableRpsLimiter, EmbeddedProxyProviderType)
}

func NewDirectProvider(chainID uint64, name, url string, enableRpsLimiter bool) *RpcProvider {
	return newRpcProvider(chainID, name, url, enableRpsLimiter, EmbeddedDirectProviderType)
}
