package networkdefaults

import (
	"fmt"
	"strings"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	// Host suffixes for providers
	SmartProxyHostSuffix = "eth-rpc.status.im"

	iconURLPrefix         = "network/"
	testNetworkIconSuffix = "-test"
)

// New eth-rpc-proxy endpoint (provider agnostic)
func getProxyHost(customUrl, stageName string) string {
	if customUrl != "" {
		return strings.TrimRight(customUrl, "/")
	}
	return fmt.Sprintf("https://%s.%s", stageName, SmartProxyHostSuffix)
}

// New eth-rpc-proxy endpoint with smart proxy URL
func smartProxyUrl(proxyHost, chainName, networkName string) security.SensitiveString {
	return security.NewSensitiveStringPrintf("%s/%s/%s/", proxyHost, chainName, networkName)
}

// New eth-rpc-proxy endpoint with smart proxy URL tragetting a specific provider
func smartProxyUrlWithProvider(proxyHost, chainName, networkName, provider string) security.SensitiveString {
	return security.NewSensitiveStringPrintf("%s/%s/%s/%s", proxyHost, chainName, networkName, provider)
}

type providerType uint8

const (
	providerTypeEthRpcProxy providerType = iota
	providerTypeEthRpcProxyWithProvider
	providerTypeDirect
)

type providerSpec struct {
	providerType     providerType
	name             string
	enableRpsLimiter bool

	// For providerKindDirect
	directURL security.SensitiveString

	// For providerKindEthRpcProxyWithProvider (URL path component)
	targetProvider string
}

type networkSpec struct {
	chainID uint64

	// Used for status smart-proxy URL generation
	proxyChainName   string
	proxyNetworkName string

	chainName              string
	providers              []providerSpec
	blockExplorerURL       string
	iconURL                string // icon URL is the same as the proxy chain name, in case of test networks, "-test" is appended to the end
	chainColor             string
	shortName              string
	nativeCurrencyName     string
	nativeCurrencySymbol   string
	nativeCurrencyDecimals uint64
	isTest                 bool
	layer                  uint64
	enabled                bool
	relatedChainID         uint64
	isActive               bool
	isDeactivatable        bool
}

// configEnv groups runtime options
type configEnv struct {
	ProxyHost          string
	EnableRpcProviders bool
	UsePuzzleAuth      bool
}

func buildRpcProvidersFromSpec(spec networkSpec, env configEnv) []params.RpcProvider {
	if !env.EnableRpcProviders {
		return []params.RpcProvider{}
	}

	out := make([]params.RpcProvider, 0, len(spec.providers))
	for _, p := range spec.providers {
		switch p.providerType {
		case providerTypeEthRpcProxy:
			out = append(out, *params.NewEthRpcProxyProvider(spec.chainID, p.name, smartProxyUrl(env.ProxyHost, spec.proxyChainName, spec.proxyNetworkName), false, env.UsePuzzleAuth))
		case providerTypeEthRpcProxyWithProvider:
			out = append(out, *params.NewEthRpcProxyProvider(spec.chainID, p.name, smartProxyUrlWithProvider(env.ProxyHost, spec.proxyChainName, spec.proxyNetworkName, p.targetProvider), false, env.UsePuzzleAuth))
		case providerTypeDirect:
			out = append(out, *params.NewDirectProvider(spec.chainID, p.name, p.directURL, p.enableRpsLimiter))
		}
	}

	return out
}

func buildNetworkFromSpec(spec networkSpec, env configEnv) params.Network {
	iconURL := fmt.Sprintf("%s%s", iconURLPrefix, spec.proxyChainName)
	if spec.isTest {
		iconURL += testNetworkIconSuffix
	}
	return params.Network{
		ChainID:                spec.chainID,
		ChainName:              spec.chainName,
		RpcProviders:           buildRpcProvidersFromSpec(spec, env),
		BlockExplorerURL:       spec.blockExplorerURL,
		IconURL:                iconURL,
		ChainColor:             spec.chainColor,
		ShortName:              spec.shortName,
		NativeCurrencyName:     spec.nativeCurrencyName,
		NativeCurrencySymbol:   spec.nativeCurrencySymbol,
		NativeCurrencyDecimals: spec.nativeCurrencyDecimals,
		IsTest:                 spec.isTest,
		Layer:                  spec.layer,
		Enabled:                spec.enabled,
		RelatedChainID:         spec.relatedChainID,
		IsActive:               spec.isActive,
		IsDeactivatable:        spec.isDeactivatable,
	}
}

func defaultNetworks(proxyHost string, thirdpartyServicesEnabled bool, usePuzzleAuth bool) []params.Network {
	env := configEnv{
		ProxyHost:          proxyHost,
		EnableRpcProviders: thirdpartyServicesEnabled,
		UsePuzzleAuth:      usePuzzleAuth,
	}
	networks := make([]params.Network, 0, len(defaultNetworkSpecs))
	for _, spec := range defaultNetworkSpecs {
		networks = append(networks, buildNetworkFromSpec(spec, env))
	}
	return networks
}

func setRPCs(networks []params.Network, walletConfig *requests.WalletSecretsConfig) []params.Network {
	authTokens := map[string]security.SensitiveString{
		"infura.io":  walletConfig.InfuraToken,
		"grove.city": walletConfig.PoktToken,
	}
	networks = networkhelper.OverrideDirectProvidersAuth(networks, authTokens)

	// Apply auth for new smart proxy
	hasSmartProxyCredentials := !walletConfig.EthRpcProxyUser.Empty() && !walletConfig.EthRpcProxyPassword.Empty()
	networks = networkhelper.OverrideBasicAuth(
		networks,
		params.EmbeddedEthRpcProxyProviderType,
		hasSmartProxyCredentials,
		walletConfig.EthRpcProxyUser,
		walletConfig.EthRpcProxyPassword)

	return networks
}

func BuildDefaultNetworks(walletSecretsConfig *requests.WalletSecretsConfig, thirdpartyServicesEnabled bool) []params.Network {
	proxyHost := getProxyHost(walletSecretsConfig.EthRpcProxyUrl.Reveal(), walletSecretsConfig.StatusProxyStageName)
	networks := setRPCs(
		defaultNetworks(proxyHost, thirdpartyServicesEnabled, walletSecretsConfig.EthRpcProxyUsePuzzleAuth),
		walletSecretsConfig,
	)
	return networkhelper.WithCommunitiesSupported(networks)
}

func GetProxyChainAndNetworkName(chainID uint64) (string, string) {
	for _, spec := range defaultNetworkSpecs {
		if spec.chainID == chainID {
			return spec.proxyChainName, spec.proxyNetworkName
		}
	}
	return "", ""
}
