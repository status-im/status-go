package segmentation

import (
	_ "embed"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation/migrations"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation/protobuf"
)

func TestMessageSegmentationSuite(t *testing.T) {
	suite.Run(t, new(MessageSegmentationSuite))
}

type MessageSegmentationSuite struct {
	suite.Suite

	segmenter   *Segmenter
	testPayload []byte
}

func (s *MessageSegmentationSuite) SetupSuite() {
	s.testPayload = make([]byte, 1000)
	for i := 0; i < 1000; i++ {
		s.testPayload[i] = byte(i)
	}
}

func (s *MessageSegmentationSuite) SetupTest() {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     migrations.AssetNames(),
			AssetFunc: migrations.Asset,
		},
	}))
	s.Require().NoError(err)

	s.segmenter = NewSegmenter(
		NewSQLitePersistence(db),
		zap.Must(zap.NewDevelopment()),
	)
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
			signer, err := crypto.GenerateKey()
			s.Require().NoError(err)

			segSize := (len(s.testPayload) + tc.segmentsCount - 1) / tc.segmentsCount // ceil[len(testPayload)/segmentsCount]
			segmentedMessages, err := s.segmenter.Segment(s.testPayload, segSize)
			s.Require().NoError(err)
			s.Require().Len(segmentedMessages, tc.segmentsCount+tc.expectedParitySegmentsCount)

			messageRecreated := false
			handledSegments := []int{}

			for i, segmentIndex := range tc.retrievedSegments {
				s.T().Log("i=", i, "segmentIndex=", segmentIndex)

				reconstructedPayload, _, err := s.segmenter.Reconstruct(segmentedMessages[segmentIndex], &signer.PublicKey, nil)
				handledSegments = append(handledSegments, segmentIndex)

				if len(handledSegments) < tc.segmentsCount {
					s.Require().ErrorIs(err, ErrIncomplete)
				} else if len(handledSegments) == tc.segmentsCount {
					s.Require().NoError(err)
					s.Require().ElementsMatch(s.testPayload, reconstructedPayload)
					messageRecreated = true
				} else {
					s.Require().ErrorIs(err, ErrAlreadyCompleted)
				}
			}

			s.Require().Equal(tc.shouldSucceed, messageRecreated)
		})
	}
}

//go:embed testdata/segmentationProtobufMissDecoding.bin
var protobufMissDecodingPayload []byte // Represents a payload that is intentionally not encoded as protobuf.SegmentMessage to test unmarshalling behavior.

func (s *MessageSegmentationSuite) TestProtobufMissDecoding() {
	// This test demonstrates how protobuf unmarshalling behaves when given a payload
	// that is not encoded as a protobuf.SegmentMessage. Protobuf attempts to decode
	// any byte sequence, and if the structure coincidentally matches valid encoding
	// patterns (e.g., varint or byte fields), it produces seemingly valid but incorrect results.

	segmentedMessage := Message{
		SegmentMessage: &protobuf.SegmentMessage{},
	}

	// Attempt to unmarshal the invalid payload into a protobuf.SegmentMessage.
	err := proto.Unmarshal(protobufMissDecodingPayload, segmentedMessage.SegmentMessage)
	s.Require().NoError(err) // Surprisingly, no error is returned.

	// Validate the unmarshalled data. The SegmentsCount field contains a value,
	// but it is incorrect because the payload was not properly encoded.
	s.Require().Equal(segmentedMessage.SegmentsCount, uint32(25)) // Incorrect but "valid" value.

	// Ensure that the sanity check for the segmented message fails, as expected.
	s.Require().False(segmentedMessage.IsValid())
}

// TestReedSolomonLastSegmentRecoveryHashMismatch reproduces issue #7444.
// When the last data segment is missing and recovered via Reed-Solomon,
// the recovered segment includes trailing zero-padding that causes a hash mismatch.
// This test FAILS until the bug is fixed.
func (s *MessageSegmentationSuite) TestReedSolomonLastSegmentRecoveryHashMismatch() {
	signer, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Create a payload that is NOT a multiple of segmentSize.
	// We need enough segments to trigger parity generation (parity rate = 0.125).
	// With 10 segments, we get floor(10 * 0.125) = 1 parity segment.
	segmentSize := 100
	payloadSize := 950 // 10 segments: 9×100 + 1×50 bytes
	payload := make([]byte, payloadSize)
	for i := 0; i < payloadSize; i++ {
		payload[i] = byte(i % 256)
	}

	// Segment the payload (this will create 10 data segments + 1 parity segment).
	segmentedMessages, err := s.segmenter.Segment(payload, segmentSize)
	s.Require().NoError(err)
	s.Require().Len(segmentedMessages, 11) // 10 data + 1 parity

	// Receive all segments EXCEPT the last data segment (index 9).
	// We receive segments at indices 0-8 and 10 (parity).
	segmentsToReceive := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 10} // Missing segment 9 (the last data segment)

	var reconstructedPayload []byte
	var reconstructErr error

	for _, segmentIndex := range segmentsToReceive {
		reconstructedPayload, _, reconstructErr = s.segmenter.Reconstruct(
			segmentedMessages[segmentIndex],
			&signer.PublicKey,
			nil,
		)
	}

	// This SHOULD succeed - Reed-Solomon recovery should work for any single missing segment,
	// including the last one. Currently this fails with ErrHashMismatch (bug #7444).
	s.Require().NoError(reconstructErr, "Reconstruction should succeed when recovering last segment via Reed-Solomon")
	s.Require().NotNil(reconstructedPayload)
	s.Require().Equal(payload, reconstructedPayload, "Reconstructed payload should match original")
}
