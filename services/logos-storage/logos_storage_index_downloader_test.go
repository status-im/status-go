package logosstorage_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	logosstorage "github.com/status-im/status-go/services/logos-storage"
	mock_logosstorage "github.com/status-im/status-go/services/logos-storage/mock"
)

// ============================================================================
// Suite 1: Real LogosStorageClient Integration Tests
// ============================================================================

// LogosStorageIndexDownloaderRealClientSuite tests successful index downloads
// using a real LogosStorageClient instance against a running LogosStorage node.
type LogosStorageIndexDownloaderRealClientSuite struct {
	suite.Suite
	client       logosstorage.LogosStorageClientInterface
	logger       *zap.Logger
	uploadedCIDs []string // Track uploaded CIDs for cleanup
}

func (suite *LogosStorageIndexDownloaderRealClientSuite) UploadRandomDataToLogosStorage(size int) (string, []byte) {
	// Generate random payload to ensure proper round-trip verification
	payload := make([]byte, size)
	_, err := rand.Read(payload)
	require.NoError(suite.T(), err, "failed to generate random payload")
	suite.T().Logf("Generated payload (first 32 bytes hex): %s", hex.EncodeToString(payload[:32]))

	cid, err := suite.client.Upload(bytes.NewReader(payload), "payload.bin")
	require.NoError(suite.T(), err, "upload failed")

	suite.uploadedCIDs = append(suite.uploadedCIDs, cid)
	return cid, payload
}

func (suite *LogosStorageIndexDownloaderRealClientSuite) SetupSuite() {
	suite.logger, _ = zap.NewDevelopment()
}

func (suite *LogosStorageIndexDownloaderRealClientSuite) SetupTest() {
	suite.client = NewLogosStorageClientTest(suite.T())
	suite.uploadedCIDs = []string{}
}

func (suite *LogosStorageIndexDownloaderRealClientSuite) TearDownTest() {
	// Clean up all uploaded CIDs
	for _, cid := range suite.uploadedCIDs {
		if err := suite.client.RemoveCid(cid); err != nil {
			suite.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		}
	}
}

func TestLogosStorageIndexDownloaderRealClientSuite(t *testing.T) {
	suite.Run(t, new(LogosStorageIndexDownloaderRealClientSuite))
}

// TestDownloadIndexFileFromLocalNode_Success tests successful download from local node
func (suite *LogosStorageIndexDownloaderRealClientSuite) TestDownloadIndexFileFromLocalNode_Success() {
	cid, payload := suite.UploadRandomDataToLogosStorage(1024)

	// Create downloader
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.client, suite.logger)

	// Act: Download the index file from local node
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromLocalNode(ctx, cid, &output)

	require.NoError(suite.T(), err, "DownloadIndexFileFromLocalNode should succeed")
	assert.Equal(suite.T(), payload, output.Bytes(), "Downloaded data should match uploaded data")
	suite.T().Logf("✅ Successfully downloaded %d bytes from local node", output.Len())
}

// TestDownloadIndexFileFromNetwork_Success tests successful download from network
func (suite *LogosStorageIndexDownloaderRealClientSuite) TestDownloadIndexFileFromNetwork_Success() {
	cid, payload := suite.UploadRandomDataToLogosStorage(1024)

	// Create downloader
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.client, suite.logger)

	// Act: Download the index file from network
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromNetwork(ctx, cid, &output)

	// Assert
	require.NoError(suite.T(), err, "DownloadIndexFileFromNetwork should succeed")
	assert.Equal(suite.T(), payload, output.Bytes(), "Downloaded data should match uploaded data")
	suite.T().Logf("✅ Successfully downloaded %d bytes from network", output.Len())
}

// TestDownloadIndexFileFromLocalNode_LargeFile tests downloading a larger file
func (suite *LogosStorageIndexDownloaderRealClientSuite) TestDownloadIndexFileFromLocalNode_LargeFile() {
	// Upload a larger file (1MB)
	cid, payload := suite.UploadRandomDataToLogosStorage(1024 * 1024)

	// Create downloader
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.client, suite.logger)

	// Act: Download the large file
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromLocalNode(ctx, cid, &output)

	// Assert
	require.NoError(suite.T(), err, "DownloadIndexFileFromLocalNode should succeed for large file")
	assert.Equal(suite.T(), len(payload), output.Len(), "Downloaded size should match uploaded size")
	assert.Equal(suite.T(), payload, output.Bytes(), "Downloaded data should match uploaded data")
	suite.T().Logf("✅ Successfully downloaded large file: %d bytes", output.Len())
}

// ============================================================================
// Suite 2: Mock LogosStorageClient Tests (Errors and Cancellations)
// ============================================================================

// LogosStorageIndexDownloaderMockClientSuite tests error handling and cancellations
// using a mocked LogosStorageClient interface.
type LogosStorageIndexDownloaderMockClientSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockClient *mock_logosstorage.MockLogosStorageClientInterface
	logger     *zap.Logger
}

func (suite *LogosStorageIndexDownloaderMockClientSuite) SetupSuite() {
	suite.logger = zap.NewNop() // Use NOP logger for unit tests
}

func (suite *LogosStorageIndexDownloaderMockClientSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockClient = mock_logosstorage.NewMockLogosStorageClientInterface(suite.ctrl)
}

func (suite *LogosStorageIndexDownloaderMockClientSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestLogosStorageIndexDownloaderMockClientSuite(t *testing.T) {
	suite.Run(t, new(LogosStorageIndexDownloaderMockClientSuite))
}

// TestDownloadIndexFileFromLocalNode_ContextCancellation tests cancellation during local download
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_ContextCancellation() {
	// Arrange
	testCid := "zDvZRwzmTestCID123"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	// Setup mock to simulate slow download that respects context cancellation
	downloadStarted := make(chan struct{})
	suite.mockClient.EXPECT().
		LocalDownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			close(downloadStarted)
			// Wait for context cancellation
			<-ctx.Done()
			return ctx.Err()
		})

	// Act: Start download and cancel context
	ctx, cancel := context.WithCancel(context.Background())

	var output bytes.Buffer
	errChan := make(chan error, 1)
	go func() {
		errChan <- downloader.DownloadIndexFileFromLocalNode(ctx, testCid, &output)
	}()

	// Wait for download to start
	<-downloadStarted

	// Cancel the context
	cancel()

	// Wait for download to complete
	err := <-errChan

	// Assert
	require.Error(suite.T(), err, "Should return error on cancellation")
	assert.ErrorIs(suite.T(), err, context.Canceled, "Error should be context.Canceled")
	suite.T().Log("✅ Correctly handled context cancellation")
}

// TestDownloadIndexFileFromLocalNode_DownloadError tests error during local download
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_DownloadError() {
	// Arrange
	testCid := "zDvZRwzmTestCID123"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)
	expectedError := errors.New("local download failed: network error")

	suite.mockClient.EXPECT().
		LocalDownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		Return(expectedError)

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromLocalNode(ctx, testCid, &output)

	// Assert
	require.Error(suite.T(), err, "Should return error on download failure")
	assert.Equal(suite.T(), expectedError, err, "Error should match expected error")
	assert.Zero(suite.T(), output.Len(), "Output should be empty on error")
	suite.T().Log("✅ Correctly propagated download error")
}

// TestDownloadIndexFileFromNetwork_ContextCancellation tests cancellation during network download
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_ContextCancellation() {
	// Arrange
	testCid := "zDvZRwzmTestCID456"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	downloadStarted := make(chan struct{})
	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			close(downloadStarted)
			<-ctx.Done()
			return ctx.Err()
		})

	// Act
	ctx, cancel := context.WithCancel(context.Background())

	var output bytes.Buffer
	errChan := make(chan error, 1)
	go func() {
		errChan <- downloader.DownloadIndexFileFromNetwork(ctx, testCid, &output)
	}()

	<-downloadStarted
	cancel()
	err := <-errChan

	// Assert
	require.Error(suite.T(), err, "Should return error on cancellation")
	assert.ErrorIs(suite.T(), err, context.Canceled, "Error should be context.Canceled")
	suite.T().Log("✅ Correctly handled network download cancellation")
}

// TestDownloadIndexFileFromNetwork_NetworkError tests network error handling
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_NetworkError() {
	// Arrange
	testCid := "zDvZRwzmTestCID789"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)
	expectedError := errors.New("network download failed: connection timeout")

	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		Return(expectedError)

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromNetwork(ctx, testCid, &output)

	// Assert
	require.Error(suite.T(), err, "Should return error on network failure")
	assert.Equal(suite.T(), expectedError, err, "Error should match expected error")
	assert.Zero(suite.T(), output.Len(), "Output should be empty on error")
	suite.T().Log("✅ Correctly propagated network error")
}

// TestDownloadIndexFileFromNetwork_TimeoutError tests timeout handling
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_TimeoutError() {
	// Arrange
	testCid := "zDvZRwzmTestCIDTimeout"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			// Simulate slow download that exceeds timeout
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				return nil
			}
		})

	// Act: Use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromNetwork(ctx, testCid, &output)

	// Assert
	require.Error(suite.T(), err, "Should return error on timeout")
	assert.ErrorIs(suite.T(), err, context.DeadlineExceeded, "Error should be context.DeadlineExceeded")
	suite.T().Log("✅ Correctly handled timeout")
}

// TestDownloadIndexFileFromNetwork_PartialWrite tests handling of partial write errors
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_PartialWrite() {
	// Arrange
	testCid := "zDvZRwzmPartialWrite"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			// Write some data first
			_, err := output.Write([]byte("partial data"))
			if err != nil {
				return err
			}
			// Then return an error
			return errors.New("write interrupted")
		})

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromNetwork(ctx, testCid, &output)

	// Assert
	require.Error(suite.T(), err, "Should return error on partial write")
	assert.Contains(suite.T(), err.Error(), "write interrupted", "Error should indicate write interruption")
	// Verify partial data was written
	assert.Equal(suite.T(), "partial data", output.String(), "Partial data should be in output buffer")
	suite.T().Log("✅ Correctly handled partial write error")
}

// TestDownloadIndexFileFromLocalNode_EmptyOutput tests downloading empty content
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_EmptyOutput() {
	// Arrange
	testCid := "zDvZRwzmEmptyCID"
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	// Mock returns success but writes nothing (empty file)
	suite.mockClient.EXPECT().
		LocalDownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			// Write nothing, just return success
			return nil
		})

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromLocalNode(ctx, testCid, &output)

	// Assert
	require.NoError(suite.T(), err, "Should succeed even with empty output")
	assert.Zero(suite.T(), output.Len(), "Output should be empty")
	suite.T().Log("✅ Correctly handled empty download")
}

// TestDownloadIndexFileFromNetwork_SuccessWithData tests successful network download with data
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_SuccessWithData() {
	// Arrange
	testCid := "zDvZRwzmSuccessCID"
	testData := []byte("test index file content")
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			_, err := output.Write(testData)
			return err
		})

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromNetwork(ctx, testCid, &output)

	// Assert
	require.NoError(suite.T(), err, "Should succeed")
	assert.Equal(suite.T(), testData, output.Bytes(), "Output should match test data")
	suite.T().Log("✅ Successfully downloaded data from network")
}

// TestDownloadIndexFileFromLocalNode_SuccessWithData tests successful local download with data
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_SuccessWithData() {
	// Arrange
	testCid := "zDvZRwzmLocalSuccessCID"
	testData := []byte("local index file content")
	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	suite.mockClient.EXPECT().
		LocalDownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			_, err := output.Write(testData)
			return err
		})

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromLocalNode(ctx, testCid, &output)

	// Assert
	require.NoError(suite.T(), err, "Should succeed")
	assert.Equal(suite.T(), testData, output.Bytes(), "Output should match test data")
	suite.T().Log("✅ Successfully downloaded data from local node")
}

// TestDownloadIndexFileFromNetwork_MultipleChunks tests downloading data written in chunks
func (suite *LogosStorageIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_MultipleChunks() {
	// Arrange
	testCid := "zDvZRwzmChunkedCID"
	chunk1 := []byte("chunk1")
	chunk2 := []byte("chunk2")
	chunk3 := []byte("chunk3")
	expectedData := append(append(chunk1, chunk2...), chunk3...)

	downloader := logosstorage.NewLogosStorageIndexDownloader(suite.mockClient, suite.logger)

	suite.mockClient.EXPECT().
		DownloadWithContext(gomock.Any(), testCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			// Write in multiple chunks
			if _, err := output.Write(chunk1); err != nil {
				return err
			}
			if _, err := output.Write(chunk2); err != nil {
				return err
			}
			if _, err := output.Write(chunk3); err != nil {
				return err
			}
			return nil
		})

	// Act
	ctx := context.Background()
	var output bytes.Buffer
	err := downloader.DownloadIndexFileFromNetwork(ctx, testCid, &output)

	// Assert
	require.NoError(suite.T(), err, "Should succeed with chunked writes")
	assert.Equal(suite.T(), expectedData, output.Bytes(), "Output should contain all chunks")
	suite.T().Log("✅ Successfully downloaded chunked data")
}
