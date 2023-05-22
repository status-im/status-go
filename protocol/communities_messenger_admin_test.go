package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/account/generator"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestAdminMessengerCommunitiesSuite(t *testing.T) {
	suite.Run(t, new(AdminMessengerCommunitiesSuite))
}

type AdminMessengerCommunitiesSuite struct {
	suite.Suite
	owner *Messenger
	admin *Messenger
	alice *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *AdminMessengerCommunitiesSuite) SetupTest() {
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

func (s *AdminMessengerCommunitiesSuite) TearDownTest() {
	s.Require().NoError(s.owner.Shutdown())
	s.Require().NoError(s.admin.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *AdminMessengerCommunitiesSuite) newMessengerWithOptions(shh types.Waku, privateKey *ecdsa.PrivateKey, options []Option) *Messenger {
	m, err := NewMessenger(
		"Test",
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		nil,
		nil,
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	config := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}

	networks := json.RawMessage("{}")
	setting := settings.Settings{
		Address:                   types.HexToAddress("0x1122334455667788990011223344556677889900"),
		AnonMetricsShouldSend:     false,
		CurrentNetwork:            "mainnet_rpc",
		DappsAddress:              types.HexToAddress("0x1122334455667788990011223344556677889900"),
		InstallationID:            "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:                    "0x1122334455667788990011223344556677889900",
		Name:                      "Test",
		Networks:                  &networks,
		PhotoPath:                 "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:            false,
		PublicKey:                 "0x04112233445566778899001122334455667788990011223344556677889900112233445566778899001122334455667788990011223344556677889900",
		SigningPhrase:             "yurt joey vibe",
		SendPushNotifications:     true,
		ProfilePicturesVisibility: 1,
		DefaultSyncPeriod:         777600,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		WalletRootAddress:         types.HexToAddress("0x1122334455667788990011223344556677889900")}

	_ = m.settings.CreateSettings(setting, config)

	return m
}

func (s *AdminMessengerCommunitiesSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	s.Require().NoError(err)
	madb, err := multiaccounts.InitializeDB(tmpfile.Name())
	s.Require().NoError(err)

	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")

	options := []Option{
		WithCustomLogger(s.logger),
		WithDatabaseConfig(":memory:", "somekey", sqlite.ReducedKDFIterationsNumber),
		WithMultiAccounts(madb),
		WithAccount(iai.ToMultiAccount()),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *AdminMessengerCommunitiesSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminEditCommunityDescription() {
	// TODO admin test: update to include edit tags, logo, banner, request to join required setting, pin setting, etc...
	community := s.setUpCommunityAndRoles()
	s.adminEditsCommunityDescription(community)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteChannels() {
	community := s.setUpCommunityAndRoles()
	s.adminCreateCommunityChannel(community)

	// TODO admin test: Create, edit and delete channels (allowed)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteCategories() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Create, edit and delete categories (allowed)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminReorderChannelsAndCategories() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Reorder channels and categories (allowed)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteBecomeMemberPermission() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Create, edit and delete 'become member' permissions
}

func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteBecomeAdminPermission() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Create, edit and delete 'become admin' permissions (restricted)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminAcceptMemberRequestToJoin() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Receive 'request to join' notifications, and ability to Accept or Reject (accept must be approved by owner node)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminKickMember() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Kick member (kick must be approved by owner node)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminBanMember() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Ban members (ban must be approved by owner node)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminDeleteAnyMessageInTheCommunity() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Delete any message in the Community
}

func (s *AdminMessengerCommunitiesSuite) TestAdminPinMessage() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Pin messages, if 'Any member can pin a message' is switched off in community settings
}

func (s *AdminMessengerCommunitiesSuite) TestAdminMintToken() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Mint Tokens (rescticted)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminAirdropTokens() {
	s.setUpCommunityAndRoles()
	// TODO admin test: Airdrop Tokens (restricted)
}

// TODO admin test:
//	- would be nice to test on a regression and check that simple user can't do this actions
//  - test when user loses his admin permissions
//  - some other tests scenarious (review)

func (s *AdminMessengerCommunitiesSuite) setUpCommunityAndRoles() *communities.Community {
	tcs2, err := s.owner.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 community")

	// owner creates a community and chat
	community := s.createCommunity()
	_ = s.createCommunityChat(community)

	// add admin and alice to the community
	s.inviteAndJoin(community, s.admin)
	s.inviteAndJoin(community, s.alice)

	s.refreshMessengerResponses()

	// grant admin permissions to the admin
	s.grantAdminPermissions(community, s.admin)

	return community
}

func (s *AdminMessengerCommunitiesSuite) inviteAndJoin(community *communities.Community, target *Messenger) {
	response, err := s.owner.InviteUsersToCommunity(&requests.InviteUsersToCommunity{
		CommunityID: community.ID(),
		Users:       []types.HexBytes{common.PubkeyToHexBytes(&target.identity.PublicKey)},
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&target.identity.PublicKey))

	_, err = WaitOnMessengerResponse(target, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "community not received")
	s.Require().NoError(err)

	response, err = target.JoinCommunity(context.Background(), community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().Len(response.Chats(), 1)

	s.Require().NoError(target.SaveChat(response.Chats()[0]))

	_, err = WaitOnMessengerResponse(target, func(response *MessengerResponse) bool {
		return len(response.Messages()) > 0
	}, "message 'You have been invited to community' not received")
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) createCommunity() *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}
	response, err := s.owner.CreateCommunity(description, false)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 0)

	return response.Communities()[0]
}

func (s *AdminMessengerCommunitiesSuite) createCommunityChat(community *communities.Community) *Chat {
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "",
			Description: "status-core community chatToModerator",
		},
	}

	response, err := s.owner.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
	return response.Chats()[0]
}

func (s *AdminMessengerCommunitiesSuite) grantAdminPermissions(community *communities.Community, target *Messenger) {
	response_add_role, err := s.owner.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(target.IdentityPublicKey()),
		Role:        protobuf.CommunityMember_ROLE_ALL,
	})
	s.Require().NoError(err)

	checkAdminRole := func(response *MessengerResponse) bool {
		if len(response.Communities()) == 0 {
			return false
		}
		r_communities := response.Communities()
		s.Require().Len(r_communities, 1)
		s.Require().True(r_communities[0].IsMemberAdmin(target.IdentityPublicKey()))
		return true
	}

	checkAdminRole(response_add_role)

	_, err = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return checkAdminRole(response)
	}, "community description changed message not received")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return checkAdminRole(response)
	}, "community description changed message not received")
	s.Require().NoError(err)

	s.refreshMessengerResponses()
}

func (s *AdminMessengerCommunitiesSuite) adminEditsCommunityDescription(community *communities.Community) {
	expected_name := "edited community name"
	expected_color := "#000000"
	expected_descr := "edited community description"

	response, err := s.admin.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_ON_REQUEST,
			Name:        expected_name,
			Color:       expected_color,
			Description: expected_descr,
		},
	})

	checkCommunityEdit := func(response *MessengerResponse) bool {
		if len(response.Communities()) == 0 {
			return false
		}

		r_communities := response.Communities()
		s.Require().Len(r_communities, 1)
		s.Equal(expected_name, r_communities[0].Name())
		s.Equal(expected_color, r_communities[0].Color())
		s.Equal(expected_descr, r_communities[0].DescriptionText())

		return true
	}

	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	checkCommunityEdit(response)

	_, err = WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		return checkCommunityEdit(response)
	}, "admin edit community message not received by owner")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return checkCommunityEdit(response)
	}, "admin edit community message not received by alice")
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) refreshMessengerResponses() {
	_, err := WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) adminCreateCommunityChannel(community *communities.Community) {
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from admin",
			Emoji:       "",
			Description: "chat created by an admin",
		},
	}

	s.refreshMessengerResponses()

	checkChannelCreated := func(response *MessengerResponse) bool {
		s.Require().NotNil(response)
		if len(response.Communities()) == 0 {
			return false
		}

		madeAssertions := false
		for _, c := range response.Communities() {
			if c.IDString() == community.IDString() {
				madeAssertions = true
				s.Require().Len(c.Chats(), 2)
			}
		}

		if !madeAssertions {
			s.Require().Equal(true, false)
		}

		return true
	}

	response, err := s.admin.CreateCommunityChat(community.ID(), orgChat)

	s.Require().NoError(err)
	s.Require().True(checkChannelCreated(response))

	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return checkChannelCreated(response)
	}, "owner did not receive new channel from admin")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		return checkChannelCreated(r)
	}, "alice did not receive new channel from admin")
	s.Require().NoError(err)
}
