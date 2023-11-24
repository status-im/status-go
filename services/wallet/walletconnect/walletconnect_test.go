package walletconnect

import (
	"reflect"
	"testing"

	"encoding/json"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"

	"github.com/stretchr/testify/assert"
)

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

			if tt.expectedValidity {
				assert.True(t, sessionProposal.Valid())
			} else {
				assert.False(t, sessionProposal.Valid())
			}
		})
	}
}

func Test_sessionProposalToSupportedChain(t *testing.T) {
	type args struct {
		chains        []string
		supportsChain func(uint64) bool
	}
	tests := []struct {
		name          string
		args          args
		wantChains    []uint64
		wantEipChains []string
	}{
		{
			name: "filter_out_unsupported_chains_and_invalid_chains",
			args: args{
				chains: []string{"eip155:1", "eip155:3", "eip155:invalid"},
				supportsChain: func(chainID uint64) bool {
					return chainID == 1
				},
			},
			wantChains:    []uint64{1},
			wantEipChains: []string{"eip155:1"},
		},
		{
			name: "no_supported_chains",
			args: args{
				chains: []string{"eip155:3", "eip155:5"},
				supportsChain: func(chainID uint64) bool {
					return false
				},
			},
			wantChains:    []uint64{},
			wantEipChains: []string{},
		},
		{
			name: "empty_proposal",
			args: args{
				chains: []string{},
				supportsChain: func(chainID uint64) bool {
					return true
				},
			},
			wantChains:    []uint64{},
			wantEipChains: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChains, gotEipChains := sessionProposalToSupportedChain(tt.args.chains, tt.args.supportsChain)
			if !reflect.DeepEqual(gotChains, tt.wantChains) {
				t.Errorf("sessionProposalToSupportedChain() gotChains = %v, want %v", gotChains, tt.wantChains)
			}
			if !reflect.DeepEqual(gotEipChains, tt.wantEipChains) {
				t.Errorf("sessionProposalToSupportedChain() gotEipChains = %v, want %v", gotEipChains, tt.wantEipChains)
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
