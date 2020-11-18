package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerCommunitiesSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesSuite))
}

type MessengerCommunitiesSuite struct {
	suite.Suite
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
	s.Require().NoError(shh.Start(nil))

	s.bob = s.newMessenger(s.shh)
	s.alice = s.newMessenger(s.shh)
	s.Require().NoError(s.bob.Start())
	s.Require().NoError(s.alice.Start())
}

func (s *MessengerCommunitiesSuite) TearDownTest() {
	s.Require().NoError(s.bob.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSuite) newMessengerWithOptions(shh types.Waku, privateKey *ecdsa.PrivateKey, options []Option) *Messenger {
	m, err := NewMessenger(
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	return m
}

func (s *MessengerCommunitiesSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)

	options := []Option{
		WithCustomLogger(s.logger),
		WithMessagesPersistenceEnabled(),
		WithDatabaseConfig(tmpFile.Name(), ""),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *MessengerCommunitiesSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerCommunitiesSuite) TestRetrieveCommunity() {
	alice := s.newMessenger(s.shh)

	description := &protobuf.CommunityDescription{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Description: "status community description",
		},
	}

	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	community := response.Communities[0]

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&alice.identity.PublicKey), &alice.identity.PublicKey, s.alice.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.bob.SaveChat(&chat)
	s.NoError(err)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities, 1)
	s.Require().Len(response.Messages, 1)
	s.Require().Equal(community.IDString(), response.Messages[0].CommunityID)
}

func (s *MessengerCommunitiesSuite) TestJoinCommunity() {
	// start alice and enable sending push notifications
	s.Require().NoError(s.alice.Start())

	description := &protobuf.CommunityDescription{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Description: "status community description",
		},
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)

	community := response.Communities[0]

	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Description: "status-core community chat",
		},
	}
	response, err = s.bob.CreateCommunityChat(community.IDString(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	s.Require().Len(response.Chats, 1)

	createdChat := response.Chats[0]
	s.Require().Equal(community.IDString(), createdChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
	s.Require().NotEmpty(createdChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
	s.Require().True(createdChat.Active)
	s.Require().NotEmpty(createdChat.Timestamp)
	s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))

	// Make sure the changes are reflect in the community
	community = response.Communities[0]
	chats := community.Chats()
	s.Require().Len(chats, 1)

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.bob.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.bob.SaveChat(&chat)
	s.NoError(err)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities, 1)
	s.Require().Len(response.Messages, 1)
	s.Require().Equal(community.IDString(), response.Messages[0].CommunityID)

	// We join the org
	response, err = s.alice.JoinCommunity(community.IDString())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	s.Require().True(response.Communities[0].Joined())
	s.Require().Len(response.Chats, 1)

	// The chat should be created
	createdChat = response.Chats[0]
	s.Require().Equal(community.IDString(), createdChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
	s.Require().NotEmpty(createdChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
	s.Require().True(createdChat.Active)
	s.Require().NotEmpty(createdChat.Timestamp)
	s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))

	// Create another org chat
	orgChat = &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core-ui",
			Description: "status-core-ui community chat",
		},
	}
	response, err = s.bob.CreateCommunityChat(community.IDString(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	s.Require().Len(response.Chats, 1)

	// Pull message, this time it should be received as advertised automatically
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err = s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities, 1)
	s.Require().Len(response.Chats, 1)

	// The chat should be created
	createdChat = response.Chats[0]
	s.Require().Equal(community.IDString(), createdChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
	s.Require().NotEmpty(createdChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
	s.Require().True(createdChat.Active)
	s.Require().NotEmpty(createdChat.Timestamp)
	s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))

	// We leave the org
	response, err = s.alice.LeaveCommunity(community.IDString())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	s.Require().False(response.Communities[0].Joined())
	s.Require().Len(response.RemovedChats, 2)
}

func (s *MessengerCommunitiesSuite) TestInviteUserToCommunity() {
	description := &protobuf.CommunityDescription{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Description: "status community description",
		},
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)

	community := response.Communities[0]

	response, err = s.bob.InviteUserToCommunity(community.IDString(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)

	community = response.Communities[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities, 1)

	community = response.Communities[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))
}

func (s *MessengerCommunitiesSuite) TestPostToCommunityChat() {
	description := &protobuf.CommunityDescription{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_INVITATION_ONLY,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Description: "status community description",
		},
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)

	community := response.Communities[0]

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

	response, err = s.bob.CreateCommunityChat(community.IDString(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	s.Require().Len(response.Chats, 1)

	response, err = s.bob.InviteUserToCommunity(community.IDString(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)

	community = response.Communities[0]
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	communities, err := s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities, 1)

	// We join the org
	response, err = s.alice.JoinCommunity(community.IDString())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)
	s.Require().True(response.Communities[0].Joined())
	s.Require().Len(response.Chats, 1)
	s.Require().Len(response.Filters, 2)

	var orgFilterFound bool
	var chatFilterFound bool
	for _, f := range response.Filters {
		orgFilterFound = orgFilterFound || f.ChatID == response.Communities[0].IDString()
		chatFilterFound = chatFilterFound || f.ChatID == response.Chats[0].ID
	}
	// Make sure an community filter has been created
	s.Require().True(orgFilterFound)
	// Make sure the chat filter has been created
	s.Require().True(chatFilterFound)

	chatID := response.Chats[0].ID
	inputMessage := &common.Message{}
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	_, err = s.alice.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Messages) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Messages, 1)
	s.Require().Len(response.Chats, 1)
	s.Require().Equal(chatID, response.Chats[0].ID)
}

func (s *MessengerCommunitiesSuite) TestImportCommunity() {
	description := &protobuf.CommunityDescription{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Description: "status community description",
		},
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities, 1)

	s.bob.logger.Info("communitise", zap.Any("COMM", response.Communities))

	community := response.Communities[0]

	privateKey, err := s.bob.ExportCommunity(community.IDString())
	s.Require().NoError(err)

	response, err = s.alice.ImportCommunity(privateKey)
	s.Require().NoError(err)
	s.Require().Len(response.Filters, 1)

	// Invite user on bob side
	newUser, err := crypto.GenerateKey()
	s.Require().NoError(err)

	_, err = s.bob.InviteUserToCommunity(community.IDString(), common.PubkeyToHex(&newUser.PublicKey))
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Communities, 1)
	community = response.Communities[0]
	s.Require().True(community.Joined())
	s.Require().True(community.IsAdmin())
	s.Require().True(community.HasMember(&newUser.PublicKey))
}
