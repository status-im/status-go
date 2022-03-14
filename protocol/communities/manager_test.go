package communities

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"

	"github.com/golang/protobuf/proto"
	_ "github.com/mutecomm/go-sqlcipher" // require go-sqlcipher that overrides default implementation

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}

type ManagerSuite struct {
	suite.Suite
	manager *Manager
}

func (s *ManagerSuite) SetupTest() {
	dbPath, err := ioutil.TempFile("", "")
	s.NoError(err, "creating temp file for db")
	db, err := appdatabase.InitializeDB(dbPath.Name(), "")
	s.NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(err)
	m, err := NewManager(&key.PublicKey, db, nil, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())
	s.manager = m
}

func (s *ManagerSuite) TestCreateCommunity() {

	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	community, err := s.manager.CreateCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	communities, err := s.manager.All()
	s.Require().NoError(err)
	// Consider status default community
	s.Require().Len(communities, 2)

	actualCommunity := communities[0]
	if bytes.Equal(community.ID(), communities[1].ID()) {
		actualCommunity = communities[1]
	}

	s.Require().Equal(community.ID(), actualCommunity.ID())
	s.Require().Equal(community.PrivateKey(), actualCommunity.PrivateKey())
	s.Require().True(proto.Equal(community.config.CommunityDescription, actualCommunity.config.CommunityDescription))
}

func (s *ManagerSuite) TestEditCommunity() {
	//create community
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	community, err := s.manager.CreateCommunity(createRequest)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	update := &requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        "statusEdited",
			Description: "status community description edited",
		},
	}

	updatedCommunity, err := s.manager.EditCommunity(update)
	s.Require().NoError(err)
	s.Require().NotNil(updatedCommunity)

	//ensure updated community successfully stored
	communities, err := s.manager.All()
	s.Require().NoError(err)
	// Consider status default community
	s.Require().Len(communities, 2)

	storedCommunity := communities[0]
	if bytes.Equal(community.ID(), communities[1].ID()) {
		storedCommunity = communities[1]
	}

	s.Require().Equal(storedCommunity.ID(), updatedCommunity.ID())
	s.Require().Equal(storedCommunity.PrivateKey(), updatedCommunity.PrivateKey())
	s.Require().Equal(storedCommunity.config.CommunityDescription.Identity.DisplayName, update.CreateCommunity.Name)
	s.Require().Equal(storedCommunity.config.CommunityDescription.Identity.Description, update.CreateCommunity.Description)
}

func (s *ManagerSuite) TestGetAdminCommuniesChatIDs() {

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)
	s.Require().NotNil(community)

	adminChatIDs, err := s.manager.GetAdminCommunitiesChatIDs()
	s.Require().NoError(err)
	s.Require().Len(adminChatIDs, 1)
}

func (s *ManagerSuite) TestStartAndStopTorrentClient() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	err := s.manager.StartTorrentClient()
	s.Require().NoError(err)
	s.Require().NotNil(s.manager.torrentClient)
	defer s.manager.StopTorrentClient()

	_, err = os.Stat(torrentConfig.DataDir)
	s.Require().NoError(err)
	s.Require().Equal(s.manager.TorrentClientStarted(), true)
}

func (s *ManagerSuite) TestStartHistoryArchiveTasksInterval() {

	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	err := s.manager.StartTorrentClient()
	s.Require().NoError(err)
	defer s.manager.StopTorrentClient()

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	interval := 10 * time.Second
	go s.manager.StartHistoryArchiveTasksInterval(community, interval)
	// Due to async exec we need to wait a bit until we check
	// the task count.
	time.Sleep(5 * time.Second)
	s.Require().Len(s.manager.historyArchiveTasks, 1)

	// We wait another 5 seconds to ensure the first tick has kicked in
	time.Sleep(5 * time.Second)

	_, err = os.Stat(s.manager.torrentFile(community.IDString()))
	s.Require().Error(err)

	s.manager.StopHistoryArchiveTasksInterval(community.ID())
	s.manager.historyArchiveTasksWaitGroup.Wait()
	s.Require().Len(s.manager.historyArchiveTasks, 0)
}

func (s *ManagerSuite) TestStopHistoryArchiveTasksIntervals() {

	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	err := s.manager.StartTorrentClient()
	s.Require().NoError(err)
	defer s.manager.StopTorrentClient()

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	interval := 10 * time.Second
	go s.manager.StartHistoryArchiveTasksInterval(community, interval)

	time.Sleep(2 * time.Second)
	s.Require().Len(s.manager.historyArchiveTasks, 1)
	s.manager.StopHistoryArchiveTasksIntervals()
	s.Require().Len(s.manager.historyArchiveTasks, 0)
}

func (s *ManagerSuite) TestStopTorrentClient_ShouldStopHistoryArchiveTasks() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	err := s.manager.StartTorrentClient()
	s.Require().NoError(err)
	defer s.manager.StopTorrentClient()

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	interval := 10 * time.Second
	go s.manager.StartHistoryArchiveTasksInterval(community, interval)
	// Due to async exec we need to wait a bit until we check
	// the task count.
	time.Sleep(2 * time.Second)
	s.Require().Len(s.manager.historyArchiveTasks, 1)

	errs := s.manager.StopTorrentClient()
	s.Require().Len(errs, 0)
	s.Require().Len(s.manager.historyArchiveTasks, 0)
}

func (s *ManagerSuite) TestCreateHistoryArchiveTorrent_WithoutMessages() {

	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToTopic(transport.ToTopic(chatID))
	topics := []types.TopicType{topic}

	// Time range of 7 days
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// Partition of 7 days
	partition := 7 * 24 * time.Hour

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	// There are no waku messages in the database so we don't expect
	// any archives to be created
	_, err = os.Stat(s.manager.archiveDataFile(community.IDString()))
	s.Require().Error(err)
	_, err = os.Stat(s.manager.archiveIndexFile(community.IDString()))
	s.Require().Error(err)
	_, err = os.Stat(s.manager.torrentFile(community.IDString()))
	s.Require().Error(err)
}

func (s *ManagerSuite) TestCreateHistoryArchiveTorrent_ShouldCreateArchive() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToTopic(transport.ToTopic(chatID))
	topics := []types.TopicType{topic}

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

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	_, err = os.Stat(s.manager.archiveDataFile(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(s.manager.archiveIndexFile(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(s.manager.torrentFile(community.IDString()))
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	totalData, err := os.ReadFile(s.manager.archiveDataFile(community.IDString()))
	s.Require().NoError(err)

	for _, metadata := range index.Archives {
		archive := &protobuf.WakuMessageArchive{}
		data := totalData[metadata.Offset : metadata.Offset+metadata.Size-metadata.Padding]

		err = proto.Unmarshal(data, archive)
		s.Require().NoError(err)

		s.Require().Len(archive.Messages, 2)
	}
}

func (s *ManagerSuite) TestCreateHistoryArchiveTorrent_ShouldCreateMultipleArchives() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToTopic(transport.ToTopic(chatID))
	topics := []types.TopicType{topic}

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

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 3)

	totalData, err := os.ReadFile(s.manager.archiveDataFile(community.IDString()))
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

func (s *ManagerSuite) TestCreateHistoryArchiveTorrent_ShouldAppendArchives() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToTopic(transport.ToTopic(chatID))
	topics := []types.TopicType{topic}

	// Time range of 1 week
	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	// 7 days partition, this should create one archive
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	// Time range of next week
	startDate = time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	endDate = time.Date(2020, 1, 14, 00, 00, 00, 0, time.UTC)

	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	err = s.manager.StoreWakuMessage(&message2)
	s.Require().NoError(err)

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	index, err = s.manager.LoadHistoryArchiveIndexFromFile(community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 2)
}

func (s *ManagerSuite) TestSeedHistoryArchiveTorrent() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	err := s.manager.StartTorrentClient()
	s.Require().NoError(err)
	defer s.manager.StopTorrentClient()

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToTopic(transport.ToTopic(chatID))
	topics := []types.TopicType{topic}

	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	err = s.manager.SeedHistoryArchiveTorrent(community.ID())
	s.Require().NoError(err)
	s.Require().Len(s.manager.torrentTasks, 1)

	metaInfoHash := s.manager.torrentTasks[community.IDString()]
	torrent, ok := s.manager.torrentClient.Torrent(metaInfoHash)
	defer torrent.Drop()

	s.Require().Equal(ok, true)
	s.Require().Equal(torrent.Seeding(), true)
}

func (s *ManagerSuite) TestUnseedHistoryArchiveTorrent() {
	torrentConfig := buildTorrentConfig()
	s.manager.SetTorrentConfig(&torrentConfig)

	err := s.manager.StartTorrentClient()
	s.Require().NoError(err)
	defer s.manager.StopTorrentClient()

	community, chatID, err := s.buildCommunityWithChat()
	s.Require().NoError(err)

	topic := types.BytesToTopic(transport.ToTopic(chatID))
	topics := []types.TopicType{topic}

	startDate := time.Date(2020, 1, 1, 00, 00, 00, 0, time.UTC)
	endDate := time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	partition := 7 * 24 * time.Hour

	message1 := buildMessage(startDate.Add(1*time.Hour), topic, []byte{1})
	err = s.manager.StoreWakuMessage(&message1)
	s.Require().NoError(err)

	_, err = s.manager.CreateHistoryArchiveTorrent(community.ID(), topics, startDate, endDate, partition)
	s.Require().NoError(err)

	err = s.manager.SeedHistoryArchiveTorrent(community.ID())
	s.Require().NoError(err)
	s.Require().Len(s.manager.torrentTasks, 1)

	metaInfoHash := s.manager.torrentTasks[community.IDString()]

	s.manager.UnseedHistoryArchiveTorrent(community.ID())
	_, ok := s.manager.torrentClient.Torrent(metaInfoHash)
	s.Require().Equal(ok, false)
}

func buildTorrentConfig() params.TorrentConfig {
	torrentConfig := params.TorrentConfig{
		Enabled:    true,
		DataDir:    os.TempDir() + "/archivedata",
		TorrentDir: os.TempDir() + "/torrents",
		Port:       9999,
	}
	return torrentConfig
}

func buildMessage(timestamp time.Time, topic types.TopicType, hash []byte) types.Message {
	message := types.Message{
		Sig:       []byte{1},
		Timestamp: uint32(timestamp.Unix()),
		Topic:     topic,
		Payload:   []byte{1},
		Padding:   []byte{1},
		Hash:      hash,
	}
	return message
}

func (s *ManagerSuite) buildCommunityWithChat() (*Community, string, error) {
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}
	community, err := s.manager.CreateCommunity(createRequest)
	if err != nil {
		return nil, "", err
	}
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "added-chat",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}
	_, changes, err := s.manager.CreateChat(community.ID(), chat)
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
