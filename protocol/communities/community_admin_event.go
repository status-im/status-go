package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToCommunityAdminEvent(eventType protobuf.CommunityAdminEvent_EventType) *protobuf.CommunityAdminEvent {
	event := &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        eventType,
		Description: o.Identity(),
		//	TokenPermissions: o.TokenPermissions(),
	}

	return event
}

func (o *Community) UpdateCommunityByAdmin(adminEvent *protobuf.CommunityAdminEvent) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if adminEvent.Clock <= o.config.CommunityDescription.Clock {
		o.config.Logger.Warn("Clock is outdated")
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_DESCRIPTION_CHANGED:
		o.config.CommunityDescription.Identity = adminEvent.Description
	default:
		// TODO
	}
}
