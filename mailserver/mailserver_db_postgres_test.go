// In order to run these tests, you must run a PostgreSQL database.
//
// Using Docker:
//   docker run --name mailserver-db -e POSTGRES_HOST_AUTH_METHOD=trust -d -p 5432:5432 postgres:9.6-alpine
//

package mailserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	waku "github.com/status-im/status-go/waku/common"
)

func TestPostgresDB_BuildIteratorWithBloomFilter(t *testing.T) {
	topic := []byte{0xaa, 0xbb, 0xcc, 0xdd}

	db, err := NewPostgresDB("postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable")
	require.NoError(t, err)

	envelope, err := newTestEnvelope(topic)
	require.NoError(t, err)
	err = db.SaveEnvelope(envelope)
	require.NoError(t, err)

	iter, err := db.BuildIterator(CursorQuery{
		start: NewDBKey(uint32(time.Now().Add(-time.Hour).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		end:   NewDBKey(uint32(time.Now().Add(time.Second).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		bloom: types.TopicToBloom(types.BytesToTopic(topic)),
		limit: 10,
	})
	require.NoError(t, err)
	hasNext := iter.Next()
	require.True(t, hasNext)
	rawValue, err := iter.GetEnvelope(nil)
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

func TestPostgresDB_BuildIteratorWithTopic(t *testing.T) {
	topic := []byte{0x01, 0x02, 0x03, 0x04}

	db, err := NewPostgresDB("postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable")
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
	require.NoError(t, err)
	hasNext := iter.Next()
	require.True(t, hasNext)
	rawValue, err := iter.GetEnvelope(nil)
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

func newTestEnvelope(topic []byte) (types.Envelope, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	params := waku.MessageParams{
		TTL:      10,
		PoW:      2.0,
		Payload:  []byte("hello world"),
		WorkTime: 1,
		Topic:    waku.BytesToTopic(topic),
		Dst:      &privateKey.PublicKey,
	}
	message, err := waku.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	envelope, err := message.Wrap(&params, now)
	if err != nil {
		return nil, err
	}
	return gethbridge.NewWakuEnvelope(envelope), nil
}
