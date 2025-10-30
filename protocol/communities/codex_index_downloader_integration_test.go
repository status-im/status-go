//go:build codex_integration
// +build codex_integration

package communities_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codex-storage/codex-go-bindings/codex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/communities"
)

// CodexIndexDownloaderIntegrationTestSuite demonstrates testify's suite functionality for CodexIndexDownloader integration tests
// These tests exercise real network calls against a running Codex node.
// Required env vars (with defaults):
//   - CODEX_HOST (default: localhost)
//   - CODEX_API_PORT (default: 8001)
//   - CODEX_TIMEOUT_MS (optional; default: 60000)
type CodexIndexDownloaderIntegrationTestSuite struct {
	suite.Suite
	client  *communities.CodexClient
	testDir string
	host    string
	port    string
	logger  *zap.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *CodexIndexDownloaderIntegrationTestSuite) SetupSuite() {
	suite.host = communities.GetEnvOrDefault("CODEX_HOST", "localhost")
	suite.port = communities.GetEnvOrDefault("CODEX_API_PORT", "8001")
	client, err := communities.NewCodexClient(codex.Config{
		DataDir:        suite.T().TempDir(),
		LogFormat:      codex.LogFormatNoColors,
		MetricsEnabled: false,
		BlockRetries:   5,
	})
	if err != nil {
		suite.T().Fatalf("Failed to create Codex client: %v", err)
	}
	suite.client = &client

	// Create logger
	suite.logger, _ = zap.NewDevelopment()
}

// SetupTest runs before each test
func (suite *CodexIndexDownloaderIntegrationTestSuite) SetupTest() {
	// Create a temporary directory for test files
	var err error
	suite.testDir, err = os.MkdirTemp("", "codex-index-integration-*")
	require.NoError(suite.T(), err)
}

// TearDownTest runs after each test
func (suite *CodexIndexDownloaderIntegrationTestSuite) TearDownTest() {
	// Clean up test directory
	if suite.testDir != "" {
		os.RemoveAll(suite.testDir)
	}
}

// TestCodexIndexDownloaderIntegrationTestSuite runs the integration test suite
func TestCodexIndexDownloaderIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CodexIndexDownloaderIntegrationTestSuite))
}

func (suite *CodexIndexDownloaderIntegrationTestSuite) TestIntegration_GotManifest() {
	// Generate random payload to create a test file
	payload := make([]byte, 2048)
	_, err := rand.Read(payload)
	require.NoError(suite.T(), err, "failed to generate random payload")
	suite.T().Logf("Generated payload (first 32 bytes hex): %s", hex.EncodeToString(payload[:32]))

	// Upload the data to Codex
	cid, err := suite.client.Upload(bytes.NewReader(payload), "index-manifest-test.bin")
	require.NoError(suite.T(), err, "upload failed")
	suite.T().Logf("Upload successful, CID: %s", cid)

	// Clean up after test
	defer func() {
		if err := suite.client.RemoveCid(cid); err != nil {
			suite.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		}
	}()

	// Create downloader with cancel channel
	cancelChan := make(chan struct{})
	defer close(cancelChan)

	filePath := filepath.Join(suite.testDir, "test-index.bin")
	downloader := communities.NewCodexIndexDownloader(suite.client, cid, filePath, cancelChan, suite.logger)

	// Test GotManifest
	manifestChan := downloader.GotManifest()

	// Wait for manifest to be fetched (with timeout)
	select {
	case <-manifestChan:
		suite.T().Log("✅ Manifest fetched successfully")
	case <-time.After(10 * time.Second):
		suite.T().Fatal("Timeout waiting for manifest to be fetched")
	}

	// Verify dataset size was recorded
	datasetSize := downloader.GetDatasetSize()
	assert.Greater(suite.T(), datasetSize, int64(0), "Dataset size should be greater than 0")
	suite.T().Logf("Dataset size from manifest: %d bytes", datasetSize)

	// Verify Length returns the same value
	assert.Equal(suite.T(), datasetSize, downloader.Length(), "Length() should return dataset size")

	// Verify no error occurred
	assert.NoError(suite.T(), downloader.GetError(), "No error should occur during manifest fetch")
	suite.T().Log("✅ No errors during manifest fetch")
}

func (suite *CodexIndexDownloaderIntegrationTestSuite) TestIntegration_DownloadIndexFile() {
	// Generate random payload
	payload := make([]byte, 1024)
	_, err := rand.Read(payload)
	require.NoError(suite.T(), err, "failed to generate random payload")
	suite.T().Logf("Generated payload (first 32 bytes hex): %s", hex.EncodeToString(payload[:32]))

	// Upload the data to Codex
	cid, err := suite.client.Upload(bytes.NewReader(payload), "index-download-test.bin")
	require.NoError(suite.T(), err, "upload failed")
	suite.T().Logf("Upload successful, CID: %s", cid)

	// Clean up after test
	defer func() {
		if err := suite.client.RemoveCid(cid); err != nil {
			suite.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		}
	}()

	// Create downloader
	cancelChan := make(chan struct{})
	defer close(cancelChan)

	filePath := filepath.Join(suite.testDir, "downloaded-index.bin")
	downloader := communities.NewCodexIndexDownloader(suite.client, cid, filePath, cancelChan, suite.logger)

	// First, get the manifest to know the expected size
	manifestChan := downloader.GotManifest()
	select {
	case <-manifestChan:
		suite.T().Log("Manifest fetched")
	case <-time.After(10 * time.Second):
		suite.T().Fatal("Timeout waiting for manifest")
	}

	expectedSize := downloader.GetDatasetSize()
	suite.T().Logf("Expected file size: %d bytes", expectedSize)

	// Verify no error from manifest fetch
	assert.NoError(suite.T(), downloader.GetError(), "No error should occur during manifest fetch")

	// Start the download
	downloader.DownloadIndexFile()

	// Wait for download to complete by monitoring progress
	require.Eventually(suite.T(), func() bool {
		return downloader.BytesCompleted() == expectedSize
	}, 30*time.Second, 100*time.Millisecond, "Download should complete")

	suite.T().Logf("✅ Download completed: %d/%d bytes", downloader.BytesCompleted(), expectedSize)

	// Verify download is marked as complete
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be marked as complete")
	suite.T().Log("✅ Download marked as complete")

	// Verify no error occurred during download
	assert.NoError(suite.T(), downloader.GetError(), "No error should occur during download")
	suite.T().Log("✅ No errors during download")

	// Verify file exists and has correct size
	stat, err := os.Stat(filePath)
	require.NoError(suite.T(), err, "Downloaded file should exist")
	assert.Equal(suite.T(), expectedSize, stat.Size(), "File size should match dataset size")

	// Verify file contents match original payload
	downloadedData, err := os.ReadFile(filePath)
	require.NoError(suite.T(), err, "Should be able to read downloaded file")
	assert.Equal(suite.T(), payload, downloadedData, "Downloaded data should match original payload")
	suite.T().Log("✅ Downloaded file contents verified")
}
