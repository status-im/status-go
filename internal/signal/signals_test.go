package signal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeCrashEventJSONMarshalling(t *testing.T) {
	errorMsg := "TestNodeCrashEventJSONMarshallingError"
	expectedJSON := fmt.Sprintf(`{"error":"%s"}`, errorMsg)
	nodeCrashEvent := &NodeCrashEvent{
		Error: errorMsg,
	}
	marshalled, err := json.Marshal(nodeCrashEvent)
	require.NoError(t, err)
	require.Equal(t, expectedJSON, string(marshalled))
}

// Mobile clients read the signal type with a prefix scan instead of parsing the whole
// envelope, so "type" must stay the first key (status-im/status-app#22641).
func TestEnvelopeMarshalsTypeFirst(t *testing.T) {
	var emitted []byte
	SetHandler(func(data []byte) { emitted = data })
	defer ResetHandler()

	send("community.found", map[string]interface{}{"type": "nested", "a": 1})

	require.True(t, bytes.HasPrefix(emitted, []byte(`{"type":"community.found",`)), string(emitted))
}

// Clients prefix-scan the epoch right after the type to reject signals of a
// previous session without parsing the event payload.
func TestEnvelopeMarshalsEpochSecond(t *testing.T) {
	var emitted []byte
	SetHandler(func(data []byte) { emitted = data })
	defer ResetHandler()

	send("community.found", map[string]interface{}{"a": 1})
	require.True(t, bytes.HasPrefix(emitted, fmt.Appendf(nil, `{"type":"community.found","epoch":%d,`, Epoch())), string(emitted))
}

func TestAdvanceEpochStampsLaterSignals(t *testing.T) {
	var emitted []byte
	SetHandler(func(data []byte) { emitted = data })
	defer ResetHandler()

	before := Epoch()
	require.NotZero(t, before)

	send("a", nil)
	var first Envelope
	require.NoError(t, json.Unmarshal(emitted, &first))
	require.Equal(t, before, first.Epoch)

	advanced := AdvanceEpoch()
	require.Equal(t, before+1, advanced)
	require.Equal(t, advanced, Epoch())

	send("b", nil)
	var second Envelope
	require.NoError(t, json.Unmarshal(emitted, &second))
	require.Equal(t, advanced, second.Epoch)
}

// Work started in a session that sends after a logout keeps that session's epoch, so
// the client can tell it from the next session's signals.
func TestSessionKeepsItsEpochAfterAdvance(t *testing.T) {
	var emitted []byte
	SetHandler(func(data []byte) { emitted = data })
	defer ResetHandler()

	session := CurrentSession()
	ended := Epoch()
	AdvanceEpoch()

	session.SendNewMessages(json.RawMessage(`{}`))
	var late Envelope
	require.NoError(t, json.Unmarshal(emitted, &late))
	require.Equal(t, ended, late.Epoch)

	SendNewMessages(json.RawMessage(`{}`))
	var current Envelope
	require.NoError(t, json.Unmarshal(emitted, &current))
	require.Equal(t, Epoch(), current.Epoch)
}

func TestZeroSessionSendsInTheCurrentSession(t *testing.T) {
	var emitted []byte
	SetHandler(func(data []byte) { emitted = data })
	defer ResetHandler()

	AdvanceEpoch()
	Session{}.SendNewMessages(json.RawMessage(`{}`))
	var envelope Envelope
	require.NoError(t, json.Unmarshal(emitted, &envelope))
	require.Equal(t, Epoch(), envelope.Epoch)
}

// The app reads these signals by name and by key.
func TestCuratedCommunitiesRefreshSignals(t *testing.T) {
	var emitted map[string]json.RawMessage
	SetHandler(func(data []byte) {
		emitted = nil
		require.NoError(t, json.Unmarshal(data, &emitted))
	})
	defer ResetHandler()

	SendCuratedCommunitiesRefreshStarted(7, true, false)
	require.Equal(t, `"curated.communities.refresh.started"`, string(emitted["type"]))
	require.JSONEq(t, `{"runId":7,"changed":true,"directoryFailed":false}`, string(emitted["event"]))

	SendCuratedCommunitiesRefreshStarted(8, false, true)
	require.JSONEq(t, `{"runId":8,"changed":false,"directoryFailed":true}`, string(emitted["event"]))

	SendCuratedCommunitiesRefreshFinished(8, 6, 4, 3, true)
	require.Equal(t, `"curated.communities.refresh.finished"`, string(emitted["type"]))
	require.JSONEq(t, `{"runId":8,"resolved":6,"notFound":4,"failed":3,"cancelled":true}`, string(emitted["event"]))
}
