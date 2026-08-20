//go:build !disable_history_archives && !use_logos_storage

package archive

import archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"

func newLogosStorageBackend(_ *archivetypes.ArchiveManagerConfig) ArchiveServiceBackend {
	return nil
}
