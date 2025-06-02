package networkhelper

import (
	"net/url"
	"strings"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/rpc/network/db"
)

// MergeProvidersPreservingUsersAndEnabledState merges new embedded providers with the current ones,
// preserving user-defined providers and maintaining the Enabled state.
func MergeProvidersPreservingUsersAndEnabledState(currentProviders, newProviders []params.RpcProvider) []params.RpcProvider {
	// Create a map for quick lookup of the Enabled state by Name
	enabledState := make(map[string]bool, len(currentProviders))
	for _, provider := range currentProviders {
		enabledState[provider.Name] = provider.Enabled
	}

	// Update the Enabled field in newProviders if the Name matches
	for i := range newProviders {
		if enabled, exists := enabledState[newProviders[i].Name]; exists {
			newProviders[i].Enabled = enabled
		}
	}

	// Retain current providers of type UserProviderType and add them to the beginning of the list
	mergedProviders := make([]params.RpcProvider, 0, len(currentProviders)+len(newProviders))
	for _, provider := range currentProviders {
		if provider.Type == params.UserProviderType {
			mergedProviders = append(mergedProviders, provider)
		}
	}

	// Add the updated newProviders
	mergedProviders = append(mergedProviders, newProviders...)

	return mergedProviders
}

// ToggleUserProviders enables or disables all user-defined providers and disables other types.
func ToggleUserProviders(providers []params.RpcProvider, enabled bool) []params.RpcProvider {
	for i := range providers {
		if providers[i].Type == params.UserProviderType {
			providers[i].Enabled = enabled
		} else {
			providers[i].Enabled = !enabled
		}
	}
	return providers
}

// GetEmbeddedProviders returns the embedded providers from the list.
func GetEmbeddedProviders(providers []params.RpcProvider) []params.RpcProvider {
	embeddedProviders := make([]params.RpcProvider, 0, len(providers))
	for _, provider := range providers {
		if provider.Type != params.UserProviderType {
			embeddedProviders = append(embeddedProviders, provider)
		}
	}
	return embeddedProviders
}

// GetUserProviders returns the user-defined providers from the list.
func GetUserProviders(providers []params.RpcProvider) []params.RpcProvider {
	userProviders := make([]params.RpcProvider, 0, len(providers))
	for _, provider := range providers {
		if provider.Type == params.UserProviderType {
			userProviders = append(userProviders, provider)
		}
	}
	return userProviders
}

// ReplaceUserProviders replaces user-defined providers with new ones, retaining the rest of the providers.
func ReplaceUserProviders(currentProviders, newUserProviders []params.RpcProvider) []params.RpcProvider {
	// Extract embedded providers from the current list
	embeddedProviders := GetEmbeddedProviders(currentProviders)
	userProviders := GetUserProviders(newUserProviders)

	// Combine new user providers with the existing embedded providers
	return append(userProviders, embeddedProviders...)
}

// ReplaceEmbeddedProviders replaces embedded providers with new ones, retaining user-defined providers.
func ReplaceEmbeddedProviders(currentProviders, newEmbeddedProviders []params.RpcProvider) []params.RpcProvider {
	// Extract user-defined providers from the current list
	userProviders := GetUserProviders(currentProviders)
	embeddedProviders := GetEmbeddedProviders(newEmbeddedProviders)

	// Combine existing user-defined providers with the new embedded providers
	return append(userProviders, embeddedProviders...)
}

// OverrideBasicAuth updates providers of the specified type in the given networks.
// It sets the `Enabled` flag and configures the `AuthLogin` and `AuthPassword` for each matching provider.
func OverrideBasicAuth(networks []params.Network, providerType params.RpcProviderType, enabled bool, user, password security.SensitiveString) []params.Network {
	updatedNetworks := DeepCopyNetworks(networks)

	for i := range updatedNetworks {
		network := &updatedNetworks[i]
		for j := range network.RpcProviders {
			provider := &network.RpcProviders[j]
			if provider.Type == providerType {
				provider.Enabled = enabled
				provider.AuthType = params.BasicAuth
				provider.AuthLogin = user
				provider.AuthPassword = password
			}
		}
	}

	return updatedNetworks
}

func DeepCopyNetworks(networks []params.Network) []params.Network {
	updatedNetworks := make([]params.Network, len(networks))
	for i, network := range networks {
		updatedNetworks[i] = network.DeepCopy()
	}
	return updatedNetworks
}

func OverrideDirectProvidersAuth(networks []params.Network, authTokens map[string]security.SensitiveString) []params.Network {
	updatedNetworks := DeepCopyNetworks(networks)

	for i := range updatedNetworks {
		network := &updatedNetworks[i]

		for j := range network.RpcProviders {
			provider := &network.RpcProviders[j]

			if provider.Type != params.EmbeddedDirectProviderType {
				continue
			}

			host, err := extractHost(provider.URL.Reveal())
			if err != nil {
				continue
			}

			for suffix, token := range authTokens {
				if strings.HasSuffix(host, suffix) && !token.Empty() {
					provider.AuthType = params.TokenAuth
					provider.AuthToken = token
					break
				}
			}
		}
		db.FillDeprecatedURLs(network, network.RpcProviders)
	}
	return updatedNetworks
}

func extractHost(providerURL string) (string, error) {
	parsedURL, err := url.Parse(providerURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Host, nil
}
