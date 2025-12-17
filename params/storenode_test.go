package params_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestStoreNode_UnmarshalJSON(t *testing.T) {
	fleets := params.GetSupportedFleets()

	for _, fleet := range fleets {
		for _, storeNode := range fleet.StoreNodes {
			jsonData, err := json.Marshal(storeNode)
			require.NoError(t, err)

			var unmarshalled messagingtypes.StoreNode
			err = json.Unmarshal(jsonData, &unmarshalled)
			require.NoError(t, err)

			require.Equal(t, storeNode, unmarshalled)
		}
	}
}
