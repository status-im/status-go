package walletconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/signal"
)

// WalletConnect protocol message tags (from spec)
const (
	tagSessionPropose         = 1100
	tagSessionProposeResult   = 1101
	tagSessionSettle          = 1102
	tagSessionSettleResponse  = 1103
	tagSessionUpdate          = 1104
	tagSessionUpdateResponse  = 1105
	tagSessionRequest         = 1108
	tagSessionRequestResponse = 1109
	tagSessionEvent           = 1110
	tagSessionEventResponse   = 1111
	tagSessionDelete          = 1112
	tagSessionDeleteResponse  = 1113
	tagSessionPing            = 1114
	tagSessionPingResponse    = 1115
	tagSessionProposeReject   = 1120
	defaultSessionExpirySecs  = 7 * 24 * 3600 // 7 days
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
	relay                  Relay
	logger                 *zap.Logger
	mu                     sync.Mutex
	handlers               *clientHandlers
	pendingProposals       map[string]*pairingContext // key = fmt.Sprintf("%d", jsonRpcId)
	pendingRequests        map[int64]chan *JSONRPCResponse
	pendingSessionRequests map[int64]bool    // tracks in-flight wc_sessionRequests for deduplication
	pairingTopics          map[string]string // topic -> pairing symKey (hex), set on Pair()
	activeSessions         map[string]string // topic -> session symKey (hex), set on ApproveSession
}

type clientHandlers struct {
	onSessionProposal func(proposalJSON string)
	onSessionRequest  func(topic string, requestJSON string)
	onSessionUpdate   func(topic string, namespacesJSON string)
	onSessionDelete   func(topic string)
}

// NewClient creates a new WalletConnect client.
func NewClient(projectID string) (*Client, error) {
	relay, err := NewRelayClient(projectID)
	if err != nil {
		return nil, fmt.Errorf("create relay client: %w", err)
	}

	c := &Client{
		relay:                  relay,
		logger:                 logutils.ZapLogger(),
		handlers:               &clientHandlers{},
		pendingProposals:       make(map[string]*pairingContext),
		pendingRequests:        make(map[int64]chan *JSONRPCResponse),
		pendingSessionRequests: make(map[int64]bool),
		pairingTopics:          make(map[string]string),
		activeSessions:         make(map[string]string),
	}

	// Set reconnection handler to re-subscribe to all active sessions
	relay.SetReconnectedHandler(func() {
		c.onReconnected()
	})

	return c, nil
}

// Pair initiates pairing with a WalletConnect URI.
// It subscribes to the pairing topic and fetches any pending session proposals.
// SetMessageHandler is called before Connect() to avoid race: readLoop starts in Connect()
// and would drop messages if handler were set only after Connect returns.
func (c *Client) Pair(ctx context.Context, uri string) error {
	parsed, err := ParseURI(uri)
	if err != nil {
		return fmt.Errorf("parse URI: %w", err)
	}

	c.mu.Lock()
	c.pairingTopics[parsed.Topic] = parsed.SymKey
	c.mu.Unlock()

	c.relay.SetMessageHandler(func(topic, message string, tag int) {
		c.handleRelayMessage(topic, message, tag)
	})

	if err := c.relay.Connect(); err != nil {
		return fmt.Errorf("connect relay: %w", err)
	}

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

// registerPending registers a pending outgoing request ID and returns a channel to receive the response.
func (c *Client) registerPending(id int64) <-chan *JSONRPCResponse {
	ch := make(chan *JSONRPCResponse, 1)
	c.mu.Lock()
	c.pendingRequests[id] = ch
	c.mu.Unlock()
	return ch
}

// resolvePending delivers a response to a pending request and returns true if it was matched.
func (c *Client) resolvePending(id int64, resp *JSONRPCResponse) bool {
	c.mu.Lock()
	ch, ok := c.pendingRequests[id]
	if ok {
		delete(c.pendingRequests, id)
	}
	c.mu.Unlock()
	if ok {
		select {
		case ch <- resp:
		default:
		}
	}
	return ok
}

// cancelPending removes a pending request entry without delivering a response,
// preventing the pendingRequests map from leaking entries on timeout.
func (c *Client) cancelPending(id int64) {
	c.mu.Lock()
	delete(c.pendingRequests, id)
	c.mu.Unlock()
}

// sendAckResponse sends a JSON-RPC success response with result: true.
func (c *Client) sendAckResponse(topic, symKey string, id int64, tag int) error {
	response := JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: true}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return err
	}
	encrypted, err := EncryptType0Envelope(symKey, responseJSON)
	if err != nil {
		return err
	}
	return c.relay.Publish(topic, encrypted, tag)
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
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		c.logger.Debug("failed to parse relay message", zap.Error(err), zap.String("plaintext", string(plaintext)))
		return
	}

	// Helper to extract int64 ID from either string or number format
	parseID := func(rawID json.RawMessage) (int64, error) {
		// Try as int64 first
		var numID int64
		if err := json.Unmarshal(rawID, &numID); err == nil {
			return numID, nil
		}
		// Try as string
		var strID string
		if err := json.Unmarshal(rawID, &strID); err == nil {
			var id int64
			if _, err := fmt.Sscanf(strID, "%d", &id); err == nil {
				return id, nil
			}
		}
		return 0, fmt.Errorf("invalid id format")
	}

	msgID, err := parseID(payload.ID)
	if err != nil {
		c.logger.Debug("ignoring message with invalid ID format", zap.Error(err))
		return
	}

	// JSON-RPC responses have no method field; route to pending outgoing requests
	if payload.Method == "" {
		var resp struct {
			Result any           `json:"result,omitempty"`
			Error  *JSONRPCError `json:"error,omitempty"`
		}
		if err := json.Unmarshal(plaintext, &resp); err != nil {
			c.logger.Debug("failed to parse response payload", zap.Error(err))
			return
		}
		jsonRpcResp := &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      msgID,
			Result:  resp.Result,
			Error:   resp.Error,
		}
		if c.resolvePending(msgID, jsonRpcResp) {
			return
		}
		c.logger.Debug("response for unknown request ID", zap.Int64("id", msgID))
		return
	}

	switch payload.Method {
	case "wc_sessionPropose":
		requestID := fmt.Sprintf("%d", msgID)
		paramsStr := string(payload.Params)

		c.mu.Lock()
		_, alreadyPending := c.pendingProposals[requestID]
		c.mu.Unlock()

		if alreadyPending {
			c.logger.Info("wc_sessionPropose duplicate ignored", zap.String("topic", topic), zap.String("requestID", requestID))
			return
		}

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
			JsonRpcID:      msgID,
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
		_, alreadyPendingReq := c.pendingSessionRequests[msgID]
		if !alreadyPendingReq {
			c.pendingSessionRequests[msgID] = true
		}
		handler := c.handlers.onSessionRequest
		c.mu.Unlock()

		// Deduplicate: relay may deliver the same request both via FetchMessages
		// and via an irn_subscription push
		if alreadyPendingReq {
			c.logger.Info("wc_sessionRequest duplicate ignored", zap.String("topic", topic), zap.Int64("msgID", msgID))
			return
		}

		if handler != nil {
			handler(topic, string(payload.Params))
		} else {
			signal.SendWCSessionRequest(topic, msgID, string(payload.Params))
		}
	case "wc_sessionPing":
		if err := c.sendAckResponse(topic, symKey, msgID, tagSessionPingResponse); err != nil {
			c.logger.Debug("failed to send ping ack", zap.Error(err))
		}
	case "wc_sessionUpdate":
		if err := c.sendAckResponse(topic, symKey, msgID, tagSessionUpdateResponse); err != nil {
			c.logger.Debug("failed to send session update ack", zap.Error(err))
		}
		c.mu.Lock()
		updateHandler := c.handlers.onSessionUpdate
		c.mu.Unlock()
		if updateHandler != nil {
			updateHandler(topic, string(payload.Params))
		}
	case "wc_sessionDelete":
		// Remove session atomically; if it was already gone this is a duplicate delivery.
		c.mu.Lock()
		_, sessionExisted := c.activeSessions[topic]
		delete(c.activeSessions, topic)
		deleteHandler := c.handlers.onSessionDelete
		c.mu.Unlock()
		if !sessionExisted {
			c.logger.Info("wc_sessionDelete duplicate ignored", zap.String("topic", topic))
			return
		}
		if deleteHandler != nil {
			deleteHandler(topic)
		}
		if err := c.sendAckResponse(topic, symKey, msgID, tagSessionDeleteResponse); err != nil {
			c.logger.Debug("failed to send session delete ack", zap.Error(err))
		}
	default:
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

// SetSessionUpdateHandler sets the callback for incoming session update requests.
func (c *Client) SetSessionUpdateHandler(handler func(topic, namespacesJSON string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers.onSessionUpdate = handler
}

// SetSessionDeleteHandler sets the callback for session delete events.
func (c *Client) SetSessionDeleteHandler(handler func(topic string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers.onSessionDelete = handler
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
	SymKey       string
}

// ApproveSession completes the WalletConnect session approval flow using typed structures:
// 1. Key agreement (X25519) with proposer's public key
// 2. Send JSON-RPC response on pairing topic (tag 1101)
// 3. Subscribe to session topic
// 4. Send wc_sessionSettle on session topic (tag 1102)
// 5. Return session details for persistence
func (c *Client) ApproveSession(ctx context.Context, proposalID string, meta SessionMetadata) (*SessionResult, error) {
	c.mu.Lock()
	pc, ok := c.pendingProposals[proposalID]
	c.mu.Unlock()
	if !ok || pc == nil {
		return nil, fmt.Errorf("%w: %s", ErrProposalNotFound, proposalID)
	}

	var proposal ProposalParams
	if err := json.Unmarshal(pc.ProposalParams, &proposal); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProposal, err)
	}

	keys, err := c.performKeyExchange(pc.ProposerPubKey)
	if err != nil {
		return nil, err
	}

	if err := c.sendProposalResponse(pc, keys); err != nil {
		return nil, fmt.Errorf("send proposal response: %w", err)
	}

	if _, err := c.relay.Subscribe(keys.SessionTopic); err != nil {
		return nil, fmt.Errorf("subscribe to session topic: %w", err)
	}

	// Step 6: Build namespaces
	expiry := computeExpiry(meta.ExpirySec)
	namespaces, _, _ := BuildNamespaces(meta, &proposal)

	// Step 7: Update state BEFORE sending settle so we can decrypt incoming messages
	// (dApp may respond immediately with ack or session request)
	c.mu.Lock()
	c.activeSessions[keys.SessionTopic] = keys.SessionSymKeyHex
	delete(c.pendingProposals, proposalID)
	c.mu.Unlock()

	if err := c.sendSessionSettle(keys, &proposal, namespaces, expiry); err != nil {
		return nil, fmt.Errorf("send settle: %w", err)
	}

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
		SymKey:       keys.SessionSymKeyHex,
	}, nil
}

// RemoveSession removes a session from activeSessions (call on disconnect).
func (c *Client) RemoveSession(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.activeSessions, topic)
}

// SendSessionDelete sends a wc_sessionDelete request to the dApp and removes the session from activeSessions
func (c *Client) SendSessionDelete(ctx context.Context, topic string) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	c.mu.Unlock()
	if !ok {
		// Session already gone locally — nothing to send.
		return nil
	}

	deleteID := payloadID()
	ch := c.registerPending(deleteID)

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      deleteID,
		Method:  "wc_sessionDelete",
		Params: map[string]interface{}{
			"code":    6000,
			"message": "User disconnected",
		},
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal session delete request: %w", err)
	}

	encrypted, err := EncryptType0Envelope(symKey, requestJSON)
	if err != nil {
		return fmt.Errorf("encrypt session delete request: %w", err)
	}

	if err := c.relay.Publish(topic, encrypted, tagSessionDelete); err != nil {
		// Still remove locally even if relay send failed.
		c.RemoveSession(topic)
		return err
	}

	// Wait briefly for the dApp's ack. Non-fatal if it times out — the dApp will
	// discover the session is gone on its next interaction.
	ackCtx := ctx
	if ackCtx == nil {
		ackCtx = context.Background()
	}
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		c.cancelPending(deleteID)
	case <-ackCtx.Done():
		c.cancelPending(deleteID)
	}

	c.RemoveSession(topic)
	return nil
}

// RespondToWCSessionRequest sends a JSON-RPC success response for a wc_sessionRequest.
func (c *Client) RespondToWCSessionRequest(topic string, requestID int64, result any) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	delete(c.pendingSessionRequests, requestID)
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
	return c.relay.Publish(topic, encrypted, tagSessionRequestResponse)
}

// RejectWCSessionRequest sends a JSON-RPC error response for a wc_sessionRequest.
func (c *Client) RejectWCSessionRequest(topic string, requestID int64, code int, message string) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	delete(c.pendingSessionRequests, requestID)
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
	return c.relay.Publish(topic, encrypted, tagSessionRequestResponse)
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

// SendSessionUpdate sends a wc_sessionUpdate request to update session namespaces (e.g., add/remove chains).
func (c *Client) SendSessionUpdate(topic string, namespaces map[string]Namespace) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, topic)
	}

	updateID := payloadID()
	ch := c.registerPending(updateID)
	updateParams := map[string]interface{}{
		"namespaces": namespaces,
	}

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      updateID,
		Method:  "wc_sessionUpdate",
		Params:  updateParams,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal update request: %w", err)
	}

	encrypted, err := EncryptType0Envelope(symKey, requestJSON)
	if err != nil {
		return fmt.Errorf("encrypt update request: %w", err)
	}

	if err := c.relay.Publish(topic, encrypted, tagSessionUpdate); err != nil {
		return err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return fmt.Errorf("update rejected: %s", resp.Error.Message)
		}
	case <-time.After(15 * time.Second):
		c.cancelPending(updateID)
		c.logger.Debug("session update response timeout (non-fatal)")
	}
	return nil
}

// SendSessionEvent sends a wc_sessionEvent to the dapp (e.g., accountsChanged, chainChanged).
func (c *Client) SendSessionEvent(topic string, event SessionEvent, chainID string) error {
	c.mu.Lock()
	symKey, ok := c.activeSessions[topic]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, topic)
	}

	eventID := payloadID()
	ch := c.registerPending(eventID)
	params := map[string]interface{}{
		"event":   event,
		"chainId": chainID,
	}
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      eventID,
		Method:  "wc_sessionEvent",
		Params:  params,
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal session event: %w", err)
	}
	encrypted, err := EncryptType0Envelope(symKey, requestJSON)
	if err != nil {
		return fmt.Errorf("encrypt session event: %w", err)
	}
	if err := c.relay.Publish(topic, encrypted, tagSessionEvent); err != nil {
		return err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			c.logger.Debug("dapp rejected session event", zap.String("event", event.Name), zap.Any("error", resp.Error))
		}
	case <-time.After(15 * time.Second):
		c.cancelPending(eventID)
		c.logger.Debug("session event response timeout (non-fatal)")
	}
	return nil
}

// onReconnected re-subscribes to all active session and pairing topics.
func (c *Client) onReconnected() {
	c.mu.Lock()
	sessionTopics := make([]string, 0, len(c.activeSessions))
	for topic := range c.activeSessions {
		sessionTopics = append(sessionTopics, topic)
	}
	pairingTopics := make([]string, 0, len(c.pairingTopics))
	for topic := range c.pairingTopics {
		pairingTopics = append(pairingTopics, topic)
	}
	c.mu.Unlock()
	c.resubscribeTopics("session", sessionTopics)
	c.resubscribeTopics("pairing", pairingTopics)
}

func (c *Client) resubscribeTopics(label string, topics []string) {
	for _, topic := range topics {
		if _, err := c.relay.Subscribe(topic); err != nil {
			c.logger.Error("failed to re-subscribe", zap.String("type", label), zap.String("topic", topic), zap.Error(err))
		} else {
			c.logger.Info("re-subscribed", zap.String("type", label), zap.String("topic", topic))
		}
	}
}

// RestoredSession holds topic and symKey for session restoration from DB.
type RestoredSession struct {
	Topic  string
	SymKey string
}

// ConnectAndResubscribe connects the relay WebSocket and subscribes to all
// session topics. Must be called after RestoreSessions
func (c *Client) ConnectAndResubscribe() error {
	c.relay.SetMessageHandler(func(topic, message string, tag int) {
		c.handleRelayMessage(topic, message, tag)
	})

	if err := c.relay.Connect(); err != nil {
		return fmt.Errorf("connect relay: %w", err)
	}

	c.mu.Lock()
	topics := make([]string, 0, len(c.activeSessions))
	for topic := range c.activeSessions {
		topics = append(topics, topic)
	}
	c.mu.Unlock()

	c.resubscribeTopics("session", topics)
	return nil
}

// RestoreSessions populates activeSessions from database. Call on startup after a restart.
func (c *Client) RestoreSessions(sessions []RestoredSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, s := range sessions {
		if s.Topic != "" && s.SymKey != "" {
			c.activeSessions[s.Topic] = s.SymKey
			c.logger.Info("restored WalletConnect session", zap.String("topic", s.Topic))
		}
	}
}
