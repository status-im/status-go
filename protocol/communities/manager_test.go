package communities

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"math"
	"math/big"
	"os"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	userimages "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	v1 "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"

	"github.com/golang/protobuf/proto"
	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation

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

func (s *ManagerSuite) buildManager(ownerVerifier OwnerVerifier) *Manager {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.Require().NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(err)
	m, err := NewManager(key, "", db, nil, nil, nil, ownerVerifier, nil, &TimeSourceStub{}, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())
	return m
}

func (s *ManagerSuite) SetupTest() {
	m := s.buildManager(nil)
	SetValidateInterval(30 * time.Millisecond)
	s.manager = m
}

func intToBig(n int64) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(n))
}

func uintToDecBig(n uint64) *bigint.BigInt {
	return &bigint.BigInt{Int: big.NewInt(int64(n))}
}

func tokenBalance(tokenID uint64, balance uint64) thirdparty.TokenBalance {
	return thirdparty.TokenBalance{
		TokenID: uintToDecBig(tokenID),
		Balance: uintToDecBig(balance),
	}
}

func (s *ManagerSuite) getHistoryTasksCount() int {
	// sync.Map doesn't have a Len function, so we need to count manually
	count := 0
	s.manager.historyArchiveTasks.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

type testCollectiblesManager struct {
	response map[uint64]map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress
}

func (m *testCollectiblesManager) setResponse(chainID uint64, walletAddress gethcommon.Address, contractAddress gethcommon.Address, balances []thirdparty.TokenBalance) {
	if m.response == nil {
		m.response = make(map[uint64]map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
	}
	if m.response[chainID] == nil {
		m.response[chainID] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
	}
	if m.response[chainID][walletAddress] == nil {
		m.response[chainID][walletAddress] = make(thirdparty.TokenBalancesPerContractAddress)
	}

	m.response[chainID][walletAddress][contractAddress] = balances
}

func (m *testCollectiblesManager) FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	return m.response[uint64(chainID)][ownerAddress], nil
}

type testTokenManager struct {
	response map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big
}

func (m *testTokenManager) setResponse(chainID uint64, walletAddress, tokenAddress gethcommon.Address, balance int64) {

	if m.response == nil {
		m.response = make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	}

	if m.response[chainID] == nil {
		m.response[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	}

	if m.response[chainID][walletAddress] == nil {
		m.response[chainID][walletAddress] = make(map[gethcommon.Address]*hexutil.Big)
	}

	m.response[chainID][walletAddress][tokenAddress] = intToBig(balance)

}

func (m *testTokenManager) GetAllChainIDs() ([]uint64, error) {
	return []uint64{5}, nil
}

func (m *testTokenManager) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	return m.response, nil
}

func (m *testTokenManager) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) *token.Token {
	return nil
}

func (s *ManagerSuite) setupManagerForTokenPermissions() (*Manager, *testCollectiblesManager, *testTokenManager) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(err)

	cm := &testCollectiblesManager{}
	tm := &testTokenManager{}

	options := []ManagerOption{
		WithWalletConfig(&params.WalletConfig{
			OpenseaAPIKey: "some-key",
		}),
		WithCollectiblesManager(cm),
		WithTokenManager(tm),
	}

	m, err := NewManager(key, "", db, nil, nil, nil, nil, nil, &TimeSourceStub{}, nil, options...)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())

	return m, cm, tm
}

func (s *ManagerSuite) TestRetrieveTokens() {
	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			Amount:            "1.000000000000000000",
			Decimals:          decimals,
		},
	}

	var permissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
				TokenCriteria: tokenCriteria,
			},
		},
	}

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}
	// Set response to exactly the right one
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	resp, err := m.PermissionChecker.CheckPermissions(permissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().True(resp.Satisfied)

	// Set response to 0
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err = m.PermissionChecker.CheckPermissions(permissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().False(resp.Satisfied)
}

func (s *ManagerSuite) TestRetrieveCollectibles() {
	m, cm, _ := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"

	tokenID := uint64(10)
	var tokenBalances []thirdparty.TokenBalance

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			TokenIds:          []uint64{tokenID},
			Type:              protobuf.CommunityTokenType_ERC721,
		},
	}

	var permissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
				TokenCriteria: tokenCriteria,
			},
		},
	}

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	// Set response to exactly the right one
	tokenBalances = []thirdparty.TokenBalance{tokenBalance(tokenID, 1)}
	cm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), tokenBalances)
	resp, err := m.PermissionChecker.CheckPermissions(permissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().True(resp.Satisfied)

	// Set balances to 0
	tokenBalances = []thirdparty.TokenBalance{}
	cm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), tokenBalances)
	resp, err = m.PermissionChecker.CheckPermissions(permissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().False(resp.Satisfied)
}

func (s *ManagerSuite) TestCreateCommunity() {
	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "token membership description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(request, true)
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
	s.Require().True(community.IsControlNode())
	s.Require().True(proto.Equal(community.config.CommunityDescription, actualCommunity.config.CommunityDescription))
}

func (s *ManagerSuite) TestCreateCommunity_WithBanner() {
	// Generate test image bigger than BannerDim
	testImage := image.NewRGBA(image.Rect(0, 0, 20, 10))

	tmpTestFilePath := s.T().TempDir() + "/test.png"
	file, err := os.Create(tmpTestFilePath)
	s.NoError(err)
	defer file.Close()

	err = png.Encode(file, testImage)
	s.Require().NoError(err)

	request := &requests.CreateCommunity{
		Name:        "with_banner",
		Description: "community with banner ",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Banner: userimages.CroppedImage{
			ImagePath: tmpTestFilePath,
			X:         1,
			Y:         1,
			Width:     10,
			Height:    5,
		},
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	communities, err := s.manager.All()
	s.Require().NoError(err)
	// Consider status default community
	s.Require().Len(communities, 2)
	s.Require().Equal(len(community.config.CommunityDescription.Identity.Images), 1)
	testIdentityImage, isMapContainsKey := community.config.CommunityDescription.Identity.Images[userimages.BannerIdentityName]
	s.Require().True(isMapContainsKey)
	s.Require().Positive(len(testIdentityImage.Payload))
}

func (s *ManagerSuite) TestEditCommunity() {
	//create community
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	community, err := s.manager.CreateCommunity(createRequest, true)
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

func (s *ManagerSuite) TestGetControlledCommunitiesChatIDs() {
	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)
	s.Require().NotNil(community)

	controlledChatIDs, err := s.manager.GetOwnedCommunitiesChatIDs()

	s.Require().NoError(err)
	s.Require().Len(controlledChatIDs, 1)
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

	count := s.getHistoryTasksCount()
	s.Require().Equal(count, 1)

	// We wait another 5 seconds to ensure the first tick has kicked in
	time.Sleep(5 * time.Second)

	_, err = os.Stat(s.manager.torrentFile(community.IDString()))
	s.Require().Error(err)

	s.manager.StopHistoryArchiveTasksInterval(community.ID())
	s.manager.historyArchiveTasksWaitGroup.Wait()
	count = s.getHistoryTasksCount()
	s.Require().Equal(count, 0)
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

	count := s.getHistoryTasksCount()
	s.Require().Equal(count, 1)

	s.manager.StopHistoryArchiveTasksIntervals()

	count = s.getHistoryTasksCount()
	s.Require().Equal(count, 0)
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

	count := s.getHistoryTasksCount()
	s.Require().Equal(count, 1)

	errs := s.manager.StopTorrentClient()
	s.Require().Len(errs, 0)

	count = s.getHistoryTasksCount()
	s.Require().Equal(count, 0)
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	_, err = os.Stat(s.manager.archiveDataFile(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(s.manager.archiveIndexFile(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(s.manager.torrentFile(community.IDString()))
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	// Time range of next week
	startDate = time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	endDate = time.Date(2020, 1, 14, 00, 00, 00, 0, time.UTC)

	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})
	err = s.manager.StoreWakuMessage(&message2)
	s.Require().NoError(err)

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err = s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 2)
}

func (s *ManagerSuite) TestCreateHistoryArchiveTorrentFromMessages() {
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromMessages(community.ID(), []*types.Message{&message1, &message2, &message3}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	_, err = os.Stat(s.manager.archiveDataFile(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(s.manager.archiveIndexFile(community.IDString()))
	s.Require().NoError(err)
	_, err = os.Stat(s.manager.torrentFile(community.IDString()))
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
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

func (s *ManagerSuite) TestCreateHistoryArchiveTorrentFromMessages_ShouldCreateMultipleArchives() {
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromMessages(community.ID(), []*types.Message{&message1, &message2, &message3, &message4}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
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

func (s *ManagerSuite) TestCreateHistoryArchiveTorrentFromMessages_ShouldAppendArchives() {
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromMessages(community.ID(), []*types.Message{&message1}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err := s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
	s.Require().NoError(err)
	s.Require().Len(index.Archives, 1)

	// Time range of next week
	startDate = time.Date(2020, 1, 7, 00, 00, 00, 0, time.UTC)
	endDate = time.Date(2020, 1, 14, 00, 00, 00, 0, time.UTC)

	message2 := buildMessage(startDate.Add(2*time.Hour), topic, []byte{2})

	_, err = s.manager.CreateHistoryArchiveTorrentFromMessages(community.ID(), []*types.Message{&message2}, topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	index, err = s.manager.LoadHistoryArchiveIndexFromFile(s.manager.identity, community.ID())
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
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

	_, err = s.manager.CreateHistoryArchiveTorrentFromDB(community.ID(), topics, startDate, endDate, partition, false)
	s.Require().NoError(err)

	err = s.manager.SeedHistoryArchiveTorrent(community.ID())
	s.Require().NoError(err)
	s.Require().Len(s.manager.torrentTasks, 1)

	metaInfoHash := s.manager.torrentTasks[community.IDString()]

	s.manager.UnseedHistoryArchiveTorrent(community.ID())
	_, ok := s.manager.torrentClient.Torrent(metaInfoHash)
	s.Require().Equal(ok, false)
}

func (s *ManagerSuite) TestCheckChannelPermissions_NoPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var viewOnlyPermissions = make([]*CommunityTokenPermission, 0)
	var viewAndPostPermissions = make([]*CommunityTokenPermission, 0)

	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err := m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Both viewOnly and viewAndPost permissions are expected to be satisfied
	// because we call `checkChannelPermissions()` with no permissions to check
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().True(resp.ViewAndPostPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckChannelPermissions_ViewOnlyPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			Amount:            "1.000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewOnlyPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: tokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	var viewAndPostPermissions = make([]*CommunityTokenPermission, 0)

	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err := m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().False(resp.ViewOnlyPermissions.Satisfied)
	// if viewOnly permissions are not satisfied then viewAndPost
	// permissions shouldn't be satisfied either
	s.Require().False(resp.ViewAndPostPermissions.Satisfied)

	// Set response to exactly the right one
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	resp, err = m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().False(resp.ViewAndPostPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckChannelPermissions_ViewAndPostPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			Amount:            "1.000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewAndPostPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: tokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	var viewOnlyPermissions = make([]*CommunityTokenPermission, 0)

	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)
	resp, err := m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().False(resp.ViewAndPostPermissions.Satisfied)
	// viewOnly permissions are flagged as not satisfied because we have no viewOnly
	// permissions on this channel and the viewAndPost permission is not satisfied either
	s.Require().False(resp.ViewOnlyPermissions.Satisfied)

	// Set response to exactly the right one
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	resp, err = m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().True(resp.ViewAndPostPermissions.Satisfied)
	// if viewAndPost is satisfied then viewOnly should be automatically satisfied
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckChannelPermissions_ViewAndPostPermissionsCombination() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var viewOnlyTokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			Amount:            "1.000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewOnlyPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: viewOnlyTokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	testContractAddresses := make(map[uint64]string)
	testContractAddresses[chainID] = "0x123"

	// Set up token criteria that won't be satisfied
	var viewAndPostTokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: testContractAddresses,
			Symbol:            "TEST",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "TEST token",
			Amount:            "1.000000000000000000",
			Decimals:          decimals,
		},
	}

	var viewAndPostPermissions = []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Id:            "some-id",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: viewAndPostTokenCriteria,
				ChatIds:       []string{"test-channel-id", "test-channel-id-2"},
			},
		},
	}

	// Set response for viewOnly permissions
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))
	// Set resopnse for viewAndPost permissions
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(testContractAddresses[chainID]), 0)

	resp, err := m.checkChannelPermissions(viewOnlyPermissions, viewAndPostPermissions, accountChainIDsCombination, false)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// viewOnly permission should be satisfied, even though viewAndPost is not satisfied
	s.Require().True(resp.ViewOnlyPermissions.Satisfied)
	s.Require().False(resp.ViewAndPostPermissions.Satisfied)
}

func (s *ManagerSuite) TestCheckAllChannelsPermissions_EmptyPermissions() {

	m, _, _ := s.setupManagerForTokenPermissions()

	createRequest := &requests.CreateCommunity{
		Name:        "channel permission community",
		Description: "some description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := m.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// create community chats
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat1",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	changes, err := m.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)

	var chatID string
	for cid := range changes.ChatsAdded {
		chatID = community.IDString() + cid
	}

	response, err := m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Channels, 1)
	// we expect both, viewOnly and viewAndPost permissions to be satisfied
	// as there aren't any permissions on this channel
	s.Require().True(response.Channels[chatID].ViewOnlyPermissions.Satisfied)
	s.Require().True(response.Channels[chatID].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID].ViewOnlyPermissions.Permissions, 0)
	s.Require().Len(response.Channels[chatID].ViewAndPostPermissions.Permissions, 0)
}

func (s *ManagerSuite) TestCheckAllChannelsPermissions() {

	m, _, tm := s.setupManagerForTokenPermissions()

	var chatID1 string
	var chatID2 string

	// create community
	createRequest := &requests.CreateCommunity{
		Name:        "channel permission community",
		Description: "some description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := m.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// create first community chat
	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat1",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	changes, err := m.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)

	for chatID := range changes.ChatsAdded {
		chatID1 = community.IDString() + chatID
	}

	// create second community chat
	chat = &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat2",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	changes, err = m.CreateChat(community.ID(), chat, true, "")
	s.Require().NoError(err)

	for chatID := range changes.ChatsAdded {
		chatID2 = community.IDString() + chatID
	}

	var chainID uint64 = 5
	contractAddresses := make(map[uint64]string)
	contractAddresses[chainID] = "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"
	var decimals uint64 = 18

	accountChainIDsCombination := []*AccountChainIDsCombination{
		&AccountChainIDsCombination{
			Address:  gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
			ChainIDs: []uint64{chainID},
		},
	}

	var tokenCriteria = []*protobuf.TokenCriteria{
		&protobuf.TokenCriteria{
			ContractAddresses: contractAddresses,
			Symbol:            "STT",
			Type:              protobuf.CommunityTokenType_ERC20,
			Name:              "Status Test Token",
			Amount:            "1.000000000000000000",
			Decimals:          decimals,
		},
	}

	// create view only permission
	viewOnlyPermission := &requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
		TokenCriteria: tokenCriteria,
		ChatIds:       []string{chatID1, chatID2},
	}

	_, changes, err = m.CreateCommunityTokenPermission(viewOnlyPermission)
	s.Require().NoError(err)

	var viewOnlyPermissionID string
	for permissionID := range changes.TokenPermissionsAdded {
		viewOnlyPermissionID = permissionID
	}

	response, err := m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// we've added to chats to the community, so there should be 2 items
	s.Require().Len(response.Channels, 2)

	// viewOnly permissions should not be satisfied because the account doesn't
	// have the necessary funds

	// channel1
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	// channel2
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	// viewAndPost permissions are flagged as not satisfied either because
	// viewOnly permission is not satisfied and there are no viewAndPost permissions

	// channel1
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 0)

	// channel2
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 0)

	// now change balance such that viewOnly permission should be satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// viewOnly permissions should be satisfied for both channels while
	// viewAndPost permissions should not be satisfied (as there aren't any)

	// channel1
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 0)

	// channel2
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 0)

	// next, create viewAndPost permission
	// create view only permission
	viewAndPostPermission := &requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL,
		TokenCriteria: tokenCriteria,
		ChatIds:       []string{chatID1, chatID2},
	}

	_, changes, err = m.CreateCommunityTokenPermission(viewAndPostPermission)
	s.Require().NoError(err)

	var viewAndPostPermissionID string
	for permissionID := range changes.TokenPermissionsAdded {
		viewAndPostPermissionID = permissionID
	}

	// now change balance such that viewAndPost permission is not satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// Both, viewOnly and viewAndPost permissions exist on channel1 and channel2
	// but shouldn't be satisfied

	// channel1
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// channel2
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// now change balance such that both, viewOnly and viewAndPost permission, are satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), int64(1*math.Pow(10, float64(decimals))))

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// Both, viewOnly and viewAndPost permissions exist on channel1 and channel2
	// and are satisfied

	// channel1
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// channel2
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Permissions[viewOnlyPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	// next, delete viewOnly permission so we can check the viewAndPost permission-only case
	deleteViewOnlyPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: viewOnlyPermissionID,
	}
	_, _, err = m.DeleteCommunityTokenPermission(deleteViewOnlyPermission)
	s.Require().NoError(err)

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// Both, channel1 and channel2 now have viewAndPost only permissions that should
	// be satisfied, there's no viewOnly permission anymore the response should mark it
	// as satisfied as well

	// channel1
	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 0)

	// channel2
	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().True(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().True(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 0)

	// now change balance such that viewAndPost permission is no longer satisfied
	tm.setResponse(chainID, accountChainIDsCombination[0].Address, gethcommon.HexToAddress(contractAddresses[chainID]), 0)

	response, err = m.CheckAllChannelsPermissions(community.ID(), []gethcommon.Address{
		gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32"),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Channels, 2)

	// because viewAndPost permission is not satisfied and there are no viewOnly permissions
	// on the channels, the response should mark the viewOnly permissions as not satisfied as well

	// channel1
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID1].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID1].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID1].ViewOnlyPermissions.Permissions, 0)

	// channel2
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions, 1)
	s.Require().Len(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria, 1)
	s.Require().False(response.Channels[chatID2].ViewAndPostPermissions.Permissions[viewAndPostPermissionID].Criteria[0])

	s.Require().False(response.Channels[chatID2].ViewOnlyPermissions.Satisfied)
	s.Require().Len(response.Channels[chatID2].ViewOnlyPermissions.Permissions, 0)
}

func buildTorrentConfig() params.TorrentConfig {
	torrentConfig := params.TorrentConfig{
		Enabled:    true,
		DataDir:    os.TempDir() + "/archivedata",
		TorrentDir: os.TempDir() + "/torrents",
		Port:       0,
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

type testOwnerVerifier struct {
	called    int
	ownersMap map[string]string
}

func (t *testOwnerVerifier) SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error) {
	t.called++
	return t.ownersMap[communityID], nil
}

func (s *ManagerSuite) TestCommunityQueue() {

	owner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m := s.buildManager(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// set verifier public key
	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = common.PubkeyToHex(&owner.PublicKey)

	description := community.config.CommunityDescription

	// safety check
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	// set up permissions
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{
					2: "some-address",
				},
			},
		}}

	// Should have now a token owner
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v1.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, owner)
	s.Require().NoError(err)

	// Create a signer, that is not the owner
	notTheOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	subscription := m.Subscribe()

	response, err := m.HandleCommunityDescriptionMessage(&notTheOwner.PublicKey, description, payload, nil, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	published := false

	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("no subscription")
		}
	}

	// Check it's not called multiple times
	s.Require().Equal(1, verifier.called)
	// Cleans up the communities to validate
	communitiesToValidate, err := m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().Empty(communitiesToValidate)
}

// 1) We create a community
// 2) We have 2 owners, but only new owner is returned by the contract
// 3) We receive the old owner community first
// 4) We receive the new owner community second
// 5) We start the queue
// 6) We should only process 4, and ignore anything else if that is successful, as that's the most recent

func (s *ManagerSuite) TestCommunityQueueMultipleDifferentSigners() {

	newOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	oldOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m := s.buildManager(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// set verifier public key
	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = common.PubkeyToHex(&newOwner.PublicKey)

	description := community.config.CommunityDescription

	// safety check
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	// set up permissions
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{
					2: "some-address",
				},
			},
		}}

	// Should have now a token owner
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	// We nil owner verifier so that messages won't be processed
	m.ownerVerifier = nil

	// Send message from old owner first

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v1.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, oldOwner)
	s.Require().NoError(err)

	subscription := m.Subscribe()

	response, err := m.HandleCommunityDescriptionMessage(&oldOwner.PublicKey, description, payload, nil, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	// Send message from new owner now

	community.config.CommunityDescription.Clock++

	clock2 := community.config.CommunityDescription.Clock

	payload, err = community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v1.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, newOwner)
	s.Require().NoError(err)

	response, err = m.HandleCommunityDescriptionMessage(&newOwner.PublicKey, description, payload, nil, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	count, err := m.persistence.getCommunitiesToValidateCount()
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	communitiesToValidate, err := m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().NotNil(communitiesToValidate)
	s.Require().NotNil(communitiesToValidate[community.IDString()])
	s.Require().Len(communitiesToValidate[community.IDString()], 2)

	// We set owner verifier so that we start processing the queue
	m.ownerVerifier = verifier

	published := false

	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("no subscription")
		}
	}

	// Check it's not called multiple times, since we should be checking newest first
	s.Require().Equal(1, verifier.called)
	// Cleans up the communities to validate
	communitiesToValidate, err = m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().Empty(communitiesToValidate)

	// Check clock of community is of the last community description
	fetchedCommunity, err := m.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(clock2, fetchedCommunity.config.CommunityDescription.Clock)

}

// 1) We create a community
// 2) We have 2 owners, but only old owner is returned by the contract
// 3) We receive the old owner community first
// 4) We receive the new owner community second (that could be a malicious user)
// 5) We start the queue
// 6) We should process both, but ignore the last community description

func (s *ManagerSuite) TestCommunityQueueMultipleDifferentSignersIgnoreIfNotReturned() {

	newOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	oldOwner, err := crypto.GenerateKey()
	s.Require().NoError(err)

	verifier := &testOwnerVerifier{}
	m := s.buildManager(verifier)

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	s.Require().NoError(err)

	// set verifier public key
	verifier.ownersMap = make(map[string]string)
	verifier.ownersMap[community.IDString()] = common.PubkeyToHex(&oldOwner.PublicKey)

	description := community.config.CommunityDescription

	// safety check
	s.Require().Equal(uint64(0), CommunityDescriptionTokenOwnerChainID(description))

	// set up permissions
	description.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
	description.TokenPermissions["some-id"] = &protobuf.CommunityTokenPermission{
		Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
		Id:   "some-token-id",
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{
					2: "some-address",
				},
			},
		}}

	// Should have now a token owner
	s.Require().Equal(uint64(2), CommunityDescriptionTokenOwnerChainID(description))

	// We nil owner verifier so that messages won't be processed
	m.ownerVerifier = nil

	clock1 := community.config.CommunityDescription.Clock
	// Send message from old owner first

	payload, err := community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v1.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, oldOwner)
	s.Require().NoError(err)

	subscription := m.Subscribe()

	response, err := m.HandleCommunityDescriptionMessage(&oldOwner.PublicKey, description, payload, nil, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	// Send message from new owner now

	community.config.CommunityDescription.Clock++

	payload, err = community.MarshaledDescription()
	s.Require().NoError(err)

	payload, err = v1.WrapMessageV1(payload, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, newOwner)
	s.Require().NoError(err)

	response, err = m.HandleCommunityDescriptionMessage(&newOwner.PublicKey, description, payload, nil, nil)
	s.Require().NoError(err)

	// No response, as it should be queued
	s.Require().Nil(response)

	count, err := m.persistence.getCommunitiesToValidateCount()
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	communitiesToValidate, err := m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().NotNil(communitiesToValidate)
	s.Require().NotNil(communitiesToValidate[community.IDString()])
	s.Require().Len(communitiesToValidate[community.IDString()], 2)

	// We set owner verifier so that we start processing the queue
	m.ownerVerifier = verifier

	published := false

	for !published {
		select {
		case event := <-subscription:
			if event.TokenCommunityValidated == nil {
				continue
			}
			published = true
		case <-time.After(2 * time.Second):
			s.FailNow("no subscription")
		}
	}

	// Check it's not called multiple times, since we should be checking newest first
	s.Require().Equal(2, verifier.called)
	// Cleans up the communities to validate
	communitiesToValidate, err = m.persistence.getCommunitiesToValidate()
	s.Require().NoError(err)
	s.Require().Empty(communitiesToValidate)

	// Check clock of community is of the first community description
	fetchedCommunity, err := m.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(clock1, fetchedCommunity.config.CommunityDescription.Clock)
}
