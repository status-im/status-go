//go:build !disable_history_archives && use_logos_storage

package protocol

import (
	"errors"

	"github.com/status-im/status-go/internal/protocol/communities/archive"
	"github.com/status-im/status-go/services/logosstorage"
)

func (m *Messenger) Connect(peerID string, addrs []string) error {
	archiveManager, ok := m.archiveManager.(*archive.ArchiveManager)
	if !ok {
		return errors.New("archiveManager is not *archive.ArchiveManager")
	}

	logosStorageManager, err := archiveManager.GetLogosStorageBackend()
	if err != nil {
		return err
	}

	client := logosStorageManager.GetLogosStorageClient()
	if client == nil {
		return errors.New("logosStorage client is not initialized")
	}

	return client.Connect(peerID, addrs)
}

func (m *Messenger) Debug() (logosstorage.LogosStorageDebugInfo, error) {
	archiveManager, ok := m.archiveManager.(*archive.ArchiveManager)
	if !ok {
		return logosstorage.LogosStorageDebugInfo{}, errors.New("archiveManager is not *archive.ArchiveManager")
	}

	logosStorageManager, err := archiveManager.GetLogosStorageBackend()
	if err != nil {
		return logosstorage.LogosStorageDebugInfo{}, err
	}

	client := logosStorageManager.GetLogosStorageClient()
	if client == nil {
		return logosstorage.LogosStorageDebugInfo{}, errors.New("logosStorage client is not initialized")
	}

	return client.Debug()
}
