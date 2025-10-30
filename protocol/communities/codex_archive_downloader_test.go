package communities_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/codex-storage/codex-go-bindings/codex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"

	"github.com/status-im/status-go/protocol/communities"

	mock_communities "github.com/status-im/status-go/protocol/communities/mock/communities"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// CodexArchiveDownloaderSuite demonstrates testify's suite functionality
type CodexArchiveDownloaderSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockClient *mock_communities.MockCodexClientInterface
	index      *protobuf.CodexWakuMessageArchiveIndex
}

// SetupTest runs before each test method
func (suite *CodexArchiveDownloaderSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockClient = mock_communities.NewMockCodexClientInterface(suite.ctrl)
	suite.index = &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"test-archive-hash-1": {
				Cid: "test-cid-1",
				Metadata: &protobuf.WakuMessageArchiveMetadata{
					From: 1000,
					To:   2000,
				},
			},
		},
	}
}

// TearDownTest runs after each test method
func (suite *CodexArchiveDownloaderSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// Run the test suite
func TestCodexArchiveDownloaderSuite(t *testing.T) {
	suite.Run(t, new(CodexArchiveDownloaderSuite))
}

func (suite *CodexArchiveDownloaderSuite) TestBasicSingleArchive() {
	// Test data
	communityID := "test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})

	// Set up mock expectations - same as before
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "test-cid-1").
		Return(codex.Manifest{Cid: "test-cid-1"}, nil).
		Times(1)

	// First HasCid call returns false, second returns true (simulating polling)
	gomock.InOrder(
		suite.mockClient.EXPECT().HasCid("test-cid-1").Return(false, nil),
		suite.mockClient.EXPECT().HasCid("test-cid-1").Return(true, nil),
	)

	// Create downloader with mock client
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, suite.index, communityID, existingArchiveIDs, cancelChan, logger)

	// Set fast polling interval for tests (10ms instead of default 1s)
	downloader.SetPollingInterval(10 * time.Millisecond)

	// Set up callback to track completion
	var callbackInvoked bool
	var callbackHash string
	var callbackFrom, callbackTo uint64

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		callbackInvoked = true
		callbackHash = hash
		callbackFrom = from
		callbackTo = to
	})

	// Verify initial state - compare testify vs standard assertions
	// Testify version is more readable and provides better error messages
	assert.Equal(suite.T(), 1, downloader.GetTotalArchivesCount(), "Total archives count should be 1")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "Downloaded archives should be 0 initially")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete initially")

	// Start the download
	downloader.StartDownload()

	// Wait for download to complete (with timeout)
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 5*time.Second, 100*time.Millisecond, "Download should complete within 5 seconds")

	// Verify final state - testify makes these assertions more expressive
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")
	assert.Equal(suite.T(), 1, downloader.GetTotalDownloadedArchivesCount(), "Should have 1 downloaded archive")

	// Verify callback was invoked - multiple related assertions grouped logically
	assert.True(suite.T(), callbackInvoked, "Callback should be invoked")
	assert.Equal(suite.T(), "test-archive-hash-1", callbackHash, "Callback hash should match expected")
	assert.Equal(suite.T(), uint64(1000), callbackFrom, "Callback from should be 1000")
	assert.Equal(suite.T(), uint64(2000), callbackTo, "Callback to should be 2000")

	suite.T().Log("✅ Basic single archive download test passed")
	suite.T().Log("   - All mock expectations satisfied")
	suite.T().Logf("   - Callback invoked: %v", callbackInvoked)
}

func (suite *CodexArchiveDownloaderSuite) TestMultipleArchives() {
	// Create test data with multiple archives
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Cid:      "cid-1",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
			},
			"archive-2": {
				Cid:      "cid-2",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
			},
			"archive-3": {
				Cid:      "cid-3",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 3000, To: 4000},
			},
		},
	}

	communityID := "test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})

	// Set up expectations for all 3 archives - testify makes verification cleaner
	expectedCids := []string{"cid-1", "cid-2", "cid-3"}

	for _, cid := range expectedCids {
		suite.mockClient.EXPECT().
			TriggerDownloadWithContext(gomock.Any(), cid).
			Return(codex.Manifest{Cid: cid}, nil).
			Times(1)

		// Each archive becomes available after one poll
		gomock.InOrder(
			suite.mockClient.EXPECT().HasCid(cid).Return(false, nil),
			suite.mockClient.EXPECT().HasCid(cid).Return(true, nil),
		)
	}

	// Create downloader
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, existingArchiveIDs, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)

	// Track the order in which archives are started (deterministic)
	var startOrder []string
	var startOrderMu sync.Mutex
	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		startOrderMu.Lock()
		startOrder = append(startOrder, hash)
		startOrderMu.Unlock()
	})

	// Track completed archives (non-deterministic due to concurrency)
	completedArchives := make(map[string]bool)
	var completedArchivesMu sync.Mutex
	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		completedArchivesMu.Lock()
		completedArchives[hash] = true
		completedArchivesMu.Unlock()
	})

	// Initial state verification
	assert.Equal(suite.T(), 3, downloader.GetTotalArchivesCount(), "Should have 3 total archives")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "Should start with 0 downloaded")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Should not be complete initially")

	// Start download
	downloader.StartDownload()

	// Wait for all downloads to complete
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 10*time.Second, 100*time.Millisecond, "All downloads should complete within 10 seconds")

	// Final state verification - testify makes these checks very readable
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")
	assert.Equal(suite.T(), 3, downloader.GetTotalDownloadedArchivesCount(), "Should have downloaded all 3 archives")

	// Verify all archives were processed (with proper synchronization)
	completedArchivesMu.Lock()
	completedLen := len(completedArchives)
	hasArchive1 := completedArchives["archive-1"]
	hasArchive2 := completedArchives["archive-2"]
	hasArchive3 := completedArchives["archive-3"]
	completedArchivesMu.Unlock()

	assert.Equal(suite.T(), 3, completedLen, "Should have completed exactly 3 archives")
	assert.True(suite.T(), hasArchive1, "Should have completed archive-1")
	assert.True(suite.T(), hasArchive2, "Should have completed archive-2")
	assert.True(suite.T(), hasArchive3, "Should have completed archive-3")

	// Verify sorting: archives should be started in most-recent-first order (deterministic)
	// This tests the internal sorting logic before concurrency begins
	startOrderMu.Lock()
	startOrderCopy := make([]string, len(startOrder))
	copy(startOrderCopy, startOrder)
	startOrderMu.Unlock()

	expectedStartOrder := []string{"archive-3", "archive-2", "archive-1"}
	assert.Equal(suite.T(), expectedStartOrder, startOrderCopy, "Archives should be started in most-recent-first order")

	suite.T().Log("✅ Multiple archives test passed")
	suite.T().Logf("   - Completed %d out of %d archives", len(completedArchives), 3)
	suite.T().Logf("   - Start order (sorted): %v", startOrder)
}

func (suite *CodexArchiveDownloaderSuite) TestErrorDuringTriggerDownload() {
	// Test that errors during TriggerDownloadWithContext are handled properly
	communityID := "test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})

	// Mock TriggerDownloadWithContext to simulate an error
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "test-cid-1").
		Return(codex.Manifest{}, assert.AnError). // Return a generic error to simulate failure
		Times(1)

	// No HasCid calls should be made since TriggerDownload fails
	// (this is the key test - we shouldn't proceed to polling)

	// Create downloader with mock client
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, suite.index, communityID, existingArchiveIDs, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)

	// Track callbacks - onArchiveDownloaded should NOT be called on failure
	var callbackInvoked bool
	var startCallbackInvoked bool

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		callbackInvoked = true
	})

	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		startCallbackInvoked = true
	})

	// Start the download
	downloader.StartDownload()

	// Wait a bit to ensure the goroutine has time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify the state - download should be marked complete (no pending downloads)
	// but no archives should be successfully downloaded
	assert.True(suite.T(), startCallbackInvoked, "Start callback should be invoked")
	assert.False(suite.T(), callbackInvoked, "Success callback should NOT be invoked on failure")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "No archives should be downloaded on failure")
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete (no pending downloads)")
	assert.Equal(suite.T(), 0, downloader.GetPendingArchivesCount(), "No archives should be pending")

	suite.T().Log("✅ Error during trigger download test passed")
	suite.T().Log("   - TriggerDownload failed as expected")
	suite.T().Log("   - No polling occurred (as intended)")
	suite.T().Log("   - Success callback was NOT invoked")
}

func (suite *CodexArchiveDownloaderSuite) TestActualCancellationDuringTriggerDownload() {
	// Test real cancellation during TriggerDownloadWithContext using DoAndReturn
	communityID := "test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})

	// Use DoAndReturn to create a realistic TriggerDownload that waits for cancellation
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "test-cid-1").
		DoAndReturn(func(ctx context.Context, cid string) (codex.Manifest, error) {
			// Simulate work by waiting for context cancellation
			select {
			case <-time.After(5 * time.Second): // This should never happen in our test
				return codex.Manifest{Cid: cid}, nil
			case <-ctx.Done(): // Wait for actual context cancellation
				return codex.Manifest{}, ctx.Err() // Return the actual cancellation error
			}
		}).
		Times(1)

	// Create downloader with mock client
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, suite.index, communityID, existingArchiveIDs, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)
	downloader.SetPollingTimeout(200 * time.Millisecond) // Short timeout - we should never reach polling

	// Track callbacks
	var callbackInvoked bool
	var startCallbackInvoked bool

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		callbackInvoked = true
	})

	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		startCallbackInvoked = true
	})

	// Start the download
	downloader.StartDownload()

	// Wait a bit for the download to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the entire operation (this should cancel the trigger download context)
	close(cancelChan)

	// Wait for the operation to complete
	time.Sleep(200 * time.Millisecond)

	// Verify the state
	assert.True(suite.T(), startCallbackInvoked, "Start callback should be invoked")
	assert.False(suite.T(), callbackInvoked, "Success callback should NOT be invoked on cancellation")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "No archives should be downloaded on cancellation")
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Download should be complete after cancellation")
	assert.True(suite.T(), downloader.IsCancelled(), "✅ Download should be marked as cancelled")
	assert.Equal(suite.T(), 0, downloader.GetPendingArchivesCount(), "No archives should be pending")

	suite.T().Log("✅ Actual cancellation during trigger download test passed")
	suite.T().Log("   - TriggerDownload was actually cancelled")
	suite.T().Log("   - No polling occurred (as intended)")
	suite.T().Log("   - Success callback was NOT invoked")
}

func (suite *CodexArchiveDownloaderSuite) TestCancellationDuringPolling() {
	// Test that cancellation during the polling phase is handled properly
	communityID := "test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})

	// Mock successful TriggerDownload
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "test-cid-1").
		Return(codex.Manifest{Cid: "test-cid-1"}, nil).
		Times(1)

	// Mock polling - allow multiple calls, but we'll cancel before completion
	suite.mockClient.EXPECT().
		HasCid("test-cid-1").
		Return(false, nil).
		AnyTimes() // Allow multiple calls since timing is unpredictable

	// Create downloader with mock client
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, suite.index, communityID, existingArchiveIDs, cancelChan, logger)
	downloader.SetPollingInterval(50 * time.Millisecond) // Longer interval to allow cancellation
	downloader.SetPollingTimeout(1 * time.Second)        // Short timeout for test (instead of 30s)

	// Track callbacks
	var successCallbackInvoked bool
	var startCallbackInvoked bool

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		successCallbackInvoked = true
	})

	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		startCallbackInvoked = true
	})

	// Start the download
	downloader.StartDownload()

	// Wait for the download to start and first poll to occur
	time.Sleep(100 * time.Millisecond)

	// Verify initial state
	assert.True(suite.T(), startCallbackInvoked, "Start callback should be invoked")
	assert.Equal(suite.T(), 1, downloader.GetPendingArchivesCount(), "Should have 1 pending download")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Download should not be complete yet")

	// Cancel the entire operation
	close(cancelChan)

	// Wait for cancellation to propagate
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 2*time.Second, 50*time.Millisecond, "Download should complete after cancellation")

	// Verify final state
	assert.False(suite.T(), successCallbackInvoked, "Success callback should NOT be invoked on cancellation")
	assert.Equal(suite.T(), 0, downloader.GetPendingArchivesCount(), "No archives should be pending after cancellation")
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Download should be complete after cancellation")
	assert.True(suite.T(), downloader.IsCancelled(), "Downloader should be marked as cancelled")

	suite.T().Log("✅ Cancellation during polling test passed")
	suite.T().Log("   - TriggerDownload succeeded")
	suite.T().Log("   - Polling started but was cancelled")
	suite.T().Log("   - Success callback was NOT invoked")
	suite.T().Log("   - Download marked as cancelled")
}

func (suite *CodexArchiveDownloaderSuite) TestPollingTimeout() {
	// Test that polling timeout is handled properly (no success callback)
	communityID := "test-community"
	existingArchiveIDs := []string{} // No existing archives
	cancelChan := make(chan struct{})

	// Mock successful TriggerDownload
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "test-cid-1").
		Return(codex.Manifest{Cid: "test-cid-1"}, nil).
		Times(1)

	// Mock polling to always return false (simulating timeout)
	suite.mockClient.EXPECT().
		HasCid("test-cid-1").
		Return(false, nil).
		AnyTimes() // Will be called multiple times until timeout

	// Create downloader with mock client
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, suite.index, communityID, existingArchiveIDs, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond) // Fast polling for test
	downloader.SetPollingTimeout(100 * time.Millisecond) // Short timeout for test (instead of 30s)

	// Track callbacks
	var successCallbackInvoked bool
	var startCallbackInvoked bool

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		successCallbackInvoked = true
	})

	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		startCallbackInvoked = true
	})

	// Start the download
	downloader.StartDownload()

	// Wait for timeout (100ms configured timeout)
	// We'll wait a bit longer to ensure timeout occurs
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 500*time.Millisecond, 50*time.Millisecond, "Download should complete after timeout")

	// Verify state after timeout
	assert.True(suite.T(), startCallbackInvoked, "Start callback should be invoked")
	assert.False(suite.T(), successCallbackInvoked, "Success callback should NOT be invoked on timeout")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "No archives should be downloaded on timeout")
	assert.Equal(suite.T(), 0, downloader.GetPendingArchivesCount(), "No archives should be pending after timeout")
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")

	suite.T().Log("✅ Polling timeout test passed")
	suite.T().Log("   - TriggerDownload succeeded")
	suite.T().Log("   - Polling timed out after 100ms (fast test)")
	suite.T().Log("   - Success callback was NOT invoked")
}

func (suite *CodexArchiveDownloaderSuite) TestWithExistingArchives() {
	// Test with some archives already downloaded (existing archive IDs)
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Cid:      "cid-1",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
			},
			"archive-2": {
				Cid:      "cid-2",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
			},
			"archive-3": {
				Cid:      "cid-3",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 3000, To: 4000},
			},
		},
	}

	communityID := "test-community"
	// Simulate that we already have archive-1 and archive-3
	existingArchiveIDs := []string{"archive-1", "archive-3"}
	cancelChan := make(chan struct{})

	// Only archive-2 should be downloaded (not in existingArchiveIDs)
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-2").
		Return(codex.Manifest{Cid: "cid-2"}, nil).
		Times(1) // Only one call expected

	// Only archive-2 should be polled
	gomock.InOrder(
		suite.mockClient.EXPECT().HasCid("cid-2").Return(false, nil),
		suite.mockClient.EXPECT().HasCid("cid-2").Return(true, nil),
	)

	// Create downloader with existing archives
	logger := zap.NewNop() // No-op logger for tests
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, existingArchiveIDs, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)

	// Track which archives are started and completed
	var startedArchives []string
	var completedArchives []string

	downloader.SetOnStartingArchiveDownload(func(hash string, from, to uint64) {
		startedArchives = append(startedArchives, hash)
	})

	downloader.SetOnArchiveDownloaded(func(hash string, from, to uint64) {
		completedArchives = append(completedArchives, hash)
	})

	// Verify initial state - should start with 2 existing archives counted
	assert.Equal(suite.T(), 3, downloader.GetTotalArchivesCount(), "Should have 3 total archives")
	assert.Equal(suite.T(), 2, downloader.GetTotalDownloadedArchivesCount(), "Should start with 2 existing archives")
	assert.False(suite.T(), downloader.IsDownloadComplete(), "Should not be complete initially")

	// Start download
	downloader.StartDownload()

	// Wait for download to complete
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 5*time.Second, 100*time.Millisecond, "Download should complete within 5 seconds")

	// Verify final state
	assert.True(suite.T(), downloader.IsDownloadComplete(), "Download should be complete")
	assert.Equal(suite.T(), 3, downloader.GetTotalDownloadedArchivesCount(), "Should have 3 total downloaded (2 existing + 1 new)")

	// Verify only missing archive was processed
	assert.Len(suite.T(), startedArchives, 1, "Should have started exactly 1 archive download")
	assert.Contains(suite.T(), startedArchives, "archive-2", "Should have started archive-2")
	assert.NotContains(suite.T(), startedArchives, "archive-1", "Should NOT have started archive-1 (existing)")
	assert.NotContains(suite.T(), startedArchives, "archive-3", "Should NOT have started archive-3 (existing)")

	assert.Len(suite.T(), completedArchives, 1, "Should have completed exactly 1 archive download")
	assert.Contains(suite.T(), completedArchives, "archive-2", "Should have completed archive-2")

	suite.T().Log("✅ Existing archives test passed")
	suite.T().Logf("   - Started with %d existing archives", len(existingArchiveIDs))
	suite.T().Logf("   - Downloaded %d missing archives", len(completedArchives))
	suite.T().Logf("   - Final count: %d total", downloader.GetTotalDownloadedArchivesCount())
}

// Test case: One success, one error
func (suite *CodexArchiveDownloaderSuite) TestPartialSuccess_OneSuccessOneError() {
	communityID := "test-community"
	cancelChan := make(chan struct{})
	defer close(cancelChan)

	// 2 archives: archive-2 (newer) succeeds, archive-1 (older) fails
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
				Cid:      "cid-1",
			},
			"archive-2": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
				Cid:      "cid-2",
			},
		},
	}

	// Archive-2 succeeds
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-2").
		Return(codex.Manifest{Cid: "cid-2"}, nil)
	suite.mockClient.EXPECT().
		HasCid("cid-2").
		Return(true, nil)

	// Archive-1 fails
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-1").
		Return(codex.Manifest{}, fmt.Errorf("trigger failed"))

	logger := zap.NewNop()
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, []string{}, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)
	downloader.SetPollingTimeout(1 * time.Second)

	downloader.StartDownload()

	// Wait for completion
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 3*time.Second, 50*time.Millisecond)

	// Assertions
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Should be complete")
	assert.False(suite.T(), downloader.IsCancelled(), "✅ Should NOT be cancelled")
	assert.Equal(suite.T(), 1, downloader.GetTotalDownloadedArchivesCount(), "✅ Should have 1 successful download")

	suite.T().Log("✅ Partial success test passed (1 success, 1 error)")
}

// Test case: One success, one error, one cancellation
func (suite *CodexArchiveDownloaderSuite) TestPartialSuccess_SuccessErrorCancellation() {
	communityID := "test-community"
	cancelChan := make(chan struct{})

	// 3 archives
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
				Cid:      "cid-1",
			},
			"archive-2": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
				Cid:      "cid-2",
			},
			"archive-3": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 3000, To: 4000},
				Cid:      "cid-3",
			},
		},
	}

	// Archive-3 (newest) succeeds
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-3").
		Return(codex.Manifest{Cid: "cid-3"}, nil)
	suite.mockClient.EXPECT().
		HasCid("cid-3").
		Return(true, nil)

	// Archive-2 fails
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-2").
		Return(codex.Manifest{}, fmt.Errorf("trigger failed"))

	// Archive-1 will be cancelled (no expectations needed)
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-1").
		DoAndReturn(func(ctx context.Context, cid string) (*codex.Manifest, error) {
			<-ctx.Done() // Wait for cancellation
			return nil, ctx.Err()
		}).
		AnyTimes()

	logger := zap.NewNop()
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, []string{}, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)
	downloader.SetPollingTimeout(1 * time.Second)

	downloader.StartDownload()

	// Wait a bit for first two to process
	time.Sleep(200 * time.Millisecond)

	// Now cancel
	close(cancelChan)

	// Wait for completion
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 3*time.Second, 50*time.Millisecond)

	// Assertions
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Should be complete")
	assert.True(suite.T(), downloader.IsCancelled(), "✅ Should be cancelled")
	assert.Equal(suite.T(), 1, downloader.GetTotalDownloadedArchivesCount(), "✅ Should have 1 successful download")

	suite.T().Log("✅ Partial success test passed (1 success, 1 error, 1 cancellation)")
}

// Test case: One success, then cancellation
func (suite *CodexArchiveDownloaderSuite) TestPartialSuccess_SuccessThenCancellation() {
	communityID := "test-community"
	cancelChan := make(chan struct{})

	// 2 archives
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
				Cid:      "cid-1",
			},
			"archive-2": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
				Cid:      "cid-2",
			},
		},
	}

	// Archive-2 (newer) succeeds
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-2").
		Return(codex.Manifest{Cid: "cid-2"}, nil)
	suite.mockClient.EXPECT().
		HasCid("cid-2").
		Return(true, nil)

	// Archive-1 will be cancelled
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-1").
		DoAndReturn(func(ctx context.Context, cid string) (*codex.Manifest, error) {
			<-ctx.Done() // Wait for cancellation
			return nil, ctx.Err()
		}).
		AnyTimes()

	logger := zap.NewNop()
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, []string{}, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)
	downloader.SetPollingTimeout(1 * time.Second)

	downloader.StartDownload()

	// Wait for first archive to complete
	time.Sleep(200 * time.Millisecond)

	// Now cancel
	close(cancelChan)

	// Wait for completion
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 3*time.Second, 50*time.Millisecond)

	// Assertions
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Should be complete")
	assert.True(suite.T(), downloader.IsCancelled(), "✅ Should be cancelled")
	assert.Equal(suite.T(), 1, downloader.GetTotalDownloadedArchivesCount(), "✅ Should have 1 successful download")

	suite.T().Log("✅ Success then cancellation test passed")
}

// Test case: No success, only cancellation
func (suite *CodexArchiveDownloaderSuite) TestNoSuccess_OnlyCancellation() {
	communityID := "test-community"
	cancelChan := make(chan struct{})

	// 2 archives
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
				Cid:      "cid-1",
			},
			"archive-2": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
				Cid:      "cid-2",
			},
		},
	}

	// Both archives will be cancelled
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string) (*codex.Manifest, error) {
			<-ctx.Done() // Wait for cancellation
			return nil, ctx.Err()
		}).
		AnyTimes()

	logger := zap.NewNop()
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, []string{}, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)
	downloader.SetPollingTimeout(1 * time.Second)

	downloader.StartDownload()

	// Cancel immediately
	time.Sleep(50 * time.Millisecond)
	close(cancelChan)

	// Wait for completion
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 3*time.Second, 50*time.Millisecond)

	// Assertions
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Should be complete")
	assert.True(suite.T(), downloader.IsCancelled(), "✅ Should be cancelled")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "✅ Should have 0 successful downloads")

	suite.T().Log("✅ Only cancellation test passed (no successful downloads)")
}

// Test case: No success, only errors
func (suite *CodexArchiveDownloaderSuite) TestNoSuccess_OnlyErrors() {
	communityID := "test-community"
	cancelChan := make(chan struct{})
	defer close(cancelChan)

	// 2 archives
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata{
			"archive-1": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 1000, To: 2000},
				Cid:      "cid-1",
			},
			"archive-2": {
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
				Cid:      "cid-2",
			},
		},
	}

	// Both archives fail
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-1").
		Return(codex.Manifest{}, fmt.Errorf("trigger failed for cid-1"))
	suite.mockClient.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-2").
		Return(codex.Manifest{}, fmt.Errorf("trigger failed for cid-2"))

	logger := zap.NewNop()
	downloader := communities.NewCodexArchiveDownloader(suite.mockClient, index, communityID, []string{}, cancelChan, logger)
	downloader.SetPollingInterval(10 * time.Millisecond)
	downloader.SetPollingTimeout(1 * time.Second)

	downloader.StartDownload()

	// Wait for completion
	require.Eventually(suite.T(), func() bool {
		return downloader.IsDownloadComplete()
	}, 3*time.Second, 50*time.Millisecond)

	// Assertions
	assert.True(suite.T(), downloader.IsDownloadComplete(), "✅ Should be complete")
	assert.False(suite.T(), downloader.IsCancelled(), "✅ Should NOT be cancelled")
	assert.Equal(suite.T(), 0, downloader.GetTotalDownloadedArchivesCount(), "✅ Should have 0 successful downloads")

	suite.T().Log("✅ Only errors test passed (no successful downloads)")
}
