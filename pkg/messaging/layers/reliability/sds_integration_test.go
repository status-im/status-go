package reliability

import (
	"bytes"
	"testing"

	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	"github.com/status-im/status-go/internal/crypto"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"
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

func TestSDSDuplicateIsNotReaddedAfterManagerRestart(t *testing.T) {
	db := newSDSStateTestDB(t)
	require.NoError(t, mvdsmigrations.Migrate(db))

	sender := newTestReliability(t)
	t.Cleanup(sender.Close)

	const channelID = "community-persistent-channel"
	payload := []byte("message received before restart")
	wrapped, _, err := sender.WrapPayloadForSDS(payload, channelID)
	require.NoError(t, err)

	persistence := NewSQLitePersistence(db)
	updateCount := 0
	updateHistory := persistence.UpdateHistory
	persistence.UpdateHistory = func(channelID string, update sds.HistoryUpdate) error {
		updateCount++
		return updateHistory(channelID, update)
	}

	newReceiver := func() *Reliability {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		receiver, err := NewReliability(
			mvdsnode.NewSQLitePersistence(db),
			key,
			func(string, []string, string) error { return nil },
			zap.NewNop(),
			persistence,
		)
		require.NoError(t, err)
		return receiver
	}

	firstReceiver := newReceiver()
	unwrapped, err := firstReceiver.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
	require.Equal(t, 1, updateCount)
	firstReceiver.Close()

	restartedReceiver := newReceiver()
	t.Cleanup(restartedReceiver.Close)
	unwrapped, err = restartedReceiver.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
	require.Equal(t, 1, updateCount, "the persisted message must take SDS's duplicate path")

	channel, err := persistence.LoadChannel(channelID)
	require.NoError(t, err)
	require.Len(t, channel.MessageHistory, 1)
}
