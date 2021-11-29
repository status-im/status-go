// In order to run these tests, you must run a PostgreSQL database.
//
// Using Docker:
//   docker run -e POSTGRES_HOST_AUTH_METHOD=trust -d -p 5432:5432 postgres:9.6-alpine
//

//nolint // TODO Fix test
package mailserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/rlp"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/postgres"
	waku "github.com/status-im/status-go/waku/common"
)

func TestMailServerPostgresDBSuite(t *testing.T) {
	// TODO Fix test
	t.Skip("Skipped")
	suite.Run(t, new(MailServerPostgresDBSuite))
}

type MailServerPostgresDBSuite struct {
	suite.Suite
}

func (s *MailServerPostgresDBSuite) SetupSuite() {
	// ResetDefaultTestPostgresDB Required to completely reset the Postgres DB
	err := postgres.ResetDefaultTestPostgresDB()
	s.NoError(err)
}

func (s *MailServerPostgresDBSuite) TestPostgresDB_BuildIteratorWithBloomFilter() {
	topic := []byte{0xaa, 0xbb, 0xcc, 0xdd}

	db, err := NewPostgresDB(postgres.DefaultTestURI)
	s.NoError(err)
	defer db.Close()

	envelope, err := newTestEnvelope(topic)
	s.NoError(err)
	err = db.SaveEnvelope(envelope)
	s.NoError(err)

	iter, err := db.BuildIterator(CursorQuery{
		start: NewDBKey(uint32(time.Now().Add(-time.Hour).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		end:   NewDBKey(uint32(time.Now().Add(time.Second).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		bloom: types.TopicToBloom(types.BytesToTopic(topic)),
		limit: 10,
	})
	s.NoError(err)
	hasNext := iter.Next()
	s.True(hasNext)
	rawValue, err := iter.GetEnvelopeByBloomFilter(nil)
	s.NoError(err)
	s.NotEmpty(rawValue)
	var receivedEnvelope waku.Envelope
	err = rlp.DecodeBytes(rawValue, &receivedEnvelope)
	s.NoError(err)
	s.EqualValues(waku.BytesToTopic(topic), receivedEnvelope.Topic)

	err = iter.Release()
	s.NoError(err)
	s.NoError(iter.Error())
}

func (s *MailServerPostgresDBSuite) TestPostgresDB_BuildIteratorWithTopic() {
	topic := []byte{0x01, 0x02, 0x03, 0x04}

	db, err := NewPostgresDB(postgres.DefaultTestURI)
	s.NoError(err)
	defer db.Close()

	envelope, err := newTestEnvelope(topic)
	s.NoError(err)
	err = db.SaveEnvelope(envelope)
	s.NoError(err)

	iter, err := db.BuildIterator(CursorQuery{
		start:  NewDBKey(uint32(time.Now().Add(-time.Hour).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		end:    NewDBKey(uint32(time.Now().Add(time.Second).Unix()), types.BytesToTopic(topic), types.Hash{}).Bytes(),
		topics: [][]byte{topic},
		limit:  10,
	})
	s.NoError(err)
	hasNext := iter.Next()
	s.True(hasNext)
	rawValue, err := iter.GetEnvelopeByBloomFilter(nil)
	s.NoError(err)
	s.NotEmpty(rawValue)
	var receivedEnvelope waku.Envelope
	err = rlp.DecodeBytes(rawValue, &receivedEnvelope)
	s.NoError(err)
	s.EqualValues(waku.BytesToTopic(topic), receivedEnvelope.Topic)

	err = iter.Release()
	s.NoError(err)
	s.NoError(iter.Error())
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
