package common

import (
	"bytes"
	"math"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/copier"
	"github.com/klauspost/reedsolomon"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var ErrMessageSegmentsIncomplete = errors.New("message segments incomplete")
var ErrMessageSegmentsAlreadyCompleted = errors.New("message segments already completed")
var ErrMessageSegmentsInvalidCount = errors.New("invalid segments count")
var ErrMessageSegmentsHashMismatch = errors.New("hash of entire payload does not match")
var ErrMessageSegmentsInvalidParity = errors.New("invalid parity segments")

const (
	segmentsParityRate          = 0.125
	segmentsReedsolomonMaxCount = 256
)

type SegmentMessage struct {
	*protobuf.SegmentMessage
}

func (s *SegmentMessage) IsValid() bool {
	return s.SegmentsCount >= 2 || s.ParitySegmentsCount > 0
}

func (s *SegmentMessage) IsParityMessage() bool {
	return s.SegmentsCount == 0 && s.ParitySegmentsCount > 0
}

func (s *MessageSender) segmentMessage(newMessage *types.NewMessage) ([]*types.NewMessage, error) {
	// We set the max message size to 3/4 of the allowed message size, to leave
	// room for segment message metadata.
	newMessages, err := segmentMessage(newMessage, int(s.transport.MaxMessageSize()/4*3))
	s.logger.Debug("message segmented", zap.Int("segments", len(newMessages)))
	return newMessages, err
}

func replicateMessageWithNewPayload(message *types.NewMessage, payload []byte) (*types.NewMessage, error) {
	copy := &types.NewMessage{}
	err := copier.Copy(copy, message)
	if err != nil {
		return nil, err
	}

	copy.Payload = payload
	copy.PowTarget = calculatePoW(payload)
	return copy, nil
}

// Segments message into smaller chunks if the size exceeds segmentSize.
func segmentMessage(newMessage *types.NewMessage, segmentSize int) ([]*types.NewMessage, error) {
	if len(newMessage.Payload) <= segmentSize {
		return []*types.NewMessage{newMessage}, nil
	}

	entireMessageHash := crypto.Keccak256(newMessage.Payload)
	entirePayloadSize := len(newMessage.Payload)

	segmentsCount := int(math.Ceil(float64(entirePayloadSize) / float64(segmentSize)))
	paritySegmentsCount := int(math.Floor(float64(segmentsCount) * segmentsParityRate))

	segmentPayloads := make([][]byte, segmentsCount+paritySegmentsCount)
	segmentMessages := make([]*types.NewMessage, segmentsCount)

	for start, index := 0, 0; start < entirePayloadSize; start += segmentSize {
		end := start + segmentSize
		if end > entirePayloadSize {
			end = entirePayloadSize
		}

		segmentPayload := newMessage.Payload[start:end]
		segmentWithMetadata := &protobuf.SegmentMessage{
			EntireMessageHash: entireMessageHash,
			Index:             uint32(index),
			SegmentsCount:     uint32(segmentsCount),
			Payload:           segmentPayload,
		}
		marshaledSegmentWithMetadata, err := proto.Marshal(segmentWithMetadata)
		if err != nil {
			return nil, err
		}
		segmentMessage, err := replicateMessageWithNewPayload(newMessage, marshaledSegmentWithMetadata)
		if err != nil {
			return nil, err
		}

		segmentPayloads[index] = segmentPayload
		segmentMessages[index] = segmentMessage
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
			EntireMessageHash:   entireMessageHash,
			SegmentsCount:       0, // indicates parity message
			ParitySegmentIndex:  uint32(index),
			ParitySegmentsCount: uint32(paritySegmentsCount),
			Payload:             segmentPayloads[i],
		}
		marshaledSegmentWithMetadata, err := proto.Marshal(segmentWithMetadata)
		if err != nil {
			return nil, err
		}
		segmentMessage, err := replicateMessageWithNewPayload(newMessage, marshaledSegmentWithMetadata)
		if err != nil {
			return nil, err
		}

		segmentMessages = append(segmentMessages, segmentMessage)
		index++
	}

	return segmentMessages, nil
}

// SegmentationLayerV1 reconstructs the message only when all segments have been successfully retrieved.
// It lacks the capability to perform forward error correction.
// Kept to test forward compatibility.
func (s *MessageSender) handleSegmentationLayerV1(message *v1protocol.StatusMessage) error {
	logger := s.logger.With(zap.String("site", "handleSegmentationLayerV1")).With(zap.String("hash", types.HexBytes(message.TransportLayer.Hash).String()))

	segmentMessage := &SegmentMessage{
		SegmentMessage: &protobuf.SegmentMessage{},
	}
	err := proto.Unmarshal(message.TransportLayer.Payload, segmentMessage.SegmentMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SegmentMessage")
	}

	logger.Debug("handling message segment", zap.String("EntireMessageHash", types.HexBytes(segmentMessage.EntireMessageHash).String()),
		zap.Uint32("Index", segmentMessage.Index), zap.Uint32("SegmentsCount", segmentMessage.SegmentsCount))

	alreadyCompleted, err := s.persistence.IsMessageAlreadyCompleted(segmentMessage.EntireMessageHash)
	if err != nil {
		return err
	}
	if alreadyCompleted {
		return ErrMessageSegmentsAlreadyCompleted
	}

	if segmentMessage.SegmentsCount < 2 {
		return ErrMessageSegmentsInvalidCount
	}

	err = s.persistence.SaveMessageSegment(segmentMessage, message.TransportLayer.SigPubKey, time.Now().Unix())
	if err != nil {
		return err
	}

	segments, err := s.persistence.GetMessageSegments(segmentMessage.EntireMessageHash, message.TransportLayer.SigPubKey)
	if err != nil {
		return err
	}

	if len(segments) != int(segmentMessage.SegmentsCount) {
		return ErrMessageSegmentsIncomplete
	}

	// Combine payload
	var entirePayload bytes.Buffer
	for _, segment := range segments {
		_, err := entirePayload.Write(segment.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to write segment payload")
		}
	}

	// Sanity check
	entirePayloadHash := crypto.Keccak256(entirePayload.Bytes())
	if !bytes.Equal(entirePayloadHash, segmentMessage.EntireMessageHash) {
		return ErrMessageSegmentsHashMismatch
	}

	err = s.persistence.CompleteMessageSegments(segmentMessage.EntireMessageHash, message.TransportLayer.SigPubKey, time.Now().Unix())
	if err != nil {
		return err
	}

	message.TransportLayer.Payload = entirePayload.Bytes()

	return nil
}

// SegmentationLayerV2 is capable of reconstructing the message from both complete and partial sets of data segments.
// It has capability to perform forward error correction.
func (s *MessageSender) handleSegmentationLayerV2(message *v1protocol.StatusMessage) error {
	logger := s.logger.With(zap.String("site", "handleSegmentationLayerV2")).With(zap.String("hash", types.HexBytes(message.TransportLayer.Hash).String()))

	segmentMessage := &SegmentMessage{
		SegmentMessage: &protobuf.SegmentMessage{},
	}
	err := proto.Unmarshal(message.TransportLayer.Payload, segmentMessage.SegmentMessage)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal SegmentMessage")
	}

	logger.Debug("handling message segment",
		zap.String("EntireMessageHash", types.HexBytes(segmentMessage.EntireMessageHash).String()),
		zap.Uint32("Index", segmentMessage.Index),
		zap.Uint32("SegmentsCount", segmentMessage.SegmentsCount),
		zap.Uint32("ParitySegmentIndex", segmentMessage.ParitySegmentIndex),
		zap.Uint32("ParitySegmentsCount", segmentMessage.ParitySegmentsCount))

	alreadyCompleted, err := s.persistence.IsMessageAlreadyCompleted(segmentMessage.EntireMessageHash)
	if err != nil {
		return err
	}
	if alreadyCompleted {
		return ErrMessageSegmentsAlreadyCompleted
	}

	if !segmentMessage.IsValid() {
		return ErrMessageSegmentsInvalidCount
	}

	err = s.persistence.SaveMessageSegment(segmentMessage, message.TransportLayer.SigPubKey, time.Now().Unix())
	if err != nil {
		return err
	}

	segments, err := s.persistence.GetMessageSegments(segmentMessage.EntireMessageHash, message.TransportLayer.SigPubKey)
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		return errors.New("unexpected state: no segments found after save operation") // This should theoretically never occur.
	}

	firstSegmentMessage := segments[0]
	lastSegmentMessage := segments[len(segments)-1]

	// First segment message must not be a parity message.
	if firstSegmentMessage.IsParityMessage() || len(segments) != int(firstSegmentMessage.SegmentsCount) {
		return ErrMessageSegmentsIncomplete
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
			return err
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
			return err
		}

		ok, err := enc.Verify(payloads)
		if err != nil {
			return err
		}
		if !ok {
			return ErrMessageSegmentsInvalidParity
		}

		if lastNonParitySegmentPayload != nil {
			payloads[firstSegmentMessage.SegmentsCount-1] = lastNonParitySegmentPayload // Bring back last segment with original length.
		}
	}

	// Combine payload.
	var entirePayload bytes.Buffer
	for i := 0; i < int(firstSegmentMessage.SegmentsCount); i++ {
		_, err := entirePayload.Write(payloads[i])
		if err != nil {
			return errors.Wrap(err, "failed to write segment payload")
		}
	}

	// Sanity check.
	entirePayloadHash := crypto.Keccak256(entirePayload.Bytes())
	if !bytes.Equal(entirePayloadHash, segmentMessage.EntireMessageHash) {
		return ErrMessageSegmentsHashMismatch
	}

	err = s.persistence.CompleteMessageSegments(segmentMessage.EntireMessageHash, message.TransportLayer.SigPubKey, time.Now().Unix())
	if err != nil {
		return err
	}

	message.TransportLayer.Payload = entirePayload.Bytes()

	return nil
}

func (s *MessageSender) CleanupSegments() error {
	monthAgo := time.Now().AddDate(0, -1, 0).Unix()

	err := s.persistence.RemoveMessageSegmentsOlderThan(monthAgo)
	if err != nil {
		return err
	}

	err = s.persistence.RemoveMessageSegmentsCompletedOlderThan(monthAgo)
	if err != nil {
		return err
	}

	return nil
}
