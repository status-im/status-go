package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/waku"
)

func TestEventToSystemMessageSuite(t *testing.T) {
	suite.Run(t, new(EventToSystemMessageSuite))
}

type EventToSystemMessageSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *EventToSystemMessageSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *EventToSystemMessageSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity

	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *EventToSystemMessageSuite) TestRun() {
	testCases := []struct {
		Name     string
		Event    v1protocol.MembershipUpdateEvent
		Expected string
		From     string
	}{
		{
			Name:     "chat created event",
			Event:    v1protocol.NewChatCreatedEvent("chat-name", "#7cda00", 12),
			From:     "admin",
			Expected: "@admin created the group chat-name",
		},
		{
			Name:     "chat name changed event",
			Event:    v1protocol.NewNameChangedEvent("chat-name-2", 12),
			From:     "admin",
			Expected: "@admin changed the group's name to chat-name-2",
		},
		{
			Name:     "chat color changed event",
			Event:    v1protocol.NewColorChangedEvent("#7cda00", 12),
			From:     "admin",
			Expected: "@admin changed the group's color",
		},
		{
			Name:     "chat image changed event",
			Event:    v1protocol.NewImageChangedEvent([]byte{1, 2, 3}, 12),
			From:     "admin",
			Expected: "@admin changed the group's image",
		},
		{
			Name:     "members added event",
			Event:    v1protocol.NewMembersAddedEvent([]string{"a", "b", "c"}, 12),
			From:     "admin",
			Expected: "@admin has added @a, @b, @c",
		},
		{
			Name:     "member joined event",
			Event:    v1protocol.NewMemberJoinedEvent(12),
			From:     "admin",
			Expected: "", // joined events are deprecated
		},
		{
			Name:     "admins added event",
			Event:    v1protocol.NewAdminsAddedEvent([]string{"a", "b", "c"}, 12),
			From:     "admin",
			Expected: "@admin has made @a, @b, @c admin",
		},
		{
			Name:     "member removed event",
			Event:    v1protocol.NewMemberRemovedEvent([]string{"a"}, 12),
			From:     "admin",
			Expected: "@a left the group",
		},
		{
			Name:     "admin removed event",
			Event:    v1protocol.NewAdminRemovedEvent("a", 12),
			From:     "admin",
			Expected: "@a is not admin anymore",
		},
	}
	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			tc.Event.From = tc.From

			systemMessage := eventToSystemMessage(tc.Event, defaultSystemMessagesTranslations)
			s.Equal(systemMessage.Text, tc.Expected)

		})
	}

}

func (s *EventToSystemMessageSuite) TestHandleMembershipUpdate() {
	adminPrivateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	adminPublicKey := types.EncodeHex(crypto.FromECDSAPub(&adminPrivateKey.PublicKey))
	ourPublicKey := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))

	event1 := v1protocol.MembershipUpdateEvent{
		Type:       protobuf.MembershipUpdateEvent_CHAT_CREATED,
		Name:       "test",
		ChatID:     "test-" + adminPublicKey,
		ClockValue: 100,
	}
	err = event1.Sign(adminPrivateKey)
	s.Require().NoError(err)

	event2 := v1protocol.MembershipUpdateEvent{
		Type:       protobuf.MembershipUpdateEvent_MEMBERS_ADDED,
		Members:    []string{adminPublicKey, ourPublicKey},
		ChatID:     "test-" + adminPublicKey,
		ClockValue: 100,
	}
	err = event2.Sign(adminPrivateKey)
	s.Require().NoError(err)

	testMembershipUpdateMessageStruct2 := v1protocol.MembershipUpdateMessage{
		ChatID: "test-" + adminPublicKey,
		Events: []v1protocol.MembershipUpdateEvent{
			event1,
			event2,
		},
	}

	rawMembershipUpdateMessage2, err := testMembershipUpdateMessageStruct2.ToProtobuf()
	s.Require().NoError(err)

	contact, err := BuildContactFromPublicKey(&adminPrivateKey.PublicKey)
	s.Require().NoError(err)

	contact.ContactRequestLocalState = ContactRequestStateSent

	currentMessageState := &CurrentMessageState{
		Contact: contact,
	}

	state := &ReceivedMessageState{
		Response:            &MessengerResponse{},
		Timesource:          s.m.transport,
		CurrentMessageState: currentMessageState,
		ExistingMessagesMap: map[string]bool{},
		AllChats:            s.m.allChats,
	}

	err = s.m.HandleMembershipUpdate(state, nil, *rawMembershipUpdateMessage2, defaultSystemMessagesTranslations)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Notifications(), 1)
	s.Require().Equal(state.Response.Notifications()[0].Category, localnotifications.CategoryGroupInvite)
	s.Require().Len(state.Response.Chats(), 1)

	chat := state.Response.Chats()[0]

	// Decline event, setting chat inactive
	response, err := s.m.LeaveGroupChat(context.Background(), chat.ID, true)
	s.Require().NoError(err)

	s.Require().Len(response.Chats(), 1)

	chat = response.Chats()[0]

	// If the same response is handled, it should not show another notification & the chat should remain inactive
	state.Response = &MessengerResponse{}
	err = s.m.HandleMembershipUpdate(state, chat, *rawMembershipUpdateMessage2, defaultSystemMessagesTranslations)
	s.Require().NoError(err)
	s.Require().Len(state.Response.Notifications(), 0)
	s.Require().Len(state.Response.Chats(), 1)
	s.Require().False(state.Response.Chats()[0].Active)
}
