package protocol

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestActivityCenterPersistence(t *testing.T) {
	suite.Run(t, new(ActivityCenterPersistenceTestSuite))
}

type ActivityCenterPersistenceTestSuite struct {
	suite.Suite
	idCounter int
}

func (s *ActivityCenterPersistenceTestSuite) SetupTest() {
	s.idCounter = 0
}

func currentMilliseconds() uint64 {
	c := time.Now().UnixMilli()
	return uint64(c)
}

func (s *ActivityCenterPersistenceTestSuite) createNotifications(p *sqlitePersistence, notifications []*ActivityCenterNotification) []*ActivityCenterNotification {
	now := currentMilliseconds()
	for index, notif := range notifications {
		if notif.Timestamp == 0 {
			notif.Timestamp = now
		}
		if len(notif.ID) == 0 {
			s.idCounter++
			notif.ID = types.HexBytes(strconv.Itoa(s.idCounter + index))
		}
		if notif.UpdatedAt == 0 {
			notif.UpdatedAt = now
		}
		_, err := p.SaveActivityCenterNotification(notif, true)
		s.Require().NoError(err, notif.ID)
	}

	// Fetches notifications to get an up-to-date slice.
	var createdNotifications []*ActivityCenterNotification
	for _, notif := range notifications {
		n, err := p.GetActivityCenterNotificationByID(notif.ID)
		s.Require().NoError(err, notif.ID)
		createdNotifications = append(createdNotifications, n)
	}

	return createdNotifications
}

func (s *ActivityCenterPersistenceTestSuite) Test_DeleteActivityCenterNotificationsWhenEmpty() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := common.NewMessage()
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	s.createNotifications(p, []*ActivityCenterNotification{
		{
			Type: ActivityCenterNotificationTypeMention,
		},
	})

	var count uint64
	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().Equal(uint64(1), count)

	_, err = p.DeleteActivityCenterNotifications([]types.HexBytes{}, currentMilliseconds())
	s.Require().NoError(err)

	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().Equal(uint64(1), count)
}

func (s *ActivityCenterPersistenceTestSuite) Test_DeleteActivityCenterNotificationsWithMultipleIds() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := common.NewMessage()
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	notifications := s.createNotifications(p, []*ActivityCenterNotification{
		{Type: ActivityCenterNotificationTypeMention},
		{Type: ActivityCenterNotificationTypeNewOneToOne},
		{Type: ActivityCenterNotificationTypeNewOneToOne},
	})

	var count uint64
	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().Equal(uint64(3), count)

	_, err = p.DeleteActivityCenterNotifications([]types.HexBytes{notifications[1].ID, notifications[2].ID}, currentMilliseconds())
	s.Require().NoError(err)

	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().Equal(uint64(1), count)
}

func (s *ActivityCenterPersistenceTestSuite) Test_DeleteActivityCenterNotificationsForMessage() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	chat2 := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat2)
	s.Require().NoError(err)

	messages := []*common.Message{
		{
			ID:          "0x1",
			ChatMessage: &protobuf.ChatMessage{},
			LocalChatID: chat.ID,
		},
		{
			ID:          "0x2",
			ChatMessage: &protobuf.ChatMessage{},
			LocalChatID: chat.ID,
		},
		{
			ChatMessage: &protobuf.ChatMessage{},
			ID:          "0x3",
		},
	}
	err = p.SaveMessages(messages)
	s.Require().NoError(err)

	chat.LastMessage = messages[1]
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	chatMessages, _, err := p.MessageByChatID(chat.ID, "", 2)
	s.Require().NoError(err)
	s.Require().Len(chatMessages, 2)

	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")
	nID3 := types.HexBytes("3")
	nID4 := types.HexBytes("4")

	s.createNotifications(p, []*ActivityCenterNotification{
		{
			ID:      nID1,
			ChatID:  chat.ID,
			Type:    ActivityCenterNotificationTypeMention,
			Message: messages[0],
		},
		{
			ID:      nID2,
			ChatID:  chat.ID,
			Type:    ActivityCenterNotificationTypeMention,
			Message: messages[1],
		},
		{
			ID:     nID3,
			ChatID: chat.ID,
			Type:   ActivityCenterNotificationTypeMention,
		},
		{
			ID:   nID4,
			Type: ActivityCenterNotificationTypeMention,
		},
	})

	// Test: soft delete only the notifications that have Message.ID == messages[0].ID.
	_, err = p.DeleteActivityCenterNotificationForMessage(chat.ID, messages[0].ID, currentMilliseconds())
	s.Require().NoError(err)

	notif, err := p.GetActivityCenterNotificationByID(nID1)
	s.Require().NoError(err)
	s.Require().True(notif.Deleted)
	s.Require().True(notif.Dismissed)
	s.Require().True(notif.Read)

	// Other notifications are not affected.
	for _, id := range []types.HexBytes{nID2, nID3, nID4} {
		notif, err = p.GetActivityCenterNotificationByID(id)
		s.Require().NoError(err)
		s.Require().False(notif.Deleted, notif.ID)
		s.Require().False(notif.Dismissed, notif.ID)
		s.Require().False(notif.Read, notif.ID)
	}

	// Test: soft delete the notifications that have Message.ID == messages[1].ID
	// or LastMessage.ID == chat.LastMessage.
	_, err = p.DeleteActivityCenterNotificationForMessage(chat.ID, messages[1].ID, currentMilliseconds())
	s.Require().NoError(err)

	for _, id := range []types.HexBytes{nID2, nID3} {
		notif, err = p.GetActivityCenterNotificationByID(id)
		s.Require().NoError(err, notif.ID)
		s.Require().True(notif.Deleted, notif.ID)
		s.Require().True(notif.Dismissed, notif.ID)
		s.Require().True(notif.Read, notif.ID)
	}

	notif, err = p.GetActivityCenterNotificationByID(nID4)
	s.Require().NoError(err)
	s.Require().False(notif.Deleted)
	s.Require().False(notif.Dismissed)
	s.Require().False(notif.Read)

	// Test: don't do anything if passed a chat and message without notifications.
	_, err = p.DeleteActivityCenterNotificationForMessage(chat2.ID, messages[2].ID, currentMilliseconds())
	s.Require().NoError(err)
}

func (s *ActivityCenterPersistenceTestSuite) Test_AcceptActivityCenterNotificationsForInvitesFromUser() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")
	nID3 := types.HexBytes("3")
	nID4 := types.HexBytes("4")

	userPublicKey := "zQ3sh"

	notifications := []*ActivityCenterNotification{
		{
			ID:        nID1,
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			Timestamp: 1,
		},
		{
			ID:        nID2,
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			Timestamp: 1,
			Author:    userPublicKey,
		},
		{
			ID:        nID3,
			Type:      ActivityCenterNotificationTypeMention,
			Timestamp: 1,
			Author:    userPublicKey,
		},
		{
			ID:        nID4,
			Timestamp: 1,
			Type:      ActivityCenterNotificationTypeMention,
		},
	}

	var notif *ActivityCenterNotification
	for _, notif = range notifications {
		_, err = p.SaveActivityCenterNotification(notif, true)
		s.Require().NoError(err, notif.ID)
	}

	// Only notifications of type new private group chat and with Author equal to
	// userPublicKey should be marked as accepted & read.
	_, err = p.GetActivityCenterNotificationByID(nID2)
	s.Require().NoError(err)
	s.Require().False(notifications[0].Accepted)
	s.Require().False(notifications[0].Read)

	notifications, err = p.AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey, 1)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID2, notifications[0].ID)

	notif, err = p.GetActivityCenterNotificationByID(nID2)
	s.Require().NoError(err)
	s.Require().True(notif.Accepted)
	s.Require().True(notif.Read)

	// Deleted notifications are ignored.
	notif = &ActivityCenterNotification{
		ID:        types.HexBytes("99"),
		Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
		Timestamp: 1,
		Author:    userPublicKey,
		Deleted:   true,
	}
	_, err = p.SaveActivityCenterNotification(notif, true)
	s.Require().NoError(err)
	_, err = p.AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey, currentMilliseconds())
	s.Require().NoError(err)
	notif, err = p.GetActivityCenterNotificationByID(notif.ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().True(notif.Deleted)

	// Dismissed notifications are ignored.
	notif = &ActivityCenterNotification{
		ID:        types.HexBytes("100"),
		Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
		Timestamp: 1,
		Author:    userPublicKey,
		Dismissed: true,
	}
	_, err = p.SaveActivityCenterNotification(notif, true)
	s.Require().NoError(err)
	_, err = p.AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey, currentMilliseconds())
	s.Require().NoError(err)
	notif, err = p.GetActivityCenterNotificationByID(notif.ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().True(notif.Dismissed)
}

func (s *ActivityCenterPersistenceTestSuite) Test_GetToProcessActivityCenterNotificationIds() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	notifications := s.createNotifications(p, []*ActivityCenterNotification{
		{
			Type:    ActivityCenterNotificationTypeNewPrivateGroupChat,
			Deleted: true,
		},
		{
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			Dismissed: true,
		},
		{
			Type:     ActivityCenterNotificationTypeMention,
			Accepted: true,
		},
		{
			Type: ActivityCenterNotificationTypeMention,
		},
	})

	ids, err := p.GetToProcessActivityCenterNotificationIds()
	s.Require().NoError(err)
	s.Require().Len(ids, 1)
	s.Require().Equal(notifications[3].ID, types.HexBytes(ids[0]))
}

func (s *ActivityCenterPersistenceTestSuite) Test_HasPendingNotificationsForChat() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	// Test: there are no notifications.
	result, err := p.HasPendingNotificationsForChat(chat.ID)
	s.Require().NoError(err)
	s.Require().False(result)

	// Test: there are only deleted, dismissed or accepted notifications,
	// therefore, no pending notifications.
	s.createNotifications(p, []*ActivityCenterNotification{
		{
			ChatID:  chat.ID,
			Type:    ActivityCenterNotificationTypeNewPrivateGroupChat,
			Deleted: true,
		},
		{
			ChatID:    chat.ID,
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			Dismissed: true,
		},
		{
			ChatID:   chat.ID,
			Type:     ActivityCenterNotificationTypeMention,
			Accepted: true,
		},
	})

	result, err = p.HasPendingNotificationsForChat(chat.ID)
	s.Require().NoError(err)
	s.Require().False(result)

	// Test: there's one pending notification.
	notif := &ActivityCenterNotification{
		ID:        types.HexBytes("99"),
		ChatID:    chat.ID,
		Type:      ActivityCenterNotificationTypeCommunityRequest,
		Timestamp: 1,
	}
	_, err = p.SaveActivityCenterNotification(notif, true)
	s.Require().NoError(err)

	result, err = p.HasPendingNotificationsForChat(chat.ID)
	s.Require().NoError(err)
	s.Require().True(result)
}

func (s *ActivityCenterPersistenceTestSuite) Test_DismissAllActivityCenterNotificationsFromUser() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	publicKey := "0x04"

	notifications := s.createNotifications(p, []*ActivityCenterNotification{
		{
			Type:    ActivityCenterNotificationTypeNewPrivateGroupChat,
			Author:  publicKey,
			Deleted: true,
		},
		{
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			Author:    publicKey,
			Dismissed: true,
		},
		{
			Type:     ActivityCenterNotificationTypeMention,
			Author:   publicKey,
			Accepted: true,
		},
		{
			Type:   ActivityCenterNotificationTypeMention,
			Author: "0x09",
		},
		{
			Type:   ActivityCenterNotificationTypeMention,
			Author: publicKey,
		},
	})

	_, err = p.DismissAllActivityCenterNotificationsFromUser(publicKey, 1)
	s.Require().NoError(err)

	// Ignores already soft deleted.
	notif, err := p.GetActivityCenterNotificationByID(notifications[0].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().True(notif.Deleted)

	// Ignores already dismissed.
	notif, err = p.GetActivityCenterNotificationByID(notifications[1].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().True(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Ignores already accepted.
	notif, err = p.GetActivityCenterNotificationByID(notifications[2].ID)
	s.Require().NoError(err)
	s.Require().True(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Ignores notification from different author.
	notif, err = p.GetActivityCenterNotificationByID(notifications[3].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Finally, dismiss and mark as read this one notification.
	notif, err = p.GetActivityCenterNotificationByID(notifications[4].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().True(notif.Read)
	s.Require().True(notif.Dismissed)
	s.Require().False(notif.Deleted)
}

func (s *ActivityCenterPersistenceTestSuite) Test_DismissAllActivityCenterNotificationsFromChatID() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chatID := "0x99"

	notifications := s.createNotifications(p, []*ActivityCenterNotification{
		{
			ChatID:  chatID,
			Type:    ActivityCenterNotificationTypeNewPrivateGroupChat,
			Deleted: true,
		},
		{
			ChatID:    chatID,
			Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
			Dismissed: true,
		},
		{
			ChatID:   chatID,
			Type:     ActivityCenterNotificationTypeMention,
			Accepted: true,
		},
		{
			Type: ActivityCenterNotificationTypeMention,
		},
		{
			ChatID: chatID,
			Type:   ActivityCenterNotificationTypeContactRequest,
		},
		{
			ChatID: chatID,
			Type:   ActivityCenterNotificationTypeMention,
		},
	})

	_, err = p.DismissAllActivityCenterNotificationsFromChatID(chatID, currentMilliseconds())
	s.Require().NoError(err)

	// Ignores already soft deleted.
	notif, err := p.GetActivityCenterNotificationByID(notifications[0].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().True(notif.Deleted)

	// Do not ignore already dismissed, because notifications can become
	// read/unread AND dismissed, and the method should still update the Read
	// column.
	notif, err = p.GetActivityCenterNotificationByID(notifications[1].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().True(notif.Read)
	s.Require().True(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Ignores already accepted.
	notif, err = p.GetActivityCenterNotificationByID(notifications[2].ID)
	s.Require().NoError(err)
	s.Require().True(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Ignores notification from different chat.
	notif, err = p.GetActivityCenterNotificationByID(notifications[3].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Ignores contact request notifications.
	notif, err = p.GetActivityCenterNotificationByID(notifications[4].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().False(notif.Read)
	s.Require().False(notif.Dismissed)
	s.Require().False(notif.Deleted)

	// Finally, dismiss and mark as read this one notification.
	notif, err = p.GetActivityCenterNotificationByID(notifications[5].ID)
	s.Require().NoError(err)
	s.Require().False(notif.Accepted)
	s.Require().True(notif.Read)
	s.Require().True(notif.Dismissed)
	s.Require().False(notif.Deleted)
}

func (s *ActivityCenterPersistenceTestSuite) Test_ActiveContactRequestNotification() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	contactID := "0x99"

	// Test: ignores deleted/dismissed/accepted notifications, as well as
	// notifications not associated to any chat.
	s.createNotifications(p, []*ActivityCenterNotification{
		{
			ChatID:  chat.ID,
			Author:  contactID,
			Type:    ActivityCenterNotificationTypeContactRequest,
			Deleted: true,
		},
		{
			ChatID:    chat.ID,
			Author:    contactID,
			Type:      ActivityCenterNotificationTypeContactRequest,
			Dismissed: true,
		},
		{
			ChatID:   chat.ID,
			Author:   contactID,
			Type:     ActivityCenterNotificationTypeContactRequest,
			Accepted: true,
		},
	})

	notif, err := p.ActiveContactRequestNotification(contactID)
	s.Require().NoError(err)
	s.Require().Nil(notif)

	// Test: Ignores notifications that are not contact requests.
	s.createNotifications(p, []*ActivityCenterNotification{
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeCommunityInvitation},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeCommunityKicked},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeCommunityMembershipRequest},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeCommunityRequest},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeContactVerification},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeMention},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeNewOneToOne},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeNewPrivateGroupChat},
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeReply},
	})

	notif, err = p.ActiveContactRequestNotification(contactID)
	s.Require().NoError(err)
	s.Require().Nil(notif)

	// Test: Returns one, and only one contact request notification for the
	// contact under test.
	s.createNotifications(p, []*ActivityCenterNotification{
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeContactRequest},
	})

	notif, err = p.ActiveContactRequestNotification(contactID)
	s.Require().NoError(err)
	s.Require().NotNil(notif)
	s.Require().Equal(ActivityCenterNotificationTypeContactRequest, notif.Type)

	// Test: In case there's more than one notification, return the most recent
	// one according to the notification's timestamp.
	expectedID := types.HexBytes("667")

	t1 := currentMilliseconds()
	t2 := currentMilliseconds()
	s.createNotifications(p, []*ActivityCenterNotification{
		{
			ID:        expectedID,
			Timestamp: t2 + 1,
			ChatID:    chat.ID,
			Author:    contactID,
			Type:      ActivityCenterNotificationTypeContactRequest,
		},
		{
			ID:        types.HexBytes("666"),
			Timestamp: t1,
			ChatID:    chat.ID,
			Author:    contactID,
			Type:      ActivityCenterNotificationTypeContactRequest,
		},
	})

	notif, err = p.ActiveContactRequestNotification(contactID)
	s.Require().NoError(err)
	s.Require().Equal(expectedID, notif.ID)
}

func (s *ActivityCenterPersistenceTestSuite) Test_UnreadActivityCenterNotificationsCount() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	s.createNotifications(p, []*ActivityCenterNotification{
		{Type: ActivityCenterNotificationTypeMention, Read: true},
		{Type: ActivityCenterNotificationTypeNewOneToOne, Deleted: true},
		{Type: ActivityCenterNotificationTypeMention, Dismissed: true},
		{Type: ActivityCenterNotificationTypeCommunityRequest, Accepted: true},
		{Type: ActivityCenterNotificationTypeContactRequest},
	})

	// Test: Ignore soft deleted and accepted.
	count, err := p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, true)
	s.Require().NoError(err)
	s.Require().Equal(uint64(3), count)
}

func (s *ActivityCenterPersistenceTestSuite) Test_UnreadAndAcceptedActivityCenterNotificationsCount() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	s.createNotifications(p, []*ActivityCenterNotification{
		{Type: ActivityCenterNotificationTypeMention, Read: true},
		{Type: ActivityCenterNotificationTypeNewOneToOne, Deleted: true},
		{Type: ActivityCenterNotificationTypeMention, Dismissed: true},
		{Type: ActivityCenterNotificationTypeCommunityRequest, Accepted: true},
		{Type: ActivityCenterNotificationTypeContactRequest},
	})

	// Test: counts everything, except soft deleted notifications.
	count, err := p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, true)
	s.Require().NoError(err)
	s.Require().Equal(uint64(3), count)

	// Test: counts everything, except soft deleted ones and limit by type.
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{
		ActivityCenterNotificationTypeContactRequest,
	}, ActivityCenterQueryParamsReadUnread, true)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)
}

func (s *ActivityCenterPersistenceTestSuite) Test_ActivityCenterPersistence() {
	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	nID3 := types.HexBytes([]byte("3"))
	nID4 := types.HexBytes([]byte("4"))

	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := common.NewMessage()
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	notification := &ActivityCenterNotification{
		ID:        nID1,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}
	_, err = p.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)

	cursor, notifications, err := p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Len(notifications, 1)
	s.Require().Equal(chat.ID, notifications[0].ChatID)
	s.Require().Equal(message, notifications[0].LastMessage)

	// Add another notification

	notification = &ActivityCenterNotification{
		ID:        nID2,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 2,
	}
	_, err = p.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)

	cursor, notifications, err = p.ActivityCenterNotifications("", 1, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().NotEmpty(cursor)
	s.Require().Equal(nID2, notifications[0].ID)

	// fetch next pagination

	cursor, notifications, err = p.ActivityCenterNotifications(cursor, 1, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Empty(cursor)
	s.Require().False(notifications[0].Read)
	s.Require().Equal(nID1, notifications[0].ID)

	// Check count
	count, err := p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), count)

	var updatedAt uint64 = 1
	// Mark first one as read
	s.Require().NoError(p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID1}, updatedAt))
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)

	// Mark first one as unread
	updatedAt++
	_, err = p.MarkActivityCenterNotificationsUnread([]types.HexBytes{nID1}, updatedAt)
	s.Require().NoError(err)
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), count)

	// Mark all read
	updatedAt++
	s.Require().NoError(p.MarkAllActivityCenterNotificationsRead(updatedAt))
	_, notifications, err = p.ActivityCenterNotifications(cursor, 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)
	s.Require().Len(notifications, 2)
	s.Require().Empty(cursor)
	s.Require().True(notifications[0].Read)
	s.Require().True(notifications[1].Read)

	// Check count
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	s.Require().NoError(err)
	s.Require().Equal(uint64(0), count)

	// Mark first one as accepted
	updatedAt++
	notifications, err = p.AcceptActivityCenterNotifications([]types.HexBytes{nID1}, updatedAt)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)
	// It should not be returned anymore
	s.Require().Len(notifications, 1)

	// Mark last one as dismissed
	updatedAt++
	s.Require().NoError(p.DismissActivityCenterNotifications([]types.HexBytes{nID2}, updatedAt))
	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)

	s.Require().Len(notifications, 1)
	s.Require().True(notifications[0].Dismissed)

	// Insert new notification
	notification = &ActivityCenterNotification{
		ID:        nID3,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 3,
	}
	_, err = p.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)

	// Mark all as accepted
	updatedAt++
	notifications, err = p.AcceptAllActivityCenterNotifications(updatedAt)
	s.Require().NoError(err)
	s.Require().Len(notifications, 2)

	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)

	s.Require().Len(notifications, 1)

	// Insert new notification
	notification = &ActivityCenterNotification{
		ID:        nID4,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 4,
	}
	_, err = p.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)

	// Mark all as dismissed
	updatedAt++
	s.Require().NoError(p.DismissAllActivityCenterNotifications(updatedAt))
	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	s.Require().NoError(err)

	s.Require().Len(notifications, 2)
	s.Require().True(notifications[0].Dismissed)
	s.Require().True(notifications[1].Dismissed)
}

func (s *ActivityCenterPersistenceTestSuite) Test_ActivityCenterReadUnreadPagination() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	initialOrFinalCursor := ""

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := common.NewMessage()
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")
	nID3 := types.HexBytes("3")
	nID4 := types.HexBytes("4")
	nID5 := types.HexBytes("5")

	allNotifications := []*ActivityCenterNotification{
		{
			ID:        nID1,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID2,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID3,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID4,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID5,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
	}

	for _, notification := range allNotifications {
		_, err = p.SaveActivityCenterNotification(notification, true)
		s.Require().NoError(err)
	}

	// Mark the notification as read
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID2}, currentMilliseconds())
	s.Require().NoError(err)
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID4}, currentMilliseconds())
	s.Require().NoError(err)

	// Fetch UNREAD notifications, first page.
	cursor, notifications, err := p.ActivityCenterNotifications(
		initialOrFinalCursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID5, notifications[0].ID)
	s.Require().NotEmpty(cursor)

	// Fetch next pages.
	cursor, notifications, err = p.ActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID3, notifications[0].ID)
	s.Require().NotEmpty(cursor)

	cursor, notifications, err = p.ActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID1, notifications[0].ID)
	s.Require().Empty(cursor)

	// Fetch READ notifications, first page.
	cursor, notifications, err = p.ActivityCenterNotifications(
		initialOrFinalCursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID4, notifications[0].ID)
	s.Require().NotEmpty(cursor)

	// Fetch next page.
	cursor, notifications, err = p.ActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID2, notifications[0].ID)
	s.Require().Empty(cursor)
}

func (s *ActivityCenterPersistenceTestSuite) Test_ActivityCenterReadUnreadFilterByTypes() {
	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := common.NewMessage()
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	initialCursor := ""
	limit := uint64(3)

	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")
	nID3 := types.HexBytes("3")

	allNotifications := []*ActivityCenterNotification{
		{
			ID:        nID1,
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID2,
			Type:      ActivityCenterNotificationTypeNewOneToOne,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
		{
			ID:        nID3,
			Type:      ActivityCenterNotificationTypeMention,
			ChatID:    chat.ID,
			Timestamp: 1,
		},
	}

	for _, notification := range allNotifications {
		_, err = p.SaveActivityCenterNotification(notification, true)
		s.Require().NoError(err)
	}

	// Don't filter by type if the array of types is empty.
	_, notifications, err := p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 3)
	s.Require().Equal(nID3, notifications[0].ID)
	s.Require().Equal(nID2, notifications[1].ID)
	s.Require().Equal(nID1, notifications[2].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID2, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 2)
	s.Require().Equal(nID3, notifications[0].ID)
	s.Require().Equal(nID1, notifications[1].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention, ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 3)
	s.Require().Equal(nID3, notifications[0].ID)
	s.Require().Equal(nID2, notifications[1].ID)
	s.Require().Equal(nID1, notifications[2].ID)

	// Mark all notifications as read.
	for _, notification := range allNotifications {
		err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{notification.ID}, currentMilliseconds())
		s.Require().NoError(err)
	}

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID2, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	s.Require().NoError(err)
	s.Require().Len(notifications, 2)
	s.Require().Equal(nID3, notifications[0].ID)
	s.Require().Equal(nID1, notifications[1].ID)
}

func (s *ActivityCenterPersistenceTestSuite) Test_ActivityCenterReadUnread() {
	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")

	db, err := openTestDB()
	s.Require().NoError(err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := common.NewMessage()
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	s.Require().NoError(err)

	notification := &ActivityCenterNotification{
		ID:        nID1,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}

	_, err = p.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)

	notification = &ActivityCenterNotification{
		ID:        nID2,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}

	_, err = p.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)

	// Mark the notification as read
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID2}, currentMilliseconds())
	s.Require().NoError(err)

	cursor, notifications, err := p.ActivityCenterNotifications(
		"",
		2,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID1, notifications[0].ID)

	cursor, notifications, err = p.ActivityCenterNotifications(
		"",
		2,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	s.Require().NoError(err)
	s.Require().Empty(cursor)
	s.Require().Len(notifications, 1)
	s.Require().Equal(nID2, notifications[0].ID)
}
