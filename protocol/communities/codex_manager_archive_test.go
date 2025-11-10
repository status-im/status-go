package communities_test

import (
	"bytes"
	"crypto/ecdsa"
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
	codexClient    communities.CodexClientInterface
	archiveManager *communities.ArchiveManager
	manager        *communities.Manager
	identity       *ecdsa.PrivateKey // Store identity for test access
	uploadedCIDs   []string          // Track uploaded CIDs for cleanup
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

func (s *CodexArchiveManagerSuite) buildManagers() (*communities.Manager, *communities.ArchiveManager, *ecdsa.PrivateKey) {
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

	return m, t, key
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
	m, t, key := s.buildManagers()
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveManager = t
	s.identity = key
	s.Require().NoError(s.archiveManager.StartCodexClient())
	client := s.archiveManager.GetCodexClient()
	s.Require().NotNil(client)
	s.codexClient = client
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
	s.Require().NoError(s.archiveManager.StopCodexClient())
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

	// Verify that the index file exists and has correct content
	loadedIndex, err := s.archiveManager.CodexLoadHistoryArchiveIndexFromFile(s.identity, communityID)
	s.Require().NoError(err, "Failed to load index file from disk")
	s.Require().NotNil(loadedIndex, "Loaded index should not be nil")
	s.Require().Equal(len(archives), len(loadedIndex.Archives), "Loaded index should contain all archives")

	// Verify each archive in the loaded index matches the original
	for _, archive := range archives {
		loadedMetadata, exists := loadedIndex.Archives[archive.hash]
		s.Require().True(exists, "Archive %s should exist in loaded index", archive.hash)
		s.Require().NotNil(loadedMetadata, "Archive metadata should not be nil for %s", archive.hash)
		s.Require().Equal(archiveCIDs[archive.hash], loadedMetadata.Cid, "CID should match for archive %s", archive.hash)
		s.Require().Equal(archive.from, loadedMetadata.Metadata.From, "From timestamp should match for archive %s", archive.hash)
		s.Require().Equal(archive.to, loadedMetadata.Metadata.To, "To timestamp should match for archive %s", archive.hash)
	}

	s.T().Logf("Index file content verified successfully!")

	// Verify that the CID file exists and contains the correct CID
	storedCid, err := s.archiveManager.GetHistoryArchiveIndexCid(communityID)
	s.Require().NoError(err, "Failed to read CID file")
	s.Require().Equal(cid, storedCid, "Stored CID should match the uploaded index CID")

	s.T().Logf("CID file content verified successfully! CID: %s", storedCid)
}

func (s *CodexArchiveManagerSuite) TestDownloadCancellationBeforeManifestFetch() {
	// Subscribe to signals
	subscription := s.manager.Subscribe()

	// Create a single test archive
	archiveData := make([]byte, 256)
	_, err := rand.Read(archiveData)
	s.Require().NoError(err, "Failed to generate random data")

	// Upload archive to Codex
	archiveCid, err := s.codexClient.Upload(bytes.NewReader(archiveData), "test-archive.bin")
	s.Require().NoError(err, "Failed to upload archive")
	s.uploadedCIDs = append(s.uploadedCIDs, archiveCid)

	// Create and upload index
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"test-hash": {
				Cid: archiveCid,
				Metadata: &protobuf.WakuMessageArchiveMetadata{
					From: 1000,
					To:   2000,
				},
			},
		},
	}

	codexIndexBytes, err := proto.Marshal(index)
	s.Require().NoError(err, "Failed to marshal index")

	indexCid, err := s.codexClient.UploadArchive(codexIndexBytes)
	s.Require().NoError(err, "Failed to upload index")
	s.uploadedCIDs = append(s.uploadedCIDs, indexCid)

	communityID := types.HexBytes("cancel-test-community-1")
	cancelChan := make(chan struct{})

	// Track signals
	downloadStartedReceived := false
	manifestFetchedReceived := false
	signalDone := make(chan struct{})
	go func() {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.DownloadingHistoryArchivesStartedSignal != nil {
					downloadStartedReceived = true
				}
				if event.ManifestFetchedSignal != nil {
					manifestFetchedReceived = true
				}
			case <-timeout:
				close(signalDone)
				return
			case <-signalDone:
				return
			}
		}
	}()

	// Cancel immediately before the download starts
	close(cancelChan)

	taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, indexCid, cancelChan)
	s.Require().NoError(err, "Download should return without error on cancellation")
	s.Require().NotNil(taskInfo, "Task info should not be nil")
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")
	s.Require().Equal(0, taskInfo.TotalDownloadedArchivesCount, "No archives should be downloaded")

	close(signalDone)
	time.Sleep(100 * time.Millisecond)

	// Verify that neither signal was received
	s.Require().False(downloadStartedReceived, "DownloadingHistoryArchivesStartedSignal should not be received when cancelled early")
	s.Require().False(manifestFetchedReceived, "ManifestFetchedSignal should not be received when cancelled early")

	s.T().Logf("Early cancellation test passed successfully!")
}

func (s *CodexArchiveManagerSuite) TestDownloadCancellationDuringIndexDownload() {
	// Subscribe to signals
	subscription := s.manager.Subscribe()

	// Create a test archive
	archiveData := make([]byte, 1024*10) // 10KB
	_, err := rand.Read(archiveData)
	s.Require().NoError(err, "Failed to generate random data")

	// Upload archive to Codex
	archiveCid, err := s.codexClient.Upload(bytes.NewReader(archiveData), "test-archive-large.bin")
	s.Require().NoError(err, "Failed to upload archive")
	s.uploadedCIDs = append(s.uploadedCIDs, archiveCid)

	// Create and upload index
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"test-hash-large": {
				Cid: archiveCid,
				Metadata: &protobuf.WakuMessageArchiveMetadata{
					From: 1000,
					To:   2000,
				},
			},
		},
	}

	codexIndexBytes, err := proto.Marshal(index)
	s.Require().NoError(err, "Failed to marshal index")

	indexCid, err := s.codexClient.UploadArchive(codexIndexBytes)
	s.Require().NoError(err, "Failed to upload index")
	s.uploadedCIDs = append(s.uploadedCIDs, indexCid)

	communityID := types.HexBytes("cancel-test-community-2")
	cancelChan := make(chan struct{})

	// Track signals
	manifestFetchedReceived := false
	indexDownloadCompletedReceived := false
	downloadStartedReceived := false
	signalDone := make(chan struct{})

	go func() {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.ManifestFetchedSignal != nil {
					manifestFetchedReceived = true
					s.T().Logf("Received ManifestFetchedSignal - now cancelling during index download")
					// Cancel as soon as we get the manifest (before index download completes)
					close(cancelChan)
				}
				if event.IndexDownloadCompletedSignal != nil {
					indexDownloadCompletedReceived = true
				}
				if event.DownloadingHistoryArchivesStartedSignal != nil {
					downloadStartedReceived = true
				}
			case <-timeout:
				close(signalDone)
				return
			case <-signalDone:
				return
			}
		}
	}()

	// Start download in goroutine
	resultChan := make(chan struct {
		taskInfo *communities.HistoryArchiveDownloadTaskInfo
		err      error
	}, 1)

	go func() {
		taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, indexCid, cancelChan)
		resultChan <- struct {
			taskInfo *communities.HistoryArchiveDownloadTaskInfo
			err      error
		}{taskInfo, err}
	}()

	result := <-resultChan
	s.Require().NoError(result.err, "Download should return without error on cancellation")
	s.Require().NotNil(result.taskInfo, "Task info should not be nil")
	s.Require().True(result.taskInfo.Cancelled, "Download should be marked as cancelled")

	close(signalDone)
	time.Sleep(100 * time.Millisecond)

	// Verify signals
	s.Require().True(manifestFetchedReceived, "Should have received ManifestFetchedSignal")
	s.Require().False(indexDownloadCompletedReceived, "Should NOT have received IndexDownloadCompletedSignal (cancelled before completion)")
	s.Require().False(downloadStartedReceived, "Should NOT have received DownloadingHistoryArchivesStartedSignal (cancelled before archives start)")

	s.T().Logf("Index download cancellation test passed! Cancelled deterministically after manifest fetch.")
}

func (s *CodexArchiveManagerSuite) TestDownloadCancellationDuringArchiveDownload() {
	// Subscribe to signals
	subscription := s.manager.Subscribe()

	// Create multiple test archives
	archives := []struct {
		hash string
		from uint64
		to   uint64
		data []byte
	}{
		{"cancel-archive-1", 1000, 2000, make([]byte, 1024*5)}, // 5KB
		{"cancel-archive-2", 2000, 3000, make([]byte, 1024*5)},
		{"cancel-archive-3", 3000, 4000, make([]byte, 1024*5)},
	}

	// Generate and upload archives
	archiveCIDs := make(map[string]string)
	for i := range archives {
		_, err := rand.Read(archives[i].data)
		s.Require().NoError(err, "Failed to generate random data")

		cid, err := s.codexClient.Upload(bytes.NewReader(archives[i].data), archives[i].hash+".bin")
		s.Require().NoError(err, "Failed to upload archive")
		archiveCIDs[archives[i].hash] = cid
		s.uploadedCIDs = append(s.uploadedCIDs, cid)
	}

	// Create and upload index
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: make(map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata),
	}

	for _, archive := range archives {
		index.Archives[archive.hash] = &protobuf.CodexWakuMessageArchiveIndexMetadata{
			Cid: archiveCIDs[archive.hash],
			Metadata: &protobuf.WakuMessageArchiveMetadata{
				From: archive.from,
				To:   archive.to,
			},
		}
	}

	codexIndexBytes, err := proto.Marshal(index)
	s.Require().NoError(err, "Failed to marshal index")

	indexCid, err := s.codexClient.UploadArchive(codexIndexBytes)
	s.Require().NoError(err, "Failed to upload index")
	s.uploadedCIDs = append(s.uploadedCIDs, indexCid)

	communityID := types.HexBytes("cancel-test-community-3")
	cancelChan := make(chan struct{})

	// Track signals
	downloadStartedReceived := false
	indexDownloadCompletedReceived := false
	archivesDownloaded := 0
	signalDone := make(chan struct{})

	go func() {
		timeout := time.After(15 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.DownloadingHistoryArchivesStartedSignal != nil {
					downloadStartedReceived = true
					s.T().Logf("Received DownloadingHistoryArchivesStartedSignal")
				}
				if event.IndexDownloadCompletedSignal != nil {
					indexDownloadCompletedReceived = true
					s.T().Logf("Received IndexDownloadCompletedSignal - waiting for first archive download before cancelling")
				}
				if event.HistoryArchiveDownloadedSignal != nil {
					archivesDownloaded++
					s.T().Logf("Received HistoryArchiveDownloadedSignal (%d archives downloaded so far)", archivesDownloaded)
					// Cancel after the first archive is downloaded
					if archivesDownloaded == 1 {
						s.T().Logf("Cancelling after first archive download")
						close(cancelChan)
					}
				}
			case <-timeout:
				close(signalDone)
				return
			case <-signalDone:
				return
			}
		}
	}()

	// Start download in goroutine
	resultChan := make(chan struct {
		taskInfo *communities.HistoryArchiveDownloadTaskInfo
		err      error
	}, 1)

	go func() {
		taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, indexCid, cancelChan)
		resultChan <- struct {
			taskInfo *communities.HistoryArchiveDownloadTaskInfo
			err      error
		}{taskInfo, err}
	}()

	result := <-resultChan
	s.Require().NoError(result.err, "Download should return without error on cancellation")
	s.Require().NotNil(result.taskInfo, "Task info should not be nil")
	s.Require().True(result.taskInfo.Cancelled, "Download should be marked as cancelled")

	close(signalDone)
	time.Sleep(100 * time.Millisecond)

	// Verify signals
	s.Require().True(downloadStartedReceived, "Should have received DownloadingHistoryArchivesStartedSignal")
	s.Require().True(indexDownloadCompletedReceived, "Should have received IndexDownloadCompletedSignal")
	s.Require().GreaterOrEqual(archivesDownloaded, 1, "Should have downloaded at least 1 archive before cancellation (via signals)")

	s.T().Logf("Archive download cancellation test passed! Cancelled deterministically after downloading %d archive(s)", archivesDownloaded)
	s.T().Logf("Task info: TotalArchivesCount=%d, TotalDownloadedArchivesCount=%d, Cancelled=%v",
		result.taskInfo.TotalArchivesCount,
		result.taskInfo.TotalDownloadedArchivesCount,
		result.taskInfo.Cancelled)

	// Note: Due to parallel downloads, the TotalDownloadedArchivesCount in taskInfo might not match
	// the number of signals received because cancellation can happen while downloads are in-flight.
	// The important thing is that we successfully cancelled based on a signal and the Cancelled flag is set.
	s.T().Logf("Signals received: %d archives downloaded, TaskInfo reports: %d archives",
		archivesDownloaded, result.taskInfo.TotalDownloadedArchivesCount)
}

// Run the integration test suite
func TestCodexArchiveManagerSuite(t *testing.T) {
	suite.Run(t, new(CodexArchiveManagerSuite))
}
