package walletconnect

import "errors"

// Sentinel errors for better error handling
var (
	ErrProposalNotFound = errors.New("proposal not found")
	ErrSessionNotFound  = errors.New("session not found")
	ErrInvalidPublicKey = errors.New("invalid public key")
	ErrNotConnected     = errors.New("not connected to relay")
	ErrInvalidProposal  = errors.New("invalid proposal params")
)

// RelayProtocol represents the relay protocol configuration
type RelayProtocol struct {
	Protocol string `json:"protocol"`
}

// SessionController represents the session controller (wallet side)
type SessionController struct {
	PublicKey string       `json:"publicKey"`
	Metadata  PeerMetadata `json:"metadata"`
}

// SessionParticipant represents either self or peer in a session
type SessionParticipant struct {
	PublicKey string       `json:"publicKey"`
	Metadata  PeerMetadata `json:"metadata"`
}

// PeerMetadata represents metadata about a session participant
type PeerMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	URL         string   `json:"url"`
	Icons       []string `json:"icons,omitempty"`
}

// Namespace represents a WalletConnect namespace (e.g., eip155)
type Namespace struct {
	Accounts []string `json:"accounts,omitempty"`
	Chains   []string `json:"chains,omitempty"`
	Methods  []string `json:"methods"`
	Events   []string `json:"events"`
}

// ProposalParams represents the parameters of a session proposal
type ProposalParams struct {
	Relays             []RelayProtocol      `json:"relays"`
	RequiredNamespaces map[string]Namespace `json:"requiredNamespaces"`
	OptionalNamespaces map[string]Namespace `json:"optionalNamespaces,omitempty"`
	Proposer           SessionParticipant   `json:"proposer"`
}

// SessionProposeResponse represents the response to a session proposal
type SessionProposeResponse struct {
	Relay              RelayProtocol `json:"relay"`
	ResponderPublicKey string        `json:"responderPublicKey"`
}

// SessionSettleParams represents the parameters for session settlement
type SessionSettleParams struct {
	Relay              RelayProtocol        `json:"relay"`
	Controller         SessionController    `json:"controller"`
	Namespaces         map[string]Namespace `json:"namespaces"`
	RequiredNamespaces map[string]Namespace `json:"requiredNamespaces"`
	OptionalNamespaces map[string]Namespace `json:"optionalNamespaces,omitempty"`
	Expiry             int64                `json:"expiry"`
}

// Session represents a fully established WalletConnect session
type Session struct {
	Topic              string               `json:"topic"`
	PairingTopic       string               `json:"pairingTopic"`
	Relay              RelayProtocol        `json:"relay"`
	Self               SessionParticipant   `json:"self"`
	Peer               SessionParticipant   `json:"peer"`
	Expiry             int64                `json:"expiry"`
	Acknowledged       bool                 `json:"acknowledged"`
	Controller         string               `json:"controller"`
	Namespaces         map[string]Namespace `json:"namespaces"`
	RequiredNamespaces map[string]Namespace `json:"requiredNamespaces"`
}

// SessionEvent represents a WC session event (e.g., accountsChanged, chainChanged).
type SessionEvent struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
