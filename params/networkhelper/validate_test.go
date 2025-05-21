package networkhelper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/security"
)

func TestValidation(t *testing.T) {
	validate := networkhelper.GetValidator()

	// Test cases for RpcProvider
	providerTests := []struct {
		name      string
		provider  params.RpcProvider
		expectErr bool
	}{
		{
			name: "Valid Provider",
			provider: params.RpcProvider{
				ChainID:          1,
				Name:             "Mainnet Provider",
				URL:              security.NewSensitiveString("https://provider.example.com"),
				Type:             params.UserProviderType,
				Enabled:          true,
				AuthType:         params.NoAuth,
				EnableRPSLimiter: false,
			},
			expectErr: false,
		},
		{
			name: "Missing Provider Name",
			provider: params.RpcProvider{
				ChainID: 1,
				URL:     security.NewSensitiveString("https://provider.example.com"),
				Type:    params.UserProviderType,
			},
			expectErr: true,
		},
		{
			name: "Invalid AuthType",
			provider: params.RpcProvider{
				ChainID:  1,
				Name:     "Invalid Auth Provider",
				URL:      security.NewSensitiveString("https://provider.example.com"),
				Type:     params.UserProviderType,
				AuthType: "invalid-auth-type",
			},
			expectErr: true,
		},
		{
			name: "BasicAuth without Login",
			provider: params.RpcProvider{
				ChainID:  1,
				Name:     "BasicAuth Provider",
				URL:      security.NewSensitiveString("https://provider.example.com"),
				Type:     params.UserProviderType,
				AuthType: params.BasicAuth,
			},
			expectErr: true,
		},
		{
			name: "TokenAuth without Token",
			provider: params.RpcProvider{
				ChainID:  1,
				Name:     "TokenAuth Provider",
				URL:      security.NewSensitiveString("https://provider.example.com"),
				Type:     params.UserProviderType,
				AuthType: params.TokenAuth,
			},
			expectErr: true,
		},
		{
			name: "NoAuth with Login",
			provider: params.RpcProvider{
				ChainID:   1,
				Name:      "NoAuth Provider",
				URL:       security.NewSensitiveString("https://provider.example.com"),
				Type:      params.UserProviderType,
				AuthType:  params.NoAuth,
				AuthLogin: security.NewSensitiveString("user"),
			},
			expectErr: true,
		},
	}

	for _, test := range providerTests {
		t.Run("RpcProvider: "+test.name, func(t *testing.T) {
			err := validate.Struct(test.provider)
			if test.expectErr {
				require.Error(t, err, "Expected error but got nil for test case '%s'", test.name)
			} else {
				require.NoError(t, err, "Did not expect error but got '%v' for test case '%s'", err, test.name)
			}
		})
	}
}

// Test cases for Network
func TestNetworkValidation(t *testing.T) {
	validate := validator.New()

	networkTests := []struct {
		name      string
		network   params.Network
		expectErr bool
	}{
		{
			name: "Valid Network",
			network: params.Network{
				ChainID:                1,
				ChainName:              "Ethereum Mainnet",
				BlockExplorerURL:       "https://etherscan.io",
				NativeCurrencyName:     "Ether",
				NativeCurrencySymbol:   "ETH",
				NativeCurrencyDecimals: 18,
				IsTest:                 false,
				Layer:                  1,
				Enabled:                true,
				ChainColor:             "#E90101",
				ShortName:              "eth",
				RpcProviders: []params.RpcProvider{
					{
						ChainID:  1,
						Name:     "Mainnet Provider",
						URL:      security.NewSensitiveString("https://provider.example.com"),
						Type:     params.UserProviderType,
						Enabled:  true,
						AuthType: params.NoAuth,
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Missing Chain Name",
			network: params.Network{
				ChainID: 1,
				RpcProviders: []params.RpcProvider{
					{
						ChainID: 1,
						Name:    "Mainnet Provider",
						URL:     security.NewSensitiveString("https://provider.example.com"),
						Type:    params.UserProviderType,
						Enabled: true,
					},
				},
			},
			expectErr: true,
		},
		{
			name: "Invalid Provider in Network",
			network: params.Network{
				ChainID:   1,
				ChainName: "Ethereum Mainnet",
				RpcProviders: []params.RpcProvider{
					{
						ChainID: 1,
						Name:    "",
						URL:     security.NewSensitiveString("https://provider.example.com"),
						Type:    params.UserProviderType,
						Enabled: true,
					},
				},
			},
			expectErr: true,
		},
	}

	for _, test := range networkTests {
		t.Run("Network: "+test.name, func(t *testing.T) {
			err := validate.Struct(test.network)
			if test.expectErr {
				require.Error(t, err, "Expected error but got nil for test case '%s'", test.name)
			} else {
				require.NoError(t, err, "Did not expect error but got '%v' for test case '%s'", err, test.name)
			}
		})
	}
}
