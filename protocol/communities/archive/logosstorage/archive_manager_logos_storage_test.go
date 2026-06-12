//go:build use_logos_storage

package logosstorage_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/archive"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	logosstorage "github.com/status-im/status-go/services/logosstorage"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ArchiveManagerLogosStorageSuite struct {
	suite.Suite
	client         logosstorage.LogosStorageClientInterface
	archiveService archive.ArchiveService
	manager        *communities.Manager
	identity       *ecdsa.PrivateKey
	uploadedCIDs   []string
}

func buildLogosStorageConfig(t *testing.T) *params.LogosStorageConfig {
	rootDir := t.TempDir()
	return &params.LogosStorageConfig{
		Enabled: true,
		NodeConfig: params.LogosStorageNodeConfig{
			DataDir:      filepath.Join(rootDir, "logos-storage", "data"),
			BlockRetries: 5,
			LogLevel:     "ERROR",
			Nat:          "none",
		},
	}
}

func (s *ArchiveManagerLogosStorageSuite) buildManagers() (*communities.Manager, archive.ArchiveService, *ecdsa.PrivateKey) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	s.Require().NoError(sqlite.Migrate(db), "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := testutils.MustCreateTestLogger()

	m, err := communities.NewManager(key, "", db, logger, nil, nil, nil, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	amc := &archivetypes.ArchiveManagerConfig{
		LogosStorageConfig: buildLogosStorageConfig(s.T()),
		Logger:             logger,
		Persistence:        m.GetPersistence(),
		Identity:           key,
		Publisher:          m,
	}

	return m, archive.NewArchiveManager(amc), key
}

func (s *ArchiveManagerLogosStorageSuite) getArchiveManager() *archive.ArchiveManager {
	archiveManager, ok := s.archiveService.(*archive.ArchiveManager)
	s.Require().True(ok)
	return archiveManager
}

func (s *ArchiveManagerLogosStorageSuite) CreateCommunity() *communities.Community {
	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "token membership description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	return community
}

func (s *ArchiveManagerLogosStorageSuite) createCommunityWithChat() (*communities.Community, string) {
	community := s.CreateCommunity()
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "added-chat",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}
	changes, err := s.manager.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)
	s.Require().Len(changes.ChatsAdded, 1)
	chatID := ""
	for cID := range changes.ChatsAdded {
		chatID = cID
		break
	}
	return community, chatID
}

func buildLogosStorageMessage(timestamp time.Time, topic messagingtypes.ContentTopic, hash []byte) messagingtypes.ReceivedMessage {
	return messagingtypes.ReceivedMessage{
		Sig:       []byte{1},
		Timestamp: uint32(timestamp.Unix()),
		Topic:     topic,
		Payload:   []byte{1},
		Padding:   []byte{1},
		Hash:      hash,
	}
}

func (s *ArchiveManagerLogosStorageSuite) SetupTest() {
	m, t, key := s.buildManagers()
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveService = t
	s.identity = key
	s.Require().NoError(s.archiveService.Start())
	archiveManager := s.getArchiveManager()
	backend, err := archiveManager.GetLogosStorageBackend()
	s.Require().NoError(err)
	s.Require().NotNil(backend)
	client := backend.GetClient()
	s.Require().NotNil(client)
	s.client = client
}

func (s *ArchiveManagerLogosStorageSuite) TearDownTest() {
	for _, cid := range s.uploadedCIDs {
		if err := s.client.RemoveCid(cid); err != nil {
			s.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		}
	}
	s.Require().NoError(s.archiveService.Stop())
	s.Require().NoError(s.manager.Stop())
}

func (s *ArchiveManagerLogosStorageSuite) TestDownloadingArchivesFromLogosStorage() {
	subscription := s.manager.Subscribe()

	archives := []struct {
		hash string
		from uint64
		to   uint64
		data []byte
	}{
		{"archive-1-hash-abc123", 1000, 2000, make([]byte, 512)},
		{"archive-2-hash-def456", 2000, 3000, make([]byte, 768)},
		{"archive-3-hash-ghi789", 3000, 4000, make([]byte, 1024)},
	}

	archiveCIDs := make(map[string]string)
	for i := range archives {
		_, err := rand.Read(archives[i].data)
		s.Require().NoError(err)
		s.T().Logf("Generated %s data (first 16 bytes hex): %s", archives[i].hash, hex.EncodeToString(archives[i].data[:16]))
	}

	for _, archive := range archives {
		cid, err := s.client.Upload(bytes.NewReader(archive.data), archive.hash+".bin")
		require.NoError(s.T(), err, "Failed to upload %s", archive.hash)

		archiveCIDs[archive.hash] = cid
		s.uploadedCIDs = append(s.uploadedCIDs, cid)

		exists, err := s.client.HasCid(cid)
		require.NoError(s.T(), err, "Failed to check CID existence for %s", archive.hash)
		require.True(s.T(), exists, "CID %s should exist after upload", cid)
	}

	index := &protobuf.LogosStorageWakuMessageArchiveIndex{
		Archives: make(map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata),
	}

	for _, archive := range archives {
		cid := archiveCIDs[archive.hash]
		index.Archives[archive.hash] = &protobuf.LogosStorageWakuMessageArchiveIndexMetadata{
			Cid: cid,
			Metadata: &protobuf.WakuMessageArchiveMetadata{
				From: archive.from,
				To:   archive.to,
			},
		}
	}

	indexBytes, err := proto.Marshal(index)
	s.Require().NoError(err, "Failed to marshal index")

	cid, err := s.client.UploadArchive(indexBytes)
	s.Require().NoError(err, "Failed to upload archive index to LogosStorage")
	s.Require().NotEmpty(cid, "Uploaded index CID should not be empty")

	communityID := types.HexBytes("test-community-id")
	cancelChan := make(chan struct{})

	receivedSignals := struct {
		downloadingStarted bool
		archiveDownloaded  map[string]bool
	}{
		archiveDownloaded: make(map[string]bool),
	}

	done := make(chan struct{})
	go func() {
		timeout := time.After(30 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.DownloadingHistoryArchivesStartedSignal != nil {
					receivedSignals.downloadingStarted = true
				}
				if event.HistoryArchiveDownloadedSignal != nil {
					for _, archive := range archives {
						if uint64(event.HistoryArchiveDownloadedSignal.From) == archive.from &&
							uint64(event.HistoryArchiveDownloadedSignal.To) == archive.to {
							receivedSignals.archiveDownloaded[archive.hash] = true
						}
					}
				}
			case <-timeout:
				close(done)
				return
			case <-done:
				return
			}
		}
	}()

	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, cid, cancelChan)
	s.Require().NoError(err, "Failed to download archives")
	s.Require().NotNil(taskInfo, "Download task info should not be nil")
	s.Require().Equal(len(archives), taskInfo.TotalArchivesCount, "Unexpected total archives count")
	s.Require().Equal(len(archives), taskInfo.TotalDownloadedArchivesCount, "Unexpected total downloaded archives count")
	s.Require().False(taskInfo.Cancelled, "Download should not be cancelled")

	close(done)
	time.Sleep(100 * time.Millisecond)

	for _, archive := range archives {
		exists, err := s.manager.GetPersistence().HasMessageArchiveID(communityID, archive.hash)
		s.Require().NoError(err, "Failed to check archive ID %s in persistence", archive.hash)
		s.Require().True(exists, "Archive hash %s should be stored in persistence", archive.hash)
	}

	s.Require().True(receivedSignals.downloadingStarted, "Should have received DownloadingHistoryArchivesStartedSignal")

	for _, archive := range archives {
		s.Require().True(receivedSignals.archiveDownloaded[archive.hash], "Should have received HistoryArchiveDownloadedSignal for archive %s", archive.hash)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	archiveManager := s.getArchiveManager()
	backend, err := archiveManager.GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	loadedIndex, err := backend.LoadHistoryArchiveIndex(ctx, s.identity, communityID, cid, true)
	s.Require().NoError(err, "Failed to load index file from disk")
	s.Require().NotNil(loadedIndex, "Loaded index should not be nil")
	s.Require().Equal(len(archives), len(loadedIndex.Archives), "Loaded index should contain all archives")

	for _, archive := range archives {
		loadedMetadata, exists := loadedIndex.Archives[archive.hash]
		s.Require().True(exists, "Archive %s should exist in loaded index", archive.hash)
		s.Require().NotNil(loadedMetadata, "Archive metadata should not be nil for %s", archive.hash)
		s.Require().Equal(archiveCIDs[archive.hash], loadedMetadata.Cid, "CID should match for archive %s", archive.hash)
		s.Require().Equal(archive.from, loadedMetadata.Metadata.From, "From timestamp should match for archive %s", archive.hash)
		s.Require().Equal(archive.to, loadedMetadata.Metadata.To, "To timestamp should match for archive %s", archive.hash)
	}
}

func (s *ArchiveManagerLogosStorageSuite) TestCreateAndSeedHistoryArchiveKeepsCumulativeIndexAndUnseedsOldIndex() {
	community, chatID := s.createCommunityWithChat()
	topic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []messagingtypes.ContentTopic{topic}

	startDate := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	partition := 10 * time.Second

	message1 := buildLogosStorageMessage(startDate.Add(1*time.Second), topic, []byte{1})
	s.Require().NoError(s.manager.StoreWakuMessages([]*messagingtypes.ReceivedMessage{&message1}))

	s.Require().NoError(s.archiveService.CreateAndSeedHistoryArchive(
		community.ID(), topics, startDate, startDate.Add(partition), partition, false,
	))

	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err)

	firstIndexCid, err := s.manager.GetPersistence().GetLastSeenArchiveLink(community.ID())
	s.Require().NoError(err)
	s.Require().NotEmpty(firstIndexCid)
	s.uploadedCIDs = append(s.uploadedCIDs, firstIndexCid)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	firstIndex, err := backend.LoadHistoryArchiveIndex(ctx, s.identity, community.ID(), firstIndexCid, true)
	s.Require().NoError(err)
	s.Require().Len(firstIndex.Archives, 1)

	message2 := buildLogosStorageMessage(startDate.Add(partition).Add(1*time.Second), topic, []byte{2})
	s.Require().NoError(s.manager.StoreWakuMessages([]*messagingtypes.ReceivedMessage{&message2}))

	s.Require().NoError(s.archiveService.CreateAndSeedHistoryArchive(
		community.ID(), topics, startDate.Add(partition), startDate.Add(2*partition), partition, false,
	))

	secondIndexCid, err := s.manager.GetPersistence().GetLastSeenArchiveLink(community.ID())
	s.Require().NoError(err)
	s.Require().NotEmpty(secondIndexCid)
	s.Require().NotEqual(firstIndexCid, secondIndexCid)
	s.uploadedCIDs = append(s.uploadedCIDs, secondIndexCid)

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	secondIndex, err := backend.LoadHistoryArchiveIndex(ctx, s.identity, community.ID(), secondIndexCid, true)
	s.Require().NoError(err)
	s.Require().Len(secondIndex.Archives, 2)

	for archiveID := range firstIndex.Archives {
		s.Require().Contains(secondIndex.Archives, archiveID)
	}

	s.Require().Eventually(func() bool {
		hasCid, err := s.client.HasCid(firstIndexCid)
		return err == nil && !hasCid
	}, 2*time.Second, 100*time.Millisecond, "old index CID should be unseeded after new cumulative index is created")
}

func TestArchiveManagerLogosStorageSuite(t *testing.T) {
	suite.Run(t, new(ArchiveManagerLogosStorageSuite))
}
