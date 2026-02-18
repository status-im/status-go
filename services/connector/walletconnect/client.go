package walletconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/signal"
)

// WalletConnect protocol message tags (from spec)
const (
	tagSessionPropose        = 1100
	tagSessionProposeResult  = 1101
	tagSessionSettle         = 1102
	tagSessionRequest        = 1108
	tagSessionProposeReject  = 1120
	defaultSessionExpirySecs = 7 * 24 * 3600 // 7 days
)

// pairingContext holds the context needed to respond to a session proposal.
// Stored when wc_sessionPropose is received, consumed by ApproveSession/RejectSession.
type pairingContext struct {
	PairingTopic   string          // topic from URI (pairing topic)
	PairingSymKey  string          // hex, symmetric key from URI
	ProposerPubKey string          // hex, proposer's X25519 public key from proposal params
	JsonRpcID      int64           // payload.ID — required for JSON-RPC response
	ProposalParams json.RawMessage // original params from the proposal
}

// Client handles WalletConnect protocol operations via the relay.
type Client struct {
	relay            *RelayClient
	logger           *zap.Logger
	mu               sync.Mutex
	handlers         *clientHandlers
	pendingProposals map[string]*pairingContext // key = fmt.Sprintf("%d", jsonRpcId)
	pairingTopics    map[string]string          // topic -> pairing symKey (hex), set on Pair()
	activeSessions   map[string]string          // topic -> session symKey (hex), set on ApproveSession
}

type clientHandlers struct {
	onSessionProposal func(proposalJSON string)
	onSessionRequest  func(topic string, requestJSON string)
}

// NewClient creates a new WalletConnect client.
func NewClient(projectID string) (*Client, error) {
	relay, err := NewRelayClient(projectID)
	if err != nil {
		return nil, fmt.Errorf("create relay client: %w", err)
	}

	return &Client{
		relay:            relay,
		logger:           logutils.ZapLogger(),
		handlers:         &clientHandlers{},
		pendingProposals: make(map[string]*pairingContext),
		pairingTopics:    make(map[string]string),
		activeSessions:   make(map[string]string),
	}, nil
}

// Pair initiates pairing with a WalletConnect URI.
// It subscribes to the pairing topic and fetches any pending session proposals.
// FIXED: Set message handler BEFORE subscribing to prevent race condition
func (c *Client) Pair(ctx context.Context, uri string) error {
	parsed, err := ParseURI(uri)
	if err != nil {
		return fmt.Errorf("parse URI: %w", err)
	}

	if err := c.relay.Connect(); err != nil {
		return fmt.Errorf("connect relay: %w", err)
	}

	c.mu.Lock()
	c.pairingTopics[parsed.Topic] = parsed.SymKey
	c.mu.Unlock()

	// Set handler BEFORE subscribing to avoid race condition
	c.relay.SetMessageHandler(func(topic, message string, tag int) {
		c.handleRelayMessage(topic, message, tag)
	})

	_, err = c.relay.Subscribe(parsed.Topic)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	messages, hasMore, err := c.relay.FetchMessages(parsed.Topic)
	if err != nil {
		c.logger.Debug("fetch messages failed", zap.Error(err))
	}
	for hasMore {
		var more []RelayMessage
		more, hasMore, err = c.relay.FetchMessages(parsed.Topic)
		if err != nil {
			break
		}
		messages = append(messages, more...)
	}

	for _, m := range messages {
		c.handleRelayMessage(m.Topic, m.Message, m.Tag)
	}

	return nil
}

// getSymKeyForTopic returns the symmetric key for the given topic (session or pairing).
func (c *Client) getSymKeyForTopic(topic string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if symKey, ok := c.activeSessions[topic]; ok {
		return symKey, true
	}
	if symKey, ok := c.pairingTopics[topic]; ok {
		return symKey, true
	}
	return "", false
}

// handleRelayMessage processes a message from the relay.
// The message is a Type 0 envelope encrypted with ChaCha20-Poly1305 using symKey.
func (c *Client) handleRelayMessage(topic, message string, tag int) {
	symKey, ok := c.getSymKeyForTopic(topic)
	if !ok {
		c.logger.Debug("no symKey for topic, skipping", zap.String("topic", topic))
		return
	}

	plaintext, err := DecryptType0Envelope(symKey, message)
	if err != nil {
		c.logger.Debug("failed to decrypt relay message", zap.Error(err), zap.String("topic", topic))
		return
	}

	var payload struct {
		ID     int64           `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		c.logger.Debug("failed to parse relay message", zap.Error(err), zap.String("plaintext", string(plaintext)))
		return
	}

	switch payload.Method {
	case "wc_sessionPropose":
		requestID := fmt.Sprintf("%d", payload.ID)
		paramsStr := string(payload.Params)

		var proposalParams struct {
			Proposer struct {
				PublicKey string `json:"publicKey"`
			} `json:"proposer"`
		}
		if err := json.Unmarshal(payload.Params, &proposalParams); err != nil {
			c.logger.Debug("failed to parse proposal params", zap.Error(err))
			return
		}
		if proposalParams.Proposer.PublicKey == "" {
			c.logger.Debug("proposal params missing proposer.publicKey")
			return
		}

		ctx := &pairingContext{
			PairingTopic:   topic,
			PairingSymKey:  symKey,
			ProposerPubKey: proposalParams.Proposer.PublicKey,
			JsonRpcID:      payload.ID,
			ProposalParams: payload.Params,
		}
		c.mu.Lock()
		c.pendingProposals[requestID] = ctx
		handler := c.handlers.onSessionProposal
		c.mu.Unlock()

		if handler != nil {
			handler(paramsStr)
		} else {
			signal.SendWCSessionProposal(requestID, "", paramsStr)
		}
	case "wc_sessionRequest":
		c.mu.Lock()
		handler := c.handlers.onSessionRequest
		c.mu.Unlock()
		if handler != nil {
			handler(topic, string(payload.Params))
		} else {
			signal.SendWCSessionRequest(topic, payload.ID, string(payload.Params))
		}
	}
}

// SetSessionProposalHandler sets the callback for session proposals.
func (c *Client) SetSessionProposalHandler(handler func(proposalJSON string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers.onSessionProposal = handler
}

// SetSessionRequestHandler sets the callback for session requests.
func (c *Client) SetSessionRequestHandler(handler func(topic, requestJSON string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers.onSessionRequest = handler
}

// Publish publishes a message to a topic.
func (c *Client) Publish(topic, message string, tag int) error {
	return c.relay.Publish(topic, message, tag)
}

// Close closes the relay connection.
func (c *Client) Close() error {
	return c.relay.Close()
}

// SessionMetadata holds the user's approval data for a session.
type SessionMetadata struct {
	Account   string  // hex address
	ChainID   uint64  // primary chain
	Chains    []int64 // allowed chains (eip155 IDs)
	DAppURL   string
	DAppName  string
	DAppIcon  string
	ExpirySec int64 // 0 = use default
}

// SessionResult is returned by ApproveSession with the established session details.
type SessionResult struct {
	Topic        string
	PairingTopic string
	SessionJSON  string
	Expiry       int64
}

// ApproveSession completes the WalletConnect session approval flow using typed structures:
// 1. Key agreement (X25519) with proposer's public key
// 2. Send JSON-RPC response on pairing topic (tag 1101)
// 3. Subscribe to session topic
// 4. Send wc_sessionSettle on session topic (tag 1102)
// 5. Return session details for persistence
func (c *Client) ApproveSession(ctx context.Context, proposalID string, meta SessionMetadata) (*SessionResult, error) {
	// Step 1: Retrieve pending proposal
	c.mu.Lock()
	pc, ok := c.pendingProposals[proposalID]
	c.mu.Unlock()
	if !ok || pc == nil {
		return nil, fmt.Errorf("%w: %s", ErrProposalNotFound, proposalID)
	}

	// Step 2: Parse proposal parameters
	var proposal ProposalParams
	if err := json.Unmarshal(pc.ProposalParams, &proposal); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProposal, err)
	}

	// Step 3: Perform key exchange
	keys, err := c.performKeyExchange(pc.ProposerPubKey)
	if err != nil {
		return nil, err
	}

	// Step 4: Send proposal response
	if err := c.sendProposalResponse(pc, keys); err != nil {
		return nil, fmt.Errorf("send proposal response: %w", err)
	}

	// Step 5: Subscribe to session topic
	if _, err := c.relay.Subscribe(keys.SessionTopic); err != nil {
		return nil, fmt.Errorf("subscribe to session topic: %w", err)
	}

	// Step 6: Build namespaces
	expiry := computeExpiry(meta.ExpirySec)
	namespaces, _, _ := buildNamespaces(meta, &proposal)

	// Step 7: Send session settle
	if err := c.sendSessionSettle(keys, &proposal, namespaces, expiry); err != nil {
		return nil, fmt.Errorf("send settle: %w", err)
	}

	// Step 8: Update state
	c.mu.Lock()
	c.activeSessions[keys.SessionTopic] = keys.SessionSymKeyHex
	delete(c.pendingProposals, proposalID)
	c.mu.Unlock()

	// Step 9: Build session object for persistence
	session, err := buildSessionObject(pc, keys, &proposal, meta, namespaces, expiry)
	if err != nil {
		return nil, fmt.Errorf("build session object: %w", err)
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("marshal session: %w", err)
	}

	return &SessionResult{
		Topic:        keys.SessionTopic,
		PairingTopic: pc.PairingTopic,
		SessionJSON:  string(sessionJSON),
		Expiry:       expiry,
	}, nil
}

// RemoveSession removes a session from activeSessions (call on disconnect).
func (c *Client) RemoveSession(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.activeSessions, topic)
}

// RespondToWCSessionRequest sends a JSON-RPC success response for a wc_sessionRequest.
func (c *Client) RespondToWCSessionRequest(topic string, requestID int64, result any) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, topic)
	}

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Result:  result,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	encrypted, err := EncryptType0Envelope(symKey, responseJSON)
	if err != nil {
		return fmt.Errorf("encrypt response: %w", err)
	}

	return c.relay.Publish(topic, encrypted, tagSessionRequest)
}

// RejectWCSessionRequest sends a JSON-RPC error response for a wc_sessionRequest.
func (c *Client) RejectWCSessionRequest(topic string, requestID int64, code int, message string) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, topic)
	}

	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      requestID,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal error response: %w", err)
	}

	encrypted, err := EncryptType0Envelope(symKey, responseJSON)
	if err != nil {
		return fmt.Errorf("encrypt error response: %w", err)
	}

	return c.relay.Publish(topic, encrypted, tagSessionRequest)
}

// RejectSession sends a JSON-RPC error response to the relay when the user rejects a session proposal.
func (c *Client) RejectSession(proposalID string) error {
	c.mu.Lock()
	pc, ok := c.pendingProposals[proposalID]
	c.mu.Unlock()
	if !ok || pc == nil {
		return fmt.Errorf("%w: %s", ErrProposalNotFound, proposalID)
	}

	errorResponse := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      pc.JsonRpcID,
		Error: &JSONRPCError{
			Code:    5000,
			Message: "User rejected",
		},
	}

	responseJSON, err := json.Marshal(errorResponse)
	if err != nil {
		return fmt.Errorf("marshal error response: %w", err)
	}

	encrypted, err := EncryptType0Envelope(pc.PairingSymKey, responseJSON)
	if err != nil {
		return fmt.Errorf("encrypt error response: %w", err)
	}

	if err := c.relay.Publish(pc.PairingTopic, encrypted, tagSessionProposeReject); err != nil {
		return fmt.Errorf("publish reject: %w", err)
	}

	c.mu.Lock()
	delete(c.pendingProposals, proposalID)
	c.mu.Unlock()
	return nil
}
