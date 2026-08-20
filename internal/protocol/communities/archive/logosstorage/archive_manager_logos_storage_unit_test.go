//go:build use_logos_storage

package logosstorage_test

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/communities/archive"
	archiveconsts "github.com/status-im/status-go/internal/protocol/communities/archive/consts"
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	logosstorage "github.com/status-im/status-go/pkg/services/logosstorage"
	mocklogosstorage "github.com/status-im/status-go/pkg/services/logosstorage/mock"

	"github.com/stretchr/testify/suite"
)

type ArchiveManagerLogosStorageMockSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	mockLogosStorage *mocklogosstorage.MockLogosStorageClientInterface
	archiveService   archive.ArchiveService
	manager          *communities.Manager
}

func (s *ArchiveManagerLogosStorageMockSuite) buildManagers() (*communities.Manager, archive.ArchiveService) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	s.Require().NoError(sqlite.Migrate(db), "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := testutils.MustCreateTestLogger()

	m, err := communities.NewManager(key, "", db, logger, nil, nil, nil, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	config := &params.LogosStorageConfig{Enabled: true}
	amc := &archivetypes.ArchiveManagerConfig{
		LogosStorageConfig: config,
		Logger:             logger,
		Persistence:        m.GetPersistence(),
		Identity:           key,
		Publisher:          m,
	}

	return m, archive.NewArchiveManager(amc)
}

func (s *ArchiveManagerLogosStorageMockSuite) getArchiveManager() *archive.ArchiveManager {
	archiveManager, ok := s.archiveService.(*archive.ArchiveManager)
	s.Require().True(ok)
	return archiveManager
}

func (s *ArchiveManagerLogosStorageMockSuite) setDownloadTimeout(timeout time.Duration) {
	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	backend.SetDownloadTimeout(timeout)
}

func (s *ArchiveManagerLogosStorageMockSuite) SetupTest() {
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

func (s *ArchiveManagerLogosStorageMockSuite) TearDownTest() {
	s.ctrl.Finish()
	s.Require().NoError(s.manager.Stop())
}

func (s *ArchiveManagerLogosStorageMockSuite) TestSeedHistoryArchiveSkipsEmptyLink() {
	communityID := types.HexBytes("seed-empty-link")

	s.Require().NoError(s.archiveService.SeedHistoryArchive(communityID, ""))
}

func (s *ArchiveManagerLogosStorageMockSuite) TestSeedHistoryArchiveSkipsWhenNotStarted() {
	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")
	backend.SetLogosStorageClient(nil)

	communityID := types.HexBytes("seed-not-started")

	s.Require().NoError(s.archiveService.SeedHistoryArchive(communityID, "index-cid"))
}

func (s *ArchiveManagerLogosStorageMockSuite) TestSeedHistoryArchiveSkipsWhenAlreadySeeding() {
	communityID := types.HexBytes("seed-already-seeding")
	archiveLink := "index-cid"

	s.mockLogosStorage.EXPECT().
		HasCid(archiveLink).
		Return(true, nil).
		Times(1)

	s.Require().NoError(s.archiveService.SeedHistoryArchive(communityID, archiveLink))
}

func (s *ArchiveManagerLogosStorageMockSuite) TestSeedHistoryArchiveTriggersDownloadWhenNotSeeding() {
	communityID := types.HexBytes("seed-trigger-download")
	archiveLink := "index-cid"

	s.mockLogosStorage.EXPECT().
		HasCid(archiveLink).
		Return(false, nil).
		Times(1)
	s.mockLogosStorage.EXPECT().
		TriggerDownload(archiveLink).
		Return(logosstorage.LogosStorageManifest{Cid: archiveLink}, nil).
		Times(1)

	s.Require().NoError(s.archiveService.SeedHistoryArchive(communityID, archiveLink))
}

func (s *ArchiveManagerLogosStorageMockSuite) TestSeedHistoryArchiveReturnsTriggerDownloadError() {
	communityID := types.HexBytes("seed-trigger-download-error")
	archiveLink := "index-cid"
	expectedErr := errors.New("trigger download failed")

	s.mockLogosStorage.EXPECT().
		HasCid(archiveLink).
		Return(false, nil).
		Times(1)
	s.mockLogosStorage.EXPECT().
		TriggerDownload(archiveLink).
		Return(logosstorage.LogosStorageManifest{}, expectedErr).
		Times(1)

	err := s.archiveService.SeedHistoryArchive(communityID, archiveLink)
	s.Require().ErrorIs(err, expectedErr)
}

func (s *ArchiveManagerLogosStorageMockSuite) TestMockDownloadCancellationBeforeIndexIsDownloaded() {
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

func (s *ArchiveManagerLogosStorageMockSuite) TestMockDownloadCancellationDuringIndexDownload() {
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

func (s *ArchiveManagerLogosStorageMockSuite) TestMockDownloadCancellationDuringArchiveDownload() {
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

	indexBytes, err := proto.Marshal(index)
	s.Require().NoError(err)

	communityID := types.HexBytes("mock-cancel-test-3")
	cancelChan := make(chan struct{})

	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			_, _ = output.Write(indexBytes)
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

func (s *ArchiveManagerLogosStorageMockSuite) TestDownloadDoesNotAbortWhenExistingAndIncomingArchiveCountsMatchButHashesDiffer() {
	indexCid := "test-index-cid-same-count-different-hash"
	communityID := types.HexBytes("mock-same-count-different-hash")
	cancelChan := make(chan struct{})
	defer close(cancelChan)

	s.Require().NoError(s.archiveService.SaveMessageArchiveID(communityID, "archive-1"))

	index := &protobuf.LogosStorageWakuMessageArchiveIndex{
		Archives: map[string]*protobuf.LogosStorageWakuMessageArchiveIndexMetadata{
			"archive-2": {
				Cid:      "cid-2",
				Metadata: &protobuf.WakuMessageArchiveMetadata{From: 2000, To: 3000},
			},
		},
	}
	indexBytes, err := proto.Marshal(index)
	s.Require().NoError(err)

	s.mockLogosStorage.EXPECT().
		DownloadWithContext(gomock.Any(), indexCid, gomock.Any()).
		DoAndReturn(func(ctx context.Context, cid string, output io.Writer) error {
			_, _ = output.Write(indexBytes)
			return nil
		}).
		Times(1)

	s.mockLogosStorage.EXPECT().
		TriggerDownloadWithContext(gomock.Any(), "cid-2").
		Return(logosstorage.LogosStorageManifest{Cid: "cid-2"}, nil).
		Times(1)

	s.mockLogosStorage.EXPECT().
		HasCid("cid-2").
		Return(true, nil).
		AnyTimes()

	s.setDownloadTimeout(5 * time.Second)
	taskInfo, err := s.archiveService.DownloadHistoryArchives(communityID, indexCid, cancelChan)
	s.Require().NoError(err)
	s.Require().NotNil(taskInfo)
	s.Require().Equal(1, taskInfo.TotalArchivesCount)
	s.Require().Equal(2, taskInfo.TotalDownloadedArchivesCount)

	downloadedArchiveIDs, err := s.archiveService.GetDownloadedMessageArchiveIDs(communityID)
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{"archive-1", "archive-2"}, downloadedArchiveIDs)
}

func (s *ArchiveManagerLogosStorageMockSuite) TestChunkArchiveMessagesReturnsEmptyForNoMessages() {
	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")

	chunks := backend.ChunkArchiveMessages(nil)
	s.Require().Empty(chunks)

	chunks = backend.ChunkArchiveMessages([]messagingtypes.ReceivedMessage{})
	s.Require().Empty(chunks)
}

func (s *ArchiveManagerLogosStorageMockSuite) TestChunkArchiveMessagesReturnsEmptyWhenAllMessagesOversized() {
	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")

	oversizedPayload := make([]byte, archiveconsts.MaxArchiveSizeInBytes+1)
	msgs := []messagingtypes.ReceivedMessage{
		{Payload: oversizedPayload, Sig: []byte("sig1")},
		{Payload: oversizedPayload, Sig: []byte("sig2")},
	}

	chunks := backend.ChunkArchiveMessages(msgs)
	s.Require().Empty(chunks)
}

func (s *ArchiveManagerLogosStorageMockSuite) TestChunkArchiveMessagesChunksNormalMessages() {
	backend, err := s.getArchiveManager().GetLogosStorageBackend()
	s.Require().NoError(err, "Failed to get LogosStorage backend")

	msgs := []messagingtypes.ReceivedMessage{
		{Payload: []byte("msg1"), Sig: []byte("sig1")},
		{Payload: []byte("msg2"), Sig: []byte("sig2")},
		{Payload: []byte("msg3"), Sig: []byte("sig3")},
	}

	chunks := backend.ChunkArchiveMessages(msgs)
	s.Require().Len(chunks, 1)
	s.Require().Len(chunks[0], 3)
}

func TestArchiveManagerLogosStorageMockSuite(t *testing.T) {
	suite.Run(t, new(ArchiveManagerLogosStorageMockSuite))
}
