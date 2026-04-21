package types

import (
	"crypto/ecdsa"
	"sync"

	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/signal"
)

type HistoryArchiveSignals struct {
	CreatingHistoryArchivesSignal            *signal.CreatingHistoryArchivesSignal
	HistoryArchivesCreatedSignal             *signal.HistoryArchivesCreatedSignal
	NoHistoryArchivesCreatedSignal           *signal.NoHistoryArchivesCreatedSignal
	HistoryArchivesSeedingSignal             *signal.HistoryArchivesSeedingSignal
	HistoryArchivesUnseededSignal            *signal.HistoryArchivesUnseededSignal
	HistoryArchiveDownloadedSignal           *signal.HistoryArchiveDownloadedSignal
	DownloadingHistoryArchivesStartedSignal  *signal.DownloadingHistoryArchivesStartedSignal
	DownloadingHistoryArchivesFinishedSignal *signal.DownloadingHistoryArchivesFinishedSignal
	ImportingHistoryArchiveMessagesSignal    *signal.ImportingHistoryArchiveMessagesSignal
}

type HistoryArchiveDownloadTask struct {
	CancelChan chan struct{}
	Waiter     sync.WaitGroup
	m          sync.RWMutex
	Cancelled  bool
}

func (t *HistoryArchiveDownloadTask) IsCancelled() bool {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.Cancelled
}

func (t *HistoryArchiveDownloadTask) Cancel() {
	t.m.Lock()
	defer t.m.Unlock()
	t.Cancelled = true
	close(t.CancelChan)
}

type HistoryArchiveDownloadTaskInfo struct {
	TotalDownloadedArchivesCount int
	TotalArchivesCount           int
	Cancelled                    bool
}

type PersistenceProvider interface {
	GetDownloadedMessageArchiveIDs(communityID cryptotypes.HexBytes) ([]string, error)
	SaveMessageArchiveID(communityID cryptotypes.HexBytes, hash string) error
	GetWakuMessagesByFilterTopic(topics []messagingtypes.ContentTopic, from uint64, to uint64) ([]messagingtypes.ReceivedMessage, error)
	GetLastMessageArchiveEndDate(communityID cryptotypes.HexBytes) (uint64, error)
	UpdateLastMessageArchiveEndDate(communityID cryptotypes.HexBytes, endDate uint64) error
	SaveLastMessageArchiveEndDate(communityID cryptotypes.HexBytes, endDate uint64) error
	GetLastSeenArchiveLink(communityID cryptotypes.HexBytes) (string, error)
	UpdateLastSeenArchiveLink(communityID cryptotypes.HexBytes, archiveLink string) error
	GetCommunityChatIDs(communityID cryptotypes.HexBytes) ([]string, error)
	CommunityExists(memberIdentity *ecdsa.PublicKey, id []byte) (bool, error)
	GetOldestWakuMessageTimestamp(topics []messagingtypes.ContentTopic) (uint64, error)
	GetMessageArchiveIDsToImport(communityID cryptotypes.HexBytes) ([]string, error)
	SetMessageArchiveIDImported(communityID cryptotypes.HexBytes, hash string, imported bool) error
}

type HistoryArchivePublisher interface {
	Publish(subscription *HistoryArchiveSignals)
}

type ArchiveManagerConfig struct {
	TorrentConfig *params.TorrentConfig
	Logger        *zap.Logger
	Persistence   PersistenceProvider
	Messaging     *messaging.API
	Identity      *ecdsa.PrivateKey
	Publisher     HistoryArchivePublisher
}
