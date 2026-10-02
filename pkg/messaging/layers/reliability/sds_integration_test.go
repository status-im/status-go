package reliability

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapPayloadForSDSIntegrationRoundTrip(t *testing.T) {
	r := newTestReliability(t)

	payload := []byte("community-sds-integration-payload")
	channelID := "community123general"

	wrapped, _, err := r.WrapPayloadForSDS(payload, channelID)
	require.NoError(t, err)
	require.NotEmpty(t, wrapped)
	require.False(t, bytes.Equal(payload, wrapped), "SDS wrap should change the payload bytes")

	unwrapped, err := r.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
}

func TestWrapPayloadForSDSIntegrationDistinctPerChannel(t *testing.T) {
	r := newTestReliability(t)

	payload := []byte("same-payload")
	channelA := "community-a-general"
	channelB := "community-b-general"

	wrappedA, _, err := r.WrapPayloadForSDS(payload, channelA)
	require.NoError(t, err)
	wrappedB, _, err := r.WrapPayloadForSDS(payload, channelB)
	require.NoError(t, err)

	require.NotEqual(t, wrappedA, wrappedB, "SDS wrap should be channel-specific")
}
