package sender

import (
	"go.uber.org/zap"
)

// reducedMaxMessageSize returns the max message size reduced to 3/4 to leave room for segment metadata
func (s *Sender) reducedMaxMessageSize() uint32 {
	return s.stack.Transport.MaxMessageSize() * 3 / 4
}

func (s *Sender) segmentMessage(payload []byte) ([][]byte, error) {
	return s.segmentMessageWithSize(payload, int(s.reducedMaxMessageSize()))
}

func (s *Sender) segmentMessageWithSize(payload []byte, segmentSize int) ([][]byte, error) {
	segments, err := s.stack.Segmentation.Segment(payload, segmentSize)
	if err != nil {
		return nil, err
	}

	if len(segments) > 1 {
		s.logger.Debug("message segmented", zap.Int("segmentsCount", len(segments)))
	}

	return segments, err
}
