package protocol

import (
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
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
	shh    types.Waku
	logger *zap.Logger
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

func (s *TokenMasterCommunityEventsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.controlNode = s.newMessenger()
	s.tokenMaster = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.controlNode.Start()
	s.Require().NoError(err)
	_, err = s.tokenMaster.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *TokenMasterCommunityEventsSuite) TearDownTest() {
	s.Require().NoError(s.controlNode.Shutdown())
	s.Require().NoError(s.tokenMaster.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *TokenMasterCommunityEventsSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	messenger, err := newCommunitiesTestMessenger(shh, privateKey, s.logger, nil, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *TokenMasterCommunityEventsSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
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
	testCreateEditDeleteBecomeMemberPermission(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotCreateBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)

	permissionRequest := createTestPermissionRequest(community)
	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

	response, err := s.tokenMaster.CreateCommunityTokenPermission(permissionRequest)
	s.Require().Nil(response)
	s.Require().Error(err)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotEditBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotEditBecomeAdminPermission(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterCannotDeleteBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testEventSenderCannotDeleteBecomeAdminPermission(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptMemberRequestToJoinNotConfirmedByControlNode() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger()
	testAcceptMemberRequestToJoinNotConfirmedByControlNode(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger()
	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalTokenMaster := s.newMessenger()
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})
	// set up additional user that will send request to join
	user := s.newMessenger()
	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalTokenMaster := s.newMessenger()
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})
	// set up additional user that will send request to join
	user := s.newMessenger()
	testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoinNotConfirmedByControlNode() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger()
	testRejectMemberRequestToJoinNotConfirmedByControlNode(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger()
	testRejectMemberRequestToJoin(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRequestToJoinStateCannotBeOverridden() {
	additionalTokenMaster := s.newMessenger()
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})

	// set up additional user that will send request to join
	user := s.newMessenger()
	testEventSenderCannotOverrideRequestToJoinState(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	additionalTokenMaster := s.newMessenger()
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})

	// set up additional user that will send request to join
	user := s.newMessenger()
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
	testEventSenderAddedCommunityToken(s, community)
}

func (s *TokenMasterCommunityEventsSuite) TestMemberReceiveTokenMasterEventsWhenControlNodeOffline() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testMemberReceiveEventsWhenControlNodeOffline(s, community)
}
