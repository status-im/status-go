package torrent_test

import (
	"os"
	"testing"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/appdatabase"
	testutils "github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/archive"
	archivetorrent "github.com/status-im/status-go/protocol/communities/archive/torrent"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestArchiveManagerTorrentSuite(t *testing.T) {
	suite.Run(t, new(ArchiveManagerTorrentSuite))
}

type ArchiveManagerTorrentSuite struct {
	suite.Suite
	manager        *communities.Manager
	archiveService archive.ArchiveService
}

type TimeSourceStub struct {
}

func (t *TimeSourceStub) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

func buildTorrentConfig() *params.TorrentConfig {
	return &params.TorrentConfig{
		Enabled:    true,
		DataDir:    os.TempDir() + "/archivedata",
		TorrentDir: os.TempDir() + "/torrents",
		Port:       0,
	}
}

func buildMessage(timestamp time.Time, topic types.ContentTopic, hash []byte) types.ReceivedMessage {
	message := types.ReceivedMessage{
		Sig:       []byte{1},
		Timestamp: uint32(timestamp.Unix()),
		Topic:     topic,
		Payload:   []byte{1},
		Padding:   []byte{1},
		Hash:      hash,
	}
	return message
}

func (s *ArchiveManagerTorrentSuite) buildCommunityWithChat() (*communities.Community, string, error) {
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	if err != nil {
		return nil, "", err
	}
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "added-chat",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}
	changes, err := s.manager.CreateChat(community.ID(), chat, true, "")
	if err != nil {
		return nil, "", err
	}

	chatID := ""
	for cID := range changes.ChatsAdded {
		chatID = cID
		break
	}
	return community, chatID, nil
}

func (s *ArchiveManagerTorrentSuite) buildManagers(ownerVerifier communities.OwnerVerifier) (*communities.Manager, archive.ArchiveService) {
	db, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.Require().NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	logger := testutils.MustCreateTestLogger()

	m, err := communities.NewManager(key, "", db, logger, nil, ownerVerifier, nil, &TimeSourceStub{}, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	amc := &archivetypes.ArchiveManagerConfig{
		TorrentConfig: buildTorrentConfig(),
		Logger:        logger,
		Persistence:   m.GetPersistence(),
		Messaging:     nil,
		Identity:      key,
		Publisher:     m,
	}
	t := archive.NewArchiveManager(amc)
	s.Require().NoError(err)

	return m, t
}

func (s *ArchiveManagerTorrentSuite) getArchiveManager() *archive.ArchiveManager {
	archiveManager, ok := s.archiveService.(*archive.ArchiveManager)
	s.Require().True(ok)
	return archiveManager
}

func (s *ArchiveManagerTorrentSuite) getTorrentBackend() *archivetorrent.ArchiveManagerTorrent {
	archiveManager := s.getArchiveManager()
	backend, err := archiveManager.GetTorrentBackend()
	s.Require().NoError(err)
	return backend
}

func (s *ArchiveManagerTorrentSuite) getTorrentConfig() *params.TorrentConfig {
	return s.getTorrentBackend().GetTorrentConfig()
}

func (s *ArchiveManagerTorrentSuite) getTorrentFilePath(communityID string) string {
	return s.getTorrentBackend().GetTorrentFilePath(communityID)
}

func (s *ArchiveManagerTorrentSuite) SetupTest() {
	m, t := s.buildManagers(nil)
	communities.SetValidateInterval(30 * time.Millisecond)
	s.manager = m
	s.archiveService = t
}

func (s *ArchiveManagerTorrentSuite) TestStartAndStopTorrentClient() {
	err := s.archiveService.Start()
	s.Require().NoError(err)
	s.Require().True(s.archiveService.IsStarted())
	defer s.archiveService.Stop() //nolint: errcheck

	torrentConfig := s.getTorrentConfig()

	_, err = os.Stat(torrentConfig.DataDir)
	s.Require().NoError(err)
}

func (s *ArchiveManagerTorrentSuite) TestStartHistoryArchiveTasksInterval() {
	err := s.archiveService.Start()
	s.Require().NoError(err)
	defer s.archiveService.Stop() //nolint: errcheck

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	interval := 10 * time.Second
	go s.archiveService.StartHistoryArchiveTasksInterval(community.ID(), community.UniversalChatID(), community.Encrypted(), interval)
	// Due to async exec we need to wait a bit until we check
	// the task count.
	time.Sleep(5 * time.Second)

	count := s.archiveService.GetHistoryTasksCount()
	s.Require().Equal(count, 1)

	// We wait another 5 seconds to ensure the first tick has kicked in
	time.Sleep(5 * time.Second)

	_, err = os.Stat(s.getTorrentFilePath(community.IDString()))
	s.Require().Error(err)

	s.archiveService.StopHistoryArchiveTasksInterval(community.ID())

	archiveManager := s.getArchiveManager()
	archiveManager.Wait()
	count = s.archiveService.GetHistoryTasksCount()
	s.Require().Equal(count, 0)
}

func (s *ArchiveManagerTorrentSuite) TestStopHistoryArchiveTasksIntervals() {
	err := s.archiveService.Start()
	s.Require().NoError(err)
	defer s.archiveService.Stop() //nolint: errcheck

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	interval := 10 * time.Second
	go s.archiveService.StartHistoryArchiveTasksInterval(community.ID(), community.UniversalChatID(), community.Encrypted(), interval)

	time.Sleep(2 * time.Second)

	count := s.archiveService.GetHistoryTasksCount()
	s.Require().Equal(count, 1)

	archiveManager := s.getArchiveManager()
	archiveManager.StopHistoryArchiveTasksIntervalsAndWait()

	count = s.archiveService.GetHistoryTasksCount()
	s.Require().Equal(count, 0)
}

func (s *ArchiveManagerTorrentSuite) TestStopTorrentClient_ShouldStopHistoryArchiveTasks() {
	err := s.archiveService.Start()
	s.Require().NoError(err)
	defer s.archiveService.Stop() //nolint: errcheck

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	interval := 10 * time.Second
	go s.archiveService.StartHistoryArchiveTasksInterval(community.ID(), community.UniversalChatID(), community.Encrypted(), interval)
	// Due to async exec we need to wait a bit until we check
	// the task count.
	time.Sleep(2 * time.Second)

	count := s.archiveService.GetHistoryTasksCount()
	s.Require().Equal(count, 1)

	err = s.archiveService.Stop()
	s.Require().NoError(err)

	count = s.archiveService.GetHistoryTasksCount()
	s.Require().Equal(count, 0)
}

func (s *ArchiveManagerTorrentSuite) TestStartTorrentClient_DelayedUntilOnline() {
	s.Require().False(s.archiveService.IsStarted())

	s.archiveService.SetOnline(true)
	s.Require().True(s.archiveService.IsStarted())
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrent_WithoutMessages() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 7 days
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// Partition of 7 days
	partition := 7 * 24 * time.Hour

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	// There are no waku messages in the database so we don't expect
	// any archives to be created
	_, err = os.Stat(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().Error(err)
	_, err = os.Stat(torrentBackend.GetArchiveIndexFilePath(community.IDString()))
	s.Require().Error(err)
	_, err = os.Stat(torrentBackend.GetTorrentFilePath(community.IDString()))
	s.Require().Error(err)
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrent_ShouldCreateArchive() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 7 days
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// Partition of 7 days, this should create a single archive
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	// This message is outside of the startDate-endDate range and should not
	// be part of the archive
	message3 := buildMessage(endDate.Add(2*time.Hour), topic, []byte{3})

	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)
	err = s.manager.StoreWakuMessage(&message2)
	s.Require().NoError(err)
	err = s.manager.StoreWakuMessage(&message3)
	s.Require().NoError(err)

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	_, err = os.Stat(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(torrentBackend.GetArchiveIndexFilePath(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(torrentBackend.GetTorrentFilePath(community.IDString()))
	s.Require().NoError(err)

	index, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	totalData, err := os.ReadFile(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().NoError(err)

	for _, metadata := range index.Archives {
		archive := &protobuf.WakuMessageArchive{}
		data := totalData[metadata.Offset : metadata.Offset+metadata.Size-metadata.Padding]

		err = proto.Unmarshal(data, archive)
		s.Require().NoError(err)

		s.Require().Len(archive.Messages, 2)
	}
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrent_ShouldCreateMultipleArchives() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 3 weeks
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 21, 00, 00, 00, 0, time.UTC)
	// 7 days partition, this should create three archives
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	// We expect 2 archives to be created for startDate - endDate of each
	// 7 days of data. This message should end up in the second archive
	message3 := buildMessage(startDate.Add(8*24*time.Hour), topic, []byte{3})
	// This one should end up in the third archive
	message4 := buildMessage(startDate.Add(14*24*time.Hour), topic, []byte{4})

	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)
	err = s.manager.StoreWakuMessage(&message2)
	s.Require().NoError(err)
	err = s.manager.StoreWakuMessage(&message3)
	s.Require().NoError(err)
	err = s.manager.StoreWakuMessage(&message4)
	s.Require().NoError(err)

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 3)

	totalData, err := os.ReadFile(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().NoError(err)

	// First archive has 2 messages
	// Second archive has 1 message
	// Third archive has 1 message
	fromMap := map[uint64]int{
		uint64(startDate.Unix()):                    2,
		uint64(startDate.Add(partition).Unix()):     1,
		uint64(startDate.Add(partition * 2).Unix()): 1,
	}

	for _, metadata := range index.Archives {
		archive := &protobuf.WakuMessageArchive{}
		data := totalData[metadata.Offset : metadata.Offset+metadata.Size-metadata.Padding]

		err = proto.Unmarshal(data, archive)
		s.Require().NoError(err)
		s.Require().Len(archive.Messages, fromMap[metadata.Metadata.From])
	}
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrent_ShouldAppendArchives() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 1 week
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// 7 days partition, this should create one archive
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	// Time range of next week
	startDate = time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	endDate = time.Date(2020, 1, 14, 00, 00, 00, 0, time.UTC)

	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	err = s.manager.StoreWakuMessage(&message2)
	s.Require().NoError(err)

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err = torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 2)
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrentFromMessages() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 7 days
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// Partition of 7 days, this should create a single archive
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	// This message is outside of the startDate-endDate range and should not
	// be part of the archive
	message3 := buildMessage(endDate.Add(2*time.Hour), topic, []byte{3})

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromMessages(community.ID(), []*types.ReceivedMessage{&message1, &message2, &message3}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	_, err = os.Stat(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(torrentBackend.GetArchiveIndexFilePath(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(torrentBackend.GetTorrentFilePath(community.IDString()))
	s.Require().NoError(err)

	index, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	totalData, err := os.ReadFile(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().NoError(err)

	for _, metadata := range index.Archives {
		archive := &protobuf.WakuMessageArchive{}
		data := totalData[metadata.Offset : metadata.Offset+metadata.Size-metadata.Padding]

		err = proto.Unmarshal(data, archive)
		s.Require().NoError(err)

		s.Require().Len(archive.Messages, 2)
	}
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrentFromMessages_ShouldCreateMultipleArchives() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 3 weeks
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 21, 00, 00, 00, 0, time.UTC)
	// 7 days partition, this should create three archives
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	// We expect 2 archives to be created for startDate - endDate of each
	// 7 days of data. This message should end up in the second archive
	message3 := buildMessage(startDate.Add(8*24*time.Hour), topic, []byte{3})
	// This one should end up in the third archive
	message4 := buildMessage(startDate.Add(14*24*time.Hour), topic, []byte{4})

	torrentBackend := s.getTorrentBackend()
	_, err = torrentBackend.CreateHistoryArchiveFromMessages(community.ID(), []*types.ReceivedMessage{&message1, &message2, &message3, &message4}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 3)

	totalData, err := os.ReadFile(torrentBackend.GetArchiveDataFilePath(community.IDString()))
	s.Require().NoError(err)

	// First archive has 2 messages
	// Second archive has 1 message
	// Third archive has 1 message
	fromMap := map[uint64]int{
		uint64(startDate.Unix()):                    2,
		uint64(startDate.Add(partition).Unix()):     1,
		uint64(startDate.Add(partition * 2).Unix()): 1,
	}

	for _, metadata := range index.Archives {
		archive := &protobuf.WakuMessageArchive{}
		data := totalData[metadata.Offset : metadata.Offset+metadata.Size-metadata.Padding]

		err = proto.Unmarshal(data, archive)
		s.Require().NoError(err)
		s.Require().Len(archive.Messages, fromMap[metadata.Metadata.From])
	}
}

func (s *ArchiveManagerTorrentSuite) TestCreateHistoryArchiveTorrentFromMessages_ShouldAppendArchives() {
	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	// Time range of 1 week
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// 7 days partition, this should create one archive
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromMessages(community.ID(), []*types.ReceivedMessage{&message1}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	// Time range of next week
	startDate = time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	endDate = time.Date(2020, 1, 14, 00, 00, 00, 0, time.UTC)

	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})

	_, err = torrentBackend.CreateHistoryArchiveFromMessages(community.ID(), []*types.ReceivedMessage{&message2}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err = torrentBackend.LoadHistoryArchiveIndexFromFile(s.manager.GetIdentity(), community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 2)
}

func (s *ArchiveManagerTorrentSuite) TestSeedHistoryArchiveTorrent() {
	err := s.archiveService.Start()
	s.Require().NoError(err)
	defer s.archiveService.Stop() //nolint: errcheck

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	err = s.archiveService.SeedHistoryArchive(community.ID(), "")
	s.Require().NoError(err)
	s.Require().Equal(torrentBackend.GetTorrentTasksCount(), 1)

	torrent, ok := torrentBackend.GetTorrentForCommunity(community.IDString())
	defer torrent.Drop()

	s.Require().Equal(ok, true)
	s.Require().Equal(torrent.Seeding(), true)
}

func (s *ArchiveManagerTorrentSuite) TestUnseedHistoryArchiveTorrent() {
	err := s.archiveService.Start()
	s.Require().NoError(err)
	defer s.archiveService.Stop() //nolint: errcheck

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToContentTopic(messaging.ToContentTopic(chatID))
	topics := []types.ContentTopic{topic}

	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)

	torrentBackend := s.getTorrentBackend()

	_, err = torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	err = torrentBackend.SeedHistoryArchive(community.ID(), "")
	s.Require().NoError(err)
	s.Require().Equal(torrentBackend.GetTorrentTasksCount(), 1)

	s.archiveService.UnseedHistoryArchive(community.ID(), "")
	_, ok := torrentBackend.GetTorrentForCommunity(community.IDString())
	s.Require().Equal(ok, false)
}
