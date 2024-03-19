package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerDeleteMessagesSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessagesSuite))
}

type MessengerDeleteMessagesSuite struct {
	suite.Suite
	owner  *Messenger
	admin  *Messenger
	bob    *Messenger
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerDeleteMessagesSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.owner = s.newMessenger()
	s.bob = s.newMessenger()
	s.admin = s.newMessenger()
}

func (s *MessengerDeleteMessagesSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.admin)
	_ = s.logger.Sync()
}

func (s *MessengerDeleteMessagesSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerDeleteMessagesSuite) sendMessageAndCheckDelivery(sender *Messenger, text string, chatID string) *common.Message {
	ctx := context.Background()
	messageToSend := common.NewMessage()
	messageToSend.ChatId = chatID
	messageToSend.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	messageToSend.Text = text
	response, err := sender.SendChatMessage(ctx, messageToSend)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)

	var message *common.Message
	if sender.identity != s.admin.identity {
		response, err := WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
			return len(response.Messages()) == 1
		}, "admin did not receive message")
		s.Require().NoError(err)
		message = response.Messages()[0]
		s.Require().Equal(messageToSend.Text, message.Text)
	}

	if sender.identity != s.owner.identity {
		response, err = WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
			return len(response.Messages()) == 1
		}, "owner did not receive message")
		s.Require().NoError(err)
		message = response.Messages()[0]
		s.Require().Equal(messageToSend.Text, message.Text)
	}

	if sender.identity != s.bob.identity {
		response, err = WaitOnMessengerResponse(s.bob, func(response *MessengerResponse) bool {
			return len(response.Messages()) == 1
		}, "bob did not receive message")
		s.Require().NoError(err)
		message = response.Messages()[0]
		s.Require().Equal(messageToSend.Text, message.Text)
	}

	return message
}

func (s *MessengerDeleteMessagesSuite) checkStoredMemberMessagesAmount(messenger *Messenger, memberPubKey string, expectedAmount int, communityID string) {
	storedMessages, err := messenger.GetCommunityMemberAllMessages(
		&requests.CommunityMemberMessages{
			CommunityID:     communityID,
			MemberPublicKey: memberPubKey})
	s.Require().NoError(err)
	s.Require().Len(storedMessages, expectedAmount)
}

func (s *MessengerDeleteMessagesSuite) checkAllMembersHasMemberMessages(memberPubKey string, expectedAmount int, communityID string) {
	s.checkStoredMemberMessagesAmount(s.bob, memberPubKey, expectedAmount, communityID)
	s.checkStoredMemberMessagesAmount(s.owner, memberPubKey, expectedAmount, communityID)
	s.checkStoredMemberMessagesAmount(s.admin, memberPubKey, expectedAmount, communityID)
}

func (s *MessengerDeleteMessagesSuite) TestDeleteMessageErrors() {
	community, communityChat := createCommunity(&s.Suite, s.owner)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}

	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community, s.owner, s.admin, request, "")

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	joinCommunity(&s.Suite, community, s.owner, s.bob, request, "")

	grantPermission(&s.Suite, community, s.owner, s.admin, protobuf.CommunityMember_ROLE_ADMIN)

	bobMessage := s.sendMessageAndCheckDelivery(s.bob, "bob message", communityChat.ID)

	expectedMsgsToRemove := 1
	communityID := community.IDString()
	s.checkAllMembersHasMemberMessages(s.bob.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// empty request
	deleteMessagesRequest := &requests.DeleteCommunityMemberMessages{}
	_, err := s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidCommunityID)

	// only community ID provided
	deleteMessagesRequest.CommunityID = community.ID()
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidMemberID)

	// only community ID and member ID provided, but delete flag false and no messages IDs
	deleteMessagesRequest.MemberPubKey = s.bob.IdentityPublicKeyString()
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidDeleteMessagesByID)

	// message provided without id
	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
		ChatId: bobMessage.ChatId,
	}}

	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidMsgID)

	// message provided without chatId
	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
		Id: bobMessage.ID,
	}}
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidMsgChatID)

	// messages id provided but with flag deleteAll
	deleteMessagesRequest.CommunityID = community.ID()
	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
		Id:     bobMessage.ID,
		ChatId: bobMessage.ChatId,
	}}
	deleteMessagesRequest.DeleteAll = true
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidDeleteAll)

	// bob tries to delete his own message
	deleteMessagesRequest.DeleteAll = false
	_, err = s.bob.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, communities.ErrNotEnoughPermissions)

	// admin tries to delete owner message
	deleteMessagesRequest.MemberPubKey = s.owner.IdentityPublicKeyString()
	_, err = s.admin.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, communities.ErrNotOwner)
}

func (s *MessengerDeleteMessagesSuite) TestDeleteMessage() {
	community, communityChat := createCommunity(&s.Suite, s.owner)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}

	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community, s.owner, s.admin, request, "")

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	joinCommunity(&s.Suite, community, s.owner, s.bob, request, "")

	grantPermission(&s.Suite, community, s.owner, s.admin, protobuf.CommunityMember_ROLE_ADMIN)

	bobMessage := s.sendMessageAndCheckDelivery(s.bob, "bob message", communityChat.ID)
	bobMessage2 := s.sendMessageAndCheckDelivery(s.bob, "bob message2", communityChat.ID)
	ownerMessage := s.sendMessageAndCheckDelivery(s.owner, "owner message", communityChat.ID)
	adminMessage := s.sendMessageAndCheckDelivery(s.admin, "admin message", communityChat.ID)

	identityString := s.bob.IdentityPublicKeyString()
	expectedMsgsToRemove := 2
	communityID := community.IDString()
	s.checkAllMembersHasMemberMessages(identityString, expectedMsgsToRemove, communityID)
	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), 1, communityID)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), 1, communityID)

	// delete bob message
	deleteMessagesRequest := &requests.DeleteCommunityMemberMessages{
		CommunityID:  community.ID(),
		MemberPubKey: identityString,
		Messages: []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
			Id:     bobMessage.ID,
			ChatId: bobMessage.ChatId,
		}},
	}
	response, err := s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().NoError(err)

	checkMessageDeleted := func(response *MessengerResponse) bool {
		if len(response.DeletedMessages()) == 0 {
			return false
		}

		if _, exists := response.DeletedMessages()[deleteMessagesRequest.Messages[0].Id]; !exists {
			return false
		}
		return true
	}

	s.Require().True(checkMessageDeleted(response))

	_, err = WaitOnMessengerResponse(s.bob, checkMessageDeleted, "message was not deleted for bob")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.admin, checkMessageDeleted, "message was not deleted for admin")
	s.Require().NoError(err)

	expectedMsgsToRemove = 1
	s.checkAllMembersHasMemberMessages(s.bob.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// check that other users messages were not removed
	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// check that admin can delete member message
	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
		Id:     bobMessage2.ID,
		ChatId: bobMessage2.ChatId,
	}}
	response, err = s.admin.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().NoError(err)
	s.Require().True(checkMessageDeleted(response))

	_, err = WaitOnMessengerResponse(s.bob, checkMessageDeleted, "message2 was not deleted for bob")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.owner, checkMessageDeleted, "message2 was not deleted for owner")
	s.Require().NoError(err)

	expectedMsgsToRemove = 0
	s.checkAllMembersHasMemberMessages(s.bob.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// check that other users messages were not removed
	expectedMsgsToRemove = 1
	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// check that owner can delete member message
	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
		Id:     adminMessage.ID,
		ChatId: adminMessage.ChatId,
	}}
	response, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().NoError(err)
	s.Require().True(checkMessageDeleted(response))

	_, err = WaitOnMessengerResponse(s.bob, checkMessageDeleted, "adminMessage was not deleted for bob")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.admin, checkMessageDeleted, "adminMessage was not deleted for admin")
	s.Require().NoError(err)

	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), 0, communityID)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), 1, communityID)

	// check that owner can delete his own message
	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{&protobuf.DeleteCommunityMemberMessage{
		Id:     ownerMessage.ID,
		ChatId: ownerMessage.ChatId,
	}}
	response, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().NoError(err)
	s.Require().True(checkMessageDeleted(response))

	_, err = WaitOnMessengerResponse(s.bob, checkMessageDeleted, "ownerMessage was not deleted for bob")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.admin, checkMessageDeleted, "ownerMessage was not deleted for admin")
	s.Require().NoError(err)

	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), 0, communityID)
}

func (s *MessengerDeleteMessagesSuite) TestDeleteAllMemberMessage() {
	community, communityChat := createCommunity(&s.Suite, s.owner)

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}

	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community, s.owner, s.admin, request, "")

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	joinCommunity(&s.Suite, community, s.owner, s.bob, request, "")

	grantPermission(&s.Suite, community, s.owner, s.admin, protobuf.CommunityMember_ROLE_ADMIN)

	_ = s.sendMessageAndCheckDelivery(s.bob, "bob message", communityChat.ID)
	_ = s.sendMessageAndCheckDelivery(s.bob, "bob message2", communityChat.ID)
	_ = s.sendMessageAndCheckDelivery(s.owner, "owner message", communityChat.ID)
	_ = s.sendMessageAndCheckDelivery(s.admin, "admin message", communityChat.ID)

	identityString := s.bob.IdentityPublicKeyString()
	expectedMsgsToRemove := 2
	communityID := community.IDString()
	s.checkAllMembersHasMemberMessages(identityString, expectedMsgsToRemove, communityID)
	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), 1, communityID)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), 1, communityID)

	// delete all bob message
	deleteMessagesRequest := &requests.DeleteCommunityMemberMessages{
		CommunityID:  community.ID(),
		MemberPubKey: identityString,
		DeleteAll:    true,
	}
	response, err := s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().NoError(err)

	checkMessageDeleted := func(response *MessengerResponse) bool {
		return len(response.DeletedMessages()) == 2
	}

	s.Require().True(checkMessageDeleted(response))

	_, err = WaitOnMessengerResponse(s.bob, checkMessageDeleted, "messages were not deleted for bob")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.admin, checkMessageDeleted, "messages were not deleted for admin")
	s.Require().NoError(err)

	expectedMsgsToRemove = 0
	s.checkAllMembersHasMemberMessages(s.bob.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// check that other users messages were not removed
	expectedMsgsToRemove = 1
	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)
}
