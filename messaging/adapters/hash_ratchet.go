package adapters

import (
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/types"
)

func FromEncryptionHashRatchet(s *encryption.HashRatchetInfo) *types.HashRatchetInfo {
	if s == nil {
		return nil
	}
	return &types.HashRatchetInfo{
		GroupID: s.GroupID,
		KeyID:   s.KeyID,
	}
}

func FromEncryptionHashRatchets(s []*encryption.HashRatchetInfo) []*types.HashRatchetInfo {
	if s == nil {
		return nil
	}
	ratchets := make([]*types.HashRatchetInfo, 0, len(s))
	for _, item := range s {
		ratchets = append(ratchets, FromEncryptionHashRatchet(item))
	}
	return ratchets
}
