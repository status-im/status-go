package walletconnect

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// sessionKeys holds the cryptographic keys for a session
type sessionKeys struct {
	PrivateKey       []byte
	PublicKey        []byte
	SharedSecret     []byte
	SessionSymKey    []byte
	SessionSymKeyHex string
	PublicKeyHex     string
	SessionTopic     string
}

// performKeyExchange executes X25519 key agreement with the proposer
func (c *Client) performKeyExchange(proposerPubKeyHex string) (*sessionKeys, error) {
	proposerPub, err := hex.DecodeString(proposerPubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode proposer public key: %w", err)
	}
	if len(proposerPub) != 32 {
		return nil, fmt.Errorf("%w: got %d bytes, expected 32", ErrInvalidPublicKey, len(proposerPub))
	}

	privKey, pubKey, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	sharedSecret, err := DeriveSharedSecret(privKey, proposerPub)
	if err != nil {
		return nil, fmt.Errorf("derive shared secret: %w", err)
	}

	sessionSymKey, err := DeriveSymmetricKey(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("derive symmetric key: %w", err)
	}

	return &sessionKeys{
		PrivateKey:       privKey,
		PublicKey:        pubKey,
		SharedSecret:     sharedSecret,
		SessionSymKey:    sessionSymKey,
		SessionSymKeyHex: hex.EncodeToString(sessionSymKey),
		PublicKeyHex:     hex.EncodeToString(pubKey),
		SessionTopic:     DeriveSessionTopic(sessionSymKey),
	}, nil
}

// sendProposalResponse sends the JSON-RPC response to the session proposal
func (c *Client) sendProposalResponse(pc *pairingContext, keys *sessionKeys) error {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      pc.JsonRpcID,
		Result: SessionProposeResponse{
			Relay: RelayProtocol{
				Protocol: "irn",
			},
			ResponderPublicKey: keys.PublicKeyHex,
		},
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	encrypted, err := EncryptType0Envelope(pc.PairingSymKey, responseJSON)
	if err != nil {
		return fmt.Errorf("encrypt response: %w", err)
	}

	return c.relay.Publish(pc.PairingTopic, encrypted, tagSessionProposeResult)
}

// BuildNamespaces constructs the session namespaces from metadata and proposal.
// Exported for use in session updates.
func BuildNamespaces(meta SessionMetadata, proposal *ProposalParams) (map[string]Namespace, []string, []string) {
	// Deduplicate chain IDs
	chainIDsSet := make(map[int64]bool)
	for _, id := range meta.Chains {
		chainIDsSet[id] = true
	}
	chainIDsSet[int64(meta.ChainID)] = true

	// Convert to slice
	chainIDs := make([]int64, 0, len(chainIDsSet))
	for id := range chainIDsSet {
		chainIDs = append(chainIDs, id)
	}

	// Build EIP155 chains and accounts
	eipChains := make([]string, len(chainIDs))
	eipAccounts := make([]string, len(chainIDs))
	for i, cid := range chainIDs {
		eipChains[i] = fmt.Sprintf("eip155:%d", cid)
		eipAccounts[i] = fmt.Sprintf("eip155:%d:%s", cid, meta.Account)
	}

	// Extract methods and events from proposal
	methods := []string{"personal_sign", "eth_signTypedData", "eth_signTypedData_v4", "eth_sendTransaction"}
	events := []string{"accountsChanged", "chainChanged"}

	if reqEip, hasEip := proposal.RequiredNamespaces["eip155"]; hasEip {
		if len(reqEip.Methods) > 0 {
			methods = reqEip.Methods
		}
		if len(reqEip.Events) > 0 {
			events = reqEip.Events
		}
	}

	namespaces := map[string]Namespace{
		"eip155": {
			Accounts: eipAccounts,
			Chains:   eipChains,
			Methods:  methods,
			Events:   events,
		},
	}

	return namespaces, eipChains, eipAccounts
}

// sendSessionSettle sends the wc_sessionSettle message
func (c *Client) sendSessionSettle(keys *sessionKeys, proposal *ProposalParams, namespaces map[string]Namespace, expiry int64) error {
	relayProtocol := "irn"
	if len(proposal.Relays) > 0 && proposal.Relays[0].Protocol != "" {
		relayProtocol = proposal.Relays[0].Protocol
	}

	// Extract chains, methods, events from namespaces
	eip155Ns := namespaces["eip155"]

	// Build required and optional namespaces
	requiredNs := map[string]Namespace{
		"eip155": {
			Chains:  eip155Ns.Chains,
			Methods: eip155Ns.Methods,
			Events:  eip155Ns.Events,
		},
	}

	optionalNs := make(map[string]Namespace)
	if proposal.OptionalNamespaces != nil {
		for k, v := range proposal.OptionalNamespaces {
			optionalNs[k] = v
		}
	}

	controllerMeta := PeerMetadata{
		Name:        "Status",
		Description: "Status Wallet",
		URL:         "https://status.im",
		Icons:       []string{"https://status.im/img/logo.png"},
	}

	settleParams := SessionSettleParams{
		Relay: RelayProtocol{
			Protocol: relayProtocol,
		},
		Controller: SessionController{
			PublicKey: keys.PublicKeyHex,
			Metadata:  controllerMeta,
		},
		Namespaces: map[string]Namespace{
			"eip155": {
				Accounts: eip155Ns.Accounts,
				Chains:   eip155Ns.Chains,
				Methods:  eip155Ns.Methods,
				Events:   eip155Ns.Events,
			},
		},
		RequiredNamespaces: requiredNs,
		OptionalNamespaces: optionalNs,
		Expiry:             expiry,
	}

	settleID := payloadID()
	ch := c.registerPending(settleID)
	settlePayload := JSONRPCRequest{
		ID:      settleID,
		JSONRPC: "2.0",
		Method:  "wc_sessionSettle",
		Params:  settleParams,
	}

	settleJSON, err := json.Marshal(settlePayload)
	if err != nil {
		return fmt.Errorf("marshal settle: %w", err)
	}

	encrypted, err := EncryptType0Envelope(keys.SessionSymKeyHex, settleJSON)
	if err != nil {
		return fmt.Errorf("encrypt settle: %w", err)
	}

	if err := c.relay.Publish(keys.SessionTopic, encrypted, tagSessionSettle); err != nil {
		return err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			c.logger.Warn("dapp rejected settle", zap.Int("code", resp.Error.Code), zap.String("message", resp.Error.Message))
			return fmt.Errorf("settle rejected: %s", resp.Error.Message)
		}
	case <-time.After(15 * time.Second):
		c.cancelPending(settleID)
		c.logger.Debug("settle response timeout (non-fatal)")
	}
	return nil
}

// buildSessionObject creates the final Session object for persistence
func buildSessionObject(pc *pairingContext, keys *sessionKeys, proposal *ProposalParams, meta SessionMetadata, namespaces map[string]Namespace, expiry int64) (*Session, error) {
	relayProtocol := "irn"
	if len(proposal.Relays) > 0 && proposal.Relays[0].Protocol != "" {
		relayProtocol = proposal.Relays[0].Protocol
	}

	controllerMeta := PeerMetadata{
		Name:        "Status",
		Description: "Status Wallet",
		URL:         "https://status.im",
		Icons:       []string{"https://status.im/img/logo.png"},
	}

	peerIcons := proposal.Proposer.Metadata.Icons
	if len(peerIcons) == 0 {
		peerIcons = []string{meta.DAppIcon}
	}

	peerMeta := PeerMetadata{
		Name:        proposal.Proposer.Metadata.Name,
		URL:         proposal.Proposer.Metadata.URL,
		Description: proposal.Proposer.Metadata.Description,
		Icons:       peerIcons,
	}

	// Extract required namespaces
	eip155Ns := namespaces["eip155"]
	requiredNs := map[string]Namespace{
		"eip155": {
			Chains:  eip155Ns.Chains,
			Methods: eip155Ns.Methods,
			Events:  eip155Ns.Events,
		},
	}

	session := &Session{
		Topic:        keys.SessionTopic,
		PairingTopic: pc.PairingTopic,
		Relay: RelayProtocol{
			Protocol: relayProtocol,
		},
		Self: SessionParticipant{
			PublicKey: keys.PublicKeyHex,
			Metadata:  controllerMeta,
		},
		Peer: SessionParticipant{
			PublicKey: pc.ProposerPubKey,
			Metadata:  peerMeta,
		},
		Expiry:             expiry,
		Acknowledged:       false,
		Controller:         keys.PublicKeyHex,
		Namespaces:         namespaces,
		RequiredNamespaces: requiredNs,
	}

	return session, nil
}

// computeExpiry calculates the session expiry time
func computeExpiry(requestedExpiry int64) int64 {
	if requestedExpiry > 0 {
		return requestedExpiry
	}
	return time.Now().Unix() + defaultSessionExpirySecs
}
