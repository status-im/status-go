package walletconnect

import (
	"crypto/ecdsa"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
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

			validRes := sessionProposal.ValidateProposal()
			if tt.expectedValidity {
				assert.True(t, validRes)
			} else {
				assert.False(t, validRes)
			}
		})
	}
}

func Test_Namespace_Valid(t *testing.T) {
	tests := []struct {
		name          string
		namespace     Namespace
		namespaceName string
		chainID       *uint64
		want          bool
	}{
		{
			name: "nil_chainID_empty_chains",
			namespace: Namespace{
				Chains: []string{},
			},
			namespaceName: "eip155",
			chainID:       nil,
			want:          false,
		},
		{
			name: "nil_chainID_valid_chains",
			namespace: Namespace{
				Chains: []string{"eip155:1", "eip155:137"},
			},
			namespaceName: "eip155",
			chainID:       nil,
			want:          true,
		},
		{
			name: "nil_chainID_namespace_mismatch",
			namespace: Namespace{
				Chains: []string{"eip155:1", "cosmos:cosmoshub-4"},
			},
			namespaceName: "eip155",
			chainID:       nil,
			want:          false,
		},
		{
			name: "nil_chainID_invalid_caip2",
			namespace: Namespace{
				Chains: []string{"eip155:1", "invalid"},
			},
			namespaceName: "eip155",
			chainID:       nil,
			want:          false,
		},
		{
			name: "chainID_provided_returns_true",
			namespace: Namespace{
				Chains: []string{"eip155:1"},
			},
			namespaceName: "eip155",
			chainID:       func() *uint64 { v := uint64(1); return &v }(),
			want:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.namespace.Valid(tt.namespaceName, tt.chainID)
			if got != tt.want {
				t.Errorf("Namespace.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

type typedDataParams struct {
	chainID           int
	skipField         bool
	excludeChainID    bool
	wrongContractType bool
}

func generateTypedDataJson(p typedDataParams) string {
	optionalKeyValueField := ""
	if !p.skipField {
		if p.wrongContractType {
			optionalKeyValueField = `,"verifyingContract": true`
		} else {
			optionalKeyValueField = `,"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"`
		}
	}

	chainIDSchemeEntry := ""
	chainIDDataEntry := ""
	if !p.excludeChainID {
		chainIDSchemeEntry = `{"name": "chainId", "type": "uint256"},`
		chainIDDataEntry = `,"chainId": ` + strconv.Itoa(p.chainID)
	}

	typedData := `{
		"types": {
			"EIP712Domain": [
				{"name": "name", "type": "string"},
				{"name": "version", "type": "string"},
				` + chainIDSchemeEntry + `
				{"name": "verifyingContract", "type": "address"}
			],
			"Person": [
				{"name": "name", "type": "string"},
				{"name": "wallet", "type": "address"}
			],
			"Mail": [
				{"name": "from", "type": "Person"},
				{"name": "to", "type": "Person"},
				{"name": "contents", "type": "string"}
			]
		},
		"primaryType": "Mail",
		"domain": {
			"name": "Ether Mail",
			"version": "1"
			` + chainIDDataEntry + `
			` + optionalKeyValueField + `
		},
		"message": {
			"from": {
				"name": "Cow",
				"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
			},
			"to": {
				"name": "Bob",
				"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
			},
			"contents": "Hello, Bob!"
		}
	}`
	return typedData
}

func TestSafeSignTypedDataForDApps(t *testing.T) {
	// 0x4f1B9Ee595bF612480ADAF623Ec583f623ae802d
	privateKey, err := crypto.HexToECDSA("efe79ae971aa8bb612de9de7c65b9224ab1b6a69e6ec733ec92110f100c7244a")
	require.NoError(t, err)
	type args struct {
		typedJson  string
		privateKey *ecdsa.PrivateKey
		chainID    uint64
		legacy     bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "sign_typed_data",
			args: args{
				typedJson: generateTypedDataJson(typedDataParams{
					chainID: 1,
				}),
				privateKey: privateKey,
				chainID:    1,
				legacy:     false,
			},
			wantErr: false,
		},
		{
			name: "sign_typed_data_legacy",
			args: args{
				typedJson: generateTypedDataJson(typedDataParams{
					chainID: 1,
				}),
				privateKey: privateKey,
				chainID:    1,
				legacy:     true,
			},
			wantErr: false,
		},
		{
			name: "sign_typed_data_invalid_json",
			args: args{
				typedJson: generateTypedDataJson(typedDataParams{
					chainID:           1,
					wrongContractType: true,
				}),
				privateKey: privateKey,
				chainID:    1,
				legacy:     false,
			},
			wantErr: true,
		},
		{
			name: "sign_typed_data_invalid_json_legacy",
			args: args{
				typedJson:  `{"invalid": "json"`,
				privateKey: privateKey,
				chainID:    1,
				legacy:     true,
			},
			wantErr: true,
		},
		{
			name: "sign_typed_data_invalid_chain_id",
			args: args{
				typedJson: generateTypedDataJson(typedDataParams{
					chainID: 1,
				}),
				privateKey: privateKey,
				chainID:    2,
				legacy:     false,
			},
			wantErr: true,
		},
		{
			name: "sign_typed_data_missing_field",
			args: args{
				typedJson: generateTypedDataJson(typedDataParams{
					chainID:   1,
					skipField: true,
				}),
				privateKey: privateKey,
				chainID:    1,
				legacy:     false,
			},
			wantErr: true,
		},
		{
			name: "sign_typed_data_exclude_chain_id",
			args: args{
				typedJson: generateTypedDataJson(typedDataParams{
					chainID:        1,
					excludeChainID: true,
				}),
				privateKey: privateKey,
				chainID:    1,
				legacy:     false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeSignTypedDataForDApps(tt.args.typedJson, tt.args.privateKey, tt.args.chainID, tt.args.legacy)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeSignTypedDataForDApps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				require.NotEmpty(t, got)
				require.Len(t, got, 65)
			}
		})
	}
}
