package protocol

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
)

type queryResponse struct {
	topics []types.TopicType
	err    error // Indicates if this response will simulate an error returned by SendMessagesRequestForTopics
	cursor []byte
}

type mockTransport struct {
	queryResponses map[string]queryResponse
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		queryResponses: make(map[string]queryResponse),
	}
}

func getInitialResponseKey(topics []types.TopicType) string {
	return hex.EncodeToString(append([]byte("start"), topics[0][:]...))
}

func (t *mockTransport) SendMessagesRequestForTopics(
	ctx context.Context,
	peerID []byte,
	from, to uint32,
	previousCursor []byte,
	previousStoreCursor *types.StoreRequestCursor,
	pubsubTopic string,
	contentTopics []types.TopicType,
	waitForResponse bool,
) (cursor []byte, storeCursor *types.StoreRequestCursor, err error) {
	var response queryResponse
	if previousCursor == nil {
		initialResponse := getInitialResponseKey(contentTopics)
		response = t.queryResponses[initialResponse]
	} else {
		response = t.queryResponses[hex.EncodeToString(previousCursor)]
	}
	return response.cursor, nil, response.err
}

func (t *mockTransport) Populate(topics []types.TopicType, responses int, includeRandomError bool) error {
	if responses <= 0 || len(topics) == 0 {
		return errors.New("invalid input parameters")
	}

	var topicBatches [][]types.TopicType

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
			topics: topicBatch,
			err:    nil,
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
				topics: topicBatch,
				err:    err,
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

func TestProcessMailserverBatchHappyPath(t *testing.T) {
	logger := tt.MustCreateTestLogger()

	mailserverID := []byte{1, 2, 3, 4, 5}
	topics := []types.TopicType{}
	for i := 0; i < 22; i++ {
		topics = append(topics, types.BytesToTopic([]byte{0, 0, 0, byte(i)}))
	}

	testTransport := newMockTransport()
	err := testTransport.Populate(topics, 10, false)
	require.NoError(t, err)

	testBatch := MailserverBatch{
		Topics: topics,
	}

	err = processMailserverBatch(context.TODO(), testTransport, testBatch, mailserverID, logger)
	require.NoError(t, err)
}

func TestProcessMailserverBatchFailure(t *testing.T) {
	logger := tt.MustCreateTestLogger()

	mailserverID := []byte{1, 2, 3, 4, 5}
	topics := []types.TopicType{}
	for i := 0; i < 5; i++ {
		topics = append(topics, types.BytesToTopic([]byte{0, 0, 0, byte(i)}))
	}

	testTransport := newMockTransport()
	err := testTransport.Populate(topics, 4, true)
	require.NoError(t, err)

	testBatch := MailserverBatch{
		Topics: topics,
	}

	err = processMailserverBatch(context.TODO(), testTransport, testBatch, mailserverID, logger)
	require.Error(t, err)
}
