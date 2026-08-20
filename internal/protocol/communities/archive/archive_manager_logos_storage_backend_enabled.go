//go:build !disable_history_archives && use_logos_storage

package archive

import (
	archivelogosstorage "github.com/status-im/status-go/internal/protocol/communities/archive/logosstorage"
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
)

func newLogosStorageBackend(amc *archivetypes.ArchiveManagerConfig) ArchiveServiceBackend {
	if amc.LogosStorageConfig == nil || !amc.LogosStorageConfig.Enabled {
		return nil
	}

	return archivelogosstorage.NewArchiveManagerLogosStorage(
		amc.LogosStorageConfig,
		amc.Logger,
		amc.Persistence,
		amc.Messaging,
		amc.Identity,
		amc.Publisher,
	)
}
