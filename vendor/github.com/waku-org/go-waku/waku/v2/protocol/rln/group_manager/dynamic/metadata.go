package dynamic

import (
	"encoding/binary"
	"errors"
)

// RLNMetadata persists attributes in the RLN database
type RLNMetadata struct {
	LastProcessedBlock uint64
}

// Serialize converts a RLNMetadata into a binary format expected by zerokit's RLN
func (r RLNMetadata) Serialize() []byte {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, r.LastProcessedBlock)
	return result
}

// DeserializeMetadata converts a byte slice into a RLNMetadata instance
func DeserializeMetadata(b []byte) (RLNMetadata, error) {
	if len(b) != 8 {
		return RLNMetadata{}, errors.New("wrong size")
	}
	return RLNMetadata{
		LastProcessedBlock: binary.LittleEndian.Uint64(b),
	}, nil
}

// SetMetadata stores some metadata into the zerokit's RLN database
func (gm *DynamicGroupManager) SetMetadata(meta RLNMetadata) error {
	b := meta.Serialize()
	return gm.rln.SetMetadata(b)
}

// GetMetadata retrieves metadata from the zerokit's RLN database
func (gm *DynamicGroupManager) GetMetadata() (RLNMetadata, error) {
	b, err := gm.rln.GetMetadata()
	if err != nil {
		return RLNMetadata{}, err
	}

	return DeserializeMetadata(b)
}
