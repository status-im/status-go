package communities

import (
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToCreateChannelCommunityEvent(channelID string, channel *protobuf.CommunityChat) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CHANNEL_CREATE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
			Channel:   channel,
		},
	}
}

func (o *Community) ToEditChannelCommunityEvent(channelID string, channel *protobuf.CommunityChat) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CHANNEL_EDIT,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
			Channel:   channel,
		},
	}
}

func (o *Community) ToDeleteChannelCommunityEvent(channelID string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CHANNEL_DELETE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
		},
	}
}

func (o *Community) ToReorderChannelCommunityEvent(categoryID string, channelID string, position int) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CHANNEL_REORDER,
		ChannelData: &protobuf.ChannelData{
			CategoryId: categoryID,
			ChannelId:  channelID,
			Position:   int32(position),
		},
	}
}

func (o *Community) ToCreateCategoryCommunityEvent(categoryID string, categoryName string, channelsIds []string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CATEGORY_CREATE,
		CategoryData: &protobuf.CategoryData{
			Name:        categoryName,
			CategoryId:  categoryID,
			ChannelsIds: channelsIds,
		},
	}
}

func (o *Community) ToEditCategoryCommunityEvent(categoryID string, categoryName string, channelsIds []string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CATEGORY_EDIT,
		CategoryData: &protobuf.CategoryData{
			Name:        categoryName,
			CategoryId:  categoryID,
			ChannelsIds: channelsIds,
		},
	}
}

func (o *Community) ToDeleteCategoryCommunityEvent(categoryID string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CATEGORY_DELETE,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
		},
	}
}

func (o *Community) ToReorderCategoryCommunityEvent(categoryID string, position int) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_CATEGORY_REORDER,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
			Position:   int32(position),
		},
	}
}

func (o *Community) ToBanCommunityMemberCommunityEvent(pubkey string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_BAN,
		MemberToAction:      pubkey,
	}
}

func (o *Community) ToUnbanCommunityMemberCommunityEvent(pubkey string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_UNBAN,
		MemberToAction:      pubkey,
	}
}

func (o *Community) ToKickCommunityMemberCommunityEvent(pubkey string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_KICK,
		MemberToAction:      pubkey,
	}
}

func (o *Community) ToCommunityEditCommunityEvent() *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_EDIT,
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

func (o *Community) ToCommunityTokenPermissionChangeCommunityEvent(permission *protobuf.CommunityTokenPermission) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE,
		TokenPermission:     permission,
		CommunityConfig: &protobuf.CommunityConfig{
			Encrypted: o.Encrypted(),
		},
	}
}

func (o *Community) ToCommunityTokenPermissionDeleteCommunityEvent(permission *protobuf.CommunityTokenPermission) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Applied:             true,
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE,
		TokenPermission:     permission,
		CommunityConfig: &protobuf.CommunityConfig{
			Encrypted: o.Encrypted(),
		},
	}
}

func (o *Community) ToCommunityRequestToJoinAcceptCommunityEvent(changes *CommunityEventChanges) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock:    o.NewCommunityEventClock(),
		Applied:                true,
		Type:                   protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
		MembersAdded:           changes.MembersAdded,
		AcceptedRequestsToJoin: changes.AcceptedRequestsToJoin,
	}
}

func (o *Community) ToCommunityRequestToJoinRejectCommunityEvent(changes *CommunityEventChanges) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock:    o.NewCommunityEventClock(),
		Applied:                true,
		Type:                   protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT,
		RejectedRequestsToJoin: changes.RejectedRequestsToJoin,
	}
}

func (o *Community) UpdateCommuntyByEvents(communityEventMessage *CommunityEventsMessage) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if communityEventMessage.CommunityDescriptionClock != o.config.CommunityDescription.Clock {
		return nil, errors.New("clock for admin event message is outdated")
	}

	// Merge community admin events to existing community. Admin events must be stored to the db
	// during saving the community
	o.mergeCommunityEvents(communityEventMessage.Events)

	// Create a deep copy of current community so we can update CommunityDescription by new admin events
	copy := o.createDeepCopy()

	// Update the copy of the Community with the new community events
	err := copy.updateCommuntyDescriptionByNewEvents()
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

func (o *Community) updateCommuntyDescriptionByNewEvents() error {
	prevAdminClock := o.config.CommunityDescription.Clock + 1
	for i := range o.config.Events {
		communityEvent := &o.config.Events[i]
		if !communityEvent.Applied {
			err := validateCommunityEvent(communityEvent)
			if err != nil {
				return err
			}
			err = o.updateCommuntyDescriptionByCommunityEvent(*communityEvent, prevAdminClock)
			if err != nil {
				return err
			}
		}
		prevAdminClock = communityEvent.CommunityEventClock
		communityEvent.Applied = true
	}
	return nil
}

func (o *Community) updateCommuntyDescriptionByAppliedEvents() error {
	prevAdminClock := o.config.CommunityDescription.Clock + 1
	for _, communityEvent := range o.config.Events {
		if communityEvent.Applied {
			err := o.updateCommuntyDescriptionByCommunityEvent(communityEvent, prevAdminClock)
			if err != nil {
				return err
			}
		}
		prevAdminClock = communityEvent.CommunityEventClock
	}
	return nil
}

func (o *Community) updateCommuntyDescriptionByCommunityEvent(communityEvent CommunityEvent, prevAdminClock uint64) error {
	if communityEvent.CommunityEventClock <= o.config.CommunityDescription.Clock {
		return errors.New("clock for admin event is outdated")
	}

	if !(prevAdminClock == communityEvent.CommunityEventClock || prevAdminClock+1 == communityEvent.CommunityEventClock) {
		return errors.New("clock for admin event is not in order or contains missing event")
	}

	switch communityEvent.Type {
	case protobuf.CommunityEvent_COMMUNITY_EDIT:
		o.config.CommunityDescription.Identity = communityEvent.CommunityConfig.Identity
		o.config.CommunityDescription.Permissions = communityEvent.CommunityConfig.Permissions
		o.config.CommunityDescription.AdminSettings = communityEvent.CommunityConfig.AdminSettings
		o.config.CommunityDescription.IntroMessage = communityEvent.CommunityConfig.IntroMessage
		o.config.CommunityDescription.OutroMessage = communityEvent.CommunityConfig.OutroMessage
		o.config.CommunityDescription.Tags = communityEvent.CommunityConfig.Tags

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE:
		_, exists := o.config.CommunityDescription.TokenPermissions[communityEvent.TokenPermission.Id]
		if exists {
			_, err := o.updateTokenPermission(communityEvent.TokenPermission)
			if err != nil {
				return err
			}
		} else {
			_, err := o.addTokenPermission(communityEvent.TokenPermission)
			if err != nil {
				return err
			}
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE:
		_, err := o.deleteTokenPermission(communityEvent.TokenPermission.Id)
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_CREATE:
		_, err := o.createCategory(communityEvent.CategoryData.CategoryId, communityEvent.CategoryData.Name, communityEvent.CategoryData.ChannelsIds)
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_DELETE:
		_, err := o.deleteCategory(communityEvent.CategoryData.CategoryId)
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_EDIT:
		_, err := o.editCategory(communityEvent.CategoryData.CategoryId, communityEvent.CategoryData.Name, communityEvent.CategoryData.ChannelsIds)
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_CREATE:
		err := o.createChat(communityEvent.ChannelData.ChannelId, communityEvent.ChannelData.Channel)
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_DELETE:
		o.deleteChat(communityEvent.ChannelData.ChannelId)

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_EDIT:
		err := o.editChat(communityEvent.ChannelData.ChannelId, communityEvent.ChannelData.Channel)
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CHANNEL_REORDER:
		_, err := o.reorderChat(communityEvent.ChannelData.CategoryId, communityEvent.ChannelData.ChannelId, int(communityEvent.ChannelData.Position))
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_CATEGORY_REORDER:
		_, err := o.reorderCategories(communityEvent.CategoryData.CategoryId, int(communityEvent.CategoryData.Position))
		if err != nil {
			return err
		}

	case protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		for pkString, addedMember := range communityEvent.MembersAdded {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return err
			}
			if !o.HasMember(pk) {
				o.addCommunityMember(pk, addedMember)
			}
		}

	case protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		for pkString := range communityEvent.RejectedRequestsToJoin {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return err
			}
			o.removeMemberFromOrg(pk)
		}

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_KICK:
		pk, err := common.HexToPubkey(communityEvent.MemberToAction)
		if err != nil {
			return err
		}

		if o.IsMemberOwnerOrAdmin(pk) {
			return errors.New("attempt to kick an owner or admin of the community from the admin side")
		}

		o.removeMemberFromOrg(pk)

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_BAN:
		pk, err := common.HexToPubkey(communityEvent.MemberToAction)
		if err != nil {
			return err
		}

		if o.IsMemberOwnerOrAdmin(pk) {
			return errors.New("attempt to ban an owner or admin of the community from the admin side")
		}
		o.banUserFromCommunity(pk)

	case protobuf.CommunityEvent_COMMUNITY_MEMBER_UNBAN:
		pk, err := common.HexToPubkey(communityEvent.MemberToAction)
		if err != nil {
			return err
		}
		o.unbanUserFromCommunity(pk)

	default:
		return errors.New("unknown admin community event")
	}
	return nil
}

func (o *Community) NewCommunityEventClock() uint64 {
	if len(o.config.Events) == 0 {
		return o.config.CommunityDescription.Clock + 1
	}
	return o.config.Events[len(o.config.Events)-1].CommunityEventClock + 1
}

func (o *Community) addNewCommunityEvent(event *CommunityEvent) error {
	err := validateCommunityEvent(event)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(event.ToProtobuf())
	if err != nil {
		return errors.New("converting CommunityEvent to protobuf failed")
	}
	event.RawPayload = data
	o.config.Events = append(o.config.Events, *event)

	return nil
}

func (o *Community) ToCommunityEventsMessage() *CommunityEventsMessage {
	return &CommunityEventsMessage{
		CommunityID:               o.ID(),
		CommunityDescriptionClock: o.Clock(),
		Events:                    o.config.Events,
	}
}
