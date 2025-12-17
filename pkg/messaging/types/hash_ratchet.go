package types

import "errors"

var (
	ErrHashRatchetGroupIDNotFound = errors.New("hash ratchet group id not found")
	ErrNoRatchetKey               = errors.New("no ratchet key for given keyID")
)

type HashRatchetInfo struct {
	GroupID []byte
	KeyID   []byte
}
