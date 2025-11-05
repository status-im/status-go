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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/messaging"
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/signal"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type ArchiveFileManager struct {
	torrentConfig *params.TorrentConfig
	codexConfig   *params.CodexConfig
	codexClient   CodexClientInterface
	logger        *zap.Logger
	persistence   *Persistence
	identity      *ecdsa.PrivateKey
	messaging     *messaging.API

	publisher Publisher
}

func NewArchiveFileManager(amc *ArchiveManagerConfig) *ArchiveFileManager {
	return &ArchiveFileManager{
		torrentConfig: amc.TorrentConfig,
		codexConfig:   amc.CodexConfig,
		logger:        amc.Logger,
		persistence:   amc.Persistence,
		identity:      amc.Identity,
		messaging:     amc.Messaging,
		publisher:     amc.Publisher,
	}
}

func (m *ArchiveFileManager) SetCodexClient(codexClient CodexClientInterface) {
	m.codexClient = codexClient
}

func (m *ArchiveFileManager) createHistoryArchiveTorrent(communityID types.HexBytes, msgs []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

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
			if msgSize > maxArchiveSizeInBytes {
				// we drop messages this big
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

		m.publisher.publish(&Subscription{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.logger.Debug("no archives created")
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

func (m *ArchiveFileManager) createHistoryArchiveCodex(communityID types.HexBytes, msgs []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

	loadFromDB := len(msgs) == 0

	from := startDate
	to := from.Add(partition)
	if to.After(endDate) {
		to = endDate
	}

	codexArchiveDir := m.codexHistoryArchiveDataDirPath(communityID)
	codexIndexPath := m.codexHistoryArchiveIndexFilePath(communityID)

	m.logger.Debug("codexArchiveDir", zap.String("codexArchiveDir", codexArchiveDir))

	codexWakuMessageArchiveIndexProto := &protobuf.CodexWakuMessageArchiveIndex{}
	codexWakuMessageArchiveIndex := make(map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata)
	codexArchiveIDs := make([]string, 0)

	if err := m.ensureCodexCommunityDir(communityID); err != nil {
		return codexArchiveIDs, err
	}

	_, err := os.Stat(codexIndexPath)
	if err == nil {
		codexWakuMessageArchiveIndexProto, err = m.CodexLoadHistoryArchiveIndexFromFile(m.identity, communityID)
		if err != nil {
			return codexArchiveIDs, err
		}
	}

	for hash, metadata := range codexWakuMessageArchiveIndexProto.Archives {
		codexWakuMessageArchiveIndex[hash] = metadata
	}

	topicsAsByteArrays := topicsAsByteArrays(topics)

	m.publisher.publish(&Subscription{CreatingHistoryArchivesSignal: &signal.CreatingHistoryArchivesSignal{
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
			m.logger.Debug("no messages in this partition")
			from = to
			to = to.Add(partition)
			if to.After(endDate) {
				to = endDate
			}
			continue
		}

		m.logger.Debug("creating Codex archive with messages", zap.Int("messagesCount", len(messages)))

		// Not only do we partition messages, we also chunk them
		// roughly by size, such that each chunk will not exceed a given
		// size and archive data doesn't get too big
		messageChunks := make([][]messagingtypes.ReceivedMessage, 0)
		currentChunkSize := 0
		currentChunk := make([]messagingtypes.ReceivedMessage, 0)

		for _, msg := range messages {
			msgSize := len(msg.Payload) + len(msg.Sig)
			if msgSize > maxArchiveSizeInBytes {
				// we drop messages this big
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
				m.logger.Error("failed to upload to codex", zap.Error(err))
				return codexArchiveIDs, err
			}

			m.logger.Debug("archive uploaded to codex", zap.String("cid", cid))

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
			m.logger.Error("failed to upload to codex", zap.Error(err))
			return codexArchiveIDs, err
		}

		err = m.writeCodexIndexToFile(communityID, codexIndexBytes)
		if err != nil {
			return codexArchiveIDs, err
		}

		err = m.writeCodexIndexCidToFile(communityID, cid)
		if err != nil {
			return codexArchiveIDs, err
		}

		m.logger.Debug("archives uploaded to Codex", zap.Any("from", startDate.Unix()), zap.Any("to", endDate.Unix()))

		m.publisher.publish(&Subscription{
			HistoryArchivesCreatedSignal: &signal.HistoryArchivesCreatedSignal{
				CommunityID: communityID.String(),
				From:        int(startDate.Unix()),
				To:          int(endDate.Unix()),
			},
		})
	} else {
		m.logger.Debug("no archives created")
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

	if lastMessageArchiveEndDate > 0 {
		err = m.persistence.UpdateLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	} else {
		err = m.persistence.SaveLastMessageArchiveEndDate(communityID, uint64(from.Unix()))
	}
	if err != nil {
		return codexArchiveIDs, err
	}
	return codexArchiveIDs, nil
}

func (m *ArchiveFileManager) archiveIndexFile(communityID string) string {
	return path.Join(m.torrentConfig.DataDir, communityID, "index")
}

func (m *ArchiveFileManager) ensureCodexCommunityDir(communityID types.HexBytes) error {
	if m.codexConfig == nil {
		return fmt.Errorf("codex config not initialized")
	}

	codexArchiveDir := m.codexHistoryArchiveDataDirPath(communityID)
	if err := os.MkdirAll(codexArchiveDir, 0700); err != nil {
		return fmt.Errorf("failed to create Codex archive directory %s: %w", codexArchiveDir, err)
	}
	return nil
}

func (m *ArchiveFileManager) codexHistoryArchiveDataDirPath(communityID types.HexBytes) string {
	return filepath.Join(m.codexConfig.HistoryArchiveDataDir, communityID.String())
}

func (m *ArchiveFileManager) codexHistoryArchiveIndexFilePath(communityID types.HexBytes) string {
	return filepath.Join(m.codexConfig.HistoryArchiveDataDir, communityID.String(), "index")
}

func (m *ArchiveFileManager) codexHistoryArchiveIndexCidFilePath(communityID types.HexBytes) string {
	return filepath.Join(m.codexConfig.HistoryArchiveDataDir, communityID.String(), "index-cid")
}

func (m *ArchiveFileManager) writeCodexIndexToFile(communityID types.HexBytes, bytes []byte) error {
	indexFilePath := m.codexHistoryArchiveIndexFilePath(communityID)
	return os.WriteFile(indexFilePath, bytes, 0644) // nolint: gosec
}

func (m *ArchiveFileManager) readCodexIndexFromFile(communityID types.HexBytes) ([]byte, error) {
	indexFilePath := m.codexHistoryArchiveIndexFilePath(communityID)
	return os.ReadFile(indexFilePath)
}

func (m *ArchiveFileManager) removeCodexIndexFile(communityID types.HexBytes) error {
	indexFilePath := m.codexHistoryArchiveIndexFilePath(communityID)
	err := os.Remove(indexFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *ArchiveFileManager) removeCodexIndexCidFile(communityID types.HexBytes) error {
	indexCidFilePath := m.codexHistoryArchiveIndexCidFilePath(communityID)
	err := os.Remove(indexCidFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *ArchiveFileManager) writeCodexIndexCidToFile(communityID types.HexBytes, cid string) error {
	cidFilePath := m.codexHistoryArchiveIndexCidFilePath(communityID)
	return os.WriteFile(cidFilePath, []byte(cid), 0644) // nolint: gosec
}

func (m *ArchiveFileManager) readCodexIndexCidFromFile(communityID types.HexBytes) ([]byte, error) {
	cidFilePath := m.codexHistoryArchiveIndexCidFilePath(communityID)
	return os.ReadFile(cidFilePath)
}

func (m *ArchiveFileManager) createWakuMessageArchive(from time.Time, to time.Time, messages []messagingtypes.ReceivedMessage, topics [][]byte) *protobuf.WakuMessageArchive {
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

func (m *ArchiveFileManager) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveTorrent(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveFileManager) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveTorrent(communityID, make([]*messagingtypes.ReceivedMessage, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveFileManager) CreateHistoryArchiveCodexFromMessages(communityID types.HexBytes, messages []*messagingtypes.ReceivedMessage, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveCodex(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveFileManager) CreateHistoryArchiveCodexFromDB(communityID types.HexBytes, topics []messagingtypes.ContentTopic, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveCodex(communityID, make([]*messagingtypes.ReceivedMessage, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveFileManager) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return m.persistence.GetMessageArchiveIDsToImport(communityID)
}

func (m *ArchiveFileManager) SaveMessageArchiveID(communityID types.HexBytes, hash string) error {
	return m.persistence.SaveMessageArchiveID(communityID, hash)
}

func (m *ArchiveFileManager) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return m.persistence.SetMessageArchiveIDImported(communityID, hash, imported)
}

func (m *ArchiveFileManager) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
	id := communityID.String()
	torrentFile := torrentFile(m.torrentConfig.TorrentDir, id)

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

func (m *ArchiveFileManager) GetHistoryArchiveIndexCid(communityID types.HexBytes) (string, error) {
	cidData, err := m.readCodexIndexCidFromFile(communityID)
	if err != nil {
		return "", err
	}

	return string(cidData), nil
}

func (m *ArchiveFileManager) archiveDataFile(communityID string) string {
	return path.Join(m.torrentConfig.DataDir, communityID, "data")
}

func (m *ArchiveFileManager) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
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

	m.logger.Debug("extracting messages from history archive",
		zap.String("communityID", communityID.String()),
		zap.String("archiveID", archiveID))
	metadata := index.Archives[archiveID]

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

func (m *ArchiveFileManager) ExtractMessagesFromCodexHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
	index, err := m.CodexLoadHistoryArchiveIndexFromFile(m.identity, communityID)
	if err != nil {
		return nil, err
	}

	metadata := index.Archives[archiveID]
	cid := metadata.Cid

	var buf bytes.Buffer
	err = m.codexClient.LocalDownload(cid, &buf)
	if err != nil {
		m.logger.Error("failed to download archive from codex", zap.Error(err))
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

func (m *ArchiveFileManager) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
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

func (m *ArchiveFileManager) CodexLoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.CodexWakuMessageArchiveIndex, error) {
	codexWakuMessageArchiveIndexProto := &protobuf.CodexWakuMessageArchiveIndex{}

	indexData, err := m.readCodexIndexFromFile(communityID)
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(indexData, codexWakuMessageArchiveIndexProto)
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
