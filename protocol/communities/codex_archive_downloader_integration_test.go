package communities_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// CodexArchiveDownloaderIntegrationSuite tests the full archive download workflow
// against a real Codex instance
type CodexArchiveDownloaderIntegrationSuite struct {
	suite.Suite
	client       communities.CodexClientInterface
	uploadedCIDs []string // Track uploaded CIDs for cleanup
}

// SetupTest runs before each test method
func (suite *CodexArchiveDownloaderIntegrationSuite) SetupTest() {
	suite.client = NewCodexClientTest(suite.T())
}

// TearDownTest runs after each test method
func (suite *CodexArchiveDownloaderIntegrationSuite) TearDownTest() {
	// Clean up all uploaded CIDs
	for _, cid := range suite.uploadedCIDs {
		if err := suite.client.RemoveCid(cid); err != nil {
			suite.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		} else {
			suite.T().Logf("Successfully removed CID: %s", cid)
		}
	}
}

func (suite *CodexArchiveDownloaderIntegrationSuite) TestFullArchiveDownloadWorkflow() {
	// Step 1: Create test archive data and upload multiple archives to Codex
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
			suite.T().Fatalf("Failed to generate random data for %s: %v", archives[i].hash, err)
		}
		suite.T().Logf("Generated %s data (first 16 bytes hex): %s",
			archives[i].hash, hex.EncodeToString(archives[i].data[:16]))
	}

	// Upload all archives to Codex
	for _, archive := range archives {
		cid, err := suite.client.Upload(bytes.NewReader(archive.data), archive.hash+".bin")
		require.NoError(suite.T(), err, "Failed to upload %s", archive.hash)

		archiveCIDs[archive.hash] = cid
		suite.uploadedCIDs = append(suite.uploadedCIDs, cid)
		suite.T().Logf("Uploaded %s to CID: %s", archive.hash, cid)

		// Verify upload succeeded
		exists, err := suite.client.HasCid(cid)
		require.NoError(suite.T(), err, "Failed to check CID existence for %s", archive.hash)
		require.True(suite.T(), exists, "CID %s should exist after upload", cid)
	}

	// Step 2: Create archive index for CodexArchiveDownloader
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

	// Step 3: Set up CodexArchiveDownloader
	communityID := "integration-test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})
	logger, _ := zap.NewDevelopment() // Use development logger for integration tests

	downloader := communities.NewCodexArchiveDownloader(
		suite.client,
		index,
		communityID,
		existingArchiveIDs,
		cancelChan,
		logger,
	)

	// Configure for integration test - use reasonable intervals
	downloader.SetPollingInterval(500 * time.Millisecond)
	downloader.SetPollingTimeout(30 * time.Second) // Generous timeout for real network

	// Step 4: Set up callbacks to track progress
	var trackerMu sync.Mutex
	startedArchives := make(map[string]bool)
	completedArchives := make(map[string]bool)

	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		trackerMu.Lock()
		startedArchives[hash] = true
		trackerMu.Unlock()
		suite.T().Logf("🚀 Started downloading archive: %s (from %d to %d)", hash, from, to)
	})

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		trackerMu.Lock()
		completedArchives[hash] = true
		trackerMu.Unlock()
		suite.T().Logf("✅ Completed downloading archive: %s (from %d to %d)", hash, from, to)
	})

	// Step 5: Verify initial state
	assert.Equal(suite.T(), 3, downloader.GetTotalArchivesCount(), "Should have 3 total archives")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "Should start with 0 downloaded")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Should not be complete initially")

	// Step 6: Start the download process
	suite.T().Log("🎯 Starting archive download process...")
	downloader.StartDownload()

	// Step 7: Wait for all downloads to complete with reasonable timeout
	require.Eventually(suite.T(), func() bool {
		complete := downloader.IsDownloadComplete()
		if !complete {
			suite.T().Logf("Download progress: %d/%d completed, %d pending",
				downloader.GetTotalDownloadedArchivesCount(),
				downloader.GetTotalArchivesCount(),
				downloader.GetPendingArchivesCount())
		}
		return complete
	}, 60*time.Second, 2*time.Second, "All downloads should complete within 60 seconds")

	// Step 8: Verify final state
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")
	assert.Equal(suite.T(), 3, downloader.GetTotalDownloadedArchivesCount(), "Should have downloaded all 3 archives")
	assert.Equal(suite.T(), 0, downloader.GetPendingArchivesCount(), "Should have no pending archives")

	// Verify all archives were processed
	trackerMu.Lock()
	assert.Len(suite.T(), startedArchives, 3, "Should have started 3 archives")
	assert.Len(suite.T(), completedArchives, 3, "Should have completed 3 archives")

	for _, archive := range archives {
		assert.Contains(suite.T(), startedArchives, archive.hash, "Should have started %s", archive.hash)
		assert.Contains(suite.T(), completedArchives, archive.hash, "Should have completed %s", archive.hash)
	}
	trackerMu.Unlock()

	// Step 9: Verify we can download the actual archive content using LocalDownloadWithContext
	suite.T().Log("🔍 Verifying archive content can be downloaded...")

	for completedHash := range completedArchives {
		cid := archiveCIDs[completedHash]

		suite.T().Logf("Downloading and verifying content for archive: %s (CID: %s)", completedHash, cid)

		// Find the original archive data for comparison
		var originalData []byte
		for _, archive := range archives {
			if archive.hash == completedHash {
				originalData = archive.data
				break
			}
		}

		// Create context with timeout for download
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		var downloadBuf bytes.Buffer
		err := suite.client.LocalDownloadWithContext(ctx, cid, &downloadBuf)
		cancel()

		require.NoError(suite.T(), err, "LocalDownload should succeed for %s", completedHash)

		downloadedData := downloadBuf.Bytes()
		assert.Equal(suite.T(), len(originalData), len(downloadedData),
			"Downloaded data length should match for %s", completedHash)
		assert.True(suite.T(), bytes.Equal(originalData, downloadedData),
			"Downloaded data should match original for %s", completedHash)

		suite.T().Logf("✅ Verified content for %s: %d bytes match", completedHash, len(downloadedData))
	}

	suite.T().Log("🎉 Full archive download workflow completed successfully!")
	suite.T().Logf("   - Uploaded %d archives to Codex", len(archives))
	suite.T().Logf("   - Triggered download for all archives")
	suite.T().Logf("   - Polled until all downloads completed")
	suite.T().Logf("   - Verified all archive content matches original data")
}

// Run the integration test suite
func TestCodexArchiveDownloaderIntegrationSuite(t *testing.T) {
	suite.Run(t, new(CodexArchiveDownloaderIntegrationSuite))
}
