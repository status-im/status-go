package walletconnect

import (
	"reflect"
	"strconv"
	"testing"

	"encoding/json"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"

	"github.com/stretchr/testify/assert"
)

func getSessionJSONFor(chains []int, expiry int) string {
	chainsStr := "["
	for i, chain := range chains {
		chainsStr += `"eip155:` + strconv.Itoa(chain) + `"`
		if i != len(chains)-1 {
			chainsStr += ","
		}
	}
	chainsStr += "]"
	expiryStr := strconv.Itoa(expiry)

	return `{
		"expiry": ` + expiryStr + `,
		"namespaces": {
			"eip155": {
				"accounts": [
					"eip155:1:0x7F47C2e18a4BBf5487E6fb082eC2D9Ab0E6d7240",
					"eip155:10:0x7F47C2e18a4BBf5487E6fb082eC2D9Ab0E6d7240",
					"eip155:42161:0x7F47C2e18a4BBf5487E6fb082eC2D9Ab0E6d7240"
				],
				"chains": ` + chainsStr + `,
				"events": [
					"accountsChanged",
					"chainChanged"
				],
				"methods": [
					"eth_sendTransaction",
					"personal_sign"
				]
			}
		},
		"optionalNamespaces": {
			"eip155": {
				"chains": [],
				"events": [],
				"methods": [],
				"rpcMap": {}
			}
		},
		"pairingTopic": "50fba141cdb5c015493c2907c46bacf9f7cbd7c8e3d4e97df891f18dddcff69c",
		"peer": {
			"metadata": {
				"description": "Test Dapp Description",
				"icons": [ "https://test.org/test.png"],
				"name": "Test Dapp",
				"url": "https://dapp.test.org"
			},
			"publicKey": "1234567890aeb6081cabed26faf48919162fd70cc66d639f118a60507ae0463d"
		},
		"relay": { "protocol": "irn"},
		"requiredNamespaces": {
			"eip155": {
				"chains": [
					"eip155:1"
				],
				"events": [
					"chainChanged",
					"accountsChanged"
				],
				"methods": [
					"eth_sendTransaction",
					"personal_sign"
				],
				"rpcMap": {
					"1": "https://mainnet.infura.io/v3/099fc58e0de9451d80b18d7c74caa7c1"
				}
			}
		},
		"self": {
			"metadata": {
				"description": "Test Wallet Description",
				"icons": [
					"https://wallet.test.org/test.svg"
				],
				"name": "Test Wallet",
				"url": "http://localhost"
			},
			"publicKey": "da4a87d5f0f54951afe870ebf020cf03f8a3522fbd219398c3fa159a37e16d54"
		},
		"topic": "e39e1f435a46b5ee6b31484d1751cfbc35be1275653af2ea340974a7592f1a19"
	}`
}

func Test_sessionProposalValidity(t *testing.T) {
	tests := []struct {
		name                string
		sessionProposalJSON string
		expectedValidity    bool
	}{
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#11-proposal-namespaces-does-not-include-an-optional-namespace
		{
			name: "proposal-namespaces-does-not-include-an-optional-namespace",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"eip155:10": {
								"methods": ["personal_sign"],
								"events": ["accountsChanged", "chainChanged"]
							}
						}
					}
				}`,
			expectedValidity: true,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#12-proposal-namespaces-must-not-have-chains-empty
		{
			name: "proposal-namespaces-must-not-have-chains-empty",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"cosmos": {
								"chains": [],
								"methods": ["cosmos_signDirect"],
								"events": ["someCosmosEvent"]
							}
						}
					}
				}`,
			expectedValidity: false,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#13-chains-might-be-omitted-if-the-caip-2-is-defined-in-the-index
		{
			name: "chains-might-be-omitted-if-the-caip-2-is-defined-in-the-index",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"eip155": {
								"chains": ["eip155:1", "eip155:137"],
								"methods": ["eth_sendTransaction", "eth_signTransaction", "eth_sign"],
								"events": ["accountsChanged", "chainChanged"]
							},
							"eip155:10": {
								"methods": ["personal_sign"],
								"events": ["accountsChanged", "chainChanged"]
							}
						}
					}
				}`,
			expectedValidity: true,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#14-chains-must-be-caip-2-compliant
		{
			name: "chains-must-be-caip-2-compliant",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"eip155": {
								"chains": ["42"],
								"methods": ["eth_sign"],
								"events": ["accountsChanged"]
							}
						}
					}
				}`,
			expectedValidity: false,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#15-proposal-namespace-methods-and-events-may-be-empty
		{
			name: "proposal-namespace-methods-and-events-may-be-empty",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"eip155": {
								"chains": ["eip155:1"],
								"methods": [],
								"events": []
							}
						}
					}
				}`,
			expectedValidity: true,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#16-all-chains-in-the-namespace-must-contain-the-namespace-prefix
		{
			name: "all-chains-in-the-namespace-must-contain-the-namespace-prefix",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"eip155": {
								"chains": ["eip155:1", "eip155:137", "cosmos:cosmoshub-4"],
								"methods": ["eth_sendTransaction"],
								"events": ["accountsChanged", "chainChanged"]
							}
						},
						"optionalNamespaces": {
							"eip155:42161": {
								"methods": ["personal_sign"],
								"events": ["accountsChanged", "chainChanged"]
							}
						}
					}
				}`,
			expectedValidity: false,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#17-namespace-key-must-comply-with-caip-2-specification
		{
			name: "namespace-key-must-comply-with-caip-2-specification",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"": {
								"chains": [":1"],
								"methods": ["personalSign"],
								"events": []
							},
							"**": {
								"chains": ["**:1"],
								"methods": ["personalSign"],
								"events": []
							}
						}
					}
				}`,
			expectedValidity: false,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#18-all-namespaces-must-be-valid
		{
			name: "all-namespaces-must-be-valid",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {
							"eip155": {
								"chains": ["eip155:1"],
								"methods": ["personalSign"],
								"events": []
							},
							"cosmos": {
								"chains": [],
								"methods": [],
								"events": []
							}
						}
					}
				}`,
			expectedValidity: false,
		},
		// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#19-proposal-namespaces-may-be-empty
		{
			name: "proposal-namespaces-may-be-empty",
			sessionProposalJSON: `{
					"params": {
						"requiredNamespaces": {}
					}
				}`,
			expectedValidity: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sessionProposal SessionProposal
			err := json.Unmarshal([]byte(tt.sessionProposalJSON), &sessionProposal)
			assert.NoError(t, err)

			validRes := sessionProposal.ValidateProposal()
			if tt.expectedValidity {
				assert.True(t, validRes)
			} else {
				assert.False(t, validRes)
			}
		})
	}
}

func Test_supportedChainInSession(t *testing.T) {
	type args struct {
		sessionProposal Session
	}
	tests := []struct {
		name           string
		args           args
		expectedChains []uint64
	}{
		{
			name: "supported_chain",
			args: args{
				sessionProposal: Session{
					Namespaces: map[string]Namespace{
						"eip155": {
							Chains: []string{"eip155:1", "eip155:2", "eip155:3", "eip155:4", "eip155:5"},
						},
					},
				},
			},
			expectedChains: []uint64{1, 2, 3, 4, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChains := supportedChainsInSession(tt.args.sessionProposal)
			if !reflect.DeepEqual(gotChains, tt.expectedChains) {
				t.Errorf("supportedChainInSessionProposal() gotChains = %v, want %v", gotChains, tt.expectedChains)
			}
		})
	}
}

func Test_caip10Accounts(t *testing.T) {
	type args struct {
		accounts []*accounts.Account
		chains   []uint64
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "generate_caip10_accounts",
			args: args{
				accounts: []*accounts.Account{
					{
						Address: types.HexToAddress("0x1"),
						Type:    accounts.AccountTypeWatch,
					},
					{
						Address: types.HexToAddress("0x2"),
						Type:    accounts.AccountTypeSeed,
					},
				},
				chains: []uint64{1, 2},
			},
			want: []string{
				"eip155:1:0x0000000000000000000000000000000000000001",
				"eip155:2:0x0000000000000000000000000000000000000001",
				"eip155:1:0x0000000000000000000000000000000000000002",
				"eip155:2:0x0000000000000000000000000000000000000002",
			},
		},
		{
			name: "empty_addresses",
			args: args{
				accounts: []*accounts.Account{},
				chains:   []uint64{1, 2},
			},
			want: []string{},
		},
		{
			name: "empty_chains",
			args: args{
				accounts: []*accounts.Account{
					{
						Address: types.HexToAddress("0x1"),
						Type:    accounts.AccountTypeWatch,
					},
					{
						Address: types.HexToAddress("0x2"),
						Type:    accounts.AccountTypeSeed,
					},
				},
				chains: []uint64{},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := caip10Accounts(tt.args.accounts, tt.args.chains); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("caip10Accounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_AddSession validates that the new added session is active (not expired and not disconnected)
func Test_AddSession(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// Add session for testnet
	expiry := 1716581732
	sessionJSON := getSessionJSONFor([]int{11155111}, expiry)
	networks := []params.Network{
		{ChainID: 1, IsTest: false},
		{ChainID: 11155111, IsTest: true},
	}
	err := AddSession(db, networks, sessionJSON)
	assert.NoError(t, err)

	dapps, err := GetActiveDapps(db, int64(expiry-1), true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dapps))
}
