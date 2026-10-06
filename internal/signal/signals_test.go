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
