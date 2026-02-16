package walletconnect

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/websocket"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
)

const (
	relayURL          = "wss://relay.walletconnect.com"
	defaultTTL        = 86400 // 24 hours
	defaultMessageTag = 1000
)

// truncate safely truncates a string to a maximum length, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// payloadID generates a WalletConnect-compliant JSON-RPC request ID with 19 digits of entropy.
// It combines the current Unix millisecond timestamp with 6 digits of random entropy,
// matching the WalletConnect specification for relay communication.
func payloadID() int64 {
	return time.Now().UnixMilli()*1_000_000 + rand.Int64N(1_000_000)
}

type (
	jsonRPCRequest struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"` // int64 for outgoing requests (marshals as JSON number)
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}

	jsonRPCResponse struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"` // accepts both string and numeric IDs
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *jsonRPCError   `json:"error,omitempty"`
	}

	jsonRPCError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	irnSubscriptionParams struct {
		ID   string `json:"id"`
		Data struct {
			Topic       string `json:"topic"`
			Message     string `json:"message"`
			PublishedAt int64  `json:"publishedAt"`
			Tag         int    `json:"tag"`
		} `json:"data"`
	}

	jsonRPCNotification struct {
		JSONRPC string                `json:"jsonrpc"`
		Method  string                `json:"method"`
		Params  irnSubscriptionParams `json:"params"`
	}
)

// idString returns the ID as a string, stripping JSON quotes if present.
func (r *jsonRPCResponse) idString() string {
	s := string(r.ID)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return s
}

// MessageHandler is called when a subscription message is received.
type MessageHandler func(topic, message string, tag int)

// RelayClient implements the WalletConnect IRN relay protocol over WebSocket.
type RelayClient struct {
	url            string
	conn           *websocket.Conn
	mu             sync.Mutex
	pending        map[string]chan *jsonRPCResponse
	messageHandler MessageHandler
	logger         *zap.Logger
	projectID      string
	auth           *Auth
	done           chan struct{}  // signals shutdown
	wg             sync.WaitGroup // tracks active goroutines
}

// NewRelayClient creates a new relay client.
func NewRelayClient(projectID string) (*RelayClient, error) {
	auth, err := NewAuth()
	if err != nil {
		return nil, fmt.Errorf("create auth: %w", err)
	}

	return &RelayClient{
		url:       relayURL,
		pending:   make(map[string]chan *jsonRPCResponse),
		logger:    logutils.ZapLogger(),
		projectID: projectID,
		auth:      auth,
		done:      make(chan struct{}),
	}, nil
}

// Connect establishes WebSocket connection to the relay server.
func (r *RelayClient) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		return nil
	}

	jwt, err := r.auth.GenerateJWT(r.url)
	if err != nil {
		return fmt.Errorf("generate auth jwt: %w", err)
	}

	u, err := url.Parse(r.url)
	if err != nil {
		return fmt.Errorf("parse relay url: %w", err)
	}
	q := u.Query()
	q.Set("auth", jwt)
	q.Set("projectId", r.projectID)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial relay: %w", err)
	}

	r.conn = conn

	r.wg.Add(1)
	go r.readLoop()
	return nil
}

// Close gracefully shuts down the WebSocket connection and waits for goroutines to finish.
func (r *RelayClient) Close() error {
	r.mu.Lock()
	if r.conn == nil {
		r.mu.Unlock()
		return nil
	}
	conn := r.conn
	r.conn = nil
	r.mu.Unlock()

	// Signal shutdown
	close(r.done)

	// Close websocket
	err := conn.Close()

	// Wait for readLoop to finish
	r.wg.Wait()

	return err
}

// Subscribe subscribes to a topic and returns the subscription ID.
func (r *RelayClient) Subscribe(topic string) (string, error) {
	result, err := r.call("irn_subscribe", map[string]string{"topic": topic})
	if err != nil {
		return "", err
	}
	var subID string
	if err := json.Unmarshal(result, &subID); err != nil {
		return "", fmt.Errorf("parse subscribe result: %w", err)
	}
	return subID, nil
}

// Publish publishes a message to a topic.
func (r *RelayClient) Publish(topic, message string, tag int) error {
	params := map[string]any{
		"topic":   topic,
		"message": message,
		"ttl":     defaultTTL,
		"tag":     tag,
	}
	if tag == 0 {
		params["tag"] = defaultMessageTag
	}
	_, err := r.call("irn_publish", params)
	return err
}

// FetchMessages fetches undelivered messages for a topic.
func (r *RelayClient) FetchMessages(topic string) ([]RelayMessage, bool, error) {
	result, err := r.call("irn_fetchMessages", map[string]string{"topic": topic})
	if err != nil {
		return nil, false, err
	}

	var fetchResult struct {
		Messages []RelayMessage `json:"messages"`
		HasMore  bool           `json:"hasMore"`
	}
	if err := json.Unmarshal(result, &fetchResult); err != nil {
		return nil, false, fmt.Errorf("parse fetch result: %w", err)
	}
	return fetchResult.Messages, fetchResult.HasMore, nil
}

// Unsubscribe unsubscribes from a topic.
func (r *RelayClient) Unsubscribe(topic, subID string) error {
	_, err := r.call("irn_unsubscribe", map[string]string{
		"topic": topic,
		"id":    subID,
	})
	return err
}

// RelayMessage represents a message received from the relay.
type RelayMessage struct {
	Topic       string `json:"topic"`
	Message     string `json:"message"`
	PublishedAt int64  `json:"publishedAt"`
	Tag         int    `json:"tag"`
}

// SetMessageHandler sets the callback for incoming subscription messages.
// The handler is called when irn_subscription messages are received.
func (r *RelayClient) SetMessageHandler(handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageHandler = handler
}

func (r *RelayClient) call(method string, params any) (json.RawMessage, error) {
	// Generate WalletConnect-compliant numeric ID with 19 digits of entropy
	id := payloadID()
	idStr := fmt.Sprintf("%d", id)

	r.mu.Lock()
	ch := make(chan *jsonRPCResponse, 1)
	r.pending[idStr] = ch
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.pending, idStr)
		r.mu.Unlock()
	}()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id, // int64 marshals as JSON number
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	conn := r.conn
	r.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("relay error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-r.done:
		return nil, fmt.Errorf("relay client shutting down")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("relay call timeout")
	}
}

func (r *RelayClient) readLoop() {
	defer common.LogOnPanic()
	defer r.wg.Done()

	for {
		r.mu.Lock()
		conn := r.conn
		r.mu.Unlock()
		if conn == nil {
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			r.logger.Debug("relay read error", zap.Error(err))
			return
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			r.logger.Debug("failed to parse relay message as response",
				zap.Error(err),
				zap.String("message_preview", truncate(string(msg), 200)))
			continue
		}

		var notif jsonRPCNotification
		if err := json.Unmarshal(msg, &notif); err == nil && notif.Method == "irn_subscription" && notif.Params.Data.Topic != "" {
			r.mu.Lock()
			handler := r.messageHandler
			r.mu.Unlock()
			if handler != nil {
				handler(notif.Params.Data.Topic, notif.Params.Data.Message, notif.Params.Data.Tag)
			}
			continue
		}

		if resp.idString() == "" {
			r.logger.Debug("ignoring relay message with empty ID")
			continue
		}

		r.mu.Lock()
		if ch, ok := r.pending[resp.idString()]; ok {
			select {
			case ch <- &resp:
			default:
			}
		} else {
			r.logger.Debug("received response for unknown request ID",
				zap.String("id", resp.idString()))
		}
		r.mu.Unlock()
	}
}
