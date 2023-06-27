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

func (c *Community) updateCommuntyDescriptionByNewAdminsEvents() error {
	prevAdminClock := c.config.CommunityDescription.Clock + 1
	for i := range c.config.AdminsEvents {
		adminEvent := &c.config.AdminsEvents[i]
		if !adminEvent.Applied {
			err := validateCommunityAdminEvent(adminEvent)
			if err != nil {
				return err
			}
			err = c.updateCommuntyDescriptionByAdminEvent(adminEvent, prevAdminClock)
			if err != nil {
				return err
			}
		}
		prevAdminClock = adminEvent.AdminEventClock
		adminEvent.Applied = true
	}
	return nil
}

func (c *Community) updateCommuntyDescriptionByAppliedAdminsEvents() error {
	prevAdminClock := c.config.CommunityDescription.Clock + 1
	for _, adminEvent := range c.config.AdminsEvents {
		if adminEvent.Applied {
			err := c.updateCommuntyDescriptionByAdminEvent(&adminEvent, prevAdminClock)
			if err != nil {
				return err
			}
		}
		prevAdminClock = adminEvent.AdminEventClock
	}
	return nil
}

func (c *Community) updateCommuntyDescriptionByAdminEvent(adminEvent *CommunityAdminEvent, prevAdminClock uint64) error {
	if adminEvent.AdminEventClock <= c.config.CommunityDescription.Clock {
		return errors.New("clock for admin event is outdated")
	}

	if !(prevAdminClock == adminEvent.AdminEventClock || prevAdminClock+1 == adminEvent.AdminEventClock) {
		return errors.New("clock for admin event is not in order or contains missing event")
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_EDIT:
		c.config.CommunityDescription.Identity = adminEvent.CommunityConfig.Identity
		c.config.CommunityDescription.Permissions = adminEvent.CommunityConfig.Permissions
		c.config.CommunityDescription.AdminSettings = adminEvent.CommunityConfig.AdminSettings
		c.config.CommunityDescription.IntroMessage = adminEvent.CommunityConfig.IntroMessage
		c.config.CommunityDescription.OutroMessage = adminEvent.CommunityConfig.OutroMessage
		c.config.CommunityDescription.Tags = adminEvent.CommunityConfig.Tags

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE:
		_, exists := c.config.CommunityDescription.TokenPermissions[adminEvent.TokenPermission.Id]
		if exists {
			_, err := c.updateTokenPermission(adminEvent.TokenPermission)
			if err != nil {
				return err
			}
		} else {
			_, err := c.addTokenPermission(adminEvent.TokenPermission)
			if err != nil {
				return err
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE:
		_, err := c.deleteTokenPermission(adminEvent.TokenPermission.Id)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATE:
		_, err := c.createCategory(adminEvent.CategoryData.CategoryId, adminEvent.CategoryData.Name, adminEvent.CategoryData.ChannelsIds)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETE:
		_, err := c.deleteCategory(adminEvent.CategoryData.CategoryId)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_EDIT:
		_, err := c.editCategory(adminEvent.CategoryData.CategoryId, adminEvent.CategoryData.Name, adminEvent.CategoryData.ChannelsIds)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATE:
		err := c.createChat(adminEvent.ChannelData.ChannelId, adminEvent.ChannelData.Channel)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETE:
		c.deleteChat(adminEvent.ChannelData.ChannelId)

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_EDIT:
		err := c.editChat(adminEvent.ChannelData.ChannelId, adminEvent.ChannelData.Channel)
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_REORDER:
		_, err := c.reorderChat(adminEvent.ChannelData.CategoryId, adminEvent.ChannelData.ChannelId, int(adminEvent.ChannelData.Position))
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_REORDER:
		_, err := c.reorderCategories(adminEvent.CategoryData.CategoryId, int(adminEvent.CategoryData.Position))
		if err != nil {
			return err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		for pkString, addedMember := range adminEvent.MembersAdded {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return err
			}
			if !c.HasMember(pk) {
				c.addCommunityMember(pk, addedMember)
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		for pkString, _ := range adminEvent.RejectedRequestsToJoin {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return err
			}
			c.removeMemberFromOrg(pk)
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return err
		}

		if c.IsMemberOwnerOrAdmin(pk) {
			return errors.New("attempt to kick an owner or admin of the community from the admin side")
		}

		c.removeMemberFromOrg(pk)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return err
		}

		if c.IsMemberOwnerOrAdmin(pk) {
			return errors.New("attempt to ban an owner or admin of the community from the admin side")
		}
		c.banUserFromCommunity(pk)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_UNBAN:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return err
		}
		c.unbanUserFromCommunity(pk)

	default:
		return errors.New("unknown admin community event")
	}
	return nil
}

func (c *Community) NewAdminEventClock() uint64 {
	if len(c.config.AdminsEvents) == 0 {
		return c.config.CommunityDescription.Clock + 1
	}
	return c.config.AdminsEvents[len(c.config.AdminsEvents)-1].AdminEventClock + 1
}

func (c *Community) addNewCommunityAdminEvent(event *CommunityAdminEvent) error {
	err := validateCommunityAdminEvent(event)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(event.ToProtobuf())
	if err != nil {
		return errors.New("converting CommunityAdminEvent to protobuf failed")
	}
	event.RawPayload = data
	c.config.AdminsEvents = append(c.config.AdminsEvents, *event)

	return nil
}

func (c *Community) ToCommunityEventsMessage() *CommunityEventsMessage {
	return &CommunityEventsMessage{
		CommunityId:               c.ID(),
		CommunityDescriptionClock: c.Clock(),
		AdminsEvents:              c.config.AdminsEvents,
	}
}
