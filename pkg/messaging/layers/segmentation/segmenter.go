package segmentation

import (
	"bytes"
	"crypto/ecdsa"
	"math"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/klauspost/reedsolomon"
	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/segmentation/protobuf"
)

const (
	segmentsParityRate          = 0.125
	segmentsReedsolomonMaxCount = 256
)

var ErrIncomplete = errors.New("message segments incomplete")
var ErrAlreadyCompleted = errors.New("message segments already completed")
var ErrInvalidPayload = errors.New("invalid segment payload")
var ErrHashMismatch = errors.New("hash of entire payload does not match")
var ErrInvalidParity = errors.New("invalid parity segments")

type Segmenter struct {
	persistence Persistence
	logger      *zap.Logger
}

func NewSegmenter(persistence Persistence, logger *zap.Logger) *Segmenter {
	return &Segmenter{
		persistence: persistence,
		logger:      logger.Named("segmentation"),
	}
}

func (s *Segmenter) Segment(payload []byte, segmentSize int) ([][]byte, error) {
	if len(payload) <= segmentSize {
		return [][]byte{payload}, nil
	}

	entireMessageHash := crypto.Keccak256(payload)
	entirePayloadSize := len(payload)

	segmentsCount := int(math.Ceil(float64(entirePayloadSize) / float64(segmentSize)))
	paritySegmentsCount := int(math.Floor(float64(segmentsCount) * segmentsParityRate))

	segmentPayloads := make([][]byte, segmentsCount+paritySegmentsCount)
	segmentMessages := make([][]byte, segmentsCount)

	for start, index := 0, 0; start < entirePayloadSize; start += segmentSize {
		end := start + segmentSize
		if end > entirePayloadSize {
			end = entirePayloadSize
		}

		segmentPayload := payload[start:end]
		segmentWithMetadata := &protobuf.SegmentMessage{
			EntireMessageHash:     entireMessageHash,
			Index:                 uint32(index),
			SegmentsCount:         uint32(segmentsCount),
			Payload:               segmentPayload,
			OriginalPayloadLength: uint64(entirePayloadSize),
		}
		marshaledSegmentWithMetadata, err := proto.Marshal(segmentWithMetadata)
		if err != nil {
			return nil, err
		}

		segmentPayloads[index] = segmentPayload
		segmentMessages[index] = marshaledSegmentWithMetadata
		index++
	}

	// Skip reedsolomon if the combined total of data and parity segments exceeds the predefined limit of segmentsReedsolomonMaxCount.
	// Exceeding this limit necessitates shard sizes to be multiples of 64, which are incompatible with clients that do not support forward error correction.
	if paritySegmentsCount == 0 || segmentsCount+paritySegmentsCount > segmentsReedsolomonMaxCount {
		return segmentMessages, nil
	}

	enc, err := reedsolomon.New(segmentsCount, paritySegmentsCount)
	if err != nil {
		return nil, err
	}

	// Align the size of the last segment payload.
	lastSegmentPayload := segmentPayloads[segmentsCount-1]
	segmentPayloads[segmentsCount-1] = make([]byte, segmentSize)
	copy(segmentPayloads[segmentsCount-1], lastSegmentPayload)

	// Make space for parity data.
	for i := segmentsCount; i < segmentsCount+paritySegmentsCount; i++ {
		segmentPayloads[i] = make([]byte, segmentSize)
	}

	err = enc.Encode(segmentPayloads)
	if err != nil {
		return nil, err
	}

	// Create parity messages.
	for i, index := segmentsCount, 0; i < segmentsCount+paritySegmentsCount; i++ {
		segmentWithMetadata := &protobuf.SegmentMessage{
			EntireMessageHash:     entireMessageHash,
			SegmentsCount:         0, // indicates parity message
			ParitySegmentIndex:    uint32(index),
			ParitySegmentsCount:   uint32(paritySegmentsCount),
			Payload:               segmentPayloads[i],
			OriginalPayloadLength: uint64(entirePayloadSize),
		}
		marshaledSegmentWithMetadata, err := proto.Marshal(segmentWithMetadata)
		if err != nil {
			return nil, err
		}

		segmentMessages = append(segmentMessages, marshaledSegmentWithMetadata)
		index++
	}

	return segmentMessages, nil
}

func (s *Segmenter) Reconstruct(payload []byte, sigPubKey *ecdsa.PublicKey, transportID []byte) ([]byte, [][]byte, error) {
	segmentMessage := &Message{
		SegmentMessage: &protobuf.SegmentMessage{},
		transportID:    transportID,
	}
	err := proto.Unmarshal(payload, segmentMessage.SegmentMessage)
	if err != nil || !segmentMessage.IsValid() {
		return nil, nil, ErrInvalidPayload
	}

	s.logger.Debug("handling message segment",
		zap.String("EntireMessageHash", cryptotypes.HexBytes(segmentMessage.EntireMessageHash).String()),
		zap.Uint32("Index", segmentMessage.Index),
		zap.Uint32("SegmentsCount", segmentMessage.SegmentsCount),
		zap.Uint32("ParitySegmentIndex", segmentMessage.ParitySegmentIndex),
		zap.Uint32("ParitySegmentsCount", segmentMessage.ParitySegmentsCount))

	alreadyCompleted, err := s.persistence.IsMessageAlreadyCompleted(segmentMessage.EntireMessageHash)
	if err != nil {
		return nil, nil, err
	}
	if alreadyCompleted {
		return nil, nil, ErrAlreadyCompleted
	}

	err = s.persistence.SaveMessageSegment(segmentMessage, sigPubKey, time.Now().Unix())
	if err != nil {
		return nil, nil, err
	}

	segments, err := s.persistence.GetMessageSegments(segmentMessage.EntireMessageHash, sigPubKey)
	if err != nil {
		return nil, nil, err
	}

	if len(segments) == 0 {
		return nil, nil, errors.New("unexpected state: no segments found after save operation") // This should theoretically never occur.
	}

	firstSegmentMessage := segments[0]
	lastSegmentMessage := segments[len(segments)-1]

	// First segment message must not be a parity message.
	if firstSegmentMessage.IsParityMessage() || len(segments) != int(firstSegmentMessage.SegmentsCount) {
		return nil, nil, ErrIncomplete
	}

	payloads := make([][]byte, firstSegmentMessage.SegmentsCount+lastSegmentMessage.ParitySegmentsCount)
	payloadSize := len(firstSegmentMessage.Payload)

	restoreUsingParityData := lastSegmentMessage.IsParityMessage()
	if !restoreUsingParityData {
		for i, segment := range segments {
			payloads[i] = segment.Payload
		}
	} else {
		enc, err := reedsolomon.New(int(firstSegmentMessage.SegmentsCount), int(lastSegmentMessage.ParitySegmentsCount))
		if err != nil {
			return nil, nil, err
		}

		var lastNonParitySegmentPayload []byte
		for _, segment := range segments {
			if !segment.IsParityMessage() {
				if segment.Index == firstSegmentMessage.SegmentsCount-1 {
					// Ensure last segment is aligned to payload size, as it is required by reedsolomon.
					payloads[segment.Index] = make([]byte, payloadSize)
					copy(payloads[segment.Index], segment.Payload)
					lastNonParitySegmentPayload = segment.Payload
				} else {
					payloads[segment.Index] = segment.Payload
				}
			} else {
				payloads[firstSegmentMessage.SegmentsCount+segment.ParitySegmentIndex] = segment.Payload
			}
		}

		err = enc.Reconstruct(payloads)
		if err != nil {
			return nil, nil, err
		}

		ok, err := enc.Verify(payloads)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, ErrInvalidParity
		}

		// Backwards compatibility: Only use lastNonParitySegmentPayload if OriginalPayloadLength is not set
		if firstSegmentMessage.OriginalPayloadLength == 0 && lastNonParitySegmentPayload != nil {
			payloads[firstSegmentMessage.SegmentsCount-1] = lastNonParitySegmentPayload // Bring back last segment with original length.
		}
	}

	// Combine payload.
	var entirePayload bytes.Buffer
	for i := 0; i < int(firstSegmentMessage.SegmentsCount); i++ {
		_, err := entirePayload.Write(payloads[i])
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to write segment payload")
		}
	}

	// Truncate to original payload length if specified (fix for issue #7444)
	reconstructedPayload := entirePayload.Bytes()
	if firstSegmentMessage.OriginalPayloadLength > 0 && uint64(len(reconstructedPayload)) > firstSegmentMessage.OriginalPayloadLength {
		reconstructedPayload = reconstructedPayload[:firstSegmentMessage.OriginalPayloadLength]
	}

	// Sanity check.
	entirePayloadHash := crypto.Keccak256(reconstructedPayload)
	if !bytes.Equal(entirePayloadHash, segmentMessage.EntireMessageHash) {
		return nil, nil, ErrHashMismatch
	}

	err = s.persistence.CompleteMessageSegments(segmentMessage.EntireMessageHash, sigPubKey, time.Now().Unix())
	if err != nil {
		return nil, nil, err
	}

	transportIDs := make([][]byte, len(segments))
	for i, segment := range segments {
		transportIDs[i] = segment.transportID
	}

	return reconstructedPayload, transportIDs, nil
}

func (s *Segmenter) CleanupStaleSegments(olderThan time.Time) error {
	err := s.persistence.RemoveMessageSegmentsOlderThan(olderThan.Unix())
	if err != nil {
		return err
	}

	err = s.persistence.RemoveMessageSegmentsCompletedOlderThan(olderThan.Unix())
	if err != nil {
		return err
	}

	return nil
}
