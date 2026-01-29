//go:build !disable_history_archives
// +build !disable_history_archives

package torrent

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"

	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/signal"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	archivecommons "github.com/status-im/status-go/protocol/communities/archive/commons"
	archiveconsts "github.com/status-im/status-go/protocol/communities/archive/consts"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	archiveutils "github.com/status-im/status-go/protocol/communities/archive/utils"
	"github.com/status-im/status-go/protocol/protobuf"
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

type ArchiveManagerTorrent struct {
	torrentConfig *params.TorrentConfig
	torrentClient *torrent.Client
	torrentTasks  map[string]metainfo.Hash

	logger      *zap.Logger
	persistence archivetypes.PersistenceProvider
	messaging   *messaging.API
	identity    *ecdsa.PrivateKey

	publisher archivetypes.HistoryArchivePublisher
}

var defaultAnnounceList = [][]string{
	{"udp://tracker.opentrackr.org:1337/announce"},
	{"udp://tracker.openbittorrent.com:6969/announce"},
}

var pieceLength = 100 * 1024

func NewArchiveManagerTorrent(
	torrentConfig *params.TorrentConfig,
	logger *zap.Logger,
	persistence archivetypes.PersistenceProvider,
	messaging *messaging.API,
	identity *ecdsa.PrivateKey,
	publisher archivetypes.HistoryArchivePublisher,
) *ArchiveManagerTorrent {
	return &ArchiveManagerTorrent{
		torrentConfig: torrentConfig,
		torrentTasks:  make(map[string]metainfo.Hash),

		logger:      logger,
		persistence: persistence,
		messaging:   messaging,
		identity:    identity,

		publisher: publisher,
	}
}

// ArchiveServiceBackend interface implementation

func (m *ArchiveManagerTorrent) SetOnline(online bool) {
	if online {
		err := m.Start()
		if err != nil {
			m.logger.Error("couldn't start torrent client", zap.Error(err))
		}
	}
}

func (m *ArchiveManagerTorrent) Start() error {
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

func (m *ArchiveManagerTorrent) Stop() error {
	if m.torrentClientStarted() {
		m.logger.Info("Stopping torrent client")
		errs := m.torrentClient.Close()
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		m.torrentClient = nil
	}
	return nil
}

func (m *ArchiveManagerTorrent) IsStarted() bool {
	return m.torrentClient != nil
}

func (m *ArchiveManagerTorrent) IsReady() bool {
	// Check if the torrent client is actually started
	// (it might not be in case of port conflicts, etc.)
	return m.torrentClientStarted()
}

func (m *ArchiveManagerTorrent) SeedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) error {
	// NOTE: archiveLink is not currently used. We the underlying torrent client
	// to make sure we are seeding.
	m.UnseedHistoryArchive(communityID, archiveLink)

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

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
		HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
			CommunityID: communityID.String(),
		},
	})

	magnetLink := metaInfo.Magnet(nil, &info).String()

	m.logger.Debug("seeding torrent", zap.String("id", id), zap.String("magnetLink", magnetLink))
	return nil
}

func (m *ArchiveManagerTorrent) UnseedHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) {
	// NOTE: archiveLink is not currently used. We simply use torrentClient to drop
	// the torrent corresponding to the communityID.
	id := communityID.String()

	hash, exists := m.torrentTasks[id]

	if exists {
		torrent, ok := m.torrentClient.Torrent(hash)
		if ok {
			m.logger.Debug("Unseeding and dropping torrent for community: ", zap.Any("id", id))
			torrent.Drop()
			delete(m.torrentTasks, id)

			m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
				HistoryArchivesUnseededSignal: &signal.HistoryArchivesUnseededSignal{
					CommunityID: id,
				},
			})
		}
	}
}

func (m *ArchiveManagerTorrent) IsSeedingHistoryArchive(communityID cryptotypes.HexBytes, archiveLink string) bool {
	// NOTE: archiveLink is not currently used. We simply use torrentClient to get
	// the torrent corresponding to the communityID and check if it's seeding.
	id := communityID.String()
	hash := m.torrentTasks[id]
	torrent, ok := m.torrentClient.Torrent(hash)
	return ok && torrent.Seeding()
}

func (m *ArchiveManagerTorrent) DownloadHistoryArchives(communityID cryptotypes.HexBytes, archiveLink string, cancelTask chan struct{}) (*archivetypes.HistoryArchiveDownloadTaskInfo, error) {
	id := communityID.String()

	ml, err := metainfo.ParseMagnetUri(archiveLink)
	if err != nil {
		return nil, err
	}

	m.logger.Debug("adding torrent via magnetlink for community", zap.String("id", id), zap.String("magnetlink", archiveLink))
	torrent, err := m.torrentClient.AddMagnet(archiveLink)
	if err != nil {
		return nil, err
	}

	downloadTaskInfo := &archivetypes.HistoryArchiveDownloadTaskInfo{
		TotalDownloadedArchivesCount: 0,
		TotalArchivesCount:           0,
		Cancelled:                    false,
	}

	m.torrentTasks[id] = ml.InfoHash
	timeout := time.After(20 * time.Second)

	m.logger.Debug("fetching torrent info", zap.String("magnetlink", archiveLink))
	select {
	case <-timeout:
		return nil, archivecommons.ErrArchiveTimedout
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

					index, err := m.loadHistoryArchiveIndexFromFile(m.identity, communityID)
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

					m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
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
						m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
							HistoryArchiveDownloadedSignal: &signal.HistoryArchiveDownloadedSignal{
								CommunityID: communityID.String(),
								From:        int(metadata.Metadata.From),
								To:          int(metadata.Metadata.To),
							},
						})
					}
					m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
						HistoryArchivesSeedingSignal: &signal.HistoryArchivesSeedingSignal{
							CommunityID: communityID.String(),
						},
					})
					m.logger.Debug("finished downloading archives")
					return downloadTaskInfo, nil
				}
			}
		}
	}
}

func (m *ArchiveManagerTorrent) CreateHistoryArchiveFromMessages(communityID cryptotypes.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveTorrent(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManagerTorrent) CreateHistoryArchiveFromDB(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveTorrent(communityID, make([]*messagingtypes.ReceivedMessage, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManagerTorrent) CreateAndSeedHistoryArchive(communityID cryptotypes.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) error {

	archiveCreatedSuccessfully := true
	m.UnseedHistoryArchive(communityID, "")
	_, err := m.CreateHistoryArchiveFromDB(communityID, topics, startDate, endDate, partition, encrypt)
	if err != nil {
		archiveCreatedSuccessfully = false
		m.logger.Error("failed to create history archive torrent", zap.Error(err))
	} else {
		err = m.SeedHistoryArchive(communityID, "")
		if err != nil {
			archiveCreatedSuccessfully = false
			m.logger.Error("failed to seed history archive torrent", zap.Error(err))
		}
	}

	if !archiveCreatedSuccessfully {
		return err
	}

	return nil
}

func (m *ArchiveManagerTorrent) LoadArchiveMessages(ctx context.Context, communityID cryptotypes.HexBytes, archiveLink string, downloadedArchiveID string) ([]*protobuf.WakuMessage, error) {
	return m.extractMessagesFromHistoryArchive(communityID, downloadedArchiveID)
}

// func (m *ArchiveManagerTorrent) GetHistoryArchiveLink(communityID cryptotypes.HexBytes) (string, error) {
// 	id := communityID.String()
// 	torrentFile := torrentFile(m.torrentConfig.TorrentDir, id)

// 	metaInfo, err := metainfo.LoadFromFile(torrentFile)
// 	if err != nil {
// 		return "", err
// 	}

// 	info, err := metaInfo.UnmarshalInfo()
// 	if err != nil {
// 		return "", err
// 	}

// 	return metaInfo.Magnet(nil, &info).String(), nil
// }

// private methods
func (m *ArchiveManagerTorrent) archiveDataFile(communityID string) string {
	return filepath.Join(m.torrentConfig.DataDir, communityID, "data")
}

func (m *ArchiveManagerTorrent) extractMessagesFromHistoryArchive(communityID cryptotypes.HexBytes, downloadedArchiveID string) ([]*protobuf.WakuMessage, error) {
	id := communityID.String()

	index, err := m.loadHistoryArchiveIndexFromFile(m.identity, communityID)
	if err != nil {
		return nil, err
	}

	dataFile, err := os.Open(m.archiveDataFile(id))
	if err != nil {
		return nil, err
	}
	defer dataFile.Close()

	m.logger.Debug("extracting messages from history archive",
		zap.String("communityID", communityID.String()),
		zap.String("downloadedArchiveID", downloadedArchiveID))
	metadata := index.Archives[downloadedArchiveID]

	_, err = dataFile.Seek(int64(metadata.Offset), 0)
	if err != nil {
		m.logger.Error("failed to seek archive data file", zap.Error(err))
		return nil, err
	}

	data := make([]byte, metadata.Size-metadata.Padding)
	m.logger.Debug("loading history archive data into memory", zap.Float64("data_size_MB", float64(metadata.Size-metadata.Padding)/1024.0/1024.0))
	_, err = dataFile.Read(data)
	if err != nil {
		m.logger.Error("failed failed to read archive data", zap.Error(err))
		return nil, err
	}

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

func (m *ArchiveManagerTorrent) torrentClientStarted() bool {
	return m.torrentClient != nil
}

// getTCPandUDPport will return the same port number given if != 0,
// otherwise, it will attempt to find a free random tcp and udp port using
// the same number for both protocols
func (m *ArchiveManagerTorrent) getTCPandUDPport(portNumber int) (int, error) {
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

func (m *ArchiveManagerTorrent) loadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID cryptotypes.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
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
		pk, err := crypto.DecompressPubkey(communityID)
		if err != nil {
			return nil, err
		}

		decryptedData, err := m.messaging.DecryptMessage(myKey, pk, indexData)
		if err != nil {
			m.logger.Error("failed to decrypt message archive", zap.Error(err))
			return nil, err
		}

		err = proto.Unmarshal(decryptedData, wakuMessageArchiveIndexProto)
		if err != nil {
			return nil, err
		}
	}

	return wakuMessageArchiveIndexProto, nil
}

func (m *ArchiveManagerTorrent) archiveIndexFile(communityID string) string {
	return filepath.Join(m.torrentConfig.DataDir, communityID, "index")
}

func (m *ArchiveManagerTorrent) createHistoryArchiveTorrent(communityID cryptotypes.HexBytes, msgs []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

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
		wakuMessageArchiveIndexProto, err = m.loadHistoryArchiveIndexFromFile(m.identity, communityID)
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
	topicsAsByteArrays := archiveutils.TopicsAsByteArrays(topics)

	m.publisher.Publish(&archivetypes.HistoryArchiveSignals{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
		CommunityID: communityID.String(),
	}})

	m.logger.Debug("creating archives",
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
			m.logger.Debug("no messages in this partition")
			from = to
			to = to.Add(partition)
			if to.After(endDate) {
				to = endDate
			}
			continue
		}

		m.logger.Debug("creating archive with messages", zap.Int("messagesCount", len(messages)))

		// Not only do we partition messages, we also chunk them
		// roughly by size, such that each chunk will not exceed a given
		// size and archive data doesn't get too big
		messageChunks := make([][]messagingtypes.ReceivedMessage, 0)
		currentChunkSize := 0
		currentChunk := make([]messagingtypes.ReceivedMessage, 0)

		for _, msg := range messages {
			msgSize := len(msg.Payload) + len(msg.Sig)
			if msgSize > archiveconsts.MaxArchiveSizeInBytes {
				// we drop messages this big
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
				return archiveIDs, err
			}

			if encrypt {
				encodedArchive, err = m.messaging.BuildHashRatchetMessage(communityID, encodedArchive)
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
			indexBytes, err = m.messaging.BuildHashRatchetMessage(communityID, indexBytes)
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
		metaInfo.CreatedBy = crypto.PubkeyToHex(&m.identity.PublicKey)

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

		err = os.WriteFile(torrentFile(m.torrentConfig.TorrentDir, communityID.String()), metaInfoBytes, 0644) // nolint: gosec
		if err != nil {
			return archiveIDs, err
		}

		m.logger.Debug("torrent created", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

		m.publisher.Publish(&archivetypes.HistoryArchiveSignals{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.logger.Debug("no archives created")
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

func (m *ArchiveManagerTorrent) createWakuMessageArchive(from time.Time, to time.Time, messages []messagingtypes.ReceivedMessage, topics [][]byte) *protobuf.WakuMessageArchive {
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

// utility functions
func torrentFile(torrentDir, communityID string) string {
	return filepath.Join(torrentDir, communityID+".torrent")
}

func findIndexFile(files []*torrent.File) (index int, ok bool) {
	for i, f := range files {
		if f.DisplayPath() == "index" {
			return i, true
		}
	}
	return 0, false
}

// Special functions
// These functions are not part of the ArchiveServiceBackend interface.
// Some legacy tests are accessing implementation details and for this reason
// we need to expose these special accessors.
func (m *ArchiveManagerTorrent) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID cryptotypes.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
	return m.loadHistoryArchiveIndexFromFile(myKey, communityID)
}

func (m *ArchiveManagerTorrent) GetTorrentFilePath(communityID string) string {
	return torrentFile(m.torrentConfig.TorrentDir, communityID)
}

func (m *ArchiveManagerTorrent) GetArchiveDataFilePath(communityID string) string {
	return m.archiveDataFile(communityID)
}

func (m *ArchiveManagerTorrent) GetArchiveIndexFilePath(communityID string) string {
	return m.archiveIndexFile(communityID)
}

func (m *ArchiveManagerTorrent) GetTorrentConfig() *params.TorrentConfig {
	return m.torrentConfig
}

func (m *ArchiveManagerTorrent) GetTorrentTasksCount() int {
	return len(m.torrentTasks)
}

func (m *ArchiveManagerTorrent) GetMetaInfoHashForCommunity(communityID string) metainfo.Hash {
	return m.torrentTasks[communityID]
}

func (m *ArchiveManagerTorrent) GetTorrentForCommunity(communityID string) (*torrent.Torrent, bool) {
	hash, exists := m.torrentTasks[communityID]
	if !exists {
		return nil, false
	}
	torrent, ok := m.torrentClient.Torrent(hash)
	return torrent, ok
}
