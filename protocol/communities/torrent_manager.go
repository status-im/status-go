package communities

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/signal"
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

	publisher Publisher
}

func NewTorrentManager(torrentConfig *params.TorrentConfig, logger, stdoutLogger *zap.Logger, persistence *Persistence, transport *transport.Transport, identity *ecdsa.PrivateKey, encryptor *encryption.Protocol, publisher Publisher) *TorrentManager {
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

		publisher: publisher,
	}
}

// LogStdout is copied directly from Manager, consider a refactor
func (m *TorrentManager) LogStdout(msg string, fields ...zap.Field) {
	m.stdoutLogger.Info(msg, fields...)
	m.logger.Debug(msg, fields...)
}

func (m *TorrentManager) SetOnline(online bool) {
	if online {
		if m.torrentConfig != nil && m.torrentConfig.Enabled && !m.TorrentClientStarted() {
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

	if m.TorrentClientStarted() {
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

func (m *TorrentManager) StopTorrentClient() []error {
	if m.TorrentClientStarted() {
		m.StopHistoryArchiveTasksIntervals()
		m.logger.Info("Stopping torrent client")
		errs := m.torrentClient.Close()
		if len(errs) > 0 {
			return errs
		}
		m.torrentClient = nil
	}
	return make([]error, 0)
}

func (m *TorrentManager) TorrentClientStarted() bool {
	return m.torrentClient != nil
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

func (m *TorrentManager) GetOldestWakuMessageTimestamp(topics []types.TopicType) (uint64, error) {
	return m.persistence.GetOldestWakuMessageTimestamp(topics)
}

func (m *TorrentManager) GetLastMessageArchiveEndDate(communityID types.HexBytes) (uint64, error) {
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

	lastArchiveEndDateTimestamp, err := m.GetLastMessageArchiveEndDate(communityID)
	if err != nil {
		m.LogStdout("failed to get last archive end date", zap.Error(err))
		return 0, err
	}

	if lastArchiveEndDateTimestamp == 0 {
		// If we don't have a tracked last message archive end date, it
		// means we haven't created an archive before, which means
		// the next thing to look at is the oldest waku message timestamp for
		// this community
		lastArchiveEndDateTimestamp, err = m.GetOldestWakuMessageTimestamp(topics)
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
	_, err := m.CreateHistoryArchiveTorrentFromDB(communityID, topics, startDate, endDate, partition, encrypt)
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

func (m *TorrentManager) StopHistoryArchiveTasksIntervals() {
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

func (m *TorrentManager) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.CreateHistoryArchiveTorrent(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *TorrentManager) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.CreateHistoryArchiveTorrent(communityID, make([]*types.Message, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *TorrentManager) CreateHistoryArchiveTorrent(communityID types.HexBytes, msgs []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

	loadFromDB := len(msgs) == 0

	from := startDate
	to := from.Add(partition)
	if to.After(endDate) {
		to = endDate
	}

	archiveDir := m.torrentConfig.DataDir + "/" + communityID.String()
	torrentDir := m.torrentConfig.TorrentDir
	indexPath := archiveDir + "/index"
	dataPath := archiveDir + "/data"

	wakuMessageArchiveIndexProto := &protobuf.WakuMessageArchiveIndex{}
	wakuMessageArchiveIndex := make(map[string]*protobuf.WakuMessageArchiveIndexMetadata)
	archiveIDs := make([]string, 0)

	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		err := os.MkdirAll(archiveDir, 0700)
		if err != nil {
			return archiveIDs, err
		}
	}
	if _, err := os.Stat(torrentDir); os.IsNotExist(err) {
		err := os.MkdirAll(torrentDir, 0700)
		if err != nil {
			return archiveIDs, err
		}
	}

	_, err := os.Stat(indexPath)
	if err == nil {
		wakuMessageArchiveIndexProto, err = m.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
		if err != nil {
			return archiveIDs, err
		}
	}

	var offset uint64 = 0

	for hash, metadata := range wakuMessageArchiveIndexProto.Archives {
		offset = offset + metadata.Size
		wakuMessageArchiveIndex[hash] = metadata
	}

	var encodedArchives []*EncodedArchiveData
	topicsAsByteArrays := topicsAsByteArrays(topics)

	m.publisher.publish(&Subscription{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
		CommunityID: communityID.String(),
	}})

	m.LogStdout("creating archives",
		zap.Any("startDate", startDate),
		zap.Any("endDate", endDate),
		zap.Duration("partition", partition),
	)
	for {
		if from.Equal(endDate) || from.After(endDate) {
			break
		}
		m.LogStdout("creating message archive",
			zap.Any("from", from),
			zap.Any("to", to),
		)

		var messages []types.Message
		if loadFromDB {
			messages, err = m.persistence.GetWakuMessagesByFilterTopic(topics, uint64(from.Unix()), uint64(to.Unix()))
			if err != nil {
				return archiveIDs, err
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
			m.LogStdout("no messages in this partition")
			from = to
			to = to.Add(partition)
			if to.After(endDate) {
				to = endDate
			}
			continue
		}

		m.LogStdout("creating archive with messages", zap.Int("messagesCount", len(messages)))

		// Not only do we partition messages, we also chunk them
		// roughly by size, such that each chunk will not exceed a given
		// size and archive data doesn't get too big
		messageChunks := make([][]types.Message, 0)
		currentChunkSize := 0
		currentChunk := make([]types.Message, 0)

		for _, msg := range messages {
			msgSize := len(msg.Payload) + len(msg.Sig)
			if msgSize > maxArchiveSizeInBytes {
				// we drop messages this big
				continue
			}

			if currentChunkSize+msgSize > maxArchiveSizeInBytes {
				messageChunks = append(messageChunks, currentChunk)
				currentChunk = make([]types.Message, 0)
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
				return archiveIDs, err
			}

			if encrypt {
				messageSpec, err := m.encryptor.BuildHashRatchetMessage(communityID, encodedArchive)
				if err != nil {
					return archiveIDs, err
				}

				encodedArchive, err = proto.Marshal(messageSpec.Message)
				if err != nil {
					return archiveIDs, err
				}
			}

			rawSize := len(encodedArchive)
			padding := 0
			size := 0

			if rawSize > pieceLength {
				size = rawSize + pieceLength - (rawSize % pieceLength)
				padding = size - rawSize
			} else {
				padding = pieceLength - rawSize
				size = rawSize + padding
			}

			wakuMessageArchiveIndexMetadata := &protobuf.WakuMessageArchiveIndexMetadata{
				Metadata: wakuMessageArchive.Metadata,
				Offset:   offset,
				Size:     uint64(size),
				Padding:  uint64(padding),
			}

			wakuMessageArchiveIndexMetadataBytes, err := proto.Marshal(wakuMessageArchiveIndexMetadata)
			if err != nil {
				return archiveIDs, err
			}

			archiveID := crypto.Keccak256Hash(wakuMessageArchiveIndexMetadataBytes).String()
			archiveIDs = append(archiveIDs, archiveID)
			wakuMessageArchiveIndex[archiveID] = wakuMessageArchiveIndexMetadata
			encodedArchives = append(encodedArchives, &EncodedArchiveData{bytes: encodedArchive, padding: padding})
			offset = offset + uint64(rawSize) + uint64(padding)
		}

		from = to
		to = to.Add(partition)
		if to.After(endDate) {
			to = endDate
		}
	}

	if len(encodedArchives) > 0 {

		dataBytes := make([]byte, 0)

		for _, encodedArchiveData := range encodedArchives {
			dataBytes = append(dataBytes, encodedArchiveData.bytes...)
			dataBytes = append(dataBytes, make([]byte, encodedArchiveData.padding)...)
		}

		wakuMessageArchiveIndexProto.Archives = wakuMessageArchiveIndex
		indexBytes, err := proto.Marshal(wakuMessageArchiveIndexProto)
		if err != nil {
			return archiveIDs, err
		}

		if encrypt {
			messageSpec, err := m.encryptor.BuildHashRatchetMessage(communityID, indexBytes)
			if err != nil {
				return archiveIDs, err
			}
			indexBytes, err = proto.Marshal(messageSpec.Message)
			if err != nil {
				return archiveIDs, err
			}
		}

		err = os.WriteFile(indexPath, indexBytes, 0644) // nolint: gosec
		if err != nil {
			return archiveIDs, err
		}

		file, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return archiveIDs, err
		}
		defer file.Close()

		_, err = file.Write(dataBytes)
		if err != nil {
			return archiveIDs, err
		}

		metaInfo := metainfo.MetaInfo{
			AnnounceList: defaultAnnounceList,
		}
		metaInfo.SetDefaults()
		metaInfo.CreatedBy = common.PubkeyToHex(&m.identity.PublicKey)

		info := metainfo.Info{
			PieceLength: int64(pieceLength),
		}

		err = info.BuildFromFilePath(archiveDir)
		if err != nil {
			return archiveIDs, err
		}

		metaInfo.InfoBytes, err = bencode.Marshal(info)
		if err != nil {
			return archiveIDs, err
		}

		metaInfoBytes, err := bencode.Marshal(metaInfo)
		if err != nil {
			return archiveIDs, err
		}

		err = os.WriteFile(m.torrentFile(communityID.String()), metaInfoBytes, 0644) // nolint: gosec
		if err != nil {
			return archiveIDs, err
		}

		m.LogStdout("torrent created", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

		m.publisher.publish(&Subscription{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.LogStdout("no archives created")
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
		return archiveIDs, err
	}

	if lastMessageArchiveEndDate > 0 {
		err = m.persistence.UpdateLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	} else {
		err = m.persistence.SaveLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	}
	if err != nil {
		return archiveIDs, err
	}
	return archiveIDs, nil
}

func (m *TorrentManager) SeedHistoryArchiveTorrent(communityID types.HexBytes) error {
	m.UnseedHistoryArchiveTorrent(communityID)

	id := communityID.String()
	torrentFile := m.torrentFile(id)

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

func (m *TorrentManager) DeleteHistoryArchiveDownloadTask(communityID string) {
	delete(m.historyArchiveDownloadTasks, communityID)
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

					index, err := m.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
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

func (m *TorrentManager) SaveMessageArchiveID(communityID types.HexBytes, hash string) error {
	return m.persistence.SaveMessageArchiveID(communityID, hash)
}

func (m *TorrentManager) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return m.persistence.GetMessageArchiveIDsToImport(communityID)
}
func (m *TorrentManager) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return m.persistence.SetMessageArchiveIDImported(communityID, hash, imported)
}

func (m *TorrentManager) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
	id := communityID.String()

	index, err := m.LoadHistoryArchiveIndexFromFile(m.identity, communityID)
	if err != nil {
		return nil, err
	}

	dataFile, err := os.Open(m.archiveDataFile(id))
	if err != nil {
		return nil, err
	}
	defer dataFile.Close()

	m.LogStdout("extracting messages from history archive",
		zap.String("communityID", communityID.String()),
		zap.String("archiveID", archiveID))
	metadata := index.Archives[archiveID]

	_, err = dataFile.Seek(int64(metadata.Offset), 0)
	if err != nil {
		m.LogStdout("failed to seek archive data file", zap.Error(err))
		return nil, err
	}

	data := make([]byte, metadata.Size-metadata.Padding)
	m.LogStdout("loading history archive data into memory", zap.Float64("data_size_MB", float64(metadata.Size-metadata.Padding)/1024.0/1024.0))
	_, err = dataFile.Read(data)
	if err != nil {
		m.LogStdout("failed failed to read archive data", zap.Error(err))
		return nil, err
	}

	archive := &protobuf.WakuMessageArchive{}

	err = proto.Unmarshal(data, archive)
	if err != nil {
		// The archive data might eb encrypted so we try to decrypt instead first
		var protocolMessage encryption.ProtocolMessage
		err := proto.Unmarshal(data, &protocolMessage)
		if err != nil {
			m.LogStdout("failed to unmarshal protocol message", zap.Error(err))
			return nil, err
		}

		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			m.logger.Debug("failed to decompress community pubkey", zap.Error(err))
			return nil, err
		}
		decryptedBytes, err := m.encryptor.HandleMessage(m.identity, pk, &protocolMessage, make([]byte, 0))
		if err != nil {
			m.LogStdout("failed to decrypt message archive", zap.Error(err))
			return nil, err
		}
		err = proto.Unmarshal(decryptedBytes.DecryptedMessage, archive)
		if err != nil {
			m.LogStdout("failed to unmarshal message archive", zap.Error(err))
			return nil, err
		}
	}
	return archive.Messages, nil
}

func (m *TorrentManager) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
	id := communityID.String()
	torrentFile := m.torrentFile(id)

	metaInfo, err := metainfo.LoadFromFile(torrentFile)
	if err != nil {
		return "", err
	}

	info, err := metaInfo.UnmarshalInfo()
	if err != nil {
		return "", err
	}

	return metaInfo.Magnet(nil, &info).String(), nil
}

func (m *TorrentManager) createWakuMessageArchive(from time.Time, to time.Time, messages []types.Message, topics [][]byte) *protobuf.WakuMessageArchive {
	var wakuMessages []*protobuf.WakuMessage

	for _, msg := range messages {
		topic := types.TopicTypeToByteArray(msg.Topic)
		wakuMessage := &protobuf.WakuMessage{
			Sig:          msg.Sig,
			Timestamp:    uint64(msg.Timestamp),
			Topic:        topic,
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

func (m *TorrentManager) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
	wakuMessageArchiveIndexProto := &protobuf.WakuMessageArchiveIndex{}

	indexPath := m.archiveIndexFile(communityID.String())
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(indexData, wakuMessageArchiveIndexProto)
	if err != nil {
		return nil, err
	}

	if len(wakuMessageArchiveIndexProto.Archives) == 0 && len(indexData) > 0 {
		// This means we're dealing with an encrypted index file, so we have to decrypt it first
		var protocolMessage encryption.ProtocolMessage
		err := proto.Unmarshal(indexData, &protocolMessage)
		if err != nil {
			return nil, err
		}
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			return nil, err
		}
		decryptedBytes, err := m.encryptor.HandleMessage(myKey, pk, &protocolMessage, make([]byte, 0))
		if err != nil {
			return nil, err
		}
		err = proto.Unmarshal(decryptedBytes.DecryptedMessage, wakuMessageArchiveIndexProto)
		if err != nil {
			return nil, err
		}
	}

	return wakuMessageArchiveIndexProto, nil
}

func (m *TorrentManager) TorrentFileExists(communityID string) bool {
	_, err := os.Stat(m.torrentFile(communityID))
	return err == nil
}

func (m *TorrentManager) torrentFile(communityID string) string {
	return path.Join(m.torrentConfig.TorrentDir, communityID+".torrent")
}

func (m *TorrentManager) archiveIndexFile(communityID string) string {
	return path.Join(m.torrentConfig.DataDir, communityID, "index")
}

func (m *TorrentManager) archiveDataFile(communityID string) string {
	return path.Join(m.torrentConfig.DataDir, communityID, "data")
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
