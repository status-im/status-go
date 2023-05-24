package communities

import (
	"fmt"
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
		tokenPermissions := make(map[string]*protobuf.CommunityTokenPermission)
		for _, tp := range o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER) {
			tokenPermissions[tp.Id] = tp
		}

		return &protobuf.CommunityAdminEvent{
			Clock:            o.Clock(),
			CommunityId:      o.ID(),
			Type:             eventType,
			TokenPermissions: tokenPermissions,
		}
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETED:
		tokenPermissions := make(map[string]*protobuf.CommunityTokenPermission)
		for _, tp := range o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER) {
			tokenPermissions[tp.Id] = tp
		}
		return &protobuf.CommunityAdminEvent{
			Clock:            o.Clock(),
			CommunityId:      o.ID(),
			Type:             eventType,
			TokenPermissions: make(map[string]*protobuf.CommunityTokenPermission, 0),
		}
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
			ChatsAdded:  changes.ChatsAdded,
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CHANGED:
		return &protobuf.CommunityAdminEvent{
			Clock:         o.Clock(),
			CommunityId:   o.ID(),
			Type:          eventType,
			ChatsModified: communityChatChangesToCommunityChats(changes.ChatsModified),
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETED:
		return &protobuf.CommunityAdminEvent{
			Clock:         o.Clock(),
			CommunityId:   o.ID(),
			Type:          eventType,
			ChatsRemoved:  changes.ChatsRemoved,
			ChatsModified: communityChatChangesToCommunityChats(changes.ChatsModified),
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATED:
		return &protobuf.CommunityAdminEvent{
			Clock:           o.Clock(),
			CommunityId:     o.ID(),
			Type:            eventType,
			CategoriesAdded: changes.CategoriesAdded,
			ChatsModified:   communityChatChangesToCommunityChats(changes.ChatsModified),
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CHANGED:
		return &protobuf.CommunityAdminEvent{
			Clock:              o.Clock(),
			CommunityId:        o.ID(),
			Type:               eventType,
			CategoriesModified: changes.CategoriesModified,
			ChatsModified:      communityChatChangesToCommunityChats(changes.ChatsModified),
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETED:
		return &protobuf.CommunityAdminEvent{
			Clock:              o.Clock(),
			CommunityId:        o.ID(),
			Type:               eventType,
			CategoriesModified: changes.CategoriesModified,
			CategoriesRemoved:  changes.CategoriesRemoved,
			ChatsModified:      communityChatChangesToCommunityChats(changes.ChatsModified),
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CATEGORY_REORDERED:
		return &protobuf.CommunityAdminEvent{
			Clock:              o.Clock(),
			CommunityId:        o.ID(),
			Type:               eventType,
			CategoriesModified: changes.CategoriesModified,
			ChatsModified:      communityChatChangesToCommunityChats(changes.ChatsModified),
		}

	default:
		// TODO: log error
	}

	return &protobuf.CommunityAdminEvent{Type: protobuf.CommunityAdminEvent_UNKNOWN}
}

func (o *Community) PatchCommunityDescriptionByAdminEvent(adminEvent *protobuf.CommunityAdminEvent) *protobuf.CommunityDescription {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if adminEvent.Clock <= o.config.CommunityDescription.Clock {
		o.config.Logger.Warn("Clock is outdated")
	}

	// create copy of current description so it can be patched
	communityDescription := &protobuf.CommunityDescription{
		Clock:                   adminEvent.Clock,
		Members:                 o.config.CommunityDescription.Members,
		Permissions:             o.config.CommunityDescription.Permissions,
		Identity:                o.config.CommunityDescription.Identity,
		Chats:                   o.config.CommunityDescription.Chats,
		BanList:                 o.config.CommunityDescription.BanList,
		Categories:              o.config.CommunityDescription.Categories,
		ArchiveMagnetlinkClock:  o.config.CommunityDescription.ArchiveMagnetlinkClock,
		AdminSettings:           o.config.CommunityDescription.AdminSettings,
		IntroMessage:            o.config.CommunityDescription.IntroMessage,
		OutroMessage:            o.config.CommunityDescription.OutroMessage,
		Encrypted:               o.config.CommunityDescription.Encrypted,
		Tags:                    o.config.CommunityDescription.Tags,
		TokenPermissions:        o.config.CommunityDescription.TokenPermissions,
		CommunityTokensMetadata: o.config.CommunityDescription.CommunityTokensMetadata,
		ActiveMembersCount:      o.config.CommunityDescription.ActiveMembersCount,
	}

	switch adminEvent.Type {
	case protobuf.CommunityAdminEvent_COMMUNITY_CONFIG_CHANGED:
		communityDescription.Identity = adminEvent.CommunityConfig.Identity
		communityDescription.Permissions = adminEvent.CommunityConfig.Permissions
		communityDescription.AdminSettings = adminEvent.CommunityConfig.AdminSettings
		communityDescription.IntroMessage = adminEvent.CommunityConfig.IntroMessage
		communityDescription.OutroMessage = adminEvent.CommunityConfig.OutroMessage
		communityDescription.Tags = adminEvent.CommunityConfig.Tags

	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGED:
		prevPermissions := o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
		newPermissions := adminEvent.TokenPermissions
		fmt.Println("\n >>>>>>>> HANDLING PERMISSIONS CHANGED")

		if len(newPermissions) < len(prevPermissions) {
			// we only handle additions and update in this event type
			fmt.Println(">>>>>>>> BREAKING HERE")
			break
		}

		if communityDescription.TokenPermissions == nil {
			communityDescription.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
		}

		if len(newPermissions) > len(prevPermissions) {
			fmt.Println(">>>>>>>> PREV PERMISSIONS", len(prevPermissions))
			fmt.Println(">>>>>>>> NEW PERMISSIONS", len(newPermissions))
			// new permission has been added
			for id, newPermission := range newPermissions {
				exists := false
				for _, prevPermission := range prevPermissions {
					if prevPermission.Id == id {
						exists = true
						fmt.Println(">>>>>>>> BREAKING HERE 2")
						break
					}
				}
				if !exists {
					communityDescription.TokenPermissions[id] = newPermission
				}
			}
			fmt.Println(">>>>>>>> BREAKING HERE 3")
			break
		}

		fmt.Println("\n >>>>>>>> UPDATING EXISTING PERMISSIONS")
		// update existing permissions
		for id, newPermission := range newPermissions {
			communityDescription.TokenPermissions[id] = newPermission
		}
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETED:
		fmt.Println("\n >>>>>>>> HANDLING PERMISSIONS DELETED")
		prevPermissions := o.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
		newPermissions := adminEvent.TokenPermissions

		if len(newPermissions) == 0 {
			communityDescription.TokenPermissions = make(map[string]*protobuf.CommunityTokenPermission)
			break
		}

		if len(newPermissions) < len(prevPermissions) {
			for _, permission := range newPermissions {
				exists := false
				for _, prevPermission := range prevPermissions {
					if prevPermission.Id == permission.Id {
						exists = true
					}
				}
				if !exists {
					delete(communityDescription.TokenPermissions, permission.Id)
				}
			}
		}
		fmt.Println("\n >>>>>>>> ENDING HERE")
		break
	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CREATED:
		for categoryID, category := range adminEvent.CategoriesAdded {
			if _, exists := communityDescription.Categories[categoryID]; !exists {
				if communityDescription.Categories == nil {
					communityDescription.Categories = make(map[string]*protobuf.CommunityCategory)
				}
				communityDescription.Categories[categoryID] = category
			}
		}
		UpdateCommunityDescriptionChannels(communityDescription, adminEvent.ChatsModified)

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_DELETED:
		for _, categoryID := range adminEvent.CategoriesRemoved {
			delete(communityDescription.Categories, categoryID)
		}
		UpdateCommunityDescriptionCategories(communityDescription, adminEvent.CategoriesModified)
		UpdateCommunityDescriptionChannels(communityDescription, adminEvent.ChatsModified)

	case protobuf.CommunityAdminEvent_COMMUNITY_CATEGORY_CHANGED:
		for categoryID, category := range adminEvent.CategoriesModified {
			if communityDescription.Categories != nil {
				if _, exists := communityDescription.Categories[categoryID]; exists {
					communityDescription.Categories[categoryID].Name = category.Name
				}
			}
		}
		UpdateCommunityDescriptionChannels(communityDescription, adminEvent.ChatsModified)

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CREATED:
		for channelID, channel := range adminEvent.ChatsAdded {
			if _, ok := communityDescription.Chats[channelID]; !ok {
				if communityDescription.Chats == nil {
					communityDescription.Chats = make(map[string]*protobuf.CommunityChat)
				}
				communityDescription.Chats[channelID] = channel
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_DELETED:
		for key := range adminEvent.ChatsRemoved {
			delete(communityDescription.Chats, key)
		}
		UpdateCommunityDescriptionChannels(communityDescription, adminEvent.ChatsModified)

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CHANGED:
		if communityDescription.Chats != nil {
			for chatID, chat := range adminEvent.ChatsModified {
				if _, exists := o.config.CommunityDescription.Chats[chatID]; exists {
					communityDescription.Chats[chatID].Identity = chat.Identity
					communityDescription.Chats[chatID].Permissions = chat.Permissions
					communityDescription.Chats[chatID].CategoryId = chat.CategoryId
					communityDescription.Chats[chatID].Position = chat.Position
				}
			}
		}

	case protobuf.CommunityAdminEvent_COMMUNITY_CHANNEL_CATEGORY_REORDERED:
		UpdateCommunityDescriptionCategories(communityDescription, adminEvent.CategoriesModified)
		UpdateCommunityDescriptionChannels(communityDescription, adminEvent.ChatsModified)

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

	return communityDescription
}

func communityChatChangesToCommunityChats(communityChatChanges map[string]*CommunityChatChanges) map[string]*protobuf.CommunityChat {
	chats := map[string]*protobuf.CommunityChat{}
	for key, value := range communityChatChanges {
		chats[key] = value.ChatModified
	}

	return chats
}

func UpdateCommunityDescriptionChannels(communityDescription *protobuf.CommunityDescription, channels map[string]*protobuf.CommunityChat) {
	for channelID, channel := range channels {
		if communityDescription.Chats != nil {
			if _, exists := communityDescription.Chats[channelID]; exists {
				communityDescription.Chats[channelID].Position = channel.Position
				communityDescription.Chats[channelID].CategoryId = channel.CategoryId
			}
		}
	}
}

func UpdateCommunityDescriptionCategories(communityDescription *protobuf.CommunityDescription, categories map[string]*protobuf.CommunityCategory) {
	for categoryID, category := range categories {
		if communityDescription.Categories != nil {
			if _, exists := communityDescription.Categories[categoryID]; exists {
				communityDescription.Categories[categoryID].Position = category.Position
			}
		}
	}
}
