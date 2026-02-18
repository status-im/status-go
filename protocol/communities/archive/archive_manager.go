//go:build !disable_history_archives
// +build !disable_history_archives

package archive

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/messaging"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	archivelogosstorage "github.com/status-im/status-go/protocol/communities/archive/logosstorage"
	archivetorrent "github.com/status-im/status-go/protocol/communities/archive/torrent"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	archiveutils "github.com/status-im/status-go/protocol/communities/archive/utils"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/signal"
)

type ArchiveManager struct {
	historyArchiveDownloadTasks  map[string]*archivetypes.HistoryArchiveDownloadTask
	downloadTasksMu              sync.RWMutex // protects historyArchiveDownloadTasks
	historyArchiveTasksWaitGroup sync.WaitGroup
	historyArchiveTasks          sync.Map // stores `chan struct{}`

	logger      *zap.Logger
	persistence archivetypes.PersistenceProvider
	messaging   *messaging.API
	identity    *ecdsa.PrivateKey

	publisher archivetypes.HistoryArchivePublisher
	backend   ArchiveServiceBackend
}

// NewArchiveManager this function is only built and called when the "disable_history_archives" build tag is not set.
// In this case this version of NewArchiveManager will return a fully functional ArchiveManager ensuring that the
// build command will import and build the archive dependencies.
// NOTE: It is intentional that this file contains the identical function name as in "archive_manager_nop.go"
//
// This function implements the NOP pattern: it ALWAYS returns a valid ArchiveService instance,
// never nil. When archive functionality is not enabled (via config.Enabled field), it returns
// the ArchiveManagerNop which safely does nothing for all operations.
// This eliminates the need for nil/Enabled checks throughout the codebase.
func NewArchiveManager(amc *archivetypes.ArchiveManagerConfig) ArchiveService {
	// Depending on which config is provided AND enabled, we instantiate the corresponding
	// concrete ArchiveManager backend implementation.
	var backend ArchiveServiceBackend

	if amc.TorrentConfig != nil && amc.TorrentConfig.Enabled {
		// Torrent-based archive backend
		backend = archivetorrent.NewArchiveManagerTorrent(
			amc.TorrentConfig,
			amc.Logger,
			amc.Persistence,
			amc.Messaging,
			amc.Identity,
			amc.Publisher,
		)
	} else if amc.LogosStorageConfig != nil && amc.LogosStorageConfig.Enabled {
		// LogosStorage-based archive backend
		backend = archivelogosstorage.NewArchiveManagerLogosStorage(
			amc.LogosStorageConfig,
			amc.Logger,
			amc.Persistence,
			amc.Messaging,
			amc.Identity,
			amc.Publisher,
		)
	} else {
		// No enabled configuration - return the NOP implementation
		// This ensures we always return a valid instance that safely does nothing
		return &ArchiveManagerNop{}
	}

	return &ArchiveManager{
		historyArchiveDownloadTasks: make(map[string]*archivetypes.HistoryArchiveDownloadTask),

		logger:      amc.Logger,
		persistence: amc.Persistence,
		messaging:   amc.Messaging,
		identity:    amc.Identity,

		publisher: amc.Publisher,
		backend:   backend,
	}
}

// ArchiveServiceBackend interface implementation - delegates to backend

func (m *ArchiveManager) SetOnline(online bool) {
	m.backend.SetOnline(online)
}

func (m *ArchiveManager) Start() error {
	return m.backend.Start()
}

func (m *ArchiveManager) Stop() error {
	// stopHistoryArchiveTasksIntervals should be called unconditionally
	m.stopHistoryArchiveTasksIntervals()
	return m.backend.Stop()
}

func (m *ArchiveManager) IsStarted() bool {
	return m.backend.IsStarted()
}

func (m *ArchiveManager) SeedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) error {
	return m.backend.SeedHistoryArchive(communityID, archiveLink)
}

func (m *ArchiveManager) UnseedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) {
	m.backend.UnseedHistoryArchive(communityID, archiveLink)
}

func (m *ArchiveManager) IsSeedingHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) bool {
	return m.backend.IsSeedingHistoryArchive(communityID, archiveLink)
}

func (m *ArchiveManager) DownloadHistoryArchives(communityID cryptotypes.HexBytes, archiveLink string, cancelTask chan struct{}) (*archivetypes.HistoryArchiveDownloadTaskInfo, error) {
	return m.backend.DownloadHistoryArchives(communityID, archiveLink, cancelTask)
}

func (m *ArchiveManager) CreateHistoryArchiveFromMessages(communityID cryptotypes.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.backend.CreateHistoryArchiveFromMessages(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManager) CreateHistoryArchiveFromDB(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.backend.CreateHistoryArchiveFromDB(communityID, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManager) LoadArchiveMessages(ctx context.Context, communityID cryptotypes.HexBytes, archiveLink string, downloadedArchiveID string) ([]*protobuf.WakuMessage, error) {
	return m.backend.LoadArchiveMessages(ctx, communityID, archiveLink, downloadedArchiveID)
}

// func (m *ArchiveManager) GetHistoryArchiveId(communityID cryptotypes.HexBytes) (string, error) {
// 	return m.backend.GetHistoryArchiveId(communityID)
// }

// ArchiveService interface implementation - storage-agnostic operations

func (m *ArchiveManager) GetHistoryArchiveLink(communityID cryptotypes.HexBytes) (string, error) {
	return m.persistence.GetLastSeenArchiveLink(communityID)
}

func (m *ArchiveManager) GetCommunityChatsFilters(communityID cryptotypes.HexBytes) (messagingtypes.ChatFilters, error) {
	chatIDs, err := m.persistence.GetCommunityChatIDs(communityID)
	if err != nil {
		return nil, err
	}

	filters := messagingtypes.ChatFilters{}
	for _, cid := range chatIDs {
		filter := m.messaging.ChatFilterByChatID(cid)
		if filter != nil {
			filters = append(filters, filter)
		}
	}
	return filters, nil
}

func (m *ArchiveManager) GetCommunityChatsTopics(communityID cryptotypes.HexBytes) ([]messagingtypes.ContentTopic, error) {
	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		return nil, err
	}

	topics := []messagingtypes.ContentTopic{}
	for _, filter := range filters {
		topics = append(topics, filter.ContentTopic())
	}

	return topics, nil
}

func (m *ArchiveManager) GetHistoryArchivePartitionStartTimestamp(communityID cryptotypes.HexBytes) (uint64, error) {
	exists, err := m.persistence.CommunityExists(&m.identity.PublicKey, communityID)
	if err != nil {
		m.logger.Error("failed to check community existence", zap.Error(err))
		return 0, err
	}

	if !exists {
		m.logger.Error("community not found for this id")
		return 0, errors.New("community not found")
	}

	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		m.logger.Error("failed to get community chats filters", zap.Error(err))
		return 0, err
	}

	universalChatID := archiveutils.UniversalChatIDFromCommunityID(communityID)
	filter := m.messaging.ChatFilterByChatID(universalChatID)
	if filter != nil {
		filters = append(filters, filter)
	}

	if len(filters) == 0 {
		// If we don't have chat filters, we likely don't have any chats
		// associated to this community, which means there's nothing more
		// to do here
		return 0, nil
	}

	topics := []messagingtypes.ContentTopic{}
	for _, filter := range filters {
		topics = append(topics, filter.ContentTopic())
	}

	lastArchiveEndDateTimestamp, err := m.getLastMessageArchiveEndDate(communityID)
	if err != nil {
		m.logger.Error("failed to get last archive end date", zap.Error(err))
		return 0, err
	}

	if lastArchiveEndDateTimestamp == 0 {
		// If we don't have a tracked last message archive end date, it
		// means we haven't created an archive before, which means
		// the next thing to look at is the oldest waku message timestamp for
		// this community
		lastArchiveEndDateTimestamp, err = m.getOldestWakuMessageTimestamp(topics)
		if err != nil {
			m.logger.Error("failed to get oldest waku message timestamp", zap.Error(err))
			return 0, err
		}
		if lastArchiveEndDateTimestamp == 0 {
			// This means there's no waku message stored for this community so far
			// (even after requesting possibly missed messages), so no messages exist yet that can be archived
			m.logger.Debug("can't find valid `lastArchiveEndTimestamp`")
			return 0, nil
		}
	}

	return lastArchiveEndDateTimestamp, nil
}

func (m *ArchiveManager) CreateAndSeedHistoryArchive(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	err := m.backend.CreateAndSeedHistoryArchive(communityID, topics, startDate, endDate, partition, encrypt)

	if err != nil {
		m.logger.Error("failed to create and seed history archive", zap.Error(err))
		return err
	}

	// one way of publishing index succeeded - we can publish the seeding signal
	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
		},
	})

	return nil
}

func (m *ArchiveManager) StartHistoryArchiveTasksInterval(communityID cryptotypes.HexBytes, chatID string, encrypted bool, interval time.Duration) {
	defer common.LogOnPanic()
	id := cryptotypes.EncodeHex(communityID)

	if _, exists := m.historyArchiveTasks.Load(id); exists {
		m.logger.Error("history archive tasks interval already in progress", zap.String("id", id))
		return
	}

	cancel := make(chan struct{})
	m.historyArchiveTasks.Store(id, cancel)
	m.historyArchiveTasksWaitGroup.Add(1)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	m.logger.Debug("starting history archive tasks interval", zap.String("id", id))
	for {
		select {
		case <-ticker.C:
			m.logger.Debug("starting archive task...", zap.String("id", id))

			lastArchiveEndDateTimestamp, err := m.GetHistoryArchivePartitionStartTimestamp(communityID)
			if err != nil {
				m.logger.Error("failed to get last archive end date", zap.Error(err))
				continue
			}

			if lastArchiveEndDateTimestamp == 0 {
				// This means there are no waku messages for this community,
				// so nothing to do here
				m.logger.Debug("couldn't determine archive start date - skipping")
				continue
			}

			topics, err := m.GetCommunityChatsTopics(communityID)
			if err != nil {
				m.logger.Error("failed to get community chat topics ", zap.Error(err))
				continue
			}
			filter := m.messaging.ChatFilterByChatID(chatID)
			if filter == nil {
				m.logger.Error("failed to get chat filter", zap.String("community's UniversalChatID", chatID))
				continue
			}
			// adding the content-topic used for member updates.
			// since member updates would not be too frequent i.e only addition/deletion would add a new message,
			// this shouldn't cause too much increase in size of archive generated.
			topics = append(topics, filter.ContentTopic())

			ts := time.Now().Unix()
			to := time.Unix(ts, 0)
			lastArchiveEndDate := time.Unix(int64(lastArchiveEndDateTimestamp), 0)

			err = m.CreateAndSeedHistoryArchive(communityID, topics, lastArchiveEndDate, to, interval, encrypted)
			if err != nil {
				m.logger.Error("failed to create and seed history archive", zap.Error(err))
				continue
			}
		case <-cancel:
			lastSeenArchiveLink, err := m.persistence.GetLastSeenArchiveLink(communityID)
			if err != nil {
				m.logger.Debug("[LogosStorage][start_history_archive_tasks_interval] failed to get last seen archive link - proceeding without un-seeding", zap.Error(err))
			} else {
				m.UnseedHistoryArchive(communityID, lastSeenArchiveLink)
			}
			m.historyArchiveTasks.Delete(id)
			m.historyArchiveTasksWaitGroup.Done()
			return
		}
	}
}

func (m *ArchiveManager) StopHistoryArchiveTasksInterval(communityID cryptotypes.HexBytes) {
	task, exists := m.historyArchiveTasks.Load(communityID.String())
	if exists {
		m.logger.Info("Stopping history archive tasks interval", zap.Any("id", communityID.String()))
		close(task.(chan struct{})) // Need to cast to the chan
	}
}

func (m *ArchiveManager) GetHistoryArchiveDownloadTask(communityID string) *archivetypes.HistoryArchiveDownloadTask {
	m.downloadTasksMu.RLock()
	defer m.downloadTasksMu.RUnlock()
	return m.historyArchiveDownloadTasks[communityID]
}

func (m *ArchiveManager) AddHistoryArchiveDownloadTask(communityID string, task *archivetypes.HistoryArchiveDownloadTask) {
	m.downloadTasksMu.Lock()
	defer m.downloadTasksMu.Unlock()
	m.historyArchiveDownloadTasks[communityID] = task
}

func (m *ArchiveManager) RemoveHistoryArchiveDownloadTask(communityID string) {
	m.downloadTasksMu.Lock()
	defer m.downloadTasksMu.Unlock()
	delete(m.historyArchiveDownloadTasks, communityID)
}

func (m *ArchiveManager) PublishHistoryArchivesSeedingSignal(communityID cryptotypes.HexBytes) {
	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
		},
	})
}

func (m *ArchiveManager) GetDownloadedMessageArchiveIDs(communityID cryptotypes.HexBytes) ([]string, error) {
	return m.persistence.GetDownloadedMessageArchiveIDs(communityID)
}

func (m *ArchiveManager) SaveMessageArchiveID(communityID cryptotypes.HexBytes, hash string) error {
	return m.persistence.SaveMessageArchiveID(communityID, hash)
}

func (m *ArchiveManager) GetMessageArchiveIDsToImport(communityID cryptotypes.HexBytes) ([]string, error) {
	return m.persistence.GetMessageArchiveIDsToImport(communityID)
}

func (m *ArchiveManager) SetMessageArchiveIDImported(communityID cryptotypes.HexBytes, hash string, imported bool) error {
	return m.persistence.SetMessageArchiveIDImported(communityID, hash, imported)
}

// private methods
func (m *ArchiveManager) stopHistoryArchiveTasksIntervals() {
	m.historyArchiveTasks.Range(func(_, task interface{}) bool {
		close(task.(chan struct{})) // Need to cast to the chan
		return true
	})
	// Stoping archive interval tasks is async, so we need
	// to wait for all of them to be closed before we shutdown
	// the torrent client
	m.historyArchiveTasksWaitGroup.Wait()
}

func (m *ArchiveManager) getOldestWakuMessageTimestamp(topics []messagingtypes.ContentTopic) (uint64, error) {
	return m.persistence.GetOldestWakuMessageTimestamp(topics)
}

func (m *ArchiveManager) getLastMessageArchiveEndDate(communityID cryptotypes.HexBytes) (uint64, error) {
	return m.persistence.GetLastMessageArchiveEndDate(communityID)
}

// Special functions
// These functions are not part of the ArchiveServiceBackend interface.
// Some legacy tests are accessing implementation details and for this reason
// we need to expose these special accessors.

// GetLogosStorageBackend returns the LogosStorage backend if available, for test purposes
func (m *ArchiveManager) GetLogosStorageBackend() (*archivelogosstorage.ArchiveManagerLogosStorage, error) {
	if logosStorageBackend, ok := m.backend.(*archivelogosstorage.ArchiveManagerLogosStorage); ok {
		return logosStorageBackend, nil
	}
	return nil, errors.New("backend is not ArchiveManagerLogosStorage")
}

// GetTorrentBackend returns the Torrent backend if available, for test purposes
func (m *ArchiveManager) GetTorrentBackend() (*archivetorrent.ArchiveManagerTorrent, error) {
	if torrentBackend, ok := m.backend.(*archivetorrent.ArchiveManagerTorrent); ok {
		return torrentBackend, nil
	}
	return nil, errors.New("backend is not ArchiveManagerTorrent")
}
