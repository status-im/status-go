package reliability

import (
	"bytes"
	"testing"
	"time"

	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
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

func TestSDSTickEnsuresChannels(t *testing.T) {
	r := newTestReliability(t)
	t.Cleanup(r.Close)

	const channelID = "sync-channel"
	require.NoError(t, r.EnsureSDSChannel(channelID))
	events, err := r.TickSDS()
	require.NoError(t, err)
	for _, event := range events {
		require.Equal(t, channelID, event.ChannelID)
	}
}

type testSDSValidatorFunc func(sds.ValidationMessage) (bool, error)

func (f testSDSValidatorFunc) Validate(message sds.ValidationMessage) (bool, error) {
	return f(message)
}

func TestSDSValidatorSurvivesManagerRestart(t *testing.T) {
	r := newTestReliability(t)
	t.Cleanup(r.Close)
	wrapped, _, err := r.WrapPayloadForSDS([]byte("payload"), "channel")
	require.NoError(t, err)
	var context []byte
	require.NoError(t, r.SetSDSValidator(testSDSValidatorFunc(func(message sds.ValidationMessage) (bool, error) {
		context = message.ValidationContext
		return false, nil
	})))
	for _, restart := range []bool{false, true} {
		if restart {
			r.Close()
			require.NoError(t, r.Start(noopDispatch))
		}
		payload, _, _, _, err := r.UnwrapPayloadFromSDSWithContext(wrapped, []byte("sender"))
		require.ErrorIs(t, err, sds.ErrMessageRejected)
		require.Nil(t, payload)
		require.Equal(t, []byte("sender"), context)
	}
	require.NoError(t, r.SetSDSValidator(nil))
	payload, err := r.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), payload)
}

func TestSDSValidationRejectsBeforeReliabilityEvents(t *testing.T) {
	receiver := newTestReliability(t)
	t.Cleanup(receiver.Close)
	peer := newTestReliability(t)
	t.Cleanup(peer.Close)
	pending, _, err := receiver.WrapPayloadForSDS([]byte("pending"), "channel")
	require.NoError(t, err)
	_, err = peer.UnwrapPayloadFromSDS(pending)
	require.NoError(t, err)
	_, _, err = peer.WrapPayloadForSDS([]byte("missing dependency"), "channel")
	require.NoError(t, err)
	wrapped, _, err := peer.WrapPayloadForSDS([]byte("unauthorized"), "channel")
	require.NoError(t, err)

	events := make(chan string, 8)
	receiver.SetMissingDependenciesHandler(func(string, []MissingDependency, string) error {
		events <- "missing dependencies"
		return nil
	})
	receiver.SetMessageSentHandler(func(string) { events <- "message sent" })
	require.NoError(t, receiver.SetSDSValidator(testSDSValidatorFunc(func(sds.ValidationMessage) (bool, error) {
		return false, nil
	})))
	_, err = receiver.UnwrapPayloadFromSDS(wrapped)
	require.ErrorIs(t, err, sds.ErrMessageRejected)
	select {
	case event := <-events:
		t.Fatalf("rejected SDS metadata emitted %s", event)
	case <-time.After(100 * time.Millisecond):
	}

	// Prove the same envelope would otherwise trigger dependency recovery.
	require.NoError(t, receiver.SetSDSValidator(nil))
	_, err = receiver.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case event := <-events:
			return event == "missing dependencies"
		default:
			return false
		}
	}, time.Second, time.Millisecond)
}

func TestSDSChannelRetrievalHintProvider(t *testing.T) {
	r := newTestReliability(t)
	t.Cleanup(r.Close)

	const (
		messageID = "message-id"
		channelID = "sync-hints"
	)
	want := []byte("retrieval-hint")
	var gotMessageID, gotChannelID string
	r.SetChannelRetrievalHintProvider(func(id, channel string) []byte {
		gotMessageID = id
		gotChannelID = channel
		return want
	})

	require.Equal(t, want, r.provideRetrievalHint(sds.MessageID(messageID), channelID))
	require.Equal(t, messageID, gotMessageID)
	require.Equal(t, channelID, gotChannelID)
}

type historyCountingPersistence struct {
	sds.PersistenceCallbacks
	updateCount int
}

func (p *historyCountingPersistence) UpdateHistory(channelID string, update sds.HistoryUpdate) error {
	p.updateCount++
	return p.PersistenceCallbacks.UpdateHistory(channelID, update)
}

func TestSDSUnwrapPersistenceErrorsRemainRetryable(t *testing.T) {
	for _, failure := range []string{"read", "metadata", "history"} {
		t.Run(failure, func(t *testing.T) {
			db := newSDSStateTestDB(t)
			require.NoError(t, mvdsmigrations.Migrate(db))
			persistence := NewSQLitePersistence(db)
			switch failure {
			case "read":
				_, err := db.Exec("ALTER TABLE sds_channel_metadata RENAME TO unavailable_metadata")
				require.NoError(t, err)
			case "metadata":
				require.NoError(t, persistence.SaveChannelMeta("channel", []byte{0xff}))
			case "history":
				require.NoError(t, persistence.UpdateHistory("channel", sds.HistoryUpdate{
					Append: []sds.HistoryMessage{{MessageID: "corrupt", Data: []byte{0xff}}},
				}))
			}
			sender := newTestReliability(t)
			t.Cleanup(sender.Close)
			wrapped, _, err := sender.WrapPayloadForSDS([]byte("payload"), "channel")
			require.NoError(t, err)
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			receiver, err := NewReliability(
				mvdsnode.NewSQLitePersistence(db), key,
				func(string, []MissingDependency, string) error { return nil },
				zap.NewNop(), persistence,
			)
			require.NoError(t, err)
			t.Cleanup(receiver.Close)

			legacy := []byte("legacy payload")
			payload, err := receiver.UnwrapPayloadFromSDS(legacy)
			require.NoError(t, err)
			require.Equal(t, legacy, payload)

			payload, err = receiver.UnwrapPayloadFromSDS(wrapped)
			require.Error(t, err)
			require.Nil(t, payload, "a failed SDS envelope must not become an application payload")

			if failure == "read" {
				_, err := db.Exec("ALTER TABLE unavailable_metadata RENAME TO sds_channel_metadata")
				require.NoError(t, err)
			}
			require.NoError(t, persistence.DropChannel("channel"))
			payload, err = receiver.UnwrapPayloadFromSDS(wrapped)
			require.NoError(t, err)
			require.Equal(t, []byte("payload"), payload)
		})
	}
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

	persistence := &historyCountingPersistence{
		PersistenceCallbacks: NewSQLitePersistence(db),
	}

	newReceiver := func() *Reliability {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		receiver, err := NewReliability(
			mvdsnode.NewSQLitePersistence(db),
			key,
			func(string, []MissingDependency, string) error { return nil },
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
	require.Equal(t, 1, persistence.updateCount)
	firstReceiver.Close()

	restartedReceiver := newReceiver()
	t.Cleanup(restartedReceiver.Close)
	unwrapped, err = restartedReceiver.UnwrapPayloadFromSDS(wrapped)
	require.NoError(t, err)
	require.Equal(t, payload, unwrapped)
	require.Equal(t, 1, persistence.updateCount, "the persisted message must take SDS's duplicate path")

	channel, err := persistence.LoadChannel(channelID)
	require.NoError(t, err)
	require.Len(t, channel.MessageHistory, 1)
}
