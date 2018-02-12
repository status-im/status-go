package signal

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeCrashEventJSONMarshalling(t *testing.T) {
	errorMsg := "TestNodeCrashEventJSONMarshallingError"
	expectedJSON := fmt.Sprintf(`{"error":"%s"}`, errorMsg)
	nodeCrashEvent := &NodeCrashEvent{
		Error: errors.New(errorMsg),
	}
	marshalled, err := json.Marshal(nodeCrashEvent)
	require.NoError(t, err)
	require.Equal(t, expectedJSON, string(marshalled))
}
