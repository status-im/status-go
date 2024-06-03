package communities

import (
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/transport"

	"go.uber.org/zap"
)

type TorrentManagerMobile struct {
	ArchiveManagerMobile
	logger *zap.Logger
}

func (tmm *TorrentManagerMobile) LogStdout(input string, fields ...zap.Field) {
	tmm.logger.Debug(input, fields...)
}

func (tmm *TorrentManagerMobile) SetOnline(online bool) {}

func (tmm *TorrentManagerMobile) SetTorrentConfig(*params.TorrentConfig) {}

func (tmm *TorrentManagerMobile) StartTorrentClient() error {
	return nil
}

func (tmm *TorrentManagerMobile) Stop() error {
	return nil
}

func (tmm *TorrentManagerMobile) IsReady() bool {
	return false
}

func (tmm *TorrentManagerMobile) GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error) {
	return nil, nil
}

func (tmm *TorrentManagerMobile) GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error) {
	return nil, nil
}

func (tmm *TorrentManagerMobile) GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error) {
	return 0, nil
}

func (tmm *TorrentManagerMobile) CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	return nil
}

func (tmm *TorrentManagerMobile) StartHistoryArchiveTasksInterval(community *Community, interval time.Duration) {
}

func (tmm *TorrentManagerMobile) StopHistoryArchiveTasksInterval(communityID types.HexBytes) {}

func (tmm *TorrentManagerMobile) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
	return nil
}

func (tmm *TorrentManagerMobile) UnseedHistoryArchiveTorrent(communityID types.HexBytes) {}

func (tmm *TorrentManagerMobile) IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool {
	return false
}

func (tmm *TorrentManagerMobile) GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask {
	return nil
}

func (tmm *TorrentManagerMobile) AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask) {
}

func (tmm *TorrentManagerMobile) DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {
	return nil, nil
}

func (tmm *TorrentManagerMobile) TorrentFileExists(communityID string) bool {
	return false
}
