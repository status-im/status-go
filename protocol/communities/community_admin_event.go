package communities

import (
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) ToCreateChannelAdminEvent(channelID string, channel *protobuf.CommunityChat) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
			Channel:   channel,
		},
	}
}

func (o *Community) ToEditChannelAdminEvent(channelID string, channel *protobuf.CommunityChat) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_EDIT,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
			Channel:   channel,
		},
	}
}

func (o *Community) ToDeleteChannelAdminEvent(channelID string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
		},
	}
}

func (o *Community) ToReorderChannelAdminEvent(categoryID string, channelID string, position int) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_REORDER,
		ChannelData: &protobuf.ChannelData{
			CategoryId: categoryID,
			ChannelId:  channelID,
			Position:   int32(position),
		},
	}
}

func (o *Community) ToCreateCategoryAdminEvent(categoryID string, categoryName string, channelsIds []string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATE,
		CategoryData: &protobuf.CategoryData{
			Name:        categoryName,
			CategoryId:  categoryID,
			ChannelsIds: channelsIds,
		},
	}
}

func (o *Community) ToEditCategoryAdminEvent(categoryID string, categoryName string, channelsIds []string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_EDIT,
		CategoryData: &protobuf.CategoryData{
			Name:        categoryName,
			CategoryId:  categoryID,
			ChannelsIds: channelsIds,
		},
	}
}

func (o *Community) ToDeleteCategoryAdminEvent(categoryID string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETE,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
		},
	}
}

func (o *Community) ToReorderCategoryAdminEvent(categoryID string, position int) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_REORDER,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
			Position:   int32(position),
		},
	}
}

func (o *Community) ToBanCommunityMemberAdminEvent(pubkey string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:          o.Clock(),
		CommunityId:    o.ID(),
		Type:           protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN,
		MemberToAction: pubkey,
	}
}

func (o *Community) ToUnbanCommunityMemberAdminEvent(pubkey string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:          o.Clock(),
		CommunityId:    o.ID(),
		Type:           protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_UNBAN,
		MemberToAction: pubkey,
	}
}

func (o *Community) ToKickCommunityMemberAdminEvent(pubkey string) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:          o.Clock(),
		CommunityId:    o.ID(),
		Type:           protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK,
		MemberToAction: pubkey,
	}
}

func (o *Community) ToCommunityEditAdminEvent() *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:       o.Clock(),
		CommunityId: o.ID(),
		Type:        protobuf.CommunityAdminEvent_COMMUNITY_EDIT,
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

func (o *Community) ToCommunityBecomeMemberTokenPermissionChangeAdminEvent() *protobuf.CommunityAdminEvent {
	tokenPermissions := make(map[string]*protobuf.CommunityTokenPermission)
	for _, tp := range o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER) {
		tokenPermissions[tp.Id] = tp
	}
	return &protobuf.CommunityAdminEvent{
		Clock:            o.Clock(),
		CommunityId:      o.ID(),
		Type:             protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE,
		TokenPermissions: tokenPermissions,
	}
}

func (o *Community) ToCommunityBecomeMemberTokenPermissionDeleteAdminEvent() *protobuf.CommunityAdminEvent {
	tokenPermissions := make(map[string]*protobuf.CommunityTokenPermission)
	for _, tp := range o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER) {
		tokenPermissions[tp.Id] = tp
	}
	return &protobuf.CommunityAdminEvent{
		Clock:            o.Clock(),
		CommunityId:      o.ID(),
		Type:             protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE,
		TokenPermissions: tokenPermissions,
	}
}

func (o *Community) ToCommunityRequestToJoinAcceptAdminEvent(changes *CommunityAdminEventChanges) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:                  o.Clock(),
		CommunityId:            o.ID(),
		Type:                   protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
		MembersAdded:           changes.MembersAdded,
		AcceptedRequestsToJoin: changes.AcceptedRequestsToJoin,
	}
}

func (o *Community) ToCommunityRequestToJoinRejectAdminEvent(changes *CommunityAdminEventChanges) *protobuf.CommunityAdminEvent {
	return &protobuf.CommunityAdminEvent{
		Clock:                  o.Clock(),
		CommunityId:            o.ID(),
		Type:                   protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT,
		RejectedRequestsToJoin: changes.RejectedRequestsToJoin,
	}
}

func (o *Community) PatchCommunityDescriptionByAdminEvent(adminEvent *protobuf.CommunityAdminEvent) (*protobuf.CommunityDescription, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if adminEvent.Clock <= o.config.CommunityDescription.Clock {
		return nil, errors.New("clock for admin event is outdated")
	}

	if adminEvent.Type == protobuf.CommunityAdminEvent_UNKNOWN {
		return nil, errors.New("unknown admin event")
	}

	// create a deep copy of current community so we can patch CommunityDescription
	copy := o.createDeepCopy()

	copy.config.CommunityDescription.Clock = adminEvent.Clock

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_EDIT:
		if adminEvent.CommunityConfig == nil || adminEvent.CommunityConfig.Identity == nil ||
			adminEvent.CommunityConfig.Permissions == nil || adminEvent.CommunityConfig.AdminSettings == nil {
			return nil, errors.New("invalid config change admin event")
		}
		copy.config.CommunityDescription.Identity = adminEvent.CommunityConfig.Identity
		copy.config.CommunityDescription.Permissions = adminEvent.CommunityConfig.Permissions
		copy.config.CommunityDescription.AdminSettings = adminEvent.CommunityConfig.AdminSettings
		copy.config.CommunityDescription.IntroMessage = adminEvent.CommunityConfig.IntroMessage
		copy.config.CommunityDescription.OutroMessage = adminEvent.CommunityConfig.OutroMessage
		copy.config.CommunityDescription.Tags = adminEvent.CommunityConfig.Tags

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE:
		prevPermissions := copy.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
		newPermissions := adminEvent.TokenPermissions

		if len(newPermissions) < len(prevPermissions) {
			// we only handle additions and update in this event type
			break
		}

		if len(newPermissions) > len(prevPermissions) {
			copy.addBecomeMemberTokenPermissions(newPermissions)
			break
		}

		// update existing permissions
		copy.updateTokenPermissions(newPermissions)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE:
		copy.deleteBecomeMemberTokenPermissions(adminEvent.TokenPermissions)

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATE:
		_, err := copy.createCategory(adminEvent.CategoryData.CategoryId, adminEvent.CategoryData.Name, adminEvent.CategoryData.ChannelsIds)
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETE:
		_, err := copy.deleteCategory(adminEvent.CategoryData.CategoryId)
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_EDIT:
		_, err := copy.editCategory(adminEvent.CategoryData.CategoryId, adminEvent.CategoryData.Name, adminEvent.CategoryData.ChannelsIds)
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATE:
		err := copy.createChat(adminEvent.ChannelData.ChannelId, adminEvent.ChannelData.Channel)
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETE:
		copy.deleteChat(adminEvent.ChannelData.ChannelId)

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_EDIT:
		err := copy.editChat(adminEvent.ChannelData.ChannelId, adminEvent.ChannelData.Channel)
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_REORDER:
		_, err := copy.reorderChat(adminEvent.ChannelData.CategoryId, adminEvent.ChannelData.ChannelId, int(adminEvent.ChannelData.Position))
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_REORDER:
		_, err := copy.reorderCategories(adminEvent.CategoryData.CategoryId, int(adminEvent.CategoryData.Position))
		if err != nil {
			return nil, err
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT:
		for pkString, addedMember := range adminEvent.MembersAdded {
			pk, err := common.HexToPubkey(pkString)
			if err != nil {
				return nil, err
			}
			if !copy.HasMember(pk) {
				copy.addCommunityMember(pk, addedMember)
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT:
		// Do not remove this case!
		//
		// if we've received a rejected request to join, there's nothing that needs
		// to be done on the `Community`.
		//
		// However, we need to leave this case here, otherwise it'll end up in
		// the default case, which will error out
		break

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_KICK:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return nil, err
		}

		if copy.IsMemberOwnerOrAdmin(pk) {
			return nil, errors.New("attempt to kick an owner or admin of the community from the admin side")
		}

		copy.removeMemberFromOrg(pk)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_BAN:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return nil, err
		}

		if copy.IsMemberOwnerOrAdmin(pk) {
			return nil, errors.New("attempt to ban an owner or admin of the community from the admin side")
		}
		copy.banUserFromCommunity(pk)

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_UNBAN:
		pk, err := common.HexToPubkey(adminEvent.MemberToAction)
		if err != nil {
			return nil, err
		}
		copy.unbanUserFromCommunity(pk)

	default:
		return nil, errors.New("unknown admin community event")
	}

	return copy.config.CommunityDescription, nil
}
