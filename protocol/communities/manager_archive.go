//go:build !disable_torrent
// +build !disable_torrent

// Attribution to Pascal Precht, for further context please view the below issues
// - https://github.com/status-im/status-go/issues/2563
// - https://github.com/status-im/status-go/issues/2565
// - https://github.com/status-im/status-go/issues/2567
// - https://github.com/status-im/status-go/issues/2568

package communities

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"maps"
	"net"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/messaging"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"
	logosstorage "github.com/status-im/status-go/services/logos-storage"
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

type ArchiveManager struct {
	torrentConfig                *params.TorrentConfig
	torrentClient                *torrent.Client
	codexConfig                  *params.CodexConfig
	codexClient                  logosstorage.CodexClientInterface
	isCodexClientStarted         bool
	codexClientMu                sync.RWMutex
	downloadTimeout              time.Duration // timeout for archive downloads, defaults to 20s
	torrentTasks                 map[string]metainfo.Hash
	historyArchiveDownloadTasks  map[string]*HistoryArchiveDownloadTask
	historyArchiveTasksWaitGroup sync.WaitGroup
	historyArchiveTasks          sync.Map // stores `chan struct{}`

	logger      *zap.Logger
	persistence *Persistence
	messaging   *messaging.API
	identity    *ecdsa.PrivateKey

	*ArchiveFileManager
	publisher Publisher
}

// NewArchiveManager this function is only built and called when the "disable_torrent" build tag is not set
// In this case this version of NewArchiveManager will return the full Desktop ArchiveManager ensuring that the
// build command will import and build the torrent deps for the Desktop OSes.
// NOTE: It is intentional that this file contains the identical function name as in "manager_archive_nop.go"
func NewArchiveManager(amc *ArchiveManagerConfig) *ArchiveManager {
	return &ArchiveManager{
		torrentConfig:               amc.TorrentConfig,
		codexConfig:                 amc.CodexConfig,
		downloadTimeout:             20 * time.Second,
		torrentTasks:                make(map[string]metainfo.Hash),
		historyArchiveDownloadTasks: make(map[string]*HistoryArchiveDownloadTask),

		logger:      amc.Logger,
		persistence: amc.Persistence,
		messaging:   amc.Messaging,
		identity:    amc.Identity,

		publisher:          amc.Publisher,
		ArchiveFileManager: NewArchiveFileManager(amc),
	}
}

func (m *ArchiveManager) GetCodexClient() logosstorage.CodexClientInterface {
	m.codexClientMu.RLock()
	defer m.codexClientMu.RUnlock()
	return m.codexClient
}

func (m *ArchiveManager) SetOnline(online bool) {
	m.logger.Info("[CODEX][set_online] testing online status:", zap.Bool("online", online))
	if online {
		m.logger.Info("[CODEX][set_online] Online: checking if torrent/codex clients need to be started...")
		m.codexClientMu.RLock()
		codexStarted := m.isCodexClientStarted
		m.codexClientMu.RUnlock()

		m.logger.Info("[CODEX][set_online] Online. codexStarted:", zap.Bool("codexStarted", codexStarted))

		if m.torrentConfig != nil && m.torrentConfig.Enabled && !m.torrentClientStarted() {
			m.logger.Info("[CODEX][set_online] Starting torrent client...")
			err := m.StartTorrentClient()
			if err != nil {
				m.logger.Error("[CODEX][set_online] couldn't start torrent client", zap.Error(err))
			}
		}

		if m.codexConfig != nil && m.codexConfig.Enabled && !codexStarted {
			m.logger.Info("[CODEX][set_online] Starting codex client...")
			err := m.StartCodexClient()
			if err != nil {
				m.logger.Error("[CODEX][set_online] couldn't start codex client", zap.Error(err))
			}
		}
	}
}

func (m *ArchiveManager) SetTorrentConfig(config *params.TorrentConfig) {
	m.torrentConfig = config
	m.ArchiveFileManager.torrentConfig = config
}

func (m *ArchiveManager) SetCodexConfig(config *params.CodexConfig) {
	m.codexConfig = config
	m.ArchiveFileManager.codexConfig = config
}

// getTCPandUDPport will return the same port number given if != 0,
// otherwise, it will attempt to find a free random tcp and udp port using
// the same number for both protocols
func (m *ArchiveManager) getTCPandUDPport(portNumber int) (int, error) {
	if portNumber != 0 {
		return portNumber, nil
	}

	// Find free port
	for range 10 {
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

func (m *ArchiveManager) StartTorrentClient() error {
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

func (m *ArchiveManager) StartCodexClient() error {
	m.codexClientMu.Lock()
	defer m.codexClientMu.Unlock()

	if m.codexConfig == nil {
		return fmt.Errorf("can't start codex client: missing codexConfig")
	}

	if m.isCodexClientStarted {
		return nil
	}

	var err error
	cfgCopy := *m.codexConfig
	cfgCopy.CodexNodeConfig = m.codexConfig.CodexNodeConfig

	m.logger.Info("[CODEX][start_codex_client] Using the following CodexNodeConfig", zap.Any("config", cfgCopy.CodexNodeConfig))

	client, err := logosstorage.NewCodexClient(cfgCopy)
	if err != nil {
		return err
	}
	m.codexClient = client
	m.ArchiveFileManager.codexClient = client

	if err := m.codexClient.Start(); err != nil {
		m.isCodexClientStarted = false
		m.codexClient = nil
		m.ArchiveFileManager.codexClient = nil
		return err
	}

	m.isCodexClientStarted = true

	return nil
}

func (m *ArchiveManager) StopCodexClient() error {
	m.codexClientMu.Lock()
	defer m.codexClientMu.Unlock()

	errs := []error{}
	if m.isCodexClientStarted {
		m.logger.Info("[CODEX] Stopping codex client")

		e := m.codexClient.Stop()
		if e != nil {
			errs = append(errs, e)
		}

		e = m.codexClient.Destroy()
		if e != nil {
			errs = append(errs, e)
		}

		m.isCodexClientStarted = false
		m.codexClient = nil
		m.ArchiveFileManager.codexClient = nil
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (m *ArchiveManager) Stop() error {
	if m.torrentClientStarted() || m.isCodexClientStarted {
		m.stopHistoryArchiveTasksIntervals()
	}

	errs := []error{}
	if m.torrentClientStarted() {
		m.logger.Info("Stopping torrent client")
		errs = m.torrentClient.Close()
		m.torrentClient = nil
	}

	err := m.StopCodexClient()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (m *ArchiveManager) SetCodexClient(client logosstorage.CodexClientInterface) {
	m.codexClientMu.Lock()
	defer m.codexClientMu.Unlock()

	m.codexClient = client
	m.ArchiveFileManager.codexClient = client
	m.isCodexClientStarted = true
}

func (m *ArchiveManager) SetDownloadTimeout(timeout time.Duration) {
	m.downloadTimeout = timeout
}

func (m *ArchiveManager) torrentClientStarted() bool {
	return m.torrentClient != nil
}

func (m *ArchiveManager) IsTorrentReady() bool {
	m.codexClientMu.RLock()
	defer m.codexClientMu.RUnlock()

	// Simply checking for `torrentConfig.Enabled`
	// isn't enough as there's a possibility that the torrent client
	// couldn't be instantiated (for example in case of port conflicts)
	return m.torrentConfig != nil && m.torrentConfig.Enabled && m.torrentClientStarted()
}

func (m *ArchiveManager) IsCodexReady() bool {
	m.codexClientMu.RLock()
	defer m.codexClientMu.RUnlock()

	// Simply checking for `codexConfig.Enabled`
	// isn't enough as there's a possibility that the codex client
	// couldn't be instantiated (for example in case of port conflicts)
	return m.codexConfig != nil && m.codexConfig.Enabled && m.isCodexClientStarted
}

func (m *ArchiveManager) IsReady() bool {
	m.codexClientMu.RLock()
	defer m.codexClientMu.RUnlock()

	return m.IsTorrentReady() || m.IsCodexReady()
}

func (m *ArchiveManager) GetCommunityChatsFilters(communityID types.HexBytes) (messagingtypes.ChatFilters, error) {
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

func (m *ArchiveManager) GetCommunityChatsTopics(communityID types.HexBytes) ([]messagingtypes.ContentTopic, error) {
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

func (m *ArchiveManager) getOldestWakuMessageTimestamp(topics []messagingtypes.ContentTopic) (uint64, error) {
	return m.persistence.GetOldestWakuMessageTimestamp(topics)
}

func (m *ArchiveManager) getLastMessageArchiveEndDate(communityID types.HexBytes) (uint64, error) {
	return m.persistence.GetLastMessageArchiveEndDate(communityID)
}

func (m *ArchiveManager) GetHistoryArchivePartitionStartTimestamp(communityID types.HexBytes) (uint64, error) {
	community, err := m.persistence.GetByID(&m.identity.PublicKey, communityID)
	if err != nil {
		m.logger.Error("failed to load community", zap.Error(err))
		return 0, err
	}

	if community == nil {
		m.logger.Error("community not found for this id")
		return 0, err
	}

	filters, err := m.GetCommunityChatsFilters(communityID)
	if err != nil {
		m.logger.Error("failed to get community chats filters", zap.Error(err))
		return 0, err
	}

	filter := m.messaging.ChatFilterByChatID(community.UniversalChatID())
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

func (m *ArchiveManager) CreateAndSeedHistoryArchive(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {
	archiveTorrentCreatedSuccessfully := false
	archiveCodexCreatedSuccessfully := false
	distributionPreference, err := m.persistence.GetArchiveDistributionPreference()
	if err != nil {
		// fallback to codex
		m.logger.Debug("[CODEX][CreateAndSeedHistoryArchive] failed to get archive distribution preference - falling back to codex", zap.Error(err))
		distributionPreference = params.ArchiveDistributionMethodCodex
	}

	var errTorrent, errCodex error

	if distributionPreference == params.ArchiveDistributionMethodTorrent {
		archiveTorrentCreatedSuccessfully = true
		m.UnseedHistoryArchiveTorrent(communityID)
		_, errTorrent := m.ArchiveFileManager.CreateHistoryArchiveTorrentFromDB(communityID, topics, startDate, endDate, partition, encrypt)
		if errTorrent != nil {
			archiveTorrentCreatedSuccessfully = false
			m.logger.Error("failed to create history archive torrent", zap.Error(errTorrent))
		} else {
			errTorrent = m.SeedHistoryArchiveTorrent(communityID)
			if errTorrent != nil {
				archiveTorrentCreatedSuccessfully = false
				m.logger.Error("failed to seed history archive torrent", zap.Error(errTorrent))
			}
		}
	}

	if distributionPreference == params.ArchiveDistributionMethodCodex {
		lastIndexCid, err := m.persistence.GetLastSeenIndexCid(communityID)
		if err != nil {
			m.UnseedHistoryArchiveIndexCid(communityID, lastIndexCid)
		} else {
			if err != nil {
				m.logger.Debug("[CODEX][CreateAndSeedHistoryArchive] failed to get last seen index cid - proceeding without un-seeding", zap.Error(err))
			}
		}
		archiveCodexCreatedSuccessfully = true
		// codexArchiveIDs, errCodex := m.ArchiveFileManager.CreateHistoryArchiveCodexFromDB(communityID, topics, startDate, endDate, partition, encrypt)
		codexArchiveIDs, errCodex := m.CreateHistoryArchiveCodexFromDB(communityID, topics, startDate, endDate, partition, encrypt)
		if errCodex != nil {
			archiveCodexCreatedSuccessfully = false
			m.logger.Error("[CODEX][CreateAndSeedHistoryArchive] failed to create history archive codex", zap.Error(errCodex))
		} else {
			if len(codexArchiveIDs) == 0 {
				// no new codex archives were created - no need to distribute new index cid
				// but we need to (re)start seeding that we stopped above
				archiveCodexCreatedSuccessfully = false
				m.logger.Debug("[CODEX][CreateAndSeedHistoryArchive] no new codex archive ids were created - re-seeding existing index cid")
				if err = m.SeedHistoryArchiveIndexCid(communityID, lastIndexCid); err != nil {
					m.logger.Error("[CODEX][CreateAndSeedHistoryArchive] failed to seed existing history archive codex index cid", zap.Error(err))
				}
			}
			// else: we created new codex archives and they are already published to codex
			// in CreateHistoryArchiveCodexFromDB (thus they are seeded)
		}
	}
	if !archiveTorrentCreatedSuccessfully && !archiveCodexCreatedSuccessfully {
		return errors.Join(errTorrent, errCodex)
	}
	// one way of publishing index succeeded - we can publish the seeding signal
	m.publisher.publish(&Subscription{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
			MagnetLink:  archiveTorrentCreatedSuccessfully, // true if torrent created successfully
			IndexCid:    archiveCodexCreatedSuccessfully,   // true if codex created successfully
		},
	})

	return nil
}

func (m *ArchiveManager) StartHistoryArchiveTasksInterval(community *Community, interval time.Duration) {
	defer common.LogOnPanic()
	id := community.IDString()
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

			lastArchiveEndDateTimestamp, err := m.GetHistoryArchivePartitionStartTimestamp(community.ID())
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

			topics, err := m.GetCommunityChatsTopics(community.ID())
			if err != nil {
				m.logger.Error("failed to get community chat topics ", zap.Error(err))
				continue
			}
			filter := m.messaging.ChatFilterByChatID(community.UniversalChatID())
			if filter == nil {
				m.logger.Error("failed to get chat filter", zap.String("community's UniversalChatID", community.UniversalChatID()))
				continue
			}
			// adding the content-topic used for member updates.
			// since member updates would not be too frequent i.e only addition/deletion would add a new message,
			// this shouldn't cause too much increase in size of archive generated.
			topics = append(topics, filter.ContentTopic())

			ts := time.Now().Unix()
			to := time.Unix(ts, 0)
			lastArchiveEndDate := time.Unix(int64(lastArchiveEndDateTimestamp), 0)

			err = m.CreateAndSeedHistoryArchive(community.ID(), topics, lastArchiveEndDate, to, interval, community.Encrypted())
			if err != nil {
				m.logger.Error("failed to create and seed history archive", zap.Error(err))
				continue
			}
		case <-cancel:
			m.UnseedHistoryArchiveTorrent(community.ID())
			lastIndexCid, err := m.persistence.GetLastSeenIndexCid(community.ID())
			if err != nil {
				m.UnseedHistoryArchiveIndexCid(community.ID(), lastIndexCid)
			} else {
				if err != nil {
					m.logger.Debug("[CODEX][start_history_archive_tasks_interval] failed to get last seen index cid - proceeding without un-seeding", zap.Error(err))
				}
			}
			m.historyArchiveTasks.Delete(id)
			m.historyArchiveTasksWaitGroup.Done()
			return
		}
	}
}

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

func (m *ArchiveManager) StopHistoryArchiveTasksInterval(communityID types.HexBytes) {
	task, exists := m.historyArchiveTasks.Load(communityID.String())
	if exists {
		m.logger.Info("Stopping history archive tasks interval", zap.Any("id", communityID.String()))
		close(task.(chan struct{})) // Need to cast to the chan
	}
}

func (m *ArchiveManager) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
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

	torrent, err := m.torrentClient.AddTorrent(metaInfo)
	if err != nil {
		return err
	}

	torrent.DownloadAll()

	magnetLink := metaInfo.Magnet(nil, &info).String()

	m.logger.Debug("seeding torrent", zap.String("id", id), zap.String("magnetLink", magnetLink))
	return nil
}

func (m *ArchiveManager) UnseedHistoryArchiveTorrent(communityID types.HexBytes) {
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

func (m *ArchiveManager) SeedHistoryArchiveIndexCid(communityID types.HexBytes, indexCid string) error {
	if indexCid == "" {
		return nil
	}
	if !m.IsCodexReady() {
		return nil
	}
	// do not seed if already seeding
	if m.IsSeedingHistoryArchiveCodex(communityID, indexCid) {
		return nil
	}

	// for the purpose of seeding, we just need to make sure that the index cid
	// is fetched to the codex node - codex will seed it by advertising it on DHT
	_, err := m.codexClient.TriggerDownload(indexCid)
	if err != nil {
		return err
	}
	return nil
}

func (m *ArchiveManager) UnseedHistoryArchiveIndexCid(communityID types.HexBytes, indexCid string) {
	if indexCid == "" {
		return
	}
	if !m.IsCodexReady() {
		return
	}
	if !m.IsSeedingHistoryArchiveCodex(communityID, indexCid) {
		return
	}
	m.logger.Debug("[CODEX] Un-seeding index CID for community", zap.String("id", communityID.String()), zap.String("cid", indexCid))

	err := m.codexClient.RemoveCid(indexCid)
	if err != nil {
		m.logger.Error("[CODEX] failed to remove CID from Codex", zap.Error(err))
	}
}

func (m *ArchiveManager) IsSeedingHistoryArchiveTorrent(communityID types.HexBytes) bool {
	id := communityID.String()
	hash := m.torrentTasks[id]
	torrent, ok := m.torrentClient.Torrent(hash)
	return ok && torrent.Seeding()
}

func (m *ArchiveManager) IsSeedingHistoryArchiveCodex(communityID types.HexBytes, indexCid string) bool {
	if indexCid == "" {
		return false
	}
	if !m.IsCodexReady() {
		return false
	}
	hasCid, err := m.codexClient.HasCid(indexCid)
	if err != nil {
		m.logger.Debug("[CODEX] failed to verify Codex CID availability", zap.String("communityID", communityID.String()), zap.String("cid", indexCid), zap.Error(err))
		return false
	}
	return hasCid
}

func (m *ArchiveManager) GetHistoryArchiveDownloadTask(communityID string) *HistoryArchiveDownloadTask {
	return m.historyArchiveDownloadTasks[communityID]
}

func (m *ArchiveManager) AddHistoryArchiveDownloadTask(communityID string, task *HistoryArchiveDownloadTask) {
	m.historyArchiveDownloadTasks[communityID] = task
}

func (m *ArchiveManager) DownloadHistoryArchivesByMagnetlink(communityID types.HexBytes, magnetlink string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {

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

	m.logger.Debug("fetching torrent info", zap.String("magnetlink", magnetlink))
	select {
	case <-timeout:
		return nil, ErrTorrentTimedout
	case <-cancelTask:
		m.logger.Debug("cancelled fetching torrent info")
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

		m.logger.Debug("downloading history archive index")
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-cancelTask:
				m.logger.Debug("cancelled downloading archive index")
				downloadTaskInfo.Cancelled = true
				return downloadTaskInfo, nil
			case <-ticker.C:
				if indexFile.BytesCompleted() == indexFile.Length() {

					index, err := m.ArchiveFileManager.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
					if err != nil {
						return nil, err
					}

					existingArchiveIDs, err := m.persistence.GetDownloadedMessageArchiveIDs(communityID)
					if err != nil {
						return nil, err
					}

					if len(existingArchiveIDs) == len(index.Archives) {
						m.logger.Debug("download cancelled, no new archives")
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
						m.logger.Debug(downloadMsg, zap.String("hash", hash))
						m.logger.Debug("pieces (start, end)", zap.Any("startIndex", startIndex), zap.Any("endIndex", endIndex-1))
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
								m.logger.Debug("downloading archive data interrupted")
								downloadTaskInfo.Cancelled = true
								return downloadTaskInfo, nil
							}
						}
						downloadTaskInfo.TotalDownloadedArchivesCount++
						err = m.persistence.SaveMessageArchiveID(communityID, hash)
						if err != nil {
							m.logger.Error("couldn't save message archive ID", zap.Error(err))
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
							MagnetLink:  true,  // Downloaded via magnet link
							IndexCid:    false, // No Codex CID in magnet link downloads
						},
					})
					m.logger.Debug("finished downloading archives")
					return downloadTaskInfo, nil
				}
			}
		}
	}
}

func (m *ArchiveManager) PublishHistoryArchivesSeedingSignal(
	communityID types.HexBytes,
	magnetLink bool,
	indexCid bool,
) {
	m.publisher.publish(&Subscription{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
			MagnetLink:  magnetLink,
			IndexCid:    indexCid,
		},
	})
}

func (m *ArchiveManager) CreateHistoryArchiveCodexFromMessages(communityID types.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveCodex(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManager) CreateHistoryArchiveCodexFromDB(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveCodex(communityID, make([]*messagingtypes.ReceivedMessage, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManager) createHistoryArchiveCodex(communityID types.HexBytes, msgs []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

	loadFromDB := len(msgs) == 0

	from := startDate
	to := from.Add(partition)
	if to.After(endDate) {
		to = endDate
	}

	codexWakuMessageArchiveIndexProto := &protobuf.CodexWakuMessageArchiveIndex{}
	codexWakuMessageArchiveIndex := make(map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata)
	codexArchiveIDs := make([]string, 0)

	lastSeenIndexCid, err := m.persistence.GetLastSeenIndexCid(communityID)
	if err != nil {
		return codexArchiveIDs, err
	}

	if m.IsSeedingHistoryArchiveCodex(communityID, lastSeenIndexCid) {
		m.logger.Debug("[CODEX][createHistoryArchiveCodex] codex index file exists, loading from file")
		ctx, cancel := context.WithTimeout(context.Background(), m.downloadTimeout)
		defer cancel()
		codexWakuMessageArchiveIndexProto, err = m.CodexLoadHistoryArchiveIndex(ctx, m.identity, communityID, lastSeenIndexCid, true)
		if err != nil {
			return codexArchiveIDs, err
		}
	}

	maps.Copy(codexWakuMessageArchiveIndex, codexWakuMessageArchiveIndexProto.Archives)

	topicsAsByteArrays := topicsAsByteArrays(topics)

	m.publisher.publish(&Subscription{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
		CommunityID: communityID.String(),
	}})

	m.logger.Debug("[CODEX][createHistoryArchiveCodex] creating archives",
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
				return codexArchiveIDs, err
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
			m.logger.Debug("[CODEX] no messages in this partition")
			from = to
			to = to.Add(partition)
			if to.After(endDate) {
				to = endDate
			}
			continue
		}

		m.logger.Debug("[CODEX][createHistoryArchiveCodex] creating Codex archive with messages", zap.Int("messagesCount", len(messages)))

		// Not only do we partition messages, we also chunk them
		// roughly by size, such that each chunk will not exceed a given
		// size and archive data doesn't get too big
		messageChunks := make([][]messagingtypes.ReceivedMessage, 0)
		currentChunkSize := 0
		currentChunk := make([]messagingtypes.ReceivedMessage, 0)

		for _, msg := range messages {
			msgSize := len(msg.Payload) + len(msg.Sig)
			m.logger.Debug("[CODEX][createHistoryArchiveCodex] message size",
				zap.Int("messageSize", msgSize),
				zap.String("contentTopic", string(msg.Topic[:])),
				zap.ByteString("payload[0:31]", msg.Payload[:min(32, len(msg.Payload))]),
			)
			if msgSize > maxArchiveSizeInBytes {
				// we drop messages this big
				m.logger.Debug("[CODEX][createHistoryArchiveCodex] dropping message due to size", zap.Int("messageSize", msgSize))
				continue
			}

			if currentChunkSize+msgSize > maxArchiveSizeInBytes {
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
				return codexArchiveIDs, err
			}

			if encrypt {
				encodedArchive, err = m.messaging.BuildHashRatchetMessage(communityID, encodedArchive)
				if err != nil {
					return codexArchiveIDs, err
				}
			}

			// upload archive to codex and get CID back
			cid, err := m.codexClient.UploadArchive(encodedArchive)
			if err != nil {
				m.logger.Error("[CODEX] failed to upload to codex", zap.Error(err))
				return codexArchiveIDs, err
			}

			m.logger.Debug("[CODEX][createHistoryArchiveCodex] archive uploaded to codex", zap.String("cid", cid))

			codexWakuMessageArchiveIndexMetadata := &protobuf.CodexWakuMessageArchiveIndexMetadata{
				Metadata: wakuMessageArchive.Metadata,
				Cid:      cid,
			}

			codexWakuMessageArchiveIndexMetadataBytes, err := proto.Marshal(codexWakuMessageArchiveIndexMetadata)
			if err != nil {
				return codexArchiveIDs, err
			}

			codexArchiveID := crypto.Keccak256Hash(codexWakuMessageArchiveIndexMetadataBytes).String()
			codexArchiveIDs = append(codexArchiveIDs, codexArchiveID)
			codexWakuMessageArchiveIndex[codexArchiveID] = codexWakuMessageArchiveIndexMetadata
		}

		from = to
		to = to.Add(partition)
		if to.After(endDate) {
			to = endDate
		}
	}

	if len(codexArchiveIDs) > 0 {
		codexWakuMessageArchiveIndexProto.Archives = codexWakuMessageArchiveIndex
		codexIndexBytes, err := proto.Marshal(codexWakuMessageArchiveIndexProto)
		if err != nil {
			return codexArchiveIDs, err
		}

		if encrypt {
			codexIndexBytes, err = m.messaging.BuildHashRatchetMessage(communityID, codexIndexBytes)
			if err != nil {
				return codexArchiveIDs, err
			}
		}

		// upload index file to codex
		cid, err := m.codexClient.UploadArchive(codexIndexBytes)
		if err != nil {
			m.logger.Error("[CODEX][createHistoryArchiveCodex] failed to upload to codex", zap.Error(err))
			return codexArchiveIDs, err
		}

		m.logger.Debug("[CODEX][createHistoryArchiveCodex] index uploaded to codex", zap.String("cid", cid))
		m.logger.Debug("[CODEX][createHistoryArchiveCodex] archives uploaded to Codex", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

		m.logger.Debug("[CODEX][create_history_archive_codex] updating last seen index cid", zap.String("cid", cid))
		err = m.persistence.UpdateLastSeenIndexCid(communityID, cid)
		if err != nil {
			return codexArchiveIDs, err
		}

		m.publisher.publish(&Subscription{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.logger.Debug("[CODEX][createHistoryArchiveCodex] no archives created")
		m.publisher.publish(&Subscription{
			NoHistoryArchivesCreatedSignal: &signal.NoHistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	}

	lastMessageArchiveEndDate, err := m.persistence.GetLastMessageArchiveEndDate(communityID)
	if err != nil {
		return codexArchiveIDs, err
	}

	m.logger.Debug("[CODEX][create_history_archive_codex] updating lastMessageArchiveEndDate", zap.Uint64("lastMessageArchiveEndDate", lastMessageArchiveEndDate))
	err = m.persistence.UpdateLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	if err != nil {
		return codexArchiveIDs, err
	}
	return codexArchiveIDs, nil
}

func (m *ArchiveManager) ExtractMessagesFromCodexHistoryArchive(communityID types.HexBytes, archiveID string, codexIndex *protobuf.CodexWakuMessageArchiveIndex) ([]*protobuf.WakuMessage, error) {
	metadata, ok := codexIndex.Archives[archiveID]
	if !ok || metadata == nil {
		return nil, fmt.Errorf("archive %s missing from codex index", archiveID)
	}
	cid := metadata.Cid

	var buf bytes.Buffer
	err := m.codexClient.LocalDownload(cid, &buf)
	if err != nil {
		m.logger.Error("[CODEX] failed to download archive from codex", zap.Error(err))
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

func (m *ArchiveManager) CodexLoadHistoryArchiveIndex(ctx context.Context, myKey *ecdsa.PrivateKey, communityID types.HexBytes, indexCid string, isLocal bool) (*protobuf.CodexWakuMessageArchiveIndex, error) {
	codexWakuMessageArchiveIndexProto := &protobuf.CodexWakuMessageArchiveIndex{}

	indexDownloader := logosstorage.NewCodexIndexDownloader(m.codexClient, m.logger)

	var indexBuf bytes.Buffer
	if isLocal {
		if err := indexDownloader.DownloadIndexFileFromLocalNode(ctx, indexCid, &indexBuf); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, ErrIndexCidTimedout
			}
			return nil, err
		}
	} else {
		if err := indexDownloader.DownloadIndexFileFromNetwork(ctx, indexCid, &indexBuf); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, ErrIndexCidTimedout
			}
			return nil, err
		}
	}
	indexData := indexBuf.Bytes()

	err := proto.Unmarshal(indexData, codexWakuMessageArchiveIndexProto)
	if err != nil {
		return nil, err
	}

	if len(codexWakuMessageArchiveIndexProto.Archives) == 0 && len(indexData) > 0 {
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

		err = proto.Unmarshal(decryptedData, codexWakuMessageArchiveIndexProto)
		if err != nil {
			return nil, err
		}
	}

	return codexWakuMessageArchiveIndexProto, nil
}

func (m *ArchiveManager) DownloadHistoryArchivesByIndexCid(communityID types.HexBytes, indexCid string, cancelTask chan struct{}) (*HistoryArchiveDownloadTaskInfo, error) {

	id := communityID.String()

	downloadTaskInfo := &HistoryArchiveDownloadTaskInfo{
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
			m.logger.Debug("[CODEX] cancelling downloading index from Codex")
			cancel()
		case <-done:
		}
	}()

	index, err := m.CodexLoadHistoryArchiveIndex(indexCtx,
		m.identity, communityID, indexCid, false)
	close(done)
	if err != nil {
		// check if error is due to timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrIndexCidTimedout
		}
		// check if error is due to cancellation
		if errors.Is(err, context.Canceled) {
			m.logger.Debug("[CODEX] cancelled downloading index from Codex")
			downloadTaskInfo.Cancelled = true
			return downloadTaskInfo, nil
		}
		return nil, err
	}

	// Publish index download completed signal
	m.publisher.publish(&Subscription{
		IndexDownloadCompletedSignal: &signal.IndexDownloadCompletedSignal{
			CommunityID: communityID.String(),
			IndexCid:    indexCid,
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
		m.logger.Debug("[CODEX] aborting download, no new archives")
		return downloadTaskInfo, nil
	}

	// Create separate cancel channel for the archive
	// downloader to avoid channel competition
	archiveDownloaderCancel := make(chan struct{})

	// Create the archive downloader using the protobuf index directly
	archiveDownloader := logosstorage.NewCodexArchiveDownloader(
		m.codexClient, index, id, existingArchiveIDs,
		archiveDownloaderCancel, m.logger)

	// Set up callback for when individual archives are downloaded
	archiveDownloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		err = m.persistence.SaveMessageArchiveID(communityID, hash)
		if err != nil {
			m.logger.Error("[CODEX] couldn't save message archive ID", zap.Error(err))
		}
		m.publisher.publish(&Subscription{
			HistoryArchiveDownloadedSignal: &signal.HistoryArchiveDownloadedSignal{
				CommunityID: communityID.String(),
				From:        int(from),
				To:          int(to),
			},
		})

		m.logger.Debug("[CODEX] archive downloaded successfully",
			zap.String("hash", hash),
			zap.Uint64("from", from),
			zap.Uint64("to", to))
	})

	m.logger.Debug("[CODEX] starting downloading individual archives from Codex")

	archiveDownloader.StartDownload()

	m.publisher.publish(&Subscription{
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
			return nil, ErrIndexCidTimedout
		case <-cancelTask:
			m.logger.Debug("[CODEX] cancelled downloading individual archives")
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

				m.logger.Info("[CODEX] downloading archives from Codex completed",
					zap.Int("totalArchives", downloadTaskInfo.TotalArchivesCount),
					zap.Int("downloadedArchives", downloadTaskInfo.TotalDownloadedArchivesCount))

				return downloadTaskInfo, nil
			} else {
				// Update progress
				downloadTaskInfo.TotalDownloadedArchivesCount =
					archiveDownloader.GetTotalDownloadedArchivesCount()
				m.logger.Debug("[CODEX] downloading archives in progress",
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

func (m *ArchiveManager) TorrentFileExists(communityID string) bool {
	_, err := os.Stat(torrentFile(m.torrentConfig.TorrentDir, communityID))
	return err == nil
}

func topicsAsByteArrays(topics []messagingtypes.ContentTopic) [][]byte {
	var topicsAsByteArrays [][]byte
	for _, t := range topics {
		topicsAsByteArrays = append(topicsAsByteArrays, t.Bytes())
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

func (m *ArchiveManager) GetDownloadedMessageArchiveIDs(communityID types.HexBytes) ([]string, error) {
	return m.persistence.GetDownloadedMessageArchiveIDs(communityID)
}
