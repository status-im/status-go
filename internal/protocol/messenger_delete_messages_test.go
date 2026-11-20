package protocol

import (
	"context"
	"crypto/ecdsa"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestMessengerDeleteMessagesSuite(t *testing.T) {
	suite.Run(t, new(MessengerDeleteMessagesSuite))
}

type MessengerDeleteMessagesSuite struct {
	CommunitiesMessengerTestSuiteBase
	owner *Messenger
	admin *Messenger
	bob   *Messenger
}

func (s *MessengerDeleteMessagesSuite) SetupTest() {
	s.CommunitiesMessengerTestSuiteBase.SetupTest()

	s.owner = s.newMessenger("", []string{})
	s.bob = s.newMessenger(bobPassword, []string{bobAddress})
	s.admin = s.newMessenger(alicePassword, []string{aliceAddress1})
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

	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.admin, alicePassword, []string{aliceAddress1})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.bob, bobPassword, []string{bobAddress})

	grantPermission(&s.Suite, community, s.owner, s.admin, protobuf.CommunityMember_ROLE_ADMIN)

	bobMessage := s.sendMessageAndCheckDelivery(s.bob, "bob message", communityChat.ID)

	expectedMsgsToRemove := 1
	communityID := community.IDString()
	s.checkAllMembersHasMemberMessages(s.bob.IdentityPublicKeyString(), expectedMsgsToRemove, communityID)

	// empty request
	deleteMessagesRequest := &requests.DeleteCommunityMemberMessages{}
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
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

	deleteMessagesRequest.Messages = []*protobuf.DeleteCommunityMemberMessage{nil}
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, requests.ErrDeleteCommunityMemberMessagesInvalidData)

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

	deleteMessagesRequest.MemberPubKey = s.admin.IdentityPublicKeyString()
	_, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().ErrorIs(err, ErrCommunityMessageScopeMismatch)
	_, err = s.owner.persistence.MessageByID(bobMessage.ID)
	s.Require().NoError(err)
}

func (s *MessengerDeleteMessagesSuite) TestDeleteMessage() {
	community, communityChat := createCommunity(&s.Suite, s.owner)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.admin, alicePassword, []string{aliceAddress1})

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.bob, bobPassword, []string{bobAddress})

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
	deleteMessagesRequest.MemberPubKey = s.admin.IdentityPublicKeyString()
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
	memberPubKey := s.owner.IdentityPublicKeyString()
	deleteMessagesRequest.MemberPubKey = memberPubKey[:2] + strings.ToUpper(memberPubKey[2:])
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

func (s *MessengerDeleteMessagesSuite) TestDeleteMessageScope() {
	community, communityChat := createCommunity(&s.Suite, s.owner)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.admin, alicePassword, []string{aliceAddress1})
	grantPermission(&s.Suite, community, s.owner, s.admin, protobuf.CommunityMember_ROLE_ADMIN)
	otherCommunity, otherChat := createCommunity(&s.Suite, s.bob)
	directChat := CreateOneToOneChat(s.admin.IdentityPublicKeyString(), s.admin.IdentityPublicKey(), s.bob.getTimesource())
	s.Require().NoError(s.bob.persistence.SaveChats([]*Chat{communityChat, otherChat, directChat}))
	s.Require().NoError(s.admin.persistence.SaveChats([]*Chat{communityChat}))

	for _, entry := range []struct {
		id, chatID, communityID, author string
	}{
		{"valid", communityChat.ID, community.IDString(), s.admin.IdentityPublicKeyString()},
		{"direct", directChat.ID, "", s.admin.IdentityPublicKeyString()},
		{"other", otherChat.ID, otherCommunity.IDString(), s.admin.IdentityPublicKeyString()},
		{"wrong-author", communityChat.ID, community.IDString(), s.bob.IdentityPublicKeyString()},
	} {
		s.Require().NoError(s.bob.persistence.SaveMessages([]*common.Message{{
			ID: entry.id, LocalChatID: entry.chatID, CommunityID: entry.communityID, From: entry.author,
			ChatMessage: &protobuf.ChatMessage{ChatId: entry.chatID, Text: entry.id},
		}}))
	}
	s.Require().False(community.ControlNode().Equal(s.owner.IdentityPublicKey()))
	for _, test := range []struct {
		name      string
		sender    *ecdsa.PublicKey
		deleteAll bool
		rejected  bool
	}{
		{name: "admin-owner-message", sender: s.admin.IdentityPublicKey(), rejected: true},
		{name: "admin-all-owner-messages", sender: s.admin.IdentityPublicKey(), deleteAll: true, rejected: true},
		{name: "owner-message", sender: s.owner.IdentityPublicKey()},
		{name: "owner-all-messages", sender: s.owner.IdentityPublicKey(), deleteAll: true},
		{name: "control-node-owner-message", sender: community.ControlNode()},
		{name: "control-node-all-owner-messages", sender: community.ControlNode(), deleteAll: true},
	} {
		s.Run(test.name, func() {
			for _, id := range []string{"owner-message", "owner-message2"} {
				s.Require().NoError(s.admin.persistence.SaveMessages([]*common.Message{{
					ID: id, LocalChatID: communityChat.ID, CommunityID: community.IDString(),
					From:        s.owner.IdentityPublicKeyString(),
					ChatMessage: &protobuf.ChatMessage{ChatId: communityChat.ID, Text: id},
				}}))
			}
			state := s.admin.buildMessageState()
			state.CurrentMessageState = &CurrentMessageState{PublicKey: test.sender}
			request := &protobuf.DeleteCommunityMemberMessages{
				CommunityId: community.ID(), MemberId: s.owner.IdentityPublicKeyString(),
			}
			if !test.deleteAll {
				request.Messages = []*protobuf.DeleteCommunityMemberMessage{{Id: "owner-message", ChatId: communityChat.ID}}
			}
			err := s.admin.HandleDeleteCommunityMemberMessages(context.Background(), state, request, nil)
			expected := map[string]string{}
			if test.rejected {
				s.Require().ErrorIs(err, communities.ErrNotOwner)
				s.Require().Empty(state.Response.DeletedMessages())
			} else {
				s.Require().NoError(err)
				expected["owner-message"] = communityChat.ID
				if test.deleteAll {
					expected["owner-message2"] = communityChat.ID
				}
				s.Require().Equal(expected, state.Response.DeletedMessages())
			}
			s.Require().Empty(state.Response.Messages())
			for _, id := range []string{"owner-message", "owner-message2"} {
				_, err = s.admin.persistence.MessageByID(id)
				if _, deleted := expected[id]; deleted {
					s.Require().ErrorIs(err, common.ErrRecordNotFound)
				} else {
					s.Require().NoError(err)
				}
			}
		})
	}

	for _, test := range []struct {
		name, id, chatID string
	}{
		{"direct-message", "direct", directChat.ID},
		{"other-community", "other", otherChat.ID},
		{"forged-community-chat", "direct", communityChat.ID},
		{"forged-chat", "valid", otherChat.ID},
		{"wrong-author", "wrong-author", communityChat.ID},
	} {
		s.Run(test.name, func() {
			state := s.bob.buildMessageState()
			state.CurrentMessageState = &CurrentMessageState{PublicKey: community.ControlNode()}
			request := &protobuf.DeleteCommunityMemberMessages{
				CommunityId: community.ID(), MemberId: s.admin.IdentityPublicKeyString(),
				Messages: []*protobuf.DeleteCommunityMemberMessage{
					{Id: "valid", ChatId: communityChat.ID},
					{Id: test.id, ChatId: test.chatID},
				},
			}
			err := s.bob.HandleDeleteCommunityMemberMessages(context.Background(), state, request, nil)
			s.Require().ErrorIs(err, ErrCommunityMessageScopeMismatch)
			s.Require().Empty(state.Response.DeletedMessages())
			s.Require().Empty(state.Response.Messages())
			for _, id := range []string{"valid", "direct", "other", "wrong-author"} {
				_, err = s.bob.persistence.MessageByID(id)
				s.Require().NoError(err)
			}
		})
	}

	for _, test := range []struct {
		name     string
		member   string
		messages []*protobuf.DeleteCommunityMemberMessage
	}{
		{name: "missing-member"},
		{name: "invalid-member", member: "not-a-public-key"},
		{name: "nil-message", member: s.admin.IdentityPublicKeyString(), messages: []*protobuf.DeleteCommunityMemberMessage{nil}},
		{name: "missing-message-id", member: s.admin.IdentityPublicKeyString(), messages: []*protobuf.DeleteCommunityMemberMessage{{ChatId: communityChat.ID}}},
		{name: "missing-chat-id", member: s.admin.IdentityPublicKeyString(), messages: []*protobuf.DeleteCommunityMemberMessage{{Id: "valid"}}},
	} {
		s.Run(test.name, func() {
			state := s.bob.buildMessageState()
			state.CurrentMessageState = &CurrentMessageState{PublicKey: community.ControlNode()}
			err := s.bob.HandleDeleteCommunityMemberMessages(context.Background(), state, &protobuf.DeleteCommunityMemberMessages{
				CommunityId: community.ID(), MemberId: test.member, Messages: test.messages,
			}, nil)
			s.Require().Error(err)
			s.Require().Empty(state.Response.DeletedMessages())
			_, err = s.bob.persistence.MessageByID("valid")
			s.Require().NoError(err)
		})
	}

	state := s.bob.buildMessageState()
	state.CurrentMessageState = &CurrentMessageState{PublicKey: community.ControlNode()}
	request := &protobuf.DeleteCommunityMemberMessages{
		CommunityId: community.ID(), MemberId: s.admin.IdentityPublicKeyString(),
		Messages: []*protobuf.DeleteCommunityMemberMessage{
			{Id: "valid", ChatId: communityChat.ID},
			{Id: "valid", ChatId: communityChat.ID},
			{Id: "missing", ChatId: communityChat.ID},
		},
	}
	s.Require().NoError(s.bob.HandleDeleteCommunityMemberMessages(context.Background(), state, request, nil))
	s.Require().Equal(map[string]string{"valid": communityChat.ID}, state.Response.DeletedMessages())
	state.Response = &MessengerResponse{}
	s.Require().NoError(s.bob.HandleDeleteCommunityMemberMessages(context.Background(), state, request, nil))
	s.Require().Empty(state.Response.DeletedMessages())
}

func (s *MessengerDeleteMessagesSuite) TestDeleteAllMemberMessage() {
	community, communityChat := createCommunity(&s.Suite, s.owner)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.admin)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.admin, aliceAccountAddress, []string{aliceAddress1})

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	joinCommunity(&s.Suite, community.ID(), s.owner, s.bob, bobPassword, []string{bobAddress})

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

	deleteMessagesRequest.MemberPubKey = s.owner.IdentityPublicKeyString()
	response, err = s.owner.DeleteCommunityMemberMessages(deleteMessagesRequest)
	s.Require().NoError(err)
	checkOwnerMessageDeleted := func(response *MessengerResponse) bool {
		return len(response.DeletedMessages()) == 1
	}
	s.Require().True(checkOwnerMessageDeleted(response))
	_, err = WaitOnMessengerResponse(s.bob, checkOwnerMessageDeleted, "owner messages were not deleted for bob")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.admin, checkOwnerMessageDeleted, "owner messages were not deleted for admin")
	s.Require().NoError(err)
	s.checkAllMembersHasMemberMessages(s.owner.IdentityPublicKeyString(), 0, communityID)
	s.checkAllMembersHasMemberMessages(s.admin.IdentityPublicKeyString(), 1, communityID)
}
