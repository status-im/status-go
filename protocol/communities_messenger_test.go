package protocol

import (
	"bytes"
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
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
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

	s.bob = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
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
		WithDatabaseConfig(tmpFile.Name(), ""),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *MessengerCommunitiesSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

func (s *MessengerCommunitiesSuite) TestRetrieveCommunity() {
	alice := s.newMessenger()

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	community := response.Communities()[0]

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&alice.identity.PublicKey), &alice.identity.PublicKey, s.alice.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.bob.SaveChat(chat)
	s.NoError(err)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

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
	s.Require().Len(response.Messages, 1)
	s.Require().Equal(community.IDString(), response.Messages[0].CommunityID)
}

func (s *MessengerCommunitiesSuite) TestJoinCommunity() {

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

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
	s.Require().Len(response.Chats(), 1)

	createdChat := response.Chats()[0]
	s.Require().Equal(community.IDString(), createdChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
	s.Require().NotEmpty(createdChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
	s.Require().True(createdChat.Active)
	s.Require().NotEmpty(createdChat.Timestamp)
	s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))

	// Make sure the changes are reflect in the community
	community = response.Communities()[0]
	chats := community.Chats()
	s.Require().Len(chats, 1)

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.alice.identity.PublicKey), &s.alice.identity.PublicKey, s.bob.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.bob.SaveChat(chat)
	s.NoError(err)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

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
	s.Require().Len(response.Messages, 1)
	s.Require().Equal(community.IDString(), response.Messages[0].CommunityID)

	// We join the org
	response, err = s.alice.JoinCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().Len(response.Chats(), 1)

	// The chat should be created
	createdChat = response.Chats()[0]
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
	response, err = s.bob.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	// Pull message, this time it should be received as advertised automatically
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
	communities, err = s.alice.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 2)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	// The chat should be created
	createdChat = response.Chats()[0]
	s.Require().Equal(community.IDString(), createdChat.CommunityID)
	s.Require().Equal(orgChat.Identity.DisplayName, createdChat.Name)
	s.Require().NotEmpty(createdChat.ID)
	s.Require().Equal(ChatTypeCommunityChat, createdChat.ChatType)
	s.Require().True(createdChat.Active)
	s.Require().NotEmpty(createdChat.Timestamp)
	s.Require().True(strings.HasPrefix(createdChat.ID, community.IDString()))

	// We leave the org
	response, err = s.alice.LeaveCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().False(response.Communities()[0].Joined())
	s.Require().Len(response.RemovedChats(), 2)
}

func (s *MessengerCommunitiesSuite) TestInviteUsersToCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
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
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

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

	// We join the org
	response, err = s.alice.JoinCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().Len(response.Chats(), 1)
	s.Require().Len(response.Filters, 2)

	var orgFilterFound bool
	var chatFilterFound bool
	for _, f := range response.Filters {
		orgFilterFound = orgFilterFound || f.ChatID == response.Communities()[0].IDString()
		chatFilterFound = chatFilterFound || f.ChatID == response.Chats()[0].ID
	}
	// Make sure an community filter has been created
	s.Require().True(orgFilterFound)
	// Make sure the chat filter has been created
	s.Require().True(chatFilterFound)

	chatID := response.Chats()[0].ID
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
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal(chatID, response.Chats()[0].ID)
}

func (s *MessengerCommunitiesSuite) TestImportCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	privateKey, err := s.bob.ExportCommunity(community.ID())
	s.Require().NoError(err)

	response, err = s.alice.ImportCommunity(privateKey)
	s.Require().NoError(err)
	s.Require().Len(response.Filters, 1)

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
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	community = response.Communities()[0]
	s.Require().True(community.Joined())
	s.Require().True(community.IsAdmin())
}

func (s *MessengerCommunitiesSuite) TestRequestAccess() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
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

	// Accept request

	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin1.ID}

	response, err = s.bob.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

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

}

func (s *MessengerCommunitiesSuite) TestRequestAccessAgain() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
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
		return nil
	})

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

func (s *MessengerCommunitiesSuite) TestShareCommunity() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]

	response, err = s.bob.ShareCommunity(
		&requests.ShareCommunity{
			CommunityID: community.ID(),
			Users:       []types.HexBytes{common.PubkeyToHexBytes(&s.alice.identity.PublicKey)},
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Messages, 1)

	// Add bob to contacts so it does not go on activity center
	bobPk := common.PubkeyToHex(&s.alice.identity.PublicKey)
	_, err = s.alice.AddContact(context.Background(), bobPk)
	s.Require().NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Messages) == 0 {
			return errors.New("community link not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Messages, 1)
}

func (s *MessengerCommunitiesSuite) TestBanUser() {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description)
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

}
