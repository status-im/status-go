package chat

import (
	"context"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var (
	ErrChatNotFound            = errors.New("can't find chat")
	ErrCommunityNotFound       = errors.New("can't find community")
	ErrCommunitiesNotSupported = errors.New("communities are not supported")
	ErrChatTypeNotSupported    = errors.New("chat type not supported")
)

type ChannelGroupType string

const Personal ChannelGroupType = "personal"
const Community ChannelGroupType = "community"

type PinnedMessages struct {
	Cursor         string
	PinnedMessages []*common.PinnedMessage
}

type Member struct {
	// Community Role
	Role protobuf.CommunityMember_Roles `json:"role,omitempty"`
	// Joined indicates if the member has joined the group chat
	Joined bool `json:"joined"`
}

type Chat struct {
	ID                       string                             `json:"id"`
	Name                     string                             `json:"name"`
	Description              string                             `json:"description"`
	Color                    string                             `json:"color"`
	Emoji                    string                             `json:"emoji"`
	Active                   bool                               `json:"active"`
	ChatType                 protocol.ChatType                  `json:"chatType"`
	Timestamp                int64                              `json:"timestamp"`
	LastClockValue           uint64                             `json:"lastClockValue"`
	DeletedAtClockValue      uint64                             `json:"deletedAtClockValue"`
	ReadMessagesAtClockValue uint64                             `json:"readMessagesAtClockValue"`
	UnviewedMessagesCount    uint                               `json:"unviewedMessagesCount"`
	UnviewedMentionsCount    uint                               `json:"unviewedMentionsCount"`
	LastMessage              *common.Message                    `json:"lastMessage"`
	Members                  map[string]Member                  `json:"members,omitempty"`
	MembershipUpdates        []v1protocol.MembershipUpdateEvent `json:"membershipUpdateEvents"`
	Alias                    string                             `json:"alias,omitempty"`
	Identicon                string                             `json:"identicon"`
	Muted                    bool                               `json:"muted"`
	InvitationAdmin          string                             `json:"invitationAdmin,omitempty"`
	ReceivedInvitationAdmin  string                             `json:"receivedInvitationAdmin,omitempty"`
	Profile                  string                             `json:"profile,omitempty"`
	CommunityID              string                             `json:"communityId"`
	CategoryID               string                             `json:"categoryId"`
	Position                 int32                              `json:"position,omitempty"`
	Permissions              *protobuf.CommunityPermissions     `json:"permissions,omitempty"`
	Joined                   int64                              `json:"joined,omitempty"`
	SyncedTo                 uint32                             `json:"syncedTo,omitempty"`
	SyncedFrom               uint32                             `json:"syncedFrom,omitempty"`
	FirstMessageTimestamp    uint32                             `json:"firstMessageTimestamp,omitempty"`
	Highlight                bool                               `json:"highlight,omitempty"`
	PinnedMessages           *PinnedMessages                    `json:"pinnedMessages,omitempty"`
	// Deprecated: CanPost is deprecated in favor of CanPostMessages/CanPostReactions/etc.
	// For now CanPost will equal to CanPostMessages.
	CanPost                 bool   `json:"canPost"`
	CanPostMessages         bool   `json:"canPostMessages"`
	CanPostReactions        bool   `json:"canPostReactions"`
	ViewersCanPostReactions bool   `json:"viewersCanPostReactions"`
	Base64Image             string `json:"image,omitempty"`
	HideIfPermissionsNotMet bool   `json:"hideIfPermissionsNotMet,omitempty"`
}

func NewAPI(service *Service) *API {
	return &API{
		s:   service,
		log: log.New("package", "status-go/services/chat.API"),
	}
}

type API struct {
	s   *Service
	log log.Logger
}

func (api *API) toAPIChat(protocolChat *protocol.Chat, community *communities.Community, pubKey string, skipPinnedMessages bool) (*Chat, error) {
	chat := &Chat{
		ID:                       strings.TrimPrefix(protocolChat.ID, protocolChat.CommunityID),
		Name:                     protocolChat.Name,
		Description:              protocolChat.Description,
		Color:                    protocolChat.Color,
		Emoji:                    protocolChat.Emoji,
		Active:                   protocolChat.Active,
		ChatType:                 protocolChat.ChatType,
		Timestamp:                protocolChat.Timestamp,
		LastClockValue:           protocolChat.LastClockValue,
		DeletedAtClockValue:      protocolChat.DeletedAtClockValue,
		ReadMessagesAtClockValue: protocolChat.ReadMessagesAtClockValue,
		UnviewedMessagesCount:    protocolChat.UnviewedMessagesCount,
		UnviewedMentionsCount:    protocolChat.UnviewedMentionsCount,
		LastMessage:              protocolChat.LastMessage,
		MembershipUpdates:        protocolChat.MembershipUpdates,
		Alias:                    protocolChat.Alias,
		Identicon:                protocolChat.Identicon,
		Muted:                    protocolChat.Muted,
		InvitationAdmin:          protocolChat.InvitationAdmin,
		ReceivedInvitationAdmin:  protocolChat.ReceivedInvitationAdmin,
		Profile:                  protocolChat.Profile,
		CommunityID:              protocolChat.CommunityID,
		CategoryID:               protocolChat.CategoryID,
		Joined:                   protocolChat.Joined,
		SyncedTo:                 protocolChat.SyncedTo,
		SyncedFrom:               protocolChat.SyncedFrom,
		FirstMessageTimestamp:    protocolChat.FirstMessageTimestamp,
		Highlight:                protocolChat.Highlight,
		Base64Image:              protocolChat.Base64Image,
	}

	if protocolChat.OneToOne() {
		chat.Name = "" // Emptying since it contains non useful data
	}

	if !skipPinnedMessages {
		pinnedMessages, cursor, err := api.s.messenger.PinnedMessageByChatID(protocolChat.ID, "", -1)
		if err != nil {
			return nil, err
		}

		if len(pinnedMessages) != 0 {
			chat.PinnedMessages = &PinnedMessages{
				Cursor:         cursor,
				PinnedMessages: pinnedMessages,
			}
		}
	}

	err := chat.populateCommunityFields(community)
	if err != nil {
		return nil, err
	}

	// chatMembers, err := getChatMembers(protocolChat, community, pubKey)
	// if err != nil {
	// 	return nil, err
	// }
	// chat.Members = chatMembers

	return chat, nil
}

func (api *API) getCommunityByID(id string) (*communities.Community, error) {
	communityID, err := hexutil.Decode(id)
	if err != nil {
		return nil, err
	}

	community, err := api.s.messenger.GetCommunityByID(communityID)
	if community == nil && err == nil {
		return nil, ErrCommunityNotFound
	}

	return community, err
}

func (chat *Chat) populateCommunityFields(community *communities.Community) error {
	chat.CanPost = true
	chat.CanPostMessages = true
	chat.CanPostReactions = true
	if community == nil {
		return nil
	}

	commChat, exists := community.Chats()[chat.ID]
	if !exists {
		// Skip unknown community chats. They might be channels that were deleted
		return nil
	}

	canPostMessages, err := community.CanMemberIdentityPost(chat.ID, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)
	if err != nil {
		return err
	}

	canPostReactions, err := community.CanMemberIdentityPost(chat.ID, protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
	if err != nil {
		return err
	}

	chat.CategoryID = commChat.CategoryId
	chat.HideIfPermissionsNotMet = commChat.HideIfPermissionsNotMet
	chat.Position = commChat.Position
	chat.Permissions = commChat.Permissions
	chat.Emoji = commChat.Identity.Emoji
	chat.Name = commChat.Identity.DisplayName
	chat.Description = commChat.Identity.Description
	chat.CanPost = canPostMessages
	chat.CanPostMessages = canPostMessages
	chat.CanPostReactions = canPostReactions
	chat.ViewersCanPostReactions = commChat.ViewersCanPostReactions

	return nil
}

func (api *API) EditChat(ctx context.Context, communityID types.HexBytes, chatID string, name string, color string, image images.CroppedImage) (*Chat, error) {
	if len(communityID) != 0 {
		return nil, ErrCommunitiesNotSupported
	}

	chatToEdit := api.s.messenger.Chat(chatID)
	if chatToEdit == nil {
		return nil, ErrChatNotFound
	}

	if chatToEdit.ChatType != protocol.ChatTypePrivateGroupChat {
		return nil, ErrChatTypeNotSupported
	}

	response, err := api.s.messenger.EditGroupChat(ctx, chatID, name, color, image)
	if err != nil {
		return nil, err
	}

	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))
	return api.toAPIChat(response.Chats()[0], nil, pubKey, false)
}
