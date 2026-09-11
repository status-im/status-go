package protocol

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/testutils"
)

func TestLocalNotificationSettingsSuite(t *testing.T) {
	suite.Run(t, new(LocalNotificationSettingsSuite))
}

type LocalNotificationSettingsSuite struct {
	MessengerBaseTestSuite
}

func (s *LocalNotificationSettingsSuite) SetupTest() {
	s.MessengerBaseTestSuite.setupMessaging()
	s.m = s.newMessenger()
	s.privateKey = s.m.identity
}

// TestOneToOneChatsTurnOff verifies that when OneToOneChats is TurnOff,
// no local notification is produced for 1:1 messages.
func (s *LocalNotificationSettingsSuite) TestOneToOneChatsTurnOff() {
	bob := s.m
	alice := s.newMessenger()

	// Bob disables 1:1 notifications
	s.Require().NoError(bob.settings.SetOneToOneChats(notifValueTurnOff))

	// Create 1:1 chat from alice -> bob and send message
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&bob.identity.PublicKey))
	chat := CreateOneToOneChat(pkString, &bob.identity.PublicKey, alice.getTimesource())
	s.Require().NoError(alice.SaveChat(chat))
	inputMessage := buildTestMessage(*chat)
	_, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Sync until bob receives the message
	var bobResponse *MessengerResponse
	err = testutils.RetryWithBackOff(func() error {
		_, err := alice.RetrieveAll()
		if err != nil {
			return err
		}
		bobResponse, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(bobResponse.Messages()) == 0 {
			return errors.New("messages not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().NotNil(bobResponse)

	// Bob should have received the message but no local notification
	s.Require().NotEmpty(bobResponse.Messages())
	s.Require().Empty(bobResponse.Notifications(), "OneToOneChats=TurnOff should produce no local notifications")
}

// TestOneToOneChatsSendAlerts verifies that when OneToOneChats is SendAlerts,
// a local notification is produced for 1:1 messages.
func (s *LocalNotificationSettingsSuite) TestOneToOneChatsSendAlerts() {
	bob := s.m
	alice := s.newMessenger()

	// Ensure 1:1 notifications are on (default)
	s.Require().NoError(bob.settings.SetOneToOneChats(notifValueSendAlerts))

	// Create 1:1 chat from alice -> bob and send message
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&bob.identity.PublicKey))
	chat := CreateOneToOneChat(pkString, &bob.identity.PublicKey, alice.getTimesource())
	s.Require().NoError(alice.SaveChat(chat))
	inputMessage := buildTestMessage(*chat)
	_, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Sync until bob receives the message
	var bobResponse *MessengerResponse
	err = testutils.RetryWithBackOff(func() error {
		_, err := alice.RetrieveAll()
		if err != nil {
			return err
		}
		bobResponse, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(bobResponse.Messages()) == 0 {
			return errors.New("messages not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().NotNil(bobResponse)

	// Bob should have a local notification
	s.Require().NotEmpty(bobResponse.Messages())
	s.Require().NotEmpty(bobResponse.Notifications(), "OneToOneChats=SendAlerts should produce a local notification")
}

// TestGroupChatsTurnOff verifies that when GroupChats is TurnOff,
// no local notification is produced for private group messages.
func (s *LocalNotificationSettingsSuite) TestGroupChatsTurnOff() {
	bob := s.m
	alice := s.newMessenger()

	// Bob disables group notifications
	s.Require().NoError(bob.settings.SetGroupChats(notifValueTurnOff))

	// Create group chat with alice as creator, add bob as member
	response, err := alice.CreateGroupChatWithMembers(context.Background(), "test-group", []string{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	chat := response.Chats()[0]
	s.Require().NoError(alice.SaveChat(chat))

	s.Require().NoError(makeMutualContact(alice, &bob.identity.PublicKey))
	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&bob.identity.PublicKey))}
	_, err = alice.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.Require().NoError(err)

	// Bob receives invite and confirms
	_, err = WaitOnMessengerResponse(bob, func(r *MessengerResponse) bool { return len(r.Chats()) > 0 }, "chat invite")
	s.Require().NoError(err)
	_, err = bob.ConfirmJoiningGroup(context.Background(), chat.ID)
	s.Require().NoError(err)

	// Wait for alice to see bob joined
	_, err = WaitOnMessengerResponse(alice, func(r *MessengerResponse) bool { return len(r.Chats()) > 0 }, "join event")
	s.Require().NoError(err)

	// Alice sends message
	inputMessage := buildTestMessage(*chat)
	_, err = alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	var bobResponse *MessengerResponse
	err = testutils.RetryWithBackOff(func() error {
		_, err := alice.RetrieveAll()
		if err != nil {
			return err
		}
		bobResponse, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		if len(bobResponse.Messages()) == 0 {
			return errors.New("messages not received")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().NotNil(bobResponse)

	s.Require().NotEmpty(bobResponse.Messages())
	s.Require().Empty(bobResponse.Notifications(), "GroupChats=TurnOff should produce no local notifications")
}

// TestContactRequestsTurnOff_OneToOneContactRequest verifies that one-to-one
// pending contact requests obey the dedicated ContactRequests setting instead
// of falling through to the generic one-to-one chat notification setting.
func (s *LocalNotificationSettingsSuite) TestContactRequestsTurnOff_OneToOneContactRequest() {
	bob := s.m
	alice := s.newMessenger()

	s.Require().NoError(bob.settings.SetContactRequests(notifValueTurnOff))
	s.Require().NoError(bob.settings.SetOneToOneChats(notifValueSendAlerts))

	request := &requests.SendContactRequest{
		ID:      types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey)),
		Message: "hello!",
	}
	_, err := alice.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	response, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) >= 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"contact request not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().NotEmpty(response.Messages())
	s.Require().NotEmpty(response.ActivityCenterNotifications())
	s.Require().Empty(response.Notifications(), "ContactRequests=TurnOff should suppress 1:1 contact request local notifications")
}

// TestContactRequestsSendAlerts_OneToOneContactRequest verifies that enabling
// ContactRequests keeps the existing one-to-one contact request alert behavior,
// even when ordinary one-to-one chat messages are muted separately.
func (s *LocalNotificationSettingsSuite) TestContactRequestsSendAlerts_OneToOneContactRequest() {
	bob := s.m
	alice := s.newMessenger()

	s.Require().NoError(bob.settings.SetContactRequests(notifValueSendAlerts))
	s.Require().NoError(bob.settings.SetOneToOneChats(notifValueTurnOff))

	request := &requests.SendContactRequest{
		ID:      types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey)),
		Message: "hello!",
	}
	_, err := alice.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	response, err := WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) >= 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"contact request not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().NotEmpty(response.Messages())
	s.Require().NotEmpty(response.ActivityCenterNotifications())
	s.Require().NotEmpty(response.Notifications(), "ContactRequests=SendAlerts should allow 1:1 contact request local notifications")
}

// TestContactRequestsTurnOff_GroupInvite verifies that when ContactRequests is TurnOff,
// no local notification is produced for private group invites.
func (s *LocalNotificationSettingsSuite) TestContactRequestsTurnOff_GroupInvite() {
	bob := s.m
	alice := s.newMessenger()

	// Bob disables contact-request notifications (group invites)
	s.Require().NoError(bob.settings.SetContactRequests(notifValueTurnOff))

	// Alice creates group and invites Bob
	response, err := alice.CreateGroupChatWithMembers(context.Background(), "test-group", []string{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	chat := response.Chats()[0]
	s.Require().NoError(alice.SaveChat(chat))

	s.Require().NoError(makeMutualContact(alice, &bob.identity.PublicKey))
	s.Require().NoError(makeMutualContact(bob, &alice.identity.PublicKey))
	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&bob.identity.PublicKey))}
	_, err = alice.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.Require().NoError(err)

	// Bob receives the group invite
	inviteResponse, err := WaitOnMessengerResponse(bob, func(r *MessengerResponse) bool { return len(r.Chats()) > 0 }, "chat invite")
	s.Require().NoError(err)
	s.Require().NotNil(inviteResponse)

	// Bob should have received the chat but no local notification for the invite
	s.Require().NotEmpty(inviteResponse.Chats())
	s.Require().Empty(inviteResponse.Notifications(), "ContactRequests=TurnOff should produce no local notification for group invite")
}

// TestContactRequestsSendAlerts_GroupInvite verifies that when ContactRequests is SendAlerts,
// a local notification is produced for private group invites.
func (s *LocalNotificationSettingsSuite) TestContactRequestsSendAlerts_GroupInvite() {
	bob := s.m
	alice := s.newMessenger()

	// Bob enables contact-request notifications (default)
	s.Require().NoError(bob.settings.SetContactRequests(notifValueSendAlerts))

	// Alice creates group and invites Bob
	response, err := alice.CreateGroupChatWithMembers(context.Background(), "test-group", []string{})
	s.Require().NoError(err)
	s.Require().Len(response.Chats(), 1)
	chat := response.Chats()[0]
	s.Require().NoError(alice.SaveChat(chat))

	s.Require().NoError(makeMutualContact(alice, &bob.identity.PublicKey))
	s.Require().NoError(makeMutualContact(bob, &alice.identity.PublicKey))
	members := []string{"0x" + hex.EncodeToString(crypto.FromECDSAPub(&bob.identity.PublicKey))}
	_, err = alice.AddMembersToGroupChat(context.Background(), chat.ID, members)
	s.Require().NoError(err)

	// Bob receives the group invite
	inviteResponse, err := WaitOnMessengerResponse(bob, func(r *MessengerResponse) bool { return len(r.Chats()) > 0 }, "chat invite")
	s.Require().NoError(err)
	s.Require().NotNil(inviteResponse)

	// Bob should have a local notification for the invite
	s.Require().NotEmpty(inviteResponse.Chats())
	s.Require().NotEmpty(inviteResponse.Notifications(), "ContactRequests=SendAlerts should produce a local notification for group invite")
}
