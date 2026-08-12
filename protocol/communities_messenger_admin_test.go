package protocol

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet/bigint"
)

func TestAdminCommunityEventsSuite(t *testing.T) {
	suite.Run(t, new(AdminCommunityEventsSuite))
}

type EventSenderCommunityEventsSuiteBase struct {
	CommunitiesMessengerTestSuiteBase
	owner       *Messenger
	eventSender *Messenger
	alice       *Messenger
}

type AdminCommunityEventsSuite struct {
	EventSenderCommunityEventsSuiteBase
}

func (s *EventSenderCommunityEventsSuiteBase) GetControlNode() *Messenger {
	return s.owner
}

func (s *EventSenderCommunityEventsSuiteBase) GetEventSender() *Messenger {
	return s.eventSender
}

func (s *EventSenderCommunityEventsSuiteBase) GetMember() *Messenger {
	return s.alice
}

func (s *EventSenderCommunityEventsSuiteBase) GetSuite() *suite.Suite {
	return &s.Suite
}

func (s *EventSenderCommunityEventsSuiteBase) GetCollectiblesServiceMock() *CollectiblesServiceMock {
	return s.collectiblesServiceMock
}

func (s *EventSenderCommunityEventsSuiteBase) GetAccountsTestData() map[string][]string {
	return s.accountsTestData
}

func (s *EventSenderCommunityEventsSuiteBase) GetAccountsPasswords() map[string]string {
	return s.accountsPasswords
}

func (s *EventSenderCommunityEventsSuiteBase) SetupTest() {
	s.CommunitiesMessengerTestSuiteBase.SetupTest()
	s.mockedBalances = createMockedWalletBalance(&s.Suite)

	s.owner = s.newMessenger("", []string{})
	s.eventSender = s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})
	s.alice = s.newMessenger(accountPassword, []string{aliceAccountAddress})
}

func (s *EventSenderCommunityEventsSuiteBase) newMessenger(password string, walletAddresses []string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	communityManagerOptions := []communities.ManagerOption{
		communities.WithAllowForcingCommunityMembersReevaluation(true),
	}

	return s.newMessengerWithConfig(testMessengerConfig{
		privateKey:   privateKey,
		extraOptions: []Option{WithCommunityManagerOptions(communityManagerOptions)},
	}, password, walletAddresses)
}

func (s *AdminCommunityEventsSuite) TestAdminEditCommunityDescription() {
	// TODO admin test: update to include edit tags, logo, banner, request to join required setting, pin setting, etc...
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	editCommunityDescription(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminCreateEditDeleteChannels() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testCreateEditDeleteChannels(s, community)
}

func (s *AdminCommunityEventsSuite) TestChannelDeletionDeletesActivityCenterNotifications() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	channelID := createCommunityChannel(s, community, &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_AUTO_ACCEPT},
		Identity:    &protobuf.ChatIdentity{DisplayName: "channel with notification"},
	})
	chatID := community.ChatID(channelID)
	now := uint64(time.Now().UnixMilli())

	adminNotification := &ActivityCenterNotification{
		ID:        types.HexBytes{0x01},
		ChatID:    chatID,
		Type:      ActivityCenterNotificationTypeMention,
		Timestamp: now,
		UpdatedAt: now,
	}
	_, err := s.eventSender.persistence.SaveActivityCenterNotification(adminNotification, true)
	s.Require().NoError(err)

	memberNotification := &ActivityCenterNotification{
		ID:        types.HexBytes{0x02},
		ChatID:    chatID,
		Type:      ActivityCenterNotificationTypeMention,
		Timestamp: now,
		UpdatedAt: now,
	}
	_, err = s.alice.persistence.SaveActivityCenterNotification(memberNotification, true)
	s.Require().NoError(err)

	response, err := s.eventSender.DeleteCommunityChat(community.ID(), channelID)
	s.Require().NoError(err)
	s.Require().Len(response.ActivityCenterNotifications(), 1)
	s.Require().Equal(adminNotification.ID, response.ActivityCenterNotifications()[0].ID)
	s.Require().True(response.ActivityCenterNotifications()[0].Deleted)

	memberResponse, err := WaitOnMessengerResponse(
		s.alice,
		func(response *MessengerResponse) bool {
			for _, activityCenterNotification := range response.ActivityCenterNotifications() {
				if activityCenterNotification.ID.String() == memberNotification.ID.String() {
					return activityCenterNotification.Deleted
				}
			}
			return false
		},
		"member did not receive deleted activity center notification",
	)
	s.Require().NoError(err)
	s.Require().Len(memberResponse.ActivityCenterNotifications(), 1)
	s.Require().Equal(memberNotification.ID, memberResponse.ActivityCenterNotifications()[0].ID)
	s.Require().True(memberResponse.ActivityCenterNotifications()[0].Deleted)

	persistedNotification, err := s.alice.persistence.GetActivityCenterNotificationByID(memberNotification.ID)
	s.Require().NoError(err)
	s.Require().True(persistedNotification.Deleted)
}

func (s *AdminCommunityEventsSuite) TestAdminCreateEditDeleteBecomeMemberPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testCreateEditDeleteBecomeMemberPermission(s, community, protobuf.CommunityTokenPermission_BECOME_MEMBER)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotCreateBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotCreatePrivilegedCommunityPermission(s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotCreateBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotCreatePrivilegedCommunityPermission(s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotEditBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotEditPrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotEditBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotEditPrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotDeleteBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotDeletePrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_ADMIN, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotDeleteBecomeTokenMasterPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotDeletePrivilegedCommunityPermission(
		s, community, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestAdminAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalAdmin := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})

	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalAdmin)
}

func (s *AdminCommunityEventsSuite) TestAdminAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})

	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *AdminCommunityEventsSuite) TestAdminRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalAdmin := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})
	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})

	testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalAdmin)
}

func (s *AdminCommunityEventsSuite) TestAdminRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})

	testRejectMemberRequestToJoin(s, community, user)
}

func (s *AdminCommunityEventsSuite) TestAdminControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	s.T().Skip("flaky test")

	additionalAdmin := s.newMessenger("qwerty", []string{eventsSenderAccountAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	testControlNodeHandlesMultipleEventSenderRequestToJoinDecisions(s, community, user, additionalAdmin)
}

func (s *AdminCommunityEventsSuite) TestAdminCreateEditDeleteCategories() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testCreateEditDeleteCategories(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminReorderChannelsAndCategories() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testReorderChannelsAndCategories(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminKickAdmin() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderKickTheSameRole(s, community)
}

func (s *AdminCommunityEventsSuite) TestOwnerKickControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderKickControlNode(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminKickMember() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	kickMember(s, community.ID(), crypto.PubkeyToHex(&s.alice.identity.PublicKey))
}

func (s *AdminCommunityEventsSuite) TestAdminBanAdmin() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testOwnerBanTheSameRole(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminBanControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testOwnerBanControlNode(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminBanUnbanMember() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testBanUnbanMember(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminDeleteAnyMessageInTheCommunity() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testDeleteAnyMessageInTheCommunity(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminPinMessage() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderPinMessage(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminAddCommunityToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)

	tokenERC721 := &token.CommunityToken{
		CommunityID:        community.IDString(),
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x123",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             &bigint.BigInt{Int: big.NewInt(123)},
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        token.Deployed,
		Base64Image:        "ABCD",
	}

	_, err := s.eventSender.SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = s.eventSender.AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().Error(err)
}

func (s *AdminCommunityEventsSuite) TestAdminAddTokenMasterAndOwnerToken() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderAddTokenMasterAndOwnerToken(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminReceiveOwnerTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testAddAndSyncOwnerTokenFromControlNode(s, community, token.OwnerLevel)
}

func (s *AdminCommunityEventsSuite) TestAdminReceiveTokenMasterTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testAddAndSyncTokenFromControlNode(s, community, token.MasterLevel)
}

func (s *AdminCommunityEventsSuite) TestAdminReceiveCommunityTokenFromControlNode() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testAddAndSyncTokenFromControlNode(s, community, token.CommunityLevel)
}

func (s *AdminCommunityEventsSuite) TestMemberReceiveOwnerEventsWhenControlNodeOffline() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testMemberReceiveEventsWhenControlNodeOffline(s, community)
}

func (s *AdminCommunityEventsSuite) TestJoinedAdminReceiveRequestsToJoinWithoutRevealedAccounts() {
	// event sender will receive privileged role during permission creation
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_NONE, []*Messenger{})

	// set up additional user (bob) that will send request to join
	bob := s.newMessenger(accountPassword, []string{bobAccountAddress})

	// set up additional user that will join to the community as TokenMaster
	newPrivilegedUser := s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})

	testJoinedPrivilegedMemberReceiveRequestsToJoin(s, community, bob, newPrivilegedUser, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestReceiveRequestsToJoinWithRevealedAccountsAfterGettingAdminRole() {
	// set up additional user (bob) that will send request to join
	bob := s.newMessenger(accountPassword, []string{bobAccountAddress})
	testMemberReceiveRequestsToJoinAfterGettingNewRole(s, bob, protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *AdminCommunityEventsSuite) TestAdminAcceptsRequestToJoinAfterMemberLeave() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("somePassword", []string{"0x0123400000000000000000000000000000000000"})
	testPrivilegedMemberAcceptsRequestToJoinAfterMemberLeave(s, community, user)
}

func (s *AdminCommunityEventsSuite) TestAdminBanMemberWithDeletingAllMessages() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testBanMemberWithDeletingAllMessages(s, community)
}
