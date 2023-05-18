package protocol

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

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
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerCommunitiesSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesSuite))
}

type MessengerCommunitiesSuite struct {
	suite.Suite
	admin *Messenger
	bob   *Messenger
	alice *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerCommunitiesSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.admin = s.newMessenger()
	s.bob = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.admin.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TearDownTest() {
	s.Require().NoError(s.admin.Shutdown())
	s.Require().NoError(s.bob.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSuite) newMessengerWithOptions(shh types.Waku, privateKey *ecdsa.PrivateKey, options []Option) *Messenger {
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

func (s *MessengerCommunitiesSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
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

func (s *MessengerCommunitiesSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerCommunitiesSuite) TestCreateCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}
	response, err := s.bob.CreateCommunity(description, true)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
}

func (s *MessengerCommunitiesSuite) TestCreateCommunity_WithoutDefaultChannel() {

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}
	response, err := s.bob.CreateCommunity(description, false)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 0)
}

func (s *MessengerCommunitiesSuite) TestRetrieveCommunity() {
	alice := s.newMessenger()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)
	s.Require().Len(response.Chats(), 1)

	community := response.Communities()[0]
	communitySettings := response.CommunitiesSettings()[0]

	s.Require().Equal(communitySettings.CommunityID, community.IDString())
	s.Require().Equal(communitySettings.HistoryArchiveSupportEnabled, false)

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&alice.identity.PublicKey), &alice.identity.PublicKey, s.alice.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.bob.SaveChat(chat)
	s.Require().NoError(err)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(community.IDString(), response.Messages()[0].CommunityID)
}

func (s *MessengerCommunitiesSuite) TestJoinCommunity() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)

	communitySettings := response.CommunitiesSettings()[0]
	community := response.Communities()[0]

	s.Require().Equal(communitySettings.CommunityID, community.IDString())
	s.Require().Equal(communitySettings.HistoryArchiveSupportEnabled, false)

	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}
	response, err = s.bob.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	createdChat := response.Chats()[0]
	s.Require().Equal(community.IDString(), createdChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
	s.Require().Equal(orgChat.Identity.Emoji, createdChat.Emoji)
	s.Require().NotEmpty(createdChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
	s.Require().True(createdChat.Active)
	s.Require().NotEmpty(createdChat.Timestamp)
	s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))

	// Make sure the changes are reflect in the community
	community = response.Communities()[0]

	var chatIds []string
	for k := range community.Chats() {
		chatIds = append(chatIds, k)
	}
	category := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "category-name",
		ChatIDs:      chatIds,
	}

	response, err = s.bob.CreateCommunityCategory(category)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].Categories(), 1)

	// Make sure the changes are reflect in the community
	community = response.Communities()[0]
	chats := community.Chats()
	s.Require().Len(chats, 2)

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.bob.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.bob.SaveChat(chat)
	s.Require().NoError(err)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(community.IDString(), response.Messages()[0].CommunityID)

	// We join the org
	response, err = s.alice.JoinCommunity(ctx, community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().Len(response.Chats(), 2)
	s.Require().Len(response.Communities()[0].Categories(), 1)

	var categoryID string
	for k := range response.Communities()[0].Categories() {
		categoryID = k
	}

	// The chat should be created

	found := false
	for _, createdChat := range response.Chats() {
		if orgChat.Identity.DisplayName == createdChat.Name {
			found = true
			s.Require().Equal(community.IDString(), createdChat.CommunityID)
			s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
			s.Require().Equal(orgChat.Identity.Emoji, createdChat.Emoji)
			s.Require().NotEmpty(createdChat.ID)
			s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
			s.Require().Equal(categoryID, createdChat.CategoryID)
			s.Require().True(createdChat.Active)
			s.Require().NotEmpty(createdChat.Timestamp)
			s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))
		}
	}
	s.Require().True(found)

	// Create another org chat
	orgChat = &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core-ui",
			Emoji:       "👍",
			Description: "status-core-ui community chat",
		},
	}
	response, err = s.bob.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	var actualChat *Chat
	// Pull message, this time it should be received as advertised automatically
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) != 1 {
			return errors.New("community not received")
		}

		for _, c := range response.Chats() {
			if c.Name == orgChat.Identity.DisplayName {
				actualChat = c
				return nil
			}
		}
		return errors.New("chat not found")
	})

	s.Require().NoError(err)
	communities, err = s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities(), 1)
	s.Require().NotNil(actualChat)
	s.Require().Equal(community.IDString(), actualChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, actualChat.Name)
	s.Require().Equal(orgChat.Identity.Emoji, actualChat.Emoji)
	s.Require().NotEmpty(actualChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, actualChat.ChatType)
	s.Require().True(actualChat.Active)
	s.Require().NotEmpty(actualChat.Timestamp)
	s.Require().True(strings.HasPrefix(actualChat.ID, community.IDString()))

	// We leave the org
	response, err = s.alice.LeaveCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().False(response.Communities()[0].Joined())
	s.Require().Len(response.RemovedChats(), 3)
}

func (s *MessengerCommunitiesSuite) createCommunity() *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.admin.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	community := response.Communities()[0]
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}

	response, err = s.admin.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	return community
}

func (s *MessengerCommunitiesSuite) advertiseCommunityTo(community *communities.Community, user *Messenger) {
	chat := CreateOneToOneChat(common.PubkeyToHex(&user.identity.PublicKey), &user.identity.PublicKey, user.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err := s.admin.SaveChat(chat)
	s.Require().NoError(err)
	_, err = s.admin.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Ensure community is received
	err = tt.RetryWithBackOff(func() error {
		response, err := user.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) joinCommunity(community *communities.Community, user *Messenger) {
	// Request to join the community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Retrieve and accept join request
	err = tt.RetryWithBackOff(func() error {
		response, err := s.admin.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (accept join request)")
		}
		if !response.Communities()[0].HasMember(&user.identity.PublicKey) {
			return errors.New("user not accepted")
		}
		return nil
	})
	s.Require().NoError(err)

	// Retrieve join request response
	err = tt.RetryWithBackOff(func() error {
		response, err := user.RetrieveAll()

		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (join request response)")
		}
		if !response.Communities()[0].HasMember(&user.identity.PublicKey) {
			return errors.New("user not a member")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestCommunityContactCodeAdvertisement() {
	community := s.createCommunity()
	s.advertiseCommunityTo(community, s.bob)
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunity(community, s.bob)
	s.joinCommunity(community, s.alice)

	// Trigger ContactCodeAdvertisement
	err := s.bob.SetDisplayName("bobby")
	s.Require().NoError(err)
	err = s.bob.SetBio("I like P2P chats")
	s.Require().NoError(err)

	// Ensure alice receives bob's ContactCodeAdvertisement
	err = tt.RetryWithBackOff(func() error {
		response, err := s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Contacts) == 0 {
			return errors.New("no contacts in response")
		}
		if response.Contacts[0].DisplayName != "bobby" {
			return errors.New("display name was not updated")
		}
		if response.Contacts[0].Bio != "I like P2P chats" {
			return errors.New("bio was not updated")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestInviteUsersToCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].HasMember(&s.bob.identity.PublicKey))
	s.Require().True(response.Communities()[0].IsMemberAdmin(&s.bob.identity.PublicKey))

	community := response.Communities()[0]

	response, err = s.bob.InviteUsersToCommunity(
		&requests.InviteUsersToCommunity{
			CommunityID: community.ID(),
			Users:       []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))
}

func (s *MessengerCommunitiesSuite) TestPostToCommunityChat() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_INVITATION_ONLY,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].Chats(), 1)
	s.Require().Len(response.Chats(), 1)

	community := response.Communities()[0]

	// Create chat
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Description: "status-core community chat",
		},
	}

	response, err = s.bob.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].Chats(), 2)
	s.Require().Len(response.Chats(), 1)

	response, err = s.bob.InviteUsersToCommunity(
		&requests.InviteUsersToCommunity{
			CommunityID: community.ID(),
			Users:       []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities(), 1)

	communityID := response.Communities()[0].ID()
	s.Require().Equal(communityID, community.ID())

	ctx := context.Background()

	// We join the org
	response, err = s.alice.JoinCommunity(ctx, community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].Chats(), 2)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().Len(response.Chats(), 2)

	chatID := response.Chats()[1].ID
	inputMessage := &common.Message{}
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	_, err = s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.messages) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(inputMessage.Text, response.Messages()[0].Text)

	// check if response contains the chat we're interested in
	// we use this instead of checking just the length of the chat because
	// a CommunityDescription message might be received in the meantime due to syncing
	// hence response.Chats() might contain the general chat, and the new chat;
	// or only the new chat if the CommunityDescription message has not arrived
	found := false
	for _, chat := range response.Chats() {
		if chat.ID == chatID {
			found = true
		}
	}
	s.Require().True(found)
}

func (s *MessengerCommunitiesSuite) TestImportCommunity() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().True(response.Communities()[0].IsAdmin())

	community := response.Communities()[0]
	communitySettings := response.CommunitiesSettings()[0]

	s.Require().Equal(communitySettings.CommunityID, community.IDString())
	s.Require().Equal(communitySettings.HistoryArchiveSupportEnabled, false)

	category := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "category-name",
		ChatIDs:      []string{},
	}

	response, err = s.bob.CreateCommunityCategory(category)
	s.Require().NoError(err)
	community = response.Communities()[0]

	privateKey, err := s.bob.ExportCommunity(community.ID())
	s.Require().NoError(err)

	_, err = s.alice.ImportCommunity(ctx, privateKey)
	s.Require().NoError(err)

	// Invite user on bob side
	newUser, err := crypto.GenerateKey()
	s.Require().NoError(err)

	_, err = s.bob.InviteUsersToCommunity(
		&requests.InviteUsersToCommunity{
			CommunityID: community.ID(),
			Users:       []types.HexBytes{common.PubkeyToHexBytes(&newUser.PublicKey)},
		},
	)
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		if !response.Communities()[0].IsAdmin() {
			return errors.New("isn't admin despite import")
		}
		return nil
	})

	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestRolesAfterImportCommunity() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create a community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().True(response.Communities()[0].IsAdmin())
	s.Require().True(response.Communities()[0].IsMemberAdmin(&s.bob.identity.PublicKey))
	s.Require().False(response.Communities()[0].IsMemberAdmin(&s.alice.identity.PublicKey))

	community := response.Communities()[0]
	communitySettings := response.CommunitiesSettings()[0]

	s.Require().Equal(communitySettings.CommunityID, community.IDString())
	s.Require().Equal(communitySettings.HistoryArchiveSupportEnabled, false)

	category := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "category-name",
		ChatIDs:      []string{},
	}

	response, err = s.bob.CreateCommunityCategory(category)
	s.Require().NoError(err)
	community = response.Communities()[0]

	privateKey, err := s.bob.ExportCommunity(community.ID())
	s.Require().NoError(err)

	response, err = s.alice.ImportCommunity(ctx, privateKey)
	s.Require().NoError(err)
	s.Require().True(response.Communities()[0].IsMemberAdmin(&s.alice.identity.PublicKey))
}

func (s *MessengerCommunitiesSuite) TestRequestAccess() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	s.Require().NoError(s.bob.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()

	// We send a community link to alice
	response, err = s.bob.SendChatMessage(ctx, message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, false)

	requestToJoin1 := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin1)
	s.Require().Equal(community.ID(), requestToJoin1.CommunityID)
	s.Require().True(requestToJoin1.Our)
	s.Require().NotEmpty(requestToJoin1.ID)
	s.Require().NotEmpty(requestToJoin1.Clock)
	s.Require().Equal(requestToJoin1.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	// Make sure clock is not empty
	s.Require().NotEmpty(requestToJoin1.Clock)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin1.Clock)

	// pull all communities to make sure we set RequestedToJoinAt

	allCommunities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(allCommunities, 2)

	if bytes.Equal(allCommunities[0].ID(), community.ID()) {
		s.Require().Equal(allCommunities[0].RequestedToJoinAt(), requestToJoin1.Clock)
	} else {
		s.Require().Equal(allCommunities[1].RequestedToJoinAt(), requestToJoin1.Clock)
	}

	// pull to make sure it has been saved
	requestsToJoin, err := s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Make sure the requests are fetched also by community
	requestsToJoin, err = s.alice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin2 := response.RequestsToJoinCommunity[0]

	s.Require().NotNil(requestToJoin2)
	s.Require().Equal(community.ID(), requestToJoin2.CommunityID)
	s.Require().False(requestToJoin2.Our)
	s.Require().NotEmpty(requestToJoin2.ID)
	s.Require().NotEmpty(requestToJoin2.Clock)
	s.Require().Equal(requestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin2.State)

	s.Require().Equal(requestToJoin1.ID, requestToJoin2.ID)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Read, false)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, false)

	// Accept request

	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin1.ID}

	response, err = s.bob.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)

	updatedCommunity := response.Communities()[0]

	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&s.alice.identity.PublicKey))

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusAccepted)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, true)
	s.Require().Equal(notification.Dismissed, false)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Equal(communities.RequestToJoinStateAccepted, response.RequestsToJoinCommunity[0].State)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusAccepted)
	s.Require().Equal(notification.Read, false)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, false)

	s.Require().Len(response.Communities(), 1)
	aliceCommunity := response.Communities()[0]

	s.Require().Equal(community.ID(), aliceCommunity.ID())
	s.Require().True(aliceCommunity.HasMember(&s.alice.identity.PublicKey))

	// Community should be joined at this point
	s.Require().True(aliceCommunity.Joined())

	// Make sure the requests are not pending on either sides
	requestsToJoin, err = s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestsToJoin, err = s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

}

func (s *MessengerCommunitiesSuite) TestDeletePendingRequestAccess() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Bob creates a community
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	s.Require().NoError(s.bob.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()

	// Bob sends the community link to Alice
	response, err = s.bob.SendChatMessage(ctx, message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community for Alice
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)

	// Alice request to join community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin)
	s.Require().Equal(community.ID(), requestToJoin.CommunityID)
	s.Require().NotEmpty(requestToJoin.ID)
	s.Require().NotEmpty(requestToJoin.Clock)
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin.State)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin.Clock)

	// updating request clock by 8 days back
	requestTime := uint64(time.Now().AddDate(0, 0, -8).Unix())
	err = s.alice.communitiesManager.UpdateClockInRequestToJoin(requestToJoin.ID, requestTime)
	s.Require().NoError(err)

	// pull to make sure it has been saved
	requestsToJoin, err := s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	requestToJoin = requestsToJoin[0]
	s.Require().Equal(requestToJoin.Clock, requestTime)

	// Make sure the requests are fetched also by community
	requestsToJoin, err = s.alice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Retrieve request to join
	bobRetrieveAll := func() (*MessengerResponse, error) {
		return s.bob.RetrieveAll()
	}
	err = tt.RetryWithBackOff(func() error {
		response, err = bobRetrieveAll()
		if err != nil {
			return err
		}

		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}

		// updating request clock by 8 days back
		requestToJoin := response.RequestsToJoinCommunity[0]
		err = s.bob.communitiesManager.UpdateClockInRequestToJoin(requestToJoin.ID, requestTime)
		if err != nil {
			return err
		}

		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Check activity center notification for Bob
	fetchActivityCenterNotificationsForAdmin := func() (*ActivityCenterPaginationResponse, error) {
		return s.bob.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
			Cursor:        "",
			Limit:         10,
			ActivityTypes: []ActivityCenterType{},
			ReadType:      ActivityCenterQueryParamsReadUnread,
		})
	}
	notifications, err := fetchActivityCenterNotificationsForAdmin()
	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)

	notification := notifications.Notifications[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Delete pending request to join
	response, err = s.alice.CheckAndDeletePendingRequestToJoinCommunity(true)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	requestToJoin = response.RequestsToJoinCommunity[0]
	s.Require().True(requestToJoin.Deleted)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusIdle)

	response, err = s.bob.CheckAndDeletePendingRequestToJoinCommunity(true)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	requestToJoin = response.RequestsToJoinCommunity[0]
	s.Require().True(requestToJoin.Deleted)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().True(notification.Deleted)

	// Alice request to join community
	request = &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Retrieve request to join and Check activity center notification for Bob
	err = tt.RetryWithBackOff(func() error {
		response, err = bobRetrieveAll()
		if err != nil {
			return err
		}

		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}

		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}

		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Check activity center notification for Bob
	notifications, err = fetchActivityCenterNotificationsForAdmin()

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)

	notification = notifications.Notifications[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

}

func (s *MessengerCommunitiesSuite) TestDeletePendingRequestAccessWithDeclinedState() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Bob creates a community
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	s.Require().NoError(s.bob.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()

	// Bob sends the community link to Alice
	response, err = s.bob.SendChatMessage(ctx, message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community for Alice
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)

	// Alice request to join community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().NotEmpty(notification.ID)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Deleted, false)
	s.Require().Equal(notification.Read, true)

	requestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin)
	s.Require().Equal(community.ID(), requestToJoin.CommunityID)
	s.Require().NotEmpty(requestToJoin.ID)
	s.Require().NotEmpty(requestToJoin.Clock)
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin.State)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin.Clock)

	// Alice deletes activity center notification
	err = s.alice.DeleteActivityCenterNotifications(ctx, []types.HexBytes{notification.ID}, false)
	s.Require().NoError(err)

	// Check activity center notification for Bob after deleting
	notifications, err := s.alice.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	})
	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 0)

	// updating request clock by 8 days back
	requestTime := uint64(time.Now().AddDate(0, 0, -8).Unix())
	err = s.alice.communitiesManager.UpdateClockInRequestToJoin(requestToJoin.ID, requestTime)
	s.Require().NoError(err)

	// pull to make sure it has been saved
	requestsToJoin, err := s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	requestToJoin = requestsToJoin[0]
	s.Require().Equal(requestToJoin.Clock, requestTime)

	// Make sure the requests are fetched also by community
	requestsToJoin, err = s.alice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	bobRetrieveAll := func() (*MessengerResponse, error) {
		return s.bob.RetrieveAll()
	}

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = bobRetrieveAll()
		if err != nil {
			return err
		}

		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}

		// updating request clock by 8 days back
		requestToJoin := response.RequestsToJoinCommunity[0]
		err = s.bob.communitiesManager.UpdateClockInRequestToJoin(requestToJoin.ID, requestTime)
		if err != nil {
			return err
		}

		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Check activity center notification for Bob
	fetchActivityCenterNotificationsForAdmin := func() (*ActivityCenterPaginationResponse, error) {
		return s.bob.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
			Cursor:        "",
			Limit:         10,
			ActivityTypes: []ActivityCenterType{},
			ReadType:      ActivityCenterQueryParamsReadUnread,
		})
	}

	notifications, err = fetchActivityCenterNotificationsForAdmin()
	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)

	notification = notifications.Notifications[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Check if admin sees requests correctly
	requestsToJoin, err = s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	requestsToJoin, err = s.bob.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	// Decline request
	declinedRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: requestToJoin.ID}
	response, err = s.bob.DeclineRequestToJoinCommunity(declinedRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusDeclined)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, true)

	// Check if admin sees requests correctly
	requestsToJoin, err = s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestsToJoin, err = s.bob.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Bob deletes activity center notification
	err = s.bob.DeleteActivityCenterNotifications(ctx, []types.HexBytes{notification.ID}, false)
	s.Require().NoError(err)

	// Check activity center notification for Bob after deleting
	notifications, err = fetchActivityCenterNotificationsForAdmin()
	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 0)

	// Delete pending request to join
	response, err = s.alice.CheckAndDeletePendingRequestToJoinCommunity(true)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin = response.RequestsToJoinCommunity[0]
	s.Require().True(requestToJoin.Deleted)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusIdle)
	s.Require().Equal(notification.Read, false)
	s.Require().Equal(notification.Deleted, false)

	notificationState := response.ActivityCenterState()
	s.Require().False(notificationState.HasSeen)

	// Alice request to join community
	request = &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Retrieve request to join and Check activity center notification for Bob
	err = tt.RetryWithBackOff(func() error {
		response, err = bobRetrieveAll()
		if err != nil {
			return err
		}

		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}

		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}

		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Check activity center notification for Bob
	notifications, err = fetchActivityCenterNotificationsForAdmin()

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)

	notification = notifications.Notifications[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().False(notification.Deleted)

}

func (s *MessengerCommunitiesSuite) TestCancelRequestAccess() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	s.Require().NoError(s.bob.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()

	// We send a community link to alice
	response, err = s.bob.SendChatMessage(ctx, message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin1 := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin1)
	s.Require().Equal(community.ID(), requestToJoin1.CommunityID)
	s.Require().True(requestToJoin1.Our)
	s.Require().NotEmpty(requestToJoin1.ID)
	s.Require().NotEmpty(requestToJoin1.Clock)
	s.Require().Equal(requestToJoin1.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	// Make sure clock is not empty
	s.Require().NotEmpty(requestToJoin1.Clock)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin1.Clock)

	// pull all communities to make sure we set RequestedToJoinAt

	allCommunities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(allCommunities, 2)

	if bytes.Equal(allCommunities[0].ID(), community.ID()) {
		s.Require().Equal(allCommunities[0].RequestedToJoinAt(), requestToJoin1.Clock)
	} else {
		s.Require().Equal(allCommunities[1].RequestedToJoinAt(), requestToJoin1.Clock)
	}

	// pull to make sure it has been saved
	requestsToJoin, err := s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Make sure the requests are fetched also by community
	requestsToJoin, err = s.alice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin2 := response.RequestsToJoinCommunity[0]

	s.Require().NotNil(requestToJoin2)
	s.Require().Equal(community.ID(), requestToJoin2.CommunityID)
	s.Require().False(requestToJoin2.Our)
	s.Require().NotEmpty(requestToJoin2.ID)
	s.Require().NotEmpty(requestToJoin2.Clock)
	s.Require().Equal(requestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin2.State)

	s.Require().Equal(requestToJoin1.ID, requestToJoin2.ID)

	// Cancel request to join community
	requestsToJoin, err = s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	requestToJoin := requestsToJoin[0]

	requestToCancel := &requests.CancelRequestToJoinCommunity{ID: requestToJoin.ID}
	response, err = s.alice.CancelRequestToJoinCommunity(requestToCancel)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Equal(communities.RequestToJoinStateCanceled, response.RequestsToJoinCommunity[0].State)

	// pull to make sure it has been saved
	cancelRequestsToJoin, err := s.alice.MyCanceledRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(cancelRequestsToJoin, 1)
	s.Require().Equal(cancelRequestsToJoin[0].State, communities.RequestToJoinStateCanceled)

	// Retrieve cancel request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Retrieve activity center notifications for admin to make sure the request notification is deleted
	notifications, err := s.bob.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	})

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 0)

	cancelRequestToJoin2 := response.RequestsToJoinCommunity[0]

	s.Require().NotNil(cancelRequestToJoin2)
	s.Require().Equal(community.ID(), cancelRequestToJoin2.CommunityID)
	s.Require().False(cancelRequestToJoin2.Our)
	s.Require().NotEmpty(cancelRequestToJoin2.ID)
	s.Require().NotEmpty(cancelRequestToJoin2.Clock)
	s.Require().Equal(cancelRequestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStateCanceled, cancelRequestToJoin2.State)

}

func (s *MessengerCommunitiesSuite) TestRequestAccessAgain() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	s.Require().NoError(s.bob.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()

	// We send a community link to alice
	response, err = s.bob.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, false)

	requestToJoin1 := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin1)
	s.Require().Equal(community.ID(), requestToJoin1.CommunityID)
	s.Require().True(requestToJoin1.Our)
	s.Require().NotEmpty(requestToJoin1.ID)
	s.Require().NotEmpty(requestToJoin1.Clock)
	s.Require().Equal(requestToJoin1.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	// Make sure clock is not empty
	s.Require().NotEmpty(requestToJoin1.Clock)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin1.Clock)

	// pull all communities to make sure we set RequestedToJoinAt

	allCommunities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(allCommunities, 2)

	if bytes.Equal(allCommunities[0].ID(), community.ID()) {
		s.Require().Equal(allCommunities[0].RequestedToJoinAt(), requestToJoin1.Clock)
	} else {
		s.Require().Equal(allCommunities[1].RequestedToJoinAt(), requestToJoin1.Clock)
	}

	// pull to make sure it has been saved
	requestsToJoin, err := s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Make sure the requests are fetched also by community
	requestsToJoin, err = s.alice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin2 := response.RequestsToJoinCommunity[0]

	s.Require().NotNil(requestToJoin2)
	s.Require().Equal(community.ID(), requestToJoin2.CommunityID)
	s.Require().False(requestToJoin2.Our)
	s.Require().NotEmpty(requestToJoin2.ID)
	s.Require().NotEmpty(requestToJoin2.Clock)
	s.Require().Equal(requestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin2.State)

	s.Require().Equal(requestToJoin1.ID, requestToJoin2.ID)

	// Check that a notification is been added to messenger

	notifications := response.Notifications()
	s.Require().Len(notifications, 1)
	s.Require().NotEqual(notifications[0].ID.Hex(), "0x0000000000000000000000000000000000000000000000000000000000000000")

	// Accept request

	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin1.ID}

	response, err = s.bob.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusAccepted)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, true)
	s.Require().Equal(notification.Dismissed, false)

	s.Require().Len(response.Communities(), 1)

	updatedCommunity := response.Communities()[0]

	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&s.alice.identity.PublicKey))

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)

	aliceCommunity := response.Communities()[0]

	s.Require().Equal(community.ID(), aliceCommunity.ID())
	s.Require().True(aliceCommunity.HasMember(&s.alice.identity.PublicKey))

	// Community should be joined at this point
	s.Require().True(aliceCommunity.Joined())

	// Make sure the requests are not pending on either sides
	requestsToJoin, err = s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestsToJoin, err = s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	// We request again
	request2 := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org, it should error as we are already a member
	response, err = s.alice.RequestToJoinCommunity(request2)
	s.Require().Error(err)

	// We kick the member
	response, err = s.bob.RemoveUserFromCommunity(
		community.ID(),
		common.PubkeyToHex(&s.alice.identity.PublicKey),
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().False(community.HasMember(&s.alice.identity.PublicKey))

	// Alice should then be removed
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("activity center notification not received")
		}
		if response.ActivityCenterState().HasSeen {
			return errors.New("activity center seen state is incorrect")
		}
		return nil
	})

	// Check we got AC notification for Alice
	aliceNotifications, err := s.alice.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{ActivityCenterNotificationTypeCommunityKicked},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	},
	)
	s.Require().NoError(err)
	s.Require().Len(aliceNotifications.Notifications, 1)
	s.Require().Equal(community.IDString(), aliceNotifications.Notifications[0].CommunityID)

	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)

	aliceCommunity = response.Communities()[0]

	s.Require().Equal(community.ID(), aliceCommunity.ID())
	s.Require().False(aliceCommunity.HasMember(&s.alice.identity.PublicKey))

	// Alice can request access again
	request3 := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err = s.alice.RequestToJoinCommunity(request3)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin3 := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin3)
	s.Require().Equal(community.ID(), requestToJoin3.CommunityID)
	s.Require().True(requestToJoin3.Our)
	s.Require().NotEmpty(requestToJoin3.ID)
	s.Require().NotEmpty(requestToJoin3.Clock)
	s.Require().Equal(requestToJoin3.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin3.State)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin3.Clock)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin4 := response.RequestsToJoinCommunity[0]

	s.Require().NotNil(requestToJoin4)
	s.Require().Equal(community.ID(), requestToJoin4.CommunityID)
	s.Require().False(requestToJoin4.Our)
	s.Require().NotEmpty(requestToJoin4.ID)
	s.Require().NotEmpty(requestToJoin4.Clock)
	s.Require().Equal(requestToJoin4.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin4.State)

	s.Require().Equal(requestToJoin3.ID, requestToJoin4.ID)
}

func (s *MessengerCommunitiesSuite) TestDeclineAccess() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	s.Require().NoError(s.bob.SaveChat(chat))

	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()

	// We send a community link to alice
	response, err = s.bob.SendChatMessage(ctx, message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin1 := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin1)
	s.Require().Equal(community.ID(), requestToJoin1.CommunityID)
	s.Require().True(requestToJoin1.Our)
	s.Require().NotEmpty(requestToJoin1.ID)
	s.Require().NotEmpty(requestToJoin1.Clock)
	s.Require().Equal(requestToJoin1.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Dismissed, false)
	s.Require().Equal(notification.Accepted, false)

	// Make sure clock is not empty
	s.Require().NotEmpty(requestToJoin1.Clock)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin1.Clock)

	// pull all communities to make sure we set RequestedToJoinAt

	allCommunities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(allCommunities, 2)

	if bytes.Equal(allCommunities[0].ID(), community.ID()) {
		s.Require().Equal(allCommunities[0].RequestedToJoinAt(), requestToJoin1.Clock)
	} else {
		s.Require().Equal(allCommunities[1].RequestedToJoinAt(), requestToJoin1.Clock)
	}

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// Check if admin sees requests correctly
	requestsToJoin, err := s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	requestsToJoin, err = s.bob.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestToJoin2 := response.RequestsToJoinCommunity[0]

	s.Require().NotNil(requestToJoin2)
	s.Require().Equal(community.ID(), requestToJoin2.CommunityID)
	s.Require().False(requestToJoin2.Our)
	s.Require().NotEmpty(requestToJoin2.ID)
	s.Require().NotEmpty(requestToJoin2.Clock)
	s.Require().Equal(requestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin2.State)

	s.Require().Equal(requestToJoin1.ID, requestToJoin2.ID)

	// Decline request
	declinedRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: requestToJoin1.ID}
	response, err = s.bob.DeclineRequestToJoinCommunity(declinedRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusDeclined)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, true)

	// Check if admin sees requests correctly
	requestsToJoin, err = s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestsToJoin, err = s.bob.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	// Accept declined request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin1.ID}
	response, err = s.bob.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)

	updatedCommunity := response.Communities()[0]

	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&s.alice.identity.PublicKey))

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusAccepted)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)

	aliceCommunity := response.Communities()[0]

	s.Require().Equal(community.ID(), aliceCommunity.ID())
	s.Require().True(aliceCommunity.HasMember(&s.alice.identity.PublicKey))

	// Community should be joined at this point
	s.Require().True(aliceCommunity.Joined())

	// Make sure the requests are not pending on either sides
	requestsToJoin, err = s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestsToJoin, err = s.bob.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestsToJoin, err = s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)
}

func (s *MessengerCommunitiesSuite) TestLeaveAndRejoinCommunity() {
	community := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)
	s.advertiseCommunityTo(community, s.bob)

	s.joinCommunity(community, s.alice)
	s.joinCommunity(community, s.bob)

	joinedCommunities, err := s.admin.communitiesManager.Joined()
	s.Require().NoError(err)
	s.Require().Equal(3, joinedCommunities[0].MembersCount())

	response, err := s.alice.LeaveCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().False(response.Communities()[0].Joined())

	// admin should receive alice's request to leave
	// and then update and advertise community members list accordingly

	verifyCommunityMembers := func(user *Messenger) error {
		response, err := user.RetrieveAll()
		if err != nil {
			return err
		}

		if len(response.Communities()) == 0 {
			return errors.New("no communities in response")
		}

		var communityMembersError error = nil

		if response.Communities()[0].MembersCount() != 2 {
			communityMembersError = fmt.Errorf("invalid number of members: %d", response.Communities()[0].MembersCount())
		} else if !response.Communities()[0].HasMember(&s.admin.identity.PublicKey) {
			communityMembersError = errors.New("admin removed from community")
		} else if !response.Communities()[0].HasMember(&s.bob.identity.PublicKey) {
			communityMembersError = errors.New("bob removed from community")
		} else if response.Communities()[0].HasMember(&s.alice.identity.PublicKey) {
			communityMembersError = errors.New("alice not removed from community")
		}

		return communityMembersError
	}
	err = tt.RetryWithBackOff(func() error {
		return verifyCommunityMembers(s.admin)
	})
	s.Require().NoError(err)
	err = tt.RetryWithBackOff(func() error {
		return verifyCommunityMembers(s.bob)
	})
	s.Require().NoError(err)

	joinedCommunities, err = s.admin.communitiesManager.Joined()
	s.Require().NoError(err)
	s.Require().Equal(2, joinedCommunities[0].MembersCount())

	chats, err := s.alice.persistence.Chats()
	s.Require().NoError(err)
	var numberInactiveChats = 0
	for i := 0; i < len(chats); i++ {
		if !chats[i].Active {
			numberInactiveChats++
		}
	}
	s.Require().Equal(3, numberInactiveChats)

	// alice can rejoin
	s.joinCommunity(community, s.alice)

	joinedCommunities, err = s.admin.communitiesManager.Joined()
	s.Require().NoError(err)
	s.Require().Equal(3, joinedCommunities[0].MembersCount())

	chats, err = s.alice.persistence.Chats()
	s.Require().NoError(err)
	numberInactiveChats = 0
	for i := 0; i < len(chats); i++ {
		if !chats[i].Active {
			numberInactiveChats++
		}
	}
	s.Require().Equal(1, numberInactiveChats)
}

func (s *MessengerCommunitiesSuite) TestShareCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	inviteMessage := "invite to community testing message"

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	response, err = s.bob.ShareCommunity(
		&requests.ShareCommunity{
			CommunityID:   community.ID(),
			Users:         []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
			InviteMessage: inviteMessage,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Messages(), 1)

	// Add bob to contacts so it does not go on activity center
	bobPk := common.PubkeyToHex(&s.bob.identity.PublicKey)
	request := &requests.AddContact{ID: bobPk}
	_, err = s.alice.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.messages) == 0 {
			return errors.New("community link not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	message := response.Messages()[0]
	s.Require().Equal(community.IDString(), message.CommunityID)
	s.Require().Equal(inviteMessage, message.Text)
}

func (s *MessengerCommunitiesSuite) TestBanUser() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	response, err = s.bob.InviteUsersToCommunity(
		&requests.InviteUsersToCommunity{
			CommunityID: community.ID(),
			Users:       []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	response, err = s.bob.BanUserFromCommunity(
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&s.alice.identity.PublicKey),
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().False(community.HasMember(&s.alice.identity.PublicKey))
	s.Require().True(community.IsBanned(&s.alice.identity.PublicKey))

	response, err = s.bob.UnbanUserFromCommunity(
		&requests.UnbanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&s.alice.identity.PublicKey),
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().False(community.IsBanned(&s.alice.identity.PublicKey))
}

func (s *MessengerCommunitiesSuite) TestSyncCommunitySettings() {
	// Create new device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	// Pair devices
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)

	s.pairTwoDevices(alicesOtherDevice, s.alice, "their-name", "their-device-type")

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "new community",
		Color:       "#000000",
		Description: "new community description",
	}

	mr, err := s.alice.CreateCommunity(createCommunityReq, true)
	s.Require().NoError(err, "s.alice.CreateCommunity")
	var newCommunity *communities.Community
	for _, com := range mr.Communities() {
		if com.Name() == createCommunityReq.Name {
			newCommunity = com
		}
	}
	s.Require().NotNil(newCommunity)

	// Check that Alice has community settings
	cs, err := s.alice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
	s.Require().NoError(err, "communitiesManager.GetCommunitySettingsByID")
	s.NotNil(cs, "Must have community settings")

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		_, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		// Do we have new synced community settings?
		syncedSettings, err := alicesOtherDevice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
		if err != nil || syncedSettings == nil {
			return fmt.Errorf("community with sync not received %w", err)
		}
		return nil
	})
	s.Require().NoError(err)

	tcs, err := alicesOtherDevice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
	s.Require().NoError(err)

	// Check the community settings on their device matched the community settings on Alice's device
	s.Equal(cs.CommunityID, tcs.CommunityID)
	s.Equal(cs.HistoryArchiveSupportEnabled, tcs.HistoryArchiveSupportEnabled)
}

func (s *MessengerCommunitiesSuite) TestSyncCommunitySettings_EditCommunity() {
	// Create new device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	// Pair devices
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)

	s.pairTwoDevices(alicesOtherDevice, s.alice, "their-name", "their-device-type")

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "new community",
		Color:       "#000000",
		Description: "new community description",
	}

	mr, err := s.alice.CreateCommunity(createCommunityReq, true)
	s.Require().NoError(err, "s.alice.CreateCommunity")
	var newCommunity *communities.Community
	for _, com := range mr.Communities() {
		if com.Name() == createCommunityReq.Name {
			newCommunity = com
		}
	}
	s.Require().NotNil(newCommunity)

	// Check that Alice has community settings
	cs, err := s.alice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
	s.Require().NoError(err, "communitiesManager.GetCommunitySettingsByID")
	s.NotNil(cs, "Must have community settings")

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		_, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		// Do we have new synced community settings?
		syncedSettings, err := alicesOtherDevice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
		if err != nil || syncedSettings == nil {
			return fmt.Errorf("community settings with sync not received %w", err)
		}
		return nil
	})
	s.Require().NoError(err)

	tcs, err := alicesOtherDevice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
	s.Require().NoError(err)

	// Check the community settings on their device matched the community settings on Alice's device
	s.Equal(cs.CommunityID, tcs.CommunityID)
	s.Equal(cs.HistoryArchiveSupportEnabled, tcs.HistoryArchiveSupportEnabled)

	req := createCommunityReq
	req.HistoryArchiveSupportEnabled = true
	editCommunityReq := &requests.EditCommunity{
		CommunityID:     newCommunity.ID(),
		CreateCommunity: *req,
	}

	mr, err = s.alice.EditCommunity(editCommunityReq)
	s.Require().NoError(err, "s.alice.EditCommunity")
	var editedCommunity *communities.Community
	for _, com := range mr.Communities() {
		if com.Name() == createCommunityReq.Name {
			editedCommunity = com
		}
	}
	s.Require().NotNil(editedCommunity)

	// Wait a bit for sync messages to reach destination
	time.Sleep(1 * time.Second)
	err = tt.RetryWithBackOff(func() error {
		_, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}
		return nil
	})
	s.Require().NoError(err)

	tcs, err = alicesOtherDevice.communitiesManager.GetCommunitySettingsByID(newCommunity.ID())
	s.Require().NoError(err)

	// Check the community settings on their device matched the community settings on Alice's device
	s.Equal(cs.CommunityID, tcs.CommunityID)
	s.Equal(req.HistoryArchiveSupportEnabled, tcs.HistoryArchiveSupportEnabled)
}

// TestSyncCommunity tests basic sync functionality between 2 Messengers
func (s *MessengerCommunitiesSuite) TestSyncCommunity() {
	// Create new device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	tcs, err := alicesOtherDevice.communitiesManager.All()
	s.Require().NoError(err, "alicesOtherDevice.communitiesManager.All")
	s.Len(tcs, 1, "Must have 1 communities")

	// Pair devices
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)

	s.pairTwoDevices(alicesOtherDevice, s.alice, "their-name", "their-device-type")

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "new community",
		Color:       "#000000",
		Description: "new community description",
	}

	mr, err := s.alice.CreateCommunity(createCommunityReq, true)
	s.Require().NoError(err, "s.alice.CreateCommunity")
	var newCommunity *communities.Community
	for _, com := range mr.Communities() {
		if com.Name() == createCommunityReq.Name {
			newCommunity = com
		}
	}
	s.Require().NotNil(newCommunity)

	// Check that Alice has 2 communities
	cs, err := s.alice.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(cs, 2, "Must have 2 communities")

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		_, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		// Do we have a new synced community?
		_, err = alicesOtherDevice.communitiesManager.GetSyncedRawCommunity(newCommunity.ID())
		if err != nil {
			return fmt.Errorf("community with sync not received %w", err)
		}

		return nil
	})
	s.Require().NoError(err)

	// Count the number of communities in their device
	tcs, err = alicesOtherDevice.communitiesManager.All()
	s.Require().NoError(err)
	s.Len(tcs, 2, "There must be 2 communities")

	s.logger.Debug("", zap.Any("tcs", tcs))

	// Get the new community from their db
	tnc, err := alicesOtherDevice.communitiesManager.GetByID(newCommunity.ID())
	s.Require().NoError(err)

	// Check the community on their device matched the new community on Alice's device
	s.Equal(newCommunity.ID(), tnc.ID())
	s.Equal(newCommunity.Name(), tnc.Name())
	s.Equal(newCommunity.DescriptionText(), tnc.DescriptionText())
	s.Equal(newCommunity.IDString(), tnc.IDString())
	s.Equal(newCommunity.PrivateKey(), tnc.PrivateKey())
	s.Equal(newCommunity.PublicKey(), tnc.PublicKey())
	s.Equal(newCommunity.Verified(), tnc.Verified())
	s.Equal(newCommunity.Muted(), tnc.Muted())
	s.Equal(newCommunity.Joined(), tnc.Joined())
	s.Equal(newCommunity.Spectated(), tnc.Spectated())
	s.Equal(newCommunity.IsAdmin(), tnc.IsAdmin())
	s.Equal(newCommunity.InvitationOnly(), tnc.InvitationOnly())
}

// TestSyncCommunity_RequestToJoin tests more complex pairing and syncing scenario where one paired device
// makes a request to join a community
func (s *MessengerCommunitiesSuite) TestSyncCommunity_RequestToJoin() {
	// Set Alice's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "alice's-device",
		DeviceType: "alice's-device-type",
	}
	err := s.alice.SetInstallationMetadata(s.alice.installationID, aim)
	s.Require().NoError(err)

	// Create Alice's other device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	im1 := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)

	// Pair alice's two devices
	s.pairTwoDevices(alicesOtherDevice, s.alice, im1.Name, im1.DeviceType)
	s.pairTwoDevices(s.alice, alicesOtherDevice, aim.Name, aim.DeviceType)

	// Check bob the admin has only one community
	tcs2, err := s.bob.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 communities")

	// Bob the admin creates a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "new community",
		Color:       "#000000",
		Description: "new community description",
	}
	mr, err := s.bob.CreateCommunity(createCommunityReq, true)
	s.Require().NoError(err, "CreateCommunity")
	s.Require().NotNil(mr)
	s.Len(mr.Communities(), 1)

	community := mr.Communities()[0]

	// Check that admin has 2 communities
	acs, err := s.bob.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(acs, 2, "Must have 2 communities")

	// Check that Alice has only 1 community on either device
	cs, err := s.alice.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(cs, 1, "Must have 1 communities")

	tcs1, err := alicesOtherDevice.communitiesManager.All()
	s.Require().NoError(err, "alicesOtherDevice.communitiesManager.All")
	s.Len(tcs1, 1, "Must have 1 communities")

	// Bob the admin opens up a 1-1 chat with alice
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)
	s.Require().NoError(s.bob.SaveChat(chat))

	// Bob the admin shares with Alice, via public chat, an invite link to the new community
	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()
	response, err := s.bob.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities received from 1-1")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check that alice now has 2 communities
	cs, err = s.alice.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(cs, 2, "Must have 2 communities")
	for _, c := range cs {
		s.False(c.Joined(), "Must not have joined the community")
	}

	// Alice requests to join the new community
	response, err = s.alice.RequestToJoinCommunity(&requests.RequestToJoinCommunity{CommunityID: community.ID()})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Len(response.RequestsToJoinCommunity, 1)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	aRtj := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(aRtj)
	s.Equal(community.ID(), aRtj.CommunityID)
	s.True(aRtj.Our)
	s.Require().NotEmpty(aRtj.ID)
	s.Require().NotEmpty(aRtj.Clock)
	s.Equal(aRtj.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Equal(communities.RequestToJoinStatePending, aRtj.State)

	// Make sure clock is not empty
	s.Require().NotEmpty(aRtj.Clock)

	s.Len(response.Communities(), 1)
	s.Equal(response.Communities()[0].RequestedToJoinAt(), aRtj.Clock)

	// pull all communities to make sure we set RequestedToJoinAt
	allCommunities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Len(allCommunities, 2)

	if bytes.Equal(allCommunities[0].ID(), community.ID()) {
		s.Equal(allCommunities[0].RequestedToJoinAt(), aRtj.Clock)
	} else {
		s.Equal(allCommunities[1].RequestedToJoinAt(), aRtj.Clock)
	}

	// pull to make sure it has been saved
	requestsToJoin, err := s.alice.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Len(requestsToJoin, 1)

	// Make sure the requests are fetched also by community
	requestsToJoin, err = s.alice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Len(requestsToJoin, 1)

	// Alice's other device retrieves sync message from the join
	err = tt.RetryWithBackOff(func() error {
		response, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		// Do we have a new synced community?
		_, err = alicesOtherDevice.communitiesManager.GetSyncedRawCommunity(community.ID())
		if err != nil {
			return fmt.Errorf("community with sync not received %w", err)
		}

		// Do we have a new pending request to join for the new community
		requestsToJoin, err = alicesOtherDevice.PendingRequestsToJoinForCommunity(community.ID())
		if err != nil {
			return err
		}
		if len(requestsToJoin) == 0 {
			return errors.New("no requests to join")
		}

		return nil
	})
	s.Require().NoError(err)
	s.Len(response.Communities(), 1)

	// Get the pending requests to join for the new community on alicesOtherDevice
	requestsToJoin, err = alicesOtherDevice.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Len(requestsToJoin, 1)

	// Check request to join on alicesOtherDevice matches the RTJ on alice
	aodRtj := requestsToJoin[0]
	s.Equal(aRtj.PublicKey, aodRtj.PublicKey)
	s.Equal(aRtj.ID, aodRtj.ID)
	s.Equal(aRtj.CommunityID, aodRtj.CommunityID)
	s.Equal(aRtj.Clock, aodRtj.Clock)
	s.Equal(aRtj.ENSName, aodRtj.ENSName)
	s.Equal(aRtj.ChatID, aodRtj.ChatID)
	s.Equal(aRtj.State, aodRtj.State)

	// Bob the admin retrieves request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.RequestsToJoinCommunity) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Len(response.RequestsToJoinCommunity, 1)

	// Check that bob the admin's newly received request to join matches what we expect
	bobRtj := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(bobRtj)
	s.Equal(community.ID(), bobRtj.CommunityID)
	s.False(bobRtj.Our)
	s.Require().NotEmpty(bobRtj.ID)
	s.Require().NotEmpty(bobRtj.Clock)
	s.Equal(bobRtj.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Equal(communities.RequestToJoinStatePending, bobRtj.State)

	s.Equal(aRtj.PublicKey, bobRtj.PublicKey)
	s.Equal(aRtj.ID, bobRtj.ID)
	s.Equal(aRtj.CommunityID, bobRtj.CommunityID)
	s.Equal(aRtj.Clock, bobRtj.Clock)
	s.Equal(aRtj.ENSName, bobRtj.ENSName)
	s.Equal(aRtj.ChatID, bobRtj.ChatID)
	s.Equal(aRtj.State, bobRtj.State)
}

func (s *MessengerCommunitiesSuite) pairTwoDevices(device1, device2 *Messenger, deviceName, deviceType string) {
	// Send pairing data
	response, err := device1.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Len(response.Chats(), 1)
	s.False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		device2,
		func(r *MessengerResponse) bool {
			for _, installation := range r.Installations {
				if installation.ID == device1.installationID {
					return installation.InstallationMetadata != nil && deviceName == installation.InstallationMetadata.Name && deviceType == installation.InstallationMetadata.DeviceType
				}
			}
			return false

		},
		"installation not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Ensure installation is enabled
	err = device2.EnableInstallation(device1.installationID)
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestSyncCommunity_Leave() {
	// Set Alice's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "alice's-device",
		DeviceType: "alice's-device-type",
	}
	err := s.alice.SetInstallationMetadata(s.alice.installationID, aim)
	s.Require().NoError(err)

	// Create Alice's other device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.alice.identity, s.logger, nil)
	s.Require().NoError(err)

	im1 := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)

	// Pair alice's two devices
	s.pairTwoDevices(alicesOtherDevice, s.alice, im1.Name, im1.DeviceType)
	s.pairTwoDevices(s.alice, alicesOtherDevice, aim.Name, aim.DeviceType)

	// Check bob the admin has only one community
	tcs2, err := s.bob.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 communities")

	// Bob the admin creates a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "new community",
		Color:       "#000000",
		Description: "new community description",
	}
	mr, err := s.bob.CreateCommunity(createCommunityReq, true)
	s.Require().NoError(err, "CreateCommunity")
	s.Require().NotNil(mr)
	s.Len(mr.Communities(), 1)

	community := mr.Communities()[0]

	// Check that admin has 2 communities
	acs, err := s.bob.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(acs, 2, "Must have 2 communities")

	// Check that Alice has only 1 community on either device
	cs, err := s.alice.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(cs, 1, "Must have 1 communities")

	tcs1, err := alicesOtherDevice.communitiesManager.All()
	s.Require().NoError(err, "alicesOtherDevice.communitiesManager.All")
	s.Len(tcs1, 1, "Must have 1 communities")

	// Bob the admin opens up a 1-1 chat with alice
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)
	s.Require().NoError(s.bob.SaveChat(chat))

	// Bob the admin shares with Alice, via public chat, an invite link to the new community
	message := buildTestMessage(*chat)
	message.CommunityID = community.IDString()
	response, err := s.bob.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve community link & community
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities received from 1-1")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check that alice now has 2 communities
	cs, err = s.alice.communitiesManager.All()
	s.Require().NoError(err, "communitiesManager.All")
	s.Len(cs, 2, "Must have 2 communities")
	for _, c := range cs {
		s.False(c.Joined(), "Must not have joined the community")
	}

	// alice joins the community
	mr, err = s.alice.JoinCommunity(context.Background(), community.ID())
	s.Require().NoError(err, "s.alice.JoinCommunity")
	s.Require().NotNil(mr)
	s.Len(mr.Communities(), 1)
	aCom := mr.Communities()[0]

	// Check that the joined community has the correct values
	s.Equal(community.ID(), aCom.ID())
	s.Equal(uint64(0x2), aCom.Clock())
	s.Equal(community.PublicKey(), aCom.PublicKey())

	// Check alicesOtherDevice receives the sync join message
	err = tt.RetryWithBackOff(func() error {
		response, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		// Do we have a new synced community?
		_, err = alicesOtherDevice.communitiesManager.GetSyncedRawCommunity(community.ID())
		if err != nil {
			return fmt.Errorf("community with sync not received %w", err)
		}

		return nil
	})
	s.Require().NoError(err)
	s.Len(response.Communities(), 1, "")

	aoCom := mr.Communities()[0]
	s.Equal(aCom, aoCom)
}

func (s *MessengerCommunitiesSuite) TestSetMutePropertyOnChatsByCategory() {
	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "new community",
		Color:       "#000000",
		Description: "new community description",
	}

	mr, err := s.alice.CreateCommunity(createCommunityReq, true)
	s.Require().NoError(err, "s.alice.CreateCommunity")
	var newCommunity *communities.Community
	for _, com := range mr.Communities() {
		if com.Name() == createCommunityReq.Name {
			newCommunity = com
		}
	}
	s.Require().NotNil(newCommunity)

	orgChat1 := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}

	orgChat2 := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core2",
			Emoji:       "😎",
			Description: "status-core community chat2",
		},
	}

	mr, err = s.alice.CreateCommunityChat(newCommunity.ID(), orgChat1)
	s.Require().NoError(err)
	s.Require().NotNil(mr)
	s.Require().Len(mr.Communities(), 1)
	s.Require().Len(mr.Chats(), 1)

	mr, err = s.alice.CreateCommunityChat(newCommunity.ID(), orgChat2)
	s.Require().NoError(err)
	s.Require().NotNil(mr)
	s.Require().Len(mr.Communities(), 1)
	s.Require().Len(mr.Chats(), 1)

	var chatIds []string
	for k := range newCommunity.Chats() {
		chatIds = append(chatIds, k)
	}
	category := &requests.CreateCommunityCategory{
		CommunityID:  newCommunity.ID(),
		CategoryName: "category-name",
		ChatIDs:      chatIds,
	}

	mr, err = s.alice.CreateCommunityCategory(category)
	s.Require().NoError(err)
	s.Require().NotNil(mr)
	s.Require().Len(mr.Communities(), 1)
	s.Require().Len(mr.Communities()[0].Categories(), 1)

	var categoryID string
	for k := range mr.Communities()[0].Categories() {
		categoryID = k
	}

	err = s.alice.SetMutePropertyOnChatsByCategory(newCommunity.IDString(), categoryID, true)
	s.Require().NoError(err)

	for _, chat := range s.alice.Chats() {
		if chat.CategoryID == categoryID {
			s.Require().True(chat.Muted)
		}
	}

	err = s.alice.SetMutePropertyOnChatsByCategory(newCommunity.IDString(), categoryID, false)
	s.Require().NoError(err)

	for _, chat := range s.alice.Chats() {
		s.Require().False(chat.Muted)
	}
}

func (s *MessengerCommunitiesSuite) TestExtractDiscordChannelsAndCategories() {

	tmpFile, err := ioutil.TempFile(os.TempDir(), "discord-channel-")
	s.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	discordMessage := &protobuf.DiscordMessage{
		Id:              "1234",
		Type:            "Default",
		Timestamp:       "2022-07-26T14:20:17.305+00:00",
		TimestampEdited: "",
		Content:         "Some discord message",
		Author: &protobuf.DiscordMessageAuthor{
			Id:            "123",
			Name:          "TestAuthor",
			Discriminator: "456",
			Nickname:      "",
			AvatarUrl:     "",
		},
	}

	messages := make([]*protobuf.DiscordMessage, 0)
	messages = append(messages, discordMessage)

	exportedDiscordData := &discord.ExportedData{
		Channel: discord.Channel{
			ID:           "12345",
			CategoryName: "test-category",
			CategoryID:   "6789",
			Name:         "test-channel",
			Description:  "This is a channel topic",
			FilePath:     tmpFile.Name(),
		},
		Messages: messages,
	}

	data, err := json.Marshal(exportedDiscordData)
	s.Require().NoError(err)

	err = os.WriteFile(tmpFile.Name(), data, 0666) // nolint: gosec
	s.Require().NoError(err)

	files := make([]string, 0)
	files = append(files, tmpFile.Name())
	mr, errs := s.bob.ExtractDiscordChannelsAndCategories(files)
	s.Require().Len(errs, 0)

	s.Require().Len(mr.DiscordCategories, 1)
	s.Require().Len(mr.DiscordChannels, 1)
	s.Require().Equal(mr.DiscordOldestMessageTimestamp, int(1658845217))
}

func (s *MessengerCommunitiesSuite) TestExtractDiscordChannelsAndCategories_WithErrors() {

	tmpFile, err := ioutil.TempFile(os.TempDir(), "discord-channel-2")
	s.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	exportedDiscordData := &discord.ExportedData{
		Channel: discord.Channel{
			ID:           "12345",
			CategoryName: "test-category",
			CategoryID:   "6789",
			Name:         "test-channel",
			Description:  "This is a channel topic",
			FilePath:     tmpFile.Name(),
		},
		Messages: make([]*protobuf.DiscordMessage, 0),
	}

	data, err := json.Marshal(exportedDiscordData)
	s.Require().NoError(err)

	err = os.WriteFile(tmpFile.Name(), data, 0666) // nolint: gosec
	s.Require().NoError(err)

	files := make([]string, 0)
	files = append(files, tmpFile.Name())
	_, errs := s.bob.ExtractDiscordChannelsAndCategories(files)
	// Expecting 1 errors since there are no messages to be extracted
	s.Require().Len(errs, 1)
}

func (s *MessengerCommunitiesSuite) TestCommunityTokensMetadata() {

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)

	community := response.Communities()[0]

	tokensMetadata := community.CommunityTokensMetadata()
	s.Require().Len(tokensMetadata, 0)

	newToken := &protobuf.CommunityTokenMetadata{
		ContractAddresses: map[uint64]string{3: "0xasd"},
		Description:       "desc1",
		Image:             "IMG1",
		TokenType:         protobuf.CommunityTokenType_ERC721,
		Symbol:            "SMB",
	}

	_, err = community.AddCommunityTokensMetadata(newToken)
	s.Require().NoError(err)
	tokensMetadata = community.CommunityTokensMetadata()
	s.Require().Len(tokensMetadata, 1)

	s.Require().Equal(tokensMetadata[0].ContractAddresses, newToken.ContractAddresses)
	s.Require().Equal(tokensMetadata[0].Description, newToken.Description)
	s.Require().Equal(tokensMetadata[0].Image, newToken.Image)
	s.Require().Equal(tokensMetadata[0].TokenType, newToken.TokenType)
	s.Require().Equal(tokensMetadata[0].Symbol, newToken.Symbol)
	s.Require().Equal(tokensMetadata[0].Name, newToken.Name)
}

func (s *MessengerCommunitiesSuite) TestCommunityBanUserRequesToJoin() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	response, err = s.bob.InviteUsersToCommunity(
		&requests.InviteUsersToCommunity{
			CommunityID: community.ID(),
			Users:       []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	response, err = s.bob.BanUserFromCommunity(
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&s.alice.identity.PublicKey),
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().False(community.HasMember(&s.alice.identity.PublicKey))
	s.Require().True(community.IsBanned(&s.alice.identity.PublicKey))

	response, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool { return len(r.communities) > 0 },
		"no communities",
	)

	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	_, rtj, err := s.alice.communitiesManager.RequestToJoin(&s.alice.identity.PublicKey, request)

	s.Require().NoError(err)

	displayName, err := s.alice.settings.DisplayName()
	s.Require().NoError(err)

	requestToJoinProto := &protobuf.CommunityRequestToJoin{
		Clock:             rtj.Clock,
		EnsName:           rtj.ENSName,
		DisplayName:       displayName,
		CommunityId:       community.ID(),
		RevealedAddresses: make(map[string][]byte),
	}

	s.Require().NoError(err)

	messageState := s.bob.buildMessageState()

	err = s.bob.HandleCommunityRequestToJoin(messageState, &s.alice.identity.PublicKey, requestToJoinProto)

	s.Require().ErrorContains(err, "can't request access")
}
