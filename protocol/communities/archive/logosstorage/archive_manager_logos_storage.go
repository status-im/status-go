//go:build !disable_history_archives && use_logos_storage
// +build !disable_history_archives,use_logos_storage

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

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	archviecommons "github.com/status-im/status-go/protocol/communities/archive/commons"
	archiveconsts "github.com/status-im/status-go/protocol/communities/archive/consts"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	archiveutils "github.com/status-im/status-go/protocol/communities/archive/utils"
	"github.com/status-im/status-go/protocol/protobuf"
	logosstorage "github.com/status-im/status-go/services/logosstorage"
	"github.com/status-im/status-go/signal"
)

type ArchiveManagerLogosStorage struct {
	logosStorageConfig   *params.LogosStorageConfig
	logosStorageClient   logosstorage.LogosStorageClientInterface
	logosStorageClientMu sync.RWMutex
	downloadTimeout      time.Duration // timeout for archive downloads, defaults to 20s

	logger      *zap.Logger
	persistence archivetypes.PersistenceProvider
	messaging   *messaging.API
	identity    *ecdsa.PrivateKey

	publisher archivetypes.HistoryArchivePublisher
}

func NewArchiveManagerLogosStorage(
	logosStorageConfig *params.LogosStorageConfig,
	logger *zap.Logger,
	persistence archivetypes.PersistenceProvider,
	messaging *messaging.API,
	identity *ecdsa.PrivateKey,
	publisher archivetypes.HistoryArchivePublisher,
) *ArchiveManagerLogosStorage {
	return &ArchiveManagerLogosStorage{
		logosStorageConfig: logosStorageConfig,
		downloadTimeout:    20 * time.Second,

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

		m.logger.Info("[LogosStorage][set_online]", zap.Bool("logosStorageStarted", m.IsStarted()))

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
	m.logosStorageClientMu.Lock()
	defer m.logosStorageClientMu.Unlock()

	if m.logosStorageConfig == nil {
		return fmt.Errorf("can't start LogosStorage client: missing LogosStorageConfig")
	}

	if m.logosStorageClient != nil {
		return nil
	}

	var err error
	cfgCopy := *m.logosStorageConfig
	cfgCopy.NodeConfig = m.logosStorageConfig.NodeConfig

	m.logger.Info("[LogosStorage][start_logosstorage_client] Using the following NodeConfig", zap.Any("config", cfgCopy.NodeConfig))

	client, err := logosstorage.NewLogosStorageClient(cfgCopy)
	if err != nil {
		return err
	}
	m.logosStorageClient = client

	if err := m.logosStorageClient.Start(); err != nil {
		m.logosStorageClient = nil
		return err
	}

	return nil
}

func (m *ArchiveManagerLogosStorage) Stop() error {
	m.logosStorageClientMu.Lock()
	defer m.logosStorageClientMu.Unlock()

	errs := []error{}
	if m.logosStorageClient != nil {
		m.logger.Info("[LogosStorage] Stopping LogosStorage client")

		e := m.logosStorageClient.Stop()
		if e != nil {
			errs = append(errs, e)
		}

		e = m.logosStorageClient.Destroy()
		if e != nil {
			errs = append(errs, e)
		}

		m.logosStorageClient = nil
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (m *ArchiveManagerLogosStorage) IsStarted() bool {
	m.logosStorageClientMu.RLock()
	defer m.logosStorageClientMu.RUnlock()
	return m.logosStorageClient != nil
}

func (m *ArchiveManagerLogosStorage) GetLogosStorageClient() logosstorage.LogosStorageClientInterface {
	m.logosStorageClientMu.RLock()
	defer m.logosStorageClientMu.RUnlock()
	return m.logosStorageClient
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
	_, err := m.logosStorageClient.TriggerDownload(archiveLink)
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
	m.logger.Debug("[LogosStorage] Un-seeding index CID for community", zap.String("id", communityID.String()), zap.String("cid", archiveLink))

	err := m.logosStorageClient.RemoveCid(archiveLink)
	if err != nil {
		m.logger.Error("[LogosStorage] failed to remove CID from LogosStorage", zap.Error(err))
	}
}

func (m *ArchiveManagerLogosStorage) IsSeedingHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) bool {
	if archiveLink == "" {
		return false
	}
	if !m.IsStarted() {
		return false
	}
	hasCid, err := m.logosStorageClient.HasCid(archiveLink)
	if err != nil {
		m.logger.Debug("[LogosStorage] failed to verify LogosStorage CID availability", zap.String("communityID", communityID.String()), zap.String("cid", archiveLink), zap.Error(err))
		return false
	}
	return hasCid
}

func (m *ArchiveManagerLogosStorage) DownloadHistoryArchives(communityID cryptotypes.HexBytes, archiveLink string, cancelTask chan struct{}) (*archivetypes.HistoryArchiveDownloadTaskInfo, error) {
	id := communityID.String()

	downloadTaskInfo := &archivetypes.HistoryArchiveDownloadTaskInfo{
		TotalDownloadedArchivesCount: 0,
		TotalArchivesCount:           0,
		Cancelled:                    false,
	}

	indexCtx, cancel := context.WithTimeout(context.Background(), m.downloadTimeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer common.LogOnPanic()
		select {
		case <-cancelTask:
			m.logger.Debug("[LogosStorage] cancelling downloading index from LogosStorage")
			cancel()
		case <-done:
		}
	}()

	index, err := m.loadHistoryArchiveIndex(indexCtx,
		m.identity, communityID, archiveLink, false)
	close(done)
	if err != nil {
		// check if error is due to timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, archviecommons.ErrArchiveTimedout
		}
		// check if error is due to cancellation
		if errors.Is(err, context.Canceled) {
			m.logger.Debug("[LogosStorage] cancelled downloading index from LogosStorage")
			downloadTaskInfo.Cancelled = true
			return downloadTaskInfo, nil
		}
		return nil, err
	}

	// Publish index download completed signal
	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		IndexDownloadCompletedSignal: &signal.IndexDownloadCompletedSignal{
			CommunityID: communityID.String(),
			IndexCid:    archiveLink,
		},
	})

	existingArchiveIDs, err := m.persistence.GetDownloadedMessageArchiveIDs(
		communityID)
	if err != nil {
		return nil, err
	}

	downloadTaskInfo.TotalDownloadedArchivesCount = len(existingArchiveIDs)
	downloadTaskInfo.TotalArchivesCount = len(index.Archives)

	if len(existingArchiveIDs) == len(index.Archives) {
		m.logger.Debug("[LogosStorage] aborting download, no new archives")
		return downloadTaskInfo, nil
	}

	// Create separate cancel channel for the archive
	// downloader to avoid channel competition
	archiveDownloaderCancel := make(chan struct{})

	// Create the archive downloader using the protobuf index directly
	archiveDownloader := logosstorage.NewLogosStorageArchiveDownloader(
		m.logosStorageClient, index, id, existingArchiveIDs,
		archiveDownloaderCancel, m.logger)

	// Set up callback for when individual archives are downloaded
	archiveDownloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		err = m.persistence.SaveMessageArchiveID(communityID, hash)
		if err != nil {
			m.logger.Error("[LogosStorage] couldn't save message archive ID", zap.Error(err))
		}
		m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
			HistoryArchiveDownloadedSignal: &signal.HistoryArchiveDownloadedSignal{
				CommunityID: communityID.String(),
				From:        int(from),
				To:          int(to),
			},
		})

		m.logger.Debug("[LogosStorage] archive downloaded successfully",
			zap.String("hash", hash),
			zap.Uint64("from", from),
			zap.Uint64("to", to))
	})

	m.logger.Debug("[LogosStorage] starting downloading individual archives from LogosStorage")

	archiveDownloader.StartDownload()

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		DownloadingHistoryArchivesStartedSignal: &signal.DownloadingHistoryArchivesStartedSignal{
			CommunityID: communityID.String(),
		},
	})

	timeout := time.After(m.downloadTimeout)

	// Monitor archive download progress
	archiveTicker := time.NewTicker(1 * time.Second)
	defer archiveTicker.Stop()

	for {
		select {
		case <-timeout:
			return nil, archviecommons.ErrArchiveTimedout
		case <-cancelTask:
			m.logger.Debug("[LogosStorage] cancelled downloading individual archives")
			close(archiveDownloaderCancel)
			downloadTaskInfo.TotalDownloadedArchivesCount = archiveDownloader.GetTotalDownloadedArchivesCount()
			downloadTaskInfo.Cancelled = true
			return downloadTaskInfo, nil
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
				downloadTaskInfo.TotalDownloadedArchivesCount =
					archiveDownloader.GetTotalDownloadedArchivesCount()

				m.logger.Info("[LogosStorage] downloading archives from LogosStorage completed",
					zap.Int("totalArchives", downloadTaskInfo.TotalArchivesCount),
					zap.Int("downloadedArchives", downloadTaskInfo.TotalDownloadedArchivesCount))

				return downloadTaskInfo, nil
			} else {
				// Update progress
				downloadTaskInfo.TotalDownloadedArchivesCount =
					archiveDownloader.GetTotalDownloadedArchivesCount()
				m.logger.Debug("[LogosStorage] downloading archives in progress",
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

func (m *ArchiveManagerLogosStorage) CreateHistoryArchiveFromMessages(communityID cryptotypes.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveLogosStorage(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManagerLogosStorage) CreateHistoryArchiveFromDB(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveLogosStorage(communityID, make([]*messagingtypes.ReceivedMessage, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManagerLogosStorage) CreateAndSeedHistoryArchive(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	lastSeenArchiveLink, err := m.persistence.GetLastSeenArchiveLink(communityID)
	if err != nil {
		m.UnseedHistoryArchive(communityID, lastSeenArchiveLink)
	} else {
		if err != nil {
			m.logger.Debug("[LogosStorage][CreateAndSeedHistoryArchive] failed to get last seen archive link - proceeding without un-seeding", zap.Error(err))
		}
	}
	archiveCreatedSuccessfully := true
	archiveIDs, err := m.CreateHistoryArchiveFromDB(communityID, topics, startDate, endDate, partition, encrypt)
	if err != nil {
		archiveCreatedSuccessfully = false
		m.logger.Error("[LogosStorage][CreateAndSeedHistoryArchive] failed to create history archive LogosStorage", zap.Error(err))
	} else {
		if len(archiveIDs) == 0 {
			// no new LogosStorage archives were created - no need to distribute new index cid
			// but we need to (re)start seeding that we stopped above
			archiveCreatedSuccessfully = false
			m.logger.Debug("[LogosStorage][CreateAndSeedHistoryArchive] no new LogosStorage archive links were created - re-seeding existing archive link")
			if err = m.SeedHistoryArchive(communityID, lastSeenArchiveLink); err != nil {
				m.logger.Error("[LogosStorage][CreateAndSeedHistoryArchive] failed to seed existing history archive LogosStorage archive link", zap.Error(err))
			}
		}
		// else: we created new LogosStorage archives and they are already published to LogosStorage
		// in CreateHistoryArchiveLogosStorageFromDB (thus they are seeded)
	}

	if !archiveCreatedSuccessfully {
		return err
	}

	return nil
}

func (m *ArchiveManagerLogosStorage) LoadArchiveMessages(ctx context.Context, communityID cryptotypes.HexBytes, archiveLink string, downloadedArchiveID string) ([]*protobuf.WakuMessage, error) {
	logosStorageIndex, err := m.loadHistoryArchiveIndex(
		ctx, m.identity, communityID, archiveLink, true)
	if err != nil {
		return nil, err
	}
	return m.extractMessagesFromHistoryArchive(communityID, downloadedArchiveID, logosStorageIndex)
}

// func (m *ArchiveManagerLogosStorage) GetHistoryArchiveLink(communityID cryptotypes.HexBytes) (string, error) {
// 	return m.persistence.GetLastSeenArchiveLink(communityID)
// }

// Private methods
func (m *ArchiveManagerLogosStorage) extractMessagesFromHistoryArchive(communityID cryptotypes.HexBytes, archiveID string, logosStorageIndex *protobuf.LogosStorageWakuMessageArchiveIndex) ([]*protobuf.WakuMessage, error) {
	metadata, ok := logosStorageIndex.Archives[archiveID]
	if !ok || metadata == nil {
		return nil, fmt.Errorf("archive %s missing from LogosStorage index", archiveID)
	}
	cid := metadata.Cid

	var buf bytes.Buffer
	err := m.logosStorageClient.LocalDownload(cid, &buf)
	if err != nil {
		m.logger.Error("[LogosStorage] failed to download archive from LogosStorage", zap.Error(err))
		return nil, err
	}
	data := buf.Bytes()

	m.logger.Debug("extracting messages from history archive",
		zap.String("communityID", communityID.String()),
		zap.String("archiveID", archiveID),
		zap.String("cid", cid),
	)

	archive := &protobuf.WakuMessageArchive{}

	err = proto.Unmarshal(data, archive)
	if err != nil {
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			m.logger.Error("failed to decompress community pubkey", zap.Error(err))
			return nil, err
		}

		decryptedData, err := m.messaging.DecryptMessage(m.identity, pk, data)
		if err != nil {
			m.logger.Error("failed to decrypt message archive", zap.Error(err))
			return nil, err
		}

		err = proto.Unmarshal(decryptedData, archive)
		if err != nil {
			m.logger.Error("failed to unmarshal message archive", zap.Error(err))
			return nil, err
		}
	}
	return archive.Messages, nil
}

func (m *ArchiveManagerLogosStorage) loadHistoryArchiveIndex(ctx context.Context, myKey *ecdsa.PrivateKey, communityID cryptotypes.HexBytes, indexCid string, isLocal bool) (*protobuf.LogosStorageWakuMessageArchiveIndex, error) {
	logosStorageWakuMessageArchiveIndexProto := &protobuf.LogosStorageWakuMessageArchiveIndex{}

	indexDownloader := logosstorage.NewLogosStorageIndexDownloader(m.logosStorageClient, m.logger)

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

	err := proto.Unmarshal(indexData, logosStorageWakuMessageArchiveIndexProto)
	if err != nil {
		return nil, err
	}

	if len(logosStorageWakuMessageArchiveIndexProto.Archives) == 0 && len(indexData) > 0 {
		// This means we're dealing with an encrypted index file, so we have to decrypt it first
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			return nil, err
		}

		decryptedData, err := m.messaging.DecryptMessage(myKey, pk, indexData)
		if err != nil {
			m.logger.Error("failed to decrypt message archive", zap.Error(err))
			return nil, err
		}

		err = proto.Unmarshal(decryptedData, logosStorageWakuMessageArchiveIndexProto)
		if err != nil {
			return nil, err
		}
	}

	return logosStorageWakuMessageArchiveIndexProto, nil
}

func (m *ArchiveManagerLogosStorage) createHistoryArchiveLogosStorage(communityID cryptotypes.HexBytes, msgs []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

	loadFromDB := len(msgs) == 0

	from := startDate
	to := from.Add(partition)
	if to.After(endDate) {
		to = endDate
	}

	logosStorageWakuMessageArchiveIndexProto := &protobuf.LogosStorageWakuMessageArchiveIndex{}
	logosStorageWakuMessageArchiveIndex := make(map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata)
	logosStorageArchiveIDs := make([]string, 0)

	lastSeenArchiveLink, err := m.persistence.GetLastSeenArchiveLink(communityID)
	if err != nil {
		return logosStorageArchiveIDs, err
	}

	if m.IsSeedingHistoryArchive(communityID, lastSeenArchiveLink) {
		m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] LogosStorage index file exists, loading from file")
		ctx, cancel := context.WithTimeout(context.Background(), m.downloadTimeout)
		defer cancel()
		logosStorageWakuMessageArchiveIndexProto, err = m.loadHistoryArchiveIndex(ctx, m.identity, communityID, lastSeenArchiveLink, true)
		if err != nil {
			return logosStorageArchiveIDs, err
		}
	}

	maps.Copy(logosStorageWakuMessageArchiveIndex, logosStorageWakuMessageArchiveIndexProto.Archives)
	topicsAsByteArrays := archiveutils.TopicsAsByteArrays(topics)

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
		CommunityID: communityID.String(),
	}})

	m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] creating archives",
		zap.Any("startDate", startDate),
		zap.Any("endDate", endDate),
		zap.Duration("partition", partition),
	)
	for {
		if from.Equal(endDate) || from.After(endDate) {
			break
		}
		m.logger.Debug("creating message archive",
			zap.Any("from", from),
			zap.Any("to", to),
		)

		var messages []messagingtypes.ReceivedMessage
		if loadFromDB {
			messages, err = m.persistence.GetWakuMessagesByFilterTopic(topics, uint64(from.Unix()), uint64(to.Unix()))
			if err != nil {
				return logosStorageArchiveIDs, err
			}
		} else {
			for _, msg := range msgs {
				if int64(msg.Timestamp) >= from.Unix() && int64(msg.Timestamp) < to.Unix() {
					messages = append(messages, *msg)
				}
			}
		}

		if len(messages) == 0 {
			// No need to create an archive with zero messages
			m.logger.Debug("[LogosStorage] no messages in this partition")
			from = to
			to = to.Add(partition)
			if to.After(endDate) {
				to = endDate
			}
			continue
		}

		m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] creating LogosStorage archive with messages", zap.Int("messagesCount", len(messages)))

		// Not only do we partition messages, we also chunk them
		// roughly by size, such that each chunk will not exceed a given
		// size and archive data doesn't get too big
		messageChunks := make([][]messagingtypes.ReceivedMessage, 0)
		currentChunkSize := 0
		currentChunk := make([]messagingtypes.ReceivedMessage, 0)

		for _, msg := range messages {
			msgSize := len(msg.Payload) + len(msg.Sig)
			m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] message size",
				zap.Int("messageSize", msgSize),
				zap.String("contentTopic", string(msg.Topic[:])),
				zap.ByteString("payload[0:31]", msg.Payload[:min(32, len(msg.Payload))]),
			)
			if msgSize > archiveconsts.MaxArchiveSizeInBytes {
				// we drop messages this big
				m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] dropping message due to size", zap.Int("messageSize", msgSize))
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
		messageChunks = append(messageChunks, currentChunk)

		for _, messages := range messageChunks {
			wakuMessageArchive := m.createWakuMessageArchive(from, to, messages, topicsAsByteArrays)
			encodedArchive, err := proto.Marshal(wakuMessageArchive)
			if err != nil {
				return logosStorageArchiveIDs, err
			}

			if encrypt {
				encodedArchive, err = m.messaging.BuildHashRatchetMessage(communityID, encodedArchive)
				if err != nil {
					return logosStorageArchiveIDs, err
				}
			}

			// upload archive to LogosStorage and get CID back
			cid, err := m.logosStorageClient.UploadArchive(encodedArchive)
			if err != nil {
				m.logger.Error("[LogosStorage] failed to upload to LogosStorage", zap.Error(err))
				return logosStorageArchiveIDs, err
			}

			m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] archive uploaded to LogosStorage", zap.String("cid", cid))

			logosStorageWakuMessageArchiveIndexMetadata := &protobuf.LogosStorageWakuMessageArchiveIndexMetadata{
				Metadata: wakuMessageArchive.Metadata,
				Cid:      cid,
			}

			logosStorageWakuMessageArchiveIndexMetadataBytes, err := proto.Marshal(logosStorageWakuMessageArchiveIndexMetadata)
			if err != nil {
				return logosStorageArchiveIDs, err
			}

			logosStorageArchiveID := crypto.Keccak256Hash(logosStorageWakuMessageArchiveIndexMetadataBytes).String()
			logosStorageArchiveIDs = append(logosStorageArchiveIDs, logosStorageArchiveID)
			logosStorageWakuMessageArchiveIndex[logosStorageArchiveID] = logosStorageWakuMessageArchiveIndexMetadata
		}

		from = to
		to = to.Add(partition)
		if to.After(endDate) {
			to = endDate
		}
	}

	if len(logosStorageArchiveIDs) > 0 {
		logosStorageWakuMessageArchiveIndexProto.Archives = logosStorageWakuMessageArchiveIndex
		logosStorageIndexBytes, err := proto.Marshal(logosStorageWakuMessageArchiveIndexProto)
		if err != nil {
			return logosStorageArchiveIDs, err
		}

		if encrypt {
			logosStorageIndexBytes, err = m.messaging.BuildHashRatchetMessage(communityID, logosStorageIndexBytes)
			if err != nil {
				return logosStorageArchiveIDs, err
			}
		}

		// upload index file to LogosStorage
		cid, err := m.logosStorageClient.UploadArchive(logosStorageIndexBytes)
		if err != nil {
			m.logger.Error("[LogosStorage][createHistoryArchiveLogosStorage] failed to upload to LogosStorage", zap.Error(err))
			return logosStorageArchiveIDs, err
		}

		m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] index uploaded to LogosStorage", zap.String("cid", cid))
		m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] archives uploaded to LogosStorage", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

		m.logger.Debug("[LogosStorage][create_history_archive_logos_storage] updating last seen archive link", zap.String("cid", cid))
		err = m.persistence.UpdateLastSeenArchiveLink(communityID, cid)
		if err != nil {
			return logosStorageArchiveIDs, err
		}

		m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.logger.Debug("[LogosStorage][createHistoryArchiveLogosStorage] no archives created")
		m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
			NoHistoryArchivesCreatedSignal: &signal.NoHistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	}

	lastMessageArchiveEndDate, err := m.persistence.GetLastMessageArchiveEndDate(communityID)
	if err != nil {
		return logosStorageArchiveIDs, err
	}

	m.logger.Debug("[LogosStorage][create_history_archive_logosstorage] updating lastMessageArchiveEndDate", zap.Uint64("lastMessageArchiveEndDate", lastMessageArchiveEndDate))
	err = m.persistence.UpdateLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	if err != nil {
		return logosStorageArchiveIDs, err
	}
	return logosStorageArchiveIDs, nil
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
	m.logosStorageClientMu.Lock()
	defer m.logosStorageClientMu.Unlock()
	m.logosStorageClient = client
}

func (m *ArchiveManagerLogosStorage) SetDownloadTimeout(timeout time.Duration) {
	m.downloadTimeout = timeout
}
