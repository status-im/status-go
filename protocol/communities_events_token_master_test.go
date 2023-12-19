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

func TestTokenMasterCommunityEventsSuite(t *testing.T) {
	suite.Run(t, new(TokenMasterCommunityEventsSuite))
}

type TokenMasterCommunityEventsSuite struct {
	suite.Suite
	controlNode *Messenger
	tokenMaster *Messenger
	alice       *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh                     types.Waku
	logger                  *zap.Logger
	mockedBalances          map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
	collectiblesServiceMock *CollectiblesServiceMock
}

func (s *TokenMasterCommunityEventsSuite) GetControlNode() *Messenger {
	return s.controlNode
}

func (s *TokenMasterCommunityEventsSuite) GetEventSender() *Messenger {
	return s.tokenMaster
}

func (s *TokenMasterCommunityEventsSuite) GetMember() *Messenger {
	return s.alice
}

func (s *TokenMasterCommunityEventsSuite) GetSuite() *suite.Suite {
	return &s.Suite
}

func (s *TokenMasterCommunityEventsSuite) GetCollectiblesServiceMock() *CollectiblesServiceMock {
	return s.collectiblesServiceMock
}

func (s *TokenMasterCommunityEventsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.controlNode = s.newMessenger("", []string{})
	s.tokenMaster = s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})
	s.alice = s.newMessenger(accountPassword, []string{aliceAccountAddress})
	_, err := s.controlNode.Start()
	s.Require().NoError(err)
	_, err = s.tokenMaster.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.mockedBalances = createMockedWalletBalance(&s.Suite)
}

func (s *TokenMasterCommunityEventsSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.controlNode)
	TearDownMessenger(&s.Suite, s.tokenMaster)
	TearDownMessenger(&s.Suite, s.alice)
	_ = s.logger.Sync()
}

func (s *TokenMasterCommunityEventsSuite) newMessenger(password string, walletAddresses []string) *Messenger {
	return newMessenger(&s.Suite, s.shh, s.logger, password, walletAddresses, &s.mockedBalances, s.collectiblesServiceMock)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterEditCommunityDescription() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	editCommunityDescription(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCreateEditDeleteChannels() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testCreateEditDeleteChannels(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCreateEditDeleteBecomeMemberPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testCreateEditDeleteBecomeMemberPermission(s, community, protobuf.CommunityTokenPermission_BECOME_MEMBER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotCreateBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotCreatePrivilegedCommunityPermission(s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotCreateBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotCreatePrivilegedCommunityPermission(s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotEditBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotEditPrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotEditBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotEditPrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotDeleteBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotDeletePrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotDeleteBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotDeletePrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalTokenMaster := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})
	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalTokenMaster := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})
	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testRejectMemberRequestToJoin(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	additionalTokenMaster := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testControlNodeHandlesMultipleEventSenderRequestToJoinDecisions(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCreateEditDeleteCategories() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testCreateEditDeleteCategories(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterReorderChannelsAndCategories() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testReorderChannelsAndCategories(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterKickOwnerWithoutCommunityKey() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderKickTheSameRole(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterKickControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderKickControlNode(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterKickMember() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	kickMember(s, community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterBanOwnerWithoutCommunityKey() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testOwnerBanTheSameRole(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterBanControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testOwnerBanControlNode(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterBanUnbanMember() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testBanUnbanMember(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterDeleteAnyMessageInTheCommunity() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testDeleteAnyMessageInTheCommunity(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterPinMessage() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderPinMessage(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAddCommunityToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testAddAndSyncTokenFromEventSenderByControlNode(s, community, token.CommunityLevel)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAddTokenMasterAndOwnerToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderAddTokenMasterAndOwnerToken(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterReceiveOwnerTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testAddAndSyncOwnerTokenFromControlNode(s, community, token.OwnerLevel)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterReceiveTokenMasterTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testAddAndSyncTokenFromControlNode(s, community, token.MasterLevel)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterReceiveCommunityTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testAddAndSyncTokenFromControlNode(s, community, token.CommunityLevel)
}

func (s *TokenMasterCommunityEventsSuite) TestMemberReceiveTokenMasterEventsWhenControlNodeOffline() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testMemberReceiveEventsWhenControlNodeOffline(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestJoinedTokenMasterReceiveRequestsToJoinWithRevealedAccounts() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})

	// set up additional user (bob) that will send request to join
	bob := s.newMessenger(accountPassword, []string{bobAccountAddress})

	// set up additional user that will join to the community as TokenMaster
	newPrivilegedUser := s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})

	testJoinedPrivilegedMemberReceiveRequestsToJoin(s, community, bob, newPrivilegedUser, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestReceiveRequestsToJoinWithRevealedAccountsAfterGettingTokenMasterRole() {
	// set up additional user (bob) that will send request to join
	bob := s.newMessenger(accountPassword, []string{bobAccountAddress})
	testMemberReceiveRequestsToJoinAfterGettingNewRole(s, bob, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptsRequestToJoinAfterMemberLeave() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testPrivilegedMemberAcceptsRequestToJoinAfterMemberLeave(s, community, user)
}
