//go:build disable_torrent
// +build disable_torrent

package communities

import (
	"crypto/ecdsa"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/transport"

	"go.uber.org/zap"
)

type TorrentManagerNop struct {
	ArchiveManagerNop
}

// NewTorrentManager this function is only built and called when the "disable_torrent" build tag is set
// In this case this version of NewTorrentManager will return the mobile "nil" TorrentManagerNop ensuring that the
// build command will not import or build the torrent deps for the mobile OS.
// NOTE: It is intentional that this file contains the identical function name as in "manager_torrent.go"
func NewTorrentManager(torrentConfig *params.TorrentConfig, logger *zap.Logger, persistence *Persistence, transport *transport.Transport, identity *ecdsa.PrivateKey, encryptor *encryption.Protocol, publisher Publisher) *TorrentManagerNop {
	return &TorrentManagerNop{}
}

func (tmm *TorrentManagerNop) SetOnline(online bool) {}

func (tmm *TorrentManagerNop) SetTorrentConfig(*params.TorrentConfig) {}

func (tmm *TorrentManagerNop) StartTorrentClient() error {
	return nil
}

func (tmm *TorrentManagerNop) Stop() error {
	return nil
}

func (tmm *TorrentManagerNop) IsReady() bool {
	return false
}

func (tmm *TorrentManagerNop) GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error) {
	return nil, nil
}

func (tmm *TorrentManagerNop) GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error) {
	return nil, nil
}

func (tmm *TorrentManagerNop) GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error) {
	return 0, nil
}

func (tmm *TorrentManagerNop) CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	return nil
}

func (tmm *TorrentManagerNop) StartHistoryArchiveTasksInterval(community *Community, interval time.Duration) {
}

func (tmm *TorrentManagerNop) StopHistoryArchiveTasksInterval(communityID types.HexBytes) {}

func (tmm *TorrentManagerNop) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
	return nil
}

func (tmm *TorrentManagerNop) UnseedHistoryArchiveTorrent(communityID types.HexBytes) {}

func (tmm *TorrentManagerNop) IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool {
	return false
}

func (tmm *TorrentManagerNop) GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask {
	return nil
}

func (tmm *TorrentManagerNop) AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask) {
}

func (tmm *TorrentManagerNop) DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {
	return nil, nil
}

func (tmm *TorrentManagerNop) TorrentFileExists(communityID string) bool {
	return false
}
