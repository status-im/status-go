package walletconnect

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
	const total = 250
	seen := make(map[int64]struct{}, total)
	for i := 0; i < total; i++ {
		if i > 0 && i%50 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
		id := payloadID()
		// 19-digit ids: id >= 10^18 also guarantees the timestamp*10^6 base layout.
		require.GreaterOrEqual(t, id, int64(1_000_000_000_000_000_000))
		_, dup := seen[id]
		require.False(t, dup, "duplicate id: %d", id)
		seen[id] = struct{}{}
	}
}

func TestJSONRPCResponse_idString(t *testing.T) {
	cases := []struct {
		name, raw, want string
	}{
		{"quoted string", `"abc123"`, "abc123"},
		{"numeric id", `999`, "999"},
		{"empty", ``, ""},
		{"single char", `"x"`, "x"},
		{"only quotes", `""`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := jsonRPCResponse{ID: json.RawMessage(tc.raw)}
			require.Equal(t, tc.want, r.idString())
		})
	}
}

func TestRelayClient_Close_NotConnected(t *testing.T) {
	client, _ := NewRelayClient("test")

	err := client.Close()
	require.NoError(t, err)
}

func TestRelayClient_ConnectAfterClose_ReturnsDisconnectRequested(t *testing.T) {
	fr := newFakeRelay(t, fakeRelayOpts{})
	client := newTestRelayClient(t, fr)

	require.NoError(t, client.Connect())
	require.NoError(t, client.Close())

	err := client.Connect()
	require.Error(t, err)
	require.ErrorContains(t, err, "disconnect requested")
}

func TestRelayClient_FreshInstance_ConnectsCleanly(t *testing.T) {
	fr := newFakeRelay(t, fakeRelayOpts{})

	a := newTestRelayClient(t, fr)
	require.NoError(t, a.Connect())
	require.NoError(t, a.Close())

	b := newTestRelayClient(t, fr)
	require.NoError(t, b.Connect())
	require.NoError(t, b.Close())
}
