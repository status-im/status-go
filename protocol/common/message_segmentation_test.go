package common

import (
	"math"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/t/helpers"
	wakutypes "github.com/status-im/status-go/waku/types"
)

func TestMessageSegmentationSuite(t *testing.T) {
	suite.Run(t, new(MessageSegmentationSuite))
}

type MessageSegmentationSuite struct {
	suite.Suite

	sender      *MessageSender
	testPayload []byte
	logger      *zap.Logger
}

func (s *MessageSegmentationSuite) SetupSuite() {
	s.testPayload = make([]byte, 1000)
	for i := 0; i < 1000; i++ {
		s.testPayload[i] = byte(i)
	}
}

func (s *MessageSegmentationSuite) SetupTest() {
	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	database, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(database)
	s.Require().NoError(err)

	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)

	s.sender, err = NewMessageSender(
		identity,
		database,
		nil,
		nil,
		s.logger,
		FeatureFlags{},
	)
	s.Require().NoError(err)
}

func (s *MessageSegmentationSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *MessageSegmentationSuite) TestHandleSegmentationLayer() {
	testCases := []struct {
		name                        string
		segmentsCount               int
		expectedParitySegmentsCount int
		retrievedSegments           []int
		retrievedParitySegments     []int
		shouldSucceed               bool
	}{
		{
			name:                        "all segments retrieved",
			segmentsCount:               2,
			expectedParitySegmentsCount: 0,
			retrievedSegments:           []int{0, 1},
			retrievedParitySegments:     []int{},
			shouldSucceed:               true,
		},
		{
			name:                        "all segments retrieved out of order",
			segmentsCount:               2,
			expectedParitySegmentsCount: 0,
			retrievedSegments:           []int{1, 0},
			retrievedParitySegments:     []int{},
			shouldSucceed:               true,
		},
		{
			name:                        "all segments&parity retrieved",
			segmentsCount:               8,
			expectedParitySegmentsCount: 1,
			retrievedSegments:           []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
			retrievedParitySegments:     []int{8},
			shouldSucceed:               true,
		},
		{
			name:                        "all segments&parity retrieved out of order",
			segmentsCount:               8,
			expectedParitySegmentsCount: 1,
			retrievedSegments:           []int{8, 0, 7, 1, 6, 2, 5, 3, 4},
			retrievedParitySegments:     []int{8},
			shouldSucceed:               true,
		},
		{
			name:                        "no segments retrieved",
			segmentsCount:               2,
			expectedParitySegmentsCount: 0,
			retrievedSegments:           []int{},
			retrievedParitySegments:     []int{},
			shouldSucceed:               false,
		},
		{
			name:                        "not all needed segments&parity retrieved",
			segmentsCount:               8,
			expectedParitySegmentsCount: 1,
			retrievedSegments:           []int{1, 2, 8},
			retrievedParitySegments:     []int{8},
			shouldSucceed:               false,
		},
		{
			name:                        "segments&parity retrieved",
			segmentsCount:               8,
			expectedParitySegmentsCount: 1,
			retrievedSegments:           []int{1, 2, 3, 4, 5, 6, 7, 8},
			retrievedParitySegments:     []int{8},
			shouldSucceed:               true, // succeed even though one segment is missing, thank you reedsolomon
		},
		{
			name:                        "segments&parity retrieved out of order",
			segmentsCount:               16,
			expectedParitySegmentsCount: 2,
			retrievedSegments:           []int{17, 0, 16, 1, 15, 2, 14, 3, 13, 4, 12, 5, 11, 6, 10, 7},
			retrievedParitySegments:     []int{16, 17},
			shouldSucceed:               true, // succeed even though two segments are missing, thank you reedsolomon
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			segmentedMessages, err := segmentMessage(&wakutypes.NewMessage{Payload: s.testPayload}, int(math.Ceil(float64(len(s.testPayload))/float64(tc.segmentsCount))))
			s.Require().NoError(err)
			s.Require().Len(segmentedMessages, tc.segmentsCount+tc.expectedParitySegmentsCount)

			message := &protocol.StatusMessage{TransportLayer: protocol.TransportLayer{
				SigPubKey: &s.sender.identity.PublicKey,
			}}

			messageRecreated := false
			handledSegments := []int{}

			for i, segmentIndex := range tc.retrievedSegments {
				s.T().Log("i=", i, "segmentIndex=", segmentIndex)

				message.TransportLayer.Payload = segmentedMessages[segmentIndex].Payload

				err = s.sender.handleSegmentationLayer(message)

				handledSegments = append(handledSegments, segmentIndex)

				if len(handledSegments) < tc.segmentsCount {
					s.Require().ErrorIs(err, ErrMessageSegmentsIncomplete)
				} else if len(handledSegments) == tc.segmentsCount {
					s.Require().NoError(err)
					s.Require().ElementsMatch(s.testPayload, message.TransportLayer.Payload)
					messageRecreated = true
				} else {
					s.Require().ErrorIs(err, ErrMessageSegmentsAlreadyCompleted)
				}
			}

			s.Require().Equal(tc.shouldSucceed, messageRecreated)
		})
	}
}
