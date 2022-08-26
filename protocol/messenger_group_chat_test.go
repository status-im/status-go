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

		defer creator.Shutdown()
		defer member.Shutdown()
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

		defer admin.Shutdown()
		defer inviter.Shutdown()
		defer member.Shutdown()
	}
}

func (s *MessengerGroupChatSuite) TestGroupChatEdit() {

}
