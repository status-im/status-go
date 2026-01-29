//go:build !disable_history_archives && !use_torrent
// +build !disable_history_archives,!use_torrent

package archive

import (
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
)

func newTorrentBackend(_ *archivetypes.ArchiveManagerConfig) ArchiveServiceBackend {
	return nil
}
