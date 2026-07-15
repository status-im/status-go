package segmentation

import (
	"crypto/ecdsa"
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

// countingPersistence wraps a real Persistence and counts GetMessageSegments
// calls, so tests can assert the expensive full-segment read (which copies every
// stored payload blob out of sqlcipher) happens only once, on completion —
// instead of on every arriving segment (the O(N^2) behavior, issue #21470-hf).
type countingPersistence struct {
	Persistence
	getSegmentsCalls int
}

func (c *countingPersistence) GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*Message, error) {
	c.getSegmentsCalls++
	return c.Persistence.GetMessageSegments(hash, sigPubKey)
}

func (s *MessageSegmentationSuite) newCountingSegmenter() (*Segmenter, *countingPersistence) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     migrations.AssetNames(),
			AssetFunc: migrations.Asset,
		},
	}))
	s.Require().NoError(err)
	counting := &countingPersistence{Persistence: NewSQLitePersistence(db)}
	return NewSegmenter(counting, zap.Must(zap.NewDevelopment())), counting
}

// TestReadsAllSegmentsOnlyOnCompletion (no parity) verifies that arriving segments
// before the last do NOT trigger a full read of every stored segment, and the last
// one triggers exactly one such read (issue #21470-hf: O(N) instead of O(N^2)).
func (s *MessageSegmentationSuite) TestReadsAllSegmentsOnlyOnCompletion() {
	segmenter, counting := s.newCountingSegmenter()

	signer, err := crypto.GenerateKey()
	s.Require().NoError(err)

	segmentsCount := 5 // floor(5*0.125)=0 parity segments
	segSize := (len(s.testPayload) + segmentsCount - 1) / segmentsCount
	segs, err := segmenter.Segment(s.testPayload, segSize)
	s.Require().NoError(err)
	s.Require().Len(segs, segmentsCount)

	for i := 0; i < segmentsCount-1; i++ {
		_, _, err := segmenter.Reconstruct(segs[i], &signer.PublicKey, nil)
		s.Require().ErrorIs(err, ErrIncomplete)
	}
	s.Require().Equal(0, counting.getSegmentsCalls, "must not read all segments before completion")

	payload, _, err := segmenter.Reconstruct(segs[segmentsCount-1], &signer.PublicKey, nil)
	s.Require().NoError(err)
	s.Require().ElementsMatch(s.testPayload, payload)
	s.Require().Equal(1, counting.getSegmentsCalls, "reads all segments exactly once, on completion")
}

// TestReadsAllSegmentsOnlyOnCompletionOutOfOrder verifies the completion precheck
// is order-independent: the full read still fires exactly once, on the arrival that
// brings the stored count up to SegmentsCount.
func (s *MessageSegmentationSuite) TestReadsAllSegmentsOnlyOnCompletionOutOfOrder() {
	segmenter, counting := s.newCountingSegmenter()

	signer, err := crypto.GenerateKey()
	s.Require().NoError(err)

	segmentsCount := 5
	segSize := (len(s.testPayload) + segmentsCount - 1) / segmentsCount
	segs, err := segmenter.Segment(s.testPayload, segSize)
	s.Require().NoError(err)

	order := []int{3, 0, 4, 1, 2}
	for i, idx := range order {
		payload, _, err := segmenter.Reconstruct(segs[idx], &signer.PublicKey, nil)
		if i < segmentsCount-1 {
			s.Require().ErrorIs(err, ErrIncomplete)
			s.Require().Equal(0, counting.getSegmentsCalls)
		} else {
			s.Require().NoError(err)
			s.Require().ElementsMatch(s.testPayload, payload)
			s.Require().Equal(1, counting.getSegmentsCalls)
		}
	}
}

// TestDuplicateSegmentDoesNotTriggerRead verifies a duplicate arrival (which
// REPLACEs its row and leaves the stored count unchanged) neither completes the
// message nor triggers a full read.
func (s *MessageSegmentationSuite) TestDuplicateSegmentDoesNotTriggerRead() {
	segmenter, counting := s.newCountingSegmenter()

	signer, err := crypto.GenerateKey()
	s.Require().NoError(err)

	segmentsCount := 5
	segSize := (len(s.testPayload) + segmentsCount - 1) / segmentsCount
	segs, err := segmenter.Segment(s.testPayload, segSize)
	s.Require().NoError(err)

	_, _, err = segmenter.Reconstruct(segs[0], &signer.PublicKey, nil)
	s.Require().ErrorIs(err, ErrIncomplete)
	_, _, err = segmenter.Reconstruct(segs[0], &signer.PublicKey, nil) // duplicate
	s.Require().ErrorIs(err, ErrIncomplete)
	s.Require().Equal(0, counting.getSegmentsCalls, "duplicate must not trigger a full read")
}

// TestParityCompletionReadsOnce verifies the completion precheck fires the single
// full read even when the completing segment is a parity segment (reedsolomon
// reconstruction with one data segment missing).
func (s *MessageSegmentationSuite) TestParityCompletionReadsOnce() {
	segmenter, counting := s.newCountingSegmenter()

	signer, err := crypto.GenerateKey()
	s.Require().NoError(err)

	segmentsCount := 8 // floor(8*0.125)=1 parity segment
	segSize := (len(s.testPayload) + segmentsCount - 1) / segmentsCount
	segs, err := segmenter.Segment(s.testPayload, segSize)
	s.Require().NoError(err)
	s.Require().Len(segs, segmentsCount+1) // 8 data + 1 parity

	// Deliver 7 of the 8 data segments (index 0..6), then the parity segment
	// (index 8) which brings the stored count to 8 and completes via parity.
	arrival := []int{0, 1, 2, 3, 4, 5, 6, 8}
	for i, idx := range arrival {
		payload, _, err := segmenter.Reconstruct(segs[idx], &signer.PublicKey, nil)
		if i < segmentsCount-1 {
			s.Require().ErrorIs(err, ErrIncomplete)
			s.Require().Equal(0, counting.getSegmentsCalls)
		} else {
			s.Require().NoError(err)
			s.Require().ElementsMatch(s.testPayload, payload)
			s.Require().Equal(1, counting.getSegmentsCalls)
		}
	}
}
