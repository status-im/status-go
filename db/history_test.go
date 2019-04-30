package db

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestTopicHistoryStoreLoadFromKey(t *testing.T) {
	db, err := NewMemoryDBNamespace(TopicHistoryBucket)
	require.NoError(t, err)
	th := TopicHistory{
		db:       db,
		Topic:    whisper.TopicType{1, 1, 1},
		Duration: 10 * time.Hour,
	}
	require.NoError(t, th.Save())
	now := time.Now()
	th.Current = now
	require.NoError(t, th.Save())

	th, err = LoadTopicHistoryFromKey(db, th.Key())
	require.NoError(t, err)
	require.Equal(t, now.Unix(), th.Current.Unix())
}

func TestTopicHistorySameRange(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		description string
		result      bool
		histories   [2]TopicHistory
	}{
		{
			description: "SameDurationCurrentNotSet",
			result:      true,
			histories: [2]TopicHistory{
				{Duration: time.Minute}, {Duration: time.Minute},
			},
		},
		{
			description: "DifferentDurationCurrentNotset",
			result:      false,
			histories: [2]TopicHistory{
				{Duration: time.Minute}, {Duration: time.Hour},
			},
		},
		{
			description: "SameCurrent",
			result:      true,
			histories: [2]TopicHistory{
				{Current: now}, {Current: now},
			},
		},
		{
			description: "DifferentCurrent",
			result:      false,
			histories: [2]TopicHistory{
				{Current: now}, {Current: now.Add(time.Hour)},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			require.Equal(t, tc.result, tc.histories[0].SameRange(tc.histories[1]))
		})
	}
}

func TestAddHistory(t *testing.T) {
	topic := whisper.TopicType{1, 1, 1}
	now := time.Now()

	topicdb, err := NewMemoryDBNamespace(TopicHistoryBucket)
	require.NoError(t, err)
	requestdb, err := NewMemoryDBNamespace(HistoryRequestBucket)
	require.NoError(t, err)

	th := TopicHistory{db: topicdb, Topic: topic, Current: now}
	id := common.Hash{1}

	req := HistoryRequest{requestDB: requestdb, topicDB: topicdb, ID: id}
	req.AddHistory(th)
	require.NoError(t, req.Save())

	req = HistoryRequest{requestDB: requestdb, topicDB: topicdb, ID: id}
	require.NoError(t, req.Load())

	require.Len(t, req.Histories(), 1)
	require.Equal(t, th.Topic, req.Histories()[0].Topic)
}

func TestRequestIncludesMethod(t *testing.T) {
	topicOne := whisper.TopicType{1}
	topicTwo := whisper.TopicType{2}
	testCases := []struct {
		description string
		result      bool
		topics      []TopicHistory
		input       TopicHistory
	}{
		{
			description: "EmptyTopic",
			result:      false,
			input:       TopicHistory{Topic: topicOne},
		},
		{
			description: "MatchesTopic",
			result:      true,
			topics:      []TopicHistory{{Topic: topicOne}},
			input:       TopicHistory{Topic: topicOne},
		},
		{
			description: "NotMatchesTopic",
			result:      false,
			topics:      []TopicHistory{{Topic: topicOne}},
			input:       TopicHistory{Topic: topicTwo},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req := HistoryRequest{}
			for _, t := range tc.topics {
				req.AddHistory(t)
			}
			require.Equal(t, tc.result, req.Includes(tc.input))
		})
	}
}
