package dynamic

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// RLNMetadata persists attributes in the RLN database
type RLNMetadata struct {
	LastProcessedBlock uint64
	ChainID            *big.Int
	ContractAddress    common.Address
}

// Serialize converts a RLNMetadata into a binary format expected by zerokit's RLN
func (r RLNMetadata) Serialize() []byte {
	chainID := r.ChainID
	if chainID == nil {
		chainID = big.NewInt(0)
	}

	var result []byte
	result = binary.LittleEndian.AppendUint64(result, r.LastProcessedBlock)
	result = binary.LittleEndian.AppendUint64(result, chainID.Uint64())
	result = append(result, r.ContractAddress.Bytes()...)
	return result
}

const lastProcessedBlockOffset = 0
const chainIDOffset = lastProcessedBlockOffset + 8
const contractAddressOffset = chainIDOffset + 8
const metadataByteLen = 8 + 8 + 20 // 2 uint64 fields and a 20bytes address

// DeserializeMetadata converts a byte slice into a RLNMetadata instance
func DeserializeMetadata(b []byte) (RLNMetadata, error) {
	if len(b) != metadataByteLen {
		return RLNMetadata{}, errors.New("wrong size")
	}

	return RLNMetadata{
		LastProcessedBlock: binary.LittleEndian.Uint64(b[lastProcessedBlockOffset:chainIDOffset]),
		ChainID:            new(big.Int).SetUint64(binary.LittleEndian.Uint64(b[chainIDOffset:contractAddressOffset])),
		ContractAddress:    common.BytesToAddress(b[contractAddressOffset:]),
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
