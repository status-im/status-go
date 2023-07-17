package communities

import (
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/v1"
)

func (o *Community) ToCreateChannelCommunityEvent(channelID string, channel *protobuf.CommunityChat) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
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
		Type:                protobuf.CommunityEvent_COMMUNITY_CHANNEL_DELETE,
		ChannelData: &protobuf.ChannelData{
			ChannelId: channelID,
		},
	}
}

func (o *Community) ToReorderChannelCommunityEvent(categoryID string, channelID string, position int) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
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
		Type:                protobuf.CommunityEvent_COMMUNITY_CATEGORY_DELETE,
		CategoryData: &protobuf.CategoryData{
			CategoryId: categoryID,
		},
	}
}

func (o *Community) ToReorderCategoryCommunityEvent(categoryID string, position int) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
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
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_BAN,
		MemberToAction:      pubkey,
	}
}

func (o *Community) ToUnbanCommunityMemberCommunityEvent(pubkey string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_UNBAN,
		MemberToAction:      pubkey,
	}
}

func (o *Community) ToKickCommunityMemberCommunityEvent(pubkey string) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_KICK,
		MemberToAction:      pubkey,
	}
}

func (o *Community) ToCommunityEditCommunityEvent(description *protobuf.CommunityDescription) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Type:                protobuf.CommunityEvent_COMMUNITY_EDIT,
		CommunityConfig: &protobuf.CommunityConfig{
			Identity:      description.Identity,
			Permissions:   description.Permissions,
			AdminSettings: description.AdminSettings,
			IntroMessage:  description.IntroMessage,
			OutroMessage:  description.OutroMessage,
			Tags:          description.Tags,
		},
	}
}

func (o *Community) ToCommunityTokenPermissionChangeCommunityEvent(permission *protobuf.CommunityTokenPermission) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_CHANGE,
		TokenPermission:     permission,
	}
}

func (o *Community) ToCommunityTokenPermissionDeleteCommunityEvent(permission *protobuf.CommunityTokenPermission) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock: o.NewCommunityEventClock(),
		Type:                protobuf.CommunityEvent_COMMUNITY_MEMBER_TOKEN_PERMISSION_DELETE,
		TokenPermission:     permission,
	}
}

func (o *Community) ToCommunityRequestToJoinAcceptCommunityEvent(changes *CommunityEventChanges) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock:    o.NewCommunityEventClock(),
		Type:                   protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_ACCEPT,
		MembersAdded:           changes.MembersAdded,
		AcceptedRequestsToJoin: changes.AcceptedRequestsToJoin,
	}
}

func (o *Community) ToCommunityRequestToJoinRejectCommunityEvent(changes *CommunityEventChanges) *CommunityEvent {
	return &CommunityEvent{
		CommunityEventClock:    o.NewCommunityEventClock(),
		Type:                   protobuf.CommunityEvent_COMMUNITY_REQUEST_TO_JOIN_REJECT,
		RejectedRequestsToJoin: changes.RejectedRequestsToJoin,
	}
}

func (o *Community) UpdateCommunityByEvents(communityEventMessage *CommunityEventsMessage) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Validate that EventsBaseCommunityDescription was signed by the control node
	description, err := validateAndGetEventsMessageCommunityDescription(communityEventMessage.EventsBaseCommunityDescription, o.config.ID)
	if err != nil {
		return nil, err
	}

	if description.Clock != o.config.CommunityDescription.Clock {
		return nil, errors.New("clock for admin event message is outdated")
	}

	// Create a deep copy of current community so we can update CommunityDescription by new admin events
	copy := o.CreateDeepCopy()

	// Merge community admin events to existing community. Admin events must be stored to the db
	// during saving the community
	o.mergeCommunityEvents(communityEventMessage)

	copy.config.CommunityDescription = description
	copy.config.EventsData = o.config.EventsData

	// Update the copy of the CommunityDescription by community events
	err = copy.updateCommunityDescriptionByEvents()
	if err != nil {
		return nil, err
	}

	// Evaluate `CommunityChanges` data by searching a difference between `CommunityDescription`
	// from the DB and `CommunityDescription` patched by community events
	changes := EvaluateCommunityChanges(o.config.CommunityDescription, copy.config.CommunityDescription)

	// TODO: need to figure out is it ok to save marshaledCommunityDescription without the signature
	marshaledCommDescr, err := proto.Marshal(copy.config.CommunityDescription)
	if err != nil {
		return nil, err
	}

	rawMessage, err := protocol.WrapMessageV1(marshaledCommDescr, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, copy.PrivateKey())
	if err != nil {
		return nil, err
	}

	copy.config.MarshaledCommunityDescription = rawMessage

	changes.Community = copy

	return changes, nil
}

func (o *Community) updateCommunityDescriptionByEvents() error {
	for i := range o.config.EventsData.Events {
		communityEvent := &o.config.EventsData.Events[i]
		err := validateCommunityEvent(communityEvent)
		if err != nil {
			return err
		}
		err = o.updateCommunityDescriptionByCommunityEvent(*communityEvent)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Community) updateCommunityDescriptionByCommunityEvent(communityEvent CommunityEvent) error {
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
	}
	return nil
}

func (o *Community) NewCommunityEventClock() uint64 {
	return uint64(time.Now().Unix())
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

	// All events must be built on top of the control node CommunityDescription
	// If there were no events before, extract CommunityDescription from MarshaledCommunityDescription
	// and check the signature
	if o.config.EventsData == nil || len(o.config.EventsData.EventsBaseCommunityDescription) == 0 {
		_, err := validateAndGetEventsMessageCommunityDescription(o.config.MarshaledCommunityDescription, o.config.ID)
		if err != nil {
			return err
		}

		o.config.EventsData = &EventsData{
			EventsBaseCommunityDescription: o.config.MarshaledCommunityDescription,
			Events:                         []CommunityEvent{},
		}
	}

	event.RawPayload = data
	o.config.EventsData.Events = append(o.config.EventsData.Events, *event)

	return nil
}

func (o *Community) ToCommunityEventsMessage() *CommunityEventsMessage {
	return &CommunityEventsMessage{
		CommunityID:                    o.ID(),
		EventsBaseCommunityDescription: o.config.EventsData.EventsBaseCommunityDescription,
		Events:                         o.config.EventsData.Events,
	}
}

func validateAndGetEventsMessageCommunityDescription(signedDescription []byte, signerPubkey *ecdsa.PublicKey) (*protobuf.CommunityDescription, error) {
	metadata := &protobuf.ApplicationMetadataMessage{}

	err := proto.Unmarshal(signedDescription, metadata)
	if err != nil {
		return nil, err
	}

	if metadata.Type != protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION {
		return nil, ErrInvalidMessage
	}

	signer, err := metadata.RecoverKey()
	if err != nil {
		return nil, err
	}

	if !signer.Equal(signerPubkey) {
		return nil, errors.New("CommunityDescription was not signed by an owner")
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(metadata.Payload, description)
	if err != nil {
		return nil, err
	}

	return description, nil
}
