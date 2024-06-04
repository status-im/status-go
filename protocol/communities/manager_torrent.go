//go:build windows || linux || darwin
// +build windows linux darwin

package communities

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/signal"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"go.uber.org/zap"
)

type archiveMDSlice []*archiveMetadata

type archiveMetadata struct {
	hash string
	from uint64
}

func (md archiveMDSlice) Len() int {
	return len(md)
}

func (md archiveMDSlice) Swap(i, j int) {
	md[i], md[j] = md[j], md[i]
}

func (md archiveMDSlice) Less(i, j int) bool {
	return md[i].from > md[j].from
}

type EncodedArchiveData struct {
	padding int
	bytes   []byte
}

type HistoryArchiveDownloadTaskInfo struct {
	TotalDownloadedArchivesCount int
	TotalArchivesCount           int
	Cancelled                    bool
}

type TorrentContract interface {
	ArchiveContract

	LogStdout(string, ...zap.Field)
	SetOnline(bool)
	SetTorrentConfig(*params.TorrentConfig)
	StartTorrentClient() error
	Stop() error
	IsReady() bool
	GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error)
	GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error)
	GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error)
	CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error
	StartHistoryArchiveTasksInterval(community *Community, interval time.Duration)
	StopHistoryArchiveTasksInterval(communityID types.HexBytes)
	SeedHistoryArchiveTorrent(communityID types.HexBytes) error
	UnseedHistoryArchiveTorrent(communityID types.HexBytes)
	IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool
	GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask
	AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask)
	DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error)
	TorrentFileExists(communityID string) bool
}

type TorrentManager struct {
	torrentConfig                *params.TorrentConfig
	torrentClient                *torrent.Client
	torrentTasks                 map[string]metainfo.Hash
	historyArchiveDownloadTasks  map[string]*HistoryArchiveDownloadTask
	historyArchiveTasksWaitGroup sync.WaitGroup
	historyArchiveTasks          sync.Map // stores `chan struct{}`

	logger       *zap.Logger
	stdoutLogger *zap.Logger

	persistence *Persistence
	transport   *transport.Transport
	identity    *ecdsa.PrivateKey
	encryptor   *encryption.Protocol

	*ArchiveManager
	publisher Publisher
}

// NewTorrentManager this function is only built and called when the "windows || linux || darwin" build OS criteria are met
// In this case this version of NewTorrentManager will return the full Desktop TorrentManager ensuring that the
// build command will import and build the torrent deps for the Desktop OSes.
// NOTE: It is intentional that this file contains the identical function name as in "manager_torrent_mobile.go"
func NewTorrentManager(torrentConfig *params.TorrentConfig, logger *zap.Logger, persistence *Persistence, transport *transport.Transport, identity *ecdsa.PrivateKey, encryptor *encryption.Protocol, publisher Publisher) (TorrentContract, error) {
	stdoutLogger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to create archive logger %w", err)
	}

	return &TorrentManager{
		torrentConfig:               torrentConfig,
		torrentTasks:                make(map[string]metainfo.Hash),
		historyArchiveDownloadTasks: make(map[string]*HistoryArchiveDownloadTask),

		logger:       logger,
		stdoutLogger: stdoutLogger,

		persistence: persistence,
		transport:   transport,
		identity:    identity,
		encryptor:   encryptor,

		publisher:      publisher,
		ArchiveManager: NewArchiveManager(torrentConfig, logger, stdoutLogger, persistence, identity, encryptor, publisher),
	}, nil
}

// LogStdout appears to be some kind of debug tool specifically for torrent functionality
func (m *TorrentManager) LogStdout(msg string, fields ...zap.Field) {
	m.stdoutLogger.Info(msg, fields...)
	m.logger.Debug(msg, fields...)
}

func (m *TorrentManager) SetOnline(online bool) {
	if online {
		if m.torrentConfig != nil && m.torrentConfig.Enabled && !m.torrentClientStarted() {
			err := m.StartTorrentClient()
			if err != nil {
				m.LogStdout("couldn't start torrent client", zap.Error(err))
			}
		}
	}
}

func (m *TorrentManager) SetTorrentConfig(config *params.TorrentConfig) {
	m.torrentConfig = config
}

// getTCPandUDPport will return the same port number given if != 0,
// otherwise, it will attempt to find a free random tcp and udp port using
// the same number for both protocols
func (m *TorrentManager) getTCPandUDPport(portNumber int) (int, error) {
	if portNumber != 0 {
		return portNumber, nil
	}

	// Find free port
	for i := 0; i < 10; i++ {
		port := func() int {
			tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				m.logger.Warn("unable to resolve tcp addr: %v", zap.Error(err))
				return 0
			}

			tcpListener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				m.logger.Warn("unable to listen on addr", zap.Stringer("addr", tcpAddr), zap.Error(err))
				return 0
			}
			defer tcpListener.Close()

			port := tcpListener.Addr().(*net.TCPAddr).Port

			udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("localhost", fmt.Sprintf("%d", port)))
			if err != nil {
				m.logger.Warn("unable to resolve udp addr: %v", zap.Error(err))
				return 0
			}

			udpListener, err := net.ListenUDP("udp", udpAddr)
			if err != nil {
				m.logger.Warn("unable to listen on addr", zap.Stringer("addr", udpAddr), zap.Error(err))
				return 0
			}
			defer udpListener.Close()

			return port
		}()

		if port != 0 {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no free port found")
}

func (m *TorrentManager) StartTorrentClient() error {
	if m.torrentConfig == nil {
		return fmt.Errorf("can't start torrent client: missing torrentConfig")
	}

	if m.torrentClientStarted() {
		return nil
	}

	port, err := m.getTCPandUDPport(m.torrentConfig.Port)
	if err != nil {
		return err
	}

	config := torrent.NewDefaultClientConfig()
	config.SetListenAddr(":" + fmt.Sprint(port))
	config.Seed = true

	config.DataDir = m.torrentConfig.DataDir

	if _, err := os.Stat(m.torrentConfig.DataDir); os.IsNotExist(err) {
		err := os.MkdirAll(m.torrentConfig.DataDir, 0700)
		if err != nil {
			return err
		}
	}

	m.logger.Info("Starting torrent client", zap.Any("port", port))
	// Instantiating the client will make it bootstrap and listen eagerly,
	// so no go routine is needed here
	client, err := torrent.NewClient(config)
	if err != nil {
		return err
	}
	m.torrentClient = client
	return nil
}

func (m *TorrentManager) Stop() error {
	if m.torrentClientStarted() {
		m.stopHistoryArchiveTasksIntervals()
		m.logger.Info("Stopping torrent client")
		errs := m.torrentClient.Close()
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		m.torrentClient = nil
	}
	return nil
}

func (m *TorrentManager) torrentClientStarted() bool {
	return m.torrentClient != nil
}

func (m *TorrentManager) IsReady() bool {
	// Simply checking for `torrentConfig.Enabled` isn't enough
	// as there's a possibility that the torrent client couldn't
	// be instantiated (for example in case of port conflicts)
	return m.torrentConfig != nil &&
		m.torrentConfig.Enabled &&
		m.torrentClientStarted()
}

func (m *TorrentManager) GetCommunityChatsFilters(communityID types.HexBytes) ([]*transport.Filter, error) {
	chatIDs, err := m.persistence.GetCommunityChatIDs(communityID)
	if err != nil {
		return nil, err
	}

	filters := []*transport.Filter{}
	for _, cid := range chatIDs {
		filters = append(filters, m.transport.FilterByChatID(cid))
	}
	return filters, nil
}

func (m *TorrentManager) GetCommunityChatsTopics(communityID types.HexBytes) ([]types.TopicType, error) {
	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		return nil, err
	}

	topics := []types.TopicType{}
	for _, filter := range filters {
		topics = append(topics, filter.ContentTopic)
	}

	return topics, nil
}

func (m *TorrentManager) getOldestWakuMessageTimestamp(topics []types.TopicType) (uint64, error) {
	return m.persistence.GetOldestWakuMessageTimestamp(topics)
}

func (m *TorrentManager) getLastMessageArchiveEndDate(communityID types.HexBytes) (uint64, error) {
	return m.persistence.GetLastMessageArchiveEndDate(communityID)
}

func (m *TorrentManager) GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error) {
	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		m.LogStdout("failed to get community chats filters", zap.Error(err))
		return 0, err
	}

	if len(filters) == 0 {
		// If we don't have chat filters, we likely don't have any chats
		// associated to this community, which means there's nothing more
		// to do here
		return 0, nil
	}

	topics := []types.TopicType{}

	for _, filter := range filters {
		topics = append(topics, filter.ContentTopic)
	}

	lastArchiveEndDateTimestamp, err := m.getLastMessageArchiveEndDate(communityID)
	if err != nil {
		m.LogStdout("failed to get last archive end date", zap.Error(err))
		return 0, err
	}

	if lastArchiveEndDateTimestamp == 0 {
		// If we don't have a tracked last message archive end date, it
		// means we haven't created an archive before, which means
		// the next thing to look at is the oldest waku message timestamp for
		// this community
		lastArchiveEndDateTimestamp, err = m.getOldestWakuMessageTimestamp(topics)
		if err != nil {
			m.LogStdout("failed to get oldest waku message timestamp", zap.Error(err))
			return 0, err
		}
		if lastArchiveEndDateTimestamp == 0 {
			// This means there's no waku message stored for this community so far
			// (even after requesting possibly missed messages), so no messages exist yet that can be archived
			m.LogStdout("can't find valid `lastArchiveEndTimestamp`")
			return 0, nil
		}
	}

	return lastArchiveEndDateTimestamp, nil
}

func (m *TorrentManager) CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	m.UnseedHistoryArchiveTorrent(communityID)
	_, err := m.ArchiveManager.CreateHistoryArchiveTorrentFromDB(communityID, topics, startDate, endDate, partition, encrypt)
	if err != nil {
		return err
	}
	return m.SeedHistoryArchiveTorrent(communityID)
}

func (m *TorrentManager) StartHistoryArchiveTasksInterval(community *Community, interval time.Duration) {
	id := community.IDString()
	if _, exists := m.historyArchiveTasks.Load(id); exists {
		m.LogStdout("history archive tasks interval already in progress", zap.String("id", id))
		return
	}

	cancel := make(chan struct{})
	m.historyArchiveTasks.Store(id, cancel)
	m.historyArchiveTasksWaitGroup.Add(1)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	m.LogStdout("starting history archive tasks interval", zap.String("id", id))
	for {
		select {
		case <-ticker.C:
			m.LogStdout("starting archive task...", zap.String("id", id))
			lastArchiveEndDateTimestamp, err := m.GetHistoryArchivePartitionStartTimestamp(community.ID())
			if err != nil {
				m.LogStdout("failed to get last archive end date", zap.Error(err))
				continue
			}

			if lastArchiveEndDateTimestamp == 0 {
				// This means there are no waku messages for this community,
				// so nothing to do here
				m.LogStdout("couldn't determine archive start date - skipping")
				continue
			}

			topics, err := m.GetCommunityChatsTopics(community.ID())
			if err != nil {
				m.LogStdout("failed to get community chat topics ", zap.Error(err))
				continue
			}

			ts := time.Now().Unix()
			to := time.Unix(ts, 0)
			lastArchiveEndDate := time.Unix(int64(lastArchiveEndDateTimestamp), 0)

			err = m.CreateAndSeedHistoryArchive(community.ID(), topics, lastArchiveEndDate, to, interval, community.Encrypted())
			if err != nil {
				m.LogStdout("failed to create and seed history archive", zap.Error(err))
				continue
			}
		case <-cancel:
			m.UnseedHistoryArchiveTorrent(community.ID())
			m.historyArchiveTasks.Delete(id)
			m.historyArchiveTasksWaitGroup.Done()
			return
		}
	}
}

func (m *TorrentManager) stopHistoryArchiveTasksIntervals() {
	m.historyArchiveTasks.Range(func(_, task interface{}) bool {
		close(task.(chan struct{})) // Need to cast to the chan
		return true
	})
	// Stoping archive interval tasks is async, so we need
	// to wait for all of them to be closed before we shutdown
	// the torrent client
	m.historyArchiveTasksWaitGroup.Wait()
}

func (m *TorrentManager) StopHistoryArchiveTasksInterval(communityID types.HexBytes) {
	task, exists := m.historyArchiveTasks.Load(communityID.String())
	if exists {
		m.logger.Info("Stopping history archive tasks interval", zap.Any("id", communityID.String()))
		close(task.(chan struct{})) // Need to cast to the chan
	}
}

func (m *TorrentManager) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
	m.UnseedHistoryArchiveTorrent(communityID)

	id := communityID.String()
	torrentFile := torrentFile(m.torrentConfig.TorrentDir, id)

	metaInfo, err := metainfo.LoadFromFile(torrentFile)
	if err != nil {
		return err
	}

	info, err := metaInfo.UnmarshalInfo()
	if err != nil {
		return err
	}

	hash := metaInfo.HashInfoBytes()
	m.torrentTasks[id] = hash

	if err != nil {
		return err
	}

	torrent, err := m.torrentClient.AddTorrent(metaInfo)
	if err != nil {
		return err
	}

	torrent.DownloadAll()

	m.publisher.publish(&Subscription{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
		},
	})

	magnetLink := metaInfo.Magnet(nil, &info).String()

	m.LogStdout("seeding torrent", zap.String("id", id), zap.String("magnetLink", magnetLink))
	return nil
}

func (m *TorrentManager) UnseedHistoryArchiveTorrent(communityID types.HexBytes) {
	id := communityID.String()

	hash, exists := m.torrentTasks[id]

	if exists {
		torrent, ok := m.torrentClient.Torrent(hash)
		if ok {
			m.logger.Debug("Unseeding and dropping torrent for community: ", zap.Any("id", id))
			torrent.Drop()
			delete(m.torrentTasks, id)

			m.publisher.publish(&Subscription{
				HistoryArchivesUnseededSignal: &signal.HistoryArchivesUnseededSignal{
					CommunityID: id,
				},
			})
		}
	}
}

func (m *TorrentManager) IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool {
	id := communityID.String()
	hash := m.torrentTasks[id]
	torrent, ok := m.torrentClient.Torrent(hash)
	return ok && torrent.Seeding()
}

func (m *TorrentManager) GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask {
	return m.historyArchiveDownloadTasks[communityID]
}

func (m *TorrentManager) AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask) {
	m.historyArchiveDownloadTasks[communityID] = task
}

func (m *TorrentManager) DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {

	id := communityID.String()

	ml, err := metainfo.ParseMagnetUri(magnetlink)
	if err != nil {
		return nil, err
	}

	m.logger.Debug("adding torrent via magnetlink for community", zap.String("id", id), zap.String("magnetlink", magnetlink))
	torrent, err := m.torrentClient.AddMagnet(magnetlink)
	if err != nil {
		return nil, err
	}

	downloadTaskInfo := &HistoryArchiveDownloadTaskInfo{
		TotalDownloadedArchivesCount: 0,
		TotalArchivesCount:           0,
		Cancelled:                    false,
	}

	m.torrentTasks[id] = ml.InfoHash
	timeout := time.After(20 * time.Second)

	m.LogStdout("fetching torrent info", zap.String("magnetlink", magnetlink))
	select {
	case <-timeout:
		return nil, ErrTorrentTimedout
	case <-cancelTask:
		m.LogStdout("cancelled fetching torrent info")
		downloadTaskInfo.Cancelled = true
		return downloadTaskInfo, nil
	case <-torrent.GotInfo():

		files := torrent.Files()

		i, ok := findIndexFile(files)
		if !ok {
			// We're dealing with a malformed torrent, so don't do anything
			return nil, errors.New("malformed torrent data")
		}

		indexFile := files[i]
		indexFile.Download()

		m.LogStdout("downloading history archive index")
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-cancelTask:
				m.LogStdout("cancelled downloading archive index")
				downloadTaskInfo.Cancelled = true
				return downloadTaskInfo, nil
			case <-ticker.C:
				if indexFile.BytesCompleted() == indexFile.Length() {

					index, err := m.ArchiveManager.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
					if err != nil {
						return nil, err
					}

					existingArchiveIDs, err := m.persistence.GetDownloadedMessageArchiveIDs(communityID)
					if err != nil {
						return nil, err
					}

					if len(existingArchiveIDs) == len(index.Archives) {
						m.LogStdout("download cancelled, no new archives")
						return downloadTaskInfo, nil
					}

					downloadTaskInfo.TotalDownloadedArchivesCount = len(existingArchiveIDs)
					downloadTaskInfo.TotalArchivesCount = len(index.Archives)

					archiveHashes := make(archiveMDSlice, 0, downloadTaskInfo.TotalArchivesCount)

					for hash, metadata := range index.Archives {
						archiveHashes = append(archiveHashes, &archiveMetadata{hash: hash, from: metadata.Metadata.From})
					}

					sort.Sort(sort.Reverse(archiveHashes))

					m.publisher.publish(&Subscription{
						DownloadingHistoryArchivesStartedSignal: &signal.DownloadingHistoryArchivesStartedSignal{
							CommunityID: communityID.String(),
						},
					})

					for _, hd := range archiveHashes {

						hash := hd.hash
						hasArchive := false

						for _, existingHash := range existingArchiveIDs {
							if existingHash == hash {
								hasArchive = true
								break
							}
						}
						if hasArchive {
							continue
						}

						metadata := index.Archives[hash]
						startIndex := int(metadata.Offset) / pieceLength
						endIndex := startIndex + int(metadata.Size)/pieceLength

						downloadMsg := fmt.Sprintf("downloading data for message archive (%d/%d)", downloadTaskInfo.TotalDownloadedArchivesCount+1, downloadTaskInfo.TotalArchivesCount)
						m.LogStdout(downloadMsg, zap.String("hash", hash))
						m.LogStdout("pieces (start, end)", zap.Any("startIndex", startIndex), zap.Any("endIndex", endIndex-1))
						torrent.DownloadPieces(startIndex, endIndex)

						piecesCompleted := make(map[int]bool)
						for i = startIndex; i < endIndex; i++ {
							piecesCompleted[i] = false
						}

						psc := torrent.SubscribePieceStateChanges()
						downloadTicker := time.NewTicker(1 * time.Second)
						defer downloadTicker.Stop()

					downloadLoop:
						for {
							select {
							case <-downloadTicker.C:
								done := true
								for i = startIndex; i < endIndex; i++ {
									piecesCompleted[i] = torrent.PieceState(i).Complete
									if !piecesCompleted[i] {
										done = false
									}
								}
								if done {
									psc.Close()
									break downloadLoop
								}
							case <-cancelTask:
								m.LogStdout("downloading archive data interrupted")
								downloadTaskInfo.Cancelled = true
								return downloadTaskInfo, nil
							}
						}
						downloadTaskInfo.TotalDownloadedArchivesCount++
						err = m.persistence.SaveMessageArchiveID(communityID, hash)
						if err != nil {
							m.LogStdout("couldn't save message archive ID", zap.Error(err))
							continue
						}
						m.publisher.publish(&Subscription{
							HistoryArchiveDownloadedSignal: &signal.HistoryArchiveDownloadedSignal{
								CommunityID: communityID.String(),
								From:        int(metadata.Metadata.From),
								To:          int(metadata.Metadata.To),
							},
						})
					}
					m.publisher.publish(&Subscription{
						HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
							CommunityID: communityID.String(),
						},
					})
					m.LogStdout("finished downloading archives")
					return downloadTaskInfo, nil
				}
			}
		}
	}
}

func (m *TorrentManager) TorrentFileExists(communityID string) bool {
	_, err := os.Stat(torrentFile(m.torrentConfig.TorrentDir, communityID))
	return err == nil
}

func topicsAsByteArrays(topics []types.TopicType) [][]byte {
	var topicsAsByteArrays [][]byte
	for _, t := range topics {
		topic := types.TopicTypeToByteArray(t)
		topicsAsByteArrays = append(topicsAsByteArrays, topic)
	}
	return topicsAsByteArrays
}

func findIndexFile(files []*torrent.File) (index int, ok bool) {
	for i, f := range files {
		if f.DisplayPath() == "index" {
			return i, true
		}
	}
	return 0, false
}

func torrentFile(torrentDir, communityID string) string {
	return path.Join(torrentDir, communityID+".torrent")
}
