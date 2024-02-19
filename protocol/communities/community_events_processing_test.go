package communities

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestEventsProcessorSuite(t *testing.T) {
	suite.Run(t, new(EventsProcessorSuite))
}

type EventsProcessorSuite struct {
	suite.Suite
}

func (s *EventsProcessorSuite) TestRetainNewestEventsPerPropertyTypeID() {
	processor := &eventsProcessor{
		eventsToApply: []CommunityEvent{
			CommunityEvent{
				CommunityEventClock: 1,
				Type:                protobuf.CommunityEvent_COMMUNITY_EDIT,
			},
			CommunityEvent{
				CommunityEventClock: 2,
				Type:                protobuf.CommunityEvent_COMMUNITY_EDIT,
			},
			CommunityEvent{
				CommunityEventClock: 3,
				Type:                protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
				MemberToAction:      "A",
			},
			CommunityEvent{
				CommunityEventClock: 4,
				Type:                protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT,
				MemberToAction:      "A",
			},
			CommunityEvent{
				CommunityEventClock: 5,
				Type:                protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
				MemberToAction:      "A",
			},
			CommunityEvent{
				CommunityEventClock: 1,
				Type:                protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
				MemberToAction:      "B",
			},
		},
	}

	processor.retainNewestEventsPerEventTypeID()
	s.Require().Len(processor.eventsToApply, 4)

	processor.sortEvents()

	s.Require().Equal(protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT, processor.eventsToApply[0].Type)
	s.Require().EqualValues(1, processor.eventsToApply[0].CommunityEventClock)

	s.Require().Equal(protobuf.CommunityEvent_COMMUNITY_EDIT, processor.eventsToApply[1].Type)
	s.Require().EqualValues(2, processor.eventsToApply[1].CommunityEventClock)

	s.Require().Equal(protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT, processor.eventsToApply[2].Type)
	s.Require().EqualValues(4, processor.eventsToApply[2].CommunityEventClock)

	s.Require().Equal(protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT, processor.eventsToApply[3].Type)
	s.Require().EqualValues(5, processor.eventsToApply[3].CommunityEventClock)
}
