package types

import "github.com/status-im/status-go/protocol/protobuf"

type SegmentMessage struct {
	*protobuf.SegmentMessage
}

func (s *SegmentMessage) IsValid() bool {
	// Check if the hash length is valid (32 bytes for Keccak256)
	if len(s.EntireMessageHash) != 32 {
		return false
	}

	// Check if the segment index is within the valid range
	if s.SegmentsCount > 0 && s.Index >= s.SegmentsCount {
		return false
	}

	// Check if the parity segment index is within the valid range
	if s.ParitySegmentsCount > 0 && s.ParitySegmentIndex >= s.ParitySegmentsCount {
		return false
	}

	return s.SegmentsCount >= 2 || s.ParitySegmentsCount > 0
}

func (s *SegmentMessage) IsParityMessage() bool {
	return s.SegmentsCount == 0 && s.ParitySegmentsCount > 0
}
