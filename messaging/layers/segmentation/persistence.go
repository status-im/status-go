package segmentation

import (
	"crypto/ecdsa"
)

type Persistence interface {
	IsMessageAlreadyCompleted(hash []byte) (bool, error)
	SaveMessageSegment(segment *Message, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*Message, error)
	CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	RemoveMessageSegmentsOlderThan(timestamp int64) error
	RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error
}
