package segmentation

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/messaging/types"
)

type Persistence interface {
	IsMessageAlreadyCompleted(hash []byte) (bool, error)
	SaveMessageSegment(segment *types.SegmentMessage, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*types.SegmentMessage, error)
	CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	RemoveMessageSegmentsOlderThan(timestamp int64) error
	RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error
}
