package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/waku/types"
)

func TestMailserver_UnmarshalJSON(t *testing.T) {
	fleets := params.GetSupportedFleets()

	for _, fleet := range fleets {
		for _, mailserver := range fleet.StoreNodes {
			jsonData, err := json.Marshal(mailserver)
			require.NoError(t, err)

			var unmarshalled types.Mailserver
			err = json.Unmarshal(jsonData, &unmarshalled)
			require.NoError(t, err)

			require.Equal(t, mailserver, unmarshalled)
		}
	}
}
