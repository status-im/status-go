//go:build disable_torrent
// +build disable_torrent

package communities

import (
	"github.com/status-im/status-go/crypto/types"
)

// CodexArchiveProcessor interface for builds without torrent support
type CodexArchiveProcessor interface {
	ProcessArchiveData(communityID types.HexBytes, archiveHash string, archiveData []byte, from, to uint64) error
}

// CodexArchiveMessageProcessorNop is a no-op implementation for builds without torrent support
type CodexArchiveMessageProcessorNop struct{}

// NewCodexArchiveMessageProcessor creates a no-op processor for builds without torrent support
func NewCodexArchiveMessageProcessor(identity interface{}, messaging interface{}, persistence interface{}, logger interface{}, messageHandler interface{}) *CodexArchiveMessageProcessorNop {
	return &CodexArchiveMessageProcessorNop{}
}

// SetOnArchiveProcessed is a no-op
func (p *CodexArchiveMessageProcessorNop) SetOnArchiveProcessed(callback func(hash string, from, to uint64)) {
	// no-op
}

// ProcessArchiveData is a no-op that returns an error
func (p *CodexArchiveMessageProcessorNop) ProcessArchiveData(communityID types.HexBytes, archiveHash string, archiveData []byte, from, to uint64) error {
	return ErrArchiveNotSupported
}