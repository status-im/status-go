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
	community := s.SetUpCommunityAndRoles()

	s.adminEditsCommunityDescription(community)
}

func (s *AdminMessengerCommunitiesSuite) SetUpCommunityAndRoles() *communities.Community {
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
		Role:        protobuf.CommunityMember_ROLE_ADMIN,
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
	response, err := s.admin.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_ON_REQUEST,
			Name:        "new community name",
			Color:       "#000000",
			Description: "new community description",
		},
	})
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	s.refreshMessengerResponses()

	_, err = WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "admin community description changed message not received")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "admin community description changed message not received")
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
