package segmentation

import (
	"crypto/ecdsa"
)

type Persistence interface {
	IsMessageAlreadyCompleted(hash []byte) (bool, error)
	SaveMessageSegment(segment *Message, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	// GetMessageSegmentsCompletionInfo returns the number of stored segments for a
	// message and the SegmentsCount its data segments advertise (0 if none stored
	// yet). It is a cheap aggregate (COUNT + MAX, no payload blobs copied) used to
	// avoid loading every stored segment via GetMessageSegments until the message
	// can actually be reassembled (issue #21470-hf).
	GetMessageSegmentsCompletionInfo(hash []byte, sigPubKey *ecdsa.PublicKey) (storedCount int, segmentsCount uint32, err error)
	GetMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey) ([]*Message, error)
	CompleteMessageSegments(hash []byte, sigPubKey *ecdsa.PublicKey, timestamp int64) error
	RemoveMessageSegmentsOlderThan(timestamp int64) error
	RemoveMessageSegmentsCompletedOlderThan(timestamp int64) error
}
