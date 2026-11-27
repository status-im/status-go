package reliability

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	mvdsnode "github.com/status-im/mvds/node"
	mvdsmigrations "github.com/status-im/mvds/persistenceutil"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/panics"
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

type recoveryTestPersistence struct {
	sds.PersistenceCallbacks
	dropErr       error
	dropCount     atomic.Int32
	alwaysCorrupt bool
	failAfterDrop bool
}

func (p *recoveryTestPersistence) LoadChannel(channelID string) (sds.ChannelData, error) {
	if p.failAfterDrop && p.dropCount.Load() != 0 {
		return sds.ChannelData{}, errors.New("injected reload failure")
	}
	if p.alwaysCorrupt {
		return sds.ChannelData{Meta: []byte{0xff}}, nil
	}
	return p.PersistenceCallbacks.LoadChannel(channelID)
}

func (p *recoveryTestPersistence) DropChannel(channelID string) error {
	p.dropCount.Add(1)
	if p.dropErr != nil {
		return p.dropErr
	}
	if p.alwaysCorrupt {
		return nil
	}
	return p.PersistenceCallbacks.DropChannel(channelID)
}

func TestSDSCorruptStateRecovery(t *testing.T) {
	for _, failure := range []string{"metadata", "history"} {
		for _, operation := range []string{"unwrap", "wrap"} {
			t.Run(failure+"/"+operation, func(t *testing.T) {
				db := newSDSStateTestDB(t)
				require.NoError(t, mvdsmigrations.Migrate(db))
				persistence := &recoveryTestPersistence{PersistenceCallbacks: NewSQLitePersistence(db)}
				require.NoError(t, persistence.SaveChannelMeta("other", []byte("unchanged")))
				switch failure {
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

				if operation == "unwrap" {
					payload, err := receiver.UnwrapPayloadFromSDS(wrapped)
					require.NoError(t, err)
					require.Equal(t, []byte("payload"), payload)
				} else {
					_, _, err := receiver.WrapPayloadForSDS([]byte("outgoing"), "channel")
					require.NoError(t, err)
				}
				require.EqualValues(t, 1, persistence.dropCount.Load())
				other, err := persistence.LoadChannel("other")
				require.NoError(t, err)
				require.Equal(t, []byte("unchanged"), other.Meta)
				channel, err := persistence.LoadChannel("channel")
				require.NoError(t, err)
				for _, message := range channel.MessageHistory {
					require.NotEqual(t, []byte{0xff}, message)
				}
				require.NotEqual(t, []byte{0xff}, channel.Meta)
			})
		}
	}
}

func TestSDSConcurrentCorruptStateRecovery(t *testing.T) {
	db := newSDSStateTestDB(t)
	require.NoError(t, mvdsmigrations.Migrate(db))
	persistence := &recoveryTestPersistence{PersistenceCallbacks: NewSQLitePersistence(db)}
	require.NoError(t, persistence.SaveChannelMeta("channel", []byte{0xff}))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	receiver, err := NewReliability(
		mvdsnode.NewSQLitePersistence(db), key,
		func(string, []MissingDependency, string) error { return nil },
		zap.NewNop(), persistence,
	)
	require.NoError(t, err)
	t.Cleanup(receiver.Close)

	sender := newTestReliability(t)
	t.Cleanup(sender.Close)
	wrapped, _, err := sender.WrapPayloadForSDS([]byte("incoming"), "channel")
	require.NoError(t, err)

	const workers = 8
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer panics.LogOnPanic()
			<-start
			if i%2 == 0 {
				_, err := receiver.UnwrapPayloadFromSDS(wrapped)
				results <- err
				return
			}
			_, _, err := receiver.WrapPayloadForSDS([]byte(fmt.Sprintf("payload-%d", i)), "channel")
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	require.Len(t, results, workers)
	for err := range results {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, persistence.dropCount.Load())
}

func TestSDSPersistenceRecoveryFailuresRemainRetryable(t *testing.T) {
	for _, failure := range []string{"read", "drop", "reload", "reload-read", "unsupported-version"} {
		t.Run(failure, func(t *testing.T) {
			db := newSDSStateTestDB(t)
			require.NoError(t, mvdsmigrations.Migrate(db))
			persistence := &recoveryTestPersistence{PersistenceCallbacks: NewSQLitePersistence(db)}
			switch failure {
			case "read":
				_, err := db.Exec("ALTER TABLE sds_channel_metadata RENAME TO unavailable_metadata")
				require.NoError(t, err)
			case "drop":
				require.NoError(t, persistence.SaveChannelMeta("channel", []byte{0xff}))
				persistence.dropErr = errors.New("injected drop failure")
			case "reload":
				persistence.alwaysCorrupt = true
			case "reload-read":
				require.NoError(t, persistence.SaveChannelMeta("channel", []byte{0xff}))
				persistence.failAfterDrop = true
			case "unsupported-version":
				require.NoError(t, persistence.SaveChannelMeta("channel", []byte{0x08, 0xe7, 0x07, 0x10, 0x00}))
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

			if failure == "read" || failure == "unsupported-version" {
				require.Zero(t, persistence.dropCount.Load())
			} else {
				require.EqualValues(t, 1, persistence.dropCount.Load())
			}
			if failure == "drop" || failure == "unsupported-version" {
				data, err := persistence.PersistenceCallbacks.LoadChannel("channel")
				require.NoError(t, err)
				require.NotEmpty(t, data.Meta)
			}
			if failure == "read" {
				_, err := db.Exec("ALTER TABLE unavailable_metadata RENAME TO sds_channel_metadata")
				require.NoError(t, err)
			}
			persistence.dropErr = nil
			persistence.alwaysCorrupt = false
			persistence.failAfterDrop = false
			if failure == "drop" || failure == "unsupported-version" {
				require.NoError(t, persistence.DropChannel("channel"))
			}
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
