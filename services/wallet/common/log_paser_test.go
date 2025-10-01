package common

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

func TestLogHopBridgeTransferSentToL2(t *testing.T) {
	eventLogTestData := `{"address":"0xb8901acb165ed027e32754e0ffe830802919727f","topics":["0x0a0607688c86ec1775abcdbab7b33a3a35a6c9cde677c9be880150c231cc6b0b","0x000000000000000000000000000000000000000000000000000000000000000a","0x000000000000000000000000d6255ae13ac335b347aa846802ad6ac39dd2543a","0x000000000000000000000000710bda329b2a6224e4b44833de30f38e7f81d564"],"data":"0x000000000000000000000000000000000000000000000000002c68af0bb14000000000000000000000000000000000000000000000000000002c29ddae06988300000000000000000000000000000000000000000000000000000000649a33b30000000000000000000000000000000000000000000000000000000000000000","blockNumber":"0x10b4baf","transactionHash":"0xfc4687f8d985ef0cc86d79a0c7abc582552b8bae0e13a90c59eeadf7e64ed569","transactionIndex":"0x48","blockHash":"0xea7f003d02a43be4dfec836803a033c0ce22ecf9a48d4a3ed3700db9c722d994","logIndex":"0xfa","removed":false}`
	var eventLog types.Log
	err := json.Unmarshal([]byte(eventLogTestData), &eventLog)
	require.NoError(t, err)

	eventType := GetEventType(&eventLog)
	require.Equal(t, HopBridgeTransferSentToL2EventType, eventType)
}

func TestLogHopBridgeTransferFromL1Completed(t *testing.T) {
	eventLogTestData := `{"address":"0x83f6244Bd87662118d96D9a6D44f09dffF14b30E","topics":["0x320958176930804eb66c2343c7343fc0367dc16249590c0f195783bee199d094","0x000000000000000000000000d6255ae13ac335b347aa846802ad6ac39dd2543a","0x000000000000000000000000710bda329b2a6224e4b44833de30f38e7f81d564"],"data":"0x000000000000000000000000000000000000000000000000002c68af0bb14000000000000000000000000000000000000000000000000000002c29ddae06988300000000000000000000000000000000000000000000000000000000649a33b30000000000000000000000000000000000000000000000000000000000000000","blockNumber":"0x64E8FF0","transactionHash":"0x67a6f407824be389e6a960ebb1ddbe7e2f24a809a86b50452ee3dc70cc708d00","transactionIndex":"0x48","blockHash":"0xea7f003d02a43be4dfec836803a033c0ce22ecf9a48d4a3ed3700db9c722d994","logIndex":"0xfa","removed":false}`
	var eventLog types.Log
	err := json.Unmarshal([]byte(eventLogTestData), &eventLog)
	require.NoError(t, err)

	eventType := GetEventType(&eventLog)
	require.Equal(t, HopBridgeTransferFromL1CompletedEventType, eventType)
}
