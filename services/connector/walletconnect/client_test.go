package walletconnect

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("test-project-id")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.relay)
	require.NotNil(t, client.handlers)
	require.NotNil(t, client.pendingProposals)
	require.NotNil(t, client.pairingTopics)
	require.NotNil(t, client.activeSessions)
}

func TestClient_SetSessionProposalHandler(t *testing.T) {
	client, _ := NewClient("test")

	called := false
	handler := func(proposalJSON string) {
		called = true
	}

	client.SetSessionProposalHandler(handler)

	client.mu.Lock()
	h := client.handlers.onSessionProposal
	client.mu.Unlock()

	h("test")
	require.True(t, called)
}

func TestClient_SetSessionRequestHandler(t *testing.T) {
	client, _ := NewClient("test")

	called := false
	handler := func(topic, requestJSON string) {
		called = true
	}

	client.SetSessionRequestHandler(handler)

	client.mu.Lock()
	h := client.handlers.onSessionRequest
	client.mu.Unlock()

	h("topic", "request")
	require.True(t, called)
}

func TestClient_GetSymKeyForTopic_PairingTopic(t *testing.T) {
	client, _ := NewClient("test")

	client.mu.Lock()
	client.pairingTopics["topic1"] = "key1"
	client.mu.Unlock()

	key, ok := client.getSymKeyForTopic("topic1")
	require.True(t, ok)
	require.Equal(t, "key1", key)
}

func TestClient_GetSymKeyForTopic_SessionTopic(t *testing.T) {
	client, _ := NewClient("test")

	client.mu.Lock()
	client.activeSessions["topic2"] = "key2"
	client.mu.Unlock()

	key, ok := client.getSymKeyForTopic("topic2")
	require.True(t, ok)
	require.Equal(t, "key2", key)
}

func TestClient_GetSymKeyForTopic_NotFound(t *testing.T) {
	client, _ := NewClient("test")

	key, ok := client.getSymKeyForTopic("nonexistent")
	require.False(t, ok)
	require.Empty(t, key)
}

func TestClient_GetSymKeyForTopic_SessionPriority(t *testing.T) {
	client, _ := NewClient("test")

	client.mu.Lock()
	client.pairingTopics["topic"] = "pairing-key"
	client.activeSessions["topic"] = "session-key"
	client.mu.Unlock()

	key, ok := client.getSymKeyForTopic("topic")
	require.True(t, ok)
	require.Equal(t, "session-key", key)
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

	symKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	client.mu.Lock()
	client.pairingTopics["topic"] = symKey
	client.mu.Unlock()

	client.handleRelayMessage("topic", "invalid-base64", 1100)
}

func TestClient_HandleRelayMessage_SessionProposal(t *testing.T) {
	client, _ := NewClient("test")

	symKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	client.mu.Lock()
	client.pairingTopics["topic"] = symKey
	client.mu.Unlock()

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
	encryptedProposal, _ := EncryptType0Envelope(symKey, proposalJSON)

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

	symKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	client.mu.Lock()
	client.activeSessions["topic"] = symKey
	client.mu.Unlock()

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
	encryptedRequest, _ := EncryptType0Envelope(symKey, requestJSON)

	client.handleRelayMessage("topic", encryptedRequest, tagSessionRequest)

	time.Sleep(10 * time.Millisecond)
	require.Equal(t, "topic", receivedTopic)
	require.NotEmpty(t, receivedRequest)
}

// FIXME we need to mock client.relay as it doesn't have a real connection in this test
// func TestClient_RejectSession(t *testing.T) {
// 	client, _ := NewClient("test")

// 	symKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
// 	client.mu.Lock()
// 	client.pendingProposals["123"] = &pairingContext{
// 		PairingTopic:  "pairing-topic",
// 		PairingSymKey: symKey,
// 		JsonRpcID:     123,
// 	}
// 	client.mu.Unlock()

// 	err := client.RejectSession("123")
// 	require.NoError(t, err)

// 	client.mu.Lock()
// 	_, exists := client.pendingProposals["123"]
// 	client.mu.Unlock()
// 	require.False(t, exists)
// }

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
