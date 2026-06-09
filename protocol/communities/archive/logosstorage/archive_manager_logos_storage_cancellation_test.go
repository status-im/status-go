//go:build use_logos_storage

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
	mocklogosstorage "github.com/status-im/status-go/services/logosstorage/mock"

	"github.com/stretchr/testify/suite"
)

type ArchiveManagerLogosStorageCancellationSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	mockLogosStorage *mocklogosstorage.MockLogosStorageClientInterface
	archiveService   archive.ArchiveService
	manager          *communities.Manager
}

func (s *ArchiveManagerLogosStorageCancellationSuite) buildManagers() (*communities.Manager, archive.ArchiveService) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	s.Require().NoError(sqlite.Migrate(db), "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := testutils.MustCreateTestLogger()

	m, err := communities.NewManager(key, "", db, logger, nil, nil, nil, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	logosStorageConfig := &params.LogosStorageConfig{Enabled: true}
	amc := &archivetypes.ArchiveManagerConfig{
		LogosStorageConfig: logosStorageConfig,
		Logger:             logger,
		Persistence:        m.GetPersistence(),
		Identity:           key,
		Publisher:          m,
	}

	return m, archive.NewArchiveManager(amc)
}

func (s *ArchiveManagerLogosStorageCancellationSuite) getArchiveManager() *archive.ArchiveManager {
	archiveManager, ok := s.archiveService.(*archive.ArchiveManager)
	s.Require().True(ok)
	return archiveManager
}

func (s *ArchiveManagerLogosStorageCancellationSuite) setDownloadTimeout(timeout time.Duration) {
	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	backend.SetDownloadTimeout(timeout)
}

func (s *ArchiveManagerLogosStorageCancellationSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockLogosStorage = mocklogosstorage.NewMockLogosStorageClientInterface(s.ctrl)

	m, am := s.buildManagers()
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveService = am

	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	backend.SetLogosStorageClient(s.mockLogosStorage)
}

func (s *ArchiveManagerLogosStorageCancellationSuite) TearDownTest() {
	s.ctrl.Finish()
	s.Require().NoError(s.manager.Stop())
}

func (s *ArchiveManagerLogosStorageCancellationSuite) TestMockDownloadCancellationBeforeIndexIsDownloaded() {
	subscription := s.manager.Subscribe()

	indexCid := "test-index-cid-xyz789"
	communityID := types.HexBytes("mock-cancel-test-1")
	cancelChan := make(chan struct{})

	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			<-ctx.Done()
			return ctx.Err()
		}).
		MaxTimes(1)

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

	close(cancelChan)
	s.setDownloadTimeout(1 * time.Second)

	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")
	s.Require().Equal(0, taskInfo.TotalDownloadedArchivesCount, "No archives should be downloaded")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	s.Require().False(indexDownloadCompletedReceived, "IndexDownloadCompletedSignal should not be received when cancelled early")
}

func (s *ArchiveManagerLogosStorageCancellationSuite) TestMockDownloadCancellationDuringIndexDownload() {
	subscription := s.manager.Subscribe()

	archiveData := make([]byte, 1024)
	_, err := rand.Read(archiveData)
	s.Require().NoError(err)

	indexCid := "test-index-cid-uvw123"
	communityID := types.HexBytes("mock-cancel-test-2")
	cancelChan := make(chan struct{})

	downloadStarted := make(chan struct{})
	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			close(downloadStarted)
			<-ctx.Done()
			return ctx.Err()
		}).
		Times(1)

	indexDownloadCompletedReceived := false
	signalDone := make(chan struct{})

	go func() {
		timeout := time.After(10 * time.Second)
		for {
			select {
			case event := <-subscription:
				if event != nil && event.IndexDownloadCompletedSignal != nil {
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

	go func() {
		<-downloadStarted
		close(cancelChan)
	}()

	s.setDownloadTimeout(1 * time.Second)

	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	s.Require().False(indexDownloadCompletedReceived, "Should NOT have received IndexDownloadCompletedSignal when download is cancelled")
}

func (s *ArchiveManagerLogosStorageCancellationSuite) TestMockDownloadCancellationDuringArchiveDownload() {
	subscription := s.manager.Subscribe()

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

	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			_, _ = output.Write(logosStorageIndexBytes)
			return nil
		}).
		Times(1)

	s.mockLogosStorage.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), archives[0].cid).
		DoAndReturn(func(ctx context.Context, cid string) (logosstorage.LogosStorageManifest, error) {
			return logosstorage.LogosStorageManifest{Cid: cid, DatasetSize: len(archives[0].data)}, nil
		}).
		Times(1)

	s.mockLogosStorage.EXPECT().
		HasCid(archives[0].cid).
		Return(true, nil).
		AnyTimes()

	for i := 1; i < len(archives); i++ {
		s.mockLogosStorage.EXPECT().
			TriggerDownloadWithContext(gomock.Any(), archives[i].cid).
			DoAndReturn(func(ctx context.Context, cid string) (logosstorage.LogosStorageManifest, error) {
				<-ctx.Done()
				return logosstorage.LogosStorageManifest{}, ctx.Err()
			}).
			Times(1)
	}

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

	s.setDownloadTimeout(5 * time.Second)

	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().True(taskInfo.Cancelled, "Download should be marked as cancelled")
	s.Require().Equal(3, taskInfo.TotalArchivesCount, "Should know total is 3 archives")

	close(signalDone)
	time.Sleep(50 * time.Millisecond)

	s.Require().True(downloadStartedReceived, "Should have received DownloadingHistoryArchivesStartedSignal")
	s.Require().True(indexDownloadCompletedReceived, "Should have received IndexDownloadCompletedSignal")
	s.Require().Equal(1, archivesDownloaded, "Should have received exactly 1 HistoryArchiveDownloadedSignal")
	s.Require().Equal(1, taskInfo.TotalDownloadedArchivesCount, "Should have downloaded exactly 1 archive")
}

func TestArchiveManagerLogosStorageCancellationSuite(t *testing.T) {
	suite.Run(t, new(ArchiveManagerLogosStorageCancellationSuite))
}
