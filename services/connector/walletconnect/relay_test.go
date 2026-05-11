package walletconnect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRelayTestWSServer starts a minimal echo-style relay endpoint for RelayClient dial tests.
func newRelayTestWSServer(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
}

func relayTestWSURL(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello world", 5, "he..."},
		{"hi", 2, "hi"},
		{"hi", 1, "h"},
		{"", 5, ""},
		{"test", 0, ""},
		{"test", 3, "tes"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		require.Equal(t, tt.expected, result)
	}
}

func TestPayloadID(t *testing.T) {
	id1 := payloadID()
	id2 := payloadID()

	require.NotEqual(t, id1, id2)
	require.Greater(t, id1, int64(0))
	require.Greater(t, id2, int64(0))

	require.Greater(t, id1, int64(1000000000000000))
}

func TestPayloadID_Format(t *testing.T) {
	id := payloadID()
	idStr := fmt.Sprintf("%d", id)
	require.GreaterOrEqual(t, len(idStr), 19)
}

func TestJSONRPCRequest_Marshal(t *testing.T) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      int64(123456),
		Method:  "test_method",
		Params:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, "2.0", decoded["jsonrpc"])
	require.Equal(t, float64(123456), decoded["id"])
	require.Equal(t, "test_method", decoded["method"])
}

func TestJSONRPCResponse_Unmarshal(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":"abc123","result":"test"}`)

	var resp jsonRPCResponse
	err := json.Unmarshal(data, &resp)
	require.NoError(t, err)
	require.Equal(t, "2.0", resp.JSONRPC)
	require.Equal(t, "abc123", resp.idString())
	require.Equal(t, json.RawMessage(`"test"`), resp.Result)
}

func TestJSONRPCResponse_UnmarshalError(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":999,"error":{"code":5000,"message":"failed"}}`)

	var resp jsonRPCResponse
	err := json.Unmarshal(data, &resp)
	require.NoError(t, err)
	require.Equal(t, "999", resp.idString())
	require.NotNil(t, resp.Error)
	require.Equal(t, 5000, resp.Error.Code)
	require.Equal(t, "failed", resp.Error.Message)
}

func TestJSONRPCNotification_Unmarshal(t *testing.T) {
	data := []byte(`{
		"jsonrpc":"2.0",
		"method":"irn_subscription",
		"params":{
			"id":"sub123",
			"data":{
				"topic":"topic1",
				"message":"msg1",
				"publishedAt":1234567890,
				"tag":1100
			}
		}
	}`)

	var notif jsonRPCNotification
	err := json.Unmarshal(data, &notif)
	require.NoError(t, err)
	require.Equal(t, "2.0", notif.JSONRPC)
	require.Equal(t, "irn_subscription", notif.Method)
	require.Equal(t, "sub123", notif.Params.ID)
	require.Equal(t, "topic1", notif.Params.Data.Topic)
	require.Equal(t, "msg1", notif.Params.Data.Message)
	require.Equal(t, int64(1234567890), notif.Params.Data.PublishedAt)
	require.Equal(t, 1100, notif.Params.Data.Tag)
}

func TestNewRelayClient(t *testing.T) {
	client, err := NewRelayClient("test-project")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, relayURL, client.url)
	require.Equal(t, "test-project", client.projectID)
	require.NotNil(t, client.auth)
	require.NotNil(t, client.pending)
	require.NotNil(t, client.logger)
}

func TestRelayClient_SetMessageHandler(t *testing.T) {
	client, _ := NewRelayClient("test")

	called := false
	handler := func(topic, message string, tag int) {
		called = true
	}

	client.SetMessageHandler(handler)

	client.mu.Lock()
	h := client.messageHandler
	client.mu.Unlock()

	require.NotNil(t, h)
	h("topic", "msg", 1)
	require.True(t, called)
}

func TestRelayClient_Close_NotConnected(t *testing.T) {
	client, _ := NewRelayClient("test")

	err := client.Close()
	require.NoError(t, err)
}

func TestRelayMessage_Unmarshal(t *testing.T) {
	data := []byte(`{
		"topic":"test-topic",
		"message":"test-message",
		"publishedAt":1234567890,
		"tag":1100
	}`)

	var msg RelayMessage
	err := json.Unmarshal(data, &msg)
	require.NoError(t, err)
	require.Equal(t, "test-topic", msg.Topic)
	require.Equal(t, "test-message", msg.Message)
	require.Equal(t, int64(1234567890), msg.PublishedAt)
	require.Equal(t, 1100, msg.Tag)
}

func TestConstants(t *testing.T) {
	require.Equal(t, "wss://relay.walletconnect.com", relayURL)
	require.Equal(t, 86400, defaultTTL)
	require.Equal(t, 1000, defaultMessageTag)
}

func TestIRNSubscriptionParams_Unmarshal(t *testing.T) {
	data := []byte(`{
		"id":"sub-id-123",
		"data":{
			"topic":"my-topic",
			"message":"my-message",
			"publishedAt":9999999,
			"tag":1108
		}
	}`)

	var params irnSubscriptionParams
	err := json.Unmarshal(data, &params)
	require.NoError(t, err)
	require.Equal(t, "sub-id-123", params.ID)
	require.Equal(t, "my-topic", params.Data.Topic)
	require.Equal(t, "my-message", params.Data.Message)
	require.Equal(t, int64(9999999), params.Data.PublishedAt)
	require.Equal(t, 1108, params.Data.Tag)
}

func TestPayloadID_UniqueAcrossCalls(t *testing.T) {
	// sleeping 1ms guarantees that batches land in different milliseconds to reduce collisions
	const batchSize = 50
	const batches = 5

	seen := make(map[int64]bool, batchSize*batches)
	for b := 0; b < batches; b++ {
		if b > 0 {
			time.Sleep(1 * time.Millisecond)
		}
		for i := 0; i < batchSize; i++ {
			id := payloadID()
			require.False(t, seen[id], "duplicate ID generated")
			seen[id] = true
		}
	}
}

func TestRelayClient_SetReconnectedHandler(t *testing.T) {
	client, _ := NewRelayClient("test")

	called := false
	handler := func() {
		called = true
	}

	client.SetReconnectedHandler(handler)

	client.mu.Lock()
	h := client.reconnectedHandler
	client.mu.Unlock()

	require.NotNil(t, h)
	h()
	require.True(t, called)
}

func TestRelayClient_DisconnectRequested(t *testing.T) {
	client, _ := NewRelayClient("test")

	require.False(t, client.disconnectRequested)

	err := client.Close()
	require.NoError(t, err)

	client.mu.Lock()
	requested := client.disconnectRequested
	client.mu.Unlock()

	require.True(t, requested)
}

func TestRelayClient_ConnectAfterClose_ReturnsDisconnectRequested(t *testing.T) {
	server := newRelayTestWSServer(t)
	defer server.Close()

	client, err := NewRelayClient("test")
	require.NoError(t, err)
	client.url = relayTestWSURL(server)

	require.NoError(t, client.Connect())
	require.NoError(t, client.Close())

	err = client.Connect()
	require.Error(t, err)
	require.ErrorContains(t, err, "disconnect requested")
}

func TestRelayClient_FreshInstance_ConnectsCleanly(t *testing.T) {
	server := newRelayTestWSServer(t)
	defer server.Close()
	wsURL := relayTestWSURL(server)

	a, err := NewRelayClient("test")
	require.NoError(t, err)
	a.url = wsURL
	require.NoError(t, a.Connect())
	require.NoError(t, a.Close())

	b, err := NewRelayClient("test")
	require.NoError(t, err)
	b.url = wsURL
	require.NoError(t, b.Connect())
	require.NoError(t, b.Close())
}
