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
	"github.com/status-im/status-go/protocol/requests"
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
	shh    types.Waku
	logger *zap.Logger
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

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.controlNode = s.newMessenger()
	s.ownerWithoutCommunityKey = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.controlNode.Start()
	s.Require().NoError(err)
	_, err = s.ownerWithoutCommunityKey.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TearDownTest() {
	s.Require().NoError(s.controlNode.Shutdown())
	s.Require().NoError(s.ownerWithoutCommunityKey.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	messenger, err := newCommunitiesTestMessenger(shh, privateKey, s.logger, nil, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
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
	testCreateEditDeleteBecomeMemberPermission(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCannotCreateBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)

	permissionRequest := createTestPermissionRequest(community)
	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

	response, err := s.ownerWithoutCommunityKey.CreateCommunityTokenPermission(permissionRequest)
	s.Require().Nil(response)
	s.Require().Error(err)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCannotEditBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	permissionRequest := createTestPermissionRequest(community)
	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

	// control node creates BECOME_ADMIN permission
	response, err := s.controlNode.CreateCommunityTokenPermission(permissionRequest)
	s.Require().NoError(err)

	var tokenPermissionID string
	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
		tokenPermissionID = id
	}
	s.Require().NotEqual(tokenPermissionID, "")

	ownerCommunity, err := s.controlNode.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(&s.Suite, ownerCommunity)

	// then, ensure event sender receives updated community
	_, err = WaitOnMessengerResponse(
		s.ownerWithoutCommunityKey,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive updated community",
	)
	s.Require().NoError(err)
	eventSenderCommunity, err := s.ownerWithoutCommunityKey.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(&s.Suite, eventSenderCommunity)

	permissionRequest.TokenCriteria[0].Symbol = "UPDATED"
	permissionRequest.TokenCriteria[0].Amount = "200"

	permissionEditRequest := &requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: *permissionRequest,
	}

	// then, event sender tries to edit permission
	response, err = s.ownerWithoutCommunityKey.EditCommunityTokenPermission(permissionEditRequest)
	s.Require().Error(err)
	s.Require().Nil(response)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerCannotDeleteBecomeAdminPermission() {

	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	permissionRequest := createTestPermissionRequest(community)
	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

	// control node creates BECOME_ADMIN permission
	response, err := s.controlNode.CreateCommunityTokenPermission(permissionRequest)
	s.Require().NoError(err)

	var tokenPermissionID string
	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
		tokenPermissionID = id
	}
	s.Require().NotEqual(tokenPermissionID, "")

	// then, ensure event sender receives updated community
	_, err = WaitOnMessengerResponse(
		s.ownerWithoutCommunityKey,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive updated community",
	)
	s.Require().NoError(err)
	eventSenderCommunity, err := s.ownerWithoutCommunityKey.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(&s.Suite, eventSenderCommunity)

	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermissionID,
	}

	// then event sender tries to delete BECOME_ADMIN permission which should fail
	response, err = s.ownerWithoutCommunityKey.DeleteCommunityTokenPermission(deleteTokenPermission)
	s.Require().Error(err)
	s.Require().Nil(response)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)

	// set up additional user that will send request to join
	user := s.newMessenger()
	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)

	// set up additional user that will send request to join
	user := s.newMessenger()
	testRejectMemberRequestToJoin(s, community, user)
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
	testEventSenderAddedCommunityToken(s, community)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestMemberReceiveOwnerEventsWhenControlNodeOffline() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER)
	testMemberReceiveEventsWhenControlNodeOffline(s, community)
}
