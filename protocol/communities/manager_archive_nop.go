//go:build disable_torrent
// +build disable_torrent

package communities

import (
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/transport"
)

type ArchiveManagerNop struct {
	*ArchiveFileManagerNop
}

// NewArchiveManager this function is only built and called when the "disable_torrent" build tag is set
// In this case this version of NewArchiveManager will return the mobile "nil" ArchiveManagerNop ensuring that the
// build command will not import or build the torrent deps for the mobile OS.
// NOTE: It is intentional that this file contains the identical function name as in "manager_archive.go"
func NewArchiveManager(amc *ArchiveManagerConfig) *ArchiveManagerNop {
	return &ArchiveManagerNop{
		&ArchiveFileManagerNop{},
	}
}

func (tmm *ArchiveManagerNop) SetOnline(online bool) {}

func (tmm *ArchiveManagerNop) SetTorrentConfig(*params.TorrentConfig) {}

func (tmm *ArchiveManagerNop) StartTorrentClient() error {
	return nil
}

func (tmm *ArchiveManagerNop) Stop() error {
	return nil
}

func (tmm *ArchiveManagerNop) IsReady() bool {
	return false
}

func (tmm *ArchiveManagerNop) GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error) {
	return nil, nil
}

func (tmm *ArchiveManagerNop) GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error) {
	return nil, nil
}

func (tmm *ArchiveManagerNop) GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error) {
	return 0, nil
}

func (tmm *ArchiveManagerNop) CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	return nil
}

func (tmm *ArchiveManagerNop) StartHistoryArchiveTasksInterval(community *Community, interval time.Duration) {
}

func (tmm *ArchiveManagerNop) StopHistoryArchiveTasksInterval(communityID types.HexBytes) {}

func (tmm *ArchiveManagerNop) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
	return nil
}

func (tmm *ArchiveManagerNop) UnseedHistoryArchiveTorrent(communityID types.HexBytes) {}

func (tmm *ArchiveManagerNop) IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool {
	return false
}

func (tmm *ArchiveManagerNop) GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask {
	return nil
}

func (tmm *ArchiveManagerNop) AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask) {
}

func (tmm *ArchiveManagerNop) DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {
	return nil, nil
}

func (tmm *ArchiveManagerNop) TorrentFileExists(communityID string) bool {
	return false
}
