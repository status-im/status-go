package protocol

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
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
	shh    types.Waku
	logger *zap.Logger
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

	s.owner = s.newMessenger()
	s.admin = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.admin.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *AdminCommunityEventsSuite) TearDownTest() {
	s.Require().NoError(s.owner.Shutdown())
	s.Require().NoError(s.admin.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *AdminCommunityEventsSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	messenger, err := newCommunitiesTestMessenger(shh, privateKey, s.logger, nil, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *AdminCommunityEventsSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
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
	testCreateEditDeleteBecomeMemberPermission(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotCreateBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)

	permissionRequest := createTestPermissionRequest(community)
	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

	response, err := s.admin.CreateCommunityTokenPermission(permissionRequest)
	s.Require().Nil(response)
	s.Require().Error(err)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotEditBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotEditBecomeAdminPermission(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminCannotDeleteBecomeAdminPermission() {
	community := setUpCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)
	testEventSenderCannotDeleteBecomeAdminPermission(s, community)
}

func (s *AdminCommunityEventsSuite) TestAdminAcceptMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)

	// set up additional user that will send request to join
	user := s.newMessenger()
	testAcceptMemberRequestToJoin(s, community, user)
}

func (s *AdminCommunityEventsSuite) TestAdminRejectMemberRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN)

	// set up additional user that will send request to join
	user := s.newMessenger()
	testRejectMemberRequestToJoin(s, community, user)
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
