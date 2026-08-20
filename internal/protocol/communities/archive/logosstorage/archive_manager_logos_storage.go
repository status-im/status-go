//go:build !disable_history_archives && use_logos_storage

package logosstorage

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"go.uber.org/zap"

	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/panics"
	archviecommons "github.com/status-im/status-go/internal/protocol/communities/archive/commons"
	archiveconsts "github.com/status-im/status-go/internal/protocol/communities/archive/consts"
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
	archiveutils "github.com/status-im/status-go/internal/protocol/communities/archive/utils"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	logosstorage "github.com/status-im/status-go/pkg/services/logosstorage"
)

type ArchiveManagerLogosStorage struct {
	config          *params.LogosStorageConfig
	client          logosstorage.LogosStorageClientInterface
	clientMu        sync.RWMutex
	downloadTimeout time.Duration // timeout for archive downloads, defaults to 20s

	logger      *zap.Logger
	persistence archivetypes.PersistenceProvider
	messaging   *messaging.API
	identity    *ecdsa.PrivateKey

	publisher archivetypes.HistoryArchivePublisher
}

func NewArchiveManagerLogosStorage(
	config *params.LogosStorageConfig,
	logger *zap.Logger,
	persistence archivetypes.PersistenceProvider,
	messaging *messaging.API,
	identity *ecdsa.PrivateKey,
	publisher archivetypes.HistoryArchivePublisher,
) *ArchiveManagerLogosStorage {
	return &ArchiveManagerLogosStorage{
		config:          config,
		downloadTimeout: 20 * time.Second,

		logger:      logger,
		persistence: persistence,
		messaging:   messaging,
		identity:    identity,

		publisher: publisher,
	}
}

// ArchiveServiceBackend interface implementation

func (m *ArchiveManagerLogosStorage) SetOnline(online bool) {
	m.logger.Info("[LogosStorage][set_online] testing online status:", zap.Bool("online", online))
	if online {
		m.logger.Info("[LogosStorage][set_online] Online: checking if LogosStorage client needs to be started...")

		m.logger.Info("[LogosStorage][set_online]", zap.Bool("clientStarted", m.IsStarted()))

		if !m.IsStarted() {
			m.logger.Info("[LogosStorage][set_online] Starting LogosStorage client...")
			err := m.Start()
			if err != nil {
				m.logger.Error("[LogosStorage][set_online] couldn't start LogosStorage client", zap.Error(err))
			}
		}
	}
}

func (m *ArchiveManagerLogosStorage) Start() error {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	if m.config == nil {
		return fmt.Errorf("can't start LogosStorage client: missing LogosStorageConfig")
	}

	if m.client != nil {
		return nil
	}

	var err error
	cfgCopy := *m.config
	cfgCopy.NodeConfig = m.config.NodeConfig

	m.logger.Info("[LogosStorage][start] Using the following NodeConfig", zap.Any("config", cfgCopy.NodeConfig))

	client, err := logosstorage.NewLogosStorageClient(cfgCopy)
	if err != nil {
		return err
	}
	m.client = client

	if err := m.client.Start(); err != nil {
		m.client = nil
		return err
	}

	return nil
}

func (m *ArchiveManagerLogosStorage) Stop() error {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	errs := []error{}
	if m.client != nil {
		m.logger.Info("[LogosStorage][stop] Stopping LogosStorage client")

		e := m.client.Stop()
		if e != nil {
			errs = append(errs, e)
		}

		e = m.client.Destroy()
		if e != nil {
			errs = append(errs, e)
		}

		m.client = nil
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (m *ArchiveManagerLogosStorage) IsStarted() bool {
	m.clientMu.RLock()
	defer m.clientMu.RUnlock()
	return m.client != nil
}

func (m *ArchiveManagerLogosStorage) GetLogosStorageClient() logosstorage.LogosStorageClientInterface {
	m.clientMu.RLock()
	defer m.clientMu.RUnlock()
	return m.client
}

func (m *ArchiveManagerLogosStorage) SeedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) error {
	if archiveLink == "" {
		return nil
	}
	if !m.IsStarted() {
		return nil
	}
	// do not seed if already seeding
	if m.IsSeedingHistoryArchive(communityID, archiveLink) {
		return nil
	}

	// for the purpose of seeding, we just need to make sure that the index cid
	// is fetched to the LogosStorage node - LogosStorage will seed it by advertising it on DHT
	_, err := m.client.TriggerDownload(archiveLink)
	if err != nil {
		return err
	}
	return nil
}

func (m *ArchiveManagerLogosStorage) UnseedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) {
	if archiveLink == "" {
		return
	}
	if !m.IsStarted() {
		return
	}
	if !m.IsSeedingHistoryArchive(communityID, archiveLink) {
		return
	}
	m.logger.Debug("[LogosStorage][unseed_history_archive] Un-seeding index CID for community", zap.String("id", communityID.String()), zap.String("cid", archiveLink))

	err := m.client.RemoveCid(archiveLink)
	if err != nil {
		m.logger.Error("[LogosStorage][unseed_history_archive] failed to remove CID from LogosStorage", zap.Error(err))
	}
}

func (m *ArchiveManagerLogosStorage) IsSeedingHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) bool {
	if archiveLink == "" {
		return false
	}
	if !m.IsStarted() {
		return false
	}
	hasCid, err := m.client.HasCid(archiveLink)
	if err != nil {
		m.logger.Debug("[LogosStorage][is_seeding_history_archive] failed to verify LogosStorage CID availability", zap.String("communityID", communityID.String()), zap.String("cid", archiveLink), zap.Error(err))
		return false
	}
	return hasCid
}

func (m *ArchiveManagerLogosStorage) DownloadHistoryArchives(communityID cryptotypes.HexBytes, archiveLink string, cancelTask chan struct{}) (*archivetypes.HistoryArchiveDownloadTaskInfo, error) {
	downloadTaskInfo := &archivetypes.HistoryArchiveDownloadTaskInfo{
		TotalDownloadedArchivesCount: 0,
		TotalArchivesCount:           0,
		Cancelled:                    false,
	}

	index, err := m.downloadHistoryArchiveIndex(cancelTask, communityID, archiveLink)
	if err != nil {
		// check if error is due to cancellation
		if errors.Is(err, context.Canceled) {
			m.logger.Debug("[LogosStorage][download_history_archives] cancelled downloading index from LogosStorage")
			downloadTaskInfo.Cancelled = true
			return downloadTaskInfo, nil
		}
		return nil, err
	}

	existingArchiveIDs, err := m.persistence.GetDownloadedMessageArchiveIDs(
		communityID,
	)
	if err != nil {
		return nil, err
	}

	downloadTaskInfo.TotalDownloadedArchivesCount = len(existingArchiveIDs)
	downloadTaskInfo.TotalArchivesCount = len(index.Archives)

	if !m.hasNewArchives(existingArchiveIDs, index) {
		m.logger.Debug("[LogosStorage][download_history_archives] aborting download, no new archives")
		return downloadTaskInfo, nil
	}

	if err := m.downloadArchives(communityID, index, existingArchiveIDs, cancelTask, downloadTaskInfo); err != nil {
		return nil, err
	}
	return downloadTaskInfo, nil
}

func (m *ArchiveManagerLogosStorage) CreateHistoryArchiveFromMessages(communityID cryptotypes.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchive(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManagerLogosStorage) CreateHistoryArchiveFromDB(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchive(communityID, make([]*messagingtypes.ReceivedMessage, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManagerLogosStorage) CreateAndSeedHistoryArchive(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	oldArchiveLink, err := m.persistence.GetLastSeenArchiveLink(communityID)
	oldArchiveLinkValid := err == nil
	if err != nil {
		m.logger.Debug("[LogosStorage][create_and_seed_history_archive] failed to get last seen archive link - proceeding without old index cleanup", zap.Error(err))
	}

	archiveIDs, err := m.CreateHistoryArchiveFromDB(communityID, topics, startDate, endDate, partition, encrypt)
	if err != nil {
		m.logger.Error("[LogosStorage][create_and_seed_history_archive] failed to create history archive LogosStorage", zap.Error(err))
		return err
	}

	if len(archiveIDs) == 0 {
		// No new LogosStorage archives were created; keep the previous index seeded.
		return nil
	}

	if oldArchiveLinkValid && oldArchiveLink != "" {
		newArchiveLink, err := m.persistence.GetLastSeenArchiveLink(communityID)
		if err != nil {
			m.logger.Debug("[LogosStorage][create_and_seed_history_archive] failed to get new archive link - skipping old index cleanup", zap.Error(err))
		} else if oldArchiveLink != newArchiveLink {
			m.UnseedHistoryArchive(communityID, oldArchiveLink)
		}
	}

	return nil
}

func (m *ArchiveManagerLogosStorage) LoadArchiveMessages(ctx context.Context, communityID cryptotypes.HexBytes, archiveLink string, downloadedArchiveID string) ([]*protobuf.WakuMessage, error) {
	archiveIndex, err := m.loadHistoryArchiveIndex(
		ctx, m.identity, communityID, archiveLink, true,
	)
	if err != nil {
		return nil, err
	}
	return m.extractMessagesFromHistoryArchive(communityID, downloadedArchiveID, archiveIndex)
}

// Private methods

func (m *ArchiveManagerLogosStorage) downloadHistoryArchiveIndex(
	cancelTask chan struct{},
	communityID cryptotypes.HexBytes,
	indexCid string,
) (*protobuf.LogosStorageWakuMessageArchiveIndex, error) {
	indexCtx, cancel := context.WithTimeout(context.Background(), m.downloadTimeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer panics.LogOnPanic()
		select {
		case <-cancelTask:
			m.logger.Debug("[LogosStorage][download_history_archive_index] cancelling downloading index from LogosStorage")
			cancel()
		case <-done:
		}
	}()

	index, err := m.loadHistoryArchiveIndex(indexCtx,
		m.identity, communityID, indexCid, false)
	close(done)

	if err != nil {
		// check if error is due to timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, archviecommons.ErrArchiveTimedout
		}
		return nil, err
	}

	// Publish index download completed signal
	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		IndexDownloadCompletedSignal: &signal.IndexDownloadCompletedSignal{
			CommunityID: communityID.String(),
			IndexCid:    indexCid,
		},
	})
	return index, nil
}

func (m *ArchiveManagerLogosStorage) hasNewArchives(existingArchiveIDs []string, index *protobuf.LogosStorageWakuMessageArchiveIndex) bool {
	existingArchiveIDSet := make(map[string]struct{}, len(existingArchiveIDs))
	for _, archiveID := range existingArchiveIDs {
		existingArchiveIDSet[archiveID] = struct{}{}
	}

	for archiveID := range index.Archives {
		if _, ok := existingArchiveIDSet[archiveID]; !ok {
			return true
		}
	}

	return false
}

func (m *ArchiveManagerLogosStorage) newArchiveDownloader(
	communityID cryptotypes.HexBytes,
	index *protobuf.LogosStorageWakuMessageArchiveIndex,
	existingArchiveIDs []string,
) (*logosstorage.LogosStorageArchiveDownloader, chan struct{}) {
	id := communityID.String()

	archiveDownloaderCancel := make(chan struct{})

	archiveDownloader := logosstorage.NewLogosStorageArchiveDownloader(
		m.client, index, id, existingArchiveIDs,
		archiveDownloaderCancel, m.logger,
	)

	archiveDownloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		err := m.persistence.SaveMessageArchiveID(communityID, hash)
		if err != nil {
			m.logger.Error("[LogosStorage][new_archive_downloader] couldn't save message archive ID", zap.Error(err))
		}
		m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
			HistoryArchiveDownloadedSignal: &signal.HistoryArchiveDownloadedSignal{
				CommunityID: communityID.String(),
				From:        int(from),
				To:          int(to),
			},
		})

		m.logger.Debug("[LogosStorage][new_archive_downloader] archive downloaded successfully",
			zap.String("hash", hash),
			zap.Uint64("from", from),
			zap.Uint64("to", to))
	})

	return archiveDownloader, archiveDownloaderCancel
}

func (m *ArchiveManagerLogosStorage) waitForArchiveDownloads(
	archiveDownloader *logosstorage.LogosStorageArchiveDownloader,
	archiveDownloaderCancel chan struct{},
	cancelTask chan struct{},
	downloadTaskInfo *archivetypes.HistoryArchiveDownloadTaskInfo,
) error {
	timeout := time.After(m.downloadTimeout)

	archiveTicker := time.NewTicker(1 * time.Second)
	defer archiveTicker.Stop()

	for {
		select {
		case <-timeout:
			return archviecommons.ErrArchiveTimedout
		case <-cancelTask:
			m.logger.Debug("[LogosStorage][wait_for_archive_downloads] cancelled downloading individual archives")
			close(archiveDownloaderCancel)
			downloadTaskInfo.TotalDownloadedArchivesCount = archiveDownloader.GetTotalDownloadedArchivesCount()
			downloadTaskInfo.Cancelled = true
			return nil
		case <-archiveTicker.C:
			// IsDownloadComplete == true also even when no single archive
			// has been downloaded (e.g. because of error or because of
			// cancellation).
			// To further check for cancellation, call IsCancelled().
			// Notice however that it does not make sense to check for
			// IsCancelled() here, because we would have already returned
			// above (<-cancelTask) in that case: this where
			// close(archiveDownloaderCancel) is called to stop the downloader.
			// To see if any archive was actually downloaded, check
			// GetTotalDownloadedArchivesCount().
			// Notice that GetTotalDownloadedArchivesCount represents
			// all successfully downloaded archives so far, not only
			// archives downloaded in this session.
			if archiveDownloader.IsDownloadComplete() {
				// Always update final progress
				downloadTaskInfo.TotalDownloadedArchivesCount = archiveDownloader.GetTotalDownloadedArchivesCount()

				m.logger.Info("[LogosStorage][wait_for_archive_downloads] downloading archives from LogosStorage completed",
					zap.Int("totalArchives", downloadTaskInfo.TotalArchivesCount),
					zap.Int("downloadedArchives", downloadTaskInfo.TotalDownloadedArchivesCount))

				return nil
			} else {
				// Update progress
				downloadTaskInfo.TotalDownloadedArchivesCount = archiveDownloader.GetTotalDownloadedArchivesCount()
				m.logger.Debug(
					"[LogosStorage][wait_for_archive_downloads] downloading archives in progress",
					zap.Int("completed", downloadTaskInfo.TotalDownloadedArchivesCount),
					zap.Int("total", downloadTaskInfo.TotalArchivesCount),
					zap.Int(
						"inProgress in this session",
						archiveDownloader.GetPendingArchivesCount(),
					),
					zap.Int(
						"total remaining archives to download",
						downloadTaskInfo.TotalArchivesCount-
							downloadTaskInfo.TotalDownloadedArchivesCount,
					),
				)
			}
		}
	}
}

func (m *ArchiveManagerLogosStorage) downloadArchives(
	communityID cryptotypes.HexBytes,
	index *protobuf.LogosStorageWakuMessageArchiveIndex,
	existingArchiveIDs []string,
	cancelTask chan struct{},
	downloadTaskInfo *archivetypes.HistoryArchiveDownloadTaskInfo,
) error {
	archiveDownloader, archiveDownloaderCancel := m.newArchiveDownloader(communityID, index, existingArchiveIDs)

	m.logger.Debug("[LogosStorage][download_archives] starting downloading individual archives from LogosStorage")

	archiveDownloader.StartDownload()

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		DownloadingHistoryArchivesStartedSignal: &signal.DownloadingHistoryArchivesStartedSignal{
			CommunityID: communityID.String(),
		},
	})

	return m.waitForArchiveDownloads(archiveDownloader, archiveDownloaderCancel, cancelTask, downloadTaskInfo)
}

func (m *ArchiveManagerLogosStorage) extractMessagesFromHistoryArchive(communityID cryptotypes.HexBytes, archiveID string, archiveIndex *protobuf.LogosStorageWakuMessageArchiveIndex) ([]*protobuf.WakuMessage, error) {
	metadata, ok := archiveIndex.Archives[archiveID]
	if !ok || metadata == nil {
		return nil, fmt.Errorf("archive %s missing from LogosStorage index", archiveID)
	}
	cid := metadata.Cid

	var buf bytes.Buffer
	err := m.client.LocalDownload(cid, &buf)
	if err != nil {
		m.logger.Error("[LogosStorage][extract_messages_from_history_archive] failed to download archive from LogosStorage", zap.Error(err))
		return nil, err
	}
	data := buf.Bytes()

	m.logger.Debug(
		"[LogosStorage][extract_messages_from_history_archive] extracting messages from history archive",
		zap.String("communityID", communityID.String()),
		zap.String("archiveID", archiveID),
		zap.String("cid", cid),
	)

	archive := &protobuf.WakuMessageArchive{}

	err = proto.Unmarshal(data, archive)
	if err != nil {
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			m.logger.Error("[LogosStorage][extract_messages_from_history_archive] failed to decompress community pubkey", zap.Error(err))
			return nil, err
		}

		decryptedData, err := m.messaging.DecryptMessage(m.identity, pk, data)
		if err != nil {
			m.logger.Error("[LogosStorage][extract_messages_from_history_archive] failed to decrypt message archive", zap.Error(err))
			return nil, err
		}

		err = proto.Unmarshal(decryptedData, archive)
		if err != nil {
			m.logger.Error("[LogosStorage][extract_messages_from_history_archive] failed to unmarshal message archive", zap.Error(err))
			return nil, err
		}
	}
	return archive.Messages, nil
}

func (m *ArchiveManagerLogosStorage) loadHistoryArchiveIndex(ctx context.Context, myKey *ecdsa.PrivateKey, communityID cryptotypes.HexBytes, indexCid string, isLocal bool) (*protobuf.LogosStorageWakuMessageArchiveIndex, error) {
	indexProto := &protobuf.LogosStorageWakuMessageArchiveIndex{}

	indexDownloader := logosstorage.NewLogosStorageIndexDownloader(m.client, m.logger)

	var indexBuf bytes.Buffer
	if isLocal {
		if err := indexDownloader.DownloadIndexFileFromLocalNode(ctx, indexCid, &indexBuf); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, archviecommons.ErrArchiveTimedout
			}
			return nil, err
		}
	} else {
		if err := indexDownloader.DownloadIndexFileFromNetwork(ctx, indexCid, &indexBuf); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, archviecommons.ErrArchiveTimedout
			}
			return nil, err
		}
	}
	indexData := indexBuf.Bytes()

	err := proto.Unmarshal(indexData, indexProto)
	if err != nil {
		return nil, err
	}

	if len(indexProto.Archives) == 0 && len(indexData) > 0 {
		// This means we're dealing with an encrypted index file, so we have to decrypt it first
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			return nil, err
		}

		decryptedData, err := m.messaging.DecryptMessage(myKey, pk, indexData)
		if err != nil {
			m.logger.Error("[LogosStorage][load_history_archive_index] failed to decrypt message archive", zap.Error(err))
			return nil, err
		}

		err = proto.Unmarshal(decryptedData, indexProto)
		if err != nil {
			return nil, err
		}
	}

	return indexProto, nil
}

func nextArchivePartitionEnd(from, endDate time.Time, partition time.Duration) time.Time {
	to := from.Add(partition)
	if to.After(endDate) {
		return endDate
	}
	return to
}

func (m *ArchiveManagerLogosStorage) loadExistingArchiveIndex(communityID cryptotypes.HexBytes) (*protobuf.LogosStorageWakuMessageArchiveIndex, map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata, error) {
	indexProto := &protobuf.LogosStorageWakuMessageArchiveIndex{}
	index := make(map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata)

	lastSeenArchiveLink, err := m.persistence.GetLastSeenArchiveLink(communityID)
	if err != nil {
		return indexProto, index, err
	}

	if m.IsSeedingHistoryArchive(communityID, lastSeenArchiveLink) {
		m.logger.Debug("[LogosStorage][load_existing_archive_index] LogosStorage index file exists, loading from file")
		ctx, cancel := context.WithTimeout(context.Background(), m.downloadTimeout)
		defer cancel()
		indexProto, err = m.loadHistoryArchiveIndex(ctx, m.identity, communityID, lastSeenArchiveLink, true)
		if err != nil {
			return indexProto, index, err
		}
	}

	maps.Copy(index, indexProto.Archives)
	return indexProto, index, nil
}

func (m *ArchiveManagerLogosStorage) updateLastMessageArchiveEndDate(communityID cryptotypes.HexBytes, from time.Time) error {
	lastMessageArchiveEndDate, err := m.persistence.GetLastMessageArchiveEndDate(communityID)
	if err != nil {
		return err
	}

	m.logger.Debug("[LogosStorage][update_last_message_archive_end_date] updating lastMessageArchiveEndDate", zap.Uint64("lastMessageArchiveEndDate", lastMessageArchiveEndDate))
	err = m.persistence.UpdateLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	if err != nil {
		return err
	}
	return nil
}

func (m *ArchiveManagerLogosStorage) messagesForArchivePartition(
	msgs []*messagingtypes.ReceivedMessage,
	topics []messagingtypes.ContentTopic,
	from time.Time,
	to time.Time,
) ([]messagingtypes.ReceivedMessage, error) {
	loadFromDB := len(msgs) == 0
	if loadFromDB {
		return m.persistence.GetWakuMessagesByFilterTopic(topics, uint64(from.Unix()), uint64(to.Unix()))
	}
	var messages []messagingtypes.ReceivedMessage
	for _, msg := range msgs {
		if int64(msg.Timestamp) >= from.Unix() && int64(msg.Timestamp) < to.Unix() {
			messages = append(messages, *msg)
		}
	}
	return messages, nil
}

func (m *ArchiveManagerLogosStorage) createHistoryArchive(communityID cryptotypes.HexBytes, msgs []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	from := startDate
	to := nextArchivePartitionEnd(from, endDate, partition)

	archiveIDs := make([]string, 0)

	indexProto, archiveIndex, err := m.loadExistingArchiveIndex(communityID)
	if err != nil {
		return archiveIDs, err
	}

	topicsAsByteArrays := archiveutils.TopicsAsByteArrays(topics)

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
		CommunityID: communityID.String(),
	}})

	m.logger.Debug(
		"[LogosStorage][create_history_archive] creating archives",
		zap.Any("startDate", startDate),
		zap.Any("endDate", endDate),
		zap.Duration("partition", partition),
	)
	for from.Before(endDate) {
		m.logger.Debug(
			"creating message archive",
			zap.Any("from", from),
			zap.Any("to", to),
		)

		messages, err := m.messagesForArchivePartition(msgs, topics, from, to)
		if err != nil {
			return archiveIDs, err
		}

		if len(messages) == 0 {
			// No need to create an archive with zero messages
			m.logger.Debug("[LogosStorage][create_history_archive] no messages in this partition")
			from = to
			to = nextArchivePartitionEnd(from, endDate, partition)
			continue
		}

		m.logger.Debug("[LogosStorage][create_history_archive] creating LogosStorage archive with messages", zap.Int("messagesCount", len(messages)))

		chunks := m.chunkArchiveMessages(messages)
		newIDs, err := m.createArchivesForPartition(communityID, from, to, chunks, topicsAsByteArrays, encrypt, archiveIndex)
		if err != nil {
			return archiveIDs, err
		}
		archiveIDs = append(archiveIDs, newIDs...)

		from = to
		to = nextArchivePartitionEnd(from, endDate, partition)
	}

	if err := m.finalizeLogosStorageArchiveIndex(communityID, indexProto, archiveIndex, archiveIDs, startDate, endDate, encrypt); err != nil {
		return archiveIDs, err
	}

	if err := m.updateLastMessageArchiveEndDate(communityID, from); err != nil {
		return archiveIDs, err
	}
	return archiveIDs, nil
}

func (m *ArchiveManagerLogosStorage) chunkArchiveMessages(messages []messagingtypes.ReceivedMessage) [][]messagingtypes.ReceivedMessage {
	messageChunks := make([][]messagingtypes.ReceivedMessage, 0)
	currentChunkSize := 0
	currentChunk := make([]messagingtypes.ReceivedMessage, 0)

	for _, msg := range messages {
		msgSize := len(msg.Payload) + len(msg.Sig)
		m.logger.Debug(
			"[LogosStorage][chunk_archive_messages] message size",
			zap.Int("messageSize", msgSize),
			zap.String("contentTopic", string(msg.Topic[:])),
			zap.ByteString("payload[0:31]", msg.Payload[:min(32, len(msg.Payload))]),
		)
		if msgSize > archiveconsts.MaxArchiveSizeInBytes {
			// we drop messages this big
			m.logger.Debug("[LogosStorage][chunk_archive_messages] dropping message due to size", zap.Int("messageSize", msgSize))
			continue
		}

		if currentChunkSize+msgSize > archiveconsts.MaxArchiveSizeInBytes {
			messageChunks = append(messageChunks, currentChunk)
			currentChunk = make([]messagingtypes.ReceivedMessage, 0)
			currentChunkSize = 0
		}
		currentChunk = append(currentChunk, msg)
		currentChunkSize = currentChunkSize + msgSize
	}
	if len(currentChunk) > 0 {
		messageChunks = append(messageChunks, currentChunk)
	}
	return messageChunks
}

func (m *ArchiveManagerLogosStorage) createArchivesForPartition(
	communityID cryptotypes.HexBytes,
	from time.Time,
	to time.Time,
	messageChunks [][]messagingtypes.ReceivedMessage,
	topicsAsByteArrays [][]byte,
	encrypt bool,
	archiveIndex map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata,
) ([]string, error) {
	archiveIDs := make([]string, 0)

	for _, messages := range messageChunks {
		wakuMessageArchive := m.createWakuMessageArchive(from, to, messages, topicsAsByteArrays)
		encodedArchive, err := proto.Marshal(wakuMessageArchive)
		if err != nil {
			return archiveIDs, err
		}

		if encrypt {
			encodedArchive, err = m.messaging.BuildHashRatchetMessage(communityID, encodedArchive)
			if err != nil {
				return archiveIDs, err
			}
		}

		// upload archive to LogosStorage and get CID back
		cid, err := m.client.UploadArchive(encodedArchive)
		if err != nil {
			m.logger.Error("[LogosStorage][create_archives_for_partition] failed to upload to LogosStorage", zap.Error(err))
			return archiveIDs, err
		}

		m.logger.Debug("[LogosStorage][create_archives_for_partition] archive uploaded to LogosStorage", zap.String("cid", cid))

		archiveMetadata := &protobuf.LogosStorageWakuMessageArchiveIndexMetadata{
			Metadata: wakuMessageArchive.Metadata,
			Cid:      cid,
		}

		metadataBytes, err := proto.Marshal(archiveMetadata)
		if err != nil {
			return archiveIDs, err
		}

		archiveID := crypto.Keccak256Hash(metadataBytes).String()
		archiveIDs = append(archiveIDs, archiveID)
		archiveIndex[archiveID] = archiveMetadata
	}

	return archiveIDs, nil
}

func (m *ArchiveManagerLogosStorage) finalizeLogosStorageArchiveIndex(
	communityID cryptotypes.HexBytes,
	indexProto *protobuf.LogosStorageWakuMessageArchiveIndex,
	archiveIndex map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata,
	archiveIDs []string,
	startDate time.Time,
	endDate time.Time,
	encrypt bool,
) error {
	if len(archiveIDs) == 0 {
		m.logger.Debug("[LogosStorage][finalize_logos_storage_archive_index] no archives created")
		m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
			NoHistoryArchivesCreatedSignal: &signal.NoHistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
		return nil
	}

	indexProto.Archives = archiveIndex
	indexBytes, err := proto.Marshal(indexProto)
	if err != nil {
		return err
	}

	if encrypt {
		indexBytes, err = m.messaging.BuildHashRatchetMessage(communityID, indexBytes)
		if err != nil {
			return err
		}
	}

	// upload index file to LogosStorage
	cid, err := m.client.UploadArchive(indexBytes)
	if err != nil {
		m.logger.Error("[LogosStorage][finalize_logos_storage_archive_index] failed to upload to LogosStorage", zap.Error(err))
		return err
	}

	m.logger.Debug("[LogosStorage][finalize_logos_storage_archive_index] index uploaded to LogosStorage", zap.String("cid", cid))
	m.logger.Debug("[LogosStorage][finalize_logos_storage_archive_index] archives uploaded to LogosStorage", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

	m.logger.Debug("[LogosStorage][finalize_logos_storage_archive_index] updating last seen archive link", zap.String("cid", cid))
	err = m.persistence.UpdateLastSeenArchiveLink(communityID, cid)
	if err != nil {
		return err
	}

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
			CommunityID: communityID.String(),
			From:        int(startDate.Unix()),
			To:          int(endDate.Unix()),
		},
	})
	return nil
}

func (m *ArchiveManagerLogosStorage) createWakuMessageArchive(from time.Time, to time.Time, messages []messagingtypes.ReceivedMessage, topics [][]byte) *protobuf.WakuMessageArchive {
	var wakuMessages []*protobuf.WakuMessage

	for _, msg := range messages {
		wakuMessage := &protobuf.WakuMessage{
			Sig:          msg.Sig,
			Timestamp:    uint64(msg.Timestamp),
			Topic:        msg.Topic.Bytes(),
			Payload:      msg.Payload,
			Padding:      msg.Padding,
			Hash:         msg.Hash,
			ThirdPartyId: msg.ThirdPartyID,
		}
		wakuMessages = append(wakuMessages, wakuMessage)
	}

	metadata := protobuf.WakuMessageArchiveMetadata{
		From:         uint64(from.Unix()),
		To:           uint64(to.Unix()),
		ContentTopic: topics,
	}

	wakuMessageArchive := &protobuf.WakuMessageArchive{
		Metadata: &metadata,
		Messages: wakuMessages,
	}
	return wakuMessageArchive
}

// Special functions
// These functions are not part of the ArchiveServiceBackend interface.
// We still have some tests that are accessing implementation details and for this reason
// we need to expose these special accessors.

func (m *ArchiveManagerLogosStorage) GetClient() logosstorage.LogosStorageClientInterface {
	return m.GetLogosStorageClient()
}

func (m *ArchiveManagerLogosStorage) LoadHistoryArchiveIndex(ctx context.Context, myKey *ecdsa.PrivateKey, communityID cryptotypes.HexBytes, indexCid string, isLocal bool) (*protobuf.LogosStorageWakuMessageArchiveIndex, error) {
	return m.loadHistoryArchiveIndex(ctx, myKey, communityID, indexCid, isLocal)
}

func (m *ArchiveManagerLogosStorage) SetLogosStorageClient(client logosstorage.LogosStorageClientInterface) {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()
	m.client = client
}

func (m *ArchiveManagerLogosStorage) SetDownloadTimeout(timeout time.Duration) {
	m.downloadTimeout = timeout
}

func (m *ArchiveManagerLogosStorage) ChunkArchiveMessages(messages []messagingtypes.ReceivedMessage) [][]messagingtypes.ReceivedMessage {
	return m.chunkArchiveMessages(messages)
}
