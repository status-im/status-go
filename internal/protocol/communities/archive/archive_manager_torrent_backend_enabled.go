//go:build !disable_history_archives && use_torrent

package archive

import (
	archivetorrent "github.com/status-im/status-go/internal/protocol/communities/archive/torrent"
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
)

func newTorrentBackend(amc *archivetypes.ArchiveManagerConfig) ArchiveServiceBackend {
	if amc.TorrentConfig == nil || !amc.TorrentConfig.Enabled {
		return nil
	}

	return archivetorrent.NewArchiveManagerTorrent(
		amc.TorrentConfig,
		amc.Logger,
		amc.Persistence,
		amc.Messaging,
		amc.Identity,
		amc.Publisher,
	)
}
