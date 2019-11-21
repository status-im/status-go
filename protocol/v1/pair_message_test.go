package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testPairMessageBytes  = []byte(`["~#p2",["installation-id","desktop","name","token"]]`)
	testPairMessageStruct = PairMessage{
		Name:           "name",
		DeviceType:     "desktop",
		FCMToken:       "token",
		InstallationID: "installation-id",
	}
)

func TestDecodePairMessage(t *testing.T) {
	val, err := decodeTransitMessage(testPairMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t, testPairMessageStruct, val)
}

func TestEncodePairMessage(t *testing.T) {
	data, err := EncodePairMessage(testPairMessageStruct)
	require.NoError(t, err)
	// Decode it back to a struct because, for example, map encoding is non-deterministic
	// and it is not possible to compare bytes.
	val, err := decodeTransitMessage(data)
	require.NoError(t, err)
	require.EqualValues(t, testPairMessageStruct, val)
}
