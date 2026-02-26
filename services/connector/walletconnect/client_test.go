package walletconnect

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/logutils"
)

// testSymKey is a 64-hex-char (32-byte) symmetric key used across tests.
const testSymKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// newMockClient creates a gomock controller, a MockRelay, and a test Client in one call.
// The controller is registered for automatic cleanup via t.Cleanup.
func newMockClient(t *testing.T) (*gomock.Controller, *MockRelay, *Client) {
	t.Helper()
	ctrl := gomock.NewController(t)
	relay := NewMockRelay(ctrl)
	return ctrl, relay, newTestClient(relay)
}

// addActiveSession registers symKey for topic in client.activeSessions under the client mutex.
func addActiveSession(client *Client, topic, symKey string) {
	client.mu.Lock()
	client.activeSessions[topic] = symKey
	client.mu.Unlock()
}

// addPairingTopic registers symKey for topic in client.pairingTopics under the client mutex.
func addPairingTopic(client *Client, topic, symKey string) {
	client.mu.Lock()
	client.pairingTopics[topic] = symKey
	client.mu.Unlock()
}

// newTestClient creates a Client with the supplied relay, bypassing NewClient.
// Use this to inject a MockRelay in tests.
func newTestClient(relay Relay) *Client {
	return &Client{
		relay:                  relay,
		logger:                 logutils.ZapLogger(),
		handlers:               &clientHandlers{},
		pendingProposals:       make(map[string]*pairingContext),
		pendingRequests:        make(map[int64]chan *JSONRPCResponse),
		pendingSessionRequests: make(map[int64]string),
		pairingTopics:          make(map[string]string),
		activeSessions:         make(map[string]string),
	}
}

// autoAckOnPublish returns a gomock Do-function that asynchronously resolves
// every pending request on c with a success result. Attach it to a Publish
// EXPECT to avoid 15-second timeouts in sendSessionSettle / SendSessionUpdate.
func autoAckOnPublish(c *Client) func(string, string, int) {
	return func(_, _ string, _ int) {
		go func() {
			time.Sleep(5 * time.Millisecond)
			c.mu.Lock()
			for id, ch := range c.pendingRequests {
				resp := &JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: true}
				select {
				case ch <- resp:
				default:
				}
			}
			c.mu.Unlock()
		}()
	}
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("test-project-id")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.relay)
	require.NotNil(t, client.handlers)
	require.NotNil(t, client.pendingProposals)
	require.NotNil(t, client.pendingRequests)
	require.NotNil(t, client.pairingTopics)
	require.NotNil(t, client.activeSessions)
}

func TestClient_SetHandlers(t *testing.T) {
	t.Run("SessionProposal", func(t *testing.T) {
		client, _ := NewClient("test")
		called := false
		client.SetSessionProposalHandler(func(string) { called = true })
		client.mu.Lock()
		h := client.handlers.onSessionProposal
		client.mu.Unlock()
		h("test")
		require.True(t, called)
	})

	t.Run("SessionRequest", func(t *testing.T) {
		client, _ := NewClient("test")
		called := false
		client.SetSessionRequestHandler(func(_, _ string) { called = true })
		client.mu.Lock()
		h := client.handlers.onSessionRequest
		client.mu.Unlock()
		h("topic", "request")
		require.True(t, called)
	})

	t.Run("SessionUpdate", func(t *testing.T) {
		client, _ := NewClient("test")
		called := false
		client.SetSessionUpdateHandler(func(_, _ string) { called = true })
		client.mu.Lock()
		h := client.handlers.onSessionUpdate
		client.mu.Unlock()
		h("topic", `{"namespaces":{}}`)
		require.True(t, called)
	})

	t.Run("SessionDelete", func(t *testing.T) {
		client, _ := NewClient("test")
		called := false
		client.SetSessionDeleteHandler(func(string) { called = true })
		client.mu.Lock()
		h := client.handlers.onSessionDelete
		client.mu.Unlock()
		require.NotNil(t, h)
		h("topic")
		require.True(t, called)
	})
}

func TestClient_GetSymKeyForTopic(t *testing.T) {
	tests := []struct {
		name         string
		setupPairing map[string]string
		setupSession map[string]string
		topic        string
		wantKey      string
		wantFound    bool
	}{
		{
			name:         "PairingTopic",
			setupPairing: map[string]string{"topic1": "key1"},
			topic:        "topic1",
			wantKey:      "key1",
			wantFound:    true,
		},
		{
			name:         "SessionTopic",
			setupSession: map[string]string{"topic2": "key2"},
			topic:        "topic2",
			wantKey:      "key2",
			wantFound:    true,
		},
		{
			name:      "NotFound",
			topic:     "nonexistent",
			wantFound: false,
		},
		{
			name:         "SessionPriority",
			setupPairing: map[string]string{"topic": "pairing-key"},
			setupSession: map[string]string{"topic": "session-key"},
			topic:        "topic",
			wantKey:      "session-key",
			wantFound:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient("test")
			client.mu.Lock()
			for k, v := range tt.setupPairing {
				client.pairingTopics[k] = v
			}
			for k, v := range tt.setupSession {
				client.activeSessions[k] = v
			}
			client.mu.Unlock()
			key, ok := client.getSymKeyForTopic(tt.topic)
			require.Equal(t, tt.wantFound, ok)
			require.Equal(t, tt.wantKey, key)
		})
	}
}

func TestClient_RemoveSession(t *testing.T) {
	client, _ := NewClient("test")

	client.mu.Lock()
	client.activeSessions["topic1"] = "key1"
	client.activeSessions["topic2"] = "key2"
	client.mu.Unlock()

	client.RemoveSession("topic1")

	_, ok := client.getSymKeyForTopic("topic1")
	require.False(t, ok)

	_, ok = client.getSymKeyForTopic("topic2")
	require.True(t, ok)
}

func TestClient_HandleRelayMessage_NoSymKey(t *testing.T) {
	client, _ := NewClient("test")

	client.handleRelayMessage("unknown-topic", "message", 1100)
}

func TestClient_HandleRelayMessage_InvalidEncryption(t *testing.T) {
	client, _ := NewClient("test")

	addPairingTopic(client, "topic", testSymKey)

	client.handleRelayMessage("topic", "invalid-base64", 1100)
}

func TestClient_HandleRelayMessage_SessionProposal(t *testing.T) {
	client, _ := NewClient("test")

	addPairingTopic(client, "topic", testSymKey)

	receivedProposal := ""
	client.SetSessionProposalHandler(func(proposalJSON string) {
		receivedProposal = proposalJSON
	})

	proposal := map[string]any{
		"id":     int64(123),
		"method": "wc_sessionPropose",
		"params": map[string]any{
			"proposer": map[string]any{
				"publicKey": "abcd1234",
			},
		},
	}
	proposalJSON, _ := json.Marshal(proposal)
	encryptedProposal, _ := EncryptType0Envelope(testSymKey, proposalJSON)

	client.handleRelayMessage("topic", encryptedProposal, tagSessionPropose)

	time.Sleep(10 * time.Millisecond)
	require.NotEmpty(t, receivedProposal)

	client.mu.Lock()
	_, exists := client.pendingProposals["123"]
	client.mu.Unlock()
	require.True(t, exists)
}

func TestClient_HandleRelayMessage_SessionRequest(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	receivedTopic := ""
	receivedRequest := ""
	client.SetSessionRequestHandler(func(topic, requestJSON string) {
		receivedTopic = topic
		receivedRequest = requestJSON
	})

	request := map[string]any{
		"id":     int64(456),
		"method": "wc_sessionRequest",
		"params": map[string]any{
			"request": map[string]any{
				"method": "personal_sign",
			},
		},
	}
	requestJSON, _ := json.Marshal(request)
	encryptedRequest, _ := EncryptType0Envelope(testSymKey, requestJSON)

	client.handleRelayMessage("topic", encryptedRequest, tagSessionRequest)

	time.Sleep(10 * time.Millisecond)
	require.Equal(t, "topic", receivedTopic)
	require.NotEmpty(t, receivedRequest)
}

func TestClient_RejectSession_NotFound(t *testing.T) {
	client, _ := NewClient("test")

	err := client.RejectSession("nonexistent")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProposalNotFound)
}

func TestClient_RespondToWCSessionRequest_NoSession(t *testing.T) {
	client, _ := NewClient("test")

	err := client.RespondToWCSessionRequest("unknown-topic", 123, "result")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestClient_ApproveSession_NoProposal(t *testing.T) {
	client, _ := NewClient("test")

	meta := SessionMetadata{
		Account: "0x1234",
		ChainID: 1,
	}
	_, err := client.ApproveSession(context.Background(), "nonexistent", meta)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProposalNotFound)
}

// TestSentinelErrors verifies that sentinel errors can be checked with errors.Is
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedError error
	}{
		{
			name:          "ErrProposalNotFound",
			err:           ErrProposalNotFound,
			expectedError: ErrProposalNotFound,
		},
		{
			name:          "ErrSessionNotFound",
			err:           ErrSessionNotFound,
			expectedError: ErrSessionNotFound,
		},
		{
			name:          "ErrInvalidPublicKey",
			err:           ErrInvalidPublicKey,
			expectedError: ErrInvalidPublicKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.True(t, errors.Is(tt.err, tt.expectedError))
		})
	}
}

func TestClient_RegisterPending_ResolvePending(t *testing.T) {
	client, _ := NewClient("test")

	ch := client.registerPending(12345)
	require.NotNil(t, ch)

	resp := &JSONRPCResponse{JSONRPC: "2.0", ID: 12345, Result: true}
	ok := client.resolvePending(12345, resp)
	require.True(t, ok)

	got := <-ch
	require.Equal(t, true, got.Result)

	// Second resolve for same ID should not match
	ok = client.resolvePending(12345, resp)
	require.False(t, ok)
}

func TestClient_HandleRelayMessage_JSONRPCResponse(t *testing.T) {
	t.Run("ResolvesPending", func(t *testing.T) {
		client, _ := NewClient("test")
		addActiveSession(client, "topic", testSymKey)

		ch := client.registerPending(999)
		resp := map[string]any{"jsonrpc": "2.0", "id": 999, "result": true}
		respJSON, _ := json.Marshal(resp)
		encrypted, _ := EncryptType0Envelope(testSymKey, respJSON)

		client.handleRelayMessage("topic", encrypted, tagSessionSettleResponse)

		select {
		case r := <-ch:
			require.NotNil(t, r)
			require.Equal(t, true, r.Result)
			require.Nil(t, r.Error)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected response to be delivered to pending channel")
		}
	})

	t.Run("WithError", func(t *testing.T) {
		client, _ := NewClient("test")
		addActiveSession(client, "topic", testSymKey)

		ch := client.registerPending(777)
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      int64(777),
			"error":   map[string]any{"code": 5000, "message": "rejected"},
		}
		respJSON, _ := json.Marshal(resp)
		encrypted, _ := EncryptType0Envelope(testSymKey, respJSON)

		client.handleRelayMessage("topic", encrypted, tagSessionSettleResponse)

		select {
		case r := <-ch:
			require.NotNil(t, r.Error)
			require.Equal(t, "rejected", r.Error.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected error response")
		}
	})
}

func TestClient_HandleRelayMessage_SessionPing(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	// Use a mock relay to capture the publish call
	// For now we just verify it doesn't panic - full test would need relay mock
	ping := map[string]any{
		"id":     int64(1),
		"method": "wc_sessionPing",
		"params": map[string]any{},
	}
	pingJSON, _ := json.Marshal(ping)
	encrypted, _ := EncryptType0Envelope(testSymKey, pingJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionPing)
	})
}

func TestClient_HandleRelayMessage_SessionUpdate(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	called := false
	client.SetSessionUpdateHandler(func(topic, namespacesJSON string) {
		called = true
		require.Equal(t, "topic", topic)
		require.Contains(t, namespacesJSON, "eip155")
	})

	update := map[string]any{
		"id":     int64(2),
		"method": "wc_sessionUpdate",
		"params": map[string]any{
			"namespaces": map[string]any{
				"eip155": map[string]any{
					"accounts": []string{"eip155:1:0x123"},
					"chains":   []string{"eip155:1"},
					"methods":  []string{"personal_sign"},
					"events":   []string{"accountsChanged"},
				},
			},
		},
	}
	updateJSON, _ := json.Marshal(update)
	encrypted, _ := EncryptType0Envelope(testSymKey, updateJSON)

	client.handleRelayMessage("topic", encrypted, tagSessionUpdate)

	time.Sleep(10 * time.Millisecond)
	require.True(t, called)
}

func TestClient_HandleRelayMessage_SessionDelete(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	called := false
	var receivedTopic string
	client.SetSessionDeleteHandler(func(topic string) {
		called = true
		receivedTopic = topic
	})

	deleteReq := map[string]any{
		"id":     int64(3),
		"method": "wc_sessionDelete",
		"params": map[string]any{"code": int64(1000), "message": "disconnect"},
	}
	deleteJSON, _ := json.Marshal(deleteReq)
	encrypted, _ := EncryptType0Envelope(testSymKey, deleteJSON)

	client.handleRelayMessage("topic", encrypted, tagSessionDelete)

	require.True(t, called)
	require.Equal(t, "topic", receivedTopic)
	client.mu.Lock()
	_, exists := client.activeSessions["topic"]
	client.mu.Unlock()
	require.False(t, exists, "session should be removed from activeSessions on wc_sessionDelete")
}

func TestClient_Close(t *testing.T) {
	client, _ := NewClient("test")

	err := client.Close()
	require.NoError(t, err)
}

func TestClient_SendSessionEvent_NoSession(t *testing.T) {
	client, _ := NewClient("test")

	err := client.SendSessionEvent("unknown-topic", SessionEvent{Name: "accountsChanged", Data: []string{"0x123"}}, "eip155:1")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestClient_RestoreSessions(t *testing.T) {
	client, _ := NewClient("test")

	sessions := []RestoredSession{
		{Topic: "topic1", SymKey: "key1"},
		{Topic: "topic2", SymKey: "key2"},
		{Topic: "", SymKey: "key3"},   // Should be ignored
		{Topic: "topic4", SymKey: ""}, // Should be ignored
	}

	client.RestoreSessions(sessions)

	client.mu.Lock()
	require.Equal(t, 2, len(client.activeSessions))
	require.Equal(t, "key1", client.activeSessions["topic1"])
	require.Equal(t, "key2", client.activeSessions["topic2"])
	_, exists := client.activeSessions[""]
	require.False(t, exists)
	_, exists = client.activeSessions["topic4"]
	require.False(t, exists)
	client.mu.Unlock()
}

// validWCURI returns a well-formed WalletConnect v2 URI for the given topic+symKey.
// Topic must be 64 hex chars (32 bytes).
const testPairingTopic = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const testPairingSymKey = "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"

func validWCURI(topic, symKey string) string {
	return fmt.Sprintf("wc:%s@2?relay-protocol=irn&symKey=%s", topic, symKey)
}

func TestClient_Pair_InvalidURI(t *testing.T) {
	_, _, client := newMockClient(t)

	err := client.Pair(context.Background(), "not-a-valid-wc-uri")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse URI")
}

func TestClient_Pair_ConnectError(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().SetMessageHandler(gomock.Any())
	relay.EXPECT().Connect().Return(fmt.Errorf("dial failed"))

	err := client.Pair(context.Background(), validWCURI(testPairingTopic, testPairingSymKey))
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect relay")
}

func TestClient_Pair_SubscribeError(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().SetMessageHandler(gomock.Any())
	relay.EXPECT().Connect().Return(nil)
	relay.EXPECT().Subscribe(testPairingTopic).Return("", fmt.Errorf("subscribe failed"))

	err := client.Pair(context.Background(), validWCURI(testPairingTopic, testPairingSymKey))
	require.Error(t, err)
	require.Contains(t, err.Error(), "subscribe")
}

func TestClient_Pair_Success_NoMessages(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().SetMessageHandler(gomock.Any())
	relay.EXPECT().Connect().Return(nil)
	relay.EXPECT().Subscribe(testPairingTopic).Return("sub-id", nil)
	relay.EXPECT().FetchMessages(testPairingTopic).Return(nil, false, fmt.Errorf("no messages"))

	err := client.Pair(context.Background(), validWCURI(testPairingTopic, testPairingSymKey))
	require.NoError(t, err)
}

func TestClient_Pair_WithFetchedMessages(t *testing.T) {
	_, relay, client := newMockClient(t)

	payload := map[string]any{
		"id":     int64(1),
		"method": "wc_unknown",
		"params": map[string]any{},
	}
	payloadJSON, _ := json.Marshal(payload)
	encrypted, _ := EncryptType0Envelope(testPairingSymKey, payloadJSON)

	relay.EXPECT().SetMessageHandler(gomock.Any())
	relay.EXPECT().Connect().Return(nil)
	relay.EXPECT().Subscribe(testPairingTopic).Return("sub-id", nil)
	relay.EXPECT().FetchMessages(testPairingTopic).Return(
		[]RelayMessage{{Topic: testPairingTopic, Message: encrypted, Tag: defaultMessageTag}},
		false, nil,
	)

	err := client.Pair(context.Background(), validWCURI(testPairingTopic, testPairingSymKey))
	require.NoError(t, err)
}

func TestClient_ConnectAndResubscribe_ConnectError(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic1", "key1")

	relay.EXPECT().SetMessageHandler(gomock.Any())
	relay.EXPECT().Connect().Return(fmt.Errorf("connect failed"))

	err := client.ConnectAndResubscribe()
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect relay")
}

func TestClient_ConnectAndResubscribe_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic1", "key1")
	addActiveSession(client, "topic2", "key2")

	relay.EXPECT().SetMessageHandler(gomock.Any())
	relay.EXPECT().Connect().Return(nil)
	relay.EXPECT().Subscribe(gomock.Any()).Return("sub-id", nil).Times(2)

	err := client.ConnectAndResubscribe()
	require.NoError(t, err)
}

func TestClient_onReconnected_ResubscribesAll(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "session-topic", "skey")
	addPairingTopic(client, "pairing-topic", "pkey")

	relay.EXPECT().Subscribe(gomock.Any()).Return("sub-id", nil).Times(2)

	client.onReconnected()
}

func TestClient_resubscribeTopics_SubscribeError(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().Subscribe(gomock.Any()).Return("", fmt.Errorf("subscribe failed")).Times(2)

	require.NotPanics(t, func() {
		client.resubscribeTopics("session", []string{"topic1", "topic2"})
	})
}

func TestClient_RejectSession_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:  "pairing-topic",
		PairingSymKey: testSymKey,
		JsonRpcID:     42,
	}
	client.mu.Unlock()

	relay.EXPECT().Publish("pairing-topic", gomock.Any(), tagSessionProposeReject).Return(nil)

	err := client.RejectSession("proposal1")
	require.NoError(t, err)

	client.mu.Lock()
	_, exists := client.pendingProposals["proposal1"]
	client.mu.Unlock()
	require.False(t, exists)
}

func TestClient_RejectSession_PublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:  "pairing-topic",
		PairingSymKey: testSymKey,
		JsonRpcID:     42,
	}
	client.mu.Unlock()

	relay.EXPECT().Publish("pairing-topic", gomock.Any(), tagSessionProposeReject).Return(fmt.Errorf("publish failed"))

	err := client.RejectSession("proposal1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "publish reject")
}

func TestClient_RespondToWCSessionRequest_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)
	client.mu.Lock()
	client.pendingSessionRequests[999] = "topic"
	client.mu.Unlock()

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionRequestResponse).Return(nil)

	err := client.RespondToWCSessionRequest("topic", 999, "0xsignature")
	require.NoError(t, err)

	client.mu.Lock()
	_, still := client.pendingSessionRequests[999]
	client.mu.Unlock()
	require.False(t, still, "pendingSessionRequests entry must be removed after respond")
}

func TestClient_RespondToWCSessionRequest_PublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionRequestResponse).Return(fmt.Errorf("publish failed"))

	err := client.RespondToWCSessionRequest("topic", 999, "0xsig")
	require.Error(t, err)
}

func TestClient_RejectWCSessionRequest_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)
	client.mu.Lock()
	client.pendingSessionRequests[999] = "topic"
	client.mu.Unlock()

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionRequestResponse).Return(nil)

	err := client.RejectWCSessionRequest("topic", 999, 4001, "User rejected")
	require.NoError(t, err)

	client.mu.Lock()
	_, still := client.pendingSessionRequests[999]
	client.mu.Unlock()
	require.False(t, still, "pendingSessionRequests entry must be removed after reject")
}

func TestClient_RejectWCSessionRequest_NoSession(t *testing.T) {
	_, _, client := newMockClient(t)

	err := client.RejectWCSessionRequest("nonexistent-topic", 999, 4001, "rejected")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestClient_RejectWCSessionRequest_PublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionRequestResponse).Return(fmt.Errorf("publish failed"))

	err := client.RejectWCSessionRequest("topic", 999, 4001, "rejected")
	require.Error(t, err)
}

func TestClient_SendSessionDelete_SessionNotFound(t *testing.T) {
	_, _, client := newMockClient(t)

	// No Publish expected — session not in activeSessions
	err := client.SendSessionDelete(context.Background(), "nonexistent-topic")
	require.NoError(t, err)
}

func TestClient_SendSessionDelete_PublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionDelete).Return(fmt.Errorf("publish failed"))

	err := client.SendSessionDelete(context.Background(), "topic")
	require.Error(t, err)

	_, ok := client.getSymKeyForTopic("topic")
	require.False(t, ok, "session should be removed even when publish fails")
}

func TestClient_SendSessionDelete_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionDelete).
		DoAndReturn(func(topic, msg string, tag int) error {
			autoAckOnPublish(client)(topic, msg, tag)
			return nil
		})

	err := client.SendSessionDelete(context.Background(), "topic")
	require.NoError(t, err)

	_, ok := client.getSymKeyForTopic("topic")
	require.False(t, ok)
}

func TestClient_SendSessionUpdate_NoSession(t *testing.T) {
	_, _, client := newMockClient(t)

	err := client.SendSessionUpdate("nonexistent-topic", map[string]Namespace{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestClient_SendSessionUpdate_PublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionUpdate).Return(fmt.Errorf("publish failed"))

	err := client.SendSessionUpdate("topic", map[string]Namespace{})
	require.Error(t, err)
}

func TestClient_SendSessionUpdate_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionUpdate).
		DoAndReturn(func(topic, msg string, tag int) error {
			autoAckOnPublish(client)(topic, msg, tag)
			return nil
		})

	ns := map[string]Namespace{
		"eip155": {Chains: []string{"eip155:1"}, Methods: []string{"personal_sign"}, Events: []string{"accountsChanged"}},
	}
	err := client.SendSessionUpdate("topic", ns)
	require.NoError(t, err)
}

func TestClient_SendSessionEvent_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionEvent).
		DoAndReturn(func(topic, msg string, tag int) error {
			autoAckOnPublish(client)(topic, msg, tag)
			return nil
		})

	err := client.SendSessionEvent("topic", SessionEvent{Name: "accountsChanged", Data: []string{"0x123"}}, "eip155:1")
	require.NoError(t, err)
}

func TestClient_SendSessionEvent_PublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	addActiveSession(client, "topic", testSymKey)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionEvent).Return(fmt.Errorf("publish failed"))

	err := client.SendSessionEvent("topic", SessionEvent{Name: "accountsChanged"}, "eip155:1")
	require.Error(t, err)
}

func TestClient_ApproveSession_InvalidProposalParams(t *testing.T) {
	_, _, client := newMockClient(t)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:   "pairing-topic",
		PairingSymKey:  "symkey",
		ProposerPubKey: "abcd1234",
		JsonRpcID:      1,
		ProposalParams: json.RawMessage(`not-valid-json`),
	}
	client.mu.Unlock()

	_, err := client.ApproveSession(context.Background(), "proposal1", SessionMetadata{Account: "0x1234", ChainID: 1})
	require.Error(t, err)
}

func TestClient_ApproveSession_InvalidProposerKey(t *testing.T) {
	_, _, client := newMockClient(t)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:   "pairing-topic",
		PairingSymKey:  "symkey",
		ProposerPubKey: "not-valid-hex!!!",
		JsonRpcID:      1,
		ProposalParams: json.RawMessage(`{}`),
	}
	client.mu.Unlock()

	_, err := client.ApproveSession(context.Background(), "proposal1", SessionMetadata{Account: "0x1234", ChainID: 1})
	require.Error(t, err)
}

func TestClient_ApproveSession_ProposalResponsePublishError(t *testing.T) {
	_, relay, client := newMockClient(t)

	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:   "pairing-topic",
		PairingSymKey:  testSymKey,
		ProposerPubKey: hex.EncodeToString(proposerPub),
		JsonRpcID:      1,
		ProposalParams: json.RawMessage(`{}`),
	}
	client.mu.Unlock()

	relay.EXPECT().Publish("pairing-topic", gomock.Any(), tagSessionProposeResult).Return(fmt.Errorf("publish failed"))

	_, err = client.ApproveSession(context.Background(), "proposal1", SessionMetadata{Account: "0x1234", ChainID: 1, Chains: []int64{1}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "send proposal response")
}

func TestClient_ApproveSession_SubscribeError(t *testing.T) {
	_, relay, client := newMockClient(t)

	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:   "pairing-topic",
		PairingSymKey:  testSymKey,
		ProposerPubKey: hex.EncodeToString(proposerPub),
		JsonRpcID:      1,
		ProposalParams: json.RawMessage(`{}`),
	}
	client.mu.Unlock()

	// Proposal response publish succeeds, session subscribe fails
	relay.EXPECT().Publish("pairing-topic", gomock.Any(), tagSessionProposeResult).Return(nil)
	relay.EXPECT().Subscribe(gomock.Any()).Return("", fmt.Errorf("subscribe failed"))

	_, err = client.ApproveSession(context.Background(), "proposal1", SessionMetadata{Account: "0x1234", ChainID: 1, Chains: []int64{1}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "subscribe to session topic")
}

func TestClient_ApproveSession_Success(t *testing.T) {
	_, relay, client := newMockClient(t)

	_, proposerPub, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	client.mu.Lock()
	client.pendingProposals["proposal1"] = &pairingContext{
		PairingTopic:   "pairing-topic",
		PairingSymKey:  testSymKey,
		ProposerPubKey: hex.EncodeToString(proposerPub),
		JsonRpcID:      1,
		ProposalParams: json.RawMessage(`{
			"relays": [{"protocol":"irn"}],
			"proposer": {"publicKey":"` + hex.EncodeToString(proposerPub) + `","metadata":{"name":"Test","url":"https://test.com","description":"","icons":["https://test.com/icon.png"]}},
			"requiredNamespaces": {}
		}`),
	}
	client.mu.Unlock()

	// 1. sendProposalResponse → Publish on pairing topic
	relay.EXPECT().Publish("pairing-topic", gomock.Any(), tagSessionProposeResult).Return(nil)
	// 2. Subscribe to session topic
	relay.EXPECT().Subscribe(gomock.Any()).Return("sub-session", nil)
	// 3. sendSessionSettle → Publish on session topic (with autoAck)
	relay.EXPECT().Publish(gomock.Any(), gomock.Any(), tagSessionSettle).
		DoAndReturn(func(topic, msg string, tag int) error {
			autoAckOnPublish(client)(topic, msg, tag)
			return nil
		})

	result, err := client.ApproveSession(context.Background(), "proposal1",
		SessionMetadata{Account: "0x1234567890abcdef1234567890abcdef12345678", ChainID: 1, Chains: []int64{1}})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Topic)
	require.Equal(t, "pairing-topic", result.PairingTopic)
	require.NotEmpty(t, result.SessionJSON)
	require.NotEmpty(t, result.SymKey)

	client.mu.Lock()
	_, exists := client.pendingProposals["proposal1"]
	client.mu.Unlock()
	require.False(t, exists)

	_, ok := client.getSymKeyForTopic(result.Topic)
	require.True(t, ok)
}

func TestClient_sendAckResponse_PublishesSuccessfully(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().Publish("topic", gomock.Any(), tagSessionPingResponse).Return(nil)

	err := client.sendAckResponse("topic", testSymKey, 1, tagSessionPingResponse)
	require.NoError(t, err)
}

func TestClient_HandleRelayMessage_InvalidID(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	payload := map[string]any{
		"id":     []string{"not", "an", "id"},
		"method": "wc_sessionPing",
		"params": map[string]any{},
	}
	payloadJSON, _ := json.Marshal(payload)
	encrypted, _ := EncryptType0Envelope(testSymKey, payloadJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionPing)
	})
}

func TestClient_HandleRelayMessage_SessionProposal_MissingPublicKey(t *testing.T) {
	client, _ := NewClient("test")

	addPairingTopic(client, "topic", testSymKey)

	proposal := map[string]any{
		"id":     int64(1),
		"method": "wc_sessionPropose",
		"params": map[string]any{"proposer": map[string]any{"publicKey": ""}},
	}
	payloadJSON, _ := json.Marshal(proposal)
	encrypted, _ := EncryptType0Envelope(testSymKey, payloadJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionPropose)
	})
}

func TestClient_HandleRelayMessage_SessionProposal_NoHandler_UsesSignal(t *testing.T) {
	client, _ := NewClient("test")

	addPairingTopic(client, "topic", testSymKey)

	proposal := map[string]any{
		"id":     int64(42),
		"method": "wc_sessionPropose",
		"params": map[string]any{"proposer": map[string]any{"publicKey": "abcd1234"}},
	}
	payloadJSON, _ := json.Marshal(proposal)
	encrypted, _ := EncryptType0Envelope(testSymKey, payloadJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionPropose)
	})

	time.Sleep(10 * time.Millisecond)

	client.mu.Lock()
	_, exists := client.pendingProposals["42"]
	client.mu.Unlock()
	require.True(t, exists)
}

func TestClient_HandleRelayMessage_JSONRPCResponse_UnknownID(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	resp := map[string]any{"jsonrpc": "2.0", "id": int64(9999), "result": true}
	respJSON, _ := json.Marshal(resp)
	encrypted, _ := EncryptType0Envelope(testSymKey, respJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionSettleResponse)
	})
}

func TestClient_HandleRelayMessage_SessionRequest_NoHandler_UsesSignal(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	req := map[string]any{
		"id":     int64(1),
		"method": "wc_sessionRequest",
		"params": map[string]any{"request": map[string]any{"method": "personal_sign"}},
	}
	reqJSON, _ := json.Marshal(req)
	encrypted, _ := EncryptType0Envelope(testSymKey, reqJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionRequest)
	})
}

func TestClient_HandleRelayMessage_UnknownMethod(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	msg := map[string]any{
		"id":     int64(1),
		"method": "wc_unknownFutureMethod",
		"params": map[string]any{},
	}
	msgJSON, _ := json.Marshal(msg)
	encrypted, _ := EncryptType0Envelope(testSymKey, msgJSON)

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, 9999)
	})
}

func TestClient_HandleRelayMessage_InvalidJSON(t *testing.T) {
	client, _ := NewClient("test")

	addActiveSession(client, "topic", testSymKey)

	encrypted, _ := EncryptType0Envelope(testSymKey, []byte("not-json"))

	require.NotPanics(t, func() {
		client.handleRelayMessage("topic", encrypted, tagSessionRequest)
	})
}

func TestClient_Publish(t *testing.T) {
	_, relay, client := newMockClient(t)

	relay.EXPECT().Publish("topic", "message", tagSessionEvent).Return(nil)

	err := client.Publish("topic", "message", tagSessionEvent)
	require.NoError(t, err)
}

// encryptMsg is a test helper that builds and encrypts a JSON-RPC message.
func encryptMsg(t *testing.T, symKey string, method string, id int64, params any) string {
	t.Helper()
	msg := map[string]any{"id": id, "method": method, "params": params}
	raw, err := json.Marshal(msg)
	require.NoError(t, err)
	enc, err := EncryptType0Envelope(symKey, raw)
	require.NoError(t, err)
	return enc
}

// --- wc_sessionPropose deduplication ---

func TestClient_HandleRelayMessage_SessionProposal_Deduplicate(t *testing.T) {
	client, _ := NewClient("test")

	addPairingTopic(client, "topic", testSymKey)

	callCount := 0
	client.SetSessionProposalHandler(func(_ string) { callCount++ })

	encrypted := encryptMsg(t, testSymKey, "wc_sessionPropose", 100, map[string]any{
		"proposer": map[string]any{"publicKey": "abcd1234"},
	})

	// First delivery — should be processed.
	client.handleRelayMessage("topic", encrypted, tagSessionPropose)
	// Second delivery (same msgID) — must be ignored.
	client.handleRelayMessage("topic", encrypted, tagSessionPropose)

	require.Equal(t, 1, callCount, "handler must be called exactly once for duplicate wc_sessionPropose")

	client.mu.Lock()
	_, exists := client.pendingProposals["100"]
	client.mu.Unlock()
	require.True(t, exists)
}

// --- wc_sessionRequest deduplication ---

func TestClient_HandleRelayMessage_SessionRequest_Deduplication(t *testing.T) {
	t.Run("DuplicateIDIgnored", func(t *testing.T) {
		client, _ := NewClient("test")
		addActiveSession(client, "topic", testSymKey)

		callCount := 0
		client.SetSessionRequestHandler(func(_, _ string) { callCount++ })

		encrypted := encryptMsg(t, testSymKey, "wc_sessionRequest", 456, map[string]any{
			"request": map[string]any{"method": "personal_sign"},
		})

		// First delivery — should be processed.
		client.handleRelayMessage("topic", encrypted, tagSessionRequest)
		// Second delivery (same msgID) — must be ignored.
		client.handleRelayMessage("topic", encrypted, tagSessionRequest)

		require.Equal(t, 1, callCount, "handler must be called exactly once for duplicate wc_sessionRequest")

		client.mu.Lock()
		tracked := client.pendingSessionRequests[456]
		client.mu.Unlock()
		require.NotEmpty(t, tracked)
	})

	t.Run("DifferentIDsBothDelivered", func(t *testing.T) {
		client, _ := NewClient("test")
		addActiveSession(client, "topic", testSymKey)

		callCount := 0
		client.SetSessionRequestHandler(func(_, _ string) { callCount++ })

		enc1 := encryptMsg(t, testSymKey, "wc_sessionRequest", 1, map[string]any{"request": map[string]any{"method": "eth_sign"}})
		enc2 := encryptMsg(t, testSymKey, "wc_sessionRequest", 2, map[string]any{"request": map[string]any{"method": "personal_sign"}})

		client.handleRelayMessage("topic", enc1, tagSessionRequest)
		client.handleRelayMessage("topic", enc2, tagSessionRequest)

		require.Equal(t, 2, callCount, "distinct request IDs must each trigger the handler")
	})
}

// --- wc_sessionDelete deduplication ---

func TestClient_HandleRelayMessage_SessionDelete_Deduplicate(t *testing.T) {
	client, _ := NewClient("test")

	// Register the key in both maps: pairingTopics keeps the symKey available for
	// decryption even after activeSessions is cleared, so we can verify that the
	// deduplication guard inside the wc_sessionDelete case fires on the second call.
	addPairingTopic(client, "topic", testSymKey)
	addActiveSession(client, "topic", testSymKey)

	callCount := 0
	client.SetSessionDeleteHandler(func(_ string) { callCount++ })

	encrypted := encryptMsg(t, testSymKey, "wc_sessionDelete", 5, map[string]any{
		"code": int64(1000), "message": "disconnect",
	})

	// First delivery — session present → processed; session removed from activeSessions.
	client.handleRelayMessage("topic", encrypted, tagSessionDelete)
	// Second delivery — session already gone → dedup guard fires and handler is skipped.
	client.handleRelayMessage("topic", encrypted, tagSessionDelete)

	require.Equal(t, 1, callCount, "delete handler must be called exactly once for duplicate wc_sessionDelete")
}

func TestClient_NewClient_InitializesPendingSessionRequests(t *testing.T) {
	client, err := NewClient("test")
	require.NoError(t, err)
	require.NotNil(t, client.pendingSessionRequests)
}
