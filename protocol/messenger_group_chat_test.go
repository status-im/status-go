package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	userimage "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestGroupChatSuite(t *testing.T) {
	suite.Run(t, new(MessengerGroupChatSuite))
}

type MessengerGroupChatSuite struct {
	suite.Suite

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerGroupChatSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, []Option{})
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerGroupChatSuite) startNewMessenger() *Messenger {
	messenger := s.newMessenger()

	_, err := messenger.Start()
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerGroupChatSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())
}

func (s *MessengerGroupChatSuite) TearDownTest() {
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
		func(r *MessengerResponse) bool { return len(r.Chats()) > 0 },
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
	contact.Added = true
	contact.HasAddedUs = true
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
		creator := s.startNewMessenger()
		member := s.startNewMessenger()
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
		admin := s.startNewMessenger()
		inviter := s.startNewMessenger()
		member := s.startNewMessenger()
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
	admin := s.startNewMessenger()
	memberA := s.startNewMessenger()
	memberB := s.startNewMessenger()
	members := []string{common.PubkeyToHex(&memberA.identity.PublicKey), common.PubkeyToHex(&memberB.identity.PublicKey)}

	s.makeMutualContacts(admin, memberA)
	s.makeMutualContacts(admin, memberB)

	groupChat := s.createGroupChat(admin, "test_group_chat", members)
	s.verifyGroupChatCreated(memberA, true)
	s.verifyGroupChatCreated(memberB, true)

	_, err := memberA.RemoveMembersFromGroupChat(context.Background(), groupChat.ID, []string{common.PubkeyToHex(&memberB.identity.PublicKey)})
	s.Require().Error(err)

	// only admin can remove members from the group
	_, err = admin.RemoveMembersFromGroupChat(context.Background(), groupChat.ID, []string{common.PubkeyToHex(&memberB.identity.PublicKey)})
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
}

func (s *MessengerGroupChatSuite) TestGroupChatEdit() {
	admin := s.startNewMessenger()
	member := s.startNewMessenger()
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
