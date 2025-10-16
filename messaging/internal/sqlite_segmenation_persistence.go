package internal

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/types"
)

type SQLiteSegmentationPersistence struct {
	*segmentation.SQLitePersistence
}

var _ types.SegmentationPersistence = (*SQLiteSegmentationPersistence)(nil)

func (s *SQLiteSegmentationPersistence) IsMessageAlreadyCompleted(hash []byte) (bool, error) {
	return false, nil
}

func (s *SQLiteSegmentationPersistence) SaveMessageSegment(segment *types.SegmentMessage, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	return nil
}

func (s *SQLiteSegmentationPersistence) GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*types.SegmentMessage, error) {
	return nil, nil
}

func (s *SQLiteSegmentationPersistence) CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error {
	return nil
}

func (s *SQLiteSegmentationPersistence) RemoveMessageSegmentsOlderThan(timestamp int64) error {
	return nil
}

func (s *SQLiteSegmentationPersistence) RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error {
	return nil
}
