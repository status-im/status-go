package communities

import (
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToCreateChannelAdminEvent(channelID string, channel *protobuf.CommunityChat) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
			Channel:   channel,
		},
	}
}

func (o *Community) ToEditChannelAdminEvent(channelID string, channel *protobuf.CommunityChat) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_EDIT,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
			Channel:   channel,
		},
	}
}

func (o *Community) ToDeleteChannelAdminEvent(channelID string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
		},
	}
}

func (o *Community) ToReorderChannelAdminEvent(categoryID string, channelID string, position int) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_REORDER,
		ChannelData: &protobuf.ChannelData{
			CategoryId: categoryID,
			ChannelId:  channelID,
			Position:   int32(position),
		},
	}
}

func (o *Community) ToCreateCategoryAdminEvent(categoryID string, categoryName string, channelsIds []string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATE,
		CategoryData: &protobuf.CategoryData{
			Name:        categoryName,
			CategoryId:  categoryID,
			ChannelsIds: channelsIds,
		},
	}
}

func (o *Community) ToEditCategoryAdminEvent(categoryID string, categoryName string, channelsIds []string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_EDIT,
		CategoryData: &protobuf.CategoryData{
			Name:        categoryName,
			CategoryId:  categoryID,
			ChannelsIds: channelsIds,
		},
	}
}

func (o *Community) ToDeleteCategoryAdminEvent(categoryID string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETE,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
		},
	}
}

func (o *Community) ToReorderCategoryAdminEvent(categoryID string, position int) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_REORDER,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
			Position:   int32(position),
		},
	}
}

func (o *Community) ToBanCommunityMemberAdminEvent(pubkey string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN,
		MemberToAction:  pubkey,
	}
}

func (o *Community) ToUnbanCommunityMemberAdminEvent(pubkey string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_UNBAN,
		MemberToAction:  pubkey,
	}
}

func (o *Community) ToKickCommunityMemberAdminEvent(pubkey string) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK,
		MemberToAction:  pubkey,
	}
}

func (o *Community) ToCommunityEditAdminEvent() *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_EDIT,
		CommunityConfig: &protobuf.CommunityConfig{
			Identity:      o.Identity(),
			Permissions:   o.Permissions(),
			AdminSettings: o.AdminSettings(),
			IntroMessage:  o.IntroMessage(),
			OutroMessage:  o.OutroMessage(),
			Tags:          o.TagsRaw(),
		},
	}
}

func (o *Community) ToCommunityTokenPermissionChangeAdminEvent(permission *protobuf.CommunityTokenPermission) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE,
		TokenPermission: permission,
		CommunityConfig: &protobuf.CommunityConfig{
			Encrypted: o.Encrypted(),
		},
	}
}

func (o *Community) ToCommunityTokenPermissionDeleteAdminEvent(permission *protobuf.CommunityTokenPermission) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock: o.NewAdminEventClock(),
		Applied:         true,
		Type:            protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE,
		TokenPermission: permission,
		CommunityConfig: &protobuf.CommunityConfig{
			Encrypted: o.Encrypted(),
		},
	}
}

func (o *Community) ToCommunityRequestToJoinAcceptAdminEvent(changes *CommunityAdminEventChanges) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock:        o.NewAdminEventClock(),
		Applied:                true,
		Type:                   protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
		MembersAdded:           changes.MembersAdded,
		AcceptedRequestsToJoin: changes.AcceptedRequestsToJoin,
	}
}

func (o *Community) ToCommunityRequestToJoinRejectAdminEvent(changes *CommunityAdminEventChanges) *CommunityAdminEvent {
	return &CommunityAdminEvent{
		AdminEventClock:        o.NewAdminEventClock(),
		Applied:                true,
		Type:                   protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT,
		RejectedRequestsToJoin: changes.RejectedRequestsToJoin,
	}
}

func (o *Community) UpdateCommuntyByAdminsEvents(adminEventMessage *CommunityEventsMessage) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if adminEventMessage.CommunityDescriptionClock != o.config.CommunityDescription.Clock {
		return nil, errors.New("clock for admin event message is outdated")
	}

	// Merge community admin events to existing community. Admin events must be stored to the db
	// during saving the community
	o.mergeCommunityAdminEvents(adminEventMessage.AdminsEvents)

	// Create a deep copy of current community so we can update CommunityDescription by new admin events
	copy := o.createDeepCopy()

	// Update the copy of the Community with the new community events
	err := copy.updateCommuntyDescriptionByNewAdminsEvents()
	if err != nil {
		return nil, err
	}

	// Collect `CommunityChanges` data by searching a difference between `CommunityDescrption`
	// with applied events and `CommunityDescrption` with new events
	changes, err := o.collectCommuntyChanges(copy.config.CommunityDescription)
	if err != nil {
		return nil, err
	}

	changes.Community = copy

	return changes, nil
}

func (o *Community) updateCommuntyDescriptionByNewAdminsEvents() error {
	prevAdminClock := o.config.CommunityDescription.Clock + 1
	for i := range o.config.AdminsEvents {
		adminEvent := &o.config.AdminsEvents[i]
		if !adminEvent.Applied {
			err := validateCommunityAdminEvent(adminEvent)
			if err != nil {
				return err
			}
			err = o.updateCommuntyDescriptionByAdminEvent(*adminEvent, prevAdminClock)
			if err != nil {
				return err
			}
		}
		prevAdminClock = adminEvent.AdminEventClock
		adminEvent.Applied = true
	}
	return nil
}

func (o *Community) updateCommuntyDescriptionByAppliedAdminsEvents() error {
	prevAdminClock := o.config.CommunityDescription.Clock + 1
	for _, adminEvent := range o.config.AdminsEvents {
		if adminEvent.Applied {
			err := o.updateCommuntyDescriptionByAdminEvent(adminEvent, prevAdminClock)
			if err != nil {
				return err
			}
		}
		prevAdminClock = adminEvent.AdminEventClock
	}
	return nil
}

func (o *Community) updateCommuntyDescriptionByAdminEvent(adminEvent CommunityAdminEvent, prevAdminClock uint64) error {
	if adminEvent.AdminEventClock <= o.config.CommunityDescription.Clock {
		return errors.New("clock for admin event is outdated")
	}

	if !(prevAdminClock == adminEvent.AdminEventClock || prevAdminClock+1 == adminEvent.AdminEventClock) {
		return errors.New("clock for admin event is not in order or contains missing event")
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_EDIT:
		o.config.CommunityDescription.Identity = adminEvent.CommunityConfig.Identity
		o.config.CommunityDescription.Permissions = adminEvent.CommunityConfig.Permissions
		o.config.CommunityDescription.AdminSettings = adminEvent.CommunityConfig.AdminSettings
		o.config.CommunityDescription.IntroMessage = adminEvent.CommunityConfig.IntroMessage
		o.config.CommunityDescription.OutroMessage = adminEvent.CommunityConfig.OutroMessage
		o.config.CommunityDescription.Tags = adminEvent.CommunityConfig.Tags

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE:
		_, exists := o.config.CommunityDescription.TokenPermissions[adminEvent.TokenPermission.Id]
		if exists {
			_, err := o.updateTokenPermission(adminEvent.TokenPermission)
			if err != nil {
				return err
			}
		} else {
			_, err := o.addTokenPermission(adminEvent.TokenPermission)
			if err != nil {
				return err
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE:
		_, err := o.deleteTokenPermission(adminEvent.TokenPermission.Id)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATE:
		_, err := o.createCategory(adminEvent.CategoryData.CategoryId, adminEvent.CategoryData.Name, adminEvent.CategoryData.ChannelsIds)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETE:
		_, err := o.deleteCategory(adminEvent.CategoryData.CategoryId)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_EDIT:
		_, err := o.editCategory(adminEvent.CategoryData.CategoryId, adminEvent.CategoryData.Name, adminEvent.CategoryData.ChannelsIds)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATE:
		err := o.createChat(adminEvent.ChannelData.ChannelId, adminEvent.ChannelData.Channel)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETE:
		o.deleteChat(adminEvent.ChannelData.ChannelId)

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_EDIT:
		err := o.editChat(adminEvent.ChannelData.ChannelId, adminEvent.ChannelData.Channel)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_REORDER:
		_, err := o.reorderChat(adminEvent.ChannelData.CategoryId, adminEvent.ChannelData.ChannelId, int(adminEvent.ChannelData.Position))
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_REORDER:
		_, err := o.reorderCategories(adminEvent.CategoryData.CategoryId, int(adminEvent.CategoryData.Position))
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		for pkString, addedMember := range adminEvent.MembersAdded {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return err
			}
			if !o.HasMember(pk) {
				o.addCommunityMember(pk, addedMember)
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		for pkString := range adminEvent.RejectedRequestsToJoin {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return err
			}
			o.removeMemberFromOrg(pk)
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return err
		}

		if o.IsMemberOwnerOrAdmin(pk) {
			return errors.New("attempt to kick an owner or admin of the community from the admin side")
		}

		o.removeMemberFromOrg(pk)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return err
		}

		if o.IsMemberOwnerOrAdmin(pk) {
			return errors.New("attempt to ban an owner or admin of the community from the admin side")
		}
		o.banUserFromCommunity(pk)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_UNBAN:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return err
		}
		o.unbanUserFromCommunity(pk)

	default:
		return errors.New("unknown admin community event")
	}
	return nil
}

func (o *Community) NewAdminEventClock() uint64 {
	if len(o.config.AdminsEvents) == 0 {
		return o.config.CommunityDescription.Clock + 1
	}
	return o.config.AdminsEvents[len(o.config.AdminsEvents)-1].AdminEventClock + 1
}

func (o *Community) addNewCommunityAdminEvent(event *CommunityAdminEvent) error {
	err := validateCommunityAdminEvent(event)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(event.ToProtobuf())
	if err != nil {
		return errors.New("converting CommunityAdminEvent to protobuf failed")
	}
	event.RawPayload = data
	o.config.AdminsEvents = append(o.config.AdminsEvents, *event)

	return nil
}

func (o *Community) ToCommunityEventsMessage() *CommunityEventsMessage {
	return &CommunityEventsMessage{
		CommunityID:               o.ID(),
		CommunityDescriptionClock: o.Clock(),
		AdminsEvents:              o.config.AdminsEvents,
	}
}
