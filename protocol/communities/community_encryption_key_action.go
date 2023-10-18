package communities

import "github.com/status-im/status-go/protocol/protobuf"

type EncryptionKeyActionType int

const (
	EncryptionKeyNone EncryptionKeyActionType = iota
	EncryptionKeyAdd
	EncryptionKeyRemove
	EncryptionKeyRekey
	EncryptionKeySendToMembers
)

type EncryptionKeyAction struct {
	ActionType EncryptionKeyActionType
	Members    map[string]*protobuf.CommunityMember
}

type EncryptionKeyActions struct {
	// community-level encryption key action
	CommunityKeyAction EncryptionKeyAction

	// channel-level encryption key actions
	ChannelKeysActions map[string]EncryptionKeyAction // key is: chatID
}

func EvaluateCommunityEncryptionKeyActions(origin, modified *Community) *EncryptionKeyActions {
	if origin == nil {
		// `modified` is a new community, create empty `origin` community
		origin = &Community{
			config: &Config{
				CommunityDescription: &protobuf.CommunityDescription{
					Members:                 map[string]*protobuf.CommunityMember{},
					Permissions:             &protobuf.CommunityPermissions{},
					Identity:                &protobuf.ChatIdentity{},
					Chats:                   map[string]*protobuf.CommunityChat{},
					Categories:              map[string]*protobuf.CommunityCategory{},
					AdminSettings:           &protobuf.CommunityAdminSettings{},
					TokenPermissions:        map[string]*protobuf.CommunityTokenPermission{},
					CommunityTokensMetadata: []*protobuf.CommunityTokenMetadata{},
				},
			},
		}
	}

	changes := EvaluateCommunityChanges(origin, modified)

	result := &EncryptionKeyActions{
		CommunityKeyAction: *evaluateCommunityLevelEncryptionKeyAction(origin, modified, changes),
		ChannelKeysActions: *evaluateChannelLevelEncryptionKeyActions(origin, modified, changes),
	}
	return result
}

func evaluateCommunityLevelEncryptionKeyAction(origin, modified *Community, changes *CommunityChanges) *EncryptionKeyAction {
	originBecomeMemberPermissions := origin.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
	modifiedBecomeMemberPermissions := modified.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)

	return evaluateEncryptionKeyAction(
		originBecomeMemberPermissions,
		modifiedBecomeMemberPermissions,
		modified.config.CommunityDescription.Members,
		changes.MembersAdded,
		changes.MembersRemoved,
		changes.ControlNodeChanged != nil,
	)
}

func evaluateChannelLevelEncryptionKeyActions(origin, modified *Community, changes *CommunityChanges) *map[string]EncryptionKeyAction {
	result := make(map[string]EncryptionKeyAction)

	for channelID := range modified.config.CommunityDescription.Chats {
		chatID := modified.IDString() + channelID

		originChannelViewOnlyPermissions := origin.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL)
		originChannelViewAndPostPermissions := origin.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL)
		originChannelPermissions := append(originChannelViewOnlyPermissions, originChannelViewAndPostPermissions...)

		modifiedChannelViewOnlyPermissions := modified.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL)
		modifiedChannelViewAndPostPermissions := modified.ChannelTokenPermissionsByType(chatID, protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL)
		modifiedChannelPermissions := append(modifiedChannelViewOnlyPermissions, modifiedChannelViewAndPostPermissions...)

		membersAdded := make(map[string]*protobuf.CommunityMember)
		membersRemoved := make(map[string]*protobuf.CommunityMember)

		chatChanges, ok := changes.ChatsModified[channelID]
		if ok {
			membersAdded = chatChanges.MembersAdded
			membersRemoved = chatChanges.MembersRemoved
		}

		result[channelID] = *evaluateEncryptionKeyAction(
			originChannelPermissions,
			modifiedChannelPermissions,
			modified.config.CommunityDescription.Chats[channelID].Members,
			membersAdded, membersRemoved,
			changes.ControlNodeChanged != nil,
		)
	}

	return &result
}

func evaluateEncryptionKeyAction(originPermissions, modifiedPermissions []*CommunityTokenPermission,
	allMembers, membersAdded, membersRemoved map[string]*protobuf.CommunityMember, controlNodeChanged bool) *EncryptionKeyAction {
	result := &EncryptionKeyAction{
		ActionType: EncryptionKeyNone,
		Members:    map[string]*protobuf.CommunityMember{},
	}

	copyMap := func(source map[string]*protobuf.CommunityMember) map[string]*protobuf.CommunityMember {
		to := make(map[string]*protobuf.CommunityMember)
		for pubKey, member := range source {
			to[pubKey] = member
		}
		return to
	}

	// control node changed on closed community/channel
	if controlNodeChanged && len(modifiedPermissions) > 0 {
		result.ActionType = EncryptionKeyRekey
		result.Members = copyMap(allMembers)
		return result
	}

	// permission was just added
	if len(modifiedPermissions) > 0 && len(originPermissions) == 0 {
		result.ActionType = EncryptionKeyAdd
		result.Members = copyMap(allMembers)
		return result
	}

	// permission was just removed
	if len(modifiedPermissions) == 0 && len(originPermissions) > 0 {
		result.ActionType = EncryptionKeyRemove
		result.Members = copyMap(allMembers)
		return result
	}

	// open community/channel does not require any actions
	if len(modifiedPermissions) == 0 {
		return result
	}

	if len(membersRemoved) > 0 {
		result.ActionType = EncryptionKeyRekey
		result.Members = copyMap(allMembers)
	} else if len(membersAdded) > 0 {
		result.ActionType = EncryptionKeySendToMembers
		result.Members = copyMap(membersAdded)
	}

	return result
}
