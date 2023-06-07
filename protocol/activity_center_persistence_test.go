package protocol

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
)

func currentMilliseconds() uint64 {
	c := time.Now().UnixMilli()
	return uint64(c)
}

func createNotifications(t *testing.T, p *sqlitePersistence, notifications []*ActivityCenterNotification) []*ActivityCenterNotification {
	now := currentMilliseconds()
	for index, notif := range notifications {
		if notif.Timestamp == 0 {
			notif.Timestamp = now
		}
		if len(notif.ID) == 0 {
			notif.ID = types.HexBytes(strconv.Itoa(index))
		}
		if notif.UpdatedAt == 0 {
			notif.UpdatedAt = now
		}
		err := p.SaveActivityCenterNotification(notif, true)
		require.NoError(t, err, notif.ID)
	}

	// Fetches notifications to get an up-to-date slice.
	var createdNotifications []*ActivityCenterNotification
	for _, notif := range notifications {
		n, err := p.GetActivityCenterNotificationByID(notif.ID)
		require.NoError(t, err, notif.ID)
		createdNotifications = append(createdNotifications, n)
	}

	return createdNotifications
}

func TestDeleteActivityCenterNotificationsWhenEmpty(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	createNotifications(t, p, []*ActivityCenterNotification{
		{
			Type: ActivityCenterNotificationTypeMention,
		},
	})

	var count uint64
	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.Equal(t, uint64(1), count)

	_, err = p.DeleteActivityCenterNotifications([]types.HexBytes{}, currentMilliseconds())
	require.NoError(t, err)

	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.Equal(t, uint64(1), count)
}

func TestDeleteActivityCenterNotificationsWithMultipleIds(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	notifications := createNotifications(t, p, []*ActivityCenterNotification{
		{Type: ActivityCenterNotificationTypeMention},
		{Type: ActivityCenterNotificationTypeNewOneToOne},
		{Type: ActivityCenterNotificationTypeNewOneToOne},
	})

	var count uint64
	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.Equal(t, uint64(3), count)

	_, err = p.DeleteActivityCenterNotifications([]types.HexBytes{notifications[1].ID, notifications[2].ID}, currentMilliseconds())
	require.NoError(t, err)

	count, _ = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.Equal(t, uint64(1), count)
}

func TestDeleteActivityCenterNotificationsForMessage(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	chat2 := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat2)
	require.NoError(t, err)

	messages := []*common.Message{
		{
			ID:          "0x1",
			LocalChatID: chat.ID,
		},
		{
			ID:          "0x2",
			LocalChatID: chat.ID,
		},
		{
			ID: "0x3",
		},
	}
	err = p.SaveMessages(messages)
	require.NoError(t, err)

	chat.LastMessage = messages[1]
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	chatMessages, _, err := p.MessageByChatID(chat.ID, "", 2)
	require.NoError(t, err)
	require.Len(t, chatMessages, 2)

	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")
	nID3 := types.HexBytes("3")
	nID4 := types.HexBytes("4")

	createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)

	notif, err := p.GetActivityCenterNotificationByID(nID1)
	require.NoError(t, err)
	require.Nil(t, notif)

	// Other notifications are not affected.
	for _, id := range []types.HexBytes{nID2, nID3, nID4} {
		notif, err = p.GetActivityCenterNotificationByID(id)
		require.NoError(t, err)
		require.False(t, notif.Deleted, notif.ID)
		require.False(t, notif.Dismissed, notif.ID)
		require.False(t, notif.Read, notif.ID)
	}

	// Test: soft delete the notifications that have Message.ID == messages[1].ID
	// or LastMessage.ID == chat.LastMessage.
	_, err = p.DeleteActivityCenterNotificationForMessage(chat.ID, messages[1].ID, currentMilliseconds())
	require.NoError(t, err)

	for _, id := range []types.HexBytes{nID2, nID3} {
		notif, err = p.GetActivityCenterNotificationByID(id)
		require.NoError(t, err)
		require.Nil(t, notif)
	}

	notif, err = p.GetActivityCenterNotificationByID(nID4)
	require.NoError(t, err)
	require.False(t, notif.Deleted)
	require.False(t, notif.Dismissed)
	require.False(t, notif.Read)

	// Test: don't do anything if passed a chat and message without notifications.
	_, err = p.DeleteActivityCenterNotificationForMessage(chat2.ID, messages[2].ID, currentMilliseconds())
	require.NoError(t, err)
}

func TestAcceptActivityCenterNotificationsForInvitesFromUser(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
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
		err = p.SaveActivityCenterNotification(notif, true)
		require.NoError(t, err, notif.ID)
	}

	// Only notifications of type new private group chat and with Author equal to
	// userPublicKey should be marked as accepted & read.
	_, err = p.GetActivityCenterNotificationByID(nID2)
	require.NoError(t, err)
	require.False(t, notifications[0].Accepted)
	require.False(t, notifications[0].Read)

	notifications, err = p.AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey, 1)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)

	notif, err = p.GetActivityCenterNotificationByID(nID2)
	require.NoError(t, err)
	require.True(t, notif.Accepted)
	require.True(t, notif.Read)

	// Deleted notifications are ignored.
	notif = &ActivityCenterNotification{
		ID:        types.HexBytes("99"),
		Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
		Timestamp: 1,
		Author:    userPublicKey,
		Deleted:   true,
	}
	err = p.SaveActivityCenterNotification(notif, true)
	require.NoError(t, err)
	_, err = p.AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey, currentMilliseconds())
	require.NoError(t, err)
	notif, err = p.GetActivityCenterNotificationByID(notif.ID)
	require.NoError(t, err)
	require.Nil(t, notif)

	// Dismissed notifications are ignored.
	notif = &ActivityCenterNotification{
		ID:        types.HexBytes("100"),
		Type:      ActivityCenterNotificationTypeNewPrivateGroupChat,
		Timestamp: 1,
		Author:    userPublicKey,
		Dismissed: true,
	}
	err = p.SaveActivityCenterNotification(notif, true)
	require.NoError(t, err)
	_, err = p.AcceptActivityCenterNotificationsForInvitesFromUser(userPublicKey, currentMilliseconds())
	require.NoError(t, err)
	notif, err = p.GetActivityCenterNotificationByID(notif.ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.False(t, notif.Read)
	require.True(t, notif.Dismissed)
}

func TestGetToProcessActivityCenterNotificationIds(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	notifications := createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, notifications[3].ID, types.HexBytes(ids[0]))
}

func TestHasPendingNotificationsForChat(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	// Test: there are no notifications.
	result, err := p.HasPendingNotificationsForChat(chat.ID)
	require.NoError(t, err)
	require.False(t, result)

	// Test: there are only deleted, dismissed or accepted notifications,
	// therefore, no pending notifications.
	createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)
	require.False(t, result)

	// Test: there's one pending notification.
	notif := &ActivityCenterNotification{
		ID:        types.HexBytes("99"),
		ChatID:    chat.ID,
		Type:      ActivityCenterNotificationTypeCommunityRequest,
		Timestamp: 1,
	}
	err = p.SaveActivityCenterNotification(notif, true)
	require.NoError(t, err)

	result, err = p.HasPendingNotificationsForChat(chat.ID)
	require.NoError(t, err)
	require.True(t, result)
}

func TestDismissAllActivityCenterNotificationsFromUser(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	publicKey := "0x04"

	notifications := createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)

	// Ignores already soft deleted.
	require.Nil(t, notifications[0])

	// Ignores already dismissed.
	notif, err := p.GetActivityCenterNotificationByID(notifications[1].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.False(t, notif.Read)
	require.True(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Ignores already accepted.
	notif, err = p.GetActivityCenterNotificationByID(notifications[2].ID)
	require.NoError(t, err)
	require.True(t, notif.Accepted)
	require.False(t, notif.Read)
	require.False(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Ignores notification from different author.
	notif, err = p.GetActivityCenterNotificationByID(notifications[3].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.False(t, notif.Read)
	require.False(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Finally, dismiss and mark as read this one notification.
	notif, err = p.GetActivityCenterNotificationByID(notifications[4].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.True(t, notif.Read)
	require.True(t, notif.Dismissed)
	require.False(t, notif.Deleted)
}

func TestDismissAllActivityCenterNotificationsFromChatID(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chatID := "0x99"

	notifications := createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)

	// Ignores already soft deleted.
	require.Nil(t, notifications[0])

	// Do not ignore already dismissed, because notifications can become
	// read/unread AND dismissed, and the method should still update the Read
	// column.
	notif, err := p.GetActivityCenterNotificationByID(notifications[1].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.True(t, notif.Read)
	require.True(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Ignores already accepted.
	notif, err = p.GetActivityCenterNotificationByID(notifications[2].ID)
	require.NoError(t, err)
	require.True(t, notif.Accepted)
	require.False(t, notif.Read)
	require.False(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Ignores notification from different chat.
	notif, err = p.GetActivityCenterNotificationByID(notifications[3].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.False(t, notif.Read)
	require.False(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Ignores contact request notifications.
	notif, err = p.GetActivityCenterNotificationByID(notifications[4].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.False(t, notif.Read)
	require.False(t, notif.Dismissed)
	require.False(t, notif.Deleted)

	// Finally, dismiss and mark as read this one notification.
	notif, err = p.GetActivityCenterNotificationByID(notifications[5].ID)
	require.NoError(t, err)
	require.False(t, notif.Accepted)
	require.True(t, notif.Read)
	require.True(t, notif.Dismissed)
	require.False(t, notif.Deleted)
}

func TestActiveContactRequestNotification(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	contactID := "0x99"

	// Test: ignores deleted/dismissed/accepted notifications, as well as
	// notifications not associated to any chat.
	createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)
	require.Nil(t, notif)

	// Test: Ignores notifications that are not contact requests.
	createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)
	require.Nil(t, notif)

	// Test: Returns one, and only one contact request notification for the
	// contact under test.
	createNotifications(t, p, []*ActivityCenterNotification{
		{ChatID: chat.ID, Author: contactID, Type: ActivityCenterNotificationTypeContactRequest},
	})

	notif, err = p.ActiveContactRequestNotification(contactID)
	require.NoError(t, err)
	require.Equal(t, ActivityCenterNotificationTypeContactRequest, notif.Type)

	// Test: In case there's more than one notification, return the most recent
	// one according to the notification's timestamp.
	expectedID := types.HexBytes("667")

	t1 := currentMilliseconds()
	t2 := currentMilliseconds()
	createNotifications(t, p, []*ActivityCenterNotification{
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
	require.NoError(t, err)
	require.Equal(t, expectedID, notif.ID)
}

func TestUnreadActivityCenterNotificationsCount(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	createNotifications(t, p, []*ActivityCenterNotification{
		{Type: ActivityCenterNotificationTypeMention, Read: true},
		{Type: ActivityCenterNotificationTypeNewOneToOne, Deleted: true},
		{Type: ActivityCenterNotificationTypeMention, Dismissed: true},
		{Type: ActivityCenterNotificationTypeCommunityRequest, Accepted: true},
		{Type: ActivityCenterNotificationTypeContactRequest},
	})

	// Test: Ignore soft deleted and accepted.
	count, err := p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, true)
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)
}

func TestUnreadAndAcceptedActivityCenterNotificationsCount(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	createNotifications(t, p, []*ActivityCenterNotification{
		{Type: ActivityCenterNotificationTypeMention, Read: true},
		{Type: ActivityCenterNotificationTypeNewOneToOne, Deleted: true},
		{Type: ActivityCenterNotificationTypeMention, Dismissed: true},
		{Type: ActivityCenterNotificationTypeCommunityRequest, Accepted: true},
		{Type: ActivityCenterNotificationTypeContactRequest},
	})

	// Test: counts everything, except soft deleted notifications.
	count, err := p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, true)
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)

	// Test: counts everything, except soft deleted ones and limit by type.
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{
		ActivityCenterNotificationTypeContactRequest,
	}, ActivityCenterQueryParamsReadUnread, true)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)
}

func TestActivityCenterPersistence(t *testing.T) {
	nID1 := types.HexBytes([]byte("1"))
	nID2 := types.HexBytes([]byte("2"))
	nID3 := types.HexBytes([]byte("3"))
	nID4 := types.HexBytes([]byte("4"))

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	notification := &ActivityCenterNotification{
		ID:        nID1,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}
	err = p.SaveActivityCenterNotification(notification, true)
	require.NoError(t, err)

	cursor, notifications, err := p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Len(t, notifications, 1)
	require.Equal(t, chat.ID, notifications[0].ChatID)
	require.Equal(t, message, notifications[0].LastMessage)

	// Add another notification

	notification = &ActivityCenterNotification{
		ID:        nID2,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 2,
	}
	err = p.SaveActivityCenterNotification(notification, true)
	require.NoError(t, err)

	cursor, notifications, err = p.ActivityCenterNotifications("", 1, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.NotEmpty(t, cursor)
	require.Equal(t, nID2, notifications[0].ID)

	// fetch next pagination

	cursor, notifications, err = p.ActivityCenterNotifications(cursor, 1, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Empty(t, cursor)
	require.False(t, notifications[0].Read)
	require.Equal(t, nID1, notifications[0].ID)

	// Check count
	count, err := p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)

	var updatedAt uint64 = 1
	// Mark first one as read
	require.NoError(t, p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID1}, updatedAt))
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	// Mark first one as unread
	updatedAt++
	_, err = p.MarkActivityCenterNotificationsUnread([]types.HexBytes{nID1}, updatedAt)
	require.NoError(t, err)
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)

	// Mark all read
	updatedAt++
	require.NoError(t, p.MarkAllActivityCenterNotificationsRead(updatedAt))
	_, notifications, err = p.ActivityCenterNotifications(cursor, 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Empty(t, cursor)
	require.True(t, notifications[0].Read)
	require.True(t, notifications[1].Read)

	// Check count
	count, err = p.ActivityCenterNotificationsCount([]ActivityCenterType{}, ActivityCenterQueryParamsReadUnread, false)
	require.NoError(t, err)
	require.Equal(t, uint64(0), count)

	// Mark first one as accepted
	updatedAt++
	notifications, err = p.AcceptActivityCenterNotifications([]types.HexBytes{nID1}, updatedAt)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)
	// It should not be returned anymore
	require.Len(t, notifications, 1)

	// Mark last one as dismissed
	updatedAt++
	require.NoError(t, p.DismissActivityCenterNotifications([]types.HexBytes{nID2}, updatedAt))
	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)

	require.Len(t, notifications, 1)
	require.True(t, notifications[0].Dismissed)

	// Insert new notification
	notification = &ActivityCenterNotification{
		ID:        nID3,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 3,
	}
	err = p.SaveActivityCenterNotification(notification, true)
	require.NoError(t, err)

	// Mark all as accepted
	updatedAt++
	notifications, err = p.AcceptAllActivityCenterNotifications(updatedAt)
	require.NoError(t, err)
	require.Len(t, notifications, 2)

	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)

	require.Len(t, notifications, 1)

	// Insert new notification
	notification = &ActivityCenterNotification{
		ID:        nID4,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		Timestamp: 4,
	}
	err = p.SaveActivityCenterNotification(notification, true)
	require.NoError(t, err)

	// Mark all as dismissed
	updatedAt++
	require.NoError(t, p.DismissAllActivityCenterNotifications(updatedAt))
	_, notifications, err = p.ActivityCenterNotifications("", 2, []ActivityCenterType{}, ActivityCenterQueryParamsReadAll, false)
	require.NoError(t, err)

	require.Len(t, notifications, 2)
	require.True(t, notifications[0].Dismissed)
	require.True(t, notifications[1].Dismissed)
}

func TestActivityCenterReadUnreadPagination(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	initialOrFinalCursor := ""

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

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
		err = p.SaveActivityCenterNotification(notification, true)
		require.NoError(t, err)
	}

	// Mark the notification as read
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID2}, currentMilliseconds())
	require.NoError(t, err)
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID4}, currentMilliseconds())
	require.NoError(t, err)

	// Fetch UNREAD notifications, first page.
	cursor, notifications, err := p.ActivityCenterNotifications(
		initialOrFinalCursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID5, notifications[0].ID)
	require.NotEmpty(t, cursor)

	// Fetch next pages.
	cursor, notifications, err = p.ActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID3, notifications[0].ID)
	require.NotEmpty(t, cursor)

	cursor, notifications, err = p.ActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID1, notifications[0].ID)
	require.Empty(t, cursor)

	// Fetch READ notifications, first page.
	cursor, notifications, err = p.ActivityCenterNotifications(
		initialOrFinalCursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID4, notifications[0].ID)
	require.NotEmpty(t, cursor)

	// Fetch next page.
	cursor, notifications, err = p.ActivityCenterNotifications(
		cursor,
		1,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)
	require.Empty(t, cursor)
}

func TestActivityCenterReadUnreadFilterByTypes(t *testing.T) {
	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

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
		err = p.SaveActivityCenterNotification(notification, true)
		require.NoError(t, err)
	}

	// Don't filter by type if the array of types is empty.
	_, notifications, err := p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 3)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID2, notifications[1].ID)
	require.Equal(t, nID1, notifications[2].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID1, notifications[1].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention, ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 3)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID2, notifications[1].ID)
	require.Equal(t, nID1, notifications[2].ID)

	// Mark all notifications as read.
	for _, notification := range allNotifications {
		err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{notification.ID}, currentMilliseconds())
		require.NoError(t, err)
	}

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)

	_, notifications, err = p.ActivityCenterNotifications(
		initialCursor,
		limit,
		[]ActivityCenterType{ActivityCenterNotificationTypeMention},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	require.Equal(t, nID3, notifications[0].ID)
	require.Equal(t, nID1, notifications[1].ID)
}

func TestActivityCenterReadUnread(t *testing.T) {
	nID1 := types.HexBytes("1")
	nID2 := types.HexBytes("2")

	db, err := openTestDB()
	require.NoError(t, err)
	p := newSQLitePersistence(db)

	chat := CreatePublicChat("test-chat", &testTimeSource{})
	message := &common.Message{}
	message.Text = "sample text"
	chat.LastMessage = message
	err = p.SaveChat(*chat)
	require.NoError(t, err)

	notification := &ActivityCenterNotification{
		ID:        nID1,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}

	err = p.SaveActivityCenterNotification(notification, true)
	require.NoError(t, err)

	notification = &ActivityCenterNotification{
		ID:        nID2,
		Type:      ActivityCenterNotificationTypeNewOneToOne,
		ChatID:    chat.ID,
		Timestamp: 1,
	}

	err = p.SaveActivityCenterNotification(notification, true)
	require.NoError(t, err)

	// Mark the notification as read
	err = p.MarkActivityCenterNotificationsRead([]types.HexBytes{nID2}, currentMilliseconds())
	require.NoError(t, err)

	cursor, notifications, err := p.ActivityCenterNotifications(
		"",
		2,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadUnread,
		false,
	)
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Len(t, notifications, 1)
	require.Equal(t, nID1, notifications[0].ID)

	cursor, notifications, err = p.ActivityCenterNotifications(
		"",
		2,
		[]ActivityCenterType{ActivityCenterNotificationTypeNewOneToOne},
		ActivityCenterQueryParamsReadRead,
		false,
	)
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Len(t, notifications, 1)
	require.Equal(t, nID2, notifications[0].ID)
}
