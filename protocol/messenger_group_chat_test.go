package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"

	userimage "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestGroupChatSuite(t *testing.T) {
	suite.Run(t, new(MessengerGroupChatSuite))
}

type MessengerGroupChatSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerGroupChatSuite) createGroupChat(creator *Messenger, name string, members []string) *Chat {
	response, err := creator.CreateGroupChatWithMembers(context.Background(), name, members)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)

	chat := response.Chats()[0]
	err = creator.SaveChat(chat)
	s.Require().NoError(err)

	return chat
}

func (s *MessengerGroupChatSuite) createEmptyGroupChat(creator *Messenger, name string) *Chat {
	return s.createGroupChat(creator, name, []string{})
}

func (s *MessengerGroupChatSuite) verifyGroupChatCreated(member *Messenger, expectedChatActive bool) {
	response, err := WaitOnMessengerResponse(
		member,
		func(r *MessengerResponse) bool {
			return len(r.Chats()) == 1 && r.Chats()[0].Active == expectedChatActive
		},
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().True(response.Chats()[0].Active == expectedChatActive)
}

func makeMutualContact(origin *Messenger, contactPubkey *ecdsa.PublicKey) error {
	contact, err := BuildContactFromPublicKey(contactPubkey)
	if err != nil {
		return err
	}
	contact.ContactRequestLocalState = ContactRequestStateSent
	contact.ContactRequestRemoteState = ContactRequestStateReceived
	origin.allContacts.Store(contact.ID, contact)

	return nil
}

func (s *MessengerGroupChatSuite) makeContact(origin *Messenger, toAdd *Messenger) {
	s.Require().NoError(makeMutualContact(origin, &toAdd.identity.PublicKey))
}

func (s *MessengerGroupChatSuite) makeMutualContacts(lhs *Messenger, rhs *Messenger) {
	s.makeContact(lhs, rhs)
	s.makeContact(rhs, lhs)
}

func (s *MessengerGroupChatSuite) TestGroupChatCreation() {
	testCases := []struct {
		name                          string
		creatorAddedMemberAsContact   bool
		memberAddedCreatorAsContact   bool
		expectedCreationSuccess       bool
		expectedAddedMemberChatActive bool
	}{
		{
			name:                          "not added - not added",
			creatorAddedMemberAsContact:   false,
			memberAddedCreatorAsContact:   false,
			expectedCreationSuccess:       false,
			expectedAddedMemberChatActive: false,
		},
		{
			name:                          "added - not added",
			creatorAddedMemberAsContact:   true,
			memberAddedCreatorAsContact:   false,
			expectedCreationSuccess:       true,
			expectedAddedMemberChatActive: false,
		},
		{
			name:                          "not added - added",
			creatorAddedMemberAsContact:   false,
			memberAddedCreatorAsContact:   true,
			expectedCreationSuccess:       false,
			expectedAddedMemberChatActive: false,
		},
		{
			name:                          "added - added",
			creatorAddedMemberAsContact:   true,
			memberAddedCreatorAsContact:   true,
			expectedCreationSuccess:       true,
			expectedAddedMemberChatActive: true,
		},
	}

	for i, testCase := range testCases {
		creator := s.newMessenger()
		member := s.newMessenger()
		members := []string{common.PubkeyToHex(&member.identity.PublicKey)}

		if testCase.creatorAddedMemberAsContact {
			s.makeContact(creator, member)
		}
		if testCase.memberAddedCreatorAsContact {
			s.makeContact(member, creator)
		}

		_, err := creator.CreateGroupChatWithMembers(context.Background(), fmt.Sprintf("test_group_chat_%d", i), members)
		if testCase.creatorAddedMemberAsContact {
			s.Require().NoError(err)
			s.verifyGroupChatCreated(member, testCase.expectedAddedMemberChatActive)
		} else {
			s.Require().EqualError(err, "group-chat: can't add members who are not mutual contacts")
		}

		defer s.NoError(creator.Shutdown())
		defer s.NoError(member.Shutdown())
	}
}

func (s *MessengerGroupChatSuite) TestGroupChatMembersAddition() {
	testCases := []struct {
		name                          string
		inviterAddedMemberAsContact   bool
		memberAddedInviterAsContact   bool
		expectedAdditionSuccess       bool
		expectedAddedMemberChatActive bool
	}{
		{
			name:                          "not added - not added",
			inviterAddedMemberAsContact:   false,
			memberAddedInviterAsContact:   false,
			expectedAdditionSuccess:       false,
			expectedAddedMemberChatActive: false,
		},
		{
			name:                          "added - not added",
			inviterAddedMemberAsContact:   true,
			memberAddedInviterAsContact:   false,
			expectedAdditionSuccess:       true,
			expectedAddedMemberChatActive: false,
		},
		{
			name:                          "not added - added",
			inviterAddedMemberAsContact:   false,
			memberAddedInviterAsContact:   true,
			expectedAdditionSuccess:       false,
			expectedAddedMemberChatActive: false,
		},
		{
			name:                          "added - added",
			inviterAddedMemberAsContact:   true,
			memberAddedInviterAsContact:   true,
			expectedAdditionSuccess:       true,
			expectedAddedMemberChatActive: true,
		},
	}

	for i, testCase := range testCases {
		admin := s.newMessenger()
		inviter := s.newMessenger()
		member := s.newMessenger()
		members := []string{common.PubkeyToHex(&member.identity.PublicKey)}

		if testCase.inviterAddedMemberAsContact {
			s.makeContact(inviter, member)
		}
		if testCase.memberAddedInviterAsContact {
			s.makeContact(member, inviter)
		}

		for j, inviterIsAlsoGroupCreator := range []bool{true, false} {
			var groupChat *Chat
			if inviterIsAlsoGroupCreator {
				groupChat = s.createEmptyGroupChat(inviter, fmt.Sprintf("test_group_chat_%d_%d", i, j))
			} else {
				s.makeContact(admin, inviter)
				groupChat = s.createGroupChat(admin, fmt.Sprintf("test_group_chat_%d_%d", i, j), []string{common.PubkeyToHex(&inviter.identity.PublicKey)})
				err := inviter.SaveChat(groupChat)
				s.Require().NoError(err)
			}

			_, err := inviter.AddMembersToGroupChat(context.Background(), groupChat.ID, members)
			if testCase.inviterAddedMemberAsContact {
				s.Require().NoError(err)
				s.verifyGroupChatCreated(member, testCase.expectedAddedMemberChatActive)
			} else {
				s.Require().EqualError(err, "group-chat: can't add members who are not mutual contacts")
			}
		}

		defer s.NoError(admin.Shutdown())
		defer s.NoError(inviter.Shutdown())
		defer s.NoError(member.Shutdown())
	}
}

func (s *MessengerGroupChatSuite) TestGroupChatMembersRemoval() {
	admin := s.newMessenger()
	memberA := s.newMessenger()
	memberB := s.newMessenger()
	memberC := s.newMessenger()
	members := []string{common.PubkeyToHex(&memberA.identity.PublicKey), common.PubkeyToHex(&memberB.identity.PublicKey),
		common.PubkeyToHex(&memberC.identity.PublicKey)}

	s.makeMutualContacts(admin, memberA)
	s.makeMutualContacts(admin, memberB)
	s.makeMutualContacts(admin, memberC)

	groupChat := s.createGroupChat(admin, "test_group_chat", members)
	s.verifyGroupChatCreated(memberA, true)
	s.verifyGroupChatCreated(memberB, true)
	s.verifyGroupChatCreated(memberC, true)

	_, err := memberA.RemoveMembersFromGroupChat(context.Background(), groupChat.ID, []string{common.PubkeyToHex(&memberB.identity.PublicKey),
		common.PubkeyToHex(&memberC.identity.PublicKey)})
	s.Require().Error(err)

	// only admin can remove members from the group
	_, err = admin.RemoveMembersFromGroupChat(context.Background(), groupChat.ID, []string{common.PubkeyToHex(&memberB.identity.PublicKey),
		common.PubkeyToHex(&memberC.identity.PublicKey)})
	s.Require().NoError(err)

	// ensure removal is propagated to other members
	response, err := WaitOnMessengerResponse(
		memberA,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().True(response.Chats()[0].Active)
	s.Require().Len(response.Chats()[0].Members, 2)

	defer s.NoError(admin.Shutdown())
	defer s.NoError(memberA.Shutdown())
	defer s.NoError(memberB.Shutdown())
	defer s.NoError(memberC.Shutdown())
}

func (s *MessengerGroupChatSuite) TestGroupChatEdit() {
	admin := s.newMessenger()
	member := s.newMessenger()
	s.makeMutualContacts(admin, member)

	groupChat := s.createGroupChat(admin, "test_group_chat", []string{common.PubkeyToHex(&member.identity.PublicKey)})
	s.verifyGroupChatCreated(member, true)

	response, err := admin.EditGroupChat(context.Background(), groupChat.ID, "test_admin_group", "#FF00FF", userimage.CroppedImage{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal("test_admin_group", response.Chats()[0].Name)
	s.Require().Equal("#FF00FF", response.Chats()[0].Color)
	// TODO: handle image

	// ensure group edit is propagated to other members
	response, err = WaitOnMessengerResponse(
		member,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal("test_admin_group", response.Chats()[0].Name)
	s.Require().Equal("#FF00FF", response.Chats()[0].Color)

	response, err = member.EditGroupChat(context.Background(), groupChat.ID, "test_member_group", "#F0F0F0", userimage.CroppedImage{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal("test_member_group", response.Chats()[0].Name)
	s.Require().Equal("#F0F0F0", response.Chats()[0].Color)

	// ensure group edit is propagated to other members
	response, err = WaitOnMessengerResponse(
		admin,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().Equal("test_member_group", response.Chats()[0].Name)
	s.Require().Equal("#F0F0F0", response.Chats()[0].Color)

	inputMessage := buildTestMessage(*groupChat)

	_, err = admin.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	response, err = WaitOnMessengerResponse(
		member,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(inputMessage.Text, response.Messages()[0].Text)

	defer s.NoError(admin.Shutdown())
	defer s.NoError(member.Shutdown())
}

func (s *MessengerGroupChatSuite) TestGroupChatDeleteMemberMessage() {
	admin := s.newMessenger()
	member := s.newMessenger()
	s.makeMutualContacts(admin, member)

	groupChat := s.createGroupChat(admin, "test_group_chat", []string{common.PubkeyToHex(&member.identity.PublicKey)})
	s.verifyGroupChatCreated(member, true)

	ctx := context.Background()
	inputMessage := buildTestMessage(*groupChat)
	_, err := member.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)

	response, err := WaitOnMessengerResponse(
		admin,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"messages not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(inputMessage.Text, response.Messages()[0].Text)

	message := response.Messages()[0]
	deleteMessageResponse, err := admin.DeleteMessageAndSend(ctx, message.ID)
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(member, func(response *MessengerResponse) bool {
		return len(response.RemovedMessages()) > 0
	}, "removed messages not received")
	s.Require().Equal(deleteMessageResponse.RemovedMessages()[0].DeletedBy, contactIDFromPublicKey(admin.IdentityPublicKey()))
	s.Require().NoError(err)
	message, err = member.MessageByID(message.ID)
	s.Require().NoError(err)
	s.Require().True(message.Deleted)

	defer s.NoError(admin.Shutdown())
	defer s.NoError(member.Shutdown())
}

func (s *MessengerGroupChatSuite) TestGroupChatHandleDeleteMemberMessage() {
	admin := s.newMessenger()
	member := s.newMessenger()
	s.makeMutualContacts(admin, member)

	groupChat := s.createGroupChat(admin, "test_group_chat", []string{common.PubkeyToHex(&member.identity.PublicKey)})
	s.verifyGroupChatCreated(member, true)

	ctx := context.Background()
	inputMessage := buildTestMessage(*groupChat)
	_, err := member.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)

	response, err := WaitOnMessengerResponse(
		admin,
		func(r *MessengerResponse) bool { return len(r.Messages()) > 0 },
		"messages not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(inputMessage.Text, response.Messages()[0].Text)

	deleteMessage := &DeleteMessage{
		DeleteMessage: &protobuf.DeleteMessage{
			Clock:       2,
			MessageType: protobuf.MessageType_PRIVATE_GROUP,
			MessageId:   inputMessage.ID,
			ChatId:      groupChat.ID,
		},
		From: common.PubkeyToHex(&admin.identity.PublicKey),
	}

	state := &ReceivedMessageState{
		Response: &MessengerResponse{},
	}

	err = member.handleDeleteMessage(state, deleteMessage)
	s.Require().NoError(err)

	removedMessages := state.Response.RemovedMessages()
	s.Require().Len(removedMessages, 1)
	s.Require().Equal(removedMessages[0].MessageID, inputMessage.ID)

	defer s.NoError(admin.Shutdown())
	defer s.NoError(member.Shutdown())
}

func (s *MessengerGroupChatSuite) TestGroupChatMembersRemovalOutOfOrder() {
	admin := s.newMessenger()
	memberA := s.newMessenger()
	members := []string{common.PubkeyToHex(&memberA.identity.PublicKey)}

	s.makeMutualContacts(admin, memberA)

	groupChat := s.createGroupChat(admin, "test_group_chat", members)

	removeMembersResponse, err := admin.removeMembersFromGroupChat(context.Background(), groupChat, []string{common.PubkeyToHex(&memberA.identity.PublicKey)})
	s.Require().NoError(err)

	encodedMessage := removeMembersResponse.encodedProtobuf

	message := protobuf.MembershipUpdateMessage{}
	err = proto.Unmarshal(encodedMessage, &message)
	s.Require().NoError(err)

	response := &MessengerResponse{}

	messageState := &ReceivedMessageState{
		ExistingMessagesMap: make(map[string]bool),
		Response:            response,
		AllChats:            new(chatMap),
		Timesource:          memberA.getTimesource(),
	}

	c, err := buildContact(admin.myHexIdentity(), &admin.identity.PublicKey)
	s.Require().NoError(err)

	messageState.CurrentMessageState = &CurrentMessageState{
		Contact: c,
	}

	err = memberA.HandleMembershipUpdate(messageState, nil, &message, memberA.systemMessagesTranslations)

	s.Require().NoError(err)
	s.Require().NotNil(messageState.Response)
	s.Require().Len(messageState.Response.Chats(), 1)
	s.Require().Len(messageState.Response.Chats()[0].Members, 1)
	defer s.NoError(admin.Shutdown())
	defer s.NoError(memberA.Shutdown())
}

func (s *MessengerGroupChatSuite) TestGroupChatMembersInfoSync() {
	admin, memberA, memberB := s.newMessenger(), s.newMessenger(), s.newMessenger()
	s.Require().NoError(admin.settings.SaveSettingField(settings.DisplayName, "admin"))
	s.Require().NoError(memberA.settings.SaveSettingField(settings.DisplayName, "memberA"))
	s.Require().NoError(memberB.settings.SaveSettingField(settings.DisplayName, "memberB"))

	members := []string{common.PubkeyToHex(&memberA.identity.PublicKey), common.PubkeyToHex(&memberB.identity.PublicKey)}

	s.makeMutualContacts(admin, memberA)
	s.makeMutualContacts(admin, memberB)

	s.createGroupChat(admin, "test_group_chat", members)
	s.verifyGroupChatCreated(memberA, true)
	s.verifyGroupChatCreated(memberB, true)

	response, err := WaitOnMessengerResponse(
		memberA,
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
		"chat invitation not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	s.Require().True(response.Chats()[0].Active)
	s.Require().Len(response.Chats()[0].Members, 3)

	_, err = WaitOnMessengerResponse(
		memberA,
		func(r *MessengerResponse) bool {
			// we republish as we don't have store nodes in tests
			err := memberB.publishContactCode()
			if err != nil {
				return false
			}
			contact, ok := memberA.allContacts.Load(common.PubkeyToHex(&memberB.identity.PublicKey))
			return ok && contact.DisplayName == "memberB"
		},
		"DisplayName is not the same",
	)
	s.Require().NoError(err)

	s.NoError(admin.Shutdown())
	s.NoError(memberA.Shutdown())
	s.NoError(memberB.Shutdown())
}
