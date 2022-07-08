package mailserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/eth-node/types"
	waku "github.com/status-im/status-go/waku/common"
)

func TestLevelDB_BuildIteratorWithTopic(t *testing.T) {
	topic := []byte{0x01, 0x02, 0x03, 0x04}

	db, err := NewLevelDB(t.TempDir())
	require.NoError(t, err)

	envelope, err := newTestEnvelope(topic)
	require.NoError(t, err)
	err = db.SaveEnvelope(envelope)
	require.NoError(t, err)

	iter, err := db.BuildIterator(CursorQuery{
		start:  NewDBKey(uint32(time.Now().Add(-time.Hour).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		end:    NewDBKey(uint32(time.Now().Add(time.Second).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		topics: [][]byte{topic},
		limit:  10,
	})
	topicsMap := make(map[types.TopicType]bool)
	topicsMap[types.BytesToTopic(topic)] = true
	require.NoError(t, err)
	hasNext := iter.Next()
	require.True(t, hasNext)
	rawValue, err := iter.GetEnvelopeByTopicsMap(topicsMap)
	require.NoError(t, err)
	require.NotEmpty(t, rawValue)
	var receivedEnvelope waku.Envelope
	err = rlp.DecodeBytes(rawValue, &receivedEnvelope)
	require.NoError(t, err)
	require.EqualValues(t, waku.BytesToTopic(topic), receivedEnvelope.Topic)

	err = iter.Release()
	require.NoError(t, err)
	require.NoError(t, iter.Error())
}
