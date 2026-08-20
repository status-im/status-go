//go:build disable_history_archives

package archive

import (
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
)

// NewArchiveManager this function is only built and called when the "disable_history_archives" build tag is set.
// In this case this version of NewArchiveManager will return the NOP implementation ensuring that the
// build command will not import or build the archive dependencies for mobile builds.
// NOTE: It is intentional that this file contains the identical function name as in "archive_manager.go"
func NewArchiveManager(amc *archivetypes.ArchiveManagerConfig) ArchiveService {
	return &ArchiveManagerNop{}
}
