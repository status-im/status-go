package protocol

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/protobuf"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/requests"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
)

const (
	actionAccept  = "Accept"
	actionDecline = "Decline"
)

type MessengerSyncActivityCenterSuite struct {
	MessengerBaseTestSuite
	m2 *Messenger
}

func TestMessengerSyncActivityCenter(t *testing.T) {
	suite.Run(t, new(MessengerSyncActivityCenterSuite))
}

func (s *MessengerSyncActivityCenterSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	m2, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)
	s.m2 = m2

	PairDevices(&s.Suite, m2, s.m)
	PairDevices(&s.Suite, s.m, m2)
}

func (s *MessengerSyncActivityCenterSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.m2)
	s.MessengerBaseTestSuite.TearDownTest()
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

	id := s.createAndSaveNotification(s.m, notiType, initial)
	s.createAndSaveNotification(s.m2, notiType, initial)

	now := uint64(time.Now().Unix())
	_, err := action(s.m, context.Background(), []types.HexBytes{id}, now+1, true)
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		return r.ActivityCenterState() != nil
	}, "activity center notification state not received")
	s.Require().NoError(err)

	notificationByID, err := s.m2.persistence.GetActivityCenterNotificationByID(id)
	s.Require().NoError(err)
	s.Require().True(validator(notificationByID))
}

func (s *MessengerSyncActivityCenterSuite) TestSyncCommunityRequestDecisionAccept() {
	s.testSyncCommunityRequestDecision(actionAccept)
}

func (s *MessengerSyncActivityCenterSuite) TestSyncCommunityRequestDecisionDecline() {
	s.testSyncCommunityRequestDecision(actionDecline)
}

func (s *MessengerSyncActivityCenterSuite) testSyncCommunityRequestDecision(action string) {
	userB := s.createUserB()
	defer func() {
		s.Require().NoError(userB.Shutdown())
	}()

	communityID := s.createClosedCommunity()

	s.addContactAndShareCommunity(userB, communityID)

	s.requestToJoinCommunity(userB, communityID)

	requestToJoinID := s.waitForRequestToJoin(s.m)

	s.waitForRequestToJoinOnDevice2()

	switch action {
	case actionAccept:
		_, err := s.m.AcceptRequestToJoinCommunity(&requests.AcceptRequestToJoinCommunity{ID: requestToJoinID})
		s.Require().NoError(err)
	case actionDecline:
		_, err := s.m.DeclineRequestToJoinCommunity(&requests.DeclineRequestToJoinCommunity{ID: requestToJoinID})
		s.Require().NoError(err)
	default:
		s.T().Fatal("Unknown action")
	}

	s.waitForDecisionOnDevice2(requestToJoinID, action)
}

func (s *MessengerSyncActivityCenterSuite) createUserB() *Messenger {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	userB, err := newMessengerWithKey(s.shh, key, s.logger, nil)
	s.Require().NoError(err)
	return userB
}

func (s *MessengerSyncActivityCenterSuite) createClosedCommunity() types.HexBytes {
	response, err := s.m.CreateClosedCommunity()
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	response, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0
	}, "community not received on device 2")
	s.Require().NoError(err)
	return response.Communities()[0].ID()
}

func (s *MessengerSyncActivityCenterSuite) addContactAndShareCommunity(userB *Messenger, communityID types.HexBytes) {
	request := &requests.AddContact{ID: common.PubkeyToHex(&s.m2.identity.PublicKey)}
	response, err := userB.AddContact(context.Background(), request)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 2)
	existContactRequestMessage := false
	for _, m := range response.Messages() {
		if m.ContentType == protobuf.ChatMessage_CONTACT_REQUEST {
			existContactRequestMessage = true
		}
	}
	s.Require().True(existContactRequestMessage)
	var contactRequestMessageID types.HexBytes
	_, err = WaitOnMessengerResponse(s.m, func(r *MessengerResponse) bool {
		if len(r.ActivityCenterNotifications()) > 0 {
			for _, n := range r.ActivityCenterNotifications() {
				if n.Type == ActivityCenterNotificationTypeContactRequest {
					contactRequestMessageID = n.ID
					return true
				}
			}
		}
		return false
	}, "contact request not received on device 1")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		if len(r.ActivityCenterNotifications()) > 0 {
			for _, n := range r.ActivityCenterNotifications() {
				if n.Type == ActivityCenterNotificationTypeContactRequest {
					return true
				}
			}
		}
		return false
	}, "contact request not received on device 2")
	s.Require().NoError(err)
	_, err = s.m.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: contactRequestMessageID})
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		if len(r.ActivityCenterNotifications()) > 0 {
			for _, n := range r.ActivityCenterNotifications() {
				if n.Type == ActivityCenterNotificationTypeContactRequest && n.Accepted {
					return true
				}
			}
		}
		return false
	}, "contact request not accepted on device 2")
	s.Require().NoError(err)
	community, err := s.m.GetCommunityByID(communityID)
	s.Require().NoError(err)
	advertiseCommunityTo(&s.Suite, community.ID(), s.m, userB)
}

func (s *MessengerSyncActivityCenterSuite) requestToJoinCommunity(userB *Messenger, communityID types.HexBytes) {
	_, err := userB.RequestToJoinCommunity(&requests.RequestToJoinCommunity{CommunityID: communityID})
	s.Require().NoError(err)
}

func (s *MessengerSyncActivityCenterSuite) waitForRequestToJoin(messenger *Messenger) types.HexBytes {
	var requestToJoinID types.HexBytes
	_, err := WaitOnMessengerResponse(messenger, func(r *MessengerResponse) bool {
		for _, n := range r.ActivityCenterNotifications() {
			if n.Type == ActivityCenterNotificationTypeCommunityMembershipRequest {
				requestToJoinID = n.ID
				return true
			}
		}
		return false
	}, "community request to join not received")
	s.Require().NoError(err)
	return requestToJoinID
}

func (s *MessengerSyncActivityCenterSuite) waitForRequestToJoinOnDevice2() {
	_, err := WaitOnMessengerResponse(s.m2, func(r *MessengerResponse) bool {
		for _, n := range r.ActivityCenterNotifications() {
			if n.Type == ActivityCenterNotificationTypeCommunityMembershipRequest {
				return true
			}
		}
		return false
	}, "community request to join not received on device 2")
	s.Require().NoError(err)
}

func (s *MessengerSyncActivityCenterSuite) waitForDecisionOnDevice2(requestToJoinID types.HexBytes, action string) {
	requestToJoinIDString := hex.Dump(requestToJoinID)
	conditionFunc := func(r *MessengerResponse) bool {
		for _, n := range r.ActivityCenterNotifications() {
			if n.Type == ActivityCenterNotificationTypeCommunityMembershipRequest && hex.Dump(n.ID) == requestToJoinIDString {
				if action == actionAccept && n.Accepted {
					return true
				} else if action == actionDecline && n.Dismissed {
					return true
				}
			}
		}
		return false
	}
	_, err := WaitOnMessengerResponse(s.m2, conditionFunc, "community request decision not received on device 2")
	s.Require().NoError(err)
}
