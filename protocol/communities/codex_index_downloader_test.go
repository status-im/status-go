package communities_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/communities"
	mock_communities "github.com/status-im/status-go/protocol/communities/mock/communities"
)

// CodexIndexDownloaderTestSuite demonstrates testify's suite functionality for CodexIndexDownloader tests
type CodexIndexDownloaderTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockClient *mock_communities.MockCodexClientInterface
	testDir    string
	cancelChan chan struct{}
	logger     *zap.Logger
}

// SetupTest runs before each test method
func (suite *CodexIndexDownloaderTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockClient = mock_communities.NewMockCodexClientInterface(suite.ctrl)

	// Create a temporary directory for test files
	var err error
	suite.testDir, err = os.MkdirTemp("", "codex-index-test-*")
	require.NoError(suite.T(), err)

	// Create a fresh cancel channel for each test
	suite.cancelChan = make(chan struct{})

	// Use NOP logger for unit tests (no output noise)
	suite.logger = zap.NewNop()
}

// TearDownTest runs after each test method
func (suite *CodexIndexDownloaderTestSuite) TearDownTest() {
	suite.ctrl.Finish()

	// Clean up cancel channel - check if it's still open before closing
	if suite.cancelChan != nil {
		select {
		case <-suite.cancelChan:
			// Already closed, do nothing
		default:
			// Still open, close it
			close(suite.cancelChan)
		}
	}

	// Clean up test directory
	if suite.testDir != "" {
		os.RemoveAll(suite.testDir)
	}
}

// TestCodexIndexDownloaderTestSuite runs the test suite
func TestCodexIndexDownloaderTestSuite(t *testing.T) {
	suite.Run(t, new(CodexIndexDownloaderTestSuite))
}

// ==================== GotManifest Tests ====================

func (suite *CodexIndexDownloaderTestSuite) TestGotManifest_SuccessClosesChannel() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "index.bin")

	// Setup mock to return a successful manifest
	expectedManifest := &communities.CodexManifest{
		CID: testCid,
	}
	expectedManifest.Manifest.DatasetSize = 1024
	expectedManifest.Manifest.TreeCid = "zDvZRwzmTreeCID"
	expectedManifest.Manifest.BlockSize = 65536

	suite.mockClient.EXPECT().
		FetchManifestWithContext(gomock.Any(), testCid).
		Return(expectedManifest, nil)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Call GotManifest
	manifestChan := downloader.GotManifest()

	// Wait for channel to close (with timeout)
	select {
	case <-manifestChan:
		// Success - channel closed as expected
		suite.T().Log("✅ GotManifest channel closed successfully")
	case <-time.After(1 * time.Second):
		suite.T().Fatal("Timeout waiting for GotManifest channel to close")
	}

	// Verify dataset size was recorded
	assert.Equal(suite.T(), int64(1024), downloader.GetDatasetSize(), "Dataset size should be recorded")

	// Verify no error was recorded
	assert.NoError(suite.T(), downloader.GetError(), "No error should be recorded on success")
}

func (suite *CodexIndexDownloaderTestSuite) TestGotManifest_ErrorDoesNotCloseChannel() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "index.bin")

	// Setup mock to return an error
	suite.mockClient.EXPECT().
		FetchManifestWithContext(gomock.Any(), testCid).
		Return(nil, errors.New("fetch error"))

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Call GotManifest
	manifestChan := downloader.GotManifest()

	// Channel should NOT close on error
	select {
	case <-manifestChan:
		suite.T().Fatal("GotManifest channel should NOT close on error")
	case <-time.After(200 * time.Millisecond):
		// Expected - channel did not close
		suite.T().Log("✅ GotManifest channel did not close on error (as expected)")
	}

	// Verify dataset size was NOT recorded (should be 0)
	assert.Equal(suite.T(), int64(0), downloader.GetDatasetSize(), "Dataset size should be 0 on error")

	// Verify download is not complete
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete on error")

	// Verify error was recorded
	assert.Error(suite.T(), downloader.GetError(), "Error should be recorded")
	assert.Contains(suite.T(), downloader.GetError().Error(), "fetch error", "Error message should contain fetch error")
	suite.T().Log("✅ Error was recorded correctly")
}

func (suite *CodexIndexDownloaderTestSuite) TestGotManifest_CidMismatchDoesNotCloseChannel() {
	testCid := "zDvZRwzmTestCID123"
	differentCid := "zDvZRwzmDifferentCID456"
	filePath := filepath.Join(suite.testDir, "index.bin")

	// Setup mock to return a manifest with different CID
	mismatchedManifest := &communities.CodexManifest{
		CID: differentCid, // Different CID!
	}
	mismatchedManifest.Manifest.DatasetSize = 1024

	suite.mockClient.EXPECT().
		FetchManifestWithContext(gomock.Any(), testCid).
		Return(mismatchedManifest, nil)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Call GotManifest
	manifestChan := downloader.GotManifest()

	// Channel should NOT close on CID mismatch
	select {
	case <-manifestChan:
		suite.T().Fatal("GotManifest channel should NOT close on CID mismatch")
	case <-time.After(200 * time.Millisecond):
		// Expected - channel did not close
		suite.T().Log("✅ GotManifest channel did not close on CID mismatch (as expected)")
	}

	// Verify dataset size was NOT recorded (should be 0)
	assert.Equal(suite.T(), int64(0), downloader.GetDatasetSize(), "Dataset size should be 0 on CID mismatch")

	// Verify download is not complete
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete on CID mismatch")

	// Verify error was recorded
	assert.Error(suite.T(), downloader.GetError(), "Error should be recorded for CID mismatch")
	assert.Contains(suite.T(), downloader.GetError().Error(), "CID mismatch", "Error message should mention CID mismatch")
	suite.T().Log("✅ Error was recorded for CID mismatch")
}

func (suite *CodexIndexDownloaderTestSuite) TestGotManifest_Cancellation() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "index.bin")

	// Setup mock with DoAndReturn to simulate slow response and check for cancellation
	fetchCalled := make(chan struct{})
	suite.mockClient.EXPECT().
		FetchManifestWithContext(gomock.Any(), testCid).
		DoAndReturn(func(ctx context.Context, cid string) (*communities.CodexManifest, error) {
			close(fetchCalled) // Signal that fetch was called

			// Wait for context cancellation
			<-ctx.Done()
			return nil, ctx.Err()
		})

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Call GotManifest
	manifestChan := downloader.GotManifest()

	// Wait for FetchManifestWithContext to be called
	select {
	case <-fetchCalled:
		suite.T().Log("FetchManifestWithContext was called")
	case <-time.After(1 * time.Second):
		suite.T().Fatal("Timeout waiting for FetchManifestWithContext to be called")
	}

	// Now trigger cancellation
	close(suite.cancelChan)

	// Channel should NOT close on cancellation
	select {
	case <-manifestChan:
		suite.T().Fatal("GotManifest channel should NOT close on cancellation")
	case <-time.After(200 * time.Millisecond):
		// Expected - channel did not close
		suite.T().Log("✅ GotManifest was cancelled and channel did not close (as expected)")
	}

	// Verify dataset size was NOT recorded
	assert.Equal(suite.T(), int64(0), downloader.GetDatasetSize(), "Dataset size should be 0 on cancellation")

	// Verify download is not complete
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete on cancellation")

	// Verify error was recorded (context cancellation)
	assert.Error(suite.T(), downloader.GetError(), "Error should be recorded for cancellation")
	assert.ErrorIs(suite.T(), downloader.GetError(), context.Canceled, "Error should be context.Canceled")
	suite.T().Log("✅ Cancellation error was recorded correctly")
}

func (suite *CodexIndexDownloaderTestSuite) TestGotManifest_RecordsDatasetSize() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "index.bin")
	expectedSize := int64(2048)

	// Setup mock to return a manifest with specific dataset size
	expectedManifest := &communities.CodexManifest{
		CID: testCid,
	}
	expectedManifest.Manifest.DatasetSize = expectedSize
	expectedManifest.Manifest.TreeCid = "zDvZRwzmTreeCID"

	suite.mockClient.EXPECT().
		FetchManifestWithContext(gomock.Any(), testCid).
		Return(expectedManifest, nil)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Initially, dataset size should be 0
	assert.Equal(suite.T(), int64(0), downloader.GetDatasetSize(), "Initial dataset size should be 0")

	// Call GotManifest
	manifestChan := downloader.GotManifest()

	// Wait for channel to close
	select {
	case <-manifestChan:
		suite.T().Log("GotManifest completed successfully")
	case <-time.After(1 * time.Second):
		suite.T().Fatal("Timeout waiting for GotManifest to complete")
	}

	// Verify dataset size was recorded correctly
	assert.Equal(suite.T(), expectedSize, downloader.GetDatasetSize(), "Dataset size should match manifest")
	suite.T().Logf("✅ Dataset size correctly recorded: %d", downloader.GetDatasetSize())

	// Verify no error was recorded
	assert.NoError(suite.T(), downloader.GetError(), "No error should be recorded on success")
}

// ==================== DownloadIndexFile Tests ====================

func (suite *CodexIndexDownloaderTestSuite) TestDownloadIndexFile_StoresFileCorrectly() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "downloaded-index.bin")
	testData := []byte("test index file content with some data")

	// Setup mock to write test data to the provided writer
	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, w io.Writer) error {
			_, err := w.Write(testData)
			return err
		})

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Start download
	downloader.DownloadIndexFile()

	// Wait for download to complete (check bytes completed first, then file existence)
	require.Eventually(suite.T(), func() bool {
		// First check: all bytes downloaded
		if downloader.BytesCompleted() != int64(len(testData)) {
			return false
		}
		// Second check: download marked as complete (file renamed)
		if !downloader.IsDownloadComplete() {
			return false
		}
		// Third check: file actually exists with correct size
		stat, err := os.Stat(filePath)
		if err != nil {
			return false
		}
		return stat.Size() == int64(len(testData))
	}, 2*time.Second, 50*time.Millisecond, "File should be fully downloaded and saved")

	// Verify file contents
	actualData, err := os.ReadFile(filePath)
	require.NoError(suite.T(), err, "Should be able to read downloaded file")
	assert.Equal(suite.T(), testData, actualData, "File contents should match")
	suite.T().Logf("✅ File downloaded successfully to: %s", filePath)

	// Verify download is complete
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")

	// Verify no error was recorded
	assert.NoError(suite.T(), downloader.GetError(), "No error should be recorded on successful download")
}

func (suite *CodexIndexDownloaderTestSuite) TestDownloadIndexFile_TracksProgress() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "progress-test.bin")
	testData := []byte("0123456789") // 10 bytes

	// Setup mock to write test data in chunks
	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, w io.Writer) error {
			// Write in 2-byte chunks to simulate streaming
			for i := 0; i < len(testData); i += 2 {
				end := i + 2
				if end > len(testData) {
					end = len(testData)
				}
				_, err := w.Write(testData[i:end])
				if err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond) // Small delay to allow progress tracking
			}
			return nil
		})

	// Create downloader and set dataset size
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Start download
	downloader.DownloadIndexFile()

	// Initially bytes completed should be 0
	assert.Equal(suite.T(), int64(0), downloader.BytesCompleted(), "Initial bytes completed should be 0")

	// Wait for some progress
	require.Eventually(suite.T(), func() bool {
		return downloader.BytesCompleted() > 0
	}, 1*time.Second, 20*time.Millisecond, "Progress should increase")

	suite.T().Logf("Progress observed: %d bytes", downloader.BytesCompleted())

	// Wait for download to complete
	require.Eventually(suite.T(), func() bool {
		return downloader.BytesCompleted() == int64(len(testData))
	}, 2*time.Second, 50*time.Millisecond, "Should download all bytes")

	assert.Equal(suite.T(), int64(len(testData)), downloader.BytesCompleted(), "All bytes should be downloaded")
	suite.T().Logf("✅ Download progress tracked correctly: %d/%d bytes", downloader.BytesCompleted(), len(testData))

	// Verify download is complete
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")

	// Verify no error was recorded
	assert.NoError(suite.T(), downloader.GetError(), "No error should be recorded on successful download")
}

func (suite *CodexIndexDownloaderTestSuite) TestDownloadIndexFile_Cancellation() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "cancel-test.bin")

	// Setup mock with DoAndReturn to simulate slow download and check for cancellation
	downloadStarted := make(chan struct{})
	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, w io.Writer) error {
			close(downloadStarted) // Signal that download started

			// Simulate slow download with cancellation check
			for range 100 {
				select {
				case <-ctx.Done():
					return ctx.Err() // Return cancellation error
				default:
					_, err := w.Write([]byte("x"))
					if err != nil {
						return err
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			return nil
		})

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Start download
	downloader.DownloadIndexFile()

	// Wait for download to start
	select {
	case <-downloadStarted:
		suite.T().Log("Download started")
	case <-time.After(1 * time.Second):
		suite.T().Fatal("Timeout waiting for download to start")
	}

	// Trigger cancellation
	close(suite.cancelChan)
	suite.T().Log("Cancellation triggered")

	// Wait a bit for cancellation to take effect
	time.Sleep(200 * time.Millisecond)

	// Verify that download was stopped (bytes completed should be small)
	bytesCompleted := downloader.BytesCompleted()
	suite.T().Logf("✅ Download cancelled after %d bytes (should be < 100)", bytesCompleted)
	assert.Less(suite.T(), bytesCompleted, int64(100), "Download should be cancelled before completing all 100 bytes")

	// Verify download is not complete
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete on cancellation")

	// Verify error was recorded (context cancellation)
	assert.Error(suite.T(), downloader.GetError(), "Error should be recorded for cancellation")
	assert.ErrorIs(suite.T(), downloader.GetError(), context.Canceled, "Error should be context.Canceled")
	suite.T().Log("✅ Cancellation error was recorded correctly")

	// Verify that the target file does NOT exist (atomic write should clean up temp file on cancellation)
	_, err := os.Stat(filePath)
	assert.True(suite.T(), os.IsNotExist(err), "Target file should not exist after cancellation")
	suite.T().Log("✅ Target file does not exist after cancellation (temp file cleaned up)")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "✅ IsDownloadComplete should be false after cancellation")
}

func (suite *CodexIndexDownloaderTestSuite) TestDownloadIndexFile_ErrorHandling() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "error-test.bin")

	// Setup mock to return an error during download
	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		Return(errors.New("download failed"))

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Start download
	downloader.DownloadIndexFile()

	// Wait a bit for the goroutine to run
	time.Sleep(200 * time.Millisecond)

	// Verify that no bytes were recorded on error
	assert.Equal(suite.T(), int64(0), downloader.BytesCompleted(), "No bytes should be recorded on error")
	suite.T().Log("✅ Error handling: no bytes recorded on download failure")

	// Verify download is not complete
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete on error")

	// Verify error was recorded
	assert.Error(suite.T(), downloader.GetError(), "Error should be recorded")
	assert.Contains(suite.T(), downloader.GetError().Error(), "download failed", "Error message should contain download failed")
	suite.T().Log("✅ Error was recorded correctly")

	// Verify that the target file does NOT exist (atomic write should clean up temp file)
	_, err := os.Stat(filePath)
	assert.True(suite.T(), os.IsNotExist(err), "Target file should not exist on download error")
	suite.T().Log("✅ Target file does not exist after download error (temp file cleaned up)")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "✅ IsDownloadComplete should be false after cancellation")
}

func (suite *CodexIndexDownloaderTestSuite) TestLength_ReturnsDatasetSize() {
	testCid := "zDvZRwzmTestCID123"
	filePath := filepath.Join(suite.testDir, "index.bin")
	expectedSize := int64(4096)

	// Setup mock to return a manifest
	expectedManifest := &communities.CodexManifest{
		CID: testCid,
	}
	expectedManifest.Manifest.DatasetSize = expectedSize

	suite.mockClient.EXPECT().
		FetchManifestWithContext(gomock.Any(), testCid).
		Return(expectedManifest, nil)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, testCid, filePath, suite.cancelChan, suite.logger)

	// Initially, Length should return 0
	assert.Equal(suite.T(), int64(0), downloader.Length(), "Initial length should be 0")

	// Fetch manifest
	manifestChan := downloader.GotManifest()
	<-manifestChan

	// Now Length should return the dataset size
	assert.Equal(suite.T(), expectedSize, downloader.Length(), "Length should return dataset size")
	suite.T().Logf("✅ Length() correctly returns dataset size: %d", downloader.Length())
}
