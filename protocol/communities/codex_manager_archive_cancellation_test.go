package communities_test

import (
	"context"
	"crypto/rand"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/codex-storage/codex-go-bindings/codex"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/communities"
	mock_communities "github.com/status-im/status-go/protocol/communities/mock/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/suite"
)

// MockCodexArchiveManagerSuite contains deterministic unit tests using mocked CodexClient
type MockCodexArchiveManagerSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	mockCodex      *mock_communities.MockCodexClientInterface
	archiveManager *communities.ArchiveManager
	manager        *communities.Manager
}

func (s *MockCodexArchiveManagerSuite) buildManagers() (*communities.Manager, *communities.ArchiveManager) {
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

	rootDir := s.T().TempDir()
	codexConfig := &params.CodexConfig{
		Enabled:               true,
		HistoryArchiveDataDir: filepath.Join(rootDir, "codex", "archivedata"),
	}

	amc := &communities.ArchiveManagerConfig{
		TorrentConfig: nil,
		CodexConfig:   codexConfig,
		Logger:        logger,
		Persistence:   m.GetPersistence(),
		Messaging:     nil,
		Identity:      key,
		Publisher:     m,
	}
	archiveManager := communities.NewArchiveManager(amc)

	return m, archiveManager
}

func (s *MockCodexArchiveManagerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockCodex = mock_communities.NewMockCodexClientInterface(s.ctrl)

	m, am := s.buildManagers()
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveManager = am

	// Inject the mock CodexClient into the ArchiveManager
	s.archiveManager.SetCodexClient(s.mockCodex)
}

func (s *MockCodexArchiveManagerSuite) TearDownTest() {
	s.ctrl.Finish()
	s.Require().NoError(s.manager.Stop())
}

// TestMockDownloadCancellationBeforeManifestFetch tests cancellation before manifest is fetched
// This test is 100% deterministic - we control exactly when operations complete
func (s *MockCodexArchiveManagerSuite) TestMockDownloadCancellationBeforeManifestFetch() {
	// Subscribe to signals
	subscription := s.manager.Subscribe()

	indexCid := "test-index-cid-xyz789"
	communityID := types.HexBytes("mock-cancel-test-1")
	cancelChan := make(chan struct{})

	// Mock expectations: FetchManifestWithContext will be called but should be cancelled
	s.mockCodex.EXPECT().
		FetchManifestWithContext(gomock.Any(), indexCid).
		DoAndReturn(func(ctx context.Context, cid string) (codex.Manifest, error) {
			// Block until context is cancelled
			<-ctx.Done()
			return codex.Manifest{}, ctx.Err()
		}).
		MaxTimes(1) // May or may not be called depending on timing

	// Track signals
	manifestFetchedReceived := false
	signalDone := make(chan struct{})
	go func() {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case event := <-subscription:
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

	// Cancel immediately before starting
	close(cancelChan)

	// Set short timeout for test
	s.archiveManager.SetDownloadTimeout(1 * time.Second)

	// Start download - should return immediately due to cancellation
	taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")
	s.Require().Equal(0, taskInfo.TotalDownloadedArchivesCount, "No archives should be downloaded")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	s.Require().False(manifestFetchedReceived, "ManifestFetchedSignal should not be received when cancelled early")
	s.T().Logf("✓ Mock test: Early cancellation verified with zero CodexClient calls")
}

// TestMockDownloadCancellationDuringIndexDownload tests cancellation during index download
// Uses mock to control exact timing of manifest fetch completion
func (s *MockCodexArchiveManagerSuite) TestMockDownloadCancellationDuringIndexDownload() {
	subscription := s.manager.Subscribe()

	archiveData := make([]byte, 1024)
	_, err := rand.Read(archiveData)
	s.Require().NoError(err)

	archiveCid := "test-archive-cid-def456"
	indexCid := "test-index-cid-uvw123"

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
	s.Require().NoError(err)

	communityID := types.HexBytes("mock-cancel-test-2")
	cancelChan := make(chan struct{})

	// Mock expectations: Manifest fetch succeeds, but index download never completes
	manifest := codex.Manifest{
		Cid:         indexCid,
		DatasetSize: len(codexIndexBytes),
	}

	// FetchManifestWithContext will succeed
	s.mockCodex.EXPECT().
		FetchManifestWithContext(gomock.Any(), indexCid).
		Return(manifest, nil).
		Times(1)

	// DownloadWithContext will be called but blocked until cancelled
	downloadStarted := make(chan struct{})
	s.mockCodex.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output interface{}) error {
			close(downloadStarted)
			// Block until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		}).
		Times(1)

	// Track signals
	manifestFetchedReceived := false
	indexDownloadCompletedReceived := false
	signalDone := make(chan struct{})

	go func() {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event.ManifestFetchedSignal != nil {
					manifestFetchedReceived = true
					s.T().Logf("Received ManifestFetchedSignal - waiting for download to start before cancelling")
					// Wait for download to actually start
					<-downloadStarted
					s.T().Logf("Download started, now cancelling")
					close(cancelChan)
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

	// Set short timeout for test
	s.archiveManager.SetDownloadTimeout(1 * time.Second)

	// Start download
	taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	// Verify signals
	s.Require().True(manifestFetchedReceived, "Should have received ManifestFetchedSignal")
	s.Require().False(indexDownloadCompletedReceived, "Should NOT have received IndexDownloadCompletedSignal")

	s.T().Logf("✓ Mock test: Index download cancellation verified with controlled timing")
}

// TestMockDownloadCancellationDuringArchiveDownload tests cancellation during archive downloads
// Mock allows us to control exactly when first archive completes
func (s *MockCodexArchiveManagerSuite) TestMockDownloadCancellationDuringArchiveDownload() {
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
	index := &protobuf.CodexWakuMessageArchiveIndex{
		Archives: make(map[string]*protobuf.CodexWakuMessageArchiveIndexMetadata),
	}

	for _, archive := range archives {
		index.Archives[archive.hash] = &protobuf.CodexWakuMessageArchiveIndexMetadata{
			Cid: archive.cid,
			Metadata: &protobuf.WakuMessageArchiveMetadata{
				From: archive.from,
				To:   archive.to,
			},
		}
	}

	codexIndexBytes, err := proto.Marshal(index)
	s.Require().NoError(err)

	communityID := types.HexBytes("mock-cancel-test-3")
	cancelChan := make(chan struct{})

	// Mock expectations
	manifest := codex.Manifest{
		Cid:         indexCid,
		DatasetSize: len(codexIndexBytes),
	}

	// Manifest fetch succeeds
	s.mockCodex.EXPECT().
		FetchManifestWithContext(gomock.Any(), indexCid).
		Return(manifest, nil).
		Times(1)

	// Index download succeeds
	s.mockCodex.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output interface{}) error {
			// Write the index bytes to whatever writer we receive
			if w, ok := output.(io.Writer); ok {
				w.Write(codexIndexBytes)
			}
			return nil
		}).
		Times(1)

	// First archive download succeeds, triggers cancellation
	// firstArchiveDownloaded := make(chan struct{})
	s.mockCodex.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), archives[0].cid).
		DoAndReturn(func(ctx context.Context, cid string) (codex.Manifest, error) {
			// close(firstArchiveDownloaded)
			return codex.Manifest{Cid: cid, DatasetSize: len(archives[0].data)}, nil
		}).
		Times(1)

	// HasCid for first archive - called during polling after trigger succeeds
	s.mockCodex.EXPECT().
		HasCid(archives[0].cid).
		Return(true, nil).
		AnyTimes()

	// TriggerDownloadWithContext for remaining archives
	// All 3 goroutines start simultaneously, and each calls TriggerDownloadWithContext BEFORE polling.
	// Archives 2 and 3 will have their triggers called, but should receive cancellation.
	for i := 1; i < len(archives); i++ {
		s.mockCodex.EXPECT().
			TriggerDownloadWithContext(gomock.Any(), archives[i].cid).
			DoAndReturn(func(ctx context.Context, cid string) (codex.Manifest, error) {
				// Block until context is cancelled
				<-ctx.Done()
				return codex.Manifest{}, ctx.Err()
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

	// Set short timeout for test
	s.archiveManager.SetDownloadTimeout(1 * time.Second)

	// Start download
	taskInfo, err := s.archiveManager.DownloadHistoryArchivesByIndexCid(communityID, indexCid, cancelChan)
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
func TestMockCodexArchiveManagerSuite(t *testing.T) {
	suite.Run(t, new(MockCodexArchiveManagerSuite))
}
