package walletconnect

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
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

// Tunable intervals for tests (defaults match production behavior).
var (
	relayWriteDeadline = 10 * time.Second
	relayReadDeadline  = 60 * time.Second
	relayPingInterval  = 30 * time.Second
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
	randomPart, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(1_000_000))
	return time.Now().UnixMilli()*1_000_000 + randomPart.Int64()
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

// ReconnectedHandler is called after a successful reconnection.
type ReconnectedHandler func()

// RelayClient implements the WalletConnect IRN relay protocol over WebSocket.
type RelayClient struct {
	url                 string
	conn                *websocket.Conn
	mu                  sync.Mutex
	pending             map[string]chan *jsonRPCResponse
	messageHandler      MessageHandler
	reconnectedHandler  ReconnectedHandler
	logger              *zap.Logger
	projectID           string
	auth                *Auth
	done                chan struct{}  // signals shutdown
	disconnectRequested bool           // true if Close() was called intentionally
	connectedOnce       bool           // true after first successful Connect(); avoids cold-start dial from call()
	wg                  sync.WaitGroup // tracks active goroutines

	writeMu      sync.Mutex // serializes writes (gorilla/websocket requires it)
	reconnectMu  sync.Mutex // serializes reconnect attempts
	readLoopOnce sync.Once  // starts readLoop at most once per RelayClient lifetime
	closeOnce    sync.Once  // closes r.done at most once (Close idempotent)
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

func (r *RelayClient) getConn() *websocket.Conn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn
}

// markBroken closes conn only if it is still the active connection and clears r.conn.
func (r *RelayClient) markBroken(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	r.mu.Lock()
	if r.conn == conn {
		_ = r.conn.Close()
		r.conn = nil
	}
	r.mu.Unlock()
}

// ensureConnected reconnects when r.conn is nil; concurrent callers serialize and share the result.
func (r *RelayClient) ensureConnected() error {
	r.reconnectMu.Lock()
	defer r.reconnectMu.Unlock()
	if r.getConn() != nil {
		return nil
	}
	return r.reconnect()
}

// dialRelay opens a new WebSocket to r.url with auth query parameters.
func (r *RelayClient) dialRelay() (*websocket.Conn, error) {
	jwt, err := r.auth.GenerateJWT(r.url)
	if err != nil {
		return nil, fmt.Errorf("generate auth jwt: %w", err)
	}

	u, err := url.Parse(r.url)
	if err != nil {
		return nil, fmt.Errorf("parse relay url: %w", err)
	}
	q := u.Query()
	q.Set("auth", jwt)
	q.Set("projectId", r.projectID)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dial relay: %w", err)
	}
	return conn, nil
}

// startHeartbeat launches a ping goroutine for conn. The goroutine exits automatically
// when conn is replaced (c != conn check) or when r.done is closed.
func (r *RelayClient) startHeartbeat(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(relayReadDeadline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(relayReadDeadline))
	})

	r.wg.Add(1)
	go func() {
		defer common.LogOnPanic()
		defer r.wg.Done()
		ticker := time.NewTicker(relayPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.done:
				return
			case <-ticker.C:
				r.writeMu.Lock()
				c := r.getConn()
				if c != conn || c == nil {
					r.writeMu.Unlock()
					return
				}
				_ = c.SetWriteDeadline(time.Now().Add(relayWriteDeadline))
				err := c.WriteMessage(websocket.PingMessage, nil)
				r.writeMu.Unlock()
				if err != nil {
					r.markBroken(c)
					return
				}
			}
		}
	}()
}

// writeMessage sends a text frame under writeMu with a write deadline.
func (r *RelayClient) writeMessage(conn *websocket.Conn, data []byte) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(relayWriteDeadline))
	return conn.WriteMessage(websocket.TextMessage, data)
}

// Connect establishes WebSocket connection to the relay server.
func (r *RelayClient) Connect() error {
	if r.getConn() != nil {
		return nil
	}

	conn, err := r.dialRelay()
	if err != nil {
		return err
	}

	r.mu.Lock()
	switch {
	case r.disconnectRequested:
		r.mu.Unlock()
		_ = conn.Close()
		return fmt.Errorf("disconnect requested")
	case r.conn != nil:
		r.mu.Unlock()
		_ = conn.Close()
		return nil
	}
	r.conn = conn
	r.connectedOnce = true
	r.mu.Unlock()

	r.startHeartbeat(conn)

	r.readLoopOnce.Do(func() {
		r.wg.Add(1)
		go r.readLoop()
	})
	return nil
}

// Close gracefully shuts down the WebSocket connection and waits for goroutines to finish.
func (r *RelayClient) Close() error {
	r.mu.Lock()
	r.disconnectRequested = true
	conn := r.conn
	r.conn = nil
	r.mu.Unlock()

	r.closeOnce.Do(func() {
		close(r.done)
	})

	var err error
	if conn != nil {
		err = conn.Close()
	}
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

// SetReconnectedHandler sets the callback that is invoked after a successful reconnection.
// The handler should re-subscribe to all active topics.
func (r *RelayClient) SetReconnectedHandler(handler ReconnectedHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reconnectedHandler = handler
}

func (r *RelayClient) call(method string, params any) (json.RawMessage, error) {
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

	conn := r.getConn()
	if conn == nil {
		r.mu.Lock()
		everConnected := r.connectedOnce
		r.mu.Unlock()
		if !everConnected {
			return nil, fmt.Errorf("not connected")
		}
		if err := r.ensureConnected(); err != nil {
			return nil, fmt.Errorf("reconnect: %w", err)
		}
		conn = r.getConn()
		if conn == nil {
			return nil, fmt.Errorf("connection lost immediately after reconnect")
		}
	}

	if err := r.writeMessage(conn, body); err != nil {
		r.markBroken(conn)
		if err2 := r.ensureConnected(); err2 != nil {
			return nil, fmt.Errorf("write failed: %v; reconnect failed: %w", err, err2)
		}
		newConn := r.getConn()
		if newConn == nil {
			return nil, fmt.Errorf("write failed: %v; connection lost after reconnect", err)
		}
		if err := r.writeMessage(newConn, body); err != nil {
			return nil, fmt.Errorf("write: %w", err)
		}
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
		disconnectRequested := r.disconnectRequested
		r.mu.Unlock()

		if conn == nil || disconnectRequested {
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			r.logger.Debug("relay read error", zap.Error(err))

			r.mu.Lock()
			wasIntentional := r.disconnectRequested
			r.mu.Unlock()

			if wasIntentional {
				return
			}

			r.markBroken(conn)

			r.logger.Info("attempting to reconnect to relay")
			if reconnectErr := r.ensureConnected(); reconnectErr != nil {
				r.logger.Error("failed to reconnect, exiting readLoop", zap.Error(reconnectErr))
				return
			}

			continue
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

// reconnect attempts to reconnect to the relay with exponential backoff.
func (r *RelayClient) reconnect() error {
	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second
	maxAttempts := 10

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		r.logger.Info("reconnect attempt", zap.Int("attempt", attempt), zap.Duration("backoff", backoff))

		r.mu.Lock()
		if r.disconnectRequested {
			r.mu.Unlock()
			return fmt.Errorf("disconnect requested during reconnect")
		}
		r.mu.Unlock()

		select {
		case <-time.After(backoff):
		case <-r.done:
			return fmt.Errorf("disconnect requested during reconnect")
		}

		conn, err := r.dialRelay()
		if err != nil {
			r.logger.Debug("failed to dial relay", zap.Error(err), zap.Int("attempt", attempt))
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		r.mu.Lock()
		if r.disconnectRequested {
			r.mu.Unlock()
			_ = conn.Close()
			return fmt.Errorf("disconnect requested during reconnect")
		}
		if r.conn != nil {
			r.mu.Unlock()
			_ = conn.Close()
			return nil
		}
		r.conn = conn
		handler := r.reconnectedHandler
		r.mu.Unlock()

		r.startHeartbeat(conn)

		r.logger.Info("successfully reconnected to relay")

		if handler != nil {
			handler()
		}

		return nil
	}
	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}
