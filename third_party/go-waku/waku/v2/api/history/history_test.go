package history

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	proto_pb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/utils"
)

type queryResponse struct {
	contentTopics []string
	messages      []*pb.WakuMessageKeyValue
	err           error // Indicates if this response will simulate an error returned by SendMessagesRequestForTopics
	cursor        []byte
}

type mockResult struct {
	cursor   []byte
	messages []*pb.WakuMessageKeyValue
}

func (r *mockResult) Cursor() []byte {
	return r.cursor
}

func (r *mockResult) Messages() []*pb.WakuMessageKeyValue {
	return r.messages
}

func (r *mockResult) IsComplete() bool {
	return false
}

func (r *mockResult) PeerID() peer.ID {
	return ""
}

func (r *mockResult) Query() *pb.StoreQueryRequest {
	return nil
}

func (r *mockResult) Response() *pb.StoreQueryResponse {
	return nil
}

func (r *mockResult) Next(ctx context.Context, opts ...store.RequestOption) error {
	return nil
}

type mockHistoryProcessor struct {
}

func (h *mockHistoryProcessor) OnEnvelope(env *protocol.Envelope, processEnvelopes bool) error {
	return nil
}

func (h *mockHistoryProcessor) OnRequestFailed(requestID []byte, peerInfo peer.AddrInfo, err error) {
}

func newMockHistoryProcessor() *mockHistoryProcessor {
	return &mockHistoryProcessor{}
}

type mockStore struct {
	queryResponses map[string]queryResponse
}

func newMockStore() *mockStore {
	return &mockStore{
		queryResponses: make(map[string]queryResponse),
	}
}

func getInitialResponseKey(contentTopics []string) string {
	sort.Strings(contentTopics)
	return hex.EncodeToString(append([]byte("start"), []byte(contentTopics[0])...))
}

func (t *mockStore) Query(ctx context.Context, peerInfo peer.AddrInfo, storeQueryRequest *pb.StoreQueryRequest) (common.StoreRequestResult, error) {
	result := &mockResult{}
	if len(storeQueryRequest.GetPaginationCursor()) == 0 {
		initialResponse := getInitialResponseKey(storeQueryRequest.GetContentTopics())
		response := t.queryResponses[initialResponse]
		if response.err != nil {
			return nil, response.err
		}
		result.cursor = response.cursor
		result.messages = response.messages
	} else {
		response := t.queryResponses[hex.EncodeToString(storeQueryRequest.GetPaginationCursor())]
		if response.err != nil {
			return nil, response.err
		}
		result.cursor = response.cursor
		result.messages = response.messages
	}

	return result, nil
}

func (t *mockStore) Populate(topics []string, responses int, includeRandomError bool) error {
	if responses <= 0 || len(topics) == 0 {
		return errors.New("invalid input parameters")
	}

	var topicBatches [][]string

	for i := 0; i < len(topics); i += maxTopicsPerRequest {
		// Split batch in 10-contentTopic subbatches
		j := i + maxTopicsPerRequest
		if j > len(topics) {
			j = len(topics)
		}
		topicBatches = append(topicBatches, topics[i:j])
	}

	randomErrIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(topicBatches))))
	if err != nil {
		return err
	}
	randomErrIdxInt := int(randomErrIdx.Int64())

	for i, topicBatch := range topicBatches {
		// Setup initial response
		initialResponseKey := getInitialResponseKey(topicBatch)
		t.queryResponses[initialResponseKey] = queryResponse{
			contentTopics: topicBatch,
			messages: []*pb.WakuMessageKeyValue{
				{
					MessageHash: protocol.GenerateRequestID(),
					Message: &proto_pb.WakuMessage{
						Payload:      []byte{1, 2, 3},
						ContentTopic: "abc",
						Timestamp:    proto.Int64(time.Now().UnixNano()),
					},
					PubsubTopic: proto.String("test"),
				},
			},
			err: nil,
		}

		prevKey := initialResponseKey
		for x := 0; x < responses-1; x++ {
			newResponseCursor := []byte(uuid.New().String())
			newResponseKey := hex.EncodeToString(newResponseCursor)

			var err error
			if includeRandomError && i == randomErrIdxInt && x == responses-2 { // Include an error in last request
				err = errors.New("random error")
			}

			t.queryResponses[newResponseKey] = queryResponse{
				contentTopics: topicBatch,
				messages: []*pb.WakuMessageKeyValue{
					{
						MessageHash: protocol.GenerateRequestID(),
						Message: &proto_pb.WakuMessage{
							Payload:      []byte{1, 2, 3},
							ContentTopic: "abc",
							Timestamp:    proto.Int64(time.Now().UnixNano()),
						},
						PubsubTopic: proto.String("test"),
					},
				},
				err: err,
			}

			// Updating prev response cursor to point to the new response
			prevResponse := t.queryResponses[prevKey]
			prevResponse.cursor = newResponseCursor
			t.queryResponses[prevKey] = prevResponse

			prevKey = newResponseKey
		}

	}

	return nil
}

func TestSuccessBatchExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	storenodeID, err := peer.Decode("16Uiu2HAkw3x97MbbZSWHbdF5bob45vcZvPPK4s4Mjyv2mxyB9GS3")
	require.NoError(t, err)

	topics := []string{}
	for i := 0; i < 50; i++ {
		topics = append(topics, uuid.NewString())
	}

	testStore := newMockStore()
	err = testStore.Populate(topics, 10, false)
	require.NoError(t, err)

	historyProcessor := newMockHistoryProcessor()

	historyRetriever := NewHistoryRetriever(testStore, historyProcessor, utils.Logger())

	criteria := store.FilterCriteria{
		ContentFilter: protocol.NewContentFilter("test", topics...),
	}

	err = historyRetriever.Query(ctx, criteria, peer.AddrInfo{ID: storenodeID}, 10, func(i int) (bool, uint64) { return true, 10 }, true)
	require.NoError(t, err)
}

func TestFailedBatchExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	storenodeID, err := peer.Decode("16Uiu2HAkw3x97MbbZSWHbdF5bob45vcZvPPK4s4Mjyv2mxyB9GS3")
	require.NoError(t, err)

	topics := []string{}
	for i := 0; i < 2; i++ {
		topics = append(topics, uuid.NewString())
	}

	testStore := newMockStore()
	err = testStore.Populate(topics, 10, true)
	require.NoError(t, err)

	historyProcessor := newMockHistoryProcessor()

	historyRetriever := NewHistoryRetriever(testStore, historyProcessor, utils.Logger())

	criteria := store.FilterCriteria{
		ContentFilter: protocol.NewContentFilter("test", topics...),
	}

	err = historyRetriever.Query(ctx, criteria, peer.AddrInfo{ID: storenodeID}, 10, func(i int) (bool, uint64) { return true, 10 }, true)
	require.Error(t, err)
}
