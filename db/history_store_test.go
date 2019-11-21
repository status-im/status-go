package db

import (
	"testing"
	"time"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
	"github.com/stretchr/testify/require"
)

func createInMemStore(t *testing.T) HistoryStore {
	db, err := NewMemoryDB()
	require.NoError(t, err)
	return NewHistoryStore(LevelDBStorage{db: db})
}

func TestGetNewHistory(t *testing.T) {
	topic := whispertypes.TopicType{1}
	duration := time.Hour
	store := createInMemStore(t)
	th, err := store.GetHistory(topic, duration)
	require.NoError(t, err)
	require.Equal(t, duration, th.Duration)
	require.Equal(t, topic, th.Topic)
}

func TestGetExistingHistory(t *testing.T) {
	topic := whispertypes.TopicType{1}
	duration := time.Hour
	store := createInMemStore(t)
	th, err := store.GetHistory(topic, duration)
	require.NoError(t, err)

	now := time.Now()
	th.Current = now
	require.NoError(t, th.Save())

	th, err = store.GetHistory(topic, duration)
	require.NoError(t, err)
	require.Equal(t, now.Unix(), th.Current.Unix())
}

func TestNewHistoryRequest(t *testing.T) {
	store := createInMemStore(t)
	id := protocol.Hash{1}
	req, err := store.GetRequest(id)
	require.Error(t, err)
	req = store.NewRequest()
	req.ID = id

	th, err := store.GetHistory(whispertypes.TopicType{1}, time.Hour)
	require.NoError(t, err)
	req.AddHistory(th)
	require.NoError(t, req.Save())

	req, err = store.GetRequest(id)
	require.NoError(t, err)
	require.Len(t, req.Histories(), 1)
}

func TestGetAllRequests(t *testing.T) {
	store := createInMemStore(t)
	idOne := protocol.Hash{1}
	idTwo := protocol.Hash{2}

	req := store.NewRequest()
	req.ID = idOne
	require.NoError(t, req.Save())

	all, err := store.GetAllRequests()
	require.NoError(t, err)
	require.Len(t, all, 1)

	req = store.NewRequest()
	req.ID = idTwo
	require.NoError(t, req.Save())

	all, err = store.GetAllRequests()
	require.NoError(t, err)
	require.Len(t, all, 2)
}
