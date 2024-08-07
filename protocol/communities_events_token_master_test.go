package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestTokenMasterCommunityEventsSuite(t *testing.T) {
	suite.Run(t, new(TokenMasterCommunityEventsSuite))
}

type TokenMasterCommunityEventsSuite struct {
	EventSenderCommunityEventsSuiteBase
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
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalTokenMaster := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalTokenMaster := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})
	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalTokenMaster)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})
	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})

	testRejectMemberRequestToJoin(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	s.T().Skip("flaky test")

	additionalTokenMaster := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{additionalTokenMaster})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
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
	// event sender will receive privileged role during permission creation
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_NONE, []*Messenger{})

	// set up additional user (bob) that will send request to join
	bob := s.newMessenger(accountPassword, []string{bobAccountAddress})

	// set up additional user that will join to the community as TokenMaster
	newPrivilegedUser := s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})

	s.SetupAdditionalMessengers([]*Messenger{bob, newPrivilegedUser})
	testJoinedPrivilegedMemberReceiveRequestsToJoin(s, community, bob, newPrivilegedUser, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestReceiveRequestsToJoinWithRevealedAccountsAfterGettingTokenMasterRole() {
	// set up additional user (bob) that will send request to join
	bob := s.newMessenger(accountPassword, []string{bobAccountAddress})
	s.SetupAdditionalMessengers([]*Messenger{bob})
	testMemberReceiveRequestsToJoinAfterGettingNewRole(s, bob, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterAcceptsRequestToJoinAfterMemberLeave() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	s.SetupAdditionalMessengers([]*Messenger{user})
	testPrivilegedMemberAcceptsRequestToJoinAfterMemberLeave(s, community, user)
}

func (s *TokenMasterCommunityEventsSuite) TestTokenMasterBanMemberWithDeletingAllMessages() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_TOKEN_MASTER)
	testBanMemberWithDeletingAllMessages(s, community)
}
