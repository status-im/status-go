//go:build !disable_history_archives && !use_logos_storage
// +build !disable_history_archives,!use_logos_storage

package archive

import (
	"errors"

	logosstorage "github.com/status-im/status-go/services/logosstorage"
)

type logosStorageClientForTests interface {
	Connect(peerID string, peerAddresses []string) error
	Debug() (logosstorage.LogosStorageDebugInfo, error)
}

type logosStorageBackendForTests interface {
	GetClient() logosStorageClientForTests
}

// GetLogosStorageBackend returns an error when LogosStorage support is not built in.
func (m *ArchiveManager) GetLogosStorageBackend() (logosStorageBackendForTests, error) {
	return nil, errors.New("backend is not ArchiveManagerLogosStorage")
}
