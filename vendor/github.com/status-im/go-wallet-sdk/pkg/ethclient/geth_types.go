package ethclient

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// rpcProgress is a copy of SyncProgress with hex-encoded fields.
type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64

	PulledStates hexutil.Uint64
	KnownStates  hexutil.Uint64

	SyncedAccounts         hexutil.Uint64
	SyncedAccountBytes     hexutil.Uint64
	SyncedBytecodes        hexutil.Uint64
	SyncedBytecodeBytes    hexutil.Uint64
	SyncedStorage          hexutil.Uint64
	SyncedStorageBytes     hexutil.Uint64
	HealedTrienodes        hexutil.Uint64
	HealedTrienodeBytes    hexutil.Uint64
	HealedBytecodes        hexutil.Uint64
	HealedBytecodeBytes    hexutil.Uint64
	HealingTrienodes       hexutil.Uint64
	HealingBytecode        hexutil.Uint64
	TxIndexFinishedBlocks  hexutil.Uint64
	TxIndexRemainingBlocks hexutil.Uint64
}

func (p *rpcProgress) toSyncProgress() *ethereum.SyncProgress {
	if p == nil {
		return nil
	}
	return &ethereum.SyncProgress{
		StartingBlock:          uint64(p.StartingBlock),
		CurrentBlock:           uint64(p.CurrentBlock),
		HighestBlock:           uint64(p.HighestBlock),
		PulledStates:           uint64(p.PulledStates),
		KnownStates:            uint64(p.KnownStates),
		SyncedAccounts:         uint64(p.SyncedAccounts),
		SyncedAccountBytes:     uint64(p.SyncedAccountBytes),
		SyncedBytecodes:        uint64(p.SyncedBytecodes),
		SyncedBytecodeBytes:    uint64(p.SyncedBytecodeBytes),
		SyncedStorage:          uint64(p.SyncedStorage),
		SyncedStorageBytes:     uint64(p.SyncedStorageBytes),
		HealedTrienodes:        uint64(p.HealedTrienodes),
		HealedTrienodeBytes:    uint64(p.HealedTrienodeBytes),
		HealedBytecodes:        uint64(p.HealedBytecodes),
		HealedBytecodeBytes:    uint64(p.HealedBytecodeBytes),
		HealingTrienodes:       uint64(p.HealingTrienodes),
		HealingBytecode:        uint64(p.HealingBytecode),
		TxIndexFinishedBlocks:  uint64(p.TxIndexFinishedBlocks),
		TxIndexRemainingBlocks: uint64(p.TxIndexRemainingBlocks),
	}
}
