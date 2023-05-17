package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToCommunityAdminEvent(eventType protobuf.CommunityAdminEvent_EventType) *protobuf.CommunityAdminEvent {
	switch eventType {
	case protobuf.CommunityAdminEvent_COMMUNITY_DESCRIPTION_CHANGED:
		return &protobuf.CommunityAdminEvent{
			Clock:       o.Clock(),
			CommunityId: o.ID(),
			Type:        eventType,
			Description: o.Identity(),
		}
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGED:
		// TODO admin permission
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CHANGED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CHANGED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CATEGORY_REORDERED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPTED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECTED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_DELETE_MESSAGE:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_PIN_MESSAGES:
		// TODO admin permission
		break
	default:
		// TODO: return error
	}

	return &protobuf.CommunityAdminEvent{}
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
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CHANGED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CHANGED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CATEGORY_REORDERED:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPTED:
		// TODO admin permission: must be accepted by owner only and owner update others
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECTED:
		// TODO admin permission: must be accepted by owner only and owner update others
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK:
		// TODO admin permission: must be accepted by owner only and owner update others
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN:
		// TODO admin permission: must be accepted by owner only and owner update others
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_DELETE_MESSAGE:
		// TODO admin permission
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_PIN_MESSAGES:
		// TODO admin permission
		break
	default:
		// TODO return error
	}
}
