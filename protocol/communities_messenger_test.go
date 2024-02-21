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

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/waku"
)

func TestMessengerCommunitiesSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesSuite))
}

type MessengerCommunitiesSuite struct {
	suite.Suite
	owner *Messenger
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

	s.owner = s.newMessenger()
	s.bob = s.newMessenger()
	s.alice = s.newMessenger()

	s.owner.communitiesManager.RekeyInterval = 50 * time.Millisecond

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.alice)
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSuite) newMessengerWithKey(privateKey *ecdsa.PrivateKey) *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, s.shh, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			privateKey: privateKey,
			logger:     s.logger,
		},
	})
}

func (s *MessengerCommunitiesSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(privateKey)
}

func (s *MessengerCommunitiesSuite) TestCreateCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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

	// Send a community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.alice.transport)

	inputMessage := common.NewMessage()
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
}

func (s *MessengerCommunitiesSuite) TestJoiningOpenCommunityReturnsChatsResponse() {
	ctx := context.Background()

	openCommunityDescription := &requests.CreateCommunity{
		Name:                         "open community",
		Description:                  "open community to join with no requests",
		Color:                        "#26a69a",
		HistoryArchiveSupportEnabled: true,
		Membership:                   protobuf.CommunityPermissions_AUTO_ACCEPT,
		PinMessageAllMembersEnabled:  false,
	}

	response, err := s.bob.CreateCommunity(openCommunityDescription, true)
	generalChannelChatID := response.Chats()[0].ID
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)
	s.Require().Len(response.Chats(), 1)

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
	response, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0
		},
		"message not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)

	// Alice request to join community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}

	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin := response.RequestsToJoinCommunity()[0]
	s.Require().NotNil(requestToJoin)
	s.Require().Equal(community.ID(), requestToJoin.CommunityID)
	s.Require().NotEmpty(requestToJoin.ID)
	s.Require().NotEmpty(requestToJoin.Clock)
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin.State)

	// Bobs receives the request to join and it's automatically accepted
	response, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.RequestsToJoinCommunity()) > 0
		},
		"message not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Alice receives the updated community description with channel information
	response, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.chats) > 0
		},
		"message not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Check whether community's general chat is available for Alice
	_, exists := response.chats[generalChannelChatID]
	s.Require().True(exists)
}

func (s *MessengerCommunitiesSuite) TestJoinCommunity() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
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

	// Send a community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.bob.transport)

	inputMessage := common.NewMessage()
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
	response, err = s.alice.JoinCommunity(ctx, community.ID(), false)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().True(response.Communities()[0].JoinedAt() > 0)
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
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
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

func (s *MessengerCommunitiesSuite) createCommunity() (*communities.Community, *Chat) {
	return createCommunity(&s.Suite, s.owner)
}

func (s *MessengerCommunitiesSuite) advertiseCommunityTo(community *communities.Community, owner *Messenger, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, owner, user)
}

func (s *MessengerCommunitiesSuite) joinCommunity(community *communities.Community, owner *Messenger, user *Messenger) {
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, owner, user, request, "")
}

func (s *MessengerCommunitiesSuite) TestCommunityContactCodeAdvertisement() {
	// add bob's profile keypair
	bobProfileKp := accounts.GetProfileKeypairForTest(true, false, false)
	bobProfileKp.KeyUID = s.bob.account.KeyUID
	bobProfileKp.Accounts[0].KeyUID = s.bob.account.KeyUID

	err := s.bob.settings.SaveOrUpdateKeypair(bobProfileKp)
	s.Require().NoError(err)

	// create community and make bob and alice join to it
	community, _ := s.createCommunity()
	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, s.bob)
	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, s.alice)

	s.joinCommunity(community, s.owner, s.bob)
	s.joinCommunity(community, s.owner, s.alice)

	// Trigger ContactCodeAdvertisement
	err = s.bob.SetDisplayName("bobby")
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

func (s *MessengerCommunitiesSuite) TestPostToCommunityChat() {
	community, chat := s.createCommunity()

	chatID := chat.ID
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	ctx := context.Background()

	s.advertiseCommunityTo(community, s.owner, s.alice)

	// Send message without even spectating fails
	_, err := s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().Error(err)

	// Sending a message without joining fails
	_, err = s.alice.SpectateCommunity(community.ID())
	s.Require().NoError(err)
	_, err = s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().Error(err)

	// Sending should work now
	s.joinCommunity(community, s.owner, s.alice)
	_, err = s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)

	var response *MessengerResponse
	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.owner.RetrieveAll()
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

func (s *MessengerCommunitiesSuite) TestPinMessageInCommunityChat() {
	ctx := context.Background()

	// Create a community
	description := &requests.CreateCommunity{
		Membership:                  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:                        "status",
		Color:                       "#ffffff",
		Description:                 "status community description",
		PinMessageAllMembersEnabled: true,
	}

	response, err := s.owner.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]
	s.Require().NotNil(community)
	s.Require().Equal(community.AllowsAllMembersToPinMessage(), true)

	// Create a community chat
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}
	response, err = s.owner.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
	chat := response.Chats()[0]
	s.Require().NotNil(chat)

	s.advertiseCommunityTo(community, s.owner, s.bob)
	s.joinCommunity(community, s.owner, s.bob)

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "message to be pinned"

	sendResponse, err := s.bob.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	// bob should be able to pin the message
	pinMessage := common.NewPinMessage()
	pinMessage.ChatId = chat.ID
	pinMessage.MessageId = inputMessage.ID
	pinMessage.Pinned = true
	sendResponse, err = s.bob.SendPinMessage(ctx, pinMessage)
	s.Require().NoError(err)
	s.Require().Len(sendResponse.PinMessages(), 1)

	// alice does not fully join the community,
	// so she should not be able to send the pin message
	s.advertiseCommunityTo(community, s.owner, s.alice)
	response, err = s.alice.SpectateCommunity(community.ID())
	s.Require().NotNil(response)
	s.Require().NoError(err)
	failedPinMessage := common.NewPinMessage()
	failedPinMessage.ChatId = chat.ID
	failedPinMessage.MessageId = inputMessage.ID
	failedPinMessage.Pinned = true
	sendResponse, err = s.alice.SendPinMessage(ctx, failedPinMessage)
	s.Require().Nil(sendResponse)
	s.Require().Error(err, "can't pin message")
}

func (s *MessengerCommunitiesSuite) TestImportCommunity() {
	ctx := context.Background()

	community, _ := s.createCommunity()

	category := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "category-name",
		ChatIDs:      []string{},
	}

	response, err := s.owner.CreateCommunityCategory(category)
	s.Require().NoError(err)
	community = response.Communities()[0]

	s.advertiseCommunityTo(community, s.owner, s.bob)
	s.joinCommunity(community, s.owner, s.bob)

	privateKey, err := s.owner.ExportCommunity(community.ID())
	s.Require().NoError(err)

	_, err = s.alice.ImportCommunity(ctx, privateKey)
	s.Require().NoError(err)

	newDescription := "new description set post import"
	_, err = s.alice.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
			Name:        community.Name(),
			Color:       community.Color(),
			Description: newDescription,
		},
	})
	s.Require().NoError(err)

	// bob receives new description
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].DescriptionText() == newDescription
	}, "new description not received")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestRemovePrivateKey() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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
	s.Require().True(community.IsControlNode())
	s.Require().True(community.IsControlNode())

	response, err = s.bob.RemovePrivateKey(community.ID())
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.IsOwner())
	s.Require().False(community.IsControlNode())
}

func (s *MessengerCommunitiesSuite) TestRolesAfterImportCommunity() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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
	s.Require().True(response.Communities()[0].IsControlNode())
	s.Require().True(response.Communities()[0].IsMemberOwner(&s.bob.identity.PublicKey))
	s.Require().False(response.Communities()[0].IsMemberOwner(&s.alice.identity.PublicKey))

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
	s.Require().True(response.Communities()[0].IsMemberOwner(&s.alice.identity.PublicKey))
}

func (s *MessengerCommunitiesSuite) TestRequestAccess() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, false)

	requestToJoin1 := response.RequestsToJoinCommunity()[0]
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
		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin2 := response.RequestsToJoinCommunity()[0]

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

	s.Require().Len(response.RequestsToJoinCommunity(), 1)
	s.Require().Equal(communities.RequestToJoinStateAccepted, response.RequestsToJoinCommunity()[0].State)

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
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin := response.RequestsToJoinCommunity()[0]
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

		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}

		// updating request clock by 8 days back
		requestToJoin := response.RequestsToJoinCommunity()[0]
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

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
	response, err = s.alice.CheckAndDeletePendingRequestToJoinCommunity(ctx, true)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	requestToJoin = response.RequestsToJoinCommunity()[0]
	s.Require().True(requestToJoin.Deleted)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusIdle)

	response, err = s.bob.CheckAndDeletePendingRequestToJoinCommunity(ctx, true)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	requestToJoin = response.RequestsToJoinCommunity()[0]
	s.Require().True(requestToJoin.Deleted)

	notification = response.ActivityCenterNotifications()[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().True(notification.Deleted)

	// Alice request to join community
	request = &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err = s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	aliceRequestToJoin := response.RequestsToJoinCommunity()[0]

	// Retrieve request to join and Check activity center notification for Bob
	err = tt.RetryWithBackOff(func() error {
		response, err = bobRetrieveAll()
		if err != nil {
			return err
		}
		// NOTE: we might receive multiple requests to join in case of re-transmissions
		// because request to join are hard deleted from the database, we can't check
		// whether that's an old one or a new one. So here we test for the specific id

		for _, r := range response.RequestsToJoinCommunity() {
			if bytes.Equal(r.ID, aliceRequestToJoin.ID) {
				return nil
			}
		}
		return errors.New("request to join not found")
	})
	s.Require().NoError(err)

	// Check activity center notification for Bob
	notifications, err = fetchActivityCenterNotificationsForAdmin()

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)

	notification = notifications.Notifications[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

}

/*
func (s *MessengerCommunitiesSuite) TestDeletePendingRequestAccessWithDeclinedState() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().NotEmpty(notification.ID)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Deleted, false)
	s.Require().Equal(notification.Read, true)

	requestToJoin := response.RequestsToJoinCommunity()[0]
	s.Require().NotNil(requestToJoin)
	s.Require().Equal(community.ID(), requestToJoin.CommunityID)
	s.Require().NotEmpty(requestToJoin.ID)
	s.Require().NotEmpty(requestToJoin.Clock)
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin.State)

	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(response.Communities()[0].RequestedToJoinAt(), requestToJoin.Clock)

	// Alice deletes activity center notification
	var updatedAt uint64 = 99
	_, err = s.alice.MarkActivityCenterNotificationsDeleted(ctx, []types.HexBytes{notification.ID}, updatedAt, true)
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

		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}

		// updating request clock by 8 days back
		requestToJoin := response.RequestsToJoinCommunity()[0]
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

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
	updatedAt++
	_, err = s.bob.MarkActivityCenterNotificationsDeleted(ctx, []types.HexBytes{notification.ID}, updatedAt, true)
	s.Require().NoError(err)

	// Check activity center notification for Bob after deleting
	notifications, err = fetchActivityCenterNotificationsForAdmin()
	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 0)

	// Delete pending request to join
	response, err = s.alice.CheckAndDeletePendingRequestToJoinCommunity(ctx, true)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin = response.RequestsToJoinCommunity()[0]
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	// Retrieve request to join and Check activity center notification for Bob
	err = tt.RetryWithBackOff(func() error {
		response, err = bobRetrieveAll()
		if err != nil {
			return err
		}

		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}

		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}

		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	// Check activity center notification for Bob
	notifications, err = fetchActivityCenterNotificationsForAdmin()

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 1)

	notification = notifications.Notifications[0]
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityMembershipRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().False(notification.Deleted)

}
*/

/*
func (s *MessengerCommunitiesSuite) TestCancelRequestAccess() {
	ctx := context.Background()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin1 := response.RequestsToJoinCommunity()[0]
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
		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}
		if len(response.ActivityCenterNotifications()) == 0 {
			return errors.New("request to join community notification not added in activity center")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin2 := response.RequestsToJoinCommunity()[0]

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
	response, err = s.alice.CancelRequestToJoinCommunity(ctx, requestToCancel)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)
	s.Require().Equal(communities.RequestToJoinStateCanceled, response.RequestsToJoinCommunity()[0].State)

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
		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	// Retrieve activity center notifications for admin to make sure the request notification is deleted
	notifications, err := s.bob.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	})

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 0)

	cancelRequestToJoin2 := response.RequestsToJoinCommunity()[0]

	s.Require().NotNil(cancelRequestToJoin2)
	s.Require().Equal(community.ID(), cancelRequestToJoin2.CommunityID)
	s.Require().False(cancelRequestToJoin2.Our)
	s.Require().NotEmpty(cancelRequestToJoin2.ID)
	s.Require().NotEmpty(cancelRequestToJoin2.Clock)
	s.Require().Equal(cancelRequestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStateCanceled, cancelRequestToJoin2.State)

}
*/

func (s *MessengerCommunitiesSuite) TestRequestAccessAgain() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)
	s.Require().Equal(notification.Read, true)
	s.Require().Equal(notification.Accepted, false)
	s.Require().Equal(notification.Dismissed, false)

	requestToJoin1 := response.RequestsToJoinCommunity()[0]
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
		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin2 := response.RequestsToJoinCommunity()[0]

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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin3 := response.RequestsToJoinCommunity()[0]
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
	response, err = WaitOnMessengerResponse(s.bob,
		func(r *MessengerResponse) bool {
			return len(r.RequestsToJoinCommunity()) == 1
		},
		"request to join community was never 1",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin4 := response.RequestsToJoinCommunity()[0]

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
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin1 := response.RequestsToJoinCommunity()[0]
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
		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	// Check if admin sees requests correctly
	requestsToJoin, err := s.bob.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 1)

	requestsToJoin, err = s.bob.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(requestsToJoin, 0)

	requestToJoin2 := response.RequestsToJoinCommunity()[0]

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
	community, _ := s.createCommunity()
	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, s.alice)
	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, s.bob)

	s.joinCommunity(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.bob)

	joinedCommunities, err := s.owner.communitiesManager.Joined()
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
		} else if !response.Communities()[0].HasMember(&s.owner.identity.PublicKey) {
			communityMembersError = errors.New("admin removed from community")
		} else if !response.Communities()[0].HasMember(&s.bob.identity.PublicKey) {
			communityMembersError = errors.New("bob removed from community")
		} else if response.Communities()[0].HasMember(&s.alice.identity.PublicKey) {
			communityMembersError = errors.New("alice not removed from community")
		}

		return communityMembersError
	}
	err = tt.RetryWithBackOff(func() error {
		return verifyCommunityMembers(s.owner)
	})
	s.Require().NoError(err)
	err = tt.RetryWithBackOff(func() error {
		return verifyCommunityMembers(s.bob)
	})
	s.Require().NoError(err)

	joinedCommunities, err = s.owner.communitiesManager.Joined()
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
	s.joinCommunity(community, s.owner, s.alice)

	joinedCommunities, err = s.owner.communitiesManager.Joined()
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
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
		Name:        "status",
		Description: "status community description",
		Color:       "#FFFFFF",
		Image:       "../_assets/tests/status.png",
		ImageAx:     0,
		ImageAy:     0,
		ImageBx:     256,
		ImageBy:     256,
		Banner: images.CroppedImage{
			ImagePath: "../_assets/tests/IMG_1205.HEIC.jpg",
			X:         0,
			Y:         0,
			Width:     160,
			Height:    90,
		},
	}

	response, err := s.owner.CreateCommunity(description, true)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
	community := response.Communities()[0]

	inputMessageText := "Come on alice, You'll like it here!"
	// Alice shares community with Bob
	response, err = s.owner.ShareCommunity(&requests.ShareCommunity{
		CommunityID:   community.ID(),
		Users:         []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
		InviteMessage: inputMessageText,
	})

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Messages(), 1)
	sentMessageText := response.Messages()[0].Text

	_, err = WaitOnMessengerResponse(s.alice, func(r *MessengerResponse) bool {
		return len(r.Messages()) > 0
	}, "Messages not received")

	communityURL := response.Messages()[0].UnfurledStatusLinks.GetUnfurledStatusLinks()[0].Url
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(fmt.Sprintf("%s\n%s", inputMessageText, communityURL), sentMessageText)
	s.Require().NotNil(response.Messages()[0].UnfurledStatusLinks.GetUnfurledStatusLinks()[0].GetCommunity().CommunityId)
}

func (s *MessengerCommunitiesSuite) TestShareCommunityWithPreviousMember() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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

	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
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

	community = response.Communities()[0]
	communityChat := response.Chats()[0]

	// Add Alice to the community before sharing it
	_, err = community.AddMember(&s.alice.identity.PublicKey, []protobuf.CommunityMember_Roles{})
	s.Require().NoError(err)

	err = s.bob.communitiesManager.SaveCommunity(community)
	s.Require().NoError(err)

	advertiseCommunityToUserOldWay(&s.Suite, community, s.bob, s.alice)

	// Add bob to contacts so it does not go on activity center
	bobPk := common.PubkeyToHex(&s.bob.identity.PublicKey)
	request := &requests.AddContact{ID: bobPk}
	_, err = s.alice.AddContact(context.Background(), request)
	s.Require().NoError(err)

	// Alice should have the Joined status for the community
	communityInResponse := response.Communities()[0]
	s.Require().Equal(community.ID(), communityInResponse.ID())
	s.Require().True(communityInResponse.Joined())

	// Alice is able to receive messages in the community
	inputMessage := buildTestMessage(*communityChat)
	sendResponse, err := s.bob.SendChatMessage(context.Background(), inputMessage)
	messageID := sendResponse.Messages()[0].ID
	s.NoError(err)
	s.Require().Len(sendResponse.Messages(), 1)

	response, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool { return len(r.messages) > 0 },
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(messageID, response.Messages()[0].ID)
}

func (s *MessengerCommunitiesSuite) TestBanUser() {
	community, _ := s.createCommunity()

	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	response, err := s.owner.BanUserFromCommunity(
		context.Background(),
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
	s.Require().Len(community.PendingAndBannedMembers(), 1)

	response, err = s.owner.UnbanUserFromCommunity(
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

func (s *MessengerCommunitiesSuite) createOtherDevice(m1 *Messenger) *Messenger {
	m2 := s.newMessengerWithKey(m1.identity)

	tcs, err := m2.communitiesManager.All()
	s.Require().NoError(err, "m2.communitiesManager.All")
	s.Len(tcs, 1, "Must have 1 communities")

	// Pair devices
	metadata := &multidevice.InstallationMetadata{
		Name:       "other-device",
		DeviceType: "other-device-type",
	}
	err = m2.SetInstallationMetadata(m2.installationID, metadata)
	s.Require().NoError(err)

	_, err = m2.Start()
	s.Require().NoError(err)

	return m2
}

func (s *MessengerCommunitiesSuite) TestSyncCommunitySettings() {
	// Create new device
	alicesOtherDevice := s.createOtherDevice(s.alice)
	PairDevices(&s.Suite, alicesOtherDevice, s.alice)

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	alicesOtherDevice := s.createOtherDevice(s.alice)
	PairDevices(&s.Suite, alicesOtherDevice, s.alice)

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	alicesOtherDevice := s.createOtherDevice(s.alice)
	PairDevices(&s.Suite, alicesOtherDevice, s.alice)

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	tcs, err := alicesOtherDevice.communitiesManager.All()
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

	// Private Key for synced community should be null
	s.Require().NotNil(newCommunity.PrivateKey())
	s.Require().Nil(tnc.PrivateKey())

	s.Equal(newCommunity.PublicKey(), tnc.PublicKey())
	s.Equal(newCommunity.Verified(), tnc.Verified())
	s.Equal(newCommunity.Muted(), tnc.Muted())
	s.Equal(newCommunity.Joined(), tnc.Joined())
	s.Equal(newCommunity.Spectated(), tnc.Spectated())

	s.True(newCommunity.IsControlNode())
	s.True(newCommunity.IsOwner())

	// Even though synced device have the private key, it is not the control node
	// There can be only one control node
	s.False(tnc.IsControlNode())
	s.True(tnc.IsOwner())
}

func (s *MessengerCommunitiesSuite) TestSyncCommunity_EncryptionKeys() {
	// Create new device
	ownersOtherDevice := s.createOtherDevice(s.owner)
	defer TearDownMessenger(&s.Suite, ownersOtherDevice)

	PairDevices(&s.Suite, ownersOtherDevice, s.owner)

	community, chat := s.createCommunity()
	s.owner.communitiesManager.RekeyInterval = 1 * time.Hour

	{ // ensure both community and channel are encrypted
		permissionRequest := requests.CreateCommunityTokenPermission{
			CommunityID: community.ID(),
			Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
			TokenCriteria: []*protobuf.TokenCriteria{
				&protobuf.TokenCriteria{
					Type:              protobuf.CommunityTokenType_ERC20,
					ContractAddresses: map[uint64]string{testChainID1: "0x123"},
					Symbol:            "TEST",
					Amount:            "100",
					Decimals:          uint64(18),
				},
			},
		}
		_, err := s.owner.CreateCommunityTokenPermission(&permissionRequest)
		s.Require().NoError(err)

		channelPermissionRequest := requests.CreateCommunityTokenPermission{
			CommunityID: community.ID(),
			Type:        protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
			TokenCriteria: []*protobuf.TokenCriteria{
				&protobuf.TokenCriteria{
					Type:              protobuf.CommunityTokenType_ERC20,
					ContractAddresses: map[uint64]string{testChainID1: "0x123"},
					Symbol:            "TEST",
					Amount:            "100",
					Decimals:          uint64(18),
				},
			},
			ChatIds: []string{chat.ID},
		}

		_, err = s.owner.CreateCommunityTokenPermission(&channelPermissionRequest)
		s.Require().NoError(err)
	}

	getKeysCount := func(m *Messenger) (communityKeysCount int, channelKeysCount int) {
		keys, err := m.encryptor.GetAllHRKeys(community.ID())
		s.Require().NoError(err)
		if keys != nil {
			communityKeysCount = len(keys.Keys)
		}

		channelKeys, err := m.encryptor.GetAllHRKeys([]byte(community.IDString() + chat.CommunityChatID()))
		s.Require().NoError(err)
		if channelKeys != nil {
			channelKeysCount = len(channelKeys.Keys)
		}
		return
	}

	communityKeysCount, channelKeysCount := getKeysCount(s.owner)
	s.Require().GreaterOrEqual(communityKeysCount, 1)
	s.Require().GreaterOrEqual(channelKeysCount, 1)

	// ensure both community and channel keys are synced
	_, err := WaitOnMessengerResponse(ownersOtherDevice, func(mr *MessengerResponse) bool {
		communityKeysCount, channelKeysCount := getKeysCount(s.owner)
		syncedCommunityKeysCount, syncedChannelKeysCount := getKeysCount(ownersOtherDevice)

		return communityKeysCount == syncedCommunityKeysCount && channelKeysCount == syncedChannelKeysCount
	}, "keys not synced")
	s.Require().NoError(err)
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
	alicesOtherDevice := s.createOtherDevice(s.alice)

	// Pair alice's two devices
	PairDevices(&s.Suite, alicesOtherDevice, s.alice)
	PairDevices(&s.Suite, s.alice, alicesOtherDevice)

	// Check bob the admin has only one community
	tcs2, err := s.bob.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 communities")

	// Bob the admin creates a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	aRtj := response.RequestsToJoinCommunity()[0]
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
		if len(response.RequestsToJoinCommunity()) == 0 {
			return errors.New("request to join community not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Len(response.RequestsToJoinCommunity(), 1)

	// Check that bob the admin's newly received request to join matches what we expect
	bobRtj := response.RequestsToJoinCommunity()[0]
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

func (s *MessengerCommunitiesSuite) TestSyncCommunity_Leave() {
	// Set Alice's installation metadata
	aim := &multidevice.InstallationMetadata{
		Name:       "alice's-device",
		DeviceType: "alice's-device-type",
	}
	err := s.alice.SetInstallationMetadata(s.alice.installationID, aim)
	s.Require().NoError(err)

	// Create Alice's other device
	alicesOtherDevice := s.createOtherDevice(s.alice)

	// Pair alice's two devices
	PairDevices(&s.Suite, alicesOtherDevice, s.alice)
	PairDevices(&s.Suite, s.alice, alicesOtherDevice)

	// Check bob the admin has only one community
	tcs2, err := s.bob.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 communities")

	// Bob the admin creates a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
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
	mr, err = s.alice.JoinCommunity(context.Background(), community.ID(), false)
	s.Require().NoError(err, "s.alice.JoinCommunity")
	s.Require().NotNil(mr)
	s.Len(mr.Communities(), 1)
	aCom := mr.Communities()[0]

	// Check that the joined community has the correct values
	s.Equal(community.ID(), aCom.ID())
	s.Equal(community.Clock(), aCom.Clock())
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

func (s *MessengerCommunitiesSuite) TestSyncCommunity_ImportCommunity() {
	// Owner creates community
	community, _ := s.createCommunity()
	s.Require().True(community.IsControlNode())

	// New device is created & paired
	ownersOtherDevice := s.createOtherDevice(s.owner)
	PairDevices(&s.Suite, ownersOtherDevice, s.owner)
	PairDevices(&s.Suite, s.owner, ownersOtherDevice)

	privateKey, err := s.owner.ExportCommunity(community.ID())
	s.Require().NoError(err)

	// New device imports the community (before it is received via sync message)
	ctx := context.Background()
	response, err := ownersOtherDevice.ImportCommunity(ctx, privateKey)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	s.Require().Equal(community.IDString(), response.Communities()[0].IDString())
	// New device becomes the control node
	s.Require().True(response.Communities()[0].IsControlNode())

	// Old device is no longer the control node
	_, err = WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		if len(response.Communities()) != 1 {
			return false
		}
		c := response.Communities()[0]
		return c.IDString() == community.IDString() && !c.IsControlNode()
	}, "community not synced")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestSetMutePropertyOnChatsByCategory() {
	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}

	orgChat2 := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
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

	err = s.alice.SetMutePropertyOnChatsByCategory(&requests.MuteCategory{
		CommunityID: newCommunity.IDString(),
		CategoryID:  categoryID,
		MutedType:   MuteTillUnmuted,
	}, true)
	s.Require().NoError(err)

	for _, chat := range s.alice.Chats() {
		if chat.CategoryID == categoryID {
			s.Require().True(chat.Muted)
		}
	}

	err = s.alice.SetMutePropertyOnChatsByCategory(&requests.MuteCategory{
		CommunityID: newCommunity.IDString(),
		CategoryID:  categoryID,
		MutedType:   Unmuted,
	}, false)
	s.Require().NoError(err)

	for _, chat := range s.alice.Chats() {
		s.Require().False(chat.Muted)
	}
}

func (s *MessengerCommunitiesSuite) TestCheckCommunitiesToUnmute() {
	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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

	currTime, err := time.Parse(time.RFC3339, time.Now().Add(-time.Hour).Format(time.RFC3339))
	s.Require().NoError(err)

	err = s.alice.communitiesManager.SetMuted(newCommunity.ID(), true)
	s.Require().NoError(err, "SetMuted to community")

	err = s.alice.communitiesManager.MuteCommunityTill(newCommunity.ID(), currTime)
	s.Require().NoError(err, "SetMuteTill to community")

	response, err := s.alice.CheckCommunitiesToUnmute()
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1, "CheckCommunitiesToUnmute should unmute the community")

	community, err := s.alice.communitiesManager.GetByID(newCommunity.ID())
	s.Require().NoError(err)
	s.Require().False(community.Muted())

}

func (s *MessengerCommunitiesSuite) TestCommunityNotInDB() {
	community, err := s.alice.communitiesManager.GetByID([]byte("0x123"))
	s.Require().ErrorIs(err, communities.ErrOrgNotFound)
	s.Require().Nil(community)
}

func (s *MessengerCommunitiesSuite) TestMuteAllCommunityChats() {
	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}

	orgChat2 := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
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

	muteDuration, err := s.alice.MuteDuration(MuteFor15Min)
	s.Require().NoError(err)

	time, err := s.alice.MuteAllCommunityChats(&requests.MuteCommunity{
		CommunityID: newCommunity.ID(),
		MutedType:   MuteFor15Min,
	})
	s.Require().NoError(err)
	s.Require().NotNil(time)

	aliceCommunity, err := s.alice.GetCommunityByID(newCommunity.ID())
	s.Require().NoError(err)
	s.Require().True(aliceCommunity.Muted())

	for _, chat := range s.alice.Chats() {
		if chat.CommunityID == newCommunity.IDString() {
			s.Require().True(chat.Muted)
			s.Require().Equal(chat.MuteTill, muteDuration)
		}
	}

	for _, chat := range s.alice.Chats() {
		if chat.CommunityID == newCommunity.IDString() {
			err = s.alice.UnmuteChat(chat.ID)
			s.Require().NoError(err)
			s.Require().False(chat.Muted)
			break
		}
	}

	aliceCommunity, err = s.alice.GetCommunityByID(newCommunity.ID())
	s.Require().NoError(err)
	s.Require().False(aliceCommunity.Muted())

	time, err = s.alice.UnMuteAllCommunityChats(newCommunity.IDString())
	s.Require().NoError(err)
	s.Require().NotNil(time)
	s.Require().False(newCommunity.Muted())

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

func (s *MessengerCommunitiesSuite) TestCommunityBanUserRequestToJoin() {
	community, _ := s.createCommunity()

	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	response, err := s.owner.BanUserFromCommunity(
		context.Background(),
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
	rtj := s.alice.communitiesManager.CreateRequestToJoin(request)

	s.Require().NoError(err)

	displayName, err := s.alice.settings.DisplayName()
	s.Require().NoError(err)

	requestToJoinProto := &protobuf.CommunityRequestToJoin{
		Clock:            rtj.Clock,
		EnsName:          rtj.ENSName,
		DisplayName:      displayName,
		CommunityId:      community.ID(),
		RevealedAccounts: make([]*protobuf.RevealedAccount, 0),
	}

	s.Require().NoError(err)

	messageState := s.owner.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{}

	messageState.CurrentMessageState.PublicKey = &s.alice.identity.PublicKey

	statusMessage := v1protocol.StatusMessage{}
	statusMessage.TransportLayer.Dst = community.PublicKey()
	err = s.owner.HandleCommunityRequestToJoin(messageState, requestToJoinProto, &statusMessage)

	s.Require().ErrorContains(err, "can't request access")
}

func (s *MessengerCommunitiesSuite) TestHandleImport() {
	community, chat := s.createCommunity()

	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	// Check that there are no messages in the chat at first
	chat, err := s.alice.persistence.Chat(chat.ID)
	s.Require().NoError(err)
	s.Require().NotNil(chat)
	s.Require().Equal(0, int(chat.UnviewedMessagesCount))

	// Create an message that will be imported
	testMessage := protobuf.ChatMessage{
		Text:        "abc123",
		ChatId:      chat.ID,
		ContentType: protobuf.ChatMessage_TEXT_PLAIN,
		MessageType: protobuf.MessageType_COMMUNITY_CHAT,
		Clock:       1,
		Timestamp:   1,
	}
	encodedPayload, err := proto.Marshal(&testMessage)
	s.Require().NoError(err)
	wrappedPayload, err := v1protocol.WrapMessageV1(
		encodedPayload,
		protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		s.owner.identity,
	)
	s.Require().NoError(err)

	message := &types.Message{}
	message.Sig = crypto.FromECDSAPub(&s.owner.identity.PublicKey)
	message.Payload = wrappedPayload

	filter := s.alice.transport.FilterByChatID(chat.ID)
	importedMessages := make(map[transport.Filter][]*types.Message, 0)

	importedMessages[*filter] = append(importedMessages[*filter], message)

	// Import that message
	err = s.alice.handleImportedMessages(importedMessages)
	s.Require().NoError(err)

	// Get the chat again and see that there is still no unread message because we don't count import messages
	chat, err = s.alice.persistence.Chat(chat.ID)
	s.Require().NoError(err)
	s.Require().NotNil(chat)
	s.Require().Equal(0, int(chat.UnviewedMessagesCount))
}

func (s *MessengerCommunitiesSuite) TestGetCommunityIdFromKey() {
	publicKey := "0x029e4777ce55f20373db33546c8681a082bd181d665c87e18d4306766de9302b53"
	privateKey := "0x3f932031cb5f94ba7eb8ab4c824c3677973ab01fde65d1b89e0b3f470003a2cd"

	// Public key returns the same
	communityID := GetCommunityIDFromKey(publicKey)
	s.Require().Equal(communityID, publicKey)

	// Private key returns the public key
	communityID = GetCommunityIDFromKey(privateKey)
	s.Require().Equal(communityID, publicKey)
}

type testPermissionChecker struct {
}

func (t *testPermissionChecker) CheckPermissionToJoin(*communities.Community, []gethcommon.Address) (*communities.CheckPermissionToJoinResponse, error) {
	return &communities.CheckPermissionsResponse{Satisfied: true}, nil

}
func (t *testPermissionChecker) CheckPermissions(permissions []*communities.CommunityTokenPermission, accountsAndChainIDs []*communities.AccountChainIDsCombination, shortcircuit bool) (*communities.CheckPermissionsResponse, error) {
	return &communities.CheckPermissionsResponse{Satisfied: true}, nil
}

func (s *MessengerCommunitiesSuite) TestStartCommunityRekeyLoop() {
	community, chat := createEncryptedCommunity(&s.Suite, s.owner)
	s.Require().True(community.Encrypted())
	s.Require().True(community.ChannelEncrypted(chat.CommunityChatID()))

	s.owner.communitiesManager.PermissionChecker = &testPermissionChecker{}

	s.advertiseCommunityTo(community, s.owner, s.bob)
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.bob)
	s.joinCommunity(community, s.owner, s.alice)

	// Check keys in the database
	communityKeys, err := s.owner.sender.GetKeysForGroup(community.ID())
	s.Require().NoError(err)
	communityKeyCount := len(communityKeys)

	channelKeys, err := s.owner.sender.GetKeysForGroup([]byte(chat.ID))
	s.Require().NoError(err)
	channelKeyCount := len(channelKeys)

	// Check that rekeying is occurring by counting the number of keyIDs in the encryptor's DB
	// This test could be flaky, as the rekey function may not be finished before RekeyInterval * 2 has passed
	for i := 0; i < 5; i++ {
		time.Sleep(s.owner.communitiesManager.RekeyInterval * 2)
		communityKeys, err = s.owner.sender.GetKeysForGroup(community.ID())
		s.Require().NoError(err)
		s.Require().Greater(len(communityKeys), communityKeyCount)
		communityKeyCount = len(communityKeys)

		channelKeys, err = s.owner.sender.GetKeysForGroup([]byte(chat.ID))
		s.Require().NoError(err)
		s.Require().Greater(len(channelKeys), channelKeyCount)
		channelKeyCount = len(channelKeys)
	}
}

func (s *MessengerCommunitiesSuite) TestCommunityRekeyAfterBan() {
	s.T().Skip("flaky test")

	s.owner.communitiesManager.RekeyInterval = 500 * time.Minute

	_, err := s.owner.Start()
	s.Require().NoError(err)

	// Create a new community
	response, err := s.owner.CreateCommunity(
		&requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
			Name:        "status",
			Color:       "#57a7e5",
			Description: "status community description",
		},
		true,
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].Members(), 1)

	// Check community is present in the DB and has default values we care about
	c, err := s.owner.GetCommunityByID(response.Communities()[0].ID())
	s.Require().NoError(err)
	s.Require().False(c.Encrypted())
	// TODO some check that there are no keys for the community. Alt for s.Require().Zero(c.RekeyedAt().Unix())

	_, err = s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID: c.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{{
			ContractAddresses: map[uint64]string{3: "0x933"},
			Type:              protobuf.CommunityTokenType_ERC20,
			Symbol:            "STT",
			Name:              "Status Test Token",
			Amount:            "10",
			Decimals:          18,
		}},
	})
	s.Require().NoError(err)

	c, err = s.owner.GetCommunityByID(c.ID())
	s.Require().NoError(err)
	s.Require().True(c.Encrypted())

	s.advertiseCommunityTo(c, s.owner, s.bob)
	s.advertiseCommunityTo(c, s.owner, s.alice)

	s.owner.communitiesManager.PermissionChecker = &testPermissionChecker{}

	s.joinCommunity(c, s.owner, s.bob)
	s.joinCommunity(c, s.owner, s.alice)

	// Check the Alice and Bob are members of the community
	c, err = s.owner.GetCommunityByID(c.ID())
	s.Require().NoError(err)
	s.Require().True(c.HasMember(&s.alice.identity.PublicKey))
	s.Require().True(c.HasMember(&s.bob.identity.PublicKey))

	// Make sure at least one key makes it to alice
	response, err = WaitOnMessengerResponse(s.alice,
		func(r *MessengerResponse) bool {
			keys, err := s.alice.encryptor.GetKeysForGroup(response.Communities()[0].ID())
			if err != nil || len(keys) != 1 {
				return false
			}
			return true

		},
		"alice does not have enough keys",
	)
	s.Require().NoError(err)

	response, err = s.owner.BanUserFromCommunity(context.Background(), &requests.BanUserFromCommunity{
		CommunityID: c.ID(),
		User:        common.PubkeyToHexBytes(&s.bob.identity.PublicKey),
	})
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	s.Require().False(response.Communities()[0].HasMember(&s.bob.identity.PublicKey))

	// Check bob has been banned
	response, err = WaitOnMessengerResponse(s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) == 1 && !r.Communities()[0].HasMember(&s.bob.identity.PublicKey)

		},
		"alice didn't receive updated description",
	)
	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(s.alice,
		func(r *MessengerResponse) bool {
			keys, err := s.alice.encryptor.GetKeysForGroup(response.Communities()[0].ID())
			if err != nil || len(keys) < 2 {
				return false
			}
			return true

		},
		"alice hasn't received updated key",
	)
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestCommunityRekeyAfterBanDisableCompatibility() {
	common.RekeyCompatibility = false
	s.owner.communitiesManager.RekeyInterval = 500 * time.Minute

	_, err := s.owner.Start()
	s.Require().NoError(err)

	// Create a new community
	response, err := s.owner.CreateCommunity(
		&requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
			Name:        "status",
			Color:       "#57a7e5",
			Description: "status community description",
		},
		true,
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// Check community is present in the DB and has default values we care about
	c, err := s.owner.GetCommunityByID(response.Communities()[0].ID())
	s.Require().NoError(err)
	s.Require().False(c.Encrypted())
	// TODO some check that there are no keys for the community. Alt for s.Require().Zero(c.RekeyedAt().Unix())

	_, err = s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID: c.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{{
			ContractAddresses: map[uint64]string{3: "0x933"},
			Type:              protobuf.CommunityTokenType_ERC20,
			Symbol:            "STT",
			Name:              "Status Test Token",
			Amount:            "10",
			Decimals:          18,
		}},
	})
	s.Require().NoError(err)

	c, err = s.owner.GetCommunityByID(c.ID())
	s.Require().NoError(err)
	s.Require().True(c.Encrypted())

	s.advertiseCommunityTo(c, s.owner, s.bob)
	s.advertiseCommunityTo(c, s.owner, s.alice)

	s.owner.communitiesManager.PermissionChecker = &testPermissionChecker{}

	s.joinCommunity(c, s.owner, s.bob)
	s.joinCommunity(c, s.owner, s.alice)

	// Check the Alice and Bob are members of the community
	c, err = s.owner.GetCommunityByID(c.ID())
	s.Require().NoError(err)
	s.Require().True(c.HasMember(&s.alice.identity.PublicKey))
	s.Require().True(c.HasMember(&s.bob.identity.PublicKey))

	// Make sure at least one key makes it to alice
	response, err = WaitOnMessengerResponse(s.alice,
		func(r *MessengerResponse) bool {
			keys, err := s.alice.encryptor.GetKeysForGroup(response.Communities()[0].ID())
			if err != nil || len(keys) != 1 {
				return false
			}
			return true

		},
		"alice does not have enough keys",
	)
	s.Require().NoError(err)

	response, err = s.owner.BanUserFromCommunity(context.Background(), &requests.BanUserFromCommunity{
		CommunityID: c.ID(),
		User:        common.PubkeyToHexBytes(&s.bob.identity.PublicKey),
	})
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	s.Require().False(response.Communities()[0].HasMember(&s.bob.identity.PublicKey))

	// Check bob has been banned
	response, err = WaitOnMessengerResponse(s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) == 1 && !r.Communities()[0].HasMember(&s.bob.identity.PublicKey)

		},
		"alice didn't receive updated description",
	)
	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(s.alice,
		func(r *MessengerResponse) bool {
			keys, err := s.alice.encryptor.GetKeysForGroup(response.Communities()[0].ID())
			if err != nil || len(keys) < 2 {
				return false
			}
			return true

		},
		"alice hasn't received updated key",
	)
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestRetrieveBigCommunity() {
	bigEmoji := make([]byte, 4*1024*1024) // 4 MB
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
		Emoji:       string(bigEmoji),
	}

	// checks that private messages are segmented
	// (community is advertised through `SendPrivate`)
	response, err := s.owner.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	community := response.Communities()[0]

	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	// checks that public messages are segmented
	// (community is advertised through `SendPublic`)
	updatedDescription := "status updated community description"
	_, err = s.owner.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
			Name:        "status",
			Color:       "#ffffff",
			Description: updatedDescription,
			Emoji:       string(bigEmoji),
		},
	})
	s.Require().NoError(err)

	// alice receives updated description
	_, err = WaitOnMessengerResponse(s.alice, func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].DescriptionText() == updatedDescription
	}, "updated description not received")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestRequestAndCancelCommunityAdminOffline() {
	ctx := context.Background()

	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err := s.alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin1 := response.RequestsToJoinCommunity()[0]
	s.Require().NotNil(requestToJoin1)
	s.Require().Equal(community.ID(), requestToJoin1.CommunityID)
	s.Require().True(requestToJoin1.Our)
	s.Require().NotEmpty(requestToJoin1.ID)
	s.Require().NotEmpty(requestToJoin1.Clock)
	s.Require().Equal(requestToJoin1.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	messageState := s.alice.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{}

	messageState.CurrentMessageState.PublicKey = &s.alice.identity.PublicKey

	statusMessage := v1protocol.StatusMessage{}
	statusMessage.TransportLayer.Dst = community.PublicKey()

	requestToJoinProto := &protobuf.CommunityRequestToJoin{
		Clock:       requestToJoin1.Clock,
		EnsName:     requestToJoin1.ENSName,
		DisplayName: "Alice",
		CommunityId: community.ID(),
	}

	err = s.owner.HandleCommunityRequestToJoin(messageState, requestToJoinProto, &statusMessage)
	s.Require().NoError(err)
	ownerCommunity, err := s.owner.GetCommunityByID(community.ID())
	// Check Alice has successfully joined at owner side, Because message order was correct
	s.Require().True(ownerCommunity.HasMember(s.alice.IdentityPublicKey()))
	s.Require().NoError(err)

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

	requestToJoin2 := response.RequestsToJoinCommunity()[0]

	s.Require().NotNil(requestToJoin2)
	s.Require().Equal(community.ID(), requestToJoin2.CommunityID)
	s.Require().NotEmpty(requestToJoin2.ID)
	s.Require().NotEmpty(requestToJoin2.Clock)
	s.Require().Equal(requestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin2.State)

	s.Require().Equal(requestToJoin1.ID, requestToJoin2.ID)

	requestToCancel := &requests.CancelRequestToJoinCommunity{ID: requestToJoin1.ID}
	response, err = s.alice.CancelRequestToJoinCommunity(ctx, requestToCancel)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)
	s.Require().Equal(communities.RequestToJoinStateCanceled, response.RequestsToJoinCommunity()[0].State)

	messageState = s.alice.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{}

	messageState.CurrentMessageState.PublicKey = &s.alice.identity.PublicKey

	statusMessage.TransportLayer.Dst = community.PublicKey()

	requestToJoinCancelProto := &protobuf.CommunityRequestToJoinResponse{
		CommunityId: community.ID(),
		Clock:       requestToJoin1.Clock + 1,
		Accepted:    true,
	}

	err = s.alice.HandleCommunityRequestToJoinResponse(messageState, requestToJoinCancelProto, &statusMessage)
	s.Require().NoError(err)
	aliceJoinedCommunities, err := s.alice.JoinedCommunities()
	s.Require().NoError(err)
	// Make sure on Alice side she hasn't joined any communities
	s.Require().Empty(aliceJoinedCommunities)

	// pull to make sure it has been saved
	cancelRequestsToJoin, err := s.alice.MyCanceledRequestToJoinForCommunityID(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(cancelRequestsToJoin)
	s.Require().Equal(cancelRequestsToJoin.State, communities.RequestToJoinStateCanceled)

	s.Require().NoError(err)

	messageState = s.alice.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{}

	messageState.CurrentMessageState.PublicKey = &s.alice.identity.PublicKey

	statusMessage.TransportLayer.Dst = community.PublicKey()

	requestToJoinResponseProto := &protobuf.CommunityRequestToJoinResponse{
		Clock:       cancelRequestsToJoin.Clock,
		CommunityId: community.ID(),
		Accepted:    true,
	}

	err = s.alice.HandleCommunityRequestToJoinResponse(messageState, requestToJoinResponseProto, &statusMessage)
	s.Require().NoError(err)
	// Make sure alice is NOT a member of the community that she cancelled her request to join to
	s.Require().False(community.HasMember(s.alice.IdentityPublicKey()))
	// Make sure there are no AC notifications for Alice
	aliceNotifications, err := s.alice.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	})
	s.Require().NoError(err)
	s.Require().Len(aliceNotifications.Notifications, 0)

	// Retrieve activity center notifications for admin to make sure the request notification is deleted
	notifications, err := s.owner.ActivityCenterNotifications(ActivityCenterNotificationsRequest{
		Cursor:        "",
		Limit:         10,
		ActivityTypes: []ActivityCenterType{},
		ReadType:      ActivityCenterQueryParamsReadUnread,
	})

	s.Require().NoError(err)
	s.Require().Len(notifications.Notifications, 0)
	cancelRequestToJoin2 := response.RequestsToJoinCommunity()[0]
	s.Require().NotNil(cancelRequestToJoin2)
	s.Require().Equal(community.ID(), cancelRequestToJoin2.CommunityID)
	s.Require().False(cancelRequestToJoin2.Our)
	s.Require().NotEmpty(cancelRequestToJoin2.ID)
	s.Require().NotEmpty(cancelRequestToJoin2.Clock)
	s.Require().Equal(cancelRequestToJoin2.PublicKey, common.PubkeyToHex(&s.alice.identity.PublicKey))
}

func (s *MessengerCommunitiesSuite) TestCommunityLastOpenedAt() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	// Mock frontend triggering communityUpdateLastOpenedAt
	lastOpenedAt1, err := s.alice.CommunityUpdateLastOpenedAt(community.IDString())
	s.Require().NoError(err)

	// Check lastOpenedAt was updated
	s.Require().True(lastOpenedAt1 > 0)

	// Nap for a bit
	time.Sleep(time.Second)

	// Check lastOpenedAt was successfully updated twice
	lastOpenedAt2, err := s.alice.CommunityUpdateLastOpenedAt(community.IDString())
	s.Require().NoError(err)

	s.Require().True(lastOpenedAt2 > lastOpenedAt1)
}

func (s *MessengerCommunitiesSuite) TestSyncCommunityLastOpenedAt() {
	// Create new device
	alicesOtherDevice := s.createOtherDevice(s.alice)
	PairDevices(&s.Suite, alicesOtherDevice, s.alice)

	// Create a community
	createCommunityReq := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
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

	// Mock frontend triggering communityUpdateLastOpenedAt
	lastOpenedAt, err := s.alice.CommunityUpdateLastOpenedAt(newCommunity.IDString())
	s.Require().NoError(err)

	// Check lastOpenedAt was updated
	s.Require().True(lastOpenedAt > 0)

	err = tt.RetryWithBackOff(func() error {
		_, err = alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}
		// Do we have a new synced community?
		_, err := alicesOtherDevice.communitiesManager.GetSyncedRawCommunity(newCommunity.ID())
		if err != nil {
			return fmt.Errorf("community with sync not received %w", err)
		}

		return nil
	})
	otherDeviceCommunity, err := alicesOtherDevice.communitiesManager.GetByID(newCommunity.ID())
	s.Require().NoError(err)
	s.Require().True(otherDeviceCommunity.LastOpenedAt() > 0)
}
