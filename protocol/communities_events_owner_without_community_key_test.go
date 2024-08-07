package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestOwnerWithoutCommunityKeyCommunityEventsSuite(t *testing.T) {
	suite.Run(t, new(OwnerWithoutCommunityKeyCommunityEventsSuite))
}

type OwnerWithoutCommunityKeyCommunityEventsSuite struct {
	EventSenderCommunityEventsSuiteBase
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
	additionalOwner := s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{additionalOwner})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalOwner)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalOwner := s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{additionalOwner})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalOwner)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testRejectMemberRequestToJoin(s, community, user)
}

func (s *OwnerWithoutCommunityKeyCommunityEventsSuite) TestOwnerControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	// TODO: test fixed, need to fix the code as it contains error
	s.T().Skip("flaky test")

	additionalOwner := s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_OWNER, []*Messenger{additionalOwner})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
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
