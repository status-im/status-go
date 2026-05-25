package walletconnect

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// --- BuildNamespaces tests ---

func TestBuildNamespaces_DefaultMethodsAndEvents(t *testing.T) {
	meta := SessionMetadata{
		Account: "0x1234567890abcdef1234567890abcdef12345678",
		ChainID: 1,
		Chains:  []int64{1},
	}
	proposal := &ProposalParams{}

	namespaces, chains, accounts := BuildNamespaces(meta, proposal)

	require.Contains(t, namespaces, "eip155")
	ns := namespaces["eip155"]
	require.Contains(t, ns.Methods, "eth_sendTransaction")
	require.Contains(t, ns.Methods, "personal_sign")
	require.Contains(t, ns.Methods, "eth_signTypedData")
	require.Contains(t, ns.Methods, "eth_signTypedData_v4")
	require.Contains(t, ns.Methods, "wallet_switchEthereumChain")
	require.Contains(t, ns.Events, "accountsChanged")
	require.Contains(t, ns.Events, "chainChanged")
	require.Contains(t, chains, "eip155:1")
	require.Contains(t, accounts, "eip155:1:"+meta.Account)
}

func TestBuildNamespaces_OverwritesWithRequiredMethodsWhenNonEmpty(t *testing.T) {
	meta := SessionMetadata{
		Account: "0xabc",
		ChainID: 1,
		Chains:  []int64{1},
	}
	proposal := &ProposalParams{
		RequiredNamespaces: map[string]Namespace{
			"eip155": {
				Methods: []string{"custom_method_a", "custom_method_b"},
				Events:  []string{"custom_event"},
			},
		},
	}

	namespaces, _, _ := BuildNamespaces(meta, proposal)

	ns := namespaces["eip155"]

	require.ElementsMatch(t, []string{"custom_method_a", "custom_method_b"}, ns.Methods)
	require.ElementsMatch(t, []string{"custom_event"}, ns.Events)
}

func TestBuildNamespaces_EmptyRequiredMethodsKeepsDefaults(t *testing.T) {
	meta := SessionMetadata{
		Account: "0x1234",
		ChainID: 1,
		Chains:  []int64{1},
	}
	proposal := &ProposalParams{
		RequiredNamespaces: map[string]Namespace{
			"eip155": {
				Methods: []string{}, // empty — should keep defaults
				Events:  []string{}, // empty — should keep defaults
			},
		},
	}

	namespaces, _, _ := BuildNamespaces(meta, proposal)

	ns := namespaces["eip155"]
	require.Contains(t, ns.Methods, "eth_sendTransaction")
	require.Contains(t, ns.Events, "accountsChanged")
}

func TestBuildNamespaces_DeduplicatesChainIDs(t *testing.T) {
	meta := SessionMetadata{
		Account: "0x1234",
		ChainID: 1,                // same as first element in Chains
		Chains:  []int64{1, 1, 2}, // 1 appears twice
	}
	proposal := &ProposalParams{}

	namespaces, chains, accounts := BuildNamespaces(meta, proposal)

	ns := namespaces["eip155"]
	require.Len(t, ns.Chains, 2, "should deduplicate chain IDs")
	require.Len(t, chains, 2)
	require.Len(t, accounts, 2)
}

func TestBuildNamespaces_MultipleChains(t *testing.T) {
	meta := SessionMetadata{
		Account: "0xABCdef",
		ChainID: 10,
		Chains:  []int64{1, 137}, // 10 + 1 + 137 = 3 unique
	}
	proposal := &ProposalParams{}

	namespaces, chains, accounts := BuildNamespaces(meta, proposal)

	ns := namespaces["eip155"]
	require.Len(t, ns.Chains, 3)
	require.Len(t, chains, 3)
	require.Len(t, accounts, 3)
	for _, acc := range ns.Accounts {
		require.Contains(t, acc, "0xABCdef")
	}
}

// --- computeExpiry tests ---

func TestComputeExpiry_WithPositiveExpiry(t *testing.T) {
	requested := int64(1000000000)
	result := computeExpiry(requested)
	require.Equal(t, requested, result)
}

func TestComputeExpiry_WithZeroUsesDefault(t *testing.T) {
	before := time.Now().Unix()
	result := computeExpiry(0)
	after := time.Now().Unix()

	require.GreaterOrEqual(t, result, before+defaultSessionExpirySecs)
	require.LessOrEqual(t, result, after+defaultSessionExpirySecs)
}

// --- buildSessionObject tests ---

func makeTestSessionKeys(t *testing.T, proposerPub []byte) *sessionKeys {
	t.Helper()

	priv, pub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	sharedSecret, err := DeriveSharedSecret(priv, proposerPub)
	require.NoError(t, err)

	sessionSymKey, err := DeriveSymmetricKey(sharedSecret)
	require.NoError(t, err)

	return &sessionKeys{
		PrivateKey:       priv,
		PublicKey:        pub,
		SharedSecret:     sharedSecret,
		SessionSymKey:    sessionSymKey,
		SessionSymKeyHex: hex.EncodeToString(sessionSymKey),
		PublicKeyHex:     hex.EncodeToString(pub),
		SessionTopic:     DeriveSessionTopic(sessionSymKey),
	}
}

func TestBuildSessionObject_Basic(t *testing.T) {
	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	keys := makeTestSessionKeys(t, proposerPub)

	pc := &pairingContext{
		PairingTopic:   "pairing-topic-xyz",
		PairingSymKey:  "pairing-sym-key",
		ProposerPubKey: hex.EncodeToString(proposerPub),
		JsonRpcID:      42,
	}

	proposal := &ProposalParams{
		Relays: []RelayProtocol{{Protocol: "irn"}},
		Proposer: SessionParticipant{
			PublicKey: hex.EncodeToString(proposerPub),
			Metadata: PeerMetadata{
				Name:        "Test DApp",
				URL:         "https://testdapp.com",
				Description: "A test dapp",
				Icons:       []string{"https://testdapp.com/icon.png"},
			},
		},
	}

	meta := SessionMetadata{
		Account: "0x1234567890abcdef",
		ChainID: 1,
		Chains:  []int64{1},
	}

	namespaces, _, _ := BuildNamespaces(meta, proposal)
	expiry := int64(9999999999)

	session, err := buildSessionObject(pc, keys, proposal, meta, namespaces, expiry)
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, keys.SessionTopic, session.Topic)
	require.Equal(t, "pairing-topic-xyz", session.PairingTopic)
	require.Equal(t, "irn", session.Relay.Protocol)
	require.Equal(t, keys.PublicKeyHex, session.Self.PublicKey)
	require.Equal(t, "Status", session.Self.Metadata.Name)
	require.Equal(t, "https://status.im", session.Self.Metadata.URL)
	require.Equal(t, hex.EncodeToString(proposerPub), session.Peer.PublicKey)
	require.Equal(t, "Test DApp", session.Peer.Metadata.Name)
	require.Equal(t, "https://testdapp.com/icon.png", session.Peer.Metadata.Icons[0])
	require.Equal(t, expiry, session.Expiry)
	require.False(t, session.Acknowledged)
	require.Equal(t, keys.PublicKeyHex, session.Controller)
}

func TestBuildSessionObject_UsesDAppIconFallback(t *testing.T) {
	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	keys := makeTestSessionKeys(t, proposerPub)

	pc := &pairingContext{
		PairingTopic:   "pairing-topic",
		ProposerPubKey: hex.EncodeToString(proposerPub),
	}

	proposal := &ProposalParams{
		Proposer: SessionParticipant{
			Metadata: PeerMetadata{
				Icons: []string{}, // no icons from proposer
			},
		},
	}

	meta := SessionMetadata{
		Account:  "0x1234",
		ChainID:  1,
		Chains:   []int64{1},
		DAppIcon: "https://fallback-icon.com/icon.png",
	}

	namespaces, _, _ := BuildNamespaces(meta, proposal)
	session, err := buildSessionObject(pc, keys, proposal, meta, namespaces, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"https://fallback-icon.com/icon.png"}, session.Peer.Metadata.Icons)
}

func TestBuildSessionObject_DefaultRelayProtocol(t *testing.T) {
	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	keys := makeTestSessionKeys(t, proposerPub)

	pc := &pairingContext{
		PairingTopic:   "topic",
		ProposerPubKey: hex.EncodeToString(proposerPub),
	}

	// No relays in proposal — should default to "irn"
	proposal := &ProposalParams{}

	meta := SessionMetadata{ChainID: 1, Chains: []int64{1}}
	namespaces, _, _ := BuildNamespaces(meta, proposal)

	session, err := buildSessionObject(pc, keys, proposal, meta, namespaces, 0)
	require.NoError(t, err)
	require.Equal(t, "irn", session.Relay.Protocol)
}

func TestBuildSessionObject_ProposalRelayProtocolUsed(t *testing.T) {
	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	keys := makeTestSessionKeys(t, proposerPub)

	pc := &pairingContext{
		PairingTopic:   "topic",
		ProposerPubKey: hex.EncodeToString(proposerPub),
	}

	proposal := &ProposalParams{
		Relays: []RelayProtocol{{Protocol: "custom-relay"}},
	}

	meta := SessionMetadata{ChainID: 1, Chains: []int64{1}}
	namespaces, _, _ := BuildNamespaces(meta, proposal)

	session, err := buildSessionObject(pc, keys, proposal, meta, namespaces, 0)
	require.NoError(t, err)
	require.Equal(t, "custom-relay", session.Relay.Protocol)
}

func TestBuildSessionObject_RequiredNamespacesExcludesAccounts(t *testing.T) {
	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	keys := makeTestSessionKeys(t, proposerPub)

	pc := &pairingContext{
		PairingTopic:   "topic",
		ProposerPubKey: hex.EncodeToString(proposerPub),
	}

	proposal := &ProposalParams{}
	meta := SessionMetadata{Account: "0xabc", ChainID: 1, Chains: []int64{1}}
	namespaces, _, _ := BuildNamespaces(meta, proposal)

	session, err := buildSessionObject(pc, keys, proposal, meta, namespaces, 0)
	require.NoError(t, err)

	// RequiredNamespaces should have chains, methods, events but NOT accounts
	reqNs := session.RequiredNamespaces["eip155"]
	require.Empty(t, reqNs.Accounts)
	require.NotEmpty(t, reqNs.Chains)
	require.NotEmpty(t, reqNs.Methods)

	// Namespaces (full) should have accounts
	fullNs := session.Namespaces["eip155"]
	require.NotEmpty(t, fullNs.Accounts)
}

// --- performKeyExchange tests ---

func TestPerformKeyExchange_InvalidHex(t *testing.T) {
	client, err := NewClient("test")
	require.NoError(t, err)

	_, err = client.performKeyExchange("not-valid-hex!!!")
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode proposer public key")
}

func TestPerformKeyExchange_WrongLength(t *testing.T) {
	client, err := NewClient("test")
	require.NoError(t, err)

	// 16 bytes = valid hex but wrong key length for X25519
	shortKeyHex := hex.EncodeToString(make([]byte, 16))
	_, err = client.performKeyExchange(shortKeyHex)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidPublicKey)
}

func TestPerformKeyExchange_Valid(t *testing.T) {
	client, err := NewClient("test")
	require.NoError(t, err)

	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	keys, err := client.performKeyExchange(hex.EncodeToString(proposerPub))
	require.NoError(t, err)
	require.NotNil(t, keys)
	require.Len(t, keys.PrivateKey, 32)
	require.Len(t, keys.PublicKey, 32)
	require.Len(t, keys.SharedSecret, 32)
	require.Len(t, keys.SessionSymKey, 32)
	require.NotEmpty(t, keys.SessionSymKeyHex)
	require.NotEmpty(t, keys.PublicKeyHex)
	require.NotEmpty(t, keys.SessionTopic)
}

func TestPerformKeyExchange_DifferentCallsProduceDifferentKeys(t *testing.T) {
	client, err := NewClient("test")
	require.NoError(t, err)

	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	proposerPubHex := hex.EncodeToString(proposerPub)

	keys1, err := client.performKeyExchange(proposerPubHex)
	require.NoError(t, err)

	keys2, err := client.performKeyExchange(proposerPubHex)
	require.NoError(t, err)

	// Different ephemeral keys should produce different session topics
	require.NotEqual(t, keys1.SessionTopic, keys2.SessionTopic)
}
