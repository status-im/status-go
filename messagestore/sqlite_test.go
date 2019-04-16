package messagestore

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/sqlite"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestMessageStoreIsolatedWithEnckey(t *testing.T) {
	tmpdb, err := ioutil.TempFile("", "messagestoredb")
	defer os.Remove(tmpdb.Name())
	require.NoError(t, err)
	db, err := sqlite.OpenDB(tmpdb.Name(), "testkey")
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	storeOne := NewSQLMessageStore(db, "one")
	storeTwo := NewSQLMessageStore(db, "two")

	msg := &whisper.ReceivedMessage{EnvelopeHash: common.Hash{1}, Payload: []byte{1, 2, 3}}
	require.NoError(t, storeOne.Add(msg))

	msgTwo := &whisper.ReceivedMessage{EnvelopeHash: common.Hash{2}}
	require.NoError(t, storeTwo.Add(msgTwo))

	msgs, err := storeOne.Pop()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, msg.EnvelopeHash, msgs[0].EnvelopeHash)
	require.Equal(t, msg.Payload, msgs[0].Payload)

	msgs, err = storeTwo.Pop()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, msgTwo.EnvelopeHash, msgs[0].EnvelopeHash)
}

func TestDeleteMessageByHash(t *testing.T) {
	tmpdb, err := ioutil.TempFile("", "messagestoredb")
	defer os.Remove(tmpdb.Name())
	require.NoError(t, err)
	db, err := sqlite.OpenDB(tmpdb.Name(), "testkey")
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	store := NewSQLMessageStore(db, "one")
	msg := &whisper.ReceivedMessage{EnvelopeHash: common.Hash{1}, Payload: []byte{1, 2, 3}}
	require.NoError(t, store.Add(msg))

	require.NoError(t, store.Delete(common.Hash{1}))

	msgs, err := store.Pop()
	require.NoError(t, err)
	require.Len(t, msgs, 0)
}
