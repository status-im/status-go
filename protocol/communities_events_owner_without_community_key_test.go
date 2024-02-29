package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestOwnerWithoutCommunityKeyCommunityEventsSuite(t *testing.T) {
	suite.Run(t, new(OwnerWithoutCommunityKeyCommunityEventsSuite))
}

type OwnerWithoutCommunityKeyCommunityEventsSuite struct {
	suite.Suite
	controlNode              *Messenger
	ownerWithoutCommunityKey *Messenger
	alice                    *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh                     types.Waku
	logger                  *zap.Logger
	mockedBalances          map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
	collectiblesServiceMock *CollectiblesServiceMock

	additionalEventSenders []*Messenger
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) GetControlNode() *Messenger {
	return s.controlNode
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) GetEventSender() *Messenger {
	return s.ownerWithoutCommunityKey
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) GetMember() *Messenger {
	return s.alice
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) GetSuite() *suite.Suite {
	return &s.Suite
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) GetCollectiblesServiceMock() *CollectiblesServiceMock {
	return s.collectiblesServiceMock
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.controlNode = s.newMessenger("", []string{})
	s.ownerWithoutCommunityKey = s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	s.alice = s.newMessenger("", []string{})
	_, err := s.controlNode.Start()
	s.Require().NoError(err)
	_, err = s.ownerWithoutCommunityKey.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.mockedBalances = createMockedWalletBalance(&s.Suite)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.controlNode)
	TearDownMessenger(&s.Suite, s.ownerWithoutCommunityKey)
	TearDownMessenger(&s.Suite, s.alice)

	for _, m := range s.additionalEventSenders {
		TearDownMessenger(&s.Suite, m)
	}
	s.additionalEventSenders = nil

	_ = s.logger.Sync()
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) SetupAdditionalMessengers(messengers []*Messenger) {
	for _, m := range messengers {
		s.additionalEventSenders = append(s.additionalEventSenders, m)
		_, err := m.Start()
		s.Require().NoError(err)
	}
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) newMessenger(password string, walletAddresses []string) *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, s.shh, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger: s.logger,
		},
		password:            password,
		walletAddresses:     walletAddresses,
		mockedBalances:      &s.mockedBalances,
		collectiblesService: s.collectiblesServiceMock,
	})
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerEditCommunityDescription() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	editCommunityDescription(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCreateEditDeleteChannels() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)

	testCreateEditDeleteChannels(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCreateEditDeleteBecomeMemberPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testCreateEditDeleteBecomeMemberPermission(s, community, protobuf.CommunityTokenPermission_BECOME_MEMBER)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCreateEditDeleteBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testCreateEditDeleteBecomeMemberPermission(s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCreateEditDeleteBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testCreateEditDeleteBecomeMemberPermission(s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalOwner := s.newMessenger("", []string{})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{additionalOwner})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalOwner)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalOwner := s.newMessenger("", []string{})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{additionalOwner})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalOwner)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testRejectMemberRequestToJoin(s, community, user)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	s.T().Skip("flaky test")

	additionalOwner := s.newMessenger("", []string{})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{additionalOwner})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testControlNodeHandlesMultipleEventSenderRequestToJoinDecisions(s, community, user, additionalOwner)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCreateEditDeleteCategories() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testCreateEditDeleteCategories(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerReorderChannelsAndCategories() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testReorderChannelsAndCategories(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerKickOwnerWithoutCommunityKey() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testEventSenderKickTheSameRole(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerKickControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testEventSenderKickControlNode(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerKickMember() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	kickMember(s, community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerBanOwnerWithoutCommunityKey() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testOwnerBanTheSameRole(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerBanControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testOwnerBanControlNode(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerBanUnbanMember() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testBanUnbanMember(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerDeleteAnyMessageInTheCommunity() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testDeleteAnyMessageInTheCommunity(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerPinMessage() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testEventSenderPinMessage(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAddCommunityToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testAddAndSyncTokenFromEventSenderByControlNode(s, community, token.CommunityLevel)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAddOwnerToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testAddAndSyncTokenFromEventSenderByControlNode(s, community, token.OwnerLevel)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAddTokenMasterToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testAddAndSyncTokenFromEventSenderByControlNode(s, community, token.MasterLevel)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerReceiveOwnerTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testAddAndSyncOwnerTokenFromControlNode(s, community, token.OwnerLevel)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerReceiveTokenMasterTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testAddAndSyncTokenFromControlNode(s, community, token.MasterLevel)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerReceiveCommunityTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testAddAndSyncTokenFromControlNode(s, community, token.CommunityLevel)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestMemberReceiveOwnerEventsWhenControlNodeOffline() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testMemberReceiveEventsWhenControlNodeOffline(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerBanMemberWithDeletingAllMessages() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testBanMemberWithDeletingAllMessages(s, community)
}
