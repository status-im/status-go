package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
)

type MessengerSyncActivityCenterSuite struct {
	MessengerBaseTestSuite
}

func TestMessengerSyncActivityCenter(t *testing.T) {
	suite.Run(t, new(MessengerSyncActivityCenterSuite))
}

func (s *MessengerSyncActivityCenterSuite) setupMessengerPair() (*Messenger, *Messenger) {
	theirMessenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	PairDevices(&s.Suite, theirMessenger, s.m)
	PairDevices(&s.Suite, s.m, theirMessenger)

	return s.m, theirMessenger
}

func (s *MessengerSyncActivityCenterSuite) createAndSaveNotification(m *Messenger, t ActivityCenterType, read bool) types.HexBytes {
	now := uint64(time.Now().Unix())
	id := types.HexBytes{0x01}
	notification := &ActivityCenterNotification{
		ID:               id,
		Timestamp:        now,
		Type:             t,
		Read:             read,
		Dismissed:        false,
		Accepted:         false,
		MembershipStatus: ActivityCenterMembershipStatusIdle,
		Deleted:          false,
		UpdatedAt:        now,
	}

	num, err := m.persistence.SaveActivityCenterNotification(notification, true)
	s.Require().NoError(err)
	s.Require().Equal(1, int(num))
	return id
}

func (s *MessengerSyncActivityCenterSuite) TestSyncUnread() {
	s.syncTest(ActivityCenterNotificationTypeMention, true, (*Messenger).MarkActivityCenterNotificationsUnread, func(n *ActivityCenterNotification) bool { return !n.Read })
}

func (s *MessengerSyncActivityCenterSuite) TestSyncDeleted() {
	s.syncTest(ActivityCenterNotificationTypeMention, true, (*Messenger).MarkActivityCenterNotificationsDeleted, func(n *ActivityCenterNotification) bool { return n.Deleted })
}

func (s *MessengerSyncActivityCenterSuite) TestSyncRead() {
	s.syncTest(ActivityCenterNotificationTypeMention, false, (*Messenger).MarkActivityCenterNotificationsRead, func(n *ActivityCenterNotification) bool { return n.Read })
}

func (s *MessengerSyncActivityCenterSuite) TestSyncAccepted() {
	s.syncTest(ActivityCenterNotificationTypeContactRequest, false, (*Messenger).AcceptActivityCenterNotifications, func(n *ActivityCenterNotification) bool { return n.Accepted })
}

func (s *MessengerSyncActivityCenterSuite) TestSyncDismissed() {
	s.syncTest(ActivityCenterNotificationTypeContactRequest, false, (*Messenger).DismissActivityCenterNotifications, func(n *ActivityCenterNotification) bool { return n.Dismissed })
}

func (s *MessengerSyncActivityCenterSuite) syncTest(notiType ActivityCenterType, initial bool, action func(*Messenger, context.Context, []types.HexBytes, uint64, bool) (*MessengerResponse, error), validator func(*ActivityCenterNotification) bool) {
	mainMessenger, theirMessenger := s.setupMessengerPair()
	defer func() {
		s.Require().NoError(theirMessenger.Shutdown())
	}()

	id := s.createAndSaveNotification(mainMessenger, notiType, initial)
	s.createAndSaveNotification(theirMessenger, notiType, initial)

	now := uint64(time.Now().Unix())
	_, err := action(mainMessenger, context.Background(), []types.HexBytes{id}, now+1, true)
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(theirMessenger, func(r *MessengerResponse) bool {
		return r.ActivityCenterState() != nil
	}, "activity center notification state not received")
	s.Require().NoError(err)

	notificationByID, err := theirMessenger.persistence.GetActivityCenterNotificationByID(id)
	s.Require().NoError(err)
	s.Require().True(validator(notificationByID))
}
