package activity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

func TestEntrySwapProviderJSONRoundTrip(t *testing.T) {
	provider := pathProcessorCommon.ProcessorRelayName
	entry := Entry{
		payloadType:  ac.SimpleTransactionPT,
		transaction:  &ac.TransactionIdentity{ChainID: 1},
		swapProvider: &provider,
	}

	data, err := json.Marshal(&entry)
	require.NoError(t, err)
	require.Contains(t, string(data), `"swapProvider":"Relay"`)

	var decoded Entry
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.NotNil(t, decoded.swapProvider)
	require.Equal(t, provider, *decoded.swapProvider)

	// omitted when unset
	data, err = json.Marshal(&Entry{payloadType: ac.SimpleTransactionPT, transaction: &ac.TransactionIdentity{ChainID: 1}})
	require.NoError(t, err)
	require.NotContains(t, string(data), "swapProvider")
}
