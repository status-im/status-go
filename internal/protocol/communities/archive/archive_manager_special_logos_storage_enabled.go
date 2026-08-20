//go:build !disable_history_archives && use_logos_storage

package archive

import (
	"errors"

	archivelogosstorage "github.com/status-im/status-go/internal/protocol/communities/archive/logosstorage"
)

// GetLogosStorageBackend returns the LogosStorage backend if available, for test/debug purposes.
func (m *ArchiveManager) GetLogosStorageBackend() (*archivelogosstorage.ArchiveManagerLogosStorage, error) {
	if logosStorageBackend, ok := m.backend.(*archivelogosstorage.ArchiveManagerLogosStorage); ok {
		return logosStorageBackend, nil
	}
	return nil, errors.New("backend is not ArchiveManagerLogosStorage")
}
