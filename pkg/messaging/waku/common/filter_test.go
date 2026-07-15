package common

import (
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const testShard = "/waku/2/rs/16/32"

type FilterTestCase struct {
	f      *Filter
	id     string
	alive  bool
	msgCnt int
}

func createLogger(t *testing.T) *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	require.NoError(t, err)
	return logger
}

func generateFilter(t *testing.T, symmetric bool) (*Filter, error) {
	var f Filter
	f.Messages = NewMemoryMessageStore()

	f.PubsubTopic = "test"

	const topicNum = 8
	f.ContentTopics = make(TopicSet, topicNum)
	for i := 0; i < topicNum; i++ {
		topic := make([]byte, 4)
		_, err := crand.Read(topic) // nolint: gosec
		require.NoError(t, err)
		topic[0] = 0x01

		f.ContentTopics[BytesToTopic(topic)] = struct{}{}
	}

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	f.Src = &key.PublicKey

	if symmetric {
		f.KeySym = make([]byte, AESKeyLength)
		_, err := crand.Read(f.KeySym) // nolint: gosec
		require.NoError(t, err)
		f.SymKeyHash = crypto.Keccak256Hash(f.KeySym)
	} else {
		f.KeyAsym, err = crypto.GenerateKey()
		require.NoError(t, err)
	}

	return &f, nil
}

func generateTestCases(t *testing.T, SizeTestFilters int) []FilterTestCase {
	cases := make([]FilterTestCase, SizeTestFilters)
	for i := 0; i < SizeTestFilters; i++ {
		f, _ := generateFilter(t, true)
		cases[i].f = f
		cases[i].alive = mrand.Int()&1 == 0 // nolint: gosec
	}
	return cases
}

func TestInstallFilters(t *testing.T) {
	const SizeTestFilters = 256
	filters := NewFilters(testShard, createLogger(t))
	tst := generateTestCases(t, SizeTestFilters)

	var err error
	var j string
	for i := 0; i < SizeTestFilters; i++ {
		j, err = filters.Install(tst[i].f)
		require.NoError(t, err)

		tst[i].id = j
		require.Len(t, j, KeyIDSize*2)
	}

	for _, testCase := range tst {
		if !testCase.alive {
			filters.Uninstall(testCase.id)
		}
	}

	for _, testCase := range tst {
		fil := filters.Get(testCase.id)
		exist := fil != nil
		require.Equal(t, exist, testCase.alive)
	}
}

func TestInstallSymKeyGeneratesHash(t *testing.T) {
	filters := NewFilters(testShard, createLogger(t))
	filter, _ := generateFilter(t, true)

	// save the current SymKeyHash for comparison
	initialSymKeyHash := filter.SymKeyHash

	// ensure the SymKeyHash is invalid, for Install to recreate it
	var invalid common.Hash
	filter.SymKeyHash = invalid

	_, err := filters.Install(filter)
	require.NoError(t, err)

	for i, b := range filter.SymKeyHash {
		require.Equal(t, b, initialSymKeyHash[i])
	}
}

func TestInstallIdenticalFilters(t *testing.T) {
	filters := NewFilters(testShard, createLogger(t))
	filter1, _ := generateFilter(t, true)

	// Copy the first filter since some of its fields
	// are randomly gnerated.
	filter2 := &Filter{
		KeySym:        filter1.KeySym,
		PubsubTopic:   filter1.PubsubTopic,
		ContentTopics: filter1.ContentTopics,
		Messages:      NewMemoryMessageStore(),
	}

	_, err := filters.Install(filter1)
	require.NoError(t, err)

	_, err = filters.Install(filter2)
	require.NoError(t, err)

	recvMessage := generateCompatibleReceivedMessage(t, filter1)
	msg := recvMessage.Open(filter1)
	require.NotNil(t, msg)
}

// TestOpenVersionZeroDecodesAsV1 covers the status-go#7499 migration: an
// incoming message whose payload is WakuV1-encrypted but whose envelope is
// labeled version=0 must still be decrypted (treated the same as version=1),
// and the original envelope must be left untouched.
func TestOpenVersionZeroDecodesAsV1(t *testing.T) {
	filter, err := generateFilter(t, true)
	require.NoError(t, err)

	data := make([]byte, 20)
	_, err = crand.Read(data) // nolint: gosec
	require.NoError(t, err)

	p := &payload.Payload{
		Data: data,
		Key:  &payload.KeyInfo{Kind: payload.Symmetric, SymKey: filter.KeySym},
	}
	encoded, err := p.Encode(1) // WakuV1 encryption
	require.NoError(t, err)

	// Label the envelope version=0 while the payload stays v1-encrypted.
	var versionZero uint32
	msg := &pb.WakuMessage{
		Payload:      encoded,
		Version:      &versionZero,
		ContentTopic: maps.Keys(filter.ContentTopics)[0].ContentTopic(),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
		Meta:         []byte{},
	}
	envelope := protocol.NewEnvelope(msg, time.Now().UnixNano(), filter.PubsubTopic)
	received := NewReceivedMessage(envelope, "test")
	received.SymKeyHash = crypto.Keccak256Hash(filter.KeySym)

	opened := received.Open(filter)
	require.NotNil(t, opened, "version=0 message should be decoded as WakuV1")
	require.Equal(t, data, opened.Data)

	// We clone before overriding, so the original envelope keeps version=0.
	require.Equal(t, uint32(0), received.Envelope.Message().GetVersion())
}

func TestInstallFilterWithSymAndAsymKeys(t *testing.T) {
	filters := NewFilters(testShard, createLogger(t))
	filter1, _ := generateFilter(t, true)

	asymKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Copy the first filter since some of its fields
	// are randomly gnerated.
	filter := &Filter{
		KeySym:        filter1.KeySym,
		KeyAsym:       asymKey,
		PubsubTopic:   filter1.PubsubTopic,
		ContentTopics: filter1.ContentTopics,
		Messages:      NewMemoryMessageStore(),
	}

	_, err = filters.Install(filter)
	require.Error(t, err)
}

func cloneFilter(orig *Filter) *Filter {
	var clone Filter
	clone.Messages = NewMemoryMessageStore()
	clone.Src = orig.Src
	clone.KeyAsym = orig.KeyAsym
	clone.KeySym = orig.KeySym
	clone.PubsubTopic = orig.PubsubTopic
	clone.ContentTopics = orig.ContentTopics
	clone.SymKeyHash = orig.SymKeyHash
	return &clone
}

func generateCompatibleReceivedMessage(t *testing.T, f *Filter) *ReceivedMessage {
	keyInfo := &payload.KeyInfo{}
	keyInfo.Kind = payload.Symmetric
	keyInfo.SymKey = f.KeySym

	var version uint32 = 1
	p := new(payload.Payload)
	p.Data = make([]byte, 20)
	_, err := crand.Read(p.Data) // nolint: gosec
	require.NoError(t, err)
	p.Key = keyInfo
	payload, err := p.Encode(version)
	require.NoError(t, err)

	msg := &pb.WakuMessage{
		Payload:      payload,
		Version:      &version,
		ContentTopic: maps.Keys(f.ContentTopics)[2].ContentTopic(),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
		Meta:         []byte{},
	}
	envelope := protocol.NewEnvelope(msg, time.Now().UnixNano(), f.PubsubTopic)

	result := NewReceivedMessage(envelope, "test")
	result.SymKeyHash = crypto.Keccak256Hash(f.KeySym)

	return result
}

func TestWatchers(t *testing.T) {
	const NumFilters = 16
	const NumMessages = 256
	var i int
	var j uint32
	var e *ReceivedMessage
	var x, firstID string
	var err error

	filters := NewFilters("/waku/2/rs/16/32", createLogger(t))
	tst := generateTestCases(t, NumFilters)
	for i = 0; i < NumFilters; i++ {
		tst[i].f.Src = nil
		x, err = filters.Install(tst[i].f)
		require.NoError(t, err)

		tst[i].id = x
		if len(firstID) == 0 {
			firstID = x
		}
	}

	lastID := x

	var envelopes [NumMessages]*ReceivedMessage
	for i = 0; i < NumMessages; i++ {
		j = mrand.Uint32() % NumFilters // nolint: gosec
		e = generateCompatibleReceivedMessage(t, tst[j].f)
		envelopes[i] = e
		tst[j].msgCnt++
	}

	for i = 0; i < NumMessages; i++ {
		filters.NotifyWatchers(envelopes[i])
	}

	var total int
	var mail []*ReceivedMessage
	var count [NumFilters]int

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		count[i] = len(mail)
		total += len(mail)
	}
	require.Equal(t, total, NumMessages)

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		require.Zero(t, len(mail))
		require.Equal(t, tst[i].msgCnt, count[i])
	}

	// another round with a cloned filter

	clone := cloneFilter(tst[0].f)
	filters.Uninstall(lastID)
	total = 0
	last := NumFilters - 1
	tst[last].f = clone
	_, err = filters.Install(clone)
	require.NoError(t, err)

	for i = 0; i < NumFilters; i++ {
		tst[i].msgCnt = 0
		count[i] = 0
	}

	// make sure that the first watcher receives at least one message
	e = generateCompatibleReceivedMessage(t, tst[0].f)
	envelopes[0] = e
	tst[0].msgCnt++
	for i = 1; i < NumMessages; i++ {
		j = mrand.Uint32() % NumFilters // nolint: gosec
		e = generateCompatibleReceivedMessage(t, tst[j].f)
		envelopes[i] = e
		tst[j].msgCnt++
	}

	for i = 0; i < NumMessages; i++ {
		filters.NotifyWatchers(envelopes[i])
	}

	for i = 0; i < NumFilters; i++ {
		mail = tst[i].f.Retrieve()
		count[i] = len(mail)
		total += len(mail)
	}

	combined := tst[0].msgCnt + tst[last].msgCnt
	require.Equal(t, total, NumMessages+count[0])
	require.Equal(t, combined, count[0])
	require.Equal(t, combined, count[last])

	for i = 1; i < NumFilters-1; i++ {
		mail = tst[i].f.Retrieve()
		require.Zero(t, len(mail))
		require.Equal(t, tst[i].msgCnt, count[i])
	}
}

// newCountTestMessage builds a distinct ReceivedMessage on the given content topic;
// distinct seeds yield distinct envelope hashes (issue #21470-hf backpressure signal).
func newCountTestMessage(t *testing.T, contentTopic string, seed byte) *ReceivedMessage {
	data := make([]byte, 8)
	data[0] = seed
	msg := &pb.WakuMessage{
		Payload:      data,
		ContentTopic: contentTopic,
		Timestamp:    proto.Int64(time.Now().UnixNano() + int64(seed)),
		Meta:         []byte{},
	}
	envelope := protocol.NewEnvelope(msg, msg.GetTimestamp(), testShard)
	rm := NewReceivedMessage(envelope, "test")
	require.NotNil(t, rm)
	return rm
}

// TestMemoryMessageStoreCount verifies the store reports its pending size, does not
// double-count duplicate hashes, and returns to zero after Pop — the primitive the
// hash-first body-fetch backpressure signal is built on (issue #21470-hf).
func TestMemoryMessageStoreCount(t *testing.T) {
	filter, err := generateFilter(t, true)
	require.NoError(t, err)
	ct := maps.Keys(filter.ContentTopics)[0].ContentTopic()

	store := NewMemoryMessageStore()
	require.Equal(t, 0, store.Count())

	for i := 0; i < 3; i++ {
		require.NoError(t, store.Add(newCountTestMessage(t, ct, byte(i))))
	}
	require.Equal(t, 3, store.Count())

	// Same message added twice is stored (and counted) once.
	dup := newCountTestMessage(t, ct, 9)
	require.NoError(t, store.Add(dup))
	require.NoError(t, store.Add(dup))
	require.Equal(t, 4, store.Count())

	popped, err := store.Pop()
	require.NoError(t, err)
	require.Len(t, popped, 4)
	require.Equal(t, 0, store.Count())
}

// TestFiltersPendingMessageCount verifies the aggregate backpressure signal sums the
// pending messages across every installed filter store and drops as they are consumed
// (issue #21470-hf).
func TestFiltersPendingMessageCount(t *testing.T) {
	filters := NewFilters(testShard, createLogger(t))

	f1, err := generateFilter(t, true)
	require.NoError(t, err)
	f2, err := generateFilter(t, true)
	require.NoError(t, err)
	_, err = filters.Install(f1)
	require.NoError(t, err)
	_, err = filters.Install(f2)
	require.NoError(t, err)

	require.Equal(t, 0, filters.PendingMessageCount())

	ct1 := maps.Keys(f1.ContentTopics)[0].ContentTopic()
	require.NoError(t, f1.Messages.Add(newCountTestMessage(t, ct1, 1)))
	require.NoError(t, f1.Messages.Add(newCountTestMessage(t, ct1, 2)))
	ct2 := maps.Keys(f2.ContentTopics)[0].ContentTopic()
	require.NoError(t, f2.Messages.Add(newCountTestMessage(t, ct2, 3)))

	require.Equal(t, 3, filters.PendingMessageCount())

	_, err = f1.Messages.Pop()
	require.NoError(t, err)
	require.Equal(t, 1, filters.PendingMessageCount())
}
