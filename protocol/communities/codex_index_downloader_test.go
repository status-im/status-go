package communities_test

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

	"github.com/status-im/status-go/protocol/communities"
	mock_communities "github.com/status-im/status-go/protocol/communities/mock/communities"
)

// ============================================================================
// Suite 1: Real CodexClient Integration Tests
// ============================================================================

// CodexIndexDownloaderRealClientSuite tests successful index downloads
// using a real CodexClient instance against a running Codex node.
type CodexIndexDownloaderRealClientSuite struct {
	suite.Suite
	client       communities.CodexClientInterface
	logger       *zap.Logger
	uploadedCIDs []string // Track uploaded CIDs for cleanup
}

func (suite *CodexIndexDownloaderRealClientSuite) UploadRandomDataToCodex(size int) (string, []byte) {
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

func (suite *CodexIndexDownloaderRealClientSuite) SetupSuite() {
	suite.logger, _ = zap.NewDevelopment()
}

func (suite *CodexIndexDownloaderRealClientSuite) SetupTest() {
	suite.client = NewCodexClientTest(suite.T())
	suite.uploadedCIDs = []string{}
}

func (suite *CodexIndexDownloaderRealClientSuite) TearDownTest() {
	// Clean up all uploaded CIDs
	for _, cid := range suite.uploadedCIDs {
		if err := suite.client.RemoveCid(cid); err != nil {
			suite.T().Logf("Warning: Failed to remove CID %s: %v", cid, err)
		}
	}
}

func TestCodexIndexDownloaderRealClientSuite(t *testing.T) {
	suite.Run(t, new(CodexIndexDownloaderRealClientSuite))
}

// TestDownloadIndexFileFromLocalNode_Success tests successful download from local node
func (suite *CodexIndexDownloaderRealClientSuite) TestDownloadIndexFileFromLocalNode_Success() {
	cid, payload := suite.UploadRandomDataToCodex(1024)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.client, suite.logger)

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
func (suite *CodexIndexDownloaderRealClientSuite) TestDownloadIndexFileFromNetwork_Success() {
	cid, payload := suite.UploadRandomDataToCodex(1024)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.client, suite.logger)

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
func (suite *CodexIndexDownloaderRealClientSuite) TestDownloadIndexFileFromLocalNode_LargeFile() {
	// Upload a larger file (1MB)
	cid, payload := suite.UploadRandomDataToCodex(1024 * 1024)

	// Create downloader
	downloader := communities.NewCodexIndexDownloader(suite.client, suite.logger)

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
// Suite 2: Mock CodexClient Tests (Errors and Cancellations)
// ============================================================================

// CodexIndexDownloaderMockClientSuite tests error handling and cancellations
// using a mocked CodexClient interface.
type CodexIndexDownloaderMockClientSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockClient *mock_communities.MockCodexClientInterface
	logger     *zap.Logger
}

func (suite *CodexIndexDownloaderMockClientSuite) SetupSuite() {
	suite.logger = zap.NewNop() // Use NOP logger for unit tests
}

func (suite *CodexIndexDownloaderMockClientSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockClient = mock_communities.NewMockCodexClientInterface(suite.ctrl)
}

func (suite *CodexIndexDownloaderMockClientSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestCodexIndexDownloaderMockClientSuite(t *testing.T) {
	suite.Run(t, new(CodexIndexDownloaderMockClientSuite))
}

// TestDownloadIndexFileFromLocalNode_ContextCancellation tests cancellation during local download
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_ContextCancellation() {
	// Arrange
	testCid := "zDvZRwzmTestCID123"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_DownloadError() {
	// Arrange
	testCid := "zDvZRwzmTestCID123"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)
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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_ContextCancellation() {
	// Arrange
	testCid := "zDvZRwzmTestCID456"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_NetworkError() {
	// Arrange
	testCid := "zDvZRwzmTestCID789"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)
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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_TimeoutError() {
	// Arrange
	testCid := "zDvZRwzmTestCIDTimeout"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_PartialWrite() {
	// Arrange
	testCid := "zDvZRwzmPartialWrite"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_EmptyOutput() {
	// Arrange
	testCid := "zDvZRwzmEmptyCID"
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_SuccessWithData() {
	// Arrange
	testCid := "zDvZRwzmSuccessCID"
	testData := []byte("test index file content")
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromLocalNode_SuccessWithData() {
	// Arrange
	testCid := "zDvZRwzmLocalSuccessCID"
	testData := []byte("local index file content")
	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
func (suite *CodexIndexDownloaderMockClientSuite) TestDownloadIndexFileFromNetwork_MultipleChunks() {
	// Arrange
	testCid := "zDvZRwzmChunkedCID"
	chunk1 := []byte("chunk1")
	chunk2 := []byte("chunk2")
	chunk3 := []byte("chunk3")
	expectedData := append(append(chunk1, chunk2...), chunk3...)

	downloader := communities.NewCodexIndexDownloader(suite.mockClient, suite.logger)

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
