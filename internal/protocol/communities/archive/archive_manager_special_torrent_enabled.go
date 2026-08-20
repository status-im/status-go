//go:build !disable_history_archives && use_torrent

package archive

import (
	"errors"

	archivetorrent "github.com/status-im/status-go/internal/protocol/communities/archive/torrent"
)

// GetTorrentBackend returns the Torrent backend if available, for test purposes.
func (m *ArchiveManager) GetTorrentBackend() (*archivetorrent.ArchiveManagerTorrent, error) {
	if torrentBackend, ok := m.backend.(*archivetorrent.ArchiveManagerTorrent); ok {
		return torrentBackend, nil
	}
	return nil, errors.New("backend is not ArchiveManagerTorrent")
}
