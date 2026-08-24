package signal

import (
	"encoding/json"
	"fmt"
	"sync"
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

func TestSignalHandlerConcurrentAccess(t *testing.T) {
	handler := Handler(func([]byte) {})
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SetHandler(handler)
			SendLocalNotifications(nil)
		}()
	}

	wg.Wait()
	ResetHandler()
}
