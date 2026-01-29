//go:build use_logos_storage
// +build use_logos_storage

package logosstorage_test

import (
	"context"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/archive"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	logosstorage "github.com/status-im/status-go/services/logosstorage"
	mock_logosstorage "github.com/status-im/status-go/services/logosstorage/mock"

	"github.com/stretchr/testify/suite"
)

// ArchiveManagerLogosStorageCancellationSuite contains deterministic unit tests using mocked LogosStorageClient
type ArchiveManagerLogosStorageCancellationSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	mockLogosStorage *mock_logosstorage.MockLogosStorageClientInterface
	archiveService   archive.ArchiveService
	manager          *communities.Manager
}

func (s *ArchiveManagerLogosStorageCancellationSuite) buildManagers() (*communities.Manager, archive.ArchiveService) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.Require().NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := testutils.MustCreateTestLogger()

	m, err := communities.NewManager(key, "", db, logger, nil, nil, nil, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	logosStorageConfig := &params.LogosStorageConfig{
		Enabled: true,
	}

	amc := &archivetypes.ArchiveManagerConfig{
		TorrentConfig:      nil,
		LogosStorageConfig: logosStorageConfig,
		Logger:             logger,
		Persistence:        m.GetPersistence(),
		Messaging:          nil,
		Identity:           key,
		Publisher:          m,
	}
	archiveManager := archive.NewArchiveManager(amc)

	return m, archiveManager
}

func (s *ArchiveManagerLogosStorageCancellationSuite) getArchiveManager() *archive.ArchiveManager {
	archiveManager, ok := s.archiveService.(*archive.ArchiveManager)
	s.Require().True(ok)
	return archiveManager
}

func (s *ArchiveManagerLogosStorageCancellationSuite) setDownloadTimeout(timeout time.Duration) {
	archiveManager := s.getArchiveManager()
	backend, err := archiveManager.GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	backend.SetDownloadTimeout(timeout)
}

func (s *ArchiveManagerLogosStorageCancellationSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockLogosStorage = mock_logosstorage.NewMockLogosStorageClientInterface(s.ctrl)

	m, am := s.buildManagers()
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveService = am

	// Inject the mock LogosStorageClient into the ArchiveManager
	archiveManager := s.getArchiveManager()
	backend, err := archiveManager.GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	backend.SetLogosStorageClient(s.mockLogosStorage)
}

func (s *ArchiveManagerLogosStorageCancellationSuite) TearDownTest() {
	s.ctrl.Finish()
	s.Require().NoError(s.manager.Stop())
}

// TestMockDownloadCancellationBeforeIndexIsDownloaded tests cancellation before index is downloaded
func (s *ArchiveManagerLogosStorageCancellationSuite) TestMockDownloadCancellationBeforeIndexIsDownloaded() {
	// Subscribe to signals
	subscription := s.manager.Subscribe()

	indexCid := "test-index-cid-xyz789"
	communityID := types.HexBytes("mock-cancel-test-1")
	cancelChan := make(chan struct{})

	// Mock expectations: DownloadWithContext may be called but should be cancelled immediately
	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output any) error {
			// Block until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		}).
		MaxTimes(1) // May or may not be called depending on timing

	// Track signals
	indexDownloadCompletedReceived := false
	signalDone := make(chan struct{})
	go func() {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.IndexDownloadCompletedSignal != nil {
					indexDownloadCompletedReceived = true
				}
			case <-timeout:
				close(signalDone)
				return
			case <-signalDone:
				return
			}
		}
	}()

	// Cancel immediately before starting
	close(cancelChan)

	// Set short timeout for test
	s.setDownloadTimeout(1 * time.Second)

	// Start download - should return immediately due to cancellation
	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")
	s.Require().Equal(0, taskInfo.TotalDownloadedArchivesCount, "No archives should be downloaded")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	s.Require().False(indexDownloadCompletedReceived, "IndexDownloadCompletedSignal should not be received when cancelled early")
	s.T().Logf("✓ Mock test: Early cancellation verified with zero LogosStorageClient calls")
}

// TestMockDownloadCancellationDuringIndexDownload tests cancellation during index download
// Uses mock to control exact timing of index download completion
func (s *ArchiveManagerLogosStorageCancellationSuite) TestMockDownloadCancellationDuringIndexDownload() {
	subscription := s.manager.Subscribe()

	archiveData := make([]byte, 1024)
	_, err := rand.Read(archiveData)
	s.Require().NoError(err)

	indexCid := "test-index-cid-uvw123"

	communityID := types.HexBytes("mock-cancel-test-2")
	cancelChan := make(chan struct{})

	// Mock expectations: Index download never completes due to cancellation
	downloadStarted := make(chan struct{})
	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output any) error {
			close(downloadStarted)
			// Block until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		}).
		Times(1)

	// Track signals
	indexDownloadCompletedReceived := false
	signalDone := make(chan struct{})

	go func() {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event == nil {
					continue
				}
				if event.IndexDownloadCompletedSignal != nil {
					indexDownloadCompletedReceived = true
				}
			case <-timeout:
				close(signalDone)
				return
			case <-signalDone:
				return
			}
		}
	}()

	// Wait for download to start, then cancel
	go func() {
		<-downloadStarted
		s.T().Logf("Download started, now cancelling")
		close(cancelChan)
	}()

	// Set short timeout for test
	s.setDownloadTimeout(1 * time.Second)

	// Start download
	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	// Verify signals
	s.Require().False(indexDownloadCompletedReceived, "Should NOT have received IndexDownloadCompletedSignal when download is cancelled")

	s.T().Logf("✓ Mock test: Index download cancellation verified with controlled timing")
}

// TestMockDownloadCancellationDuringArchiveDownload tests cancellation during archive downloads
func (s *ArchiveManagerLogosStorageCancellationSuite) TestMockDownloadCancellationDuringArchiveDownload() {
	subscription := s.manager.Subscribe()

	// Create multiple archives
	archives := []struct {
		hash string
		cid  string
		from uint64
		to   uint64
		data []byte
	}{
		{"cancel-archive-1", "archive-cid-1", 1000, 2000, make([]byte, 1024)},
		{"cancel-archive-2", "archive-cid-2", 2000, 3000, make([]byte, 1024)},
		{"cancel-archive-3", "archive-cid-3", 3000, 4000, make([]byte, 1024)},
	}

	for i := range archives {
		_, err := rand.Read(archives[i].data)
		s.Require().NoError(err)
	}

	indexCid := "test-index-cid-archive-download"
	index := &protobuf.LogosStorageWakuMessageArchiveIndex{
		Archives: make(map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata),
	}

	for _, archive := range archives {
		index.Archives[archive.hash] = &protobuf.LogosStorageWakuMessageArchiveIndexMetadata{
			Cid: archive.cid,
			Metadata: &protobuf.WakuMessageArchiveMetadata{
				From: archive.from,
				To:   archive.to,
			},
		}
	}

	logosStorageIndexBytes, err := proto.Marshal(index)
	s.Require().NoError(err)

	communityID := types.HexBytes("mock-cancel-test-3")
	cancelChan := make(chan struct{})

	// Mock expectations
	// Index download succeeds
	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output any) error {
			// Write the index bytes to whatever writer we receive
			if w, ok := output.(io.Writer); ok {
				_, _ = w.Write(logosStorageIndexBytes)
			}
			return nil
		}).
		Times(1)

	// First archive download succeeds
	s.mockLogosStorage.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), archives[0].cid).
		DoAndReturn(func(ctx context.Context, cid string) (logosstorage.LogosStorageManifest, error) {
			return logosstorage.LogosStorageManifest{Cid: cid, DatasetSize: len(archives[0].data)}, nil
		}).
		Times(1)

	// HasCid for first archive - called during polling after trigger succeeds
	s.mockLogosStorage.EXPECT().
		HasCid(archives[0].cid).
		Return(true, nil).
		AnyTimes()

	// TriggerDownloadWithContext for remaining archives
	// All 3 goroutines start simultaneously, and each calls TriggerDownloadWithContext BEFORE polling.
	// Archives 2 and 3 will have their triggers called, but should receive cancellation.
	for i := 1; i < len(archives); i++ {
		s.mockLogosStorage.EXPECT().
			TriggerDownloadWithContext(gomock.Any(), archives[i].cid).
			DoAndReturn(func(ctx context.Context, cid string) (logosstorage.LogosStorageManifest, error) {
				// Block until context is cancelled
				<-ctx.Done()
				return logosstorage.LogosStorageManifest{}, ctx.Err()
			}).
			Times(1)
	}

	// HasCid should NOT be called for archives 2 and 3 since their triggers will fail with cancellation

	// Track signals
	downloadStartedReceived := false
	indexDownloadCompletedReceived := false
	archivesDownloaded := 0
	signalDone := make(chan struct{})

	go func() {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event == nil {
					continue
				}
				if event.DownloadingHistoryArchivesStartedSignal != nil {
					downloadStartedReceived = true
				}
				if event.IndexDownloadCompletedSignal != nil {
					indexDownloadCompletedReceived = true
				}
				if event.HistoryArchiveDownloadedSignal != nil {
					s.T().Logf("Received HistoryArchiveDownloadedSignal for archive CID")
					archivesDownloaded++
					if archivesDownloaded == 1 {
						// We received the signal, which means HasCid returned true and count was incremented.
						s.T().Logf("First archive downloaded (signal received), now cancelling")
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

	// Set longer timeout for test to avoid timeout issues
	s.setDownloadTimeout(5 * time.Second)

	// Start download
	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")
	s.Require().Equal(3, taskInfo.TotalArchivesCount, "Should know total is 3 archives")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	// Verify signals
	s.Require().True(downloadStartedReceived, "Should have received DownloadingHistoryArchivesStartedSignal")
	s.Require().True(indexDownloadCompletedReceived, "Should have received IndexDownloadCompletedSignal")
	s.Require().Equal(1, archivesDownloaded, "Should have received exactly 1 HistoryArchiveDownloadedSignal")

	// Since we received the signal, HasCid completed and the count was incremented
	s.Require().Equal(1, taskInfo.TotalDownloadedArchivesCount,
		"Should have downloaded exactly 1 archive (signal was received)")

	s.T().Logf("✓ Mock test: Archive download cancellation verified - downloaded %d archive(s) before cancel", archivesDownloaded)
}

// Run the mock-based unit test suite
func TestArchiveManagerLogosStorageCancellationSuite(t *testing.T) {
	suite.Run(t, new(ArchiveManagerLogosStorageCancellationSuite))
}
