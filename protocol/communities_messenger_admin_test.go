package protocol

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/waku"
)

func TestAdminCommunityEventsSuite(t *testing.T) {
	suite.Run(t, new(AdminCommunityEventsSuite))
}

type AdminCommunityEventsSuite struct {
	suite.Suite
	owner *Messenger
	admin *Messenger
	alice *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh            types.Waku
	logger         *zap.Logger
	mockedBalances map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
}

func (s *AdminCommunityEventsSuite) GetControlNode() *Messenger {
	return s.owner
}

func (s *AdminCommunityEventsSuite) GetEventSender() *Messenger {
	return s.admin
}

func (s *AdminCommunityEventsSuite) GetMember() *Messenger {
	return s.alice
}

func (s *AdminCommunityEventsSuite) GetSuite() *suite.Suite {
	return &s.Suite
}

func (s *AdminCommunityEventsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.owner = s.newMessenger("", []string{})
	s.admin = s.newMessenger("qwerty", []string{commmunitiesEventsEventSenderAddress})
	s.alice = s.newMessenger("", []string{})
	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.admin.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.mockedBalances = createMockedWalletBalance(&s.Suite)
}

func (s *AdminCommunityEventsSuite) TearDownTest() {
	s.Require().NoError(s.owner.Shutdown())
	s.Require().NoError(s.admin.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *AdminCommunityEventsSuite) newMessenger(password string, walletAddresses []string) *Messenger {
	return newMessenger(&s.Suite, s.shh, s.logger, password, walletAddresses, &s.mockedBalances)
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
	additionalAdmin := s.newMessenger("qwerty", []string{commmunitiesEventsEventSenderAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})
	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalAdmin)
}

func (s *AdminCommunityEventsSuite) TestAdminAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *AdminCommunityEventsSuite) TestAdminRejectMemberRequestToJoinResponseSharedWithOtherEventSenders() {
	additionalAdmin := s.newMessenger("qwerty", []string{commmunitiesEventsEventSenderAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})
	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(s, community, user, additionalAdmin)
}

func (s *AdminCommunityEventsSuite) TestAdminRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testRejectMemberRequestToJoin(s, community, user)
}

func (s *AdminCommunityEventsSuite) TestAdminRequestToJoinStateCannotBeOverridden() {
	additionalAdmin := s.newMessenger("qwerty", []string{commmunitiesEventsEventSenderAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	testEventSenderCannotOverrideRequestToJoinState(s, community, user, additionalAdmin)
}

func (s *AdminCommunityEventsSuite) TestAdminControlNodeHandlesMultipleEventSenderRequestToJoinDecisions() {
	additionalAdmin := s.newMessenger("qwerty", []string{commmunitiesEventsEventSenderAddress})
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{additionalAdmin})

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
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
	kickMember(s, community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
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

	_, err := s.admin.SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = s.admin.AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().Error(err)
}

func (s *AdminCommunityEventsSuite) TestMemberReceiveAdminEventsWhenOwnerOffline() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testMemberReceiveEventsWhenControlNodeOffline(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminResendRejectedEvents() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)

	// admin modifies community description
	adminEditRequest := &requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        "admin name",
			Description: "admin description",
			Color:       "#FFFFFF",
			Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		},
	}
	_, err := s.admin.EditCommunity(adminEditRequest)
	s.Require().NoError(err)

	// in the meantime, control node updates community description as well
	ownerEditRequest := &requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        "control node name",
			Description: "control node description",
			Color:       "#FFFFFF",
			Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		},
	}
	_, err = s.owner.EditCommunity(ownerEditRequest)
	s.Require().NoError(err)

	waitOnAdminEventsRejection := waitOnCommunitiesEvent(s.owner, func(s *communities.Subscription) bool {
		return s.CommunityEventsMessageInvalidClock != nil
	})

	// control node receives admin event and rejects it
	_, err = WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		select {
		case err := <-waitOnAdminEventsRejection:
			s.Require().NoError(err)
			return true
		default:
			return false
		}
	}, "")
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(ownerEditRequest.Description, community.DescriptionText())

	// admin receives rejected events and re-applies them
	// there is no signal whatsoever, we just wait for admin to process all incoming messages
	_, _ = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return false
	}, "")

	// control node receives re-applied admin event and accepts it
	response, err := WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "no communities in response")
	s.Require().NoError(err)
	s.Require().Equal(adminEditRequest.Description, response.Communities()[0].DescriptionText())

	// admin receives updated community description
	response, err = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "no communities in response")
	s.Require().NoError(err)
	s.Require().Equal(adminEditRequest.Description, response.Communities()[0].DescriptionText())
}
