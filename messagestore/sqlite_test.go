package messagestore

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestMessageStore(t *testing.T) {
	tmpdb, err := ioutil.TempFile("", "messagestoredb")
	defer os.Remove(tmpdb.Name())
	require.NoError(t, err)
	db, err := sqlite.OpenDB(tmpdb.Name(), "testkey")
	require.NoError(t, err)
	store, err := InitializeSQLMessageStore(db)
	require.NoError(t, err)
	msg := &whisperv6.ReceivedMessage{EnvelopeHash: common.Hash{1}, Payload: []byte{1, 2, 3}}
	require.NoError(t, store.Add(msg))
	msgs, err := store.Pop()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, msg.EnvelopeHash, msgs[0].EnvelopeHash)
	require.Equal(t, msg.Payload, msgs[0].Payload)
}
