package archive

import (
	"context"
	"time"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// ArchiveManagerNop is a no-op implementation of ArchiveService.
// This type is always compiled (no build tags) so it can be used
// both when history archives are disabled at compile time, and when
// they're enabled but no configuration is provided at runtime.
type ArchiveManagerNop struct{}

// ArchiveServiceBackend interface implementation

func (m *ArchiveManagerNop) SetOnline(online bool) {}

func (m *ArchiveManagerNop) Start() error {
	return nil
}

func (m *ArchiveManagerNop) Stop() error {
	return nil
}

func (m *ArchiveManagerNop) IsStarted() bool {
	return false
}

func (m *ArchiveManagerNop) IsReady() bool {
	return false
}

func (m *ArchiveManagerNop) SeedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) error {
	return nil
}

func (m *ArchiveManagerNop) UnseedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) {
}

func (m *ArchiveManagerNop) IsSeedingHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) bool {
	return false
}

func (m *ArchiveManagerNop) DownloadHistoryArchives(communityID cryptotypes.HexBytes, archiveLink string, cancelTask chan struct{}) (*archivetypes.HistoryArchiveDownloadTaskInfo, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) CreateHistoryArchiveFromMessages(communityID cryptotypes.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) CreateHistoryArchiveFromDB(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) LoadArchiveMessages(ctx context.Context, communityID cryptotypes.HexBytes, archiveLink string, downloadedArchiveID string) ([]*protobuf.WakuMessage, error) {
	return nil, nil
}

// ArchiveService interface implementation

func (m *ArchiveManagerNop) GetHistoryArchiveLink(communityID cryptotypes.HexBytes) (string, error) {
	return "", nil
}

func (m *ArchiveManagerNop) GetCommunityChatsFilters(communityID cryptotypes.HexBytes) (messagingtypes.ChatFilters, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) GetCommunityChatsTopics(communityID cryptotypes.HexBytes) ([]messagingtypes.ContentTopic, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) GetHistoryArchivePartitionStartTimestamp(communityID cryptotypes.HexBytes) (uint64, error) {
	return 0, nil
}

func (m *ArchiveManagerNop) CreateAndSeedHistoryArchive(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	return nil
}

func (m *ArchiveManagerNop) StartHistoryArchiveTasksInterval(communityID cryptotypes.HexBytes, chatID string, encrypted bool, interval time.Duration) {
}

func (m *ArchiveManagerNop) StopHistoryArchiveTasksInterval(communityID cryptotypes.HexBytes) {}

func (m *ArchiveManagerNop) GetHistoryArchiveDownloadTask(communityID string) *archivetypes.HistoryArchiveDownloadTask {
	return nil
}

func (m *ArchiveManagerNop) AddHistoryArchiveDownloadTask(communityID string, task *archivetypes.HistoryArchiveDownloadTask) {
}

func (m *ArchiveManagerNop) RemoveHistoryArchiveDownloadTask(communityID string) {
}

func (m *ArchiveManagerNop) PublishHistoryArchivesSeedingSignal(communityID cryptotypes.HexBytes) {
}

func (m *ArchiveManagerNop) GetDownloadedMessageArchiveIDs(communityID cryptotypes.HexBytes) ([]string, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) SaveMessageArchiveID(communityID cryptotypes.HexBytes, hash string) error {
	return nil
}

func (m *ArchiveManagerNop) GetMessageArchiveIDsToImport(communityID cryptotypes.HexBytes) ([]string, error) {
	return nil, nil
}

func (m *ArchiveManagerNop) SetMessageArchiveIDImported(communityID cryptotypes.HexBytes, hash string, imported bool) error {
	return nil
}

func (m *ArchiveManagerNop) GetHistoryTasksCount() int {
	return 0
}
