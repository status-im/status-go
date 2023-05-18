package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToCommunityAdminEvent(eventType protobuf.CommunityAdminEvent_EventType) *protobuf.CommunityAdminEvent {
	switch eventType {
	case protobuf.CommunityAdminEvent_COMMUNITY_CONFIG_CHANGED:
		return &protobuf.CommunityAdminEvent{
			Clock:       o.Clock(),
			CommunityId: o.ID(),
			Type:        eventType,
			CommunityConfig: &protobuf.CommunityConfig{
				Identity:      o.Identity(),
				Permissions:   o.Permissions(),
				AdminSettings: o.AdminSettings(),
				IntroMessage:  o.IntroMessage(),
				OutroMessage:  o.OutroMessage(),
				Tags:          o.TagsRaw(),
			},
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
		// TODO: log error
	}

	return &protobuf.CommunityAdminEvent{Type: protobuf.CommunityAdminEvent_UNKNOWN}
}

func (o *Community) ChangesToCommunityAdminEvent(eventType protobuf.CommunityAdminEvent_EventType,
	changes *CommunityChanges) *protobuf.CommunityAdminEvent {
	switch eventType {
	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATED:
		return &protobuf.CommunityAdminEvent{
			Clock:       o.Clock(),
			CommunityId: o.ID(),
			Type:        eventType,
			Chats:       changes.ChatsAdded,
		}
	default:
		// TODO: log error
	}
	return &protobuf.CommunityAdminEvent{Type: protobuf.CommunityAdminEvent_UNKNOWN}
}

func (o *Community) UpdateCommunityByAdmin(adminEvent *protobuf.CommunityAdminEvent) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if adminEvent.Clock <= o.config.CommunityDescription.Clock {
		o.config.Logger.Warn("Clock is outdated")
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_CONFIG_CHANGED:
		o.config.CommunityDescription.Identity = adminEvent.CommunityConfig.Identity
		o.config.CommunityDescription.Permissions = adminEvent.CommunityConfig.Permissions
		o.config.CommunityDescription.AdminSettings = adminEvent.CommunityConfig.AdminSettings
		o.config.CommunityDescription.IntroMessage = adminEvent.CommunityConfig.IntroMessage
		o.config.CommunityDescription.OutroMessage = adminEvent.CommunityConfig.OutroMessage
		o.config.CommunityDescription.Tags = adminEvent.CommunityConfig.Tags
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
		for key, value := range adminEvent.Chats {
			if _, ok := o.config.CommunityDescription.Chats[key]; !ok {
				if o.config.CommunityDescription.Chats == nil {
					o.config.CommunityDescription.Chats = make(map[string]*protobuf.CommunityChat)
				}
				o.config.CommunityDescription.Chats[key] = value
			}
		}
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
