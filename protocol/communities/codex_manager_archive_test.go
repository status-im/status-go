package communities_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/codex-storage/codex-go-bindings/codex"
	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CodexArchiveManagerSuite struct {
	suite.Suite
	codexClient *communities.CodexClient
	// codexConfig    *params.CodexConfig
	archiveManager *communities.ArchiveManager
	manager        *communities.Manager
	uploadedCIDs   []string // Track uploaded CIDs for cleanup
}

func buildCodexConfig(t *testing.T) *params.CodexConfig {
	rootDir := t.TempDir()
	return &params.CodexConfig{
		Enabled:               true,
		HistoryArchiveDataDir: filepath.Join(rootDir, "codex", "archivedata"),
		CodexNodeConfig: codex.Config{
			DataDir:      filepath.Join(rootDir, "codex", "codexdata"),
			BlockRetries: 5,
			LogLevel:     "ERROR",
			Nat:          "none",
		},
	}
}

func (s *CodexArchiveManagerSuite) buildManagers() (*communities.Manager, *communities.ArchiveManager) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.Require().NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := tt.MustCreateTestLogger()

	m, err := communities.NewManager(key, "", db, logger, nil, nil, nil, &communities.TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	amc := &communities.ArchiveManagerConfig{
		TorrentConfig: nil,
		CodexConfig:   buildCodexConfig(s.T()),
		Logger:        logger,
		Persistence:   m.GetPersistence(),
		Messaging:     nil,
		Identity:      key,
		Publisher:     m,
	}
	t := communities.NewArchiveManager(amc)
	s.Require().NoError(err)

	return m, t
}

func (s *CodexArchiveManagerSuite) CreateCommunity() *communities.Community {
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

// SetupSuite runs once before all tests in the suite
func (s *CodexArchiveManagerSuite) SetupTest() {
	m, t := s.buildManagers()
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveManager = t
	s.Require().NoError(s.archiveManager.StartCodexClient())
	s.codexClient = s.archiveManager.GetCodexClient()
}

// TearDownSuite runs once after each test in the suite
func (s *CodexArchiveManagerSuite) TearDownTest() {
	// Clean up all uploaded CIDs
	for _, cid := range s.uploadedCIDs {
		if err := s.codexClient.RemoveCid(cid); err != nil {
			s.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		} else {
			s.T().Logf("Successfully removed CID: %s", cid)
		}
	}
	s.archiveManager.StopCodexClient()
	s.Require().NoError(s.manager.Stop())
}

func (s *CodexArchiveManagerSuite) TestDownloadingArchivesFromCodex() {
	// Subscribe to signals before starting the test
	subscription := s.manager.Subscribe()

	// Create test archive data and upload multiple archives to Codex
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

	// Generate random data for each archive
	archiveCIDs := make(map[string]string) // archive hash -> CID
	for i := range archives {
		if _, err := rand.Read(archives[i].data); err != nil {
			s.T().Fatalf("Failed to generate random data for %s: %v", archives[i].hash, err)
		}
		s.T().Logf("Generated %s data (first 16 bytes hex): %s",
			archives[i].hash, hex.EncodeToString(archives[i].data[:16]))
	}

	// Upload all archives to Codex
	for _, archive := range archives {
		cid, err := s.codexClient.Upload(bytes.NewReader(archive.data), archive.hash+".bin")
		require.NoError(s.T(), err, "Failed to upload %s", archive.hash)

		archiveCIDs[archive.hash] = cid
		s.uploadedCIDs = append(s.uploadedCIDs, cid)
		s.T().Logf("Uploaded %s to CID: %s", archive.hash, cid)

		// Verify upload succeeded
		exists, err := s.codexClient.HasCid(cid)
		require.NoError(s.T(), err, "Failed to check CID existence for %s", archive.hash)
		require.True(s.T(), exists, "CID %s should exist after upload", cid)
	}

	// Create archive index for CodexArchiveDownloader
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: make(map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata),
	}

	for _, archive := range archives {
		cid := archiveCIDs[archive.hash]
		index.Archives[archive.hash] = &protobuf.CodexWakuMessageArchiveIndexMetadata{
			Cid: cid,
			Metadata: &protobuf.WakuMessageArchiveMetadata{
				From: archive.from,
				To:   archive.to,
			},
		}
	}

	// upload archive index to codex
	codexIndexBytes, err := proto.Marshal(index)
	s.Require().NoError(err, "Failed to marshal index")

	cid, err := s.codexClient.UploadArchive(codexIndexBytes)
	s.Require().NoError(err, "Failed to upload archive index to Codex")
	s.Require().NotEmpty(cid, "Uploaded index CID should not be empty")

	s.T().Logf("Uploaded archive index to CID: %s", cid)

	// Now that we have both the individual archives and the index uploaded to Codex,
	// we can proceed with the download workflow.

	communityID := types.HexBytes("test-community-id")
	cancelChan := make(chan struct{})

	// Track received signals
	receivedSignals := struct {
		downloadingStarted bool
		archiveDownloaded  map[string]bool // hash -> received
		seedingSignal      bool
	}{
		archiveDownloaded: make(map[string]bool),
	}

	// Start goroutine to collect signals
	done := make(chan struct{})
	go func() {
		timeout := time.After(30 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.DownloadingHistoryArchivesStartedSignal != nil {
					receivedSignals.downloadingStarted = true
					s.T().Logf("Received DownloadingHistoryArchivesStartedSignal for community: %s",
						event.DownloadingHistoryArchivesStartedSignal.CommunityID)
				}
				if event.HistoryArchiveDownloadedSignal != nil {
					s.T().Logf("Received HistoryArchiveDownloadedSignal for community: %s, From: %d, To: %d",
						event.HistoryArchiveDownloadedSignal.CommunityID,
						event.HistoryArchiveDownloadedSignal.From,
						event.HistoryArchiveDownloadedSignal.To)
					// Find which archive this corresponds to
					for _, archive := range archives {
						if uint64(event.HistoryArchiveDownloadedSignal.From) == archive.from &&
							uint64(event.HistoryArchiveDownloadedSignal.To) == archive.to {
							receivedSignals.archiveDownloaded[archive.hash] = true
						}
					}
				}
				if event.HistoryArchivesSeedingSignal != nil {
					receivedSignals.seedingSignal = true
					s.T().Logf("Received HistoryArchivesSeedingSignal for community: %s, MagnetLink: %v, IndexCid: %v",
						event.HistoryArchivesSeedingSignal.CommunityID,
						event.HistoryArchivesSeedingSignal.MagnetLink,
						event.HistoryArchivesSeedingSignal.IndexCid)
				}
			case <-timeout:
				close(done)
				return
			case <-done:
				return
			}
		}
	}()

	taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, cid, cancelChan)
	s.Require().NoError(err, "Failed to download archives")
	s.Require().NotNil(taskInfo, "Download task info should not be nil")
	s.Require().Equal(len(archives), taskInfo.TotalArchivesCount, "Unexpected total archives count")
	s.Require().Equal(len(archives), taskInfo.TotalDownloadedArchivesCount, "Unexpected total downloaded archives count")
	s.Require().False(taskInfo.Cancelled, "Download should not be cancelled")

	s.T().Logf("Download task info: %+v", taskInfo)

	// Stop the signal collection goroutine
	close(done)

	// Wait a bit for any remaining signals to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify that archives are stored in persistence
	for _, archive := range archives {
		exists, err := s.manager.GetPersistence().HasMessageArchiveID(communityID, archive.hash)
		s.Require().NoError(err, "Failed to check archive ID %s in persistence", archive.hash)
		s.Require().True(exists, "Archive hash %s should be stored in persistence", archive.hash)
	}

	// Verify that all expected signals were received
	s.Require().True(receivedSignals.downloadingStarted, "Should have received DownloadingHistoryArchivesStartedSignal")
	s.Require().True(receivedSignals.seedingSignal, "Should have received HistoryArchivesSeedingSignal")

	// Verify that we received download signals for all archives
	for _, archive := range archives {
		s.Require().True(receivedSignals.archiveDownloaded[archive.hash],
			"Should have received HistoryArchiveDownloadedSignal for archive %s", archive.hash)
	}

	s.T().Logf("All signals verified successfully!")
}

// Run the integration test suite
func TestCodexArchiveManagerSuite(t *testing.T) {
	suite.Run(t, new(CodexArchiveManagerSuite))
}
