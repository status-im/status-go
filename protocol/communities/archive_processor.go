//go:build !disable_torrent
// +build !disable_torrent

package communities

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/messaging"
	"github.com/status-im/status-go/protocol/protobuf"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// CodexArchiveMessageProcessor implements the CodexArchiveProcessor interface
// It handles the extraction and processing of messages from archive data
type CodexArchiveMessageProcessor struct {
	identity    *ecdsa.PrivateKey
	messaging   *messaging.API
	persistence *Persistence
	logger      *zap.Logger

	// Message processing dependencies
	messageHandler func([]*protobuf.WakuMessage) error

	// Callback for when archive processing is complete
	onArchiveProcessed func(hash string, from, to uint64)
}

// NewCodexArchiveMessageProcessor creates a new codex archive message processor
func NewCodexArchiveMessageProcessor(
	identity *ecdsa.PrivateKey,
	messaging *messaging.API,
	persistence *Persistence,
	logger *zap.Logger,
	messageHandler func([]*protobuf.WakuMessage) error,
) *CodexArchiveMessageProcessor {
	return &CodexArchiveMessageProcessor{
		identity:       identity,
		messaging:      messaging,
		persistence:    persistence,
		logger:         logger,
		messageHandler: messageHandler,
	}
}

// SetOnArchiveProcessed sets a callback to be called when an archive is successfully processed
func (p *CodexArchiveMessageProcessor) SetOnArchiveProcessed(callback func(hash string, from, to uint64)) {
	p.onArchiveProcessed = callback
}

// ProcessArchiveData extracts and processes messages from archive data
func (p *CodexArchiveMessageProcessor) ProcessArchiveData(communityID types.HexBytes, archiveHash string, archiveData []byte, from, to uint64) error {
	p.logger.Debug("processing archive data",
		zap.String("communityID", communityID.String()),
		zap.String("archiveHash", archiveHash),
		zap.Int("dataSize", len(archiveData)),
		zap.Uint64("from", from),
		zap.Uint64("to", to))

	// Extract messages from the archive data
	messages, err := p.extractMessagesFromArchiveData(communityID, archiveData)
	if err != nil {
		return fmt.Errorf("failed to extract messages from archive: %w", err)
	}

	p.logger.Debug("extracted messages from archive",
		zap.String("archiveHash", archiveHash),
		zap.Int("messageCount", len(messages)))

	// Process messages through the same pipeline as torrent archives
	if len(messages) > 0 {
		err = p.messageHandler(messages)
		if err != nil {
			return fmt.Errorf("failed to process archive messages: %w", err)
		}
	}

	// Save the archive ID to persistence (marking it as downloaded)
	err = p.persistence.SaveMessageArchiveID(communityID, archiveHash)
	if err != nil {
		return fmt.Errorf("failed to save message archive ID: %w", err)
	}

	// Call the completion callback
	if p.onArchiveProcessed != nil {
		p.onArchiveProcessed(archiveHash, from, to)
	}

	return nil
}

// extractMessagesFromArchiveData extracts messages from raw archive data
// This is similar to ExtractMessagesFromHistoryArchive but works with raw data instead of files
func (p *CodexArchiveMessageProcessor) extractMessagesFromArchiveData(communityID types.HexBytes, archiveData []byte) ([]*protobuf.WakuMessage, error) {
	archive := &protobuf.WakuMessageArchive{}

	// Try to unmarshal the data directly first
	err := proto.Unmarshal(archiveData, archive)
	if err != nil {
		// If direct unmarshaling fails, try to decrypt the data first
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			p.logger.Error("failed to decompress community pubkey", zap.Error(err))
			return nil, err
		}

		decryptedData, err := p.messaging.DecryptMessage(p.identity, pk, archiveData)
		if err != nil {
			p.logger.Error("failed to decrypt message archive", zap.Error(err))
			return nil, err
		}

		err = proto.Unmarshal(decryptedData, archive)
		if err != nil {
			p.logger.Error("failed to unmarshal message archive", zap.Error(err))
			return nil, err
		}
	}

	return archive.Messages, nil
}
