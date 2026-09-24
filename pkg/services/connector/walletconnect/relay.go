package walletconnect

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/websocket"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
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
	relayReconnectWait = 5 * time.Second // how long a call waits for an in-flight reconnect
	relayCallTimeout   = 30 * time.Second

	relayReconnectBackoff    = time.Second // first pause before a redial
	relayReconnectMaxBackoff = time.Minute
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

// ErrRelayClosed is returned by Connect once the relay client has been closed.
var ErrRelayClosed = errors.New("relay client closed: disconnect requested")

// RelayClient implements the WalletConnect IRN relay protocol over WebSocket.
//
// One goroutine (run) owns the connection state machine: Connect, Close,
// calls, dials, readers, heartbeats and timers send it events, and it executes
// the commands Machine.Step returns. See RELAY_SPEC.md.
type RelayClient struct {
	url       string
	projectID string
	auth      *Auth
	logger    *zap.Logger

	messageHandler     atomic.Pointer[MessageHandler]
	reconnectedHandler atomic.Pointer[ReconnectedHandler]

	pendingMu sync.Mutex
	pending   map[string]chan *jsonRPCResponse

	writeMu sync.Mutex // serializes writes (gorilla/websocket requires it)

	events    chan loopEvent // unbuffered: a sender knows the loop took its event
	startOnce sync.Once
	loopDone  chan struct{} // closed once the loop has exited in Closed
	final     machine       // the machine the loop exited with; read only after loopDone
	nextCall  atomic.Uint64
	wg        sync.WaitGroup // the loop and every dial, reader and heartbeat
}

// loopEvent is a machine event plus what the shell needs to act on it.
type loopEvent struct {
	relayEvent
	connectReply chan error      // eventConnect
	callReply    chan callResult // eventCall
	ws           *websocket.Conn // eventDialOK
	backoffGen   uint64          // eventBackoffElapsed
}

type callResult struct {
	conn *relayConn
	err  error
}

type relayConn struct {
	id   connID
	ws   *websocket.Conn
	stop chan struct{} // closed by CloseConn; stops the heartbeat
}

type waitingCall struct {
	reply chan callResult
	timer *time.Timer // wait budget, set while parked
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
		events:    make(chan loopEvent),
		loopDone:  make(chan struct{}),
	}, nil
}

// send hands ev to the loop, starting it on first use. It returns false once
// the loop has exited; the caller then answers from the final machine.
func (r *RelayClient) send(ev loopEvent) bool {
	r.startOnce.Do(func() {
		r.wg.Add(1)
		go r.run()
	})
	select {
	case r.events <- ev:
		return true
	case <-r.loopDone:
		return false
	}
}

// post is send for readers, heartbeats and timers, whose events do not
// matter once the loop has exited.
func (r *RelayClient) post(ev loopEvent) {
	select {
	case r.events <- ev:
	case <-r.loopDone:
	}
}

// replyAfterClose answers an event the loop no longer takes, using the machine
// it exited with (which is Closed).
func (r *RelayClient) replyAfterClose(e relayEvent) error {
	_, cmds := r.final.Step(e)
	for _, c := range cmds {
		if c.kind == cmdReplyConnect || c.kind == cmdFail {
			return c.err
		}
	}
	return nil
}

func (r *RelayClient) lost(c *relayConn) {
	r.post(loopEvent{relayEvent: relayEvent{kind: eventConnLost, conn: c.id}})
}

// relayLoop is the state owned by the loop goroutine.
type relayLoop struct {
	r *RelayClient
	m machine

	conns    map[connID]*relayConn
	nextConn connID

	dialing    bool
	dialCancel context.CancelFunc

	backoffTimer *time.Timer
	backoffGen   uint64 // tells the running backoff timer from stopped ones

	connectWaiters []chan error
	calls          map[callID]*waitingCall
}

func (r *RelayClient) run() {
	defer panics.LogOnPanic()
	defer r.wg.Done()

	l := &relayLoop{
		r:     r,
		m:     newMachine(relayReconnectBackoff, relayReconnectMaxBackoff),
		conns: make(map[connID]*relayConn),
		calls: make(map[callID]*waitingCall),
	}
	for {
		ev := <-r.events
		if !l.accept(&ev) {
			continue
		}
		var cmds []relayCommand
		l.m, cmds = l.m.Step(ev.relayEvent)
		for _, c := range cmds {
			l.exec(c)
		}
		if l.m.state == stateClosed && !l.dialing {
			r.final = l.m
			close(r.loopDone)
			return
		}
	}
}

// accept records what the commands for ev will need and drops events of a
// backoff timer that was stopped. It returns false for dropped events.
func (l *relayLoop) accept(ev *loopEvent) bool {
	switch ev.kind {
	case eventConnect:
		l.connectWaiters = append(l.connectWaiters, ev.connectReply)
	case eventCall:
		l.calls[ev.call] = &waitingCall{reply: ev.callReply}
	case eventDialOK:
		l.dialDone()
		l.nextConn++
		ev.conn = l.nextConn
		l.conns[ev.conn] = &relayConn{id: ev.conn, ws: ev.ws, stop: make(chan struct{})}
	case eventDialFailed:
		l.dialDone()
		l.r.logger.Debug("failed to dial relay", zap.Error(ev.err), zap.Int("attempt", l.m.attempt))
	case eventConnLost:
		if l.m.state == stateConnected && ev.conn == l.m.conn {
			l.r.logger.Info("relay connection lost, reconnecting")
		}
	case eventBackoffElapsed:
		if ev.backoffGen != l.backoffGen || l.backoffTimer == nil {
			return false
		}
		l.backoffTimer = nil
	}
	return true
}

func (l *relayLoop) dialDone() {
	l.dialing = false
	l.dialCancel()
	l.dialCancel = nil
}

func (l *relayLoop) exec(c relayCommand) {
	r := l.r
	switch c.kind {
	case cmdStartDial:
		if !l.m.first {
			r.logger.Info("reconnect attempt", zap.Int("attempt", l.m.attempt), zap.Duration("backoff", l.m.delay))
		}
		ctx, cancel := context.WithCancel(context.Background())
		l.dialing, l.dialCancel = true, cancel
		r.wg.Add(1)
		go r.dial(ctx)
	case cmdAbortDial:
		l.dialCancel()
	case cmdCloseConn:
		conn := l.conns[c.conn]
		delete(l.conns, c.conn)
		close(conn.stop)
		_ = conn.ws.Close()
	case cmdStartBackoff:
		l.backoffGen++
		gen := l.backoffGen
		l.backoffTimer = time.AfterFunc(c.delay, func() {
			defer panics.LogOnPanic()
			r.post(loopEvent{relayEvent: relayEvent{kind: eventBackoffElapsed}, backoffGen: gen})
		})
	case cmdStopBackoff:
		if l.backoffTimer != nil {
			l.backoffTimer.Stop()
			l.backoffTimer = nil
		}
		l.backoffGen++
	case cmdStartHeartbeat:
		conn := l.conns[c.conn]
		_ = conn.ws.SetReadDeadline(time.Now().Add(relayReadDeadline))
		conn.ws.SetPongHandler(func(string) error {
			return conn.ws.SetReadDeadline(time.Now().Add(relayReadDeadline))
		})
		r.wg.Add(1)
		go r.heartbeat(conn)
	case cmdStartReader:
		r.wg.Add(1)
		go r.read(l.conns[c.conn])
	case cmdNotifyReconnected:
		r.logger.Info("successfully reconnected to relay")
		if handler := r.reconnectedHandler.Load(); handler != nil && *handler != nil {
			go func() {
				defer panics.LogOnPanic()
				(*handler)()
			}()
		}
	case cmdReplyConnect:
		for _, reply := range l.connectWaiters {
			reply <- c.err
		}
		l.connectWaiters = nil
	case cmdServe:
		l.answer(c.call, callResult{conn: l.conns[c.conn]})
	case cmdPark:
		id := c.call
		l.calls[id].timer = time.AfterFunc(relayReconnectWait, func() {
			defer panics.LogOnPanic()
			r.post(loopEvent{relayEvent: relayEvent{kind: eventWaitExpired, call: id}})
		})
	case cmdFail:
		l.answer(c.call, callResult{err: c.err})
	}
}

func (l *relayLoop) answer(id callID, res callResult) {
	call := l.calls[id]
	delete(l.calls, id)
	if call.timer != nil {
		call.timer.Stop()
	}
	call.reply <- res
}

func (r *RelayClient) dial(ctx context.Context) {
	defer panics.LogOnPanic()
	defer r.wg.Done()
	ws, err := r.dialRelay(ctx)
	if err != nil {
		r.events <- loopEvent{relayEvent: relayEvent{kind: eventDialFailed, err: err}}
		return
	}
	r.events <- loopEvent{relayEvent: relayEvent{kind: eventDialOK}, ws: ws}
}

// dialRelay opens a new WebSocket to r.url with auth query parameters.
// Cancelling dialCtx aborts the dial, including a handshake the relay never answers.
func (r *RelayClient) dialRelay(dialCtx context.Context) (*websocket.Conn, error) {
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

	// gorilla stops a handshake on context deadlines only, not on cancellation,
	// so the socket is closed on cancellation to abort a handshake the relay never answers.
	var stopWatch func() bool
	dialer := *websocket.DefaultDialer
	dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netConn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		stopWatch = context.AfterFunc(dialCtx, func() { _ = netConn.Close() })
		return netConn, nil
	}
	conn, resp, err := dialer.DialContext(dialCtx, u.String(), nil)
	if stopWatch != nil {
		stopWatch()
	}
	if err != nil {
		// gorilla collapses every non-101 answer into "bad handshake".
		status, body := 0, ""
		if resp != nil {
			defer resp.Body.Close()
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			status, body = resp.StatusCode, truncate(string(raw), 256)
		}
		failure := classifyDialFailure(status, body)
		r.logger.Warn("relay handshake rejected",
			zap.Int("status", status),
			zap.String("class", failure.class),
			zap.Bool("retryable", failure.retryable),
			zap.String("body", body),
			zap.Int("projectIdLen", len(r.projectID)))
		if status != 0 {
			return nil, fmt.Errorf("dial relay: %w (http %d %s, retryable=%t: %s)",
				err, status, failure.class, failure.retryable, body)
		}
		return nil, fmt.Errorf("dial relay: %w (%s, retryable=%t)", err, failure.class, failure.retryable)
	}
	return conn, nil
}

// heartbeat pings conn until CloseConn stops it or a ping fails.
func (r *RelayClient) heartbeat(conn *relayConn) {
	defer panics.LogOnPanic()
	defer r.wg.Done()
	ticker := time.NewTicker(relayPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-conn.stop:
			return
		case <-ticker.C:
			r.writeMu.Lock()
			_ = conn.ws.SetWriteDeadline(time.Now().Add(relayWriteDeadline))
			err := conn.ws.WriteMessage(websocket.PingMessage, nil)
			r.writeMu.Unlock()
			if err != nil {
				r.lost(conn)
				return
			}
		}
	}
}

// writeMessage sends a text frame under writeMu with a write deadline.
func (r *RelayClient) writeMessage(conn *relayConn, data []byte) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_ = conn.ws.SetWriteDeadline(time.Now().Add(relayWriteDeadline))
	return conn.ws.WriteMessage(websocket.TextMessage, data)
}

// Connect establishes WebSocket connection to the relay server.
func (r *RelayClient) Connect() error {
	reply := make(chan error, 1)
	ev := loopEvent{relayEvent: relayEvent{kind: eventConnect}, connectReply: reply}
	if !r.send(ev) {
		return r.replyAfterClose(ev.relayEvent)
	}
	return <-reply
}

// Close shuts the connection down and waits for the relay goroutines to finish.
func (r *RelayClient) Close() error {
	r.send(loopEvent{relayEvent: relayEvent{kind: eventClose}})
	<-r.loopDone
	r.wg.Wait()
	return nil
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
	r.messageHandler.Store(&handler)
}

// SetReconnectedHandler sets the callback that is invoked after a successful reconnection.
// The handler should re-subscribe to all active topics.
func (r *RelayClient) SetReconnectedHandler(handler ReconnectedHandler) {
	r.reconnectedHandler.Store(&handler)
}

// acquire waits until the loop lets a call use a connection or fails it.
func (r *RelayClient) acquire() (*relayConn, error) {
	reply := make(chan callResult, 1)
	ev := loopEvent{relayEvent: relayEvent{kind: eventCall, call: callID(r.nextCall.Add(1))}, callReply: reply}
	if !r.send(ev) {
		return nil, r.replyAfterClose(ev.relayEvent)
	}
	res := <-reply
	return res.conn, res.err
}

func (r *RelayClient) call(method string, params any) (json.RawMessage, error) {
	id := payloadID()
	idStr := fmt.Sprintf("%d", id)

	r.pendingMu.Lock()
	ch := make(chan *jsonRPCResponse, 1)
	r.pending[idStr] = ch
	r.pendingMu.Unlock()

	defer func() {
		r.pendingMu.Lock()
		delete(r.pending, idStr)
		r.pendingMu.Unlock()
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

	conn, err := r.acquire()
	if err != nil {
		return nil, err
	}

	if werr := r.writeMessage(conn, body); werr != nil {
		r.lost(conn)
		conn, err = r.acquire()
		if err != nil {
			return nil, fmt.Errorf("write %v; %w", werr, err)
		}
		if err := r.writeMessage(conn, body); err != nil {
			r.lost(conn)
			return nil, fmt.Errorf("write: %w", err)
		}
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("relay error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-r.loopDone:
		return nil, errShuttingDown
	case <-time.After(relayCallTimeout):
		return nil, fmt.Errorf("relay call timeout")
	}
}

// read delivers the messages of conn until reading fails.
func (r *RelayClient) read(conn *relayConn) {
	defer panics.LogOnPanic()
	defer r.wg.Done()

	for {
		_, msg, err := conn.ws.ReadMessage()
		if err != nil {
			r.logger.Debug("relay read error", zap.Error(err))
			r.lost(conn)
			return
		}
		r.dispatch(msg)
	}
}

func (r *RelayClient) dispatch(msg []byte) {
	var resp jsonRPCResponse
	if err := json.Unmarshal(msg, &resp); err != nil {
		r.logger.Debug("failed to parse relay message as response",
			zap.Error(err),
			zap.String("message_preview", truncate(string(msg), 200)))
		return
	}

	var notif jsonRPCNotification
	if err := json.Unmarshal(msg, &notif); err == nil && notif.Method == "irn_subscription" && notif.Params.Data.Topic != "" {
		if handler := r.messageHandler.Load(); handler != nil && *handler != nil {
			(*handler)(notif.Params.Data.Topic, notif.Params.Data.Message, notif.Params.Data.Tag)
		}
		return
	}

	id := resp.idString()
	if id == "" {
		r.logger.Debug("ignoring relay message with empty ID")
		return
	}

	r.pendingMu.Lock()
	defer r.pendingMu.Unlock()
	if ch, ok := r.pending[id]; ok {
		select {
		case ch <- &resp:
		default:
		}
	} else {
		r.logger.Debug("received response for unknown request ID",
			zap.String("id", id))
	}
}
