package archive

//go:generate go tool mockgen -package=mock_archive -source=archive_service.go -destination=mock/archive/archive_service.go

import (
	"context"
	"time"

	"github.com/status-im/status-go/internal/crypto/types"
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

type ArchiveServiceBackend interface {
	SetOnline(bool)
	Start() error
	Stop() error
	IsStarted() bool
	SeedHistoryArchive(communityID types.HexBytes, archiveLink string) error
	UnseedHistoryArchive(communityID types.HexBytes, archiveLink string)
	IsSeedingHistoryArchive(communityID types.HexBytes, archiveLink string) bool
	DownloadHistoryArchives(communityID types.HexBytes, archiveLink string, cancelTask chan struct{}) (*archivetypes.HistoryArchiveDownloadTaskInfo, error)
	CreateHistoryArchiveFromMessages(communityID types.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error)
	CreateHistoryArchiveFromDB(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error)
	CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error
	LoadArchiveMessages(ctx context.Context, communityID types.HexBytes, archiveLink string, downloadedArchiveID string) ([]*protobuf.WakuMessage, error)
}

type ArchiveService interface {
	// ArchiveServiceBackend provides a proxy interface to the underlying storage-specific
	// implementation of the archive service.
	ArchiveServiceBackend
	// Storage-agnostic operations
	GetHistoryArchiveLink(communityID types.HexBytes) (string, error)
	GetCommunityChatsFilters(communityID types.HexBytes) (messagingtypes.ChatFilters, error)
	GetCommunityChatsTopics(communityID types.HexBytes) ([]messagingtypes.ContentTopic, error)
	GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error)
	CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error
	StartHistoryArchiveTasksInterval(communityID types.HexBytes, chatID string, encrypted bool, interval time.Duration)
	StopHistoryArchiveTasksInterval(communityID types.HexBytes)
	GetHistoryArchiveDownloadTask(communityID string) *archivetypes.HistoryArchiveDownloadTask
	AddHistoryArchiveDownloadTask(communityID string, task *archivetypes.HistoryArchiveDownloadTask)
	RemoveHistoryArchiveDownloadTask(communityID string)
	PublishHistoryArchivesSeedingSignal(communityID types.HexBytes)
	GetDownloadedMessageArchiveIDs(communityID types.HexBytes) ([]string, error)
	SaveMessageArchiveID(communityID types.HexBytes, hash string) error
	GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error)
	SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error
	GetHistoryTasksCount() int
}
