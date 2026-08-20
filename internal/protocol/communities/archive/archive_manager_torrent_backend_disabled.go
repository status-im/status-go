//go:build !disable_history_archives && !use_torrent

package archive

import archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"

func newTorrentBackend(_ *archivetypes.ArchiveManagerConfig) ArchiveServiceBackend {
	return nil
}
