package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	v1protocol "github.com/status-im/status-go/protocol/v1"
)

func TestEventToSystemMessageSuite(t *testing.T) {
	suite.Run(t, new(EventToSystemMessageSuite))
}

type EventToSystemMessageSuite struct {
	suite.Suite
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
			Event:    v1protocol.NewChatCreatedEvent("chat-name", 12),
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
			Name:     "members added event",
			Event:    v1protocol.NewMembersAddedEvent([]string{"a", "b", "c"}, 12),
			From:     "admin",
			Expected: "@admin has invited @a, @b, @c",
		},
		{
			Name:     "member joined event",
			Event:    v1protocol.NewMemberJoinedEvent(12),
			From:     "admin",
			Expected: "@admin joined the group",
		},
		{
			Name:     "admins added event",
			Event:    v1protocol.NewAdminsAddedEvent([]string{"a", "b", "c"}, 12),
			From:     "admin",
			Expected: "@admin has made @a, @b, @c admin",
		},
		{
			Name:     "member removed event",
			Event:    v1protocol.NewMemberRemovedEvent("a", 12),
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
