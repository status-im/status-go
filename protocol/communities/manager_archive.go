package communities

import (
	"crypto/ecdsa"
	"go.uber.org/zap"
	"os"
	"path"
	"time"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/signal"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/golang/protobuf/proto"
)

type ArchiveManager struct {
	torrentConfig *params.TorrentConfig

	logger       *zap.Logger
	stdoutLogger *zap.Logger

	persistence *Persistence
	identity    *ecdsa.PrivateKey
	encryptor   *encryption.Protocol

	publisher Publisher
}

func NewArchiveManager(torrentConfig *params.TorrentConfig, logger, stdoutLogger *zap.Logger, persistence *Persistence, identity *ecdsa.PrivateKey, encryptor *encryption.Protocol, publisher Publisher) *ArchiveManager {
	return &ArchiveManager{
		torrentConfig: torrentConfig,
		logger:        logger,
		stdoutLogger:  stdoutLogger,
		persistence:   persistence,
		identity:      identity,
		encryptor:     encryptor,
		publisher:     publisher,
	}
}

// LogStdout appears to be some kind of debug tool specifically for torrent functionality
func (m *ArchiveManager) LogStdout(msg string, fields ...zap.Field) {
	m.stdoutLogger.Info(msg, fields...)
	m.logger.Debug(msg, fields...)
}

func (m *ArchiveManager) createHistoryArchiveTorrent(communityID types.HexBytes, msgs []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {

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

		err = os.WriteFile(torrentFile(m.torrentConfig.TorrentDir, communityID.String()), metaInfoBytes, 0644) // nolint: gosec
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

func (m *ArchiveManager) archiveIndexFile(communityID string) string {
	return path.Join(m.torrentConfig.DataDir, communityID, "index")
}

func (m *ArchiveManager) createWakuMessageArchive(from time.Time, to time.Time, messages []types.Message, topics [][]byte) *protobuf.WakuMessageArchive {
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

func (m *ArchiveManager) CreateHistoryArchiveTorrentFromMessages(communityID types.HexBytes, messages []*types.Message, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveTorrent(communityID, messages, topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManager) CreateHistoryArchiveTorrentFromDB(communityID types.HexBytes, topics []types.TopicType, startDate time.Time, endDate time.Time, partition time.Duration, encrypt bool) ([]string, error) {
	return m.createHistoryArchiveTorrent(communityID, make([]*types.Message, 0), topics, startDate, endDate, partition, encrypt)
}

func (m *ArchiveManager) GetMessageArchiveIDsToImport(communityID types.HexBytes) ([]string, error) {
	return m.persistence.GetMessageArchiveIDsToImport(communityID)
}

func (m *ArchiveManager) SaveMessageArchiveID(communityID types.HexBytes, hash string) error {
	return m.persistence.SaveMessageArchiveID(communityID, hash)
}

func (m *ArchiveManager) SetMessageArchiveIDImported(communityID types.HexBytes, hash string, imported bool) error {
	return m.persistence.SetMessageArchiveIDImported(communityID, hash, imported)
}

func (m *ArchiveManager) GetHistoryArchiveMagnetlink(communityID types.HexBytes) (string, error) {
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

func (m *ArchiveManager) archiveDataFile(communityID string) string {
	return path.Join(m.torrentConfig.DataDir, communityID, "data")
}

func (m *ArchiveManager) ExtractMessagesFromHistoryArchive(communityID types.HexBytes, archiveID string) ([]*protobuf.WakuMessage, error) {
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

func (m *ArchiveManager) LoadHistoryArchiveIndexFromFile(myKey *ecdsa.PrivateKey, communityID types.HexBytes) (*protobuf.WakuMessageArchiveIndex, error) {
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
