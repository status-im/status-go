package shhext

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/mailserver"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createInMemStore(t *testing.T) db.HistoryStore {
	mdb, err := db.NewMemoryDB()
	require.NoError(t, err)
	return db.NewHistoryStore(mdb)
}

func TestRenewRequest(t *testing.T) {
	req := db.HistoryRequest{}
	duration := time.Hour
	req.AddHistory(db.TopicHistory{Duration: duration})

	firstNow := time.Now()
	RenewRequests([]db.HistoryRequest{req}, firstNow)

	initial := firstNow.Add(-duration).Unix()

	th := req.Histories()[0]
	require.Equal(t, initial, th.Current.Unix())
	require.Equal(t, initial, th.First.Unix())
	require.Equal(t, firstNow.Unix(), th.End.Unix())

	secondNow := time.Now()
	RenewRequests([]db.HistoryRequest{req}, secondNow)

	require.Equal(t, initial, th.Current.Unix())
	require.Equal(t, initial, th.First.Unix())
	require.Equal(t, secondNow.Unix(), th.End.Unix())
}

func TestCreateTopicOptionsFromRequest(t *testing.T) {
	req := db.HistoryRequest{}
	topic := whisper.TopicType{1}
	now := time.Now()
	req.AddHistory(db.TopicHistory{Topic: topic, Current: now, End: now})
	options := CreateTopicOptionsFromRequest(req)
	require.Len(t, options, len(req.Histories()),
		"length must be equal to the number of topic histories attached to request")
	require.Equal(t, topic, options[0].Topic)
	require.Equal(t, uint64(now.Add(-WhisperTimeAllowance).Unix()), options[0].Range.Start,
		"start of the range must be adjusted by the whisper time allowance")
	require.Equal(t, uint64(now.Unix()), options[0].Range.End)
}

func TestTopicOptionsToBloom(t *testing.T) {
	options := TopicOptions{
		{Topic: whisper.TopicType{1}, Range: Range{Start: 1, End: 10}},
		{Topic: whisper.TopicType{2}, Range: Range{Start: 3, End: 12}},
	}
	bloom := options.ToBloomFilterOption()
	require.Equal(t, uint64(3), bloom.Range.Start, "Start must be the latest Start across all options")
	require.Equal(t, uint64(12), bloom.Range.End, "End must be the latest End across all options")
	require.Equal(t, topicsToBloom(options[0].Topic, options[1].Topic), bloom.Filter)
}

func TestBloomFilterToMessageRequestPayload(t *testing.T) {
	var (
		start   uint32 = 10
		end     uint32 = 20
		filter         = []byte{1, 1, 1, 1}
		message        = mailserver.MessagesRequestPayload{
			Lower: start,
			Upper: end,
			Bloom: filter,
			Limit: 100000,
		}
		bloomOption = BloomFilterOption{
			Filter: filter,
			Range: Range{
				Start: uint64(start),
				End:   uint64(end),
			},
		}
	)
	expected, err := rlp.EncodeToBytes(message)
	require.NoError(t, err)
	payload, err := bloomOption.ToMessagesRequestPayload()
	require.NoError(t, err)
	require.Equal(t, expected, payload)
}

func TestCreateRequestsEmptyState(t *testing.T) {
	now := time.Now()
	reactor := NewHistoryUpdateReactor(
		createInMemStore(t), NewRequestsRegistry(0),
		func() time.Time { return now })
	requests, err := reactor.CreateRequests([]TopicRequest{
		{Topic: whisper.TopicType{1}, Duration: time.Hour},
		{Topic: whisper.TopicType{2}, Duration: time.Hour},
		{Topic: whisper.TopicType{3}, Duration: 10 * time.Hour},
	})
	require.NoError(t, err)
	require.Len(t, requests, 2)
	var (
		oneTopic, twoTopic db.HistoryRequest
	)
	if len(requests[0].Histories()) == 1 {
		oneTopic, twoTopic = requests[0], requests[1]
	} else {
		oneTopic, twoTopic = requests[1], requests[0]
	}
	require.Len(t, oneTopic.Histories(), 1)
	require.Len(t, twoTopic.Histories(), 2)

}

func TestCreateRequestsWithExistingRequest(t *testing.T) {
	store := createInMemStore(t)
	req := store.NewRequest()
	req.ID = common.Hash{1}
	th := store.NewHistory(whisper.TopicType{1}, time.Hour)
	req.AddHistory(th)
	require.NoError(t, req.Save())
	reactor := NewHistoryUpdateReactor(store, NewRequestsRegistry(0), time.Now)
	requests, err := reactor.CreateRequests([]TopicRequest{
		{Topic: whisper.TopicType{1}, Duration: time.Hour},
		{Topic: whisper.TopicType{2}, Duration: time.Hour},
		{Topic: whisper.TopicType{3}, Duration: time.Hour},
	})
	require.NoError(t, err)
	require.Len(t, requests, 2)

	var (
		oneTopic, twoTopic db.HistoryRequest
	)
	if len(requests[0].Histories()) == 1 {
		oneTopic, twoTopic = requests[0], requests[1]
	} else {
		oneTopic, twoTopic = requests[1], requests[0]
	}
	assert.Len(t, oneTopic.Histories(), 1)
	assert.Len(t, twoTopic.Histories(), 2)
}

func TestCreateMultiRequestsWithSameTopic(t *testing.T) {
	now := time.Now()
	reactor := NewHistoryUpdateReactor(
		createInMemStore(t), NewRequestsRegistry(0),
		func() time.Time { return now })
	topic := whisper.TopicType{1}
	requests, err := reactor.CreateRequests([]TopicRequest{
		{Topic: topic, Duration: time.Hour},
	})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	requests[0].ID = common.Hash{1}
	require.NoError(t, requests[0].Save())

	// duration changed. request wasn't finished
	requests, err = reactor.CreateRequests([]TopicRequest{
		{Topic: topic, Duration: 10 * time.Hour},
	})
	require.NoError(t, err)
	require.Len(t, requests, 2)
	longest := 0
	for i := range requests {
		r := &requests[i]
		r.ID = common.Hash{byte(i)}
		require.NoError(t, r.Save())
		require.Len(t, r.Histories(), 1)
		if r.Histories()[0].Duration == 10*time.Hour {
			longest = i
		}
	}
	require.Equal(t, requests[longest].Histories()[0].End, requests[longest^1].Histories()[0].First)

	for _, r := range requests {
		require.NoError(t, reactor.UpdateFinishedRequest(r.ID))
	}
	requests, err = reactor.CreateRequests([]TopicRequest{
		{Topic: topic, Duration: 10 * time.Hour},
	})
	require.NoError(t, err)
	require.Len(t, requests, 1)

	topics, err := reactor.store.GetHistoriesByTopic(topic)
	require.NoError(t, err)
	require.Len(t, topics, 1)
	require.Equal(t, 10*time.Hour, topics[0].Duration)
}

func TestRequestFinishedUpdate(t *testing.T) {
	store := createInMemStore(t)
	req := store.NewRequest()
	req.ID = common.Hash{1}
	now := time.Now()
	thOne := store.NewHistory(whisper.TopicType{1}, time.Hour)
	thOne.End = now
	thTwo := store.NewHistory(whisper.TopicType{2}, time.Hour)
	thTwo.End = now
	req.AddHistory(thOne)
	req.AddHistory(thTwo)
	require.NoError(t, req.Save())

	reactor := NewHistoryUpdateReactor(store, NewRequestsRegistry(0), time.Now)
	require.NoError(t, reactor.UpdateTopicHistory(thOne.Topic, now.Add(-time.Minute)))
	require.NoError(t, reactor.UpdateFinishedRequest(req.ID))
	_, err := store.GetRequest(req.ID)
	require.EqualError(t, err, "leveldb: not found")

	require.NoError(t, thOne.Load())
	require.NoError(t, thTwo.Load())
	require.Equal(t, now.Unix(), thOne.Current.Unix())
	require.Equal(t, now.Unix(), thTwo.Current.Unix())
}

func TestTopicHistoryUpdate(t *testing.T) {
	reqID := common.Hash{1}
	store := createInMemStore(t)
	request := store.NewRequest()
	request.ID = reqID
	now := time.Now()
	require.NoError(t, request.Save())
	th := store.NewHistory(whisper.TopicType{1}, time.Hour)
	th.RequestID = request.ID
	th.End = now
	require.NoError(t, th.Save())
	reactor := NewHistoryUpdateReactor(store, NewRequestsRegistry(0), time.Now)
	timestamp := now.Add(-time.Minute)

	require.NoError(t, reactor.UpdateTopicHistory(th.Topic, timestamp))
	require.NoError(t, th.Load())
	require.Equal(t, timestamp.Unix(), th.Current.Unix())

	require.NoError(t, reactor.UpdateTopicHistory(th.Topic, now))
	require.NoError(t, th.Load())
	require.Equal(t, timestamp.Unix(), th.Current.Unix())
}

func TestGroupHistoriesByRequestTimestamp(t *testing.T) {
	requests := GroupHistoriesByRequestTimespan(createInMemStore(t), []db.TopicHistory{
		{Topic: whisper.TopicType{1}, Duration: time.Hour},
		{Topic: whisper.TopicType{2}, Duration: time.Hour},
		{Topic: whisper.TopicType{3}, Duration: 2 * time.Hour},
		{Topic: whisper.TopicType{4}, Duration: 2 * time.Hour},
		{Topic: whisper.TopicType{5}, Duration: 3 * time.Hour},
		{Topic: whisper.TopicType{6}, Duration: 3 * time.Hour},
	})
	require.Len(t, requests, 3)
	for _, req := range requests {
		require.Len(t, req.Histories(), 2)
	}
}

// initial creation of the history index. no other histories in store
func TestAdjustHistoryWithNoOtherHistories(t *testing.T) {
	store := createInMemStore(t)
	th := store.NewHistory(whisper.TopicType{1}, time.Hour)
	adjusted, err := adjustRequestedHistories(store, []db.TopicHistory{th})
	require.NoError(t, err)
	require.Len(t, adjusted, 1)
	require.Equal(t, th.Topic, adjusted[0].Topic)
}

// Duration for the history index with same topic was gradually incresed:
// {Duration: 1h} {Duration: 2h} {Duration: 3h}
// But actual request wasn't sent
// So when we receive {Duration: 4h} we can merge all of them into single index
// that covers all of them e.g. {Duration: 4h}
func TestAdjustHistoryWithExistingLowerRanges(t *testing.T) {
	store := createInMemStore(t)
	topic := whisper.TopicType{1}
	histories := make([]db.TopicHistory, 3)
	i := 0
	for i = range histories {
		histories[i] = store.NewHistory(topic, time.Duration(i+1)*time.Hour)
		require.NoError(t, histories[i].Save())
	}
	i++
	th := store.NewHistory(topic, time.Duration(i+1)*time.Hour)
	adjusted, err := adjustRequestedHistories(store, []db.TopicHistory{th})
	require.NoError(t, err)
	require.Len(t, adjusted, 1)
	require.Equal(t, th.Duration, adjusted[0].Duration)

	all, err := store.GetHistoriesByTopic(topic)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, th.Duration, all[0].Duration)
}

// Precondition is based on the previous test. We have same information in the database
// but now every history index request was successfully completed. And End timstamp is set to the First of the next index.
// So, we have:
// {First: now-1h, End: now} {First: now-2h, End: now-1h} {First: now-3h: End: now-2h}
// When we want to create new request with {Duration: 4h}
// We see that there is no reason to keep all indexes and we can squash them.
func TestAdjustHistoriesWithExistingCoveredLowerRanges(t *testing.T) {
	store := createInMemStore(t)
	topic := whisper.TopicType{1}
	histories := make([]db.TopicHistory, 3)
	i := 0
	now := time.Now()
	for i = range histories {
		duration := time.Duration(i+1) * time.Hour
		prevduration := time.Duration(i) * time.Hour
		histories[i] = store.NewHistory(topic, duration)
		histories[i].First = now.Add(-duration)
		histories[i].Current = now.Add(-prevduration)
		require.NoError(t, histories[i].Save())
	}
	i++
	th := store.NewHistory(topic, time.Duration(i+1)*time.Hour)
	th.Current = now.Add(-time.Duration(i) * time.Hour)
	adjusted, err := adjustRequestedHistories(store, []db.TopicHistory{th})
	require.NoError(t, err)
	require.Len(t, adjusted, 1)
	require.Equal(t, th.Duration, adjusted[0].Duration)
}

func TestAdjustHistoryReplaceTopicWithHigherDuration(t *testing.T) {
	store := createInMemStore(t)
	topic := whisper.TopicType{1}
	hour := store.NewHistory(topic, time.Hour)
	require.NoError(t, hour.Save())
	minute := store.NewHistory(topic, time.Minute)
	adjusted, err := adjustRequestedHistories(store, []db.TopicHistory{minute})
	require.NoError(t, err)
	require.Len(t, adjusted, 1)
	require.Equal(t, hour.Duration, adjusted[0].Duration)
}

// if client requested lower duration than the one we have in the index already it will
// it will be discarded and we will use existing index
func TestAdjustHistoryRemoveTopicIfPendingWithHigherDuration(t *testing.T) {
	store := createInMemStore(t)
	topic := whisper.TopicType{1}
	hour := store.NewHistory(topic, time.Hour)
	hour.RequestID = common.Hash{1}
	require.NoError(t, hour.Save())
	minute := store.NewHistory(topic, time.Minute)
	adjusted, err := adjustRequestedHistories(store, []db.TopicHistory{minute})
	require.NoError(t, err)
	require.Len(t, adjusted, 0)
}
